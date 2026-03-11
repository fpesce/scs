package pipeline

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joke/scs/cli"
	"github.com/joke/scs/graph"
)

// Chromosome represents a single individual in the GA population.
// path is a permutation of indices into the island slice.
// fitness is the total suffix-prefix overlap score.
type Chromosome struct {
	path    []int
	fitness int
}

// lazyCache is a per-goroutine JIT overlap cache.
// No mutex needed — it runs in a single worker goroutine.
type lazyCache struct {
	island []string
	cache  map[uint64]int
}

func newLazyCache(island []string) *lazyCache {
	return &lazyCache{
		island: island,
		cache:  make(map[uint64]int),
	}
}

// get returns the overlap between island[leftID] and island[rightID],
// computing and caching it on first access (JIT).
func (c *lazyCache) get(leftID, rightID int) int {
	key := (uint64(leftID) << 32) | uint64(rightID)
	if v, ok := c.cache[key]; ok {
		return v
	}
	v := graph.CalculateMaxOverlap(c.island[leftID], c.island[rightID])
	c.cache[key] = v
	return v
}

// evaluateFitness computes the total overlap of a path by summing
// adjacent pair overlaps via the lazy cache.
// If a deadline is set, evaluation stops early when time expires.
func evaluateFitness(path []int, cache *lazyCache, deadline time.Time) int {
	total := 0
	for i := 0; i < len(path)-1; i++ {
		if i&127 == 0 && !deadline.IsZero() && time.Now().After(deadline) {
			break
		}
		total += cache.get(path[i], path[i+1])
	}
	return total
}

// --- Mutations (zero-allocation, in-place) ---

// mutateSwap picks two random distinct indices and swaps their values.
func mutateSwap(path []int, rng *rand.Rand) {
	n := len(path)
	if n < 2 {
		return
	}
	i := rng.Intn(n)
	j := rng.Intn(n - 1)
	if j >= i {
		j++
	}
	path[i], path[j] = path[j], path[i]
}

// mutateFlip picks two random distinct indices and reverses the sub-slice.
func mutateFlip(path []int, rng *rand.Rand) {
	n := len(path)
	if n < 2 {
		return
	}
	i := rng.Intn(n)
	j := rng.Intn(n - 1)
	if j >= i {
		j++
	}
	if i > j {
		i, j = j, i
	}
	for i < j {
		path[i], path[j] = path[j], path[i]
		i++
		j--
	}
}

// mutateScramble picks two random distinct indices and shuffles the sub-slice.
func mutateScramble(path []int, rng *rand.Rand) {
	n := len(path)
	if n < 2 {
		return
	}
	i := rng.Intn(n)
	j := rng.Intn(n - 1)
	if j >= i {
		j++
	}
	if i > j {
		i, j = j, i
	}
	sub := path[i : j+1]
	rng.Shuffle(len(sub), func(a, b int) {
		sub[a], sub[b] = sub[b], sub[a]
	})
}

// --- Sequential Constructive Crossover (SCX) ---

// crossoverSCX produces an offspring permutation using SCX.
// offspring, visited, posA, and posB are pre-allocated by the caller
// (zero-allocation loop — avoids expensive map allocs per crossover).
func crossoverSCX(offspring, parentA, parentB []int, cache *lazyCache, visited []bool, rng *rand.Rand, posA, posB []int) {
	n := len(parentA)

	// Clear visited array.
	for i := 0; i < n; i++ {
		visited[i] = false
	}

	// Build position lookup for each parent using pre-allocated slices.
	for i, v := range parentA {
		posA[v] = i
	}
	for i, v := range parentB {
		posB[v] = i
	}

	// Start with first node of Parent A.
	current := parentA[0]
	offspring[0] = current
	visited[current] = true

	for pos := 1; pos < n; pos++ {
		// Find candidates: next unvisited node in each parent.
		candA := -1
		candB := -1

		idxA := posA[current]
		if idxA+1 < n {
			c := parentA[idxA+1]
			if !visited[c] {
				candA = c
			}
		}

		idxB := posB[current]
		if idxB+1 < n {
			c := parentB[idxB+1]
			if !visited[c] {
				candB = c
			}
		}

		var next int
		switch {
		case candA < 0 && candB < 0:
			// Both exhausted or visited — pick random unvisited.
			next = randomUnvisited(visited, n, rng)
		case candA < 0:
			next = candB
		case candB < 0:
			next = candA
		default:
			// Both available — pick higher overlap.
			ovA := cache.get(current, candA)
			ovB := cache.get(current, candB)
			if ovA > ovB {
				next = candA
			} else if ovB > ovA {
				next = candB
			} else {
				// Tied — pick randomly.
				if rng.Intn(2) == 0 {
					next = candA
				} else {
					next = candB
				}
			}
		}

		offspring[pos] = next
		visited[next] = true
		current = next
	}
}

