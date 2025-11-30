// Package comms facilitates receiving requests from and writing responses to Git via the remote helpers protocol.
//
// Protocol Reference: https://git-scm.com/docs/gitremote-helpers.
package comms

import (
	"fmt"
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
	WriteCapabilitiesResponse(capabilities []git.Capability) error
	WriteOptionResponse(supported git.OptionResult) error
	WriteListResponse(refs git.ReferenceLister) error
	WritePushResponse() error
	WriteFetchResponse() error
}
