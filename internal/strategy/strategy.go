// Package strategy defines the strategy framework.
//
// Design rules (from spec):
//   - A strategy is a pure decision engine.
//   - Strategies are stateless, deterministic, and testable in isolation.
//   - AI advises (via scores), rules decide.
//   - AI never places orders — strategies produce TradeIntents,
//     which are then validated by risk management before execution.
package strategy

import (
	"time"
)

// TradeAction represents what a strategy wants to do.
type TradeAction string

const (
	ActionBuy  TradeAction = "BUY"
	ActionHold TradeAction = "HOLD"
	ActionExit TradeAction = "EXIT"
	ActionSkip TradeAction = "SKIP"
)

// MarketRegime represents the AI-determined market condition.
type MarketRegime string

const (
	RegimeBull     MarketRegime = "BULL"
	RegimeSideways MarketRegime = "SIDEWAYS"
	RegimeBear     MarketRegime = "BEAR"
)

// MarketRegimeData holds the AI-generated market regime assessment.
type MarketRegimeData struct {
	Date       string       `json:"date"`
	Regime     MarketRegime `json:"regime"`
	Confidence float64      `json:"confidence"`
}

// StockScore holds all AI-generated scores for a single stock.
// These are read from the file-based AI output (Parquet/JSON).
type StockScore struct {
	Symbol              string  `json:"symbol"`
	TrendStrengthScore  float64 `json:"trend_strength_score"`  // 0–1
	BreakoutQualityScore float64 `json:"breakout_quality_score"` // 0–1
	VolatilityScore     float64 `json:"volatility_score"`      // 0–1
	RiskScore            float64 `json:"risk_score"`             // 0–1
	LiquidityScore       float64 `json:"liquidity_score"`       // 0–1
	CompositeScore       float64 `json:"composite_score"`       // 0–1, weighted combination
	Rank                 int     `json:"rank"`                  // 1-based rank (1 = best)
}

// Candle represents a single OHLCV bar.
type Candle struct {
	Symbol string
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

// PositionInfo provides current position state to the strategy.
type PositionInfo struct {
	Symbol       string
	EntryPrice   float64
	Quantity     int
	StopLoss     float64
	Target       float64
	EntryTime    time.Time
	StrategyID   string
	SignalID     string
}

// StrategyInput is the complete input bundle passed to a strategy.
// It contains everything a strategy needs to make a decision.
type StrategyInput struct {
	// Current date being evaluated.
	Date time.Time

	// AI-determined market regime.
	Regime MarketRegimeData

	// AI scores for the stock being evaluated.
	Score StockScore

	// Historical candles for this stock (most recent last).
	Candles []Candle

	// Current position in this stock, nil if no position.
	CurrentPosition *PositionInfo

	// Total number of currently open positions (across all stocks).
	OpenPositionCount int

	// Available capital for new trades.
	AvailableCapital float64
}

// TradeIntent is what a strategy produces — a desire to trade.
// This is NOT an order. It must pass risk management before becoming an order.
type TradeIntent struct {
	// StrategyID identifies which strategy generated this intent.
	StrategyID string

	// SignalID is a unique identifier for this specific signal (for tracing).
	SignalID string

	// Symbol is the stock ticker.
	Symbol string

	// Action is what the strategy wants to do.
	Action TradeAction

	// Price is the desired entry/exit price.
	Price float64

	// StopLoss is the mandatory stop-loss price for BUY intents.
	StopLoss float64

	// Target is the profit target price.
	Target float64

	// Quantity is the desired number of shares.
	Quantity int

	// Reason explains why this decision was made (for logging/explainability).
	Reason string

	// Scores snapshot at the time of decision (for audit trail).
	Scores StockScore
}

// Strategy is the interface that all trading strategies must implement.
// Strategies must be:
//   - Pure functions: same input → same output.
//   - Stateless: no internal mutable state.
//   - Deterministic: no randomness.
type Strategy interface {
	// ID returns the unique identifier for this strategy.
	ID() string

	// Name returns a human-readable name for this strategy.
	Name() string

	// Evaluate takes a StrategyInput and produces a TradeIntent.
	// It must never produce side effects (no I/O, no state changes).
	Evaluate(input StrategyInput) TradeIntent
}
