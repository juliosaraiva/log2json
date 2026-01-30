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
	scanner    *bufio.Scanner
	lineNumber int
	maxSize    int
}

// Option configures the StreamReader.
type Option func(*StreamReader)

// WithMaxLineSize sets the maximum allowed line size.
// Lines exceeding this are truncated with an error.
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

	// Create scanner with custom buffer
	scanner := bufio.NewScanner(input)
	buf := make([]byte, DefaultBufferSize)
	scanner.Buffer(buf, reader.maxSize)

	reader.scanner = scanner
	return reader
}

// Lines returns a channel that yields lines as they are read.
// The channel is closed when EOF is reached or an error occurs.
// This method should only be called once per reader.
func (r *StreamReader) Lines() <-chan Line {
	lines := make(chan Line)

	go func() {
		defer close(lines)

		for r.scanner.Scan() {
			r.lineNumber++
			lines <- Line{
				Text:   r.scanner.Text(),
				Number: r.lineNumber,
			}
		}

		// Check for scanner errors (not EOF)
		if err := r.scanner.Err(); err != nil {
			lines <- Line{
				Number: r.lineNumber + 1,
				Err:    err,
			}
		}
	}()

	return lines
}

// ReadAll reads all lines synchronously and returns them as a slice.
// Useful for testing; for production use Lines() for streaming.
func (r *StreamReader) ReadAll() ([]Line, error) {
	var lines []Line

	for r.scanner.Scan() {
		r.lineNumber++
		lines = append(lines, Line{
			Text:   r.scanner.Text(),
			Number: r.lineNumber,
		})
	}

	if err := r.scanner.Err(); err != nil {
		return lines, err
	}

	return lines, nil
}
