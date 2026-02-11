// Package strategy - mean_reversion.go implements a mean reversion swing strategy.
//
// This strategy buys fundamentally strong stocks that have dropped to oversold levels,
// expecting a reversion to the mean (20-day SMA).
//
// It works best in BULL or SIDEWAYS regimes where stocks oscillate around their average
// rather than trending persistently in one direction.
//
// Entry rules:
//   - Market regime is BULL or SIDEWAYS
//   - Trend strength < threshold (stock is NOT trending — necessary for reversion)
//   - RSI(14) < oversold threshold (35)
//   - Current price < 20-day SMA (below the mean)
//   - Risk score <= threshold
//   - Liquidity score >= threshold
//   - Sufficient candle history (30+)
//
// Exit rules:
//   - Price crosses above 20-day SMA (mean reversion target reached)
//   - RSI > overbought threshold (65) (reversion overshot)
//   - Trend strength rises above 0.7 (stock started trending — wrong strategy)
//   - Market regime changes to BEAR
package strategy

import (
	"fmt"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// MeanReversionStrategy implements a mean reversion swing strategy.
type MeanReversionStrategy struct {
	// Entry thresholds.
	MaxTrendStrength       float64 // stock must NOT be trending (default 0.4)
	RSIOversoldThreshold   float64 // default 35
	MaxRiskScore           float64 // default 0.6
	MinLiquidity           float64 // default 0.4
	SMALookback            int     // default 20

	// Exit thresholds.
	RSIOverboughtThreshold float64 // default 65
	ExitTrendStrength      float64 // stock started trending — exit (default 0.7)

	// ATR multiplier for stop-loss.
	ATRStopMultiplier float64 // default 1.5 (tighter than trend follow)

	// Risk-reward ratio for position sizing validation.
	RiskRewardRatio float64 // default 1.5

	// Risk config for position sizing.
	RiskConfig config.RiskConfig
}

// NewMeanReversionStrategy creates a mean reversion strategy with sensible defaults.
func NewMeanReversionStrategy(riskCfg config.RiskConfig) *MeanReversionStrategy {
	return &MeanReversionStrategy{
		MaxTrendStrength:       0.4,
		RSIOversoldThreshold:   35,
		MaxRiskScore:           0.6,
		MinLiquidity:           0.4,
		SMALookback:            20,
		RSIOverboughtThreshold: 65,
		ExitTrendStrength:      0.7,
		ATRStopMultiplier:      1.5,
		RiskRewardRatio:        1.5,
		RiskConfig:             riskCfg,
	}
}

func (s *MeanReversionStrategy) ID() string   { return "mean_reversion_v1" }
func (s *MeanReversionStrategy) Name() string { return "Mean Reversion Swing" }

// Evaluate applies the mean reversion rules to produce a TradeIntent.
func (s *MeanReversionStrategy) Evaluate(input StrategyInput) TradeIntent {
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

func (s *MeanReversionStrategy) evaluateEntry(input StrategyInput, intent TradeIntent) TradeIntent {
	// Rule 1: Only trade in BULL or SIDEWAYS regime.
	if input.Regime.Regime == RegimeBear {
		intent.Action = ActionSkip
		intent.Reason = "market regime is BEAR, mean reversion requires BULL or SIDEWAYS"
		return intent
	}

	// Rule 2: Regime confidence must be sufficient.
	if input.Regime.Confidence < 0.5 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("regime confidence %.2f < 0.50", input.Regime.Confidence)
		return intent
	}

	// Rule 3: Stock must NOT be strongly trending (mean reversion works on non-trending).
	if input.Score.TrendStrengthScore >= s.MaxTrendStrength {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("trend strength %.2f >= %.2f (stock is trending, not mean-reverting)",
			input.Score.TrendStrengthScore, s.MaxTrendStrength)
		return intent
	}

	// Rule 4: Risk score check.
	if input.Score.RiskScore > s.MaxRiskScore {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("risk score %.2f > %.2f", input.Score.RiskScore, s.MaxRiskScore)
		return intent
	}

	// Rule 5: Liquidity check.
	if input.Score.LiquidityScore < s.MinLiquidity {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("liquidity %.2f < %.2f", input.Score.LiquidityScore, s.MinLiquidity)
		return intent
	}

	// Rule 6: Must have sufficient candle history.
	if len(input.Candles) < 30 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("insufficient candle history: %d < 30", len(input.Candles))
		return intent
	}

	// Rule 7: RSI must be oversold.
	rsi := CalculateRSI(input.Candles, 14)
	if rsi >= s.RSIOversoldThreshold {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("RSI %.1f >= %.1f (not oversold)", rsi, s.RSIOversoldThreshold)
		return intent
	}

	// Rule 8: Price must be below the 20-day SMA (below the mean).
	sma := CalculateSMA(input.Candles, s.SMALookback)
	lastCandle := input.Candles[len(input.Candles)-1]
	if lastCandle.Close >= sma {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("price %.2f >= SMA(20) %.2f (not below mean)", lastCandle.Close, sma)
		return intent
	}

	// All entry conditions met — calculate stop loss, target, and quantity.
	atr := CalculateATR(input.Candles, 14)
	entryPrice := lastCandle.Close
	stopLoss := entryPrice - (atr * s.ATRStopMultiplier)
	// Target is the SMA (the mean we expect price to revert to).
	target := sma
	riskPerShare := entryPrice - stopLoss

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
		"mean_rev: RSI=%.1f SMA=%.2f price=%.2f | ATR=%.2f SL=%.2f TGT=%.2f",
		rsi, sma, entryPrice, atr, stopLoss, target,
	)
	return intent
}

