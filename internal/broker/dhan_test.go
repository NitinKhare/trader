package broker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// makeTestDhanBroker creates a DhanBroker pointing at a mock server.
func makeTestDhanBroker(t *testing.T, serverURL string) *DhanBroker {
	t.Helper()

	tmpDir := t.TempDir()

	// Write a test instruments file.
	instruments := map[string]interface{}{
		"instruments": map[string]string{
			"RELIANCE": "2885",
			"TCS":      "11536",
			"INFY":     "1594",
		},
	}
	instData, _ := json.Marshal(instruments)
	instFile := filepath.Join(tmpDir, "instruments.json")
	os.WriteFile(instFile, instData, 0644)

	cfgJSON, _ := json.Marshal(DhanConfig{
		ClientID:       "test-client",
		AccessToken:    "test-token",
		BaseURL:        serverURL,
		InstrumentFile: instFile,
	})

	b, err := NewDhanBroker(cfgJSON)
	if err != nil {
		t.Fatalf("failed to create dhan broker: %v", err)
	}
	return b.(*DhanBroker)
}

func TestDhanBroker_PlaceOrder_Market(t *testing.T) {
	var receivedReq dhanPlaceOrderReq
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v2/orders" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("access-token") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		json.NewDecoder(r.Body).Decode(&receivedReq)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dhanPlaceOrderResp{
			OrderID:     "ORD-12345",
			OrderStatus: "PENDING",
		})
	}))
	defer server.Close()

	b := makeTestDhanBroker(t, server.URL)

	resp, err := b.PlaceOrder(context.Background(), Order{
		Symbol:   "RELIANCE",
		Exchange: "NSE",
		Side:     OrderSideBuy,
		Type:     OrderTypeMarket,
		Quantity: 10,
		Product:  "CNC",
		Tag:      "trend-follow-1",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.OrderID != "ORD-12345" {
		t.Errorf("expected order ID ORD-12345, got %s", resp.OrderID)
	}
	if resp.Status != OrderStatusPending {
		t.Errorf("expected PENDING status, got %s", resp.Status)
	}

	// Verify the API request was correctly formed.
	if receivedReq.SecurityID != "2885" {
		t.Errorf("expected securityId 2885, got %s", receivedReq.SecurityID)
	}
	if receivedReq.TransactionType != "BUY" {
		t.Errorf("expected BUY, got %s", receivedReq.TransactionType)
	}
	if receivedReq.OrderType != "MARKET" {
		t.Errorf("expected MARKET, got %s", receivedReq.OrderType)
	}
	if receivedReq.ExchangeSegment != "NSE_EQ" {
		t.Errorf("expected NSE_EQ, got %s", receivedReq.ExchangeSegment)
	}
	if receivedReq.ProductType != "CNC" {
		t.Errorf("expected CNC, got %s", receivedReq.ProductType)
	}
	if receivedReq.Quantity != 10 {
		t.Errorf("expected quantity 10, got %d", receivedReq.Quantity)
	}
	if receivedReq.CorrelationID != "trend-follow-1" {
		t.Errorf("expected correlationId trend-follow-1, got %s", receivedReq.CorrelationID)
	}
}

func TestDhanBroker_PlaceOrder_Limit(t *testing.T) {
	var receivedReq dhanPlaceOrderReq
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dhanPlaceOrderResp{
			OrderID:     "ORD-22222",
			OrderStatus: "PENDING",
		})
	}))
	defer server.Close()

	b := makeTestDhanBroker(t, server.URL)

	resp, err := b.PlaceOrder(context.Background(), Order{
		Symbol:   "TCS",
		Exchange: "NSE",
		Side:     OrderSideBuy,
		Type:     OrderTypeLimit,
		Quantity: 5,
		Price:    4200.50,
		Product:  "CNC",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.OrderID != "ORD-22222" {
		t.Errorf("expected ORD-22222, got %s", resp.OrderID)
	}
	if receivedReq.OrderType != "LIMIT" {
		t.Errorf("expected LIMIT, got %s", receivedReq.OrderType)
	}
	if receivedReq.Price != 4200.50 {
		t.Errorf("expected price 4200.50, got %f", receivedReq.Price)
	}
	if receivedReq.SecurityID != "11536" {
		t.Errorf("expected securityId 11536, got %s", receivedReq.SecurityID)
	}
}

func TestDhanBroker_PlaceOrder_StopLoss(t *testing.T) {
	var receivedReq dhanPlaceOrderReq
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dhanPlaceOrderResp{
			OrderID:     "ORD-33333",
			OrderStatus: "PENDING",
		})
	}))
	defer server.Close()

	b := makeTestDhanBroker(t, server.URL)

	_, err := b.PlaceOrder(context.Background(), Order{
		Symbol:       "INFY",
		Exchange:     "NSE",
		Side:         OrderSideSell,
		Type:         OrderTypeSL,
		Quantity:     15,
		Price:        1800.00,
		TriggerPrice: 1810.00,
		Product:      "CNC",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedReq.OrderType != "STOP_LOSS" {
		t.Errorf("expected STOP_LOSS, got %s", receivedReq.OrderType)
	}
	if receivedReq.TransactionType != "SELL" {
		t.Errorf("expected SELL, got %s", receivedReq.TransactionType)
	}
	if receivedReq.TriggerPrice != 1810.00 {
		t.Errorf("expected trigger price 1810, got %f", receivedReq.TriggerPrice)
	}
	if receivedReq.Price != 1800.00 {
		t.Errorf("expected price 1800, got %f", receivedReq.Price)
	}
}

func TestDhanBroker_GetOrderStatus_Traded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v2/orders/ORD-99999" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dhanOrderDetailResp{
			OrderID:            "ORD-99999",
			OrderStatus:        "TRADED",
			FilledQty:          10,
			RemainingQuantity:  0,
			AverageTradedPrice: 2500.75,
			Quantity:           10,
		})
	}))
	defer server.Close()

	b := makeTestDhanBroker(t, server.URL)

	status, err := b.GetOrderStatus(context.Background(), "ORD-99999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != OrderStatusCompleted {
		t.Errorf("expected COMPLETED, got %s", status.Status)
	}
	if status.FilledQty != 10 {
		t.Errorf("expected filledQty 10, got %d", status.FilledQty)
	}
	if status.AveragePrice != 2500.75 {
		t.Errorf("expected avgPrice 2500.75, got %f", status.AveragePrice)
	}
	if status.PendingQty != 0 {
		t.Errorf("expected pendingQty 0, got %d", status.PendingQty)
	}
}

