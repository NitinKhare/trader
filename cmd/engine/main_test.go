package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/broker"
	"github.com/nitinkhare/algoTradingAgent/internal/config"
	"github.com/nitinkhare/algoTradingAgent/internal/market"
	"github.com/nitinkhare/algoTradingAgent/internal/risk"
	"github.com/nitinkhare/algoTradingAgent/internal/scheduler"
	"github.com/nitinkhare/algoTradingAgent/internal/storage"
	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// ────────────────────────────────────────────────────────────────────
// Test helpers
// ────────────────────────────────────────────────────────────────────

// writeJSON writes a value as JSON to a file, creating parent dirs.
func writeJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

// writeCSV writes raw CSV content to a file.
func writeCSV(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// writeTrendingCandles writes a synthetic CSV with n trading-day candles
// ending near today. Prices drift up steadily for positive ATR/trend signals.
func writeTrendingCandles(t *testing.T, dir, symbol string, n int, basePrice float64) {
	t.Helper()
	lines := "date,open,high,low,close,volume\n"

	today := time.Now().In(market.IST)
	// Generate n business days ending today.
	dates := make([]time.Time, 0, n)
	d := today
	for len(dates) < n {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			dates = append([]time.Time{d}, dates...)
		}
		d = d.AddDate(0, 0, -1)
	}

	for i, date := range dates {
		// Steady uptrend with controlled volatility.
		drift := float64(i) * 2.0
		open := basePrice + drift
		high := open + 12.0
		low := open - 8.0
		close := open + 5.0 + float64(i%3)*2.0 // slight variation
		vol := 1000000 + i*10000

		lines += date.Format("2006-01-02") + ","
		lines += fmtFloat(open) + "," + fmtFloat(high) + "," + fmtFloat(low) + ","
		lines += fmtFloat(close) + ","
		lines += fmtFloat(float64(vol)) + "\n"
	}

	writeCSV(t, filepath.Join(dir, symbol+".csv"), lines)
}

func fmtFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

// runJobsDirectly uses ForceRunMarketHourJobs to execute the trading
// pipeline regardless of current time. This bypasses the scheduler's
// IsMarketOpen check so tests work at any hour.
func runJobsDirectly(t *testing.T, cfg *config.Config,
	b broker.Broker, strats []strategy.Strategy,
	riskMgr *risk.Manager, logger *log.Logger,
) {
	t.Helper()
	cal := market.NewCalendarFromHolidays(map[string]string{})
	sched := scheduler.New(cal, logger)
	registerMarketJobs(sched, cfg, b, strats, riskMgr, nil, nil, nil, nil, nil, logger)
	ctx := context.Background()
	if err := sched.ForceRunMarketHourJobs(ctx); err != nil {
		t.Fatalf("ForceRunMarketHourJobs failed: %v", err)
	}
}

// ────────────────────────────────────────────────────────────────────
// E2E Test: Paper mode full pipeline
// ────────────────────────────────────────────────────────────────────

