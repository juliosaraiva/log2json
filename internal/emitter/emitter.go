// Package emitter handles JSON output serialization.
package emitter

import (
	"bufio"
	"encoding/json"
	"io"
	"time"

	"github.com/juliosaraiva/log2json/internal/parser"
)

// Options configures the JSON emitter behavior.
type Options struct {
	// Pretty enables indented JSON output.
	// Not recommended for pipe output (breaks NDJSON).
	Pretty bool

	// Fields limits output to only these fields.
	// Empty means output all fields.
	Fields []string

	// AddTimestamp adds _ingestTime with current timestamp.
	AddTimestamp bool

	// AddLineNumber adds _lineNumber field.
	AddLineNumber bool

	// AddRaw includes the original line as _raw field.
	AddRaw bool

	// OmitEmpty skips entries with parse errors.
	OmitEmpty bool
}

// Emitter serializes parsed log entries to JSON and writes to output.
type Emitter struct {
	writer  *bufio.Writer
	options Options
	encoder *json.Encoder
}

// New creates a new JSON emitter writing to the given output.
func New(output io.Writer, opts Options) *Emitter {
	writer := bufio.NewWriter(output)
	encoder := json.NewEncoder(writer)

	if opts.Pretty {
		encoder.SetIndent("", "  ")
	}

	// Don't escape HTML characters (cleaner output)
	encoder.SetEscapeHTML(false)

	return &Emitter{
		writer:  writer,
		options: opts,
		encoder: encoder,
	}
}

// Emit writes a parsed entry as JSON to the output.
// Each entry is written as a single line (NDJSON format).
func (e *Emitter) Emit(entry *parser.Entry) error {
	// Skip empty entries if configured
	if e.options.OmitEmpty && entry.ParseError != nil {
		return nil
	}

	// Build output object
	output := e.buildOutput(entry)

	// Encode and write
	if err := e.encoder.Encode(output); err != nil {
		return err
	}

	// Flush immediately for real-time output
	return e.writer.Flush()
}

// buildOutput constructs the output map from an entry.
func (e *Emitter) buildOutput(entry *parser.Entry) map[string]any {
	// Start with entry fields or create new map
	var output map[string]any

	if len(e.options.Fields) > 0 {
		// Filter to only requested fields
		output = make(map[string]any)
		for _, field := range e.options.Fields {
			if val, ok := entry.Fields[field]; ok {
				output[field] = val
			}
		}
	} else {
		// Copy all fields
		output = make(map[string]any, len(entry.Fields)+3)
		for k, v := range entry.Fields {
			output[k] = v
		}
	}

	// Add metadata fields (prefixed with _)
	if e.options.AddTimestamp {
		output["_ingestTime"] = time.Now().UTC().Format(time.RFC3339Nano)
	}

	if e.options.AddLineNumber {
		output["_lineNumber"] = entry.LineNum
	}

	if e.options.AddRaw {
		output["_raw"] = entry.Raw
	}

	// Add parse error if present
	if entry.ParseError != nil {
		output["_parseError"] = entry.ParseError.Error()
	}

	return output
}

// Close flushes any remaining data.
func (e *Emitter) Close() error {
	return e.writer.Flush()
}
