// Package market - data.go handles market data ingestion and caching.
//
// Design rules (from spec):
//   - Market data ≠ broker API.
//   - No strategy uses live broker candles.
//   - All strategies use stored candles.
//   - Data is fetched daily (EOD) and cached locally.
package market

import (
	"context"
	"fmt"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// DataProvider is the interface for fetching market data.
// Implementations may use free APIs, paid data vendors, or file-based sources.
// This is intentionally separate from the broker interface.
type DataProvider interface {
	// FetchDailyCandles retrieves daily OHLCV data for a symbol within a date range.
	FetchDailyCandles(ctx context.Context, symbol string, from, to time.Time) ([]strategy.Candle, error)

	// FetchBulkDailyCandles retrieves daily OHLCV data for multiple symbols.
	FetchBulkDailyCandles(ctx context.Context, symbols []string, from, to time.Time) (map[string][]strategy.Candle, error)
}

// DataStore is the interface for persisting and retrieving cached market data.
type DataStore interface {
	// SaveCandles persists candle data to storage.
	SaveCandles(ctx context.Context, candles []strategy.Candle) error

	// GetCandles retrieves cached candle data from storage.
	GetCandles(ctx context.Context, symbol string, from, to time.Time) ([]strategy.Candle, error)

	// GetLatestCandleDate returns the most recent candle date for a symbol.
	GetLatestCandleDate(ctx context.Context, symbol string) (time.Time, error)
}

// DataManager coordinates data fetching, caching, and retrieval.
// It ensures strategies always get data from the local store, never directly from the provider.
type DataManager struct {
	provider DataProvider
	store    DataStore
}

// NewDataManager creates a new data manager.
func NewDataManager(provider DataProvider, store DataStore) *DataManager {
	return &DataManager{
		provider: provider,
		store:    store,
	}
}

// SyncCandles ensures local data is up to date for the given symbols.
// It fetches only missing data from the provider and stores it locally.
func (dm *DataManager) SyncCandles(ctx context.Context, symbols []string, upToDate time.Time) error {
	for _, symbol := range symbols {
		latest, err := dm.store.GetLatestCandleDate(ctx, symbol)
		if err != nil {
			// No data exists yet; fetch full history.
			latest = upToDate.AddDate(-1, 0, 0) // Default: 1 year of history.
		}

		// Only fetch if there's new data to get.
		if !latest.Before(upToDate) {
			continue
		}

		fetchFrom := latest.AddDate(0, 0, 1) // Day after last stored candle.
		candles, err := dm.provider.FetchDailyCandles(ctx, symbol, fetchFrom, upToDate)
		if err != nil {
			return fmt.Errorf("data manager: fetch %s: %w", symbol, err)
		}

		if len(candles) > 0 {
			if err := dm.store.SaveCandles(ctx, candles); err != nil {
				return fmt.Errorf("data manager: save %s: %w", symbol, err)
			}
		}
	}

	return nil
}

// GetCandles retrieves candle data from the local store.
// This is the only method strategies should use for market data.
func (dm *DataManager) GetCandles(ctx context.Context, symbol string, from, to time.Time) ([]strategy.Candle, error) {
	return dm.store.GetCandles(ctx, symbol, from, to)
}
