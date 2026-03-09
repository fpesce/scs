package pipeline

import (
	"strings"
	"testing"
)

func TestAssembleConcurrently(t *testing.T) {
	tests := []struct {
		name       string
		islands    [][]string
		dpLimit    int
		minOverlap int
	}{
		{
			name: "mixed small and large islands",
			islands: [][]string{
				{"abcdef", "defghi", "ghijkl"},
				{"xyz", "abc"},
			},
			dpLimit:    15,
			minOverlap: 3,
		},
		{
			name: "all single-string islands",
			islands: [][]string{
				{"hello"},
				{"world"},
				{"test"},
			},
			dpLimit:    15,
			minOverlap: 3,
		},
		{
			name:       "empty input",
			islands:    [][]string{},
			dpLimit:    15,
			minOverlap: 3,
		},
		{
			name: "single island",
			islands: [][]string{
				{"abcdef", "defghi"},
			},
			dpLimit:    15,
			minOverlap: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AssembleConcurrently(tt.islands, tt.dpLimit, tt.minOverlap, false)

			if tt.name == "empty input" {
				if result != nil {
					t.Fatalf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.islands) {
				t.Fatalf("got %d results, want %d", len(result), len(tt.islands))
			}

			// Verify each result contains all strings from the corresponding island.
			for i, island := range tt.islands {
				for _, s := range island {
					if !strings.Contains(result[i], s) {
						t.Errorf("result[%d] = %q does not contain %q", i, result[i], s)
					}
				}
			}
		})
	}
}

func TestAssembleConcurrently_Deterministic(t *testing.T) {
	// Run multiple times to verify deterministic ordering.
	islands := [][]string{
		{"a1b2c3", "c3d4e5"},
		{"x1y2", "y2z3"},
		{"hello"},
	}

	var prev []string
	for run := 0; run < 5; run++ {
		result := AssembleConcurrently(islands, 15, 2, false)
		if prev != nil {
			for i := range result {
				if result[i] != prev[i] {
					t.Fatalf("run %d: result[%d] changed from %q to %q", run, i, prev[i], result[i])
				}
			}
		}
		prev = result
	}
}
