package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/juliosaraiva/log2json/internal/reader"
)

func TestRegression_RecordsAfterAnOversizedLineAreStillEmitted(t *testing.T) {
	input := "level=info msg=first n=1\n" +
		strings.Repeat("x", reader.DefaultMaxLineSize+100) + "\n" +
		"level=info msg=third n=3\n" +
		"level=info msg=fourth n=4\n"

	var stdout, stderr bytes.Buffer
	if err := runPipeline(Config{}, strings.NewReader(input), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}

	records := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(records) != 3 {
		t.Fatalf("got %d records, want 3 (lines 1, 3, 4):\n%s", len(records), stdout.String())
	}
	if !strings.Contains(records[2], `"msg":"fourth"`) {
		t.Errorf("last record is %s", records[2])
	}
	if got := strings.Count(stderr.String(), "read error at line 2"); got != 1 {
		t.Errorf("want exactly one read error for line 2, stderr: %s", stderr.String())
	}
}
