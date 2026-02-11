// Package analytics computes performance metrics from closed trade records.
//
// It provides:
//   - Win rate, total P&L, average P&L
//   - Maximum drawdown (absolute and percentage)
//   - Sharpe ratio (annualized, assuming 252 trading days)
//   - Profit factor (gross profits / gross losses)
//   - Average hold time, min/max hold days
//   - Per-strategy breakdown
//   - Human-readable formatted report
//
// All functions are stateless and work on slices of TradeRecord.
package analytics

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/storage"
)

// PerformanceReport holds all computed performance metrics.
type PerformanceReport struct {
	// Overall trade stats.
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	WinRate       float64 // percentage (0-100)

	// P&L.
	TotalPnL   float64
	AveragePnL float64
	GrossProfit float64
	GrossLoss   float64

	// Risk metrics.
	MaxDrawdown    float64 // absolute drawdown
	MaxDrawdownPct float64 // percentage drawdown from peak
	SharpeRatio    float64 // annualized
	ProfitFactor   float64 // gross profit / gross loss

	// Time metrics.
	AverageHoldDays float64
	MaxHoldDays     int
	MinHoldDays     int

	// Strategy breakdown.
	StrategyReports map[string]*StrategyReport
}

// StrategyReport holds per-strategy performance metrics.
type StrategyReport struct {
	StrategyID    string
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	WinRate       float64
	TotalPnL      float64
	AveragePnL    float64
	MaxDrawdown   float64
	SharpeRatio   float64
	AverageHoldDays float64
}

// EquityCurvePoint represents a point on the equity curve.
type EquityCurvePoint struct {
	Date     time.Time
	Equity   float64
	Drawdown float64
}

// Analyze computes the full performance report from a slice of closed trades.
// Trades should have ExitTime set. initialCapital is the starting equity.
// Returns an empty report (not nil) if no trades are provided.
func Analyze(trades []storage.TradeRecord, initialCapital float64) *PerformanceReport {
	report := &PerformanceReport{
		StrategyReports: make(map[string]*StrategyReport),
	}

	if len(trades) == 0 {
		return report
	}

	// Sort by exit time for sequential analysis.
	sorted := make([]storage.TradeRecord, len(trades))
	copy(sorted, trades)
	sort.Slice(sorted, func(i, j int) bool {
		exitI := exitTime(sorted[i])
		exitJ := exitTime(sorted[j])
		return exitI.Before(exitJ)
	})

	// Compute overall metrics.
	var totalHoldDays float64
	var pnls []float64
	report.MinHoldDays = math.MaxInt32

	for _, t := range sorted {
		pnl := t.PnL
		pnls = append(pnls, pnl)
		report.TotalTrades++
		report.TotalPnL += pnl

		if pnl > 0 {
			report.WinningTrades++
			report.GrossProfit += pnl
		} else if pnl < 0 {
			report.LosingTrades++
			report.GrossLoss += math.Abs(pnl)
		}

		// Hold time.
		holdDays := holdDaysForTrade(t)
		totalHoldDays += float64(holdDays)
		if holdDays > report.MaxHoldDays {
			report.MaxHoldDays = holdDays
		}
		if holdDays < report.MinHoldDays {
			report.MinHoldDays = holdDays
		}

		// Per-strategy stats.
		sr, ok := report.StrategyReports[t.StrategyID]
		if !ok {
			sr = &StrategyReport{StrategyID: t.StrategyID}
			report.StrategyReports[t.StrategyID] = sr
		}
		sr.TotalTrades++
		sr.TotalPnL += pnl
		sr.AverageHoldDays += float64(holdDays)
		if pnl > 0 {
			sr.WinningTrades++
		} else if pnl < 0 {
			sr.LosingTrades++
		}
	}

	if report.TotalTrades == 0 {
		report.MinHoldDays = 0
		return report
	}

	// Win rate.
	report.WinRate = float64(report.WinningTrades) / float64(report.TotalTrades) * 100

	// Average P&L.
	report.AveragePnL = report.TotalPnL / float64(report.TotalTrades)

	// Average hold time.
	report.AverageHoldDays = totalHoldDays / float64(report.TotalTrades)

	// Profit factor.
	if report.GrossLoss > 0 {
		report.ProfitFactor = report.GrossProfit / report.GrossLoss
	} else if report.GrossProfit > 0 {
		report.ProfitFactor = math.Inf(1)
	}

	// Max drawdown from equity curve.
	equity := initialCapital
	peak := equity
	for _, pnl := range pnls {
		equity += pnl
		if equity > peak {
			peak = equity
		}
		dd := peak - equity
		if dd > report.MaxDrawdown {
			report.MaxDrawdown = dd
			if peak > 0 {
				report.MaxDrawdownPct = (dd / peak) * 100
			}
		}
	}

	// Sharpe ratio (annualized).
	report.SharpeRatio = computeSharpeRatio(pnls)

	// Per-strategy calculations.
	for _, sr := range report.StrategyReports {
		if sr.TotalTrades > 0 {
			sr.WinRate = float64(sr.WinningTrades) / float64(sr.TotalTrades) * 100
			sr.AveragePnL = sr.TotalPnL / float64(sr.TotalTrades)
			sr.AverageHoldDays = sr.AverageHoldDays / float64(sr.TotalTrades)
		}
		// Per-strategy drawdown and Sharpe could be added, but we keep it simple.
	}

	return report
}

