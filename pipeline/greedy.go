package pipeline

import (
	"strings"

	"github.com/joke/scs/graph"
)

// SolveGreedyHeap solves a large island using a progressive top-down scan.
// Instead of precomputing all overlaps into a heap, it iterates overlap length
// L from maxLen down to 1. The first match found at any L is guaranteed to be
// the longest possible overlap — no heap needed.
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

	maxLen := 0
	for _, s := range island {
		if len(s) > maxLen {
			maxLen = len(s)
		}
	}

	next := make([]int, n)
	prev := make([]int, n)
	nextOverlap := make([]int, n)
	for i := 0; i < n; i++ {
		next[i] = -1
		prev[i] = -1
	}

	dsu := graph.InstantiateDSU(n)
	mergedCount := 0

	// Progressively match from longest possible overlap down to 1.
	for L := maxLen; L >= 1; L-- {
		// Build PrefixMap of length-L prefixes for available v nodes (prev[v]==-1).
		prefixMap := make(map[string][]int)
		for v := 0; v < n; v++ {
			if prev[v] == -1 && len(island[v]) >= L {
				p := island[v][:L]
				prefixMap[p] = append(prefixMap[p], v)
			}
		}

		if len(prefixMap) == 0 {
			continue
		}

		// For each available u (next[u]==-1), check if its suffix matches.
		for u := 0; u < n; u++ {
			if next[u] != -1 || len(island[u]) < L {
				continue
			}
			suffix := island[u][len(island[u])-L:]
			candidates, ok := prefixMap[suffix]
			if !ok {
				continue
			}
			for _, v := range candidates {
				if u == v || prev[v] != -1 || dsu.Find(u) == dsu.Find(v) {
					continue
				}
				// Guard: overlaps below minOverlap only accepted if at least
				// one string is shorter than minOverlap.
				if L < minOverlap && len(island[u]) >= minOverlap && len(island[v]) >= minOverlap {
					continue
				}

				next[u] = v
				prev[v] = u
				nextOverlap[u] = L
				dsu.Union(u, v)
				mergedCount++
				break // First match at this L is the best for u.
			}
		}

		if mergedCount >= n-1 {
			break
		}
	}

	// Pre-calculate exact builder capacity for zero-realloc final assembly.
	totalLen := 0
	for i := 0; i < n; i++ {
		if prev[i] == -1 {
			curr := i
			totalLen += len(island[curr])
			for next[curr] != -1 {
				nxt := next[curr]
				totalLen += len(island[nxt]) - nextOverlap[curr]
				curr = nxt
			}
		}
	}

	var result strings.Builder
	result.Grow(totalLen)

	// Reconstruct chains from path heads (nodes with no predecessor).
	for i := 0; i < n; i++ {
		if prev[i] == -1 {
			curr := i
			result.WriteString(island[curr])
			for next[curr] != -1 {
				nxt := next[curr]
				ov := nextOverlap[curr]
				result.WriteString(island[nxt][ov:])
				curr = nxt
			}
		}
	}

	return result.String()
}
