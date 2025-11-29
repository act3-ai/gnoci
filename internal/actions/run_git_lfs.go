// Package actions holds actions called by the root git-remote-oci command.
package actions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry"

	"github.com/act3-ai/gnoci/internal/model"
	"github.com/act3-ai/gnoci/internal/progress"
	"github.com/act3-ai/gnoci/pkg/apis"
	"github.com/act3-ai/gnoci/pkg/apis/gnoci.act3-ai.io/v1alpha1"
	"github.com/act3-ai/gnoci/pkg/protocol/lfs"
	"github.com/act3-ai/gnoci/pkg/protocol/lfs/comms"
	"github.com/act3-ai/go-common/pkg/config"
	"github.com/go-git/go-git/v5"
	"github.com/opencontainers/go-digest"
)

// GitLFS represents the base action.
type GitLFS struct {
	version   string
	apiScheme *runtime.Scheme
	// ConfigFiles contains a list of potential configuration file locations.
	ConfigFiles []string

	// local temp files
	ociStore *file.Store
	lfsStore string

	// OCI remote
	ref registry.Reference
	gt  oras.GraphTarget

	// git-lfs request and response handler
	comm comms.Communicator
}

// NewGitLFS creates a new Tool with default values.
func NewGitLFS(in io.Reader, out io.Writer, version string, cfgFiles []string) *GitLFS {
	return &GitLFS{
		version:     version,
		apiScheme:   apis.NewScheme(),
		ConfigFiles: cfgFiles,
		comm:        comms.NewCommunicator(in, out),
	}
}

func (action *GitLFS) init(ctx context.Context, initReq *lfs.InitRequest) (func() error, error) {
	cfg, err := action.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting configuration: %w", err)
	}

	// TODO: How can we get the actual git dir?
	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, fmt.Errorf("opening local repository: %w", err)
	}

	ref, err := resolveAddress(ctx, initReq.Remote, repo)
	if err != nil {
		return nil, fmt.Errorf("resolving remote URL: %w", err)
	}
	action.ref = ref

	// var fstorePath string
	action.gt, _, action.ociStore, err = initRemoteConn(ctx, ref, repoOptsFromConfig(ref.Host(), cfg))
	if err != nil {
		return nil, fmt.Errorf("initializing remote connection: %w", err)
	}

	if initReq.Operation == lfs.DownloadOperation {
		action.lfsStore, err = os.MkdirTemp(os.TempDir(), "git-lfs-remote-oci-pull-*")
		if err != nil {
			return nil, fmt.Errorf("preparing temporary LFS pull directory: %w", err)
		}
	}

	cleanUpFn := func() error {
		var errs []error
		if err := action.ociStore.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing oci file store: %w", err))
		}

		if err := os.RemoveAll(action.lfsStore); err != nil {
			errs = append(errs, fmt.Errorf("removing temporary LFS file store: %w", err))
		}

		// if err := os.RemoveAll(fstorePath); err != nil {
		// 	slog.ErrorContext(ctx, "cleaning up temporary files", slog.String("error", err.Error()))
		// 	errs = append(errs, fmt.Errorf("cleaning up temporary"))
		// }

		return errors.Join(errs...)
	}

	return cleanUpFn, nil
}

// Run runs the the primary git-remote-oci action.
func (action *GitLFS) Run(ctx context.Context) error {
	slog.DebugContext(ctx, "running git-lfs-remote-oci")

	initReq, err := action.comm.ReceiveInitRequest(ctx)
	if err != nil {
		return fmt.Errorf("receiving InitRequest: %w", err)
	}

	cleanup, err := action.init(ctx, initReq)
	defer func() {
		if err := cleanup(); err != nil {
			slog.ErrorContext(ctx, "cleaning up temporary files", slog.String("error", err.Error()))
		}
	}()
	if err != nil {
		return action.comm.WriteInitResponse(ctx, err)
	}

	remote := model.NewLFSModeler(action.ref, action.ociStore, action.gt)

	subject, err := remote.FetchOrDefault(ctx)
	if err != nil {
		return fmt.Errorf("fetching base git OCI metadata: %w", err)
	}

	if _, err = remote.FetchLFSOrDefault(ctx); err != nil {
		return err
	}

	if err := action.comm.WriteInitResponse(ctx, nil); err != nil {
		return err
	}

	switch initReq.Operation {
	case lfs.DownloadOperation:
		return action.runDownload(ctx, remote)
	case lfs.UploadOperation:
		return action.runUpload(ctx, subject, remote)
	default:
		// theoretically impossible
		return fmt.Errorf("%w: %s", lfs.ErrInvalidOperation, initReq.Operation)
	}
}

