package actions

import (
	"fmt"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// list handles the `list` command. Lists refs, one per line.
func (action *GitOCI) list() error {

	//TODO: until fetch/push is implemented we'll focus on the output for now.

	localRepo, err := git.PlainOpen(action.gitDir)
	if err != nil {
		return fmt.Errorf("opening local repository: %w", err)
	}

	// TODO: Is there really no other option than an iter?
	rIter, err := localRepo.References()
	if err != nil {
		return fmt.Errorf("getting reference iter: %w", err)
	}

	localHashRefs := make([]string, 0, 1)
	_ = rIter.ForEach(func(r *plumbing.Reference) error {
		localHashRefs = append(localHashRefs,
			fmt.Sprintf("%s %s", r.Hash().String(), r.Name().String()))
		return nil
	})

	if err := action.batcher.WriteBatch(localHashRefs...); err != nil {
		return fmt.Errorf("writing local refs: %w", err)
	}

	return fmt.Errorf("not implemented")
}

// listForPush handles the `list for-push` command.
// Similar to list, except only used to prepare for a push.
// func (action *GitOCI) listForPush() error {

// }
