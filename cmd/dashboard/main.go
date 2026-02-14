package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/analytics"
	"github.com/nitinkhare/algoTradingAgent/internal/config"
	"github.com/nitinkhare/algoTradingAgent/internal/dashboard"
	"github.com/nitinkhare/algoTradingAgent/internal/storage"
)

// Server holds all dependencies for the dashboard API
type Server struct {
	store       storage.Store
	cfg         *config.Config
	logger      *log.Logger
	port        string
	broadcaster *dashboard.Broadcaster
	listener    *dashboard.EventListener
	cancelCtx   context.CancelFunc
}

func main() {
	// Parse flags
	configPath := flag.String("config", "config/config.json", "Path to config file")
	port := flag.String("port", "8081", "Dashboard server port")
	flag.Parse()

	// Setup logger
	logger := log.New(os.Stdout, "[dashboard] ", log.LstdFlags|log.Lshortfile)

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatalf("failed to load config: %v", err)
	}

	// Connect to database
	store, err := storage.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		logger.Fatalf("failed to connect to database: %v", err)
	}
	defer store.Close()

	// Create broadcaster for real-time updates
	broadcaster := dashboard.NewBroadcaster(logger)

	// Create event listener for database notifications
	eventListener := dashboard.NewEventListener(cfg.DatabaseURL, broadcaster, logger)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Create server
	server := &Server{
		store:       store,
		cfg:         cfg,
		logger:      logger,
		port:        *port,
		broadcaster: broadcaster,
		listener:    eventListener,
		cancelCtx:   cancel,
	}

	// Start broadcaster in goroutine
	go broadcaster.Run()
	logger.Println("broadcaster: started")

	// Start event listener in goroutine
	eventListener.Start(ctx)
	logger.Println("event listener: started")

	// Start periodic broadcast of metrics
	go server.startPeriodicBroadcast(ctx)
	logger.Println("periodic broadcast: started")

	// Setup routes
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/metrics", server.handleMetrics)
	mux.HandleFunc("/api/positions/open", server.handlePositionsOpen)
	mux.HandleFunc("/api/charts/equity", server.handleChartsEquity)
	mux.HandleFunc("/api/status", server.handleStatus)
	mux.HandleFunc("/api/stocks/list", server.handleStocksList)
	mux.HandleFunc("/api/stocks/candles", server.handleStockCandles)
	mux.HandleFunc("/health", server.handleHealth)

	// WebSocket endpoint for real-time updates
	mux.HandleFunc("/ws", server.handleWebSocket)

	// Setup HTTP server
	httpServer := &http.Server{
		Addr:         ":" + *port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		server.logger.Printf("dashboard API starting on port %s", *port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			server.logger.Fatalf("server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	server.logger.Println("shutting down dashboard server...")

	// Cancel context to stop all goroutines
	cancel()
	time.Sleep(100 * time.Millisecond) // Allow goroutines to finish

	// Stop event listener
	eventListener.Stop()
	time.Sleep(100 * time.Millisecond)

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		server.logger.Printf("shutdown error: %v", err)
	}

	// Close broadcaster
	broadcaster.Shutdown()

	server.logger.Println("dashboard server stopped")
}

// handleMetrics returns current performance metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// Get all closed trades
	trades, err := s.store.GetAllClosedTrades(ctx)
	if err != nil {
		s.logger.Printf("failed to get trades: %v", err)
		s.respondError(w, http.StatusInternalServerError, "failed to fetch trades")
		return
	}

	// If no trades, return empty metrics
	if len(trades) == 0 {
		resp := MetricsResponse{
			TotalPnL:       0,
			WinRate:        0,
			ProfitFactor:   0,
			TotalTrades:    0,
			InitialCapital: s.cfg.Capital,
			FinalCapital:   s.cfg.Capital,
			Timestamp:      time.Now(),
		}
		s.respondJSON(w, http.StatusOK, resp)
		return
	}

	// Analyze trades
	report := analytics.Analyze(trades, s.cfg.Capital)

	resp := MetricsResponse{
		TotalPnL:         report.TotalPnL,
		TotalPnLPercent:  (report.TotalPnL / s.cfg.Capital) * 100,
		WinRate:          report.WinRate,
		ProfitFactor:     report.ProfitFactor,
		Drawdown:         report.MaxDrawdown,
		DrawdownPercent:  report.MaxDrawdownPct,
		SharpeRatio:      report.SharpeRatio,
		TotalTrades:      report.TotalTrades,
		WinningTrades:    report.WinningTrades,
		LosingTrades:     report.LosingTrades,
		AvgPnL:           report.AveragePnL,
		GrossProfit:      report.GrossProfit,
		GrossLoss:        report.GrossLoss,
		AvgHoldDays:      report.AverageHoldDays,
		InitialCapital:   s.cfg.Capital,
		FinalCapital:     s.cfg.Capital + report.TotalPnL,
		Timestamp:        time.Now(),
	}

	s.respondJSON(w, http.StatusOK, resp)
}

