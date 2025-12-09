// Package refcomp provides utilities for comparing local and remote git references.
package refcomp

import (
	"testing"

	"github.com/act3-ai/gnoci/internal/mocks/gitmock"
	"github.com/act3-ai/gnoci/internal/mocks/modelmock"
	"github.com/act3-ai/gnoci/internal/model"
	"github.com/act3-ai/gnoci/pkg/testutils"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewCachedRefComparer(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repoMock := gitmock.NewMockRepository(ctrl)
		modelMock := modelmock.NewMockModeler(ctrl)

		rc := NewCachedRefComparer(repoMock, modelMock)
		assert.NotNil(t, rc)

		rcConcrete, ok := rc.(*refCompareCached)
		assert.True(t, ok)
		assert.NotNil(t, rcConcrete.local)
		assert.NotNil(t, rcConcrete.remote)
		assert.NotNil(t, rcConcrete.refs)
	})
}

func Test_refCompareCached_Compare(t *testing.T) {
	const (
		localBranchName       = "local-foo"
		remoteBranchName      = "remote-foo"
		nonexistantBranchName = "doesnotexist"
	)

	var (
		localBranchRefName       = plumbing.NewBranchReferenceName(localBranchName)
		remoteBranchRefName      = plumbing.NewBranchReferenceName(remoteBranchName)
		nonexistantBranchRefName = plumbing.NewBranchReferenceName(nonexistantBranchName)
	)

	t.Run("Cached Reference Comparison", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repoMock := gitmock.NewMockRepository(ctrl)
		modelMock := modelmock.NewMockModeler(ctrl)

		localHash := plumbing.ComputeHash(plumbing.CommitObject, []byte("foo"))
		remoteHash := plumbing.ComputeHash(plumbing.CommitObject, []byte("bar"))

		expectedRefPair := RefPair{
			Local:  plumbing.NewHashReference(localBranchRefName, localHash),
			Remote: plumbing.NewHashReference(remoteBranchRefName, remoteHash),
			Status: StatusUpdateRef | StatusAddCommit,
		}

		rc := &refCompareCached{
			local:  repoMock,
			remote: modelMock,
			refs: map[plumbing.ReferenceName]RefPair{
				remoteBranchRefName: expectedRefPair,
			},
		}

		rp, err := rc.Compare(t.Context(), false, localBranchRefName, remoteBranchRefName)
		assert.Nil(t, err)
		assert.Equal(t, expectedRefPair, rp)
	})

	type (
		wantFunc  func(t *testing.T, refPair RefPair, err error)
		setupFunc func(t *testing.T,
			repoBuilder *testutils.RepoBuilder,
			repoMock *gitmock.MockRepository,
			modelMock *modelmock.MockModeler) wantFunc
	)

	tests := []struct {
		name string
		// setup
		setupFn setupFunc
		// execute
		force      bool
		localName  plumbing.ReferenceName
		remoteName plumbing.ReferenceName
	}{
		{name: "Remote DNE",
			setupFn: func(t *testing.T,
				repoBuilder *testutils.RepoBuilder,
				repoMock *gitmock.MockRepository,
				modelMock *modelmock.MockModeler) wantFunc {
				t.Helper()

				hash, err := repoBuilder.CreateRandomCommit(10)
				assert.NoError(t, err)
				_, err = repoBuilder.CreateBranch(localBranchName, hash)
				assert.NoError(t, err)

				localRef, err := repoBuilder.Repo().Reference(localBranchRefName, true)
				assert.NoError(t, err)

				// The following EXPECTs are in sequential order, regardless of mocked interface

				repoMock.EXPECT().
					Reference(localBranchRefName, true).
					Return(localRef, nil)

				modelMock.EXPECT().
					ResolveRef(gomock.Any(), remoteBranchRefName).
					Return(nil, "", model.ErrReferenceNotFound)

				localCommitObj, err := repoBuilder.Repo().CommitObject(localRef.Hash())
				assert.NoError(t, err)

				repoMock.EXPECT().
					CommitObject(localRef.Hash()).
					Return(localCommitObj, nil)

				modelMock.EXPECT().
					CommitExists(repoMock, localCommitObj).
					Return("", nil)

				return func(t *testing.T, refPair RefPair, err error) {
					t.Helper()

					assert.NoError(t, err)
					assert.Equal(t, localRef, refPair.Local)
					assert.Equal(t,
						plumbing.NewHashReference(remoteBranchRefName, plumbing.ZeroHash),
						refPair.Remote)
					assert.Equal(t, StatusAddCommit|StatusUpdateRef, refPair.Status)
				}
			},
			force:      false,
			localName:  localBranchRefName,
			remoteName: remoteBranchRefName,
		},
		{name: "Remote Is Ancestor",
			setupFn: func(t *testing.T,
				repoBuilder *testutils.RepoBuilder,
				repoMock *gitmock.MockRepository,
				modelMock *modelmock.MockModeler) wantFunc {
				t.Helper()

				remoteHash, err := repoBuilder.CreateRandomCommit(10)
				assert.NoError(t, err)
				localHash, err := repoBuilder.CreateRandomCommit(10)
				assert.NoError(t, err)
				_, err = repoBuilder.CreateBranch(localBranchName, localHash)
				assert.NoError(t, err)

				localRef, err := repoBuilder.Repo().Reference(localBranchRefName, true)
				assert.NoError(t, err)

				repoMock.EXPECT().
					Reference(localBranchRefName, true).
					Return(localRef, nil)

				remoteRef := plumbing.NewHashReference(
					remoteBranchRefName,
					remoteHash)

				modelMock.EXPECT().
					ResolveRef(gomock.Any(), remoteBranchRefName).
					Return(remoteRef, digest.FromString("foo"), nil)

				localCommitObj, err := repoBuilder.Repo().CommitObject(localRef.Hash())
				assert.NoError(t, err)

				repoMock.EXPECT().
					CommitObject(localRef.Hash()).
					Return(localCommitObj, nil)

				modelMock.EXPECT().
					CommitExists(repoMock, localCommitObj).
					Return("", nil)

				repoMock.EXPECT().
					CommitObject(remoteHash).
					DoAndReturn(func(h plumbing.Hash) (*object.Commit, error) {
						return repoBuilder.Repo().CommitObject(remoteHash)
					})

				return func(t *testing.T, refPair RefPair, err error) {
					t.Helper()

					assert.NoError(t, err)
					assert.Equal(t, localRef, refPair.Local)
					assert.Equal(t, remoteRef, refPair.Remote)
					assert.Equal(t, StatusAddCommit|StatusUpdateRef, refPair.Status)
				}
			},
			force:      false,
			localName:  localBranchRefName,
			remoteName: remoteBranchRefName,
		},
		{name: "Remote Is Not Ancestor",
			setupFn: func(t *testing.T,
				repoBuilder *testutils.RepoBuilder,
				repoMock *gitmock.MockRepository,
				modelMock *modelmock.MockModeler) wantFunc {
				t.Helper()

				// All we did here was flip localHash and remoteHash (and the wantFn) so remote is ahead of local, i.e. not an ancestor
				localHash, err := repoBuilder.CreateRandomCommit(10)
				assert.NoError(t, err)
				remoteHash, err := repoBuilder.CreateRandomCommit(10)
				assert.NoError(t, err)
				_, err = repoBuilder.CreateBranch(localBranchName, localHash)
				assert.NoError(t, err)

				localRef, err := repoBuilder.Repo().Reference(localBranchRefName, true)
				assert.NoError(t, err)

				repoMock.EXPECT().
					Reference(localBranchRefName, true).
					Return(localRef, nil)

				remoteRef := plumbing.NewHashReference(
					remoteBranchRefName,
					remoteHash)

				modelMock.EXPECT().
					ResolveRef(gomock.Any(), remoteBranchRefName).
					Return(remoteRef, digest.FromString("foo"), nil)

				localCommitObj, err := repoBuilder.Repo().CommitObject(localRef.Hash())
				assert.NoError(t, err)

				repoMock.EXPECT().
					CommitObject(localRef.Hash()).
					Return(localCommitObj, nil)

				modelMock.EXPECT().
					CommitExists(repoMock, localCommitObj).
					Return("", nil)

				repoMock.EXPECT().
					CommitObject(remoteRef.Hash()).
					DoAndReturn(func(h plumbing.Hash) (*object.Commit, error) {
						return repoBuilder.Repo().CommitObject(remoteRef.Hash())
					})

				return func(t *testing.T, refPair RefPair, err error) {
					t.Helper()

					assert.Nil(t, refPair.Local)
					assert.Nil(t, refPair.Remote)
					assert.Error(t, err)
					assert.Equal(t, Status(0), refPair.Status)
				}
			},
			force:      false,
			localName:  localBranchRefName,
			remoteName: remoteBranchRefName,
		},
		{name: "Remote Is Not Ancestor but Force",
			setupFn: func(t *testing.T,
				repoBuilder *testutils.RepoBuilder,
				repoMock *gitmock.MockRepository,
				modelMock *modelmock.MockModeler) wantFunc {
				t.Helper()

				// All we did here was flip localHash and remoteHash (and the wantFn) so remote is ahead of local, i.e. not an ancestor
				localHash, err := repoBuilder.CreateRandomCommit(10)
				assert.NoError(t, err)
				remoteHash, err := repoBuilder.CreateRandomCommit(10)
				assert.NoError(t, err)
				_, err = repoBuilder.CreateBranch(localBranchName, localHash)
				assert.NoError(t, err)

				localRef, err := repoBuilder.Repo().Reference(localBranchRefName, true)
				assert.NoError(t, err)

				repoMock.EXPECT().
					Reference(localBranchRefName, true).
					Return(localRef, nil)

				remoteRef := plumbing.NewHashReference(
					remoteBranchRefName,
					remoteHash)

				modelMock.EXPECT().
					ResolveRef(gomock.Any(), remoteBranchRefName).
					Return(remoteRef, digest.FromString("foo"), nil)

				localCommitObj, err := repoBuilder.Repo().CommitObject(localRef.Hash())
				assert.NoError(t, err)

				repoMock.EXPECT().
					CommitObject(localRef.Hash()).
					Return(localCommitObj, nil)

				modelMock.EXPECT().
					CommitExists(repoMock, localCommitObj).
					Return("", nil)

				repoMock.EXPECT().
					CommitObject(remoteRef.Hash()).
					DoAndReturn(func(h plumbing.Hash) (*object.Commit, error) {
						return repoBuilder.Repo().CommitObject(remoteRef.Hash())
					})

				return func(t *testing.T, refPair RefPair, err error) {
					t.Helper()

					assert.Equal(t, localRef, refPair.Local)
					assert.Equal(t, remoteRef, refPair.Remote)
					assert.NoError(t, err)
					assert.Equal(t, StatusForce|StatusUpdateRef|StatusAddCommit, refPair.Status)
				}
			},
			force:      true,
			localName:  localBranchRefName,
			remoteName: remoteBranchRefName,
		},
		{name: "Remote Is Ancestor - Commit Exists",
			setupFn: func(t *testing.T,
				repoBuilder *testutils.RepoBuilder,
				repoMock *gitmock.MockRepository,
				modelMock *modelmock.MockModeler) wantFunc {
				t.Helper()

				remoteHash, err := repoBuilder.CreateRandomCommit(10)
				assert.NoError(t, err)
				localHash, err := repoBuilder.CreateRandomCommit(10)
				assert.NoError(t, err)
				_, err = repoBuilder.CreateBranch(localBranchName, localHash)
				assert.NoError(t, err)

				localRef, err := repoBuilder.Repo().Reference(localBranchRefName, true)
				assert.NoError(t, err)

				repoMock.EXPECT().
					Reference(localBranchRefName, true).
					Return(localRef, nil)

				remoteRef := plumbing.NewHashReference(
					remoteBranchRefName,
					remoteHash)

				modelMock.EXPECT().
					ResolveRef(gomock.Any(), remoteBranchRefName).
					Return(remoteRef, digest.FromString("foo"), nil)

				localCommitObj, err := repoBuilder.Repo().CommitObject(localRef.Hash())
				assert.NoError(t, err)

				repoMock.EXPECT().
					CommitObject(localRef.Hash()).
					Return(localCommitObj, nil)

				modelMock.EXPECT().
					CommitExists(repoMock, localCommitObj).
					Return(digest.FromString("foo"), nil)

				repoMock.EXPECT().
					CommitObject(remoteHash).
					DoAndReturn(func(h plumbing.Hash) (*object.Commit, error) {
						return repoBuilder.Repo().CommitObject(remoteHash)
					})

				return func(t *testing.T, refPair RefPair, err error) {
					t.Helper()

					assert.NoError(t, err)
					assert.Equal(t, localRef, refPair.Local)
					assert.Equal(t, remoteRef, refPair.Remote)
					assert.Equal(t, StatusUpdateRef, refPair.Status)
				}
			},
			force:      false,
			localName:  localBranchRefName,
			remoteName: remoteBranchRefName,
		},
		{name: "Delete Reference",
			setupFn: func(t *testing.T,
				repoBuilder *testutils.RepoBuilder,
				repoMock *gitmock.MockRepository,
				modelMock *modelmock.MockModeler) wantFunc {
				t.Helper()

				hash, err := repoBuilder.CreateRandomCommit(10)
				assert.NoError(t, err)

				remoteRef := plumbing.NewHashReference(
					remoteBranchRefName,
					hash)

				modelMock.EXPECT().
					ResolveRef(gomock.Any(), remoteBranchRefName).
					Return(remoteRef, digest.FromString("foo"), nil)

				return func(t *testing.T, refPair RefPair, err error) {
					t.Helper()

					assert.NoError(t, err)
					assert.Nil(t, refPair.Local)
					assert.Equal(t, remoteRef, refPair.Remote)
					assert.Equal(t, StatusDelete|StatusUpdateRef, refPair.Status)
				}
			},
			force:      false,
			localName:  "",
			remoteName: remoteBranchRefName,
		},
		{
			name: "ErrUnsupportedReferenceType",
			setupFn: func(t *testing.T,
				repoBuilder *testutils.RepoBuilder,
				repoMock *gitmock.MockRepository,
				modelMock *modelmock.MockModeler) wantFunc {
				t.Helper()

				hash, err := repoBuilder.CreateRandomCommit(10)
				assert.NoError(t, err)
				_, err = repoBuilder.CreateBranch(localBranchName, hash)
				assert.NoError(t, err)

				localRef, err := repoBuilder.Repo().Reference(localBranchRefName, true)
				assert.NoError(t, err)

				// The following EXPECTs are in sequential order, regardless of mocked interface

				repoMock.EXPECT().
					Reference(localBranchRefName, true).
					Return(localRef, nil)

				modelMock.EXPECT().
					ResolveRef(gomock.Any(), remoteBranchRefName).
					Return(nil, "", model.ErrUnsupportedReferenceType)

				return func(t *testing.T, refPair RefPair, err error) {
					t.Helper()

					assert.ErrorIs(t, err, model.ErrUnsupportedReferenceType)
					assert.Nil(t, refPair.Local)
					assert.Nil(t, refPair.Remote)
					assert.Equal(t, Status(0), refPair.Status)
				}
			},
			force:      false,
			localName:  localBranchRefName,
			remoteName: remoteBranchRefName,
		},
		{name: "Failed to Resolve Reference",
			setupFn: func(t *testing.T,
				repoBuilder *testutils.RepoBuilder,
				repoMock *gitmock.MockRepository,
				modelMock *modelmock.MockModeler) wantFunc {
				t.Helper()

				hash, err := repoBuilder.CreateRandomCommit(10)
				assert.NoError(t, err)
				_, err = repoBuilder.CreateBranch(localBranchName, hash)
				assert.NoError(t, err)

				// The following EXPECTs are in sequential order, regardless of mocked interface

				repoMock.EXPECT().
					Reference(nonexistantBranchRefName, true).
					DoAndReturn(func(rn plumbing.ReferenceName, b bool) (*plumbing.Reference, error) {
						return repoBuilder.Repo().Reference(nonexistantBranchRefName, true)
					})

				return func(t *testing.T, refPair RefPair, err error) {
					t.Helper()

					assert.Error(t, err)
					assert.Nil(t, refPair.Local)
					assert.Nil(t, refPair.Remote)
					assert.Equal(t, Status(0), refPair.Status)
				}
			},
			force:      false,
			localName:  nonexistantBranchRefName,
			remoteName: remoteBranchRefName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repoMock := gitmock.NewMockRepository(ctrl)
			modelMock := modelmock.NewMockModeler(ctrl)

			repoBuilder, err := testutils.NewRepoBuilder(t.TempDir())
			assert.NoError(t, err)

			wantFn := tt.setupFn(t, repoBuilder, repoMock, modelMock)

			rc := &refCompareCached{
				local:  repoMock,
				remote: modelMock,
				refs:   map[plumbing.ReferenceName]RefPair{},
			}

			gotRefPair, err := rc.Compare(t.Context(), tt.force, tt.localName, tt.remoteName)
			wantFn(t, gotRefPair, err)
		})
	}
}
