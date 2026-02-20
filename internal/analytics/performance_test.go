package analytics

import (
	"math"
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/storage"
)

func makeTestTrades(pnls []float64, stopLoss float64) []storage.TradeRecord {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	trades := make([]storage.TradeRecord, len(pnls))
	for i, pnl := range pnls {
		exitTime := base.Add(time.Duration(i+1) * 24 * time.Hour)
		entryPrice := 100.0
		exitPrice := entryPrice + pnl/10.0
		trades[i] = storage.TradeRecord{
			StrategyID: "test_strat",
			Symbol:     "TEST",
			EntryPrice: entryPrice,
			ExitPrice:  exitPrice,
			StopLoss:   stopLoss,
			PnL:        pnl,
			EntryTime:  base.Add(time.Duration(i) * 24 * time.Hour),
			ExitTime:   &exitTime,
			Status:     "closed",
		}
	}
	return trades
}

func TestAnalyzeExtended_WinningTrades(t *testing.T) {
	trades := makeTestTrades([]float64{100, 200, 50, 150, 300}, 0)
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(365 * 24 * time.Hour)
	m := AnalyzeExtended(trades, 100000, start, end)
	if m.CAGR <= 0 {
		t.Errorf("expected positive CAGR, got %f", m.CAGR)
	}
	if m.Expectancy <= 0 {
		t.Errorf("expected positive expectancy, got %f", m.Expectancy)
	}
	if m.MaxConsecutiveWins != 5 {
		t.Errorf("expected 5 consecutive wins, got %d", m.MaxConsecutiveWins)
	}
}

func TestAnalyzeExtended_LosingTrades(t *testing.T) {
	trades := makeTestTrades([]float64{-100, -200, -50, -150}, 0)
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(365 * 24 * time.Hour)
	m := AnalyzeExtended(trades, 100000, start, end)
	if m.CAGR >= 0 {
		t.Errorf("expected negative CAGR, got %f", m.CAGR)
	}
	if m.Expectancy >= 0 {
		t.Errorf("expected negative expectancy, got %f", m.Expectancy)
	}
	if m.MaxConsecutiveLosses != 4 {
		t.Errorf("expected 4 consecutive losses, got %d", m.MaxConsecutiveLosses)
	}
}

func TestAnalyzeExtended_MixedTrades(t *testing.T) {
	trades := makeTestTrades([]float64{100, -50, 200, -30, 150, -100, 80}, 0)
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(365 * 24 * time.Hour)
	m := AnalyzeExtended(trades, 100000, start, end)
	if m.PayoffRatio <= 0 {
		t.Errorf("expected positive payoff ratio, got %f", m.PayoffRatio)
	}
	if m.MaxConsecutiveWins < 1 || m.MaxConsecutiveLosses < 1 {
		t.Error("expected at least 1 streak for wins and losses")
	}
}

func TestAnalyzeExtended_WithStopLoss(t *testing.T) {
	trades := makeTestTrades([]float64{100, -50, 200}, 90.0)
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(365 * 24 * time.Hour)
	m := AnalyzeExtended(trades, 100000, start, end)
	if m.AvgRMultiple == 0 {
		t.Error("expected non-zero AvgRMultiple with stop loss set")
	}
}

func TestAnalyzeExtended_EmptyTrades(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(365 * 24 * time.Hour)
	m := AnalyzeExtended(nil, 100000, start, end)
	if m == nil {
		t.Fatal("expected non-nil result for empty trades")
	}
	if m.CAGR != 0 {
		t.Errorf("expected 0 CAGR for empty trades, got %f", m.CAGR)
	}
}

func TestAnalyzeExtended_ZeroCapital(t *testing.T) {
	trades := makeTestTrades([]float64{100}, 0)
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(365 * 24 * time.Hour)
	m := AnalyzeExtended(trades, 0, start, end)
	if m.CAGR != 0 {
		t.Errorf("expected 0 CAGR for zero capital, got %f", m.CAGR)
	}
}

func TestAnalyzeExtended_SortinoRatio(t *testing.T) {
	trades := makeTestTrades([]float64{100, 200, -50, 300, -20}, 0)
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(365 * 24 * time.Hour)
	m := AnalyzeExtended(trades, 100000, start, end)
	if math.IsNaN(m.SortinoRatio) || math.IsInf(m.SortinoRatio, 0) {
		t.Errorf("expected finite Sortino ratio, got %f", m.SortinoRatio)
	}
}
