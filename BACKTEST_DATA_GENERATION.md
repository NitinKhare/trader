# Backtest Data Generation Guide

## Why Your Backtest Was Always the Same

The backtest showed the same results because it only had **AI scoring data for 2 days** (2026-02-09 and 2026-02-11).

The backtest engine works like this:

```
1. Load AI scores from ai_outputs/{date}/ for each trading day
2. For each date with scores, find trading opportunities
3. Simulate trades based on your strategies
4. Report performance metrics
```

**If you only have 2 days of AI data → you only get trades for 2 days → same results every time**

## The Solution: Backfill AI Scores

Your market data goes back ~1 year, but you only generated AI scores for 2 days.

The solution is to **run the AI scoring pipeline on all historical dates** where you have market data. This creates AI output for each date, which the backtest can then use.

## How to Backfill AI Scores

### Quick Start - Last 30 Days

```bash
python3 scripts/backfill_ai_scores.py --days-back 30
```

This will:
1. Find all trading dates in the last 30 days with market data
2. Run AI scoring for each date
3. Generate `ai_outputs/{date}/` directories with scores
4. Show progress as it goes

### Full Year of Data

```bash
python3 scripts/backfill_ai_scores.py --days-back 365
```

### Specific Date Range

```bash
python3 scripts/backfill_ai_scores.py \
  --from-date 2026-01-01 \
  --to-date 2026-02-13
```

### Specific Dates Only

```bash
python3 scripts/backfill_ai_scores.py \
  --dates 2026-02-08,2026-02-09,2026-02-10,2026-02-11
```

### Skip Already-Generated Scores

```bash
python3 scripts/backfill_ai_scores.py --days-back 30 --skip-existing
```

This is useful if you run it multiple times - it won't re-generate scores that already exist.

### Dry Run (See What Would Happen)

```bash
python3 scripts/backfill_ai_scores.py --days-back 30 --dry-run
```

Shows you the dates that would be processed without actually running them.

## What Gets Generated

When you run the backfill script, it creates directories for each trading date:

```
ai_outputs/
├── 2026-01-15/
│   ├── stock_scores.json    ← AI rankings for all stocks
│   └── market_regime.json    ← Market condition (UPTREND/DOWNTREND/SIDEWAYS)
├── 2026-01-18/
│   ├── stock_scores.json
│   └── market_regime.json
├── 2026-01-19/
│   ├── stock_scores.json
│   └── market_regime.json
... (one per trading day)
```

Each file contains:
- **stock_scores.json**: AI composite scores for all stocks (0-100 scale)
- **market_regime.json**: Overall market condition for that day

## Backtest Results Improve With More Data

As you add more historical AI scores, your backtest becomes more meaningful:

| Scenario | Trading Days | Trades | Reliability |
|----------|-------------|--------|-------------|
| 2 days | 2 | 5 | Very low (sample size too small) |
| 1 month | 20 | 50+ | Better (but still short-term) |
| 3 months | 60 | 150+ | Good (reasonable period) |
| 1 year | 250+ | 500+ | Excellent (long-term validation) |

## Performance Metrics Interpretation

Once you have months of data:

### Good Indicators
✅ **Profit factor > 1.5** - You're making more than you're losing
✅ **Win rate > 40%** - Majority of trades profitable
✅ **Sharpe ratio > 1.0** - Good risk-adjusted returns
✅ **Max drawdown < 5%** - Manageable losses

### Warning Signs
⚠️ **Profit factor < 1.0** - Strategy loses money
⚠️ **Win rate < 30%** - Too many losing trades
⚠️ **Max drawdown > 10%** - Excessive losses
⚠️ **Sharpe ratio < 0.5** - Poor risk-adjusted performance

## Example: Before and After

### Before Backfill (2 days)
```
[backtest] found 2 trading days with AI data: 2026-02-09 to 2026-02-11
Total trades:    5
Total P&L:       ₹1105.25
Winning trades:  4 (80.0%)
```

### After Backfill (20 days)
```
[backtest] found 20 trading days with AI data: 2026-01-15 to 2026-02-11
Total trades:    50
Total P&L:       ₹4040.15
Winning trades:  26 (52.0%)
```

Much more realistic! The 80% win rate compressed to 52% with more trades, which is more realistic for production trading.

## Workflow: Develop → Test → Deploy

### 1. Development (Your First Week)
```bash
# Generate 1 month of scores for testing
python3 scripts/backfill_ai_scores.py --days-back 30

# Run backtest to see how strategy performs
go run ./cmd/engine --mode backtest

# If results look good, continue to testing
```

### 2. Testing (Extended Validation)
```bash
# Generate 1 year of scores for comprehensive validation
python3 scripts/backfill_ai_scores.py --days-back 365

# Run backtest on full year
go run ./cmd/engine --mode backtest

# Analyze results across different market conditions
```

### 3. Deployment (Live Trading)
```bash
# Add new stocks to universe if desired
python3 scripts/update_instruments_dynamic.py --add-stocks BANKBARODA

# Continue generating fresh AI scores daily for live trading
python3 -m python_ai.run_scoring  # Runs with today's date by default

# Run live/paper trading
go run ./cmd/engine --mode paper
```

## Troubleshooting

### "No trading dates found"
Your market data directory is empty. Generate market data first:
```bash
python3 -m python_ai.data.fetch_dhan --days-back 365
```

### "Only 1-2 trading dates found"
Your market data is limited. The script can only generate scores for dates with market data.
Check: `ls -1 ./market_data/*.csv | wc -l` to see how many stocks have data.

### Scores seem to be taking a long time
One score generation takes ~5 seconds. For 250 trading days, expect ~20 minutes.
You can run it in the background:
```bash
python3 scripts/backfill_ai_scores.py --days-back 365 &
```

### Want to regenerate specific dates?
Remove the ai_outputs directory for that date and rerun:
```bash
rm -rf ai_outputs/2026-02-09
python3 scripts/backfill_ai_scores.py --dates 2026-02-09
```

## Daily Operations

### For Daily Live Trading

Every day before market close or after market hours:

```bash
# 1. Fetch latest market data (if not automated)
python3 -m python_ai.data.fetch_dhan --days-back 30

# 2. Generate today's AI scores
python3 -m python_ai.run_scoring  # Uses today's date by default

# 3. Run live/paper trading
go run ./cmd/engine --mode paper  # Paper trading
# OR
go run ./cmd/engine --mode live   # Real trading (use with caution!)
```

### For Regular Backtesting

Whenever you modify your strategies:

```bash
# 1. Ensure you have recent AI scores
python3 scripts/backfill_ai_scores.py --days-back 30 --skip-existing

# 2. Run backtest with latest strategies
go run ./cmd/engine --mode backtest
```

## Summary

| Task | Command |
|------|---------|
| Backfill 1 month | `python3 scripts/backfill_ai_scores.py --days-back 30` |
| Backfill 1 year | `python3 scripts/backfill_ai_scores.py --days-back 365` |
| Check what would run | `python3 scripts/backfill_ai_scores.py --days-back 30 --dry-run` |
| Skip already-generated | `python3 scripts/backfill_ai_scores.py --days-back 30 --skip-existing` |
| Generate today's score | `python3 -m python_ai.run_scoring` |
| Run backtest | `go run ./cmd/engine --mode backtest` |

The more historical AI data you generate, the more meaningful your backtest results become! 📊

