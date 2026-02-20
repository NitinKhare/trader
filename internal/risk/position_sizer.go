// Package risk - position_sizer.go provides centralized, volatility-based position sizing.
//
// Instead of each strategy computing its own position size, the position sizer
// uses ATR to determine stop distance and sizes positions so that the risk per trade
// is a fixed percentage of account equity.
//
// Formula:
//
//	StopDistance = ATR(14) × ATRMultiplier
//	RiskAmount  = Equity × RiskPerTradePct / 100
//	Quantity    = RiskAmount / StopDistance
//
// The position is also capped so that its total value doesn't exceed MaxPositionPct of equity.
package risk

import (
	"math"

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// PositionSizerConfig holds position sizing parameters.
type PositionSizerConfig struct {
	// ATRMultiplier determines stop distance. StopDistance = ATR × this value.
	// Default: 1.5
	ATRMultiplier float64 `json:"atr_multiplier"`

	// RiskPerTradePct is the percentage of equity risked per trade.
	// Default: 1.0 (1% of equity)
	RiskPerTradePct float64 `json:"risk_per_trade_pct"`

	// MaxPositionPct caps the total position value as a percentage of equity.
	// Default: 10.0 (10% of equity)
	MaxPositionPct float64 `json:"max_position_pct"`
}

// PositionSizer computes position sizes based on volatility (ATR).
type PositionSizer struct {
	config PositionSizerConfig
}

// NewPositionSizer creates a position sizer with the given config.
// Applies defaults for zero-value fields.
func NewPositionSizer(config PositionSizerConfig) *PositionSizer {
	if config.ATRMultiplier <= 0 {
		config.ATRMultiplier = 1.5
	}
	if config.RiskPerTradePct <= 0 {
		config.RiskPerTradePct = 1.0
	}
	if config.MaxPositionPct <= 0 {
		config.MaxPositionPct = 10.0
	}
	return &PositionSizer{config: config}
}

// SizeResult holds the computed position sizing output.
type SizeResult struct {
	Quantity     int     // Number of shares to buy
	StopLoss     float64 // Computed stop-loss price
	StopDistance float64 // Distance from entry to stop (ATR-based)
	RiskAmount   float64 // Total capital at risk
	OrderValue   float64 // Total cost of the position (entry × quantity)
}

// Size computes the position size for a given stock.
// equity: current total portfolio equity
// entryPrice: proposed entry price
// candles: historical OHLCV data for ATR calculation
//
// Returns SizeResult. If ATR is 0 or data is insufficient, returns 0 quantity.
func (ps *PositionSizer) Size(equity, entryPrice float64, candles []strategy.Candle) SizeResult {
	result := SizeResult{}

	if equity <= 0 || entryPrice <= 0 || len(candles) == 0 {
		return result
	}

	// Compute ATR for stop distance.
	atr := strategy.CalculateATR(candles, 14)
	if atr <= 0 {
		return result
	}

	result.StopDistance = atr * ps.config.ATRMultiplier
	result.StopLoss = entryPrice - result.StopDistance

	// Ensure stop loss is positive.
	if result.StopLoss <= 0 {
		result.StopLoss = entryPrice * 0.95 // fallback: 5% below entry
		result.StopDistance = entryPrice - result.StopLoss
	}

	// Risk amount = equity × risk percentage.
	result.RiskAmount = equity * (ps.config.RiskPerTradePct / 100.0)

	// Quantity from risk.
	qtyFromRisk := result.RiskAmount / result.StopDistance

	// Cap by maximum position size.
	maxPositionValue := equity * (ps.config.MaxPositionPct / 100.0)
	qtyFromMaxPosition := maxPositionValue / entryPrice

	// Take the smaller of the two.
	qty := math.Min(qtyFromRisk, qtyFromMaxPosition)

	// Round down to whole shares.
	result.Quantity = int(math.Floor(qty))
	if result.Quantity < 1 {
		result.Quantity = 0 // can't afford even 1 share within risk limits
		return result
	}

	result.OrderValue = entryPrice * float64(result.Quantity)
	// Recalculate actual risk with final quantity.
	result.RiskAmount = result.StopDistance * float64(result.Quantity)

	return result
}