func TestPaperMode_EndToEnd(t *testing.T) {
	// Setup temp directories.
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, "ai_outputs")
	dataDir := filepath.Join(tmpDir, "market_data")

	// Today's date (the execute_trades job reads from today's directory).
	today := time.Now().In(market.IST).Format("2006-01-02")
	todayDir := filepath.Join(aiDir, today)

	// ── 1. Create market regime ──
	regime := strategy.MarketRegimeData{
		Date:       today,
		Regime:     strategy.RegimeBull,
		Confidence: 0.95,
	}
	writeJSON(t, filepath.Join(todayDir, "market_regime.json"), regime)

	// ── 2. Create stock scores ──
	// 3 stocks: TESTSTOCK1 (high scores, will buy), TESTSTOCK2 (high scores, will buy),
	// TESTSTOCK3 (low trend, should skip).
	scores := []strategy.StockScore{
		{
			Symbol:               "TESTSTOCK1",
			TrendStrengthScore:   0.85,
			BreakoutQualityScore: 0.90,
			VolatilityScore:      0.50,
			RiskScore:            0.10,
			LiquidityScore:       0.80,
			CompositeScore:       0.85,
			Rank:                 1,
		},
		{
			Symbol:               "TESTSTOCK2",
			TrendStrengthScore:   0.75,
			BreakoutQualityScore: 0.70,
			VolatilityScore:      0.45,
			RiskScore:            0.20,
			LiquidityScore:       0.65,
			CompositeScore:       0.70,
			Rank:                 2,
		},
		{
			Symbol:               "TESTSTOCK3",
			TrendStrengthScore:   0.30, // Below 0.6 threshold
			BreakoutQualityScore: 0.20,
			VolatilityScore:      0.80,
			RiskScore:            0.70, // Above 0.5 threshold
			LiquidityScore:       0.30,
			CompositeScore:       0.30,
			Rank:                 3,
		},
	}
	writeJSON(t, filepath.Join(todayDir, "stock_scores.json"), scores)

	// ── 3. Create candle data (50 candles each for ATR calculation) ──
	writeTrendingCandles(t, dataDir, "TESTSTOCK1", 50, 500.0)
	writeTrendingCandles(t, dataDir, "TESTSTOCK2", 50, 200.0)
	writeTrendingCandles(t, dataDir, "TESTSTOCK3", 50, 100.0)

	// ── 4. Create config ──
	cfg := &config.Config{
		ActiveBroker: "dhan",
		TradingMode:  config.ModePaper,
		Capital:      500000.0,
		Risk: config.RiskConfig{
			MaxRiskPerTradePct:      1.0,
			MaxOpenPositions:        5,
			MaxDailyLossPct:         3.0,
			MaxCapitalDeploymentPct: 80.0,
		},
		Paths: config.PathsConfig{
			AIOutputDir:   aiDir,
			MarketDataDir: dataDir,
			LogDir:        filepath.Join(tmpDir, "logs"),
		},
		DatabaseURL:        "postgres://unused@localhost/unused?sslmode=disable",
		MarketCalendarPath: "",
	}

	// ── 5. Initialize components ──
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	paperBroker := broker.NewPaperBroker(cfg.Capital)
	riskMgr := risk.NewManager(cfg.Risk, cfg.Capital)
	strats := []strategy.Strategy{
		strategy.NewTrendFollowStrategy(cfg.Risk),
	}

	// ── 6. Force-run market-hour jobs (bypasses IsMarketOpen check) ──
	runJobsDirectly(t, cfg, paperBroker, strats, riskMgr, logger)

	// ── 7. Verify results ──
	ctx := context.Background()
	funds, err := paperBroker.GetFunds(ctx)
	if err != nil {
		t.Fatalf("GetFunds: %v", err)
	}
	holdings, err := paperBroker.GetHoldings(ctx)
	if err != nil {
		t.Fatalf("GetHoldings: %v", err)
	}

	t.Logf("=== PAPER MODE RESULTS ===")
	t.Logf("Available cash: %.2f (started: %.2f)", funds.AvailableCash, cfg.Capital)
	t.Logf("Used margin: %.2f", funds.UsedMargin)
	t.Logf("Holdings: %d", len(holdings))
	for _, h := range holdings {
		t.Logf("  %s: qty=%d avg=%.2f", h.Symbol, h.Quantity, h.AveragePrice)
	}

	// A) Capital should have decreased (some stocks were bought).
	if funds.AvailableCash >= cfg.Capital {
		t.Errorf("expected capital to decrease after buys, got %.2f (started %.2f)",
			funds.AvailableCash, cfg.Capital)
	}

	// B) Should have at least 1 holding (TESTSTOCK1 should definitely pass all filters).
	if len(holdings) < 1 {
		t.Errorf("expected at least 1 holding, got %d", len(holdings))
	}

	// C) TESTSTOCK1 should be in holdings (rank 1, high scores, BULL regime).
	found := false
	for _, h := range holdings {
		if h.Symbol == "TESTSTOCK1" {
			found = true
			if h.Quantity <= 0 {
				t.Errorf("TESTSTOCK1 should have positive quantity, got %d", h.Quantity)
			}
		}
	}
	if !found {
		t.Errorf("expected TESTSTOCK1 to be in holdings")
	}

	// D) TESTSTOCK3 should NOT be in holdings (low scores, should be skipped).
	for _, h := range holdings {
		if h.Symbol == "TESTSTOCK3" {
			t.Errorf("TESTSTOCK3 should NOT be in holdings (low scores), but found qty=%d", h.Quantity)
		}
	}

	// E) Deployed capital should be positive.
	totalDeployed := cfg.Capital - funds.AvailableCash
	if totalDeployed < 0 {
		t.Errorf("deployed capital is negative: %.2f", totalDeployed)
	}
	t.Logf("Total deployed: %.2f (%.1f%% of capital)", totalDeployed, totalDeployed/cfg.Capital*100)
}

