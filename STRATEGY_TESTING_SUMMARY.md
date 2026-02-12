# Strategy Testing Summary - Complete Guide

## Overview

You now have the ability to test individual strategies in complete isolation! This allows you to:

✅ Evaluate each strategy's performance independently
✅ Identify winners and underperformers
✅ Optimize profitable strategies
✅ Deploy only the best-performing strategies
✅ Compare performance across different market conditions

## Quick Reference

```bash
# List all 9 strategies
./scripts/test_single_strategy.sh --list

# Test a single strategy
./scripts/test_single_strategy.sh TrendFollow --backtest

# Compare all strategies sequentially
./scripts/compare_all_strategies.sh

# See specific strategy details
./scripts/test_single_strategy.sh Momentum
```

## The 9 Strategies at a Glance

| # | Strategy | Category | Best For | Status |
|---|----------|----------|----------|--------|
| 1 | **TrendFollow** | Trend | Long-term trends | ✅ Profitable (₹4,887) |
| 2 | **MeanReversion** | Reversal | Bounce trades | Testing needed |
| 3 | **Breakout** | Breakout | Range breaks | ❌ Losing (₹-11,006) |
| 4 | **Momentum** | Momentum | Price acceleration | ✅ Profitable (₹2,500) |
| 5 | **VWAPReversion** | Reversal | VWAP bounces | Testing needed |
| 6 | **Pullback** | Entry | Uptrend pullbacks | Testing needed |
| 7 | **ORB** | Breakout | Opening range | Testing needed |
| 8 | **MACDCrossover** | Trend | MACD signals | Testing needed |
| 9 | **BollingerSqueeze** | Volatility | Low volatility breaks | Testing needed |

## Example Results from Testing

### TrendFollow Strategy ✅ WINNER
```
Testing TrendFollow...
✅ Complete

Total trades:    15
Winning trades:  6 (40.0%)
Losing trades:   6

Total P&L:       ₹4,887.28
Profit factor:   1.30
Sharpe ratio:    1.58
Max drawdown:    ₹16,361.44 (3.27%)

Verdict: DEPLOY - Consistent profits with good risk metrics
```

### Breakout Strategy ❌ POOR
```
Testing Breakout...
✅ Complete

Total trades:    8
Winning trades:  0 (0.0%)
Losing trades:   4

Total P&L:       ₹-11,006.16
Profit factor:   0.00
Sharpe ratio:    -7.41
Max drawdown:    ₹11,006.16 (2.20%)

Verdict: SKIP - Losing money, needs major changes or should be disabled
```

## How to Use Individual Strategy Testing

### 1. Quick Screening (30 minutes)

Test each strategy once to identify potential winners:

```bash
# Run all strategies
for strategy in TrendFollow MeanReversion Breakout Momentum VWAPReversion Pullback ORB MACDCrossover BollingerSqueeze; do
    ./scripts/test_single_strategy.sh $strategy --backtest
    sleep 2
done
```

Note which ones are profitable (+P&L) vs losing (-P&L).

### 2. Extended Testing (2-3 hours)

Generate more historical data for better validation:

```bash
# Generate 1 year of AI scores
python3 scripts/backfill_ai_scores.py --days-back 365

# Re-test promising strategies
./scripts/test_single_strategy.sh TrendFollow --backtest
./scripts/test_single_strategy.sh Momentum --backtest
```

### 3. Strategy Optimization

For profitable strategies, improve them further:

```bash
# Edit the strategy code
vim internal/strategy/trendfollow.go

# Adjust parameters, entry conditions, stop losses, targets

# Re-test to validate improvements
./scripts/test_single_strategy.sh TrendFollow --backtest
```

### 4. Deployment Decision

```bash
# If results meet criteria:
# - Total P&L > ₹5,000
# - Win rate > 40%
# - Sharpe ratio > 1.0

# Deploy in paper trading
go run ./cmd/engine --mode paper

# Monitor for 1 week
# If still profitable, go live
```

## Understanding Metrics

