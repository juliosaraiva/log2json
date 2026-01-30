package reader

import (
	"bufio"
	"fmt"
	"strings"
	"testing"
)

func TestStreamReader_ReadAll(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantTexts   []string
		wantNumbers []int
	}{
		{
			name:        "multiple lines",
			input:       "line1\nline2\nline3",
			wantTexts:   []string{"line1", "line2", "line3"},
			wantNumbers: []int{1, 2, 3},
		},
		{
			name:        "empty input",
			input:       "",
			wantTexts:   nil,
			wantNumbers: nil,
		},
		{
			name:        "single line without newline",
			input:       "hello",
			wantTexts:   []string{"hello"},
			wantNumbers: []int{1},
		},
		{
			name:        "blank line in the middle",
			input:       "a\n\nb",
			wantTexts:   []string{"a", "", "b"},
			wantNumbers: []int{1, 2, 3},
		},
		{
			name:        "trailing newline",
			input:       "a\nb\n",
			wantTexts:   []string{"a", "b"},
			wantNumbers: []int{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(strings.NewReader(tt.input))
			lines, err := r.ReadAll()
			if err != nil {
				t.Fatalf("ReadAll() unexpected error: %v", err)
			}

			if len(lines) != len(tt.wantTexts) {
				t.Fatalf("ReadAll() returned %d lines, want %d", len(lines), len(tt.wantTexts))
			}

			for i, line := range lines {
				if line.Text != tt.wantTexts[i] {
					t.Errorf("line %d text = %q, want %q", i, line.Text, tt.wantTexts[i])
				}
				if line.Number != tt.wantNumbers[i] {
					t.Errorf("line %d number = %d, want %d", i, line.Number, tt.wantNumbers[i])
				}
				if line.Err != nil {
					t.Errorf("line %d unexpected error: %v", i, line.Err)
				}
			}
		})
	}
}

func TestStreamReader_Lines(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantTexts   []string
		wantNumbers []int
	}{
		{
			name:        "multiple lines",
			input:       "line1\nline2\nline3",
			wantTexts:   []string{"line1", "line2", "line3"},
			wantNumbers: []int{1, 2, 3},
		},
		{
			name:        "empty input",
			input:       "",
			wantTexts:   nil,
			wantNumbers: nil,
		},
		{
			name:        "single line without newline",
			input:       "hello",
			wantTexts:   []string{"hello"},
			wantNumbers: []int{1},
		},
		{
			name:        "blank line in the middle",
			input:       "a\n\nb",
			wantTexts:   []string{"a", "", "b"},
			wantNumbers: []int{1, 2, 3},
		},
		{
			name:        "trailing newline",
			input:       "a\nb\n",
			wantTexts:   []string{"a", "b"},
			wantNumbers: []int{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(strings.NewReader(tt.input))
			ch := r.Lines()

			var lines []Line
			for line := range ch {
				lines = append(lines, line)
			}

			if len(lines) != len(tt.wantTexts) {
				t.Fatalf("Lines() returned %d lines, want %d", len(lines), len(tt.wantTexts))
			}

			for i, line := range lines {
				if line.Text != tt.wantTexts[i] {
					t.Errorf("line %d text = %q, want %q", i, line.Text, tt.wantTexts[i])
				}
				if line.Number != tt.wantNumbers[i] {
					t.Errorf("line %d number = %d, want %d", i, line.Number, tt.wantNumbers[i])
				}
				if line.Err != nil {
					t.Errorf("line %d unexpected error: %v", i, line.Err)
				}
			}
		})
	}
}

func TestStreamReader_LargeInput(t *testing.T) {
	const totalLines = 10000

	var b strings.Builder
	for i := 1; i <= totalLines; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}

	r := New(strings.NewReader(b.String()))
	lines, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() unexpected error: %v", err)
	}

	if len(lines) != totalLines {
		t.Fatalf("ReadAll() returned %d lines, want %d", len(lines), totalLines)
	}

	for i, line := range lines {
		wantNumber := i + 1
		wantText := fmt.Sprintf("line %d", wantNumber)
		if line.Number != wantNumber {
			t.Errorf("line %d number = %d, want %d", i, line.Number, wantNumber)
		}
		if line.Text != wantText {
			t.Errorf("line %d text = %q, want %q", i, line.Text, wantText)
		}
	}
}

func TestStreamReader_WithMaxLineSize(t *testing.T) {
	// bufio.Scanner only checks maxTokenSize when the buffer needs to
	// grow beyond its current capacity. New() allocates an initial buffer
	// of DefaultBufferSize (64KB), so a line must exceed that size to
	// trigger a growth attempt and the subsequent ErrTooLong check.
	oversizedLen := DefaultBufferSize + 1

	t.Run("ReadAll returns error for oversized line", func(t *testing.T) {
		longLine := strings.Repeat("x", oversizedLen)
		r := New(strings.NewReader(longLine), WithMaxLineSize(DefaultBufferSize))

		_, err := r.ReadAll()
		if err == nil {
			t.Fatal("ReadAll() expected error for oversized line, got nil")
		}
		if err != bufio.ErrTooLong {
			t.Errorf("ReadAll() error = %v, want %v", err, bufio.ErrTooLong)
		}
	})

	t.Run("Lines sends error for oversized line", func(t *testing.T) {
		longLine := strings.Repeat("x", oversizedLen)
		r := New(strings.NewReader(longLine), WithMaxLineSize(DefaultBufferSize))

		var gotErr error
		for line := range r.Lines() {
			if line.Err != nil {
				gotErr = line.Err
			}
		}

		if gotErr == nil {
			t.Fatal("Lines() expected error for oversized line, got nil")
		}
		if gotErr != bufio.ErrTooLong {
			t.Errorf("Lines() error = %v, want %v", gotErr, bufio.ErrTooLong)
		}
	})

	t.Run("line within limit succeeds", func(t *testing.T) {
		shortLine := strings.Repeat("y", DefaultBufferSize-1)
		r := New(strings.NewReader(shortLine), WithMaxLineSize(DefaultBufferSize))

		lines, err := r.ReadAll()
		if err != nil {
			t.Fatalf("ReadAll() unexpected error: %v", err)
		}
		if len(lines) != 1 {
			t.Fatalf("ReadAll() returned %d lines, want 1", len(lines))
		}
		if lines[0].Text != shortLine {
			t.Errorf("line text length = %d, want %d", len(lines[0].Text), len(shortLine))
		}
	})
}
