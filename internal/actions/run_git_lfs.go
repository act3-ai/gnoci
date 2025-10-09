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

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"

	"github.com/act3-ai/gnoci/internal/lfs"
	"github.com/act3-ai/gnoci/internal/ociutil/model"
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

	var fstorePath string
	var fstore *file.Store
	action.gt, fstorePath, fstore, err = initRemoteConn(ctx, initReq.Remote)
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

	action.remote = model.NewLFSModeler(initReq.Remote, fstore, action.gt)

	if err := action.remote.Fetch(ctx); err != nil {
		return fmt.Errorf("fetching base git metadata: %w", err)
	}

	if err := action.remote.FetchLFSOrDefault(ctx); err != nil {
		return err
	}

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
	return errors.New("not implemented")
}

func (action *GitLFS) runUpload(ctx context.Context) error {
	// TODO: by their protocol spec, they block until the transfer is complete
	// this is far less than ideal for us. Unfortunately, we may not be able
	// to do this concurrently as we would not be able to update the LFS
	// manifest across processes, as their concurrency is done with multiple
	// invocations of our tool.
	// TODO: if the above is true, we need to refactor such that we write err
	// responses to git-lfs within the goroutines spun up by model.LFSModeler.PushLFS
	for {
		line, err := action.readLine()
		if err != nil {
			return err
		}

		var transferReq lfs.TransferRequest
		if err := json.Unmarshal(line, &transferReq); err != nil {
			// TODO: is it possible to write back to git-lfs here or is this fatal?
			return fmt.Errorf("decoding TransferRequest: %w", err)
		}

		// TODO: validate the transfer request

		if transferReq.Event == lfs.TerminateEvent {
			break
		}

		if _, err := action.remote.AddLFSFile(ctx, transferReq.Path); err != nil {
			action.writeResponse(ctx, transferReq.Oid, transferReq.Path, fmt.Errorf("preparing git-lfs file for transfer: %w", err))
			continue
		}

		// HACK: is this necessary? per the spec, it seems like it is...
		action.writeProgress(ctx, transferReq.Oid, 1, 1)

		// notify completion
		action.writeResponse(ctx, transferReq.Oid, "", nil)
	}

	_, err := action.remote.PushLFS(ctx)
	if err != nil {
		return fmt.Errorf("pushing LFS to OCI: %w", err)
	}

	return nil
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

	if _, err := action.out.Write(raw); err != nil {
		log.ErrorContext(ctx, "writing progress response", slog.String("error", err.Error()))
	}
}

func (action *GitLFS) writeResponse(ctx context.Context, oid string, path string, err error) {
	log := slog.With(slog.String("event", string(lfs.UploadEvent)), slog.String("oid", oid))

	transferResp := lfs.TransferResponse{
		Event: lfs.CompleteEvent,
		Path:  path,
		Oid:   oid,
	}

	// TODO: is this necessary?
	// if path != "" {
	// 	transferResp.Path = path
	// }

	if err != nil {
		transferResp.Error = lfs.ErrCodeMessage{
			Code:    1,
			Message: err.Error(),
		}
	}

	raw, err := json.Marshal(transferResp)
	if err != nil {
		log.ErrorContext(ctx, "encoding transfer response", slog.String("error", err.Error()))
		return
	}

	if _, err := action.out.Write(raw); err != nil {
		log.ErrorContext(ctx, "writing transfer response", slog.String("error", err.Error()))
	}
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
