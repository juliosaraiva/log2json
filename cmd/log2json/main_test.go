package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// helper to run the pipeline and return stdout/stderr output
func runTest(t *testing.T, cfg Config, input string) (stdout string, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	err := runPipeline(cfg, strings.NewReader(input), &out, &errOut)
	if err != nil {
		t.Fatalf("runPipeline returned error: %v", err)
	}
	return out.String(), errOut.String()
}

// helper to parse each line of NDJSON output into maps
func parseNDJSON(t *testing.T, output string) []map[string]any {
	t.Helper()
	var results []map[string]any
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	for i, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("line %d is not valid JSON: %v\nline: %s", i+1, err, line)
		}
		results = append(results, m)
	}
	return results
}

func TestIntegration_SyslogFormat(t *testing.T) {
	input := `Jan 15 10:30:45 webserver nginx[1234]: 192.168.1.100 - - GET /index.html
Jan 15 10:30:46 webserver sshd[5678]: Accepted password for user from 192.168.1.1`

	stdout, _ := runTest(t, Config{Quiet: true}, input)
	results := parseNDJSON(t, stdout)

	if len(results) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(results))
	}

	// First line
	if results[0]["host"] != "webserver" {
		t.Errorf("expected host=webserver, got %v", results[0]["host"])
	}
	if results[0]["program"] != "nginx" {
		t.Errorf("expected program=nginx, got %v", results[0]["program"])
	}
	// PID should be a number (JSON decodes as float64)
	if pid, ok := results[0]["pid"].(float64); !ok || pid != 1234 {
		t.Errorf("expected pid=1234, got %v", results[0]["pid"])
	}
}

func TestIntegration_ApacheFormat(t *testing.T) {
	input := `192.168.1.1 - john [15/Jan/2024:10:30:45 +0000] "GET /index.html HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`

	stdout, _ := runTest(t, Config{Format: "apache", Quiet: true}, input)
	results := parseNDJSON(t, stdout)

	if len(results) != 1 {
		t.Fatalf("expected 1 line, got %d", len(results))
	}

	r := results[0]
	if r["ip"] != "192.168.1.1" {
		t.Errorf("expected ip=192.168.1.1, got %v", r["ip"])
	}
	if r["user"] != "john" {
		t.Errorf("expected user=john, got %v", r["user"])
	}
	if r["method"] != "GET" {
		t.Errorf("expected method=GET, got %v", r["method"])
	}
	if r["path"] != "/index.html" {
		t.Errorf("expected path=/index.html, got %v", r["path"])
	}
	// status is int in Go but float64 in JSON
	if status, ok := r["status"].(float64); !ok || status != 200 {
		t.Errorf("expected status=200, got %v", r["status"])
	}
	if size, ok := r["size"].(float64); !ok || size != 1234 {
		t.Errorf("expected size=1234, got %v", r["size"])
	}
}

func TestIntegration_JSONFormat(t *testing.T) {
	input := `{"level":"INFO","message":"Application started","port":8080}
{"level":"ERROR","message":"Connection failed","code":500}`

	stdout, _ := runTest(t, Config{Quiet: true}, input)
	results := parseNDJSON(t, stdout)

	if len(results) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(results))
	}

	if results[0]["level"] != "INFO" {
		t.Errorf("expected level=INFO, got %v", results[0]["level"])
	}
	if results[0]["message"] != "Application started" {
		t.Errorf("expected message=Application started, got %v", results[0]["message"])
	}
	if port, ok := results[0]["port"].(float64); !ok || port != 8080 {
		t.Errorf("expected port=8080, got %v", results[0]["port"])
	}
}

func TestIntegration_KVFormat(t *testing.T) {
	input := `time=2024-01-15T10:30:45Z level=info msg="Server started" port=8080`

	stdout, _ := runTest(t, Config{Quiet: true}, input)
	results := parseNDJSON(t, stdout)

	if len(results) != 1 {
		t.Fatalf("expected 1 line, got %d", len(results))
	}

	r := results[0]
	if r["level"] != "info" {
		t.Errorf("expected level=info, got %v", r["level"])
	}
	if r["msg"] != "Server started" {
		t.Errorf("expected msg=Server started, got %v", r["msg"])
	}
	// port is int64 in Go but float64 in JSON
	if port, ok := r["port"].(float64); !ok || port != 8080 {
		t.Errorf("expected port=8080, got %v", r["port"])
	}
}

