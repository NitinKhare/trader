package analytics

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/regime"
	"github.com/nitinkhare/algoTradingAgent/internal/storage"
	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

func TestAnalyzeByRegime_AIRegime(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	exitTime1 := base.Add(24 * time.Hour)
	exitTime2 := base.Add(48 * time.Hour)
	trades := []storage.TradeRecord{
		{StrategyID: "s1", Symbol: "A", EntryPrice: 100, ExitPrice: 110, PnL: 100, EntryTime: base, ExitTime: &exitTime1, Status: "closed"},
		{StrategyID: "s1", Symbol: "B", EntryPrice: 100, ExitPrice: 90, PnL: -100, EntryTime: base.Add(24 * time.Hour), ExitTime: &exitTime2, Status: "closed"},
	}
	aiHistory := map[string]strategy.MarketRegime{
		"2024-01-01": strategy.RegimeBull,
		"2024-01-02": strategy.RegimeBear,
	}
	detHistory := map[string]regime.DetectedRegime{}

	result := AnalyzeByRegime(trades, 100000, aiHistory, detHistory)
	if len(result.ByAIRegime) != 2 {
		t.Errorf("expected 2 AI regime groups, got %d", len(result.ByAIRegime))
	}
	if r, ok := result.ByAIRegime[strategy.RegimeBull]; !ok || r.TotalPnL <= 0 {
		t.Error("expected positive PnL in bull regime")
	}
	if r, ok := result.ByAIRegime[strategy.RegimeBear]; !ok || r.TotalPnL >= 0 {
		t.Error("expected negative PnL in bear regime")
	}
}

func TestAnalyzeByRegime_DetectedRegime(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	exitTime := base.Add(24 * time.Hour)
	trades := []storage.TradeRecord{
		{StrategyID: "s1", Symbol: "A", EntryPrice: 100, ExitPrice: 120, PnL: 200, EntryTime: base, ExitTime: &exitTime, Status: "closed"},
	}
	detHistory := map[string]regime.DetectedRegime{
		"2024-01-01": regime.RegimeTrending,
	}
	result := AnalyzeByRegime(trades, 100000, nil, detHistory)
	if len(result.ByDetectedRegime) != 1 {
		t.Errorf("expected 1 detected regime group, got %d", len(result.ByDetectedRegime))
	}
}

func TestAnalyzeByRegime_EmptyTrades(t *testing.T) {
	result := AnalyzeByRegime(nil, 100000, nil, nil)
	if len(result.ByAIRegime) != 0 || len(result.ByDetectedRegime) != 0 {
		t.Error("expected empty results for empty trades")
	}
}

func TestAnalyzeByRegime_NoMatchingRegime(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	exitTime := base.Add(24 * time.Hour)
	trades := []storage.TradeRecord{
		{StrategyID: "s1", Symbol: "A", EntryPrice: 100, ExitPrice: 110, PnL: 100, EntryTime: base, ExitTime: &exitTime, Status: "closed"},
	}
	aiHistory := map[string]strategy.MarketRegime{
		"2024-06-01": strategy.RegimeBull,
	}
	result := AnalyzeByRegime(trades, 100000, aiHistory, nil)
	if len(result.ByAIRegime) != 0 {
		t.Errorf("expected 0 AI regime groups for non-matching dates, got %d", len(result.ByAIRegime))
	}
}
