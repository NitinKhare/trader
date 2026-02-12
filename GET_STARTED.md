# 🚀 Get Started in 3 Steps

## Step 1: Update Your Configuration (1 minute)

Edit `config/config.json` and ensure:

```bash
nano config/config.json
```

Verify these fields have YOUR Dhan credentials:

```json
{
  "active_broker": "dhan",
  "trading_mode": "paper",           ← START WITH PAPER FOR TESTING
  "capital": 500000.00,              ← Your capital
  "broker_config": {
    "dhan": {
      "client_id": "YOUR_CLIENT_ID",           ← FILL THIS
      "access_token": "YOUR_ACCESS_TOKEN",     ← FILL THIS
      "base_url": "https://api.dhan.co",
      "instrument_file": "./config/dhan_instruments.json"
    }
  }
}
```

---

## Step 2: Run One-Command Setup (5-10 minutes)

```bash
cd /path/to/algoTradingAgent
./scripts/setup.sh
```

The script will:

1. ✅ Detect your operating system
2. ✅ Ask if this is for local machine or server deployment
3. ✅ Install all dependencies (Go, Python, PostgreSQL client)
4. ✅ Validate your configuration
5. ✅ Set up the database
6. ✅ Configure automated jobs
7. ✅ Verify everything works

**Example output:**

```
🔍 Detecting OS...
✓ Detected: MacOS

📋 Deployment Type Selection
1) Local machine (testing)
2) Server deployment (production with cron)
3) Docker deployment
4) Cloud deployment

Choose an option [1]: 1

📦 Installing dependencies...
✓ Go is installed
✓ Python3 is installed
✓ PostgreSQL client is installed

✓ Setup complete! You're ready to trade!
```

---

## Step 3: Start Trading Engine

### For Testing (Paper Mode)

```bash
./scripts/start_engine.sh
```

The engine will:
- Fetch today's market data
- Generate AI scores
- Start trading with simulated capital
- Run automatically every 5 minutes
- Log all activity

**Watch it run:**

```bash
# In another terminal
tail -f logs/engine_*.log
```

**Stop it:**

```bash
./scripts/stop_engine.sh
```

---

## Testing Workflow (Local Machine)

### Day 1: Setup & Verify

```bash
# 1. Update config
nano config/config.json

# 2. Run setup
./scripts/setup.sh

# 3. Check system health
./scripts/health_check.sh --verbose

# Expected output:
# ✓ Engine status: ready
# ✓ Database: connected
# ✓ Config: valid
# ✓ Dependencies: installed
```

### Day 2-5: Paper Trading

```bash
# Each morning during market hours
./scripts/start_engine.sh

# Monitor in separate terminal
tail -f logs/engine_*.log | grep "\[funds\]\|\[trade\]"

# Each evening
./scripts/stop_engine.sh

# Weekly analysis
./scripts/health_check.sh --verbose
```

### After 1 week: Check Results

```bash
# See all trades from the week
tail -500 logs/engine_*.log | grep "trade\|P&L"

# If profitable, proceed to next step
# If losing, investigate or optimize strategies
```

---

## Server/Cloud Deployment Workflow

### For Server (Automatic Daily Runs)

During setup step 2, choose "2) Server deployment"

The system will:
- Install systemd service (if Linux)
- OR setup cron jobs (if Mac/Linux)
- Automatically run at:
  - **8:50 AM**: Fetch market data
  - **9:10 AM**: Start trading engine
  - **3:35 PM**: Stop engine and run analysis
  - **Friday 4 PM**: Weekly analysis

### Monitor Remotely

```bash
# SSH to your server
ssh user@server.com

# Check status
./scripts/health_check.sh

# View today's trades
tail -100 logs/engine_*.log

# View scheduled jobs
crontab -l
```

---

## Docker Deployment (Cloud-Ready)

### During setup, choose "3) Docker deployment"

