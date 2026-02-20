// Package strategy - bollinger.go implements a Bollinger Band Squeeze strategy.
//
// The Bollinger Band Squeeze identifies periods of low volatility (tight bands)
// that precede explosive moves. When the bands contract to an extreme and price
// then breaks above the upper band, it signals a high-probability breakout.
//
// This is a volatility-contraction/expansion strategy — the squeeze "loads the spring"
// and the breakout "releases" it.
//
// Entry rules:
//   - Market regime is BULL or SIDEWAYS
//   - Bollinger Bandwidth is below the squeeze threshold (6% — genuinely tight)
//   - Price breaks above the upper band (expansion begins)
//   - Volume confirms the breakout (2.0× average)
//   - RSI > 50 (momentum is building, not fading)
//   - ADX > 20 (a trend is actually forming)
//   - Trend strength >= threshold (0.4)
//   - Risk score <= threshold
//   - Sufficient candle history (30+)
//
// Exit rules:
//   - Price falls below entry minus 2× ATR (ATR trailing stop)
//   - Price falls below lower Bollinger Band (breakdown)
//   - Market regime changes to BEAR
//   - Trend strength collapses
package strategy

import (
	"fmt"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// BollingerSqueezeStrategy implements a Bollinger Band squeeze breakout strategy.
type BollingerSqueezeStrategy struct {
	// Bollinger Band parameters.
	BBPeriod     int     // default 20
	BBMultiplier float64 // default 2.0

	// Entry thresholds.
	SqueezeBandwidth float64 // max bandwidth for squeeze (default 0.06 = 6%)
	VolumeMultiplier float64 // default 2.0 (require real breakout volume)
	MinTrendStrength float64 // default 0.4
	MaxRiskScore     float64 // default 0.4
	MinLiquidity     float64 // default 0.5
	MinRSI           float64 // default 50 — momentum must be building
	MinADX           float64 // default 20 — a trend must be forming

	// Exit thresholds.
	ExitTrendStrength float64 // default 0.2
	ATRExitMultiplier float64 // default 2.0 — ATR-based trailing stop

	// ATR multiplier for stop-loss.
	ATRStopMultiplier float64 // default 2.0

	// Risk-reward ratio.
	RiskRewardRatio float64 // default 3.0

	// Risk config for position sizing.
	RiskConfig config.RiskConfig
}

// NewBollingerSqueezeStrategy creates a Bollinger squeeze strategy with sensible defaults.
func NewBollingerSqueezeStrategy(riskCfg config.RiskConfig) *BollingerSqueezeStrategy {
	return &BollingerSqueezeStrategy{
		BBPeriod:          20,
		BBMultiplier:      2.0,
		SqueezeBandwidth:  0.10,  // was 0.06 — restored from original 0.10; 6% too tight for NSE stocks
		VolumeMultiplier:  1.5,   // was 2.0 — 2× volume rare on Indian exchanges; 1.5× is strong confirmation
		MinTrendStrength:  0.4,   // was 0.3 — need emerging trend confirmation
		MaxRiskScore:      0.4,   // was 0.5 — be pickier about risk
		MinLiquidity:      0.5,   // was 0.4 — trade liquid stocks only
		MinRSI:            50,    // NEW — momentum must be building
		MinADX:            20,    // NEW — a trend must be forming
		ExitTrendStrength: 0.2,
		ATRExitMultiplier: 2.0,   // NEW — ATR-based trailing stop replaces middle-band exit
		ATRStopMultiplier: 2.0,   // was 1.5 — wider stop gives trade room to breathe
		RiskRewardRatio:   3.0,   // was 2.5 — demand better payoff per trade
		RiskConfig:        riskCfg,
	}
}

func (s *BollingerSqueezeStrategy) ID() string                    { return "bollinger_squeeze_v1" }
func (s *BollingerSqueezeStrategy) Name() string                  { return "Bollinger Band Squeeze" }
func (s *BollingerSqueezeStrategy) GetStrategyType() StrategyType { return StrategyTypeVolatility }

// Evaluate applies the Bollinger squeeze rules to produce a TradeIntent.
func (s *BollingerSqueezeStrategy) Evaluate(input StrategyInput) TradeIntent {
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

func (s *BollingerSqueezeStrategy) evaluateEntry(input StrategyInput, intent TradeIntent) TradeIntent {
	// Rule 1: Only trade in BULL or SIDEWAYS regime.
	if input.Regime.Regime == RegimeBear {
		intent.Action = ActionSkip
		intent.Reason = "market regime is BEAR, Bollinger squeeze requires BULL or SIDEWAYS"
		return intent
	}

	// Rule 2: Regime confidence.
	if input.Regime.Confidence < 0.5 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("regime confidence %.2f < 0.50", input.Regime.Confidence)
		return intent
	}

	// Rule 3: Trend strength check.
	if input.Score.TrendStrengthScore < s.MinTrendStrength {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("trend strength %.2f < %.2f", input.Score.TrendStrengthScore, s.MinTrendStrength)
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

	// Rule 6: Sufficient candle history.
	if len(input.Candles) < 30 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("insufficient candle history: %d < 30", len(input.Candles))
		return intent
	}

	// Rule 7: Bollinger Band squeeze — bandwidth must be genuinely tight (< 6%).
	priorCandles := input.Candles[:len(input.Candles)-1]
	_, _, _, priorBandwidth := CalculateBollingerBands(priorCandles, s.BBPeriod, s.BBMultiplier)
	if priorBandwidth == 0 {
		intent.Action = ActionSkip
		intent.Reason = "prior Bollinger bandwidth is zero"
		return intent
	}
	if priorBandwidth > s.SqueezeBandwidth {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("prior bandwidth %.4f > %.4f (no squeeze)",
			priorBandwidth, s.SqueezeBandwidth)
		return intent
	}

	// Rule 8: Price must break above the current upper Bollinger Band.
	lastCandle := input.Candles[len(input.Candles)-1]
	_, upper, _, _ := CalculateBollingerBands(input.Candles, s.BBPeriod, s.BBMultiplier)
	if upper == 0 {
		intent.Action = ActionSkip
		intent.Reason = "upper Bollinger band is zero"
		return intent
	}
	if lastCandle.Close <= upper {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("price %.2f <= upper BB %.2f (no breakout above band)",
			lastCandle.Close, upper)
		return intent
	}

	// Rule 9: Volume confirmation — require 2× average for a real breakout.
	avgVol := AverageVolume(priorCandles, s.BBPeriod)
	if avgVol > 0 && float64(lastCandle.Volume) < avgVol*s.VolumeMultiplier {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("volume %d < %.0f (need %.1f× avg for breakout)",
			lastCandle.Volume, avgVol*s.VolumeMultiplier, s.VolumeMultiplier)
		return intent
	}

	// Rule 10: RSI must show building momentum (> 50).
	rsi := CalculateRSI(input.Candles, 14)
	if rsi < s.MinRSI {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("RSI %.1f < %.1f (no momentum confirmation)", rsi, s.MinRSI)
		return intent
	}

	// Rule 11: ADX must confirm a trend is forming (> 20).
	adx := CalculateADX(input.Candles, 14)
	if adx < s.MinADX {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("ADX %.1f < %.1f (no trend forming)", adx, s.MinADX)
		return intent
	}

	// All entry conditions met — high-quality squeeze breakout.
	atr := CalculateATR(input.Candles, 14)
	entryPrice := lastCandle.Close
	middle, _, lower, _ := CalculateBollingerBands(input.Candles, s.BBPeriod, s.BBMultiplier)

	// Stop loss: 2× ATR below entry, or lower BB if tighter.
	stopLoss := entryPrice - (atr * s.ATRStopMultiplier)
	if lower < stopLoss && lower > 0 {
		stopLoss = lower
	}
	riskPerShare := entryPrice - stopLoss
	if riskPerShare <= 0 {
		intent.Action = ActionSkip
		intent.Reason = "stop loss is above entry price"
		return intent
	}
	target := entryPrice + (riskPerShare * s.RiskRewardRatio)

	// Position sizing: risk-based with 50% capital reduction for volatility strategy.
	speculativeRisk := s.RiskConfig.MaxRiskPerTradePct * 0.5
	maxRiskAmount := input.AvailableCapital * (speculativeRisk / 100.0)
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
		"bb_squeeze: BW=%.4f upper=%.2f mid=%.2f price=%.2f vol=%d RSI=%.0f ADX=%.0f | SL=%.2f TGT=%.2f",
		priorBandwidth, upper, middle, entryPrice, lastCandle.Volume, rsi, adx, stopLoss, target,
	)
	return intent
}

