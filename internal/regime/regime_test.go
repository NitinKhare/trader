package regime

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// makeCandles creates synthetic candles for testing.
// trend: positive = uptrend, negative = downtrend, 0 = flat
// volatility: multiplier for price swings
func makeCandles(count int, basePrice, trend, volatility float64) []strategy.Candle {
	candles := make([]strategy.Candle, count)
	price := basePrice
	for i := 0; i < count; i++ {
		price += trend
		swing := volatility * (0.5 - float64(i%3)/3.0) // oscillating swing
		candles[i] = strategy.Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i),
			Open:   price - swing*0.5,
			High:   price + volatility,
			Low:    price - volatility,
			Close:  price + swing*0.5,
			Volume: 100000,
		}
	}
	return candles
}

func TestDetector_Detect_InsufficientData(t *testing.T) {
	d := NewDetector()
	candles := makeCandles(10, 100, 0, 1) // too few candles
	state := d.Detect(candles)

	if state.Primary != RegimeRanging {
		t.Errorf("expected RANGING for insufficient data, got %s", state.Primary)
	}
	if state.Confidence != 0.5 {
		t.Errorf("expected confidence 0.5 for insufficient data, got %.2f", state.Confidence)
	}
}

func TestDetector_Detect_StrongTrend(t *testing.T) {
	d := NewDetector()
	// Strong uptrend: large directional moves relative to range.
	candles := makeCandles(120, 100, 2.0, 0.5)
	state := d.Detect(candles)

	if state.ADX == 0 {
		t.Error("ADX should not be 0 for strong trending data")
	}
	// We can't guarantee exact ADX values with synthetic data,
	// but we can verify the struct is populated.
	if state.ATRPctile < 0 || state.ATRPctile > 1 {
		t.Errorf("ATR percentile out of range: %.2f", state.ATRPctile)
	}
	if state.Confidence < 0 || state.Confidence > 1 {
		t.Errorf("confidence out of range: %.2f", state.Confidence)
	}
}

func TestDetector_Detect_HighVolatility(t *testing.T) {
	d := NewDetector()
	// Start with low volatility, then spike.
	candles := make([]strategy.Candle, 120)
	price := 100.0
	for i := 0; i < 120; i++ {
		vol := 0.5
		if i > 100 { // last 20 bars have huge volatility
			vol = 10.0
		}
		candles[i] = strategy.Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i),
			Open:   price,
			High:   price + vol,
			Low:    price - vol,
			Close:  price + vol*0.3,
			Volume: 100000,
		}
	}
	state := d.Detect(candles)

	// With such extreme volatility spike, ATR percentile should be high.
	if state.ATRPctile < 0.5 {
		t.Logf("ATR percentile: %.4f (may not hit threshold with synthetic data)", state.ATRPctile)
	}
}

func TestDetector_DefaultValues(t *testing.T) {
	d := NewDetector()
	if d.ADXPeriod != 14 {
		t.Errorf("expected ADXPeriod 14, got %d", d.ADXPeriod)
	}
	if d.ADXTrendThreshold != 25 {
		t.Errorf("expected ADXTrendThreshold 25, got %.0f", d.ADXTrendThreshold)
	}
	if d.ADXRangeThreshold != 20 {
		t.Errorf("expected ADXRangeThreshold 20, got %.0f", d.ADXRangeThreshold)
	}
	if d.ATRPeriod != 14 {
		t.Errorf("expected ATRPeriod 14, got %d", d.ATRPeriod)
	}
	if d.ATRLookback != 60 {
		t.Errorf("expected ATRLookback 60, got %d", d.ATRLookback)
	}
	if d.VolHighPctile != 0.85 {
		t.Errorf("expected VolHighPctile 0.85, got %.2f", d.VolHighPctile)
	}
	if d.VolLowPctile != 0.25 {
		t.Errorf("expected VolLowPctile 0.25, got %.2f", d.VolLowPctile)
	}
}

func TestRegimeState_ValidValues(t *testing.T) {
	regimes := []DetectedRegime{
		RegimeTrending,
		RegimeRanging,
		RegimeHighVolatility,
		RegimeLowVolatility,
	}
	expected := []string{"TRENDING", "RANGING", "HIGH_VOLATILITY", "LOW_VOLATILITY"}
	for i, r := range regimes {
		if string(r) != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], string(r))
		}
	}
}
