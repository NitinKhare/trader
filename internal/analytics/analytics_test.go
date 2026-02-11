package analytics

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/storage"
)

func makeClosedTrade(id int64, strategyID, symbol string, entryPrice, exitPrice float64, qty int, holdDays int) storage.TradeRecord {
	entry := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	exit := entry.Add(time.Duration(holdDays) * 24 * time.Hour)
	pnl := float64(qty) * (exitPrice - entryPrice)
	return storage.TradeRecord{
		ID:         id,
		StrategyID: strategyID,
		Symbol:     symbol,
		Side:       "BUY",
		Quantity:   qty,
		EntryPrice: entryPrice,
		ExitPrice:  exitPrice,
		StopLoss:   entryPrice * 0.95,
		Target:     entryPrice * 1.10,
		EntryTime:  entry,
		ExitTime:   &exit,
		ExitReason: "strategy_exit",
		PnL:        pnl,
		Status:     "closed",
	}
}

func TestAnalyze_EmptyTrades(t *testing.T) {
	report := Analyze(nil, 500000)
	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if report.TotalTrades != 0 {
		t.Errorf("expected 0 trades, got %d", report.TotalTrades)
	}
	if report.WinRate != 0 {
		t.Errorf("expected 0 win rate, got %.2f", report.WinRate)
	}
}

func TestAnalyze_AllWins(t *testing.T) {
	trades := []storage.TradeRecord{
		makeClosedTrade(1, "trend_follow_v1", "RELIANCE", 100, 110, 10, 5),
		makeClosedTrade(2, "trend_follow_v1", "TCS", 200, 220, 5, 3),
		makeClosedTrade(3, "trend_follow_v1", "INFY", 150, 160, 8, 7),
	}

	report := Analyze(trades, 500000)

	if report.TotalTrades != 3 {
		t.Errorf("expected 3 trades, got %d", report.TotalTrades)
	}
	if report.WinningTrades != 3 {
		t.Errorf("expected 3 winning trades, got %d", report.WinningTrades)
	}
	if report.LosingTrades != 0 {
		t.Errorf("expected 0 losing trades, got %d", report.LosingTrades)
	}
	if report.WinRate != 100 {
		t.Errorf("expected 100%% win rate, got %.2f%%", report.WinRate)
	}
	if report.TotalPnL <= 0 {
		t.Errorf("expected positive PnL, got %.2f", report.TotalPnL)
	}
	// 10*(110-100) + 5*(220-200) + 8*(160-150) = 100 + 100 + 80 = 280
	if report.TotalPnL != 280 {
		t.Errorf("expected TotalPnL=280, got %.2f", report.TotalPnL)
	}
	if report.MaxDrawdown != 0 {
		t.Errorf("expected 0 drawdown for all wins, got %.2f", report.MaxDrawdown)
	}
}

func TestAnalyze_AllLosses(t *testing.T) {
	trades := []storage.TradeRecord{
		makeClosedTrade(1, "trend_follow_v1", "RELIANCE", 100, 90, 10, 5),
		makeClosedTrade(2, "trend_follow_v1", "TCS", 200, 180, 5, 3),
	}

	report := Analyze(trades, 500000)

	if report.WinRate != 0 {
		t.Errorf("expected 0%% win rate, got %.2f%%", report.WinRate)
	}
	if report.TotalPnL >= 0 {
		t.Errorf("expected negative PnL, got %.2f", report.TotalPnL)
	}
	// 10*(90-100) + 5*(180-200) = -100 + -100 = -200
	if report.TotalPnL != -200 {
		t.Errorf("expected TotalPnL=-200, got %.2f", report.TotalPnL)
	}
	if report.MaxDrawdown != 200 {
		t.Errorf("expected MaxDrawdown=200, got %.2f", report.MaxDrawdown)
	}
	if report.ProfitFactor != 0 {
		t.Errorf("expected ProfitFactor=0 (no profits), got %.2f", report.ProfitFactor)
	}
}

