package graph

import "container/heap"

// OverlapNode represents a candidate merge between two strings.
type OverlapNode struct {
	LeftID     int
	RightID    int
	OverlapLen int
}

// OverlapHeap implements a max-heap of OverlapNode via container/heap.
// Tie-breaking is non-deterministic (natural pop order).
type OverlapHeap []OverlapNode

func (h OverlapHeap) Len() int           { return len(h) }
func (h OverlapHeap) Less(i, j int) bool { return h[i].OverlapLen > h[j].OverlapLen }
func (h OverlapHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

// Push adds an element to the heap.
func (h *OverlapHeap) Push(x any) {
	*h = append(*h, x.(OverlapNode))
}

// Pop removes and returns the maximum element.
func (h *OverlapHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// PushNode is a typed convenience wrapper around heap.Push.
func (h *OverlapHeap) PushNode(node OverlapNode) {
	heap.Push(h, node)
}

// PopNode is a typed convenience wrapper around heap.Pop.
func (h *OverlapHeap) PopNode() OverlapNode {
	return heap.Pop(h).(OverlapNode)
}
