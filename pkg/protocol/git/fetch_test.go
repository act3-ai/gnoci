package git

import (
	"fmt"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
)

func TestFetchRequest_Parse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		hash := plumbing.ComputeHash(plumbing.CommitObject, []byte("foobar"))
		refName := "foo"

		expectedReq := FetchRequest{
			Cmd: Fetch,
			Ref: plumbing.NewHashReference(
				plumbing.ReferenceName(refName),
				hash,
			),
		}
		fields := []string{string(Fetch), hash.String(), refName}

		var req FetchRequest
		err := req.Parse(fields)
		assert.NoError(t, err)
		assert.Equal(t, expectedReq, req)
	})

	t.Run("Insufficient Fields", func(t *testing.T) {
		hash := plumbing.ComputeHash(plumbing.CommitObject, []byte("foobar"))

		fields := []string{string(Fetch), hash.String()}

		var req FetchRequest
		err := req.Parse(fields)
		assert.ErrorIs(t, err, ErrBadRequest)
	})

	t.Run("Unexpected Request", func(t *testing.T) {
		hash := plumbing.ComputeHash(plumbing.CommitObject, []byte("foobar"))
		refName := "foo"

		fields := []string{string(Push), hash.String(), refName}

		var req FetchRequest
		err := req.Parse(fields)
		assert.ErrorIs(t, err, ErrUnexpectedRequest)
	})

	t.Run("Nil", func(t *testing.T) {
		var req FetchRequest
		err := req.Parse(nil)
		assert.ErrorIs(t, err, ErrBadRequest)
	})
}

func TestFetchRequest_String(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		hash := plumbing.ComputeHash(plumbing.CommitObject, []byte("foobar"))
		refName := "foo"

		req := FetchRequest{
			Cmd: Fetch,
			Ref: plumbing.NewHashReference(
				plumbing.ReferenceName(refName),
				hash,
			),
		}

		str := req.String()
		assert.Equal(t, fmt.Sprintf("%s %s %s", Fetch, hash, refName), str)
	})
}
