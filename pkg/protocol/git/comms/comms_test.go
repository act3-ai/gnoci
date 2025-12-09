// Package comms facilitates receiving requests from and writing responses to Git via the remote helpers protocol.
//
// Protocol Reference: https://git-scm.com/docs/gitremote-helpers.
package comms

import (
	"bufio"
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/act3-ai/gnoci/pkg/protocol/git"
	"github.com/act3-ai/gnoci/pkg/testutils"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
)

func TestNewCommunicator(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := strings.NewReader("foo")
		out := new(bytes.Buffer)

		comm := NewCommunicator(in, out)
		assert.NotNil(t, comm)
		defaultComm, ok := comm.(*defaultCommunicator)
		assert.True(t, ok)
		assert.NotNil(t, defaultComm)
		assert.NotNil(t, defaultComm.in)
		assert.Equal(t, out, defaultComm.out)
	})
}

func Test_defaultCommunicator_LookAhead(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendCapabilitiesRequest()
		assert.NoError(t, err)

		cmd, err := comm.LookAhead()
		assert.NoError(t, err)
		assert.Equal(t, git.Capabilities, cmd)

		assert.Equal(t, []string{string(git.Capabilities)}, comm.previous)

		// ensure previous is reset appropriately
		req, err := comm.ParseCapabilitiesRequest()
		assert.NoError(t, err)
		assert.NotNil(t, req)
		assert.Nil(t, comm.previous)
	})

	t.Run("Empty Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		_, err := in.WriteString("\n")
		assert.NoError(t, err)

		_, err = comm.LookAhead()
		assert.ErrorIs(t, err, git.ErrEmptyRequest)
	})

	t.Run("Unsupported Command", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		_, err := in.WriteString("foo bar\n")
		assert.NoError(t, err)

		_, err = comm.LookAhead()
		assert.Error(t, err)
	})
}

func Test_defaultCommunicator_previousOrNext(t *testing.T) {
	t.Run("Previous", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		// ensure this isn't read
		_, err := in.WriteString("bar bar\n")
		assert.NoError(t, err)

		expected := []string{"foo", "bar"}
		comm.previous = slices.Clone(expected)

		res, err := comm.previousOrNext()
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, expected, res)
	})

	t.Run("Next", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		// ensure this IS read
		_, err := in.WriteString("bar bar\n")
		assert.NoError(t, err)

		res, err := comm.previousOrNext()
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, []string{"bar", "bar"}, res)
	})

	t.Run("Empty Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		_, err := in.WriteString("\n")
		assert.NoError(t, err)

		_, err = comm.previousOrNext()
		assert.ErrorIs(t, err, git.ErrEmptyRequest)
	})
}

func Test_defaultCommunicator_next(t *testing.T) {
	t.Run("Next", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		// ensure this is irrelevant
		comm.previous = []string{"foo", "bar"}

		_, err := in.WriteString("bar bar\n")
		assert.NoError(t, err)

		res, err := comm.next()
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, []string{"bar", "bar"}, res)
	})

	t.Run("Empty Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		// ensure this is irrelevant
		comm.previous = []string{"foo", "bar"}

		_, err := in.WriteString("\n")
		assert.NoError(t, err)

		res, err := comm.next()
		assert.ErrorIs(t, err, git.ErrEmptyRequest)
		assert.Nil(t, res)
	})

	t.Run("End of Input", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		_, err := comm.next()
		assert.ErrorIs(t, err, git.ErrEndOfInput)
	})
}

