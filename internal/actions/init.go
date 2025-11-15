package actions

import (
	"context"
	"fmt"
	"os"

	"github.com/act3-ai/gnoci/internal/ociutil"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry"
)

// initRemoteConn initializes intermediary objects used when fetching from or
// pushing to the OCI remote.
//
// It is the caller's responsibility to clean all three return types up.
func initRemoteConn(ctx context.Context, ref registry.Reference, opts *ociutil.RepositoryOptions) (oras.GraphTarget, string, *file.Store, error) {
	gt, err := ociutil.NewGraphTarget(ctx, ref, opts)
	if err != nil {
		return nil, "", nil, fmt.Errorf("initializing remote graph target: %w", err)
	}

	tmpDir := os.TempDir()
	fstorePath, err := os.MkdirTemp(tmpDir, "GnOCI-fstore-*")
	if err != nil {
		return nil, "", nil, fmt.Errorf("creating temporary directory for intermediate OCI file store: %w", err)
	}

	fstore, err := file.New(fstorePath)
	if err != nil {
		return nil, "", nil, fmt.Errorf("initializing OCI filestore: %w", err)
	}

	return gt, fstorePath, fstore, nil
}