func TestIntegration_ForcedFormat(t *testing.T) {
	input := `Jan 15 10:30:45 myhost prog[99]: hello world`

	stdout, _ := runTest(t, Config{Format: "syslog", Quiet: true}, input)
	results := parseNDJSON(t, stdout)

	if len(results) != 1 {
		t.Fatalf("expected 1 line, got %d", len(results))
	}

	if results[0]["program"] != "prog" {
		t.Errorf("expected program=prog, got %v", results[0]["program"])
	}
}

func TestIntegration_CustomPattern(t *testing.T) {
	input := `2024-01-15 INFO hello world
2024-01-16 ERROR something failed`

	cfg := Config{
		Pattern: `(?P<date>\d{4}-\d{2}-\d{2}) (?P<level>\w+) (?P<msg>.+)`,
		Quiet:   true,
	}

	stdout, _ := runTest(t, cfg, input)
	results := parseNDJSON(t, stdout)

	if len(results) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(results))
	}

	if results[0]["date"] != "2024-01-15" {
		t.Errorf("expected date=2024-01-15, got %v", results[0]["date"])
	}
	if results[0]["level"] != "INFO" {
		t.Errorf("expected level=INFO, got %v", results[0]["level"])
	}
	if results[0]["msg"] != "hello world" {
		t.Errorf("expected msg=hello world, got %v", results[0]["msg"])
	}
}

func TestIntegration_AdaptiveMode(t *testing.T) {
	input := `{"level":"info","msg":"json line"}
Jan 15 10:30:46 host prog[1]: syslog line`

	cfg := Config{
		Adaptive: true,
		Quiet:    true,
	}

	stdout, _ := runTest(t, cfg, input)
	results := parseNDJSON(t, stdout)

	if len(results) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(results))
	}

	// First line: JSON
	if results[0]["level"] != "info" {
		t.Errorf("expected level=info, got %v", results[0]["level"])
	}
	if results[0]["msg"] != "json line" {
		t.Errorf("expected msg=json line, got %v", results[0]["msg"])
	}

	// Second line: syslog
	if results[1]["host"] != "host" {
		t.Errorf("expected host=host, got %v", results[1]["host"])
	}
	if results[1]["program"] != "prog" {
		t.Errorf("expected program=prog, got %v", results[1]["program"])
	}
}

func TestIntegration_FieldFiltering(t *testing.T) {
	input := `Jan 15 10:30:45 myhost sshd[1234]: Accepted password`

	cfg := Config{
		Fields: []string{"host", "message"},
		Quiet:  true,
	}

	stdout, _ := runTest(t, cfg, input)
	results := parseNDJSON(t, stdout)

	if len(results) != 1 {
		t.Fatalf("expected 1 line, got %d", len(results))
	}

	r := results[0]
	if r["host"] != "myhost" {
		t.Errorf("expected host=myhost, got %v", r["host"])
	}
	if r["message"] != "Accepted password" {
		t.Errorf("expected message=Accepted password, got %v", r["message"])
	}
	// Should NOT contain other fields
	if _, ok := r["program"]; ok {
		t.Error("expected program field to be filtered out")
	}
	if _, ok := r["pid"]; ok {
		t.Error("expected pid field to be filtered out")
	}
}

func TestIntegration_OmitEmpty(t *testing.T) {
	input := "valid line\n\nanother valid line"

	cfg := Config{
		OmitEmpty: true,
		Quiet:     true,
	}

	stdout, _ := runTest(t, cfg, input)
	results := parseNDJSON(t, stdout)

	// Empty line should be omitted (has ParseError=ErrEmptyLine)
	if len(results) != 2 {
		t.Fatalf("expected 2 lines (empty omitted), got %d", len(results))
	}
}

func TestIntegration_AddMetadata(t *testing.T) {
	input := `Jan 15 10:30:45 myhost sshd[1234]: test message`

	cfg := Config{
		AddTimestamp:  true,
		AddLineNumber: true,
		AddRaw:        true,
		Quiet:         true,
	}

	stdout, _ := runTest(t, cfg, input)
	results := parseNDJSON(t, stdout)

	if len(results) != 1 {
		t.Fatalf("expected 1 line, got %d", len(results))
	}

	r := results[0]
	if _, ok := r["_ingestTime"]; !ok {
		t.Error("expected _ingestTime field")
	}
	if lineNum, ok := r["_lineNumber"].(float64); !ok || lineNum != 1 {
		t.Errorf("expected _lineNumber=1, got %v", r["_lineNumber"])
	}
	if r["_raw"] != input {
		t.Errorf("expected _raw to be original line, got %v", r["_raw"])
	}
}

