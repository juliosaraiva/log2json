package parser

import (
	"errors"
	"testing"
)

func TestSyslogParser_CanParse(t *testing.T) {
	p := NewSyslogParser()

	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "standard syslog with PID",
			line: "Jan 15 10:30:45 myhost sshd[1234]: Accepted password for user",
			want: true,
		},
		{
			name: "ISO timestamp syslog",
			line: "2024-01-15T10:30:45Z myhost sshd[1234]: Accepted password for user",
			want: true,
		},
		{
			name: "plain text",
			line: "this is just plain text",
			want: false,
		},
		{
			name: "standard syslog without PID",
			line: "Jan 15 10:30:45 myhost kernel: some kernel message",
			want: true,
		},
		{
			name: "empty string",
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

func TestSyslogParser_Parse(t *testing.T) {
	p := NewSyslogParser()

	tests := []struct {
		name           string
		line           string
		wantFields     map[string]any
		wantAbsent     []string
		wantParseError error
	}{
		{
			name: "standard with PID",
			line: "Jan 15 10:30:45 myhost sshd[1234]: Accepted password for user",
			wantFields: map[string]any{
				"timestamp": "Jan 15 10:30:45",
				"host":      "myhost",
				"program":   "sshd",
				"pid":       1234,
				"message":   "Accepted password for user",
			},
		},
		{
			name: "without PID",
			line: "Jan 15 10:30:45 myhost kernel: some kernel message",
			wantFields: map[string]any{
				"timestamp": "Jan 15 10:30:45",
				"host":      "myhost",
				"program":   "kernel",
				"message":   "some kernel message",
			},
			wantAbsent: []string{"pid"},
		},
		{
			name: "ISO timestamp format",
			line: "2024-01-15T10:30:45Z myhost app[5678]: started successfully",
			wantFields: map[string]any{
				"timestamp": "2024-01-15T10:30:45Z",
				"host":      "myhost",
				"program":   "app",
				"pid":       5678,
				"message":   "started successfully",
			},
		},
		{
			name:           "no match",
			line:           "this is not a syslog line",
			wantParseError: ErrNoMatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
					t.Errorf("Parse(%q): expected 'raw' field on error", tt.line)
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

			for _, key := range tt.wantAbsent {
				if _, ok := entry.Fields[key]; ok {
					t.Errorf("Parse(%q): field %q should be absent but was present", tt.line, key)
				}
			}
		})
	}
}
