package testutils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/act3-ai/gnoci/pkg/protocol/git"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
)

func TestNewReverseCommunicator(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := strings.NewReader("foo")
		out := new(bytes.Buffer)

		revCommI := NewReverseCommunicator(in, out)
		assert.NotNil(t, revCommI)

		revComm, ok := revCommI.(*reverseCommunicator)
		assert.True(t, ok)
		assert.NotNil(t, revComm)
		assert.NotNil(t, revComm.in)
		assert.Equal(t, out, revComm.out)
	})
}

func Test_reverseCommunicator_SendCapabilitiesRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		out := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(nil, out)

		err := revcomm.SendCapabilitiesRequest()
		assert.NoError(t, err)

		outRaw, err := io.ReadAll(out)
		assert.NoError(t, err)
		assert.Equal(t, "capabilities\n", string(outRaw))
	})
}

func Test_reverseCommunicator_SendOptionRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		out := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(nil, out)

		opt := git.Verbosity
		value := "10"
		err := revcomm.SendOptionRequest(opt, value)
		assert.NoError(t, err)

		outRaw, err := io.ReadAll(out)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("%s %s %s\n", git.Options, opt, value), string(outRaw))
	})

	t.Run("Empty Option", func(t *testing.T) {
		out := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(nil, out)

		opt := git.Option("")
		value := "10"
		err := revcomm.SendOptionRequest(opt, value)
		assert.Error(t, err)
	})

	t.Run("Empty Value", func(t *testing.T) {
		out := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(nil, out)

		opt := git.Verbosity
		value := ""
		err := revcomm.SendOptionRequest(opt, value)
		assert.Error(t, err)
	})
}

func Test_reverseCommunicator_SendListRequest(t *testing.T) {
	t.Run("Success - Basic", func(t *testing.T) {
		out := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(nil, out)

		err := revcomm.SendListRequest(false)
		assert.NoError(t, err)

		outRaw, err := io.ReadAll(out)
		assert.NoError(t, err)

		assert.Equal(t, fmt.Sprintf("%s\n", git.List), string(outRaw))
	})

	t.Run("Success - ForPush", func(t *testing.T) {
		out := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(nil, out)

		err := revcomm.SendListRequest(true)
		assert.NoError(t, err)

		outRaw, err := io.ReadAll(out)
		assert.NoError(t, err)

		assert.Equal(t, fmt.Sprintf("%s for-push\n", git.List), string(outRaw))
	})
}

func Test_reverseCommunicator_SendPushRequestBatch(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		out := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(nil, out)

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

		scan := bufio.NewScanner(out)

		// TODO: weird, but iterating through a map is non-deterministic
		for range refs {
			ok := scan.Scan()
			assert.True(t, ok)
			line := scan.Text()
			fields := strings.Fields(line)

			var req git.PushRequest
			err := req.Parse(fields)
			assert.NoError(t, err)
			assert.False(t, req.Force)

			switch req.Src {
			case plumbing.ReferenceName(ref1):
				assert.Equal(t, plumbing.ReferenceName(refs[ref1]), req.Remote)
			case plumbing.ReferenceName(ref2):
				assert.Equal(t, plumbing.ReferenceName(refs[ref2]), req.Remote)
			case plumbing.ReferenceName(ref3):
				assert.Equal(t, plumbing.ReferenceName(refs[ref3]), req.Remote)
			default:
				t.Fatalf("unexpected reference %s", line)
			}

		}
		ok := scan.Scan()
		assert.True(t, ok)
		assert.Equal(t, "", scan.Text())
	})

	t.Run("Success with Force", func(t *testing.T) {
		out := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(nil, out)

		ref1 := "+foo"
		ref2 := "+bar"
		ref3 := "+foobar"
		refs := map[string]string{
			ref1: "commit1",
			ref2: "commit2",
			ref3: "commit3",
		}

		err := revcomm.SendPushRequestBatch(refs)
		assert.NoError(t, err)

		scan := bufio.NewScanner(out)

		// TODO: weird, but iterating through a map is non-deterministic
		for range refs {
			ok := scan.Scan()
			assert.True(t, ok)
			line := scan.Text()
			fields := strings.Fields(line)

			var req git.PushRequest
			err := req.Parse(fields)
			assert.NoError(t, err)
			assert.True(t, req.Force)

			switch "+" + req.Src {
			case plumbing.ReferenceName(ref1):
				assert.Equal(t, plumbing.ReferenceName(refs[ref1]), req.Remote)
			case plumbing.ReferenceName(ref2):
				assert.Equal(t, plumbing.ReferenceName(refs[ref2]), req.Remote)
			case plumbing.ReferenceName(ref3):
				assert.Equal(t, plumbing.ReferenceName(refs[ref3]), req.Remote)
			default:
				t.Fatalf("unexpected reference %s", line)
			}

		}
		ok := scan.Scan()
		assert.True(t, ok)
		assert.Equal(t, "", scan.Text())
	})
}

