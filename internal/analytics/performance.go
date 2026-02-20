// Package analytics - performance.go provides extended performance metrics
// beyond the basic PerformanceReport. Includes CAGR, Sortino ratio, Calmar ratio,
// expectancy, R-multiples, payoff ratio, recovery factor, and streak analysis.
package analytics

import (
	"math"
	"sort"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/storage"
)

// ExtendedMetrics holds advanced performance analytics computed from trade records.
type ExtendedMetrics struct {
	CAGR                 float64
	SortinoRatio         float64
	CalmarRatio          float64
	Expectancy           float64
	AvgRMultiple         float64
	PayoffRatio          float64
	RecoveryFactor       float64
	MaxConsecutiveWins   int
	MaxConsecutiveLosses int
}

// AnalyzeExtended computes extended performance metrics from a slice of closed trades.
// initialCapital is the starting equity. startDate and endDate define the evaluation period.
// Returns nil if there are no trades.
func AnalyzeExtended(trades []storage.TradeRecord, initialCapital float64, startDate, endDate time.Time) *ExtendedMetrics {
	if len(trades) == 0 || initialCapital <= 0 {
		return &ExtendedMetrics{}
	}

	// Sort trades by exit time.
	sorted := make([]storage.TradeRecord, len(trades))
	copy(sorted, trades)
	sort.Slice(sorted, func(i, j int) bool {
		ei := exitTimeHelper(sorted[i])
		ej := exitTimeHelper(sorted[j])
		return ei.Before(ej)
	})

	metrics := &ExtendedMetrics{}

	// Collect PnLs.
	pnls := make([]float64, len(sorted))
	var totalPnL float64
	for i, t := range sorted {
		pnls[i] = t.PnL
		totalPnL += t.PnL
	}

	finalEquity := initialCapital + totalPnL

	// --- CAGR ---
	days := endDate.Sub(startDate).Hours() / 24.0
	if days > 0 && finalEquity > 0 && initialCapital > 0 {
		metrics.CAGR = math.Pow(finalEquity/initialCapital, 365.0/days) - 1
	}

	// --- Sortino Ratio ---
	// mean(pnls) / downside_deviation * sqrt(252)
	// downside_deviation = sqrt(mean(min(pnl,0)^2))
	meanPnL := totalPnL / float64(len(pnls))
	var downsideSumSq float64
	for _, pnl := range pnls {
		if pnl < 0 {
			downsideSumSq += pnl * pnl
		}
	}
	downsideDeviation := math.Sqrt(downsideSumSq / float64(len(pnls)))
	if downsideDeviation > 0 {
		metrics.SortinoRatio = (meanPnL / downsideDeviation) * math.Sqrt(252)
	}

	// --- Max Drawdown for CalmarRatio and RecoveryFactor ---
	// Reuse the drawdown calculation logic from analytics.go.
	equity := initialCapital
	peak := equity
	var maxDrawdown float64
	var maxDrawdownPct float64
	for _, pnl := range pnls {
		equity += pnl
		if equity > peak {
			peak = equity
		}
		dd := peak - equity
		if dd > maxDrawdown {
			maxDrawdown = dd
			if peak > 0 {
				maxDrawdownPct = (dd / peak) * 100
			}
		}
	}

	// --- Calmar Ratio ---
	// CAGR / MaxDrawdownPct
	if maxDrawdownPct > 0 {
		metrics.CalmarRatio = metrics.CAGR / (maxDrawdownPct / 100.0)
	}

	// --- Win/Loss stats for Expectancy and PayoffRatio ---
	var wins, losses int
	var totalWinPnL, totalLossPnL float64
	for _, pnl := range pnls {
		if pnl > 0 {
			wins++
			totalWinPnL += pnl
		} else if pnl < 0 {
			losses++
			totalLossPnL += math.Abs(pnl)
		}
	}

	var avgWin, avgLoss float64
	if wins > 0 {
		avgWin = totalWinPnL / float64(wins)
	}
	if losses > 0 {
		avgLoss = totalLossPnL / float64(losses)
	}

	winRate := float64(wins) / float64(len(pnls))
	lossRate := float64(losses) / float64(len(pnls))

	// --- Expectancy ---
	// (winRate * avgWin) - (lossRate * avgLoss)
	metrics.Expectancy = (winRate * avgWin) - (lossRate * avgLoss)

	// --- PayoffRatio ---
	// avgWin / avgLoss
	if avgLoss > 0 {
		metrics.PayoffRatio = avgWin / avgLoss
	}

	// --- AvgRMultiple ---
	// mean(pnl / initialRisk) where initialRisk = entryPrice - stopLoss; skip if stopLoss is 0
	var rMultipleSum float64
	var rMultipleCount int
	for _, t := range sorted {
		if t.StopLoss > 0 {
			initialRisk := math.Abs(t.EntryPrice - t.StopLoss)
			if initialRisk > 0 {
				rMultipleSum += t.PnL / initialRisk
				rMultipleCount++
			}
		}
	}
	if rMultipleCount > 0 {
		metrics.AvgRMultiple = rMultipleSum / float64(rMultipleCount)
	}

	// --- RecoveryFactor ---
	// totalPnL / maxDrawdown
	if maxDrawdown > 0 {
		metrics.RecoveryFactor = totalPnL / maxDrawdown
	}

	// --- MaxConsecutiveWins / MaxConsecutiveLosses ---
	var currentWinStreak, currentLossStreak int
	for _, pnl := range pnls {
		if pnl > 0 {
			currentWinStreak++
			currentLossStreak = 0
			if currentWinStreak > metrics.MaxConsecutiveWins {
				metrics.MaxConsecutiveWins = currentWinStreak
			}
		} else if pnl < 0 {
			currentLossStreak++
			currentWinStreak = 0
			if currentLossStreak > metrics.MaxConsecutiveLosses {
				metrics.MaxConsecutiveLosses = currentLossStreak
			}
		} else {
			// pnl == 0, reset both
			currentWinStreak = 0
			currentLossStreak = 0
		}
	}

	return metrics
}

// exitTimeHelper safely extracts the exit time from a trade record.
func exitTimeHelper(t storage.TradeRecord) time.Time {
	if t.ExitTime != nil {
		return *t.ExitTime
	}
	return t.EntryTime
}
