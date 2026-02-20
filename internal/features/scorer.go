// Package features - scorer.go provides a configurable signal scorer that
// converts a FeatureVector into a 0-1 probability score using a weighted
// linear combination passed through a sigmoid function.
package features

import (
	"encoding/json"
	"math"
	"os"
)

// SignalScorer scores feature vectors using a weighted linear model with sigmoid activation.
type SignalScorer struct {
	weights map[string]float64
	bias    float64
}

// NewSignalScorer creates a SignalScorer with heuristic default weights.
//
// Default weights are tuned for Indian equity swing trading:
//   - RSI14: -0.08 (lower RSI better for mean reversion entries)
//   - ATRPercentile: -0.05 (moderate volatility preferred over extremes)
//   - VolumeSpikePct: 0.12 (volume confirmation is positive)
//   - DistFromVWAP: -0.15 (below VWAP strongly preferred for value entries)
//   - ADX14: 0.25 (trend strength is the strongest predictor of winning trades)
//   - RollingReturn5: 0.08 (recent momentum, slight weight)
//   - RollingReturn20: 0.10 (medium-term trend confirmation)
//   - BollingerPctB: 0.03 (position within bands, weak signal alone)
//   - MACDHistogram: 0.15 (positive histogram preferred)
func NewSignalScorer() *SignalScorer {
	return &SignalScorer{
		weights: map[string]float64{
			"RSI14":          -0.08,  // was -0.02: reward oversold (lower RSI = better entry)
			"ATRPercentile":  -0.05,  // was -0.1: moderate vol preferred, less penalty
			"VolumeSpikePct": 0.12,   // was 0.15: volume confirms, slightly reduced
			"DistFromVWAP":   -0.15,  // was -0.1: below VWAP is strong value signal
			"ADX14":          0.25,   // was 0.02: trend strength is critical predictor
			"RollingReturn5":  0.08,  // was 0.1: reduce short-term noise weight
			"RollingReturn20": 0.10,  // was 0.05: medium-term momentum matters more
			"BollingerPctB":  0.03,   // was 0.05: less predictive on its own
			"MACDHistogram":  0.15,   // was 0.2: still important, slightly reduced
		},
		bias: 0.05, // was 0.0: slight positive bias to reduce false negatives
	}
}

// Score computes a 0-1 signal score for the given feature vector.
// The score is computed as sigmoid(weighted_sum + bias).
// Returns 0.5 (neutral) if the feature vector is nil.
func (ss *SignalScorer) Score(fv *FeatureVector) float64 {
	if fv == nil {
		return 0.5
	}

	// Build feature map from the vector.
	features := map[string]float64{
		"RSI14":          fv.RSI14,
		"ATRPercentile":  fv.ATRPercentile,
		"VolumeSpikePct": fv.VolumeSpikePct,
		"DistFromVWAP":   fv.DistFromVWAP,
		"ADX14":          fv.ADX14,
		"RollingReturn5":  fv.RollingReturn5,
		"RollingReturn20": fv.RollingReturn20,
		"BollingerPctB":  fv.BollingerPctB,
		"MACDHistogram":  fv.MACDHistogram,
	}

	// Compute weighted sum.
	var sum float64
	for name, value := range features {
		if weight, ok := ss.weights[name]; ok {
			sum += weight * value
		}
	}
	sum += ss.bias

	// Apply sigmoid: 1 / (1 + exp(-x))
	return sigmoid(sum)
}

// LoadWeights reads a JSON file containing custom weights and bias, overriding
// the current scorer configuration.
//
// Expected JSON format:
//
//	{
//	    "weights": {
//	        "RSI14": -0.02,
//	        "ATRPercentile": -0.1,
//	        ...
//	    },
//	    "bias": 0.0
//	}
//
// Only weights present in the JSON file are updated; missing weights retain
// their current values.
func (ss *SignalScorer) LoadWeights(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var config struct {
		Weights map[string]float64 `json:"weights"`
		Bias    float64            `json:"bias"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Override weights that are present in the loaded config.
	if config.Weights != nil {
		for k, v := range config.Weights {
			ss.weights[k] = v
		}
	}
	ss.bias = config.Bias

	return nil
}

// sigmoid computes the logistic sigmoid function: 1 / (1 + exp(-x)).
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}
