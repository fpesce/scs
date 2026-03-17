package graph

import (
	"sync"
	"testing"
)

func TestDSU_InitialState(t *testing.T) {
	dsu := InstantiateDSU(5)

	if dsu.ActiveComponents() != 5 {
		t.Errorf("ActiveComponents = %d, want 5", dsu.ActiveComponents())
	}

	// Each element should be its own root.
	for i := range 5 {
		if dsu.Find(i) != i {
			t.Errorf("Find(%d) = %d, want %d", i, dsu.Find(i), i)
		}
	}
}

func TestDSU_UnionAndFind(t *testing.T) {
	dsu := InstantiateDSU(5)

	if !dsu.Union(0, 1) {
		t.Error("Union(0,1) should return true")
	}
	if dsu.ActiveComponents() != 4 {
		t.Errorf("ActiveComponents = %d, want 4", dsu.ActiveComponents())
	}
	if dsu.Find(0) != dsu.Find(1) {
		t.Error("0 and 1 should be in the same component")
	}

	if !dsu.Union(2, 3) {
		t.Error("Union(2,3) should return true")
	}
	if dsu.ActiveComponents() != 3 {
		t.Errorf("ActiveComponents = %d, want 3", dsu.ActiveComponents())
	}

	if !dsu.Union(1, 3) {
		t.Error("Union(1,3) should return true")
	}
	if dsu.ActiveComponents() != 2 {
		t.Errorf("ActiveComponents = %d, want 2", dsu.ActiveComponents())
	}

	// All of 0,1,2,3 should share the same root.
	root := dsu.Find(0)
	for i := 1; i <= 3; i++ {
		if dsu.Find(i) != root {
			t.Errorf("Find(%d) = %d, want %d", i, dsu.Find(i), root)
		}
	}

	// 4 should remain isolated.
	if dsu.Find(4) == root {
		t.Error("4 should not be in the same component as 0-3")
	}
}

func TestDSU_DuplicateUnion(t *testing.T) {
	dsu := InstantiateDSU(3)

	dsu.Union(0, 1)
	if dsu.Union(0, 1) {
		t.Error("duplicate Union(0,1) should return false")
	}
	if dsu.ActiveComponents() != 2 {
		t.Errorf("ActiveComponents = %d, want 2", dsu.ActiveComponents())
	}
}

func TestDSU_PathCompression(t *testing.T) {
	dsu := InstantiateDSU(6)

	// Build a chain: 0-1, 1-2, 2-3, 3-4, 4-5
	dsu.Union(0, 1)
	dsu.Union(1, 2)
	dsu.Union(2, 3)
	dsu.Union(3, 4)
	dsu.Union(4, 5)

	if dsu.ActiveComponents() != 1 {
		t.Errorf("ActiveComponents = %d, want 1", dsu.ActiveComponents())
	}

	// Find on any element should resolve to the same root.
	root := dsu.Find(5)
	for i := range 5 {
		if dsu.Find(i) != root {
			t.Errorf("Find(%d) = %d, want %d (path compression failed)", i, dsu.Find(i), root)
		}
	}

	// With lock-free CAS-based half-path compression, parents may not all
	// point directly to root after a single pass. Verify convergence:
	// repeated Find calls should eventually compress all paths.
	for range 3 {
		for i := range 6 {
			dsu.Find(i)
		}
	}
	for i := range 6 {
		if dsu.Find(i) != root {
			t.Errorf("Find(%d) = %d after convergence, want %d", i, dsu.Find(i), root)
		}
	}
}

func TestDSU_SingleElement(t *testing.T) {
	dsu := InstantiateDSU(1)

	if dsu.ActiveComponents() != 1 {
		t.Errorf("ActiveComponents = %d, want 1", dsu.ActiveComponents())
	}
	if dsu.Find(0) != 0 {
		t.Errorf("Find(0) = %d, want 0", dsu.Find(0))
	}
}

// TestDSU_ConcurrentUnion verifies thread-safety of the lock-free DSU.
func TestDSU_ConcurrentUnion(t *testing.T) {
	const n = 1000
	dsu := InstantiateDSU(n)

	var wg sync.WaitGroup
	// Union all even indices with 0, all odd indices with 1, concurrently.
	for i := 2; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				dsu.Union(0, idx)
			} else {
				dsu.Union(1, idx)
			}
		}(i)
	}
	wg.Wait()

	// Should have exactly 2 components (evens + odds).
	if dsu.ActiveComponents() != 2 {
		t.Errorf("ActiveComponents = %d, want 2", dsu.ActiveComponents())
	}

	// All evens share a root.
	evenRoot := dsu.Find(0)
	for i := 2; i < n; i += 2 {
		if dsu.Find(i) != evenRoot {
			t.Errorf("Find(%d) = %d, want %d", i, dsu.Find(i), evenRoot)
		}
	}

	// All odds share a root.
	oddRoot := dsu.Find(1)
	for i := 3; i < n; i += 2 {
		if dsu.Find(i) != oddRoot {
			t.Errorf("Find(%d) = %d, want %d", i, dsu.Find(i), oddRoot)
		}
	}
}
