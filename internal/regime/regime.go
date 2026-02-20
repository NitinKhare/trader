// Package regime provides OHLCV-based market regime detection.
//
// This complements the AI-determined regime (BULL/SIDEWAYS/BEAR) with a
// technical overlay based on price action indicators. The detected regime
// is used by the strategy selector to filter which strategy types to run.
//
// DetectedRegime is distinct from the AI's MarketRegime:
//   - AI regime: BULL/SIDEWAYS/BEAR (directional bias from Python scoring)
//   - Detected regime: TRENDING/RANGING/HIGH_VOLATILITY/LOW_VOLATILITY (from OHLCV)
package regime

import (
	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// DetectedRegime is the OHLCV-derived market regime.
type DetectedRegime string

const (
	RegimeTrending       DetectedRegime = "TRENDING"
	RegimeRanging        DetectedRegime = "RANGING"
	RegimeHighVolatility DetectedRegime = "HIGH_VOLATILITY"
	RegimeLowVolatility  DetectedRegime = "LOW_VOLATILITY"
)

// RegimeState holds the full regime assessment for a stock.
type RegimeState struct {
	Primary    DetectedRegime // The dominant regime classification
	ADX        float64        // Raw ADX value (0-100)
	ATRPctile  float64        // ATR percentile vs lookback (0-1)
	Volatility float64        // Annualized rolling volatility
	Confidence float64        // Confidence in the assessment (0-1)
}

// Detector detects market regime from OHLCV data.
// Stateless: same candles always produce the same result.
type Detector struct {
	ADXPeriod         int     // ADX calculation period (default 14)
	ADXTrendThreshold float64 // ADX > this = trending (default 25)
	ADXRangeThreshold float64 // ADX < this = ranging (default 20)
	ATRPeriod         int     // ATR calculation period (default 14)
	ATRLookback       int     // ATR percentile lookback window (default 60)
	VolHighPctile     float64 // ATR percentile > this = high volatility (default 0.75)
	VolLowPctile      float64 // ATR percentile < this = low volatility (default 0.25)
	VolatilityPeriod  int     // Rolling volatility window (default 20)
}

// NewDetector creates a Detector with sensible defaults.
func NewDetector() *Detector {
	return &Detector{
		ADXPeriod:         14,
		ADXTrendThreshold: 25,
		ADXRangeThreshold: 20,
		ATRPeriod:         14,
		ATRLookback:       60,
		VolHighPctile:     0.85, // was 0.75 — Indian equities are inherently volatile; only flag genuinely extreme conditions
		VolLowPctile:      0.25,
		VolatilityPeriod:  20,
	}
}

// Detect analyzes candles and returns the detected regime.
//
// Logic:
//  1. Compute ADX — if > ADXTrendThreshold: trending; if < ADXRangeThreshold: ranging
//  2. Compute ATR percentile — if > VolHighPctile: high volatility overrides;
//     if < VolLowPctile: low volatility
//  3. High volatility overrides trending/ranging since risk management differs
//  4. Confidence is derived from how decisive the signals are
func (d *Detector) Detect(candles []strategy.Candle) RegimeState {
	state := RegimeState{
		Primary:    RegimeRanging, // default
		Confidence: 0.5,
	}

	if len(candles) < d.ATRLookback+d.ATRPeriod {
		return state
	}

	// Compute indicators.
	state.ADX = strategy.CalculateADX(candles, d.ADXPeriod)
	state.ATRPctile = strategy.CalculateATRPercentile(candles, d.ATRPeriod, d.ATRLookback)
	state.Volatility = strategy.CalculateRollingVolatility(candles, d.VolatilityPeriod)

	// Step 1: Check volatility extremes first (they override trend/range).
	if state.ATRPctile > d.VolHighPctile {
		state.Primary = RegimeHighVolatility
		// Confidence based on how far above the threshold.
		state.Confidence = 0.5 + 0.5*(state.ATRPctile-d.VolHighPctile)/(1-d.VolHighPctile)
		return state
	}
	if state.ATRPctile < d.VolLowPctile {
		state.Primary = RegimeLowVolatility
		state.Confidence = 0.5 + 0.5*(d.VolLowPctile-state.ATRPctile)/d.VolLowPctile
		return state
	}

	// Step 2: Determine trend vs range based on ADX.
	if state.ADX > d.ADXTrendThreshold {
		state.Primary = RegimeTrending
		// Confidence scales from 0.5 at threshold to 1.0 at ADX=50.
		excess := state.ADX - d.ADXTrendThreshold
		maxExcess := 50 - d.ADXTrendThreshold
		if maxExcess > 0 {
			state.Confidence = 0.5 + 0.5*(excess/maxExcess)
			if state.Confidence > 1.0 {
				state.Confidence = 1.0
			}
		}
		return state
	}

	if state.ADX < d.ADXRangeThreshold {
		state.Primary = RegimeRanging
		// Confidence increases as ADX approaches 0.
		if d.ADXRangeThreshold > 0 {
			state.Confidence = 0.5 + 0.5*(d.ADXRangeThreshold-state.ADX)/d.ADXRangeThreshold
		}
		return state
	}

	// ADX is between range and trend thresholds — ambiguous zone.
	// Default to ranging with low confidence.
	state.Primary = RegimeRanging
	state.Confidence = 0.3
	return state
}
