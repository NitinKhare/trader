# Single Strategy Backtest Guide

You can now backtest individual strategies in isolation to evaluate their performance independently!

## Quick Start

### List All Strategies

```bash
./scripts/test_single_strategy.sh --list
```

Output:
```
Available Strategies:
====================
  - TrendFollow
  - MeanReversion
  - Breakout
  - Momentum
  - VWAPReversion
  - Pullback
  - ORB
  - MACDCrossover
  - BollingerSqueeze
```

### Backtest a Single Strategy

```bash
./scripts/test_single_strategy.sh TrendFollow --backtest
```

This will:
1. Temporarily modify main.go to use only TrendFollow
2. Build and run the backtest
3. Show performance metrics for that strategy
4. Restore main.go to original state

## Available Strategies

| # | Strategy | Type | Use Case |
|---|----------|------|----------|
| 1 | **TrendFollow** | Trend | Ride long-term trends using moving averages |
| 2 | **MeanReversion** | Reversal | Trade bounces from overbought/oversold levels |
| 3 | **Breakout** | Breakout | Catch strong moves through support/resistance |
| 4 | **Momentum** | Momentum | Trade price acceleration and strength |
| 5 | **VWAPReversion** | Reversal | Trade deviations from VWAP |
| 6 | **Pullback** | Entry | Enter on pullbacks in uptrends |
| 7 | **ORB** | Breakout | Opening Range Breakout strategy |
| 8 | **MACDCrossover** | Trend | MACD crossover signals |
| 9 | **BollingerSqueeze** | Volatility | Trade breakouts from low volatility |

## Examples

### Test Individual Strategies

```bash
# Test Breakout strategy
./scripts/test_single_strategy.sh Breakout --backtest

# Test Mean Reversion
./scripts/test_single_strategy.sh MeanReversion --backtest

# Test Momentum strategy
./scripts/test_single_strategy.sh Momentum --backtest
```

### Just Status Check (No Backtest)

```bash
# Shows strategy details without running full backtest
./scripts/test_single_strategy.sh TrendFollow
```

## Understanding the Results

When you run a backtest for a single strategy, you'll see:

```
[backtest] loaded 1 strategies          ← Only 1 strategy running
[backtest] found 20 trading days        ← Uses all available AI data
[backtest] completed: 15 closed trades  ← Trades from this strategy only

── TRADE SUMMARY ──
  Total trades:    15
  Winning trades:  6 (40.0%)           ← Win rate for this strategy
  Losing trades:   6

── PROFIT & LOSS ──
  Total P&L:       ₹4887.28            ← Strategy profit/loss
  Average P&L:     ₹325.82
  Gross profit:    ₹21248.72
  Gross loss:      ₹16361.44
  Profit factor:   1.30                ← Profit/Loss ratio
```

### Key Metrics Explained

- **Total trades**: Number of trades this strategy placed
- **Win rate**: % of trades that made money
- **Total P&L**: Overall profit/loss from this strategy
- **Average P&L**: Average profit per trade
- **Profit factor**: Gross profit ÷ Gross loss (>1.0 is good)
- **Max drawdown**: Largest peak-to-trough loss
- **Sharpe ratio**: Risk-adjusted return (>1.0 is good)

## Comparing Strategies

Run multiple strategies back-to-back to compare:

```bash
# Test all trend-following strategies
./scripts/test_single_strategy.sh TrendFollow --backtest
./scripts/test_single_strategy.sh Momentum --backtest
./scripts/test_single_strategy.sh Pullback --backtest

# Test all breakout strategies
./scripts/test_single_strategy.sh Breakout --backtest
./scripts/test_single_strategy.sh ORB --backtest
./scripts/test_single_strategy.sh BollingerSqueeze --backtest
```

## Strategy Selection Guide

### If You Want...

**Maximum Win Rate:**
- Test: MeanReversion, VWAPReversion (these catch reversals)

**Highest Profit:**
- Test: TrendFollow, Momentum, Pullback

**Lower Drawdown:**
- Test: VWAPReversion, MeanReversion (mean reversion has tighter stops)

**Consistency:**
- Test: All, pick the one with best Sharpe ratio

### Example Results

**TrendFollow (Good Strategy)**
```
Total trades: 15
Winning: 6 (40.0%)
Total P&L: ₹4,887
Profit factor: 1.30
Sharpe ratio: 1.58
```

