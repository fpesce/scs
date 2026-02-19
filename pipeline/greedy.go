package pipeline

import (
	"container/heap"

	"github.com/joke/scs/graph"
)

// SolveGreedyHeap solves a large island using a greedy algorithm backed by
// a Max-Heap priority queue. At each step, it pops the pair with the largest
// overlap, merges them, and re-evaluates overlaps with remaining active strings.
func SolveGreedyHeap(island []string, minOverlap int) string {
	n := len(island)
	if n == 0 {
		return ""
	}
	if n == 1 {
		return island[0]
	}

	// active[i] is true if string i has not been merged yet.
	active := make([]bool, n)
	for i := range active {
		active[i] = true
	}

	// strs holds the current string content for each index.
	// When two strings merge, the new string is stored at a new index.
	strs := make([]string, n)
	copy(strs, island)

	// Calculate initial O(N^2) pairwise overlaps and push to heap.
	h := &graph.OverlapHeap{}
	heap.Init(h)

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			ov := graph.CalculateMaxOverlap(strs[i], strs[j])
			if ov > 0 {
				h.PushNode(graph.OverlapNode{
					LeftID:     i,
					RightID:    j,
					OverlapLen: ov,
				})
			}
		}
	}

	activeCount := n

	for activeCount > 1 {
		// If the heap is exhausted but we still have active strings,
		// they are disjoint — just concatenate them.
		if h.Len() == 0 {
			break
		}

		node := h.PopNode()

		// Skip if either side has been merged already.
		if !active[node.LeftID] || !active[node.RightID] {
			continue
		}

		// Merge: left + right[overlap:]
		merged := strs[node.LeftID] + strs[node.RightID][node.OverlapLen:]

		// Deactivate the old entries.
		active[node.LeftID] = false
		active[node.RightID] = false

		// Create a new index for the merged string.
		newIdx := len(strs)
		strs = append(strs, merged)
		active = append(active, true)

		// Calculate overlaps between new string and all remaining active strings.
		for i := range active {
			if !active[i] || i == newIdx {
				continue
			}
			// new → i
			ov := graph.CalculateMaxOverlap(strs[newIdx], strs[i])
			if ov > 0 {
				h.PushNode(graph.OverlapNode{
					LeftID:     newIdx,
					RightID:    i,
					OverlapLen: ov,
				})
			}
			// i → new
			ov = graph.CalculateMaxOverlap(strs[i], strs[newIdx])
			if ov > 0 {
				h.PushNode(graph.OverlapNode{
					LeftID:     i,
					RightID:    newIdx,
					OverlapLen: ov,
				})
			}
		}

		activeCount--
	}

	// Concatenate all remaining active strings.
	var result string
	for i := range active {
		if active[i] {
			result += strs[i]
		}
	}

	return result
}