// randomUnvisited returns a random unvisited node index.
// Uses random probing first for speed, then falls back to linear scan.
func randomUnvisited(visited []bool, n int, rng *rand.Rand) int {
	// Fast path: try a few random probes.
	for i := 0; i < 5; i++ {
		idx := rng.Intn(n)
		if !visited[idx] {
			return idx
		}
	}
	// Slow path: linear scan from a random start.
	start := rng.Intn(n)
	for i := start; i < n; i++ {
		if !visited[i] {
			return i
		}
	}
	for i := 0; i < start; i++ {
		if !visited[i] {
			return i
		}
	}
	return 0
}

// --- Population Initialization ---

// initPopulation creates the initial GA population with a mixed approach:
// 1 elite baseline, ~80% mutated clones, ~20% random permutations.
// greedyPath is a pre-computed baseline permutation shared across all workers.
// If the deadline expires, returns the population assembled so far.
func initPopulation(popSize int, island []string, greedyPath []int, cache *lazyCache, rng *rand.Rand, deadline time.Time) []Chromosome {
	n := len(island)
	pop := make([]Chromosome, 0, popSize)

	// Individual 0: Elite baseline from pre-computed greedy path.
	c0 := Chromosome{path: make([]int, n)}
	copy(c0.path, greedyPath)
	c0.fitness = evaluateFitness(c0.path, cache, deadline)
	pop = append(pop, c0)

	// ~80% mutated clones of elite.
	eliteEnd := 1 + (popSize-1)*80/100
	for i := 1; i < eliteEnd && i < popSize; i++ {
		if !deadline.IsZero() && time.Now().After(deadline) {
			return pop
		}
		c := Chromosome{path: make([]int, n)}
		copy(c.path, pop[0].path)
		// Apply 1-3 random mutations.
		numMutations := 1 + rng.Intn(3)
		for m := 0; m < numMutations; m++ {
			switch rng.Intn(3) {
			case 0:
				mutateSwap(c.path, rng)
			case 1:
				mutateFlip(c.path, rng)
			case 2:
				mutateScramble(c.path, rng)
			}
		}
		c.fitness = evaluateFitness(c.path, cache, deadline)
		pop = append(pop, c)
	}

	// ~20% pure random permutations.
	for i := len(pop); i < popSize; i++ {
		if !deadline.IsZero() && time.Now().After(deadline) {
			return pop
		}
		c := Chromosome{path: make([]int, n)}
		perm := rng.Perm(n)
		copy(c.path, perm)
		c.fitness = evaluateFitness(c.path, cache, deadline)
		pop = append(pop, c)
	}

	return pop
}

// --- Evolutionary Loop & String Reconstruction ---

// clamp returns v clamped to [lo, hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// tournamentSelect picks the best individual from a random subset.
func tournamentSelect(pop []Chromosome, tourneySize int, rng *rand.Rand) int {
	best := rng.Intn(len(pop))
	for i := 1; i < tourneySize; i++ {
		cand := rng.Intn(len(pop))
		if pop[cand].fitness > pop[best].fitness {
			best = cand
		}
	}
	return best
}