**Breakout (Struggling)**
```
Total trades: 8
Winning: 0 (0.0%)
Total P&L: ₹-11,006
Profit factor: 0.00
Sharpe ratio: -7.41
```

## Workflow: Test → Improve → Deploy

### 1. Individual Strategy Testing
```bash
# Test each strategy separately
for strategy in TrendFollow Breakout Momentum MeanReversion Pullback; do
    echo "Testing $strategy..."
    ./scripts/test_single_strategy.sh $strategy --backtest
    echo ""
done
```

### 2. Identify Best Performers
- Note which strategies have positive P&L
- Check Sharpe ratio (risk-adjusted returns)
- Look at win rate consistency

### 3. Optimize Top Performers
- Modify strategy code in `internal/strategy/`
- Adjust parameters (stops, targets, entry conditions)
- Re-test to validate improvements

### 4. Deploy Winners
Once you've identified profitable strategies:
```bash
# Run all strategies (only the profitable ones still active)
go run ./cmd/engine --mode backtest  # Final validation

# Then live test
go run ./cmd/engine --mode paper     # Paper trading

# Finally real trading (if confident)
go run ./cmd/engine --mode live --confirm-live
```

## Performance Tiers

### Excellent (Deploy as-is)
- Total P&L > ₹5,000
- Win rate > 50%
- Profit factor > 1.5
- Sharpe ratio > 2.0

### Good (Can deploy with monitoring)
- Total P&L > ₹2,000
- Win rate > 40%
- Profit factor > 1.2
- Sharpe ratio > 1.0

### Fair (Needs improvement)
- Total P&L > ₹500
- Win rate > 30%
- Profit factor > 1.0
- Sharpe ratio > 0.5

### Poor (Needs rework)
- Total P&L < 0
- Win rate < 30%
- Profit factor < 1.0
- Sharpe ratio < 0

## Troubleshooting

### "No trading dates found"
Generate AI scores first:
```bash
python3 scripts/backfill_ai_scores.py --days-back 30
```

### Strategy shows 0 trades
- Check if AI data exists: `ls -1 ai_outputs | wc -l`
- Check if stock data exists: `ls -1 market_data/*.csv | wc -l`
- Verify both are available for the same dates

### Script permission denied
Make it executable:
```bash
chmod +x scripts/test_single_strategy.sh
```

### Main.go not restored
If interrupted, manually restore:
```bash
if [[ -f cmd/engine/main.go.backup ]]; then
    mv cmd/engine/main.go.backup cmd/engine/main.go
fi
```

## Real-World Strategy Testing Process

### Day 1: Initial Screening
```bash
# Quick test all strategies
./scripts/test_single_strategy.sh TrendFollow --backtest    # ₹4,887 ✅
./scripts/test_single_strategy.sh Breakout --backtest       # -₹11,006 ❌
./scripts/test_single_strategy.sh Momentum --backtest       # ₹2,500 ✅
# ... test the rest
```

### Day 2: Deep Dive on Winners
```bash
# Focus on top performers with more data
python3 scripts/backfill_ai_scores.py --days-back 365
./scripts/test_single_strategy.sh TrendFollow --backtest
./scripts/test_single_strategy.sh Momentum --backtest
```

### Day 3: Optimization
```bash
# Modify top strategies for even better performance
# Edit: internal/strategy/trendfollow.go
# Edit: internal/strategy/momentum.go

# Re-test
./scripts/test_single_strategy.sh TrendFollow --backtest
./scripts/test_single_strategy.sh Momentum --backtest
```

### Day 4: Live Deployment
```bash
# Paper trading with optimized strategies
go run ./cmd/engine --mode paper

# Monitor for 1 week
# If results match backtest, consider real trading
```

## Summary

| Task | Command |
|------|---------|
| List strategies | `./scripts/test_single_strategy.sh --list` |
| Test one strategy | `./scripts/test_single_strategy.sh TrendFollow --backtest` |
| Test with status | `./scripts/test_single_strategy.sh TrendFollow` |
| Generate more data | `python3 scripts/backfill_ai_scores.py --days-back 365` |
| Full backtest | `go run ./cmd/engine --mode backtest` |

Now you can scientifically evaluate each strategy and deploy only the winners! 🚀

