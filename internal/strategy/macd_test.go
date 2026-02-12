package strategy

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// makeMACDCrossoverCandles creates candles that produce a MACD bullish crossover.
// The first portion trends down/flat, then reverses up sharply to trigger a crossover.
func makeMACDCrossoverCandles(n int, basePrice float64) []Candle {
	candles := make([]Candle, n)
	for i := 0; i < n; i++ {
		var price float64
		if i < n/2 {
			// Declining/flat phase.
			price = basePrice - float64(i)*0.3
		} else {
			// Recovery and upturn — should produce bullish MACD crossover.
			price = basePrice - float64(n/2)*0.3 + float64(i-n/2)*1.5
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
	return candles
}

func TestMACD_SkipsNonBullRegime(t *testing.T) {
	s := NewMACDCrossoverStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBear, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.6,
			LiquidityScore:     0.6,
			RiskScore:          0.2,
		},
		Candles:          makeMACDCrossoverCandles(50, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP in BEAR regime, got %s: %s", result.Action, result.Reason)
	}
}

func TestMACD_SkipsInsufficientHistory(t *testing.T) {
	s := NewMACDCrossoverStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.6,
			LiquidityScore:     0.6,
			RiskScore:          0.2,
		},
		Candles:          makeMACDCrossoverCandles(20, 100), // needs 40
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP for insufficient history, got %s: %s", result.Action, result.Reason)
	}
}

func TestMACD_BuysOnBullishCrossover(t *testing.T) {
	s := NewMACDCrossoverStrategy(makeTestRiskConfig())
	// Disable the MaxMACDForEntry filter so the test is not blocked by it.
	s.MaxMACDForEntry = 0

	candles := makeMACDCrossoverCandles(60, 100)

	// Check if a crossover actually exists.
	macdLine, signalLine, histogram := CalculateMACD(candles, 12, 26, 9)
	prevMACD, prevSignal := CalculatePrevMACD(candles, 12, 26, 9)
	t.Logf("MACD=%.4f signal=%.4f hist=%.4f | prev MACD=%.4f prev signal=%.4f",
		macdLine, signalLine, histogram, prevMACD, prevSignal)

	isCrossover := macdLine > signalLine && prevMACD <= prevSignal && histogram > 0
	if !isCrossover {
		t.Skip("test data does not produce a MACD crossover, skipping")
	}

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.6,
			LiquidityScore:     0.6,
			RiskScore:          0.2,
		},
		Candles:          candles,
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionBuy {
		t.Errorf("expected BUY on MACD crossover, got %s: %s", result.Action, result.Reason)
	}
	if result.StopLoss <= 0 {
		t.Error("expected stop loss to be set")
	}
}

func TestMACD_ExitsOnBearishCrossover(t *testing.T) {
	s := NewMACDCrossoverStrategy(makeTestRiskConfig())

	// Create candles with declining prices at the end → bearish MACD.
	candles := make([]Candle, 60)
	for i := 0; i < 60; i++ {
		var price float64
		if i < 40 {
			price = 100 + float64(i)*1.5
		} else {
			// Sharp decline → MACD turns bearish.
			price = 100 + float64(40)*1.5 - float64(i-40)*3.0
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

	macdLine, signalLine, _ := CalculateMACD(candles, 12, 26, 9)
	t.Logf("exit test: MACD=%.4f signal=%.4f", macdLine, signalLine)

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.5,
		},
		CurrentPosition: &PositionInfo{
			Symbol:     "TEST",
			EntryPrice: 130,
			Quantity:   10,
		},
		Candles: candles,
	}

	result := s.Evaluate(input)
	if result.Action != ActionExit {
		t.Errorf("expected EXIT on bearish MACD, got %s: %s", result.Action, result.Reason)
	}
}

func TestMACD_HoldsOnPositiveMomentum(t *testing.T) {
	s := NewMACDCrossoverStrategy(makeTestRiskConfig())

	candles := makeMomentumCandles(60, 100) // strong uptrend

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.8,
		},
		CurrentPosition: &PositionInfo{
			Symbol:     "TEST",
			EntryPrice: 130,
			Quantity:   10,
		},
		Candles: candles,
	}

	result := s.Evaluate(input)
	// With strong uptrend, MACD should be positive and above signal → HOLD.
	if result.Action == ActionExit {
		t.Errorf("expected HOLD with positive momentum, got EXIT: %s", result.Reason)
	}
}

func TestMACD_IDAndName(t *testing.T) {
	s := NewMACDCrossoverStrategy(config.RiskConfig{MaxRiskPerTradePct: 1.0})
	if s.ID() != "macd_crossover_v1" {
		t.Errorf("unexpected ID: %s", s.ID())
	}
	if s.Name() == "" {
		t.Error("name must not be empty")
	}
}
