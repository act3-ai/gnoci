// Package comms facilitates receiving requests from and writing responses to Git via the remote helpers protocol.
//
// Protocol Reference: https://git-scm.com/docs/gitremote-helpers.
package comms

import (
	"fmt"
	"io"
	"strings"

	"github.com/act3-ai/gnoci/pkg/protocol/git"
)

// RequestParser parses and returns a typed Git command.
type RequestParser[T git.CapabilitiesRequest |
	git.OptionRequest |
	git.ListRequest |
	git.FetchRequest |
	git.PushRequest] func(line string) (*T, error)

// ParseRequest implements [RequestParser].
func ParseRequest[T git.Parsable](line string) (T, error) {
	var zero T

	req := strings.Fields(line)
	if len(req) < 1 {
		return zero, fmt.Errorf("empty request")
	}

	cmd := git.Command(req[0])
	if !git.SupportedCommand(cmd) {
		return zero, fmt.Errorf("%w: %s", git.ErrUnsupportedRequest, req[0])
	}

	var typedReq = any(new(T)).(T)
	if err := typedReq.Parse(req); err != nil {
		return zero, fmt.Errorf("parsing request: %w", err)
	}

	return typedReq, nil
}

// ResponseHandler sends Git remote helper protocol responses.
type ResponseHandler interface {
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

// responseHandler implements [ResponseHandler].
type responseHandler struct {
	out io.Writer
}

// NewResponseHandler initializes a [ResponseHandler] capable of writing formatted
// responses to Git.
func NewResponseHandler(out io.Writer) ResponseHandler {
	return &responseHandler{out: out}
}

// WriteCapabilitiesResponse lists capabilities to Git. The response to a
// [git.CapabilitiesRequest].
func (r *responseHandler) WriteCapabilitiesResponse(capabilities []git.Capability) error {
	for _, capability := range capabilities {
		if _, err := fmt.Fprintln(r.out, capability); err != nil {
			return fmt.Errorf("writing capability %s: %w", capability, err)
		}
	}

	// end of batched output
	if _, err := fmt.Fprintln(r.out); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}

	return nil
}

// WriteOptionResponse indicates if an option in a [git.OptionRequest] is supported.
func (r *responseHandler) WriteOptionResponse(supported bool) error {
	const (
		optionSupported   string = "ok"
		optionNotSupportd string = "unsupported"
	)

	line := optionNotSupportd
	if supported {
		line = optionSupported
	}

	if _, err := fmt.Fprintln(r.out, line); err != nil {
		return fmt.Errorf("writing option response: %w", err)
	}
	return nil
}

// WriteListResponse lists Git references and their commits. The response to
// a [git.ListRequest].
func (r *responseHandler) WriteListResponse(resps []*git.ListResponse) error {
	if err := writeResponseBatch(r.out, resps); err != nil {
		return fmt.Errorf("writing list responses: %w", err)
	}
	return nil
}

// WritePushResponse lists the results of push actions. The response to one
// or more [git.PushRequest]s.
func (r *responseHandler) WritePushResponse(resps []*git.PushResponse) error {
	if err := writeResponseBatch(r.out, resps); err != nil {
		return fmt.Errorf("writing push responses: %w", err)
	}
	return nil
}

// WriteFetchResponse indicates fetching has completed. The response to one
// or more [git.FetchRequest]s.
func (r *responseHandler) WriteFetchResponse() error {
	if _, err := fmt.Fprintln(r.out); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}

	return nil
}

// TODO: I thought there was a go native interface for this?
type stringer interface {
	String() string
}

// writeResponseBatch is a helper function for writing batches of responses.
func writeResponseBatch[T stringer](out io.Writer, resps []T) error {
	for _, resp := range resps {
		if _, err := fmt.Fprintln(out, resp.String()); err != nil {
			return fmt.Errorf("writing response: %w", err)
		}
	}

	// end of batched output
	if _, err := fmt.Fprintln(out); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}

	return nil
}