func (s *BollingerSqueezeStrategy) evaluateExit(input StrategyInput, intent TradeIntent) TradeIntent {
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

	if len(input.Candles) > 0 {
		lastPrice := input.Candles[len(input.Candles)-1].Close
		atr := CalculateATR(input.Candles, 14)
		entryPrice := input.CurrentPosition.EntryPrice

		// Exit Rule 2: ATR trailing stop — price fell more than 2× ATR below entry.
		// Replaces the old "price < middle band" exit which was far too tight and
		// exited on every normal pullback within a breakout.
		atrStop := entryPrice - (atr * s.ATRExitMultiplier)
		if lastPrice < atrStop {
			intent.Action = ActionExit
			intent.Price = lastPrice
			intent.Quantity = input.CurrentPosition.Quantity
			intent.Reason = fmt.Sprintf("price %.2f fell below ATR stop %.2f (entry %.2f - %.1f*ATR %.2f)",
				lastPrice, atrStop, entryPrice, s.ATRExitMultiplier, atr)
			return intent
		}

		// Exit Rule 3: Price fell below lower Bollinger Band — complete breakdown.
		if len(input.Candles) >= s.BBPeriod {
			_, _, lower, _ := CalculateBollingerBands(input.Candles, s.BBPeriod, s.BBMultiplier)
			if lower > 0 && lastPrice < lower {
				intent.Action = ActionExit
				intent.Price = lastPrice
				intent.Quantity = input.CurrentPosition.Quantity
				intent.Reason = fmt.Sprintf("price %.2f broke below lower BB %.2f — breakdown",
					lastPrice, lower)
				return intent
			}
		}
	}

	// Exit Rule 4: Trend strength collapsed.
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

	// Otherwise, hold — let the breakout run.
	intent.Action = ActionHold
	intent.Reason = "holding: Bollinger squeeze breakout intact"
	return intent
}
