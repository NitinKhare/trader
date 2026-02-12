// Package strategy - vwap.go implements a VWAP reversion swing strategy.
//
// This strategy buys when price dips significantly below VWAP (Volume Weighted
// Average Price) and shows signs of reverting back. VWAP acts as a fair value
// anchor — institutional traders often accumulate near VWAP, making it a
// natural support/resistance level.
//
// Entry rules:
//   - Market regime is BULL or SIDEWAYS
//   - Price is below VWAP by at least the deviation threshold
//   - RSI(14) is in oversold zone (< 40) confirming the dip
//   - Volatility score is moderate (not too wild)
//   - Liquidity score >= threshold (VWAP is meaningless without volume)
//   - Risk score <= threshold
//   - Sufficient candle history (30+)
//
// Exit rules:
//   - Price crosses above VWAP (reversion target reached)
//   - Price exceeds VWAP by overshoot threshold (take profit)
//   - RSI becomes overbought (> 65)
//   - Market regime changes to BEAR
package strategy

import (
	"fmt"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// VWAPReversionStrategy implements a VWAP-anchored mean reversion strategy.
type VWAPReversionStrategy struct {
	// Entry thresholds.
	VWAPDeviationPct     float64 // min % price must be below VWAP (default 2.0)
	RSIOversoldThreshold float64 // default 40
	MaxVolatility        float64 // max volatility score (default 0.7)
	MinLiquidity         float64 // default 0.5 (VWAP needs good volume)
	MaxRiskScore         float64 // default 0.5
	VWAPLookback         int     // candles for VWAP calc (default 20)

	// Exit thresholds.
	VWAPOvershootPct       float64 // % above VWAP to take profit (default 1.5)
	RSIOverboughtThreshold float64 // default 65

	// ATR multiplier for stop-loss.
	ATRStopMultiplier float64 // default 1.5

	// Risk config for position sizing.
	RiskConfig config.RiskConfig
}

// NewVWAPReversionStrategy creates a VWAP reversion strategy with sensible defaults.
func NewVWAPReversionStrategy(riskCfg config.RiskConfig) *VWAPReversionStrategy {
	return &VWAPReversionStrategy{
		VWAPDeviationPct:       2.0,
		RSIOversoldThreshold:   40,
		MaxVolatility:          0.7,
		MinLiquidity:           0.5,
		MaxRiskScore:           0.5,
		VWAPLookback:           20,
		VWAPOvershootPct:       1.5,
		RSIOverboughtThreshold: 65,
		ATRStopMultiplier:      1.5,
		RiskConfig:             riskCfg,
	}
}

func (s *VWAPReversionStrategy) ID() string   { return "vwap_reversion_v1" }
func (s *VWAPReversionStrategy) Name() string { return "VWAP Reversion" }

// Evaluate applies the VWAP reversion rules to produce a TradeIntent.
func (s *VWAPReversionStrategy) Evaluate(input StrategyInput) TradeIntent {
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

func (s *VWAPReversionStrategy) evaluateEntry(input StrategyInput, intent TradeIntent) TradeIntent {
	// Rule 1: Only trade in BULL or SIDEWAYS regime.
	if input.Regime.Regime == RegimeBear {
		intent.Action = ActionSkip
		intent.Reason = "market regime is BEAR, VWAP reversion requires BULL or SIDEWAYS"
		return intent
	}

	// Rule 2: Regime confidence.
	if input.Regime.Confidence < 0.5 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("regime confidence %.2f < 0.50", input.Regime.Confidence)
		return intent
	}

	// Rule 3: Liquidity check (VWAP is meaningless in illiquid stocks).
	if input.Score.LiquidityScore < s.MinLiquidity {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("liquidity %.2f < %.2f (VWAP unreliable)", input.Score.LiquidityScore, s.MinLiquidity)
		return intent
	}

	// Rule 4: Risk score check.
	if input.Score.RiskScore > s.MaxRiskScore {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("risk score %.2f > %.2f", input.Score.RiskScore, s.MaxRiskScore)
		return intent
	}

	// Rule 5: Volatility check (too volatile = unreliable reversion).
	if input.Score.VolatilityScore > s.MaxVolatility {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("volatility %.2f > %.2f (too volatile for VWAP reversion)",
			input.Score.VolatilityScore, s.MaxVolatility)
		return intent
	}

	// Rule 6: Sufficient candle history.
	if len(input.Candles) < 30 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("insufficient candle history: %d < 30", len(input.Candles))
		return intent
	}

	lastCandle := input.Candles[len(input.Candles)-1]
	vwap := CalculateVWAP(input.Candles, s.VWAPLookback)
	if vwap == 0 {
		intent.Action = ActionSkip
		intent.Reason = "VWAP is zero (no volume data)"
		return intent
	}

	// Rule 7: Price must be below VWAP by at least the deviation threshold.
	deviationPct := (vwap - lastCandle.Close) / vwap * 100
	if deviationPct < s.VWAPDeviationPct {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("price %.2f only %.1f%% below VWAP %.2f (need %.1f%%)",
			lastCandle.Close, deviationPct, vwap, s.VWAPDeviationPct)
		return intent
	}

	// Rule 8: RSI confirms oversold.
	rsi := CalculateRSI(input.Candles, 14)
	if rsi >= s.RSIOversoldThreshold {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("RSI %.1f >= %.1f (not oversold enough)", rsi, s.RSIOversoldThreshold)
		return intent
	}

	// All entry conditions met.
	atr := CalculateATR(input.Candles, 14)
	entryPrice := lastCandle.Close
	stopLoss := entryPrice - (atr * s.ATRStopMultiplier)
	// Target is VWAP (the fair value we expect price to revert to).
	target := vwap
	riskPerShare := entryPrice - stopLoss

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
		"vwap_rev: VWAP=%.2f price=%.2f dev=%.1f%% RSI=%.1f | ATR=%.2f SL=%.2f TGT=%.2f",
		vwap, entryPrice, deviationPct, rsi, atr, stopLoss, target,
	)
	return intent
}

