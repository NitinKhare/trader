package risk

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

func makeTestRiskConfig() config.RiskConfig {
	return config.RiskConfig{
		MaxRiskPerTradePct:      1.0,
		MaxOpenPositions:        5,
		MaxDailyLossPct:         3.0,
		MaxCapitalDeploymentPct: 80.0,
	}
}

func TestRisk_RejectsNoStopLoss(t *testing.T) {
	mgr := NewManager(makeTestRiskConfig(), 500000)

	intent := strategy.TradeIntent{
		Action:   strategy.ActionBuy,
		Symbol:   "TEST",
		Price:    100,
		StopLoss: 0, // No stop loss.
		Quantity: 10,
	}

	result := mgr.Validate(intent, nil, DailyPnL{}, 500000)

	if result.Approved {
		t.Error("expected rejection for missing stop loss")
	}
	if len(result.Rejections) == 0 {
		t.Error("expected rejection reasons")
	}
	if result.Rejections[0].Rule != "MANDATORY_STOP_LOSS" {
		t.Errorf("expected MANDATORY_STOP_LOSS rule, got %s", result.Rejections[0].Rule)
	}
}

func TestRisk_RejectsStopLossAboveEntry(t *testing.T) {
	mgr := NewManager(makeTestRiskConfig(), 500000)

	intent := strategy.TradeIntent{
		Action:   strategy.ActionBuy,
		Symbol:   "TEST",
		Price:    100,
		StopLoss: 105, // Above entry.
		Quantity: 10,
	}

	result := mgr.Validate(intent, nil, DailyPnL{}, 500000)

	if result.Approved {
		t.Error("expected rejection for invalid stop loss")
	}
}

func TestRisk_RejectsExcessiveRiskPerTrade(t *testing.T) {
	mgr := NewManager(makeTestRiskConfig(), 500000)

	// Risk = (100 - 50) * 200 = 10000 = 2% of 500000 > 1% limit.
	intent := strategy.TradeIntent{
		Action:   strategy.ActionBuy,
		Symbol:   "TEST",
		Price:    100,
		StopLoss: 50,
		Quantity: 200,
	}

	result := mgr.Validate(intent, nil, DailyPnL{}, 500000)

	if result.Approved {
		t.Error("expected rejection for excessive risk per trade")
	}
}

func TestRisk_RejectsExceedingMaxPositions(t *testing.T) {
	mgr := NewManager(makeTestRiskConfig(), 500000)

	// Already at max positions.
	positions := make([]strategy.PositionInfo, 5)
	for i := range positions {
		positions[i] = strategy.PositionInfo{Symbol: "STOCK" + string(rune('A'+i))}
	}

	intent := strategy.TradeIntent{
		Action:   strategy.ActionBuy,
		Symbol:   "NEWSTOCK",
		Price:    100,
		StopLoss: 95,
		Quantity: 10,
	}

	result := mgr.Validate(intent, positions, DailyPnL{}, 500000)

	if result.Approved {
		t.Error("expected rejection for exceeding max positions")
	}
}

func TestRisk_RejectsDuplicatePosition(t *testing.T) {
	mgr := NewManager(makeTestRiskConfig(), 500000)

	positions := []strategy.PositionInfo{
		{Symbol: "TEST", EntryPrice: 100, Quantity: 10},
	}

	intent := strategy.TradeIntent{
		Action:   strategy.ActionBuy,
		Symbol:   "TEST",
		Price:    105,
		StopLoss: 100,
		Quantity: 10,
	}

	result := mgr.Validate(intent, positions, DailyPnL{}, 500000)

	if result.Approved {
		t.Error("expected rejection for duplicate position")
	}
}

func TestRisk_RejectsAtDailyLossLimit(t *testing.T) {
	mgr := NewManager(makeTestRiskConfig(), 500000)

	dailyPnL := DailyPnL{
		Date:        time.Now(),
		RealizedPnL: -15000, // 3% of 500000 = 15000.
	}

	intent := strategy.TradeIntent{
		Action:   strategy.ActionBuy,
		Symbol:   "TEST",
		Price:    100,
		StopLoss: 95,
		Quantity: 10,
	}

	result := mgr.Validate(intent, nil, dailyPnL, 500000)

	if result.Approved {
		t.Error("expected rejection for daily loss limit breach")
	}
}

func TestRisk_ApprovesValidTrade(t *testing.T) {
	mgr := NewManager(makeTestRiskConfig(), 500000)

	intent := strategy.TradeIntent{
		Action:   strategy.ActionBuy,
		Symbol:   "TEST",
		Price:    100,
		StopLoss: 95,
		Quantity: 50, // Risk = 5 * 50 = 250 = 0.05% — well under limit.
	}

	result := mgr.Validate(intent, nil, DailyPnL{}, 500000)

	if !result.Approved {
		t.Errorf("expected approval, got rejections: %v", result.Rejections)
	}
}

func TestRisk_AlwaysAllowsExit(t *testing.T) {
	mgr := NewManager(makeTestRiskConfig(), 500000)

	intent := strategy.TradeIntent{
		Action: strategy.ActionExit,
		Symbol: "TEST",
	}

	// Even at daily loss limit with max positions, exits should be allowed.
	dailyPnL := DailyPnL{RealizedPnL: -20000}
	positions := make([]strategy.PositionInfo, 5)

	result := mgr.Validate(intent, positions, dailyPnL, 0)

	if !result.Approved {
		t.Error("EXIT intents should always be approved")
	}
}

func TestRisk_AlwaysAllowsSkip(t *testing.T) {
	mgr := NewManager(makeTestRiskConfig(), 500000)

	intent := strategy.TradeIntent{
		Action: strategy.ActionSkip,
	}

	result := mgr.Validate(intent, nil, DailyPnL{}, 0)

	if !result.Approved {
		t.Error("SKIP intents should always be approved")
	}
}

func TestRisk_RejectsInsufficientCapital(t *testing.T) {
	mgr := NewManager(makeTestRiskConfig(), 500000)

	intent := strategy.TradeIntent{
		Action:   strategy.ActionBuy,
		Symbol:   "TEST",
		Price:    100,
		StopLoss: 95,
		Quantity: 100,
	}

	result := mgr.Validate(intent, nil, DailyPnL{}, 5000) // Only 5000 available.

	if result.Approved {
		t.Error("expected rejection for insufficient capital")
	}
}
