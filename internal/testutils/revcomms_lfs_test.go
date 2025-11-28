package testutils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/act3-ai/gnoci/pkg/protocol/lfs"
	"github.com/stretchr/testify/assert"
)

func TestNewReverseCommunicatorLFS(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := strings.NewReader("foo")
		out := new(bytes.Buffer)

		revCommI := NewReverseCommunicatorLFS(in, out)
		assert.NotNil(t, revCommI)

		revComm, ok := revCommI.(*reverseCommunicatorLFS)
		assert.True(t, ok)
		assert.NotNil(t, revComm)
		assert.NotNil(t, revComm.in)
		assert.Equal(t, out, revComm.out)
	})
}

func Test_reverseCommunicatorLFS_SendInitRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		out := new(bytes.Buffer)

		revcomm := NewReverseCommunicatorLFS(nil, out)

		err := revcomm.SendInitRequest(lfs.UploadOperation, "foo/bar/foofoo/barbar.foo")
		assert.NoError(t, err)

		outRaw, err := io.ReadAll(out)
		assert.NoError(t, err)

		var req lfs.InitRequest
		err = json.Unmarshal(outRaw, &req)
		assert.NoError(t, err)

		err = req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Invalid Request", func(t *testing.T) {
		out := new(bytes.Buffer)

		revcomm := NewReverseCommunicatorLFS(nil, out)

		err := revcomm.SendInitRequest(lfs.Operation("foo"), "foo/bar/foofoo/barbar.foo")
		assert.Error(t, err)

	})
}

func Test_reverseCommunicatorLFS_ReceiveInitResponse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicatorLFS(in, nil)

		raw := []byte("{}")
		_, err := in.Write(withNewline(raw))
		assert.NoError(t, err)

		err = revcomm.ReceiveInitResponse()
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicatorLFS(in, nil)

		initResp := lfs.InitResponse{
			Error: lfs.ErrCodeMessage{
				Code:    1,
				Message: errors.New("init error").Error(),
			},
		}

		raw, err := json.Marshal(initResp)
		assert.NoError(t, err)

		_, err = in.Write(withNewline(raw))
		assert.NoError(t, err)

		err = revcomm.ReceiveInitResponse()
		assert.Error(t, err)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicatorLFS(in, nil)

		raw := []byte("invalid json")
		_, err := in.Write(withNewline(raw))
		assert.NoError(t, err)

		err = revcomm.ReceiveInitResponse()
		assert.Error(t, err)
	})
}

func Test_reverseCommunicatorLFS_SendTransferUploadRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		out := new(bytes.Buffer)

		revcomm := NewReverseCommunicatorLFS(nil, out)

		const (
			oid  = "123456789"
			size = int64(10)
			path = "path/foo"
		)

		err := revcomm.SendTransferUploadRequest(oid, size, path)
		assert.NoError(t, err)

		outRaw, err := io.ReadAll(out)
		assert.NoError(t, err)

		var req lfs.TransferRequest
		err = json.Unmarshal(outRaw, &req)
		assert.NoError(t, err)

		// validate is not sufficient by itself
		err = req.Validate()
		assert.NoError(t, err)

		assert.Equal(t, lfs.UploadEvent, req.Event)
		assert.Equal(t, oid, req.Oid)
		assert.Equal(t, size, req.Size)
		assert.Equal(t, path, req.Path)
	})

	t.Run("Mission Oid", func(t *testing.T) {
		out := new(bytes.Buffer)

		revcomm := NewReverseCommunicatorLFS(nil, out)

		const (
			oid  = ""
			size = int64(10)
			path = "path/foo"
		)

		err := revcomm.SendTransferUploadRequest(oid, size, path)
		assert.ErrorIs(t, err, lfs.ErrEmptyOID)
	})

	t.Run("Invalid Size", func(t *testing.T) {
		out := new(bytes.Buffer)

		revcomm := NewReverseCommunicatorLFS(nil, out)

		const (
			oid  = "123456789"
			size = int64(0)
			path = "path/foo"
		)

		err := revcomm.SendTransferUploadRequest(oid, size, path)
		assert.ErrorIs(t, err, lfs.ErrInvalidSize)
	})

	t.Run("Missing Path", func(t *testing.T) {
		out := new(bytes.Buffer)

		revcomm := NewReverseCommunicatorLFS(nil, out)

		const (
			oid  = "123456789"
			size = int64(10)
			path = ""
		)

		err := revcomm.SendTransferUploadRequest(oid, size, path)
		assert.ErrorIs(t, err, lfs.ErrEmptyPath)
	})
}

func Test_reverseCommunicatorLFS_SendTransferDownloadRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		out := new(bytes.Buffer)

		revcomm := NewReverseCommunicatorLFS(nil, out)

		const (
			oid  = "123456789"
			size = int64(10)
		)

		err := revcomm.SendTransferDownloadRequest(oid, size)
		assert.NoError(t, err)

		outRaw, err := io.ReadAll(out)
		assert.NoError(t, err)

		var req lfs.TransferRequest
		err = json.Unmarshal(outRaw, &req)
		assert.NoError(t, err)

		// validate is not sufficient by itself
		err = req.Validate()
		assert.NoError(t, err)

		assert.Equal(t, lfs.DownloadEvent, req.Event)
		assert.Equal(t, oid, req.Oid)
		assert.Equal(t, size, req.Size)
	})

	t.Run("Missing Oid", func(t *testing.T) {
		out := new(bytes.Buffer)

		revcomm := NewReverseCommunicatorLFS(nil, out)

		const (
			oid  = ""
			size = int64(10)
		)

		err := revcomm.SendTransferDownloadRequest(oid, size)
		assert.ErrorIs(t, err, lfs.ErrEmptyOID)
	})

	t.Run("Invalid Size", func(t *testing.T) {
		out := new(bytes.Buffer)

		revcomm := NewReverseCommunicatorLFS(nil, out)

		const (
			oid  = "123456789"
			size = int64(-1)
		)

		err := revcomm.SendTransferDownloadRequest(oid, size)
		assert.ErrorIs(t, err, lfs.ErrInvalidSize)
	})
}

