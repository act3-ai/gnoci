package cmd

import (
	"bytes"
	"testing"

	"github.com/act3-ai/gnoci/internal/testutils"
	"github.com/act3-ai/gnoci/pkg/protocol/git/comms"
	"github.com/stretchr/testify/assert"
)

func TestHandleCapabilities(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := comms.NewCommunicator(in, out)
		revcomm := testutils.NewReverseCommunicator(out, in)

		err := revcomm.SendCapabilitiesRequest()
		assert.NoError(t, err)

		err = HandleCapabilities(t.Context(), comm)
		assert.NoError(t, err)

		err = revcomm.ReceiveCapabilitiesResponse()
		assert.NoError(t, err)
	})

	t.Run("Invalid Capabilities Request", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := comms.NewCommunicator(in, out)
		revcomm := testutils.NewReverseCommunicator(out, in)

		// send something unexpected
		err := revcomm.SendListRequest(false)
		assert.NoError(t, err)

		err = HandleCapabilities(t.Context(), comm)
		assert.Error(t, err)
	})
}
