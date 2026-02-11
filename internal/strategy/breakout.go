// Package strategy - breakout.go implements a breakout swing strategy.
//
// This strategy buys when price breaks above a resistance level (20-day high)
// with volume confirmation. Breakouts tend to lead to strong momentum moves.
//
// Entry rules:
//   - Market regime must be BULL
//   - Breakout quality score >= threshold (0.7 — high bar)
//   - Trend strength >= threshold (0.5 — moderate uptrend)
//   - Current price > 20-day high (breakout condition)
//   - Current volume > 1.5× average volume (volume confirmation)
//   - Risk score <= threshold (0.4 — strict)
//   - Liquidity score >= threshold (0.5)
//   - Sufficient candle history (30+)
//
// Exit rules:
//   - Price falls back below the breakout level (failed breakout)
//   - Volume dries up significantly
//   - Market regime changes to BEAR
//   - Trend strength drops (momentum fading)
package strategy

import (
	"fmt"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// BreakoutStrategy implements a breakout swing strategy.
type BreakoutStrategy struct {
	// Entry thresholds.
	MinBreakoutQuality float64 // default 0.7 (only strong breakouts)
	MinTrendStrength   float64 // default 0.5
	MinLiquidity       float64 // default 0.5
	MaxRiskScore       float64 // default 0.4 (strict)
	VolumeMultiplier   float64 // default 1.5 (volume must be 1.5× avg)
	HighLookback       int     // default 20 (20-day high for breakout)

	// Exit thresholds.
	ExitTrendStrength float64 // default 0.3

	// ATR multiplier for stop-loss.
	ATRStopMultiplier float64 // default 1.5

	// Risk-reward ratio for target.
	RiskRewardRatio float64 // default 3.0 (breakouts can run far)

	// Risk config for position sizing.
	RiskConfig config.RiskConfig
}

// NewBreakoutStrategy creates a breakout strategy with sensible defaults.
func NewBreakoutStrategy(riskCfg config.RiskConfig) *BreakoutStrategy {
	return &BreakoutStrategy{
		MinBreakoutQuality: 0.7,
		MinTrendStrength:   0.5,
		MinLiquidity:       0.5,
		MaxRiskScore:       0.4,
		VolumeMultiplier:   1.5,
		HighLookback:       20,
		ExitTrendStrength:  0.3,
		ATRStopMultiplier:  1.5,
		RiskRewardRatio:    3.0,
		RiskConfig:         riskCfg,
	}
}

func (s *BreakoutStrategy) ID() string   { return "breakout_v1" }
func (s *BreakoutStrategy) Name() string { return "Breakout Swing" }

// Evaluate applies the breakout rules to produce a TradeIntent.
func (s *BreakoutStrategy) Evaluate(input StrategyInput) TradeIntent {
	intent := TradeIntent{
		StrategyID: s.ID(),
		SignalID:   fmt.Sprintf("%s-%s-%s", s.ID(), input.Score.Symbol, input.Date.Format("2006-01-02")),
		Symbol:     input.Score.Symbol,
		Scores:     input.Score,
	}

	if input.CurrentPosition != nil {
		return s.evaluateExit(input, intent)
	}

	return s.evaluateEntry(input, intent)
}

func (s *BreakoutStrategy) evaluateEntry(input StrategyInput, intent TradeIntent) TradeIntent {
	// Rule 1: Only trade in BULL regime.
	if input.Regime.Regime != RegimeBull {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("market regime is %s, breakout requires BULL", input.Regime.Regime)
		return intent
	}

	// Rule 2: Regime confidence must be sufficient.
	if input.Regime.Confidence < 0.6 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("regime confidence %.2f < 0.60", input.Regime.Confidence)
		return intent
	}

	// Rule 3: Breakout quality must be high.
	if input.Score.BreakoutQualityScore < s.MinBreakoutQuality {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("breakout quality %.2f < %.2f", input.Score.BreakoutQualityScore, s.MinBreakoutQuality)
		return intent
	}

	// Rule 4: Trend strength check.
	if input.Score.TrendStrengthScore < s.MinTrendStrength {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("trend strength %.2f < %.2f", input.Score.TrendStrengthScore, s.MinTrendStrength)
		return intent
	}

	// Rule 5: Liquidity check.
	if input.Score.LiquidityScore < s.MinLiquidity {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("liquidity %.2f < %.2f", input.Score.LiquidityScore, s.MinLiquidity)
		return intent
	}

	// Rule 6: Risk score check.
	if input.Score.RiskScore > s.MaxRiskScore {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("risk score %.2f > %.2f", input.Score.RiskScore, s.MaxRiskScore)
		return intent
	}

	// Rule 7: Must have sufficient candle history.
	if len(input.Candles) < 30 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("insufficient candle history: %d < 30", len(input.Candles))
		return intent
	}

	lastCandle := input.Candles[len(input.Candles)-1]

	// Rule 8: Price must be above the N-day high (breakout).
	// Look at the high EXCLUDING the last candle (the breakout candle).
	priorCandles := input.Candles[:len(input.Candles)-1]
	resistance := HighestHigh(priorCandles, s.HighLookback)
	if lastCandle.Close <= resistance {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("price %.2f <= %d-day high %.2f (no breakout)", lastCandle.Close, s.HighLookback, resistance)
		return intent
	}

	// Rule 9: Volume must confirm the breakout.
	avgVol := AverageVolume(priorCandles, s.HighLookback)
	if avgVol > 0 && float64(lastCandle.Volume) < avgVol*s.VolumeMultiplier {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("volume %d < %.0f×%.0f (no volume confirmation)",
			lastCandle.Volume, s.VolumeMultiplier, avgVol)
		return intent
	}

	// All entry conditions met.
	atr := CalculateATR(input.Candles, 14)
	entryPrice := lastCandle.Close
	// Stop loss just below the breakout level.
	stopLoss := resistance - (atr * s.ATRStopMultiplier)
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
		"breakout: price=%.2f > resist=%.2f vol=%d/avg=%.0f | ATR=%.2f SL=%.2f TGT=%.2f",
		entryPrice, resistance, lastCandle.Volume, avgVol, atr, stopLoss, target,
	)
	return intent
}

func (s *BreakoutStrategy) evaluateExit(input StrategyInput, intent TradeIntent) TradeIntent {
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

	// Exit Rule 2: Trend strength collapsed (momentum fading).
	if input.Score.TrendStrengthScore < s.ExitTrendStrength {
		intent.Action = ActionExit
		intent.Quantity = input.CurrentPosition.Quantity
		intent.Reason = fmt.Sprintf("trend strength dropped to %.2f < %.2f — breakout momentum fading",
			input.Score.TrendStrengthScore, s.ExitTrendStrength)
		if len(input.Candles) > 0 {
			intent.Price = input.Candles[len(input.Candles)-1].Close
		}
		return intent
	}

	// Exit Rule 3: Price fell back below entry (failed breakout).
	if len(input.Candles) > 0 && input.CurrentPosition.EntryPrice > 0 {
		lastPrice := input.Candles[len(input.Candles)-1].Close
		if lastPrice < input.CurrentPosition.EntryPrice {
			intent.Action = ActionExit
			intent.Price = lastPrice
			intent.Quantity = input.CurrentPosition.Quantity
			intent.Reason = fmt.Sprintf("price %.2f fell below entry %.2f — failed breakout",
				lastPrice, input.CurrentPosition.EntryPrice)
			return intent
		}
	}

	// Otherwise, hold.
	intent.Action = ActionHold
	intent.Reason = fmt.Sprintf("holding: breakout intact, trend=%.2f", input.Score.TrendStrengthScore)
	return intent
}
