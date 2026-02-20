package main

import (
	"net/http"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/analytics"
)

// handleExtendedMetrics returns advanced performance metrics (CAGR, Sortino, Calmar, etc.)
func (s *Server) handleExtendedMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	trades, err := s.store.GetAllClosedTrades(ctx)
	if err != nil {
		s.logger.Printf("failed to get trades: %v", err)
		s.respondError(w, http.StatusInternalServerError, "failed to fetch trades")
		return
	}

	// Compute basic report for standard fields.
	report := analytics.Analyze(trades, s.cfg.Capital)

	// Determine date range from trades.
	var startDate, endDate time.Time
	if len(trades) > 0 {
		startDate = trades[0].EntryTime
		endDate = time.Now()
		for _, t := range trades {
			if t.EntryTime.Before(startDate) {
				startDate = t.EntryTime
			}
			if t.ExitTime != nil && t.ExitTime.After(endDate) {
				endDate = *t.ExitTime
			}
		}
	} else {
		startDate = time.Now().AddDate(-1, 0, 0)
		endDate = time.Now()
	}

	// Compute extended metrics.
	extended := analytics.AnalyzeExtended(trades, s.cfg.Capital, startDate, endDate)

	resp := ExtendedMetricsResponse{
		TotalPnL:             report.TotalPnL,
		WinRate:              report.WinRate,
		SharpeRatio:          report.SharpeRatio,
		MaxDrawdown:          report.MaxDrawdown,
		ProfitFactor:         report.ProfitFactor,
		TotalTrades:          report.TotalTrades,
		CAGR:                 extended.CAGR,
		SortinoRatio:         extended.SortinoRatio,
		CalmarRatio:          extended.CalmarRatio,
		Expectancy:           extended.Expectancy,
		AvgRMultiple:         extended.AvgRMultiple,
		PayoffRatio:          extended.PayoffRatio,
		RecoveryFactor:       extended.RecoveryFactor,
		MaxConsecutiveWins:   extended.MaxConsecutiveWins,
		MaxConsecutiveLosses: extended.MaxConsecutiveLosses,
		Timestamp:            time.Now(),
	}
	s.respondJSON(w, http.StatusOK, resp)
}

// handleStrategiesPerformance returns rolling performance metrics per strategy.
func (s *Server) handleStrategiesPerformance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	trades, err := s.store.GetAllClosedTrades(ctx)
	if err != nil {
		s.logger.Printf("failed to get trades: %v", err)
		s.respondError(w, http.StatusInternalServerError, "failed to fetch trades")
		return
	}

	report := analytics.Analyze(trades, s.cfg.Capital)

	var items []StrategyPerformanceItem
	for _, sr := range report.StrategyReports {
		items = append(items, StrategyPerformanceItem{
			StrategyID:    sr.StrategyID,
			TotalTrades:   sr.TotalTrades,
			WinRate:       sr.WinRate,
			TotalPnL:      sr.TotalPnL,
			AvgPnL:        sr.AveragePnL,
			RollingSharpe: sr.SharpeRatio,
			MaxDrawdown:   sr.MaxDrawdown,
		})
	}

	resp := StrategyPerformanceResponse{
		Strategies: items,
		Timestamp:  time.Now(),
	}
	s.respondJSON(w, http.StatusOK, resp)
}

// handleStrategiesAllocation returns current capital allocation across strategies.
func (s *Server) handleStrategiesAllocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	trades, err := s.store.GetAllClosedTrades(ctx)
	if err != nil {
		s.logger.Printf("failed to get trades: %v", err)
		s.respondError(w, http.StatusInternalServerError, "failed to fetch trades")
		return
	}

	report := analytics.Analyze(trades, s.cfg.Capital)

	// Build equal-weight allocation based on strategy count.
	stratCount := len(report.StrategyReports)
	if stratCount == 0 {
		stratCount = 1
	}
	pctEach := 100.0 / float64(stratCount)

	var allocations []StrategyAllocationItem
	for _, sr := range report.StrategyReports {
		allocations = append(allocations, StrategyAllocationItem{
			StrategyID:    sr.StrategyID,
			CapitalPct:    pctEach,
			Capital:       s.cfg.Capital * (pctEach / 100.0),
			RollingSharpe: sr.SharpeRatio,
			WinRate:       sr.WinRate,
			TradeCount:    sr.TotalTrades,
		})
	}

	resp := StrategyAllocationResponse{
		Allocations: allocations,
		Mode:        "EQUAL",
		Timestamp:   time.Now(),
	}
	s.respondJSON(w, http.StatusOK, resp)
}

// handleRegimeCurrent returns the current market regime assessment.
func (s *Server) handleRegimeCurrent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Return a default regime status.
	// In production, this would query the latest regime detection from the engine.
	resp := RegimeResponse{
		AIRegime:       "SIDEWAYS",
		DetectedRegime: "RANGING",
		ADX:            0,
		ATRPercentile:  0.5,
		Volatility:     0,
		Confidence:     0.5,
		Timestamp:      time.Now(),
	}
	s.respondJSON(w, http.StatusOK, resp)
}
