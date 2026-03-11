package pipeline

import (
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joke/scs/cli"
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

// SolveHierarchicalGA splits massive islands into chunks of chunkSize,
// distributes proportional fractions of the time budget across chunks,
// solves each chunk with the GA solver, and recursively reduces.
// By ensuring N never exceeds chunkSize (20000), memory stays bounded.
func SolveHierarchicalGA(island []string, minOverlap, chunkSize int, totalWallClock time.Duration, totalCores int, cfg *cli.BuildConfig, logger *GALogger) string {
	if len(island) <= chunkSize {
		if totalWallClock >= 10*time.Millisecond {
			return SolveGA(island, minOverlap, totalWallClock, totalCores, cfg, logger)
		}
		var dl time.Time
		if totalWallClock > 0 {
			dl = time.Now().Add(totalWallClock)
		}
		return SolveGreedyHeapWithDeadline(island, minOverlap, dl)
	}

	n := len(island)
	sortedIsland := make([]string, n)
	copy(sortedIsland, island)
	sort.Strings(sortedIsland)

	var chunks [][]string
	for i := 0; i < n; i += chunkSize {
		end := i + chunkSize
		if end > n {
			end = n
		}
		chunks = append(chunks, sortedIsland[i:end])
	}

	start := time.Now()
	results := make([]string, len(chunks))
	var wg sync.WaitGroup

	if totalCores <= 0 {
		totalCores = 1
	}

	// Time budget: 60% initial pass, 15% remix pass, 25% reduce phase.
	initialWallClock := time.Duration(float64(totalWallClock) * 0.60)
	remixWallClock := time.Duration(float64(totalWallClock) * 0.15)

	rounds := float64(len(chunks)) / float64(totalCores)
	if rounds < 1 {
		rounds = 1
	}
	chunkWallClock := time.Duration(float64(initialWallClock) / rounds)

	chunkCores := totalCores / len(chunks)
	if chunkCores < 1 {
		chunkCores = 1
	}

	activeWorkers := totalCores
	if len(chunks) < activeWorkers {
		activeWorkers = len(chunks)
	}

	jobChan := make(chan int, len(chunks))
	for i := range chunks {
		jobChan <- i
	}
	close(jobChan)

	for w := 0; w < activeWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobChan {
				if chunkWallClock >= 10*time.Millisecond {
					results[idx] = SolveGA(chunks[idx], minOverlap, chunkWallClock, chunkCores, cfg, logger)
				} else {
					var dl time.Time
					if chunkWallClock > 0 {
						dl = time.Now().Add(chunkWallClock)
					}
					results[idx] = SolveGreedyHeapWithDeadline(chunks[idx], minOverlap, dl)
				}
			}
		}()
	}
	wg.Wait()

	// --- Chunk Remixing ---
	// Evaluate compression ratio per chunk and remix underperformers.
	if len(chunks) >= 3 {
		type crEntry struct {
			idx int
			cr  float64
		}
		entries := make([]crEntry, len(chunks))
		for i, chunk := range chunks {
			rawLen := 0
			for _, s := range chunk {
				rawLen += len(s)
			}
			cr := 1.0
			if rawLen > 0 {
				cr = float64(len(results[i])) / float64(rawLen)
			}
			entries[i] = crEntry{idx: i, cr: cr}
		}

		// Sort by CR descending to find worst performers.
		sort.Slice(entries, func(a, b int) bool {
			return entries[a].cr > entries[b].cr
		})

		// Worst 20%, but only those with CR > 0.85.
		maxBad := len(entries) / 5
		if maxBad < 1 {
			maxBad = 1
		}
		var badIdxs []int
		for i := 0; i < maxBad && i < len(entries); i++ {
			if entries[i].cr > 0.85 {
				badIdxs = append(badIdxs, entries[i].idx)
			}
		}

		if len(badIdxs) > 0 && remixWallClock >= 10*time.Millisecond {
			// Dissolve bad chunks back into a remix pool.
			var remixPool []string
			for _, bi := range badIdxs {
				remixPool = append(remixPool, chunks[bi]...)
			}

			logger.LogRemix(len(badIdxs), len(remixPool))

			// Shuffle to break alphabetical bias.
			rand.Shuffle(len(remixPool), func(i, j int) {
				remixPool[i], remixPool[j] = remixPool[j], remixPool[i]
			})

			// Re-chunk the remix pool.
			var remixChunks [][]string
			for i := 0; i < len(remixPool); i += chunkSize {
				end := i + chunkSize
				if end > len(remixPool) {
					end = len(remixPool)
				}
				remixChunks = append(remixChunks, remixPool[i:end])
			}

			// Solve remix chunks with the reserved time budget.
			remixRounds := float64(len(remixChunks)) / float64(totalCores)
			if remixRounds < 1 {
				remixRounds = 1
			}
			remixChunkWallClock := time.Duration(float64(remixWallClock) / remixRounds)

			remixResults := make([]string, len(remixChunks))
			remixJobs := make(chan int, len(remixChunks))
			for i := range remixChunks {
				remixJobs <- i
			}
			close(remixJobs)

			remixWorkers := totalCores
			if len(remixChunks) < remixWorkers {
				remixWorkers = len(remixChunks)
			}

			var remixWg sync.WaitGroup
			for w := 0; w < remixWorkers; w++ {
				remixWg.Add(1)
				go func() {
					defer remixWg.Done()
					for idx := range remixJobs {
						if remixChunkWallClock >= 10*time.Millisecond {
							remixResults[idx] = SolveGA(remixChunks[idx], minOverlap, remixChunkWallClock, chunkCores, cfg, logger)
						} else {
							var dl time.Time
							if remixChunkWallClock > 0 {
								dl = time.Now().Add(remixChunkWallClock)
							}
							remixResults[idx] = SolveGreedyHeapWithDeadline(remixChunks[idx], minOverlap, dl)
						}
					}
				}()
			}
			remixWg.Wait()

			// Compare: only replace if remix yields shorter total length.
			oldLen := 0
			for _, bi := range badIdxs {
				oldLen += len(results[bi])
			}
			newLen := 0
			for _, r := range remixResults {
				newLen += len(r)
			}

			if newLen < oldLen {
				// Replace bad chunk results with remixed results.
				// Remove bad indices from results, append remix results.
				badSet := make(map[int]bool, len(badIdxs))
				for _, bi := range badIdxs {
					badSet[bi] = true
				}
				var keptResults []string
				for i, r := range results {
					if !badSet[i] {
						keptResults = append(keptResults, r)
					}
				}
				results = append(keptResults, remixResults...)
			}
		}
	}

	elapsed := time.Since(start)
	nextWallClock := totalWallClock - elapsed
	if nextWallClock < 10*time.Millisecond {
		nextWallClock = 10 * time.Millisecond
	}

	return SolveHierarchicalGA(results, minOverlap, chunkSize, nextWallClock, totalCores, cfg, logger)
}

