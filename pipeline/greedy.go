package pipeline

import (
	"strings"
	"time"

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
	return SolveGreedyHeapWithDeadline(island, minOverlap, time.Time{})
}

// SolveGreedyHeapWithDeadline is like SolveGreedyHeap but aborts gracefully
// when the deadline is exceeded, returning a partially-assembled result.
func SolveGreedyHeapWithDeadline(island []string, minOverlap int, deadline time.Time) string {
	n := len(island)
	if n == 0 {
		return ""
	}
	if n == 1 {
		return island[0]
	}

	path, overlaps := solveGreedyPathWithDeadline(island, minOverlap, deadline)

	// Pre-calculate exact builder capacity for zero-realloc final assembly.
	totalLen := len(island[path[0]])
	for i := 0; i < len(path)-1; i++ {
		totalLen += len(island[path[i+1]]) - overlaps[i]
	}

	var result strings.Builder
	result.Grow(totalLen)

	result.WriteString(island[path[0]])
	for i := 0; i < len(path)-1; i++ {
		ov := overlaps[i]
		result.WriteString(island[path[i+1]][ov:])
	}

	return result.String()
}

// solveGreedyPath performs the core greedy matching and returns the
// permutation path (index order) and the overlap between consecutive
// path elements. This is used by SolveGreedyHeap and to seed the GA.
func solveGreedyPath(island []string, minOverlap int) ([]int, []int) {
	return solveGreedyPathWithDeadline(island, minOverlap, time.Time{})
}

// solveGreedyPathWithDeadline is like solveGreedyPath but respects a deadline.
// When the deadline expires, it stops matching and builds the path from
// whatever chains have been assembled so far.
func solveGreedyPathWithDeadline(island []string, minOverlap int, deadline time.Time) ([]int, []int) {
	if minOverlap <= 0 {
		minOverlap = 1
	}

	n := len(island)

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
		// Break cleanly when deadline expires — yields a partially-assembled sequence.
		if !deadline.IsZero() && time.Now().After(deadline) {
			goto buildPath
		}

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
			// Throttle syscall checks with bitwise mask.
			if !deadline.IsZero() && u&4095 == 0 && time.Now().After(deadline) {
				goto buildPath
			}

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

buildPath:
	// Build the permutation path by tracing chains from head nodes.
	// When multiple chains exist, insert overlap=0 at chain boundaries.
	path := make([]int, 0, n)
	overlaps := make([]int, 0, n-1)
	for i := 0; i < n; i++ {
		if prev[i] == -1 {
			// Insert zero-overlap boundary between chains.
			if len(path) > 0 {
				overlaps = append(overlaps, 0)
			}
			curr := i
			path = append(path, curr)
			for next[curr] != -1 {
				overlaps = append(overlaps, nextOverlap[curr])
				curr = next[curr]
				path = append(path, curr)
			}
		}
	}

	return path, overlaps
}
