package parser

import (
	"fmt"
	"regexp"
)

// RegexParser handles custom user-defined patterns.
// Users provide a regex with named capture groups like (?P<field>pattern).
type RegexParser struct {
	pattern     *regexp.Regexp
	patternText string
}

// NewRegexParser creates a parser from a custom regex pattern.
// The pattern should use named capture groups: (?P<name>pattern)
// Returns error if the pattern is invalid.
func NewRegexParser(patternText string) (*RegexParser, error) {
	// Validate pattern compiles
	pattern, err := regexp.Compile(patternText)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Check that it has at least one named group
	names := pattern.SubexpNames()
	hasNamedGroup := false
	for _, name := range names {
		if name != "" {
			hasNamedGroup = true
			break
		}
	}
	if !hasNamedGroup {
		return nil, fmt.Errorf("pattern must have at least one named group: (?P<name>...)")
	}

	return &RegexParser{
		pattern:     pattern,
		patternText: patternText,
	}, nil
}

// Name returns the parser identifier.
func (p *RegexParser) Name() string {
	return "regex"
}

// Description returns a human-readable description.
func (p *RegexParser) Description() string {
	return fmt.Sprintf("Custom regex pattern: %s", p.patternText)
}

// CanParse checks if the line matches the custom pattern.
func (p *RegexParser) CanParse(line string) bool {
	return p.pattern.MatchString(line)
}

// Parse extracts named groups from the log line.
func (p *RegexParser) Parse(line string) (*Entry, error) {
	entry := NewEntry(line)

	matches := p.pattern.FindStringSubmatch(line)
	if matches == nil {
		entry.ParseError = ErrNoMatch
		entry.Fields["raw"] = line
		return entry, nil
	}

	names := p.pattern.SubexpNames()
	for i, match := range matches {
		if i == 0 || names[i] == "" {
			continue
		}
		// Try to infer type for numeric values
		entry.Fields[names[i]] = inferType(match)
	}

	return entry, nil
}