// AssembleConcurrently processes all islands using a worker pool pattern.
// Small islands (len <= dpLimit) are solved with exact DP, large ones with
// hierarchical sort+slice chunking for full CPU saturation.
// If cfg.GATime > 0, eligible large islands are routed to the GA solver
// via SolveHierarchicalGA, which chunks them into safe 20000-element blocks.
// Jobs are sorted descending by island size to prevent tail stalling.
func AssembleConcurrently(islands [][]string, cfg *cli.BuildConfig) []string {
	n := len(islands)
	if n == 0 {
		return nil
	}

	numWorkers := runtime.NumCPU()

	// Calculate GA budgets if enabled.
	var budgets []time.Duration
	if cfg.GATime > 0 {
		budgets = calculateIslandBudgets(cfg.GATime, numWorkers, islands, cfg.DPLimit)
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
	var printMu sync.Mutex

	// Create GA logger (nil-safe: no-ops if verbose is false).
	gaLogger := NewGALogger(&printMu, cfg.Verbose)

	// Spin up workers.
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				island := islands[idx]
				if len(island) <= cfg.DPLimit {
					results[idx] = SolveExactDP(island, cfg.MinOverlap)
				} else if budgets != nil && budgets[idx] >= 10*time.Millisecond {
					wallClock := cfg.GATime
					if wallClock <= 0 {
						wallClock = budgets[idx]
					}
					cores := 1
					if wallClock > 0 && budgets[idx] >= wallClock {
						cores = int(budgets[idx] / wallClock)
					}
					if cores < 1 {
						cores = 1
					}

					if cfg.Verbose {
						printMu.Lock()
						if cores > 1 {
							fmt.Printf("\r\033[K  GA: island %d (%d strings), wall-clock %v (%d cores)\n",
								idx, len(island), wallClock, cores)
						} else {
							fmt.Printf("\r\033[K  GA: island %d (%d strings), budget %v\n",
								idx, len(island), budgets[idx])
						}
						printMu.Unlock()
					}
					// Divert massive workloads into partitioned fraction-budgeted recursive chunking.
					results[idx] = SolveHierarchicalGA(island, cfg.MinOverlap, 20000, wallClock, cores, cfg, gaLogger)
				} else {
					// Route giant islands into hierarchical sort+slice chunker.
					results[idx] = SolveHierarchicalGreedy(island, cfg.MinOverlap, 20000)
				}
				if cfg.Verbose {
					c := atomic.AddInt32(&completedStrings, int32(len(island)))
					printMu.Lock()
					fmt.Printf("\r\033[K  Assembling... %d/%d strings (%d%%)",
						c, totalStrings, int(c)*100/totalStrings)
					printMu.Unlock()
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

	if cfg.Verbose && n > 0 {
		fmt.Println()
	}

	return results
}