// TestPaperMode_ExitMonitoring verifies that the monitor_exits job
// can detect and exit positions when regime changes to BEAR.
func TestPaperMode_ExitMonitoring(t *testing.T) {
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, "ai_outputs")
	dataDir := filepath.Join(tmpDir, "market_data")
	today := time.Now().In(market.IST).Format("2006-01-02")
	todayDir := filepath.Join(aiDir, today)

	// Create BEAR regime — should trigger exits.
	regime := strategy.MarketRegimeData{
		Date:       today,
		Regime:     strategy.RegimeBear,
		Confidence: 0.90,
	}
	writeJSON(t, filepath.Join(todayDir, "market_regime.json"), regime)

	// Scores for the stock we hold.
	scores := []strategy.StockScore{
		{
			Symbol:               "EXITSTOCK",
			TrendStrengthScore:   0.20, // Below exit threshold (0.3)
			BreakoutQualityScore: 0.30,
			VolatilityScore:      0.60,
			RiskScore:            0.50,
			LiquidityScore:       0.40,
			CompositeScore:       0.30,
			Rank:                 1,
		},
	}
	writeJSON(t, filepath.Join(todayDir, "stock_scores.json"), scores)

	// Candle data for EXITSTOCK.
	writeTrendingCandles(t, dataDir, "EXITSTOCK", 50, 300.0)

	cfg := &config.Config{
		ActiveBroker: "dhan",
		TradingMode:  config.ModePaper,
		Capital:      500000.0,
		Risk: config.RiskConfig{
			MaxRiskPerTradePct:      1.0,
			MaxOpenPositions:        5,
			MaxDailyLossPct:         3.0,
			MaxCapitalDeploymentPct: 80.0,
		},
		Paths: config.PathsConfig{
			AIOutputDir:   aiDir,
			MarketDataDir: dataDir,
			LogDir:        filepath.Join(tmpDir, "logs"),
		},
		DatabaseURL:        "postgres://unused@localhost/unused?sslmode=disable",
		MarketCalendarPath: "",
	}

	logger := log.New(os.Stdout, "[test-exit] ", log.LstdFlags)

	// Create paper broker with a pre-existing holding.
	paperBroker := broker.NewPaperBroker(cfg.Capital)
	ctx := context.Background()

	// Buy the stock first to create a holding.
	buyOrder := broker.Order{
		Symbol:   "EXITSTOCK",
		Exchange: "NSE",
		Side:     broker.OrderSideBuy,
		Type:     broker.OrderTypeLimit,
		Quantity: 10,
		Price:    325.0,
		Product:  "CNC",
	}
	resp, err := paperBroker.PlaceOrder(ctx, buyOrder)
	if err != nil {
		t.Fatalf("pre-buy failed: %v", err)
	}
	t.Logf("Pre-buy: order=%s status=%s", resp.OrderID, resp.Status)

	// Verify holding exists.
	holdingsBefore, _ := paperBroker.GetHoldings(ctx)
	if len(holdingsBefore) != 1 || holdingsBefore[0].Symbol != "EXITSTOCK" {
		t.Fatalf("expected 1 holding of EXITSTOCK, got %d", len(holdingsBefore))
	}
	t.Logf("Holding before exit: EXITSTOCK qty=%d avg=%.2f",
		holdingsBefore[0].Quantity, holdingsBefore[0].AveragePrice)

	// Run jobs — BEAR regime should trigger monitor_exits to sell.
	riskMgr := risk.NewManager(cfg.Risk, cfg.Capital)
	strats := []strategy.Strategy{
		strategy.NewTrendFollowStrategy(cfg.Risk),
	}

	runJobsDirectly(t, cfg, paperBroker, strats, riskMgr, logger)

	// Check results — BEAR regime should trigger exit.
	holdingsAfter, _ := paperBroker.GetHoldings(ctx)
	fundsAfter, _ := paperBroker.GetFunds(ctx)

	t.Logf("Holdings after exit monitoring: %d", len(holdingsAfter))
	t.Logf("Available cash after: %.2f", fundsAfter.AvailableCash)

	if len(holdingsAfter) != 0 {
		t.Errorf("expected 0 holdings after BEAR regime exit, got %d", len(holdingsAfter))
		for _, h := range holdingsAfter {
			t.Logf("  still holding: %s qty=%d", h.Symbol, h.Quantity)
		}
	}
}

