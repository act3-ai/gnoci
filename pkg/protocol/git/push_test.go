package git

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
)

func TestPushRequest_Parse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		src := plumbing.ReferenceName("foo")
		remote := plumbing.ReferenceName("bar")

		expectedReq := PushRequest{
			Cmd:    Push,
			Force:  false,
			Src:    src,
			Remote: remote,
		}
		fields := []string{string(Push), fmt.Sprintf("%s:%s", src, remote)}

		var req PushRequest
		err := req.Parse(fields)
		assert.NoError(t, err)
		assert.Equal(t, expectedReq, req)
	})

	t.Run("Success", func(t *testing.T) {
		src := plumbing.ReferenceName("foo")
		remote := plumbing.ReferenceName("bar")

		expectedReq := PushRequest{
			Cmd:    Push,
			Force:  true,
			Src:    src,
			Remote: remote,
		}
		fields := []string{string(Push), fmt.Sprintf("+%s:%s", src, remote)}

		var req PushRequest
		err := req.Parse(fields)
		assert.NoError(t, err)
		assert.Equal(t, expectedReq, req)
	})

	t.Run("Insufficient Fields", func(t *testing.T) {
		fields := []string{string(Push)}

		var req PushRequest
		err := req.Parse(fields)
		assert.ErrorIs(t, err, ErrBadRequest)
	})

	t.Run("Unexpected Request", func(t *testing.T) {
		hash := plumbing.ComputeHash(plumbing.CommitObject, []byte("foobar"))
		refName := "foo"

		fields := []string{string(Fetch), hash.String(), refName}

		var req PushRequest
		err := req.Parse(fields)
		assert.ErrorIs(t, err, ErrUnexpectedRequest)
	})

	t.Run("Nil", func(t *testing.T) {
		var req PushRequest
		err := req.Parse(nil)
		assert.ErrorIs(t, err, ErrBadRequest)
	})
}

func TestPushRequest_String(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		src := plumbing.ReferenceName("foo")
		remote := plumbing.ReferenceName("bar")

		req := PushRequest{
			Cmd:    Push,
			Force:  false,
			Src:    src,
			Remote: remote,
		}

		str := req.String()
		assert.Equal(t, fmt.Sprintf("%s %s:%s", Push, src, remote), str)
	})

	t.Run("Success - Force", func(t *testing.T) {
		src := plumbing.ReferenceName("foo")
		remote := plumbing.ReferenceName("bar")

		req := PushRequest{
			Cmd:    Push,
			Force:  true,
			Src:    src,
			Remote: remote,
		}

		str := req.String()
		assert.Equal(t, fmt.Sprintf("%s +%s:%s", Push, src, remote), str)
	})
}

func TestPushResponse_String(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		remote := plumbing.ReferenceName("bar")
		resp := PushResponse{
			Remote: remote,
		}

		str := resp.String()
		assert.Equal(t, fmt.Sprintf("ok %s", remote), str)
	})

	t.Run("Error Message", func(t *testing.T) {
		remote := plumbing.ReferenceName("bar")
		err := errors.New("foo error")
		resp := PushResponse{
			Remote: remote,
			Error:  err,
		}

		str := resp.String()
		assert.Equal(t, fmt.Sprintf("error %s %s", remote, err.Error()), str)
	})
}
