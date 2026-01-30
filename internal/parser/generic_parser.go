package parser

import (
	"regexp"
	"strings"
)

// GenericParser handles common log patterns with timestamp and level.
// Falls back to wrapping the entire line as "message" if no pattern matches.
// Example: 2024-01-15 10:30:45 INFO This is a log message
type GenericParser struct {
	// patterns to try in order
	patterns []*regexp.Regexp
}

// NewGenericParser creates a new generic log parser.
func NewGenericParser() *GenericParser {
	patterns := []*regexp.Regexp{
		// ISO timestamp with level: 2024-01-15 10:30:45.123 INFO message
		regexp.MustCompile(
			`^(?P<timestamp>\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)\s+` +
				`(?P<level>DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|TRACE)\s+` +
				`(?P<message>.+)$`,
		),
		// Level first: INFO 2024-01-15 10:30:45 message
		regexp.MustCompile(
			`^(?P<level>DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|TRACE)\s+` +
				`(?P<timestamp>\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d+)?)\s+` +
				`(?P<message>.+)$`,
		),
		// Bracketed level: [INFO] 2024-01-15 message or 2024-01-15 [INFO] message
		regexp.MustCompile(
			`^(?:(?P<timestamp>\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d+)?)\s+)?` +
				`\[(?P<level>DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|TRACE)\]\s+` +
				`(?P<message>.+)$`,
		),
		// Just level and message: INFO: message or INFO - message
		regexp.MustCompile(
			`^(?P<level>DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|TRACE)[:\-\s]+(?P<message>.+)$`,
		),
	}

	return &GenericParser{patterns: patterns}
}

// Name returns the parser identifier.
func (p *GenericParser) Name() string {
	return "generic"
}

// Description returns a human-readable description.
func (p *GenericParser) Description() string {
	return "Generic timestamp/level patterns (fallback)"
}

// CanParse always returns true as this is the fallback parser.
func (p *GenericParser) CanParse(line string) bool {
	return true
}

// Parse attempts to extract fields using common patterns.
// Falls back to wrapping the line as "message" if no pattern matches.
func (p *GenericParser) Parse(line string) (*Entry, error) {
	entry := NewEntry(line)

	// Skip empty lines
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		entry.Fields["message"] = ""
		entry.ParseError = ErrEmptyLine
		return entry, nil
	}

	// Try each pattern
	for _, pattern := range p.patterns {
		matches := pattern.FindStringSubmatch(line)
		if matches != nil {
			names := pattern.SubexpNames()
			for i, match := range matches {
				if i == 0 || names[i] == "" || match == "" {
					continue
				}
				// Normalize level to uppercase
				if names[i] == "level" {
					match = strings.ToUpper(match)
				}
				entry.Fields[names[i]] = match
			}
			return entry, nil
		}
	}

	// Fallback: wrap entire line as message
	entry.Fields["message"] = trimmed
	return entry, nil
}