func Test_defaultCommunicator_ParseCapabilitiesRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendCapabilitiesRequest()
		assert.NoError(t, err)

		req, err := comm.ParseCapabilitiesRequest()
		assert.NoError(t, err)
		assert.NotNil(t, req)
	})

	t.Run("Handle Previous", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Capabilities)},
		}

		// ensure previous takes precedence
		_, err := in.WriteString("foo bar\n")
		assert.NoError(t, err)

		req, err := comm.ParseCapabilitiesRequest()
		assert.NoError(t, err)
		assert.NotNil(t, req)

		// ensure previous is wiped
		assert.Nil(t, comm.previous)
	})

	t.Run("Invalid Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		_, err := in.WriteString("foo\n")
		assert.NoError(t, err)

		req, err := comm.ParseCapabilitiesRequest()
		assert.ErrorIs(t, err, git.ErrUnexpectedRequest)
		assert.Nil(t, req)
	})

	t.Run("Empty Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		_, err := in.WriteString("\n")
		assert.NoError(t, err)

		req, err := comm.ParseCapabilitiesRequest()
		assert.ErrorIs(t, err, git.ErrEmptyRequest)
		assert.Nil(t, req)
	})
}

func Test_defaultCommunicator_ParseOptionRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendOptionRequest(git.Verbosity, "10")
		assert.NoError(t, err)

		req, err := comm.ParseOptionRequest()
		assert.NoError(t, err)
		assert.NotNil(t, req)
	})

	t.Run("Handle Previous", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Options), string(git.Verbosity), "10"},
		}

		// ensure previous takes precedence
		_, err := in.WriteString("foo bar\n")
		assert.NoError(t, err)

		req, err := comm.ParseOptionRequest()
		assert.NoError(t, err)
		assert.NotNil(t, req)

		// ensure previous is wiped
		assert.Nil(t, comm.previous)
	})

	t.Run("Invalid Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		_, err := in.WriteString("foo\n")
		assert.NoError(t, err)

		req, err := comm.ParseOptionRequest()
		assert.ErrorIs(t, err, git.ErrBadRequest)
		assert.Nil(t, req)
	})

	t.Run("Empty Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		_, err := in.WriteString("\n")
		assert.NoError(t, err)

		req, err := comm.ParseOptionRequest()
		assert.ErrorIs(t, err, git.ErrEmptyRequest)
		assert.Nil(t, req)
	})
}

func Test_defaultCommunicator_ParseListRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendListRequest(false)
		assert.NoError(t, err)

		req, err := comm.ParseListRequest()
		assert.NoError(t, err)
		assert.NotNil(t, req)
	})

	t.Run("Success with ForPush", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendListRequest(true)
		assert.NoError(t, err)

		req, err := comm.ParseListRequest()
		assert.NoError(t, err)
		assert.NotNil(t, req)
	})

	t.Run("Handle Previous", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.List)},
		}

		// ensure previous takes precedence
		_, err := in.WriteString("foo bar\n")
		assert.NoError(t, err)

		req, err := comm.ParseListRequest()
		assert.NoError(t, err)
		assert.NotNil(t, req)

		// ensure previous is wiped
		assert.Nil(t, comm.previous)
	})

	t.Run("Invalid Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		_, err := in.WriteString("foo\n")
		assert.NoError(t, err)

		req, err := comm.ParseListRequest()
		assert.ErrorIs(t, err, git.ErrUnexpectedRequest)
		assert.Nil(t, req)
	})

	t.Run("Empty Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		_, err := in.WriteString("\n")
		assert.NoError(t, err)

		req, err := comm.ParseListRequest()
		assert.ErrorIs(t, err, git.ErrEmptyRequest)
		assert.Nil(t, req)
	})
}

