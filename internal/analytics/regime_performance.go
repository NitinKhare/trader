// Package analytics - regime_performance.go provides performance analysis
// segmented by market regime. Trades are grouped by the regime active at their
// entry date, then each group is analyzed separately.
package analytics

import (
	"github.com/nitinkhare/algoTradingAgent/internal/regime"
	"github.com/nitinkhare/algoTradingAgent/internal/storage"
	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// RegimePerformance holds performance reports broken down by regime type.
type RegimePerformance struct {
	ByAIRegime       map[strategy.MarketRegime]*PerformanceReport
	ByDetectedRegime map[regime.DetectedRegime]*PerformanceReport
}

// AnalyzeByRegime groups trades by the regime active at their entry date and computes
// performance reports for each regime.
//
// Parameters:
//   - trades: slice of closed trade records to analyze
//   - initialCapital: the starting equity for each sub-analysis (used per-group proportionally)
//   - aiRegimeHistory: map of date string (YYYY-MM-DD) to AI market regime
//   - detectedRegimeHistory: map of date string (YYYY-MM-DD) to detected regime
//
// Trades whose entry date is not found in the regime history maps are skipped for
// that particular breakdown.
func AnalyzeByRegime(
	trades []storage.TradeRecord,
	initialCapital float64,
	aiRegimeHistory map[string]strategy.MarketRegime,
	detectedRegimeHistory map[string]regime.DetectedRegime,
) *RegimePerformance {
	result := &RegimePerformance{
		ByAIRegime:       make(map[strategy.MarketRegime]*PerformanceReport),
		ByDetectedRegime: make(map[regime.DetectedRegime]*PerformanceReport),
	}

	if len(trades) == 0 {
		return result
	}

	// Group trades by AI regime at entry date.
	aiGroups := make(map[strategy.MarketRegime][]storage.TradeRecord)
	for _, t := range trades {
		dateKey := t.EntryTime.Format("2006-01-02")
		if r, ok := aiRegimeHistory[dateKey]; ok {
			aiGroups[r] = append(aiGroups[r], t)
		}
	}

	// Analyze each AI regime group.
	for r, group := range aiGroups {
		// Use proportional capital based on the fraction of trades in this group.
		proportionalCapital := initialCapital * float64(len(group)) / float64(len(trades))
		if proportionalCapital <= 0 {
			proportionalCapital = initialCapital
		}
		result.ByAIRegime[r] = Analyze(group, proportionalCapital)
	}

	// Group trades by detected regime at entry date.
	detectedGroups := make(map[regime.DetectedRegime][]storage.TradeRecord)
	for _, t := range trades {
		dateKey := t.EntryTime.Format("2006-01-02")
		if r, ok := detectedRegimeHistory[dateKey]; ok {
			detectedGroups[r] = append(detectedGroups[r], t)
		}
	}

	// Analyze each detected regime group.
	for r, group := range detectedGroups {
		proportionalCapital := initialCapital * float64(len(group)) / float64(len(trades))
		if proportionalCapital <= 0 {
			proportionalCapital = initialCapital
		}
		result.ByDetectedRegime[r] = Analyze(group, proportionalCapital)
	}

	return result
}