func Test_reverseCommunicatorLFS_ReceiveProgressResponse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicatorLFS(in, nil)

		req := &lfs.ProgressResponse{
			Event:          lfs.ProgessEvent,
			Oid:            "123456789",
			BytesSoFar:     10,
			BytesSinceLast: 5,
		}

		raw, err := json.Marshal(req)
		assert.NoError(t, err)

		_, err = in.Write(withNewline(raw))
		assert.NoError(t, err)

		err = revcomm.ReceiveProgressResponse()
		assert.NoError(t, err)
	})

	t.Run("Invalid Event", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicatorLFS(in, nil)

		req := &lfs.ProgressResponse{
			Event:          lfs.Event("foo"),
			Oid:            "123456789",
			BytesSoFar:     10,
			BytesSinceLast: 5,
		}

		raw, err := json.Marshal(req)
		assert.NoError(t, err)

		_, err = in.Write(withNewline(raw))
		assert.NoError(t, err)

		err = revcomm.ReceiveProgressResponse()
		assert.Error(t, err)
	})

	t.Run("Missing Oid", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicatorLFS(in, nil)

		req := &lfs.ProgressResponse{
			Event:          lfs.ProgessEvent,
			Oid:            "",
			BytesSoFar:     10,
			BytesSinceLast: 5,
		}

		raw, err := json.Marshal(req)
		assert.NoError(t, err)

		_, err = in.Write(withNewline(raw))
		assert.NoError(t, err)

		err = revcomm.ReceiveProgressResponse()
		assert.Error(t, err)
	})
}

func Test_reverseCommunicatorLFS_ReceiveTransferResponse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicatorLFS(in, nil)

		req := &lfs.TransferResponse{
			Event: lfs.CompleteEvent,
			Oid:   "123456789",
			Path:  "path/foo",
		}

		raw, err := json.Marshal(req)
		assert.NoError(t, err)

		_, err = in.Write(withNewline(raw))
		assert.NoError(t, err)

		err = revcomm.ReceiveTransferResponse(lfs.DownloadEvent)
		assert.NoError(t, err)
	})

	t.Run("Missing Path - Download", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicatorLFS(in, nil)

		req := &lfs.TransferResponse{
			Event: lfs.CompleteEvent,
			Oid:   "123456789",
			Path:  "",
		}

		raw, err := json.Marshal(req)
		assert.NoError(t, err)

		_, err = in.Write(withNewline(raw))
		assert.NoError(t, err)

		err = revcomm.ReceiveTransferResponse(lfs.DownloadEvent)
		assert.Error(t, err)
	})

	t.Run("Missing Path - Upload", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicatorLFS(in, nil)

		req := &lfs.TransferResponse{
			Event: lfs.CompleteEvent,
			Oid:   "123456789",
			Path:  "",
		}

		raw, err := json.Marshal(req)
		assert.NoError(t, err)

		_, err = in.Write(withNewline(raw))
		assert.NoError(t, err)

		err = revcomm.ReceiveTransferResponse(lfs.UploadEvent)
		assert.NoError(t, err)
	})

	t.Run("Missing Oid", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicatorLFS(in, nil)

		req := &lfs.TransferResponse{
			Event: lfs.CompleteEvent,
			Oid:   "",
			Path:  "path/foo",
		}

		raw, err := json.Marshal(req)
		assert.NoError(t, err)

		_, err = in.Write(withNewline(raw))
		assert.NoError(t, err)

		err = revcomm.ReceiveTransferResponse(lfs.DownloadEvent)
		assert.Error(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		// this test intentionally does not use a comms.Communicator, incase
		// bugs are introduced.
		in := new(bytes.Buffer)
		revcomm := NewReverseCommunicatorLFS(in, nil)

		req := &lfs.TransferResponse{
			Event: lfs.CompleteEvent,
			Oid:   "123456789",
			Path:  "path/foo",
			Error: lfs.ErrCodeMessage{
				Code:    1,
				Message: errors.New("transfer error").Error(),
			},
		}

		raw, err := json.Marshal(req)
		assert.NoError(t, err)

		_, err = in.Write(withNewline(raw))
		assert.NoError(t, err)

		err = revcomm.ReceiveTransferResponse(lfs.DownloadEvent)
		assert.Error(t, err)
	})
}

func Test_reverseCommunicatorLFS_SendTerminateRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		out := new(bytes.Buffer)

		revcomm := NewReverseCommunicatorLFS(nil, out)

		err := revcomm.SendTerminateRequest()
		assert.NoError(t, err)

		outRaw, err := io.ReadAll(out)
		assert.NoError(t, err)

		var req lfs.TransferRequest
		err = json.Unmarshal(outRaw, &req)
		assert.NoError(t, err)

		assert.Equal(t, lfs.TerminateEvent, req.Event)
	})
}

func Test_reverseCommunicatorLFS_readLine(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)

		revcomm := &reverseCommunicatorLFS{
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

		revcomm := &reverseCommunicatorLFS{
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