The system will:
- Create `Dockerfile` and `docker-compose.yml`
- Build the image
- Create containers for engine and database

### Run with Docker

```bash
# Start the engine
docker-compose up -d

# Monitor
docker-compose logs -f trading-engine

# Stop
docker-compose down
```

---

## Cloud Deployment (AWS/GCP/Azure)

### Prepare deployment package

```bash
./scripts/cloud_deploy.sh
```

This generates:
- `cloud-deployment-*/AWS_DEPLOYMENT.md`
- `cloud-deployment-*/GCP_DEPLOYMENT.md`
- `cloud-deployment-*/AZURE_DEPLOYMENT.md`

### Follow the guides

```bash
# Read the guide for your cloud provider
cat cloud-deployment-AWS/AWS_DEPLOYMENT.md

# Deploy using the instructions
# (Includes Terraform templates if you want them)
```

---

## Essential Commands

### Check System Health

```bash
# Quick status
./scripts/health_check.sh

# Detailed report
./scripts/health_check.sh --verbose

# JSON format (for monitoring tools)
./scripts/health_check.sh --json
```

### View Activity

```bash
# Real-time logs
tail -f logs/engine_*.log

# Recent trades
grep "trade\|order" logs/engine_*.log | tail -20

# Error investigation
grep "ERROR\|error" logs/*.log
```

### Start/Stop Engine

```bash
# Normal startup (with safety checks)
./scripts/start_engine.sh

# Startup with minimal checks
./scripts/start_engine.sh --force

# Test run (no real trading)
./scripts/start_engine.sh --dry-run

# Graceful stop
./scripts/stop_engine.sh

# Force stop if needed
./scripts/stop_engine.sh --force
```

### Manage Scheduled Jobs

```bash
# Setup automatic scheduling
./scripts/cron_setup.sh

# List current jobs
./scripts/cron_setup.sh --list

# Remove all jobs
./scripts/cron_setup.sh --remove
```

---

## Configuration Examples

### Paper Mode (Testing)
```json
{
  "trading_mode": "paper",
  "capital": 500000.00,
  "max_capital_deployment_pct": 80.0
}
```

### Live Mode (Real Money)
```json
{
  "trading_mode": "live",
  "capital": 500000.00,
  "max_capital_deployment_pct": 70.0,
  "risk": {
    "max_risk_per_trade_pct": 1.0,
    "max_open_positions": 5,
    "max_daily_loss_pct": 3.0
  }
}
```

---

## Switching from Testing to Live Trading

### Prerequisites

1. ✅ Run paper trading for at least 1 week
2. ✅ Verify results match backtest expectations
3. ✅ Understand all risk limits
4. ✅ Backup your configuration
5. ✅ Have a stop loss plan

### Switch to Live

```bash
# 1. Update config
nano config/config.json
# Change: "trading_mode": "paper" → "trading_mode": "live"

# 2. Start engine (will require confirmation)
./scripts/start_engine.sh

# 3. When prompted, type exactly:
# I UNDERSTAND THE RISKS

# 4. Carefully monitor
tail -f logs/engine_*.log
```

---

## Troubleshooting

### "Setup script fails"

```bash
# Run with verbose logging
bash -x ./scripts/setup.sh 2>&1 | tee setup_debug.log

# Check logs
cat logs/setup_*.log
```

### "Engine won't start"

```bash
# Detailed health check
./scripts/health_check.sh --verbose

# Check config validity
cat config/config.json | jq '.'

# Verify credentials
grep "client_id\|access_token" config/config.json
```

### "No trades being placed"

```bash
# Check AI data exists
ls ai_outputs/ | wc -l
# Should show multiple dates

# Check market data exists
ls market_data/ | head -5
# Should show CSV files for multiple dates

# Generate missing data
python3 python_ai/data/fetch_market_data.py --date today
python3 python_ai/scoring_engine.py --date today
```

### "Database connection failed"

