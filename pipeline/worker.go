package pipeline

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
)

// SolveHierarchicalGreedy partitions a massive dataset into chunks using a
// fast lexicographic sort (which naturally groups strings with shared prefixes),
// solves them in parallel via an inner worker pool, and recursively reduces.
//
// This replaces the O(N²) BFS chunking loop that pegged a single core.
func SolveHierarchicalGreedy(island []string, minOverlap, chunkSize int) string {
	if len(island) <= chunkSize {
		return SolveGreedyHeap(island, minOverlap)
	}

	n := len(island)

	// 1. FAST GROUPING: Sort alphabetically to natively group strings with shared prefixes.
	sortedIsland := make([]string, n)
	copy(sortedIsland, island)
	sort.Strings(sortedIsland)

	// 2. Slice the sorted array into manageable chunks instantly.
	var chunks [][]string
	for i := 0; i < n; i += chunkSize {
		end := i + chunkSize
		if end > n {
			end = n
		}
		chunks = append(chunks, sortedIsland[i:end])
	}

	results := make([]string, len(chunks))
	var wg sync.WaitGroup

	// 3. Process chunks using an INNER worker pool for full CPU saturation.
	numWorkers := runtime.NumCPU()
	if numWorkers <= 0 {
		numWorkers = 1
	}

	jobChan := make(chan int, len(chunks))
	for i := range chunks {
		jobChan <- i
	}
	close(jobChan)

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobChan {
				results[idx] = SolveGreedyHeap(chunks[idx], minOverlap)
			}
		}()
	}
	wg.Wait()

	// 4. Reduce Phase: Recursively combine the resulting mini-superstrings.
	return SolveHierarchicalGreedy(results, minOverlap, chunkSize)
}

// AssembleConcurrently processes all islands using a worker pool pattern.
// Small islands (len <= dpLimit) are solved with exact DP, large ones with
// hierarchical sort+slice chunking for full CPU saturation.
// Jobs are sorted descending by island size to prevent tail stalling.
func AssembleConcurrently(islands [][]string, dpLimit, minOverlap int, verbose bool) []string {
	n := len(islands)
	if n == 0 {
		return nil
	}

	numWorkers := runtime.NumCPU()
	if numWorkers > n {
		numWorkers = n
	}

	// Count total strings for smooth progress reporting.
	totalStrings := 0
	for _, island := range islands {
		totalStrings += len(island)
	}

	jobs := make(chan int, n)
	results := make([]string, n)
	var wg sync.WaitGroup
	var completedStrings int32

	// Spin up workers.
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				island := islands[idx]
				if len(island) <= dpLimit {
					results[idx] = SolveExactDP(island, minOverlap)
				} else {
					// Route giant islands into hierarchical sort+slice chunker.
					results[idx] = SolveHierarchicalGreedy(island, minOverlap, 2000)
				}
				if verbose {
					c := atomic.AddInt32(&completedStrings, int32(len(island)))
					fmt.Printf("\r  Assembling... %d/%d strings (%d%%)",
						c, totalStrings, int(c)*100/totalStrings)
				}
			}
		}()
	}

	// Sort jobs descending by island size to prevent tail stalling.
	type jobDesc struct{ idx, size int }
	jobList := make([]jobDesc, n)
	for i := 0; i < n; i++ {
		jobList[i] = jobDesc{idx: i, size: len(islands[i])}
	}
	sort.Slice(jobList, func(i, j int) bool {
		return jobList[i].size > jobList[j].size
	})

	for _, j := range jobList {
		jobs <- j.idx
	}
	close(jobs)

	// Wait for all workers to finish.
	wg.Wait()

	if verbose && n > 0 {
		fmt.Println()
	}

	return results
}
