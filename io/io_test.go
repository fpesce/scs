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
			name:      "empty lines preserved",
			content:   "hello\n\n\nworld\n",
			wantLines: []string{"hello", "", "", "world"},
		},
		{
			name:      "whitespace preserved",
			content:   "  hello  \n  world  \n",
			wantLines: []string{"  hello  ", "  world  "},
		},
		{
			name:      "completely empty file",
			content:   "",
			wantLines: nil,
		},
		{
			name:      "only newlines",
			content:   "\n\n\n",
			wantLines: []string{"", "", ""},
		},
		{
			name:      "tabs preserved",
			content:   "\thello\t\n\tworld\t\n",
			wantLines: []string{"\thello\t", "\tworld\t"},
		},
		{
			name:      "single line no trailing newline",
			content:   "only",
			wantLines: []string{"only"},
		},
		{
			name:      "whitespace-only lines preserved",
			content:   "  \n\t\n",
			wantLines: []string{"  ", "\t"},
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

func TestReadSeparated_BinarySafe(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "binary.dat")

	sep := byte(0x1F) // Unit Separator
	// Build payload: "hello\nworld" SEP "\x00data" SEP "end" SEP
	// The embedded \n and \0 must survive.
	payload := []byte{'h', 'e', 'l', 'l', 'o', '\n', 'w', 'o', 'r', 'l', 'd'}
	payload = append(payload, sep)
	payload = append(payload, '\x00', 'd', 'a', 't', 'a')
	payload = append(payload, sep)
	payload = append(payload, 'e', 'n', 'd')
	payload = append(payload, sep)

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got, err := ReadSeparated(path, []byte{sep})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"hello\nworld", "\x00data", "end"}
	if len(got) != len(want) {
		t.Fatalf("got %d elements, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("element %d = %q, want %q", i, got[i], want[i])
		}
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
