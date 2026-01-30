package parser

import (
	"regexp"
	"strconv"
)

// ApacheParser handles Apache/Nginx Combined Log Format.
// Example: 192.168.1.1 - user [15/Jan/2024:10:30:45 +0000] "GET /page HTTP/1.1" 200 1234 "http://ref.com" "Mozilla/5.0"
type ApacheParser struct {
	pattern *regexp.Regexp
}

// NewApacheParser creates a new Apache combined log format parser.
func NewApacheParser() *ApacheParser {
	// Combined Log Format pattern
	pattern := regexp.MustCompile(
		`^(?P<ip>\S+)\s+` + // IP address
			`(?P<ident>\S+)\s+` + // Ident (usually -)
			`(?P<user>\S+)\s+` + // User (usually -)
			`\[(?P<timestamp>[^\]]+)\]\s+` + // Timestamp in brackets
			`"(?P<method>\S+)\s+(?P<path>\S+)\s+(?P<protocol>[^"]+)"\s+` + // Request line
			`(?P<status>\d+)\s+` + // Status code
			`(?P<size>\S+)` + // Response size (or -)
			`(?:\s+"(?P<referer>[^"]*)"\s+"(?P<useragent>[^"]*)")?`, // Optional referer and user agent
	)
	return &ApacheParser{pattern: pattern}
}

// Name returns the parser identifier.
func (p *ApacheParser) Name() string {
	return "apache"
}

// Description returns a human-readable description.
func (p *ApacheParser) Description() string {
	return "Apache/Nginx Combined Log Format"
}

// CanParse checks if the line matches Apache log format.
// Quick check: contains timestamp in brackets and quoted request.
func (p *ApacheParser) CanParse(line string) bool {
	return p.pattern.MatchString(line)
}

// Parse extracts fields from an Apache log line.
func (p *ApacheParser) Parse(line string) (*Entry, error) {
	entry := NewEntry(line)

	matches := p.pattern.FindStringSubmatch(line)
	if matches == nil {
		entry.ParseError = ErrNoMatch
		entry.Fields["raw"] = line
		return entry, nil
	}

	names := p.pattern.SubexpNames()
	for i, match := range matches {
		if i == 0 || names[i] == "" || match == "" || match == "-" {
			continue
		}

		name := names[i]

		// Convert numeric fields
		switch name {
		case "status":
			if status, err := strconv.Atoi(match); err == nil {
				entry.Fields[name] = status
				continue
			}
		case "size":
			if size, err := strconv.ParseInt(match, 10, 64); err == nil {
				entry.Fields[name] = size
				continue
			}
		}

		entry.Fields[name] = match
	}

	return entry, nil
}
