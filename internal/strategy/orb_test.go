package strategy

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// makeORBCandles creates candles simulating a volatile period, then tight
// consolidation, then a breakout candle at the end.
// Layout (for n=50):
//   candles 0 to n-13:  volatile (wide range, ~10 points)
//   candles n-12 to n-2: tight consolidation (range ~0.6 points)
//   candle n-1: breakout above consolidation high with volume
//
// Short ATR (5) on prior candles covers 5 tight candles → low.
// Long ATR (20) on prior candles covers ~8 volatile + 12 tight → high.
// This ensures shortATR/longATR < 0.6 (compression detected).
func makeORBCandles(n int, basePrice float64) []Candle {
	candles := make([]Candle, n)
	tightStart := n - 12

	for i := 0; i < n; i++ {
		price := basePrice
		highSpread := 5.0
		lowSpread := 5.0
		vol := int64(100000)

		if i == n-1 {
			// Breakout candle.
			price = basePrice + 3.0
			highSpread = 2.0
			lowSpread = 1.0
			vol = 250000
		} else if i >= tightStart {
			// Tight consolidation (12 candles before breakout).
			price = basePrice + float64(i%3)*0.1 - float64(i%2)*0.05
			highSpread = 0.3
			lowSpread = 0.3
		} else {
			// Volatile phase.
			price = basePrice + float64(i%5)*3 - float64(i%3)*2
		}

		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   price - 0.5,
			High:   price + highSpread,
			Low:    price - lowSpread,
			Close:  price,
			Volume: vol,
		}
	}
	return candles
}

func TestORB_SkipsNonBullRegime(t *testing.T) {
	s := NewORBStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBear, Confidence: 0.8},
		Score: StockScore{
			Symbol:               "TEST",
			TrendStrengthScore:   0.6,
			BreakoutQualityScore: 0.7,
			LiquidityScore:       0.6,
			RiskScore:            0.2,
		},
		Candles:          makeORBCandles(50, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP in BEAR regime, got %s: %s", result.Action, result.Reason)
	}
}

func TestORB_BuysOnRangeBreakout(t *testing.T) {
	s := NewORBStrategy(makeTestRiskConfig())

	candles := makeORBCandles(50, 100)

	// Verify ATR compression exists.
	shortATR := CalculateATR(candles, 5)
	longATR := CalculateATR(candles, 20)
	if longATR > 0 {
		ratio := shortATR / longATR
		t.Logf("ATR compression ratio: %.4f (threshold: %.2f)", ratio, s.ATRCompressionRatio)
	}

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:               "TEST",
			TrendStrengthScore:   0.6,
			BreakoutQualityScore: 0.7,
			LiquidityScore:       0.6,
			RiskScore:            0.2,
		},
		Candles:          candles,
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionBuy {
		t.Errorf("expected BUY on range breakout, got %s: %s", result.Action, result.Reason)
	}
	if result.StopLoss <= 0 {
		t.Error("expected stop loss to be set")
	}
}

func TestORB_SkipsNoCompression(t *testing.T) {
	s := NewORBStrategy(makeTestRiskConfig())

	// Create volatile candles (no compression).
	candles := make([]Candle, 50)
	for i := 0; i < 50; i++ {
		price := 100.0 + float64(i%10)*5 - float64(i%7)*3
		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   price - 3,
			High:   price + 5,
			Low:    price - 5,
			Close:  price,
			Volume: 200000,
		}
	}

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:               "TEST",
			TrendStrengthScore:   0.6,
			BreakoutQualityScore: 0.7,
			LiquidityScore:       0.6,
			RiskScore:            0.2,
		},
		Candles:          candles,
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action == ActionBuy {
		t.Errorf("expected SKIP for volatile candles (no compression), got BUY: %s", result.Reason)
	}
}

func TestORB_ExitsOnFailedBreakout(t *testing.T) {
	s := NewORBStrategy(makeTestRiskConfig())

	candles := makeORBCandles(50, 100)
	// Make last price below entry.
	candles[len(candles)-1].Close = 98

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.5,
		},
		CurrentPosition: &PositionInfo{
			Symbol:     "TEST",
			EntryPrice: 105,
			Quantity:   10,
		},
		Candles: candles,
	}

	result := s.Evaluate(input)
	if result.Action != ActionExit {
		t.Errorf("expected EXIT on failed breakout, got %s: %s", result.Action, result.Reason)
	}
}

func TestORB_IDAndName(t *testing.T) {
	s := NewORBStrategy(config.RiskConfig{MaxRiskPerTradePct: 1.0})
	if s.ID() != "orb_v1" {
		t.Errorf("unexpected ID: %s", s.ID())
	}
	if s.Name() == "" {
		t.Error("name must not be empty")
	}
}
