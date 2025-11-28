// Package comms facilitates receiving requests from and writing responses to git-lfs.
package comms

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/act3-ai/gnoci/pkg/protocol/lfs"
)

// Communicator provides handling of git-lfs transfer protocol
// requests and responses.
type Communicator interface {
	RequestHandler
	ResponseHandler
}

// RequestHandler receives git-lfs transfer protocol requests.
type RequestHandler interface {
	// ReceiveInitRequest reads and validates an [lfs.InitRequest].
	ReceiveInitRequest(ctx context.Context) (*lfs.InitRequest, error)
	// ReceiveTransferRequest reads a [lfs.TransferRequest].
	ReceiveTransferRequest(ctx context.Context) (*lfs.TransferRequest, error)
}

// ResponseHandler sends git-lfs transfer protocol responses.
type ResponseHandler interface {
	// WriteInitResponse responds to a [lfs.TransferRequest].
	// Returns err joined with any response handling errors.
	WriteInitResponse(ctx context.Context, err error) error
	// WriteProgress sends a [lfs.ProgressResponse] message, after an
	// [lfs.TransferRequest], and before an [lfs.TransferResponse].
	WriteProgress(ctx context.Context, oid string, soFar, sinceLast int) error
	// WriteTransferUploadResponse is the final response to an [lfs.TransferRequest],
	// with event type [lfs.UploadEvent] after zero or more
	// [ResponseHandler.WriteProgress] messages.
	WriteTransferUploadResponse(ctx context.Context, oid string, err error) error
	// WriteTransferDownloadResponse is the final response to an
	// [lfs.TransferRequest], with event type [lfs.DownloadEvent] after zero or more
	// [ResponseHandler.WriteProgress] messages.
	WriteTransferDownloadResponse(ctx context.Context, oid string, path string, err error) error
}

// NewCommunicator initializes a [Communicator].
func NewCommunicator(in io.Reader, out io.Writer) Communicator {
	return &defaultCommunicator{
		in:  bufio.NewScanner(in),
		out: out,
	}
}

// defaultCommunicator is the default implementation of [Communicator].
type defaultCommunicator struct {
	in  *bufio.Scanner
	out io.Writer
}

// ReceiveInitRequest reads and validates an [lfs.InitRequest].
func (c *defaultCommunicator) ReceiveInitRequest(ctx context.Context) (*lfs.InitRequest, error) {
	slog.DebugContext(ctx, "receiving InitRequest from git-lfs")
	line, err := c.readLine()
	if err != nil {
		return nil, err
	}

	var initReq *lfs.InitRequest
	if err := json.Unmarshal(line, &initReq); err != nil {
		return nil, fmt.Errorf("decoding InitRequest: %w", err)
	}

	if err := initReq.Validate(); err != nil {
		return initReq, err
	}

	return initReq, nil
}

// ReceiveTransferRequest reads a [lfs.TransferRequest].
func (c *defaultCommunicator) ReceiveTransferRequest(ctx context.Context) (*lfs.TransferRequest, error) {
	slog.DebugContext(ctx, "receiving TransferRequest from git-lfs")
	line, err := c.readLine()
	if err != nil {
		return nil, fmt.Errorf("reading TransferRequest: %w", err)
	}

	var transferReq *lfs.TransferRequest
	if err := json.Unmarshal(line, &transferReq); err != nil {
		return nil, fmt.Errorf("decoding TransferRequest: %w", err)
	}

	return transferReq, nil
}

// WriteInitResponse responds to a [lfs.TransferRequest].
// Returns err joined with any response handling errors.
func (c *defaultCommunicator) WriteInitResponse(ctx context.Context, initErr error) error {
	slog.DebugContext(ctx, "writing InitResponse to git-lfs")

	// TODO: is this necessary or can we marshal with an empty error?
	// TODO: test addition of "omitempty"
	raw := []byte("{}")
	if initErr != nil {
		initResp := lfs.InitResponse{
			Error: lfs.ErrCodeMessage{
				Code:    1,
				Message: initErr.Error(),
			},
		}

		var err error
		raw, err = json.Marshal(initResp)
		if err != nil {
			return errors.Join(initErr, fmt.Errorf("encoding init response: %w", err))
		}
	}

	if _, err := c.out.Write(withNewline(raw)); err != nil {
		return errors.Join(initErr, fmt.Errorf("writing init response: %w", err))
	}

	return initErr
}

