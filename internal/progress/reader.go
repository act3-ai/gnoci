// Package progress facilitates tracking progress for a standard library io.Reader.
package progress

import (
	"io"
	"sync"
)

// Inspired by https://github.com/machinebox/progress/blob/master/reader.go.

// readCloser maintains progress information on the bytes read through it.
// Implements [Evaluator].
type readCloser struct {
	rc io.ReadCloser

	mu    sync.RWMutex
	total int
	delta int
	err   error
}

// NewEvalReadCloser wraps an [io.Reader] with capabilities to report bytes read
// so far and bytes read since the last check.
func NewEvalReadCloser(rc io.ReadCloser) EvalReadCloser {
	return &readCloser{
		rc: rc,
	}
}

// Read wraps [io.Reader.Read] with internal progress updates.
func (r *readCloser) Read(p []byte) (n int, err error) {
	n, err = r.rc.Read(p)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.total += n
	r.delta += n
	r.err = err
	return
}

// Progress returns the total number of bytes that have been read so far as
// well as bytes read since the last call to Progress.
func (r *readCloser) Progress() (soFar, sinceLast int, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	soFar = r.total
	sinceLast = r.delta
	r.delta = 0
	err = r.err
	return
}

// Close wraps an [io.ReadCloser.Close] method.
func (r *readCloser) Close() error {
	return r.rc.Close()
}
