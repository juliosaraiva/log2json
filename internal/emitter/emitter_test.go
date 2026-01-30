package emitter

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/juliosaraiva/log2json/internal/parser"
)

func TestEmitter_Emit_Basic(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, Options{})

	entry := parser.NewEntry("level=info msg=hi")
	entry.Fields["level"] = "info"
	entry.Fields["msg"] = "hi"

	if err := em.Emit(entry); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	output := buf.String()

	// Must be terminated by newline (json.Encoder appends one)
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("output should end with newline, got %q", output)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if decoded["level"] != "info" {
		t.Errorf("expected level=info, got %v", decoded["level"])
	}
	if decoded["msg"] != "hi" {
		t.Errorf("expected msg=hi, got %v", decoded["msg"])
	}
}

func TestEmitter_Emit_Pretty(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, Options{Pretty: true})

	entry := parser.NewEntry("level=info")
	entry.Fields["level"] = "info"

	if err := em.Emit(entry); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "  ") {
		t.Errorf("pretty output should contain 2-space indentation, got:\n%s", output)
	}

	// Still must be valid JSON
	var decoded map[string]any
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("pretty output is not valid JSON: %v", err)
	}
}

func TestEmitter_Emit_FieldFiltering(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, Options{Fields: []string{"level"}})

	entry := parser.NewEntry("level=info msg=hi")
	entry.Fields["level"] = "info"
	entry.Fields["msg"] = "hi"

	if err := em.Emit(entry); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if decoded["level"] != "info" {
		t.Errorf("expected level=info, got %v", decoded["level"])
	}
	if _, exists := decoded["msg"]; exists {
		t.Errorf("field 'msg' should not be present when filtering for only 'level'")
	}
}

func TestEmitter_Emit_FieldFiltering_MissingField(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, Options{Fields: []string{"nonexistent"}})

	entry := parser.NewEntry("level=info")
	entry.Fields["level"] = "info"

	if err := em.Emit(entry); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}

	if _, exists := decoded["nonexistent"]; exists {
		t.Errorf("field 'nonexistent' should not be present since it was not in entry")
	}
	if _, exists := decoded["level"]; exists {
		t.Errorf("field 'level' should not be present since it was not in the filter list")
	}
}

func TestEmitter_Emit_AddLineNumber(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, Options{AddLineNumber: true})

	entry := parser.NewEntry("some line")
	entry.Fields["msg"] = "test"
	entry.LineNum = 42

	if err := em.Emit(entry); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	lineNum, ok := decoded["_lineNumber"]
	if !ok {
		t.Fatal("expected _lineNumber key in output")
	}

	// json.Unmarshal decodes numbers as float64 by default
	if lineNum != float64(42) {
		t.Errorf("expected _lineNumber=42, got %v", lineNum)
	}
}

func TestEmitter_Emit_AddTimestamp(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, Options{AddTimestamp: true})

	entry := parser.NewEntry("some line")
	entry.Fields["msg"] = "test"

	if err := em.Emit(entry); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	ingestTime, ok := decoded["_ingestTime"]
	if !ok {
		t.Fatal("expected _ingestTime key in output")
	}

	ts, ok := ingestTime.(string)
	if !ok {
		t.Fatalf("expected _ingestTime to be a string, got %T", ingestTime)
	}

	// RFC3339Nano contains "T" as date-time separator
	if !strings.Contains(ts, "T") {
		t.Errorf("timestamp %q does not look like RFC3339Nano (missing 'T')", ts)
	}

	// RFC3339Nano ends with timezone: "Z" for UTC or "+"/"-" offset
	if !strings.Contains(ts, "Z") && !strings.Contains(ts, "+") && !strings.Contains(ts, "-") {
		t.Errorf("timestamp %q does not look like RFC3339Nano (missing timezone indicator)", ts)
	}
}

func TestEmitter_Emit_AddRaw(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, Options{AddRaw: true})

	entry := parser.NewEntry("original line")
	entry.Fields["msg"] = "test"

	if err := em.Emit(entry); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	raw, ok := decoded["_raw"]
	if !ok {
		t.Fatal("expected _raw key in output")
	}
	if raw != "original line" {
		t.Errorf("expected _raw=%q, got %q", "original line", raw)
	}
}

func TestEmitter_Emit_OmitEmpty(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, Options{OmitEmpty: true})

	// Entry with ParseError should be skipped
	errEntry := parser.NewEntry("bad line")
	errEntry.ParseError = errors.New("parse failed")

	if err := em.Emit(errEntry); err != nil {
		t.Fatalf("Emit returned error for skipped entry: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for entry with ParseError when OmitEmpty=true, got %q", buf.String())
	}

	// Normal entry should be emitted
	goodEntry := parser.NewEntry("good line")
	goodEntry.Fields["msg"] = "ok"

	if err := em.Emit(goodEntry); err != nil {
		t.Fatalf("Emit returned error for good entry: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected output for normal entry, got nothing")
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if decoded["msg"] != "ok" {
		t.Errorf("expected msg=ok, got %v", decoded["msg"])
	}
}

func TestEmitter_Emit_ParseError(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, Options{})

	entry := parser.NewEntry("bad line")
	entry.ParseError = errors.New("something went wrong")

	if err := em.Emit(entry); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	parseErr, ok := decoded["_parseError"]
	if !ok {
		t.Fatal("expected _parseError key in output")
	}
	if parseErr != "something went wrong" {
		t.Errorf("expected _parseError=%q, got %q", "something went wrong", parseErr)
	}
}

func TestEmitter_Emit_MultipleEntries(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, Options{})

	for i := 0; i < 3; i++ {
		entry := parser.NewEntry("line")
		entry.Fields["index"] = i
		if err := em.Emit(entry); err != nil {
			t.Fatalf("Emit entry %d returned error: %v", i, err)
		}
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 newline-delimited JSON lines, got %d: %q", len(lines), output)
	}

	for i, line := range lines {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Errorf("line %d is not valid JSON: %v\nline: %s", i, err, line)
		}
	}
}

func TestEmitter_Emit_HTMLEscaping(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, Options{})

	entry := parser.NewEntry("<script>alert()</script>")
	entry.Fields["payload"] = "<script>alert()</script>"

	if err := em.Emit(entry); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	output := buf.String()

	// SetEscapeHTML(false) means literal < and > should appear, not \u003c and \u003e
	if strings.Contains(output, `\u003c`) || strings.Contains(output, `\u003e`) {
		t.Errorf("output should not HTML-escape angle brackets, got: %s", output)
	}
	if !strings.Contains(output, "<script>") {
		t.Errorf("output should contain literal <script>, got: %s", output)
	}
}

func TestEmitter_Close(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, Options{})

	entry := parser.NewEntry("test line")
	entry.Fields["msg"] = "flush check"

	// Emit already flushes, but Close should also flush without error
	if err := em.Emit(entry); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	if err := em.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	// Verify the buffer has the expected content after Close
	if buf.Len() == 0 {
		t.Error("expected output after Close, got nothing")
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output after Close is not valid JSON: %v", err)
	}
	if decoded["msg"] != "flush check" {
		t.Errorf("expected msg=%q, got %v", "flush check", decoded["msg"])
	}
}
