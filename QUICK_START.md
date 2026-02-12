# Quick Start Guide - Algo Trading Engine

## TL;DR - One Command to Setup Everything

```bash
./scripts/setup.sh
```

That's it! The script will:
- Detect your operating system
- Install all dependencies (Go, Python3, PostgreSQL)
- Validate your configuration
- Set up the database
- Ask what deployment type you need
- Configure automated jobs
- Verify everything works

## Common Workflows

### Development/Testing (Local Machine)

```bash
# 1. Initial setup
./scripts/setup.sh
# When asked, choose: "1) Local machine"

# 2. Start the engine
./scripts/start_engine.sh
# Follow the prompts, confirm when ready

# 3. Monitor in another terminal
./scripts/health_check.sh --verbose

# 4. Stop when done
./scripts/stop_engine.sh
```

### Server Deployment (Production)

```bash
# 1. SSH to your server
ssh user@server.com

# 2. Clone the repo and setup
cd algoTradingAgent
./scripts/setup.sh
# When asked, choose: "2) Server"

# 3. Cron jobs are automatically configured
crontab -l  # View scheduled tasks

# 4. Monitor engine
./scripts/health_check.sh

# 5. View logs
tail -f logs/market_open_*.log
```

### Docker Deployment

```bash
# 1. Setup Docker configuration
./scripts/setup.sh
# When asked, choose: "3) Docker"

# 2. Start the engine
docker-compose up -d

# 3. Monitor
docker-compose logs -f trading-engine

# 4. Stop
docker-compose down
```

### Cloud Deployment (AWS/GCP/Azure)

```bash
# 1. Prepare deployment package
./scripts/cloud_deploy.sh

# 2. Follow the guide in the output directory
# (Contains specific instructions for your cloud provider)

# 3. Deploy using the provider's tools

# 4. Access the deployment guide
cat cloud-deployment-*/AWS_DEPLOYMENT.md
cat cloud-deployment-*/GCP_DEPLOYMENT.md
cat cloud-deployment-*/AZURE_DEPLOYMENT.md
```

## Essential Commands

### Check Engine Status

```bash
# Full health report
./scripts/health_check.sh

# Verbose with details
./scripts/health_check.sh --verbose

# JSON format for monitoring
./scripts/health_check.sh --json
```

### View Logs

```bash
# Real-time engine logs
tail -f logs/engine_*.log

# Recent errors
grep ERROR logs/*.log | head -20

# All activity
tail -100 logs/engine_*.log
```

### Start/Stop Engine

```bash
# Start with safety checks
./scripts/start_engine.sh

# Start without prompts (force)
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

# View cron logs
grep CRON /var/log/syslog | tail -20
```

## Before You Start

### Prerequisites

1. **Configuration File**
   - Update `config/config.json` with your Dhan broker credentials
   - Set `trading_mode` to "paper" for testing, "live" for real trading
   - Verify `capital` and risk limits

2. **Database**
   - PostgreSQL should be accessible at the URL in config
   - Database credentials should be correct

3. **API Credentials**
   - Dhan client_id (from dhanhq.co)
   - Dhan access_token (from dhanhq.co)

### Configuration Checklist

```bash
# Verify config is valid
cat config/config.json | jq '.'

# Check trading mode
grep "trading_mode" config/config.json

# Check broker credentials are present
grep -A5 "broker_config" config/config.json
```

## Common Issues & Solutions

### "Config file not found"
```bash
# Make sure you're in project root
pwd  # Should end with /algoTradingAgent

# Check config exists
ls config/config.json
```

### "Database connection failed"
```bash
# Test connection
psql postgres://user:pass@localhost:5432/algo_trading -c "SELECT 1"

# Update config if needed
nano config/config.json
```

### "Go not installed"
```bash
# Install Go
./scripts/install_dependencies.sh
# Select "yes" when prompted for Go installation
```

### "Engine won't start"
```bash
# Check health
./scripts/health_check.sh --verbose

# Check logs
tail -50 logs/engine_*.log

# Verify config
grep -E "trading_mode|client_id" config/config.json
```

### "Cron jobs not running"
```bash
# Verify they're scheduled
crontab -l

# Check if today is trading day
./scripts/is_trading_day.sh && echo "Trading day" || echo "Non-trading day"

# Check system cron logs
tail -20 /var/log/syslog | grep CRON
```

## Directory Structure

```
algoTradingAgent/
├── scripts/
│   ├── setup.sh                    ← START HERE
│   ├── install_dependencies.sh     (Dependency installer)
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
├── SETUP_SYSTEM.md                 (Detailed documentation)
└── QUICK_START.md                  (This file)
```

## Trading Day Schedule

The system automatically:
- Fetches market data at **8:50 AM** IST
- Starts trading at **9:10 AM** IST
- Closes trades at **3:35 PM** IST
- Analyzes weekly results on **Friday at 4 PM** IST

Runs **Monday to Friday only**, skips weekends and holidays.

## Important Safety Notes

1. **Live Trading is Dangerous**
   - Always test with "paper" mode first
   - Start small when switching to "live"
   - Monitor regularly, don't leave it unattended
   - Have a stop loss and maximum daily loss limit

2. **Backup Your Configuration**
   ```bash
   cp config/config.json config/config.json.backup
   ```

3. **Monitor Logs Daily**
   ```bash
   tail -f logs/engine_*.log
   ```

4. **Keep Dependencies Updated**
   ```bash
   go mod update
   pip install --upgrade -r python_ai/requirements.txt
   ```

## Getting Help

### View Full Documentation
```bash
# Comprehensive setup guide
cat SETUP_SYSTEM.md

# README with all details
cat README.md
```

### Check Logs
```bash
# Setup logs
logs/setup_*.log

# Engine logs
logs/engine_*.log

# Health checks
logs/health_check_*.log
```

### Get System Status
```bash
./scripts/health_check.sh --verbose
```

## Next Steps

1. **Configure your API credentials**
   - Update `config/config.json`
   - Add Dhan client_id and access_token

2. **Run setup**
   ```bash
   ./scripts/setup.sh
   ```

3. **Test with paper trading**
   - Keep `trading_mode: "paper"` initially
   - Run `./scripts/start_engine.sh`
   - Verify it works for a few days

4. **Switch to live trading** (if confident)
   - Change `trading_mode: "live"` in config.json
   - Run `./scripts/start_engine.sh`
   - Type "I UNDERSTAND THE RISKS" when prompted
   - Monitor closely

## Support Resources

- **Main docs:** `SETUP_SYSTEM.md`
- **Full README:** `README.md`
- **Status check:** `./scripts/health_check.sh --verbose`
- **Logs:** `logs/` directory
- **Config:** `config/config.json`

## Quick Reference Cards

### Before Market Opens
```bash
./scripts/health_check.sh               # Verify system
tail -20 logs/engine_*.log              # Check for errors
./scripts/start_engine.sh               # Start trading
```

### During Market Hours
```bash
./scripts/health_check.sh --verbose     # Monitor health
tail -f logs/engine_*.log               # Watch activity
```

### After Market Closes
```bash
./scripts/stop_engine.sh                # Graceful shutdown
./scripts/health_check.sh               # Final health check
tail -50 logs/engine_*.log              # Review session
```

### Weekly Maintenance
```bash
crontab -l                              # Verify cron jobs
./scripts/health_check.sh --verbose     # Full system check
grep ERROR logs/*.log | wc -l           # Count errors
tail logs/weekly_analysis_*.log         # Review analysis
```

---

**Start with:** `./scripts/setup.sh`

**Questions?** See `SETUP_SYSTEM.md` for detailed documentation.
