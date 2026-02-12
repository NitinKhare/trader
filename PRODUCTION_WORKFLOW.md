# Complete Production Workflow Guide

This guide shows you the complete daily and ongoing workflow for running your trading engine in production.

---

## Quick Reference: Command Timeline

```
BEFORE MARKET OPENS (Before 9:15 AM IST)
├─ Fetch market data for today
├─ Generate AI scores for today
└─ Start trading engine

DURING MARKET HOURS (9:15 AM - 3:30 PM IST)
├─ Engine runs automatically every 5 minutes
├─ Places orders based on strategies
└─ Logs all trades and funds

AFTER MARKET CLOSES (After 3:30 PM IST)
├─ Engine stops automatically
├─ Run nightly batch jobs
├─ Generate reports
└─ Prepare for next day

WEEKLY
├─ Review strategy performance
├─ Check if any strategies need optimization
└─ Validate backtest results

MONTHLY
├─ Full backtest with new data
├─ Strategy re-optimization if needed
└─ Risk analysis and reporting
```

---

## Detailed Daily Workflow

### STEP 1: Pre-Market Setup (8:45 AM - 9:10 AM IST)

Run these commands **before market opens at 9:15 AM**:

#### 1a. Fetch Today's Market Data
```bash
# Fetches latest market data for all 64 stocks
python3 python_ai/data/fetch_market_data.py --date today
```

**What it does:**
- Downloads OHLCV (Open, High, Low, Close, Volume) data for today
- Stores in `market_data/` directory
- Required for AI scoring and live trading

**Time:** ~2-3 minutes

#### 1b. Generate AI Scores for Today
```bash
# Generates AI signals/scores for all stocks using today's market data
python3 python_ai/scoring_engine.py --date today --output ./ai_outputs
```

**What it does:**
- Analyzes market data using your ML models
- Generates score for each stock (0-100)
- Generates market regime (Bullish/Bearish/Neutral)
- Outputs to `ai_outputs/{today_date}/stock_scores.json`

**Time:** ~5-10 minutes (depending on ML model complexity)

#### 1c. Verify Data Before Trading
```bash
# Quick status check
go run ./cmd/engine --mode status
```

**What it does:**
- Confirms all data loaded correctly
- Shows strategies loaded
- Verifies database connection
- Displays current funds

**Expected output:**
```
[engine] Config loaded: broker=dhan mode=paper capital=500000.00
[engine] Connected to Dhan broker
[engine] Loaded 9 strategies
[engine] Database connected
[engine] ✅ System ready for trading
```

**Time:** ~5 seconds

### STEP 2: Start Trading Engine (9:10 AM IST)

```bash
# For LIVE TRADING (Real Money)
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live

# OR for PAPER TRADING (Simulated - Recommended for testing)
go run ./cmd/engine --mode market
```

**What it does:**
- Connects to Dhan broker
- Fetches current market data
- Evaluates all 9 strategies
- Places real (or simulated) orders
- **Runs automatically every 5 minutes** until market close

**Expected output (every 5 minutes):**
```
[engine] 2026/02/13 10:30:00 main.go:1475: [market-loop] running jobs at 10:30:00...
[engine] 2026/02/13 10:30:01 main.go:1480: [market-loop] [funds] available=₹450000.00 used_margin=₹50000.00 total=₹500000.00
[engine] 2026/02/13 10:30:02 main.go:1485: [market-loop] Evaluated 64 stocks across 9 strategies
[engine] 2026/02/13 10:30:03 main.go:1490: [market-loop] Placed 2 BUY orders, 1 SELL order
[engine] 2026/02/13 10:30:05 main.go:1495: [market-loop] jobs completed successfully
```

**Time:** Runs continuously until market close

---

### STEP 3: During Market Hours (9:15 AM - 3:30 PM IST)

**You don't need to do anything!** The engine runs automatically.

What's happening in the background:

- ✅ Every 5 minutes: Evaluates all strategies
- ✅ Every 5 minutes: Places new orders if signals trigger
- ✅ Every 5 minutes: Manages exits (stop losses, take profits)
- ✅ Every 5 minutes: Logs funds, positions, P&L
- ✅ Continuously: Monitors risk limits (max 5 open positions, max 3% daily loss, etc.)

