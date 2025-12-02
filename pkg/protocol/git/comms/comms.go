// Package comms facilitates receiving requests from and writing responses to Git via the remote helpers protocol.
//
// Protocol Reference: https://git-scm.com/docs/gitremote-helpers.
package comms

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/act3-ai/gnoci/pkg/protocol/git"
)

// Communicator provides handling of Git remote helper protocol
// requests and responses.
type Communicator interface {
	RequestParser
	ResponseWriter
}

// RequestParser reads and parses Git protocol requests.
type RequestParser interface {
	// LookAhead aids in determining the type of the next request if no specific
	// command is expected. The request is saved and is included in a subsequent
	// method call.
	LookAhead() (git.Command, error)
	// ParseCapabilitiesRequest parses the next request as a [git.CapabilitiesRequest].
	ParseCapabilitiesRequest() (*git.CapabilitiesRequest, error)
	// ParseOptionRequest parses the next request as a [git.OptionRequest].
	ParseOptionRequest() (*git.OptionRequest, error)
	// ParseListRequest parses the next request as a [git.ListRequest].
	ParseListRequest() (*git.ListRequest, error)
	// ParseFetchRequestBatch parses a batch of [git.FetchRequest]s.
	ParseFetchRequestBatch() ([]git.FetchRequest, error)
	// ParsePushRequestBatch parses a batch of [git.PushRequest]s.
	ParsePushRequestBatch() ([]git.PushRequest, error)
}

// ResponseWriter sends Git remote helper protocol responses.
type ResponseWriter interface {
	// WriteCapabilitiesResponse lists capabilities to Git. The response to a
	// [git.CapabilitiesRequest].
	WriteCapabilitiesResponse(capabilities []git.Capability) error
	// WriteOptionResponse indicates if an option in a [git.OptionRequest] is supported.
	WriteOptionResponse(supported bool) error
	// WriteListResponse lists Git references and their commits. The response to
	// a [git.ListRequest].
	WriteListResponse(resps []*git.ListResponse) error
	// WritePushResponse lists the results of push actions. The response to one
	// or more [git.PushRequest]s.
	WritePushResponse(resp []*git.PushResponse) error
	// WriteFetchResponse indicates fetching has completed. The response to one
	// or more [git.FetchRequest]s.
	WriteFetchResponse() error
}

// defaultCommunicator implements [Communicator].
type defaultCommunicator struct {
	in  bufio.Scanner
	out io.Writer

	previous []string // last lookahead read, parsed as fields
}

// NewCommunicator initializes a [Communicator] capable of writing formatted
// responses to Git.
func NewCommunicator(in io.Reader, out io.Writer) Communicator {
	return &defaultCommunicator{
		in:  *bufio.NewScanner(in),
		out: out,
	}
}

// LookAhead aids in determining the type of the next request if no specific
// command is expected. The request is saved and is included in a subsequent
// method call.
func (c *defaultCommunicator) LookAhead() (git.Command, error) {
	line, err := c.readLine()
	if err != nil {
		return "", err
	}

	req := strings.Fields(line)
	if len(req) < 1 {
		return "", git.ErrEmptyRequest
	}
	c.previous = req

	cmd := git.Command(req[0])
	if !git.SupportedCommand(cmd) {
		return "", fmt.Errorf("%w: %s", git.ErrUnsupportedRequest, req[0])
	}

	return cmd, nil
}

// previousOrNext determines if [defaultCommunicator.LookAhead] has been called,
// if not it reads the next line.
func (c *defaultCommunicator) previousOrNext() ([]string, error) {
	if len(c.previous) > 0 {
		return c.previous, nil
	}

	return c.next()
}

func (c *defaultCommunicator) next() ([]string, error) {
	line, err := c.readLine()
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(line)
	if len(fields) < 1 {
		return nil, git.ErrEmptyRequest
	}

	return fields, nil
}

// ParseCapabilitiesRequest parses the next request as a [git.CapabilitiesRequest].
func (c *defaultCommunicator) ParseCapabilitiesRequest() (*git.CapabilitiesRequest, error) {
	defer func() { c.previous = nil }()
	fields, err := c.previousOrNext()
	if err != nil {
		return nil, err
	}

	var req *git.CapabilitiesRequest
	if err := req.Parse(fields); err != nil {
		return nil, fmt.Errorf("parsing capabilities request: %w", err)
	}

	return req, nil
}

// ParseOptionRequest parses the next request as a [git.OptionRequest].
func (c *defaultCommunicator) ParseOptionRequest() (*git.OptionRequest, error) {
	defer func() { c.previous = nil }()
	fields, err := c.previousOrNext()
	if err != nil {
		return nil, err
	}

	var req *git.OptionRequest
	if err := req.Parse(fields); err != nil {
		return nil, fmt.Errorf("parsing option request: %w", err)
	}

	return req, nil
}