// handlePositionsOpen returns all open positions
func (s *Server) handlePositionsOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// Get open trades
	openTrades, err := s.store.GetOpenTrades(ctx)
	if err != nil {
		s.logger.Printf("failed to get open trades: %v", err)
		s.respondError(w, http.StatusInternalServerError, "failed to fetch positions")
		return
	}

	positions := make([]PositionResponse, 0)
	totalCapitalUsed := 0.0

	for _, trade := range openTrades {
		pos := PositionResponse{
			ID:         trade.ID,
			Symbol:     trade.Symbol,
			Quantity:   trade.Quantity,
			EntryPrice: trade.EntryPrice,
			EntryTime:  trade.EntryTime,
			StopLoss:   trade.StopLoss,
			Target:     trade.Target,
			StrategyID: trade.StrategyID,
		}

		// Calculate capital used for this position
		capitalUsed := trade.EntryPrice * float64(trade.Quantity)
		totalCapitalUsed += capitalUsed

		positions = append(positions, pos)
	}

	// Calculate availability
	availableCapital := s.cfg.Capital - totalCapitalUsed
	if availableCapital < 0 {
		availableCapital = 0
	}

	utilizationPercent := 0.0
	if s.cfg.Capital > 0 {
		utilizationPercent = (totalCapitalUsed / s.cfg.Capital) * 100
	}

	resp := PositionsResponse{
		Positions:                positions,
		TotalCapitalUsed:         totalCapitalUsed,
		AvailableCapital:         availableCapital,
		CapitalUtilizationPercent: utilizationPercent,
		OpenPositionCount:        len(openTrades),
		Timestamp:                time.Now(),
	}

	s.respondJSON(w, http.StatusOK, resp)
}

// handleChartsEquity returns equity curve for charting
func (s *Server) handleChartsEquity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// Get all closed trades
	trades, err := s.store.GetAllClosedTrades(ctx)
	if err != nil {
		s.logger.Printf("failed to get trades: %v", err)
		s.respondError(w, http.StatusInternalServerError, "failed to fetch trades")
		return
	}

	// If no trades, return empty equity curve
	if len(trades) == 0 {
		resp := EquityCurveResponse{
			Points:         make([]EquityCurvePoint, 0),
			StartEquity:    s.cfg.Capital,
			FinalEquity:    s.cfg.Capital,
			Timestamp:      time.Now(),
		}
		s.respondJSON(w, http.StatusOK, resp)
		return
	}

	// Generate equity curve
	analyticsCurve := analytics.EquityCurve(trades, s.cfg.Capital)

	// Convert to response format
	points := make([]EquityCurvePoint, len(analyticsCurve))
	maxDrawdown := 0.0
	maxDrawdownPercent := 0.0

	for i, p := range analyticsCurve {
		ddPercent := 0.0
		if s.cfg.Capital > 0 {
			ddPercent = (p.Drawdown / s.cfg.Capital) * 100
		}

		points[i] = EquityCurvePoint{
			Date:            p.Date,
			Equity:          p.Equity,
			Drawdown:        p.Drawdown,
			DrawdownPercent: ddPercent,
		}

		// Track max drawdown
		if p.Drawdown > maxDrawdown {
			maxDrawdown = p.Drawdown
			maxDrawdownPercent = ddPercent
		}
	}

	// Calculate totals
	finalEquity := analyticsCurve[len(analyticsCurve)-1].Equity
	totalReturn := finalEquity - s.cfg.Capital
	totalReturnPercent := (totalReturn / s.cfg.Capital) * 100

	resp := EquityCurveResponse{
		Points:             points,
		StartEquity:        s.cfg.Capital,
		FinalEquity:        finalEquity,
		MaxDrawdown:        maxDrawdown,
		MaxDrawdownPercent: maxDrawdownPercent,
		TotalReturn:        totalReturn,
		TotalReturnPercent: totalReturnPercent,
		Timestamp:          time.Now(),
	}

	s.respondJSON(w, http.StatusOK, resp)
}

// handleStatus returns system status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// Get open trades
	openTrades, err := s.store.GetOpenTrades(ctx)
	if err != nil {
		s.logger.Printf("failed to get open trades: %v", err)
		s.respondError(w, http.StatusInternalServerError, "failed to fetch status")
		return
	}

	// Calculate capital used
	totalCapitalUsed := 0.0
	for _, trade := range openTrades {
		totalCapitalUsed += trade.EntryPrice * float64(trade.Quantity)
	}

	// Get daily P&L
	today := time.Now().Truncate(24 * time.Hour)
	dailyPnL, _ := s.store.GetDailyPnL(ctx, today)

	availableCapital := s.cfg.Capital - totalCapitalUsed
	if availableCapital < 0 {
		availableCapital = 0
	}

	resp := StatusResponse{
		IsRunning:        true,
		OpenPositions:    len(openTrades),
		AvailableCapital: availableCapital,
		TotalCapital:     s.cfg.Capital,
		DailyPnL:         dailyPnL,
		Message:          fmt.Sprintf("%d positions open, ₹%.2f available", len(openTrades), availableCapital),
		Timestamp:        time.Now(),
	}

	s.respondJSON(w, http.StatusOK, resp)
}

