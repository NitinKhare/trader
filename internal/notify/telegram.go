// Package notify provides trade notification delivery.
// Currently supports Telegram via the Bot API.
// All sends are fire-and-forget (goroutine) so they never block trade execution.
package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const telegramAPI = "https://api.telegram.org/bot%s/sendMessage"

// Telegram sends trade notifications to a Telegram chat.
type Telegram struct {
	botToken string
	chatID   string
	client   *http.Client
	logger   *log.Logger
}

// NewTelegram creates a Telegram notifier.
// Pass logger=nil to suppress log output.
func NewTelegram(botToken, chatID string, logger *log.Logger) *Telegram {
	return &Telegram{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{Timeout: 10 * time.Second},
		logger:   logger,
	}
}

// NotifyBuy sends a BUY order filled notification.
func (t *Telegram) NotifyBuy(symbol, strategy string, qty int, fillPrice, sl, target float64, orderID string) {
	riskPct := 0.0
	if fillPrice > 0 && sl > 0 {
		riskPct = ((fillPrice - sl) / fillPrice) * 100
	}
	rewardPct := 0.0
	if fillPrice > 0 && target > 0 {
		rewardPct = ((target - fillPrice) / fillPrice) * 100
	}

	msg := fmt.Sprintf(
		"BUY FILLED\n"+
			"Symbol:    %s\n"+
			"Strategy:  %s\n"+
			"Qty:       %d\n"+
			"Price:     %.2f\n"+
			"Stop Loss: %.2f  (-%.2f%%)\n"+
			"Target:    %.2f  (+%.2f%%)\n"+
			"Order ID:  %s\n"+
			"Time:      %s",
		symbol, strategy, qty,
		fillPrice,
		sl, riskPct,
		target, rewardPct,
		orderID,
		time.Now().In(ist()).Format("02 Jan 15:04:05"),
	)
	go t.send(msg)
}

// NotifySell sends a SELL (exit) order placed notification.
func (t *Telegram) NotifySell(symbol, strategy, reason string, qty int, exitPrice, entryPrice float64, orderID string) {
	pnlPerShare := exitPrice - entryPrice
	pnlTotal := pnlPerShare * float64(qty)
	pnlPct := 0.0
	if entryPrice > 0 {
		pnlPct = (pnlPerShare / entryPrice) * 100
	}

	sign := "+"
	if pnlTotal < 0 {
		sign = ""
	}

	msg := fmt.Sprintf(
		"SELL ORDER PLACED\n"+
			"Symbol:    %s\n"+
			"Strategy:  %s\n"+
			"Reason:    %s\n"+
			"Qty:       %d\n"+
			"Exit:      %.2f\n"+
			"Entry:     %.2f\n"+
			"P&L:       %s%.2f  (%s%.2f%%)\n"+
			"Order ID:  %s\n"+
			"Time:      %s",
		symbol, strategy, reason, qty,
		exitPrice, entryPrice,
		sign, pnlTotal, sign, pnlPct,
		orderID,
		time.Now().In(ist()).Format("02 Jan 15:04:05"),
	)
	go t.send(msg)
}

// NotifySLTriggered sends a stop-loss hit notification.
func (t *Telegram) NotifySLTriggered(symbol string, qty int, exitPrice, entryPrice float64) {
	pnlPerShare := exitPrice - entryPrice
	pnlTotal := pnlPerShare * float64(qty)
	pnlPct := 0.0
	if entryPrice > 0 {
		pnlPct = (pnlPerShare / entryPrice) * 100
	}

	msg := fmt.Sprintf(
		"STOP LOSS TRIGGERED\n"+
			"Symbol:  %s\n"+
			"Qty:     %d\n"+
			"SL Hit:  %.2f\n"+
			"Entry:   %.2f\n"+
			"Loss:    %.2f  (%.2f%%)\n"+
			"Time:    %s",
		symbol, qty,
		exitPrice, entryPrice,
		pnlTotal, pnlPct,
		time.Now().In(ist()).Format("02 Jan 15:04:05"),
	)
	go t.send(msg)
}

// send delivers a message to the configured chat. Called in a goroutine.
func (t *Telegram) send(text string) {
	url := fmt.Sprintf(telegramAPI, t.botToken)

	body, _ := json.Marshal(map[string]string{
		"chat_id": t.chatID,
		"text":    text,
	})

	resp, err := t.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		if t.logger != nil {
			t.logger.Printf("[telegram] send failed: %v", err)
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if t.logger != nil {
			t.logger.Printf("[telegram] send returned HTTP %d", resp.StatusCode)
		}
	}
}

// ist returns IST timezone, falling back to UTC if not available.
func ist() *time.Location {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return time.UTC
	}
	return loc
}
