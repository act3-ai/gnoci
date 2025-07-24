package actions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"oras.land/oras-go/v2/errdef"

	"github.com/act3-ai/gitoci/internal/cmd"
	"github.com/act3-ai/gitoci/pkg/oci"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage/filesystem/dotgit"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// TODO: passing around *git.Repository and oci.ConfigGit may beg for an interface

var ErrRemoteRefNotFound = errors.New("remote reference not found")

type status uint8

const (
	// delete indicates a ref should be removed from the remote
	delete status = 1 << iota
	// ref indicates a ref should be updated in the remote
	ref
	// forward indicates the ref's commit object should be added to the remote
	forward
	// force indicates a force update should be performed
	force
	// rewritten indicates history has been rewritten
	// TODO: necessary?
	// rewritten
)

// TODO: compareRefs is intended for resolving the min set of hashes needed for a thin packfile.
func compareRefs(localRepo *dotgit.DotGit, localRef, remoteRef *plumbing.Reference) (status, error) {
	var s status

	// if local is empty, status += delete
	if localRef == nil {
		s = s | delete
	}

	// TODO
	return 0, fmt.Errorf("not implemented")
}

// func sample(localRepo *git.Repository) error {
// 	tmpRepo := filesystem.NewStorageWithOptions(
// 		osfs.New(os.TempDir()),
// 		cache.NewObjectLRUDefault(),
// 		filesystem.Options{AlternatesFS: localRepo.Storer})
// 	git.InitWithOptions()

// }

// push handles the `push` command.
func (action *GitOCI) push(ctx context.Context, cmds []cmd.Git) error {
	// fetch config
	man, cfg, err := action.fetchMetadata(ctx)
	switch {
	case errors.Is(err, errdef.ErrNotFound):
		slog.InfoContext(ctx, "remote does not exist, starting fresh")
		cfg = oci.ConfigGit{
			Heads: make(map[string]oci.ReferenceInfo, 0),
			Tags:  make(map[string]oci.ReferenceInfo, 0),
		}
	case err != nil:
		return fmt.Errorf("fetching remote metadata: %w", err)
	}

	repo, err := git.PlainOpen(action.gitDir)
	if err != nil {
		return fmt.Errorf("opening local repository: %w", err)
	}
	// use pkg dotgit rather than git so we have access to manage packfiles
	// repo := dotgit.New(osfs.New(action.gitDir))

	// HACK: For now we're creating an packfile of the entire repo,
	// but we'll keep this around for testing
	// resolve state of refs in remote
	for _, c := range cmds {
		l, r, err := parseRefPair(c)
		if err != nil {
			return fmt.Errorf("parsing push command: %w", err)
		}

		local, err := resolveLocal(ctx, repo, l)
		if err != nil {
			return err
		}
		slog.InfoContext(ctx, "resolved local reference", "ref", l.String(), "hash", local.Hash().String())

		remote, err := resolveRemote(ctx, cfg, r)
		if err != nil {
			return err
		}
		slog.InfoContext(ctx, "resolved remote reference", "ref", l.String(), "hash", remote.Hash().String())
	}

	// TODO: resolve common ancestors for thin pack

	// TODO: if not common ancestors (bad object?) then we must pull down everything from OCI, rebuild into a repo, and resolve. OR we could just require the user to force push; isn't this what Git requires anyhow?

	// HACK
	packHash, err := packAll(repo)
	if err != nil {
		return fmt.Errorf("building packfile: %w", err)
	}

	// TODO: hopefully this isn't necessary, and we can open a reader using go-git methods
	packPath := path.Join(action.gitDir, "objects", "pack", fmt.Sprintf("pack-%s.pack", packHash.String()))
	// idxPath := path.Join(action.gitDir, "objects", "pack", fmt.Sprintf("pack-%s.idx", packHash.String()))

	buildOCI(packPath, man, cfg)

	return fmt.Errorf("not implemented")
}

