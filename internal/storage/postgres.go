// Package storage - postgres.go provides the Postgres/TimescaleDB implementation stub.
//
// This file defines the concrete storage implementation.
// Full SQL queries will be added when the database schema is finalized.
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// PostgresStore implements the Store interface using Postgres + TimescaleDB.
type PostgresStore struct {
	connStr string
	// db *sql.DB — will be added when pgx or database/sql is integrated.
}

// NewPostgresStore creates a new Postgres store.
// TODO: Add actual database connection using pgx.
func NewPostgresStore(connStr string) (*PostgresStore, error) {
	if connStr == "" {
		return nil, fmt.Errorf("postgres store: connection string is required")
	}
	return &PostgresStore{connStr: connStr}, nil
}

func (ps *PostgresStore) SaveCandles(_ context.Context, _ []strategy.Candle) error {
	return fmt.Errorf("postgres store: SaveCandles not yet implemented")
}

func (ps *PostgresStore) GetCandles(_ context.Context, _ string, _, _ time.Time) ([]strategy.Candle, error) {
	return nil, fmt.Errorf("postgres store: GetCandles not yet implemented")
}

func (ps *PostgresStore) GetLatestCandleDate(_ context.Context, _ string) (time.Time, error) {
	return time.Time{}, fmt.Errorf("postgres store: GetLatestCandleDate not yet implemented")
}

func (ps *PostgresStore) SaveTrade(_ context.Context, _ *TradeRecord) error {
	return fmt.Errorf("postgres store: SaveTrade not yet implemented")
}

func (ps *PostgresStore) GetOpenTrades(_ context.Context) ([]TradeRecord, error) {
	return nil, fmt.Errorf("postgres store: GetOpenTrades not yet implemented")
}

func (ps *PostgresStore) GetTradesByStrategy(_ context.Context, _ string) ([]TradeRecord, error) {
	return nil, fmt.Errorf("postgres store: GetTradesByStrategy not yet implemented")
}

func (ps *PostgresStore) CloseTrade(_ context.Context, _ int64, _ float64, _ string) error {
	return fmt.Errorf("postgres store: CloseTrade not yet implemented")
}

func (ps *PostgresStore) SaveSignal(_ context.Context, _ *SignalRecord) error {
	return fmt.Errorf("postgres store: SaveSignal not yet implemented")
}

func (ps *PostgresStore) GetSignalsByDate(_ context.Context, _ time.Time) ([]SignalRecord, error) {
	return nil, fmt.Errorf("postgres store: GetSignalsByDate not yet implemented")
}

func (ps *PostgresStore) SaveAIScores(_ context.Context, _ []AIScoreRecord) error {
	return fmt.Errorf("postgres store: SaveAIScores not yet implemented")
}

func (ps *PostgresStore) GetAIScores(_ context.Context, _ string, _ time.Time) (*AIScoreRecord, error) {
	return nil, fmt.Errorf("postgres store: GetAIScores not yet implemented")
}

func (ps *PostgresStore) SaveTradeLog(_ context.Context, _ *TradeLog) error {
	return fmt.Errorf("postgres store: SaveTradeLog not yet implemented")
}

func (ps *PostgresStore) GetTradeLogs(_ context.Context, _, _ time.Time) ([]TradeLog, error) {
	return nil, fmt.Errorf("postgres store: GetTradeLogs not yet implemented")
}

func (ps *PostgresStore) GetDailyPnL(_ context.Context, _ time.Time) (float64, error) {
	return 0, fmt.Errorf("postgres store: GetDailyPnL not yet implemented")
}

func (ps *PostgresStore) Ping(_ context.Context) error {
	return fmt.Errorf("postgres store: Ping not yet implemented")
}
