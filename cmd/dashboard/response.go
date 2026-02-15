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

// StrategyParameter describes a single parameter for a strategy
type StrategyParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // "float", "int", "bool", "string"
	DisplayName string      `json:"display_name"`
	Default     interface{} `json:"default"`
	Min         interface{} `json:"min,omitempty"`
	Max         interface{} `json:"max,omitempty"`
	Step        interface{} `json:"step,omitempty"`
	Description string      `json:"description,omitempty"`
}

// StrategyInfo describes an available trading strategy
type StrategyInfo struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Parameters  []StrategyParameter `json:"parameters"`
}

// StrategiesResponse contains all available strategies
type StrategiesResponse struct {
	Strategies []StrategyInfo `json:"strategies"`
	Timestamp  time.Time      `json:"timestamp"`
}

// BacktestRunRequest is the request body for starting a backtest
type BacktestRunRequest struct {
	Name      string                 `json:"name"`
	StrategyID string                 `json:"strategy_id"`
	Stocks    []string               `json:"stocks"` // nil means all stocks
	Parameters map[string]interface{} `json:"parameters"`
	DateFrom  string                 `json:"date_from"` // YYYY-MM-DD format
	DateTo    string                 `json:"date_to"`   // YYYY-MM-DD format
}

// BacktestRun represents a single backtest execution
type BacktestRun struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	StrategyID     string                 `json:"strategy_id"`
	Stocks         []string               `json:"stocks"`
	Parameters     map[string]interface{} `json:"parameters"`
	DateFrom       string                 `json:"date_from"`
	DateTo         string                 `json:"date_to"`
	Status         string                 `json:"status"` // PENDING, RUNNING, COMPLETED, FAILED
	ProgressPercent int                   `json:"progress_percent"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	StartedAt      *time.Time             `json:"started_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// BacktestResults contains performance metrics for a backtest
type BacktestResults struct {
	ID               string    `json:"id"`
	BacktestRunID    string    `json:"backtest_run_id"`
	TotalTrades      int       `json:"total_trades"`
	WinningTrades    int       `json:"winning_trades"`
	LosingTrades     int       `json:"losing_trades"`
	WinRate          float64   `json:"win_rate"`
	TotalPnL         float64   `json:"total_pnl"`
	PnLPercent       float64   `json:"pnl_percent"`
	ProfitFactor     float64   `json:"profit_factor"`
	SharpeRatio      float64   `json:"sharpe_ratio"`
	MaxDrawdown      float64   `json:"max_drawdown"`
	MaxDrawdownDate  string    `json:"max_drawdown_date"`
	AvgHoldDays      int       `json:"avg_hold_days"`
	BestTradePnL     float64   `json:"best_trade_pnl"`
	WorstTradePnL    float64   `json:"worst_trade_pnl"`
	CreatedAt        time.Time `json:"created_at"`
}

// BacktestTrade represents a single trade from a backtest
type BacktestTrade struct {
	ID            string    `json:"id"`
	BacktestRunID string    `json:"backtest_run_id"`
	Symbol        string    `json:"symbol"`
	EntryDate     string    `json:"entry_date"`
	ExitDate      string    `json:"exit_date"`
	EntryPrice    float64   `json:"entry_price"`
	ExitPrice     float64   `json:"exit_price"`
	Quantity      int       `json:"quantity"`
	PnL           float64   `json:"pnl"`
	PnLPercent    float64   `json:"pnl_percent"`
	ExitReason    string    `json:"exit_reason"`
	CreatedAt     time.Time `json:"created_at"`
}

// BacktestEquityCurvePoint represents equity progression during a backtest
type BacktestEquityCurvePoint struct {
	Date     string  `json:"date"`
	Equity   float64 `json:"equity"`
	Drawdown float64 `json:"drawdown"`
}

// BacktestRunResponse is returned when a backtest starts
type BacktestRunResponse struct {
	BacktestRunID string    `json:"backtest_run_id"`
	Status        string    `json:"status"`
	Message       string    `json:"message"`
	Timestamp     time.Time `json:"timestamp"`
}

// BacktestDetailResponse contains full backtest data including results and trades
type BacktestDetailResponse struct {
	BacktestRun  BacktestRun                  `json:"backtest_run"`
	Results      BacktestResults              `json:"results"`
	Trades       []BacktestTrade              `json:"trades"`
	EquityCurve  []BacktestEquityCurvePoint   `json:"equity_curve"`
	Timestamp    time.Time                    `json:"timestamp"`
}

// BacktestListResponse contains a list of backtest runs
type BacktestListResponse struct {
	Runs       []BacktestRun `json:"runs"`
	TotalCount int           `json:"total_count"`
	Limit      int           `json:"limit"`
	Offset     int           `json:"offset"`
	Timestamp  time.Time     `json:"timestamp"`
}

// BacktestComparisonMetric represents metrics for one run in a comparison
type BacktestComparisonMetric struct {
	BacktestRunID string  `json:"backtest_run_id"`
	Name          string  `json:"name"`
	StrategyID    string  `json:"strategy_id"`
	TotalTrades   int     `json:"total_trades"`
	WinRate       float64 `json:"win_rate"`
	TotalPnL      float64 `json:"total_pnl"`
	SharpeRatio   float64 `json:"sharpe_ratio"`
	MaxDrawdown   float64 `json:"max_drawdown"`
	ProfitFactor  float64 `json:"profit_factor"`
	CreatedAt     time.Time `json:"created_at"`
}

// BacktestComparisonResponse contains comparison data for multiple backtests
type BacktestComparisonResponse struct {
	Comparison []BacktestComparisonMetric  `json:"comparison"`
	BestBy     map[string]string           `json:"best_by"` // metric -> run_id
	Timestamp  time.Time                   `json:"timestamp"`
}
