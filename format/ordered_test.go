package format

import (
	"bytes"
	"testing"
)

func TestEncodeOrdered_Simple(t *testing.T) {
	lines := []string{"hello", "world", "foo"}
	offsets := map[string]int{
		"hello": 0,
		"world": 5,
		"foo":   10,
	}
	supLen := 13 // "helloworld" + "foo"

	raw, _ := EncodeOrdered(lines, offsets, supLen)

	tuples, err := DecodeOrderedWithContext(raw, supLen)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(tuples) != 3 {
		t.Fatalf("got %d tuples, want 3", len(tuples))
	}

	expected := [][2]int{{0, 5}, {5, 5}, {10, 3}}
	for i, e := range expected {
		if tuples[i] != e {
			t.Errorf("tuple %d = %v, want %v", i, tuples[i], e)
		}
	}
}

func TestEncodeOrdered_EmptyLines(t *testing.T) {
	lines := []string{"hello", "", "world"}
	offsets := map[string]int{
		"hello": 0,
		"world": 5,
	}
	supLen := 10

	raw, _ := EncodeOrdered(lines, offsets, supLen)

	tuples, err := DecodeOrderedWithContext(raw, supLen)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(tuples) != 3 {
		t.Fatalf("got %d tuples, want 3", len(tuples))
	}

	// Empty line should have length=0, offset=0.
	if tuples[1][1] != 0 {
		t.Errorf("empty line length = %d, want 0", tuples[1][1])
	}
}

func TestEncodeOrdered_SingleLine(t *testing.T) {
	lines := []string{"test"}
	offsets := map[string]int{"test": 0}
	supLen := 4

	raw, _ := EncodeOrdered(lines, offsets, supLen)

	tuples, err := DecodeOrderedWithContext(raw, supLen)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(tuples) != 1 {
		t.Fatalf("got %d tuples, want 1", len(tuples))
	}
	if tuples[0] != [2]int{0, 4} {
		t.Errorf("tuple = %v, want [0, 4]", tuples[0])
	}
}

func TestBitsNeeded(t *testing.T) {
	tests := []struct {
		n    uint64
		want uint
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 2},
		{4, 3},
		{7, 3},
		{8, 4},
		{127, 7},
		{128, 8},
		{255, 8},
		{256, 9},
	}
	for _, tt := range tests {
		got := bitsNeeded(tt.n)
		if got != tt.want {
			t.Errorf("bitsNeeded(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

// Silence the unused import warning.
var _ = bytes.NewReader
