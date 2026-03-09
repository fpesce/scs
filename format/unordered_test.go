package format

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestEncodeUnordered_Simple(t *testing.T) {
	words := []string{"ab", "cd", "xyz"}
	offsets := map[string]int{
		"ab":  0,
		"cd":  5,
		"xyz": 10,
	}

	result := EncodeUnordered(words, offsets)

	// Decode and verify the binary structure.
	raw, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		t.Fatalf("invalid Base64: %v", err)
	}

	r := bytes.NewReader(raw)

	// Group 1: length=2, count=2 (ab, cd)
	wordLen, _, err := DecodeULEB128(r)
	if err != nil {
		t.Fatalf("reading word length: %v", err)
	}
	if wordLen != 2 {
		t.Errorf("word length = %d, want 2", wordLen)
	}

	count, _, err := DecodeULEB128(r)
	if err != nil {
		t.Fatalf("reading count: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	// Offsets: 0, 5 -> deltas: 0, 5
	delta1, _, err := DecodeULEB128(r)
	if err != nil {
		t.Fatalf("reading delta1: %v", err)
	}
	if delta1 != 0 {
		t.Errorf("delta1 = %d, want 0", delta1)
	}

	delta2, _, err := DecodeULEB128(r)
	if err != nil {
		t.Fatalf("reading delta2: %v", err)
	}
	if delta2 != 5 {
		t.Errorf("delta2 = %d, want 5", delta2)
	}

	// Group 2: length=3, count=1 (xyz)
	wordLen2, _, err := DecodeULEB128(r)
	if err != nil {
		t.Fatalf("reading word length 2: %v", err)
	}
	if wordLen2 != 3 {
		t.Errorf("word length 2 = %d, want 3", wordLen2)
	}

	count2, _, err := DecodeULEB128(r)
	if err != nil {
		t.Fatalf("reading count 2: %v", err)
	}
	if count2 != 1 {
		t.Errorf("count 2 = %d, want 1", count2)
	}

	delta3, _, err := DecodeULEB128(r)
	if err != nil {
		t.Fatalf("reading delta3: %v", err)
	}
	if delta3 != 10 {
		t.Errorf("delta3 = %d, want 10", delta3)
	}
}

func TestEncodeUnordered_Deterministic(t *testing.T) {
	words := []string{"hello", "world", "foo", "bar", "baz"}
	offsets := map[string]int{
		"hello": 0,
		"world": 20,
		"foo":   5,
		"bar":   10,
		"baz":   15,
	}

	// Run 10 times, assert identical output.
	var results []string
	for i := 0; i < 10; i++ {
		results = append(results, EncodeUnordered(words, offsets))
	}
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Fatalf("non-deterministic output: run %d = %q, run 0 = %q", i, results[i], results[0])
		}
	}
}

func TestEncodeUnordered_SingleWord(t *testing.T) {
	words := []string{"test"}
	offsets := map[string]int{"test": 42}

	result := EncodeUnordered(words, offsets)
	raw, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		t.Fatalf("invalid Base64: %v", err)
	}

	r := bytes.NewReader(raw)

	wordLen, _, _ := DecodeULEB128(r)
	if wordLen != 4 {
		t.Errorf("word length = %d, want 4", wordLen)
	}

	count, _, _ := DecodeULEB128(r)
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	delta, _, _ := DecodeULEB128(r)
	if delta != 42 {
		t.Errorf("delta = %d, want 42", delta)
	}
}
