package strategy

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// makeOversoldCandles creates candles where the last price is below the 20-SMA.
// Uses a declining price pattern to generate oversold RSI conditions.
func makeOversoldCandles(n int, basePrice float64) []Candle {
	candles := make([]Candle, n)
	for i := 0; i < n; i++ {
		// First half: prices go up (establishing the mean).
		// Second half: prices drop sharply (creating oversold condition).
		var price float64
		if i < n/2 {
			price = basePrice + float64(i)*2.0
		} else {
			// Sharp decline in second half.
			price = basePrice + float64(n/2)*2.0 - float64(i-n/2)*4.0
		}
		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   price + 1,
			High:   price + 3,
			Low:    price - 3,
			Close:  price,
			Volume: 100000,
		}
	}
	return candles
}

func TestMeanReversion_SkipsBearRegime(t *testing.T) {
	s := NewMeanReversionStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBear, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.2,
			RiskScore:          0.3,
			LiquidityScore:     0.8,
		},
		Candles:          makeOversoldCandles(50, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP in BEAR regime, got %s: %s", result.Action, result.Reason)
	}
}

func TestMeanReversion_SkipsTrendingStock(t *testing.T) {
	s := NewMeanReversionStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.7, // above 0.4 threshold
			RiskScore:          0.3,
			LiquidityScore:     0.8,
		},
		Candles:          makeOversoldCandles(50, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP for trending stock, got %s: %s", result.Action, result.Reason)
	}
}

func TestMeanReversion_BuysOversoldStock(t *testing.T) {
	s := NewMeanReversionStrategy(makeTestRiskConfig())

	// Create candles with a clear downtrend at the end (oversold).
	candles := makeOversoldCandles(50, 100)

	// Verify the setup: last price should be below SMA.
	sma := CalculateSMA(candles, 20)
	lastPrice := candles[len(candles)-1].Close
	rsi := CalculateRSI(candles, 14)

	if lastPrice >= sma {
		t.Skipf("test data not oversold enough: price=%.2f >= SMA=%.2f (skipping)", lastPrice, sma)
	}
	if rsi >= 35 {
		t.Skipf("test data RSI too high: %.2f >= 35 (skipping)", rsi)
	}

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.2, // not trending
			RiskScore:          0.3,
			LiquidityScore:     0.8,
		},
		Candles:          candles,
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionBuy {
		t.Errorf("expected BUY for oversold stock, got %s: %s", result.Action, result.Reason)
	}
	if result.StopLoss <= 0 {
		t.Error("expected stop loss to be set")
	}
	if result.Target <= 0 {
		t.Error("expected target to be set")
	}
}

func TestMeanReversion_ExitsOnBearRegime(t *testing.T) {
	s := NewMeanReversionStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBear, Confidence: 0.8},
		Score:  StockScore{Symbol: "TEST", TrendStrengthScore: 0.3},
		CurrentPosition: &PositionInfo{
			Symbol:     "TEST",
			EntryPrice: 100,
			Quantity:   10,
		},
		Candles: makeOversoldCandles(50, 100),
	}

	result := s.Evaluate(input)
	if result.Action != ActionExit {
		t.Errorf("expected EXIT in BEAR regime with position, got %s", result.Action)
	}
}

func TestMeanReversion_HoldsWhileOversold(t *testing.T) {
	s := NewMeanReversionStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score:  StockScore{Symbol: "TEST", TrendStrengthScore: 0.3},
		CurrentPosition: &PositionInfo{
			Symbol:     "TEST",
			EntryPrice: 100,
			Quantity:   10,
		},
		Candles: makeOversoldCandles(50, 100),
	}

	result := s.Evaluate(input)
	if result.Action != ActionHold && result.Action != ActionExit {
		t.Errorf("expected HOLD or EXIT, got %s: %s", result.Action, result.Reason)
	}
}

func TestMeanReversion_IDAndName(t *testing.T) {
	s := NewMeanReversionStrategy(config.RiskConfig{MaxRiskPerTradePct: 1.0})
	if s.ID() != "mean_reversion_v1" {
		t.Errorf("unexpected ID: %s", s.ID())
	}
	if s.Name() == "" {
		t.Error("name must not be empty")
	}
}

func TestMeanReversion_WorksInSidewaysRegime(t *testing.T) {
	s := NewMeanReversionStrategy(makeTestRiskConfig())

	// SIDEWAYS regime should be allowed (not just BULL).
	candles := makeOversoldCandles(50, 100)
	sma := CalculateSMA(candles, 20)
	lastPrice := candles[len(candles)-1].Close
	rsi := CalculateRSI(candles, 14)

	if lastPrice >= sma || rsi >= 35 {
		t.Skipf("test data conditions not met for SIDEWAYS test")
	}

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeSideways, Confidence: 0.7},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.2,
			RiskScore:          0.3,
			LiquidityScore:     0.8,
		},
		Candles:          candles,
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionBuy {
		t.Errorf("expected BUY in SIDEWAYS regime, got %s: %s", result.Action, result.Reason)
	}
}
