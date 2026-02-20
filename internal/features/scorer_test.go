package features

import (
	"testing"
)

func TestSignalScorer_NilFeatureVector(t *testing.T) {
	ss := NewSignalScorer()
	score := ss.Score(nil)
	if score != 0.5 {
		t.Errorf("expected 0.5 for nil fv, got %f", score)
	}
}

func TestSignalScorer_NeutralFeatures(t *testing.T) {
	ss := NewSignalScorer()
	fv := &FeatureVector{
		RSI14:          50,
		ATRPercentile:  0.5,
		VolumeSpikePct: 0,
		DistFromVWAP:   0,
		ADX14:          15,
		RollingReturn5:  0,
		RollingReturn20: 0,
		BollingerPctB:  0.5,
		MACDHistogram:  0,
	}
	score := ss.Score(fv)
	if score < 0.3 || score > 0.7 {
		t.Errorf("expected score near 0.5 for neutral features, got %f", score)
	}
}

func TestSignalScorer_ScoreRange(t *testing.T) {
	ss := NewSignalScorer()
	fv := &FeatureVector{
		RSI14:          80,
		ATRPercentile:  0.9,
		VolumeSpikePct: 2.0,
		DistFromVWAP:   0.05,
		ADX14:          40,
		RollingReturn5:  0.1,
		RollingReturn20: 0.05,
		BollingerPctB:  0.8,
		MACDHistogram:  1.0,
	}
	score := ss.Score(fv)
	if score < 0 || score > 1 {
		t.Errorf("score should be in [0,1], got %f", score)
	}
}

func TestSignalScorer_PositiveMomentum(t *testing.T) {
	ss := NewSignalScorer()
	fv := &FeatureVector{
		VolumeSpikePct: 1.0,
		RollingReturn5:  0.5,
		MACDHistogram:  2.0,
	}
	score := ss.Score(fv)
	if score <= 0.5 {
		t.Errorf("expected score > 0.5 for positive momentum, got %f", score)
	}
}

func TestSignalScorer_LoadWeights_FileNotFound(t *testing.T) {
	ss := NewSignalScorer()
	err := ss.LoadWeights("/nonexistent/path.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
