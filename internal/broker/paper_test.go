package broker

import (
	"context"
	"testing"
)

func TestPaperBroker_InitialFunds(t *testing.T) {
	pb := NewPaperBroker(500000)
	ctx := context.Background()

	funds, err := pb.GetFunds(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if funds.AvailableCash != 500000 {
		t.Errorf("expected 500000, got %.2f", funds.AvailableCash)
	}
}

func TestPaperBroker_BuyReducesCash(t *testing.T) {
	pb := NewPaperBroker(500000)
	ctx := context.Background()

	order := Order{
		Symbol:   "RELIANCE",
		Exchange: "NSE",
		Side:     OrderSideBuy,
		Type:     OrderTypeLimit,
		Quantity: 10,
		Price:    2500,
		Product:  "CNC",
	}

	resp, err := pb.PlaceOrder(ctx, order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != OrderStatusCompleted {
		t.Errorf("expected COMPLETED, got %s", resp.Status)
	}

	funds, _ := pb.GetFunds(ctx)
	expectedCash := 500000.0 - (2500.0 * 10)
	if funds.AvailableCash != expectedCash {
		t.Errorf("expected %.2f, got %.2f", expectedCash, funds.AvailableCash)
	}
}

func TestPaperBroker_SellIncreaseCash(t *testing.T) {
	pb := NewPaperBroker(500000)
	ctx := context.Background()

	// Buy first.
	buyOrder := Order{
		Symbol: "TCS", Exchange: "NSE", Side: OrderSideBuy,
		Type: OrderTypeLimit, Quantity: 5, Price: 3500, Product: "CNC",
	}
	pb.PlaceOrder(ctx, buyOrder)

	// Sell.
	sellOrder := Order{
		Symbol: "TCS", Exchange: "NSE", Side: OrderSideSell,
		Type: OrderTypeLimit, Quantity: 5, Price: 3600, Product: "CNC",
	}
	resp, err := pb.PlaceOrder(ctx, sellOrder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != OrderStatusCompleted {
		t.Errorf("expected COMPLETED, got %s", resp.Status)
	}

	funds, _ := pb.GetFunds(ctx)
	// Started with 500000, bought 5*3500=17500, sold 5*3600=18000.
	expectedCash := 500000.0 - 17500.0 + 18000.0
	if funds.AvailableCash != expectedCash {
		t.Errorf("expected %.2f, got %.2f", expectedCash, funds.AvailableCash)
	}
}

func TestPaperBroker_RejectsInsufficientFunds(t *testing.T) {
	pb := NewPaperBroker(1000)
	ctx := context.Background()

	order := Order{
		Symbol: "RELIANCE", Exchange: "NSE", Side: OrderSideBuy,
		Type: OrderTypeLimit, Quantity: 10, Price: 2500, Product: "CNC",
	}

	resp, err := pb.PlaceOrder(ctx, order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != OrderStatusRejected {
		t.Errorf("expected REJECTED, got %s", resp.Status)
	}
}

func TestPaperBroker_RejectsInsufficientHoldings(t *testing.T) {
	pb := NewPaperBroker(500000)
	ctx := context.Background()

	order := Order{
		Symbol: "TCS", Exchange: "NSE", Side: OrderSideSell,
		Type: OrderTypeLimit, Quantity: 10, Price: 3500, Product: "CNC",
	}

	resp, err := pb.PlaceOrder(ctx, order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != OrderStatusRejected {
		t.Errorf("expected REJECTED, got %s", resp.Status)
	}
}

func TestPaperBroker_HoldingsTrack(t *testing.T) {
	pb := NewPaperBroker(500000)
	ctx := context.Background()

	order := Order{
		Symbol: "INFY", Exchange: "NSE", Side: OrderSideBuy,
		Type: OrderTypeLimit, Quantity: 20, Price: 1500, Product: "CNC",
	}
	pb.PlaceOrder(ctx, order)

	holdings, err := pb.GetHoldings(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}
	if holdings[0].Symbol != "INFY" || holdings[0].Quantity != 20 {
		t.Errorf("unexpected holding: %+v", holdings[0])
	}
}

func TestPaperBroker_OrderStatusTracked(t *testing.T) {
	pb := NewPaperBroker(500000)
	ctx := context.Background()

	order := Order{
		Symbol: "SBIN", Exchange: "NSE", Side: OrderSideBuy,
		Type: OrderTypeLimit, Quantity: 50, Price: 600, Product: "CNC",
	}
	resp, _ := pb.PlaceOrder(ctx, order)

	status, err := pb.GetOrderStatus(ctx, resp.OrderID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != OrderStatusCompleted {
		t.Errorf("expected COMPLETED, got %s", status.Status)
	}
	if status.FilledQty != 50 {
		t.Errorf("expected filled qty 50, got %d", status.FilledQty)
	}
}

func TestPaperBroker_RestoreHolding(t *testing.T) {
	pb := NewPaperBroker(500000)
	ctx := context.Background()

	pb.RestoreHolding("RELIANCE", "NSE", 10, 2500.0)

	holdings, err := pb.GetHoldings(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}
	if holdings[0].Symbol != "RELIANCE" || holdings[0].Quantity != 10 || holdings[0].AveragePrice != 2500.0 {
		t.Errorf("unexpected holding: %+v", holdings[0])
	}

	// Verify funds adjusted.
	funds, _ := pb.GetFunds(ctx)
	expectedCash := 500000.0 - 25000.0
	if funds.AvailableCash != expectedCash {
		t.Errorf("expected cash %.2f, got %.2f", expectedCash, funds.AvailableCash)
	}
	if funds.UsedMargin != 25000.0 {
		t.Errorf("expected used margin 25000.00, got %.2f", funds.UsedMargin)
	}
}

func TestPaperBroker_RestoreHolding_ThenSell(t *testing.T) {
	pb := NewPaperBroker(500000)
	ctx := context.Background()

	// Restore a holding.
	pb.RestoreHolding("TCS", "NSE", 5, 3500.0)

	// Sell it at a profit.
	sellOrder := Order{
		Symbol: "TCS", Exchange: "NSE", Side: OrderSideSell,
		Type: OrderTypeLimit, Quantity: 5, Price: 3600.0, Product: "CNC",
	}
	resp, err := pb.PlaceOrder(ctx, sellOrder)
	if err != nil {
		t.Fatalf("sell failed: %v", err)
	}
	if resp.Status != OrderStatusCompleted {
		t.Errorf("expected COMPLETED, got %s", resp.Status)
	}

	// Holdings should be empty.
	holdings, _ := pb.GetHoldings(ctx)
	if len(holdings) != 0 {
		t.Errorf("expected 0 holdings after sell, got %d", len(holdings))
	}

	// Cash should reflect profit: 500000 - 17500 (restore) + 18000 (sell) = 500500.
	funds, _ := pb.GetFunds(ctx)
	expectedCash := 500000.0 - 17500.0 + 18000.0
	if funds.AvailableCash != expectedCash {
		t.Errorf("expected cash %.2f, got %.2f", expectedCash, funds.AvailableCash)
	}
}

func TestPaperBroker_RestoreHolding_Multiple(t *testing.T) {
	pb := NewPaperBroker(500000)
	ctx := context.Background()

	pb.RestoreHolding("RELIANCE", "NSE", 10, 2500.0)
	pb.RestoreHolding("TCS", "NSE", 5, 3500.0)

	holdings, _ := pb.GetHoldings(ctx)
	if len(holdings) != 2 {
		t.Fatalf("expected 2 holdings, got %d", len(holdings))
	}

	funds, _ := pb.GetFunds(ctx)
	expectedCash := 500000.0 - 25000.0 - 17500.0
	if funds.AvailableCash != expectedCash {
		t.Errorf("expected cash %.2f, got %.2f", expectedCash, funds.AvailableCash)
	}
}
