package graph

import "sync/atomic"

// DSU implements a lock-free Disjoint Set Union using atomic Compare-And-Swap
// operations for thread-safe concurrent Find and Union without mutexes.
type DSU struct {
	parent []int32
	count  int32
}

// InstantiateDSU creates a DSU where each element is its own isolated component.
func InstantiateDSU(capacity int) *DSU {
	dsu := &DSU{
		parent: make([]int32, capacity),
		count:  int32(capacity),
	}
	for i := int32(0); i < int32(capacity); i++ {
		dsu.parent[i] = i
	}
	return dsu
}

// Find returns the root representative of the component containing index.
// Uses lock-free path splitting (half-path compression) via atomic CAS.
func (dsu *DSU) Find(index int) int {
	curr := int32(index)
	for {
		p := atomic.LoadInt32(&dsu.parent[curr])
		if p == curr {
			return int(curr)
		}
		// Path compression via atomic CAS — point curr to grandparent.
		pp := atomic.LoadInt32(&dsu.parent[p])
		atomic.CompareAndSwapInt32(&dsu.parent[curr], p, pp)
		curr = p
	}
}

// Union merges the components containing u and v.
// Returns true if a merge occurred, false if they were already connected.
// Uses union-by-index (smaller root wins) to maintain deterministic structure.
func (dsu *DSU) Union(u, v int) bool {
	for {
		rootU := int32(dsu.Find(u))
		rootV := int32(dsu.Find(v))
		if rootU == rootV {
			return false
		}
		// Union by index rank to avoid cycles.
		if rootU > rootV {
			rootU, rootV = rootV, rootU
		}
		if atomic.CompareAndSwapInt32(&dsu.parent[rootV], rootV, rootU) {
			atomic.AddInt32(&dsu.count, -1)
			return true
		}
	}
}

// ActiveComponents returns the current number of disjoint components.
func (dsu *DSU) ActiveComponents() int {
	return int(atomic.LoadInt32(&dsu.count))
}
