package graph

import (
	"testing"
)

func TestDSU_InitialState(t *testing.T) {
	dsu := InstantiateDSU(5)

	if dsu.ActiveComponents() != 5 {
		t.Errorf("ActiveComponents = %d, want 5", dsu.ActiveComponents())
	}

	// Each element should be its own root.
	for i := 0; i < 5; i++ {
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
	for i := 0; i < 5; i++ {
		if dsu.Find(i) != root {
			t.Errorf("Find(%d) = %d, want %d (path compression failed)", i, dsu.Find(i), root)
		}
	}

	// After path compression, all parents should point directly to root.
	for i := 0; i < 6; i++ {
		if dsu.parent[i] != root {
			t.Errorf("parent[%d] = %d, want %d (path compression not flattened)", i, dsu.parent[i], root)
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
