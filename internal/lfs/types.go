package lfs

import "time"

// See protocol docs: https://github.com/git-lfs/git-lfs/blob/main/docs/custom-transfers.md.
// Also their end of the protocol:  https://github.com/git-lfs/git-lfs/blob/a06ae195185ebc0fb00f3e4eef5a138d17338773/tq/custom.go#L78

type Event string

const (
	InitEvent      Event = "init"
	TerminateEvent Event = "terminate"
	CompleteEvent  Event = "complete"
	ProgessEvent   Event = "progress"
)

type Operation string

const (
	DownloadOperation Operation = "download"
	UploadOperation   Operation = "upload"
)

type InitRequest struct {
	Event     Event     `json:"event"`
	Operation Operation `json:"operation"`
	Remote    string    `json:"remote"` // shortname or full URL
	// TODO: We should use default concurrency values.
	Concurrent          bool `json:"concurrent"`
	ConcurrentTransfers int  `json:"concurrenttransfers"`
}

type InitResponse struct {
	Error ErrCodeMessage `json:"error"`
}

// TODO: after init request has been processed, respond with `{ }`
// Or on error : { "error": { "code": 32, "message": "Some init failure message" } }

// See https://github.com/git-lfs/git-lfs/blob/main/docs/custom-transfers.md#uploads
// See https://github.com/git-lfs/git-lfs/blob/main/docs/custom-transfers.md#downloads
type TransferRequest struct {
	// common between upload/download
	Event string `json:"event"`
	Oid   string `json:"oid"`
	Size  int64  `json:"size"`
	Path  string `json:"path,omitempty"`
	// TODO: we're likely doing a "standalone transfer agent", which is always null if the user sets up their settings properly
	Action *Action `json:"action"`
}

type Action struct {
	Href      string            `json:"href"`
	Header    map[string]string `json:"header,omitempty"`
	ExpiresAt time.Time         `json:"expires_at,omitempty"`
	ExpiresIn int               `json:"expires_in,omitempty"`
	Id        string            `json:"-"`
	Token     string            `json:"-"`

	createdAt time.Time
}

// See https://github.com/git-lfs/git-lfs/blob/main/docs/custom-transfers.md#uploads
// See https://github.com/git-lfs/git-lfs/blob/main/docs/custom-transfers.md#downloads
type TransferResponse struct {
	Event Event          `json:"event"`
	Oid   string         `json:"oid"`
	Path  string         `json:"path,omitempty"`
	Error ErrCodeMessage `json:"error"`
}

// See https://github.com/git-lfs/git-lfs/blob/main/docs/custom-transfers.md#progress
type ProgressResponse struct {
	Event          Event  `json:"event"`
	Oid            string `json:"oid"`
	BytesSoFar     int    `json:"bytesSoFar"`
	BytesSinceLast int    `json:"bytesSinceLast"`
}

type FinishResponse struct {
	Event Event `json:"event"` // always "terminate"
}

type ErrCodeMessage struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
