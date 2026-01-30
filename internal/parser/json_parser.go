package parser

import (
	"encoding/json"
	"strings"
)

// JSONParser handles lines that are already valid JSON.
// This is the highest priority parser since JSON is already structured.
type JSONParser struct{}

// NewJSONParser creates a new JSON parser.
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Name returns the parser identifier.
func (p *JSONParser) Name() string {
	return "json"
}

// Description returns a human-readable description.
func (p *JSONParser) Description() string {
	return "JSON formatted logs (already structured)"
}

// CanParse checks if the line looks like JSON.
// Quick check: must start with { and end with }
func (p *JSONParser) CanParse(line string) bool {
	trimmed := strings.TrimSpace(line)
	return len(trimmed) >= 2 &&
		trimmed[0] == '{' &&
		trimmed[len(trimmed)-1] == '}'
}

// Parse extracts data from a JSON log line.
func (p *JSONParser) Parse(line string) (*Entry, error) {
	entry := NewEntry(line)

	// Unmarshal into the fields map directly
	if err := json.Unmarshal([]byte(line), &entry.Fields); err != nil {
		entry.ParseError = err
		entry.Fields["raw"] = line
		entry.Fields["_parseError"] = err.Error()
	}

	return entry, nil
}
