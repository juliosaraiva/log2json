package parser

import (
	"errors"
	"testing"
)

func TestKeyValueParser_CanParse(t *testing.T) {
	p := NewKeyValueParser()

	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "multiple KV pairs",
			line: `level=info msg=hello user=alice`,
			want: true,
		},
		{
			name: "single pair below threshold",
			line: `level=info`,
			want: false,
		},
		{
			name: "no pairs",
			line: "this is plain text",
			want: false,
		},
		{
			name: "quoted values",
			line: `level=info msg="hello world"`,
			want: true,
		},
		{
			name: "single-quoted values",
			line: `name='alice' age=30`,
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

func TestKeyValueParser_Parse(t *testing.T) {
	p := NewKeyValueParser()

	tests := []struct {
		name           string
		line           string
		wantFields     map[string]any
		wantParseError error
	}{
		{
			name: "basic pairs",
			line: `level=info msg=hello`,
			wantFields: map[string]any{
				"level": "info",
				"msg":   "hello",
			},
		},
		{
			name: "double-quoted values",
			line: `msg="hello world" user=alice`,
			wantFields: map[string]any{
				"msg":  "hello world",
				"user": "alice",
			},
		},
		{
			name: "single-quoted values",
			line: `msg='hello world' user=bob`,
			wantFields: map[string]any{
				"msg":  "hello world",
				"user": "bob",
			},
		},
		{
			name: "numeric values",
			line: `count=123 rate=45.67`,
			wantFields: map[string]any{
				"count": int64(123),
				"rate":  float64(45.67),
			},
		},
		{
			name: "boolean values",
			line: `active=true disabled=false`,
			wantFields: map[string]any{
				"active":   true,
				"disabled": false,
			},
		},
		{
			name: "mixed types",
			line: `level=info count=42 rate=3.14 ok=true`,
			wantFields: map[string]any{
				"level": "info",
				"count": int64(42),
				"rate":  float64(3.14),
				"ok":    true,
			},
		},
		{
			name:           "no matches",
			line:           "this is plain text without any kv pairs",
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
		})
	}
}

func TestInferType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  any
	}{
		{
			name:  "integer",
			input: "123",
			want:  int64(123),
		},
		{
			name:  "float",
			input: "45.67",
			want:  float64(45.67),
		},
		{
			name:  "bool true lowercase",
			input: "true",
			want:  true,
		},
		{
			name:  "bool false lowercase",
			input: "false",
			want:  false,
		},
		{
			name:  "bool TRUE uppercase",
			input: "TRUE",
			want:  true,
		},
		{
			name:  "string",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "zero integer",
			input: "0",
			want:  int64(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferType(tt.input)
			if got != tt.want {
				t.Errorf("inferType(%q) = %v (%T), want %v (%T)", tt.input, got, got, tt.want, tt.want)
			}
		})
	}
}