func Test_defaultCommunicator_ParseFetchRequestBatch(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		refs := []plumbing.Reference{
			*plumbing.NewHashReference(
				plumbing.NewBranchReferenceName("foo"),
				plumbing.ComputeHash(plumbing.CommitObject, []byte("foo")),
			),
			*plumbing.NewHashReference(
				plumbing.NewBranchReferenceName("bar"),
				plumbing.ComputeHash(plumbing.CommitObject, []byte("bar")),
			),
		}

		err := revcomm.SendFetchRequestBatch(refs)
		assert.NoError(t, err)

		// test for batch processing, the values themselves are tested in the parser test
		reqs, err := comm.ParseFetchRequestBatch()
		assert.NoError(t, err)
		assert.NotNil(t, reqs)
		assert.Equal(t, len(refs), len(reqs))
	})

	t.Run("Success with Previous", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Fetch), plumbing.ComputeHash(plumbing.CommitObject, []byte("foo")).String(), plumbing.NewBranchReferenceName("foo").String()},
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		refs := []plumbing.Reference{
			*plumbing.NewHashReference(
				plumbing.NewBranchReferenceName("bar"),
				plumbing.ComputeHash(plumbing.CommitObject, []byte("bar")),
			),
		}

		err := revcomm.SendFetchRequestBatch(refs)
		assert.NoError(t, err)

		// test for batch processing, the values themselves are tested in the parser test
		reqs, err := comm.ParseFetchRequestBatch()
		assert.NoError(t, err)
		assert.NotNil(t, reqs)
		assert.Equal(t, len(refs)+1, len(reqs))
	})

	t.Run("Invalid Previous", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Fetch), plumbing.ComputeHash(plumbing.CommitObject, []byte("foo")).String()},
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		refs := []plumbing.Reference{
			*plumbing.NewHashReference(
				plumbing.NewBranchReferenceName("bar"),
				plumbing.ComputeHash(plumbing.CommitObject, []byte("bar")),
			),
		}

		err := revcomm.SendFetchRequestBatch(refs)
		assert.NoError(t, err)

		// test for batch processing, the values themselves are tested in the parser test
		reqs, err := comm.ParseFetchRequestBatch()
		assert.Error(t, err)
		assert.Nil(t, reqs)
	})

	t.Run("Invalid Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		_, err := in.WriteString("foo\n")
		assert.NoError(t, err)

		req, err := comm.ParseFetchRequestBatch()
		assert.ErrorIs(t, err, git.ErrBadRequest)
		assert.Nil(t, req)
	})

	t.Run("Empty Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		req, err := comm.ParseFetchRequestBatch()
		assert.Error(t, err)
		assert.Nil(t, req)
	})

	t.Run("Insufficient Batch Size", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		_, err := in.WriteString("\n")
		assert.NoError(t, err)

		req, err := comm.ParseFetchRequestBatch()
		assert.Error(t, err)
		assert.Nil(t, req)
	})
}

func Test_defaultCommunicator_ParsePushRequestBatch(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		ref1 := "foo"
		ref2 := "bar"
		ref3 := "foobar"
		refs := map[string]string{
			ref1: "commit1",
			ref2: "commit2",
			ref3: "commit3",
		}

		err := revcomm.SendPushRequestBatch(refs)
		assert.NoError(t, err)

		// test for batch processing, the values themselves are tested in the parser test
		reqs, err := comm.ParsePushRequestBatch()
		assert.NoError(t, err)
		assert.NotNil(t, reqs)
		assert.Equal(t, len(refs), len(reqs))
	})

	t.Run("Success with Previous", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Push), "foo:commit1"},
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		ref2 := "bar"
		ref3 := "foobar"
		refs := map[string]string{
			ref2: "commit2",
			ref3: "commit3",
		}

		err := revcomm.SendPushRequestBatch(refs)
		assert.NoError(t, err)

		// test for batch processing, the values themselves are tested in the parser test
		reqs, err := comm.ParsePushRequestBatch()
		assert.NoError(t, err)
		assert.NotNil(t, reqs)
		assert.Equal(t, len(refs)+1, len(reqs))
	})

	t.Run("Invalid Previous", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Push)},
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		ref2 := "bar"
		ref3 := "foobar"
		refs := map[string]string{
			ref2: "commit2",
			ref3: "commit3",
		}

		err := revcomm.SendPushRequestBatch(refs)
		assert.NoError(t, err)

		// test for batch processing, the values themselves are tested in the parser test
		reqs, err := comm.ParsePushRequestBatch()
		assert.Error(t, err)
		assert.Nil(t, reqs)
	})

	t.Run("Invalid Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		_, err := in.WriteString("foo\n")
		assert.NoError(t, err)

		req, err := comm.ParsePushRequestBatch()
		assert.ErrorIs(t, err, git.ErrBadRequest)
		assert.Nil(t, req)
	})

	t.Run("Empty Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		req, err := comm.ParsePushRequestBatch()
		assert.Error(t, err)
		assert.Nil(t, req)
	})

	t.Run("Insufficient Batch Size", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:  *bufio.NewScanner(in),
			out: out,
		}

		_, err := in.WriteString("\n")
		assert.NoError(t, err)

		req, err := comm.ParsePushRequestBatch()
		assert.Error(t, err)
		assert.Nil(t, req)
	})
}

