package features

import (
	"math"
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

func makeTestCandles(count int, basePrice float64) []strategy.Candle {
	candles := make([]strategy.Candle, count)
	price := basePrice
	for i := 0; i < count; i++ {
		volatility := price * 0.02
		candles[i] = strategy.Candle{
			Symbol: "TEST",
			Date:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * 24 * time.Hour),
			Open:   price,
			High:   price + volatility,
			Low:    price - volatility,
			Close:  price + volatility*0.3,
			Volume: 100000 + int64(i*1000),
		}
		price = candles[i].Close
	}
	return candles
}

func TestGenerator_InsufficientData(t *testing.T) {
	gen := NewGenerator()
	fv := gen.Generate(makeTestCandles(30, 100))
	if fv != nil {
		t.Error("expected nil for insufficient data (<75 candles)")
	}
}

func TestGenerator_SufficientData(t *testing.T) {
	gen := NewGenerator()
	fv := gen.Generate(makeTestCandles(100, 100))
	if fv == nil {
		t.Fatal("expected non-nil feature vector for 100 candles")
	}
}

func TestGenerator_SymbolSet(t *testing.T) {
	gen := NewGenerator()
	fv := gen.Generate(makeTestCandles(100, 100))
	if fv.Symbol != "TEST" {
		t.Errorf("expected symbol TEST, got %s", fv.Symbol)
	}
}

func TestGenerator_RSIRange(t *testing.T) {
	gen := NewGenerator()
	fv := gen.Generate(makeTestCandles(100, 100))
	if fv.RSI14 < 0 || fv.RSI14 > 100 {
		t.Errorf("RSI14 should be 0-100, got %f", fv.RSI14)
	}
}

func TestGenerator_ATRPercentileRange(t *testing.T) {
	gen := NewGenerator()
	fv := gen.Generate(makeTestCandles(100, 100))
	if fv.ATRPercentile < 0 || fv.ATRPercentile > 1 {
		t.Errorf("ATRPercentile should be 0-1, got %f", fv.ATRPercentile)
	}
}

func TestGenerator_BollingerPctB(t *testing.T) {
	gen := NewGenerator()
	fv := gen.Generate(makeTestCandles(100, 100))
	if math.IsNaN(fv.BollingerPctB) {
		t.Error("BollingerPctB should not be NaN")
	}
}
