package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nitinkhare/algoTradingAgent/internal/analytics"
	"github.com/nitinkhare/algoTradingAgent/internal/dashboard"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// In production, restrict to specific domains
		return true
	},
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer ws.Close()

	// Create client
	client := &dashboard.Client{
		ID:   r.RemoteAddr,
		Send: make(chan interface{}, 256),
	}

	// Register client with broadcaster
	s.broadcaster.Register(client)
	defer s.broadcaster.Unregister(client)

	s.logger.Printf("websocket: client connected from %s", client.ID)

	// Start goroutine to handle sending messages to this client
	go s.writePump(ws, client)

	// Read from client (handles ping/pong and disconnection detection)
	s.readPump(ws, client)
}

// writePump handles sending messages to a WebSocket client
func (s *Server) writePump(ws *websocket.Conn, client *dashboard.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		ws.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel is closed, close connection
				ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Write message as JSON
			if err := ws.WriteJSON(message); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					s.logger.Printf("websocket write error for %s: %v", client.ID, err)
				}
				return
			}

		case <-ticker.C:
			ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump handles receiving messages from WebSocket client
func (s *Server) readPump(ws *websocket.Conn, client *dashboard.Client) {
	defer func() {
		s.broadcaster.Unregister(client)
		s.logger.Printf("websocket: client disconnected from %s", client.ID)
	}()

	ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		messageType, _, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Printf("websocket read error for %s: %v", client.ID, err)
			}
			return
		}

		// We only expect ping/pong frames, not text messages
		// but we can handle them if needed
		if messageType == websocket.TextMessage {
			s.logger.Printf("websocket: received text message from %s", client.ID)
		}
	}
}

// broadcastMetrics sends updated metrics to all connected WebSocket clients
func (s *Server) broadcastMetrics(ctx context.Context) error {
	trades, err := s.store.GetAllClosedTrades(ctx)
	if err != nil {
		return err
	}

	// Fetch open trades for position data
	openTrades, err := s.store.GetOpenTrades(ctx)
	if err != nil {
		return err
	}

	// Calculate metrics
	var resp interface{}

	if len(trades) == 0 {
		// No trades yet
		resp = dashboard.WebSocketMessage{
			Type: "metrics",
			Data: MetricsResponse{
				TotalPnL:       0,
				WinRate:        0,
				ProfitFactor:   0,
				TotalTrades:    0,
				InitialCapital: s.cfg.Capital,
				FinalCapital:   s.cfg.Capital,
				Timestamp:      time.Now(),
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
	} else {
		// Calculate metrics from trades
		report := analytics.Analyze(trades, s.cfg.Capital)
		metricsResp := MetricsResponse{
			TotalPnL:        report.TotalPnL,
			TotalPnLPercent: (report.TotalPnL / s.cfg.Capital) * 100,
			WinRate:         report.WinRate,
			ProfitFactor:    report.ProfitFactor,
			Drawdown:        report.MaxDrawdown,
			DrawdownPercent: report.MaxDrawdownPct,
			SharpeRatio:     report.SharpeRatio,
			TotalTrades:     report.TotalTrades,
			WinningTrades:   report.WinningTrades,
			LosingTrades:    report.LosingTrades,
			AvgPnL:          report.AveragePnL,
			GrossProfit:     report.GrossProfit,
			GrossLoss:       report.GrossLoss,
			AvgHoldDays:     report.AverageHoldDays,
			InitialCapital:  s.cfg.Capital,
			FinalCapital:    s.cfg.Capital + report.TotalPnL,
			Timestamp:       time.Now(),
		}

		// Add position count to data
		positionData := map[string]interface{}{
			"metrics":             metricsResp,
			"open_position_count": len(openTrades),
		}

		resp = dashboard.WebSocketMessage{
			Type:      "metrics",
			Data:      positionData,
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	s.broadcaster.Broadcast(resp)
	return nil
}

// startPeriodicBroadcast sends periodic updates to all connected WebSocket clients
func (s *Server) startPeriodicBroadcast(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.broadcastMetrics(ctx); err != nil {
				s.logger.Printf("failed to broadcast metrics: %v", err)
			}

		case <-ctx.Done():
			return
		}
	}
}