func TestAnalyze_MixedTrades(t *testing.T) {
	trades := []storage.TradeRecord{
		makeClosedTrade(1, "trend_follow_v1", "WIN1", 100, 120, 10, 5),  // +200
		makeClosedTrade(2, "trend_follow_v1", "LOSS1", 100, 90, 10, 3),  // -100
		makeClosedTrade(3, "trend_follow_v1", "WIN2", 100, 115, 10, 7),  // +150
		makeClosedTrade(4, "trend_follow_v1", "LOSS2", 100, 85, 10, 2),  // -150
	}

	report := Analyze(trades, 500000)

	if report.TotalTrades != 4 {
		t.Errorf("expected 4 trades, got %d", report.TotalTrades)
	}
	if report.WinningTrades != 2 {
		t.Errorf("expected 2 wins, got %d", report.WinningTrades)
	}
	if report.WinRate != 50 {
		t.Errorf("expected 50%% win rate, got %.2f%%", report.WinRate)
	}
	// Total PnL = 200 - 100 + 150 - 150 = 100
	if report.TotalPnL != 100 {
		t.Errorf("expected TotalPnL=100, got %.2f", report.TotalPnL)
	}
	// GrossProfit = 200 + 150 = 350, GrossLoss = 100 + 150 = 250
	if report.GrossProfit != 350 {
		t.Errorf("expected GrossProfit=350, got %.2f", report.GrossProfit)
	}
	if report.GrossLoss != 250 {
		t.Errorf("expected GrossLoss=250, got %.2f", report.GrossLoss)
	}
	// ProfitFactor = 350 / 250 = 1.4
	if math.Abs(report.ProfitFactor-1.4) > 0.01 {
		t.Errorf("expected ProfitFactor=1.4, got %.2f", report.ProfitFactor)
	}
}

func TestAnalyze_MaxDrawdown(t *testing.T) {
	// Sequence: +100, -200, -100, +500
	// Equity: 500000 → 500100 → 499900 → 499800 → 500300
	// Peak = 500100, lowest after = 499800, drawdown = 300
	trades := []storage.TradeRecord{
		makeClosedTrade(1, "s1", "A", 100, 110, 10, 1),  // +100
		makeClosedTrade(2, "s1", "B", 100, 80, 10, 2),   // -200
		makeClosedTrade(3, "s1", "C", 100, 90, 10, 3),   // -100
		makeClosedTrade(4, "s1", "D", 100, 150, 10, 4),  // +500
	}

	report := Analyze(trades, 500000)

	if report.MaxDrawdown != 300 {
		t.Errorf("expected MaxDrawdown=300, got %.2f", report.MaxDrawdown)
	}
}

func TestAnalyze_SharpeRatio(t *testing.T) {
	// All same P&L → stddev=0 → Sharpe=0
	trades := []storage.TradeRecord{
		makeClosedTrade(1, "s1", "A", 100, 110, 10, 1),
		makeClosedTrade(2, "s1", "B", 100, 110, 10, 2),
		makeClosedTrade(3, "s1", "C", 100, 110, 10, 3),
	}

	report := Analyze(trades, 500000)

	if report.SharpeRatio != 0 {
		t.Errorf("expected Sharpe=0 for zero stddev, got %.2f", report.SharpeRatio)
	}
}

func TestAnalyze_SharpeRatio_Varied(t *testing.T) {
	// Varied P&L → non-zero Sharpe
	trades := []storage.TradeRecord{
		makeClosedTrade(1, "s1", "A", 100, 120, 10, 1),  // +200
		makeClosedTrade(2, "s1", "B", 100, 90, 10, 2),   // -100
		makeClosedTrade(3, "s1", "C", 100, 130, 10, 3),  // +300
		makeClosedTrade(4, "s1", "D", 100, 95, 10, 4),   // -50
	}

	report := Analyze(trades, 500000)

	// With mixed but net positive returns, Sharpe should be positive.
	if report.SharpeRatio <= 0 {
		t.Errorf("expected positive Sharpe for net positive returns, got %.2f", report.SharpeRatio)
	}
}

