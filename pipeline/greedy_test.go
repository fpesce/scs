package pipeline

import (
	"strings"
	"testing"
)

func TestSolveGreedyHeap(t *testing.T) {
	tests := []struct {
		name       string
		island     []string
		minOverlap int
		maxLen     int
	}{
		{
			name:       "two overlapping strings",
			island:     []string{"abcde", "cdefg"},
			minOverlap: 3,
			maxLen:     7,
		},
		{
			name:       "three chained overlaps",
			island:     []string{"abcdef", "defghi", "ghijkl"},
			minOverlap: 3,
			maxLen:     12,
		},
		{
			name:       "single string",
			island:     []string{"hello"},
			minOverlap: 3,
			maxLen:     5,
		},
		{
			name:       "no overlap",
			island:     []string{"abc", "xyz"},
			minOverlap: 3,
			maxLen:     6,
		},
		{
			name:       "four strings with greedy merge",
			island:     []string{"abc", "bcd", "cde", "def"},
			minOverlap: 1,
			maxLen:     6, // "abcdef"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SolveGreedyHeap(tt.island, tt.minOverlap)

			// Verify all original strings are substrings of the result.
			for _, s := range tt.island {
				if !strings.Contains(result, s) {
					t.Errorf("result %q does not contain %q", result, s)
				}
			}

			// Verify within length bound.
			if len(result) > tt.maxLen {
				t.Errorf("result %q has length %d, exceeds max %d", result, len(result), tt.maxLen)
			}
		})
	}
}

func TestSolveGreedyHeap_Empty(t *testing.T) {
	result := SolveGreedyHeap([]string{}, 3)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestSolveGreedyPath(t *testing.T) {
	island := []string{"abcdef", "defghi", "ghijkl", "xyz"}
	path, overlaps := solveGreedyPath(island, 3)

	// Verify path is a valid permutation: all indices 0..N-1 exactly once.
	n := len(island)
	if len(path) != n {
		t.Fatalf("path length = %d, want %d", len(path), n)
	}

	seen := make(map[int]bool)
	for _, idx := range path {
		if idx < 0 || idx >= n {
			t.Fatalf("path contains out-of-range index %d", idx)
		}
		if seen[idx] {
			t.Fatalf("path contains duplicate index %d", idx)
		}
		seen[idx] = true
	}

	// Verify overlaps length = len(path) - 1.
	if len(overlaps) != n-1 {
		t.Fatalf("overlaps length = %d, want %d", len(overlaps), n-1)
	}

	// Verify non-negative overlaps.
	for i, ov := range overlaps {
		if ov < 0 {
			t.Errorf("overlaps[%d] = %d, want >= 0", i, ov)
		}
	}
}
