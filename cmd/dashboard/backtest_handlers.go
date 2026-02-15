package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// generateUUID creates a simple UUID for backtest runs
func generateUUID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), fmt.Sprintf("%X", time.Now().Unix()))
}

// handleBacktestRun handles POST /api/backtest/run - submits a backtest job
func (s *Server) handleBacktestRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req BacktestRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if req.StrategyID == "" {
		s.respondError(w, http.StatusBadRequest, "strategy_id is required")
		return
	}
	if req.Name == "" {
		req.Name = req.StrategyID + " - " + time.Now().Format("2006-01-02 15:04:05")
	}
	if req.DateFrom == "" || req.DateTo == "" {
		s.respondError(w, http.StatusBadRequest, "date_from and date_to are required")
		return
	}

	// Create backtest run record
	runID := generateUUID()
	_ = r.Context() // ctx is available if needed

	// TODO: Save to database and trigger async backtest execution
	// For now, return success response

	resp := BacktestRunResponse{
		BacktestRunID: runID,
		Status:        "PENDING",
		Message:       "Backtest queued for execution",
		Timestamp:     time.Now(),
	}

	s.respondJSON(w, http.StatusAccepted, resp)
}

// handleBacktestStrategies handles GET /api/backtest/strategies - returns available strategies
func (s *Server) handleBacktestStrategies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Define available strategies with their parameters
	// TODO: Extract from actual strategy implementations
	strategies := []StrategyInfo{
		{
			ID:          "trend_follow_v1",
			Name:        "Trend Following Strategy",
			Description: "Follows market trends using momentum indicators",
			Parameters: []StrategyParameter{
				{
					Name:        "trend_strength_threshold",
					Type:        "float",
					DisplayName: "Trend Strength Threshold",
					Default:     0.4,
					Min:         0.1,
					Max:         0.9,
					Step:        0.1,
				},
				{
					Name:        "risk_reward_ratio",
					Type:        "float",
					DisplayName: "Risk/Reward Ratio",
					Default:     2.5,
					Min:         1.0,
					Max:         5.0,
					Step:        0.5,
				},
			},
		},
		{
			ID:          "mean_reversion_v1",
			Name:        "Mean Reversion Strategy",
			Description: "Trades mean reversion patterns",
			Parameters: []StrategyParameter{
				{
					Name:        "zscore_threshold",
					Type:        "float",
					DisplayName: "Z-Score Threshold",
					Default:     2.0,
					Min:         1.0,
					Max:         3.0,
					Step:        0.5,
				},
			},
		},
		{
			ID:          "breakout_v1",
			Name:        "Breakout Strategy",
			Description: "Trades breakout patterns",
			Parameters: []StrategyParameter{
				{
					Name:        "lookback_period",
					Type:        "int",
					DisplayName: "Lookback Period (days)",
					Default:     20,
					Min:         5,
					Max:         100,
					Step:        5,
				},
			},
		},
		{
			ID:          "momentum_v1",
			Name:        "Momentum Strategy",
			Description: "Trades based on momentum indicators",
			Parameters: []StrategyParameter{
				{
					Name:        "momentum_threshold",
					Type:        "float",
					DisplayName: "Momentum Threshold",
					Default:     0.5,
					Min:         0.1,
					Max:         1.0,
					Step:        0.1,
				},
			},
		},
		{
			ID:          "vwap_v1",
			Name:        "VWAP Reversion Strategy",
			Description: "Reverts to volume weighted average price",
			Parameters: []StrategyParameter{
				{
					Name:        "deviation_threshold",
					Type:        "float",
					DisplayName: "Deviation from VWAP %",
					Default:     3.0,
					Min:         1.0,
					Max:         10.0,
					Step:        0.5,
				},
			},
		},
		{
			ID:          "pullback_v1",
			Name:        "Pullback Strategy",
			Description: "Trades pullbacks in trending markets",
			Parameters: []StrategyParameter{
				{
					Name:        "pullback_threshold",
					Type:        "float",
					DisplayName: "Pullback %",
					Default:     2.0,
					Min:         0.5,
					Max:         5.0,
					Step:        0.5,
				},
			},
		},
		{
			ID:          "orb_v1",
			Name:        "Opening Range Breakout",
			Description: "Breaks out of opening range",
			Parameters: []StrategyParameter{
				{
					Name:        "orb_minutes",
					Type:        "int",
					DisplayName: "Opening Range Minutes",
					Default:     30,
					Min:         15,
					Max:         120,
					Step:        15,
				},
			},
		},
		{
			ID:          "macd_crossover_v1",
			Name:        "MACD Crossover Strategy",
			Description: "Trades MACD signal line crossovers",
			Parameters: []StrategyParameter{
				{
					Name:        "fast_period",
					Type:        "int",
					DisplayName: "MACD Fast Period",
					Default:     12,
					Min:         6,
					Max:         24,
					Step:        1,
				},
				{
					Name:        "slow_period",
					Type:        "int",
					DisplayName: "MACD Slow Period",
					Default:     26,
					Min:         13,
					Max:         52,
					Step:        1,
				},
			},
		},
		{
			ID:          "bollinger_squeeze_v1",
			Name:        "Bollinger Squeeze Strategy",
			Description: "Trades Bollinger Band squeeze breakouts",
			Parameters: []StrategyParameter{
				{
					Name:        "bb_period",
					Type:        "int",
					DisplayName: "Bollinger Period",
					Default:     20,
					Min:         10,
					Max:         50,
					Step:        5,
				},
				{
					Name:        "bb_multiplier",
					Type:        "float",
					DisplayName: "Bollinger Multiplier",
					Default:     2.0,
					Min:         1.0,
					Max:         3.0,
					Step:        0.5,
				},
			},
		},
	}

	resp := StrategiesResponse{
		Strategies: strategies,
		Timestamp:  time.Now(),
	}

	s.respondJSON(w, http.StatusOK, resp)
}

