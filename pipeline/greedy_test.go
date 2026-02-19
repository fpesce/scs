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
