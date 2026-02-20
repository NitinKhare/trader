package portfolio

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/storage"
)

func TestPerformanceMonitor_Update(t *testing.T) {
	pm := NewPerformanceMonitor(30)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	var trades []storage.TradeRecord
	for i := 0; i < 10; i++ {
		exitTime := base.Add(time.Duration(i+1) * 24 * time.Hour)
		trades = append(trades, storage.TradeRecord{
			StrategyID: "strat1",
			PnL:        100,
			EntryTime:  base.Add(time.Duration(i) * 24 * time.Hour),
			ExitTime:   &exitTime,
			Status:     "closed",
		})
	}
	snapshots := pm.Update(trades)
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
	}
	snap := snapshots["strat1"]
	if snap.TradeCount != 10 {
		t.Errorf("expected 10 trades, got %d", snap.TradeCount)
	}
	if snap.RecentWinRate != 100 {
		t.Errorf("expected 100%% win rate, got %f", snap.RecentWinRate)
	}
}

func TestPerformanceMonitor_EmptyTrades(t *testing.T) {
	pm := NewPerformanceMonitor(30)
	snapshots := pm.Update(nil)
	if len(snapshots) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(snapshots))
	}
}

func TestPerformanceMonitor_GetAllocationWeights(t *testing.T) {
	pm := NewPerformanceMonitor(30)
	snapshots := map[string]*StrategyPerformanceSnapshot{
		"s1": {StrategyID: "s1", RollingSharpe: 2.0, TradeCount: 10},
		"s2": {StrategyID: "s2", RollingSharpe: 1.0, TradeCount: 10},
	}
	weights := pm.GetAllocationWeights(snapshots)
	if len(weights) != 2 {
		t.Fatalf("expected 2 weights, got %d", len(weights))
	}
	if weights["s1"] <= weights["s2"] {
		t.Errorf("expected s1 weight > s2 weight: s1=%f s2=%f", weights["s1"], weights["s2"])
	}
}

func TestPerformanceMonitor_DefaultWindow(t *testing.T) {
	pm := NewPerformanceMonitor(0)
	if pm.windowSize != 30 {
		t.Errorf("expected default window 30, got %d", pm.windowSize)
	}
}
