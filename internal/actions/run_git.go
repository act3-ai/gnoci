// Package actions holds actions called by the root git-remote-oci command.
package actions

import (
	"context"
	"errors"
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
	gittypes "github.com/act3-ai/gnoci/pkg/protocol/git"
	"github.com/act3-ai/gnoci/pkg/protocol/git/comms"
	"github.com/act3-ai/go-common/pkg/config"
)

// Git represents the base action.
type Git struct {
	version   string
	apiScheme *runtime.Scheme
	// ConfigFiles contains a list of potential configuration file locations.
	ConfigFiles []string

	comm comms.Communicator

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
		comm:        comms.NewCommunicator(in, out),
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
	c, err := action.comm.LookAhead()
	switch {
	case errors.Is(err, gittypes.ErrEndOfInput):
		return true, nil
	case errors.Is(err, gittypes.ErrEmptyRequest):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("command look ahead: %w", err)
	}
	slog.DebugContext(ctx, "read git request", slog.String("request", string(c)))

	// Disregard defined ordering of commands sent by Git, staying robust
	// to potential changes
	switch c {
	case gittypes.Capabilities:
		if err := cmd.HandleCapabilities(ctx, action.comm); err != nil {
			return false, fmt.Errorf("handling capabilities request: %w", err)
		}
	case gittypes.Options:
		if err := cmd.HandleOption(ctx, action.comm); err != nil {
			return false, fmt.Errorf("handling option request: %w", err)
		}
	case gittypes.List:
		if err := action.handleList(ctx); err != nil {
			return false, fmt.Errorf("handling list request: %w", err)
		}
	case gittypes.Push:
		if err := action.handlePush(ctx); err != nil {
			return false, fmt.Errorf("handling push request batch: %w", err)
		}
	case gittypes.Fetch:
		if err := action.handleFetch(ctx); err != nil {
			return false, fmt.Errorf("handling fetch request batch: %w", err)
		}
	default:
		return false, fmt.Errorf("%w: %s", gittypes.ErrUnsupportedRequest, c)
	}

	return false, nil
}

// localRepo opens the local repository if it hasn't been opened already.
func (action *Git) localRepo(ctx context.Context) (git.Repository, error) {
	if action.local == nil {
		if action.gitDir == "" {
			return nil, fmt.Errorf("action.gitDir not defined, unable to open local repository")
		}

		slog.DebugContext(ctx, "opening local repository")
		r, err := gogit.PlainOpen(action.gitDir)
		if err != nil {
			return nil, fmt.Errorf("opening local repository: %w", err)
		}
		action.local = git.NewRepository(r)

	}

	return action.local, nil
}

func (action *Git) handleList(ctx context.Context) error {
	var local git.Repository
	var err error
	slog.DebugContext(ctx, "handleList git directory", slog.String("gitDir", action.gitDir))
	if action.gitDir != "" {
		local, err = action.localRepo(ctx)
		if err != nil {
			return err
		}
	}

	_, err = action.remote.FetchOrDefault(ctx)
	if err != nil {
		return err
	}

	if err := cmd.HandleList(ctx, local, action.remote, action.comm); err != nil {
		return fmt.Errorf("running list command: %w", err)
	}

	return nil
}

func (action *Git) handlePush(ctx context.Context) error {
	// TODO: should we not fully push to the remote until all push batches are resolved locally? Just push the packs?
	local, err := action.localRepo(ctx)
	if err != nil {
		return err
	}

	_, err = action.remote.FetchOrDefault(ctx)
	if err != nil {
		return err
	}

	if err := cmd.HandlePush(ctx, local, action.gitDir, action.remote, action.comm); err != nil {
		return fmt.Errorf("running push commands: %w", err)
	}

	return nil
}

func (action *Git) handleFetch(ctx context.Context) error {
	local, err := action.localRepo(ctx)
	if err != nil {
		return err
	}

	if err := cmd.HandleFetch(ctx, local, action.remote, action.comm); err != nil {
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
