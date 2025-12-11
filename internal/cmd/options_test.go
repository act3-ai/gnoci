package cmd

import (
	"bytes"
	"testing"

	"github.com/act3-ai/gnoci/internal/testutils"
	"github.com/act3-ai/gnoci/pkg/protocol/git"
	"github.com/act3-ai/gnoci/pkg/protocol/git/comms"
	"github.com/stretchr/testify/assert"
)

func TestHandleOption(t *testing.T) {
	t.Run("Success - Verbosity Debug", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := comms.NewCommunicator(in, out)
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendOptionRequest(git.Verbosity, "10")
		assert.NoError(t, err)

		err = HandleOption(t.Context(), comm)
		assert.NoError(t, err)

		err = revcomm.ReceiveOptionResponse()
		assert.NoError(t, err)
	})

	t.Run("Success - Verbosity Info", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := comms.NewCommunicator(in, out)
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendOptionRequest(git.Verbosity, "2")
		assert.NoError(t, err)

		err = HandleOption(t.Context(), comm)
		assert.NoError(t, err)

		err = revcomm.ReceiveOptionResponse()
		assert.NoError(t, err)
	})

	t.Run("Success - Verbosity Warn", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := comms.NewCommunicator(in, out)
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendOptionRequest(git.Verbosity, "1")
		assert.NoError(t, err)

		err = HandleOption(t.Context(), comm)
		assert.NoError(t, err)

		err = revcomm.ReceiveOptionResponse()
		assert.NoError(t, err)
	})

	t.Run("Success - Verbosity Error", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := comms.NewCommunicator(in, out)
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendOptionRequest(git.Verbosity, "-1")
		assert.NoError(t, err)

		err = HandleOption(t.Context(), comm)
		assert.NoError(t, err)

		err = revcomm.ReceiveOptionResponse()
		assert.NoError(t, err)
	})

	t.Run("Unsupported Option", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := comms.NewCommunicator(in, out)
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendOptionRequest(git.Option("foo"), "bar")
		assert.NoError(t, err)

		err = HandleOption(t.Context(), comm)
		assert.NoError(t, err)

		err = revcomm.ReceiveOptionResponse()
		assert.NoError(t, err)
	})

	t.Run("Verbosity Invalid Value", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := comms.NewCommunicator(in, out)
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendOptionRequest(git.Verbosity, "foo")
		assert.NoError(t, err)

		err = HandleOption(t.Context(), comm)
		assert.Error(t, err)
	})
}
