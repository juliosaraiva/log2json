package parser

import (
	"regexp"
)

// KeyValueParser handles logs in key=value format.
// Common in structured logging frameworks like logfmt.
// Example: level=info msg="User logged in" user_id=123 duration=0.5
type KeyValueParser struct {
	// pattern matches key=value or key="quoted value" pairs
	pattern *regexp.Regexp
}

// NewKeyValueParser creates a new key-value parser.
func NewKeyValueParser() *KeyValueParser {
	// Match: key=value or key="value with spaces" or key='value'
	pattern := regexp.MustCompile(`(\w+)=(?:"([^"]*)"|'([^']*)'|(\S+))`)
	return &KeyValueParser{pattern: pattern}
}

// Name returns the parser identifier.
func (p *KeyValueParser) Name() string {
	return "kv"
}

// Description returns a human-readable description.
func (p *KeyValueParser) Description() string {
	return "Key=value format (logfmt style)"
}

// CanParse checks if the line contains key=value patterns.
// Requires at least 2 key=value pairs to avoid false positives.
func (p *KeyValueParser) CanParse(line string) bool {
	matches := p.pattern.FindAllString(line, -1)
	return len(matches) >= 2
}

// Parse extracts key-value pairs from the log line.
func (p *KeyValueParser) Parse(line string) (*Entry, error) {
	entry := NewEntry(line)

	matches := p.pattern.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		entry.ParseError = ErrNoMatch
		entry.Fields["raw"] = line
		return entry, nil
	}

	for _, match := range matches {
		key := match[1]

		// Value is in one of the capture groups (quoted or unquoted)
		var value string
		switch {
		case match[2] != "": // double-quoted
			value = match[2]
		case match[3] != "": // single-quoted
			value = match[3]
		default: // unquoted
			value = match[4]
		}

		// Try to convert to appropriate type
		entry.Fields[key] = inferType(value)
	}

	return entry, nil
}