**Monitor in a separate terminal:**
```bash
# Watch the logs in real-time
tail -f logs/engine.log

# Or grep for specific patterns
tail -f logs/engine.log | grep "\[funds\]\|\[trade\]\|\[order\]"
```

---

### STEP 4: After Market Close (3:30 PM - 4:00 PM IST)

#### 4a. Engine Stops Automatically
The engine detects market close and stops execution automatically (no manual stop needed).

#### 4b. Run Nightly Batch Jobs
```bash
# Generate end-of-day reports, close positions if needed, etc.
go run ./cmd/engine --mode nightly
```

**What it does:**
- Closes any remaining open positions
- Generates end-of-day P&L report
- Updates database with today's trades
- Calculates daily statistics

**Time:** ~2-5 minutes

#### 4c. Optional: Analyze Today's Results
```bash
# Generate daily performance report
python3 scripts/analyze_daily_results.py --date today

# Shows:
# - Total P&L for the day
# - Win rate
# - Best performing strategy
# - Worst performing strategy
# - Risk metrics
```

---

## Weekly Tasks (Once Per Week)

### Run on Friday or Saturday (1-2 hours)

```bash
# Test individual strategies to see which performed best this week
./scripts/test_single_strategy.sh TrendFollow --backtest
./scripts/test_single_strategy.sh Momentum --backtest
./scripts/test_single_strategy.sh MeanReversion --backtest
# ... run all 9

# Compare all strategies
./scripts/compare_all_strategies.sh
```

**Decision points:**
- ✅ If a strategy is losing money consistently → disable or redesign
- ✅ If a strategy is very profitable → consider optimizing it further
- ✅ If results differ from backtest → investigate why

---

## Monthly Tasks (Once Per Month)

### Full Validation (3-5 hours on a weekend)

```bash
# 1. Generate 1 month of historical AI data
python3 scripts/backfill_ai_scores.py --days-back 30

# 2. Run full backtest with all strategies
go run ./cmd/engine --mode backtest

# 3. Compare backtest results vs live trading results
# - Did strategies perform as expected?
# - Any improvements needed?
# - Any risk issues?

# 4. If backtest looks good and live trading matches:
# - You can increase capital deployment %
# - Or add new strategies
# - Or optimize existing ones
```

---

## Complete Daily Example (Production-Ready)

### Scenario: You run your trading engine daily

**8:45 AM - Pre-Market Prep:**
```bash
# Terminal 1: Fetch and prepare data
python3 python_ai/data/fetch_market_data.py --date today
python3 python_ai/scoring_engine.py --date today --output ./ai_outputs
go run ./cmd/engine --mode status
```

**9:10 AM - Start Trading:**
```bash
# Terminal 1: Start the engine (keep this terminal open)
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live
```

**9:15 AM - 3:30 PM - During Market:**
```bash
# Terminal 2: Monitor logs in real-time
tail -f logs/engine.log | grep "\[funds\]\|\[trade\]"
```

**3:45 PM - After Market Close:**
```bash
# Terminal 1 (after engine stops): Run nightly jobs
go run ./cmd/engine --mode nightly

# Terminal 2: Analyze results
python3 scripts/analyze_daily_results.py --date today
```

**Expected output from nightly:**
```
[engine] Running nightly batch jobs...
[engine] Total trades today: 15
[engine] Today's P&L: ₹12,350
[engine] Win rate: 66.7%
[engine] Average trade: ₹823
[engine] Largest win: ₹2,500
[engine] Largest loss: ₹500
[engine] ✅ Nightly batch complete
```

---

## Running on a Server (24/7 Deployment)

If you want the engine to run automatically every day without manual commands:

### Option 1: Systemd Service (Linux)

Create `/etc/systemd/system/algo-trader.service`:

```ini
[Unit]
Description=Algorithmic Trading Engine
After=network.target

[Service]
Type=simple
User=your_username
WorkingDirectory=/path/to/algoTradingAgent
ExecStart=/bin/bash -c 'source ~/.bashrc && ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live'
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable algo-trader
sudo systemctl start algo-trader
```

