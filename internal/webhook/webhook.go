// Package webhook provides an HTTP server to receive order postback
// notifications from the Dhan broker API.
//
// Dhan sends a POST request with a JSON payload whenever an order's status
// changes (e.g. PENDING → TRADED, PENDING → REJECTED). The postback URL
// is configured in the Dhan web dashboard when generating an access token.
//
// This package:
//   - Starts a lightweight HTTP server on a configurable port.
//   - Parses the Dhan postback payload.
//   - Maps it to the broker-agnostic OrderUpdate type.
//   - Invokes registered callback functions so the engine can react
//     (log to DB, adjust positions, send alerts, etc.).
package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/broker"
)

// ────────────────────────────────────────────────────────────────────
// Configuration
// ────────────────────────────────────────────────────────────────────

// Config holds webhook server settings.
type Config struct {
	Port    int    `json:"port"`    // e.g. 8080
	Path    string `json:"path"`    // e.g. "/webhook/dhan/order"
	Enabled bool   `json:"enabled"` // master switch
}

// ────────────────────────────────────────────────────────────────────
// Dhan postback payload (matches Dhan API v2)
// ────────────────────────────────────────────────────────────────────

// DhanPostback is the JSON body Dhan sends when an order status changes.
type DhanPostback struct {
	DhanClientID       string  `json:"dhanClientId"`
	OrderID            string  `json:"orderId"`
	CorrelationID      string  `json:"correlationId"`
	OrderStatus        string  `json:"orderStatus"` // TRANSIT, PENDING, REJECTED, CANCELLED, TRADED, EXPIRED
	TransactionType    string  `json:"transactionType"`
	ExchangeSegment    string  `json:"exchangeSegment"`
	ProductType        string  `json:"productType"`
	OrderType          string  `json:"orderType"`
	Validity           string  `json:"validity"`
	TradingSymbol      string  `json:"tradingSymbol"`
	SecurityID         string  `json:"securityId"`
	Quantity           int     `json:"quantity"`
	DisclosedQuantity  int     `json:"disclosedQuantity"`
	Price              float64 `json:"price"`
	TriggerPrice       float64 `json:"triggerPrice"`
	FilledQty          int     `json:"filled_qty"`
	AverageTradedPrice float64 `json:"averageTradedPrice"`
	RemainingQuantity  int     `json:"remainingQuantity"`
	CreateTime         string  `json:"createTime"`
	UpdateTime         string  `json:"updateTime"`
	ExchangeTime       string  `json:"exchangeTime"`
	DrvExpiryDate      string  `json:"drvExpiryDate"`
	DrvOptionType      string  `json:"drvOptionType"`
	DrvStrikePrice     float64 `json:"drvStrikePrice"`
	OmsErrorCode       string  `json:"omsErrorCode"`
	OmsErrorDescription string `json:"omsErrorDescription"`
}

// ────────────────────────────────────────────────────────────────────
// Broker-agnostic order update
// ────────────────────────────────────────────────────────────────────

// OrderUpdate is the broker-agnostic representation of a status change.
// Callbacks receive this instead of the raw Dhan payload so that
// upstream code is not coupled to Dhan.
type OrderUpdate struct {
	OrderID       string
	CorrelationID string // maps to the Tag/SignalID used when placing the order
	Symbol        string
	Status        broker.OrderStatus
	Side          string  // "BUY" or "SELL"
	Quantity      int     // total order quantity
	FilledQty     int     // quantity filled so far
	PendingQty    int     // remaining quantity
	AveragePrice  float64 // average fill price
	ErrorCode     string  // OMS error code (if rejected/cancelled)
	ErrorMessage  string  // human-readable error (if rejected/cancelled)
	ReceivedAt    time.Time
}

// ────────────────────────────────────────────────────────────────────
// Callback type
// ────────────────────────────────────────────────────────────────────

// OrderUpdateHandler is called whenever a valid postback is received.
type OrderUpdateHandler func(update OrderUpdate)

// ────────────────────────────────────────────────────────────────────
// Server
// ────────────────────────────────────────────────────────────────────

// Server is the HTTP webhook receiver.
type Server struct {
	cfg      Config
	logger   *log.Logger
	srv      *http.Server
	mu       sync.RWMutex
	handlers []OrderUpdateHandler
	updates  []OrderUpdate // ring buffer for recent updates (for debugging)
}

