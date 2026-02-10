package storage

import (
	"encoding/json"
	"testing"

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

func TestLogInputs(t *testing.T) {
	intent := strategy.TradeIntent{
		StrategyID: "trend_follow_v1",
		SignalID:   "sig_20260210_RELIANCE",
		Symbol:     "RELIANCE",
		Action:     strategy.ActionBuy,
		Price:      1250.50,
		StopLoss:   1200.00,
		Target:     1350.00,
		Quantity:   10,
		Reason:     "strong trend",
		Scores: strategy.StockScore{
			TrendStrengthScore:   0.85,
			BreakoutQualityScore: 0.72,
			VolatilityScore:      0.45,
			RiskScore:            0.30,
			LiquidityScore:       0.90,
		},
	}

	result := LogInputs(intent, "BULL")

	// Verify it's valid JSON.
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("LogInputs produced invalid JSON: %v", err)
	}

	// Check key fields.
	if parsed["symbol"] != "RELIANCE" {
		t.Errorf("expected symbol=RELIANCE, got %v", parsed["symbol"])
	}
	if parsed["action"] != "BUY" {
		t.Errorf("expected action=BUY, got %v", parsed["action"])
	}
	if parsed["regime"] != "BULL" {
		t.Errorf("expected regime=BULL, got %v", parsed["regime"])
	}
	if parsed["price"] != 1250.50 {
		t.Errorf("expected price=1250.50, got %v", parsed["price"])
	}
	if parsed["stop_loss"] != 1200.0 {
		t.Errorf("expected stop_loss=1200.0, got %v", parsed["stop_loss"])
	}
	if parsed["quantity"] != float64(10) {
		t.Errorf("expected quantity=10, got %v", parsed["quantity"])
	}

	// Check scores nested map.
	scores, ok := parsed["scores"].(map[string]interface{})
	if !ok {
		t.Fatalf("scores is not a map: %T", parsed["scores"])
	}
	if scores["trend"] != 0.85 {
		t.Errorf("expected trend=0.85, got %v", scores["trend"])
	}
	if scores["breakout"] != 0.72 {
		t.Errorf("expected breakout=0.72, got %v", scores["breakout"])
	}
	if scores["risk"] != 0.30 {
		t.Errorf("expected risk=0.30, got %v", scores["risk"])
	}
}

func TestLogInputs_EmptyIntent(t *testing.T) {
	intent := strategy.TradeIntent{}
	result := LogInputs(intent, "")

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("LogInputs produced invalid JSON for empty intent: %v", err)
	}

	if parsed["symbol"] != "" {
		t.Errorf("expected empty symbol, got %v", parsed["symbol"])
	}
}

func TestNewPostgresStore_EmptyConnStr(t *testing.T) {
	_, err := NewPostgresStore("")
	if err == nil {
		t.Fatal("expected error for empty connection string")
	}
}

func TestNewPostgresStore_BadConnStr(t *testing.T) {
	// This will fail at ping since no server is running.
	_, err := NewPostgresStore("postgres://invalid:invalid@localhost:59999/nonexistent?sslmode=disable&connect_timeout=1")
	if err == nil {
		t.Fatal("expected error for unreachable database")
	}
}
