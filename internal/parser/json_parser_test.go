package parser

import (
	"testing"
)

func TestJSONParser_CanParse(t *testing.T) {
	p := NewJSONParser()

	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "valid JSON object",
			line: `{"key": "value"}`,
			want: true,
		},
		{
			name: "broken JSON with braces",
			line: `{this is not json}`,
			want: true,
		},
		{
			name: "plain text",
			line: "hello world",
			want: false,
		},
		{
			name: "JSON array",
			line: `["a", "b"]`,
			want: false,
		},
		{
			name: "whitespace-wrapped JSON",
			line: `   {"key": "value"}   `,
			want: true,
		},
		{
			name: "empty string",
			line: "",
			want: false,
		},
		{
			name: "single character brace open",
			line: "{",
			want: false,
		},
		{
			name: "minimal braces",
			line: "{}",
			want: true,
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

func TestJSONParser_Parse(t *testing.T) {
	p := NewJSONParser()

	tests := []struct {
		name           string
		line           string
		wantFields     map[string]any
		wantParseError bool
		wantRawField   bool
	}{
		{
			name: "simple object",
			line: `{"level": "info", "msg": "hello"}`,
			wantFields: map[string]any{
				"level": "info",
				"msg":   "hello",
			},
			wantParseError: false,
		},
		{
			name: "nested object",
			line: `{"user": {"name": "alice"}}`,
			wantParseError: false,
		},
		{
			name: "numbers and booleans",
			line: `{"count": 42, "pi": 3.14, "ok": true}`,
			wantFields: map[string]any{
				"count": float64(42),
				"pi":    float64(3.14),
				"ok":    true,
			},
			wantParseError: false,
		},
		{
			name:           "invalid JSON",
			line:           `{not valid json}`,
			wantParseError: true,
			wantRawField:   true,
		},
		{
			name:           "empty object",
			line:           `{}`,
			wantParseError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := p.Parse(tt.line)
			if err != nil {
				t.Fatalf("Parse(%q) returned unexpected error: %v", tt.line, err)
			}

			if tt.wantParseError && entry.ParseError == nil {
				t.Errorf("Parse(%q): expected ParseError to be set", tt.line)
			}
			if !tt.wantParseError && entry.ParseError != nil {
				t.Errorf("Parse(%q): unexpected ParseError: %v", tt.line, entry.ParseError)
			}

			if tt.wantRawField {
				if _, ok := entry.Fields["raw"]; !ok {
					t.Errorf("Parse(%q): expected 'raw' field in Fields", tt.line)
				}
				if _, ok := entry.Fields["_parseError"]; !ok {
					t.Errorf("Parse(%q): expected '_parseError' field in Fields", tt.line)
				}
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
