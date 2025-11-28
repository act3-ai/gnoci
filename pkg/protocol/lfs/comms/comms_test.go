// Package comms facilitates receiving requests from and writing responses to git-lfs.
package comms

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/act3-ai/gnoci/internal/testutils"
	"github.com/act3-ai/gnoci/pkg/protocol/lfs"
	"github.com/stretchr/testify/assert"
)

const (
	testRemote = "example.com/foo/bar"
	oid        = "123456789"
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

func Test_defaultCommunicator_ReceiveInitRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		revcomm := testutils.NewReverseCommunicatorLFS(lfsIn, lfsOut)
		comm := NewCommunicator(lfsOut, lfsIn)

		err := revcomm.SendInitRequest(lfs.UploadOperation, testRemote)
		assert.NoError(t, err)

		req, err := comm.ReceiveInitRequest(t.Context())
		assert.NoError(t, err)

		err = req.Validate()
		assert.NoError(t, err)
	})
}

func Test_defaultCommunicator_ReceiveTransferRequest(t *testing.T) {
	t.Run("Success - Download", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		revcomm := testutils.NewReverseCommunicatorLFS(lfsIn, lfsOut)
		comm := NewCommunicator(lfsOut, lfsIn)

		err := revcomm.SendTransferDownloadRequest(oid, int64(len(oid)))
		assert.NoError(t, err)

		req, err := comm.ReceiveTransferRequest(t.Context())
		assert.NoError(t, err)

		err = req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Success - Upload", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		revcomm := testutils.NewReverseCommunicatorLFS(lfsIn, lfsOut)
		comm := NewCommunicator(lfsOut, lfsIn)

		err := revcomm.SendTransferUploadRequest(oid, int64(len(oid)), "path/foo")
		assert.NoError(t, err)

		req, err := comm.ReceiveTransferRequest(t.Context())
		assert.NoError(t, err)

		err = req.Validate()
		assert.NoError(t, err)
	})
}

func Test_defaultCommunicator_WriteInitResponse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		revcomm := testutils.NewReverseCommunicatorLFS(lfsIn, lfsOut)
		comm := NewCommunicator(lfsOut, lfsIn)

		err := comm.WriteInitResponse(t.Context(), nil)
		assert.NoError(t, err)

		err = revcomm.ReceiveInitResponse()
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		revcomm := testutils.NewReverseCommunicatorLFS(lfsIn, lfsOut)
		comm := NewCommunicator(lfsOut, lfsIn)

		testErr := errors.New("init error")
		err := comm.WriteInitResponse(t.Context(), testErr)
		assert.ErrorIs(t, err, testErr) // ensure we're wrapping the error and returning it

		err = revcomm.ReceiveInitResponse()
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), testErr.Error()))
	})
}

func Test_defaultCommunicator_WriteProgress(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		soFar := 10
		sinceLast := 5

		revcomm := testutils.NewReverseCommunicatorLFS(lfsIn, lfsOut)
		comm := NewCommunicator(lfsOut, lfsIn)

		err := comm.WriteProgress(t.Context(), oid, soFar, sinceLast)
		assert.NoError(t, err)

		err = revcomm.ReceiveProgressResponse()
		assert.NoError(t, err)
	})

	t.Run("Invalid BytesSoFar Value", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		soFar := -1
		sinceLast := 5

		comm := NewCommunicator(lfsOut, lfsIn)

		err := comm.WriteProgress(t.Context(), oid, soFar, sinceLast)
		assert.ErrorIs(t, err, lfs.ErrInvalidProgressValue)
	})

	t.Run("Invalid BytesSinceLast Value", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		soFar := 10
		sinceLast := -1

		comm := NewCommunicator(lfsOut, lfsIn)

		err := comm.WriteProgress(t.Context(), oid, soFar, sinceLast)
		assert.ErrorIs(t, err, lfs.ErrInvalidProgressValue)
	})
}

