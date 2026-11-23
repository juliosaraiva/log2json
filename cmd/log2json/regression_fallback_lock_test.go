package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRegression_UnstructuredFirstLineDoesNotLockDetection(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "banner then JSON",
			input: "Starting service v1.2\n" + `{"level":"info","msg":"ready"}` + "\n",
			want:  []string{`{"message":"Starting service v1.2"}`, `{"level":"info","msg":"ready"}`},
		},
		{
			name:  "banner then key=value",
			input: "==> /var/log/app.log <==\nlevel=info port=8080\n",
			want:  []string{`{"message":"==> /var/log/app.log <=="}`, `{"level":"info","port":8080}`},
		},
		{
			name:  "banner then syslog",
			input: "-- Logs begin --\nJan 15 10:30:45 host sshd[7]: ok\n",
			want:  []string{`{"message":"-- Logs begin --"}`, `{"host":"host","message":"ok","pid":7,"program":"sshd","timestamp":"Jan 15 10:30:45"}`},
		},
		{
			name:  "structured generic first line still locks the format",
			input: "2024-01-15 10:30:45 INFO started\nlevel=info port=8080\n",
			want:  []string{`{"level":"INFO","message":"started","timestamp":"2024-01-15 10:30:45"}`, `{"message":"level=info port=8080"}`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if err := runPipeline(Config{}, strings.NewReader(tt.input), &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			got := strings.Split(strings.TrimSpace(stdout.String()), "\n")
			if len(got) != len(tt.want) {
				t.Fatalf("got %d records, want %d:\n%s", len(got), len(tt.want), stdout.String())
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("record %d\n got: %s\nwant: %s", i+1, got[i], tt.want[i])
				}
			}
		})
	}
}
