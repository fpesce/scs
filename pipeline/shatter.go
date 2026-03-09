package pipeline

import (
	"sort"

	"github.com/joke/scs/graph"
)

// ShatterGraph partitions the survivors into weakly connected components (islands)
// based on suffix-prefix overlaps of at least minOverlap characters.
//
// Phase 2: Hash partitioning via PrefixMap for quick candidate lookup.
// Phase 3: Multi-length overlap verification and component tracking via DSU.
func ShatterGraph(survivors []string, minOverlap int) [][]string {
	if minOverlap <= 0 {
		minOverlap = 1
	}

	n := len(survivors)
	if n == 0 {
		return nil
	}
	if n == 1 {
		return [][]string{survivors}
	}

	dsu := graph.InstantiateDSU(n)

	// Separate strings into "short" (len < minOverlap) and "long" (len >= minOverlap).
	var shortIndices []int
	var longIndices []int
	for i, s := range survivors {
		if len(s) < minOverlap {
			shortIndices = append(shortIndices, i)
		} else {
			longIndices = append(longIndices, i)
		}
	}

	// Phase 2: Build PrefixMap for long strings.
	// Maps the first minOverlap characters (prefix signature) to indices.
	prefixMap := make(map[string][]int)
	for _, i := range longIndices {
		prefix := survivors[i][:minOverlap]
		prefixMap[prefix] = append(prefixMap[prefix], i)
	}

	// Phase 3: For each long string, evaluate all suffix lengths >= minOverlap.
	// Extract the first minOverlap chars of each candidate suffix as the PrefixMap key,
	// then verify the full match. This catches overlaps of ANY length >= minOverlap,
	// fixing the bug where only the exact minOverlap-length suffix was checked.
	for _, i := range longIndices {
		s := survivors[i]
		for L := minOverlap; L <= len(s); L++ {
			p := s[len(s)-L : len(s)-L+minOverlap]
			candidates, ok := prefixMap[p]
			if !ok {
				continue
			}
			suffix := s[len(s)-L:]

			for _, j := range candidates {
				if i == j || dsu.Find(i) == dsu.Find(j) {
					continue
				}
				if len(survivors[j]) >= L && survivors[j][:L] == suffix {
					dsu.Union(i, j)
				}
			}
		}
	}

	// Evaluate short strings against the entire dataset manually.
	for _, si := range shortIndices {
		for j := 0; j < n; j++ {
			if si == j || dsu.Find(si) == dsu.Find(j) {
				continue
			}
			if graph.CalculateMaxOverlap(survivors[si], survivors[j]) >= minOverlap {
				dsu.Union(si, j)
				continue
			}
			if graph.CalculateMaxOverlap(survivors[j], survivors[si]) >= minOverlap {
				dsu.Union(si, j)
			}
		}
	}

	// Group strings into islands based on DSU root.
	groups := make(map[int][]int)
	for i := 0; i < n; i++ {
		root := dsu.Find(i)
		groups[root] = append(groups[root], i)
	}

	// Extract and sort keys for deterministic iteration.
	roots := make([]int, 0, len(groups))
	for root := range groups {
		roots = append(roots, root)
	}
	sort.Ints(roots)

	islands := make([][]string, 0, len(groups))
	for _, root := range roots {
		indices := groups[root]
		island := make([]string, len(indices))
		for k, idx := range indices {
			island[k] = survivors[idx]
		}
		islands = append(islands, island)
	}

	return islands
}
