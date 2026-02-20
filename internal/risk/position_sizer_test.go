package risk

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

func makeSizerCandles(count int, basePrice, volatility float64) []strategy.Candle {
	candles := make([]strategy.Candle, count)
	for i := 0; i < count; i++ {
		candles[i] = strategy.Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i),
			Open:   basePrice,
			High:   basePrice + volatility,
			Low:    basePrice - volatility,
			Close:  basePrice,
			Volume: 100000,
		}
	}
	return candles
}

func TestPositionSizer_BasicSizing(t *testing.T) {
	ps := NewPositionSizer(PositionSizerConfig{
		ATRMultiplier:   1.5,
		RiskPerTradePct: 1.0,
		MaxPositionPct:  10.0,
	})

	// ATR for candles with volatility=5 → high-low = 10, ATR ≈ 10
	candles := makeSizerCandles(30, 100, 5)
	result := ps.Size(500000, 100, candles)

	if result.Quantity <= 0 {
		t.Error("expected positive quantity")
	}
	if result.StopLoss <= 0 {
		t.Error("expected positive stop loss")
	}
	if result.StopLoss >= 100 {
		t.Errorf("stop loss %.2f should be below entry 100", result.StopLoss)
	}
	if result.OrderValue <= 0 {
		t.Error("expected positive order value")
	}
	// Risk amount should not exceed 1% of 500k = 5000.
	if result.RiskAmount > 5100 { // small tolerance
		t.Errorf("risk amount %.2f exceeds 1%% of equity (5000)", result.RiskAmount)
	}
}

func TestPositionSizer_HighVolatilityReducesSize(t *testing.T) {
	ps := NewPositionSizer(PositionSizerConfig{
		ATRMultiplier:   1.5,
		RiskPerTradePct: 1.0,
		MaxPositionPct:  10.0,
	})

	lowVolCandles := makeSizerCandles(30, 100, 2)   // ATR ≈ 4
	highVolCandles := makeSizerCandles(30, 100, 10)  // ATR ≈ 20

	lowVolResult := ps.Size(500000, 100, lowVolCandles)
	highVolResult := ps.Size(500000, 100, highVolCandles)

	if highVolResult.Quantity >= lowVolResult.Quantity {
		t.Errorf("high volatility should reduce quantity: low=%d high=%d",
			lowVolResult.Quantity, highVolResult.Quantity)
	}
	if highVolResult.StopDistance <= lowVolResult.StopDistance {
		t.Errorf("high volatility should increase stop distance: low=%.2f high=%.2f",
			lowVolResult.StopDistance, highVolResult.StopDistance)
	}
}

func TestPositionSizer_MaxPositionCap(t *testing.T) {
	ps := NewPositionSizer(PositionSizerConfig{
		ATRMultiplier:   0.1,       // tiny stop distance → lots of risk-based shares
		RiskPerTradePct: 5.0,       // 5% risk → lots of shares
		MaxPositionPct:  5.0,       // but cap at 5% of equity
	})

	candles := makeSizerCandles(30, 100, 0.5) // very low volatility
	result := ps.Size(500000, 100, candles)

	// Max position value = 5% of 500k = 25000
	// At price 100, max qty = 250
	if result.OrderValue > 25100 { // small tolerance
		t.Errorf("order value %.2f exceeds max position cap 25000", result.OrderValue)
	}
}

func TestPositionSizer_ZeroEquity(t *testing.T) {
	ps := NewPositionSizer(PositionSizerConfig{})
	candles := makeSizerCandles(30, 100, 5)
	result := ps.Size(0, 100, candles)
	if result.Quantity != 0 {
		t.Errorf("expected 0 quantity for zero equity, got %d", result.Quantity)
	}
}

func TestPositionSizer_EmptyCandles(t *testing.T) {
	ps := NewPositionSizer(PositionSizerConfig{})
	result := ps.Size(500000, 100, nil)
	if result.Quantity != 0 {
		t.Errorf("expected 0 quantity for nil candles, got %d", result.Quantity)
	}
}

func TestPositionSizer_DefaultConfig(t *testing.T) {
	ps := NewPositionSizer(PositionSizerConfig{})
	if ps.config.ATRMultiplier != 1.5 {
		t.Errorf("expected default ATRMultiplier 1.5, got %.1f", ps.config.ATRMultiplier)
	}
	if ps.config.RiskPerTradePct != 1.0 {
		t.Errorf("expected default RiskPerTradePct 1.0, got %.1f", ps.config.RiskPerTradePct)
	}
	if ps.config.MaxPositionPct != 10.0 {
		t.Errorf("expected default MaxPositionPct 10.0, got %.1f", ps.config.MaxPositionPct)
	}
}
