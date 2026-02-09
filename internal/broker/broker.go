// Package broker defines the broker abstraction layer.
//
// Design rules (from spec):
//   - Only one broker is active at a time.
//   - No strategy logic inside broker.
//   - No AI logic inside broker.
//   - Broker layer must be stateless.
//   - Broker APIs are used only for execution and account state.
package broker

import (
	"context"
	"fmt"
	"time"
)

// OrderSide represents buy or sell.
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderType represents the order type.
type OrderType string

const (
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeSL     OrderType = "SL"     // Stop-loss limit
	OrderTypeSLM    OrderType = "SL-M"   // Stop-loss market
)

// OrderStatus represents the current state of an order.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusOpen      OrderStatus = "OPEN"
	OrderStatusCompleted OrderStatus = "COMPLETED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusRejected  OrderStatus = "REJECTED"
)

// Order represents a trade order to be placed with the broker.
type Order struct {
	Symbol       string
	Exchange     string // "NSE"
	Side         OrderSide
	Type         OrderType
	Quantity     int
	Price        float64 // For limit orders
	TriggerPrice float64 // For stop-loss orders
	Product      string  // "CNC" for delivery
	Tag          string  // Strategy identifier for tracking
}

// OrderResponse is returned after placing an order.
type OrderResponse struct {
	OrderID   string
	Status    OrderStatus
	Message   string
	Timestamp time.Time
}

// OrderStatusResponse provides the current state of an existing order.
type OrderStatusResponse struct {
	OrderID       string
	Status        OrderStatus
	FilledQty     int
	PendingQty    int
	AveragePrice  float64
	Message       string
	Timestamp     time.Time
}

// Fund represents available trading funds.
type Fund struct {
	AvailableCash  float64
	UsedMargin     float64
	TotalBalance   float64
}

// Holding represents a stock held in the demat account.
type Holding struct {
	Symbol       string
	Exchange     string
	Quantity     int
	AveragePrice float64
	LastPrice    float64
	PnL          float64
}

// Position represents a current trading position.
type Position struct {
	Symbol       string
	Exchange     string
	Quantity     int
	AveragePrice float64
	LastPrice    float64
	PnL          float64
	Product      string
}

// Broker defines the interface that all broker implementations must satisfy.
// This is the only contract between the trading engine and any broker.
// Implementations must be stateless — all state lives in the database.
type Broker interface {
	// GetFunds returns the current available funds and margin information.
	GetFunds(ctx context.Context) (*Fund, error)

	// GetHoldings returns all delivery holdings in the demat account.
	GetHoldings(ctx context.Context) ([]Holding, error)

	// GetPositions returns all current open positions.
	GetPositions(ctx context.Context) ([]Position, error)

	// PlaceOrder submits a new order to the exchange.
	PlaceOrder(ctx context.Context, order Order) (*OrderResponse, error)

	// CancelOrder cancels an existing pending/open order.
	CancelOrder(ctx context.Context, orderID string) error

	// GetOrderStatus returns the current status of an order.
	GetOrderStatus(ctx context.Context, orderID string) (*OrderStatusResponse, error)
}

// Registry maps broker names to their factory functions.
// New broker implementations register here.
var Registry = map[string]func(configJSON []byte) (Broker, error){}

// New creates a broker instance by name using the registry.
func New(name string, configJSON []byte) (Broker, error) {
	factory, ok := Registry[name]
	if !ok {
		return nil, fmt.Errorf("broker: unknown broker %q, registered: %v", name, registeredNames())
	}
	return factory(configJSON)
}

func registeredNames() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	return names
}
