package pipeline

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/joke/scs/cli"
)

func TestLazyCacheKey(t *testing.T) {
	// ATSP asymmetry: Overlap(A,B) != Overlap(B,A)
	island := []string{"abcdef", "defabc"}

	cache := newLazyCache(island)

	fwd := cache.get(0, 1) // suffix "def" of "abcdef" matches prefix "def" of "defabc" → 3
	rev := cache.get(1, 0) // suffix "abc" of "defabc" matches prefix "abc" of "abcdef" → 3

	// Both should be 3 in this case, but they're stored under different keys.
	if fwd != 3 {
		t.Errorf("get(0,1) = %d, want 3", fwd)
	}
	if rev != 3 {
		t.Errorf("get(1,0) = %d, want 3", rev)
	}

	// Verify separate cache entries exist.
	key01 := (uint64(0) << 32) | uint64(1)
	key10 := (uint64(1) << 32) | uint64(0)
	if key01 == key10 {
		t.Errorf("keys should differ for asymmetric pairs")
	}
	if _, ok := cache.cache[key01]; !ok {
		t.Error("cache miss for key (0,1) after get")
	}
	if _, ok := cache.cache[key10]; !ok {
		t.Error("cache miss for key (1,0) after get")
	}
}

func TestLazyCacheAsymmetry(t *testing.T) {
	// Ensure truly asymmetric overlaps are computed correctly.
	island := []string{"hellowor", "worzzzzz"}

	cache := newLazyCache(island)

	fwd := cache.get(0, 1) // suffix "wor" of "hellowor" matches prefix "wor" of "worzzzzz" → 3
	rev := cache.get(1, 0) // suffix "zzzzz" of "worzzzzz" does NOT match prefix "hello" → 0

	if fwd != 3 {
		t.Errorf("get(0,1) = %d, want 3", fwd)
	}
	if rev != 0 {
		t.Errorf("get(1,0) = %d, want 0", rev)
	}
}

func TestEvaluateFitness(t *testing.T) {
	island := []string{"abcdef", "defghi", "ghijkl"}
	cache := newLazyCache(island)

	// Path 0 → 1 → 2: overlap(0,1)=3, overlap(1,2)=3 → fitness=6
	fitness := evaluateFitness([]int{0, 1, 2}, cache, time.Time{})
	if fitness != 6 {
		t.Errorf("evaluateFitness = %d, want 6", fitness)
	}

	// Reversed path 2 → 1 → 0: overlap(2,1)=0, overlap(1,0)=0 → fitness=0
	fitness2 := evaluateFitness([]int{2, 1, 0}, cache, time.Time{})
	if fitness2 != 0 {
		t.Errorf("evaluateFitness reversed = %d, want 0", fitness2)
	}
}

// isValidPermutation checks that path contains exactly 0..n-1.
func isValidPermutation(path []int, n int) bool {
	if len(path) != n {
		return false
	}
	seen := make([]bool, n)
	for _, v := range path {
		if v < 0 || v >= n || seen[v] {
			return false
		}
		seen[v] = true
	}
	return true
}

func TestMutateSwap(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for trial := 0; trial < 200; trial++ {
		n := 20
		path := rng.Perm(n)
		mutateSwap(path, rng)
		if !isValidPermutation(path, n) {
			t.Fatalf("mutateSwap produced invalid permutation: %v", path)
		}
	}
}

func TestMutateInsert(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for trial := 0; trial < 200; trial++ {
		n := 20
		path := rng.Perm(n)
		mutateInsert(path, rng)
		if !isValidPermutation(path, n) {
			t.Fatalf("mutateInsert produced invalid permutation: %v", path)
		}
	}
}

func TestMutateScramble(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for trial := 0; trial < 200; trial++ {
		n := 20
		path := rng.Perm(n)
		mutateScramble(path, rng)
		if !isValidPermutation(path, n) {
			t.Fatalf("mutateScramble produced invalid permutation: %v", path)
		}
	}
}

func TestSCX_ValidPermutation(t *testing.T) {
	island := []string{"abcdef", "defghi", "ghijkl", "jklmno", "mnopqr"}
	n := len(island)
	cache := newLazyCache(island)
	rng := rand.New(rand.NewSource(42))

	offspring := make([]int, n)
	visited := make([]bool, n)
	posA := make([]int, n)
	posB := make([]int, n)

	for trial := 0; trial < 500; trial++ {
		parentA := rng.Perm(n)
		parentB := rng.Perm(n)
		crossoverSCX(offspring, parentA, parentB, cache, visited, rng, posA, posB)
		if !isValidPermutation(offspring, n) {
			t.Fatalf("trial %d: SCX produced invalid permutation: %v (parentA=%v, parentB=%v)",
				trial, offspring, parentA, parentB)
		}
	}
}

func TestInitPopulation(t *testing.T) {
	island := []string{"abcdef", "defghi", "ghijkl", "jklmno", "mnopqr",
		"alpha", "bravo", "charlie", "delta", "echo",
		"foxtrot", "golf", "hotel", "india", "juliet",
		"kilo", "lima", "mike", "november", "oscar"}
	n := len(island)
	cache := newLazyCache(island)
	rng := rand.New(rand.NewSource(42))

	// Pre-compute greedy path (matching SolveGA's new flow).
	greedyPath, _ := solveGreedyPath(island, 3)
	pop := initPopulation(50, island, greedyPath, cache, rng, time.Time{})

	if len(pop) != 50 {
		t.Fatalf("population size = %d, want 50", len(pop))
	}

	for i, ind := range pop {
		if len(ind.path) != n {
			t.Fatalf("individual %d: path length = %d, want %d", i, len(ind.path), n)
		}
		if !isValidPermutation(ind.path, n) {
			t.Fatalf("individual %d: invalid permutation", i)
		}
		if ind.fitness < 0 {
			t.Fatalf("individual %d: negative fitness %d", i, ind.fitness)
		}
	}
}

func TestGAAnytimeBreak(t *testing.T) {
	// Build a large-enough island for GA to run multiple iterations.
	island := make([]string, 50)
	for i := range island {
		// Generate unique strings with some overlap potential.
		island[i] = strings.Repeat(string(rune('a'+i%26)), 5) + strings.Repeat(string(rune('b'+i%26)), 5)
	}

	cfg := &cli.BuildConfig{
		GAPop:     50,
		GATourney: 3,
		GAStag:    100,
	}

	budget := 50 * time.Millisecond
	start := time.Now()
	result := SolveGA(island, 1, budget, 1, cfg, nil)
	elapsed := time.Since(start)

	// Must return a valid string.
	if result == "" {
		t.Fatal("SolveGA returned empty string")
	}

	// All original strings must be substrings of the result.
	for _, s := range island {
		if !strings.Contains(result, s) {
			t.Errorf("result does not contain %q", s)
		}
	}

	// Should finish within ~2x the budget (generous margin for CI).
	if elapsed > 5*budget {
		t.Errorf("SolveGA took %v, expected roughly %v", elapsed, budget)
	}
}
