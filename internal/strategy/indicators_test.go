package strategy

import (
	"math"
	"testing"
	"time"
)

// makeIndicatorCandles creates candles with known prices for indicator testing.
func makeIndicatorCandles(closes []float64) []Candle {
	candles := make([]Candle, len(closes))
	for i, close := range closes {
		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   close - 1,
			High:   close + 2,
			Low:    close - 2,
			Close:  close,
			Volume: 100000 + int64(i*1000),
		}
	}
	return candles
}

func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestCalculateATR_Basic(t *testing.T) {
	candles := makeIndicatorCandles([]float64{
		100, 102, 104, 103, 105, 107, 106, 108, 110, 109,
		111, 113, 112, 114, 116, 115,
	})

	atr := CalculateATR(candles, 14)
	if atr <= 0 {
		t.Errorf("expected positive ATR, got %.4f", atr)
	}
}

func TestCalculateATR_InsufficientData(t *testing.T) {
	candles := makeIndicatorCandles([]float64{100, 102, 104})

	// With only 3 candles and period=14, should fallback to last candle range.
	atr := CalculateATR(candles, 14)
	lastCandle := candles[len(candles)-1]
	expected := lastCandle.High - lastCandle.Low
	if atr != expected {
		t.Errorf("expected fallback ATR %.4f, got %.4f", expected, atr)
	}
}

func TestCalculateATR_EmptyCandles(t *testing.T) {
	atr := CalculateATR(nil, 14)
	if atr != 0 {
		t.Errorf("expected 0 ATR for empty candles, got %.4f", atr)
	}
}

func TestCalculateRSI_Neutral(t *testing.T) {
	// With insufficient data, should return 50 (neutral).
	candles := makeIndicatorCandles([]float64{100, 102, 104})
	rsi := CalculateRSI(candles, 14)
	if rsi != 50 {
		t.Errorf("expected RSI=50 for insufficient data, got %.2f", rsi)
	}
}

func TestCalculateRSI_AllGains(t *testing.T) {
	// All gains → RSI should be 100 or very close.
	prices := make([]float64, 20)
	for i := range prices {
		prices[i] = 100 + float64(i)*2
	}
	candles := makeIndicatorCandles(prices)
	rsi := CalculateRSI(candles, 14)
	if rsi < 95 {
		t.Errorf("expected RSI near 100 for all gains, got %.2f", rsi)
	}
}

func TestCalculateRSI_AllLosses(t *testing.T) {
	// All losses → RSI should be near 0.
	prices := make([]float64, 20)
	for i := range prices {
		prices[i] = 200 - float64(i)*2
	}
	candles := makeIndicatorCandles(prices)
	rsi := CalculateRSI(candles, 14)
	if rsi > 5 {
		t.Errorf("expected RSI near 0 for all losses, got %.2f", rsi)
	}
}

func TestCalculateRSI_Range(t *testing.T) {
	// Mixed data → RSI should be between 0 and 100.
	prices := make([]float64, 30)
	for i := range prices {
		prices[i] = 100 + float64(i%5)*3 - float64(i%3)*2
	}
	candles := makeIndicatorCandles(prices)
	rsi := CalculateRSI(candles, 14)
	if rsi < 0 || rsi > 100 {
		t.Errorf("RSI out of range: %.2f", rsi)
	}
}

func TestCalculateSMA_Basic(t *testing.T) {
	candles := makeIndicatorCandles([]float64{10, 20, 30, 40, 50})
	sma := CalculateSMA(candles, 5)
	expected := (10 + 20 + 30 + 40 + 50) / 5.0
	if !almostEqual(sma, expected, 0.01) {
		t.Errorf("expected SMA=%.2f, got %.2f", expected, sma)
	}
}

func TestCalculateSMA_PartialPeriod(t *testing.T) {
	candles := makeIndicatorCandles([]float64{10, 20, 30})
	// Period 3 with 3 candles: (10+20+30)/3 = 20
	sma := CalculateSMA(candles, 3)
	if !almostEqual(sma, 20, 0.01) {
		t.Errorf("expected SMA=20, got %.2f", sma)
	}
}

func TestCalculateSMA_InsufficientData(t *testing.T) {
	candles := makeIndicatorCandles([]float64{10, 20})
	sma := CalculateSMA(candles, 5) // Not enough candles
	if sma != 0 {
		t.Errorf("expected SMA=0 for insufficient data, got %.2f", sma)
	}
}

func TestCalculateROC_Basic(t *testing.T) {
	// Price went from 100 to 110 over 5 periods → ROC = 10%
	candles := makeIndicatorCandles([]float64{100, 102, 104, 106, 108, 110})
	roc := CalculateROC(candles, 5)
	expected := (110 - 100) / 100.0 // 0.1 = 10%
	if !almostEqual(roc, expected, 0.01) {
		t.Errorf("expected ROC=%.4f, got %.4f", expected, roc)
	}
}

func TestCalculateROC_Negative(t *testing.T) {
	// Price went from 100 to 90 → negative ROC
	candles := makeIndicatorCandles([]float64{100, 98, 96, 94, 92, 90})
	roc := CalculateROC(candles, 5)
	if roc >= 0 {
		t.Errorf("expected negative ROC, got %.4f", roc)
	}
}

func TestCalculateROC_InsufficientData(t *testing.T) {
	candles := makeIndicatorCandles([]float64{100, 102})
	roc := CalculateROC(candles, 5) // Not enough
	if roc != 0 {
		t.Errorf("expected ROC=0 for insufficient data, got %.4f", roc)
	}
}

func TestHighestHigh_Basic(t *testing.T) {
	candles := makeIndicatorCandles([]float64{100, 110, 105, 120, 115})
	// High = Close + 2 for each candle
	hh := HighestHigh(candles, 5)
	expected := 120 + 2.0 // Candle at close=120 has high=122
	if hh != expected {
		t.Errorf("expected HighestHigh=%.2f, got %.2f", expected, hh)
	}
}

func TestLowestLow_Basic(t *testing.T) {
	candles := makeIndicatorCandles([]float64{100, 110, 105, 120, 115})
	// Low = Close - 2 for each candle
	ll := LowestLow(candles, 5)
	expected := 100 - 2.0 // Candle at close=100 has low=98
	if ll != expected {
		t.Errorf("expected LowestLow=%.2f, got %.2f", expected, ll)
	}
}

func TestAverageVolume_Basic(t *testing.T) {
	candles := makeIndicatorCandles([]float64{100, 102, 104, 106, 108})
	avgVol := AverageVolume(candles, 5)
	// Volumes: 100000, 101000, 102000, 103000, 104000
	expected := (100000 + 101000 + 102000 + 103000 + 104000) / 5.0
	if !almostEqual(avgVol, expected, 1) {
		t.Errorf("expected AvgVol=%.0f, got %.0f", expected, avgVol)
	}
}

func TestHighestHigh_Empty(t *testing.T) {
	hh := HighestHigh(nil, 5)
	if hh != 0 {
		t.Errorf("expected 0 for empty candles, got %.2f", hh)
	}
}

func TestLowestLow_Empty(t *testing.T) {
	ll := LowestLow(nil, 5)
	if ll != 0 {
		t.Errorf("expected 0 for empty candles, got %.2f", ll)
	}
}

func TestAverageVolume_Empty(t *testing.T) {
	avgVol := AverageVolume(nil, 5)
	if avgVol != 0 {
		t.Errorf("expected 0 for empty candles, got %.0f", avgVol)
	}
}
