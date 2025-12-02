package git

import (
	"fmt"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
)

func TestListRequest_Parse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		fields := []string{string(List)}

		var req ListRequest
		err := req.Parse(fields)
		assert.NoError(t, err)
		assert.Equal(t, List, req.Cmd)
		assert.False(t, req.ForPush)
	})

	t.Run("Success - ForPush", func(t *testing.T) {
		fields := []string{string(List), "for-push"}

		var req ListRequest
		err := req.Parse(fields)
		assert.NoError(t, err)
		assert.Equal(t, List, req.Cmd)
		assert.True(t, req.ForPush)
	})

	t.Run("Empty", func(t *testing.T) {
		fields := []string{}

		var req ListRequest
		err := req.Parse(fields)
		assert.ErrorIs(t, err, ErrBadRequest)
	})

	t.Run("Nil", func(t *testing.T) {
		var req ListRequest
		err := req.Parse(nil)
		assert.ErrorIs(t, err, ErrBadRequest)
	})

	t.Run("Unexpected Request", func(t *testing.T) {
		fields := []string{"foo"}

		var req ListRequest
		err := req.Parse(fields)
		assert.ErrorIs(t, err, ErrUnexpectedRequest)
	})
}

func TestListRequest_String(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		req := ListRequest{
			Cmd: List,
		}

		str := req.String()
		assert.Equal(t, string(List), str)
	})

	t.Run("Success - ForPush", func(t *testing.T) {
		req := ListRequest{
			Cmd:     List,
			ForPush: true,
		}

		str := req.String()
		assert.Equal(t, string(List)+" for-push", str)
	})
}

func TestListResponse_String(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		hash := plumbing.ComputeHash(plumbing.CommitObject, []byte("foobar"))
		refName := plumbing.ReferenceName("foo")

		resp := ListResponse{
			Reference: refName,
			Commit:    hash.String(),
		}

		str := resp.String()
		assert.Equal(t, fmt.Sprintf("%s %s", hash.String(), refName.String()), str)
	})
}
