package testutils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/act3-ai/gnoci/pkg/protocol/lfs"
)

// ReverseCommunicatorLFS is the reverse of lfs [comms.Communicator], acting as git-lfs
// sending custom transfer protocol requests and receiving responses.
//
// In addition to regular errors, methods return errors if the underlying message
// contains an error.
type ReverseCommunicatorLFS interface {
	// SendInitRequest sends an [lfs.InitRequest].
	SendInitRequest(op lfs.Operation, remote string) error
	// ReceiveInitResponse receives an [lfs.InitResponse].
	ReceiveInitResponse() error
	// SendTransferUploadRequest sends an [lfs.TransferRequest] with event
	// [lfs.UploadEvent].
	SendTransferUploadRequest(oid string, size int64, path string) error
	// SendTransferDownloadRequest sends an [lfs.TransferRequest] with event
	// [lfs.DownloadEvent].
	SendTransferDownloadRequest(oid string, size int64) error
	// ReceiveProgressResponse receives an [lfs.ProgressResponse].
	ReceiveProgressResponse() error
	// ReceiveTransferResponse receives an [lfs.TransferResponse].
	ReceiveTransferResponse() error
	// SendTerminateRequest sends an [lfs.TransferRequest] with event
	// [lfs.TerminateEvent].
	SendTerminateRequest() error
}

// NewReverseCommunicatorLFS initializes a [ReverseCommunicatorLFS].
func NewReverseCommunicatorLFS(in io.Reader, out io.Writer) ReverseCommunicatorLFS {
	return &reverseCommunicatorLFS{
		in:  bufio.NewScanner(in),
		out: out,
	}
}

// reverseCommunicatorLFS implements [ReverseCommunicatorLFS].
type reverseCommunicatorLFS struct {
	in  *bufio.Scanner
	out io.Writer
}

func (g *reverseCommunicatorLFS) SendInitRequest(op lfs.Operation, remote string) error {
	// TODO: we don't support concurrency settings, but if we ever do add plumbing here
	req := &lfs.InitRequest{
		Event:               lfs.InitEvent,
		Operation:           op,
		Remote:              remote,
		Concurrent:          false,
		ConcurrentTransfers: 1,
	}

	// validate op and remote
	err := req.Validate()
	if err != nil {
		return fmt.Errorf("invalid InitRequest: %w", err)
	}

	reqRaw, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encoding InitRequest: %w", err)
	}

	_, err = g.out.Write(withNewline(reqRaw))
	if err != nil {
		return fmt.Errorf("writing InitRequest: %w", err)
	}

	return nil
}

func (g *reverseCommunicatorLFS) ReceiveInitResponse() error {

	line, err := g.readLine()
	if err != nil {
		return err
	}

	var initResp *lfs.InitResponse
	if err := json.Unmarshal(line, &initResp); err != nil {
		return fmt.Errorf("decoding InitRequest: %w", err)
	}

	if initResp.Error.Message != "" {
		return fmt.Errorf("received InitResponse error: code = %d, message = %s", initResp.Error.Code, initResp.Error.Message)
	}

	return nil
}

func (g *reverseCommunicatorLFS) SendTransferUploadRequest(oid string, size int64, path string) error {
	req := &lfs.TransferRequest{
		Event: lfs.UploadEvent,
		Oid:   oid,
		Size:  size,
		Path:  path,
	}

	if err := req.Validate(); err != nil {
		return err
	}

	reqRaw, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encoding TransferRequest: %w", err)
	}

	_, err = g.out.Write(withNewline(reqRaw))
	if err != nil {
		return fmt.Errorf("writing TransferRequest: %w", err)
	}

	return nil
}

func (g *reverseCommunicatorLFS) SendTransferDownloadRequest(oid string, size int64) error {
	req := &lfs.TransferRequest{
		Event: lfs.DownloadEvent,
		Oid:   oid,
		Size:  size,
	}

	if err := req.Validate(); err != nil {
		return err
	}

	reqRaw, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encoding TransferRequest: %w", err)
	}

	_, err = g.out.Write(withNewline(reqRaw))
	if err != nil {
		return fmt.Errorf("writing TransferRequest: %w", err)
	}

	return nil
}

func (g *reverseCommunicatorLFS) ReceiveProgressResponse() error {
	line, err := g.readLine()
	if err != nil {
		return err
	}

	var progressResp *lfs.ProgressResponse
	if err := json.Unmarshal(line, &progressResp); err != nil {
		return fmt.Errorf("decoding TransferResponse: %w", err)
	}

	return progressResp.Validate()
}

func (g *reverseCommunicatorLFS) ReceiveTransferResponse() error {
	line, err := g.readLine()
	if err != nil {
		return err
	}

	var transferResp *lfs.TransferResponse
	if err := json.Unmarshal(line, &transferResp); err != nil {
		return fmt.Errorf("decoding TransferResponse: %w", err)
	}

	if transferResp.Path == "" {
		return fmt.Errorf("no path provided in TransferResponse")
	}

	if transferResp.Oid == "" {
		return fmt.Errorf("no oid provided in TransferResponse")
	}

	if transferResp.Error.Message != "" {
		return fmt.Errorf("received TransferResponse error: code = %d, message = %s", transferResp.Error.Code, transferResp.Error.Message)
	}

	return nil
}

func (g *reverseCommunicatorLFS) SendTerminateRequest() error {
	req := &lfs.TransferRequest{
		Event: lfs.TerminateEvent,
	}

	reqRaw, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encoding terminate TransferRequest: %w", err)
	}

	_, err = g.out.Write(withNewline(reqRaw))
	if err != nil {
		return fmt.Errorf("writing terminate TransferRequest: %w", err)
	}

	return nil
}

func (g *reverseCommunicatorLFS) readLine() ([]byte, error) {
	ok := g.in.Scan()
	switch {
	case !ok && g.in.Err() != nil:
		return nil, fmt.Errorf("reading single command from git-lfs-remote-oci: %w", g.in.Err())
	case !ok:
		// EOF
		return nil, nil
	default:
		return g.in.Bytes(), nil
	}
}

func withNewline(line []byte) []byte {
	return append(line, []byte("\n")...) // TODO: can we use a rune?
}
