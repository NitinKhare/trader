# 📊 Backtest Capital Usage Tracking

## Overview

When running backtests, the engine now logs detailed capital usage metrics to help you understand:
- How much capital is actually being deployed
- Peak capital usage during the backtest period
- Final capital after all trades close
- Total P&L as percentage of initial capital

## What Gets Logged

When you run a backtest, you'll see output like this:

```
[backtest] starting capital: ₹1,000,000.00
[backtest] found 60 trading days with AI data: 2025-01-01 to 2025-03-31

... trading happens ...

[backtest] completed: 47 closed trades, 3 still open at end
[backtest] capital usage summary:
  Starting Capital:    ₹1,000,000.00
  Maximum Used:        ₹850,000.00 (85.0% of starting)
  Final Capital:       ₹1,050,000.00
  Total P&L:           ₹50,000.00 (5.00%)
```

## Capital Usage Calculation

### Daily Capital Tracking

Each trading day, the system calculates:

```go
capitalUsed := 0.0
for _, pos := range positions {
    capitalUsed += pos.entryPrice * float64(pos.quantity)
}
```

**Example:**
- You have 2 open positions:
  - Position 1: TITAN @ ₹5000/share × 20 shares = ₹100,000
  - Position 2: SBIN @ ₹1000/share × 50 shares = ₹50,000
- **Capital Used That Day:** ₹150,000

### Maximum Capital Used

The system tracks the highest capital usage across all trading days:

```go
if capitalUsed > maxCapitalUsed {
    maxCapitalUsed = capitalUsed
}
```

**Why This Matters:**
- Tells you the peak number of concurrent positions
- Shows if you hit capital constraints
- Validates risk management is working (max 70% deployment by default)

### Final Capital Calculation

After all trades close, the system calculates:

```go
finalCapital := cfg.Capital  // Start with initial capital
for _, trade := range closedTrades {
    finalCapital += trade.PnL  // Add all profits/losses
}
```

**Example:**
- Initial: ₹1,000,000
- Trade 1 PnL: +₹5,000
- Trade 2 PnL: -₹2,000
- Trade 3 PnL: +₹47,000
- **Final Capital:** ₹1,050,000

## Interpreting the Output

### Maximum Used Percentage

```
Maximum Used: ₹850,000.00 (85.0% of starting)
```

This tells you:
- ✅ **85% used** = Good capital utilization, still have buffer
- ✅ **70% used** = Conservative, typical for live trading
- ❌ **100% used** = All capital deployed, no buffer (risky)
- ⚠️ **>100%** = Would require margin (not allowed in your config)

### P&L Percentage

```
Total P&L: ₹50,000.00 (5.00%)
```

This shows your return on initial capital:
- ✅ **+5% to +20%** = Good strategy performance
- ✅ **+1% to +5%** = Consistent but modest returns
- ⚠️ **-5% to 0%** = Losing strategy, needs adjustment
- ❌ **<-5%** = Poor strategy, should not use for live trading

## Key Metrics to Watch

### 1. Capital Efficiency
```
Efficiency = Total P&L / Maximum Capital Used
```

**Example:**
- Maximum Used: ₹850,000
- Total P&L: ₹50,000
- **Efficiency: 5.88%** (good return on deployed capital)

### 2. Risk-Adjusted Return
```
Return % = (Final Capital - Starting Capital) / Starting Capital × 100
```

**Goal:** Aim for 1-5% monthly return in paper trading

### 3. Capital Deployment Trend

Look at the maximum used percentage:
- **Stable at ~70%** = Risk management working perfectly
- **Fluctuates 30-90%** = Capital constraints changing daily
- **Grows over time** = Accumulating profits, more positions possible

## Example Backtest Output