// TestPaperMode_RiskRejection verifies that risk manager blocks
// trades that exceed position limits.
func TestPaperMode_RiskRejection(t *testing.T) {
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, "ai_outputs")
	dataDir := filepath.Join(tmpDir, "market_data")
	today := time.Now().In(market.IST).Format("2006-01-02")
	todayDir := filepath.Join(aiDir, today)

	regime := strategy.MarketRegimeData{
		Date:       today,
		Regime:     strategy.RegimeBull,
		Confidence: 0.95,
	}
	writeJSON(t, filepath.Join(todayDir, "market_regime.json"), regime)

	// Create 7 high-scoring stocks but max_open_positions is 5.
	var scores []strategy.StockScore
	for i := 1; i <= 7; i++ {
		sym := "STOCK" + string(rune('A'+i-1))
		scores = append(scores, strategy.StockScore{
			Symbol:               sym,
			TrendStrengthScore:   0.80,
			BreakoutQualityScore: 0.75,
			VolatilityScore:      0.50,
			RiskScore:            0.15,
			LiquidityScore:       0.70,
			CompositeScore:       0.75,
			Rank:                 i,
		})
		writeTrendingCandles(t, dataDir, sym, 50, 100.0+float64(i)*50)
	}
	writeJSON(t, filepath.Join(todayDir, "stock_scores.json"), scores)

	cfg := &config.Config{
		ActiveBroker: "dhan",
		TradingMode:  config.ModePaper,
		Capital:      500000.0,
		Risk: config.RiskConfig{
			MaxRiskPerTradePct:      1.0,
			MaxOpenPositions:        5, // Only 5 allowed
			MaxDailyLossPct:         3.0,
			MaxCapitalDeploymentPct: 80.0,
		},
		Paths: config.PathsConfig{
			AIOutputDir:   aiDir,
			MarketDataDir: dataDir,
			LogDir:        filepath.Join(tmpDir, "logs"),
		},
		DatabaseURL:        "postgres://unused@localhost/unused?sslmode=disable",
		MarketCalendarPath: "",
	}

	logger := log.New(os.Stdout, "[test-risk] ", log.LstdFlags)
	paperBroker := broker.NewPaperBroker(cfg.Capital)
	riskMgr := risk.NewManager(cfg.Risk, cfg.Capital)
	strats := []strategy.Strategy{
		strategy.NewTrendFollowStrategy(cfg.Risk),
	}

	runJobsDirectly(t, cfg, paperBroker, strats, riskMgr, logger)

	ctx := context.Background()
	holdings, _ := paperBroker.GetHoldings(ctx)

	t.Logf("Holdings: %d (max allowed: %d)", len(holdings), cfg.Risk.MaxOpenPositions)
	for _, h := range holdings {
		t.Logf("  %s: qty=%d avg=%.2f", h.Symbol, h.Quantity, h.AveragePrice)
	}

	// Should have at most 5 positions.
	if len(holdings) > cfg.Risk.MaxOpenPositions {
		t.Errorf("expected max %d positions, got %d", cfg.Risk.MaxOpenPositions, len(holdings))
	}
}

