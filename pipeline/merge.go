package pipeline

import (
	"index/suffixarray"
	"runtime"
	"sort"
	"sync"

	"github.com/joke/scs/graph"
)

// MergeEliminateSubstrings builds a suffix array from primaryPayload and
// concurrently checks which updateWords exist within it. Words found inside
// the primary payload are "eliminated" (remapped to primary's address space).
// Returns survivors (words not found) and a map of eliminated words to their offsets.
func MergeEliminateSubstrings(primaryPayload string, updateWords []string) (survivors []string, eliminatedMap map[string]int) {
	eliminatedMap = make(map[string]int)
	if len(updateWords) == 0 || len(primaryPayload) == 0 {
		return updateWords, eliminatedMap
	}

	sa := suffixarray.New([]byte(primaryPayload))

	numWorkers := runtime.NumCPU()
	chunkSize := (len(updateWords) + numWorkers - 1) / numWorkers

	type workerResult struct {
		survivors []string
		elim      map[string]int
	}
	results := make([]workerResult, numWorkers)

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		if start >= len(updateWords) {
			break
		}
		end := start + chunkSize
		if end > len(updateWords) {
			end = len(updateWords)
		}

		wg.Add(1)
		go func(w, start, end int) {
			defer wg.Done()

			var localSurvivors []string
			localElim := make(map[string]int)

			for i := start; i < end; i++ {
				word := updateWords[i]
				if word == "" {
					localElim[word] = 0
					continue
				}
				matches := sa.Lookup([]byte(word), 1)
				if len(matches) > 0 {
					localElim[word] = matches[0]
				} else {
					localSurvivors = append(localSurvivors, word)
				}
			}

			results[w] = workerResult{
				survivors: localSurvivors,
				elim:      localElim,
			}
		}(w, start, end)
	}
	wg.Wait()

	// Recombine results sequentially.
	for _, res := range results {
		if len(res.survivors) > 0 {
			survivors = append(survivors, res.survivors...)
		}
		for k, v := range res.elim {
			eliminatedMap[k] = v
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
