package comms

import (
	"bufio"
	"io"
)

// Communicator provides handling of git-lfs transfer protocol
// requests and responses.
type Communicator interface {
	RequestHandler
	ResponseHandler
}

// RequestHandler receives git-lfs transfer protocol requests.
type RequestHandler interface {
	ReceiveCapabilitiesRequest()
	ReceiveOp
}

// ResponseHandler sends git-lfs transfer protocol responses.
type ResponseHandler interface {
}

// NewCommunicator initializes a [Communicator].
func NewCommunicator(in io.Reader, out io.Writer) Communicator {
	return &defaultCommunicator{
		in:  bufio.NewScanner(in),
		out: out,
	}
}

// defaultCommunicator is the default implementation of [Communicator].
type defaultCommunicator struct {
	in  *bufio.Scanner
	out io.Writer
}