func TestDhanBroker_GetOrderStatus_Rejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dhanOrderDetailResp{
			OrderID:      "ORD-88888",
			OrderStatus:  "REJECTED",
			OmsErrorCode: "16388",
			OmsErrorDesc: "Insufficient balance",
		})
	}))
	defer server.Close()

	b := makeTestDhanBroker(t, server.URL)

	status, err := b.GetOrderStatus(context.Background(), "ORD-88888")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != OrderStatusRejected {
		t.Errorf("expected REJECTED, got %s", status.Status)
	}
	if status.Message == "" {
		t.Error("expected error message for rejected order")
	}
}

func TestDhanBroker_CancelOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v2/orders/ORD-55555" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(dhanCancelOrderResp{
			OrderID:     "ORD-55555",
			OrderStatus: "CANCELLED",
		})
	}))
	defer server.Close()

	b := makeTestDhanBroker(t, server.URL)

	err := b.CancelOrder(context.Background(), "ORD-55555")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDhanBroker_GetFunds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v2/fundlimit" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// Note: "availabelBalance" has Dhan's official typo.
		w.Write([]byte(`{
			"dhanClientId": "test-client",
			"availabelBalance": 450000.50,
			"sodLimit": 500000.00,
			"utilizedAmount": 49999.50,
			"withdrawableBalance": 450000.50
		}`))
	}))
	defer server.Close()

	b := makeTestDhanBroker(t, server.URL)

	funds, err := b.GetFunds(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if funds.AvailableCash != 450000.50 {
		t.Errorf("expected available cash 450000.50, got %f", funds.AvailableCash)
	}
	if funds.UsedMargin != 49999.50 {
		t.Errorf("expected used margin 49999.50, got %f", funds.UsedMargin)
	}
	if funds.TotalBalance != 500000.00 {
		t.Errorf("expected total balance 500000.00, got %f", funds.TotalBalance)
	}
}

