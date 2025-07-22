package actions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"oras.land/oras-go/v2/errdef"

	"github.com/act3-ai/gitoci/internal/cmd"
	"github.com/act3-ai/gitoci/pkg/oci"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type refComparer struct {
	localRepo    *git.Repository
	remoteConfig oci.ConfigGit
	remoteRepo   *git.Repository

	refs map[string]refPair // full pairs, as provided by Git, to parsed comparer
}

type refPair struct {
	local  *plumbing.Reference
	remote *plumbing.Reference

	status
}

type status uint8

const (
	// delete indicates a ref should be removed from the remote
	delete status = 1 << iota
	// ref indicates a ref should be updated in the remote
	ref
	// pack indicates the ref's commit object should be added to the remote
	pack
	// force indicates a force update should be performed
	force
	// rewritten indicates history has been rewritten
	// TODO: necessary?
	rewritten
)

func newRefComparer(localRepo *git.Repository) (*refComparer, error) {

	return &refComparer{}, nil
}

func (rc *refComparer) Status(pair string, force bool) status {
	// cached
	p, ok := rc.refs[pair]
	if !force && ok {
		return p.status
	}

	rc.compare()
	return p.status
}

func (rc *refComparer) compare() {
	// TODO

	// if local is empty, status += delete

	// if
}

// push handles the `push` command.
func (action *GitOCI) push(ctx context.Context, cmds []cmd.Git) error {
	// fetch config
	var cfg oci.ConfigGit
	var err error
	cfg, err = action.fetchConfig(ctx)
	switch {
	case errors.Is(err, errdef.ErrNotFound):
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

	// resolve state of refs in remote
	for _, c := range cmds {
		l, r, err := parseRefPair(c)
		if err != nil {
			return fmt.Errorf("parsing push command: %w", err)
		}

		// resolve local, empty local indicates deletion from remote
		var localRef *plumbing.Reference
		if l != "" {
			localRef, err = repo.Reference(l, true)
			if err != nil {
				return fmt.Errorf("resolving hash of local reference: %w", err)
			}
		}

		// resolve remote
		var ok bool
		var rInfo oci.ReferenceInfo
		switch {
		case r.IsBranch():
			rInfo, ok = cfg.Heads[r.String()]
		case r.IsTag():
			rInfo, ok = cfg.Tags[r.String()]
		default:
			slog.WarnContext(ctx, "skipping unknown reference type", "reference", r.String())
		}

		var remoteRef *plumbing.Reference
		if ok {
			remoteRef = plumbing.NewHashReference(r, plumbing.NewHash(rInfo.Commit))
		}

	}

	// resolve common ancestors for thin pack

	// if not common ancestors (bad object?) then we must pull down everything from OCI, rebuild into a repo, and resolve

	// resolve commits of local refs
	//   while in the loop, check if were ff-ing, deleting, or we're behind
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
