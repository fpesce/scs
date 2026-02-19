package pipeline

import (
	"sort"
	"testing"
)

func TestShatterGraph(t *testing.T) {
	tests := []struct {
		name       string
		survivors  []string
		minOverlap int
		wantGroups int
		// wantIslands maps expected island sizes to their count
		wantIslandSizes []int
	}{
		{
			name:            "overlapping strings form one island",
			survivors:       []string{"abcdef", "defghi", "ghijkl"},
			minOverlap:      3,
			wantGroups:      1,
			wantIslandSizes: []int{3},
		},
		{
			name:            "no overlaps form separate islands",
			survivors:       []string{"abc", "xyz", "mnop"},
			minOverlap:      3,
			wantGroups:      3,
			wantIslandSizes: []int{1, 1, 1},
		},
		{
			name:            "mixed overlapping and disjoint",
			survivors:       []string{"abcdef", "defghi", "xyz123", "123456"},
			minOverlap:      3,
			wantGroups:      2,
			wantIslandSizes: []int{2, 2},
		},
		{
			name:            "single string",
			survivors:       []string{"alone"},
			minOverlap:      3,
			wantGroups:      1,
			wantIslandSizes: []int{1},
		},
		{
			name:            "empty input",
			survivors:       []string{},
			minOverlap:      3,
			wantGroups:      0,
			wantIslandSizes: []int{},
		},
		{
			name:            "overlap below threshold stays separate",
			survivors:       []string{"abcd", "cdef"},
			minOverlap:      3,
			wantGroups:      2,
			wantIslandSizes: []int{1, 1},
		},
		{
			name:            "overlap exactly at threshold merges",
			survivors:       []string{"abcde", "cdefg"},
			minOverlap:      3,
			wantGroups:      1,
			wantIslandSizes: []int{2},
		},
		{
			name:            "multi-length overlap above threshold merges",
			survivors:       []string{"abcdefgh", "defghxyz"},
			minOverlap:      3,
			wantGroups:      1,
			wantIslandSizes: []int{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			islands := ShatterGraph(tt.survivors, tt.minOverlap)

			if len(islands) != tt.wantGroups {
				t.Fatalf("got %d groups, want %d. Islands: %v", len(islands), tt.wantGroups, islands)
			}

			// Verify island sizes match expected.
			var gotSizes []int
			for _, island := range islands {
				gotSizes = append(gotSizes, len(island))
			}
			sort.Ints(gotSizes)
			sort.Ints(tt.wantIslandSizes)

			if len(gotSizes) != len(tt.wantIslandSizes) {
				t.Fatalf("island size count mismatch: got %v, want %v", gotSizes, tt.wantIslandSizes)
			}
			for i := range gotSizes {
				if gotSizes[i] != tt.wantIslandSizes[i] {
					t.Errorf("island size %d: got %d, want %d", i, gotSizes[i], tt.wantIslandSizes[i])
				}
			}
		})
	}
}
