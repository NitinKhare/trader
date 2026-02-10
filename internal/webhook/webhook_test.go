package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/broker"
)

func newTestServer() *Server {
	logger := log.New(os.Stdout, "[test-webhook] ", log.LstdFlags)
	return NewServer(Config{
		Port:    0, // not used in tests (we use httptest)
		Path:    "/webhook/dhan/order",
		Enabled: true,
	}, logger)
}

// postJSON sends a POST request with a JSON body to the server's handler.
func postJSON(s *Server, body interface{}) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/webhook/dhan/order", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handlePostback(w, req)
	return w
}

func TestPostback_Traded(t *testing.T) {
	s := newTestServer()

	var received OrderUpdate
	var mu sync.Mutex
	s.OnOrderUpdate(func(u OrderUpdate) {
		mu.Lock()
		received = u
		mu.Unlock()
	})

	pb := DhanPostback{
		DhanClientID:       "1109435372",
		OrderID:            "ORD-123456",
		CorrelationID:      "sig_trend_follow_RELIANCE",
		OrderStatus:        "TRADED",
		TransactionType:    "BUY",
		ExchangeSegment:    "NSE_EQ",
		ProductType:        "CNC",
		OrderType:          "LIMIT",
		TradingSymbol:      "RELIANCE",
		SecurityID:         "2885",
		Quantity:           10,
		Price:              1250.50,
		FilledQty:          10,
		AverageTradedPrice: 1249.80,
		RemainingQuantity:  0,
	}

	resp := postJSON(s, pb)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	mu.Lock()
	defer mu.Unlock()
	if received.OrderID != "ORD-123456" {
		t.Errorf("expected OrderID ORD-123456, got %s", received.OrderID)
	}
	if received.Status != broker.OrderStatusCompleted {
		t.Errorf("expected status COMPLETED, got %s", received.Status)
	}
	if received.Symbol != "RELIANCE" {
		t.Errorf("expected symbol RELIANCE, got %s", received.Symbol)
	}
	if received.Side != "BUY" {
		t.Errorf("expected side BUY, got %s", received.Side)
	}
	if received.FilledQty != 10 {
		t.Errorf("expected filledQty 10, got %d", received.FilledQty)
	}
	if received.AveragePrice != 1249.80 {
		t.Errorf("expected avgPrice 1249.80, got %.2f", received.AveragePrice)
	}
	if received.CorrelationID != "sig_trend_follow_RELIANCE" {
		t.Errorf("expected correlationID sig_trend_follow_RELIANCE, got %s", received.CorrelationID)
	}
	if received.PendingQty != 0 {
		t.Errorf("expected pendingQty 0, got %d", received.PendingQty)
	}
}

func TestPostback_Rejected(t *testing.T) {
	s := newTestServer()

	var received OrderUpdate
	var mu sync.Mutex
	s.OnOrderUpdate(func(u OrderUpdate) {
		mu.Lock()
		received = u
		mu.Unlock()
	})

	pb := DhanPostback{
		OrderID:             "ORD-789",
		OrderStatus:         "REJECTED",
		TransactionType:     "BUY",
		TradingSymbol:       "TCS",
		SecurityID:          "11536",
		Quantity:            5,
		OmsErrorCode:        "OMS-001",
		OmsErrorDescription: "Insufficient margin",
	}

	resp := postJSON(s, pb)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	mu.Lock()
	defer mu.Unlock()
	if received.Status != broker.OrderStatusRejected {
		t.Errorf("expected status REJECTED, got %s", received.Status)
	}
	if received.ErrorCode != "OMS-001" {
		t.Errorf("expected errorCode OMS-001, got %s", received.ErrorCode)
	}
	if received.ErrorMessage != "Insufficient margin" {
		t.Errorf("expected errorMessage 'Insufficient margin', got %s", received.ErrorMessage)
	}
}

func TestPostback_Cancelled(t *testing.T) {
	s := newTestServer()

	var received OrderUpdate
	var mu sync.Mutex
	s.OnOrderUpdate(func(u OrderUpdate) {
		mu.Lock()
		received = u
		mu.Unlock()
	})

	pb := DhanPostback{
		OrderID:         "ORD-CXL-100",
		OrderStatus:     "CANCELLED",
		TransactionType: "SELL",
		TradingSymbol:   "INFY",
		Quantity:        20,
		FilledQty:       0,
	}

	resp := postJSON(s, pb)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	mu.Lock()
	defer mu.Unlock()
	if received.Status != broker.OrderStatusCancelled {
		t.Errorf("expected CANCELLED, got %s", received.Status)
	}
	if received.Side != "SELL" {
		t.Errorf("expected side SELL, got %s", received.Side)
	}
}

func TestPostback_PartialFill(t *testing.T) {
	s := newTestServer()

	var received OrderUpdate
	var mu sync.Mutex
	s.OnOrderUpdate(func(u OrderUpdate) {
		mu.Lock()
		received = u
		mu.Unlock()
	})

	pb := DhanPostback{
		OrderID:            "ORD-PART-200",
		OrderStatus:        "PART_TRADED",
		TransactionType:    "BUY",
		TradingSymbol:      "HDFCBANK",
		Quantity:           100,
		FilledQty:          40,
		RemainingQuantity:  60,
		AverageTradedPrice: 1650.25,
	}

	resp := postJSON(s, pb)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	mu.Lock()
	defer mu.Unlock()
	if received.Status != broker.OrderStatusOpen {
		t.Errorf("expected OPEN (PART_TRADED), got %s", received.Status)
	}
	if received.FilledQty != 40 {
		t.Errorf("expected filledQty 40, got %d", received.FilledQty)
	}
	if received.PendingQty != 60 {
		t.Errorf("expected pendingQty 60, got %d", received.PendingQty)
	}
}

