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
//   - Bollinger Bandwidth is below the squeeze threshold (tight bands)
//   - Price breaks above the upper band (expansion begins)
//   - Volume confirms the breakout
//   - Trend strength >= threshold
//   - Risk score <= threshold
//   - Sufficient candle history (30+)
//
// Exit rules:
//   - Price falls below the middle band (SMA — momentum lost)
//   - Bandwidth contracts again after expansion (end of move)
//   - Market regime changes to BEAR
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
	SqueezeBandwidth   float64 // max bandwidth for squeeze (default 0.10 = 10%)
	VolumeMultiplier   float64 // default 1.2
	MinTrendStrength   float64 // default 0.3
	MaxRiskScore       float64 // default 0.5
	MinLiquidity       float64 // default 0.4

	// Exit thresholds.
	ExitTrendStrength float64 // default 0.2

	// ATR multiplier for stop-loss.
	ATRStopMultiplier float64 // default 1.5

	// Risk-reward ratio.
	RiskRewardRatio float64 // default 2.5

	// Risk config for position sizing.
	RiskConfig config.RiskConfig
}

// NewBollingerSqueezeStrategy creates a Bollinger squeeze strategy with sensible defaults.
func NewBollingerSqueezeStrategy(riskCfg config.RiskConfig) *BollingerSqueezeStrategy {
	return &BollingerSqueezeStrategy{
		BBPeriod:          20,
		BBMultiplier:      2.0,
		SqueezeBandwidth:  0.10,
		VolumeMultiplier:  1.2,
		MinTrendStrength:  0.3,
		MaxRiskScore:      0.5,
		MinLiquidity:      0.4,
		ExitTrendStrength: 0.2,
		ATRStopMultiplier: 1.5,
		RiskRewardRatio:   2.5,
		RiskConfig:        riskCfg,
	}
}

func (s *BollingerSqueezeStrategy) ID() string   { return "bollinger_squeeze_v1" }
func (s *BollingerSqueezeStrategy) Name() string { return "Bollinger Band Squeeze" }

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

	// Rule 7: Check Bollinger Band squeeze — bandwidth must be tight.
	// Use prior candles (excluding last) to detect squeeze BEFORE breakout.
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

	// Rule 9: Volume confirmation.
	avgVol := AverageVolume(priorCandles, s.BBPeriod)
	if avgVol > 0 && float64(lastCandle.Volume) < avgVol*s.VolumeMultiplier {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("volume %d < %.0f×%.0f (no volume confirmation)",
			lastCandle.Volume, s.VolumeMultiplier, avgVol)
		return intent
	}

	// All entry conditions met.
	atr := CalculateATR(input.Candles, 14)
	entryPrice := lastCandle.Close
	middle, _, lower, _ := CalculateBollingerBands(input.Candles, s.BBPeriod, s.BBMultiplier)
	// Stop loss at the lower Bollinger band.
	stopLoss := lower
	if stopLoss >= entryPrice {
		stopLoss = entryPrice - (atr * s.ATRStopMultiplier)
	}
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
		"bb_squeeze: BW=%.4f upper=%.2f mid=%.2f price=%.2f vol=%d | SL=%.2f TGT=%.2f",
		priorBandwidth, upper, middle, entryPrice, lastCandle.Volume, stopLoss, target,
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

	// Exit Rule 2: Price fell below the middle Bollinger Band (momentum lost).
	if len(input.Candles) >= s.BBPeriod {
		middle, _, _, _ := CalculateBollingerBands(input.Candles, s.BBPeriod, s.BBMultiplier)
		lastPrice := input.Candles[len(input.Candles)-1].Close
		if middle > 0 && lastPrice < middle {
			intent.Action = ActionExit
			intent.Price = lastPrice
			intent.Quantity = input.CurrentPosition.Quantity
			intent.Reason = fmt.Sprintf("price %.2f fell below middle BB %.2f — momentum lost",
				lastPrice, middle)
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
	intent.Reason = "holding: Bollinger squeeze breakout intact"
	return intent
}
