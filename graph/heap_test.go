package graph

import (
	"container/heap"
	"testing"
)

func TestOverlapHeap_MaxOrder(t *testing.T) {
	h := &OverlapHeap{}
	heap.Init(h)

	h.PushNode(OverlapNode{LeftID: 0, RightID: 1, OverlapLen: 3})
	h.PushNode(OverlapNode{LeftID: 1, RightID: 2, OverlapLen: 5})
	h.PushNode(OverlapNode{LeftID: 2, RightID: 3, OverlapLen: 1})
	h.PushNode(OverlapNode{LeftID: 3, RightID: 4, OverlapLen: 4})

	expected := []int{5, 4, 3, 1}
	for i, want := range expected {
		got := h.PopNode()
		if got.OverlapLen != want {
			t.Errorf("pop %d: OverlapLen = %d, want %d", i, got.OverlapLen, want)
		}
	}

	if h.Len() != 0 {
		t.Errorf("heap should be empty, has %d elements", h.Len())
	}
}

func TestOverlapHeap_SingleElement(t *testing.T) {
	h := &OverlapHeap{}
	heap.Init(h)

	h.PushNode(OverlapNode{LeftID: 0, RightID: 1, OverlapLen: 7})
	got := h.PopNode()
	if got.OverlapLen != 7 {
		t.Errorf("OverlapLen = %d, want 7", got.OverlapLen)
	}
}

func TestOverlapHeap_EqualOverlaps(t *testing.T) {
	h := &OverlapHeap{}
	heap.Init(h)

	h.PushNode(OverlapNode{LeftID: 0, RightID: 1, OverlapLen: 3})
	h.PushNode(OverlapNode{LeftID: 2, RightID: 3, OverlapLen: 3})
	h.PushNode(OverlapNode{LeftID: 4, RightID: 5, OverlapLen: 3})

	// All should pop with overlap 3 (order is non-deterministic).
	for i := range 3 {
		got := h.PopNode()
		if got.OverlapLen != 3 {
			t.Errorf("pop %d: OverlapLen = %d, want 3", i, got.OverlapLen)
		}
	}
}