func Test_reverseCommunicator_SendFetchRequestBatch(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		out := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(nil, out)

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

		scan := bufio.NewScanner(out)
		for _, ref := range refs {
			ok := scan.Scan()
			assert.True(t, ok)
			line := scan.Text()
			fields := strings.Fields(line)

			var req git.FetchRequest
			err := req.Parse(fields)
			assert.NoError(t, err)
			assert.Equal(t, ref, *req.Ref)
		}
		ok := scan.Scan()
		assert.True(t, ok)
		assert.Equal(t, "", scan.Text())
	})
}

func Test_reverseCommunicator_ReceiveCapabilitiesResponse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		_, err := fmt.Fprintf(in, "%s\n", git.CapabilityOption)
		assert.NoError(t, err)
		_, err = fmt.Fprintf(in, "%s\n", git.CapabilityFetch)
		assert.NoError(t, err)
		_, err = fmt.Fprintf(in, "%s\n", git.CapabilityPush)
		assert.NoError(t, err)
		_, err = fmt.Fprint(in, "\n")
		assert.NoError(t, err)

		err = revcomm.ReceiveCapabilitiesResponse()
		assert.NoError(t, err)
	})

	t.Run("Missing Capability", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		_, err := fmt.Fprintf(in, "%s\n", git.CapabilityOption)
		assert.NoError(t, err)
		_, err = fmt.Fprintf(in, "%s\n", git.CapabilityFetch)
		assert.NoError(t, err)
		_, err = fmt.Fprint(in, "\n")
		assert.NoError(t, err)

		err = revcomm.ReceiveCapabilitiesResponse()
		assert.Error(t, err)
	})

	t.Run("Unrecognized Capability", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		_, err := fmt.Fprintf(in, "%s\n", git.Capability("foo"))
		assert.NoError(t, err)
		_, err = fmt.Fprint(in, "\n")
		assert.NoError(t, err)

		err = revcomm.ReceiveCapabilitiesResponse()
		assert.Error(t, err)
	})
}

func Test_reverseCommunicator_ReceiveOptionResponse(t *testing.T) {
	t.Run("Option Supported", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		_, err := fmt.Fprintf(in, "%s\n", git.OptionSupported)
		assert.NoError(t, err)

		err = revcomm.ReceiveOptionResponse()
		assert.NoError(t, err)
	})

	t.Run("Option Not Supported", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		_, err := fmt.Fprintf(in, "%s\n", git.OptionNotSupported)
		assert.NoError(t, err)

		err = revcomm.ReceiveOptionResponse()
		assert.NoError(t, err)
	})

	t.Run("Unrecognized Response", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		_, err := fmt.Fprintf(in, "%s\n", git.Option("foo"))
		assert.NoError(t, err)

		err = revcomm.ReceiveOptionResponse()
		assert.Error(t, err)
	})
}