```
[backtest] starting capital: ₹1,000,000.00
[backtest] found 252 trading days with AI data: 2024-01-01 to 2024-12-31

[Strategy TrendFollow]
  Day 1: Open TITAN @ ₹5000 × 20 = ₹100,000 used
  Day 5: Close TITAN @ ₹5050, PnL = +₹1,000
  Day 6: Open SBIN @ ₹1000 × 50 = ₹50,000 used
  Day 10: Open GRASIM @ ₹2900 × 30 = ₹87,000 used
           Total Used Now: ₹137,000
  Day 12: Close SBIN @ ₹1010, PnL = +₹500
  Day 15: Close GRASIM @ ₹2920, PnL = +₹600
           Max Use Today Was: ₹137,000

... many more trades ...

Peak usage day (Day 45):
  Open positions: TITAN, SBIN, GRASIM, INFY, BAJAJ
  Capital used: ₹850,000 (85.0% of ₹1M)

[backtest] completed: 145 closed trades, 2 still open at end
[backtest] capital usage summary:
  Starting Capital:    ₹1,000,000.00
  Maximum Used:        ₹850,000.00 (85.0% of starting)
  Final Capital:       ₹1,075,000.00
  Total P&L:           ₹75,000.00 (7.50%)
```

## Capital Usage vs Risk Management

The system enforces these constraints **during backtesting**:

```go
// 1% risk per trade (position size constraint)
max_risk = capital × 1%
quantity = max_risk / risk_per_share

// Max 5 concurrent positions
if len(open_positions) >= 5 {
    skip new entries
}

// Capital check (no over-leveraging)
if cost > available_capital {
    reduce quantity or skip
}
```

### Example Risk Calculation

Starting capital: **₹1,000,000**
Risk per trade: **1%** = ₹10,000 max loss per trade

For TITAN @ ₹5000 with ATR of ₹100:
- Stop Loss distance: ₹200 (2 × ATR)
- Max loss: ₹10,000
- **Max quantity: 50 shares** (₹10,000 ÷ ₹200)
- **Capital used: ₹250,000** (₹5000 × 50)

## Comparing Across Backtests

Use these metrics to compare different backtest runs:

```bash
# Backtest 1: Conservative strategy
Maximum Used: ₹600,000 (60%)
Final Capital: ₹1,080,000 (+8.0%)

# Backtest 2: Aggressive strategy
Maximum Used: ₹950,000 (95%)
Final Capital: ₹1,085,000 (+8.5%)
```

**Analysis:**
- Backtest 2 has slightly better return (+0.5%)
- But uses 35% more capital (higher risk)
- Backtest 1 is safer with consistent performance

## Troubleshooting Capital Issues

### "Maximum Used > 70%"
**Problem:** More capital used than expected
**Solutions:**
- Check max positions (should be 5-10)
- Verify position sizing (1% risk rule)
- Check if positions holding longer than expected

### "Maximum Used < 30%"
**Problem:** Not enough capital being deployed
**Solutions:**
- Reduce risk threshold to open more trades
- Lower required scores for entries
- Check market regime (only trades in BULL?)

### "Final Capital < Starting Capital"
**Problem:** Losing money
**Solutions:**
- Strategy needs adjustment
- Check entry/exit criteria
- Increase stop-loss buffer
- Don't use for live trading

## Logging Location

Capital usage logs are written to:
- **Console:** Printed with `[backtest]` prefix
- **Log File:** `logs/engine_YYYY-MM-DD.log` (if file logging enabled)

To see live logs while backtesting:
```bash
# Run backtest and see all output
./engine --mode backtest 2>&1 | grep -E "(backtest|capital)"

# Or watch the log file
tail -f logs/engine_*.log | grep -E "(backtest|capital)"
```

## Next Steps

1. **Run a backtest** to see capital usage:
   ```bash
   go run ./cmd/engine --mode backtest
   ```

2. **Verify capital metrics:**
   - Is max used between 50-80%? ✅
   - Is final capital positive? ✅
   - Is P&L > 1% monthly? ✅

3. **If good, run dry-run:**
   ```bash
   go run ./cmd/engine --mode dry-run
   ```

4. **Then go live (if confident):**
   ```bash
   go run ./cmd/engine --mode market
   ```

---

**Key Takeaway:** Capital usage tracking helps you validate that:
- Your position sizing is correct
- Risk management is working
- Strategy is profitable enough to trade live
- Capital deployment matches expectations
