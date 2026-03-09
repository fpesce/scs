package pipeline

import (
	"sort"

	"github.com/joke/scs/graph"
)

// MergeEliminateSubstrings builds an Aho-Corasick automaton from updateWords
// and streams the primaryPayload through it. Words found organically inside
// the primary payload are "eliminated" (remapped to primary's address space).
// Returns survivors (words not found) and a map of eliminated words to their offsets.
func MergeEliminateSubstrings(primaryPayload string, updateWords []string) (survivors []string, eliminatedMap map[string]int) {
	eliminatedMap = make(map[string]int)

	if len(updateWords) == 0 || len(primaryPayload) == 0 {
		return updateWords, eliminatedMap
	}

	// Build automaton from update words.
	ac := graph.InitializeAutomaton(len(updateWords) * 4)
	for i, w := range updateWords {
		if w != "" {
			ac.InsertPattern(w, i)
		}
	}
	ac.ComputeFailureLinks()

	// Stream primary payload through the automaton.
	matches := ac.Search(primaryPayload)

	// Track which update words were found, recording the first offset.
	found := make(map[int]int) // patternID -> offset
	for _, m := range matches {
		if _, exists := found[m.PatternID]; !exists {
			found[m.PatternID] = m.Start
		}
	}

	// Partition into eliminated and survivors.
	for i, w := range updateWords {
		if offset, ok := found[i]; ok {
			eliminatedMap[w] = offset
		} else {
			survivors = append(survivors, w)
		}
	}

	return survivors, eliminatedMap
}

// TruncateAndAppend generates a compressed mini-superstring from surviving words,
// then calculates the maximum suffix-prefix overlap with the primary payload
// to truncate redundant boundary characters.
func TruncateAndAppend(primaryPayload string, survivors []string, minOverlap, dpLimit int) (fragment string, overlapLength int) {
	if len(survivors) == 0 {
		return "", 0
	}

	// Deduplicate survivors.
	unique := ExactDeduplication(survivors)

	// Eliminate substrings within survivors.
	unique = EliminateSubstrings(unique)

	if len(unique) == 0 {
		return "", 0
	}

	// Generate mini-superstring.
	islands := ShatterGraph(unique, minOverlap)
	superWords := AssembleConcurrently(islands, dpLimit, minOverlap, false)

	// Sort the same way as build.
	sort.Slice(superWords, func(i, j int) bool {
		if len(superWords[i]) == len(superWords[j]) {
			return superWords[i] < superWords[j]
		}
		return len(superWords[i]) > len(superWords[j])
	})

	miniSuper := ""
	for _, sw := range superWords {
		miniSuper += sw
	}

	if miniSuper == "" {
		return "", 0
	}

	// Calculate suffix-prefix overlap between primary payload and miniSuper.
	overlapLength = graph.CalculateMaxOverlap(primaryPayload, miniSuper)

	// Truncate the overlap from the front of miniSuper.
	fragment = miniSuper[overlapLength:]

	return fragment, overlapLength
}