func TestPostback_Expired(t *testing.T) {
	s := newTestServer()

	var received OrderUpdate
	var mu sync.Mutex
	s.OnOrderUpdate(func(u OrderUpdate) {
		mu.Lock()
		received = u
		mu.Unlock()
	})

	pb := DhanPostback{
		OrderID:         "ORD-EXP-300",
		OrderStatus:     "EXPIRED",
		TransactionType: "BUY",
		TradingSymbol:   "SBIN",
		Quantity:        50,
	}

	resp := postJSON(s, pb)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	mu.Lock()
	defer mu.Unlock()
	// EXPIRED maps to CANCELLED.
	if received.Status != broker.OrderStatusCancelled {
		t.Errorf("expected CANCELLED (expired), got %s", received.Status)
	}
}

func TestPostback_Pending(t *testing.T) {
	s := newTestServer()

	var received OrderUpdate
	var mu sync.Mutex
	s.OnOrderUpdate(func(u OrderUpdate) {
		mu.Lock()
		received = u
		mu.Unlock()
	})

	pb := DhanPostback{
		OrderID:         "ORD-PND-400",
		OrderStatus:     "PENDING",
		TransactionType: "BUY",
		TradingSymbol:   "WIPRO",
		Quantity:        30,
	}

	resp := postJSON(s, pb)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	mu.Lock()
	defer mu.Unlock()
	if received.Status != broker.OrderStatusPending {
		t.Errorf("expected PENDING, got %s", received.Status)
	}
}

func TestPostback_Transit(t *testing.T) {
	s := newTestServer()

	var received OrderUpdate
	var mu sync.Mutex
	s.OnOrderUpdate(func(u OrderUpdate) {
		mu.Lock()
		received = u
		mu.Unlock()
	})

	pb := DhanPostback{
		OrderID:         "ORD-TRS-500",
		OrderStatus:     "TRANSIT",
		TransactionType: "BUY",
		TradingSymbol:   "LT",
		Quantity:        15,
	}

	resp := postJSON(s, pb)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	mu.Lock()
	defer mu.Unlock()
	if received.Status != broker.OrderStatusPending {
		t.Errorf("expected PENDING (transit), got %s", received.Status)
	}
}

func TestPostback_InvalidJSON(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/webhook/dhan/order",
		bytes.NewReader([]byte(`{not valid json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handlePostback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestPostback_MissingOrderID(t *testing.T) {
	s := newTestServer()

	pb := DhanPostback{
		OrderStatus:     "TRADED",
		TransactionType: "BUY",
		TradingSymbol:   "RELIANCE",
	}

	resp := postJSON(s, pb)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing orderId, got %d", resp.Code)
	}
}

func TestPostback_WrongMethod(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/webhook/dhan/order", nil)
	w := httptest.NewRecorder()
	s.handlePostback(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", w.Code)
	}
}

func TestPostback_MultipleHandlers(t *testing.T) {
	s := newTestServer()

	var wg sync.WaitGroup
	count := 0
	var mu sync.Mutex

	// Register 3 handlers.
	for i := 0; i < 3; i++ {
		wg.Add(1)
		s.OnOrderUpdate(func(_ OrderUpdate) {
			mu.Lock()
			count++
			mu.Unlock()
			wg.Done()
		})
	}

	pb := DhanPostback{
		OrderID:         "ORD-MULTI-600",
		OrderStatus:     "TRADED",
		TransactionType: "BUY",
		TradingSymbol:   "ITC",
		Quantity:        100,
	}

	postJSON(s, pb)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if count != 3 {
		t.Errorf("expected 3 handler invocations, got %d", count)
	}
}

func TestRecentUpdates(t *testing.T) {
	s := newTestServer()

	// Send 5 postbacks.
	for i := 1; i <= 5; i++ {
		pb := DhanPostback{
			OrderID:         fmt.Sprintf("ORD-%d", i),
			OrderStatus:     "TRADED",
			TransactionType: "BUY",
			TradingSymbol:   "RELIANCE",
			Quantity:        10,
		}
		postJSON(s, pb)
	}

	// Request last 3.
	recent := s.RecentUpdates(3)
	if len(recent) != 3 {
		t.Fatalf("expected 3 recent updates, got %d", len(recent))
	}
	if recent[0].OrderID != "ORD-3" {
		t.Errorf("expected first recent to be ORD-3, got %s", recent[0].OrderID)
	}
	if recent[2].OrderID != "ORD-5" {
		t.Errorf("expected last recent to be ORD-5, got %s", recent[2].OrderID)
	}
}

func TestServerStartShutdown(t *testing.T) {
	logger := log.New(os.Stdout, "[test-webhook] ", log.LstdFlags)
	s := NewServer(Config{
		Port:    18923, // unlikely to be in use
		Path:    "/webhook/dhan/order",
		Enabled: true,
	}, logger)

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Give server time to bind.
	time.Sleep(50 * time.Millisecond)

	// Health check.
	resp, err := http.Get("http://localhost:18923/health")
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("health check expected 200, got %d", resp.StatusCode)
	}

	// Shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
}

