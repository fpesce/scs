package pipeline

import (
	"runtime"
	"sync"
)

// AssembleConcurrently processes all islands using a worker pool pattern.
// Small islands (len <= dpLimit) are solved with exact DP, large ones with greedy.
// Workers are spun up equal to runtime.NumCPU().
func AssembleConcurrently(islands [][]string, dpLimit, minOverlap int) []string {
	n := len(islands)
	if n == 0 {
		return nil
	}

	numWorkers := runtime.NumCPU()
	if numWorkers > n {
		numWorkers = n
	}

	jobs := make(chan int, n)
	results := make([]string, n)
	var wg sync.WaitGroup

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
			}
		}()
	}

	// Feed jobs.
	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)

	// Wait for all workers to finish.
	wg.Wait()

	return results
}
