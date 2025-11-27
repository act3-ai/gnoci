// Package git provides interfaces go-git concrete types.
package git

import (
	"context"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage"
)

// Repository represents a git repository.
//
// An interface for the [gogit.Repository] concrete type.
type Repository interface {
	// Storer returns the underlying file storage interface.
	//
	// An extension of [gogit.Repository].
	Storer() storage.Storer

	// BlobObject returns a Blob with the given hash. If not found
	// plumbing.ErrObjectNotFound is returned.
	BlobObject(h plumbing.Hash) (*object.Blob, error)

	// BlobObjects returns an unsorted BlobIter with all the blobs in the repository.
	BlobObjects() (*object.BlobIter, error)

	// Branch return a Branch if exists.
	Branch(name string) (*config.Branch, error)

	// Branches returns all the References that are Branches.
	Branches() (storer.ReferenceIter, error)

	// CommitObject return a Commit with the given hash. If not found
	// plumbing.ErrObjectNotFound is returned.
	CommitObject(h plumbing.Hash) (*object.Commit, error)

	// CommitObjects returns an unsorted CommitIter with all the commits in the repository.
	CommitObjects() (object.CommitIter, error)

	// Config returns the repository config. In a filesystem backed repository this
	// means read the `.git/config`.
	Config() (*config.Config, error)

	// ConfigScoped returns the repository config, merged with requested scope and
	// lower. For example if, config.GlobalScope is given the local and global config
	// are returned merged in one config value.
	ConfigScoped(scope config.Scope) (*config.Config, error)

	// CreateBranch creates a new Branch.
	CreateBranch(c *config.Branch) error

	// CreateRemote creates a new remote.
	CreateRemote(c *config.RemoteConfig) (*gogit.Remote, error)

	// CreateRemoteAnonymous creates a new anonymous remote. c.Name must be "anonymous".
	// It's used like 'git fetch git@github.com:src-d/go-gogit.git master:master'.
	CreateRemoteAnonymous(c *config.RemoteConfig) (*gogit.Remote, error)

	// CreateTag creates a tag. If opts is included, the tag is an annotated tag,
	// otherwise a lightweight tag is created.
	CreateTag(name string, hash plumbing.Hash, opts *gogit.CreateTagOptions) (*plumbing.Reference, error)

	// DeleteBranch delete a Branch from the repository and delete the config.
	DeleteBranch(name string) error

	// DeleteObject deletes an object from a repository.
	// The type conveniently matches PruneHandler.
	DeleteObject(hash plumbing.Hash) error

	// DeleteRemote delete a remote from the repository and delete the config.
	DeleteRemote(name string) error

	// DeleteTag deletes a tag from the repository.
	DeleteTag(name string) error

	// Fetch fetches references along with the objects necessary to complete
	// their histories, from the remote named as FetchOptions.RemoteName.
	//
	// Returns nil if the operation is successful, NoErrAlreadyUpToDate if there are
	// no changes to be fetched, or an error.
	Fetch(o *gogit.FetchOptions) error

	// FetchContext fetches references along with the objects necessary to complete
	// their histories, from the remote named as FetchOptions.RemoteName.
	//
	// Returns nil if the operation is successful, NoErrAlreadyUpToDate if there are
	// no changes to be fetched, or an error.
	//
	// The provided Context must be non-nil. If the context expires before the
	// operation is complete, an error is returned. The context only affects the
	// transport operations.
	FetchContext(ctx context.Context, o *gogit.FetchOptions) error

	// Grep performs grep on a repository.
	Grep(opts *gogit.GrepOptions) ([]gogit.GrepResult, error)

	// Head returns the reference where HEAD is pointing to.
	Head() (*plumbing.Reference, error)

	// Log returns the commit history from the given LogOptions.
	Log(o *gogit.LogOptions) (object.CommitIter, error)

	// Merge merges the reference branch into the current branch.
	//
	// If the merge is not possible (or supported) returns an error without changing
	// the HEAD for the current branch. Possible errors include:
	//   - The merge strategy is not supported.
	//   - The specific strategy cannot be used (e.g. using FastForwardMerge when one is not possible).
	Merge(ref plumbing.Reference, opts gogit.MergeOptions) error

	// Notes returns all the References that are notes. For more information:
	// https://git-scm.com/docs/git-notes.
	Notes() (storer.ReferenceIter, error)

	// Object returns an Object with the given hash. If not found
	// plumbing.ErrObjectNotFound is returned.
	Object(t plumbing.ObjectType, h plumbing.Hash) (object.Object, error)

	// Objects returns an unsorted ObjectIter with all the objects in the repository.
	Objects() (*object.ObjectIter, error)

	// Prune removes loose objects from storage.
	Prune(opt gogit.PruneOptions) error

	// Push performs a push to the remote. Returns NoErrAlreadyUpToDate if
	// the remote was already up-to-date, from the remote named as
	// FetchOptions.RemoteName.
	Push(o *gogit.PushOptions) error

	// PushContext performs a push to the remote. Returns NoErrAlreadyUpToDate if
	// the remote was already up-to-date, from the remote named as
	// FetchOptions.RemoteName.
	//
	// The provided Context must be non-nil. If the context expires before the
	// operation is complete, an error is returned. The context only affects the
	// transport operations.
	PushContext(ctx context.Context, o *gogit.PushOptions) error

	// Reference returns the reference for a given reference name. If resolved is
	// true, any symbolic reference will be resolved.
	Reference(name plumbing.ReferenceName, resolved bool) (*plumbing.Reference, error)

	// References returns an unsorted ReferenceIter for all references.
	References() (storer.ReferenceIter, error)

	// Remote return a remote if exists.
	Remote(name string) (*gogit.Remote, error)

	// Remotes returns a list with all the remotes.
	Remotes() ([]*gogit.Remote, error)

	// RepackObjects creates a new packfile from existing packfiles.
	RepackObjects(cfg *gogit.RepackConfig) (err error)

	// ResolveRevision resolves revision to corresponding hash. It will always
	// resolve to a commit hash, not a tree or annotated tag.
	//
	// Implemented resolvers : HEAD, branch, tag, heads/branch, refs/heads/branch,
	// refs/tags/tag, refs/remotes/origin/branch, refs/remotes/origin/HEAD, tilde
	// and caret (HEAD~1, master~^, tag~2, ref/heads/master~1, ...),
	// selection by text (HEAD^{/fix nasty bug}), hash (prefix and full).
	ResolveRevision(in plumbing.Revision) (*plumbing.Hash, error)

	// SetConfig marshalls and writes the repository config. In a filesystem backed
	// repository this means write the `.git/config`. This function should be called
	// with the result of `Repository.Config` and never with the output of
	// `Repository.ConfigScoped`.
	SetConfig(cfg *config.Config) error

	// Tag returns a tag from the repository.
	//
	// If you want to check to see if the tag is an annotated tag, you can call
	// TagObject on the hash of the reference in ForEach:
	//
	//	ref, err := r.Tag("v0.1.0")
	//	if err != nil {
	//	  // Handle error
	//	}
	//
	//	obj, err := r.TagObject(ref.Hash())
	//	switch err {
	//	case nil:
	//	  // Tag object present
	//	case plumbing.ErrObjectNotFound:
	//	  // Not a tag object
	//	default:
	//	  // Some other error
	//	}.
	Tag(name string) (*plumbing.Reference, error)

	// TagObject returns a Tag with the given hash. If not found
	// plumbing.ErrObjectNotFound is returned. This method only returns
	// annotated Tags, no lightweight Tags.
	TagObject(h plumbing.Hash) (*object.Tag, error)

	// TagObjects returns a unsorted TagIter that can step through all of the annotated
	// tags in the repository.
	TagObjects() (*object.TagIter, error)

	// Tags returns all the tag References in a repository.
	//
	// If you want to check to see if the tag is an annotated tag, you can call
	// TagObject on the hash Reference passed in through ForEach:
	//
	//	iter, err := r.Tags()
	//	if err != nil {
	//	  // Handle error
	//	}
	//
	//	if err := iter.ForEach(func (ref *plumbing.Reference) error {
	//	  obj, err := r.TagObject(ref.Hash())
	//	  switch err {
	//	  case nil:
	//	    // Tag object present
	//	  case plumbing.ErrObjectNotFound:
	//	    // Not a tag object
	//	  default:
	//	    // Some other error
	//	    return err
	//	  }
	//	}); err != nil {
	//	  // Handle outer iterator error
	//	}.
	Tags() (storer.ReferenceIter, error)

	// TreeObject return a Tree with the given hash. If not found
	// plumbing.ErrObjectNotFound is returned.
	TreeObject(h plumbing.Hash) (*object.Tree, error)

	// TreeObjects returns an unsorted TreeIter with all the trees in the repository.
	TreeObjects() (*object.TreeIter, error)

	// Worktree returns a worktree based on the given fs, if nil the default
	// worktree will be used.
	Worktree() (*gogit.Worktree, error)
}

// Repo implements [Repository].
type Repo struct {
	*gogit.Repository
}

// Storer returns the underlying file storage interface.
//
// An extension of [gogit.Repository].
func (r *Repo) Storer() storage.Storer {
	return r.Repository.Storer
}

// NewRepository wraps a [gogit.Repository].
func NewRepository(repo *gogit.Repository) Repository {
	return &Repo{repo}
}
