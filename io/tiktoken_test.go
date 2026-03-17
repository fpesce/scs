package io

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath(name string) string {
	//nolint:dogsled // runtime.Caller returns 4 values, only filename is needed
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "testdata", name)
}

func TestReadTiktoken_Synthetic(t *testing.T) {
	words, rankMap, err := ReadTiktoken(testdataPath("synthetic.tiktoken"))
	if err != nil {
		t.Fatalf("ReadTiktoken: %v", err)
	}

	// synthetic.tiktoken contains:
	//   Cg==       0   → "\n"
	//   AA==       1   → "\x00"
	//   aGVsbG8=   2   → "hello"
	//   bG93b3JsZA== 3 → "loworld"
	//   d29ybGQ=   4   → "world"
	//   YWJj       5   → "abc"
	wantWords := []string{"\n", "\x00", "hello", "loworld", "world", "abc"}
	wantRanks := map[string]uint32{
		"\n":      0,
		"\x00":    1,
		"hello":   2,
		"loworld": 3,
		"world":   4,
		"abc":     5,
	}

	if len(words) != len(wantWords) {
		t.Fatalf("got %d words, want %d", len(words), len(wantWords))
	}

	for i, got := range words {
		if got != wantWords[i] {
			t.Errorf("word[%d] = %q, want %q", i, got, wantWords[i])
		}
	}

	for word, wantRank := range wantRanks {
		gotRank, ok := rankMap[word]
		if !ok {
			t.Errorf("rankMap missing word %q", word)
			continue
		}
		if gotRank != wantRank {
			t.Errorf("rank[%q] = %d, want %d", word, gotRank, wantRank)
		}
	}
}

func TestReadTiktoken_MissingFile(t *testing.T) {
	_, _, err := ReadTiktoken("/nonexistent/file.tiktoken")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
