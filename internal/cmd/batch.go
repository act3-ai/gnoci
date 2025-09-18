package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// BatchReadWriter supports both reading from and writing to Git in batches.
type BatchReadWriter interface {
	BatchReader
	BatchWriter
}

// BatchReader extends Reader to support reading sets of commands
// provided by Git.
type BatchReader interface {
	Reader

	// ReadBatch reads lines from Git until an empty line is encountered.
	ReadBatch(context.Context) ([]Git, error)
}

// BatchWriter extends Writer to support writing sets of responses to Git.
type BatchWriter interface {
	Writer

	// WriteBatch writes a batch of messages to Git, which
	// MAY need to be followed up with a flush.
	WriteBatch(ctx context.Context, lines ...string) error
}

// type ReadWriter interface {
// 	Reader
// 	Writer
// }

// Reader reads a single command from Git.
type Reader interface {
	// Read reads a single line from Git.
	Read(context.Context) (Git, error)
}

// Writer is used to Write single lines to Git, completed with a Flush.
type Writer interface {
	// Write buffers a single line write to Git. One or more
	// calls MAY need to be followed up with a flush.
	Write(ctx context.Context, line string) error

	// Flush writes buffered Write(s) to Git, optionally followed up with a blank line.
	Flush(newline bool) error
}

// batcher implements BatchReadWriter.
type batcher struct {
	in  *bufio.Scanner
	out *bufio.Writer
}

// NewBatcher returns a buffered BatchReadWriter.
func NewBatcher(in io.Reader, out io.Writer) BatchReadWriter {
	return &batcher{
		in:  bufio.NewScanner(in),
		out: bufio.NewWriter(out),
	}
}

// Read parses a single command received by Git.
func (b *batcher) Read(ctx context.Context) (Git, error) {
	ok := b.in.Scan()
	switch {
	case !ok && b.in.Err() != nil:
		return Git{}, fmt.Errorf("reading single command from Git: %w", b.in.Err())
	case !ok:
		// EOF
		return Git{Cmd: Done}, nil
	default:
		txt := b.in.Text()
		slog.DebugContext(ctx, "read line from Git", "text", txt)
		cmd, err := parse(ctx, txt)
		if err != nil {
			return Git{}, fmt.Errorf("parsing Git command: %w", err)
		}
		return cmd, nil
	}
}

// ReadBatch reads lines from Git until an empty line is encountered.
func (b *batcher) ReadBatch(ctx context.Context) ([]Git, error) {
	result := make([]Git, 0, 2)
	for b.in.Scan() {
		txt := b.in.Text()
		slog.DebugContext(ctx, "read line from Git", "text", txt)
		if txt == "" {
			break
		}
		cmd, err := parse(ctx, txt)
		if err != nil {
			return nil, fmt.Errorf("parsing Git command: %w", err)
		}
		result = append(result, cmd)
	}
	if b.in.Err() != nil {
		return result, fmt.Errorf("scanning input: %w", b.in.Err())
	}
	return result, nil
}

// WriteBatch writes Message(s) to Git, completing the batch with a blank line, and flushing the buffered writes to Git.
func (b *batcher) WriteBatch(ctx context.Context, lines ...string) error {
	for _, line := range lines {
		if err := b.Write(ctx, line); err != nil {
			return err
		}
	}

	return b.Flush(true)
}

// Write buffers a single line write to Git, must be followed up with a flush.
func (b *batcher) Write(ctx context.Context, line string) error {
	slog.DebugContext(ctx, "writing line to git", "line", line)
	if _, err := fmt.Fprintln(b.out, line); err != nil {
		return fmt.Errorf("writing to Git, line: %s: %w", line, err)
	}

	return nil
}

// Flush writes buffered Write(s) to Git, followed up with a blank line.
func (b *batcher) Flush(blankLine bool) error {
	if blankLine {
		if _, err := fmt.Fprintln(b.out); err != nil {
			return fmt.Errorf("writing blank line to Git: %w", err)
		}
	}

	if err := b.out.Flush(); err != nil {
		return fmt.Errorf("flushing writes to Git: %w", err)
	}

	return nil
}

// parse parses a single line received from Git, turning it into a cmd.Git
// easily identified by Type.
func parse(ctx context.Context, line string) (Git, error) {
	fields := strings.Fields(line)
	if len(fields) < 1 {
		return Git{
			Cmd: Empty,
		}, nil
	}

	cmd := Command(fields[0])
	switch cmd {
	case Capabilities:
		return Git{
			Cmd: Capabilities,
		}, nil
	case Option:
		if err := validOption(ctx, fields...); err != nil {
			return Git{}, err
		}

		return Git{
			Cmd:    Option,
			SubCmd: Command(fields[1]),
			Data:   fields[2:],
		}, nil
	case List:
		res := Git{
			Cmd: List,
		}
		if len(fields) > 1 {
			res.SubCmd = Command(fields[1])
		}
		return res, nil
	case Push:
		if len(fields) < 2 {
			return Git{}, fmt.Errorf("insufficient args for push command")
		}
		return Git{
			Cmd:  Push,
			Data: fields[1:],
		}, nil
	case Fetch:
		if len(fields) < 2 {
			return Git{}, fmt.Errorf("insufficient args for fetch command")
		}
		return Git{
			Cmd:  Fetch,
			Data: fields[1:],
		}, nil
	default:
		return Git{}, fmt.Errorf("%w: %s", ErrUnsupportedCommand, cmd)
	}
}
