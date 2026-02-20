// Package portfolio provides capital allocation, correlation management,
// and performance monitoring across multiple trading strategies.
package portfolio

// AllocationMode determines how capital is distributed across strategies.
type AllocationMode string

const (
	// AllocationEqual distributes capital equally among all strategies (1/N).
	AllocationEqual AllocationMode = "EQUAL"
	// AllocationSharpe weights capital allocation by each strategy's rolling Sharpe ratio.
	AllocationSharpe AllocationMode = "SHARPE_WEIGHTED"
)

// StrategyPerformanceSnapshot holds a point-in-time performance snapshot for a strategy.
type StrategyPerformanceSnapshot struct {
	StrategyID    string
	RollingSharpe float64
	RecentWinRate float64
	TradeCount    int
}

// Allocation represents the capital allocated to a single strategy.
type Allocation struct {
	StrategyID string
	CapitalPct float64
	Capital    float64
	Reason     string
}

// Allocator manages capital distribution across strategies.
type Allocator struct {
	mode         AllocationMode
	totalCapital float64
}

// NewAllocator creates a new Allocator with the given total capital and allocation mode.
func NewAllocator(totalCapital float64, mode AllocationMode) *Allocator {
	return &Allocator{
		mode:         mode,
		totalCapital: totalCapital,
	}
}

// Allocate distributes capital across the given strategies based on their performance.
//
// For AllocationEqual: each strategy gets 1/N of the capital.
//
// For AllocationSharpe:
//   - Strategies with Sharpe <= 0 get 0 allocation.
//   - Strategies with fewer than 5 trades get equal weight (insufficient data).
//   - Remaining strategies are weighted proportionally to their Sharpe ratio.
//   - Weights are normalized so they sum to 1.
func (a *Allocator) Allocate(strategyIDs []string, perf map[string]*StrategyPerformanceSnapshot) []Allocation {
	if len(strategyIDs) == 0 {
		return nil
	}

	allocations := make([]Allocation, 0, len(strategyIDs))

	switch a.mode {
	case AllocationSharpe:
		allocations = a.allocateSharpeWeighted(strategyIDs, perf)
	default:
		// Default to equal allocation.
		allocations = a.allocateEqual(strategyIDs)
	}

	return allocations
}

// allocateEqual distributes capital equally among all strategies.
func (a *Allocator) allocateEqual(strategyIDs []string) []Allocation {
	n := len(strategyIDs)
	pct := 1.0 / float64(n)
	capital := a.totalCapital * pct

	allocations := make([]Allocation, n)
	for i, id := range strategyIDs {
		allocations[i] = Allocation{
			StrategyID: id,
			CapitalPct: pct,
			Capital:    capital,
			Reason:     "equal_weight",
		}
	}
	return allocations
}

// allocateSharpeWeighted distributes capital weighted by Sharpe ratio.
func (a *Allocator) allocateSharpeWeighted(strategyIDs []string, perf map[string]*StrategyPerformanceSnapshot) []Allocation {
	n := len(strategyIDs)

	// Separate strategies into those with sufficient data and those without.
	type stratWeight struct {
		id     string
		weight float64
		reason string
	}
	weights := make([]stratWeight, 0, n)

	// Count strategies with insufficient data for equal weight fallback.
	var insufficientCount int
	var sufficientSharpeSum float64
	sufficientStrategies := make([]stratWeight, 0)

	for _, id := range strategyIDs {
		snap, ok := perf[id]
		if !ok || snap.TradeCount < 5 {
			// Insufficient data: will get equal weight.
			insufficientCount++
			continue
		}
		if snap.RollingSharpe <= 0 {
			// Negative or zero Sharpe: no allocation.
			weights = append(weights, stratWeight{id: id, weight: 0, reason: "sharpe_negative"})
			continue
		}
		sufficientSharpeSum += snap.RollingSharpe
		sufficientStrategies = append(sufficientStrategies, stratWeight{id: id, weight: snap.RollingSharpe})
	}

	// If no strategies have positive Sharpe and sufficient data, fall back to equal.
	totalParticipants := insufficientCount + len(sufficientStrategies)
	if totalParticipants == 0 {
		// All strategies have negative Sharpe; give equal weight to all.
		return a.allocateEqual(strategyIDs)
	}

	// Compute equal weight for insufficient-data strategies.
	equalWeight := 0.0
	if totalParticipants > 0 {
		equalWeight = 1.0 / float64(totalParticipants)
	}

	// Add insufficient-data strategies with equal weight.
	for _, id := range strategyIDs {
		snap, ok := perf[id]
		if !ok || snap.TradeCount < 5 {
			weights = append(weights, stratWeight{id: id, weight: equalWeight, reason: "insufficient_trades_equal_weight"})
		}
	}

	// Normalize Sharpe-weighted strategies.
	// Their total weight should equal len(sufficientStrategies) * equalWeight
	// so that each group is proportionally fair.
	sharpePoolWeight := float64(len(sufficientStrategies)) * equalWeight
	for _, ss := range sufficientStrategies {
		normalizedWeight := 0.0
		if sufficientSharpeSum > 0 {
			normalizedWeight = (ss.weight / sufficientSharpeSum) * sharpePoolWeight
		}
		weights = append(weights, stratWeight{id: ss.id, weight: normalizedWeight, reason: "sharpe_weighted"})
	}

	// Normalize all weights to sum to 1.
	var totalWeight float64
	for _, w := range weights {
		totalWeight += w.weight
	}

	allocations := make([]Allocation, len(weights))
	for i, w := range weights {
		pct := 0.0
		if totalWeight > 0 {
			pct = w.weight / totalWeight
		}
		allocations[i] = Allocation{
			StrategyID: w.id,
			CapitalPct: pct,
			Capital:    a.totalCapital * pct,
			Reason:     w.reason,
		}
	}

	return allocations
}

// UpdateCapital updates the total capital available for allocation.
func (a *Allocator) UpdateCapital(newCapital float64) {
	a.totalCapital = newCapital
}
