package parser

import (
	"errors"
	"testing"
)

func TestNewRegexParser(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		wantError bool
	}{
		{
			name:      "valid pattern with named groups",
			pattern:   `(?P<level>\w+)\s+(?P<message>.+)`,
			wantError: false,
		},
		{
			name:      "invalid regex",
			pattern:   `(?P<level>\w+`,
			wantError: true,
		},
		{
			name:      "no named groups",
			pattern:   `(\w+)\s+(.+)`,
			wantError: true,
		},
		{
			name:      "single named group",
			pattern:   `(?P<msg>.+)`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewRegexParser(tt.pattern)
			if tt.wantError {
				if err == nil {
					t.Errorf("NewRegexParser(%q): expected error, got nil", tt.pattern)
				}
				if p != nil {
					t.Errorf("NewRegexParser(%q): expected nil parser on error", tt.pattern)
				}
			} else {
				if err != nil {
					t.Errorf("NewRegexParser(%q): unexpected error: %v", tt.pattern, err)
				}
				if p == nil {
					t.Errorf("NewRegexParser(%q): expected non-nil parser", tt.pattern)
				}
			}
		})
	}
}

func TestRegexParser_CanParse(t *testing.T) {
	p, err := NewRegexParser(`(?P<level>INFO|ERROR)\s+(?P<message>.+)`)
	if err != nil {
		t.Fatalf("NewRegexParser failed: %v", err)
	}

	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "matching line",
			line: "INFO something happened",
			want: true,
		},
		{
			name: "non-matching line",
			line: "DEBUG something happened",
			want: false,
		},
		{
			name: "empty line",
			line: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.CanParse(tt.line)
			if got != tt.want {
				t.Errorf("CanParse(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestRegexParser_Parse(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		line           string
		wantFields     map[string]any
		wantParseError error
	}{
		{
			name:    "successful extraction",
			pattern: `(?P<level>\w+)\s+(?P<message>.+)`,
			line:    "INFO server started",
			wantFields: map[string]any{
				"level":   "INFO",
				"message": "server started",
			},
		},
		{
			name:    "numeric type inference",
			pattern: `(?P<code>\d+)\s+(?P<msg>.+)`,
			line:    "200 OK",
			wantFields: map[string]any{
				"code": int64(200),
				"msg":  "OK",
			},
		},
		{
			name:           "no match",
			pattern:        `(?P<level>INFO|ERROR)\s+(?P<message>.+)`,
			line:           "DEBUG this won't match",
			wantParseError: ErrNoMatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewRegexParser(tt.pattern)
			if err != nil {
				t.Fatalf("NewRegexParser(%q) failed: %v", tt.pattern, err)
			}

			entry, err := p.Parse(tt.line)
			if err != nil {
				t.Fatalf("Parse(%q) returned unexpected error: %v", tt.line, err)
			}

			if tt.wantParseError != nil {
				if entry.ParseError == nil {
					t.Errorf("Parse(%q): expected ParseError %v, got nil", tt.line, tt.wantParseError)
				} else if !errors.Is(entry.ParseError, tt.wantParseError) {
					t.Errorf("Parse(%q): ParseError = %v, want %v", tt.line, entry.ParseError, tt.wantParseError)
				}
				if _, ok := entry.Fields["raw"]; !ok {
					t.Errorf("Parse(%q): expected 'raw' field on no match", tt.line)
				}
				return
			}

			if entry.ParseError != nil {
				t.Errorf("Parse(%q): unexpected ParseError: %v", tt.line, entry.ParseError)
			}

			for key, want := range tt.wantFields {
				got, ok := entry.Fields[key]
				if !ok {
					t.Errorf("Parse(%q): missing field %q", tt.line, key)
					continue
				}
				if got != want {
					t.Errorf("Parse(%q): field %q = %v (%T), want %v (%T)", tt.line, key, got, got, want, want)
				}
			}
		})
	}
}
