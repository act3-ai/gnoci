package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapabilitiesRequest_Parse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		fields := []string{string(Capabilities)}

		var req CapabilitiesRequest
		err := req.Parse(fields)
		assert.NoError(t, err)
		assert.Equal(t, Capabilities, req.Cmd)
	})

	t.Run("Empty", func(t *testing.T) {
		fields := []string{}

		var req CapabilitiesRequest
		err := req.Parse(fields)
		assert.ErrorIs(t, err, ErrBadRequest)
	})

	t.Run("Nil", func(t *testing.T) {
		var req CapabilitiesRequest
		err := req.Parse(nil)
		assert.ErrorIs(t, err, ErrBadRequest)
	})

	t.Run("Unexpected Request", func(t *testing.T) {
		fields := []string{"foo"}

		var req CapabilitiesRequest
		err := req.Parse(fields)
		assert.ErrorIs(t, err, ErrUnexpectedRequest)
	})
}

func TestCapabilitiesRequest_String(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		req := CapabilitiesRequest{
			Cmd: Capabilities,
		}

		str := req.String()
		assert.Equal(t, string(Capabilities), str)
	})
}
