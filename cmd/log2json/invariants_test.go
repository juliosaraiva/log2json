package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"
)

// The invariants in this file hold for every input, not for chosen examples.
// A bug found in production or by fuzzing becomes a new invariant here, or a
// tighter generator, in the same pull request as its fix.

type pipelineResult struct {
	records    []map[string]any
	stdout     string
	errorLines int
}

func runForInvariants(t testing.TB, cfg Config, input string) pipelineResult {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if err := runPipeline(cfg, strings.NewReader(input), &stdout, &stderr); err != nil {
		t.Fatalf("runPipeline returned an error for valid configuration: %v", err)
	}

	result := pipelineResult{stdout: stdout.String()}
	for _, line := range strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("stdout line is not a JSON object: %v\nline: %q\ninput: %q", err, line, input)
		}
		result.records = append(result.records, record)
	}
	for _, line := range strings.Split(stderr.String(), "\n") {
		if strings.Contains(line, " error at line ") {
			result.errorLines++
		}
	}
	return result
}

func inputLineCount(input string) int {
	if input == "" {
		return 0
	}
	return len(strings.Split(strings.TrimSuffix(input, "\n"), "\n"))
}

type lineGenerator struct {
	rng *rand.Rand
}

func (g lineGenerator) word() string {
	words := []string{"alpha", "beta", "gamma", "user", "login", "timeout", "GET", "error", "ok", "db-1", "x_y", "Zürich", "日本"}
	return words[g.rng.Intn(len(words))]
}

func (g lineGenerator) garbage() string {
	const alphabet = `abcXYZ 019 =":'[]{}<>-_/\.,;!?%&|+*#@~` + "\t" + "é漢"
	runes := []rune(alphabet)
	n := g.rng.Intn(60)
	out := make([]rune, n)
	for i := range out {
		out[i] = runes[g.rng.Intn(len(runes))]
	}
	return string(out)
}

func (g lineGenerator) line() string {
	switch g.rng.Intn(9) {
	case 0:
		return fmt.Sprintf(`{"level":"%s","msg":"%s %s","n":%d}`, g.word(), g.word(), g.word(), g.rng.Intn(1000))
	case 1:
		return fmt.Sprintf(`level=%s msg="%s %s" count=%d ratio=%d.%d`, g.word(), g.word(), g.word(), g.rng.Intn(1000), g.rng.Intn(10), g.rng.Intn(10))
	case 2:
		return fmt.Sprintf(`Jan %2d 10:%02d:%02d host-%d %s[%d]: %s %s`, 1+g.rng.Intn(28), g.rng.Intn(60), g.rng.Intn(60), g.rng.Intn(9), g.word(), g.rng.Intn(65536), g.word(), g.word())
	case 3:
		return fmt.Sprintf(`10.0.%d.%d - - [15/Jan/2024:10:30:45 +0000] "GET /%s HTTP/1.1" %d %d "-" "%s"`, g.rng.Intn(256), g.rng.Intn(256), g.word(), 200+g.rng.Intn(300), g.rng.Intn(100000), g.word())
	case 4:
		return fmt.Sprintf(`2024-01-15 10:30:%02d ERROR %s %s`, g.rng.Intn(60), g.word(), g.word())
	case 5:
		return ""
	case 6:
		return "   "
	default:
		return g.garbage()
	}
}

func (g lineGenerator) input() string {
	n := g.rng.Intn(12)
	lines := make([]string, n)
	for i := range lines {
		lines[i] = g.line()
	}
	if n == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func checkInvariants(t testing.TB, input string) {
	t.Helper()
	want := inputLineCount(input)

	for _, mode := range []struct {
		name string
		cfg  Config
	}{
		{"auto", Config{}},
		{"adaptive", Config{Adaptive: true}},
		{"kv", Config{Format: "kv"}},
		{"json", Config{Format: "json"}},
		{"syslog", Config{Format: "syslog"}},
		{"apache", Config{Format: "apache"}},
		{"generic", Config{Format: "generic"}},
		{"regex", Config{Pattern: `(?P<first>\S+)\s*(?P<rest>.*)`}},
	} {
		base := runForInvariants(t, mode.cfg, input)

		if got := len(base.records) + base.errorLines; got != want {
			t.Fatalf("[%s] record conservation: %d input lines became %d records + %d reported errors\ninput: %q\nstdout: %q", mode.name, want, len(base.records), base.errorLines, input, base.stdout)
		}

		again := runForInvariants(t, mode.cfg, input)
		if again.stdout != base.stdout {
			t.Fatalf("[%s] determinism: two runs over the same input differ\ninput: %q\nfirst: %q\nsecond: %q", mode.name, input, base.stdout, again.stdout)
		}

		meta := mode.cfg
		meta.AddLineNumber = true
		meta.AddRaw = true
		withMeta := runForInvariants(t, meta, input)
		inputLines := strings.Split(strings.TrimSuffix(input, "\n"), "\n")
		previous := 0.0
		for _, record := range withMeta.records {
			number, ok := record["_lineNumber"].(float64)
			if !ok || number <= previous || int(number) > want {
				t.Fatalf("[%s] line numbers: got %v after %v with %d input lines\ninput: %q", mode.name, record["_lineNumber"], previous, want, input)
			}
			previous = number
			if raw, _ := record["_raw"].(string); raw != inputLines[int(number)-1] {
				t.Fatalf("[%s] raw fidelity: line %d is %q but _raw is %q", mode.name, int(number), inputLines[int(number)-1], raw)
			}
		}

		filtered := mode.cfg
		filtered.Fields = []string{"level"}
		for _, record := range runForInvariants(t, filtered, input).records {
			for key := range record {
				if key != "level" && key != "_parseError" {
					t.Fatalf("[%s] field filter: --fields=level leaked key %q\ninput: %q", mode.name, key, input)
				}
			}
		}
	}
}

func TestInvariants_GeneratedInputs(t *testing.T) {
	const seeds = 120
	for seed := int64(1); seed <= seeds; seed++ {
		generator := lineGenerator{rng: rand.New(rand.NewSource(seed))} //nolint:gosec // reproducible test inputs, not a security decision
		checkInvariants(t, generator.input())
	}
}

func fuzzable(input string) bool {
	if len(input) > 2*1024 || !utf8.ValidString(input) {
		return false
	}
	return !strings.ContainsAny(input, "\r\x00")
}

func FuzzPipelineInvariants(f *testing.F) {
	for _, seed := range []string{
		"",
		"\n",
		`{"a":1}` + "\n",
		"level=info msg=\"hi\" n=1\nlevel=warn msg='x y' n=2\n",
		"Jan 15 10:30:45 myhost sshd[1234]: Accepted password\n",
		`192.168.1.1 - bob [15/Jan/2024:10:30:45 +0000] "GET / HTTP/1.1" 200 12` + "\n",
		"2024-01-15 10:30:45 INFO started\nplain text\n\n{broken\n",
		"a=1 b=2\n{\"a\":1}\nWARN: mixed\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if !fuzzable(input) {
			t.Skip()
		}
		checkInvariants(t, input)
	})
}
