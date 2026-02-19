package pipeline

import (
	"container/heap"
	"strings"

	"github.com/joke/scs/graph"
)

// SolveGreedyHeap solves a large island using a Kruskal-style max-weight
// path cover. Overlaps are computed once using PrefixMap acceleration,
// then edges are popped from a max-heap. No physical string merging occurs
// during assembly — only pointer chains are built.
func SolveGreedyHeap(island []string, minOverlap int) string {
	if minOverlap <= 0 {
		minOverlap = 1
	}

	n := len(island)
	if n == 0 {
		return ""
	}
	if n == 1 {
		return island[0]
	}

	// Build PrefixMap for O(1) candidate lookup.
	prefixMap := make(map[string][]int)
	var shortIndices []int
	for i, s := range island {
		if len(s) >= minOverlap {
			p := s[:minOverlap]
			prefixMap[p] = append(prefixMap[p], i)
		} else {
			shortIndices = append(shortIndices, i)
		}
	}

	// Compute all overlaps once and collect into a slice for bulk heap init.
	hSlice := make(graph.OverlapHeap, 0, n)
	h := &hSlice

	seen := make([]int, n)
	for i := range seen {
		seen[i] = -1
	}

	for i, s := range island {
		if len(s) >= minOverlap {
			// Try all suffix lengths from longest to shortest.
			for L := len(s); L >= minOverlap; L-- {
				p := s[len(s)-L : len(s)-L+minOverlap]
				candidates, ok := prefixMap[p]
				if !ok {
					continue
				}
				suffix := s[len(s)-L:]

				for _, j := range candidates {
					if i == j || seen[j] == i {
						continue
					}
					if len(island[j]) >= L && island[j][:L] == suffix {
						seen[j] = i
						*h = append(*h, graph.OverlapNode{LeftID: i, RightID: j, OverlapLen: L})
					}
				}
			}
			// Handle short right-side strings.
			for _, j := range shortIndices {
				if i == j {
					continue
				}
				ov := graph.CalculateMaxOverlap(s, island[j])
				if ov > 0 {
					*h = append(*h, graph.OverlapNode{LeftID: i, RightID: j, OverlapLen: ov})
				}
			}
		} else {
			// Short string: brute-force against all.
			for j := 0; j < n; j++ {
				if i == j {
					continue
				}
				ov := graph.CalculateMaxOverlap(s, island[j])
				if ov > 0 {
					*h = append(*h, graph.OverlapNode{LeftID: i, RightID: j, OverlapLen: ov})
				}
			}
		}
	}

	heap.Init(h)

	// Kruskal-style path cover via pointer chains.
	next := make([]int, n)
	prev := make([]int, n)
	nextOverlap := make([]int, n)
	for i := 0; i < n; i++ {
		next[i] = -1
		prev[i] = -1
	}

	dsu := graph.InstantiateDSU(n)
	mergedCount := 0

	for h.Len() > 0 && mergedCount < n-1 {
		node := h.PopNode()
		u, v := node.LeftID, node.RightID
		// Accept edge only if u has no outgoing, v has no incoming, and no cycle.
		if next[u] == -1 && prev[v] == -1 && dsu.Find(u) != dsu.Find(v) {
			next[u] = v
			prev[v] = u
			nextOverlap[u] = node.OverlapLen
			dsu.Union(u, v)
			mergedCount++
		}
	}

	// Reconstruct chains from path heads (nodes with no predecessor).
	var result strings.Builder
	for i := 0; i < n; i++ {
		if prev[i] == -1 {
			curr := i
			result.WriteString(island[curr])
			for next[curr] != -1 {
				ov := nextOverlap[curr]
				nxt := next[curr]
				result.WriteString(island[nxt][ov:])
				curr = nxt
			}
		}
	}

	return result.String()
}
