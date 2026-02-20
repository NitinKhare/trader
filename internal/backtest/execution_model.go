// Package backtest provides backtesting infrastructure including execution cost
// modeling, walk-forward analysis, and realistic trade simulation.
package backtest

import (
	"math"

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// ExecutionModelConfig holds all the parameters for realistic execution cost modeling.
// Defaults are set for Indian equity markets (NSE/BSE delivery trades).
type ExecutionModelConfig struct {
	BrokeragePerTrade  float64 // flat INR per trade (default 20)
	BrokeragePct       float64 // percentage of trade value (default 0.03)
	STTPct             float64 // Securities Transaction Tax on sell (default 0.1)
	ExchangeChargesPct float64 // NSE charges (default 0.00345)
	StampDutyPct       float64 // stamp duty (default 0.015)
	GSTPct             float64 // GST on brokerage (default 18.0)
	SlippageBps        float64 // basis points slippage (default 5)
	VolatilitySlippage bool    // if true, scale slippage with ATR
}

// AdjustedPrices holds the execution-adjusted entry/exit prices and cost breakdown.
type AdjustedPrices struct {
	EntryPrice   float64
	ExitPrice    float64
	TotalCosts   float64
	SlippageCost float64
}

// ExecutionModel applies realistic execution costs and slippage to trade prices.
type ExecutionModel struct {
	config ExecutionModelConfig
}

// NewExecutionModel creates an ExecutionModel with Indian market default costs.
func NewExecutionModel() *ExecutionModel {
	return &ExecutionModel{
		config: ExecutionModelConfig{
			BrokeragePerTrade:  20.0,
			BrokeragePct:       0.03,
			STTPct:             0.1,
			ExchangeChargesPct: 0.00345,
			StampDutyPct:       0.015,
			GSTPct:             18.0,
			SlippageBps:        5.0,
			VolatilitySlippage: false,
		},
	}
}

// Apply adjusts entry and exit prices for slippage and computes total transaction costs.
//
// Slippage: Entry price is adjusted UP (worse fill for buy), exit price is adjusted DOWN.
// If VolatilitySlippage is enabled, slippage is scaled by the ATR/price ratio.
//
// Costs include:
//   - Brokerage: max(flat per trade, pct * trade value) on both legs
//   - STT: applied only on sell side for delivery trades
//   - Exchange charges: on both legs
//   - Stamp duty: on buy side
//   - GST: 18% on total brokerage
func (em *ExecutionModel) Apply(entryPrice, exitPrice float64, quantity int, candles []strategy.Candle) AdjustedPrices {
	if quantity <= 0 || entryPrice <= 0 || exitPrice <= 0 {
		return AdjustedPrices{
			EntryPrice: entryPrice,
			ExitPrice:  exitPrice,
		}
	}

	cfg := em.config
	qty := float64(quantity)

	// --- Slippage ---
	slippageFactor := cfg.SlippageBps / 10000.0

	if cfg.VolatilitySlippage && len(candles) > 0 {
		// Scale slippage by ATR/price ratio.
		atr := strategy.CalculateATR(candles, 14)
		lastClose := candles[len(candles)-1].Close
		if lastClose > 0 && atr > 0 {
			atrRatio := atr / lastClose
			slippageFactor *= atrRatio / (cfg.SlippageBps / 10000.0)
			// Normalize: if ATR/price equals default slippage bps, factor stays the same.
			// Actually, scale it multiplicatively by ATR ratio.
			slippageFactor = (cfg.SlippageBps / 10000.0) * (atrRatio * 100) // atrRatio amplifies slippage
			if slippageFactor > 0.01 {
				slippageFactor = 0.01 // cap at 1%
			}
		}
	}

	adjustedEntry := entryPrice * (1 + slippageFactor)  // worse fill for buy
	adjustedExit := exitPrice * (1 - slippageFactor)     // worse fill for sell
	slippageCost := (adjustedEntry - entryPrice) * qty + (exitPrice - adjustedExit) * qty

	// --- Brokerage ---
	entryValue := adjustedEntry * qty
	exitValue := adjustedExit * qty

	entryBrokerage := math.Max(cfg.BrokeragePerTrade, entryValue*cfg.BrokeragePct/100.0)
	exitBrokerage := math.Max(cfg.BrokeragePerTrade, exitValue*cfg.BrokeragePct/100.0)
	totalBrokerage := entryBrokerage + exitBrokerage

	// --- STT (Securities Transaction Tax) - only on sell side for delivery ---
	stt := exitValue * cfg.STTPct / 100.0

	// --- Exchange charges - on both legs ---
	exchangeCharges := (entryValue + exitValue) * cfg.ExchangeChargesPct / 100.0

	// --- Stamp duty - on buy side ---
	stampDuty := entryValue * cfg.StampDutyPct / 100.0

	// --- GST on brokerage ---
	gst := totalBrokerage * cfg.GSTPct / 100.0

	// --- Total costs ---
	totalCosts := totalBrokerage + stt + exchangeCharges + stampDuty + gst + slippageCost

	return AdjustedPrices{
		EntryPrice:   adjustedEntry,
		ExitPrice:    adjustedExit,
		TotalCosts:   totalCosts,
		SlippageCost: slippageCost,
	}
}
