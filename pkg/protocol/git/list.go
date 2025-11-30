package git

import (
	"fmt"
	"iter"

	"github.com/go-git/go-git/v5/plumbing"
)

// ListRequest is a command received from Git requesting a list of references.
//
// https://git-scm.com/docs/gitremote-helpers#Documentation/gitremote-helpers.txt-list.
type ListRequest struct {
	Cmd     Command
	ForPush bool
}

// Parse decodes request fields ensuring the [ListRequest] is of the correct type, is supported,
// and has a valid value.
//
// Implements [Parsable].
func (r *ListRequest) Parse(fields []string) error {
	if len(fields) != 1 || len(fields) != 2 {
		return fmt.Errorf("%w: invalid fields for list request: got %v", ErrBadRequest, fields)
	}

	cmd := Command(fields[0])
	if cmd != List {
		return fmt.Errorf("%w: got %s, want %s", ErrUnexpectedRequest, cmd, List)
	}
	r.Cmd = cmd

	if len(fields) == 2 {
		if fields[2] != "for-push" {
			return fmt.Errorf("%w: invalid option for list request: got %v", ErrBadRequest, fields)
		}
		r.ForPush = true
	}

	return nil
}

// String condenses [ListRequest] into a string, the raw request received from Git.
func (r *ListRequest) String() string {
	str := string(r.Cmd)
	if r.ForPush {
		str += " for-push"
	}
	return str
}

// ReferenceLister iterates through a set of references.
type ReferenceLister iter.Seq[plumbing.Reference]

// ListRefs creates an [iter.Seq] for a slice of [plumbing.Reference]s.
func ListRefs(refs ...plumbing.Reference) ReferenceLister {
	return func(yield func(plumbing.Reference) bool) {
		for _, item := range refs {
			if !yield(item) {
				return
			}
		}
	}
}
