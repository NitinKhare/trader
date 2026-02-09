// Package broker - dhan.go provides the Dhan broker implementation stub.
//
// This is a placeholder that defines the structure for the Dhan API integration.
// Actual API calls will be implemented when the Dhan API credentials are available.
package broker

import (
	"context"
	"encoding/json"
	"fmt"
)

// DhanConfig holds Dhan-specific API configuration.
type DhanConfig struct {
	ClientID    string `json:"client_id"`
	AccessToken string `json:"access_token"`
	BaseURL     string `json:"base_url"`
}

// DhanBroker implements the Broker interface for Dhan.
type DhanBroker struct {
	config DhanConfig
}

func init() {
	Registry["dhan"] = NewDhanBroker
}

// NewDhanBroker creates a new Dhan broker instance from JSON config.
func NewDhanBroker(configJSON []byte) (Broker, error) {
	var cfg DhanConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("dhan broker: parse config: %w", err)
	}
	if cfg.ClientID == "" || cfg.AccessToken == "" {
		return nil, fmt.Errorf("dhan broker: client_id and access_token are required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.dhan.co"
	}
	return &DhanBroker{config: cfg}, nil
}

// GetFunds retrieves available funds from Dhan API.
// TODO: Implement actual Dhan API call.
func (d *DhanBroker) GetFunds(_ context.Context) (*Fund, error) {
	return nil, fmt.Errorf("dhan broker: GetFunds not yet implemented")
}

// GetHoldings retrieves delivery holdings from Dhan API.
// TODO: Implement actual Dhan API call.
func (d *DhanBroker) GetHoldings(_ context.Context) ([]Holding, error) {
	return nil, fmt.Errorf("dhan broker: GetHoldings not yet implemented")
}

// GetPositions retrieves open positions from Dhan API.
// TODO: Implement actual Dhan API call.
func (d *DhanBroker) GetPositions(_ context.Context) ([]Position, error) {
	return nil, fmt.Errorf("dhan broker: GetPositions not yet implemented")
}

// PlaceOrder submits an order via Dhan API.
// TODO: Implement actual Dhan API call.
func (d *DhanBroker) PlaceOrder(_ context.Context, _ Order) (*OrderResponse, error) {
	return nil, fmt.Errorf("dhan broker: PlaceOrder not yet implemented")
}

// CancelOrder cancels a pending order via Dhan API.
// TODO: Implement actual Dhan API call.
func (d *DhanBroker) CancelOrder(_ context.Context, _ string) error {
	return fmt.Errorf("dhan broker: CancelOrder not yet implemented")
}

// GetOrderStatus checks order status via Dhan API.
// TODO: Implement actual Dhan API call.
func (d *DhanBroker) GetOrderStatus(_ context.Context, _ string) (*OrderStatusResponse, error) {
	return nil, fmt.Errorf("dhan broker: GetOrderStatus not yet implemented")
}
