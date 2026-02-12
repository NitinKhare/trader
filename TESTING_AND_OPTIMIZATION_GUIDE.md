# Complete Testing & Optimization Guide

This is your master guide for testing, optimizing, and deploying your trading strategies.

## Table of Contents

1. [Data Generation](#data-generation) - Generate historical AI data for backtesting
2. [Strategy Testing](#strategy-testing) - Test individual strategies
3. [Results Analysis](#results-analysis) - Interpret backtest metrics
4. [Optimization](#optimization) - Improve strategy performance
5. [Deployment](#deployment) - Deploy to paper/live trading

---

## Data Generation

### Generate Historical AI Scores

Before backtesting, you need AI scores for multiple dates. The market data goes back 1 year, but you need to generate AI scores for each date.

**Quick Test (1 month of data):**
```bash
python3 scripts/backfill_ai_scores.py --days-back 30
```

**Extended Test (3 months):**
```bash
python3 scripts/backfill_ai_scores.py --days-back 90
```

**Full Year (Best Validation):**
```bash
python3 scripts/backfill_ai_scores.py --days-back 365
```

**Specific Date Range:**
```bash
python3 scripts/backfill_ai_scores.py \
  --from-date 2026-01-01 \
  --to-date 2026-02-13
```

**Skip Already-Generated Dates:**
```bash
python3 scripts/backfill_ai_scores.py --days-back 30 --skip-existing
```

See: [BACKTEST_DATA_GENERATION.md](BACKTEST_DATA_GENERATION.md)

---

## Strategy Testing

### Test Individual Strategies

You have 9 strategies. Test them one at a time to identify winners.

**List All Strategies:**
```bash
./scripts/test_single_strategy.sh --list
```

**Test a Single Strategy:**
```bash
./scripts/test_single_strategy.sh TrendFollow --backtest
```

Output:
```
[backtest] loaded 1 strategies     ← Only this strategy runs
[backtest] completed: 15 closed trades

Total P&L:       ₹4,887.28
Win rate:        40.0%
Profit factor:   1.30
Sharpe ratio:    1.58
```

**Test All Strategies (Sequential):**
```bash
./scripts/compare_all_strategies.sh
```

### Strategy Overview

| Strategy | Type | Best For | Status |
|----------|------|----------|--------|
| TrendFollow | Trend | Long-term trends | ✅ Profitable |
| MeanReversion | Reversal | Bounce trades | ⏳ Testing |
| Breakout | Breakout | Range breaks | ❌ Losing |
| Momentum | Momentum | Acceleration | ✅ Profitable |
| VWAPReversion | Reversal | VWAP deviations | ⏳ Testing |
| Pullback | Entry | Pullbacks in uptrends | ⏳ Testing |
| ORB | Breakout | Opening range | ⏳ Testing |
| MACDCrossover | Trend | MACD signals | ⏳ Testing |
| BollingerSqueeze | Volatility | Low vol breaks | ⏳ Testing |

See: [SINGLE_STRATEGY_BACKTEST.md](SINGLE_STRATEGY_BACKTEST.md)

---

## Results Analysis

### Key Metrics

**Total P&L** (Profit and Loss)
- What: Sum of all trade profits and losses
- Good: > ₹2,000
- Excellent: > ₹5,000

**Win Rate** (% of profitable trades)
- What: Percentage of trades that made money
- Good: > 40%
- Excellent: > 50%
- Example: 40% = 6 out of 15 trades made money

**Profit Factor** (Gross profit ÷ Gross loss)
- What: How much you made vs how much you lost
- Good: > 1.2
- Excellent: > 1.5
- Example: ₹21,248 ÷ ₹16,361 = 1.30

**Sharpe Ratio** (Risk-adjusted return)
- What: Returns per unit of risk
- Good: > 1.0
- Excellent: > 2.0

**Max Drawdown** (Largest loss from peak)
- What: Biggest peak-to-trough decline
- Good: < 5% of capital
- Excellent: < 2%

### Example Interpretation

**TrendFollow Strategy (20 days of backtest):**
```
Total trades:    15
Winning:         6 (40.0%)
Total P&L:       ₹4,887.28
Profit factor:   1.30
Sharpe ratio:    1.58
Max drawdown:    ₹16,361.44 (3.27%)

✅ VERDICT: PROFITABLE & CONSISTENT
   - Positive P&L ✓
   - Acceptable win rate ✓
   - Good profit factor ✓
   - Good Sharpe ratio ✓
   - Manageable drawdown ✓
   → DEPLOY
```

**Breakout Strategy (20 days of backtest):**
```
Total trades:    8
Winning:         0 (0.0%)
Total P&L:       ₹-11,006.16
Profit factor:   0.00
Sharpe ratio:    -7.41
Max drawdown:    ₹11,006.16 (2.20%)

❌ VERDICT: UNPROFITABLE
   - Negative P&L ✗
   - Zero win rate ✗
   - Zero profit factor ✗
   - Negative Sharpe ✗
   → SKIP or REDESIGN
```

---

## Optimization

### Step 1: Identify Candidates

Run backtest on individual strategies:
```bash
./scripts/test_single_strategy.sh TrendFollow --backtest
./scripts/test_single_strategy.sh Momentum --backtest
./scripts/test_single_strategy.sh Breakout --backtest
```

Note which ones are profitable (positive P&L).

### Step 2: Validate with Extended Data

```bash
# Generate 1 year of data
python3 scripts/backfill_ai_scores.py --days-back 365

# Re-test profitable strategies
./scripts/test_single_strategy.sh TrendFollow --backtest
./scripts/test_single_strategy.sh Momentum --backtest
```

If still profitable, they're solid candidates for optimization.

### Step 3: Improve Code

Edit the strategy code to improve it:

```bash
# Edit TrendFollow strategy
vim internal/strategy/trendfollow.go

# Possible improvements:
# - Adjust moving average periods
# - Tighten stop losses
# - Improve entry conditions
# - Add filters (trend strength, volatility)
# - Optimize take profit targets
```

### Step 4: Re-test

After modifications, test again:

```bash
./scripts/test_single_strategy.sh TrendFollow --backtest
```

Compare P&L before and after optimization.

### Step 5: Validate

Run with extended data to ensure improvement holds:

```bash
python3 scripts/backfill_ai_scores.py --days-back 365 --skip-existing
./scripts/test_single_strategy.sh TrendFollow --backtest
```

If improved, proceed to deployment.

---

## Deployment

### Pre-Deployment Checklist

- [ ] Strategy is profitable (P&L > ₹2,000)
- [ ] Win rate acceptable (> 40%)
- [ ] Profit factor good (> 1.2)
- [ ] Sharpe ratio positive (> 1.0)
- [ ] Drawdown manageable (< 5%)
- [ ] Results validated with 1 year of data
- [ ] Unprofitable strategies are disabled

### Paper Trading (Risk-Free Testing)

```bash
# Run in paper mode (simulated, no real money)
go run ./cmd/engine --mode paper

# Monitor results for 1-2 weeks
# Compare paper results vs backtest results
# If similar, proceed to live
```

### Live Trading (Real Money)

**WARNING:** Only deploy to live trading after thorough validation!

```bash
# Requires TWO confirmations for safety:
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode live --confirm-live

# This deploys ONLY after:
# 1. Strong backtest results
# 2. Successful paper trading
# 3. Explicit safety confirmations
```

---

## Complete Workflow Example

### Day 1: Initial Screening (1 hour)

```bash
# 1. Ensure AI data exists
python3 scripts/backfill_ai_scores.py --days-back 30

# 2. Test all 9 strategies quickly
for strat in TrendFollow MeanReversion Breakout Momentum VWAPReversion Pullback ORB MACDCrossover BollingerSqueeze; do
    ./scripts/test_single_strategy.sh $strat --backtest 2>&1 | grep -E "P&L|factor|ratio"
    sleep 1
done

# 3. Note the winners
# TrendFollow:  ₹4,887 ✅
# Momentum:     ₹2,500 ✅
# Breakout:     -₹11,006 ❌
# ... (continue for all)
```

### Day 2: Validation (2-3 hours)

```bash
# 1. Generate full year of data
python3 scripts/backfill_ai_scores.py --days-back 365

# 2. Re-test winners
./scripts/test_single_strategy.sh TrendFollow --backtest
./scripts/test_single_strategy.sh Momentum --backtest

# 3. Verify results hold (look for improvement or stability)
```

### Day 3: Optimization (Variable)

```bash
# 1. Edit promising strategies
vim internal/strategy/trendfollow.go
vim internal/strategy/momentum.go

# 2. Re-test after changes
./scripts/test_single_strategy.sh TrendFollow --backtest
./scripts/test_single_strategy.sh Momentum --backtest

# 3. Compare with previous results
# TrendFollow improved from ₹4,887 to ₹7,500? ✅
# If yes, optimization successful
```

### Day 4+: Deployment

```bash
# 1. Run full backtest with all strategies
go run ./cmd/engine --mode backtest

# 2. Paper trade
go run ./cmd/engine --mode paper
# Monitor for 1 week

# 3. Live trade (if paper matches backtest)
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode live --confirm-live
```

---

## Quick Reference Commands

```bash
# Data Generation
python3 scripts/backfill_ai_scores.py --days-back 30
python3 scripts/backfill_ai_scores.py --days-back 365
python3 scripts/backfill_ai_scores.py --skip-existing

# Strategy Testing
./scripts/test_single_strategy.sh --list
./scripts/test_single_strategy.sh TrendFollow --backtest
./scripts/test_single_strategy.sh Momentum --backtest
./scripts/compare_all_strategies.sh

# Backtesting
go run ./cmd/engine --mode backtest
go run ./cmd/engine --mode paper
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode live --confirm-live
```

---

## Troubleshooting

**"No trading dates found"**
```bash
python3 scripts/backfill_ai_scores.py --days-back 30
```

**"Permission denied" on script**
```bash
chmod +x scripts/test_single_strategy.sh
chmod +x scripts/compare_all_strategies.sh
```

**Strategy shows 0 trades**
- Check AI data exists: `ls ai_outputs | wc -l`
- Check market data exists: `ls market_data/*.csv | wc -l`
- Regenerate if needed: `python3 scripts/backfill_ai_scores.py --days-back 30`

**Results vary between runs**
- Use same date range for comparison
- Run on same data (don't generate new data between runs)
- Use `--skip-existing` to preserve baseline

---

## Further Reading

- **BACKTEST_DATA_GENERATION.md** - Detailed data generation guide
- **SINGLE_STRATEGY_BACKTEST.md** - Individual strategy testing guide
- **STRATEGY_TESTING_SUMMARY.md** - Quick reference for strategy metrics

---

## Summary

**The Complete Testing Pipeline:**

```
Generate Data
     ↓
Test Individual Strategies
     ↓
Identify Winners
     ↓
Validate with Extended Data
     ↓
Optimize Code
     ↓
Re-test & Validate
     ↓
Paper Trade
     ↓
Live Trade (if approved)
```

**Your Three Most Important Commands:**

```bash
# 1. Generate data
python3 scripts/backfill_ai_scores.py --days-back 365

# 2. Test strategies
./scripts/test_single_strategy.sh TrendFollow --backtest

# 3. Deploy winners
go run ./cmd/engine --mode paper
```

Now you have a complete, scientific framework for testing and optimizing strategies! 🚀

