package io

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadLines(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantLines []string
		wantErr   bool
	}{
		{
			name:      "normal lines",
			content:   "hello\nworld\nfoo\n",
			wantLines: []string{"hello", "world", "foo"},
		},
		{
			name:      "empty lines stripped",
			content:   "hello\n\n\nworld\n  \nfoo\n",
			wantLines: []string{"hello", "world", "foo"},
		},
		{
			name:      "whitespace trimmed",
			content:   "  hello  \n  world  \n",
			wantLines: []string{"hello", "world"},
		},
		{
			name:      "completely empty file",
			content:   "",
			wantLines: nil,
		},
		{
			name:      "only whitespace and newlines",
			content:   "\n  \n\t\n",
			wantLines: nil,
		},
		{
			name:      "single line no trailing newline",
			content:   "only",
			wantLines: []string{"only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "input.txt")

			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("setup: %v", err)
			}

			got, err := ReadLines(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != len(tt.wantLines) {
				t.Fatalf("got %d lines, want %d", len(got), len(tt.wantLines))
			}
			for i := range got {
				if got[i] != tt.wantLines[i] {
					t.Errorf("line %d = %q, want %q", i, got[i], tt.wantLines[i])
				}
			}
		})
	}
}

func TestReadLines_MissingFile(t *testing.T) {
	_, err := ReadLines("/nonexistent/path/to/file.txt")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestWriteResult(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.txt")

	content := "hello world superstring"
	if err := WriteResult(path, content); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if string(got) != content {
		t.Errorf("got %q, want %q", string(got), content)
	}
}

func TestWriteResult_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roundtrip.txt")

	lines := "alpha\nbeta\ngamma"
	if err := WriteResult(path, lines); err != nil {
		t.Fatalf("write error: %v", err)
	}

	got, err := ReadLines(path)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	want := []string{"alpha", "beta", "gamma"}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}
