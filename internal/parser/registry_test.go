package parser

import (
	"errors"
	"strings"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	parsers := r.ListParsers()

	expectedOrder := []string{"json", "kv", "syslog", "apache", "generic"}

	if len(parsers) != len(expectedOrder) {
		t.Fatalf("NewRegistry: expected %d parsers, got %d", len(expectedOrder), len(parsers))
	}

	for i, expected := range expectedOrder {
		if parsers[i].Name != expected {
			t.Errorf("NewRegistry: parser[%d].Name = %q, want %q", i, parsers[i].Name, expected)
		}
	}
}

func TestRegistry_GetParser(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name      string
		format    string
		wantFound bool
	}{
		{name: "json lowercase", format: "json", wantFound: true},
		{name: "JSON uppercase", format: "JSON", wantFound: true},
		{name: "kv format", format: "kv", wantFound: true},
		{name: "syslog format", format: "syslog", wantFound: true},
		{name: "apache format", format: "apache", wantFound: true},
		{name: "generic format", format: "generic", wantFound: true},
		{name: "mixed case Json", format: "Json", wantFound: true},
		{name: "nonexistent", format: "nonexistent", wantFound: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := r.GetParser(tt.format)
			if tt.wantFound && p == nil {
				t.Errorf("GetParser(%q) = nil, want non-nil", tt.format)
			}
			if !tt.wantFound && p != nil {
				t.Errorf("GetParser(%q) = %v, want nil", tt.format, p)
			}
		})
	}
}

func TestRegistry_Parse_AutoDetect(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantFields []string // fields that should be present
	}{
		{
			name:       "JSON line",
			line:       `{"level": "info", "msg": "hello"}`,
			wantFields: []string{"level", "msg"},
		},
		{
			name:       "key-value line",
			line:       `level=info msg=hello user=alice`,
			wantFields: []string{"level", "msg", "user"},
		},
		{
			name:       "syslog line",
			line:       "Jan 15 10:30:45 myhost sshd[1234]: Accepted password",
			wantFields: []string{"timestamp", "host", "program", "pid", "message"},
		},
		{
			name:       "apache line",
			line:       `192.168.1.1 - admin [15/Jan/2024:10:30:45 +0000] "GET /page HTTP/1.1" 200 1234 "http://ref.com" "Mozilla/5.0"`,
			wantFields: []string{"ip", "method", "path", "status"},
		},
		{
			name:       "plain text falls to generic",
			line:       "just some random text here",
			wantFields: []string{"message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry(WithAdaptiveMode())
			entry, err := r.Parse(tt.line)
			if err != nil {
				t.Fatalf("Parse(%q) returned error: %v", tt.line, err)
			}

			for _, field := range tt.wantFields {
				if _, ok := entry.Fields[field]; !ok {
					t.Errorf("Parse(%q): missing expected field %q, got fields: %v", tt.line, field, fieldKeys(entry.Fields))
				}
			}
		})
	}
}

func TestRegistry_Parse_EmptyLine(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name string
		line string
	}{
		{name: "empty string", line: ""},
		{name: "whitespace only", line: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := r.Parse(tt.line)
			if err != nil {
				t.Fatalf("Parse(%q) returned error: %v", tt.line, err)
			}
			if !errors.Is(entry.ParseError, ErrEmptyLine) {
				t.Errorf("Parse(%q): ParseError = %v, want %v", tt.line, entry.ParseError, ErrEmptyLine)
			}
		})
	}
}

func TestRegistry_Parse_ForcedFormat(t *testing.T) {
	r := NewRegistry(WithForcedFormat("syslog"))

	line := "Jan 15 10:30:45 myhost sshd[1234]: Accepted password"
	entry, err := r.Parse(line)
	if err != nil {
		t.Fatalf("Parse(%q) returned error: %v", line, err)
	}
	if entry.ParseError != nil {
		t.Errorf("Parse(%q): unexpected ParseError: %v", line, entry.ParseError)
	}

	wantFields := []string{"timestamp", "host", "program", "pid", "message"}
	for _, field := range wantFields {
		if _, ok := entry.Fields[field]; !ok {
			t.Errorf("Parse(%q): missing field %q", line, field)
		}
	}
}

