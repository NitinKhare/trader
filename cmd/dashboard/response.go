package main

import "time"

// MetricsResponse contains overall performance metrics
type MetricsResponse struct {
	TotalPnL          float64 `json:"total_pnl"`
	TotalPnLPercent   float64 `json:"total_pnl_percent"`
	WinRate           float64 `json:"win_rate"`
	ProfitFactor      float64 `json:"profit_factor"`
	Drawdown          float64 `json:"drawdown"`
	DrawdownPercent   float64 `json:"drawdown_percent"`
	SharpeRatio       float64 `json:"sharpe_ratio"`
	TotalTrades       int     `json:"total_trades"`
	WinningTrades     int     `json:"winning_trades"`
	LosingTrades      int     `json:"losing_trades"`
	AvgPnL            float64 `json:"avg_pnl"`
	GrossProfit       float64 `json:"gross_profit"`
	GrossLoss         float64 `json:"gross_loss"`
	AvgHoldDays       float64 `json:"avg_hold_days"`
	InitialCapital    float64 `json:"initial_capital"`
	FinalCapital      float64 `json:"final_capital"`
	Timestamp         time.Time `json:"timestamp"`
}

// PositionResponse represents a single open position
type PositionResponse struct {
	ID              int64     `json:"id"`
	Symbol          string    `json:"symbol"`
	Quantity        int       `json:"quantity"`
	EntryPrice      float64   `json:"entry_price"`
	EntryTime       time.Time `json:"entry_time"`
	StopLoss        float64   `json:"stop_loss"`
	Target          float64   `json:"target"`
	StrategyID      string    `json:"strategy_id"`
	UnrealizedPnL   float64   `json:"unrealized_pnl"`
	UnrealizedPnLPercent float64 `json:"unrealized_pnl_percent"`
}

// PositionsResponse contains all open positions
type PositionsResponse struct {
	Positions        []PositionResponse `json:"positions"`
	TotalCapitalUsed float64            `json:"total_capital_used"`
	AvailableCapital float64            `json:"available_capital"`
	CapitalUtilizationPercent float64   `json:"capital_utilization_percent"`
	OpenPositionCount int               `json:"open_position_count"`
	Timestamp        time.Time          `json:"timestamp"`
}

// EquityCurvePoint represents a single point in the equity curve
type EquityCurvePoint struct {
	Date      time.Time `json:"date"`
	Equity    float64   `json:"equity"`
	Drawdown  float64   `json:"drawdown"`
	DrawdownPercent float64 `json:"drawdown_percent"`
}

// EquityCurveResponse contains the equity curve data for charting
type EquityCurveResponse struct {
	Points          []EquityCurvePoint `json:"points"`
	StartEquity     float64            `json:"start_equity"`
	FinalEquity     float64            `json:"final_equity"`
	MaxDrawdown     float64            `json:"max_drawdown"`
	MaxDrawdownPercent float64         `json:"max_drawdown_percent"`
	TotalReturn     float64            `json:"total_return"`
	TotalReturnPercent float64         `json:"total_return_percent"`
	Timestamp       time.Time          `json:"timestamp"`
}

// StatusResponse contains system status information
type StatusResponse struct {
	IsRunning       bool      `json:"is_running"`
	OpenPositions   int       `json:"open_positions"`
	AvailableCapital float64  `json:"available_capital"`
	TotalCapital    float64   `json:"total_capital"`
	DailyPnL        float64   `json:"daily_pnl"`
	Message         string    `json:"message"`
	Timestamp       time.Time `json:"timestamp"`
}

// ErrorResponse is returned when an error occurs
type ErrorResponse struct {
	Error   string    `json:"error"`
	Message string    `json:"message"`
	Code    int       `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

// CandleData represents a single OHLCV candlestick
type CandleData struct {
	Date   time.Time `json:"date"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume int64     `json:"volume"`
}

// StockSummary represents a stock in the list view with stats
type StockSummary struct {
	Symbol            string    `json:"symbol"`
	LatestDate        time.Time `json:"latest_date"`
	LatestClose       float64   `json:"latest_close"`
	HighPrice         float64   `json:"high_price"`
	LowPrice          float64   `json:"low_price"`
	PercentChange     float64   `json:"percent_change"`     // % change from first to latest
	AverageVolume     float64   `json:"average_volume"`
	WinningTradesCount int      `json:"winning_trades_count"`
	TotalPnL          float64   `json:"total_pnl"`
}

// StocksListResponse contains all stocks with their summaries
type StocksListResponse struct {
	Stocks    []StockSummary `json:"stocks"`
	Timestamp time.Time      `json:"timestamp"`
}

// StockCandlesResponse contains OHLCV data for a specific stock
type StockCandlesResponse struct {
	Symbol    string        `json:"symbol"`
	Candles   []CandleData  `json:"candles"`
	FromDate  time.Time     `json:"from_date"`
	ToDate    time.Time     `json:"to_date"`
	Timestamp time.Time     `json:"timestamp"`
}
