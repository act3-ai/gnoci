// Package progress facilitates tracking progress for a standard library io.Reader.
package progress

import (
	"io"
	"sync"
)

// Inspired by https://github.com/machinebox/progress/blob/master/reader.go.

// Reader maintains progress information on the bytes read through it.
// Implements [Evaluator].
type Reader struct {
	reader io.Reader

	mu    sync.RWMutex
	total int
	delta int
	err   error
}

// NewReader wraps an [io.Reader] with capabilities to report bytes read
// so far and bytes read since the last check.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		reader: r,
	}
}

// Read wraps [io.Reader.Read] with internal progress updates.
func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.total += n
	r.delta += n
	r.err = err
	return
}

// Progress returns the total number of bytes that have been read so far as
// well as bytes read since the last call to Progress.
func (r *Reader) Progress() (soFar, sinceLast int, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	soFar = r.total
	sinceLast = r.delta
	r.delta = 0
	err = r.err
	return
}