func Test_defaultCommunicator_WriteCapabilitiesResponse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Push)},
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := comm.WriteCapabilitiesResponse([]git.Capability{git.CapabilityOption, git.CapabilityFetch, git.CapabilityPush})
		assert.NoError(t, err)

		err = revcomm.ReceiveCapabilitiesResponse()
		assert.NoError(t, err)
	})

	t.Run("No Capabilities", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Push)},
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := comm.WriteCapabilitiesResponse([]git.Capability{})
		assert.NoError(t, err)

		err = revcomm.ReceiveCapabilitiesResponse()
		assert.Error(t, err)
	})
}

func Test_defaultCommunicator_WriteOptionResponse(t *testing.T) {
	t.Run("Supported Option", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Push)},
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := comm.WriteOptionResponse(true)
		assert.NoError(t, err)

		err = revcomm.ReceiveOptionResponse()
		assert.NoError(t, err)
	})

	t.Run("Unsupported Option", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Push)},
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := comm.WriteOptionResponse(false)
		assert.NoError(t, err)

		err = revcomm.ReceiveOptionResponse()
		assert.NoError(t, err)
	})
}

func Test_defaultCommunicator_WriteListResponse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Push)},
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		resps := []git.ListResponse{
			{
				Reference: plumbing.ReferenceName("foo"),
				Commit:    "commit1",
			},
			{
				Reference: plumbing.ReferenceName("bar"),
				Commit:    "commit2",
			},
		}
		err := comm.WriteListResponse(resps)
		assert.NoError(t, err)

		err = revcomm.ReceiveListResponse()
		assert.NoError(t, err)
	})

	t.Run("Empty List", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Push)},
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := comm.WriteListResponse([]git.ListResponse{})
		assert.NoError(t, err)

		err = revcomm.ReceiveListResponse()
		assert.NoError(t, err)
	})
}

func Test_defaultCommunicator_WritePushResponse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Push)},
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		resps := []git.PushResponse{
			{
				Remote: plumbing.ReferenceName("foo"),
			},
			{
				Remote: plumbing.ReferenceName("bar"),
			},
		}
		err := comm.WritePushResponse(resps)
		assert.NoError(t, err)

		err = revcomm.ReceivePushResponseBatch()
		assert.NoError(t, err)
	})

	t.Run("Push Errors", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Push)},
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		resps := []git.PushResponse{
			{
				Remote: plumbing.ReferenceName("foo"),
				Error:  errors.New("failed to push foo"),
			},
			{
				Remote: plumbing.ReferenceName("bar"),
				Error:  errors.New("failed to push bar"),
			},
		}
		err := comm.WritePushResponse(resps)
		assert.NoError(t, err)

		err = revcomm.ReceivePushResponseBatch()
		assert.NoError(t, err)
	})
}

func Test_defaultCommunicator_WriteFetchResponse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := &defaultCommunicator{
			in:       *bufio.NewScanner(in),
			out:      out,
			previous: []string{string(git.Push)},
		}
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := comm.WriteFetchResponse()
		assert.NoError(t, err)

		err = revcomm.ReceiveFetchResponse()
		assert.NoError(t, err)
	})
}
