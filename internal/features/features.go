// Package features generates feature vectors from OHLCV candle data using
// technical indicators. These feature vectors can be used for signal scoring
// and machine learning model inputs.
package features

import (
	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// FeatureVector holds computed technical features for a single symbol.
type FeatureVector struct {
	Symbol           string
	RSI14            float64
	ATRPercentile    float64
	VolumeSpikePct   float64
	DistFromVWAP     float64
	ADX14            float64
	RollingReturn5   float64
	RollingReturn20  float64
	VolatilityPctile float64
	BollingerPctB    float64
	MACDHistogram    float64
}

// Generator creates feature vectors from candle data.
type Generator struct{}

// NewGenerator creates a new feature Generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate computes a FeatureVector from the given candle data.
// Returns nil if there is insufficient data to compute the indicators
// (requires at least 60 candles for ATR percentile and rolling calculations).
func (g *Generator) Generate(candles []strategy.Candle) *FeatureVector {
	// Minimum data requirements:
	// - RSI14 needs 15+ candles
	// - ATRPercentile(14, 60) needs 74+ candles
	// - ADX(14) needs 29+ candles
	// - MACD(12,26,9) needs 35+ candles
	// - BollingerBands(20, 2.0) needs 20+ candles
	// - RollingReturn(20) needs 21+ candles
	// Most restrictive is ATRPercentile needing ~74 candles.
	if len(candles) < 75 {
		return nil
	}

	fv := &FeatureVector{}

	// Extract symbol from candles.
	if len(candles) > 0 {
		fv.Symbol = candles[len(candles)-1].Symbol
	}

	// RSI14 = CalculateRSI(candles, 14)
	fv.RSI14 = strategy.CalculateRSI(candles, 14)

	// ATRPercentile = CalculateATRPercentile(candles, 14, 60)
	fv.ATRPercentile = strategy.CalculateATRPercentile(candles, 14, 60)

	// VolumeSpikePct = (lastVolume / AverageVolume(20)) - 1
	lastVolume := float64(candles[len(candles)-1].Volume)
	avgVol := strategy.AverageVolume(candles, 20)
	if avgVol > 0 {
		fv.VolumeSpikePct = (lastVolume / avgVol) - 1
	}

	// DistFromVWAP = (lastClose - CalculateVWAP(20)) / CalculateVWAP(20)
	lastClose := candles[len(candles)-1].Close
	vwap := strategy.CalculateVWAP(candles, 20)
	if vwap > 0 {
		fv.DistFromVWAP = (lastClose - vwap) / vwap
	}

	// ADX14 = CalculateADX(candles, 14)
	fv.ADX14 = strategy.CalculateADX(candles, 14)

	// RollingReturn5 = CalculateRollingReturn(candles, 5)
	fv.RollingReturn5 = strategy.CalculateRollingReturn(candles, 5)

	// RollingReturn20 = CalculateRollingReturn(candles, 20)
	fv.RollingReturn20 = strategy.CalculateRollingReturn(candles, 20)

	// VolatilityPctile = CalculateATRPercentile(candles, 14, 60) (simplified, same as ATRPercentile)
	fv.VolatilityPctile = strategy.CalculateATRPercentile(candles, 14, 60)

	// BollingerPctB = (lastClose - lower) / (upper - lower)
	_, upper, lower, _ := strategy.CalculateBollingerBands(candles, 20, 2.0)
	bandWidth := upper - lower
	if bandWidth > 0 {
		fv.BollingerPctB = (lastClose - lower) / bandWidth
	}

	// MACDHistogram = 3rd return from CalculateMACD(12, 26, 9)
	_, _, histogram := strategy.CalculateMACD(candles, 12, 26, 9)
	fv.MACDHistogram = histogram

	return fv
}
