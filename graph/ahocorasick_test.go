package graph

import (
	"sort"
	"testing"
)

func TestAhoCorasick_BasicSearch(t *testing.T) {
	ac := InitializeAutomaton(10)
	ac.InsertPattern("he", 0)
	ac.InsertPattern("she", 1)
	ac.InsertPattern("his", 2)
	ac.InsertPattern("hers", 3)
	ac.ComputeFailureLinks()

	matches := ac.Search("ushers")

	ids := make(map[int]bool)
	for _, m := range matches {
		ids[m.PatternID] = true
	}

	if !ids[1] {
		t.Error("expected to find pattern 'she' (ID 1)")
	}
	if !ids[0] {
		t.Error("expected to find pattern 'he' (ID 0)")
	}
	if !ids[3] {
		t.Error("expected to find pattern 'hers' (ID 3)")
	}
}

func TestAhoCorasick_NestedSubstring(t *testing.T) {
	// "art" should be found inside "cart"
	ac := InitializeAutomaton(10)
	ac.InsertPattern("art", 0)
	ac.InsertPattern("cart", 1)
	ac.ComputeFailureLinks()

	matches := ac.Search("cart")

	ids := make(map[int]bool)
	for _, m := range matches {
		ids[m.PatternID] = true
	}

	if !ids[0] {
		t.Error("expected to find nested pattern 'art' (ID 0) inside 'cart'")
	}
	if !ids[1] {
		t.Error("expected to find pattern 'cart' (ID 1)")
	}
}

func TestAhoCorasick_NoMatch(t *testing.T) {
	ac := InitializeAutomaton(5)
	ac.InsertPattern("xyz", 0)
	ac.ComputeFailureLinks()

	matches := ac.Search("abcdef")
	if len(matches) != 0 {
		t.Errorf("expected no matches, got %d", len(matches))
	}
}

func TestAhoCorasick_OverlappingPatterns(t *testing.T) {
	ac := InitializeAutomaton(10)
	ac.InsertPattern("ab", 0)
	ac.InsertPattern("abc", 1)
	ac.InsertPattern("bc", 2)
	ac.ComputeFailureLinks()

	matches := ac.Search("abc")

	ids := make(map[int]bool)
	for _, m := range matches {
		ids[m.PatternID] = true
	}

	if !ids[0] {
		t.Error("expected 'ab' (ID 0)")
	}
	if !ids[1] {
		t.Error("expected 'abc' (ID 1)")
	}
	if !ids[2] {
		t.Error("expected 'bc' (ID 2)")
	}
}

func TestAhoCorasick_MatchPositions(t *testing.T) {
	ac := InitializeAutomaton(5)
	ac.InsertPattern("ana", 0)
	ac.ComputeFailureLinks()

	matches := ac.Search("banana")

	// "ana" appears at positions 1 and 3 in "banana"
	var starts []int
	for _, m := range matches {
		starts = append(starts, m.Start)
	}
	sort.Ints(starts)

	want := []int{1, 3}
	if len(starts) != len(want) {
		t.Fatalf("got %d matches, want %d: %v", len(starts), len(want), starts)
	}
	for i := range starts {
		if starts[i] != want[i] {
			t.Errorf("match %d: Start = %d, want %d", i, starts[i], want[i])
		}
	}
}

func TestAhoCorasick_EmptyPatternIgnored(t *testing.T) {
	ac := InitializeAutomaton(5)
	ac.InsertPattern("", 0)
	ac.InsertPattern("abc", 1)
	ac.ComputeFailureLinks()

	// Empty pattern insertion should not cause panics during search.
	matches := ac.Search("abc")

	found := false
	for _, m := range matches {
		if m.PatternID == 1 {
			found = true
		}
	}
	if !found {
		t.Error("expected to find 'abc' (ID 1)")
	}
}

func TestAhoCorasick_SingleCharPatterns(t *testing.T) {
	ac := InitializeAutomaton(10)
	ac.InsertPattern("a", 0)
	ac.InsertPattern("b", 1)
	ac.ComputeFailureLinks()

	matches := ac.Search("abba")

	counts := make(map[int]int)
	for _, m := range matches {
		counts[m.PatternID]++
	}

	if counts[0] != 2 {
		t.Errorf("expected 2 matches for 'a', got %d", counts[0])
	}
	if counts[1] != 2 {
		t.Errorf("expected 2 matches for 'b', got %d", counts[1])
	}
}
