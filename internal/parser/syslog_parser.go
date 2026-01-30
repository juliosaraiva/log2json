package parser

import (
	"regexp"
	"strconv"
)

// SyslogParser handles traditional syslog format.
// Example: Jan 15 10:30:45 myhost sshd[1234]: Accepted password for user
type SyslogParser struct {
	pattern *regexp.Regexp
}

// NewSyslogParser creates a new syslog format parser.
func NewSyslogParser() *SyslogParser {
	// Syslog format: timestamp hostname program[pid]: message
	// Timestamp: "Jan 15 10:30:45" or "2024-01-15T10:30:45"
	pattern := regexp.MustCompile(
		`^(?P<timestamp>(?:\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})|(?:\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?))` +
			`\s+(?P<host>\S+)` +
			`\s+(?P<program>[^\s\[:]+)` +
			`(?:\[(?P<pid>\d+)\])?` +
			`:\s*(?P<message>.*)$`,
	)
	return &SyslogParser{pattern: pattern}
}

// Name returns the parser identifier.
func (p *SyslogParser) Name() string {
	return "syslog"
}

// Description returns a human-readable description.
func (p *SyslogParser) Description() string {
	return "Standard syslog format (RFC 3164/5424)"
}

// CanParse checks if the line matches syslog format.
func (p *SyslogParser) CanParse(line string) bool {
	return p.pattern.MatchString(line)
}

// Parse extracts fields from a syslog line.
func (p *SyslogParser) Parse(line string) (*Entry, error) {
	entry := NewEntry(line)

	matches := p.pattern.FindStringSubmatch(line)
	if matches == nil {
		entry.ParseError = ErrNoMatch
		entry.Fields["raw"] = line
		return entry, nil
	}

	// Extract named groups
	names := p.pattern.SubexpNames()
	for i, match := range matches {
		if i == 0 || names[i] == "" || match == "" {
			continue
		}

		// Convert PID to integer
		if names[i] == "pid" {
			if pid, err := strconv.Atoi(match); err == nil {
				entry.Fields[names[i]] = pid
				continue
			}
		}

		entry.Fields[names[i]] = match
	}

	return entry, nil
}
