package portfolio

import (
	"math"
	"testing"

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

func makeCandles(prices []float64) []strategy.Candle {
	candles := make([]strategy.Candle, len(prices))
	for i, p := range prices {
		candles[i] = strategy.Candle{Close: p, High: p + 1, Low: p - 1, Open: p, Volume: 1000}
	}
	return candles
}

func TestCorrelationEngine_PerfectCorrelation(t *testing.T) {
	ce := NewCorrelationEngine(0.75, 30)
	prices := []float64{100, 102, 104, 106, 108, 110, 112, 114, 116, 118, 120}
	data := map[string][]strategy.Candle{
		"A": makeCandles(prices),
		"B": makeCandles(prices),
	}
	matrix := ce.ComputeCorrelation(data)
	var aIdx, bIdx int
	for i, sym := range matrix.Symbols {
		if sym == "A" {
			aIdx = i
		}
		if sym == "B" {
			bIdx = i
		}
	}
	corr := matrix.Values[aIdx][bIdx]
	if corr < 0.99 {
		t.Errorf("expected near-perfect correlation, got %f", corr)
	}
}

func TestCorrelationEngine_SelfCorrelation(t *testing.T) {
	ce := NewCorrelationEngine(0.75, 30)
	data := map[string][]strategy.Candle{
		"A": makeCandles([]float64{100, 102, 104, 106, 108}),
	}
	matrix := ce.ComputeCorrelation(data)
	if matrix.Values[0][0] != 1.0 {
		t.Errorf("expected self-correlation of 1.0, got %f", matrix.Values[0][0])
	}
}

func TestCorrelationEngine_ShouldReduceAllocation(t *testing.T) {
	ce := NewCorrelationEngine(0.75, 30)
	prices := []float64{100, 102, 104, 106, 108, 110, 112, 114, 116, 118, 120}
	data := map[string][]strategy.Candle{
		"A": makeCandles(prices),
		"B": makeCandles(prices),
	}
	matrix := ce.ComputeCorrelation(data)
	reduce, _, corrVal := ce.ShouldReduceAllocation("B", []string{"A"}, matrix)
	if !reduce {
		t.Errorf("expected reduce=true for perfectly correlated stocks, corr=%f", corrVal)
	}
}

func TestCorrelationEngine_ShouldNotReduce(t *testing.T) {
	ce := NewCorrelationEngine(0.75, 30)
	// A goes up steadily, B oscillates independently
	data := map[string][]strategy.Candle{
		"A": makeCandles([]float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110}),
		"B": makeCandles([]float64{100, 95, 105, 90, 110, 85, 115, 80, 120, 75, 125}),
	}
	matrix := ce.ComputeCorrelation(data)
	reduce, _, _ := ce.ShouldReduceAllocation("B", []string{"A"}, matrix)
	if reduce {
		t.Error("expected reduce=false for uncorrelated stocks")
	}
}

func TestCorrelationEngine_NilMatrix(t *testing.T) {
	ce := NewCorrelationEngine(0.75, 30)
	reduce, _, _ := ce.ShouldReduceAllocation("A", []string{"B"}, nil)
	if reduce {
		t.Error("expected reduce=false for nil matrix")
	}
}

func TestCorrelationEngine_DefaultConfig(t *testing.T) {
	ce := NewCorrelationEngine(0, 0)
	if ce.maxCorrelation != 0.75 {
		t.Errorf("expected default maxCorrelation 0.75, got %f", ce.maxCorrelation)
	}
	if ce.lookbackDays != 30 {
		t.Errorf("expected default lookbackDays 30, got %d", ce.lookbackDays)
	}
}

func TestCorrelationEngine_UncorrelatedData(t *testing.T) {
	ce := NewCorrelationEngine(0.75, 30)
	// A trends up steadily, B oscillates with growing amplitude
	data := map[string][]strategy.Candle{
		"A": makeCandles([]float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110}),
		"B": makeCandles([]float64{100, 95, 105, 90, 110, 85, 115, 80, 120, 75, 125}),
	}
	matrix := ce.ComputeCorrelation(data)
	var aIdx, bIdx int
	for i, sym := range matrix.Symbols {
		if sym == "A" {
			aIdx = i
		}
		if sym == "B" {
			bIdx = i
		}
	}
	corr := math.Abs(matrix.Values[aIdx][bIdx])
	if corr > 0.5 {
		t.Errorf("expected low correlation, got %f", corr)
	}
}
