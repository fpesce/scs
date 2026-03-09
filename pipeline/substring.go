package pipeline

import (
	"github.com/joke/scs/graph"
)

// EliminateSubstrings removes any string that is entirely contained within
// another string in the dataset. Uses Aho-Corasick for O(N+Z) matching.
func EliminateSubstrings(uniqueStrings []string) []string {
	n := len(uniqueStrings)
	if n <= 1 {
		return uniqueStrings
	}

	ac := graph.InitializeAutomaton(n * 4)
	for i, s := range uniqueStrings {
		ac.InsertPattern(s, i)
	}
	ac.ComputeFailureLinks()

	// swallowed[i] = true means string i is fully contained in another string.
	swallowed := make([]bool, n)

	for i, text := range uniqueStrings {
		matches := ac.Search(text)
		for _, m := range matches {
			pid := m.PatternID
			if pid == i {
				continue
			}
			matchedLen := m.End - m.Start + 1
			// If the matched pattern's full length equals what was matched,
			// it is entirely contained within this text.
			if matchedLen == len(uniqueStrings[pid]) {
				// Only swallow the shorter string. If equal length, swallow the later index
				// to ensure deterministic deduplication.
				if len(uniqueStrings[pid]) < len(text) ||
					(len(uniqueStrings[pid]) == len(text) && pid > i) {
					swallowed[pid] = true
				}
			}
		}
	}

	survivors := make([]string, 0, n)
	for i, s := range uniqueStrings {
		if !swallowed[i] {
			survivors = append(survivors, s)
		}
	}

	return survivors
}
