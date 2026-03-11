package pipeline

import "time"

// calculateIslandBudgets distributes a global GA wall-clock time budget across
// islands using quadratic weighting based on combinatorial difficulty.
// Only islands with len > dpLimit are eligible for GA time.
// Returns raw CPU time budgets — a single island may receive more than
// globalTime if multiple cores are available (Island Model parallelism).
func calculateIslandBudgets(globalTime time.Duration, numWorkers int, islands [][]string, dpLimit int) []time.Duration {
	n := len(islands)
	budgets := make([]time.Duration, n)
	if globalTime <= 0 || n == 0 {
		return budgets
	}

	// Total CPU time available across all worker goroutines.
	totalCPU := globalTime * time.Duration(numWorkers)

	// Compute quadratic weights for eligible islands.
	var wTotal float64
	weights := make([]float64, n)
	for i, island := range islands {
		sz := len(island)
		if sz > dpLimit {
			w := float64(sz) * float64(sz)
			weights[i] = w
			wTotal += w
		}
	}

	if wTotal == 0 {
		return budgets
	}

	// Distribute proportionally — no cap, allowing multi-core budgets.
	for i, w := range weights {
		if w > 0 {
			budgets[i] = time.Duration(float64(totalCPU) * w / wTotal)
		}
	}

	return budgets
}