func (s *MeanReversionStrategy) evaluateExit(input StrategyInput, intent TradeIntent) TradeIntent {
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

	if len(input.Candles) >= s.SMALookback {
		sma := CalculateSMA(input.Candles, s.SMALookback)
		lastPrice := input.Candles[len(input.Candles)-1].Close

		// Exit Rule 2: Price crossed above SMA (mean reversion target reached).
		if lastPrice > sma {
			intent.Action = ActionExit
			intent.Price = lastPrice
			intent.Quantity = input.CurrentPosition.Quantity
			intent.Reason = fmt.Sprintf("price %.2f crossed above SMA(20) %.2f — mean reversion target reached", lastPrice, sma)
			return intent
		}
	}

	// Exit Rule 3: RSI became overbought (reversion overshot).
	if len(input.Candles) > 14 {
		rsi := CalculateRSI(input.Candles, 14)
		if rsi > s.RSIOverboughtThreshold {
			intent.Action = ActionExit
			intent.Quantity = input.CurrentPosition.Quantity
			intent.Reason = fmt.Sprintf("RSI %.1f > %.1f — overbought, reversion complete", rsi, s.RSIOverboughtThreshold)
			if len(input.Candles) > 0 {
				intent.Price = input.Candles[len(input.Candles)-1].Close
			}
			return intent
		}
	}

	// Exit Rule 4: Stock started trending strongly (wrong strategy for trending).
	if input.Score.TrendStrengthScore > s.ExitTrendStrength {
		intent.Action = ActionExit
		intent.Quantity = input.CurrentPosition.Quantity
		intent.Reason = fmt.Sprintf("trend strength %.2f > %.2f — stock is trending, exit mean reversion",
			input.Score.TrendStrengthScore, s.ExitTrendStrength)
		if len(input.Candles) > 0 {
			intent.Price = input.Candles[len(input.Candles)-1].Close
		}
		return intent
	}

	// Otherwise, hold.
	intent.Action = ActionHold
	intent.Reason = fmt.Sprintf("holding: RSI near oversold, waiting for mean reversion to SMA")
	return intent
}
