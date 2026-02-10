// Package broker - dhan.go implements the Broker interface using Dhan's v2 API.
//
// Dhan API v2:
//   - Base URL: https://api.dhan.co/v2
//   - Auth: access-token header (JWT, 24h validity)
//   - Orders: POST/GET/DELETE /v2/orders
//   - Funds: GET /v2/fundlimit
//   - Holdings: GET /v2/holdings
//   - Positions: GET /v2/positions
//   - Rate limit: 10 orders/sec, 250/min, 1000/hr, 7000/day
//   - Static IP whitelisting required for order APIs
package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// DhanConfig holds Dhan-specific API configuration.
type DhanConfig struct {
	ClientID       string `json:"client_id"`
	AccessToken    string `json:"access_token"`
	BaseURL        string `json:"base_url"`
	InstrumentFile string `json:"instrument_file"`
}

// DhanBroker implements the Broker interface for Dhan.
type DhanBroker struct {
	config      DhanConfig
	client      *http.Client
	instruments map[string]string // ticker -> securityId
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
	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("dhan broker: access_token is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.dhan.co"
	}

	b := &DhanBroker{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}

	// Load instrument mapping if configured.
	if cfg.InstrumentFile != "" {
		if err := b.loadInstruments(cfg.InstrumentFile); err != nil {
			return nil, fmt.Errorf("dhan broker: %w", err)
		}
	}

	return b, nil
}

