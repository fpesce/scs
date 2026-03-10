package pipeline

import (
	"strings"

	"github.com/joke/scs/graph"
)

// SolveGreedyHeap solves a large island using a progressive top-down scan.
// Instead of precomputing all overlaps into a heap, it iterates overlap length
// L from maxLen down to 1. The first match found at any L is guaranteed to be
// the longest possible overlap — no heap needed.
//
// Uses a zero-allocation inline linked list (head + listNext) instead of
// dynamically allocated map[string][]int slices to eliminate GC thrashing.
//
// maxLen is capped at 1000 to prevent O(maxLen * N) string hashing blowup
// during hierarchical recursive merging where superstrings grow to hundreds
// of thousands of characters.
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

	// Cap maxLen to prevent O(maxLen * N) string hashing blowup on massive
	// superstrings generated during hierarchical merging recursion.
	const maxSearchLimit = 1000
	if maxLen > maxSearchLimit {
		maxLen = maxSearchLimit
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

	// Allocate strictly ONCE outside the loop — reuse via clear().
	head := make(map[string]int)
	listNext := make([]int, n)

	// Progressively match from longest possible overlap down to 1.
	for L := maxLen; L >= 1; L-- {
		clear(head)

		// Build inline linked list of length-L prefixes for available v nodes.
		// Iterate backward so prepended list retains chronological order.
		for v := n - 1; v >= 0; v-- {
			if prev[v] == -1 && len(island[v]) >= L {
				p := island[v][:L]
				// Build linked list using 1-indexed values (0 = nil).
				listNext[v] = head[p]
				head[p] = v + 1
			}
		}

		if len(head) == 0 {
			continue
		}

		// For each available u (next[u]==-1), check if its suffix matches.
		for u := 0; u < n; u++ {
			if next[u] != -1 || len(island[u]) < L {
				continue
			}
			suffix := island[u][len(island[u])-L:]

			// Traverse the zero-allocation inline linked list.
			curr := head[suffix]
			for curr != 0 {
				v := curr - 1
				curr = listNext[v] // advance

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
				break
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
