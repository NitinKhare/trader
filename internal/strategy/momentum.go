// Package strategy - momentum.go implements a momentum swing strategy.
//
// This strategy buys top-ranked stocks with the strongest upward price momentum,
// measured by Rate of Change (ROC) and composite AI score ranking.
//
// Momentum works on the principle that stocks that have been going up tend to
// continue going up (persistence). We select the highest-ranked stocks by AI
// composite score and confirm with price ROC.
//
// Entry rules:
//   - Market regime must be BULL
//   - Composite score rank <= threshold (top N, default 5)
//   - Trend strength >= threshold (0.7 — very strong trend)
//   - Breakout quality >= threshold (0.6)
//   - ROC(10) > threshold (5% — strong upward momentum)
//   - Risk score <= threshold (0.3 — very strict)
//   - Sufficient candle history (30+)
//
// Exit rules:
//   - ROC turns negative (momentum reversal)
//   - Stock drops out of top 10 rank
//   - Trend strength drops below 0.5
//   - Market regime changes to BEAR
package strategy

import (
	"fmt"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// MomentumStrategy implements a momentum-based swing strategy.
type MomentumStrategy struct {
	// Entry thresholds.
	MaxRank            int     // default 5 (top 5 stocks only)
	MinTrendStrength   float64 // default 0.7
	MinBreakoutQuality float64 // default 0.6
	MinROC             float64 // default 0.05 (5%)
	MaxRiskScore       float64 // default 0.3 (very strict)
	MinLiquidity       float64 // default 0.6
	ROCPeriod          int     // default 10

	// Exit thresholds.
	ExitMaxRank        int     // default 10 (exit if drops out of top 10)
	ExitTrendStrength  float64 // default 0.5

	// ATR multiplier for stop-loss (wider for volatile momentum stocks).
	ATRStopMultiplier float64 // default 2.5

	// Risk-reward ratio.
	RiskRewardRatio float64 // default 2.5

	// Risk config for position sizing.
	RiskConfig config.RiskConfig
}

// NewMomentumStrategy creates a momentum strategy with sensible defaults.
func NewMomentumStrategy(riskCfg config.RiskConfig) *MomentumStrategy {
	return &MomentumStrategy{
		MaxRank:            5,
		MinTrendStrength:   0.7,
		MinBreakoutQuality: 0.6,
		MinROC:             0.05,
		MaxRiskScore:       0.3,
		MinLiquidity:       0.6,
		ROCPeriod:          10,
		ExitMaxRank:        10,
		ExitTrendStrength:  0.5,
		ATRStopMultiplier:  2.5,
		RiskRewardRatio:    2.5,
		RiskConfig:         riskCfg,
	}
}

func (s *MomentumStrategy) ID() string   { return "momentum_v1" }
func (s *MomentumStrategy) Name() string { return "Momentum Swing" }

// Evaluate applies the momentum rules to produce a TradeIntent.
func (s *MomentumStrategy) Evaluate(input StrategyInput) TradeIntent {
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

func (s *MomentumStrategy) evaluateEntry(input StrategyInput, intent TradeIntent) TradeIntent {
	// Rule 1: Only trade in BULL regime.
	if input.Regime.Regime != RegimeBull {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("market regime is %s, momentum requires BULL", input.Regime.Regime)
		return intent
	}

	// Rule 2: Regime confidence must be high.
	if input.Regime.Confidence < 0.7 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("regime confidence %.2f < 0.70", input.Regime.Confidence)
		return intent
	}

	// Rule 3: Stock must be top-ranked by composite score.
	if input.Score.Rank > s.MaxRank {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("rank %d > %d (not in top momentum stocks)", input.Score.Rank, s.MaxRank)
		return intent
	}

	// Rule 4: Trend strength must be very strong.
	if input.Score.TrendStrengthScore < s.MinTrendStrength {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("trend strength %.2f < %.2f", input.Score.TrendStrengthScore, s.MinTrendStrength)
		return intent
	}

	// Rule 5: Breakout quality check.
	if input.Score.BreakoutQualityScore < s.MinBreakoutQuality {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("breakout quality %.2f < %.2f", input.Score.BreakoutQualityScore, s.MinBreakoutQuality)
		return intent
	}

	// Rule 6: Liquidity check.
	if input.Score.LiquidityScore < s.MinLiquidity {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("liquidity %.2f < %.2f", input.Score.LiquidityScore, s.MinLiquidity)
		return intent
	}

	// Rule 7: Risk score check (very strict for momentum).
	if input.Score.RiskScore > s.MaxRiskScore {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("risk score %.2f > %.2f", input.Score.RiskScore, s.MaxRiskScore)
		return intent
	}

	// Rule 8: Must have sufficient candle history.
	if len(input.Candles) < 30 {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("insufficient candle history: %d < 30", len(input.Candles))
		return intent
	}

	// Rule 9: ROC must be positive and above threshold.
	roc := CalculateROC(input.Candles, s.ROCPeriod)
	if roc < s.MinROC {
		intent.Action = ActionSkip
		intent.Reason = fmt.Sprintf("ROC(10) %.2f%% < %.2f%% (insufficient momentum)", roc*100, s.MinROC*100)
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
		"momentum: rank=%d ROC=%.1f%% trend=%.2f | ATR=%.2f SL=%.2f TGT=%.2f",
		input.Score.Rank, roc*100, input.Score.TrendStrengthScore, atr, stopLoss, target,
	)
	return intent
}

func (s *MomentumStrategy) evaluateExit(input StrategyInput, intent TradeIntent) TradeIntent {
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

	// Exit Rule 2: Momentum reversal (ROC turned negative).
	if len(input.Candles) >= s.ROCPeriod+1 {
		roc := CalculateROC(input.Candles, s.ROCPeriod)
		if roc < 0 {
			intent.Action = ActionExit
			intent.Quantity = input.CurrentPosition.Quantity
			intent.Reason = fmt.Sprintf("ROC(10) = %.1f%% — momentum reversal", roc*100)
			if len(input.Candles) > 0 {
				intent.Price = input.Candles[len(input.Candles)-1].Close
			}
			return intent
		}
	}

	// Exit Rule 3: Stock dropped out of top N.
	if input.Score.Rank > s.ExitMaxRank {
		intent.Action = ActionExit
		intent.Quantity = input.CurrentPosition.Quantity
		intent.Reason = fmt.Sprintf("rank dropped to %d (> %d)", input.Score.Rank, s.ExitMaxRank)
		if len(input.Candles) > 0 {
			intent.Price = input.Candles[len(input.Candles)-1].Close
		}
		return intent
	}

	// Exit Rule 4: Trend strength fading.
	if input.Score.TrendStrengthScore < s.ExitTrendStrength {
		intent.Action = ActionExit
		intent.Quantity = input.CurrentPosition.Quantity
		intent.Reason = fmt.Sprintf("trend strength dropped to %.2f < %.2f — momentum fading",
			input.Score.TrendStrengthScore, s.ExitTrendStrength)
		if len(input.Candles) > 0 {
			intent.Price = input.Candles[len(input.Candles)-1].Close
		}
		return intent
	}

	// Otherwise, hold.
	intent.Action = ActionHold
	intent.Reason = fmt.Sprintf("holding: rank=%d trend=%.2f — momentum intact",
		input.Score.Rank, input.Score.TrendStrengthScore)
	return intent
}
