package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRegression_CommaMillisecondTimestamps(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "python logging default",
			line: `2024-01-15 10:30:45,123 ERROR db timeout`,
			want: `{"level":"ERROR","message":"db timeout","timestamp":"2024-01-15 10:30:45,123"}`,
		},
		{
			name: "level first",
			line: `WARN 2024-01-15 10:30:45,123 slow query`,
			want: `{"level":"WARN","message":"slow query","timestamp":"2024-01-15 10:30:45,123"}`,
		},
		{
			name: "bracketed level, log4j style",
			line: `2024-01-15 10:30:45,123 [INFO] started`,
			want: `{"level":"INFO","message":"started","timestamp":"2024-01-15 10:30:45,123"}`,
		},
		{
			name: "dot milliseconds are unchanged",
			line: `2024-01-15 10:30:45.123 ERROR db timeout`,
			want: `{"level":"ERROR","message":"db timeout","timestamp":"2024-01-15 10:30:45.123"}`,
		},
		{
			name: "no fraction is unchanged",
			line: `2024-01-15 10:30:45 ERROR db timeout`,
			want: `{"level":"ERROR","message":"db timeout","timestamp":"2024-01-15 10:30:45"}`,
		},
	}

	for _, cfg := range []Config{{}, {Format: "generic"}} {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				if err := runPipeline(cfg, strings.NewReader(tt.line+"\n"), &stdout, &stderr); err != nil {
					t.Fatal(err)
				}
				if got := strings.TrimSpace(stdout.String()); got != tt.want {
					t.Errorf("\n got: %s\nwant: %s", got, tt.want)
				}
			})
		}
	}
}
