package strategy

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// makeMomentumCandles creates candles with strong upward momentum (ROC > 5%).
func makeMomentumCandles(n int, basePrice float64) []Candle {
	candles := make([]Candle, n)
	for i := 0; i < n; i++ {
		// Strong upward drift to generate positive ROC.
		price := basePrice + float64(i)*3.0
		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   price - 1,
			High:   price + 4,
			Low:    price - 3,
			Close:  price,
			Volume: 200000,
		}
	}
	return candles
}

func TestMomentum_SkipsBearRegime(t *testing.T) {
	s := NewMomentumStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBear, Confidence: 0.8},
		Score: StockScore{
			Symbol:               "TEST",
			TrendStrengthScore:   0.9,
			BreakoutQualityScore: 0.8,
			LiquidityScore:       0.8,
			RiskScore:            0.1,
			Rank:                 1,
		},
		Candles:          makeMomentumCandles(50, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP in BEAR regime, got %s: %s", result.Action, result.Reason)
	}
}

func TestMomentum_SkipsLowRank(t *testing.T) {
	s := NewMomentumStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:               "TEST",
			TrendStrengthScore:   0.9,
			BreakoutQualityScore: 0.8,
			LiquidityScore:       0.8,
			RiskScore:            0.1,
			Rank:                 20, // rank too low (> 5)
		},
		Candles:          makeMomentumCandles(50, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP for low rank, got %s: %s", result.Action, result.Reason)
	}
}

func TestMomentum_BuysTopRankedMomentum(t *testing.T) {
	s := NewMomentumStrategy(makeTestRiskConfig())

	candles := makeMomentumCandles(50, 100)

	// Verify ROC is above threshold.
	roc := CalculateROC(candles, 10)
	if roc < 0.05 {
		t.Skipf("test data ROC too low: %.4f < 0.05", roc)
	}

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:               "TEST",
			TrendStrengthScore:   0.9,
			BreakoutQualityScore: 0.8,
			LiquidityScore:       0.8,
			RiskScore:            0.1,
			Rank:                 1,
		},
		Candles:          candles,
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionBuy {
		t.Errorf("expected BUY for top-ranked momentum stock, got %s: %s", result.Action, result.Reason)
	}
	if result.StopLoss <= 0 {
		t.Error("expected stop loss to be set")
	}
	if result.Target <= result.Price {
		t.Error("expected target above entry price")
	}
}

func TestMomentum_ExitsOnMomentumLoss(t *testing.T) {
	s := NewMomentumStrategy(makeTestRiskConfig())

	// Create candles with negative ROC (declining prices at end).
	candles := make([]Candle, 50)
	for i := 0; i < 50; i++ {
		var price float64
		if i < 40 {
			price = 100 + float64(i)*2.0
		} else {
			// Last 10 candles decline sharply.
			price = 100 + float64(40)*2.0 - float64(i-40)*5.0
		}
		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   price - 1,
			High:   price + 3,
			Low:    price - 3,
			Close:  price,
			Volume: 200000,
		}
	}

	roc := CalculateROC(candles, 10)
	if roc >= 0 {
		t.Skipf("test data ROC not negative: %.4f (skipping)", roc)
	}

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.6,
			Rank:               3,
		},
		CurrentPosition: &PositionInfo{
			Symbol:     "TEST",
			EntryPrice: 150,
			Quantity:   10,
		},
		Candles: candles,
	}

	result := s.Evaluate(input)
	if result.Action != ActionExit {
		t.Errorf("expected EXIT on momentum reversal (ROC=%.2f%%), got %s: %s",
			roc*100, result.Action, result.Reason)
	}
}

func TestMomentum_ExitsOnRankDrop(t *testing.T) {
	s := NewMomentumStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.7,
			Rank:               15, // dropped out of top 10
		},
		CurrentPosition: &PositionInfo{
			Symbol:     "TEST",
			EntryPrice: 100,
			Quantity:   10,
		},
		Candles: makeMomentumCandles(50, 100),
	}

	result := s.Evaluate(input)
	if result.Action != ActionExit {
		t.Errorf("expected EXIT when rank dropped to 15, got %s: %s", result.Action, result.Reason)
	}
}

func TestMomentum_HoldsTopRankedPosition(t *testing.T) {
	s := NewMomentumStrategy(makeTestRiskConfig())

	candles := makeMomentumCandles(50, 100)

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.8,
			Rank:               2, // still in top 10
		},
		CurrentPosition: &PositionInfo{
			Symbol:     "TEST",
			EntryPrice: 100,
			Quantity:   10,
		},
		Candles: candles,
	}

	result := s.Evaluate(input)
	if result.Action != ActionHold {
		t.Errorf("expected HOLD for top-ranked position, got %s: %s", result.Action, result.Reason)
	}
}

func TestMomentum_IDAndName(t *testing.T) {
	s := NewMomentumStrategy(config.RiskConfig{MaxRiskPerTradePct: 1.0})
	if s.ID() != "momentum_v1" {
		t.Errorf("unexpected ID: %s", s.ID())
	}
	if s.Name() == "" {
		t.Error("name must not be empty")
	}
}