Monitor:
```bash
sudo journalctl -u algo-trader -f
```

### Option 2: Cron Job (Linux/Mac)

Create `scripts/cron_start_engine.sh`:

```bash
#!/bin/bash
cd /path/to/algoTradingAgent

# Pre-market data fetch (8:50 AM every trading day)
python3 python_ai/data/fetch_market_data.py --date today
python3 python_ai/scoring_engine.py --date today --output ./ai_outputs

# Start engine (9:10 AM every trading day)
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live &
ENGINE_PID=$!

# Kill engine after market close (3:35 PM)
sleep 21900  # 6.08 hours = ~market hours
kill $ENGINE_PID

# Run nightly jobs
go run ./cmd/engine --mode nightly
```

Add to crontab:
```bash
crontab -e

# Add this line (runs on weekdays only, 8:50 AM)
50 08 * * 1-5 bash /path/to/algoTradingAgent/scripts/cron_start_engine.sh
```

### Option 3: Docker (Any Server/Cloud)

Create `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine

WORKDIR /app
COPY . .

RUN go build -o engine ./cmd/engine

ENV ALGO_LIVE_CONFIRMED=true

CMD ["./engine", "--mode", "market", "--confirm-live"]
```

Build and run:
```bash
docker build -t algo-trader .
docker run -d --name algo-trader-prod algo-trader
docker logs -f algo-trader-prod
```

---

## Command Reference by Time

### Before Market (8:45 AM)
```bash
python3 python_ai/data/fetch_market_data.py --date today
python3 python_ai/scoring_engine.py --date today --output ./ai_outputs
go run ./cmd/engine --mode status
```

### Market Open (9:10 AM)
```bash
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live
```

### During Market (9:15 AM - 3:30 PM)
```bash
# Monitor in separate terminal
tail -f logs/engine.log | grep "\[funds\]\|\[trade\]"
```

### After Market (3:45 PM)
```bash
go run ./cmd/engine --mode nightly
python3 scripts/analyze_daily_results.py --date today
```

### Weekly (Friday)
```bash
./scripts/test_single_strategy.sh TrendFollow --backtest
./scripts/compare_all_strategies.sh
```

### Monthly (Weekend)
```bash
python3 scripts/backfill_ai_scores.py --days-back 30
go run ./cmd/engine --mode backtest
```

---

## Troubleshooting

### Issue: Engine starts in PAPER MODE instead of LIVE
**Check:** Is `config.json` set to `"trading_mode": "live"`?
```bash
grep trading_mode config/config.json
```
If it shows `"paper"`, update it to `"live"`

### Issue: "No market data for today"
**Solution:**
```bash
# Fetch missing data
python3 python_ai/data/fetch_market_data.py --date today --force
```

### Issue: "No AI scores for today"
**Solution:**
```bash
# Generate scores
python3 python_ai/scoring_engine.py --date today --output ./ai_outputs --force
```

### Issue: Engine stops unexpectedly
**Check logs:**
```bash
tail -100 logs/engine.log
```
Look for errors like:
- Connection issues to Dhan broker
- Risk limit exceeded
- Insufficient funds

---

## Summary

**Your Daily Workflow:**

| Time | Action | Command |
|------|--------|---------|
| 8:45 AM | Fetch market data | `python3 python_ai/data/fetch_market_data.py --date today` |
| 8:50 AM | Generate AI scores | `python3 python_ai/scoring_engine.py --date today` |
| 9:00 AM | Verify system | `go run ./cmd/engine --mode status` |
| 9:10 AM | **START TRADING** | `ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live` |
| 9:15-3:30 PM | Monitor logs | `tail -f logs/engine.log` |
| 3:35 PM | Run nightly jobs | `go run ./cmd/engine --mode nightly` |
| 3:40 PM | Analyze results | `python3 scripts/analyze_daily_results.py --date today` |

**For Server Deployment:**
- Use systemd service for automatic daily runs
- Or use cron job for scheduled execution
- Or use Docker for cloud deployment

Now you're ready for production trading! 🚀

