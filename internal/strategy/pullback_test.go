package strategy

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// makePullbackCandles creates candles simulating an uptrend with a pullback.
// First 50 candles trend up moderately, then the last 20 pull back.
// This creates a scenario where price is above 50-EMA but near 20-EMA with
// an RSI in the healthy pullback range (40-60).
func makePullbackCandles(n int, basePrice float64) []Candle {
	candles := make([]Candle, n)
	for i := 0; i < n; i++ {
		var price float64
		if i < n-20 {
			// Moderate uptrend.
			price = basePrice + float64(i)*1.0
		} else {
			// Extended pullback: 20 candles of decline to bring RSI down.
			peak := basePrice + float64(n-21)*1.0
			price = peak - float64(i-(n-20))*0.5
		}
		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   price - 1,
			High:   price + 3,
			Low:    price - 2,
			Close:  price,
			Volume: 150000,
		}
	}
	return candles
}

func TestPullback_SkipsNonBullRegime(t *testing.T) {
	s := NewPullbackStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeSideways, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.7,
			LiquidityScore:     0.6,
			RiskScore:          0.2,
		},
		Candles:          makePullbackCandles(70, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP in SIDEWAYS regime, got %s: %s", result.Action, result.Reason)
	}
}

func TestPullback_SkipsInsufficientHistory(t *testing.T) {
	s := NewPullbackStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.7,
			LiquidityScore:     0.6,
			RiskScore:          0.2,
		},
		Candles:          makePullbackCandles(30, 100), // needs 60
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP for insufficient history, got %s: %s", result.Action, result.Reason)
	}
}

func TestPullback_BuysOnPullbackToEMA(t *testing.T) {
	s := NewPullbackStrategy(makeTestRiskConfig())
	// Widen the pullback tolerance and RSI range for test data.
	s.PullbackPct = 5.0
	s.RSILow = 30
	s.RSIHigh = 75

	candles := makePullbackCandles(70, 100)

	// Verify conditions: price > 50-EMA and near 20-EMA.
	slowEMA := CalculateEMA(candles, 50)
	lastPrice := candles[len(candles)-1].Close
	if lastPrice <= slowEMA {
		t.Skipf("test data: price %.2f not above 50-EMA %.2f", lastPrice, slowEMA)
	}

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.7,
			LiquidityScore:     0.6,
			RiskScore:          0.2,
		},
		Candles:          candles,
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionBuy {
		t.Errorf("expected BUY on pullback, got %s: %s", result.Action, result.Reason)
	}
	if result.StopLoss <= 0 {
		t.Error("expected stop loss to be set")
	}
	if result.Target <= result.Price {
		t.Error("expected target above entry price")
	}
}

func TestPullback_ExitsBelowSlowEMA(t *testing.T) {
	s := NewPullbackStrategy(makeTestRiskConfig())

	// Create candles where price drops below 50-EMA.
	candles := make([]Candle, 70)
	for i := 0; i < 70; i++ {
		var price float64
		if i < 50 {
			price = 100 + float64(i)*1.5
		} else {
			// Sharp decline.
			price = 100 + float64(50)*1.5 - float64(i-50)*8.0
		}
		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   price - 1,
			High:   price + 2,
			Low:    price - 2,
			Close:  price,
			Volume: 150000,
		}
	}

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.4,
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
		t.Errorf("expected EXIT below 50-EMA, got %s: %s", result.Action, result.Reason)
	}
}

func TestPullback_IDAndName(t *testing.T) {
	s := NewPullbackStrategy(config.RiskConfig{MaxRiskPerTradePct: 1.0})
	if s.ID() != "pullback_v1" {
		t.Errorf("unexpected ID: %s", s.ID())
	}
	if s.Name() == "" {
		t.Error("name must not be empty")
	}
}
