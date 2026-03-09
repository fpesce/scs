package format

import (
	"github.com/joke/scs/graph"
)

// MapOffsets performs a single O(N) Aho-Corasick pass over the masterString
// to locate one valid starting byte offset for every unique source string.
func MapOffsets(masterString string, uniqueSourceStrings []string) map[string]int {
	if len(uniqueSourceStrings) == 0 {
		return make(map[string]int)
	}

	// Build the automaton.
	ac := graph.InitializeAutomaton(len(uniqueSourceStrings) * 4)
	for i, s := range uniqueSourceStrings {
		if s != "" {
			ac.InsertPattern(s, i)
		}
	}
	ac.ComputeFailureLinks()

	// Single-pass search.
	matches := ac.Search(masterString)

	// Record exactly one offset per unique string (first occurrence).
	offsetMap := make(map[string]int, len(uniqueSourceStrings))
	found := make(map[int]bool, len(uniqueSourceStrings))

	for _, m := range matches {
		if found[m.PatternID] {
			continue
		}
		found[m.PatternID] = true
		offsetMap[uniqueSourceStrings[m.PatternID]] = m.Start
	}

	return offsetMap
}