// ────────────────────────────────────────────────────────────────────
// Position Restore & Enrichment Tests
// ────────────────────────────────────────────────────────────────────

func TestPositionRestore_PaperBroker(t *testing.T) {
	pb := broker.NewPaperBroker(500000)
	ctx := context.Background()

	// Simulate a trade context as if loaded from DB.
	tc := &tradeContext{
		trades: map[string]*storage.TradeRecord{
			"RELIANCE": {
				ID:         1,
				StrategyID: "trend_follow_v1",
				SignalID:   "sig-RELIANCE-2026-02-08",
				Symbol:     "RELIANCE",
				Side:       "BUY",
				Quantity:   10,
				EntryPrice: 2500.0,
				StopLoss:   2400.0,
				Target:     2700.0,
				EntryTime:  time.Now().Add(-48 * time.Hour),
				Status:     "open",
			},
		},
	}

	// Seed paper broker from trade context.
	for _, trade := range tc.trades {
		pb.RestoreHolding(trade.Symbol, "NSE", trade.Quantity, trade.EntryPrice)
	}

	// Verify holdings restored.
	holdings, err := pb.GetHoldings(ctx)
	if err != nil {
		t.Fatalf("GetHoldings: %v", err)
	}
	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}
	if holdings[0].Symbol != "RELIANCE" || holdings[0].Quantity != 10 {
		t.Errorf("unexpected holding: %+v", holdings[0])
	}

	// Verify funds were adjusted.
	funds, _ := pb.GetFunds(ctx)
	expectedCash := 500000.0 - (2500.0 * 10)
	if funds.AvailableCash != expectedCash {
		t.Errorf("expected cash %.2f, got %.2f", expectedCash, funds.AvailableCash)
	}

	// Verify enrichPosition fills all context fields.
	pos := enrichPosition(holdings[0], tc)
	if pos.StopLoss != 2400.0 {
		t.Errorf("expected StopLoss 2400.0, got %.2f", pos.StopLoss)
	}
	if pos.Target != 2700.0 {
		t.Errorf("expected Target 2700.0, got %.2f", pos.Target)
	}
	if pos.StrategyID != "trend_follow_v1" {
		t.Errorf("expected StrategyID trend_follow_v1, got %s", pos.StrategyID)
	}
	if pos.SignalID != "sig-RELIANCE-2026-02-08" {
		t.Errorf("expected SignalID, got %s", pos.SignalID)
	}
	// enrichPosition should use DB entry price over broker average price.
	if pos.EntryPrice != 2500.0 {
		t.Errorf("expected EntryPrice 2500.0, got %.2f", pos.EntryPrice)
	}
}

func TestEnrichPosition_NilTradeContext(t *testing.T) {
	h := broker.Holding{
		Symbol:       "TCS",
		Exchange:     "NSE",
		AveragePrice: 3500.0,
		Quantity:     5,
	}

	// With nil trade context, should return basic position info.
	pos := enrichPosition(h, nil)
	if pos.Symbol != "TCS" || pos.EntryPrice != 3500.0 || pos.Quantity != 5 {
		t.Errorf("basic fields wrong: %+v", pos)
	}
	if pos.StopLoss != 0 || pos.Target != 0 || pos.StrategyID != "" {
		t.Errorf("expected zero enrichment with nil context, got: sl=%.2f tgt=%.2f strat=%s",
			pos.StopLoss, pos.Target, pos.StrategyID)
	}
}

func TestEnrichPosition_MissingSymbol(t *testing.T) {
	h := broker.Holding{
		Symbol:       "UNKNOWN",
		Exchange:     "NSE",
		AveragePrice: 1000.0,
		Quantity:     5,
	}

	tc := &tradeContext{
		trades: map[string]*storage.TradeRecord{
			"RELIANCE": {Symbol: "RELIANCE", StopLoss: 2400.0, Target: 2700.0},
		},
	}

	// UNKNOWN is not in trade context, should return basic position info.
	pos := enrichPosition(h, tc)
	if pos.Symbol != "UNKNOWN" || pos.EntryPrice != 1000.0 {
		t.Errorf("basic fields wrong: %+v", pos)
	}
	if pos.StopLoss != 0 || pos.Target != 0 {
		t.Errorf("expected zero SL/target for unknown symbol, got: sl=%.2f tgt=%.2f",
			pos.StopLoss, pos.Target)
	}
}

