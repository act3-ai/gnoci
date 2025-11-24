// Package lfs defines types used in the Git LFS custom transfer protocol.
//
// Protocol: https://github.com/git-lfs/git-lfs/blob/main/docs/custom-transfers.md
package lfs

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidOperation indicates an operation is not supported or is unexpected.
	ErrInvalidOperation = errors.New("invalid operation")
	// ErrNoRemote indicates missing remote URL information.
	ErrNoRemote = errors.New("no remote specified")
)

// Event describes the type of request made by git-lfs.
type Event string

const (
	// InitEvent describes an [InitRequest] event.
	InitEvent Event = "init"
	// DownloadEvent describes a [TransferRequest] as a download action.
	DownloadEvent Event = "download"
	// UploadEvent describes a [TransferRequest] as an upload action.
	UploadEvent Event = "upload"
	// TerminateEvent describes the end of git-lfs requests.
	TerminateEvent Event = "terminate"
	// CompleteEvent describes a [TransferResponse] completion.
	CompleteEvent Event = "complete"
	// ProgessEvent describes a [ProgressResponse] during the handling of a
	// [TransferRequest].
	ProgessEvent Event = "progress"
)

// Operation indicates git-lfs will be performing a upload or download.
// Provided in an [InitRequest].
type Operation string

const (
	// DownloadOperation indicates git-lfs will be sending [TransferRequest]s
	// with [DownloadEvent]s.
	DownloadOperation Operation = "download"
	// UploadOperation indicates git-lfs will be sending [TransferRequest]s
	// with [UploadEvent]s.
	UploadOperation Operation = "upload"
)

// InitRequest is the first request sent by git-lfs.
type InitRequest struct {
	Event     Event     `json:"event"`     // always "init"
	Operation Operation `json:"operation"` // either [DownloadOperation] or [UploadOperation]
	Remote    string    `json:"remote"`    // shortname or full URL
	// TODO: We should set default concurrency values if these are not set.
	// TODO: some of these we don't support at this time?
	Concurrent          bool `json:"concurrent"`
	ConcurrentTransfers int  `json:"concurrenttransfers"`
}

// Validate ensures sufficient information for processing an [InitRequest].
func (r *InitRequest) Validate() error {
	// report as much as possibles
	var errs []error

	if r.Event != InitEvent {
		errs = append(errs, fmt.Errorf("expected event '%s', got '%s'", InitEvent, r.Event))
	}

	if r.Operation != DownloadOperation && r.Operation != UploadOperation {
		errs = append(errs, fmt.Errorf("%w: %s", ErrInvalidOperation, r.Operation))
	}

	if r.Remote == "" {
		errs = append(errs, ErrNoRemote)
	}

	if len(errs) > 0 {
		errs = append([]error{errors.New("invalid InitRequest")}, errs...)
	}

	return errors.Join(errs...)
}

// InitResponse is the response to an [InitRequest].
type InitResponse struct {
	Error ErrCodeMessage `json:"error,omitempty"`
}

// TransferRequest is sent by git-lfs for 0..N transfers.
//
// See:
//
// - https://github.com/git-lfs/git-lfs/blob/main/docs/custom-transfers.md#uploads
//
// - https://github.com/git-lfs/git-lfs/blob/main/docs/custom-transfers.md#downloads
type TransferRequest struct {
	Event Event  `json:"event"` // either [DownloadEvent] or [UploadEvent]
	Oid   string `json:"oid"`
	Size  int64  `json:"size"`
	Path  string `json:"path,omitempty"` // only included with [UploadEvent]
	// TODO: we implement a "standalone transfer agent", which is always null if the user sets up their settings properly
	// Action *Action `json:"action"`
}

// type Action struct {
// 	Href      string            `json:"href"`
// 	Header    map[string]string `json:"header,omitempty"`
// 	ExpiresAt time.Time         `json:"expires_at,omitempty"`
// 	ExpiresIn int               `json:"expires_in,omitempty"`
// 	Id        string            `json:"-"`
// 	Token     string            `json:"-"`

// 	createdAt time.Time
// }

// TransferResponse is the response to a [TransferRequest].
//
// See:
//
// - https://github.com/git-lfs/git-lfs/blob/main/docs/custom-transfers.md#uploads
//
// - https://github.com/git-lfs/git-lfs/blob/main/docs/custom-transfers.md#downloads
type TransferResponse struct {
	Event Event           `json:"event"`
	Oid   string          `json:"oid"`
	Path  string          `json:"path,omitempty"`
	Error *ErrCodeMessage `json:"error,omitempty"`
}

// ProgressResponse is sent periodically while processing a [TransferRequest].
//
// See:
//
// -  https://github.com/git-lfs/git-lfs/blob/main/docs/custom-transfers.md#progress
type ProgressResponse struct {
	Event          Event  `json:"event"`
	Oid            string `json:"oid"`
	BytesSoFar     int    `json:"bytesSoFar"`
	BytesSinceLast int    `json:"bytesSinceLast"`
}

// FinishRequest is received from git-lfs, indicating no other requests will be made.
type FinishRequest struct {
	Event Event `json:"event"` // always [TerminateEvent]
}

// ErrCodeMessage is included in a [InitResponse] or [TransferResponse] to indicate
// errors while handling requests.
type ErrCodeMessage struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