// SolveGA runs a time-bounded steady-state genetic algorithm on an island.
// SolveHierarchicalGA guarantees N <= 20000, so memory is bounded.
// concurrency controls the number of independent island-model populations.
// logger may be nil to suppress all GA telemetry output.
func SolveGA(island []string, minOverlap int, wallClock time.Duration, concurrency int, cfg *cli.BuildConfig, logger *GALogger) (result string) {
	n := len(island)
	if n <= 1 {
		if n == 1 {
			return island[0]
		}
		return ""
	}

	// Panic recovery: fall back to greedy on any unexpected error.
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "WARNING: GA panicked: %v — falling back to greedy\n", r)
			result = SolveGreedyHeapWithDeadline(island, minOverlap, time.Time{})
		}
	}()

	start := time.Now()

	// Micro-budget bypass: if wall-clock < 10ms, return greedy baseline.
	if wallClock < 10*time.Millisecond {
		return SolveGreedyHeapWithDeadline(island, minOverlap, time.Time{})
	}

	// Dynamic hyperparameters: use CLI overrides or scale from N.
	popSize := cfg.GAPop
	if popSize <= 0 {
		popSize = clamp(n/10, 50, 500)
	}
	tourneySize := cfg.GATourney
	if tourneySize <= 0 {
		tourneySize = 3
	}
	stagnationLimit := cfg.GAStag
	if stagnationLimit <= 0 {
		stagnationLimit = n * 2
	}
	if concurrency < 1 {
		concurrency = 1
	}

	deadline := start.Add(wallClock)

	// Compute greedy baseline ONCE — shared across all island-model workers.
	greedyPath, _ := solveGreedyPathWithDeadline(island, minOverlap, deadline)

	// Island Model: fan out to `concurrency` independent GA populations.
	type gaResult struct {
		path    []int
		fitness int
		cache   *lazyCache
	}

	// Cross-worker global best for logging — only truly new records get reported.
	var globalBestAtomic atomic.Int64
	globalBestAtomic.Store(-1)

	resChan := make(chan gaResult, concurrency)
	var wg sync.WaitGroup

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			defer func() {
				_ = recover() // Catch panics locally so they don't crash the build.
			}()

			// Unique PRNG seed per worker to explore different genetic paths.
			seed := time.Now().UnixNano() + int64(workerID)*1e9
			rng := rand.New(rand.NewSource(seed))

			cache := newLazyCache(island)
			pop := initPopulation(popSize, island, greedyPath, cache, rng, deadline)
			if len(pop) == 0 {
				return
			}

			globalBestPath := make([]int, n)
			globalBestFitness := -1
			for i := range pop {
				if pop[i].fitness > globalBestFitness {
					globalBestFitness = pop[i].fitness
					copy(globalBestPath, pop[i].path)
				}
			}

			if len(pop) < 2 {
				resPath := make([]int, n)
				copy(resPath, globalBestPath)
				resChan <- gaResult{
					path:    resPath,
					fitness: globalBestFitness,
					cache:   cache,
				}
				return
			}

			offspringPath := make([]int, n)
			visited := make([]bool, n)
			posA := make([]int, n)
			posB := make([]int, n)
			stagnation := 0

			for iter := 0; ; iter++ {
				// Syscall optimization: check time only every 128 iterations.
				if iter&127 == 0 {
					if time.Now().After(deadline) {
						break
					}
				}

				// Tournament selection.
				idxA := tournamentSelect(pop, tourneySize, rng)
				idxB := tournamentSelect(pop, tourneySize, rng)

				var offspringFitness int

				if stagnation < stagnationLimit {
					// Standard mode: SCX + 10% mutation.
					crossoverSCX(offspringPath, pop[idxA].path, pop[idxB].path, cache, visited, rng, posA, posB)
					if rng.Intn(10) == 0 {
						switch rng.Intn(2) {
						case 0:
							mutateSwap(offspringPath, rng)
						case 1:
							mutateFlip(offspringPath, rng)
						}
					}
				} else {
					// Panic mode: aggressive mutations to break local optimum.
					copy(offspringPath, pop[idxA].path)
					mutateScramble(offspringPath, rng)
					mutateFlip(offspringPath, rng)
					mutateSwap(offspringPath, rng)
					logger.LogStagnation(n, stagnation)
				}

				offspringFitness = evaluateFitness(offspringPath, cache, deadline)

				// Zero-allocation replacement: find worst individual and overwrite.
				worstIdx := 0
				for i := 1; i < len(pop); i++ {
					if pop[i].fitness < pop[worstIdx].fitness {
						worstIdx = i
					}
				}
				copy(pop[worstIdx].path, offspringPath)
				pop[worstIdx].fitness = offspringFitness

				// Global best tracking.
				if offspringFitness > globalBestFitness {
					globalBestFitness = offspringFitness
					copy(globalBestPath, offspringPath)
					stagnation = 0
					// Log only if this is a new cross-worker global record.
					if int64(offspringFitness) > globalBestAtomic.Load() {
						globalBestAtomic.Store(int64(offspringFitness))
						logger.LogOptimum(n, offspringFitness)
					}
				} else {
					stagnation++
				}
			}

			resPath := make([]int, n)
			copy(resPath, globalBestPath)
			resChan <- gaResult{
				path:    resPath,
				fitness: globalBestFitness,
				cache:   cache,
			}
		}(w)
	}

	wg.Wait()
	close(resChan)

	// Pick the best result across all island populations.
	var overallBest gaResult
	overallBest.fitness = -1
	for res := range resChan {
		if res.fitness > overallBest.fitness {
			overallBest = res
		}
	}

	if overallBest.fitness == -1 {
		return SolveGreedyHeapWithDeadline(island, minOverlap, time.Time{})
	}

	// String reconstruction from the winning path.
	bestPath := overallBest.path
	cache := overallBest.cache

	var builder strings.Builder
	totalLen := len(island[bestPath[0]])
	for i := 0; i < n-1; i++ {
		ov := cache.get(bestPath[i], bestPath[i+1])
		totalLen += len(island[bestPath[i+1]]) - ov
	}
	builder.Grow(totalLen)

	builder.WriteString(island[bestPath[0]])
	for i := 0; i < n-1; i++ {
		ov := cache.get(bestPath[i], bestPath[i+1])
		builder.WriteString(island[bestPath[i+1]][ov:])
	}

	result = builder.String()
	return result
}
