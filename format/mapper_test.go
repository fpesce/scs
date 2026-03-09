package format

import "testing"

func TestMapOffsets_Simple(t *testing.T) {
	master := "abcdefghijkl"
	words := []string{"abc", "def", "ghi", "jkl"}

	offsets := MapOffsets(master, words)

	expected := map[string]int{
		"abc": 0,
		"def": 3,
		"ghi": 6,
		"jkl": 9,
	}

	for word, want := range expected {
		got, ok := offsets[word]
		if !ok {
			t.Errorf("word %q not found in offset map", word)
			continue
		}
		if got != want {
			t.Errorf("offset[%q] = %d, want %d", word, got, want)
		}
	}
}

func TestMapOffsets_Overlapping(t *testing.T) {
	// Superstring built from overlapping words.
	master := "passwordswordfishbone"
	words := []string{"password", "swordfish", "fishbone"}

	offsets := MapOffsets(master, words)

	// Verify each word is found at its correct position.
	for _, w := range words {
		off, ok := offsets[w]
		if !ok {
			t.Errorf("word %q not found in offset map", w)
			continue
		}
		// Verify the offset actually points to the word.
		if master[off:off+len(w)] != w {
			t.Errorf("offset[%q] = %d, but master[%d:%d] = %q",
				w, off, off, off+len(w), master[off:off+len(w)])
		}
	}
}

func TestMapOffsets_Empty(t *testing.T) {
	offsets := MapOffsets("test", []string{})
	if len(offsets) != 0 {
		t.Errorf("expected empty map, got %d entries", len(offsets))
	}
}

func TestMapOffsets_SingleCharWords(t *testing.T) {
	master := "abc"
	words := []string{"a", "b", "c"}

	offsets := MapOffsets(master, words)
	if len(offsets) != 3 {
		t.Fatalf("expected 3 offsets, got %d", len(offsets))
	}

	for _, w := range words {
		off := offsets[w]
		if master[off:off+1] != w {
			t.Errorf("offset[%q] = %d, but master[%d] = %q", w, off, off, string(master[off]))
		}
	}
}
