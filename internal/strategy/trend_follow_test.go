package strategy

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

func makeTestCandles(n int, basePrice float64) []Candle {
	candles := make([]Candle, n)
	for i := 0; i < n; i++ {
		price := basePrice + float64(i)*0.5
		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   price - 1,
			High:   price + 2,
			Low:    price - 2,
			Close:  price,
			Volume: 100000,
		}
	}
	return candles
}

func makeTestRiskConfig() config.RiskConfig {
	return config.RiskConfig{
		MaxRiskPerTradePct:     1.0,
		MaxOpenPositions:       5,
		MaxDailyLossPct:        3.0,
		MaxCapitalDeploymentPct: 80.0,
	}
}

func TestTrendFollow_SkipsNonBullRegime(t *testing.T) {
	s := NewTrendFollowStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:             time.Now(),
		Regime:           MarketRegimeData{Regime: RegimeBear, Confidence: 0.8},
		Score:            StockScore{Symbol: "TEST", TrendStrengthScore: 0.9},
		Candles:          makeTestCandles(50, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)

	if result.Action != ActionSkip {
		t.Errorf("expected SKIP in BEAR regime, got %s", result.Action)
	}
}

func TestTrendFollow_SkipsLowTrendStrength(t *testing.T) {
	s := NewTrendFollowStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:              "TEST",
			TrendStrengthScore:  0.3,
			BreakoutQualityScore: 0.7,
			LiquidityScore:      0.8,
			RiskScore:           0.2,
		},
		Candles:          makeTestCandles(50, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)

	if result.Action != ActionSkip {
		t.Errorf("expected SKIP for low trend strength, got %s", result.Action)
	}
}

func TestTrendFollow_BuysOnAllConditionsMet(t *testing.T) {
	s := NewTrendFollowStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:               "TEST",
			TrendStrengthScore:   0.8,
			BreakoutQualityScore: 0.7,
			VolatilityScore:      0.6,
			RiskScore:            0.3,
			LiquidityScore:       0.7,
		},
		Candles:          makeTestCandles(50, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)

	if result.Action != ActionBuy {
		t.Errorf("expected BUY when all conditions met, got %s (reason: %s)", result.Action, result.Reason)
	}
	if result.StopLoss <= 0 {
		t.Error("expected stop loss to be set")
	}
	if result.Target <= result.Price {
		t.Error("expected target above entry price")
	}
	if result.Quantity <= 0 {
		t.Error("expected positive quantity")
	}
}

func TestTrendFollow_ExitsOnBearRegime(t *testing.T) {
	s := NewTrendFollowStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBear, Confidence: 0.8},
		Score:  StockScore{Symbol: "TEST", TrendStrengthScore: 0.5},
		CurrentPosition: &PositionInfo{
			Symbol:     "TEST",
			EntryPrice: 100,
			Quantity:   10,
			StopLoss:   95,
		},
		Candles: makeTestCandles(50, 100),
	}

	result := s.Evaluate(input)

	if result.Action != ActionExit {
		t.Errorf("expected EXIT in BEAR regime with position, got %s", result.Action)
	}
	if result.Quantity != 10 {
		t.Errorf("expected exit full quantity 10, got %d", result.Quantity)
	}
}

func TestTrendFollow_HoldsInBullWithPosition(t *testing.T) {
	s := NewTrendFollowStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score:  StockScore{Symbol: "TEST", TrendStrengthScore: 0.7},
		CurrentPosition: &PositionInfo{
			Symbol:     "TEST",
			EntryPrice: 100,
			Quantity:   10,
			StopLoss:   95,
		},
		Candles: makeTestCandles(50, 100),
	}

	result := s.Evaluate(input)

	if result.Action != ActionHold {
		t.Errorf("expected HOLD with strong trend, got %s", result.Action)
	}
}

func TestTrendFollow_StrategyIsDeterministic(t *testing.T) {
	s := NewTrendFollowStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:               "TEST",
			TrendStrengthScore:   0.8,
			BreakoutQualityScore: 0.7,
			VolatilityScore:      0.6,
			RiskScore:            0.3,
			LiquidityScore:       0.7,
		},
		Candles:          makeTestCandles(50, 100),
		AvailableCapital: 500000,
	}

	result1 := s.Evaluate(input)
	result2 := s.Evaluate(input)

	if result1.Action != result2.Action {
		t.Errorf("strategy is not deterministic: %s vs %s", result1.Action, result2.Action)
	}
	if result1.Quantity != result2.Quantity {
		t.Errorf("strategy is not deterministic: qty %d vs %d", result1.Quantity, result2.Quantity)
	}
	if result1.StopLoss != result2.StopLoss {
		t.Errorf("strategy is not deterministic: SL %.2f vs %.2f", result1.StopLoss, result2.StopLoss)
	}
}

func TestTrendFollow_IDAndName(t *testing.T) {
	s := NewTrendFollowStrategy(makeTestRiskConfig())

	if s.ID() == "" {
		t.Error("strategy ID must not be empty")
	}
	if s.Name() == "" {
		t.Error("strategy name must not be empty")
	}
}