func TestAnalyze_StrategyBreakdown(t *testing.T) {
	trades := []storage.TradeRecord{
		makeClosedTrade(1, "trend_follow_v1", "A", 100, 110, 10, 5),
		makeClosedTrade(2, "trend_follow_v1", "B", 100, 120, 10, 3),
		makeClosedTrade(3, "mean_reversion_v1", "C", 100, 105, 10, 7),
		makeClosedTrade(4, "mean_reversion_v1", "D", 100, 90, 10, 4),
	}

	report := Analyze(trades, 500000)

	if len(report.StrategyReports) != 2 {
		t.Errorf("expected 2 strategy reports, got %d", len(report.StrategyReports))
	}

	tf := report.StrategyReports["trend_follow_v1"]
	if tf == nil {
		t.Fatal("missing trend_follow_v1 report")
	}
	if tf.TotalTrades != 2 {
		t.Errorf("expected 2 trend follow trades, got %d", tf.TotalTrades)
	}
	if tf.WinRate != 100 {
		t.Errorf("expected 100%% win rate for trend follow, got %.2f%%", tf.WinRate)
	}

	mr := report.StrategyReports["mean_reversion_v1"]
	if mr == nil {
		t.Fatal("missing mean_reversion_v1 report")
	}
	if mr.TotalTrades != 2 {
		t.Errorf("expected 2 mean reversion trades, got %d", mr.TotalTrades)
	}
	if mr.WinRate != 50 {
		t.Errorf("expected 50%% win rate for mean reversion, got %.2f%%", mr.WinRate)
	}
}

func TestAnalyze_AverageHoldTime(t *testing.T) {
	trades := []storage.TradeRecord{
		makeClosedTrade(1, "s1", "A", 100, 110, 10, 4),
		makeClosedTrade(2, "s1", "B", 100, 120, 10, 6),
		makeClosedTrade(3, "s1", "C", 100, 105, 10, 8),
	}

	report := Analyze(trades, 500000)

	// Average: (4 + 6 + 8) / 3 = 6.0
	if math.Abs(report.AverageHoldDays-6.0) > 0.1 {
		t.Errorf("expected AverageHoldDays=6.0, got %.1f", report.AverageHoldDays)
	}
	if report.MinHoldDays != 4 {
		t.Errorf("expected MinHoldDays=4, got %d", report.MinHoldDays)
	}
	if report.MaxHoldDays != 8 {
		t.Errorf("expected MaxHoldDays=8, got %d", report.MaxHoldDays)
	}
}

func TestEquityCurve(t *testing.T) {
	trades := []storage.TradeRecord{
		makeClosedTrade(1, "s1", "A", 100, 110, 10, 1), // +100
		makeClosedTrade(2, "s1", "B", 100, 90, 10, 2),  // -100
		makeClosedTrade(3, "s1", "C", 100, 120, 10, 3), // +200
	}

	curve := EquityCurve(trades, 500000)
	if len(curve) == 0 {
		t.Fatal("expected non-empty equity curve")
	}

	// First point should be initial capital.
	if curve[0].Equity != 500000 {
		t.Errorf("expected first point equity=500000, got %.2f", curve[0].Equity)
	}

	// Last point equity = 500000 + 100 - 100 + 200 = 500200
	last := curve[len(curve)-1]
	if last.Equity != 500200 {
		t.Errorf("expected last equity=500200, got %.2f", last.Equity)
	}
}

func TestFormatReport_EmptyTrades(t *testing.T) {
	report := Analyze(nil, 500000)
	formatted := FormatReport(report)
	if !strings.Contains(formatted, "No closed trades") {
		t.Errorf("expected 'No closed trades' message, got: %s", formatted)
	}
}

func TestFormatReport_WithTrades(t *testing.T) {
	trades := []storage.TradeRecord{
		makeClosedTrade(1, "trend_follow_v1", "A", 100, 110, 10, 5),
		makeClosedTrade(2, "mean_reversion_v1", "B", 100, 90, 10, 3),
	}

	report := Analyze(trades, 500000)
	formatted := FormatReport(report)

	if !strings.Contains(formatted, "PERFORMANCE REPORT") {
		t.Error("expected report header")
	}
	if !strings.Contains(formatted, "Total trades") {
		t.Error("expected total trades in report")
	}
	if !strings.Contains(formatted, "STRATEGY BREAKDOWN") {
		t.Error("expected strategy breakdown for multi-strategy report")
	}
}