func TestIntegration_VerboseMode(t *testing.T) {
	input := `Jan 15 10:30:45 myhost sshd[1234]: test`

	cfg := Config{
		Verbose: true,
		Quiet:   true,
	}

	_, stderr := runTest(t, cfg, input)

	if !strings.Contains(stderr, "processed 1 lines") {
		t.Errorf("expected verbose summary in stderr, got: %s", stderr)
	}
}

func TestIntegration_UnknownFormat(t *testing.T) {
	var out, errOut bytes.Buffer
	cfg := Config{Format: "bogus"}
	err := runPipeline(cfg, strings.NewReader("test"), &out, &errOut)
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("expected unknown format error, got: %v", err)
	}
}

func TestIntegration_InvalidPattern(t *testing.T) {
	var out, errOut bytes.Buffer
	cfg := Config{Pattern: "(?P<broken"}
	err := runPipeline(cfg, strings.NewReader("test"), &out, &errOut)
	if err == nil {
		t.Fatal("expected error for invalid pattern")
	}
	if !strings.Contains(err.Error(), "invalid pattern") {
		t.Errorf("expected invalid pattern error, got: %v", err)
	}
}

func TestIntegration_WithSampleFiles(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		format   string
		minLines int
	}{
		{"syslog_file", "../../testdata/sample_syslog.log", "syslog", 6},
		{"apache_file", "../../testdata/sample_apache.log", "apache", 5},
		{"json_file", "../../testdata/sample_json.log", "json", 5},
		{"kv_file", "../../testdata/sample_kv.log", "kv", 5},
		{"generic_file", "../../testdata/sample_generic.log", "", 6},
		{"mixed_file", "../../testdata/sample_mixed.log", "", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Skipf("sample file not found: %s", tt.file)
			}

			cfg := Config{Quiet: true}
			if tt.format != "" {
				cfg.Format = tt.format
			}
			if tt.name == "mixed_file" {
				cfg.Adaptive = true
			}

			var out, errOut bytes.Buffer
			err = runPipeline(cfg, bytes.NewReader(data), &out, &errOut)
			if err != nil {
				t.Fatalf("runPipeline error: %v", err)
			}

			results := parseNDJSON(t, out.String())
			if len(results) < tt.minLines {
				t.Errorf("expected at least %d lines, got %d", tt.minLines, len(results))
			}

			// Verify each line is valid JSON (already done by parseNDJSON)
			for i, r := range results {
				if len(r) == 0 {
					t.Errorf("line %d has no fields", i+1)
				}
			}
		})
	}
}

func TestIntegration_PrettyOutput(t *testing.T) {
	input := `{"level":"info","msg":"test"}`

	cfg := Config{
		Pretty: true,
		Quiet:  true,
	}

	stdout, _ := runTest(t, cfg, input)

	// Pretty output should contain indentation
	if !strings.Contains(stdout, "  ") {
		t.Error("expected indented output with --pretty")
	}

	// Should still be valid JSON
	var m map[string]any
	if err := json.Unmarshal([]byte(stdout), &m); err != nil {
		t.Fatalf("pretty output is not valid JSON: %v", err)
	}
}

func TestIntegration_EmptyInput(t *testing.T) {
	stdout, _ := runTest(t, Config{Quiet: true, OmitEmpty: true}, "")

	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty output for empty input with omit-empty, got: %s", stdout)
	}
}

// Ensure runPipeline writes nothing if input is empty and OmitEmpty is false
func TestIntegration_EmptyInputNoOmit(t *testing.T) {
	var out, errOut bytes.Buffer
	err := runPipeline(Config{Quiet: true}, strings.NewReader(""), &out, &errOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No lines to process, so output should be empty
	if out.Len() != 0 {
		t.Errorf("expected empty output, got: %s", out.String())
	}
}

// Ensure Close is called even when no lines processed (via defer)
func TestIntegration_CloseOnEmpty(t *testing.T) {
	var out bytes.Buffer
	err := runPipeline(Config{Quiet: true}, strings.NewReader(""), &out, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