// WriteProgress sends transfer progress information, after an [lfs.TransferRequest],
// and before an [lfs.TransferResponse].
func (c *defaultCommunicator) WriteProgress(ctx context.Context, oid string, soFar, sinceLast int) error {
	slog.DebugContext(ctx, "writing progress response",
		slog.String("oid", oid),
		slog.Int("soFar", soFar),
		slog.Int("sinceLast", sinceLast))

	progressResp := lfs.ProgressResponse{
		Event:          lfs.ProgessEvent,
		Oid:            oid,
		BytesSoFar:     soFar,
		BytesSinceLast: sinceLast,
	}

	if err := progressResp.Validate(); err != nil {
		return err
	}

	raw, err := json.Marshal(progressResp)
	if err != nil {
		return fmt.Errorf("encoding progress response: %w", err)
	}

	if _, err := c.out.Write(withNewline(raw)); err != nil {
		return fmt.Errorf("writing progress response: %w", err)
	}

	return nil
}

// WriteTransferUploadResponse is the final response to an [lfs.TransferRequest],
// with event type [lfs.UploadEvent] after zero or more
// [ResponseHandler.WriteProgress] messages.
func (c *defaultCommunicator) WriteTransferUploadResponse(ctx context.Context, oid string, err error) error {
	log := slog.With(slog.String("oid", oid))
	log.DebugContext(ctx, "writing transfer response")

	transferResp := lfs.TransferResponse{
		Event: lfs.CompleteEvent,
		Oid:   oid,
	}
	if err != nil {
		transferResp.Error = lfs.ErrCodeMessage{
			Code:    1,
			Message: err.Error(),
		}
	}

	if err := transferResp.Validate(lfs.UploadEvent); err != nil {
		return err
	}

	raw, err := json.Marshal(transferResp)
	if err != nil {
		return fmt.Errorf("encoding transfer response: %w", err)
	}

	if _, err := c.out.Write(withNewline(raw)); err != nil {
		log.ErrorContext(ctx, "writing transfer response", slog.String("error", err.Error()))
		return fmt.Errorf("writing transfer response: %w", err)
	}

	return nil
}

// WriteTransferDownloadResponse is the final response to an
// [lfs.TransferRequest], with event type [lfs.DownloadEvent] after zero or more
// [ResponseHandler.WriteProgress] messages.
func (c *defaultCommunicator) WriteTransferDownloadResponse(ctx context.Context, oid string, path string, err error) error {
	log := slog.With(slog.String("oid", oid))
	log.DebugContext(ctx, "writing transfer response")

	transferResp := lfs.TransferResponse{
		Event: lfs.CompleteEvent,
		Path:  path,
		Oid:   oid,
	}
	if err != nil {
		transferResp.Error = lfs.ErrCodeMessage{
			Code:    1,
			Message: err.Error(),
		}
	}

	if err := transferResp.Validate(lfs.DownloadEvent); err != nil {
		return err
	}

	raw, err := json.Marshal(transferResp)
	if err != nil {
		return fmt.Errorf("encoding transfer response: %w", err)
	}

	if _, err := c.out.Write(withNewline(raw)); err != nil {
		log.ErrorContext(ctx, "writing transfer response", slog.String("error", err.Error()))
		return fmt.Errorf("writing transfer response: %w", err)
	}

	return nil
}

func (c *defaultCommunicator) readLine() ([]byte, error) {
	ok := c.in.Scan()
	switch {
	case !ok && c.in.Err() != nil:
		return nil, fmt.Errorf("reading single command from git-lfs: %w", c.in.Err())
	case !ok:
		// EOF
		return nil, nil
	default:
		return c.in.Bytes(), nil
	}
}

func withNewline(line []byte) []byte {
	return append(line, []byte("\n")...) // TODO: can we use a rune?
}
