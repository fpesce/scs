package format

import (
	"index/suffixarray"
	"runtime"
	"sync"
	"unsafe"
)

// MapOffsets performs a single suffix array lookup over the masterString
// to locate one valid starting byte offset for every unique source string.
// Uses concurrent workers for O(N log M) matching with flat ~4x memory footprint.
// Zero-copy unsafe cast avoids heap allocation per string→[]byte conversion.
func MapOffsets(masterString string, uniqueSourceStrings []string) map[string]int {
	if len(uniqueSourceStrings) == 0 {
		return make(map[string]int)
	}

	sa := suffixarray.New([]byte(masterString))

	offsetMap := make(map[string]int, len(uniqueSourceStrings))
	var mu sync.Mutex

	numWorkers := runtime.NumCPU()
	chunkSize := (len(uniqueSourceStrings) + numWorkers - 1) / numWorkers

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		if start >= len(uniqueSourceStrings) {
			break
		}
		end := start + chunkSize
		if end > len(uniqueSourceStrings) {
			end = len(uniqueSourceStrings)
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()

			// Use local map to prevent lock contention.
			localMap := make(map[string]int, end-start)
			for i := start; i < end; i++ {
				s := uniqueSourceStrings[i]
				if s == "" {
					localMap[s] = 0
					continue
				}
				// Zero-copy cast: string → []byte without allocation.
				b := unsafe.Slice(unsafe.StringData(s), len(s))
				matches := sa.Lookup(b, 1)
				if len(matches) > 0 {
					localMap[s] = matches[0]
				}
			}

			mu.Lock()
			for k, v := range localMap {
				offsetMap[k] = v
			}
			mu.Unlock()
		}(start, end)
	}
	wg.Wait()

	return offsetMap
}
