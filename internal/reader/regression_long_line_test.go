package reader

import (
	"bufio"
	"strings"
	"testing"
)

func TestRegression_OversizedLineDoesNotEndTheStream(t *testing.T) {
	const limit = 1024
	input := "first\n" + strings.Repeat("x", limit*5) + "\nthird\nfourth\r\n" + strings.Repeat("y", limit+1) + "\nlast"

	check := func(t *testing.T, lines []Line) {
		t.Helper()
		want := []struct {
			text    string
			tooLong bool
		}{
			{"first", false},
			{"", true},
			{"third", false},
			{"fourth", false},
			{"", true},
			{"last", false},
		}
		if len(lines) != len(want) {
			t.Fatalf("got %d lines, want %d: %+v", len(lines), len(want), lines)
		}
		for i, w := range want {
			if lines[i].Number != i+1 {
				t.Errorf("line %d has Number %d", i+1, lines[i].Number)
			}
			if lines[i].Text != w.text {
				t.Errorf("line %d text has length %d, want %q", i+1, len(lines[i].Text), w.text)
			}
			if w.tooLong != (lines[i].Err == bufio.ErrTooLong) {
				t.Errorf("line %d Err = %v, want tooLong=%v", i+1, lines[i].Err, w.tooLong)
			}
		}
	}

	t.Run("Lines", func(t *testing.T) {
		var got []Line
		for line := range New(strings.NewReader(input), WithMaxLineSize(limit)).Lines() {
			got = append(got, line)
		}
		check(t, got)
	})

	t.Run("ReadAll", func(t *testing.T) {
		got, err := New(strings.NewReader(input), WithMaxLineSize(limit)).ReadAll()
		if err != bufio.ErrTooLong {
			t.Errorf("ReadAll error = %v, want the first line error", err)
		}
		check(t, got)
	})
}

func TestRegression_LineAtTheLimitIsKept(t *testing.T) {
	const limit = 1024
	exact := strings.Repeat("z", limit)
	got, err := New(strings.NewReader(exact+"\nnext\n"), WithMaxLineSize(limit)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Text != exact || got[1].Text != "next" {
		t.Fatalf("a line of exactly the limit must be kept, got %d lines", len(got))
	}
}

func TestRegression_MatchesScannerForOrdinaryInput(t *testing.T) {
	alphabet := []string{"a", "bc", "\n", "\n", "\r\n", "\r", " ", "é"}
	state := uint32(1)
	next := func(n int) int {
		state = state*1664525 + 1013904223
		return int(state>>16) % n
	}

	for round := 0; round < 2000; round++ {
		var b strings.Builder
		for i := next(12); i > 0; i-- {
			b.WriteString(alphabet[next(len(alphabet))])
		}
		input := b.String()

		var want []string
		scanner := bufio.NewScanner(strings.NewReader(input))
		for scanner.Scan() {
			want = append(want, scanner.Text())
		}

		got, err := New(strings.NewReader(input)).ReadAll()
		if err != nil {
			t.Fatalf("input %q: %v", input, err)
		}
		if len(got) != len(want) {
			t.Fatalf("input %q: got %d lines, bufio.Scanner yields %d", input, len(got), len(want))
		}
		for i := range want {
			if got[i].Text != want[i] || got[i].Number != i+1 {
				t.Fatalf("input %q line %d: got %q (number %d), bufio.Scanner yields %q", input, i+1, got[i].Text, got[i].Number, want[i])
			}
		}
	}
}
