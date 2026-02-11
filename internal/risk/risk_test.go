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

	result := mgr.Validate(intent, nil, DailyPnL{}, 500000, nil)

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

	result := mgr.Validate(intent, nil, DailyPnL{}, 500000, nil)

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

	result := mgr.Validate(intent, nil, DailyPnL{}, 500000, nil)

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

	result := mgr.Validate(intent, positions, DailyPnL{}, 500000, nil)

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

	result := mgr.Validate(intent, positions, DailyPnL{}, 500000, nil)

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

	result := mgr.Validate(intent, nil, dailyPnL, 500000, nil)

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

	result := mgr.Validate(intent, nil, DailyPnL{}, 500000, nil)

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

	result := mgr.Validate(intent, positions, dailyPnL, 0, nil)

	if !result.Approved {
		t.Error("EXIT intents should always be approved")
	}
}

func TestRisk_AlwaysAllowsSkip(t *testing.T) {
	mgr := NewManager(makeTestRiskConfig(), 500000)

	intent := strategy.TradeIntent{
		Action: strategy.ActionSkip,
	}

	result := mgr.Validate(intent, nil, DailyPnL{}, 0, nil)

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

	result := mgr.Validate(intent, nil, DailyPnL{}, 5000, nil) // Only 5000 available.

	if result.Approved {
		t.Error("expected rejection for insufficient capital")
	}
}

func TestRisk_UpdateCapital(t *testing.T) {
	mgr := NewManager(makeTestRiskConfig(), 100000) // Start with 100k

	// Risk per trade = 1% of 100k = 1000
	// Trade risk = (100-95) * 250 = 1250 > 1000 → should reject
	intent := strategy.TradeIntent{
		Action:   strategy.ActionBuy,
		Symbol:   "TEST",
		Price:    100,
		StopLoss: 95,
		Quantity: 250,
	}
	result := mgr.Validate(intent, nil, DailyPnL{}, 500000, nil)
	if result.Approved {
		t.Error("expected rejection at 100k capital")
	}

	// Add money: update capital to 200k
	mgr.UpdateCapital(200000)

	// Now 1% of 200k = 2000 > 1250 → should approve
	result = mgr.Validate(intent, nil, DailyPnL{}, 500000, nil)
	if !result.Approved {
		t.Errorf("expected approval at 200k capital, got: %v", result.Rejections)
	}
}

func TestRisk_UpdateCapital_IgnoresZero(t *testing.T) {
	mgr := NewManager(makeTestRiskConfig(), 100000)

	// Updating with zero or negative should be ignored.
	mgr.UpdateCapital(0)
	mgr.UpdateCapital(-50000)

	// Should still use original 100k: 1% = 1000
	// Trade risk = (100-95) * 150 = 750 < 1000 → should approve
	intent := strategy.TradeIntent{
		Action:   strategy.ActionBuy,
		Symbol:   "TEST",
		Price:    100,
		StopLoss: 95,
		Quantity: 150,
	}
	result := mgr.Validate(intent, nil, DailyPnL{}, 500000, nil)
	if !result.Approved {
		t.Errorf("expected approval with original capital, got: %v", result.Rejections)
	}
}

func TestRisk_SectorConcentration_Allowed(t *testing.T) {
	riskCfg := makeTestRiskConfig()
	riskCfg.MaxPerSector = 2
	mgr := NewManager(riskCfg, 500000)

	// One banking stock already held.
	positions := []strategy.PositionInfo{
		{Symbol: "HDFCBANK", EntryPrice: 1500, Quantity: 10},
	}

	sectorMap := map[string]string{
		"HDFCBANK":  "BANKING",
		"ICICIBANK": "BANKING",
	}

	intent := strategy.TradeIntent{
		Action:   strategy.ActionBuy,
		Symbol:   "ICICIBANK",
		Price:    900,
		StopLoss: 880,
		Quantity: 10,
	}

	result := mgr.Validate(intent, positions, DailyPnL{}, 500000, sectorMap)
	if !result.Approved {
		t.Errorf("expected approval (1 of 2 banking slots used), got: %v", result.Rejections)
	}
}

func TestRisk_SectorConcentration_Blocked(t *testing.T) {
	riskCfg := makeTestRiskConfig()
	riskCfg.MaxPerSector = 2
	mgr := NewManager(riskCfg, 500000)

	// Two banking stocks already held — should block third.
	positions := []strategy.PositionInfo{
		{Symbol: "HDFCBANK", EntryPrice: 1500, Quantity: 10},
		{Symbol: "ICICIBANK", EntryPrice: 900, Quantity: 10},
	}

	sectorMap := map[string]string{
		"HDFCBANK":  "BANKING",
		"ICICIBANK": "BANKING",
		"SBIN":      "BANKING",
	}

	intent := strategy.TradeIntent{
		Action:   strategy.ActionBuy,
		Symbol:   "SBIN",
		Price:    500,
		StopLoss: 490,
		Quantity: 10,
	}

	result := mgr.Validate(intent, positions, DailyPnL{}, 500000, sectorMap)
	if result.Approved {
		t.Error("expected rejection for exceeding sector concentration (2 banking already)")
	}
	foundSectorRule := false
	for _, r := range result.Rejections {
		if r.Rule == "MAX_SECTOR_CONCENTRATION" {
			foundSectorRule = true
		}
	}
	if !foundSectorRule {
		t.Error("expected MAX_SECTOR_CONCENTRATION rejection rule")
	}
}

func TestRisk_SectorConcentration_NilMap(t *testing.T) {
	riskCfg := makeTestRiskConfig()
	riskCfg.MaxPerSector = 2
	mgr := NewManager(riskCfg, 500000)

	intent := strategy.TradeIntent{
		Action:   strategy.ActionBuy,
		Symbol:   "TEST",
		Price:    100,
		StopLoss: 95,
		Quantity: 50,
	}

	// Nil sector map should skip sector check entirely.
	result := mgr.Validate(intent, nil, DailyPnL{}, 500000, nil)
	if !result.Approved {
		t.Errorf("expected approval with nil sector map, got: %v", result.Rejections)
	}
}

func TestRisk_SectorConcentration_Disabled(t *testing.T) {
	riskCfg := makeTestRiskConfig()
	riskCfg.MaxPerSector = 0 // disabled
	mgr := NewManager(riskCfg, 500000)

	positions := []strategy.PositionInfo{
		{Symbol: "HDFCBANK", EntryPrice: 1500, Quantity: 10},
		{Symbol: "ICICIBANK", EntryPrice: 900, Quantity: 10},
	}

	sectorMap := map[string]string{
		"HDFCBANK":  "BANKING",
		"ICICIBANK": "BANKING",
		"SBIN":      "BANKING",
	}

	intent := strategy.TradeIntent{
		Action:   strategy.ActionBuy,
		Symbol:   "SBIN",
		Price:    500,
		StopLoss: 490,
		Quantity: 10,
	}

	// MaxPerSector=0 means disabled — should approve.
	result := mgr.Validate(intent, positions, DailyPnL{}, 500000, sectorMap)
	if !result.Approved {
		t.Errorf("expected approval with sector check disabled, got: %v", result.Rejections)
	}
}
