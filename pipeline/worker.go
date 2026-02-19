package pipeline

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
)

// AssembleConcurrently processes all islands using a worker pool pattern.
// Small islands (len <= dpLimit) are solved with exact DP, large ones with greedy.
// Workers are spun up equal to runtime.NumCPU().
// Jobs are sorted descending by island size to prevent tail stalling.
// When verbose is true, progress tracks strings assembled (not island count)
// for smooth reporting even when large islands dominate processing time.
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
					results[idx] = SolveGreedyHeap(island, minOverlap)
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