func (action *GitLFS) runDownload(ctx context.Context, remote model.ReadOnlyLFSModeler) error {
	slog.DebugContext(ctx, "handling download requests")

	for {
		slog.DebugContext(ctx, "waiting for download request")

		transferReq, err := action.comm.ReceiveTransferRequest(ctx)
		switch {
		case err != nil:
			return err
		case transferReq.Event == lfs.TerminateEvent:
			// done
			slog.DebugContext(ctx, "received terminate request")
			return nil
		case transferReq.Event != lfs.DownloadEvent:
			// git-lfs did not adhere to it's own protocol
			return fmt.Errorf("unexpected event %s, expected %s", transferReq.Event, lfs.DownloadEvent)
		default:
			path, err := action.downloadLFSLayer(ctx, transferReq, remote)
			err = action.comm.WriteTransferDownloadResponse(ctx, transferReq.Oid, path, err)
			if err != nil {
				return fmt.Errorf("writing transfer response: %w", err)
			}
		}
	}
}

func (action *GitLFS) downloadLFSLayer(ctx context.Context, transferReq *lfs.TransferRequest, remote model.ReadOnlyLFSModeler) (string, error) {
	pChan := make(chan progress.Progress)
	done := make(chan struct{})
	go func() {
		for pUpdate := range pChan {
			err := action.comm.WriteProgress(ctx,
				transferReq.Oid,
				pUpdate.Total,
				pUpdate.Delta)
			if err != nil {
				slog.WarnContext(ctx, "writing progress",
					slog.String("error", err.Error()))
			}
		}
		done <- struct{}{}
	}()

	fetchOpts := &model.FetchLFSOptions{
		Progress: &model.ProgressOptions{
			Info: pChan,
		},
	}

	// TODO: convenient that LFS uses sha256 by default, but are other digest methods out there?
	rc, err := remote.FetchLFSLayer(ctx, digest.Digest(transferReq.Oid), fetchOpts)
	if err != nil {
		return "", fmt.Errorf("fetching LFS file: %w", err)
	}
	defer rc.Close()

	tmpFilePath := filepath.Join(action.lfsStore, transferReq.Oid)
	f, err := os.OpenFile(tmpFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return "", fmt.Errorf("opening LFS temp file: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, rc)
	<-done
	switch {
	case err != nil:
		return "", fmt.Errorf("copying LFS temp file: %w", err)
	case n != transferReq.Size:
		// TODO: double check protocol spec, LFS may handle this validation for us
		return "", fmt.Errorf("unexpected LFS file size, expected %d, got %d", transferReq.Size, n)
	default:
		return tmpFilePath, nil
	}
}

func (action *GitLFS) runUpload(ctx context.Context, subject ocispec.Descriptor, remote model.LFSModeler) error {
	slog.DebugContext(ctx, "handling upload requests")

	for {
		slog.DebugContext(ctx, "waiting for upload request")

		transferReq, err := action.comm.ReceiveTransferRequest(ctx)
		switch {
		case err != nil:
			return err
		case transferReq.Event == lfs.TerminateEvent:
			// done with LFS files
			slog.DebugContext(ctx, "received terminate request")
			_, err := remote.PushLFSManifest(ctx, subject)
			if err != nil {
				return fmt.Errorf("pushing LFS manifest to OCI: %w", err)
			}
			return nil
		case transferReq.Event != lfs.UploadEvent:
			// git-lfs did not adhere to it's own protocol
			return fmt.Errorf("unexpected event %s, expected %s", transferReq.Event, lfs.UploadEvent)
		default:
			err := action.uploadLFSLayer(ctx, transferReq, remote)
			err = action.comm.WriteTransferUploadResponse(ctx, transferReq.Oid, err)
			if err != nil {
				return fmt.Errorf("writing transfer response: %w", err)
			}
		}
	}
}

func (action *GitLFS) uploadLFSLayer(ctx context.Context, transferReq *lfs.TransferRequest, remote model.LFSModeler) error {
	pChan := make(chan progress.Progress) // closed by [progress.NewTicker] when reading completes
	done := make(chan struct{})
	go func() {
		for pUpdate := range pChan {
			err := action.comm.WriteProgress(ctx, transferReq.Oid, pUpdate.Total, pUpdate.Delta)
			if err != nil {
				slog.WarnContext(ctx, "writing progress update", slog.String("error", err.Error()))
			}
		}
		done <- struct{}{}
	}()

	pushOpts := &model.PushLFSOptions{
		Progress: &model.ProgressOptions{
			Info: pChan,
		},
	}
	if _, err := remote.PushLFSFile(ctx, transferReq.Path, pushOpts); err != nil {
		err := action.comm.WriteTransferUploadResponse(ctx, transferReq.Oid, fmt.Errorf("preparing git-lfs file for transfer: %w", err))
		if err != nil {
			<-done
			return err
		}
	}
	<-done

	return nil
}

// GetScheme returns the runtime scheme used for configuration file loading.
func (action *GitLFS) GetScheme() *runtime.Scheme {
	return action.apiScheme
}

// GetConfig loads Configuration using the current git-remote-oci options.
func (action *GitLFS) GetConfig(ctx context.Context) (c *v1alpha1.Configuration, err error) {
	c = &v1alpha1.Configuration{}

	slog.DebugContext(ctx, "searching for configuration files", slog.Any("cfgFiles", action.ConfigFiles))

	err = config.Load(slog.Default(), action.GetScheme(), c, action.ConfigFiles)
	if err != nil {
		return c, fmt.Errorf("loading configuration: %w", err)
	}

	defer slog.DebugContext(ctx, "using config", slog.Any("configuration", c))

	return c, nil
}

// resolveAddress trims an OCI URL or resolves a shortname to a URL.
func resolveAddress(ctx context.Context, remote string, repo *git.Repository) (registry.Reference, error) {
	var remoteURL string
	if strings.HasPrefix(remote, "oci://") {
		slog.DebugContext(ctx, "received full remote URL", slog.String("url", remote))
		remoteURL = trimProtocol(remote)
	} else {
		slog.DebugContext(ctx, "received remote shortname", slog.String("shortname", remote))

		// Look up the remote by name
		remote, err := repo.Remote(trimProtocol(remote)) // sanity?
		if err != nil {
			return registry.Reference{}, fmt.Errorf("resolving remote URL for %s: %w", remote, err)
		}
		remoteURLs := remote.Config().URLs
		if len(remoteURLs) < 1 {
			return registry.Reference{}, fmt.Errorf("no URLs configured for remote %s", remote)
		}
		remoteURL = trimProtocol(remoteURLs[0]) // TODO: do we just push to multiple if more than one URL is provided? How would git-remote-oci handle this?
		slog.DebugContext(ctx, "resolved remote URL", "url", remoteURL)
	}

	parsedRef, err := registry.ParseReference(remoteURL)
	if err != nil {
		return registry.Reference{}, fmt.Errorf("invalid reference %s: %w", remoteURL, err)
	}

	return parsedRef, nil
}

// trimProtocol trims a oci:// protocol prefix.
func trimProtocol(remote string) string {
	return strings.TrimPrefix(remote, "oci://")
}
