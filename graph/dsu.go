package graph

// DSU implements a flat-array Disjoint Set Union with path compression
// and union-by-size for near-O(1) amortized operations.
type DSU struct {
	parent []int
	size   []int
	count  int
}

// InstantiateDSU creates a DSU where each element is its own isolated component.
func InstantiateDSU(capacity int) *DSU {
	dsu := &DSU{
		parent: make([]int, capacity),
		size:   make([]int, capacity),
		count:  capacity,
	}
	for i := 0; i < capacity; i++ {
		dsu.parent[i] = i
		dsu.size[i] = 1
	}
	return dsu
}

// Find returns the root representative of the component containing index.
// Path compression is applied to flatten the tree on each query.
func (dsu *DSU) Find(index int) int {
	if dsu.parent[index] == index {
		return index
	}
	dsu.parent[index] = dsu.Find(dsu.parent[index])
	return dsu.parent[index]
}

// Union merges the components containing u and v.
// Returns true if a merge occurred, false if they were already connected.
// Uses union-by-size to maintain balanced trees.
func (dsu *DSU) Union(u, v int) bool {
	rootU := dsu.Find(u)
	rootV := dsu.Find(v)

	if rootU == rootV {
		return false
	}

	if dsu.size[rootU] < dsu.size[rootV] {
		dsu.parent[rootU] = rootV
		dsu.size[rootV] += dsu.size[rootU]
	} else {
		dsu.parent[rootV] = rootU
		dsu.size[rootU] += dsu.size[rootV]
	}

	dsu.count--
	return true
}

// ActiveComponents returns the current number of disjoint components.
func (dsu *DSU) ActiveComponents() int {
	return dsu.count
}
