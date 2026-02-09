// Package storage defines the database storage interfaces and types.
//
// Uses Postgres + TimescaleDB for:
//   - OHLCV candles
//   - Signals
//   - Trades
//   - Positions
//   - AI scores
//   - Logs
package storage

import (
	"context"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// TradeRecord represents a completed or active trade in the database.
// Every trade has full traceability: strategy ID, signal ID, entry/exit details, and reason.
type TradeRecord struct {
	ID           int64
	StrategyID   string
	SignalID     string
	Symbol       string
	Side         string // "BUY" or "SELL"
	Quantity     int
	EntryPrice   float64
	ExitPrice    float64
	StopLoss     float64
	Target       float64
	EntryTime    time.Time
	ExitTime     *time.Time // nil if still open
	ExitReason   string     // "stop_loss", "target", "time_exit", "strategy_invalidation", "manual"
	PnL          float64
	Status       string // "open", "closed"
	CreatedAt    time.Time
}

// SignalRecord represents a strategy signal in the database.
type SignalRecord struct {
	ID           int64
	StrategyID   string
	SignalID     string
	Symbol       string
	Action       string // "BUY", "HOLD", "EXIT", "SKIP"
	Price        float64
	StopLoss     float64
	Target       float64
	Quantity     int
	Reason       string
	Scores       strategy.StockScore
	Approved     bool   // Whether risk management approved this signal
	RejectionReason string
	Date         time.Time
	CreatedAt    time.Time
}

// AIScoreRecord stores AI scores for audit and historical analysis.
type AIScoreRecord struct {
	ID                   int64
	Symbol               string
	Date                 time.Time
	TrendStrengthScore   float64
	BreakoutQualityScore float64
	VolatilityScore      float64
	RiskScore            float64
	LiquidityScore       float64
	CreatedAt            time.Time
}

// TradeLog represents a detailed log entry for audit trail.
type TradeLog struct {
	ID         int64
	Timestamp  time.Time
	StrategyID string
	Symbol     string
	Action     string
	ReasonCode string
	Message    string
	InputsJSON string // JSON snapshot of inputs for this decision
}

// Store defines the complete storage interface for the trading system.
type Store interface {
	// Candle operations (implements market.DataStore).
	SaveCandles(ctx context.Context, candles []strategy.Candle) error
	GetCandles(ctx context.Context, symbol string, from, to time.Time) ([]strategy.Candle, error)
	GetLatestCandleDate(ctx context.Context, symbol string) (time.Time, error)

	// Trade operations.
	SaveTrade(ctx context.Context, trade *TradeRecord) error
	GetOpenTrades(ctx context.Context) ([]TradeRecord, error)
	GetTradesByStrategy(ctx context.Context, strategyID string) ([]TradeRecord, error)
	CloseTrade(ctx context.Context, tradeID int64, exitPrice float64, exitReason string) error

	// Signal operations.
	SaveSignal(ctx context.Context, signal *SignalRecord) error
	GetSignalsByDate(ctx context.Context, date time.Time) ([]SignalRecord, error)

	// AI score operations.
	SaveAIScores(ctx context.Context, scores []AIScoreRecord) error
	GetAIScores(ctx context.Context, symbol string, date time.Time) (*AIScoreRecord, error)

	// Trade log operations.
	SaveTradeLog(ctx context.Context, log *TradeLog) error
	GetTradeLogs(ctx context.Context, from, to time.Time) ([]TradeLog, error)

	// Daily P&L.
	GetDailyPnL(ctx context.Context, date time.Time) (float64, error)

	// Health check.
	Ping(ctx context.Context) error
}