// NewServer creates a new webhook server. It does not start listening
// until Start() is called.
func NewServer(cfg Config, logger *log.Logger) *Server {
	return &Server{
		cfg:    cfg,
		logger: logger,
	}
}

// OnOrderUpdate registers a handler that will be called for every
// validated postback. Multiple handlers may be registered.
func (s *Server) OnOrderUpdate(h OrderUpdateHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, h)
}

// RecentUpdates returns a copy of the last N order updates (for status/debug).
func (s *Server) RecentUpdates(n int) []OrderUpdate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if n > len(s.updates) {
		n = len(s.updates)
	}
	out := make([]OrderUpdate, n)
	copy(out, s.updates[len(s.updates)-n:])
	return out
}

// Start begins listening for postback HTTP requests.
// It returns immediately; the server runs in a background goroutine.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	path := s.cfg.Path
	if path == "" {
		path = "/webhook/dhan/order"
	}
	mux.HandleFunc(path, s.handlePostback)

	// Health check endpoint.
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	addr := fmt.Sprintf(":%d", s.cfg.Port)
	s.srv = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Printf("[webhook] starting server on %s%s", addr, path)

	go func() {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Printf("[webhook] server error: %v", err)
		}
	}()

	return nil
}

// Shutdown gracefully stops the webhook server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	s.logger.Println("[webhook] shutting down server")
	return s.srv.Shutdown(ctx)
}

// ────────────────────────────────────────────────────────────────────
// HTTP handler
// ────────────────────────────────────────────────────────────────────

func (s *Server) handlePostback(w http.ResponseWriter, r *http.Request) {
	// Only accept POST.
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode JSON body.
	var pb DhanPostback
	if err := json.NewDecoder(r.Body).Decode(&pb); err != nil {
		s.logger.Printf("[webhook] invalid JSON payload: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Basic validation.
	if pb.OrderID == "" {
		s.logger.Println("[webhook] missing orderId in postback")
		http.Error(w, "missing orderId", http.StatusBadRequest)
		return
	}

	// Map to broker-agnostic OrderUpdate.
	update := OrderUpdate{
		OrderID:       pb.OrderID,
		CorrelationID: pb.CorrelationID,
		Symbol:        pb.TradingSymbol,
		Status:        mapDhanPostbackStatus(pb.OrderStatus),
		Side:          pb.TransactionType,
		Quantity:      pb.Quantity,
		FilledQty:     pb.FilledQty,
		PendingQty:    pb.RemainingQuantity,
		AveragePrice:  pb.AverageTradedPrice,
		ErrorCode:     pb.OmsErrorCode,
		ErrorMessage:  pb.OmsErrorDescription,
		ReceivedAt:    time.Now(),
	}

	s.logger.Printf("[webhook] postback: order=%s symbol=%s status=%s filled=%d/%d price=%.2f",
		update.OrderID, update.Symbol, update.Status, update.FilledQty, update.Quantity, update.AveragePrice)

	// Store in recent updates buffer (keep last 100).
	s.mu.Lock()
	s.updates = append(s.updates, update)
	if len(s.updates) > 100 {
		s.updates = s.updates[len(s.updates)-100:]
	}
	// Copy handlers under lock to avoid holding lock during callbacks.
	handlers := make([]OrderUpdateHandler, len(s.handlers))
	copy(handlers, s.handlers)
	s.mu.Unlock()

	// Invoke all registered handlers.
	for _, h := range handlers {
		h(update)
	}

	// Respond 200 OK to Dhan.
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"received":true}`)
}

// ────────────────────────────────────────────────────────────────────
// Status mapping
// ────────────────────────────────────────────────────────────────────

// mapDhanPostbackStatus converts Dhan's orderStatus string to the
// broker-agnostic OrderStatus enum.
func mapDhanPostbackStatus(s string) broker.OrderStatus {
	switch s {
	case "TRADED":
		return broker.OrderStatusCompleted
	case "CANCELLED":
		return broker.OrderStatusCancelled
	case "REJECTED":
		return broker.OrderStatusRejected
	case "PENDING", "TRANSIT":
		return broker.OrderStatusPending
	case "PART_TRADED", "TRIGGERED":
		return broker.OrderStatusOpen
	case "EXPIRED":
		return broker.OrderStatusCancelled // treat expired as cancelled
	default:
		return broker.OrderStatusPending
	}
}