func Test_defaultCommunicator_WriteTransferUploadResponse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		revcomm := testutils.NewReverseCommunicatorLFS(lfsIn, lfsOut)
		comm := NewCommunicator(lfsOut, lfsIn)

		err := comm.WriteTransferUploadResponse(t.Context(), oid, nil)
		assert.NoError(t, err)

		err = revcomm.ReceiveTransferResponse(lfs.UploadEvent)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		revcomm := testutils.NewReverseCommunicatorLFS(lfsIn, lfsOut)
		comm := NewCommunicator(lfsOut, lfsIn)

		testErr := errors.New("download error")
		err := comm.WriteTransferUploadResponse(t.Context(), oid, testErr)
		assert.ErrorIs(t, err, testErr) // ensure we're wrapping the error and returning it

		err = revcomm.ReceiveTransferResponse(lfs.UploadEvent)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), testErr.Error()))
	})

	t.Run("Missing Oid", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		revcomm := testutils.NewReverseCommunicatorLFS(lfsIn, lfsOut)
		comm := NewCommunicator(lfsOut, lfsIn)

		err := comm.WriteTransferUploadResponse(t.Context(), "", nil)
		assert.ErrorIs(t, err, lfs.ErrEmptyOID)

		err = revcomm.ReceiveTransferResponse(lfs.UploadEvent)
		assert.Error(t, err)
	})
}

func Test_defaultCommunicator_WriteTransferDownloadResponse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		revcomm := testutils.NewReverseCommunicatorLFS(lfsIn, lfsOut)
		comm := NewCommunicator(lfsOut, lfsIn)

		err := comm.WriteTransferDownloadResponse(t.Context(), oid, "path/foo", nil)
		assert.NoError(t, err)

		err = revcomm.ReceiveTransferResponse(lfs.DownloadEvent)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		revcomm := testutils.NewReverseCommunicatorLFS(lfsIn, lfsOut)
		comm := NewCommunicator(lfsOut, lfsIn)

		testErr := errors.New("download error")
		err := comm.WriteTransferDownloadResponse(t.Context(), oid, "path/foo", testErr)
		assert.ErrorIs(t, err, testErr) // ensure we're wrapping the error and returning it

		err = revcomm.ReceiveTransferResponse(lfs.DownloadEvent)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), testErr.Error()))
	})

	t.Run("Missing Oid", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		revcomm := testutils.NewReverseCommunicatorLFS(lfsIn, lfsOut)
		comm := NewCommunicator(lfsOut, lfsIn)

		err := comm.WriteTransferDownloadResponse(t.Context(), "", "path/foo", nil)
		assert.ErrorIs(t, err, lfs.ErrEmptyOID)

		err = revcomm.ReceiveTransferResponse(lfs.UploadEvent)
		assert.Error(t, err)
	})

	t.Run("Missing Path", func(t *testing.T) {
		lfsIn := new(bytes.Buffer)
		lfsOut := new(bytes.Buffer)

		revcomm := testutils.NewReverseCommunicatorLFS(lfsIn, lfsOut)
		comm := NewCommunicator(lfsOut, lfsIn)

		err := comm.WriteTransferDownloadResponse(t.Context(), oid, "", nil)
		assert.ErrorIs(t, err, lfs.ErrEmptyPath)

		err = revcomm.ReceiveTransferResponse(lfs.DownloadEvent)
		assert.Error(t, err)
	})
}

func Test_defaultCommunicator_readLine(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)

		revcomm := &defaultCommunicator{
			in:  bufio.NewScanner(in),
			out: nil,
		}

		_, err := in.WriteString("foo\n")
		assert.NoError(t, err)

		outRaw, err := revcomm.readLine()
		assert.NoError(t, err)
		assert.Equal(t, string(outRaw), "foo")
	})

	t.Run("EOF", func(t *testing.T) {
		in := new(bytes.Buffer)

		revcomm := &defaultCommunicator{
			in:  bufio.NewScanner(in),
			out: nil,
		}

		_, err := in.WriteString("")
		assert.NoError(t, err)

		outRaw, err := revcomm.readLine()
		assert.NoError(t, err)
		assert.Nil(t, outRaw)
	})
}
