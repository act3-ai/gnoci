// Package actions holds actions called by the root git-remote-oci command.
package actions

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"

	"github.com/act3-ai/gnoci/internal/lfs"
	"github.com/act3-ai/gnoci/internal/ociutil/model"
	"github.com/go-git/go-git/v5"
)

// GitLFS represents the base action.
type GitLFS struct {
	// OCI remote
	gt     oras.GraphTarget
	remote model.LFSModeler
	in     *bufio.Scanner
	out    io.Writer

	version string
}

// NewGitLFS creates a new Tool with default values.
func NewGitLFS(in io.Reader, out io.Writer, version string) *GitLFS {
	return &GitLFS{
		in:      bufio.NewScanner(in),
		out:     out,
		version: version,
	}
}

// Run runs the the primary git-remote-oci action.
func (action *GitLFS) Run(ctx context.Context) error {
	slog.DebugContext(ctx, "running git-lfs-remote-oci")
	line, err := action.readLine()
	if err != nil {
		return err
	}

	// first message is always an InitRequest
	var initReq lfs.InitRequest
	if err := json.Unmarshal(line, &initReq); err != nil {
		return fmt.Errorf("decoding InitRequest: %w", err)
	}

	if err := initReq.Validate(); err != nil {
		return err
	}

	// TODO: How can we get the actual git dir?
	repo, err := git.PlainOpen(".")
	if err != nil {
		return fmt.Errorf("opening local repository: %w", err)
	}

	// Look up the remote by name
	remote, err := repo.Remote(initReq.Remote)
	if err != nil {
		return fmt.Errorf("resolving remote URL for %s: %w", initReq.Remote, err)
	}
	remoteURLs := remote.Config().URLs
	if len(remoteURLs) < 1 {
		return fmt.Errorf("no URLs configured for remote %s", initReq.Remote)
	}
	remoteName := strings.TrimPrefix(remoteURLs[0], "oci://")
	slog.DebugContext(ctx, "resolved remote URL", "url", remoteName)

	var fstorePath string
	var fstore *file.Store
	action.gt, fstorePath, fstore, err = initRemoteConn(ctx, remoteName)
	if err != nil {
		return fmt.Errorf("initializing: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(fstorePath); err != nil {
			slog.ErrorContext(ctx, "cleaning up temporary files", slog.String("error", err.Error()))
		}
	}()
	defer func() {
		if err := fstore.Close(); err != nil {
			slog.ErrorContext(ctx, "closing OCI file store", slog.String("error", err.Error()))
		}
	}()

	action.remote = model.NewLFSModeler(remoteName, fstore, action.gt)

	if err := action.remote.FetchOrDefault(ctx); err != nil {
		return fmt.Errorf("fetching base git OCI metadata: %w", err)
	}

	slog.DebugContext(ctx, "fetching LFS manifest or defaulting")
	if err := action.remote.FetchLFSOrDefault(ctx); err != nil {
		return err
	}

	action.writeInitResponse(ctx, nil)

	switch initReq.Operation {
	case lfs.DownloadOperation:
		return action.runDownload(ctx)
	case lfs.UploadOperation:
		return action.runUpload(ctx)
	default:
		// theoretically impossible
		return fmt.Errorf("%w: %s", lfs.ErrInvalidOperation, initReq.Operation)
	}
}

func (action *GitLFS) runDownload(ctx context.Context) error {
	slog.DebugContext(ctx, "handling download requests")
	return errors.New("not implemented")
}

