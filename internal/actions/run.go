package actions

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/act3-ai/gnoci/internal/cmd"
	"github.com/act3-ai/gnoci/internal/ociutil"
	"github.com/act3-ai/gnoci/internal/ociutil/model"
	"github.com/go-git/go-git/v5"
	"oras.land/oras-go/v2/content/file"
)

// GnOCI represents the base action.
type GnOCI struct {
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

// NewGnOCI creates a new Tool with default values.
func NewGnOCI(in io.Reader, out io.Writer, gitDir, shortname, address, version string) *GnOCI {
	return &GnOCI{
		batcher: cmd.NewBatcher(in, out),
		gitDir:  gitDir,
		name:    shortname,
		addess:  strings.TrimPrefix(address, "oci://"),
		version: version,
	}
}

// Run runs the the primary git-remote-oci action.
func (action *GnOCI) Run(ctx context.Context) error {
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
	defer os.RemoveAll(fstorePath)

	fstore, err := file.New(fstorePath)
	if err != nil {
		return fmt.Errorf("initializing OCI filestore: %w", err)
	}
	defer fstore.Close()

	action.remote = model.NewModeler(fstore, gt)

	var done bool
	for !done {
		gc, err := action.batcher.Read(ctx)
		if err != nil {
			return fmt.Errorf("reading next line: %w", err)
		}

		switch gc.Cmd {
		case cmd.Done:
			done = true
		case cmd.Empty:
			continue
		case cmd.Capabilities:
			// Git should only need this once on the first cmd, but here is safer
			if err := cmd.HandleCapabilities(ctx, gc, action.batcher); err != nil {
				return fmt.Errorf("running capabilities command: %w", err)
			}
		case cmd.Option:
			if err := cmd.HandleOption(ctx, gc, action.batcher); err != nil {
				return fmt.Errorf("running option command: %w", err)
			}
		case cmd.List:
			var local *git.Repository
			var err error
			if (gc.SubCmd == cmd.ListForPush) && action.gitDir != "" {
				local, err = action.localRepo()
				if err != nil {
					return err
				}
			}
			var remote model.Modeler
			if err := action.remote.FetchOrDefault(ctx, action.addess); err != nil {
				return err
			}

			if err := cmd.HandleList(ctx, local, remote, (gc.SubCmd == cmd.ListForPush), gc, action.batcher); err != nil {
				return fmt.Errorf("running list command: %w", err)
			}
		case cmd.Push:
			// TODO: we shouldn't fully push to the remote until all push batches are resolved locally
			batch, err := action.batcher.ReadBatch(ctx)
			if err != nil {
				return fmt.Errorf("reading push batch: %w", err)
			}
			fullBatch := append([]cmd.Git{gc}, batch...)

			if err := action.push(ctx, fullBatch); err != nil {
				return fmt.Errorf("running push command: %w", err)
			}
		case cmd.Fetch:
			batch, err := action.batcher.ReadBatch(ctx)
			if err != nil {
				return fmt.Errorf("reading fetch batch: %w", err)
			}
			fullBatch := append([]cmd.Git{gc}, batch...)

			if err := action.fetch(ctx, fullBatch); err != nil {
				return fmt.Errorf("running fetch command: %w", err)
			}
		default:
			return fmt.Errorf("default case hit")
		}
	}

	// // TODO: Next command is 'list', can be read in a batch
	// slog.InfoContext(ctx, "reading batch")
	// _, err = action.batcher.ReadBatch(ctx)
	// if err != nil {
	// 	return fmt.Errorf("reading batch input: %w", err)
	// }

	// TODO: not a fan of this pattern, but it's nice to catch the errors...
	// ideally we "do our best to cleanup"
	if err := fstore.Close(); err != nil {
		return fmt.Errorf("closing OCI file store: %w", err)
	}

	if err := os.RemoveAll(fstorePath); err != nil {
		return fmt.Errorf("cleaning up temporary files: %w", err)
	}

	return nil
}

// localRepo opens the local repository if it hasn't been opened already.
func (action *GnOCI) localRepo() (*git.Repository, error) {
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
