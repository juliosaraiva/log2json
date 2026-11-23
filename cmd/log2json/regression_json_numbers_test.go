package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRegression_JSONNumbersSurviveUnchanged(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"integer above 2^53", `{"id":9007199254740993}`},
		{"20 digit trace id", `{"trace":12345678901234567890}`},
		{"negative 64-bit integer", `{"offset":-9223372036854775808}`},
		{"high precision decimal", `{"price":0.1234567890123456789}`},
		{"nested large integer", `{"span":{"parent":18446744073709551615}}`},
		{"large integer in array", `{"ids":[9007199254740993,9007199254740995]}`},
		{"ordinary numbers", `{"count":42,"ratio":3.14}`},
	}

	for _, mode := range []Config{{}, {Format: "json"}, {Adaptive: true}} {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				if err := runPipeline(mode, strings.NewReader(tt.line+"\n"), &stdout, &stderr); err != nil {
					t.Fatal(err)
				}
				if got := strings.TrimSpace(stdout.String()); got != tt.line {
					t.Errorf("number changed in transit\n in: %s\nout: %s", tt.line, got)
				}
				if stderr.Len() != 0 {
					t.Errorf("unexpected stderr: %s", stderr.String())
				}
			})
		}
	}
}

func TestRegression_JSONTrailingDataIsStillAParseError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := Config{Format: "json"}
	if err := runPipeline(cfg, strings.NewReader(`{"a":1} {"b":2}`+"\n"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"_parseError"`) {
		t.Errorf("two objects on one line must stay a parse error, got: %s", stdout.String())
	}
}