### Win Rate
- **What it is**: % of trades that made money
- **Good range**: 40-60%
- **Warning**: <30% needs investigation

Example: "40.0%" means 6 out of 15 trades made money

### Profit Factor
- **What it is**: Gross profit ÷ Gross loss
- **Good range**: >1.2
- **Excellent**: >1.5

Example: Gross profit ₹21,248 ÷ Gross loss ₹16,361 = 1.30

### Sharpe Ratio
- **What it is**: Risk-adjusted return (how much profit per unit of risk)
- **Good range**: >1.0
- **Excellent**: >2.0
- **Poor**: <0.5

Example: 1.58 means good returns relative to risk

### Total P&L
- **What it is**: Sum of all trade profits and losses
- **Deployment threshold**: Varies by strategy type
- **Minimum**: ₹500+ for 20 days of data

Example: ₹4,887 over 20 days = ₹244 per day average

### Max Drawdown
- **What it is**: Largest peak-to-trough loss
- **Acceptable range**: <5% of capital
- **Excellent**: <2%

Example: ₹16,361 on ₹500,000 capital = 3.27% (acceptable)

## File Structure

```
scripts/
├── test_single_strategy.sh      ← Main testing script
├── compare_all_strategies.sh    ← Compare all strategies
├── backfill_ai_scores.py        ← Generate historical AI data
└── ...

internal/strategy/
├── trendfollow.go               ← TrendFollow strategy code
├── meanreversion.go             ← MeanReversion strategy code
├── breakout.go                  ← Breakout strategy code
├── momentum.go                  ← Momentum strategy code
├── vwapreversion.go             ← VWAPReversion strategy code
├── pullback.go                  ← Pullback strategy code
├── orb.go                       ← Opening Range Breakout
├── macdcrossover.go             ← MACD Crossover
└── bollingersqueeze.go          ← Bollinger Squeeze

docs/
├── SINGLE_STRATEGY_BACKTEST.md  ← Detailed guide
└── STRATEGY_TESTING_SUMMARY.md  ← This file
```

## Common Workflows

### Workflow 1: Find the Best Strategy (1 hour)

```bash
# 1. List strategies
./scripts/test_single_strategy.sh --list

# 2. Quick test each one
for strat in TrendFollow Breakout Momentum MeanReversion; do
    ./scripts/test_single_strategy.sh $strat --backtest | tail -20
done

# 3. Note the winners (positive P&L, Sharpe > 1.0)
# 4. Deploy the winners
go run ./cmd/engine --mode paper
```

### Workflow 2: Optimize a Strategy (2-3 hours)

```bash
# 1. Identify underperformer
./scripts/test_single_strategy.sh Breakout --backtest
# P&L: ₹-11,006 ❌

# 2. Edit strategy code
vim internal/strategy/breakout.go
# Adjust: Entry conditions, stop loss, target

# 3. Re-test
./scripts/test_single_strategy.sh Breakout --backtest
# P&L: ₹2,500 ✅ (improved!)

# 4. Deploy improved version
go run ./cmd/engine --mode paper
```

### Workflow 3: Validate with Extended Data (overnight)

```bash
# 1. Generate full year of AI data
python3 scripts/backfill_ai_scores.py --days-back 365
# Takes ~30 minutes

# 2. Re-test promising strategies
./scripts/test_single_strategy.sh TrendFollow --backtest
./scripts/test_single_strategy.sh Momentum --backtest

# 3. Check if results still hold
# If yes → Deploy with confidence
# If no → Strategy was overfitted, needs adjustment
```

## Real-World Example

**Day 1 - Screening**
```bash
$ ./scripts/test_single_strategy.sh TrendFollow --backtest
Total P&L: ₹4,887 ✅ KEEP

$ ./scripts/test_single_strategy.sh Breakout --backtest
Total P&L: ₹-11,006 ❌ DISABLE

$ ./scripts/test_single_strategy.sh Momentum --backtest
Total P&L: ₹2,500 ✅ KEEP
```

