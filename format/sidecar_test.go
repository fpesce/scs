package format

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteRankSidecar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.rank")

	orderedWords := []string{"hello", "world", "abc"}
	rankMap := map[string]uint32{
		"hello": 42,
		"world": 7,
		"abc":   100,
	}

	if err := WriteRankSidecar(path, orderedWords, rankMap); err != nil {
		t.Fatalf("WriteRankSidecar: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading rank file: %v", err)
	}

	// Expect 3 * 4 = 12 bytes.
	if len(data) != 12 {
		t.Fatalf("rank file length = %d, want 12", len(data))
	}

	// Verify each LE uint32.
	wantRanks := []uint32{42, 7, 100}
	for i, want := range wantRanks {
		got := binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
		if got != want {
			t.Errorf("rank[%d] = %d, want %d", i, got, want)
		}
	}
}

func TestWriteRankSidecar_MissingRank(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.rank")

	orderedWords := []string{"hello", "missing"}
	rankMap := map[string]uint32{"hello": 1}

	err := WriteRankSidecar(path, orderedWords, rankMap)
	if err == nil {
		t.Fatal("expected error for missing rank, got nil")
	}
}

func TestWriteMetadataSidecar(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "in.json")
	outPath := filepath.Join(dir, "out.json")

	content := []byte(`{"pat_str": "test", "special_tokens": {}}`)
	if err := os.WriteFile(inPath, content, 0o644); err != nil {
		t.Fatalf("writing input: %v", err)
	}

	if err := WriteMetadataSidecar(outPath, inPath); err != nil {
		t.Fatalf("WriteMetadataSidecar: %v", err)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	if string(got) != string(content) {
		t.Errorf("metadata mismatch:\ngot:  %q\nwant: %q", string(got), string(content))
	}
}
