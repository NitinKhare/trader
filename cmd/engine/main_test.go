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
	registerMarketJobs(sched, cfg, b, strats, riskMgr, nil, logger)
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
