// Package storage - postgres.go provides the Postgres/TimescaleDB implementation.
//
// Uses database/sql with pgx driver for connection pooling, prepared
// statements, and full Postgres type support.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for database/sql

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// PostgresStore implements the Store interface using Postgres + TimescaleDB.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore opens a connection pool to Postgres and verifies connectivity.
func NewPostgresStore(connStr string) (*PostgresStore, error) {
	if connStr == "" {
		return nil, fmt.Errorf("postgres store: connection string is required")
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("postgres store: open: %w", err)
	}

	// Configure pool.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	// Verify connectivity.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("postgres store: ping: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

// Close closes the database connection pool.
func (ps *PostgresStore) Close() error {
	if ps.db != nil {
		return ps.db.Close()
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────
// Health
// ────────────────────────────────────────────────────────────────────

func (ps *PostgresStore) Ping(ctx context.Context) error {
	return ps.db.PingContext(ctx)
}

// ────────────────────────────────────────────────────────────────────
// Trade Log operations
// ────────────────────────────────────────────────────────────────────

func (ps *PostgresStore) SaveTradeLog(ctx context.Context, tl *TradeLog) error {
	_, err := ps.db.ExecContext(ctx,
		`INSERT INTO trade_logs (timestamp, strategy_id, symbol, action, reason_code, message, inputs_json)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		tl.Timestamp, tl.StrategyID, tl.Symbol, tl.Action, tl.ReasonCode, tl.Message, tl.InputsJSON,
	)
	if err != nil {
		return fmt.Errorf("save trade log: %w", err)
	}
	return nil
}

func (ps *PostgresStore) GetTradeLogs(ctx context.Context, from, to time.Time) ([]TradeLog, error) {
	rows, err := ps.db.QueryContext(ctx,
		`SELECT id, timestamp, strategy_id, symbol, action, reason_code, message, COALESCE(inputs_json::text, '')
		 FROM trade_logs
		 WHERE timestamp >= $1 AND timestamp <= $2
		 ORDER BY timestamp DESC`,
		from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("get trade logs: %w", err)
	}
	defer rows.Close()

	var logs []TradeLog
	for rows.Next() {
		var tl TradeLog
		if err := rows.Scan(&tl.ID, &tl.Timestamp, &tl.StrategyID, &tl.Symbol,
			&tl.Action, &tl.ReasonCode, &tl.Message, &tl.InputsJSON); err != nil {
			return nil, fmt.Errorf("scan trade log: %w", err)
		}
		logs = append(logs, tl)
	}
	return logs, rows.Err()
}

// ────────────────────────────────────────────────────────────────────
// Signal operations
// ────────────────────────────────────────────────────────────────────

func (ps *PostgresStore) SaveSignal(ctx context.Context, sig *SignalRecord) error {
	_, err := ps.db.ExecContext(ctx,
		`INSERT INTO signals (strategy_id, signal_id, symbol, action, price, stop_loss, target, quantity,
		                      reason, trend_strength_score, breakout_quality_score, volatility_score,
		                      risk_score, liquidity_score, approved, rejection_reason, signal_date)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		 ON CONFLICT (signal_id) DO NOTHING`,
		sig.StrategyID, sig.SignalID, sig.Symbol, sig.Action,
		sig.Price, sig.StopLoss, sig.Target, sig.Quantity,
		sig.Reason,
		sig.Scores.TrendStrengthScore, sig.Scores.BreakoutQualityScore,
		sig.Scores.VolatilityScore, sig.Scores.RiskScore, sig.Scores.LiquidityScore,
		sig.Approved, sig.RejectionReason, sig.Date,
	)
	if err != nil {
		return fmt.Errorf("save signal: %w", err)
	}
	return nil
}

func (ps *PostgresStore) GetSignalsByDate(ctx context.Context, date time.Time) ([]SignalRecord, error) {
	rows, err := ps.db.QueryContext(ctx,
		`SELECT id, strategy_id, signal_id, symbol, action, price, stop_loss, target, quantity,
		        reason, trend_strength_score, breakout_quality_score, volatility_score,
		        risk_score, liquidity_score, approved, COALESCE(rejection_reason, ''), signal_date, created_at
		 FROM signals WHERE signal_date = $1 ORDER BY created_at DESC`,
		date,
	)
	if err != nil {
		return nil, fmt.Errorf("get signals by date: %w", err)
	}
	defer rows.Close()

	var signals []SignalRecord
	for rows.Next() {
		var s SignalRecord
		if err := rows.Scan(&s.ID, &s.StrategyID, &s.SignalID, &s.Symbol, &s.Action,
			&s.Price, &s.StopLoss, &s.Target, &s.Quantity,
			&s.Reason,
			&s.Scores.TrendStrengthScore, &s.Scores.BreakoutQualityScore,
			&s.Scores.VolatilityScore, &s.Scores.RiskScore, &s.Scores.LiquidityScore,
			&s.Approved, &s.RejectionReason, &s.Date, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan signal: %w", err)
		}
		signals = append(signals, s)
	}
	return signals, rows.Err()
}

// ────────────────────────────────────────────────────────────────────
// Trade operations
// ────────────────────────────────────────────────────────────────────

func (ps *PostgresStore) SaveTrade(ctx context.Context, trade *TradeRecord) error {
	err := ps.db.QueryRowContext(ctx,
		`INSERT INTO trades (strategy_id, signal_id, symbol, side, quantity,
		                     entry_price, exit_price, stop_loss, target,
		                     order_id, sl_order_id,
		                     entry_time, exit_time, exit_reason, pnl, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		 RETURNING id`,
		trade.StrategyID, trade.SignalID, trade.Symbol, trade.Side, trade.Quantity,
		trade.EntryPrice, nullFloat(trade.ExitPrice), trade.StopLoss, nullFloat(trade.Target),
		nullString(trade.OrderID), nullString(trade.SLOrderID),
		trade.EntryTime, nullTime(trade.ExitTime), nullString(trade.ExitReason),
		nullFloat(trade.PnL), trade.Status,
	).Scan(&trade.ID)
	if err != nil {
		return fmt.Errorf("save trade: %w", err)
	}
	return nil
}

func (ps *PostgresStore) GetOpenTrades(ctx context.Context) ([]TradeRecord, error) {
	rows, err := ps.db.QueryContext(ctx,
		`SELECT id, strategy_id, signal_id, symbol, side, quantity,
		        entry_price, COALESCE(exit_price, 0), stop_loss, COALESCE(target, 0),
		        COALESCE(order_id, ''), COALESCE(sl_order_id, ''),
		        entry_time, exit_time, COALESCE(exit_reason, ''), COALESCE(pnl, 0), status, created_at
		 FROM trades WHERE status = 'open' ORDER BY entry_time DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("get open trades: %w", err)
	}
	defer rows.Close()

	return scanTrades(rows)
}

func (ps *PostgresStore) GetTradesByStrategy(ctx context.Context, strategyID string) ([]TradeRecord, error) {
	rows, err := ps.db.QueryContext(ctx,
		`SELECT id, strategy_id, signal_id, symbol, side, quantity,
		        entry_price, COALESCE(exit_price, 0), stop_loss, COALESCE(target, 0),
		        COALESCE(order_id, ''), COALESCE(sl_order_id, ''),
		        entry_time, exit_time, COALESCE(exit_reason, ''), COALESCE(pnl, 0), status, created_at
		 FROM trades WHERE strategy_id = $1 ORDER BY entry_time DESC`,
		strategyID,
	)
	if err != nil {
		return nil, fmt.Errorf("get trades by strategy: %w", err)
	}
	defer rows.Close()

	return scanTrades(rows)
}

func (ps *PostgresStore) UpdateTradeSLOrderID(ctx context.Context, tradeID int64, slOrderID string) error {
	_, err := ps.db.ExecContext(ctx,
		`UPDATE trades SET sl_order_id = $1 WHERE id = $2`,
		slOrderID, tradeID,
	)
	if err != nil {
		return fmt.Errorf("update trade sl_order_id: %w", err)
	}
	return nil
}

func (ps *PostgresStore) UpdateTradeStopLoss(ctx context.Context, tradeID int64, newStopLoss float64, newSLOrderID string) error {
	_, err := ps.db.ExecContext(ctx,
		`UPDATE trades SET stop_loss = $1, sl_order_id = $2 WHERE id = $3`,
		newStopLoss, newSLOrderID, tradeID,
	)
	if err != nil {
		return fmt.Errorf("update trade stop_loss: %w", err)
	}
	return nil
}

func (ps *PostgresStore) CloseTrade(ctx context.Context, tradeID int64, exitPrice float64, exitReason string) error {
	now := time.Now()
	_, err := ps.db.ExecContext(ctx,
		`UPDATE trades SET
			exit_price = $1,
			exit_time = $2,
			exit_reason = $3,
			exit_fill_price = $1,
			exit_fill_time = $2,
			exit_order_status = 'COMPLETED',
			position_state = 'EXIT_FILLED',
			pnl = (CASE WHEN side = 'BUY' THEN
				(COALESCE(entry_fill_price, entry_price) - $1) * quantity
			              ELSE
				($1 - COALESCE(entry_fill_price, entry_price)) * quantity
			         END),
			status = 'closed'
		 WHERE id = $4`,
		exitPrice, now, exitReason, tradeID,
	)
	if err != nil {
		return fmt.Errorf("close trade: %w", err)
	}
	return nil
}

// MarkEntryFilled records when an entry order actually fills (from broker confirmation).
// This marks the trade as truly opened with actual fill price and time.
func (ps *PostgresStore) MarkEntryFilled(ctx context.Context, tradeID int64, fillPrice float64) error {
	now := time.Now()
	_, err := ps.db.ExecContext(ctx,
		`UPDATE trades SET
			entry_order_status = 'COMPLETED',
			entry_fill_price = $1,
			entry_fill_time = $2,
			position_state = 'ENTRY_FILLED'
		 WHERE id = $3`,
		fillPrice, now, tradeID,
	)
	if err != nil {
		return fmt.Errorf("mark entry filled: %w", err)
	}
	return nil
}

// MarkExitOrderPlaced records when an exit order is placed (waiting for fill).
func (ps *PostgresStore) MarkExitOrderPlaced(ctx context.Context, tradeID int64, exitOrderID string, targetPrice float64) error {
	_, err := ps.db.ExecContext(ctx,
		`UPDATE trades SET
			exit_order_id = $1,
			exit_price = $2,
			exit_order_status = 'PENDING',
			position_state = 'EXIT_PENDING'
		 WHERE id = $3`,
		exitOrderID, targetPrice, tradeID,
	)
	if err != nil {
		return fmt.Errorf("mark exit order placed: %w", err)
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────
// Candle operations
// ────────────────────────────────────────────────────────────────────

func (ps *PostgresStore) SaveCandles(ctx context.Context, candles []strategy.Candle) error {
	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("save candles begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO candles (symbol, date, open, high, low, close, volume)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (symbol, date) DO UPDATE SET
		   open = EXCLUDED.open, high = EXCLUDED.high, low = EXCLUDED.low,
		   close = EXCLUDED.close, volume = EXCLUDED.volume`)
	if err != nil {
		return fmt.Errorf("save candles prepare: %w", err)
	}
	defer stmt.Close()

	for _, c := range candles {
		if _, err := stmt.ExecContext(ctx, c.Symbol, c.Date, c.Open, c.High, c.Low, c.Close, c.Volume); err != nil {
			return fmt.Errorf("save candle %s %s: %w", c.Symbol, c.Date.Format("2006-01-02"), err)
		}
	}

	return tx.Commit()
}

func (ps *PostgresStore) GetCandles(ctx context.Context, symbol string, from, to time.Time) ([]strategy.Candle, error) {
	rows, err := ps.db.QueryContext(ctx,
		`SELECT symbol, date, open, high, low, close, volume
		 FROM candles WHERE symbol = $1 AND date >= $2 AND date <= $3
		 ORDER BY date ASC`,
		symbol, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("get candles: %w", err)
	}
	defer rows.Close()

	var candles []strategy.Candle
	for rows.Next() {
		var c strategy.Candle
		if err := rows.Scan(&c.Symbol, &c.Date, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume); err != nil {
			return nil, fmt.Errorf("scan candle: %w", err)
		}
		candles = append(candles, c)
	}
	return candles, rows.Err()
}

func (ps *PostgresStore) GetLatestCandleDate(ctx context.Context, symbol string) (time.Time, error) {
	var date time.Time
	err := ps.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(date), '1970-01-01') FROM candles WHERE symbol = $1`, symbol,
	).Scan(&date)
	if err != nil {
		return time.Time{}, fmt.Errorf("get latest candle date: %w", err)
	}
	return date, nil
}

// ────────────────────────────────────────────────────────────────────
// AI Score operations
// ────────────────────────────────────────────────────────────────────

func (ps *PostgresStore) SaveAIScores(ctx context.Context, scores []AIScoreRecord) error {
	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("save ai scores begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO ai_scores (symbol, score_date, trend_strength_score, breakout_quality_score,
		                        volatility_score, risk_score, liquidity_score)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (symbol, score_date) DO UPDATE SET
		   trend_strength_score = EXCLUDED.trend_strength_score,
		   breakout_quality_score = EXCLUDED.breakout_quality_score,
		   volatility_score = EXCLUDED.volatility_score,
		   risk_score = EXCLUDED.risk_score,
		   liquidity_score = EXCLUDED.liquidity_score`)
	if err != nil {
		return fmt.Errorf("save ai scores prepare: %w", err)
	}
	defer stmt.Close()

	for _, s := range scores {
		if _, err := stmt.ExecContext(ctx, s.Symbol, s.Date,
			s.TrendStrengthScore, s.BreakoutQualityScore,
			s.VolatilityScore, s.RiskScore, s.LiquidityScore); err != nil {
			return fmt.Errorf("save ai score %s: %w", s.Symbol, err)
		}
	}

	return tx.Commit()
}

func (ps *PostgresStore) GetAIScores(ctx context.Context, symbol string, date time.Time) (*AIScoreRecord, error) {
	var s AIScoreRecord
	err := ps.db.QueryRowContext(ctx,
		`SELECT id, symbol, score_date, trend_strength_score, breakout_quality_score,
		        volatility_score, risk_score, liquidity_score, created_at
		 FROM ai_scores WHERE symbol = $1 AND score_date = $2`,
		symbol, date,
	).Scan(&s.ID, &s.Symbol, &s.Date,
		&s.TrendStrengthScore, &s.BreakoutQualityScore,
		&s.VolatilityScore, &s.RiskScore, &s.LiquidityScore, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get ai scores: %w", err)
	}
	return &s, nil
}

// ────────────────────────────────────────────────────────────────────
// Analytics queries
// ────────────────────────────────────────────────────────────────────

func (ps *PostgresStore) GetAllClosedTrades(ctx context.Context) ([]TradeRecord, error) {
	rows, err := ps.db.QueryContext(ctx,
		`SELECT id, strategy_id, signal_id, symbol, side, quantity,
		        entry_price, COALESCE(exit_price, 0), stop_loss, COALESCE(target, 0),
		        COALESCE(order_id, ''), COALESCE(sl_order_id, ''),
		        entry_time, exit_time, COALESCE(exit_reason, ''), COALESCE(pnl, 0), status, created_at
		 FROM trades WHERE status = 'closed' ORDER BY exit_time ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("get all closed trades: %w", err)
	}
	defer rows.Close()

	return scanTrades(rows)
}

func (ps *PostgresStore) GetClosedTradesByDateRange(ctx context.Context, from, to time.Time) ([]TradeRecord, error) {
	rows, err := ps.db.QueryContext(ctx,
		`SELECT id, strategy_id, signal_id, symbol, side, quantity,
		        entry_price, COALESCE(exit_price, 0), stop_loss, COALESCE(target, 0),
		        COALESCE(order_id, ''), COALESCE(sl_order_id, ''),
		        entry_time, exit_time, COALESCE(exit_reason, ''), COALESCE(pnl, 0), status, created_at
		 FROM trades WHERE status = 'closed' AND exit_time >= $1 AND exit_time <= $2
		 ORDER BY exit_time ASC`,
		from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("get closed trades by date range: %w", err)
	}
	defer rows.Close()

	return scanTrades(rows)
}

// ────────────────────────────────────────────────────────────────────
// Daily P&L
// ────────────────────────────────────────────────────────────────────

func (ps *PostgresStore) GetDailyPnL(ctx context.Context, date time.Time) (float64, error) {
	var pnl float64
	err := ps.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(pnl), 0) FROM trades
		 WHERE status = 'closed' AND DATE(exit_time) = $1`,
		date,
	).Scan(&pnl)
	if err != nil {
		return 0, fmt.Errorf("get daily pnl: %w", err)
	}
	return pnl, nil
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

func scanTrades(rows *sql.Rows) ([]TradeRecord, error) {
	var trades []TradeRecord
	for rows.Next() {
		var t TradeRecord
		var exitTime sql.NullTime
		if err := rows.Scan(&t.ID, &t.StrategyID, &t.SignalID, &t.Symbol, &t.Side, &t.Quantity,
			&t.EntryPrice, &t.ExitPrice, &t.StopLoss, &t.Target,
			&t.OrderID, &t.SLOrderID,
			&t.EntryTime, &exitTime, &t.ExitReason, &t.PnL, &t.Status, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan trade: %w", err)
		}
		if exitTime.Valid {
			t.ExitTime = &exitTime.Time
		}
		trades = append(trades, t)
	}
	return trades, rows.Err()
}

func nullFloat(f float64) sql.NullFloat64 {
	if f == 0 {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: f, Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// LogInputs serializes a trade intent and score into JSON for the
// trade_logs.inputs_json column.
func LogInputs(intent strategy.TradeIntent, regime string) string {
	inputs := map[string]interface{}{
		"symbol":    intent.Symbol,
		"action":    string(intent.Action),
		"price":     intent.Price,
		"stop_loss": intent.StopLoss,
		"target":    intent.Target,
		"quantity":  intent.Quantity,
		"regime":    regime,
		"scores": map[string]float64{
			"trend":    intent.Scores.TrendStrengthScore,
			"breakout": intent.Scores.BreakoutQualityScore,
			"vol":      intent.Scores.VolatilityScore,
			"risk":     intent.Scores.RiskScore,
			"liq":      intent.Scores.LiquidityScore,
		},
	}
	b, _ := json.Marshal(inputs)
	return string(b)
}
