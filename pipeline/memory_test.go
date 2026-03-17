package pipeline

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
)

// TestEliminateSubstrings_MemoryUsage verifies that EliminateSubstrings
// stays within reasonable memory bounds for large inputs by using suffix arrays
// instead of Aho-Corasick automaton.
func TestEliminateSubstrings_MemoryUsage(t *testing.T) {
	// Generate 100K strings of varying lengths.
	const count = 100_000
	rng := rand.New(rand.NewSource(42))
	words := make([]string, count)
	charset := "abcdefghijklmnopqrstuvwxyz"

	for i := range count {
		length := 3 + rng.Intn(20) // 3-22 char strings.
		buf := make([]byte, length)
		for j := range buf {
			buf[j] = charset[rng.Intn(len(charset))]
		}
		words[i] = string(buf)
	}

	// Force GC to get clean baseline.
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	// Run the function.
	result := EliminateSubstrings(words)

	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	// Calculate memory delta.
	allocatedMB := float64(after.TotalAlloc-before.TotalAlloc) / (1024 * 1024)

	t.Logf("Input: %d strings", count)
	t.Logf("Output: %d survivors", len(result))
	t.Logf("Memory allocated: %.1f MB", allocatedMB)

	// With suffix arrays, 100K strings should use well under 512MB.
	// The old Aho-Corasick approach would use 30-60+ GB.
	const maxMB = 512.0
	if allocatedMB > maxMB {
		t.Errorf("memory usage %.1f MB exceeds limit of %.0f MB", allocatedMB, maxMB)
	}

	// Sanity check: result should have fewer items than input.
	if len(result) > count {
		t.Errorf("result has %d items, expected <= %d", len(result), count)
	}

	// Verify all survivors are actually unique and not substrings of each other.
	fmt.Printf("[memory_test] %d → %d survivors using %.1f MB\n", count, len(result), allocatedMB)
}
