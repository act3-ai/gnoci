package git

import "fmt"

// Capability defines a git-remote-helper capability.
//
// See https://git-scm.com/docs/gitremote-helpers#_capabilities.
type Capability = string

// Capabilities with a '*' prefix marks them as mandatory.
const (
	// CapabilityOption indicates a git remote helper is capable
	// of handling option commands.
	CapabilityOption Capability = "option"
	// CapabilityOption indicates a git remote helper is capable
	// of handling fetch commands.
	CapabilityFetch Capability = "fetch"
	// CapabilityOption indicates a git remote helper is capable
	// of handling push commands.
	CapabilityPush Capability = "push"
)

// CapabilitiesRequest is a command received from Git requesting a list of
// supported capabilities
//
// https://git-scm.com/docs/gitremote-helpers#Documentation/gitremote-helpers.txt-capabilities.
type CapabilitiesRequest struct {
	Cmd Command
}

// Parse decodes request fields ensuring the [CapabilitiesRequest] is of the correct type and
// contains sufficient information to handle the request.
//
// Implements [Parsable].
func (r *CapabilitiesRequest) Parse(fields []string) error {
	if len(fields) != 1 {
		return fmt.Errorf("%w: invalid fields for capabilities request: got %v", ErrBadRequest, fields)
	}

	cmd := Command(fields[0])
	if cmd != Capabilities {
		return fmt.Errorf("%w: got %s, want %s", ErrUnexpectedRequest, fields[0], Capabilities)
	}
	r.Cmd = Capabilities

	return nil
}

// String condenses [CapabilitiesRequest] into a string, the raw request received from Git.
func (r *CapabilitiesRequest) String() string {
	return string(r.Cmd)
}
