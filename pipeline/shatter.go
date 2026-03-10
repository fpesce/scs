package pipeline

import (
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/joke/scs/graph"
)

// ShatterGraph partitions the survivors into weakly connected components (islands)
// based on suffix-prefix overlaps of at least minOverlap characters.
//
// Uses a sorted index + binary search for O(log N) candidate lookup instead of
// O(N) prefix hash collisions. Parallelized across CPU cores via lock-free DSU.
func ShatterGraph(survivors []string, minOverlap int) [][]string {
	if minOverlap <= 0 {
		minOverlap = 1
	}
	n := len(survivors)
	if n <= 1 {
		if n == 1 {
			return [][]string{survivors}
		}
		return nil
	}

	dsu := graph.InstantiateDSU(n)

	// Only process "long" strings (len >= minOverlap).
	// Short strings (len < minOverlap) are mathematically incapable of
	// producing an overlap >= minOverlap, so they are skipped entirely.
	var longIndices []int
	for i, s := range survivors {
		if len(s) >= minOverlap {
			longIndices = append(longIndices, i)
		}
	}

	// 1. Sort indices alphabetically by the strings they point to.
	// This replaces the O(N) map collisions with an O(log N) binary search.
	sortedIndices := make([]int, len(longIndices))
	copy(sortedIndices, longIndices)
	sort.Slice(sortedIndices, func(i, j int) bool {
		return survivors[sortedIndices[i]] < survivors[sortedIndices[j]]
	})

	numWorkers := runtime.NumCPU()
	chunkSize := (len(longIndices) + numWorkers - 1) / numWorkers
	if chunkSize == 0 {
		chunkSize = 1
	}

	var wg sync.WaitGroup

	// 2. Distribute overlap checks across all CPU cores.
	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		if start >= len(longIndices) {
			break
		}
		end := start + chunkSize
		if end > len(longIndices) {
			end = len(longIndices)
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()

			for idx := start; idx < end; idx++ {
				i := longIndices[idx]
				s := survivors[i]

				for L := minOverlap; L <= len(s); L++ {
					suffix := s[len(s)-L:]

					// Binary search instantly jumps directly to valid matches.
					searchIdx := sort.Search(len(sortedIndices), func(k int) bool {
						return survivors[sortedIndices[k]] >= suffix
					})

					// Iterate only through the validated candidates.
					for k := searchIdx; k < len(sortedIndices); k++ {
						j := sortedIndices[k]

						// Strings are sorted; break early when prefix stops matching.
						if !strings.HasPrefix(survivors[j], suffix) {
							break
						}

						// Lock-free DSU allows concurrent check AND write.
						if i != j && dsu.Find(i) != dsu.Find(j) {
							dsu.Union(i, j)
						}
					}
				}
			}
		}(start, end)
	}
	wg.Wait()

	// 3. Zero-allocation island grouping via linked list arrays.
	head := make([]int, n)
	next := make([]int, n)
	for i := 0; i < n; i++ {
		head[i] = -1
	}

	for i := 0; i < n; i++ {
		root := dsu.Find(i)
		next[i] = head[root]
		head[root] = i
	}

	var islands [][]string
	for root := 0; root < n; root++ {
		if head[root] != -1 {
			count := 0
			for curr := head[root]; curr != -1; curr = next[curr] {
				count++
			}

			// Build island exactly to size, appending backwards to preserve
			// ascending index order.
			island := make([]string, count)
			idx := count - 1
			for curr := head[root]; curr != -1; curr = next[curr] {
				island[idx] = survivors[curr]
				idx--
			}
			islands = append(islands, island)
		}
	}

	return islands
}
