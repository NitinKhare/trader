// Package risk implements hard risk guardrails for the trading system.
//
// Design rules (from spec):
//   - Risk rules are implemented in Go.
//   - They CANNOT be overridden by strategy or AI.
//   - Every trade MUST have a stop loss.
//   - Capital preservation > returns.
//   - System must prefer not trading over bad trades.
package risk

import (
	"fmt"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// RejectionReason explains why a trade intent was rejected by risk management.
type RejectionReason struct {
	Rule    string
	Message string
}

func (r RejectionReason) Error() string {
	return fmt.Sprintf("risk rejected [%s]: %s", r.Rule, r.Message)
}

// ValidationResult holds the outcome of risk validation.
type ValidationResult struct {
	Approved  bool
	Intent    strategy.TradeIntent
	Rejections []RejectionReason
}

// DailyPnL tracks realized and unrealized P&L for the day.
type DailyPnL struct {
	Date         time.Time
	RealizedPnL  float64
	UnrealizedPnL float64
}

// Manager enforces all risk rules. It is the final gatekeeper before any order is placed.
// The risk manager is deliberately strict: it rejects intents that violate any rule,
// even if the strategy is confident.
type Manager struct {
	config         config.RiskConfig
	totalCapital   float64
}

// NewManager creates a new risk manager with the given configuration and capital.
func NewManager(riskCfg config.RiskConfig, totalCapital float64) *Manager {
	return &Manager{
		config:       riskCfg,
		totalCapital: totalCapital,
	}
}

// UpdateCapital updates the capital base used for percentage-based risk calculations.
// Called on each trading run with the live broker's total balance so that risk limits
// automatically adjust when money is added to or withdrawn from the account.
func (m *Manager) UpdateCapital(newCapital float64) {
	if newCapital > 0 {
		m.totalCapital = newCapital
	}
}

// UpdateRiskConfig replaces the risk configuration atomically.
// Used by config hot-reload to update risk params without restarting.
func (m *Manager) UpdateRiskConfig(newCfg config.RiskConfig) {
	m.config = newCfg
}

// Validate checks a TradeIntent against all risk rules.
// It takes the current state of open positions, daily P&L, and an optional sector map.
// sectorMap maps symbol → sector name. Pass nil to skip sector checks.
// Returns ValidationResult with approval status and any rejection reasons.
func (m *Manager) Validate(
	intent strategy.TradeIntent,
	openPositions []strategy.PositionInfo,
	dailyPnL DailyPnL,
	availableCapital float64,
	sectorMap map[string]string,
) ValidationResult {
	result := ValidationResult{
		Approved: true,
		Intent:   intent,
	}

	// SKIP and HOLD intents don't need risk validation.
	if intent.Action == strategy.ActionSkip || intent.Action == strategy.ActionHold {
		return result
	}

	// EXIT intents are always allowed — we want to be able to exit positions.
	if intent.Action == strategy.ActionExit {
		return result
	}

	// BUY intents go through all risk checks.
	if intent.Action == strategy.ActionBuy {
		m.checkStopLoss(&result, intent)
		m.checkMaxRiskPerTrade(&result, intent)
		m.checkMaxOpenPositions(&result, intent, openPositions)
		m.checkMaxDailyLoss(&result, dailyPnL)
		m.checkMaxCapitalDeployment(&result, intent, openPositions, availableCapital)
		m.checkPositionSize(&result, intent, availableCapital)
		m.checkSectorConcentration(&result, intent, openPositions, sectorMap)
	}

	return result
}

// checkStopLoss ensures every BUY intent has a mandatory stop loss.
func (m *Manager) checkStopLoss(result *ValidationResult, intent strategy.TradeIntent) {
	if intent.StopLoss <= 0 {
		m.reject(result, "MANDATORY_STOP_LOSS", "every trade must have a stop loss")
		return
	}
	if intent.StopLoss >= intent.Price {
		m.reject(result, "INVALID_STOP_LOSS", fmt.Sprintf(
			"stop loss %.2f must be below entry price %.2f", intent.StopLoss, intent.Price,
		))
	}
}

// checkMaxRiskPerTrade ensures the risk amount doesn't exceed the per-trade limit.
func (m *Manager) checkMaxRiskPerTrade(result *ValidationResult, intent strategy.TradeIntent) {
	riskPerShare := intent.Price - intent.StopLoss
	totalRisk := riskPerShare * float64(intent.Quantity)
	maxAllowedRisk := m.totalCapital * (m.config.MaxRiskPerTradePct / 100.0)

	if totalRisk > maxAllowedRisk {
		m.reject(result, "MAX_RISK_PER_TRADE", fmt.Sprintf(
			"trade risk %.2f exceeds max allowed %.2f (%.1f%% of %.2f)",
			totalRisk, maxAllowedRisk, m.config.MaxRiskPerTradePct, m.totalCapital,
		))
	}
}

// checkMaxOpenPositions ensures we don't exceed the position limit.
func (m *Manager) checkMaxOpenPositions(result *ValidationResult, intent strategy.TradeIntent, positions []strategy.PositionInfo) {
	// Check if we already have a position in this stock.
	for _, pos := range positions {
		if pos.Symbol == intent.Symbol {
			m.reject(result, "DUPLICATE_POSITION", fmt.Sprintf(
				"already have an open position in %s", intent.Symbol,
			))
			return
		}
	}

	if len(positions) >= m.config.MaxOpenPositions {
		m.reject(result, "MAX_OPEN_POSITIONS", fmt.Sprintf(
			"at position limit: %d/%d", len(positions), m.config.MaxOpenPositions,
		))
	}
}

// checkMaxDailyLoss ensures we haven't exceeded the daily loss limit.
func (m *Manager) checkMaxDailyLoss(result *ValidationResult, dailyPnL DailyPnL) {
	totalLoss := dailyPnL.RealizedPnL + dailyPnL.UnrealizedPnL
	maxDailyLoss := m.totalCapital * (m.config.MaxDailyLossPct / 100.0)

	if totalLoss < 0 && (-totalLoss) >= maxDailyLoss {
		m.reject(result, "MAX_DAILY_LOSS", fmt.Sprintf(
			"daily loss %.2f has reached limit %.2f", -totalLoss, maxDailyLoss,
		))
	}
}

// checkMaxCapitalDeployment ensures total deployed capital doesn't exceed the limit.
func (m *Manager) checkMaxCapitalDeployment(
	result *ValidationResult,
	intent strategy.TradeIntent,
	positions []strategy.PositionInfo,
	availableCapital float64,
) {
	// Calculate currently deployed capital.
	var deployedCapital float64
	for _, pos := range positions {
		deployedCapital += pos.EntryPrice * float64(pos.Quantity)
	}

	// Add the proposed trade.
	proposedTotal := deployedCapital + (intent.Price * float64(intent.Quantity))
	maxDeployment := m.totalCapital * (m.config.MaxCapitalDeploymentPct / 100.0)

	if proposedTotal > maxDeployment {
		m.reject(result, "MAX_CAPITAL_DEPLOYMENT", fmt.Sprintf(
			"total deployment %.2f would exceed limit %.2f (%.1f%% of %.2f)",
			proposedTotal, maxDeployment, m.config.MaxCapitalDeploymentPct, m.totalCapital,
		))
	}
}

// checkPositionSize ensures we can afford the trade.
func (m *Manager) checkPositionSize(result *ValidationResult, intent strategy.TradeIntent, availableCapital float64) {
	totalCost := intent.Price * float64(intent.Quantity)
	if totalCost > availableCapital {
		m.reject(result, "INSUFFICIENT_CAPITAL", fmt.Sprintf(
			"trade cost %.2f exceeds available capital %.2f", totalCost, availableCapital,
		))
	}
}

// checkSectorConcentration ensures we don't hold too many positions in the same sector.
// If sectorMap is nil or MaxPerSector is 0, this check is skipped (disabled).
func (m *Manager) checkSectorConcentration(result *ValidationResult, intent strategy.TradeIntent,
	positions []strategy.PositionInfo, sectorMap map[string]string) {
	if sectorMap == nil || m.config.MaxPerSector <= 0 {
		return // sector check disabled
	}

	intentSector, hasSector := sectorMap[intent.Symbol]
	if !hasSector {
		return // no sector info for this stock, skip check
	}

	// Count how many existing positions are in the same sector.
	sectorCount := 0
	for _, pos := range positions {
		if posSector, ok := sectorMap[pos.Symbol]; ok && posSector == intentSector {
			sectorCount++
		}
	}

	if sectorCount >= m.config.MaxPerSector {
		m.reject(result, "MAX_SECTOR_CONCENTRATION", fmt.Sprintf(
			"already have %d positions in sector %s (max %d)",
			sectorCount, intentSector, m.config.MaxPerSector,
		))
	}
}

func (m *Manager) reject(result *ValidationResult, rule, message string) {
	result.Approved = false
	result.Rejections = append(result.Rejections, RejectionReason{
		Rule:    rule,
		Message: message,
	})
}
