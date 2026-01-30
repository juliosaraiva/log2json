package parser

import (
	"errors"
	"testing"
)

func TestGenericParser_CanParse(t *testing.T) {
	p := NewGenericParser()

	tests := []struct {
		name string
		line string
	}{
		{name: "plain text", line: "hello world"},
		{name: "empty string", line: ""},
		{name: "JSON-like", line: `{"key": "val"}`},
		{name: "syslog-like", line: "Jan 15 10:30:45 host prog: msg"},
		{name: "random numbers", line: "12345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.CanParse(tt.line)
			if !got {
				t.Errorf("CanParse(%q) = false, want true (generic always returns true)", tt.line)
			}
		})
	}
}

func TestGenericParser_Parse(t *testing.T) {
	p := NewGenericParser()

	tests := []struct {
		name           string
		line           string
		wantFields     map[string]any
		wantParseError error
	}{
		{
			name: "pattern 0: ISO timestamp space level",
			line: "2024-01-15 10:30:45 INFO message here",
			wantFields: map[string]any{
				"timestamp": "2024-01-15 10:30:45",
				"level":     "INFO",
				"message":   "message here",
			},
		},
		{
			name: "pattern 0: ISO timestamp T separator with Z",
			line: "2024-01-15T10:30:45Z ERROR fail",
			wantFields: map[string]any{
				"timestamp": "2024-01-15T10:30:45Z",
				"level":     "ERROR",
				"message":   "fail",
			},
		},
		{
			name: "pattern 1: level first",
			line: "INFO 2024-01-15 10:30:45 message",
			wantFields: map[string]any{
				"level":     "INFO",
				"timestamp": "2024-01-15 10:30:45",
				"message":   "message",
			},
		},
		{
			name: "pattern 2: bracketed level",
			line: "[WARN] Something happened",
			wantFields: map[string]any{
				"level":   "WARN",
				"message": "Something happened",
			},
		},
		{
			name: "pattern 3: level colon message",
			line: "ERROR: something broke",
			wantFields: map[string]any{
				"level":   "ERROR",
				"message": "something broke",
			},
		},
		{
			name: "pattern 3: level dash message",
			line: "INFO - some message",
			wantFields: map[string]any{
				"level":   "INFO",
				"message": "some message",
			},
		},
		{
			name: "fallback: unstructured text",
			line: "random 12345 text",
			wantFields: map[string]any{
				"message": "random 12345 text",
			},
		},
		{
			name: "empty line",
			line: "",
			wantFields: map[string]any{
				"message": "",
			},
			wantParseError: ErrEmptyLine,
		},
		{
			name: "whitespace-only line",
			line: "   ",
			wantFields: map[string]any{
				"message": "",
			},
			wantParseError: ErrEmptyLine,
		},
		{
			name: "lowercase level does not match patterns",
			line: "info this should fallback",
			wantFields: map[string]any{
				"message": "info this should fallback",
			},
		},
		{
			name: "WARNING level variant",
			line: "2024-01-15 10:30:45 WARNING watch out",
			wantFields: map[string]any{
				"timestamp": "2024-01-15 10:30:45",
				"level":     "WARNING",
				"message":   "watch out",
			},
		},
		{
			name: "TRACE level",
			line: "TRACE: deep debug info",
			wantFields: map[string]any{
				"level":   "TRACE",
				"message": "deep debug info",
			},
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
			} else if entry.ParseError != nil {
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
