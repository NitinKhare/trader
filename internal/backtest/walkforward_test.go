package backtest

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/analytics"
)

func TestWalkForwardEngine_DefaultConfig(t *testing.T) {
	wfe := NewWalkForwardEngine(WalkForwardConfig{})
	if wfe.config.TrainWindowDays != 252 {
		t.Errorf("expected default train=252, got %d", wfe.config.TrainWindowDays)
	}
	if wfe.config.TestWindowDays != 63 {
		t.Errorf("expected default test=63, got %d", wfe.config.TestWindowDays)
	}
	if wfe.config.StepDays != 63 {
		t.Errorf("expected default step=63, got %d", wfe.config.StepDays)
	}
}

func TestWalkForwardEngine_AllPositiveWindows(t *testing.T) {
	wfe := NewWalkForwardEngine(WalkForwardConfig{
		TrainWindowDays: 30,
		TestWindowDays:  10,
		StepDays:        10,
	})
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(120 * 24 * time.Hour)

	backtestFunc := func(from, to time.Time) *analytics.PerformanceReport {
		return &analytics.PerformanceReport{TotalPnL: 1000}
	}

	result := wfe.Run(start, end, backtestFunc)
	if len(result.Windows) == 0 {
		t.Fatal("expected at least one window")
	}
	if result.Robustness != 100 {
		t.Errorf("expected 100%% robustness, got %f", result.Robustness)
	}
}

func TestWalkForwardEngine_MixedResults(t *testing.T) {
	wfe := NewWalkForwardEngine(WalkForwardConfig{
		TrainWindowDays: 30,
		TestWindowDays:  10,
		StepDays:        10,
	})
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(120 * 24 * time.Hour)

	callCount := 0
	backtestFunc := func(from, to time.Time) *analytics.PerformanceReport {
		callCount++
		if callCount%4 == 0 {
			return &analytics.PerformanceReport{TotalPnL: -500}
		}
		return &analytics.PerformanceReport{TotalPnL: 1000}
	}

	result := wfe.Run(start, end, backtestFunc)
	if result.Robustness == 100 || result.Robustness == 0 {
		t.Errorf("expected mixed robustness, got %f", result.Robustness)
	}
}

func TestWalkForwardEngine_DateRangeTooShort(t *testing.T) {
	wfe := NewWalkForwardEngine(WalkForwardConfig{
		TrainWindowDays: 252,
		TestWindowDays:  63,
		StepDays:        63,
	})
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(30 * 24 * time.Hour)

	backtestFunc := func(from, to time.Time) *analytics.PerformanceReport {
		return &analytics.PerformanceReport{TotalPnL: 1000}
	}

	result := wfe.Run(start, end, backtestFunc)
	if len(result.Windows) != 0 {
		t.Errorf("expected 0 windows for short range, got %d", len(result.Windows))
	}
}
