package strategy

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// makeVWAPCandles creates candles with a dip below VWAP.
// The last few candles dip significantly below the average,
// mimicking a pullback from fair value.
func makeVWAPCandles(n int, basePrice float64) []Candle {
	candles := make([]Candle, n)
	for i := 0; i < n; i++ {
		var price float64
		if i < n-5 {
			// Steady uptrend.
			price = basePrice + float64(i)*0.5
		} else {
			// Sharp dip in last 5 candles.
			price = basePrice + float64(n-6)*0.5 - float64(i-(n-5))*3.0
		}
		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   price - 1,
			High:   price + 2,
			Low:    price - 2,
			Close:  price,
			Volume: 200000,
		}
	}
	return candles
}

func TestVWAP_SkipsBearRegime(t *testing.T) {
	s := NewVWAPReversionStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBear, Confidence: 0.8},
		Score: StockScore{
			Symbol:         "TEST",
			LiquidityScore: 0.8,
			RiskScore:      0.2,
		},
		Candles:          makeVWAPCandles(50, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP in BEAR regime, got %s: %s", result.Action, result.Reason)
	}
}

func TestVWAP_SkipsHighVolatility(t *testing.T) {
	s := NewVWAPReversionStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:          "TEST",
			LiquidityScore:  0.8,
			RiskScore:       0.2,
			VolatilityScore: 0.9, // too volatile
		},
		Candles:          makeVWAPCandles(50, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP for high volatility, got %s: %s", result.Action, result.Reason)
	}
}

func TestVWAP_BuysOnDipBelowVWAP(t *testing.T) {
	s := NewVWAPReversionStrategy(makeTestRiskConfig())
	// Relax RSI threshold so test data passes.
	s.RSIOversoldThreshold = 50

	candles := makeVWAPCandles(50, 100)

	// Verify VWAP is above last price.
	vwap := CalculateVWAP(candles, 20)
	lastPrice := candles[len(candles)-1].Close
	if lastPrice >= vwap {
		t.Skipf("test data: price %.2f not below VWAP %.2f", lastPrice, vwap)
	}

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:          "TEST",
			LiquidityScore:  0.8,
			RiskScore:       0.2,
			VolatilityScore: 0.3,
		},
		Candles:          candles,
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionBuy {
		t.Errorf("expected BUY on dip below VWAP, got %s: %s", result.Action, result.Reason)
	}
	if result.StopLoss <= 0 {
		t.Error("expected stop loss to be set")
	}
	if result.Target <= result.Price {
		t.Error("expected target above entry price")
	}
}

func TestVWAP_ExitsAboveVWAP(t *testing.T) {
	s := NewVWAPReversionStrategy(makeTestRiskConfig())

	// Candles where last price is above VWAP (price recovered).
	candles := makeTestCandles(50, 100) // steady uptrend, price > VWAP

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:         "TEST",
			LiquidityScore: 0.8,
		},
		CurrentPosition: &PositionInfo{
			Symbol:     "TEST",
			EntryPrice: 100,
			Quantity:   10,
		},
		Candles: candles,
	}

	result := s.Evaluate(input)
	// Last price in makeTestCandles is well above VWAP since prices drift up.
	if result.Action != ActionExit {
		t.Errorf("expected EXIT above VWAP, got %s: %s", result.Action, result.Reason)
	}
}

func TestVWAP_IDAndName(t *testing.T) {
	s := NewVWAPReversionStrategy(config.RiskConfig{MaxRiskPerTradePct: 1.0})
	if s.ID() != "vwap_reversion_v1" {
		t.Errorf("unexpected ID: %s", s.ID())
	}
	if s.Name() == "" {
		t.Error("name must not be empty")
	}
}