func TestRegistry_Parse_UnknownFormat(t *testing.T) {
	r := NewRegistry(WithForcedFormat("bogus"))

	line := "some log line"
	_, err := r.Parse(line)
	if err == nil {
		t.Fatal("Parse with unknown format: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown format: bogus") {
		t.Errorf("Parse with unknown format: error = %q, want it to contain %q", err.Error(), "unknown format: bogus")
	}
}

func TestRegistry_Parse_CachingBehavior(t *testing.T) {
	// Strict mode (default): first successful parser is cached
	r := NewRegistry()

	// First line: JSON -> caches JSON parser
	jsonLine := `{"level": "info"}`
	entry1, err := r.Parse(jsonLine)
	if err != nil {
		t.Fatalf("Parse(%q) returned error: %v", jsonLine, err)
	}
	if entry1.ParseError != nil {
		t.Fatalf("Parse(%q): unexpected ParseError: %v", jsonLine, entry1.ParseError)
	}

	// Second line: syslog -> but cached parser (JSON) will be used
	syslogLine := "Jan 15 10:30:45 myhost sshd[1234]: message"
	entry2, err := r.Parse(syslogLine)
	if err != nil {
		t.Fatalf("Parse(%q) returned error: %v", syslogLine, err)
	}

	// The cached JSON parser should fail to parse syslog properly
	// It will get a ParseError because syslog is not valid JSON
	if entry2.ParseError == nil {
		t.Errorf("Parse(%q) in strict mode with cached JSON parser: expected ParseError, got nil", syslogLine)
	}
}

func TestRegistry_Parse_AdaptiveMode(t *testing.T) {
	r := NewRegistry(WithAdaptiveMode())

	// First line: JSON
	jsonLine := `{"level": "info"}`
	entry1, err := r.Parse(jsonLine)
	if err != nil {
		t.Fatalf("Parse(%q) returned error: %v", jsonLine, err)
	}
	if entry1.ParseError != nil {
		t.Fatalf("Parse(%q): unexpected ParseError: %v", jsonLine, entry1.ParseError)
	}
	if _, ok := entry1.Fields["level"]; !ok {
		t.Errorf("Parse(%q): missing 'level' field", jsonLine)
	}

	// Second line: syslog -> should re-detect and succeed
	syslogLine := "Jan 15 10:30:45 myhost sshd[1234]: Accepted password"
	entry2, err := r.Parse(syslogLine)
	if err != nil {
		t.Fatalf("Parse(%q) returned error: %v", syslogLine, err)
	}
	if entry2.ParseError != nil {
		t.Errorf("Parse(%q): unexpected ParseError: %v", syslogLine, entry2.ParseError)
	}

	wantFields := []string{"timestamp", "host", "program", "pid", "message"}
	for _, field := range wantFields {
		if _, ok := entry2.Fields[field]; !ok {
			t.Errorf("Parse(%q): missing field %q", syslogLine, field)
		}
	}
}

func TestRegistry_ListParsers(t *testing.T) {
	r := NewRegistry()
	parsers := r.ListParsers()

	if len(parsers) != 5 {
		t.Fatalf("ListParsers: expected 5 entries, got %d", len(parsers))
	}

	for _, p := range parsers {
		if p.Name == "" {
			t.Error("ListParsers: found entry with empty Name")
		}
		if p.Description == "" {
			t.Error("ListParsers: found entry with empty Description")
		}
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	initialCount := len(r.ListParsers())

	// Register a custom regex parser
	custom, err := NewRegexParser(`(?P<msg>.+)`)
	if err != nil {
		t.Fatalf("NewRegexParser failed: %v", err)
	}
	r.Register(custom)

	newCount := len(r.ListParsers())
	if newCount != initialCount+1 {
		t.Errorf("Register: expected %d parsers after registration, got %d", initialCount+1, newCount)
	}

	// The newly registered parser should be last
	parsers := r.ListParsers()
	last := parsers[len(parsers)-1]
	if last.Name != "regex" {
		t.Errorf("Register: last parser name = %q, want %q", last.Name, "regex")
	}
}

// fieldKeys returns a sorted list of keys from a map for diagnostic output.
func fieldKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