func (s *VWAPReversionStrategy) evaluateExit(input StrategyInput, intent TradeIntent) TradeIntent {
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

	if len(input.Candles) >= s.VWAPLookback {
		vwap := CalculateVWAP(input.Candles, s.VWAPLookback)
		lastPrice := input.Candles[len(input.Candles)-1].Close

		// Exit Rule 2: Price crossed above VWAP (reversion target).
		if vwap > 0 && lastPrice > vwap {
			overshootPct := (lastPrice - vwap) / vwap * 100
			// Exit if at or above VWAP — reversion complete.
			intent.Action = ActionExit
			intent.Price = lastPrice
			intent.Quantity = input.CurrentPosition.Quantity
			if overshootPct >= s.VWAPOvershootPct {
				intent.Reason = fmt.Sprintf("price %.2f is %.1f%% above VWAP %.2f — taking profit",
					lastPrice, overshootPct, vwap)
			} else {
				intent.Reason = fmt.Sprintf("price %.2f crossed above VWAP %.2f — reversion complete",
					lastPrice, vwap)
			}
			return intent
		}
	}

	// Exit Rule 3: RSI overbought.
	if len(input.Candles) > 14 {
		rsi := CalculateRSI(input.Candles, 14)
		if rsi > s.RSIOverboughtThreshold {
			intent.Action = ActionExit
			intent.Quantity = input.CurrentPosition.Quantity
			intent.Reason = fmt.Sprintf("RSI %.1f > %.1f — overbought", rsi, s.RSIOverboughtThreshold)
			if len(input.Candles) > 0 {
				intent.Price = input.Candles[len(input.Candles)-1].Close
			}
			return intent
		}
	}

	// Otherwise, hold.
	intent.Action = ActionHold
	intent.Reason = "holding: waiting for price to revert to VWAP"
	return intent
}
