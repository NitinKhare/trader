package backtest

import (
	"testing"
)

func TestExecutionModel_BasicCosts(t *testing.T) {
	em := NewExecutionModel()
	result := em.Apply(100.0, 110.0, 10, nil)
	if result.TotalCosts <= 0 {
		t.Error("expected positive total costs")
	}
	if result.EntryPrice <= 100.0 {
		t.Errorf("expected adjusted entry > 100, got %f", result.EntryPrice)
	}
	if result.ExitPrice >= 110.0 {
		t.Errorf("expected adjusted exit < 110, got %f", result.ExitPrice)
	}
}

func TestExecutionModel_ZeroQuantity(t *testing.T) {
	em := NewExecutionModel()
	result := em.Apply(100.0, 110.0, 0, nil)
	if result.EntryPrice != 100.0 || result.ExitPrice != 110.0 {
		t.Error("expected original prices for zero quantity")
	}
	if result.TotalCosts != 0 {
		t.Errorf("expected 0 costs for zero qty, got %f", result.TotalCosts)
	}
}

func TestExecutionModel_SlippageDirection(t *testing.T) {
	em := NewExecutionModel()
	result := em.Apply(100.0, 100.0, 100, nil)
	if result.EntryPrice <= 100.0 {
		t.Error("entry should be adjusted UP (worse fill for buy)")
	}
	if result.ExitPrice >= 100.0 {
		t.Error("exit should be adjusted DOWN (worse fill for sell)")
	}
	if result.SlippageCost <= 0 {
		t.Error("expected positive slippage cost")
	}
}

func TestExecutionModel_CostsIncludeSTT(t *testing.T) {
	em := NewExecutionModel()
	result := em.Apply(1000.0, 1100.0, 10, nil)
	if result.TotalCosts < 10 {
		t.Errorf("expected significant costs, got %f", result.TotalCosts)
	}
}
