// Package strategy - pullback.go implements an EMA pullback strategy.
//
// This strategy buys dips in a strong uptrend. When a stock is trending up
// (price above 50-EMA), it waits for a pullback to the 20-EMA as a
// buying opportunity. This is one of the most reliable swing setups because
// it enters at a discount within an established trend.
//
// Entry rules:
//   - Market regime is BULL
//   - Trend strength >= threshold (must be in an uptrend)
//   - Price is above 50-EMA (uptrend confirmed)
//   - Price has pulled back to near 20-EMA (within tolerance)
//   - RSI(14) is in the 40-60 zone (not oversold/overbought — healthy pullback)
//   - Risk score <= threshold
//   - Liquidity score >= threshold
//   - Sufficient candle history (60+)
//
// Exit rules:
//   - Price breaks below 50-EMA (uptrend broken)
//   - Trend strength drops below exit threshold
//   - Market regime changes to BEAR
package strategy

import (
	"fmt"
	"math"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// PullbackStrategy implements an EMA pullback strategy.
type PullbackStrategy struct {
	// Entry thresholds.
	MinTrendStrength float64 // default 0.5
	MinLiquidity     float64 // default 0.4
	MaxRiskScore     float64 // default 0.5
	FastEMAPeriod    int     // default 20
	SlowEMAPeriod    int     // default 50
	PullbackPct      float64 // max % above fast EMA to count as pullback (default 1.0)
	RSILow           float64 // min RSI for healthy pullback (default 40)
	RSIHigh          float64 // max RSI for healthy pullback (default 60)

	// Exit thresholds.
	ExitTrendStrength float64 // default 0.3

	// ATR multiplier for stop-loss.
	ATRStopMultiplier float64 // default 2.0

	// Risk-reward ratio for target.
	RiskRewardRatio float64 // default 2.5

	// Risk config for position sizing.
	RiskConfig config.RiskConfig
}

// NewPullbackStrategy creates a pullback strategy with sensible defaults.
func NewPullbackStrategy(riskCfg config.RiskConfig) *PullbackStrategy {
	return &PullbackStrategy{
		MinTrendStrength:  0.5,
		MinLiquidity:      0.4,
		MaxRiskScore:      0.5,
		FastEMAPeriod:     20,
		SlowEMAPeriod:     50,
		PullbackPct:       1.0,
		RSILow:            40,
		RSIHigh:           60,
		ExitTrendStrength: 0.3,
		ATRStopMultiplier: 2.0,
		RiskRewardRatio:   2.5,
		RiskConfig:        riskCfg,
	}
}

func (s *PullbackStrategy) ID() string   { return "pullback_v1" }
func (s *PullbackStrategy) Name() string { return "EMA Pullback" }

// Evaluate applies the pullback rules to produce a TradeIntent.
func (s *PullbackStrategy) Evaluate(input StrategyInput) TradeIntent {
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

func (s *PullbackStrategy) evaluateEntry(input StrategyInput, intent TradeIntent) TradeIntent {
	// Rule 1: Only trade in BULL regime.
	if input.Regime.Regime != RegimeBull {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("market regime is %s, pullback requires BULL", input.Regime.Regime)
		return intent
	}

	// Rule 2: Regime confidence.
	if input.Regime.Confidence < 0.6 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("regime confidence %.2f < 0.60", input.Regime.Confidence)
		return intent
	}

	// Rule 3: Trend strength check (stock must be in an uptrend).
	if input.Score.TrendStrengthScore < s.MinTrendStrength {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("trend strength %.2f < %.2f (need uptrend for pullback)",
			input.Score.TrendStrengthScore, s.MinTrendStrength)
		return intent
	}

	// Rule 4: Liquidity check.
	if input.Score.LiquidityScore < s.MinLiquidity {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("liquidity %.2f < %.2f", input.Score.LiquidityScore, s.MinLiquidity)
		return intent
	}

	// Rule 5: Risk score check.
	if input.Score.RiskScore > s.MaxRiskScore {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("risk score %.2f > %.2f", input.Score.RiskScore, s.MaxRiskScore)
		return intent
	}

	// Rule 6: Sufficient candle history for 50-EMA.
	if len(input.Candles) < 60 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("insufficient candle history: %d < 60", len(input.Candles))
		return intent
	}

	lastCandle := input.Candles[len(input.Candles)-1]
	fastEMA := CalculateEMA(input.Candles, s.FastEMAPeriod)
	slowEMA := CalculateEMA(input.Candles, s.SlowEMAPeriod)

	// Rule 7: Price must be above 50-EMA (confirming uptrend).
	if lastCandle.Close <= slowEMA {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("price %.2f <= 50-EMA %.2f (not in uptrend)",
			lastCandle.Close, slowEMA)
		return intent
	}

	// Rule 8: Price must have pulled back to near the 20-EMA.
	// We check that price is within PullbackPct of the fast EMA.
	distPct := math.Abs(lastCandle.Close-fastEMA) / fastEMA * 100
	if distPct > s.PullbackPct {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("price %.2f is %.1f%% from 20-EMA %.2f (need within %.1f%%)",
			lastCandle.Close, distPct, fastEMA, s.PullbackPct)
		return intent
	}

	// Rule 9: RSI must be in the healthy pullback zone.
	rsi := CalculateRSI(input.Candles, 14)
	if rsi < s.RSILow || rsi > s.RSIHigh {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("RSI %.1f outside pullback zone [%.0f-%.0f]", rsi, s.RSILow, s.RSIHigh)
		return intent
	}

	// All entry conditions met.
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
		"pullback: 20-EMA=%.2f 50-EMA=%.2f price=%.2f dist=%.1f%% RSI=%.1f | ATR=%.2f SL=%.2f TGT=%.2f",
		fastEMA, slowEMA, entryPrice, distPct, rsi, atr, stopLoss, target,
	)
	return intent
}

func (s *PullbackStrategy) evaluateExit(input StrategyInput, intent TradeIntent) TradeIntent {
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

	// Exit Rule 2: Price broke below 50-EMA (uptrend broken).
	if len(input.Candles) >= s.SlowEMAPeriod {
		slowEMA := CalculateEMA(input.Candles, s.SlowEMAPeriod)
		lastPrice := input.Candles[len(input.Candles)-1].Close
		if lastPrice < slowEMA {
			intent.Action = ActionExit
			intent.Price = lastPrice
			intent.Quantity = input.CurrentPosition.Quantity
			intent.Reason = fmt.Sprintf("price %.2f broke below 50-EMA %.2f — uptrend broken",
				lastPrice, slowEMA)
			return intent
		}
	}

	// Exit Rule 3: Trend strength collapsed.
	if input.Score.TrendStrengthScore < s.ExitTrendStrength {
		intent.Action = ActionExit
		intent.Quantity = input.CurrentPosition.Quantity
		intent.Reason = fmt.Sprintf("trend strength dropped to %.2f < %.2f",
			input.Score.TrendStrengthScore, s.ExitTrendStrength)
		if len(input.Candles) > 0 {
			intent.Price = input.Candles[len(input.Candles)-1].Close
		}
		return intent
	}

	// Otherwise, hold.
	intent.Action = ActionHold
	intent.Reason = fmt.Sprintf("holding: trend=%.2f, pullback position intact",
		input.Score.TrendStrengthScore)
	return intent
}