// loadInstruments reads the ticker-to-securityId mapping from JSON.
func (d *DhanBroker) loadInstruments(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load instruments %s: %w", path, err)
	}

	var file struct {
		Instruments map[string]string `json:"instruments"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse instruments %s: %w", path, err)
	}

	d.instruments = file.Instruments
	return nil
}

// resolveSecurityID maps a ticker symbol to a Dhan securityId.
func (d *DhanBroker) resolveSecurityID(symbol string) (string, error) {
	if d.instruments == nil {
		return "", fmt.Errorf("no instrument mapping loaded — set instrument_file in config")
	}
	id, ok := d.instruments[symbol]
	if !ok {
		return "", fmt.Errorf("no securityId for symbol %q", symbol)
	}
	return id, nil
}

// mapExchangeSegment converts our exchange string to Dhan's enum.
func mapExchangeSegment(exchange string) string {
	switch exchange {
	case "NSE":
		return "NSE_EQ"
	case "BSE":
		return "BSE_EQ"
	default:
		return "NSE_EQ"
	}
}

// mapOrderType converts our OrderType to Dhan's enum.
func mapOrderType(ot OrderType) string {
	switch ot {
	case OrderTypeLimit:
		return "LIMIT"
	case OrderTypeMarket:
		return "MARKET"
	case OrderTypeSL:
		return "STOP_LOSS"
	case OrderTypeSLM:
		return "STOP_LOSS_MARKET"
	default:
		return "MARKET"
	}
}

// mapDhanStatus converts Dhan order status to our OrderStatus.
func mapDhanStatus(s string) OrderStatus {
	switch s {
	case "TRADED":
		return OrderStatusCompleted
	case "CANCELLED":
		return OrderStatusCancelled
	case "REJECTED":
		return OrderStatusRejected
	case "PENDING", "TRANSIT":
		return OrderStatusPending
	case "PART_TRADED", "TRIGGERED":
		return OrderStatusOpen
	default:
		return OrderStatusPending
	}
}

// --- Dhan API request/response types ---

// dhanPlaceOrderReq is the POST body for /v2/orders.
type dhanPlaceOrderReq struct {
	DhanClientID      string  `json:"dhanClientId"`
	CorrelationID     string  `json:"correlationId,omitempty"`
	TransactionType   string  `json:"transactionType"`
	ExchangeSegment   string  `json:"exchangeSegment"`
	ProductType       string  `json:"productType"`
	OrderType         string  `json:"orderType"`
	Validity          string  `json:"validity"`
	SecurityID        string  `json:"securityId"`
	Quantity          int     `json:"quantity"`
	DisclosedQuantity int     `json:"disclosedQuantity"`
	Price             float64 `json:"price"`
	TriggerPrice      float64 `json:"triggerPrice"`
	AfterMarketOrder  bool    `json:"afterMarketOrder"`
	AmoTime           string  `json:"amoTime,omitempty"`
	BoProfitValue     float64 `json:"boProfitValue"`
	BoStopLossValue   float64 `json:"boStopLossValue"`
}

// dhanPlaceOrderResp is the response from POST /v2/orders.
type dhanPlaceOrderResp struct {
	OrderID     string `json:"orderId"`
	OrderStatus string `json:"orderStatus"`
}

// dhanOrderDetailResp is the response from GET /v2/orders/{id}.
type dhanOrderDetailResp struct {
	OrderID            string  `json:"orderId"`
	OrderStatus        string  `json:"orderStatus"`
	FilledQty          int     `json:"filledQty"`
	RemainingQuantity  int     `json:"remainingQuantity"`
	AverageTradedPrice float64 `json:"averageTradedPrice"`
	OmsErrorCode       string  `json:"omsErrorCode"`
	OmsErrorDesc       string  `json:"omsErrorDescription"`
	Quantity           int     `json:"quantity"`
	CreateTime         string  `json:"createTime"`
	UpdateTime         string  `json:"updateTime"`
}

// dhanCancelOrderResp is the response from DELETE /v2/orders/{id}.
type dhanCancelOrderResp struct {
	OrderID     string `json:"orderId"`
	OrderStatus string `json:"orderStatus"`
}

// dhanFundResp is the response from GET /v2/fundlimit.
// Note: Dhan has a typo — "availabelBalance" (missing 'l').
type dhanFundResp struct {
	DhanClientID       string  `json:"dhanClientId"`
	AvailabelBalance   float64 `json:"availabelBalance"`
	SodLimit           float64 `json:"sodLimit"`
	UtilizedAmount     float64 `json:"utilizedAmount"`
	WithdrawableBalance float64 `json:"withdrawableBalance"`
}

// dhanHoldingResp is a single holding from GET /v2/holdings.
type dhanHoldingResp struct {
	Exchange      string  `json:"exchange"`
	TradingSymbol string  `json:"tradingSymbol"`
	SecurityID    string  `json:"securityId"`
	ISIN          string  `json:"isin"`
	TotalQty      int     `json:"totalQty"`
	AvailableQty  int     `json:"availableQty"`
	AvgCostPrice  float64 `json:"avgCostPrice"`
}

// dhanPositionResp is a single position from GET /v2/positions.
type dhanPositionResp struct {
	TradingSymbol    string  `json:"tradingSymbol"`
	SecurityID       string  `json:"securityId"`
	ExchangeSegment  string  `json:"exchangeSegment"`
	ProductType      string  `json:"productType"`
	NetQty           int     `json:"netQty"`
	CostPrice        float64 `json:"costPrice"`
	BuyAvg           float64 `json:"buyAvg"`
	SellAvg          float64 `json:"sellAvg"`
	RealizedProfit   float64 `json:"realizedProfit"`
	UnrealizedProfit float64 `json:"unrealizedProfit"`
}

// dhanErrorResp is the standard Dhan error response.
type dhanErrorResp struct {
	ErrorType    string `json:"errorType"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

// --- HTTP helper ---

// doRequest makes an authenticated request to the Dhan API.
func (d *DhanBroker) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := d.config.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(bodyJSON)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access-token", d.config.AccessToken)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Handle error responses.
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed (401): access token may have expired")
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited (429): too many requests")
	}

	// Dhan returns 4xx/5xx with error JSON.
	if resp.StatusCode >= 400 {
		var dhanErr dhanErrorResp
		if json.Unmarshal(respBody, &dhanErr) == nil && dhanErr.ErrorCode != "" {
			return nil, fmt.Errorf("dhan API error %s (%s): %s", dhanErr.ErrorCode, dhanErr.ErrorType, dhanErr.ErrorMessage)
		}
		return nil, fmt.Errorf("dhan API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// --- Broker interface implementation ---

// PlaceOrder submits an order to Dhan via POST /v2/orders.
func (d *DhanBroker) PlaceOrder(ctx context.Context, order Order) (*OrderResponse, error) {
	secID, err := d.resolveSecurityID(order.Symbol)
	if err != nil {
		return nil, fmt.Errorf("dhan broker: %w", err)
	}

	product := order.Product
	if product == "" {
		product = "CNC" // Default to delivery.
	}

	dhanReq := dhanPlaceOrderReq{
		DhanClientID:      d.config.ClientID,
		CorrelationID:     order.Tag,
		TransactionType:   string(order.Side),
		ExchangeSegment:   mapExchangeSegment(order.Exchange),
		ProductType:       product,
		OrderType:         mapOrderType(order.Type),
		Validity:          "DAY",
		SecurityID:        secID,
		Quantity:          order.Quantity,
		DisclosedQuantity: 0,
		Price:             order.Price,
		TriggerPrice:      order.TriggerPrice,
		AfterMarketOrder:  false,
	}

	respBody, err := d.doRequest(ctx, http.MethodPost, "/v2/orders", dhanReq)
	if err != nil {
		return nil, fmt.Errorf("dhan broker PlaceOrder: %w", err)
	}

	var dhanResp dhanPlaceOrderResp
	if err := json.Unmarshal(respBody, &dhanResp); err != nil {
		return nil, fmt.Errorf("dhan broker PlaceOrder: parse response: %w", err)
	}

	return &OrderResponse{
		OrderID:   dhanResp.OrderID,
		Status:    mapDhanStatus(dhanResp.OrderStatus),
		Message:   fmt.Sprintf("order placed: %s %d %s @ %s", order.Side, order.Quantity, order.Symbol, mapOrderType(order.Type)),
		Timestamp: time.Now(),
	}, nil
}

// GetOrderStatus checks order status via GET /v2/orders/{order-id}.
func (d *DhanBroker) GetOrderStatus(ctx context.Context, orderID string) (*OrderStatusResponse, error) {
	respBody, err := d.doRequest(ctx, http.MethodGet, "/v2/orders/"+orderID, nil)
	if err != nil {
		return nil, fmt.Errorf("dhan broker GetOrderStatus: %w", err)
	}

	var detail dhanOrderDetailResp
	if err := json.Unmarshal(respBody, &detail); err != nil {
		return nil, fmt.Errorf("dhan broker GetOrderStatus: parse response: %w", err)
	}

	msg := ""
	if detail.OmsErrorDesc != "" {
		msg = fmt.Sprintf("%s: %s", detail.OmsErrorCode, detail.OmsErrorDesc)
	}

	return &OrderStatusResponse{
		OrderID:      detail.OrderID,
		Status:       mapDhanStatus(detail.OrderStatus),
		FilledQty:    detail.FilledQty,
		PendingQty:   detail.RemainingQuantity,
		AveragePrice: detail.AverageTradedPrice,
		Message:      msg,
		Timestamp:    time.Now(),
	}, nil
}

// CancelOrder cancels a pending order via DELETE /v2/orders/{order-id}.
func (d *DhanBroker) CancelOrder(ctx context.Context, orderID string) error {
	_, err := d.doRequest(ctx, http.MethodDelete, "/v2/orders/"+orderID, nil)
	if err != nil {
		return fmt.Errorf("dhan broker CancelOrder: %w", err)
	}
	return nil
}

// GetFunds retrieves available funds via GET /v2/fundlimit.
func (d *DhanBroker) GetFunds(ctx context.Context) (*Fund, error) {
	respBody, err := d.doRequest(ctx, http.MethodGet, "/v2/fundlimit", nil)
	if err != nil {
		return nil, fmt.Errorf("dhan broker GetFunds: %w", err)
	}

	var fundResp dhanFundResp
	if err := json.Unmarshal(respBody, &fundResp); err != nil {
		return nil, fmt.Errorf("dhan broker GetFunds: parse response: %w", err)
	}

	return &Fund{
		AvailableCash: fundResp.AvailabelBalance,
		UsedMargin:    fundResp.UtilizedAmount,
		TotalBalance:  fundResp.SodLimit,
	}, nil
}

// GetHoldings retrieves delivery holdings via GET /v2/holdings.
func (d *DhanBroker) GetHoldings(ctx context.Context) ([]Holding, error) {
	respBody, err := d.doRequest(ctx, http.MethodGet, "/v2/holdings", nil)
	if err != nil {
		return nil, fmt.Errorf("dhan broker GetHoldings: %w", err)
	}

	var dhanHoldings []dhanHoldingResp
	if err := json.Unmarshal(respBody, &dhanHoldings); err != nil {
		return nil, fmt.Errorf("dhan broker GetHoldings: parse response: %w", err)
	}

	holdings := make([]Holding, 0, len(dhanHoldings))
	for _, h := range dhanHoldings {
		holdings = append(holdings, Holding{
			Symbol:       h.TradingSymbol,
			Exchange:     h.Exchange,
			Quantity:     h.TotalQty,
			AveragePrice: h.AvgCostPrice,
		})
	}

	return holdings, nil
}

// GetPositions retrieves open positions via GET /v2/positions.
func (d *DhanBroker) GetPositions(ctx context.Context) ([]Position, error) {
	respBody, err := d.doRequest(ctx, http.MethodGet, "/v2/positions", nil)
	if err != nil {
		return nil, fmt.Errorf("dhan broker GetPositions: %w", err)
	}

	var dhanPositions []dhanPositionResp
	if err := json.Unmarshal(respBody, &dhanPositions); err != nil {
		return nil, fmt.Errorf("dhan broker GetPositions: parse response: %w", err)
	}

	positions := make([]Position, 0, len(dhanPositions))
	for _, p := range dhanPositions {
		positions = append(positions, Position{
			Symbol:       p.TradingSymbol,
			Exchange:     "NSE",
			Quantity:     p.NetQty,
			AveragePrice: p.CostPrice,
			PnL:          p.UnrealizedProfit,
			Product:      p.ProductType,
		})
	}

	return positions, nil
}
