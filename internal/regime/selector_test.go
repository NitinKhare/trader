package regime

import (
	"testing"

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// mockStrategy implements strategy.Strategy but NOT ClassifiedStrategy.
type mockStrategy struct {
	id string
}

func (m *mockStrategy) ID() string                              { return m.id }
func (m *mockStrategy) Name() string                            { return m.id }
func (m *mockStrategy) Evaluate(strategy.StrategyInput) strategy.TradeIntent {
	return strategy.TradeIntent{}
}

// classifiedMock implements both Strategy and ClassifiedStrategy.
type classifiedMock struct {
	id         string
	stratType  strategy.StrategyType
}

func (c *classifiedMock) ID() string                              { return c.id }
func (c *classifiedMock) Name() string                            { return c.id }
func (c *classifiedMock) Evaluate(strategy.StrategyInput) strategy.TradeIntent {
	return strategy.TradeIntent{}
}
func (c *classifiedMock) GetStrategyType() strategy.StrategyType { return c.stratType }

// noAI returns a zero-value MarketRegimeData (no AI input).
func noAI() strategy.MarketRegimeData {
	return strategy.MarketRegimeData{}
}

// bullAI returns a confident BULL AI regime.
func bullAI(confidence float64) strategy.MarketRegimeData {
	return strategy.MarketRegimeData{
		Regime:     strategy.RegimeBull,
		Confidence: confidence,
	}
}

// bearAI returns a confident BEAR AI regime.
func bearAI(confidence float64) strategy.MarketRegimeData {
	return strategy.MarketRegimeData{
		Regime:     strategy.RegimeBear,
		Confidence: confidence,
	}
}

func TestStrategySelector_TrendingRegime(t *testing.T) {
	ss := NewStrategySelector()
	strategies := []strategy.Strategy{
		&classifiedMock{id: "trend1", stratType: strategy.StrategyTypeTrend},
		&classifiedMock{id: "mr1", stratType: strategy.StrategyTypeMeanReversion},
		&classifiedMock{id: "breakout1", stratType: strategy.StrategyTypeBreakout},
		&classifiedMock{id: "momentum1", stratType: strategy.StrategyTypeMomentum},
		&classifiedMock{id: "vol1", stratType: strategy.StrategyTypeVolatility},
	}

	state := RegimeState{Primary: RegimeTrending}
	selected := ss.SelectStrategies(state, noAI(), strategies)

	// Trending allows TREND, MOMENTUM, BREAKOUT.
	if len(selected) != 3 {
		t.Errorf("expected 3 strategies for trending, got %d", len(selected))
	}

	ids := make(map[string]bool)
	for _, s := range selected {
		ids[s.ID()] = true
	}
	for _, expected := range []string{"trend1", "breakout1", "momentum1"} {
		if !ids[expected] {
			t.Errorf("expected %s in selected strategies", expected)
		}
	}
}

func TestStrategySelector_RangingRegime(t *testing.T) {
	ss := NewStrategySelector()
	strategies := []strategy.Strategy{
		&classifiedMock{id: "trend1", stratType: strategy.StrategyTypeTrend},
		&classifiedMock{id: "mr1", stratType: strategy.StrategyTypeMeanReversion},
		&classifiedMock{id: "vol1", stratType: strategy.StrategyTypeVolatility},
	}

	state := RegimeState{Primary: RegimeRanging}
	selected := ss.SelectStrategies(state, noAI(), strategies)

	// Ranging allows MEAN_REVERSION, VOLATILITY.
	if len(selected) != 2 {
		t.Errorf("expected 2 strategies for ranging, got %d", len(selected))
	}
}

func TestStrategySelector_UnclassifiedAlwaysIncluded(t *testing.T) {
	ss := NewStrategySelector()
	strategies := []strategy.Strategy{
		&mockStrategy{id: "unclassified"},
		&classifiedMock{id: "trend1", stratType: strategy.StrategyTypeTrend},
	}

	// In ranging regime, trend strategy is excluded but unclassified is included.
	state := RegimeState{Primary: RegimeRanging}
	selected := ss.SelectStrategies(state, noAI(), strategies)

	if len(selected) != 1 {
		t.Errorf("expected 1 strategy (unclassified only), got %d", len(selected))
	}
	if selected[0].ID() != "unclassified" {
		t.Errorf("expected unclassified strategy, got %s", selected[0].ID())
	}
}

func TestStrategySelector_HighVolatility(t *testing.T) {
	ss := NewStrategySelector()
	strategies := []strategy.Strategy{
		&classifiedMock{id: "mr1", stratType: strategy.StrategyTypeMeanReversion},
		&classifiedMock{id: "vol1", stratType: strategy.StrategyTypeVolatility},
		&classifiedMock{id: "trend1", stratType: strategy.StrategyTypeTrend},
	}

	state := RegimeState{Primary: RegimeHighVolatility}
	// Without AI regime, high vol allows only MEAN_REVERSION, VOLATILITY.
	selected := ss.SelectStrategies(state, noAI(), strategies)

	if len(selected) != 2 {
		t.Errorf("expected 2 strategies for high volatility (no AI), got %d", len(selected))
	}
}

func TestStrategySelector_HighVolatilityWithBullAI(t *testing.T) {
	ss := NewStrategySelector()
	strategies := []strategy.Strategy{
		&classifiedMock{id: "mr1", stratType: strategy.StrategyTypeMeanReversion},
		&classifiedMock{id: "vol1", stratType: strategy.StrategyTypeVolatility},
		&classifiedMock{id: "trend1", stratType: strategy.StrategyTypeTrend},
		&classifiedMock{id: "momentum1", stratType: strategy.StrategyTypeMomentum},
		&classifiedMock{id: "breakout1", stratType: strategy.StrategyTypeBreakout},
	}

	state := RegimeState{Primary: RegimeHighVolatility}
	// With confident BULL AI regime, high vol should also allow TREND, MOMENTUM, BREAKOUT.
	selected := ss.SelectStrategies(state, bullAI(0.90), strategies)

	// All 5 strategy types should be allowed.
	if len(selected) != 5 {
		t.Errorf("expected 5 strategies for HIGH_VOL + BULL AI, got %d", len(selected))
	}

	ids := make(map[string]bool)
	for _, s := range selected {
		ids[s.ID()] = true
	}
	for _, expected := range []string{"mr1", "vol1", "trend1", "momentum1", "breakout1"} {
		if !ids[expected] {
			t.Errorf("expected %s in selected strategies for HIGH_VOL + BULL AI", expected)
		}
	}
}

func TestStrategySelector_HighVolatilityWithBearAI(t *testing.T) {
	ss := NewStrategySelector()
	strategies := []strategy.Strategy{
		&classifiedMock{id: "mr1", stratType: strategy.StrategyTypeMeanReversion},
		&classifiedMock{id: "vol1", stratType: strategy.StrategyTypeVolatility},
		&classifiedMock{id: "trend1", stratType: strategy.StrategyTypeTrend},
	}

	state := RegimeState{Primary: RegimeHighVolatility}
	// With BEAR AI regime, should NOT expand — stay with MEAN_REVERSION, VOLATILITY only.
	selected := ss.SelectStrategies(state, bearAI(0.90), strategies)

	if len(selected) != 2 {
		t.Errorf("expected 2 strategies for HIGH_VOL + BEAR AI, got %d", len(selected))
	}

	ids := make(map[string]bool)
	for _, s := range selected {
		ids[s.ID()] = true
	}
	if ids["trend1"] {
		t.Error("trend1 should NOT be selected for HIGH_VOL + BEAR AI")
	}
}

func TestStrategySelector_HighVolatilityWithLowConfidenceBullAI(t *testing.T) {
	ss := NewStrategySelector()
	strategies := []strategy.Strategy{
		&classifiedMock{id: "mr1", stratType: strategy.StrategyTypeMeanReversion},
		&classifiedMock{id: "vol1", stratType: strategy.StrategyTypeVolatility},
		&classifiedMock{id: "trend1", stratType: strategy.StrategyTypeTrend},
	}

	state := RegimeState{Primary: RegimeHighVolatility}
	// With low-confidence BULL AI (< 0.70), should NOT expand.
	selected := ss.SelectStrategies(state, bullAI(0.50), strategies)

	if len(selected) != 2 {
		t.Errorf("expected 2 strategies for HIGH_VOL + low-confidence BULL AI, got %d", len(selected))
	}

	ids := make(map[string]bool)
	for _, s := range selected {
		ids[s.ID()] = true
	}
	if ids["trend1"] {
		t.Error("trend1 should NOT be selected for HIGH_VOL + low-confidence BULL AI")
	}
}

func TestStrategySelector_LowVolatility(t *testing.T) {
	ss := NewStrategySelector()
	strategies := []strategy.Strategy{
		&classifiedMock{id: "trend1", stratType: strategy.StrategyTypeTrend},
		&classifiedMock{id: "breakout1", stratType: strategy.StrategyTypeBreakout},
		&classifiedMock{id: "mr1", stratType: strategy.StrategyTypeMeanReversion},
	}

	state := RegimeState{Primary: RegimeLowVolatility}
	selected := ss.SelectStrategies(state, noAI(), strategies)

	// Low vol allows TREND, BREAKOUT.
	if len(selected) != 2 {
		t.Errorf("expected 2 strategies for low volatility, got %d", len(selected))
	}
}

func TestStrategySelector_EmptyStrategies(t *testing.T) {
	ss := NewStrategySelector()
	selected := ss.SelectStrategies(RegimeState{Primary: RegimeTrending}, noAI(), nil)
	if len(selected) != 0 {
		t.Errorf("expected 0 strategies for nil input, got %d", len(selected))
	}
}

func TestStrategySelector_SetAllowedTypes(t *testing.T) {
	ss := NewStrategySelector()
	// Override trending to only allow VOLATILITY.
	ss.SetAllowedTypes(RegimeTrending, []strategy.StrategyType{strategy.StrategyTypeVolatility})

	strategies := []strategy.Strategy{
		&classifiedMock{id: "trend1", stratType: strategy.StrategyTypeTrend},
		&classifiedMock{id: "vol1", stratType: strategy.StrategyTypeVolatility},
	}

	state := RegimeState{Primary: RegimeTrending}
	selected := ss.SelectStrategies(state, noAI(), strategies)

	if len(selected) != 1 {
		t.Errorf("expected 1 strategy after override, got %d", len(selected))
	}
	if selected[0].ID() != "vol1" {
		t.Errorf("expected vol1, got %s", selected[0].ID())
	}
}
