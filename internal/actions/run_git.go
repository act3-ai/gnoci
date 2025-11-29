// Package actions holds actions called by the root git-remote-oci command.
package actions

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"k8s.io/apimachinery/pkg/runtime"
	"oras.land/oras-go/v2/registry"

	"github.com/act3-ai/gnoci/internal/cmd"
	"github.com/act3-ai/gnoci/internal/git"
	"github.com/act3-ai/gnoci/internal/model"
	"github.com/act3-ai/gnoci/internal/ociutil"
	"github.com/act3-ai/gnoci/pkg/apis"
	"github.com/act3-ai/gnoci/pkg/apis/gnoci.act3-ai.io/v1alpha1"
	"github.com/act3-ai/go-common/pkg/config"
)

// Git represents the base action.
type Git struct {
	version   string
	apiScheme *runtime.Scheme
	// ConfigFiles contains a list of potential configuration file locations.
	ConfigFiles []string
	// TODO: Could be dangerous when storing in struct like this... mutex?
	batcher cmd.BatchReadWriter

	// local repository
	gitDir string
	local  git.Repository

	// OCI remote
	name    string // may have same value as address
	address string
	remote  model.Modeler
}

// NewGit creates a new Tool with default values.
func NewGit(in io.Reader, out io.Writer, gitDir, shortname, address, version string, cfgFiles []string) *Git {
	return &Git{
		version:     version,
		apiScheme:   apis.NewScheme(),
		ConfigFiles: cfgFiles,
		batcher:     cmd.NewBatcher(in, out),
		gitDir:      gitDir,
		name:        shortname,
		address:     strings.TrimPrefix(address, "oci://"),
	}
}

// Run runs the the primary git-remote-oci action.
func (action *Git) Run(ctx context.Context) error {
	cfg, err := action.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("getting configuration: %w", err)
	}

	parsedRef, err := registry.ParseReference(action.address)
	if err != nil {
		return fmt.Errorf("invalid reference %s: %w", action.address, err)
	}

	gt, fstorePath, fstore, err := initRemoteConn(ctx, parsedRef, repoOptsFromConfig(parsedRef.Host(), cfg))
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

	action.remote = model.NewModeler(parsedRef, fstore, gt)

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
func (action *Git) handleCmd(ctx context.Context) (bool, error) {
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
func (action *Git) localRepo() (git.Repository, error) {
	if action.local == nil {
		if action.gitDir == "" {
			return nil, fmt.Errorf("action.gitDir not defined, unable to open local repository")
		}

		r, err := gogit.PlainOpen(action.gitDir)
		if err != nil {
			return nil, fmt.Errorf("opening local repository: %w", err)
		}
		action.local = git.NewRepository(r)

	}

	return action.local, nil
}

func (action *Git) handleList(ctx context.Context, gc cmd.Git) error {
	var local git.Repository
	var err error
	if (gc.SubCmd == cmd.ListForPush) && action.gitDir != "" {
		local, err = action.localRepo()
		if err != nil {
			return err
		}
	}

	_, err = action.remote.FetchOrDefault(ctx)
	if err != nil {
		return err
	}

	if err := cmd.HandleList(ctx, local, action.remote, (gc.SubCmd == cmd.ListForPush), gc, action.batcher); err != nil {
		return fmt.Errorf("running list command: %w", err)
	}

	return nil
}

func (action *Git) handlePush(ctx context.Context, gc cmd.Git) error {
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

	_, err = action.remote.FetchOrDefault(ctx)
	if err != nil {
		return err
	}

	if err := cmd.HandlePush(ctx, local, action.gitDir, action.remote, fullBatch, action.batcher); err != nil {
		return fmt.Errorf("running push commands: %w", err)
	}

	return nil
}

func (action *Git) handleFetch(ctx context.Context, gc cmd.Git) error {
	batch, err := action.batcher.ReadBatch(ctx)
	if err != nil {
		return fmt.Errorf("reading fetch batch: %w", err)
	}
	fullBatch := append([]cmd.Git{gc}, batch...)

	local, err := action.localRepo()
	if err != nil {
		return err
	}

	if err := cmd.HandleFetch(ctx, local, action.remote, fullBatch, action.batcher); err != nil {
		return fmt.Errorf("running fetch command: %w", err)
	}

	return nil
}

// GetScheme returns the runtime scheme used for configuration file loading.
func (action *Git) GetScheme() *runtime.Scheme {
	return action.apiScheme
}

// GetConfig loads Configuration using the current git-remote-oci options.
func (action *Git) GetConfig(ctx context.Context) (c *v1alpha1.Configuration, err error) {
	c = &v1alpha1.Configuration{}

	slog.DebugContext(ctx, "searching for configuration files", slog.Any("cfgFiles", action.ConfigFiles))

	err = config.Load(slog.Default(), action.GetScheme(), c, action.ConfigFiles)
	if err != nil {
		return c, fmt.Errorf("loading configuration: %w", err)
	}

	defer slog.DebugContext(ctx, "using config", slog.Any("configuration", c))

	return c, nil
}

func repoOptsFromConfig(host string, cfg *v1alpha1.Configuration) *ociutil.RepositoryOptions {
	repoOpts := &ociutil.RepositoryOptions{
		UserAgent: ociutil.GitUserAgent,
	}

	regCfg, ok := cfg.RegistryConfig.Registries[host]
	if ok {
		repoOpts.PlainHTTP = regCfg.PlainHTTP
		repoOpts.NonCompliant = regCfg.NonCompliant
	}

	return repoOpts
}
