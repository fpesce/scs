package pipeline

import (
	"strings"
	"testing"
)

func TestMergeEliminateSubstrings_Basic(t *testing.T) {
	primary := "passwordswordfishbone"
	updateWords := []string{"password", "newword", "fish"}

	survivors, eliminated := MergeEliminateSubstrings(primary, updateWords)

	// "password" and "fish" exist in the primary payload.
	if _, ok := eliminated["password"]; !ok {
		t.Error("expected 'password' to be eliminated")
	}
	if _, ok := eliminated["fish"]; !ok {
		t.Error("expected 'fish' to be eliminated")
	}

	// "newword" should survive.
	if len(survivors) != 1 || survivors[0] != "newword" {
		t.Errorf("survivors = %v, want [newword]", survivors)
	}
}

func TestMergeEliminateSubstrings_AllFound(t *testing.T) {
	primary := "abcdefgh"
	updateWords := []string{"abc", "def", "gh"}

	survivors, eliminated := MergeEliminateSubstrings(primary, updateWords)

	if len(survivors) != 0 {
		t.Errorf("expected no survivors, got %v", survivors)
	}
	if len(eliminated) != 3 {
		t.Errorf("expected 3 eliminated, got %d", len(eliminated))
	}
}

func TestMergeEliminateSubstrings_NoneFound(t *testing.T) {
	primary := "abcdefgh"
	updateWords := []string{"xyz", "qrs"}

	survivors, eliminated := MergeEliminateSubstrings(primary, updateWords)

	if len(survivors) != 2 {
		t.Errorf("expected 2 survivors, got %d", len(survivors))
	}
	if len(eliminated) != 0 {
		t.Errorf("expected 0 eliminated, got %d", len(eliminated))
	}
}

func TestMergeEliminateSubstrings_Empty(t *testing.T) {
	survivors, eliminated := MergeEliminateSubstrings("payload", []string{})
	if len(survivors) != 0 {
		t.Errorf("expected no survivors, got %v", survivors)
	}
	if len(eliminated) != 0 {
		t.Errorf("expected no eliminated, got %d", len(eliminated))
	}
}

func TestTruncateAndAppend_Basic(t *testing.T) {
	primary := "abcdef"
	survivors := []string{"defghi"} // "def" overlaps with primary suffix.

	_, fragment, overlap := TruncateAndAppend(primary, survivors, 3, 15)

	// The mini-superstring should be "defghi", overlap with "abcdef" is "def" (3 chars).
	// Fragment should be "ghi".
	if overlap != 3 {
		t.Errorf("overlap = %d, want 3", overlap)
	}
	if fragment != "ghi" {
		t.Errorf("fragment = %q, want %q", fragment, "ghi")
	}

	// Verify combined payload contains the survivor.
	combined := primary + fragment
	if !strings.Contains(combined, "defghi") {
		t.Errorf("combined %q does not contain 'defghi'", combined)
	}
}

func TestTruncateAndAppend_NoOverlap(t *testing.T) {
	primary := "abcdef"
	survivors := []string{"xyz"}

	_, fragment, overlap := TruncateAndAppend(primary, survivors, 3, 15)

	if overlap != 0 {
		t.Errorf("overlap = %d, want 0", overlap)
	}
	if fragment != "xyz" {
		t.Errorf("fragment = %q, want %q", fragment, "xyz")
	}
}

func TestTruncateAndAppend_EmptySurvivors(t *testing.T) {
	_, fragment, overlap := TruncateAndAppend("primary", []string{}, 3, 15)

	if fragment != "" {
		t.Errorf("fragment = %q, want empty", fragment)
	}
	if overlap != 0 {
		t.Errorf("overlap = %d, want 0", overlap)
	}
}
