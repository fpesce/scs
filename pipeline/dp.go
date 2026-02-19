package pipeline

import (
	"math"

	"github.com/joke/scs/graph"
)

// SolveExactDP finds the mathematically shortest common superstring encoding
// for a small island using bitmask dynamic programming (equivalent to ATSP).
// Only feasible for islands where len(island) <= dpLimit (typically ~15).
func SolveExactDP(island []string, minOverlap int) string {
	n := len(island)
	if n == 0 {
		return ""
	}
	if n == 1 {
		return island[0]
	}

	// Precompute the N×N overlap matrix using KMP.
	overlap := make([][]int, n)
	for i := 0; i < n; i++ {
		overlap[i] = make([]int, n)
		for j := 0; j < n; j++ {
			if i != j {
				overlap[i][j] = graph.CalculateMaxOverlap(island[i], island[j])
			}
		}
	}

	fullMask := (1 << n) - 1

	// dp[mask][last] = maximum total overlap achievable visiting the subset 'mask'
	// ending at string 'last'.
	dp := make([][]int, fullMask+1)
	parent := make([][]int, fullMask+1)
	for mask := 0; mask <= fullMask; mask++ {
		dp[mask] = make([]int, n)
		parent[mask] = make([]int, n)
		for j := 0; j < n; j++ {
			dp[mask][j] = -1
			parent[mask][j] = -1
		}
	}

	// Base cases: start with each individual string.
	for i := 0; i < n; i++ {
		dp[1<<i][i] = 0
	}

	// Fill DP: try extending every state by adding one more string.
	for mask := 1; mask <= fullMask; mask++ {
		for last := 0; last < n; last++ {
			if dp[mask][last] < 0 {
				continue
			}
			for next := 0; next < n; next++ {
				if mask&(1<<next) != 0 {
					continue
				}
				newMask := mask | (1 << next)
				newOverlap := dp[mask][last] + overlap[last][next]
				if newOverlap > dp[newMask][next] {
					dp[newMask][next] = newOverlap
					parent[newMask][next] = last
				}
			}
		}
	}

	// Find the best ending string in the full mask.
	bestLast := 0
	bestOverlap := math.MinInt
	for i := 0; i < n; i++ {
		if dp[fullMask][i] > bestOverlap {
			bestOverlap = dp[fullMask][i]
			bestLast = i
		}
	}

	// Reconstruct the optimal path.
	path := make([]int, n)
	mask := fullMask
	for i := n - 1; i >= 0; i-- {
		path[i] = bestLast
		prev := parent[mask][bestLast]
		mask ^= (1 << bestLast)
		bestLast = prev
	}

	// Merge along the optimal path.
	result := island[path[0]]
	for i := 1; i < n; i++ {
		prev := path[i-1]
		cur := path[i]
		ov := overlap[prev][cur]
		result += island[cur][ov:]
	}

	return result
}