```bash
# Test connection
psql postgres://user:pass@localhost:5432/algo_trading -c "SELECT 1"

# Update config if needed
nano config/config.json
# Check: "database_url" is correct
```

---

## Directory Structure

```
algoTradingAgent/
├── scripts/
│   ├── setup.sh                    ← RUN THIS FIRST
│   ├── start_engine.sh             (Start trading)
│   ├── stop_engine.sh              (Stop gracefully)
│   ├── health_check.sh             (System status)
│   ├── cron_setup.sh               (Auto scheduling)
│   ├── docker_setup.sh             (Docker config)
│   └── cloud_deploy.sh             (Cloud prep)
├── config/
│   ├── config.json                 ← UPDATE WITH YOUR CREDENTIALS
│   ├── holidays_2026.json
│   └── dhan_instruments.json
├── logs/                           (Created by setup)
├── ai_outputs/                     (Created by setup)
├── market_data/                    (Created by setup)
├── QUICK_START.md                  (Quick reference)
├── SETUP_SYSTEM.md                 (Detailed docs)
└── PRODUCTION_WORKFLOW.md          (Daily workflow)
```

---

## Quick Timeline

| Time | Action | Command |
|------|--------|---------|
| **Now** | Update config | `nano config/config.json` |
| **Now** | Run setup | `./scripts/setup.sh` |
| **Tomorrow (8:45 AM)** | Start trading | `./scripts/start_engine.sh` |
| **Tomorrow (9:15 AM - 3:30 PM)** | Monitor | `tail -f logs/engine_*.log` |
| **Tomorrow (3:35 PM)** | Stop trading | `./scripts/stop_engine.sh` |
| **Next Week** | Review results | `./scripts/health_check.sh --verbose` |
| **After 1 week** | Switch to live (if good) | Update config & restart |

---

## What Happens When You Run `./scripts/setup.sh`

```
┌─────────────────────────────────────────┐
│ ./scripts/setup.sh                      │
└─────────────────────────────────────────┘
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
   Detect OS              Check Config
   (Mac/Linux)            (Valid JSON?)
        │                       │
        └───────────┬───────────┘
                    ▼
        ┌─────────────────────────┐
        │ Deployment Type?        │
        ├─────────────────────────┤
        │ 1. Local machine        │
        │ 2. Server               │
        │ 3. Docker               │
        │ 4. Cloud                │
        └─────────────────────────┘
                    │
        ┌───────────┼───────────┬───────────┐
        ▼           ▼           ▼           ▼
    Local       Server      Docker      Cloud
    Setup       Setup       Setup       Setup
        │           │           │           │
        └───────────┼───────────┼───────────┘
                    ▼
        Install Dependencies
        (Go, Python, PostgreSQL)
                    │
                    ▼
        Validate Configuration
        (Check credentials)
                    │
                    ▼
        Setup Directories
        (logs, ai_outputs, market_data)
                    │
                    ▼
        Initialize Database
                    │
                    ▼
        Run System Checks
                    │
                    ▼
        ✅ READY TO TRADE
```

---

## Getting Help

### Check Documentation

```bash
# This file (quick start)
cat GET_STARTED.md

# Comprehensive setup guide
cat SETUP_SYSTEM.md

# Daily workflow
cat PRODUCTION_WORKFLOW.md

# System status
./scripts/health_check.sh --verbose
```

### Review Logs

```bash
# Setup logs
cat logs/setup_*.log

# Engine logs
tail -100 logs/engine_*.log

# Error logs
grep ERROR logs/*.log
```

---

## Start Now! 🎯

```bash
# 1. Configure
nano config/config.json

# 2. Setup (ONE COMMAND)
./scripts/setup.sh

# 3. Start trading
./scripts/start_engine.sh

# 4. Monitor
./scripts/health_check.sh --verbose
```

That's it! You're ready to go! 🚀

---

**Next step:** `nano config/config.json` → `./scripts/setup.sh`
