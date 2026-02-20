// Package backtest - walkforward.go implements walk-forward optimization analysis.
// Slides a train/test window across a date range, evaluating strategy performance
// on out-of-sample data to measure robustness.
package backtest

import (
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/analytics"
	"github.com/nitinkhare/algoTradingAgent/internal/storage"
)

// WalkForwardConfig defines the window sizes for walk-forward analysis.
type WalkForwardConfig struct {
	TrainWindowDays int // training window in days (default 252 = ~1 year)
	TestWindowDays  int // test window in days (default 63 = ~1 quarter)
	StepDays        int // step size in days (default 63 = ~1 quarter)
}

// WindowResult holds the performance results for a single train/test window.
type WindowResult struct {
	TrainStart  time.Time
	TrainEnd    time.Time
	TestStart   time.Time
	TestEnd     time.Time
	TrainReport *analytics.PerformanceReport
	TestReport  *analytics.PerformanceReport
}

// WalkForwardResult holds the aggregate results of a walk-forward analysis.
type WalkForwardResult struct {
	Windows    []WindowResult
	Aggregate  *analytics.ExtendedMetrics
	Robustness float64 // percentage of test windows with positive PnL (0-100)
}

// WalkForwardEngine runs walk-forward analysis over a date range.
type WalkForwardEngine struct {
	config WalkForwardConfig
}

// NewWalkForwardEngine creates a WalkForwardEngine with the given config.
// Applies defaults for any zero-valued fields.
func NewWalkForwardEngine(config WalkForwardConfig) *WalkForwardEngine {
	if config.TrainWindowDays <= 0 {
		config.TrainWindowDays = 252
	}
	if config.TestWindowDays <= 0 {
		config.TestWindowDays = 63
	}
	if config.StepDays <= 0 {
		config.StepDays = 63
	}
	return &WalkForwardEngine{
		config: config,
	}
}

// Run executes the walk-forward analysis by sliding train/test windows from startDate to endDate.
//
// For each window:
//   - Train period: [windowStart, windowStart + TrainWindowDays)
//   - Test period: [trainEnd, trainEnd + TestWindowDays)
//   - The window advances by StepDays each iteration.
//
// backtestFunc is called for each period and should return a PerformanceReport
// for trades within that date range.
//
// Robustness = percentage of test windows where TotalPnL > 0.
func (wfe *WalkForwardEngine) Run(
	startDate, endDate time.Time,
	backtestFunc func(from, to time.Time) *analytics.PerformanceReport,
) *WalkForwardResult {
	result := &WalkForwardResult{
		Windows: make([]WindowResult, 0),
	}

	trainDuration := time.Duration(wfe.config.TrainWindowDays) * 24 * time.Hour
	testDuration := time.Duration(wfe.config.TestWindowDays) * 24 * time.Hour
	stepDuration := time.Duration(wfe.config.StepDays) * 24 * time.Hour

	// Collect all test window PnLs for aggregate metrics.
	var allTestWindows []windowPnL
	var positiveWindows int

	cursor := startDate
	for {
		trainStart := cursor
		trainEnd := trainStart.Add(trainDuration)
		testStart := trainEnd
		testEnd := testStart.Add(testDuration)

		// Stop if the test window extends beyond the end date.
		if testEnd.After(endDate) {
			break
		}

		// Run backtest for train period.
		trainReport := backtestFunc(trainStart, trainEnd)

		// Run backtest for test period.
		testReport := backtestFunc(testStart, testEnd)

		window := WindowResult{
			TrainStart:  trainStart,
			TrainEnd:    trainEnd,
			TestStart:   testStart,
			TestEnd:     testEnd,
			TrainReport: trainReport,
			TestReport:  testReport,
		}
		result.Windows = append(result.Windows, window)

		// Track test window P&L for robustness calculation.
		if testReport != nil && testReport.TotalPnL > 0 {
			positiveWindows++
		}

		if testReport != nil {
			allTestWindows = append(allTestWindows, windowPnL{
				pnl:   testReport.TotalPnL,
				start: testStart,
				end:   testEnd,
			})
		}

		// Advance cursor by step size.
		cursor = cursor.Add(stepDuration)
	}

	// Compute robustness.
	if len(result.Windows) > 0 {
		result.Robustness = float64(positiveWindows) / float64(len(result.Windows)) * 100
	}

	// Compute aggregate extended metrics from all test windows combined.
	// We create synthetic trade records from window PnLs for ExtendedMetrics computation.
	if len(allTestWindows) > 0 {
		syntheticTrades := make([]storage.TradeRecord, len(allTestWindows))
		for i, wt := range allTestWindows {
			exitTime := wt.end
			syntheticTrades[i] = storage.TradeRecord{
				StrategyID: "walkforward_aggregate",
				EntryPrice: 100, // placeholder
				ExitPrice:  100 + wt.pnl,
				PnL:        wt.pnl,
				EntryTime:  wt.start,
				ExitTime:   &exitTime,
				Status:     "closed",
			}
		}

		// Use the full date range for CAGR calculation.
		initialCapital := 1000000.0 // default 10L for aggregate metrics
		result.Aggregate = analytics.AnalyzeExtended(syntheticTrades, initialCapital, startDate, endDate)
	}

	return result
}

// windowPnL is an internal type to track per-window P&L.
type windowPnL struct {
	pnl   float64
	start time.Time
	end   time.Time
}
