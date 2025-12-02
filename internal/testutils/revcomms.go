package testutils

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/act3-ai/gnoci/pkg/protocol/git"
	"github.com/go-git/go-git/v5/plumbing"
)

// ReverseCommunicator is the reverse of git comms.Communicator, acting as Git
// sending remote helper protocol requests and receiving responses.
type ReverseCommunicator interface {
	RequestSender
	ResponseReceiver
}

// RequestSender provides methods for sending requests.
type RequestSender interface {
	// SendCapabilitiesRequest sends a [git.CapabilitiesRequest].
	SendCapabilitiesRequest() error
	// SendOptionRequest sends a [git.OptionRequest].
	SendOptionRequest(opt git.Option, value string) error
	// SendListRequest sends a [git.ListRequest].
	SendListRequest(forPush bool) error
	// SendPushRequestBatch sends a batch of [git.PushRequest]s.
	//
	// Prepend a "+" to a src ref for a force push.
	SendPushRequestBatch(refs map[string]string) error
	// SendFetchRequestBatch sends a batch of [git.FetchRequest]s.
	SendFetchRequestBatch(refs []plumbing.Reference) error
}

// ResponseReceiver provides methods for receiving and parsing responses.
type ResponseReceiver interface {
	// ReceiveCapabilitiesResponse receives a response to a [git.CapabilitiesRequest].
	ReceiveCapabilitiesResponse() error
	// ReceiveOptionResponse receives a response to a [git.OptionRequest].
	ReceiveOptionResponse() error
	// ReceiveListResponse receives a response to a [git.ListRequest].
	ReceiveListResponse() error
	// ReceivePushResponse receives a batch of responses to a batch of [git.PushRequest]s.
	ReceivePushResponseBatch() error
	// ReceiveFetchResponse receives a response to a batch of [git.FetchRequest]s.
	ReceiveFetchResponse() error
}

// NewReverseCommunicator initializes a [ReverseCommunicator].
func NewReverseCommunicator(in io.Reader, out io.Writer) ReverseCommunicator {
	return &reverseCommunicator{
		in:  bufio.NewScanner(in),
		out: out,
	}
}

type reverseCommunicator struct {
	in  *bufio.Scanner
	out io.Writer
}

// SendCapabilitiesRequest sends a [git.CapabilitiesRequest].
func (c *reverseCommunicator) SendCapabilitiesRequest() error {
	req := &git.CapabilitiesRequest{
		Cmd: git.Capabilities,
	}

	_, err := c.out.Write([]byte(req.String() + "\n"))
	if err != nil {
		return fmt.Errorf("writing CapabilitiesRequest: %w", err)
	}

	return nil
}

// SendOptionRequest sends a [git.OptionRequest].
func (c *reverseCommunicator) SendOptionRequest(opt git.Option, value string) error {
	req := &git.OptionRequest{
		Cmd:   git.Options,
		Opt:   opt,
		Value: value,
	}

	_, err := c.out.Write([]byte(req.String() + "\n"))
	if err != nil {
		return fmt.Errorf("writing OptionRequest: %w", err)
	}

	return nil
}

// SendListRequest sends a [git.ListRequest].
func (c *reverseCommunicator) SendListRequest(forPush bool) error {
	req := &git.ListRequest{
		Cmd:     git.List,
		ForPush: forPush,
	}

	_, err := c.out.Write([]byte(req.String() + "\n"))
	if err != nil {
		return fmt.Errorf("writing ListRequest: %w", err)
	}

	return nil
}

// SendPushRequestBatch sends a batch of [git.PushRequest]s. Takes a map[src]remote.
//
// Prepend a "+" to a src ref for a force push.
func (c *reverseCommunicator) SendPushRequestBatch(refs map[string]string) error {
	for src, remote := range refs {
		var force bool
		if strings.HasPrefix(src, "+") {
			force = true
			src = strings.TrimPrefix(src, "+")
		}
		req := git.PushRequest{
			Cmd:    git.Push,
			Force:  force,
			Src:    plumbing.ReferenceName(src),
			Remote: plumbing.ReferenceName(remote),
		}

		_, err := c.out.Write([]byte(req.String() + "\n"))
		if err != nil {
			return fmt.Errorf("writing PushRequest: %w", err)
		}
	}

	_, err := c.out.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("completing PushRequest: %w", err)
	}

	return nil
}

// SendFetchRequestBatch sends a batch of [git.FetchRequest]s.
func (c *reverseCommunicator) SendFetchRequestBatch(refs []plumbing.Reference) error {
	for _, ref := range refs {
		req := git.FetchRequest{
			Cmd: git.Fetch,
			Ref: &ref,
		}

		_, err := c.out.Write([]byte(req.String() + "\n"))
		if err != nil {
			return fmt.Errorf("writing FetchRequest: %w", err)
		}
	}

	_, err := c.out.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("completing FetchRequest: %w", err)
	}

	return nil
}

// ReceiveCapabilitiesResponse receives a response to a [git.CapabilitiesRequest].
func (c *reverseCommunicator) ReceiveCapabilitiesResponse() error {
	var option, fetch, push bool
	for {
		line, err := c.readLine()
		if err != nil {
			return err
		}

		if line == "" {
			if !option || !fetch || !push {
				return fmt.Errorf("did not receive expected capabilities: got option %t fetch %t push %t", option, fetch, push)
			}
			return nil
		}

		switch git.Capability(line) {
		case git.CapabilityOption:
			option = true
		case git.CapabilityFetch:
			fetch = true
		case git.CapabilityPush:
			push = true
		default:
			return fmt.Errorf("unrecognized capability %s", line)
		}
	}
}

// ReceiveOptionResponse receives a response to a [git.OptionRequest].
func (c *reverseCommunicator) ReceiveOptionResponse() error {
	line, err := c.readLine()
	if err != nil {
		return err
	}

	switch line {
	case git.OptionSupported:
		return nil
	case git.OptionNotSupported:
		return nil
	default:
		return fmt.Errorf("unrecognized OptionResponse %s", line)
	}
}

// ReceiveListResponse receives a response to a [git.ListRequest].
func (c *reverseCommunicator) ReceiveListResponse() error {
	for {
		line, err := c.readLine()
		if err != nil {
			return err
		}

		if line == "" {
			return nil
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			return fmt.Errorf("insufficient number of fields in ListResponse: %s", line)
		}
	}
}

// ReceivePushResponse receives a batch of responses to a batch of [git.PushRequest]s.
func (c *reverseCommunicator) ReceivePushResponseBatch() error {
	for {
		line, err := c.readLine()
		if err != nil {
			return err
		}

		if line == "" {
			return nil
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			return fmt.Errorf("insufficient number of fields in PushResponse: %s", line)
		}

		switch fields[0] {
		case "ok":
			if len(fields) != 2 {
				return fmt.Errorf("malformed PushResponse ok status: %s", line)
			}
		case "error":
			if len(fields) != 3 {
				return fmt.Errorf("malformed PushResponse error status: %s", line)
			}
		default:
			return nil
		}
	}

}

// ReceiveFetchResponse receives a response to a batch of [git.FetchRequest]s.
func (c *reverseCommunicator) ReceiveFetchResponse() error {
	line, err := c.readLine()
	if err != nil {
		return err
	}

	if line != "" {
		return fmt.Errorf("unexpected FetchResponse: %s", line)
	}

	return nil
}

func (c *reverseCommunicator) readLine() (string, error) {
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
