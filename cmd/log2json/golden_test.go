package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update-golden", false, "rewrite testdata/golden from current behavior")

// TestGoldenCorpus pins the exact output for every sample log in testdata.
// A behavior change shows up as a diff in testdata/golden that a reviewer can
// read without reading the code. Regenerate with:
//
//	go test ./cmd/log2json -run TestGoldenCorpus -update-golden
func TestGoldenCorpus(t *testing.T) {
	samples, err := filepath.Glob(filepath.Join("..", "..", "testdata", "sample_*.log"))
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) == 0 {
		t.Fatal("no testdata/sample_*.log files found")
	}

	modes := []struct {
		name string
		cfg  Config
	}{
		{"auto", Config{}},
		{"adaptive", Config{Adaptive: true}},
	}

	for _, sample := range samples {
		input, err := os.ReadFile(sample) //nolint:gosec // path comes from a fixed glob inside the repository
		if err != nil {
			t.Fatal(err)
		}
		normalized := strings.ReplaceAll(string(input), "\r\n", "\n")
		base := strings.TrimSuffix(filepath.Base(sample), ".log")

		for _, mode := range modes {
			name := base + "." + mode.name
			t.Run(name, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				if err := runPipeline(mode.cfg, strings.NewReader(normalized), &stdout, &stderr); err != nil {
					t.Fatal(err)
				}
				got := stdout.String()
				if stderr.Len() > 0 {
					got += "--- stderr ---\n" + stderr.String()
				}

				goldenPath := filepath.Join("..", "..", "testdata", "golden", name+".jsonl")
				if *updateGolden {
					if err := os.WriteFile(goldenPath, []byte(got), 0o600); err != nil {
						t.Fatal(err)
					}
					return
				}

				want, err := os.ReadFile(goldenPath) //nolint:gosec // path is built from a fixed glob inside the repository
				if err != nil {
					t.Fatalf("missing golden file; run with -update-golden and review the new file: %v", err)
				}
				if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
					t.Errorf("output for %s changed.\nIf the change is intended, run:\n  go test ./cmd/log2json -run TestGoldenCorpus -update-golden\nand include the testdata/golden diff in the pull request.\n\ngot:\n%s\nwant:\n%s", name, got, want)
				}
			})
		}
	}
}
