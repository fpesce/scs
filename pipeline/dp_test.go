package pipeline

import (
	"strings"
	"testing"
)

func TestSolveExactDP(t *testing.T) {
	tests := []struct {
		name       string
		island     []string
		minOverlap int
		// We check the result contains all strings and is as short as possible.
		maxLen int // upper bound on result length
	}{
		{
			name:       "two overlapping strings",
			island:     []string{"abcde", "cdefg"},
			minOverlap: 3,
			maxLen:     7, // "abcdefg"
		},
		{
			name:       "three chained overlaps",
			island:     []string{"abcdef", "defghi", "ghijkl"},
			minOverlap: 3,
			maxLen:     12, // "abcdefghijkl"
		},
		{
			name:       "single string",
			island:     []string{"hello"},
			minOverlap: 3,
			maxLen:     5,
		},
		{
			name:       "no overlap between strings",
			island:     []string{"abc", "xyz"},
			minOverlap: 3,
			maxLen:     6, // "abcxyz" or "xyzabc"
		},
		{
			name:       "identical overlap",
			island:     []string{"abc", "abc"},
			minOverlap: 3,
			maxLen:     3, // "abc"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SolveExactDP(tt.island, tt.minOverlap)

			// Verify all original strings are substrings of the result.
			for _, s := range tt.island {
				if !strings.Contains(result, s) {
					t.Errorf("result %q does not contain %q", result, s)
				}
			}

			// Verify the result is within the expected length bound.
			if len(result) > tt.maxLen {
				t.Errorf("result %q has length %d, exceeds max %d", result, len(result), tt.maxLen)
			}
		})
	}
}

func TestSolveExactDP_Empty(t *testing.T) {
	result := SolveExactDP([]string{}, 3)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestSolveExactDP_Optimality(t *testing.T) {
	// Known optimal: "abcdefghijkl" (length 12)
	island := []string{"ghijkl", "abcdef", "defghi"}
	result := SolveExactDP(island, 3)

	if len(result) != 12 {
		t.Errorf("expected optimal length 12, got %d (%q)", len(result), result)
	}
	if result != "abcdefghijkl" {
		t.Errorf("expected 'abcdefghijkl', got %q", result)
	}
}
