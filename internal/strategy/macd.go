// Package strategy - macd.go implements a MACD crossover swing strategy.
//
// This strategy buys when the MACD line crosses above the signal line (bullish
// crossover), confirming with the histogram turning positive and the overall
// market trend. MACD combines trend and momentum in one indicator — the crossover
// signals a shift from bearish to bullish momentum.
//
// Standard MACD parameters: fast=12, slow=26, signal=9.
//
// Entry rules:
//   - Market regime is BULL
//   - MACD line crosses above signal line (bullish crossover)
//   - MACD histogram is positive (confirms crossover)
//   - MACD line is negative or near zero (early in the move, not late)
//   - Trend strength >= threshold
//   - Risk score <= threshold
//   - Liquidity >= threshold
//   - Sufficient candle history (40+)
//
// Exit rules:
//   - MACD line crosses below signal line (bearish crossover)
//   - Histogram turns negative after being positive
//   - Market regime changes to BEAR
package strategy

import (
	"fmt"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// MACDCrossoverStrategy implements a MACD crossover swing strategy.
type MACDCrossoverStrategy struct {
	// MACD parameters.
	FastPeriod   int // default 12
	SlowPeriod   int // default 26
	SignalPeriod int // default 9

	// Entry thresholds.
	MinTrendStrength float64 // default 0.4
	MaxRiskScore     float64 // default 0.5
	MinLiquidity     float64 // default 0.4
	MaxMACDForEntry  float64 // MACD line must be below this to avoid late entries (default 0)

	// Exit thresholds.
	ExitTrendStrength float64 // default 0.25

	// ATR multiplier for stop-loss.
	ATRStopMultiplier float64 // default 2.0

	// Risk-reward ratio.
	RiskRewardRatio float64 // default 2.0

	// Risk config for position sizing.
	RiskConfig config.RiskConfig
}

// NewMACDCrossoverStrategy creates a MACD crossover strategy with sensible defaults.
func NewMACDCrossoverStrategy(riskCfg config.RiskConfig) *MACDCrossoverStrategy {
	return &MACDCrossoverStrategy{
		FastPeriod:        12,
		SlowPeriod:        26,
		SignalPeriod:      9,
		MinTrendStrength:  0.4,
		MaxRiskScore:      0.5,
		MinLiquidity:      0.4,
		MaxMACDForEntry:   0,
		ExitTrendStrength: 0.25,
		ATRStopMultiplier: 2.0,
		RiskRewardRatio:   2.0,
		RiskConfig:        riskCfg,
	}
}

func (s *MACDCrossoverStrategy) ID() string   { return "macd_crossover_v1" }
func (s *MACDCrossoverStrategy) Name() string { return "MACD Crossover" }

// Evaluate applies the MACD crossover rules to produce a TradeIntent.
func (s *MACDCrossoverStrategy) Evaluate(input StrategyInput) TradeIntent {
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

func (s *MACDCrossoverStrategy) evaluateEntry(input StrategyInput, intent TradeIntent) TradeIntent {
	// Rule 1: Only trade in BULL regime.
	if input.Regime.Regime != RegimeBull {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("market regime is %s, MACD crossover requires BULL", input.Regime.Regime)
		return intent
	}

	// Rule 2: Regime confidence.
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

	// Rule 6: Sufficient candle history for MACD (26 + 9 + buffer).
	if len(input.Candles) < 40 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("insufficient candle history: %d < 40", len(input.Candles))
		return intent
	}

	// Rule 7: MACD bullish crossover — current MACD > signal, previous MACD <= signal.
	macdLine, signalLine, histogram := CalculateMACD(input.Candles, s.FastPeriod, s.SlowPeriod, s.SignalPeriod)
	prevMACD, prevSignal := CalculatePrevMACD(input.Candles, s.FastPeriod, s.SlowPeriod, s.SignalPeriod)

	// Current: MACD above signal.
	if macdLine <= signalLine {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("MACD %.4f <= signal %.4f (no bullish crossover)", macdLine, signalLine)
		return intent
	}

	// Previous: MACD was at or below signal.
	if prevMACD > prevSignal {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("prev MACD %.4f > prev signal %.4f (not a fresh crossover)", prevMACD, prevSignal)
		return intent
	}

	// Rule 8: Histogram must be positive (confirms crossover momentum).
	if histogram <= 0 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("MACD histogram %.4f <= 0 (no momentum confirmation)", histogram)
		return intent
	}

	// Rule 9: MACD line should not be too far above zero (avoid late entries).
	if s.MaxMACDForEntry != 0 && macdLine > s.MaxMACDForEntry {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("MACD %.4f > %.4f (too late in the move)", macdLine, s.MaxMACDForEntry)
		return intent
	}

	// All entry conditions met.
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
		"macd_xover: MACD=%.4f signal=%.4f hist=%.4f | ATR=%.2f SL=%.2f TGT=%.2f",
		macdLine, signalLine, histogram, atr, stopLoss, target,
	)
	return intent
}

func (s *MACDCrossoverStrategy) evaluateExit(input StrategyInput, intent TradeIntent) TradeIntent {
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

	// Exit Rule 2: MACD bearish crossover (MACD crosses below signal).
	if len(input.Candles) >= s.SlowPeriod+s.SignalPeriod {
		macdLine, signalLine, histogram := CalculateMACD(input.Candles, s.FastPeriod, s.SlowPeriod, s.SignalPeriod)
		if macdLine < signalLine || histogram < 0 {
			intent.Action = ActionExit
			intent.Quantity = input.CurrentPosition.Quantity
			intent.Reason = fmt.Sprintf("MACD bearish: MACD=%.4f signal=%.4f hist=%.4f",
				macdLine, signalLine, histogram)
			if len(input.Candles) > 0 {
				intent.Price = input.Candles[len(input.Candles)-1].Close
			}
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
	intent.Reason = "holding: MACD crossover intact"
	return intent
}