func (action *GitLFS) runUpload(ctx context.Context) error {
	slog.DebugContext(ctx, "handling upload requests")
	// TODO: by their protocol spec, they block until the transfer is complete
	// this is far less than ideal for us. Unfortunately, we may not be able
	// to do this concurrently as we would not be able to update the LFS
	// manifest across processes, as their concurrency is done with multiple
	// invocations of our tool.
	// TODO: if the above is true, we need to refactor such that we write err
	// responses to git-lfs within the goroutines spun up by model.LFSModeler.PushLFS
	for {
		slog.DebugContext(ctx, "waiting for upload request")
		line, err := action.readLine()
		if err != nil {
			return err
		}
		slog.DebugContext(ctx, "received upload request", slog.String("request", string(line)))

		var transferReq lfs.TransferRequest
		if err := json.Unmarshal(line, &transferReq); err != nil {
			// TODO: is it possible to write back to git-lfs here or is this fatal?
			return fmt.Errorf("decoding TransferRequest: %w", err)
		}

		// TODO: validate the transfer request

		if transferReq.Event == lfs.TerminateEvent {
			slog.DebugContext(ctx, "received terminate request")
			break
		}

		if _, err := action.remote.AddLFSFile(ctx, transferReq.Path); err != nil {
			action.writeTransferResponse(ctx, transferReq.Oid, transferReq.Path, fmt.Errorf("preparing git-lfs file for transfer: %w", err))
			continue
		}

		// HACK: is this necessary? per the spec, it seems like it is...
		action.writeProgress(ctx, transferReq.Oid, 1, 1)

		// notify completion
		action.writeTransferResponse(ctx, transferReq.Oid, "", nil)
	}

	_, err := action.remote.PushLFS(ctx)
	if err != nil {
		return fmt.Errorf("pushing LFS to OCI: %w", err)
	}

	return nil
}

func (action *GitLFS) writeInitResponse(ctx context.Context, err error) {
	slog.DebugContext(ctx, "writing init response")

	// TODO: is this necessary or can we marshal with an empty error?
	raw := []byte("{}")
	if err != nil {
		initResp := lfs.InitResponse{
			Error: lfs.ErrCodeMessage{
				Code:    1,
				Message: err.Error(),
			},
		}

		var err error
		raw, err = json.Marshal(initResp)
		if err != nil {
			slog.ErrorContext(ctx, "encoding init response", slog.String("error", err.Error()))
			return
		}
	}

	if _, err := action.out.Write(withNewline(raw)); err != nil {
		slog.ErrorContext(ctx, "writing init response", slog.String("error", err.Error()))
	}
	slog.DebugContext(ctx, "wrote init resonse", slog.String("response", string(raw)))
}

func (action *GitLFS) writeProgress(ctx context.Context, oid string, soFar, sinceLast int) {
	log := slog.With(slog.String("event", string(lfs.UploadEvent)), slog.String("oid", oid))

	progressResp := lfs.ProgressResponse{
		Event:          lfs.ProgessEvent,
		Oid:            oid,
		BytesSoFar:     soFar,
		BytesSinceLast: sinceLast,
	}

	raw, err := json.Marshal(progressResp)
	if err != nil {
		log.ErrorContext(ctx, "encoding progress response", slog.String("error", err.Error()))
		return
	}

	if _, err := action.out.Write(withNewline(raw)); err != nil {
		log.ErrorContext(ctx, "writing progress response", slog.String("error", err.Error()))
	}
}

func (action *GitLFS) writeTransferResponse(ctx context.Context, oid string, path string, err error) {
	log := slog.With(slog.String("oid", oid))
	log.DebugContext(ctx, "writing transfer response")

	transferResp := lfs.TransferResponse{
		Event: lfs.CompleteEvent,
		Path:  path,
		Oid:   oid,
	}
	if err != nil {
		transferResp.Error = &lfs.ErrCodeMessage{
			Code:    1,
			Message: err.Error(),
		}
	}

	raw, err := json.Marshal(transferResp)
	if err != nil {
		log.ErrorContext(ctx, "encoding transfer response", slog.String("error", err.Error()))
		return
	}

	if _, err := action.out.Write(withNewline(raw)); err != nil {
		log.ErrorContext(ctx, "writing transfer response", slog.String("error", err.Error()))
	}

	log.DebugContext(ctx, "wrote transfer response", slog.String("response", string(raw)))
}

// readLine
// TODO: consider making this generic.
func (action *GitLFS) readLine() ([]byte, error) {
	ok := action.in.Scan()
	switch {
	case !ok && action.in.Err() != nil:
		return nil, fmt.Errorf("reading single command from git-lfs: %w", action.in.Err())
	case !ok:
		// EOF
		return nil, nil
	default:
		return action.in.Bytes(), nil
	}
}

func withNewline(line []byte) []byte {
	return append(line, []byte("\n")...)
}
