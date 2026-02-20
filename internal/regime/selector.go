// Package regime - selector.go maps detected regimes to allowed strategy types.
//
// The strategy selector filters the available strategies based on the current
// market regime. Strategies that implement ClassifiedStrategy are checked against
// the regime mapping. Strategies that don't implement it are always included
// (backward compatible).
package regime

import (
	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// StrategySelector maps detected regimes to allowed strategy types.
type StrategySelector struct {
	allowedTypes map[DetectedRegime][]strategy.StrategyType
}

// NewStrategySelector creates a selector with default regime-to-strategy mappings.
//
// Default mapping:
//
//	TRENDING        → [TREND, MOMENTUM, BREAKOUT]
//	RANGING         → [MEAN_REVERSION, VOLATILITY]
//	HIGH_VOLATILITY → [MEAN_REVERSION, VOLATILITY]
//	LOW_VOLATILITY  → [TREND, BREAKOUT]
func NewStrategySelector() *StrategySelector {
	return &StrategySelector{
		allowedTypes: map[DetectedRegime][]strategy.StrategyType{
			RegimeTrending: {
				strategy.StrategyTypeTrend,
				strategy.StrategyTypeMomentum,
				strategy.StrategyTypeBreakout,
			},
			RegimeRanging: {
				strategy.StrategyTypeMeanReversion,
				strategy.StrategyTypeVolatility,
			},
			RegimeHighVolatility: {
				strategy.StrategyTypeMeanReversion,
				strategy.StrategyTypeVolatility,
			},
			RegimeLowVolatility: {
				strategy.StrategyTypeTrend,
				strategy.StrategyTypeBreakout,
			},
		},
	}
}

// SelectStrategies filters strategies to only those appropriate for the detected regime.
// Strategies that don't implement ClassifiedStrategy are always included (backward compatible).
//
// The AI regime (BULL/SIDEWAYS/BEAR from the Python scoring pipeline) is used to expand
// the allowed strategy types in specific scenarios. For example, when the AI confidently
// identifies a BULL market but the technical regime shows HIGH_VOLATILITY, trend and
// momentum strategies are also allowed — since a volatile bull market still rewards
// directional bets with appropriate position sizing.
func (ss *StrategySelector) SelectStrategies(
	regimeState RegimeState,
	aiRegime strategy.MarketRegimeData,
	strategies []strategy.Strategy,
) []strategy.Strategy {
	allowed, ok := ss.allowedTypes[regimeState.Primary]
	if !ok {
		// Unknown regime — include all strategies.
		return strategies
	}

	// Build a set of allowed types for fast lookup.
	allowedSet := make(map[strategy.StrategyType]bool, len(allowed))
	for _, t := range allowed {
		allowedSet[t] = true
	}

	// When AI confidently says BULL but technical regime is HIGH_VOLATILITY,
	// also allow trend/momentum/breakout strategies. A volatile bull market
	// still rewards directional strategies (with wider ATR-based stops).
	if regimeState.Primary == RegimeHighVolatility &&
		aiRegime.Regime == strategy.RegimeBull &&
		aiRegime.Confidence >= 0.70 {
		for _, t := range []strategy.StrategyType{
			strategy.StrategyTypeTrend,
			strategy.StrategyTypeMomentum,
			strategy.StrategyTypeBreakout,
		} {
			allowedSet[t] = true
		}
	}

	var selected []strategy.Strategy
	for _, s := range strategies {
		cs, isClassified := s.(strategy.ClassifiedStrategy)
		if !isClassified {
			// Unclassified strategies are always included (backward compatible).
			selected = append(selected, s)
			continue
		}
		if allowedSet[cs.GetStrategyType()] {
			selected = append(selected, s)
		}
	}
	return selected
}

// SetAllowedTypes overrides the default mapping for a specific regime.
// Useful for configuration-driven regime mappings.
func (ss *StrategySelector) SetAllowedTypes(regime DetectedRegime, types []strategy.StrategyType) {
	ss.allowedTypes[regime] = types
}
