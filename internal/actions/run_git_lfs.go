// Package actions holds actions called by the root git-remote-oci command.
package actions

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/act3-ai/gnoci/internal/cmd"
	"github.com/act3-ai/gnoci/internal/ociutil"
	"github.com/act3-ai/gnoci/internal/ociutil/model"
	"github.com/go-git/go-git/v5"
	"oras.land/oras-go/v2/content/file"
)

// GitLFS represents the base action.
type GitLFS struct {
	// TODO: Could be dangerous when storing in struct like this... mutex?
	batcher cmd.BatchReadWriter

	// local repository
	gitDir string
	local  *git.Repository

	// OCI remote
	name   string // may have same value as address
	addess string
	remote model.Modeler

	version string
}

// NewGitLFS creates a new Tool with default values.
func NewGitLFS(in io.Reader, out io.Writer, gitDir, shortname, address, version string) *GitLFS {
	return &GitLFS{
		batcher: cmd.NewBatcher(in, out),
		gitDir:  gitDir,
		name:    shortname,
		addess:  strings.TrimPrefix(address, "oci://"),
		version: version,
	}
}

// Run runs the the primary git-remote-oci action.
func (action *GitLFS) Run(ctx context.Context) error {
	// TODO: This is a bit early, but sync.Once seems too much
	// TODO: The next 5 "sections" are alot of setup that should be condensed
	gt, err := ociutil.NewGraphTarget(ctx, action.addess)
	if err != nil {
		return fmt.Errorf("initializing remote graph target: %w", err)
	}

	tmpDir := os.TempDir()
	fstorePath, err := os.MkdirTemp(tmpDir, "GnOCI-fstore-*")
	if err != nil {
		return fmt.Errorf("creating temporary directory for intermediate OCI file store: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(fstorePath); err != nil {
			slog.ErrorContext(ctx, "cleaning up temporary files", slog.String("error", err.Error()))
		}
	}()

	fstore, err := file.New(fstorePath)
	if err != nil {
		return fmt.Errorf("initializing OCI filestore: %w", err)
	}
	defer func() {
		if err := fstore.Close(); err != nil {
			slog.ErrorContext(ctx, "closing OCI file store", slog.String("error", err.Error()))
		}
	}()

	action.remote = model.NewModeler(fstore, gt)

	var done bool
	for !done {
		done, err = action.handleCmd(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// handleCmd returns true, nil if command handling is complete.
func (action *GitLFS) handleCmd(ctx context.Context) (bool, error) {
	gc, err := action.batcher.Read(ctx)
	if err != nil {
		return false, fmt.Errorf("reading next line: %w", err)
	}

	switch gc.Cmd {
	case cmd.Done:
		return true, nil
	case cmd.Empty:
		return false, nil
	case cmd.Capabilities:
		// Git should only need this once on the first cmd, but here is safer
		if err := cmd.HandleCapabilities(ctx, gc, action.batcher); err != nil {
			return false, fmt.Errorf("handling capabilities command: %w", err)
		}
	case cmd.Option:
		if err := cmd.HandleOption(ctx, gc, action.batcher); err != nil {
			return false, fmt.Errorf("handling option command: %w", err)
		}
	case cmd.List:
		if err := action.handleList(ctx, gc); err != nil {
			return false, fmt.Errorf("handling list command: %w", err)
		}
	case cmd.Push:
		if err := action.handlePush(ctx, gc); err != nil {
			return false, fmt.Errorf("handling push command batch: %w", err)
		}
	case cmd.Fetch:
		if err := action.handleFetch(ctx, gc); err != nil {
			return false, fmt.Errorf("handling fetch command batch: %w", err)
		}
	default:
		return false, fmt.Errorf("%w: %s", cmd.ErrUnsupportedCommand, gc.String())
	}

	return false, nil
}

// localRepo opens the local repository if it hasn't been opened already.
func (action *GitLFS) localRepo() (*git.Repository, error) {
	if action.local == nil {
		if action.gitDir == "" {
			return nil, fmt.Errorf("action.gitDir not defined, unable to open local repository")
		}
		var err error
		action.local, err = git.PlainOpen(action.gitDir)
		if err != nil {
			return nil, fmt.Errorf("opening local repository: %w", err)
		}
	}

	return action.local, nil
}

func (action *GitLFS) handleList(ctx context.Context, gc cmd.Git) error {
	var local *git.Repository
	var err error
	if (gc.SubCmd == cmd.ListForPush) && action.gitDir != "" {
		local, err = action.localRepo()
		if err != nil {
			return err
		}
	}

	if err := action.remote.FetchOrDefault(ctx, action.addess); err != nil {
		return err
	}

	if err := cmd.HandleList(ctx, local, action.remote, (gc.SubCmd == cmd.ListForPush), gc, action.batcher); err != nil {
		return fmt.Errorf("running list command: %w", err)
	}

	return nil
}

func (action *GitLFS) handlePush(ctx context.Context, gc cmd.Git) error {
	// TODO: we shouldn't fully push to the remote until all push batches are resolved locally
	batch, err := action.batcher.ReadBatch(ctx)
	if err != nil {
		return fmt.Errorf("reading push batch: %w", err)
	}
	fullBatch := append([]cmd.Git{gc}, batch...)

	local, err := action.localRepo()
	if err != nil {
		return err
	}

	if err := action.remote.FetchOrDefault(ctx, action.addess); err != nil {
		return err
	}

	if err := cmd.HandlePush(ctx, local, action.gitDir, action.remote, action.addess, fullBatch, action.batcher); err != nil {
		return fmt.Errorf("running push commands: %w", err)
	}

	return nil
}

func (action *GitLFS) handleFetch(ctx context.Context, gc cmd.Git) error {
	batch, err := action.batcher.ReadBatch(ctx)
	if err != nil {
		return fmt.Errorf("reading fetch batch: %w", err)
	}
	fullBatch := append([]cmd.Git{gc}, batch...)

	local, err := action.localRepo()
	if err != nil {
		return err
	}

	if err := cmd.HandleFetch(ctx, local, action.remote, action.addess, fullBatch, action.batcher); err != nil {
		return fmt.Errorf("running fetch command: %w", err)
	}

	return nil
}