// handleBacktestRuns handles GET /api/backtest/runs - returns list of backtest runs
func (s *Server) handleBacktestRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// TODO: Fetch from database with pagination and filters
	// For now, return empty list
	resp := BacktestListResponse{
		Runs:       []BacktestRun{},
		TotalCount: 0,
		Limit:      20,
		Offset:     0,
		Timestamp:  time.Now(),
	}

	s.respondJSON(w, http.StatusOK, resp)
}

// handleBacktestResults handles GET /api/backtest/results/:id - returns full backtest results
func (s *Server) handleBacktestResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract backtest run ID from URL path
	// Path is /api/backtest/results/{id}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		s.respondError(w, http.StatusBadRequest, "backtest run ID is required")
		return
	}

	runID := pathParts[4]
	if runID == "" {
		s.respondError(w, http.StatusBadRequest, "backtest run ID is required")
		return
	}

	// TODO: Fetch from database
	// For now, return empty response
	resp := BacktestDetailResponse{
		BacktestRun: BacktestRun{
			ID: runID,
		},
		Results:     BacktestResults{},
		Trades:      []BacktestTrade{},
		EquityCurve: []BacktestEquityCurvePoint{},
		Timestamp:   time.Now(),
	}

	s.respondJSON(w, http.StatusOK, resp)
}

// handleBacktestCompare handles POST /api/backtest/results/compare - compares multiple backtests
func (s *Server) handleBacktestCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		BacktestRunIDs []string `json:"backtest_run_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.BacktestRunIDs) == 0 {
		s.respondError(w, http.StatusBadRequest, "backtest_run_ids is required")
		return
	}

	// TODO: Fetch from database and compare
	// For now, return empty response
	resp := BacktestComparisonResponse{
		Comparison: []BacktestComparisonMetric{},
		BestBy:     make(map[string]string),
		Timestamp:  time.Now(),
	}

	s.respondJSON(w, http.StatusOK, resp)
}