// handleStocksList returns all unique stocks with summaries
func (s *Server) handleStocksList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// Get all closed trades to calculate per-stock statistics
	trades, err := s.store.GetAllClosedTrades(ctx)
	if err != nil {
		s.logger.Printf("failed to get trades: %v", err)
		s.respondError(w, http.StatusInternalServerError, "failed to fetch trades")
		return
	}

	// Group trades by symbol
	symbolTradeStats := make(map[string]struct {
		winCount int
		totalPnL float64
	})

	for _, trade := range trades {
		stats := symbolTradeStats[trade.Symbol]
		stats.totalPnL += trade.PnL
		if trade.PnL > 0 {
			stats.winCount++
		}
		symbolTradeStats[trade.Symbol] = stats
	}

	// Get all unique symbols and their latest candle data
	// We'll get this by querying the candles table for distinct symbols
	// For now, we'll extract unique symbols from trades if they exist
	symbolMap := make(map[string]bool)
	for _, trade := range trades {
		symbolMap[trade.Symbol] = true
	}

	stocks := make([]StockSummary, 0)
	now := time.Now()
	oneYearAgo := now.AddDate(-1, 0, 0)

	for symbol := range symbolMap {
		// Get candles for this symbol
		candles, err := s.store.GetCandles(ctx, symbol, oneYearAgo, now)
		if err != nil || len(candles) == 0 {
			continue
		}

		// Calculate stats from candles
		high := candles[0].High
		low := candles[0].Low
		totalVolume := int64(0)

		for _, candle := range candles {
			if candle.High > high {
				high = candle.High
			}
			if candle.Low < low {
				low = candle.Low
			}
			totalVolume += int64(candle.Volume)
		}

		percentChange := 0.0
		if candles[0].Close > 0 {
			percentChange = ((candles[len(candles)-1].Close - candles[0].Close) / candles[0].Close) * 100
		}

		avgVolume := float64(totalVolume) / float64(len(candles))
		tradeStats := symbolTradeStats[symbol]

		stocks = append(stocks, StockSummary{
			Symbol:            symbol,
			LatestDate:        candles[len(candles)-1].Date,
			LatestClose:       candles[len(candles)-1].Close,
			HighPrice:         high,
			LowPrice:          low,
			PercentChange:     percentChange,
			AverageVolume:     avgVolume,
			WinningTradesCount: tradeStats.winCount,
			TotalPnL:          tradeStats.totalPnL,
		})
	}

	resp := StocksListResponse{
		Stocks:    stocks,
		Timestamp: time.Now(),
	}

	s.respondJSON(w, http.StatusOK, resp)
}

// handleStockCandles returns OHLCV data for a specific stock
func (s *Server) handleStockCandles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse query parameters
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		s.respondError(w, http.StatusBadRequest, "symbol query parameter required")
		return
	}

	// Parse date range
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	now := time.Now()
	from := now.AddDate(-1, 0, 0) // Default: last 1 year
	to := now

	if fromStr != "" {
		parsedFrom, err := time.Parse("2006-01-02", fromStr)
		if err == nil {
			from = parsedFrom
		}
	}

	if toStr != "" {
		parsedTo, err := time.Parse("2006-01-02", toStr)
		if err == nil {
			to = parsedTo
		}
	}

	ctx := r.Context()

	// Get candles from database
	candles, err := s.store.GetCandles(ctx, symbol, from, to)
	if err != nil {
		s.logger.Printf("failed to get candles for %s: %v", symbol, err)
		s.respondError(w, http.StatusInternalServerError, "failed to fetch candles")
		return
	}

	// Convert to CandleData
	candleData := make([]CandleData, len(candles))
	for i, candle := range candles {
		candleData[i] = CandleData{
			Date:   candle.Date,
			Open:   candle.Open,
			High:   candle.High,
			Low:    candle.Low,
			Close:  candle.Close,
			Volume: candle.Volume,
		}
	}

	resp := StockCandlesResponse{
		Symbol:    symbol,
		Candles:   candleData,
		FromDate:  from,
		ToDate:    to,
		Timestamp: time.Now(),
	}

	s.respondJSON(w, http.StatusOK, resp)
}

// handleHealth returns health check
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Helper methods

func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	resp := ErrorResponse{
		Error:     http.StatusText(status),
		Message:   message,
		Code:      status,
		Timestamp: time.Now(),
	}
	s.respondJSON(w, status, resp)
}
