package strategy

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// makeBreakoutCandles creates candles where the last candle breaks above the
// prior N-day high with high volume.
func makeBreakoutCandles(n int, basePrice float64) []Candle {
	candles := make([]Candle, n)
	for i := 0; i < n; i++ {
		price := basePrice + float64(i)*0.5
		vol := int64(100000)
		if i == n-1 {
			// Last candle breaks out above prior range with 2x volume.
			price = basePrice + float64(n)*2.0 // well above prior highs
			vol = 300000                        // 3x normal volume
		}
		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   price - 1,
			High:   price + 2,
			Low:    price - 2,
			Close:  price,
			Volume: vol,
		}
	}
	return candles
}

func TestBreakout_SkipsSidewaysRegime(t *testing.T) {
	s := NewBreakoutStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeSideways, Confidence: 0.8},
		Score: StockScore{
			Symbol:               "TEST",
			BreakoutQualityScore: 0.9,
			TrendStrengthScore:   0.7,
			LiquidityScore:       0.8,
			RiskScore:            0.2,
		},
		Candles:          makeBreakoutCandles(50, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP in SIDEWAYS regime, got %s: %s", result.Action, result.Reason)
	}
}

func TestBreakout_SkipsLowBreakoutQuality(t *testing.T) {
	s := NewBreakoutStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:               "TEST",
			BreakoutQualityScore: 0.4, // below 0.7 threshold
			TrendStrengthScore:   0.7,
			LiquidityScore:       0.8,
			RiskScore:            0.2,
		},
		Candles:          makeBreakoutCandles(50, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP for low breakout quality, got %s: %s", result.Action, result.Reason)
	}
}

func TestBreakout_BuysOnVolumeBreakout(t *testing.T) {
	s := NewBreakoutStrategy(makeTestRiskConfig())

	candles := makeBreakoutCandles(50, 100)

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:               "TEST",
			BreakoutQualityScore: 0.9,
			TrendStrengthScore:   0.7,
			LiquidityScore:       0.8,
			RiskScore:            0.2,
		},
		Candles:          candles,
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionBuy {
		t.Errorf("expected BUY on breakout, got %s: %s", result.Action, result.Reason)
	}
	if result.StopLoss <= 0 {
		t.Error("expected stop loss to be set")
	}
	if result.Target <= result.Price {
		t.Error("expected target above entry price")
	}
}

func TestBreakout_SkipsLowVolume(t *testing.T) {
	s := NewBreakoutStrategy(makeTestRiskConfig())

	// Create candles where last candle breaks above high but volume is normal.
	candles := make([]Candle, 50)
	for i := 0; i < 50; i++ {
		price := 100.0 + float64(i)*0.5
		vol := int64(100000)
		if i == 49 {
			price = 200.0 // above prior highs
			vol = 100000  // same volume — no confirmation
		}
		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   price - 1,
			High:   price + 2,
			Low:    price - 2,
			Close:  price,
			Volume: vol,
		}
	}

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:               "TEST",
			BreakoutQualityScore: 0.9,
			TrendStrengthScore:   0.7,
			LiquidityScore:       0.8,
			RiskScore:            0.2,
		},
		Candles:          candles,
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP for low volume breakout, got %s: %s", result.Action, result.Reason)
	}
}

func TestBreakout_ExitsOnBearRegime(t *testing.T) {
	s := NewBreakoutStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBear, Confidence: 0.8},
		Score:  StockScore{Symbol: "TEST", TrendStrengthScore: 0.5},
		CurrentPosition: &PositionInfo{
			Symbol:     "TEST",
			EntryPrice: 150,
			Quantity:   10,
		},
		Candles: makeBreakoutCandles(50, 100),
	}

	result := s.Evaluate(input)
	if result.Action != ActionExit {
		t.Errorf("expected EXIT in BEAR regime, got %s: %s", result.Action, result.Reason)
	}
}

func TestBreakout_ExitsOnFailedBreakout(t *testing.T) {
	s := NewBreakoutStrategy(makeTestRiskConfig())

	// Create candles where current price is below entry.
	candles := makeTestCandles(50, 100)

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score:  StockScore{Symbol: "TEST", TrendStrengthScore: 0.6},
		CurrentPosition: &PositionInfo{
			Symbol:     "TEST",
			EntryPrice: 200, // Price fell back below entry
			Quantity:   10,
		},
		Candles: candles,
	}

	result := s.Evaluate(input)
	if result.Action != ActionExit {
		t.Errorf("expected EXIT for failed breakout, got %s: %s", result.Action, result.Reason)
	}
}

func TestBreakout_IDAndName(t *testing.T) {
	s := NewBreakoutStrategy(config.RiskConfig{MaxRiskPerTradePct: 1.0})
	if s.ID() != "breakout_v1" {
		t.Errorf("unexpected ID: %s", s.ID())
	}
	if s.Name() == "" {
		t.Error("name must not be empty")
	}
}