func Test_reverseCommunicator_ReceiveListResponse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		ref1 := "foo"
		ref2 := "bar"
		ref3 := "foobar"
		commit1 := "commit1"
		commit2 := "commit2"
		commit3 := "commit3"

		_, err := fmt.Fprintf(in, "%s %s\n", commit1, ref1)
		assert.NoError(t, err)
		_, err = fmt.Fprintf(in, "%s %s\n", commit2, ref2)
		assert.NoError(t, err)
		_, err = fmt.Fprintf(in, "%s %s\n", commit3, ref3)
		assert.NoError(t, err)
		_, err = fmt.Fprint(in, "\n")
		assert.NoError(t, err)

		err = revcomm.ReceiveListResponse()
		assert.NoError(t, err)
	})

	t.Run("Insufficient Fields", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		commit1 := "commit1"

		_, err := fmt.Fprintf(in, "%s\n", commit1)
		assert.NoError(t, err)
		_, err = fmt.Fprint(in, "\n")
		assert.NoError(t, err)

		err = revcomm.ReceiveListResponse()
		assert.Error(t, err)
	})
}

func Test_reverseCommunicator_ReceivePushResponseBatch(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		remote1 := "foo"
		remote2 := "bar"
		remote3 := "foobar"

		_, err := fmt.Fprintf(in, "ok %s\n", remote1)
		assert.NoError(t, err)
		_, err = fmt.Fprintf(in, "ok %s\n", remote2)
		assert.NoError(t, err)
		_, err = fmt.Fprintf(in, "ok %s\n", remote3)
		assert.NoError(t, err)
		_, err = fmt.Fprint(in, "\n")
		assert.NoError(t, err)

		err = revcomm.ReceivePushResponseBatch()
		assert.NoError(t, err)
	})

	t.Run("Malformed ok Response", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		remote1 := "foo"

		_, err := fmt.Fprintf(in, "ok too many fields%s\n", remote1)
		assert.NoError(t, err)
		_, err = fmt.Fprint(in, "\n")
		assert.NoError(t, err)

		err = revcomm.ReceivePushResponseBatch()
		assert.Error(t, err)
	})

	t.Run("Success with Push Errors", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		remote1 := "foo"
		remote2 := "bar"
		remote3 := "foobar"

		_, err := fmt.Fprintf(in, "ok %s\n", remote1)
		assert.NoError(t, err)
		_, err = fmt.Fprintf(in, "error %s something bad happened\n", remote2)
		assert.NoError(t, err)
		_, err = fmt.Fprintf(in, "error %s oh no not again\n", remote3)
		assert.NoError(t, err)
		_, err = fmt.Fprint(in, "\n")
		assert.NoError(t, err)

		err = revcomm.ReceivePushResponseBatch()
		assert.NoError(t, err)
	})

	t.Run("Malformed error Response", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		remote1 := "foo"
		remote2 := "bar"

		_, err := fmt.Fprintf(in, "ok %s\n", remote1)
		assert.NoError(t, err)
		_, err = fmt.Fprintf(in, "error %s\n", remote2)
		assert.NoError(t, err)
		_, err = fmt.Fprint(in, "\n")
		assert.NoError(t, err)

		err = revcomm.ReceivePushResponseBatch()
		assert.Error(t, err)
	})

	t.Run("Malformed Response", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		_, err := in.Write([]byte("foo bar\n"))
		assert.NoError(t, err)

		err = revcomm.ReceivePushResponseBatch()
		assert.Error(t, err)
	})
}

func Test_reverseCommunicator_ReceiveFetchResponse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		_, err := in.Write([]byte("\n"))
		assert.NoError(t, err)

		err = revcomm.ReceiveListResponse()
		assert.NoError(t, err)
	})

	t.Run("Unexpected Response", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicator(in, nil)

		_, err := in.Write([]byte("fooo\n"))
		assert.NoError(t, err)

		err = revcomm.ReceiveFetchResponse()
		assert.Error(t, err)
	})
}