func TestDhanBroker_GetHoldings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/holdings":
			json.NewEncoder(w).Encode([]dhanHoldingResp{
				{
					Exchange:      "NSE",
					TradingSymbol: "RELIANCE",
					SecurityID:    "2885",
					ISIN:          "INE002A01018",
					TotalQty:      20,
					AvailableQty:  20,
					AvgCostPrice:  2450.50,
				},
				{
					Exchange:      "NSE",
					TradingSymbol: "TCS",
					SecurityID:    "11536",
					TotalQty:      10,
					AvailableQty:  10,
					AvgCostPrice:  4100.00,
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v2/marketfeed/ltp":
			// Return mock LTP data for the two securities.
			json.NewEncoder(w).Encode(dhanLTPResp{
				Status: "success",
				Data: map[string]map[string]dhanLTPEntry{
					"NSE_EQ": {
						"2885":  {LastPrice: 2655.00},
						"11536": {LastPrice: 4250.75},
					},
				},
			})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	b := makeTestDhanBroker(t, server.URL)

	holdings, err := b.GetHoldings(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holdings) != 2 {
		t.Fatalf("expected 2 holdings, got %d", len(holdings))
	}

	// Verify RELIANCE holding with LTP-enriched fields.
	if holdings[0].Symbol != "RELIANCE" {
		t.Errorf("expected RELIANCE, got %s", holdings[0].Symbol)
	}
	if holdings[0].Quantity != 20 {
		t.Errorf("expected qty 20, got %d", holdings[0].Quantity)
	}
	if holdings[0].AveragePrice != 2450.50 {
		t.Errorf("expected avg price 2450.50, got %f", holdings[0].AveragePrice)
	}
	if holdings[0].LastPrice != 2655.00 {
		t.Errorf("expected last price 2655.00, got %f", holdings[0].LastPrice)
	}
	// PnL = 20 * (2655.00 - 2450.50) = 20 * 204.50 = 4090.00
	expectedPnL := 20.0 * (2655.00 - 2450.50)
	if holdings[0].PnL != expectedPnL {
		t.Errorf("expected PnL %.2f, got %.2f", expectedPnL, holdings[0].PnL)
	}

	// Verify TCS holding.
	if holdings[1].Symbol != "TCS" {
		t.Errorf("expected TCS, got %s", holdings[1].Symbol)
	}
	if holdings[1].LastPrice != 4250.75 {
		t.Errorf("expected last price 4250.75, got %f", holdings[1].LastPrice)
	}
	// PnL = 10 * (4250.75 - 4100.00) = 10 * 150.75 = 1507.50
	expectedPnL2 := 10.0 * (4250.75 - 4100.00)
	if holdings[1].PnL != expectedPnL2 {
		t.Errorf("expected PnL %.2f, got %.2f", expectedPnL2, holdings[1].PnL)
	}
}

// TestDhanBroker_GetHoldings_LTPFailure verifies graceful degradation
// when the LTP API fails — holdings should still be returned with zero LTP/PnL.
func TestDhanBroker_GetHoldings_LTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/holdings":
			json.NewEncoder(w).Encode([]dhanHoldingResp{
				{
					Exchange:      "NSE",
					TradingSymbol: "RELIANCE",
					SecurityID:    "2885",
					TotalQty:      20,
					AvgCostPrice:  2450.50,
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v2/marketfeed/ltp":
			// Simulate LTP API failure.
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"errorType":"ServerError","errorCode":"DH-500","errorMessage":"Internal server error"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	b := makeTestDhanBroker(t, server.URL)

	holdings, err := b.GetHoldings(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}
	if holdings[0].Symbol != "RELIANCE" {
		t.Errorf("expected RELIANCE, got %s", holdings[0].Symbol)
	}
	// LTP should be zero (graceful degradation).
	if holdings[0].LastPrice != 0 {
		t.Errorf("expected last price 0 on LTP failure, got %f", holdings[0].LastPrice)
	}
	// PnL = 20 * (0 - 2450.50) = negative, but that's expected with zero LTP.
	expectedPnL := 20.0 * (0 - 2450.50)
	if holdings[0].PnL != expectedPnL {
		t.Errorf("expected PnL %.2f on LTP failure, got %.2f", expectedPnL, holdings[0].PnL)
	}
}

func TestDhanBroker_GetPositions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v2/positions" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]dhanPositionResp{
			{
				TradingSymbol:    "INFY",
				SecurityID:       "1594",
				ExchangeSegment:  "NSE_EQ",
				ProductType:      "CNC",
				NetQty:           15,
				CostPrice:        1850.00,
				UnrealizedProfit: 225.50,
			},
		})
	}))
	defer server.Close()

	b := makeTestDhanBroker(t, server.URL)

	positions, err := b.GetPositions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(positions))
	}
	if positions[0].Symbol != "INFY" {
		t.Errorf("expected INFY, got %s", positions[0].Symbol)
	}
	if positions[0].Quantity != 15 {
		t.Errorf("expected qty 15, got %d", positions[0].Quantity)
	}
	if positions[0].PnL != 225.50 {
		t.Errorf("expected PnL 225.50, got %f", positions[0].PnL)
	}
	if positions[0].Product != "CNC" {
		t.Errorf("expected CNC, got %s", positions[0].Product)
	}
}

func TestDhanBroker_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errorType":"Invalid_Authentication","errorCode":"DH-901","errorMessage":"Invalid token"}`))
	}))
	defer server.Close()

	b := makeTestDhanBroker(t, server.URL)

	_, err := b.GetFunds(context.Background())
	if err == nil {
		t.Error("expected error for 401 response")
	}

	_, err = b.PlaceOrder(context.Background(), Order{
		Symbol:   "RELIANCE",
		Exchange: "NSE",
		Side:     OrderSideBuy,
		Type:     OrderTypeMarket,
		Quantity: 1,
	})
	if err == nil {
		t.Error("expected error for 401 on PlaceOrder")
	}
}

func TestDhanBroker_MissingToken(t *testing.T) {
	cfgJSON, _ := json.Marshal(DhanConfig{
		AccessToken: "",
	})
	_, err := NewDhanBroker(cfgJSON)
	if err == nil {
		t.Error("expected error for missing access_token")
	}
}

func TestDhanBroker_UnknownSymbol(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	b := makeTestDhanBroker(t, server.URL)

	_, err := b.PlaceOrder(context.Background(), Order{
		Symbol:   "UNKNOWN_STOCK",
		Exchange: "NSE",
		Side:     OrderSideBuy,
		Type:     OrderTypeMarket,
		Quantity: 1,
	})
	if err == nil {
		t.Error("expected error for unknown symbol")
	}
}
