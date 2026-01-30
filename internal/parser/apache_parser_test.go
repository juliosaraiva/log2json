package parser

import (
	"errors"
	"testing"
)

func TestApacheParser_CanParse(t *testing.T) {
	p := NewApacheParser()

	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "valid combined log line",
			line: `192.168.1.1 - admin [15/Jan/2024:10:30:45 +0000] "GET /index.html HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
			want: true,
		},
		{
			name: "plain text",
			line: "this is just plain text",
			want: false,
		},
		{
			name: "partial log line",
			line: `192.168.1.1 - - [15/Jan/2024:10:30:45 +0000] "GET / HTTP/1.1" 200 512`,
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

func TestApacheParser_Parse(t *testing.T) {
	p := NewApacheParser()

	tests := []struct {
		name           string
		line           string
		wantFields     map[string]any
		wantAbsent     []string
		wantParseError error
	}{
		{
			name: "full combined log",
			line: `192.168.1.1 - admin [15/Jan/2024:10:30:45 +0000] "GET /page HTTP/1.1" 200 1234 "http://ref.com" "Mozilla/5.0"`,
			wantFields: map[string]any{
				"ip":        "192.168.1.1",
				"user":      "admin",
				"timestamp": "15/Jan/2024:10:30:45 +0000",
				"method":    "GET",
				"path":      "/page",
				"protocol":  "HTTP/1.1",
				"status":    200,
				"size":      int64(1234),
				"referer":   "http://ref.com",
				"useragent": "Mozilla/5.0",
			},
		},
		{
			name: "dash user is absent",
			line: `10.0.0.1 - - [15/Jan/2024:10:30:45 +0000] "POST /api HTTP/1.1" 201 56 "http://example.com" "curl/7.68"`,
			wantFields: map[string]any{
				"ip":        "10.0.0.1",
				"timestamp": "15/Jan/2024:10:30:45 +0000",
				"method":    "POST",
				"path":      "/api",
				"protocol":  "HTTP/1.1",
				"status":    201,
				"size":      int64(56),
				"referer":   "http://example.com",
				"useragent": "curl/7.68",
			},
			wantAbsent: []string{"user", "ident"},
		},
		{
			name: "status and size type conversion",
			line: `1.2.3.4 - user [01/Feb/2024:00:00:00 +0000] "GET /test HTTP/2.0" 404 0 "http://x.com" "Bot/1.0"`,
			wantFields: map[string]any{
				"status": 404,
				"size":   int64(0),
			},
		},
		{
			name:           "no match",
			line:           "this is not an apache log line",
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
