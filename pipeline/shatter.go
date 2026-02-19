package pipeline

import (
	"github.com/joke/scs/graph"
)

// ShatterGraph partitions the survivors into weakly connected components (islands)
// based on suffix-prefix overlaps of at least minOverlap characters.
//
// Phase 2: Hash partitioning via PrefixMap for quick candidate lookup.
// Phase 3: Exact overlap verification via KMP and component tracking via DSU.
func ShatterGraph(survivors []string, minOverlap int) [][]string {
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

	// Phase 3: For each long string, extract its suffix signature and look up candidates.
	for _, i := range longIndices {
		suffix := survivors[i][len(survivors[i])-minOverlap:]
		candidates, ok := prefixMap[suffix]
		if !ok {
			continue
		}
		for _, j := range candidates {
			if i == j {
				continue
			}
			overlap := graph.CalculateMaxOverlap(survivors[i], survivors[j])
			if overlap >= minOverlap {
				dsu.Union(i, j)
			}
		}
	}

	// Evaluate short strings against the entire dataset manually.
	for _, si := range shortIndices {
		for j := 0; j < n; j++ {
			if si == j {
				continue
			}
			// Check both directions: short→other and other→short.
			overlap := graph.CalculateMaxOverlap(survivors[si], survivors[j])
			if overlap >= minOverlap {
				dsu.Union(si, j)
				continue
			}
			overlap = graph.CalculateMaxOverlap(survivors[j], survivors[si])
			if overlap >= minOverlap {
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

	islands := make([][]string, 0, len(groups))
	for _, indices := range groups {
		island := make([]string, len(indices))
		for k, idx := range indices {
			island[k] = survivors[idx]
		}
		islands = append(islands, island)
	}

	return islands
}
