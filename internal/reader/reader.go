// Package reader provides streaming line-based reading from io.Reader sources.
package reader

import (
	"bufio"
	"io"
)

// Default configuration values.
const (
	DefaultMaxLineSize = 1024 * 1024 // 1MB max line size
	DefaultBufferSize  = 64 * 1024   // 64KB initial buffer
)

// Line represents a single line read from the input stream.
type Line struct {
	// Text contains the line content (without newline).
	Text string

	// Number is the 1-based line number in the input.
	Number int

	// Err contains any error that occurred reading this line.
	// If Err is non-nil, Text may be empty.
	Err error
}

// StreamReader reads lines from an io.Reader in a streaming fashion.
// Designed for processing stdin in real-time (pipe-friendly).
type StreamReader struct {
	input      *bufio.Reader
	lineNumber int
	maxSize    int
}

// Option configures the StreamReader.
type Option func(*StreamReader)

// WithMaxLineSize sets the maximum allowed line size in bytes.
// A longer line is reported as a Line with Err set to bufio.ErrTooLong and
// empty Text; reading continues with the line after it.
func WithMaxLineSize(size int) Option {
	return func(r *StreamReader) {
		r.maxSize = size
	}
}

// New creates a StreamReader from an io.Reader.
// The reader processes input line-by-line, suitable for streaming.
func New(input io.Reader, opts ...Option) *StreamReader {
	reader := &StreamReader{
		maxSize: DefaultMaxLineSize,
	}

	// Apply options
	for _, opt := range opts {
		opt(reader)
	}

	reader.input = bufio.NewReaderSize(input, DefaultBufferSize)
	return reader
}

// next reads one line. It returns false when the input is exhausted.
// An oversized line is consumed to its end and returned with Err set, so one
// bad line never hides the lines after it. Any other read error ends the input.
func (r *StreamReader) next() (Line, bool) {
	var text []byte
	tooLong := false
	sawData := false

	for {
		fragment, err := r.input.ReadSlice('\n')
		if len(fragment) > 0 {
			sawData = true
			if !tooLong {
				text = append(text, fragment...)
				if len(trimLineEnding(text)) > r.maxSize {
					tooLong = true
					text = nil
				}
			}
		}

		switch err {
		case nil:
		case bufio.ErrBufferFull:
			continue
		case io.EOF:
			if !sawData {
				return Line{}, false
			}
		default:
			r.lineNumber++
			return Line{Number: r.lineNumber, Err: err}, true
		}

		r.lineNumber++
		if tooLong {
			return Line{Number: r.lineNumber, Err: bufio.ErrTooLong}, true
		}
		return Line{Text: string(trimLineEnding(text)), Number: r.lineNumber}, true
	}
}

func trimLineEnding(line []byte) []byte {
	if n := len(line); n > 0 && line[n-1] == '\n' {
		line = line[:n-1]
	}
	if n := len(line); n > 0 && line[n-1] == '\r' {
		line = line[:n-1]
	}
	return line
}

// Lines returns a channel that yields lines as they are read.
// The channel is closed when EOF is reached or a read fails.
// This method should only be called once per reader.
func (r *StreamReader) Lines() <-chan Line {
	lines := make(chan Line)

	go func() {
		defer close(lines)

		for {
			line, ok := r.next()
			if !ok {
				return
			}
			lines <- line
			if line.Err != nil && line.Err != bufio.ErrTooLong {
				return
			}
		}
	}()

	return lines
}

// ReadAll reads all lines synchronously and returns them as a slice.
// Useful for testing; for production use Lines() for streaming.
// The returned error is the first line error encountered, if any.
func (r *StreamReader) ReadAll() ([]Line, error) {
	var lines []Line
	var firstErr error

	for {
		line, ok := r.next()
		if !ok {
			return lines, firstErr
		}
		if line.Err != nil && firstErr == nil {
			firstErr = line.Err
		}
		if line.Err != nil && line.Err != bufio.ErrTooLong {
			return lines, firstErr
		}
		lines = append(lines, line)
	}
}