// ParseListRequest parses the next request as a [git.ListRequest].
func (c *defaultCommunicator) ParseListRequest() (*git.ListRequest, error) {
	defer func() { c.previous = nil }()
	fields, err := c.previousOrNext()
	if err != nil {
		return nil, err
	}

	var req *git.ListRequest
	if err := req.Parse(fields); err != nil {
		return nil, fmt.Errorf("parsing capabilities request: %w", err)
	}

	return req, nil
}

// ParseFetchRequestBatch parses a batch of [git.FetchRequest]s.
func (c *defaultCommunicator) ParseFetchRequestBatch() ([]git.FetchRequest, error) {
	defer func() { c.previous = nil }()

	// if lookahead was used, include in batch
	var reqs []git.FetchRequest
	if len(c.previous) > 0 {
		var req git.FetchRequest
		if err := req.Parse(c.previous); err != nil {
			return nil, err
		}
	}

	// ingest batch
	for {
		fields, err := c.next()
		switch {
		case errors.Is(err, git.ErrEmptyRequest):
			// batch complete
			return reqs, nil
		case err != nil:
			return nil, err
		default:
			var req git.FetchRequest
			if err := req.Parse(fields); err != nil {
				return nil, fmt.Errorf("parsing fetch request: %w", err)
			}
			reqs = append(reqs, req)
		}
	}
}

// ParsePushRequestBatch parses a batch of [git.PushRequest]s.
func (c *defaultCommunicator) ParsePushRequestBatch() ([]git.PushRequest, error) {
	defer func() { c.previous = nil }()

	// if lookahead was used, include in batch
	var reqs []git.PushRequest
	if len(c.previous) > 0 {
		var req git.PushRequest
		if err := req.Parse(c.previous); err != nil {
			return nil, err
		}
	}

	// ingest batch
	for {
		fields, err := c.next()
		switch {
		case errors.Is(err, git.ErrEmptyRequest):
			// batch complete
			return reqs, nil
		case err != nil:
			return nil, err
		default:
			var req git.PushRequest
			if err := req.Parse(fields); err != nil {
				return nil, fmt.Errorf("parsing push request: %w", err)
			}
			reqs = append(reqs, req)
		}
	}
}

// WriteCapabilitiesResponse lists capabilities to Git. The response to a
// [git.CapabilitiesRequest].
func (c *defaultCommunicator) WriteCapabilitiesResponse(capabilities []git.Capability) error {
	for _, capability := range capabilities {
		if _, err := fmt.Fprintln(c.out, capability); err != nil {
			return fmt.Errorf("writing capability %s: %w", capability, err)
		}
	}

	// end of batched output
	if _, err := fmt.Fprintln(c.out); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}

	return nil
}

// WriteOptionResponse indicates if an option in a [git.OptionRequest] is supported.
func (c *defaultCommunicator) WriteOptionResponse(supported bool) error {
	line := git.OptionNotSupported
	if supported {
		line = git.OptionSupported
	}

	if _, err := fmt.Fprintln(c.out, line); err != nil {
		return fmt.Errorf("writing option response: %w", err)
	}
	return nil
}

// WriteListResponse lists Git references and their commits. The response to
// a [git.ListRequest].
func (c *defaultCommunicator) WriteListResponse(resps []*git.ListResponse) error {
	for _, resp := range resps {
		if _, err := fmt.Fprintln(c.out, resp.String()); err != nil {
			return fmt.Errorf("writing list response: %w", err)
		}
	}

	// end of batched output
	if _, err := fmt.Fprintln(c.out); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}

	return nil
}

// WritePushResponse lists the results of push actions. The response to one
// or more [git.PushRequest]s.
func (c *defaultCommunicator) WritePushResponse(resps []*git.PushResponse) error {
	for _, resp := range resps {
		if _, err := fmt.Fprintln(c.out, resp.String()); err != nil {
			return fmt.Errorf("writing push response: %w", err)
		}
	}

	// end of batched output
	if _, err := fmt.Fprintln(c.out); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}

	return nil
}

// WriteFetchResponse indicates fetching has completed. The response to one
// or more [git.FetchRequest]s.
func (c *defaultCommunicator) WriteFetchResponse() error {
	if _, err := fmt.Fprintln(c.out); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}

	return nil
}

func (c *defaultCommunicator) readLine() (string, error) {
	ok := c.in.Scan()
	switch {
	case !ok && c.in.Err() != nil:
		return "", fmt.Errorf("reading single command from git-lfs: %w", c.in.Err())
	case !ok:
		// EOF
		return "", git.ErrEndOfInput
	default:
		return c.in.Text(), nil
	}
}
