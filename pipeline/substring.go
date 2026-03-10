package pipeline

import (
	"bytes"
	"index/suffixarray"
	"runtime"
	"sync"
	"unsafe"
)

// EliminateSubstrings removes any string that is entirely contained within
// another string in the dataset. Uses a suffix array over a \n-delimited
// buffer for O(N log N) matching with flat ~4x memory footprint.
//
// NOTE: \n is safe as the internal delimiter because ReadLines() splits on \n,
// guaranteeing that individual words never contain \n. This prevents cross-word
// matching in the suffix array. This is unrelated to the --sep flag, which
// controls .scs output format.
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

	// 1. Calculate total length for pre-allocation.
	totalLen := 0
	for _, s := range deduped {
		totalLen += len(s) + 1
	}

	// 2. Build a single buffer containing all words delimited by \n.
	var buf bytes.Buffer
	buf.Grow(totalLen)
	for _, s := range deduped {
		buf.WriteString(s)
		buf.WriteByte('\n')
	}

	// 3. Build a highly efficient Suffix Array (~4x memory footprint).
	sa := suffixarray.New(buf.Bytes())
	swallowed := make([]bool, nd)

	// 4. Concurrently lookup substrings using O(log N) operations.
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

	// 5. Gather survivors while maintaining original chronological order.
	survivors := make([]string, 0, nd)
	for i, s := range deduped {
		if !swallowed[i] {
			survivors = append(survivors, s)
		}
	}

	return survivors
}