func buildOCI(packPath string, man ocispec.Manifest, cfg oci.ConfigGit) {
	// if no mediatype, we know it's new
	if man.MediaType == "" {
		man.MediaType = ocispec.MediaTypeImageManifest
		man.ArtifactType = oci.ArtifactTypeGitManifest
	}
}

// resolveLocal resolves the hash of a local reference.
func resolveLocal(_ context.Context, repo *git.Repository, ref plumbing.ReferenceName) (*plumbing.Reference, error) {
	localRef, err := repo.Reference(ref, true)
	if err != nil {
		return nil, fmt.Errorf("resolving hash of local reference %s: %w", ref.String(), err)
	}
	return localRef, nil
}

// resolveRemote resolves the hash of a remote reference, if one exists.
func resolveRemote(ctx context.Context, cfg oci.ConfigGit, ref plumbing.ReferenceName) (*plumbing.Reference, error) {
	var ok bool
	var rInfo oci.ReferenceInfo
	switch {
	case ref.IsBranch():
		rInfo, ok = cfg.Heads[ref.String()]
	case ref.IsTag():
		rInfo, ok = cfg.Tags[ref.String()]
	default:
		slog.WarnContext(ctx, "skipping unknown remote reference type", "reference", ref.String())
	}

	if ok {
		return plumbing.NewHashReference(ref, plumbing.NewHash(rInfo.Commit)), nil
	}
	return nil, fmt.Errorf("%w: %s", ErrRemoteRefNotFound, ref.String())
}

// HACK: having trouble creating packfiles, let alone thin packs, so we'll do the entire repo for now. If needed, we can fallback to shelling out and contribute to go-git later.
func packAll(repo *git.Repository) (h plumbing.Hash, err error) {
	err = repo.RepackObjects(&git.RepackConfig{UseRefDeltas: true})
	if err != nil {
		return h, fmt.Errorf("repacking all objects: %w", err)
	}

	pos, ok := repo.Storer.(storer.PackedObjectStorer)
	if !ok {
		return h, fmt.Errorf("repository storer is not a storer.PackedObjectStorer")
	}

	hs, err := pos.ObjectPacks()
	switch {
	case err != nil:
		return h, err

	case len(hs) != 1:
		return h, fmt.Errorf("expected 1 packfile, got %d", len(hs))
	default:
		return hs[0], nil
	}
}

// createPack builds a packfile using a set of hashes.
// TODO: not used
func createPack(repo *git.Repository, hashes []plumbing.Hash) (h plumbing.Hash, err error) {
	// reference implementation: https://github.com/go-git/go-git/blob/v5.16.2/repository.go#L1815
	pfw, ok := repo.Storer.(storer.PackfileWriter)
	if !ok {
		return h, fmt.Errorf("repository storer is not a storer.PackfileWriter")
	}
	wc, err := pfw.PackfileWriter()
	if err != nil {
		return h, fmt.Errorf("initializing packfile writer: %w", err)
	}

	// TODO: What is a ref delta?
	enc := packfile.NewEncoder(wc, repo.Storer, true)
	h, err = enc.Encode(hashes, 10) // default window
	if err != nil {
		return h, fmt.Errorf("encoding packfile: %w", err)
	}
	return h, nil
}

// parseRefPair validates a reference pair, <local>:<remote>, returning the local and remote references respectively.
func parseRefPair(c cmd.Git) (plumbing.ReferenceName, plumbing.ReferenceName, error) {
	if c.Data == nil {
		return "", "", fmt.Errorf("no arguments in push command")
	}

	pair := c.Data[0]
	if pair == "" {
		return "", "", errors.New("empty reference pair string, expected <local>:<remote>")
	}

	s := strings.Split(pair, ":")
	if len(s) != 2 {
		return "", "", fmt.Errorf("failed to split reference pair string, got %s, expected <local>:<remote>", pair)
	}

	return plumbing.ReferenceName(s[0]), plumbing.ReferenceName(s[1]), nil
}
