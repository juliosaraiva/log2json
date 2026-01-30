// Package parser provides interfaces and types for log parsing.
package parser

import "errors"

// Common errors returned by parsers.
var (
	ErrNoMatch     = errors.New("line does not match parser pattern")
	ErrEmptyLine   = errors.New("empty line")
	ErrInvalidData = errors.New("invalid data in line")
)

// Entry represents a parsed log line with extracted fields.
type Entry struct {
	// Fields contains the extracted key-value pairs from the log line.
	Fields map[string]any

	// Raw holds the original unparsed line.
	Raw string

	// LineNum is the line number in the input stream (1-based).
	LineNum int

	// ParseError contains any error that occurred during parsing.
	// If set, Fields may be empty or partial.
	ParseError error
}

// NewEntry creates a new Entry with initialized fields map.
func NewEntry(raw string) *Entry {
	return &Entry{
		Fields: make(map[string]any),
		Raw:    raw,
	}
}

// Parser defines the interface that all log format parsers must implement.
// Each parser handles a specific log format (syslog, apache, etc.).
type Parser interface {
	// Name returns the unique identifier for this parser.
	// Used for format selection via CLI flags.
	Name() string

	// Description returns a human-readable description of the format.
	Description() string

	// CanParse performs a quick check to determine if this parser
	// can likely handle the given line. Should be fast (simple regex match).
	// Returns true if the parser should attempt to parse this line.
	CanParse(line string) bool

	// Parse extracts structured data from the log line.
	// Returns an Entry with extracted fields, or an error.
	// Even on error, Entry.Raw will contain the original line.
	Parse(line string) (*Entry, error)
}
