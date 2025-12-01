package testutils

import (
	"bufio"
	"encoding/json"
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

type RequestSender interface {
	// SendCapabilitiesRequest sends a [git.CapabilitiesRequest].
	SendCapabilitiesRequest() error
	// SendOptionRequest sends a [git.OptionRequest].
	SendOptionRequest(opt git.Option, value string) error
	// SendListRequest sends a [git.ListRequest].
	SendListRequest() error
	// SendPushRequestBatch sends a batch of [git.PushRequest]s.
	//
	// Prepend a "+" to a src ref for a force push.
	SendPushRequestBatch() error
	// SendFetchRequestBatch sends a batch of [git.FetchRequest]s.
	SendFetchRequestBatch() error
}

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

	reqRaw, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encoding CapabilitiesRequest: %w", err)
	}

	_, err = c.out.Write(withNewline(reqRaw))
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

	reqRaw, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encoding OptionRequest: %w", err)
	}

	_, err = c.out.Write(withNewline(reqRaw))
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

	reqRaw, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encoding ListRequest: %w", err)
	}

	_, err = c.out.Write(withNewline(reqRaw))
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

		reqRaw, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("encoding PushRequest: %w", err)
		}

		_, err = c.out.Write(withNewline(reqRaw))
		if err != nil {
			return fmt.Errorf("writing PushRequest: %w", err)
		}
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

		reqRaw, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("encoding FetchRequest: %w", err)
		}

		_, err = c.out.Write(withNewline(reqRaw))
		if err != nil {
			return fmt.Errorf("writing FetchRequest: %w", err)
		}
	}

	return nil
}

// ReceiveCapabilitiesResponse receives a response to a [git.CapabilitiesRequest].
func (c *reverseCommunicator) ReceiveCapabilitiesResponse() error

// ReceiveOptionResponse receives a response to a [git.OptionRequest].
func (c *reverseCommunicator) ReceiveOptionResponse() error

// ReceiveListResponse receives a response to a [git.ListRequest].
func (c *reverseCommunicator) ReceiveListResponse() error

// ReceivePushResponse receives a batch of responses to a batch of [git.PushRequest]s.
func (c *reverseCommunicator) ReceivePushResponseBatch() error

// ReceiveFetchResponse receives a response to a batch of [git.FetchRequest]s.
func (c *reverseCommunicator) ReceiveFetchResponse() error
