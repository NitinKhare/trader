// Package market - dhan_data.go implements the DataProvider interface using Dhan's historical data API.
//
// This is intentionally separate from the broker layer (internal/broker/dhan.go).
// Market data fetching is a data concern, not an execution concern.
//
// Dhan API details:
//   - Endpoint: POST https://api.dhan.co/v2/charts/historical
//   - Auth: access-token header (Client-Id is optional)
//   - Rate limit: 10 req/sec
//   - Max 90 days per request (requires chunking)
//   - Response: arrays of open, high, low, close, volume, timestamp (epoch)
//   - Symbols: uses numeric securityId, mapped from ticker via dhan_instruments.json
package market

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

const (
	// dhanMaxChunkDays is the maximum number of days Dhan allows per historical data request.
	dhanMaxChunkDays = 90

	// dhanRateLimitInterval is the minimum time between API requests (10 req/sec).
	dhanRateLimitInterval = 110 * time.Millisecond
)

// DhanDataConfig holds configuration for the Dhan data provider.
type DhanDataConfig struct {
	ClientID       string `json:"client_id"`
	AccessToken    string `json:"access_token"`
	BaseURL        string `json:"base_url"`
	InstrumentFile string `json:"instrument_file"`
	MarketDataDir  string `json:"market_data_dir"`
}

// DhanDataProvider implements DataProvider using Dhan's historical data API.
type DhanDataProvider struct {
	config      DhanDataConfig
	client      *http.Client
	instruments map[string]string // ticker -> securityId
	rateMu      sync.Mutex
	lastRequest time.Time
}

// dhanChartRequest is the POST body for /v2/charts/historical.
type dhanChartRequest struct {
	SecurityID      string `json:"securityId"`
	ExchangeSegment string `json:"exchangeSegment"`
	Instrument      string `json:"instrument"`
	ExpiryCode      int    `json:"expiryCode"`
	FromDate        string `json:"fromDate"`
	ToDate          string `json:"toDate"`
}

// dhanChartResponse is the JSON response from Dhan historical API.
type dhanChartResponse struct {
	Open      []float64 `json:"open"`
	High      []float64 `json:"high"`
	Low       []float64 `json:"low"`
	Close     []float64 `json:"close"`
	Volume    []int64   `json:"volume"`
	Timestamp []int64   `json:"timestamp"`
}

// NewDhanDataProvider creates a new Dhan data provider.
func NewDhanDataProvider(cfg DhanDataConfig) (*DhanDataProvider, error) {
	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("dhan data: access_token is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.dhan.co"
	}

	dp := &DhanDataProvider{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}

	if cfg.InstrumentFile != "" {
		if err := dp.loadInstruments(cfg.InstrumentFile); err != nil {
			return nil, fmt.Errorf("dhan data: load instruments: %w", err)
		}
	}

	return dp, nil
}

