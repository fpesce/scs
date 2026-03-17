package pipeline

import (
	"math"
	"testing"
	"time"
)

func TestTimeDistribution(t *testing.T) {
	// Three islands: 50, 100, 200 strings. dpLimit=15, so all are eligible.
	// Weights: 2500, 10000, 40000. Total: 52500.
	// With 1 worker and 60s global time → 60s total CPU.
	// Max budget is 40000/52500*60 ≈ 45.7s which is below 60s cap.
	islands := [][]string{
		make([]string, 50),
		make([]string, 100),
		make([]string, 200),
	}
	global := 60 * time.Second
	budgets := calculateIslandBudgets(global, 1, islands, 15)

	if len(budgets) != 3 {
		t.Fatalf("expected 3 budgets, got %d", len(budgets))
	}

	// Island 200 should get roughly 4× the budget of island 100.
	ratio := float64(budgets[2]) / float64(budgets[1])
	if math.Abs(ratio-4.0) > 0.01 {
		t.Errorf("expected 200-island to get ~4x 100-island budget, got ratio %.3f (budgets: %v, %v)", ratio, budgets[1], budgets[2])
	}

	// Island 100 should get roughly 4× the budget of island 50.
	ratio2 := float64(budgets[1]) / float64(budgets[0])
	if math.Abs(ratio2-4.0) > 0.01 {
		t.Errorf("expected 100-island to get ~4x 50-island budget, got ratio %.3f", ratio2)
	}
}

func TestTimeDistribution_MultiCore(t *testing.T) {
	// Single massive island with 2 workers. Budget = 5s × 2 = 10s total CPU.
	// With no cap, the sole eligible island gets the full 10s.
	islands := [][]string{make([]string, 1000)}
	global := 5 * time.Second
	budgets := calculateIslandBudgets(global, 2, islands, 15)

	expected := 10 * time.Second
	if budgets[0] != expected {
		t.Errorf("budget = %v, want %v (full multi-core CPU capacity)", budgets[0], expected)
	}
}

func TestTimeDistribution_Ineligible(t *testing.T) {
	// Islands smaller than dpLimit should get 0 budget.
	islands := [][]string{
		make([]string, 10), // below dpLimit=15
		make([]string, 50), // above
	}
	budgets := calculateIslandBudgets(10*time.Second, 4, islands, 15)

	if budgets[0] != 0 {
		t.Errorf("ineligible island got budget %v, want 0", budgets[0])
	}
	if budgets[1] <= 0 {
		t.Errorf("eligible island got zero budget")
	}
}

func TestTimeDistribution_ZeroTime(t *testing.T) {
	islands := [][]string{make([]string, 50)}
	budgets := calculateIslandBudgets(0, 4, islands, 15)
	if budgets[0] != 0 {
		t.Errorf("expected 0 budget for zero global time, got %v", budgets[0])
	}
}
