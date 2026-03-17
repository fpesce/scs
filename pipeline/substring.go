package pipeline

import (
	"bytes"
	"index/suffixarray"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

// EliminateSubstrings removes any string that is entirely contained within
// another string in the dataset. Uses a suffix array over a delimiter-separated
// buffer for O(N log N) matching with flat ~4x memory footprint.
//
// The delimiter byte is dynamically chosen to be a value not present in any
// input word, making this safe for arbitrary binary data (e.g. tiktoken tokens).
//
// Internal dedup prevents duplicate strings from causing false SA elimination.
func EliminateSubstrings(uniqueStrings []string) []string {
	n := len(uniqueStrings)
	if n <= 1 {
		return uniqueStrings
	}

	// Lightweight dedup — prevents identical strings from appearing >1 times
	// in the SA buffer which would cause false elimination.
	seen := make(map[string]bool, n)
	deduped := make([]string, 0, n)
	for _, s := range uniqueStrings {
		if !seen[s] {
			seen[s] = true
			deduped = append(deduped, s)
		}
	}

	nd := len(deduped)
	if nd <= 1 {
		return deduped
	}

	// 1. Find a delimiter byte that does not appear in any word.
	// This prevents cross-word matching in the suffix array and avoids
	// false elimination when a word IS the delimiter byte itself
	// (e.g. a literal "\n" token in tiktoken mode).
	usedBytes := [256]bool{}
	for _, s := range deduped {
		for _, b := range []byte(s) {
			usedBytes[b] = true
		}
	}
	delim := byte(0xFF) // sentinel
	found := false
	for b := 0; b < 256; b++ {
		if !usedBytes[byte(b)] {
			delim = byte(b)
			found = true
			break
		}
	}
	if !found {
		// Extremely rare: all 256 byte values used. Fall back to O(n²).
		return eliminateSubstringsBrute(deduped)
	}

	// 2. Calculate total length for pre-allocation.
	totalLen := 0
	for _, s := range deduped {
		totalLen += len(s) + 1
	}

	// 3. Build a single buffer containing all words delimited by the chosen byte.
	var buf bytes.Buffer
	buf.Grow(totalLen)
	for _, s := range deduped {
		buf.WriteString(s)
		buf.WriteByte(delim)
	}

	// 4. Build a highly efficient Suffix Array (~4x memory footprint).
	sa := suffixarray.New(buf.Bytes())
	swallowed := make([]bool, nd)

	// 5. Concurrently lookup substrings using O(log N) operations.
	numWorkers := runtime.NumCPU()
	chunkSize := (nd + numWorkers - 1) / numWorkers

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		if start >= nd {
			break
		}
		end := start + chunkSize
		if end > nd {
			end = nd
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				s := deduped[i]
				if len(s) == 0 {
					swallowed[i] = true
					continue
				}
				// Zero-copy cast: string → []byte without allocation.
				b := unsafe.Slice(unsafe.StringData(s), len(s))
				// Look for up to 2 occurrences. If > 1, it exists inside another word.
				if len(sa.Lookup(b, 2)) > 1 {
					swallowed[i] = true
				}
			}
		}(start, end)
	}
	wg.Wait()

	// 6. Gather survivors while maintaining original chronological order.
	survivors := make([]string, 0, nd)
	for i, s := range deduped {
		if !swallowed[i] {
			survivors = append(survivors, s)
		}
	}

	return survivors
}

// eliminateSubstringsBrute is an O(n²) fallback for the rare case where
// all 256 byte values are used in the input, making suffix-array delimiter
// selection impossible.
func eliminateSubstringsBrute(words []string) []string {
	swallowed := make([]bool, len(words))
	for i, a := range words {
		if swallowed[i] {
			continue
		}
		for j, b := range words {
			if i == j || swallowed[j] {
				continue
			}
			if len(a) < len(b) && strings.Contains(b, a) {
				swallowed[i] = true
				break
			}
		}
	}
	survivors := make([]string, 0, len(words))
	for i, s := range words {
		if !swallowed[i] {
			survivors = append(survivors, s)
		}
	}
	return survivors
}