// loadInstruments reads the ticker-to-securityId mapping from a JSON file.
func (d *DhanDataProvider) loadInstruments(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var file struct {
		Instruments map[string]string `json:"instruments"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	d.instruments = file.Instruments
	return nil
}

// resolveSecurityID maps a ticker symbol to a Dhan securityId.
func (d *DhanDataProvider) resolveSecurityID(symbol string) (string, error) {
	id, ok := d.instruments[symbol]
	if !ok {
		return "", fmt.Errorf("dhan data: no securityId for symbol %q", symbol)
	}
	return id, nil
}

// FetchDailyCandles implements DataProvider.
// Handles 90-day chunking automatically for any date range.
func (d *DhanDataProvider) FetchDailyCandles(ctx context.Context, symbol string, from, to time.Time) ([]strategy.Candle, error) {
	secID, err := d.resolveSecurityID(symbol)
	if err != nil {
		return nil, err
	}

	// Determine instrument type: INDEX for NIFTY50, EQUITY for stocks.
	instrument := "EQUITY"
	exchangeSegment := "NSE_EQ"
	if symbol == "NIFTY50" {
		instrument = "INDEX"
		exchangeSegment = "IDX_I"
	}

	var allCandles []strategy.Candle
	chunkStart := from

	for chunkStart.Before(to) || chunkStart.Equal(to) {
		chunkEnd := chunkStart.AddDate(0, 0, dhanMaxChunkDays-1)
		if chunkEnd.After(to) {
			chunkEnd = to
		}

		d.throttle()

		resp, err := d.fetchChunk(ctx, secID, exchangeSegment, instrument, chunkStart, chunkEnd)
		if err != nil {
			return allCandles, fmt.Errorf("fetch %s chunk [%s to %s]: %w",
				symbol, chunkStart.Format("2006-01-02"), chunkEnd.Format("2006-01-02"), err)
		}

		// Convert response arrays to candles.
		if resp != nil && len(resp.Timestamp) > 0 {
			for i := range resp.Timestamp {
				t := time.Unix(resp.Timestamp[i], 0).In(IST)
				allCandles = append(allCandles, strategy.Candle{
					Symbol: symbol,
					Date:   time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, IST),
					Open:   resp.Open[i],
					High:   resp.High[i],
					Low:    resp.Low[i],
					Close:  resp.Close[i],
					Volume: resp.Volume[i],
				})
			}
		}

		chunkStart = chunkEnd.AddDate(0, 0, 1)
		if chunkStart.After(to) {
			break
		}
	}

	return allCandles, nil
}

// FetchBulkDailyCandles implements DataProvider.
// Fetches each symbol sequentially with rate limiting.
func (d *DhanDataProvider) FetchBulkDailyCandles(ctx context.Context, symbols []string, from, to time.Time) (map[string][]strategy.Candle, error) {
	result := make(map[string][]strategy.Candle, len(symbols))

	for _, symbol := range symbols {
		candles, err := d.FetchDailyCandles(ctx, symbol, from, to)
		if err != nil {
			// Log and skip — don't abort the entire batch for one symbol.
			fmt.Printf("dhan data: WARN: failed to fetch %s: %v\n", symbol, err)
			continue
		}
		result[symbol] = candles
	}

	return result, nil
}

// fetchChunk makes a single API call for a <=90 day range.
func (d *DhanDataProvider) fetchChunk(
	ctx context.Context,
	securityID, exchangeSegment, instrument string,
	from, to time.Time,
) (*dhanChartResponse, error) {
	reqBody := dhanChartRequest{
		SecurityID:      securityID,
		ExchangeSegment: exchangeSegment,
		Instrument:      instrument,
		ExpiryCode:      0,
		FromDate:        from.Format("2006-01-02"),
		ToDate:          to.Format("2006-01-02"),
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := d.config.BaseURL + "/v2/charts/historical"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access-token", d.config.AccessToken)
	if d.config.ClientID != "" {
		req.Header.Set("Client-Id", d.config.ClientID)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed (401): check client_id and access_token — token may have expired")
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited (429): slow down requests")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var chartResp dhanChartResponse
	if err := json.Unmarshal(body, &chartResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &chartResp, nil
}

// throttle enforces the 10 req/sec rate limit.
func (d *DhanDataProvider) throttle() {
	d.rateMu.Lock()
	defer d.rateMu.Unlock()

	elapsed := time.Since(d.lastRequest)
	if elapsed < dhanRateLimitInterval {
		time.Sleep(dhanRateLimitInterval - elapsed)
	}
	d.lastRequest = time.Now()
}

// ExportCSV writes candles to a CSV file in market_data/ for Python consumption.
// If the file already exists, it merges new data with existing data (deduplicates by date).
func (d *DhanDataProvider) ExportCSV(symbol string, candles []strategy.Candle) error {
	if d.config.MarketDataDir == "" {
		return fmt.Errorf("dhan data: market_data_dir not configured")
	}

	if err := os.MkdirAll(d.config.MarketDataDir, 0755); err != nil {
		return fmt.Errorf("create market data dir: %w", err)
	}

	filePath := filepath.Join(d.config.MarketDataDir, symbol+".csv")

	// Merge with existing data if file exists.
	existing := loadExistingCSV(filePath)
	merged := mergeCandles(existing, candles)

	// Sort by date.
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Date.Before(merged[j].Date)
	})

	// Write CSV.
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create csv file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header matching Python's expected format.
	if err := w.Write([]string{"date", "open", "high", "low", "close", "volume"}); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	for _, c := range merged {
		record := []string{
			c.Date.Format("2006-01-02"),
			strconv.FormatFloat(c.Open, 'f', 2, 64),
			strconv.FormatFloat(c.High, 'f', 2, 64),
			strconv.FormatFloat(c.Low, 'f', 2, 64),
			strconv.FormatFloat(c.Close, 'f', 2, 64),
			strconv.FormatInt(c.Volume, 10),
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("write csv record: %w", err)
		}
	}

	return nil
}

// loadExistingCSV reads candles from an existing CSV file.
// Returns empty slice if file doesn't exist or has errors.
func loadExistingCSV(path string) []strategy.Candle {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil
	}

	var candles []strategy.Candle
	for i, record := range records {
		if i == 0 {
			continue // Skip header.
		}
		if len(record) < 6 {
			continue
		}

		date, err := time.Parse("2006-01-02", record[0])
		if err != nil {
			continue
		}
		open, _ := strconv.ParseFloat(record[1], 64)
		high, _ := strconv.ParseFloat(record[2], 64)
		low, _ := strconv.ParseFloat(record[3], 64)
		closeP, _ := strconv.ParseFloat(record[4], 64)
		volume, _ := strconv.ParseInt(record[5], 10, 64)

		candles = append(candles, strategy.Candle{
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closeP,
			Volume: volume,
		})
	}

	return candles
}

// mergeCandles combines two slices of candles, deduplicating by date.
// New candles take precedence over existing ones for the same date.
func mergeCandles(existing, newCandles []strategy.Candle) []strategy.Candle {
	byDate := make(map[string]strategy.Candle)

	for _, c := range existing {
		byDate[c.Date.Format("2006-01-02")] = c
	}
	for _, c := range newCandles {
		byDate[c.Date.Format("2006-01-02")] = c // New data overrides.
	}

	result := make([]strategy.Candle, 0, len(byDate))
	for _, c := range byDate {
		result = append(result, c)
	}
	return result
}