func TestReconcile_NilStore_NoPanic(t *testing.T) {
	tc := &tradeContext{
		trades: map[string]*storage.TradeRecord{
			"RELIANCE": {ID: 1, Symbol: "RELIANCE", Quantity: 10},
		},
	}
	logger := log.New(os.Stdout, "[test-reconcile] ", log.LstdFlags)
	ctx := context.Background()

	// Should not panic with nil store.
	reconcilePositions(ctx, nil, logger, tc, []broker.Holding{})

	// Trade context should be unchanged (nil store = no-op).
	if _, ok := tc.trades["RELIANCE"]; !ok {
		t.Error("expected RELIANCE to remain in trade context with nil store")
	}
}

func TestReconcile_NilTradeContext_NoPanic(t *testing.T) {
	logger := log.New(os.Stdout, "[test-reconcile] ", log.LstdFlags)
	ctx := context.Background()

	// Should not panic with nil trade context.
	reconcilePositions(ctx, nil, logger, nil, []broker.Holding{
		{Symbol: "TCS", Quantity: 5},
	})
}

// ────────────────────────────────────────────────────────────────────
// Order Status Polling Tests
// ────────────────────────────────────────────────────────────────────

func TestPollOrderStatus_PaperBrokerImmediateComplete(t *testing.T) {
	logger := log.New(os.Stdout, "[test-poll] ", log.LstdFlags)
	pb := broker.NewPaperBroker(100000)
	ctx := context.Background()

	// Place a regular buy order (paper broker fills immediately).
	resp, err := pb.PlaceOrder(ctx, broker.Order{
		Symbol: "TCS", Exchange: "NSE", Side: broker.OrderSideBuy,
		Type: broker.OrderTypeLimit, Quantity: 5, Price: 3500.0, Product: "CNC",
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	// Poll should return COMPLETED immediately (no actual polling needed).
	status, err := pollOrderStatus(ctx, pb, resp.OrderID, 1*time.Second, 5*time.Second, logger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status.Status != broker.OrderStatusCompleted {
		t.Errorf("expected COMPLETED, got %s", status.Status)
	}
	if status.FilledQty != 5 {
		t.Errorf("expected FilledQty=5, got %d", status.FilledQty)
	}
}

func TestIsTerminalOrderStatus(t *testing.T) {
	tests := []struct {
		status   broker.OrderStatus
		terminal bool
	}{
		{broker.OrderStatusCompleted, true},
		{broker.OrderStatusRejected, true},
		{broker.OrderStatusCancelled, true},
		{broker.OrderStatusPending, false},
		{broker.OrderStatusOpen, false},
	}
	for _, tt := range tests {
		if got := isTerminalOrderStatus(tt.status); got != tt.terminal {
			t.Errorf("isTerminalOrderStatus(%s) = %v, want %v", tt.status, got, tt.terminal)
		}
	}
}

// ────────────────────────────────────────────────────────────────────
// Stop-Loss Order Tests
// ────────────────────────────────────────────────────────────────────

func TestStopLossOrder_PaperBroker(t *testing.T) {
	pb := broker.NewPaperBroker(100000)
	ctx := context.Background()

	// Place entry order first.
	_, err := pb.PlaceOrder(ctx, broker.Order{
		Symbol: "TCS", Exchange: "NSE", Side: broker.OrderSideBuy,
		Type: broker.OrderTypeLimit, Quantity: 5, Price: 3500.0, Product: "CNC",
	})
	if err != nil {
		t.Fatalf("entry order failed: %v", err)
	}

	// Place SL-M order.
	slResp, err := pb.PlaceOrder(ctx, broker.Order{
		Symbol: "TCS", Exchange: "NSE", Side: broker.OrderSideSell,
		Type: broker.OrderTypeSLM, Quantity: 5, TriggerPrice: 3400.0, Product: "CNC",
	})
	if err != nil {
		t.Fatalf("SL order failed: %v", err)
	}

	// SL order should be PENDING (not executed in paper mode).
	if slResp.Status != broker.OrderStatusPending {
		t.Errorf("expected SL order status PENDING, got %s", slResp.Status)
	}

	// Holdings should still exist (SL didn't sell).
	holdings, _ := pb.GetHoldings(ctx)
	if len(holdings) != 1 || holdings[0].Symbol != "TCS" {
		t.Errorf("expected TCS holding to persist after SL placement, got %v", holdings)
	}

	// GetOrderStatus should reflect PENDING.
	slStatus, err := pb.GetOrderStatus(ctx, slResp.OrderID)
	if err != nil {
		t.Fatalf("GetOrderStatus: %v", err)
	}
	if slStatus.Status != broker.OrderStatusPending {
		t.Errorf("expected PENDING, got %s", slStatus.Status)
	}
	if slStatus.PendingQty != 5 {
		t.Errorf("expected PendingQty=5, got %d", slStatus.PendingQty)
	}
}

func TestCancelStopLossOrder_PaperBroker(t *testing.T) {
	logger := log.New(os.Stdout, "[test-sl] ", log.LstdFlags)
	pb := broker.NewPaperBroker(100000)
	ctx := context.Background()

	// Place SL order.
	slResp, _ := pb.PlaceOrder(ctx, broker.Order{
		Symbol: "TCS", Exchange: "NSE", Side: broker.OrderSideSell,
		Type: broker.OrderTypeSLM, Quantity: 5, TriggerPrice: 3400.0, Product: "CNC",
	})

	// Cancel SL order.
	ok := cancelStopLossOrder(ctx, pb, logger, slResp.OrderID, "TCS")
	if !ok {
		t.Error("expected cancel to succeed")
	}

	// Status should be CANCELLED.
	status, _ := pb.GetOrderStatus(ctx, slResp.OrderID)
	if status.Status != broker.OrderStatusCancelled {
		t.Errorf("expected CANCELLED, got %s", status.Status)
	}
}

func TestCancelStopLossOrder_EmptyID(t *testing.T) {
	logger := log.New(os.Stdout, "[test-sl] ", log.LstdFlags)
	pb := broker.NewPaperBroker(100000)
	ctx := context.Background()

	// Empty SL order ID should return true (nothing to cancel).
	ok := cancelStopLossOrder(ctx, pb, logger, "", "TCS")
	if !ok {
		t.Error("expected true for empty SL order ID")
	}
}

// ────────────────────────────────────────────────────────────────────
// Daily PnL Tests
// ────────────────────────────────────────────────────────────────────

func TestCalculateDailyPnL_NilStore(t *testing.T) {
	logger := log.New(os.Stdout, "[test-pnl] ", log.LstdFlags)
	ctx := context.Background()

	// With nil store and nil tradeContext, should return zero PnL.
	pnl := calculateDailyPnL(ctx, nil, nil, nil, logger)
	if pnl.RealizedPnL != 0 || pnl.UnrealizedPnL != 0 {
		t.Errorf("expected zero PnL with nil store, got realized=%.2f unrealized=%.2f",
			pnl.RealizedPnL, pnl.UnrealizedPnL)
	}
}

func TestCalculateDailyPnL_UnrealizedFromHoldings(t *testing.T) {
	logger := log.New(os.Stdout, "[test-pnl] ", log.LstdFlags)
	ctx := context.Background()

	// Holdings with known entry prices.
	tc := &tradeContext{
		trades: map[string]*storage.TradeRecord{
			"RELIANCE": {Symbol: "RELIANCE", EntryPrice: 2500.0},
			"TCS":      {Symbol: "TCS", EntryPrice: 3500.0},
		},
	}

	holdings := []broker.Holding{
		{Symbol: "RELIANCE", Quantity: 10, AveragePrice: 2500.0, LastPrice: 2600.0},
		{Symbol: "TCS", Quantity: 5, AveragePrice: 3500.0, LastPrice: 3400.0},
	}

	pnl := calculateDailyPnL(ctx, nil, tc, holdings, logger)

	// RELIANCE: 10 * (2600 - 2500) = +1000
	// TCS: 5 * (3400 - 3500) = -500
	// Total unrealized: +500
	expectedUnrealized := 500.0
	if pnl.UnrealizedPnL != expectedUnrealized {
		t.Errorf("expected unrealized=%.2f, got %.2f", expectedUnrealized, pnl.UnrealizedPnL)
	}

	// No store means realized PnL is 0.
	if pnl.RealizedPnL != 0 {
		t.Errorf("expected realized=0 with nil store, got %.2f", pnl.RealizedPnL)
	}
}

func TestCalculateDailyPnL_NoTradeContext(t *testing.T) {
	logger := log.New(os.Stdout, "[test-pnl] ", log.LstdFlags)
	ctx := context.Background()

	// Without tradeContext, falls back to broker's AveragePrice.
	holdings := []broker.Holding{
		{Symbol: "INFY", Quantity: 20, AveragePrice: 1400.0, LastPrice: 1450.0},
	}

	pnl := calculateDailyPnL(ctx, nil, nil, holdings, logger)

	// 20 * (1450 - 1400) = 1000
	if pnl.UnrealizedPnL != 1000.0 {
		t.Errorf("expected unrealized=1000.0, got %.2f", pnl.UnrealizedPnL)
	}
}

// ────────────────────────────────────────────────────────────────────
// E2E: Full pipeline with SL orders
// ────────────────────────────────────────────────────────────────────

func TestPaperMode_BuyWithStopLoss(t *testing.T) {
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, "ai_outputs")
	dataDir := filepath.Join(tmpDir, "market_data")
	today := time.Now().In(market.IST).Format("2006-01-02")
	todayDir := filepath.Join(aiDir, today)

	// BULL regime.
	regime := strategy.MarketRegimeData{
		Date: today, Regime: strategy.RegimeBull, Confidence: 0.95,
	}
	writeJSON(t, filepath.Join(todayDir, "market_regime.json"), regime)

	// High-scoring stock.
	scores := []strategy.StockScore{{
		Symbol: "SLTEST", TrendStrengthScore: 0.85, BreakoutQualityScore: 0.90,
		VolatilityScore: 0.50, RiskScore: 0.10, LiquidityScore: 0.80,
		CompositeScore: 0.85, Rank: 1,
	}}
	writeJSON(t, filepath.Join(todayDir, "stock_scores.json"), scores)
	writeTrendingCandles(t, dataDir, "SLTEST", 50, 500.0)

	cfg := &config.Config{
		ActiveBroker: "dhan", TradingMode: config.ModePaper, Capital: 500000.0,
		Risk: config.RiskConfig{
			MaxRiskPerTradePct: 1.0, MaxOpenPositions: 5,
			MaxDailyLossPct: 3.0, MaxCapitalDeploymentPct: 80.0,
		},
		Paths: config.PathsConfig{
			AIOutputDir: aiDir, MarketDataDir: dataDir,
			LogDir: filepath.Join(tmpDir, "logs"),
		},
		DatabaseURL:        "postgres://unused@localhost/unused?sslmode=disable",
		MarketCalendarPath: "",
	}

	logger := log.New(os.Stdout, "[test-sl-e2e] ", log.LstdFlags)
	pb := broker.NewPaperBroker(cfg.Capital)
	riskMgr := risk.NewManager(cfg.Risk, cfg.Capital)
	strats := []strategy.Strategy{strategy.NewTrendFollowStrategy(cfg.Risk)}

	runJobsDirectly(t, cfg, pb, strats, riskMgr, logger)

	ctx := context.Background()
	holdings, _ := pb.GetHoldings(ctx)

	// Should have bought SLTEST.
	if len(holdings) < 1 {
		t.Fatal("expected at least 1 holding after E2E buy")
	}

	found := false
	for _, h := range holdings {
		if h.Symbol == "SLTEST" {
			found = true
			t.Logf("SLTEST: qty=%d avg=%.2f", h.Quantity, h.AveragePrice)
		}
	}
	if !found {
		t.Error("expected SLTEST in holdings")
	}
}
