package git

import (
	"fmt"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
)

func TestOptionRequest_Parse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		opt := Verbosity
		value := "10"

		expectedReq := OptionRequest{
			Cmd:   Options,
			Opt:   opt,
			Value: value,
		}
		fields := []string{string(Options), string(opt), value}

		var req OptionRequest
		err := req.Parse(fields)
		assert.NoError(t, err)
		assert.Equal(t, expectedReq, req)
	})

	t.Run("Insufficient Fields", func(t *testing.T) {
		fields := []string{string(Options), string(Verbosity)}

		var req OptionRequest
		err := req.Parse(fields)
		assert.ErrorIs(t, err, ErrBadRequest)
	})

	t.Run("Unexpected Request", func(t *testing.T) {
		hash := plumbing.ComputeHash(plumbing.CommitObject, []byte("foobar"))
		refName := "foo"

		fields := []string{string(Push), hash.String(), refName}

		var req OptionRequest
		err := req.Parse(fields)
		assert.ErrorIs(t, err, ErrUnexpectedRequest)
	})

	t.Run("Nil", func(t *testing.T) {
		var req OptionRequest
		err := req.Parse(nil)
		assert.ErrorIs(t, err, ErrBadRequest)
	})
}

func TestOptionRequest_String(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		opt := Verbosity
		value := "10"

		req := OptionRequest{
			Cmd:   Options,
			Opt:   opt,
			Value: value,
		}

		str := req.String()
		assert.Equal(t, fmt.Sprintf("%s %s %s", Options, opt, value), str)
	})
}
