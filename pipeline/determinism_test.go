package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"github.com/joke/scs/cli"
)

// TestPipelineDeterminism generates random overlapping strings, runs them
// through the full pipeline 10 times, and asserts the output hash is
// identical every time.
func TestPipelineDeterminism(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	// Generate a random base string and extract overlapping substrings.
	const baseLen = 200
	base := make([]byte, baseLen)
	for i := range base {
		base[i] = byte('a' + rng.Intn(6)) // Small alphabet increases overlaps.
	}

	words := make([]string, 0, 50)
	for range 50 {
		start := rng.Intn(baseLen - 10)
		length := 5 + rng.Intn(20)
		if start+length > baseLen {
			length = baseLen - start
		}
		words = append(words, string(base[start:start+length]))
	}

	// Run the pipeline 10 times and collect hashes.
	const iterations = 10
	hashes := make([]string, 0, iterations)

	for range iterations {
		// Copy words to avoid mutation.
		input := make([]string, len(words))
		copy(input, words)

		survivors := ExactDeduplication(input)
		survivors = EliminateSubstrings(survivors)
		islands := ShatterGraph(survivors, 3)
		superWords := AssembleConcurrently(islands, &cli.BuildConfig{
			DPLimit:    15,
			MinOverlap: 3,
		})

		sort.Slice(superWords, func(i, j int) bool {
			if len(superWords[i]) == len(superWords[j]) {
				return superWords[i] < superWords[j]
			}
			return len(superWords[i]) > len(superWords[j])
		})

		result := strings.Join(superWords, "")
		h := sha256.Sum256([]byte(result))
		hashes = append(hashes, hex.EncodeToString(h[:]))
	}

	// Assert all hashes are identical.
	for i := 1; i < len(hashes); i++ {
		if hashes[i] != hashes[0] {
			t.Fatalf("determinism broken: iteration %d hash %q != iteration 0 hash %q",
				i, hashes[i], hashes[0])
		}
	}
}