// EquityCurve generates the equity curve from trades sorted by exit date.
func EquityCurve(trades []storage.TradeRecord, initialCapital float64) []EquityCurvePoint {
	if len(trades) == 0 {
		return nil
	}

	sorted := make([]storage.TradeRecord, len(trades))
	copy(sorted, trades)
	sort.Slice(sorted, func(i, j int) bool {
		return exitTime(sorted[i]).Before(exitTime(sorted[j]))
	})

	equity := initialCapital
	peak := equity
	points := make([]EquityCurvePoint, 0, len(sorted)+1)

	// Starting point.
	points = append(points, EquityCurvePoint{
		Date:   sorted[0].EntryTime,
		Equity: equity,
	})

	for _, t := range sorted {
		equity += t.PnL
		if equity > peak {
			peak = equity
		}
		dd := peak - equity
		points = append(points, EquityCurvePoint{
			Date:     exitTime(t),
			Equity:   equity,
			Drawdown: dd,
		})
	}

	return points
}

// FormatReport returns a human-readable text summary of the performance report.
func FormatReport(report *PerformanceReport) string {
	if report == nil || report.TotalTrades == 0 {
		return "No closed trades to analyze."
	}

	var b strings.Builder

	b.WriteString("═══════════════════════════════════════════════════\n")
	b.WriteString("              PERFORMANCE REPORT\n")
	b.WriteString("═══════════════════════════════════════════════════\n\n")

	// Overall stats.
	b.WriteString("── TRADE SUMMARY ──\n")
	fmt.Fprintf(&b, "  Total trades:    %d\n", report.TotalTrades)
	fmt.Fprintf(&b, "  Winning trades:  %d (%.1f%%)\n", report.WinningTrades, report.WinRate)
	fmt.Fprintf(&b, "  Losing trades:   %d\n", report.LosingTrades)
	b.WriteString("\n")

	// P&L.
	b.WriteString("── PROFIT & LOSS ──\n")
	fmt.Fprintf(&b, "  Total P&L:       ₹%.2f\n", report.TotalPnL)
	fmt.Fprintf(&b, "  Average P&L:     ₹%.2f\n", report.AveragePnL)
	fmt.Fprintf(&b, "  Gross profit:    ₹%.2f\n", report.GrossProfit)
	fmt.Fprintf(&b, "  Gross loss:      ₹%.2f\n", report.GrossLoss)
	fmt.Fprintf(&b, "  Profit factor:   %.2f\n", report.ProfitFactor)
	b.WriteString("\n")

	// Risk.
	b.WriteString("── RISK METRICS ──\n")
	fmt.Fprintf(&b, "  Max drawdown:    ₹%.2f (%.2f%%)\n", report.MaxDrawdown, report.MaxDrawdownPct)
	fmt.Fprintf(&b, "  Sharpe ratio:    %.2f\n", report.SharpeRatio)
	b.WriteString("\n")

	// Time.
	b.WriteString("── HOLD TIME ──\n")
	fmt.Fprintf(&b, "  Average:         %.1f days\n", report.AverageHoldDays)
	fmt.Fprintf(&b, "  Min:             %d days\n", report.MinHoldDays)
	fmt.Fprintf(&b, "  Max:             %d days\n", report.MaxHoldDays)
	b.WriteString("\n")

	// Strategy breakdown.
	if len(report.StrategyReports) > 1 {
		b.WriteString("── STRATEGY BREAKDOWN ──\n")
		for _, sr := range report.StrategyReports {
			fmt.Fprintf(&b, "  [%s]\n", sr.StrategyID)
			fmt.Fprintf(&b, "    Trades: %d | Win rate: %.1f%% | P&L: ₹%.2f | Avg hold: %.1f days\n",
				sr.TotalTrades, sr.WinRate, sr.TotalPnL, sr.AverageHoldDays)
		}
		b.WriteString("\n")
	}

	b.WriteString("═══════════════════════════════════════════════════\n")

	return b.String()
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

// exitTime safely extracts the exit time from a trade record.
func exitTime(t storage.TradeRecord) time.Time {
	if t.ExitTime != nil {
		return *t.ExitTime
	}
	return t.EntryTime // fallback if exit time not set
}

// holdDaysForTrade calculates the number of calendar days a trade was held.
func holdDaysForTrade(t storage.TradeRecord) int {
	exit := exitTime(t)
	days := int(exit.Sub(t.EntryTime).Hours() / 24)
	if days < 0 {
		days = 0
	}
	return days
}

// computeSharpeRatio calculates the annualized Sharpe ratio from a slice of P&L values.
// Assumes zero risk-free rate and 252 trading days per year.
func computeSharpeRatio(pnls []float64) float64 {
	if len(pnls) < 2 {
		return 0
	}

	// Mean return.
	var sum float64
	for _, p := range pnls {
		sum += p
	}
	mean := sum / float64(len(pnls))

	// Standard deviation.
	var variance float64
	for _, p := range pnls {
		diff := p - mean
		variance += diff * diff
	}
	variance /= float64(len(pnls) - 1) // sample variance
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return 0
	}

	// Annualize: Sharpe = (mean / stdDev) * sqrt(252)
	return (mean / stdDev) * math.Sqrt(252)
}