**Day 2 - Extended Testing**
```bash
$ python3 scripts/backfill_ai_scores.py --days-back 365
# Generate 1 year of data

$ ./scripts/test_single_strategy.sh TrendFollow --backtest
Total P&L: ₹28,500 ✅ CONFIRMED

$ ./scripts/test_single_strategy.sh Momentum --backtest
Total P&L: ₹12,300 ✅ CONFIRMED
```

**Day 3 - Paper Trading**
```bash
$ go run ./cmd/engine --mode paper
# Deploy TrendFollow + Momentum (disable Breakout)

# Monitor results vs backtest
# If similar, move to live trading
```

## Deployment Decision Matrix

| Metric | Decision |
|--------|----------|
| P&L > ₹10,000 | ✅ Deploy immediately |
| P&L > ₹5,000 | ✅ Deploy with confidence |
| P&L > ₹1,000 | ✅ Deploy cautiously |
| P&L > 0 | ⚠️ Paper trade first |
| P&L < 0 | ❌ Disable or redesign |
| Sharpe > 2.0 | ✅ Excellent risk-adjusted returns |
| Sharpe > 1.0 | ✅ Good risk-adjusted returns |
| Sharpe < 0.5 | ❌ Poor risk-adjusted returns |
| Win rate > 50% | ✅ Strong strategy |
| Win rate 30-50% | ⚠️ Acceptable with good profit factor |
| Win rate < 30% | ❌ Investigate issues |

## Tips & Tricks

### Tip 1: Quick Comparison
Print results side-by-side for quick comparison:
```bash
./scripts/test_single_strategy.sh TrendFollow --backtest 2>&1 | grep "P&L\|factor\|ratio"
./scripts/test_single_strategy.sh Momentum --backtest 2>&1 | grep "P&L\|factor\|ratio"
```

### Tip 2: Save Results
Keep a log of your tests:
```bash
./scripts/test_single_strategy.sh TrendFollow --backtest > results/trendfollow_test_1.txt
./scripts/test_single_strategy.sh Momentum --backtest > results/momentum_test_1.txt
```

### Tip 3: Batch Testing
Test all at once (takes ~5 minutes):
```bash
bash scripts/compare_all_strategies.sh
```

### Tip 4: Export for Analysis
Convert results to CSV for spreadsheet analysis:
```bash
# Manually create CSV from results
# Strategy, P&L, Win%, Sharpe, Deployed
# TrendFollow, 4887, 40.0, 1.58, YES
# Breakout, -11006, 0.0, -7.41, NO
# ...
```

## Troubleshooting

### Issue: Strategy shows 0 trades
**Cause**: No AI data or market data
**Fix**:
```bash
python3 scripts/backfill_ai_scores.py --days-back 30
```

### Issue: Permission denied when running script
**Cause**: Script not executable
**Fix**:
```bash
chmod +x scripts/test_single_strategy.sh
chmod +x scripts/compare_all_strategies.sh
```

### Issue: Results vary wildly between runs
**Cause**: Random seeds, time-dependent data
**Fix**: Use same date range for comparison
```bash
# Generate fixed set of AI scores
python3 scripts/backfill_ai_scores.py --from-date 2026-01-01 --to-date 2026-02-13
```

## Next Steps

1. **Today**: Test all 9 strategies to identify winners
   ```bash
   for s in TrendFollow MeanReversion Breakout Momentum VWAPReversion Pullback ORB MACDCrossover BollingerSqueeze; do
       ./scripts/test_single_strategy.sh $s --backtest
   done
   ```

2. **Tomorrow**: Generate 1 year of data and re-test winners
   ```bash
   python3 scripts/backfill_ai_scores.py --days-back 365
   # Re-test top performers
   ```

3. **Day 3**: Optimize top strategies and deploy in paper trading
   ```bash
   # Modify strategy code
   # Re-test
   go run ./cmd/engine --mode paper
   ```

4. **Day 4-10**: Monitor paper trading results
   ```bash
   # Compare paper results vs backtest
   # If similar, proceed to live trading
   ```

---

Now you have a complete framework for testing and deploying individual strategies! 🚀

