package portfolio

import (
	"math"
	"testing"
)

func TestAllocator_EqualAllocation(t *testing.T) {
	a := NewAllocator(100000, AllocationEqual)
	ids := []string{"s1", "s2", "s3"}
	allocs := a.Allocate(ids, nil)
	if len(allocs) != 3 {
		t.Fatalf("expected 3 allocations, got %d", len(allocs))
	}
	for _, alloc := range allocs {
		if math.Abs(alloc.CapitalPct-1.0/3.0) > 0.001 {
			t.Errorf("expected ~33%% allocation, got %f", alloc.CapitalPct)
		}
	}
}

func TestAllocator_SharpeWeighted(t *testing.T) {
	a := NewAllocator(100000, AllocationSharpe)
	ids := []string{"s1", "s2"}
	perf := map[string]*StrategyPerformanceSnapshot{
		"s1": {StrategyID: "s1", RollingSharpe: 2.0, TradeCount: 10},
		"s2": {StrategyID: "s2", RollingSharpe: 1.0, TradeCount: 10},
	}
	allocs := a.Allocate(ids, perf)
	var s1Pct, s2Pct float64
	for _, alloc := range allocs {
		if alloc.StrategyID == "s1" {
			s1Pct = alloc.CapitalPct
		}
		if alloc.StrategyID == "s2" {
			s2Pct = alloc.CapitalPct
		}
	}
	if s1Pct <= s2Pct {
		t.Errorf("expected s1 (sharpe=2.0) > s2 (sharpe=1.0): s1=%f s2=%f", s1Pct, s2Pct)
	}
}

func TestAllocator_NegativeSharpe(t *testing.T) {
	a := NewAllocator(100000, AllocationSharpe)
	ids := []string{"s1", "s2"}
	perf := map[string]*StrategyPerformanceSnapshot{
		"s1": {StrategyID: "s1", RollingSharpe: -0.5, TradeCount: 10},
		"s2": {StrategyID: "s2", RollingSharpe: 1.0, TradeCount: 10},
	}
	allocs := a.Allocate(ids, perf)
	var total float64
	for _, alloc := range allocs {
		total += alloc.CapitalPct
	}
	if math.Abs(total-1.0) > 0.01 {
		t.Errorf("allocations should sum to 1.0, got %f", total)
	}
}

func TestAllocator_EmptyStrategies(t *testing.T) {
	a := NewAllocator(100000, AllocationEqual)
	allocs := a.Allocate(nil, nil)
	if allocs != nil {
		t.Errorf("expected nil for empty strategies, got %v", allocs)
	}
}

func TestAllocator_UpdateCapital(t *testing.T) {
	a := NewAllocator(100000, AllocationEqual)
	a.UpdateCapital(200000)
	allocs := a.Allocate([]string{"s1"}, nil)
	if allocs[0].Capital != 200000 {
		t.Errorf("expected 200000, got %f", allocs[0].Capital)
	}
}
