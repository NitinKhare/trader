// Package broker - paper.go implements the paper trading broker.
//
// The paper broker simulates order execution using candle data.
// It uses the same interface as live brokers so all engine logic
// remains identical between paper and live modes.
package broker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PaperBroker simulates broker operations for paper trading.
// Orders are filled immediately at the requested price (simplified).
// In a more advanced version, fills would be based on next-candle OHLCV.
type PaperBroker struct {
	mu       sync.Mutex
	funds    Fund
	orders   map[string]*paperOrder
	holdings map[string]*Holding
	nextID   int
}

type paperOrder struct {
	Order    Order
	Response OrderStatusResponse
}

// NewPaperBroker creates a paper broker with the given initial capital.
func NewPaperBroker(initialCapital float64) *PaperBroker {
	return &PaperBroker{
		funds: Fund{
			AvailableCash: initialCapital,
			TotalBalance:  initialCapital,
		},
		orders:   make(map[string]*paperOrder),
		holdings: make(map[string]*Holding),
	}
}

func (pb *PaperBroker) GetFunds(_ context.Context) (*Fund, error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	f := pb.funds
	return &f, nil
}

func (pb *PaperBroker) GetHoldings(_ context.Context) ([]Holding, error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	result := make([]Holding, 0, len(pb.holdings))
	for _, h := range pb.holdings {
		result = append(result, *h)
	}
	return result, nil
}

func (pb *PaperBroker) GetPositions(_ context.Context) ([]Position, error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	// In delivery trading, positions are essentially holdings.
	positions := make([]Position, 0, len(pb.holdings))
	for _, h := range pb.holdings {
		positions = append(positions, Position{
			Symbol:       h.Symbol,
			Exchange:     h.Exchange,
			Quantity:     h.Quantity,
			AveragePrice: h.AveragePrice,
			LastPrice:    h.LastPrice,
			PnL:          h.PnL,
			Product:      "CNC",
		})
	}
	return positions, nil
}

// PlaceOrder simulates order placement.
// For paper trading, market and limit orders are filled immediately at the order price.
func (pb *PaperBroker) PlaceOrder(_ context.Context, order Order) (*OrderResponse, error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.nextID++
	orderID := fmt.Sprintf("PAPER-%d", pb.nextID)

	fillPrice := order.Price
	if order.Type == OrderTypeMarket {
		// In paper mode, market orders fill at the specified price.
		// Real implementation would use last traded price.
		fillPrice = order.Price
	}

	cost := fillPrice * float64(order.Quantity)

	if order.Side == OrderSideBuy {
		if cost > pb.funds.AvailableCash {
			return &OrderResponse{
				OrderID:   orderID,
				Status:    OrderStatusRejected,
				Message:   "insufficient funds",
				Timestamp: time.Now(),
			}, nil
		}

		pb.funds.AvailableCash -= cost
		pb.funds.UsedMargin += cost

		// Update or create holding.
		if h, exists := pb.holdings[order.Symbol]; exists {
			totalQty := h.Quantity + order.Quantity
			h.AveragePrice = (h.AveragePrice*float64(h.Quantity) + fillPrice*float64(order.Quantity)) / float64(totalQty)
			h.Quantity = totalQty
		} else {
			pb.holdings[order.Symbol] = &Holding{
				Symbol:       order.Symbol,
				Exchange:     order.Exchange,
				Quantity:     order.Quantity,
				AveragePrice: fillPrice,
				LastPrice:    fillPrice,
			}
		}
	} else if order.Side == OrderSideSell {
		h, exists := pb.holdings[order.Symbol]
		if !exists || h.Quantity < order.Quantity {
			return &OrderResponse{
				OrderID:   orderID,
				Status:    OrderStatusRejected,
				Message:   "insufficient holdings",
				Timestamp: time.Now(),
			}, nil
		}

		proceeds := fillPrice * float64(order.Quantity)
		pb.funds.AvailableCash += proceeds
		pb.funds.UsedMargin -= h.AveragePrice * float64(order.Quantity)

		h.Quantity -= order.Quantity
		if h.Quantity == 0 {
			delete(pb.holdings, order.Symbol)
		}
	}

	// Record order as completed.
	pb.orders[orderID] = &paperOrder{
		Order: order,
		Response: OrderStatusResponse{
			OrderID:      orderID,
			Status:       OrderStatusCompleted,
			FilledQty:    order.Quantity,
			PendingQty:   0,
			AveragePrice: fillPrice,
			Message:      "paper fill",
			Timestamp:    time.Now(),
		},
	}

	return &OrderResponse{
		OrderID:   orderID,
		Status:    OrderStatusCompleted,
		Message:   "paper order filled",
		Timestamp: time.Now(),
	}, nil
}

func (pb *PaperBroker) CancelOrder(_ context.Context, orderID string) error {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	po, exists := pb.orders[orderID]
	if !exists {
		return fmt.Errorf("paper broker: order %s not found", orderID)
	}
	if po.Response.Status == OrderStatusCompleted {
		return fmt.Errorf("paper broker: order %s already completed", orderID)
	}

	po.Response.Status = OrderStatusCancelled
	return nil
}

func (pb *PaperBroker) GetOrderStatus(_ context.Context, orderID string) (*OrderStatusResponse, error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	po, exists := pb.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("paper broker: order %s not found", orderID)
	}

	resp := po.Response
	return &resp, nil
}
