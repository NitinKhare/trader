// Package portfolio - monitor.go tracks rolling performance metrics per strategy
// and computes allocation weights based on recent Sharpe ratios.
package portfolio

import (
	"math"
	"sort"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/storage"
)

// PerformanceMonitor tracks strategy performance over a rolling window of trades.
type PerformanceMonitor struct {
	windowSize int
}

// NewPerformanceMonitor creates a PerformanceMonitor with the given rolling window size.
// Default window size is 30 if a non-positive value is provided.
func NewPerformanceMonitor(windowSize int) *PerformanceMonitor {
	if windowSize <= 0 {
		windowSize = 30
	}
	return &PerformanceMonitor{
		windowSize: windowSize,
	}
}

// Update groups closed trades by strategy and computes rolling Sharpe ratio and
// win rate over the last windowSize trades for each strategy.
func (pm *PerformanceMonitor) Update(closedTrades []storage.TradeRecord) map[string]*StrategyPerformanceSnapshot {
	snapshots := make(map[string]*StrategyPerformanceSnapshot)

	if len(closedTrades) == 0 {
		return snapshots
	}

	// Group trades by strategy.
	byStrategy := make(map[string][]storage.TradeRecord)
	for _, t := range closedTrades {
		byStrategy[t.StrategyID] = append(byStrategy[t.StrategyID], t)
	}

	for stratID, trades := range byStrategy {
		// Sort by exit time (most recent last).
		sort.Slice(trades, func(i, j int) bool {
			ei := monitorExitTime(trades[i])
			ej := monitorExitTime(trades[j])
			return ei.Before(ej)
		})

		// Take the last windowSize trades.
		window := trades
		if len(window) > pm.windowSize {
			window = window[len(window)-pm.windowSize:]
		}

		// Compute PnLs for the window.
		pnls := make([]float64, len(window))
		var wins int
		for i, t := range window {
			pnls[i] = t.PnL
			if t.PnL > 0 {
				wins++
			}
		}

		snap := &StrategyPerformanceSnapshot{
			StrategyID:    stratID,
			RollingSharpe: computeRollingSharpe(pnls),
			RecentWinRate: 0,
			TradeCount:    len(trades), // total trades, not just window
		}

		if len(window) > 0 {
			snap.RecentWinRate = float64(wins) / float64(len(window)) * 100
		}

		snapshots[stratID] = snap
	}

	return snapshots
}

// GetAllocationWeights returns normalized allocation weights based on Sharpe ratios.
//
// Rules:
//   - Strategies with Sharpe <= 0 get weight 0.
//   - Strategies with fewer than 5 total trades get equal weight.
//   - Remaining strategies get weight proportional to their Sharpe ratio.
//   - All weights are normalized to sum to 1.
func (pm *PerformanceMonitor) GetAllocationWeights(snapshots map[string]*StrategyPerformanceSnapshot) map[string]float64 {
	weights := make(map[string]float64)

	if len(snapshots) == 0 {
		return weights
	}

	// Separate into categories.
	var insufficientIDs []string
	var positiveSharpeIDs []string
	var totalPositiveSharpe float64

	for id, snap := range snapshots {
		if snap.TradeCount < 5 {
			insufficientIDs = append(insufficientIDs, id)
			continue
		}
		if snap.RollingSharpe <= 0 {
			weights[id] = 0
			continue
		}
		positiveSharpeIDs = append(positiveSharpeIDs, id)
		totalPositiveSharpe += snap.RollingSharpe
	}

	// Count participants (strategies that get non-zero weight).
	participants := len(insufficientIDs) + len(positiveSharpeIDs)
	if participants == 0 {
		// All strategies have negative Sharpe; give equal weight to all.
		equalWeight := 1.0 / float64(len(snapshots))
		for id := range snapshots {
			weights[id] = equalWeight
		}
		return weights
	}

	equalWeight := 1.0 / float64(participants)

	// Assign equal weight to insufficient-data strategies.
	for _, id := range insufficientIDs {
		weights[id] = equalWeight
	}

	// Assign Sharpe-proportional weight to positive Sharpe strategies.
	sharpePoolWeight := float64(len(positiveSharpeIDs)) * equalWeight
	for _, id := range positiveSharpeIDs {
		if totalPositiveSharpe > 0 {
			weights[id] = (snapshots[id].RollingSharpe / totalPositiveSharpe) * sharpePoolWeight
		}
	}

	// Normalize to sum to 1.
	var totalWeight float64
	for _, w := range weights {
		totalWeight += w
	}
	if totalWeight > 0 {
		for id := range weights {
			weights[id] /= totalWeight
		}
	}

	return weights
}

// computeRollingSharpe calculates annualized Sharpe ratio from a slice of PnL values.
func computeRollingSharpe(pnls []float64) float64 {
	if len(pnls) < 2 {
		return 0
	}

	var sum float64
	for _, p := range pnls {
		sum += p
	}
	mean := sum / float64(len(pnls))

	var variance float64
	for _, p := range pnls {
		diff := p - mean
		variance += diff * diff
	}
	variance /= float64(len(pnls) - 1)
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return 0
	}

	return (mean / stdDev) * math.Sqrt(252)
}

// monitorExitTime safely extracts the exit time from a trade record.
func monitorExitTime(t storage.TradeRecord) time.Time {
	if t.ExitTime != nil {
		return *t.ExitTime
	}
	return t.EntryTime
}
