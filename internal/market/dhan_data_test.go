package market

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// makeMockDhanServer creates a test HTTP server that mimics the Dhan historical API.
func makeMockDhanServer(t *testing.T, response dhanChartResponse, statusCode int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify correct headers (only access-token is required; Client-Id is optional).
		if r.Header.Get("access-token") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"missing access-token"}`))
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Verify request body.
		var req dhanChartRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	}))
}

// makeTestDhanProvider creates a DhanDataProvider pointing at a mock server.
func makeTestDhanProvider(t *testing.T, serverURL string, tmpDir string) *DhanDataProvider {
	t.Helper()

	// Write a test instruments file.
	instruments := map[string]interface{}{
		"instruments": map[string]string{
			"RELIANCE": "2885",
			"TCS":      "11536",
			"NIFTY50":  "13",
		},
	}
	instData, _ := json.Marshal(instruments)
	instFile := filepath.Join(tmpDir, "instruments.json")
	os.WriteFile(instFile, instData, 0644)

	dp, err := NewDhanDataProvider(DhanDataConfig{
		ClientID:       "test-client",
		AccessToken:    "test-token",
		BaseURL:        serverURL,
		InstrumentFile: instFile,
		MarketDataDir:  filepath.Join(tmpDir, "market_data"),
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	return dp
}

func TestDhanData_ResolveSecurityID(t *testing.T) {
	tmpDir := t.TempDir()
	dp := makeTestDhanProvider(t, "http://localhost", tmpDir)

	id, err := dp.resolveSecurityID("RELIANCE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "2885" {
		t.Errorf("expected 2885, got %s", id)
	}

	_, err = dp.resolveSecurityID("UNKNOWN")
	if err == nil {
		t.Error("expected error for unknown symbol")
	}
}

func TestDhanData_FetchDailyCandles_SingleChunk(t *testing.T) {
	// 30 days of data — fits in a single chunk.
	now := time.Date(2026, 2, 8, 0, 0, 0, 0, IST)
	timestamps := make([]int64, 20)
	opens := make([]float64, 20)
	highs := make([]float64, 20)
	lows := make([]float64, 20)
	closes := make([]float64, 20)
	volumes := make([]int64, 20)

	for i := 0; i < 20; i++ {
		d := now.AddDate(0, 0, -20+i)
		timestamps[i] = d.Unix()
		opens[i] = 2500 + float64(i)
		highs[i] = 2510 + float64(i)
		lows[i] = 2490 + float64(i)
		closes[i] = 2505 + float64(i)
		volumes[i] = 1000000 + int64(i*10000)
	}

	mockResp := dhanChartResponse{
		Open: opens, High: highs, Low: lows, Close: closes,
		Volume: volumes, Timestamp: timestamps,
	}

	server := makeMockDhanServer(t, mockResp, http.StatusOK)
	defer server.Close()

	tmpDir := t.TempDir()
	dp := makeTestDhanProvider(t, server.URL, tmpDir)

	from := now.AddDate(0, 0, -30)
	candles, err := dp.FetchDailyCandles(context.Background(), "RELIANCE", from, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candles) != 20 {
		t.Errorf("expected 20 candles, got %d", len(candles))
	}
	if candles[0].Symbol != "RELIANCE" {
		t.Errorf("expected symbol RELIANCE, got %s", candles[0].Symbol)
	}
}

func TestDhanData_FetchDailyCandles_MultipleChunks(t *testing.T) {
	// Track how many API calls are made.
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Return 5 candles per chunk.
		now := time.Now()
		resp := dhanChartResponse{
			Open:      []float64{100, 101, 102, 103, 104},
			High:      []float64{105, 106, 107, 108, 109},
			Low:       []float64{95, 96, 97, 98, 99},
			Close:     []float64{102, 103, 104, 105, 106},
			Volume:    []int64{10000, 10001, 10002, 10003, 10004},
			Timestamp: []int64{now.Unix(), now.Unix() + 86400, now.Unix() + 86400*2, now.Unix() + 86400*3, now.Unix() + 86400*4},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	dp := makeTestDhanProvider(t, server.URL, tmpDir)

	// Request 180 days — should result in multiple chunks.
	from := time.Date(2025, 8, 1, 0, 0, 0, 0, IST)
	to := time.Date(2026, 1, 28, 0, 0, 0, 0, IST)

	_, err := dp.FetchDailyCandles(context.Background(), "RELIANCE", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must make more than 1 call for 180 days (each chunk is max 90 days).
	if callCount < 2 {
		t.Errorf("expected at least 2 API calls for 180-day range, got %d", callCount)
	}
}

func TestDhanData_FetchDailyCandles_365Days(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := dhanChartResponse{
			Open: []float64{100}, High: []float64{105}, Low: []float64{95},
			Close: []float64{102}, Volume: []int64{10000},
			Timestamp: []int64{time.Now().Unix()},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	dp := makeTestDhanProvider(t, server.URL, tmpDir)

	from := time.Date(2025, 2, 8, 0, 0, 0, 0, IST)
	to := time.Date(2026, 2, 8, 0, 0, 0, 0, IST)

	_, err := dp.FetchDailyCandles(context.Background(), "TCS", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 365 days / 90 days per chunk = 5 chunks (ceil).
	expectedChunks := 5
	if callCount != expectedChunks {
		t.Errorf("expected %d API calls for 365-day range, got %d", expectedChunks, callCount)
	}
}

func TestDhanData_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid token"}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	// Create provider that sends auth headers (mock server ignores them for this test).
	instruments := map[string]interface{}{
		"instruments": map[string]string{"RELIANCE": "2885"},
	}
	instData, _ := json.Marshal(instruments)
	instFile := filepath.Join(tmpDir, "instruments.json")
	os.WriteFile(instFile, instData, 0644)

	dp, _ := NewDhanDataProvider(DhanDataConfig{
		ClientID:       "bad-client",
		AccessToken:    "bad-token",
		BaseURL:        server.URL,
		InstrumentFile: instFile,
	})

	_, err := dp.FetchDailyCandles(context.Background(), "RELIANCE",
		time.Now().AddDate(0, 0, -10), time.Now())

	if err == nil {
		t.Error("expected error for 401 response")
	}
}

func TestDhanData_EmptyResponse(t *testing.T) {
	server := makeMockDhanServer(t, dhanChartResponse{}, http.StatusOK)
	defer server.Close()

	tmpDir := t.TempDir()
	dp := makeTestDhanProvider(t, server.URL, tmpDir)

	candles, err := dp.FetchDailyCandles(context.Background(), "RELIANCE",
		time.Now().AddDate(0, 0, -5), time.Now())

	if err != nil {
		t.Fatalf("empty response should not be an error: %v", err)
	}
	if len(candles) != 0 {
		t.Errorf("expected 0 candles for empty response, got %d", len(candles))
	}
}

func TestDhanData_ExportCSV(t *testing.T) {
	server := makeMockDhanServer(t, dhanChartResponse{}, http.StatusOK)
	defer server.Close()

	tmpDir := t.TempDir()
	dp := makeTestDhanProvider(t, server.URL, tmpDir)

	candles := []strategy.Candle{
		{Symbol: "TEST", Date: time.Date(2026, 1, 5, 0, 0, 0, 0, IST), Open: 100, High: 110, Low: 95, Close: 105, Volume: 50000},
		{Symbol: "TEST", Date: time.Date(2026, 1, 6, 0, 0, 0, 0, IST), Open: 105, High: 115, Low: 100, Close: 112, Volume: 60000},
		{Symbol: "TEST", Date: time.Date(2026, 1, 7, 0, 0, 0, 0, IST), Open: 112, High: 118, Low: 108, Close: 115, Volume: 55000},
	}

	err := dp.ExportCSV("TEST", candles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file exists and has correct format.
	csvPath := filepath.Join(tmpDir, "market_data", "TEST.csv")
	data, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("csv file not created: %v", err)
	}

	content := string(data)
	// Check header.
	if content[:len("date,open,high,low,close,volume")] != "date,open,high,low,close,volume" {
		t.Errorf("unexpected CSV header: %s", content[:50])
	}
	// Should have header + 3 data rows (csv.Writer adds trailing newline).
	lines := 0
	for _, c := range content {
		if c == '\n' {
			lines++
		}
	}
	if lines < 3 {
		t.Errorf("expected at least 3 newlines (header + 3 rows), got %d", lines)
	}
}

func TestDhanData_ExportCSV_MergesExistingData(t *testing.T) {
	server := makeMockDhanServer(t, dhanChartResponse{}, http.StatusOK)
	defer server.Close()

	tmpDir := t.TempDir()
	dp := makeTestDhanProvider(t, server.URL, tmpDir)

	// Write initial data.
	initial := []strategy.Candle{
		{Symbol: "TEST", Date: time.Date(2026, 1, 5, 0, 0, 0, 0, IST), Open: 100, High: 110, Low: 95, Close: 105, Volume: 50000},
		{Symbol: "TEST", Date: time.Date(2026, 1, 6, 0, 0, 0, 0, IST), Open: 105, High: 115, Low: 100, Close: 112, Volume: 60000},
	}
	dp.ExportCSV("TEST", initial)

	// Write new data with one overlapping date and one new date.
	newData := []strategy.Candle{
		{Symbol: "TEST", Date: time.Date(2026, 1, 6, 0, 0, 0, 0, IST), Open: 106, High: 116, Low: 101, Close: 113, Volume: 65000}, // Updated.
		{Symbol: "TEST", Date: time.Date(2026, 1, 7, 0, 0, 0, 0, IST), Open: 113, High: 120, Low: 110, Close: 118, Volume: 70000}, // New.
	}
	err := dp.ExportCSV("TEST", newData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read back and verify: should have 3 unique dates.
	csvPath := filepath.Join(tmpDir, "market_data", "TEST.csv")
	candles := LoadExistingCSV(csvPath)
	if len(candles) != 3 {
		t.Errorf("expected 3 merged candles, got %d", len(candles))
	}
}

func TestDhanData_MissingCredentials(t *testing.T) {
	// Only access_token is required; client_id is optional.
	_, err := NewDhanDataProvider(DhanDataConfig{
		ClientID:    "optional",
		AccessToken: "",
	})
	if err == nil {
		t.Error("expected error for missing access_token")
	}

	// Should succeed with just access_token (no client_id).
	instruments := map[string]interface{}{
		"instruments": map[string]string{"RELIANCE": "2885"},
	}
	instData, _ := json.Marshal(instruments)
	tmpDir := t.TempDir()
	instFile := filepath.Join(tmpDir, "instruments.json")
	os.WriteFile(instFile, instData, 0644)

	dp, err := NewDhanDataProvider(DhanDataConfig{
		ClientID:       "",
		AccessToken:    "some-token",
		InstrumentFile: instFile,
	})
	if err != nil {
		t.Errorf("should succeed with only access_token: %v", err)
	}
	if dp == nil {
		t.Error("provider should not be nil")
	}
}

func TestDhanData_NIFTY50UsesIndexSegment(t *testing.T) {
	var receivedReq dhanChartRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)
		resp := dhanChartResponse{
			Open: []float64{22000}, High: []float64{22100}, Low: []float64{21900},
			Close: []float64{22050}, Volume: []int64{0},
			Timestamp: []int64{time.Now().Unix()},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	dp := makeTestDhanProvider(t, server.URL, tmpDir)

	dp.FetchDailyCandles(context.Background(), "NIFTY50",
		time.Now().AddDate(0, 0, -5), time.Now())

	if receivedReq.ExchangeSegment != "IDX_I" {
		t.Errorf("expected IDX_I for NIFTY50, got %s", receivedReq.ExchangeSegment)
	}
	if receivedReq.Instrument != "INDEX" {
		t.Errorf("expected INDEX for NIFTY50, got %s", receivedReq.Instrument)
	}
}
