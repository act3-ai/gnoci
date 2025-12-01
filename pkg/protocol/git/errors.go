package git

import "errors"

var (
	// ErrBadRequest indicates a request does not contain sufficient information
	// to handle.
	ErrBadRequest = errors.New("request is invalid")
	// ErrUnexpectedRequest indicates a request was not the expected type.
	ErrUnexpectedRequest = errors.New("unexpected request")
	// ErrUnsupportedRequest indicates a request is not supported by this package.
	ErrUnsupportedRequest = errors.New("unsupported request")
	// ErrEmptyRequest is an indicator that Git is either breaking up a batch of
	// requests or has finished sending requests.
	ErrEmptyRequest = errors.New("empty request")
	// ErrEndOfInput marks the end of input from Git.
	ErrEndOfInput = errors.New("end of input")
)
