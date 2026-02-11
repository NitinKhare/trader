// Package strategy - trend_follow.go implements a swing trading trend-following strategy.
//
// This is the first concrete strategy for the system.
// It buys strong-trending stocks in bull markets and exits on weakness.
//
// Entry rules:
//   - Market regime must be BULL
//   - Trend strength score >= threshold
//   - Breakout quality score >= threshold
//   - Liquidity score >= threshold
//   - Risk score <= threshold (lower is safer)
//
// Exit rules:
//   - Stop loss hit (handled by order management, not here)
//   - Target hit (handled by order management, not here)
//   - Trend strength drops below exit threshold
//   - Market regime changes to BEAR
package strategy

import (
	"fmt"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// TrendFollowStrategy implements a simple trend-following swing strategy.
type TrendFollowStrategy struct {
	// Entry thresholds — all must be met.
	MinTrendStrength  float64
	MinBreakoutQuality float64
	MinLiquidity      float64
	MaxRiskScore      float64

	// Exit thresholds.
	ExitTrendStrength float64

	// ATR multiplier for stop-loss calculation.
	ATRStopMultiplier float64

	// Risk-reward ratio for target calculation.
	RiskRewardRatio float64

	// Risk config for position sizing.
	RiskConfig config.RiskConfig
}

// NewTrendFollowStrategy creates a trend-following strategy with sensible defaults.
func NewTrendFollowStrategy(riskCfg config.RiskConfig) *TrendFollowStrategy {
	return &TrendFollowStrategy{
		MinTrendStrength:   0.6,
		MinBreakoutQuality: 0.5,
		MinLiquidity:       0.4,
		MaxRiskScore:       0.5,
		ExitTrendStrength:  0.3,
		ATRStopMultiplier:  2.0,
		RiskRewardRatio:    2.0,
		RiskConfig:         riskCfg,
	}
}

func (s *TrendFollowStrategy) ID() string   { return "trend_follow_v1" }
func (s *TrendFollowStrategy) Name() string { return "Trend Following Swing" }

// Evaluate applies the trend-following rules to produce a TradeIntent.
func (s *TrendFollowStrategy) Evaluate(input StrategyInput) TradeIntent {
	intent := TradeIntent{
		StrategyID: s.ID(),
		SignalID:   fmt.Sprintf("%s-%s-%s", s.ID(), input.Score.Symbol, input.Date.Format("2006-01-02")),
		Symbol:     input.Score.Symbol,
		Scores:     input.Score,
	}

	// If we have a position, evaluate exit conditions.
	if input.CurrentPosition != nil {
		return s.evaluateExit(input, intent)
	}

	// Otherwise, evaluate entry conditions.
	return s.evaluateEntry(input, intent)
}

func (s *TrendFollowStrategy) evaluateEntry(input StrategyInput, intent TradeIntent) TradeIntent {
	// Rule 1: Only trade in BULL regime.
	if input.Regime.Regime != RegimeBull {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("market regime is %s, require BULL", input.Regime.Regime)
		return intent
	}

	// Rule 2: Regime confidence must be sufficient.
	if input.Regime.Confidence < 0.6 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("regime confidence %.2f < 0.60", input.Regime.Confidence)
		return intent
	}

	// Rule 3: Trend strength check.
	if input.Score.TrendStrengthScore < s.MinTrendStrength {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("trend strength %.2f < %.2f", input.Score.TrendStrengthScore, s.MinTrendStrength)
		return intent
	}

	// Rule 4: Breakout quality check.
	if input.Score.BreakoutQualityScore < s.MinBreakoutQuality {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("breakout quality %.2f < %.2f", input.Score.BreakoutQualityScore, s.MinBreakoutQuality)
		return intent
	}

	// Rule 5: Liquidity check.
	if input.Score.LiquidityScore < s.MinLiquidity {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("liquidity %.2f < %.2f", input.Score.LiquidityScore, s.MinLiquidity)
		return intent
	}

	// Rule 6: Risk score check (lower is safer).
	if input.Score.RiskScore > s.MaxRiskScore {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("risk score %.2f > %.2f", input.Score.RiskScore, s.MaxRiskScore)
		return intent
	}

	// Rule 7: Must have sufficient candle history.
	if len(input.Candles) < 20 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("insufficient candle history: %d < 20", len(input.Candles))
		return intent
	}

	// All entry conditions met — calculate stop loss, target, and quantity.
	lastCandle := input.Candles[len(input.Candles)-1]
	atr := CalculateATR(input.Candles, 14)

	entryPrice := lastCandle.Close
	stopLoss := entryPrice - (atr * s.ATRStopMultiplier)
	riskPerShare := entryPrice - stopLoss
	target := entryPrice + (riskPerShare * s.RiskRewardRatio)

	// Position sizing: risk-based.
	maxRiskAmount := input.AvailableCapital * (s.RiskConfig.MaxRiskPerTradePct / 100.0)
	quantity := int(maxRiskAmount / riskPerShare)
	if quantity <= 0 {
		intent.Action = ActionSkip
		intent.Reason = "calculated quantity is zero (risk per share too large)"
		return intent
	}

	// Ensure we don't exceed available capital.
	totalCost := entryPrice * float64(quantity)
	if totalCost > input.AvailableCapital {
		quantity = int(input.AvailableCapital / entryPrice)
	}
	if quantity <= 0 {
		intent.Action = ActionSkip
		intent.Reason = "insufficient capital for minimum position"
		return intent
	}

	intent.Action = ActionBuy
	intent.Price = entryPrice
	intent.StopLoss = stopLoss
	intent.Target = target
	intent.Quantity = quantity
	intent.Reason = fmt.Sprintf(
		"trend=%.2f breakout=%.2f liq=%.2f risk=%.2f | ATR=%.2f SL=%.2f TGT=%.2f",
		input.Score.TrendStrengthScore,
		input.Score.BreakoutQualityScore,
		input.Score.LiquidityScore,
		input.Score.RiskScore,
		atr, stopLoss, target,
	)
	return intent
}

func (s *TrendFollowStrategy) evaluateExit(input StrategyInput, intent TradeIntent) TradeIntent {
	// Exit Rule 1: Market turned BEAR.
	if input.Regime.Regime == RegimeBear {
		intent.Action = ActionExit
		intent.Quantity = input.CurrentPosition.Quantity
		intent.Reason = "market regime turned BEAR"
		if len(input.Candles) > 0 {
			intent.Price = input.Candles[len(input.Candles)-1].Close
		}
		return intent
	}

	// Exit Rule 2: Trend strength collapsed.
	if input.Score.TrendStrengthScore < s.ExitTrendStrength {
		intent.Action = ActionExit
		intent.Quantity = input.CurrentPosition.Quantity
		intent.Reason = fmt.Sprintf("trend strength dropped to %.2f < %.2f", input.Score.TrendStrengthScore, s.ExitTrendStrength)
		if len(input.Candles) > 0 {
			intent.Price = input.Candles[len(input.Candles)-1].Close
		}
		return intent
	}

	// Otherwise, hold.
	intent.Action = ActionHold
	intent.Reason = fmt.Sprintf("holding: trend=%.2f regime=%s", input.Score.TrendStrengthScore, input.Regime.Regime)
	return intent
}

// calculateATR delegates to the shared CalculateATR indicator.
// Kept as a method for backward compatibility within the strategy.
func (s *TrendFollowStrategy) calculateATR(candles []Candle, period int) float64 {
	return CalculateATR(candles, period)
}
