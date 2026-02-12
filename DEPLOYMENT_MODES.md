# Deployment Modes Guide

Your trading engine supports multiple modes for different purposes. Here's how to use each one.

## Mode Overview

| Mode | Config | Purpose | Real Money | Real Orders | Best For |
|------|--------|---------|-----------|------------|----------|
| **status** | any | Check system status | ❌ No | ❌ No | Diagnostics |
| **backtest** | paper | Historical backtesting | ❌ No | ❌ No | Strategy validation |
| **dry-run** | any | Simulate live without orders | ❌ No | ❌ No | Testing logic |
| **paper** | paper | Paper trading (simulated) | ❌ No | ❌ No | Safe testing |
| **market** | paper | Live market data with paper orders | ⚠️ Real data | ❌ No | Learning/monitoring |
| **nightly** | any | Overnight batch processing | Depends | Depends | Scheduled jobs |
| **live** | live | Real trading | ✅ Yes | ✅ Yes | Production trading |

## Configuration Changes Needed

Before switching modes, you may need to update `config/config.json`:

### For Paper Mode (Safe - Recommended First)
```json
{
  "trading_mode": "paper",
  "risk": {
    "max_capital_deployment_pct": 80.0
  }
}
```

Run:
```bash
go run ./cmd/engine --mode market
```

### For Live Mode (Real Trading - Requires Extra Confirmations)
```json
{
  "trading_mode": "live",
  "risk": {
    "max_capital_deployment_pct": 70.0
  }
}
```

Run:
```bash
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live
```

## Mode Details

### 1. Status Mode
**Purpose:** Check system health and configuration

```bash
go run ./cmd/engine --mode status
```

**Output:** Shows config, strategies loaded, database status

### 2. Backtest Mode
**Purpose:** Historical backtesting against past data

```bash
go run ./cmd/engine --mode backtest
```

**Requirements:**
- AI scores in `ai_outputs/` for multiple dates
- Market data in `market_data/`

**Use Case:** Validate strategy performance before trading

### 3. Dry-Run Mode
**Purpose:** Simulate trading logic without placing real or paper orders

```bash
go run ./cmd/engine --mode dry-run
```

**Use Case:** Test entry/exit logic, position management, etc.

### 4. Paper Mode
**Purpose:** Safe simulated trading with fake capital

```bash
go run ./cmd/engine --mode paper
```

**Config:**
```json
"trading_mode": "paper",
"capital": 500000.00
```

**Behavior:**
- No real money involved
- No orders sent to broker
- Simulates fills at current market prices
- Logs all trades to database
- Perfect for learning and testing strategies

**Use Case:** Test strategies in live market conditions without risk

### 5. Market Mode (Paper)
**Purpose:** Connect to live market data with paper trading

```bash
go run ./cmd/engine --mode market
```

**With Paper Config:**
```json
"trading_mode": "paper"
```

**Behavior:**
- Fetches real-time market data
- Places paper (simulated) orders
- Useful for monitoring live market performance of strategies
- Log and analyze what would happen in real trading

**Use Case:** Monitor strategy performance in live markets before going live

### 6. Market Mode (Live)
**Purpose:** REAL TRADING - Place actual orders with real money

```bash
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live
```

**With Live Config:**
```json
"trading_mode": "live",
"max_capital_deployment_pct": 70.0
```

**Requirements:**
1. Set `trading_mode: "live"` in config.json
2. Reduce `max_capital_deployment_pct` to ≤ 70%
3. Provide TWO explicit confirmations:
   - CLI flag: `--confirm-live`
   - Environment variable: `ALGO_LIVE_CONFIRMED=true`

**Behavior:**
- Places REAL orders on the exchange
- Uses REAL capital
- REAL money is at risk
- Strict risk limits enforced
- All trades logged for audit

**⚠️ WARNING: Only use after thorough testing!**

### 7. Nightly Mode
**Purpose:** Overnight batch processing (scoring, reporting, etc.)

```bash
go run ./cmd/engine --mode nightly
```

**Use Case:** Run after market close for end-of-day processing

## Recommended Progression

### Week 1: Learning
```bash
# 1. Check system status
go run ./cmd/engine --mode status

# 2. Run backtest on historical data
go run ./cmd/engine --mode backtest

# 3. Dry-run to test logic
go run ./cmd/engine --mode dry-run
```

### Week 2: Paper Trading
```bash
# Update config.json
# trading_mode: "paper"

# Run in paper mode during market hours
go run ./cmd/engine --mode market

# Monitor results
# Compare with backtest
```

### Week 3: Paper Market Mode (if confident)
```bash
# Keep config.json as paper
# trading_mode: "paper"

# Run live market data with paper orders
go run ./cmd/engine --mode market

# Monitor results in live conditions
# Ensure paper matches backtest expectations
```

### Week 4+: Live Trading (if results are good)
```bash
# Update config.json for live
# trading_mode: "live"
# max_capital_deployment_pct: 70.0

# Run live trading (requires extra confirmations)
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live

# Monitor VERY carefully
# Be ready to stop if something goes wrong
```

## Current Configuration

Your config is set up for:

```json
{
  "trading_mode": "paper",
  "capital": 500000.00,
  "risk": {
    "max_risk_per_trade_pct": 1.0,
    "max_open_positions": 5,
    "max_daily_loss_pct": 3.0,
    "max_capital_deployment_pct": 70.0
  }
}
```

**Current Status:** Paper trading mode active

**To run paper trading:**
```bash
go run ./cmd/engine --mode market
```

**To switch to live (only when ready):**
```bash
# 1. Update config.json:
#    "trading_mode": "live"

# 2. Run with confirmations:
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live
```

## Safety Features

### Paper Mode Protections
✅ No real orders sent
✅ No real capital at risk
✅ Full simulation of trading
✅ Can test indefinitely

### Live Mode Protections
✅ Requires explicit confirmations (2 levels)
✅ Enforces stricter risk limits
✅ Max 70% capital deployment
✅ Max 1% risk per trade
✅ Max 3% daily loss
✅ Max 5 open positions
✅ All trades audited
✅ Circuit breaker stops trading on repeated failures

## Quick Command Reference

```bash
# Check status
go run ./cmd/engine --mode status

# Backtest
go run ./cmd/engine --mode backtest

# Paper trading (simulated)
go run ./cmd/engine --mode market

# Live trading (REAL MONEY - requires confirmations)
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live
```

## Troubleshooting

### "PAPER MODE — simulated orders only"
This means `config.json` has `"trading_mode": "paper"`. This is correct for safe testing.

### "Live mode requires TWO explicit confirmations"
You need BOTH:
1. Config: `"trading_mode": "live"`
2. Command: `ALGO_LIVE_CONFIRMED=true --confirm-live`

### "max_capital_deployment_pct cannot exceed 70% in live mode"
Live mode enforces stricter limits. Change your config:
```json
"max_capital_deployment_pct": 70.0
```

## Summary

| Your Workflow | Command |
|---------------|---------|
| Test strategy | `go run ./cmd/engine --mode backtest` |
| Paper trade (safe) | `go run ./cmd/engine --mode market` |
| Live trade (REAL $) | `ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live` |

**Recommendation:** Stay in paper mode until you're completely confident! 🚀

