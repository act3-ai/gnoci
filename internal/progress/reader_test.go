// Package progress facilitates tracking progress for a standard library io.Reader.
package progress

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/act3-ai/gnoci/internal/mocks/iomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewEvalReadCloser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		rc := io.NopCloser(strings.NewReader("foo"))

		evalrc := NewEvalReadCloser(rc)
		assert.NotNil(t, evalrc)

		erc, ok := evalrc.(*readCloser)
		assert.True(t, ok)
		assert.NotNil(t, erc)
		assert.Equal(t, rc, erc.rc)
	})
}

func Test_readCloser_Read(t *testing.T) {
	const content = "foo"
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rcMock := iomock.NewMockReadCloser(ctrl)

		rcMock.EXPECT().
			Read([]byte(content)).
			Return(len(content), nil)

		rc := &readCloser{
			rc: rcMock,
		}

		n, err := rc.Read([]byte(content))
		assert.NoError(t, err)
		assert.Equal(t, len(content), n)
		assert.Equal(t, len(content), rc.total)
		assert.Equal(t, len(content), rc.delta)
		assert.Equal(t, nil, rc.err)
	})

	t.Run("Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rcMock := iomock.NewMockReadCloser(ctrl)

		expectedErr := errors.New("read error")
		rcMock.EXPECT().
			Read([]byte(content)).
			Return(len(content), expectedErr)

		rc := &readCloser{
			rc: rcMock,
		}

		n, err := rc.Read([]byte(content))
		assert.Error(t, err)
		assert.Equal(t, len(content), n)
		assert.Equal(t, len(content), rc.total)
		assert.Equal(t, len(content), rc.delta)
		assert.Equal(t, expectedErr, rc.err)
	})
}

func Test_readCloser_Progress(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		expectedTotal := 10
		expectedDelta := 5
		rc := &readCloser{
			rc:    nil,
			total: expectedTotal,
			delta: expectedDelta,
		}

		actualSoFar, actualSinceLast, err := rc.Progress()
		assert.NoError(t, err)
		assert.Equal(t, nil, rc.err)
		assert.Equal(t, expectedTotal, actualSoFar)
		assert.Equal(t, expectedDelta, actualSinceLast)
		assert.Equal(t, rc.delta, 0)
	})

	t.Run("Error", func(t *testing.T) {
		expectedTotal := 10
		expectedDelta := 5
		expectedErr := errors.New("read error")
		rc := &readCloser{
			rc:    nil,
			total: expectedTotal,
			delta: expectedDelta,
			err:   expectedErr,
		}

		actualSoFar, actualSinceLast, err := rc.Progress()
		assert.Error(t, err)
		assert.Equal(t, expectedErr, rc.err)
		assert.Equal(t, expectedTotal, actualSoFar)
		assert.Equal(t, expectedDelta, actualSinceLast)
		assert.Equal(t, rc.delta, 0)
	})
}

func Test_readCloser_Close(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rcMock := iomock.NewMockReadCloser(ctrl)

		rcMock.EXPECT().
			Close().
			Return(nil)

		rc := &readCloser{
			rc: rcMock,
		}

		err := rc.Close()
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rcMock := iomock.NewMockReadCloser(ctrl)

		expectedErr := errors.New("close error")
		rcMock.EXPECT().
			Close().
			Return(expectedErr)

		rc := &readCloser{
			rc: rcMock,
		}

		err := rc.Close()
		assert.ErrorIs(t, err, expectedErr)
	})
}
