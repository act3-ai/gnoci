package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/act3-ai/gnoci/internal/mocks/gitmock"
	"github.com/act3-ai/gnoci/internal/mocks/modelmock"
	"github.com/act3-ai/gnoci/internal/testutils"
	"github.com/act3-ai/gnoci/pkg/oci"
	"github.com/act3-ai/gnoci/pkg/protocol/git/comms"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleList(t *testing.T) {
	t.Run("Success - Not For Push", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelMock := modelmock.NewMockModeler(ctrl)

		expectedHeads := map[plumbing.ReferenceName]oci.ReferenceInfo{
			plumbing.ReferenceName("refs/heads/main"): {
				Commit: "32396c14a264a71cbd47cc7a8678cebb2cdd15ed",
				Layer:  digest.Digest("sha256:eba70958398124d1699b1d5733b916677c9bc2f7629153191eed4d7086976070"),
			},
		}

		expectedTags := map[plumbing.ReferenceName]oci.ReferenceInfo{
			plumbing.ReferenceName("refs/tags/v1.0.0"): {
				Commit: "32396c14a264a71cbd47cc7a8678cebb2cdd15ed",
				Layer:  digest.Digest("sha256:eba70958398124d1699b1d5733b916677c9bc2f7629153191eed4d7086976070"),
			},
		}

		modelMock.EXPECT().
			HeadRefs().
			Return(expectedHeads).
			Times(1)

		modelMock.EXPECT().
			TagRefs().
			Return(expectedTags).
			Times(1)

		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := comms.NewCommunicator(in, out)
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendListRequest(false)
		assert.NoError(t, err)

		err = HandleList(t.Context(), nil, modelMock, comm)
		assert.NoError(t, err)

		err = revcomm.ReceiveListResponse()
		assert.NoError(t, err)
	})

	t.Run("Success - Not For Push with HEAD", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelMock := modelmock.NewMockModeler(ctrl)
		gitMock := gitmock.NewMockRepository(ctrl)

		headRef := plumbing.ReferenceName("refs/heads/main")
		headHash := plumbing.ComputeHash(plumbing.CommitObject, []byte("foo"))
		expectedHeads := map[plumbing.ReferenceName]oci.ReferenceInfo{
			headRef: {
				Commit: headHash.String(),
				Layer:  digest.Digest("sha256:eba70958398124d1699b1d5733b916677c9bc2f7629153191eed4d7086976070"),
			},
		}

		expectedTags := map[plumbing.ReferenceName]oci.ReferenceInfo{
			plumbing.ReferenceName("refs/tags/v1.0.0"): {
				Commit: "32396c14a264a71cbd47cc7a8678cebb2cdd15ed",
				Layer:  digest.Digest("sha256:eba70958398124d1699b1d5733b916677c9bc2f7629153191eed4d7086976070"),
			},
		}

		modelMock.EXPECT().
			HeadRefs().
			Return(expectedHeads).
			Times(1)

		modelMock.EXPECT().
			TagRefs().
			Return(expectedTags).
			Times(1)

		gitMock.EXPECT().
			Head().
			Return(plumbing.NewHashReference(headRef, headHash), nil)

		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := comms.NewCommunicator(in, out)
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendListRequest(false)
		assert.NoError(t, err)

		err = HandleList(t.Context(), gitMock, modelMock, comm)
		assert.NoError(t, err)

		err = revcomm.ReceiveListResponse()
		assert.NoError(t, err)
	})

	t.Run("Success - Not For Push HEAD not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelMock := modelmock.NewMockModeler(ctrl)
		gitMock := gitmock.NewMockRepository(ctrl)

		headRef := plumbing.ReferenceName("refs/heads/main")
		headHash := plumbing.ComputeHash(plumbing.CommitObject, []byte("foo"))
		expectedHeads := map[plumbing.ReferenceName]oci.ReferenceInfo{
			headRef: {
				Commit: headHash.String(),
				Layer:  digest.Digest("sha256:eba70958398124d1699b1d5733b916677c9bc2f7629153191eed4d7086976070"),
			},
		}

		expectedTags := map[plumbing.ReferenceName]oci.ReferenceInfo{
			plumbing.ReferenceName("refs/tags/v1.0.0"): {
				Commit: "32396c14a264a71cbd47cc7a8678cebb2cdd15ed",
				Layer:  digest.Digest("sha256:eba70958398124d1699b1d5733b916677c9bc2f7629153191eed4d7086976070"),
			},
		}

		modelMock.EXPECT().
			HeadRefs().
			Return(expectedHeads).
			Times(1)

		modelMock.EXPECT().
			TagRefs().
			Return(expectedTags).
			Times(1)

		gitMock.EXPECT().
			Head().
			Return(nil, errors.New("head not found"))

		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := comms.NewCommunicator(in, out)
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendListRequest(false)
		assert.NoError(t, err)

		err = HandleList(t.Context(), gitMock, modelMock, comm)
		assert.NoError(t, err)

		err = revcomm.ReceiveListResponse()
		assert.NoError(t, err)
	})

	t.Run("Invalid List Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := comms.NewCommunicator(in, out)
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendCapabilitiesRequest()
		assert.NoError(t, err)

		err = HandleList(t.Context(), nil, nil, comm)
		assert.Error(t, err)
	})
}
