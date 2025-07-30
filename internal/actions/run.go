package actions

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/act3-ai/gitoci/internal/cmd"
	"github.com/act3-ai/gitoci/internal/ociutil"
	"github.com/act3-ai/gitoci/internal/ociutil/model"
	"github.com/go-git/go-git/v5"
	"oras.land/oras-go/v2/content/file"
)

// GitOCI represents the base action
type GitOCI struct {
	// TODO: Could be dangerous when storing in struct like this... mutex?
	batcher cmd.BatchReadWriter

	// local repository
	gitDir    string
	localRepo *git.Repository

	// OCI remote
	name   string // may have same value as address
	addess string
	remote model.Modeler

	Option

	version string
}

// NewGitOCI creates a new Tool with default values
func NewGitOCI(in io.Reader, out io.Writer, gitDir, shortname, address, version string) *GitOCI {
	return &GitOCI{
		batcher: cmd.NewBatcher(in, out),
		gitDir:  gitDir,
		name:    shortname,
		addess:  strings.TrimPrefix(address, "oci://"),
		version: version,
	}
}

// Runs the Hello action
func (action *GitOCI) Run(ctx context.Context) error {
	// TODO: This is a bit early, but sync.Once seems too much
	// TODO: The next 5 "sections" are alot of setup that should be condensed
	gt, err := ociutil.NewGraphTarget(ctx, action.addess)
	if err != nil {
		return fmt.Errorf("initializing remote graph target: %w", err)
	}

	action.localRepo, err = git.PlainOpen(action.gitDir)
	if err != nil {
		return fmt.Errorf("opening local repository: %w", err)
	}

	tmpDir := os.TempDir()
	fstorePath, err := os.MkdirTemp(tmpDir, "GitOCI-fstore-*")
	if err != nil {
		return fmt.Errorf("creating temporary directory for intermediate OCI file store: %w", err)
	}

	fstore, err := file.New(fstorePath)
	if err != nil {
		return fmt.Errorf("initializing OCI filestore: %w", err)
	}
	defer fstore.Close()

	action.remote = model.NewModeler(fstore, gt)

	// first command is always "capabilities"
	c, err := action.batcher.Read(ctx)
	switch {
	case err != nil:
		return fmt.Errorf("reading initial command: %w", err)
	case c.Cmd != cmd.Capabilities:
		return fmt.Errorf("unexpected first command %s, expected 'capabilities'", c.Cmd)
	default:
		if err := action.capabilities(ctx); err != nil {
			return err
		}
	}

	var done bool
	for !done {
		c, err := action.batcher.Read(ctx)
		if err != nil {
			return fmt.Errorf("reading next line: %w", err)
		}

		switch c.Cmd {
		case cmd.Done:
			done = true
		case cmd.Empty:
			continue
		case cmd.Capabilities:
			// Git shouldn't need to do this again, but let's be safe
			if err := action.capabilities(ctx); err != nil {
				return fmt.Errorf("running capabilities command: %w", err)
			}
		case cmd.Option:
			if err := action.option(ctx, c); err != nil {
				return fmt.Errorf("running option command: %w", err)
			}
		case cmd.List:
			if err := action.list(ctx, (c.SubCmd == cmd.ListForPush)); err != nil {
				return fmt.Errorf("running list command: %w", err)
			}
		case cmd.Push:
			// TODO: we shouldn't fully push to the remote until all push batches are resolved locally
			batch, err := action.batcher.ReadBatch(ctx)
			if err != nil {
				return fmt.Errorf("reading push batch: %w", err)
			}
			fullBatch := append([]cmd.Git{c}, batch...)

			if err := action.push(ctx, fullBatch); err != nil {
				return fmt.Errorf("running push command: %w", err)
			}
		case cmd.Fetch:
			batch, err := action.batcher.ReadBatch(ctx)
			if err != nil {
				return fmt.Errorf("reading fetch batch: %w", err)
			}
			fullBatch := append([]cmd.Git{c}, batch...)

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
