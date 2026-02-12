# Comprehensive Setup System for Algo Trading Engine

This document describes the complete setup system for the trading engine, allowing you to deploy in any environment with a single command.

## Overview

The setup system provides:

1. **Automated dependency installation** for Go, Python3, PostgreSQL
2. **Multi-environment support** (local, server, Docker, cloud)
3. **Smart configuration validation** with safety checks
4. **Automated scheduling** via cron jobs or systemd
5. **Health monitoring** and diagnostics
6. **Cloud deployment preparation** for AWS, GCP, Azure

## Quick Start

### One-Command Setup

```bash
./scripts/setup.sh
```

This interactive script will:
1. Detect your OS (macOS/Linux/Windows WSL)
2. Install all dependencies
3. Validate configuration
4. Set up database
5. Guide you through deployment options
6. Configure automated jobs
7. Verify system health

## Scripts Overview

### 1. `setup.sh` - Main Setup Orchestrator

The entry point for all setup operations. Provides interactive guidance for the entire setup process.

**Features:**
- OS detection (Linux/Mac/Windows)
- Interactive deployment type selection
- Dependency validation
- Configuration validation
- Database setup
- Health check validation

**Usage:**
```bash
./scripts/setup.sh
```

**Output:** Configured system ready for operation

---

### 2. `install_dependencies.sh` - Dependency Manager

Installs all required tools and libraries.

**Installs:**
- Go 1.24+ (with go.mod/go.sum management)
- Python3 3.8+ (with pip dependencies)
- PostgreSQL client
- Docker (optional)

**OS Support:**
- macOS (via Homebrew)
- Ubuntu/Debian
- CentOS/RHEL/Fedora

**Usage:**
```bash
./scripts/install_dependencies.sh
```

**Features:**
- Detects already-installed tools
- Skips reinstallation if present
- Validates installation success
- Logs all operations

---

### 3. `start_engine.sh` - Smart Engine Startup

Safely starts the trading engine with validation and confirmations.

**Pre-startup Checks:**
- Engine binary availability (auto-builds if needed)
- Configuration validation
- Trading day verification
- Market data availability
- AI scores check
- **Live mode safety confirmations**

**Features:**
- Fetches market data if missing
- Validates API credentials
- Requires explicit confirmation for live trading
- Comprehensive logging
- Graceful error handling

**Usage:**
```bash
# Interactive startup with confirmations
./scripts/start_engine.sh

# Bypass confirmations (use with caution)
./scripts/start_engine.sh --force

# Test without actual trading
./scripts/start_engine.sh --dry-run
```

**Safety Features:**
- Weekend/holiday detection
- Live mode confirmation (requires typing specific text)
- Capital and risk parameter display
- Pre-flight validation
- Detailed logging

---

### 4. `stop_engine.sh` - Graceful Shutdown

Stops the engine cleanly with proper cleanup.

**Shutdown Process:**
1. Saves open positions
2. Sends SIGTERM (graceful shutdown)
3. Waits up to 30 seconds
4. Runs nightly batch jobs (after market close)
5. Cleans up temporary files
6. Validates successful shutdown

**Features:**
- Graceful shutdown with timeout
- Force kill option if graceful fails
- Automatic nightly job execution
- Resource cleanup
- Status validation

**Usage:**
```bash
# Graceful shutdown
./scripts/stop_engine.sh

# Force kill if graceful fails
./scripts/stop_engine.sh --force
```

---

### 5. `health_check.sh` - System Diagnostics

Comprehensive health monitoring for the trading engine.

**Checks:**
- Engine process status
- Recent log validation
- Configuration integrity
- Database connectivity
- System resources (CPU, memory)
- Disk space availability
- Dependencies installation
- AI output availability

**Modes:**
- Standard (returns 0=OK, 1=WARNING, 2=FAILED)
- Verbose (detailed output)
- JSON (for monitoring systems)

**Usage:**
```bash
# Standard health check
./scripts/health_check.sh

# Verbose output
./scripts/health_check.sh --verbose

# JSON output for monitoring systems
./scripts/health_check.sh --json

# Silent mode (no alerts)
./scripts/health_check.sh --no-alert
```

**Output:**
- Exit codes for scripting
- Colorized output for humans
- Detailed logs in `logs/health_check_*.log`
- JSON format for monitoring integration

---

### 6. `cron_setup.sh` - Automated Job Scheduling

Configures cron jobs for automated trading operations.

**Scheduled Tasks:**
| Time | Task | Days |
|------|------|------|
| 8:50 AM | Pre-market data fetch | Mon-Fri |
| 9:10 AM | Market open signal | Mon-Fri |
| 3:35 PM | Market close cleanup | Mon-Fri |
| 4:00 PM | Weekly analysis | Friday only |
| Hourly | Health checks | 9 AM-3 PM, Mon-Fri |

**Features:**
- Holiday awareness (checks holidays_2026.json)
- Weekend skipping
- Trading day validation
- Automatic log rotation
- Easy management (add/remove/list)

**Usage:**
```bash
# Add all cron jobs
./scripts/cron_setup.sh

# Remove all cron jobs
./scripts/cron_setup.sh --remove

# List current jobs
./scripts/cron_setup.sh --list
```

**Time Zone Note:**
All times are in IST (Indian Standard Time). Adjust in the script if needed:
```bash
# Edit the schedule variables to change times
schedule="50 8 * * 1-5"  # Adjust hours/minutes as needed
```

---

### 7. `docker_setup.sh` - Docker Configuration

Creates complete Docker deployment configuration.

**Creates:**
- `Dockerfile` - Multi-stage build with all dependencies
- `docker-compose.yml` - Full service orchestration
- `.dockerignore` - Exclude unnecessary files
- `docker.env` - Environment variable template
- Helper scripts (`docker_build.sh`, `docker_run.sh`)

**Services:**
- PostgreSQL database
- Trading engine container
- Health checks
- Volume mounts for persistence

**Usage:**
```bash
# Create Docker configuration
./scripts/docker_setup.sh

# Create and build image
./scripts/docker_setup.sh --build

# Then run:
docker-compose up -d

# Monitor:
docker-compose logs -f trading-engine

# Stop:
docker-compose down
```

**Features:**
- Multi-stage build optimization
- Health checks configured
- Automatic database migrations
- Volume persistence
- Easy scaling

---

### 8. `cloud_deploy.sh` - Cloud Deployment Preparation

Prepares code and infrastructure for cloud deployment.

**Outputs:**
- Cloud-ready source code
- Configuration templates
- Provider-specific deployment guides
- Infrastructure-as-code templates (Terraform)

**Supported Providers:**
- AWS (ECS Fargate, Lambda, EC2)
- Google Cloud (Cloud Run, GKE)
- Azure (Container Instances, App Service, AKS)

**Usage:**
```bash
# Prepare for all cloud providers
./scripts/cloud_deploy.sh

# AWS-specific (with CloudFormation templates)
./scripts/cloud_deploy.sh --aws

# GCP-specific (with gcloud commands)
./scripts/cloud_deploy.sh --gcp

# Azure-specific (with az CLI commands)
./scripts/cloud_deploy.sh --azure

# Create deployment package
./scripts/cloud_deploy.sh --package
```

**Output Structure:**
```
cloud-deployment-YYYYMMDD_HHMMSS/
├── src/                          # Ready-to-deploy source
├── config-templates/             # Configuration templates
├── iac/                         # Terraform templates
├── CLOUD_DEPLOYMENT.md          # General guide
├── AWS_DEPLOYMENT.md            # AWS guide
├── GCP_DEPLOYMENT.md            # GCP guide
└── AZURE_DEPLOYMENT.md          # Azure guide
```

---

## Deployment Workflows

### Local Development Setup

```bash
# 1. Run main setup
./scripts/setup.sh
# Choose: "1) Local machine"

# 2. Start engine
./scripts/start_engine.sh

# 3. Monitor health
./scripts/health_check.sh --verbose

# 4. Stop engine
./scripts/stop_engine.sh
```

### Server Deployment (Linux)

```bash
# 1. Run main setup
./scripts/setup.sh
# Choose: "2) Server"

# 2. Cron jobs are automatically configured

# 3. Or setup systemd service (if prompted)

# 4. Monitor health
./scripts/health_check.sh

# Check cron jobs
crontab -l

# View logs
tail -f logs/market_open_*.log
```

### Docker Deployment

```bash
# 1. Run main setup
./scripts/setup.sh
# Choose: "3) Docker"

# 2. Docker files are created

# 3. Build image
docker build -t algo-trading-engine:latest .

# 4. Start with docker-compose
docker-compose up -d

# 5. Monitor
docker-compose logs -f trading-engine

# 6. Stop
docker-compose down
```

### Cloud Deployment (AWS Example)

```bash
# 1. Prepare cloud package
./scripts/cloud_deploy.sh --aws

# 2. Follow AWS_DEPLOYMENT.md in output

# 3. Build and push image to ECR

# 4. Deploy to ECS/Fargate using CloudFormation

# 5. Monitor via CloudWatch
```

---

## Configuration

### Main Configuration File

**Location:** `config/config.json`

```json
{
  "active_broker": "dhan",
  "trading_mode": "live",
  "capital": 500000.00,
  "broker_config": {
    "dhan": {
      "client_id": "YOUR_CLIENT_ID",
      "access_token": "YOUR_ACCESS_TOKEN",
      "base_url": "https://api.dhan.co"
    }
  },
  "database_url": "postgres://user:pass@localhost:5432/algo_trading",
  "paths": {
    "ai_output_dir": "./ai_outputs",
    "market_data_dir": "./market_data",
    "log_dir": "./logs"
  }
}
```

**Required Fields:**
- `active_broker` - Broker to use (dhan)
- `trading_mode` - "live" or "paper"
- `capital` - Trading capital in currency
- `broker_config` - API credentials
- `database_url` - PostgreSQL connection string

### Environment Variables

For Docker and cloud deployments:

```bash
DATABASE_URL=postgres://user:pass@host:5432/db
DHAN_CLIENT_ID=your_client_id
DHAN_ACCESS_TOKEN=your_token
LOG_DIR=/app/logs
AI_OUTPUT_DIR=/app/ai_outputs
MARKET_DATA_DIR=/app/market_data
```

---

## Monitoring and Logs

### Log Locations

```
logs/
├── setup_*.log                 # Setup process logs
├── install_*.log               # Dependency installation logs
├── engine_*.log                # Engine execution logs
├── start_*.log                 # Startup process logs
├── stop_*.log                  # Shutdown process logs
├── health_check_*.log          # Health check logs
├── cron_setup_*.log            # Cron setup logs
├── premarket_*.log             # Pre-market job logs
├── market_open_*.log           # Market open job logs
├── market_close_*.log          # Market close job logs
└── weekly_analysis_*.log       # Weekly analysis logs
```

### View Logs

```bash
# Real-time engine logs
tail -f logs/engine_*.log

# Recent errors
grep ERROR logs/*.log | tail -20

# Specific date
tail -f logs/engine_*_2026-02-13.log

# Health check results
tail -f logs/health_check_*.log
```

### Monitoring Integration

Health check supports JSON output for monitoring systems:

```bash
# Get metrics in JSON
./scripts/health_check.sh --json > /var/www/metrics.json

# Integrate with monitoring dashboard
# Use the JSON output in your monitoring system
```

---

## Troubleshooting

### Common Issues

**Engine won't start:**
```bash
# Check prerequisites
./scripts/health_check.sh --verbose

# Check if already running
pgrep -f engine

# View logs
tail -50 logs/engine_*.log
```

**Database connection failed:**
```bash
# Test database connectivity
psql postgres://user:pass@host:5432/db -c "SELECT 1"

# Check database URL in config
grep database_url config/config.json

# Review setup log
tail logs/setup_*.log
```

**Cron jobs not running:**
```bash
# List current jobs
crontab -l

# Check if job is trading day
./scripts/is_trading_day.sh && echo "Trading day" || echo "Non-trading day"

# Check cron logs
grep CRON /var/log/syslog | tail -20
```

**Docker container won't start:**
```bash
# Check logs
docker-compose logs trading-engine

# Test database connectivity
docker-compose exec trading-engine psql $DATABASE_URL -c "SELECT 1"

# Rebuild image
docker-compose build --no-cache
```

---

## Security Best Practices

1. **API Credentials:**
   - Never commit credentials to git
   - Use environment variables or secrets management
   - Rotate tokens regularly
   - Use cloud provider secret services (Secrets Manager, Key Vault, Secret Manager)

2. **Database:**
   - Use strong passwords
   - Enable SSL connections
   - Restrict network access
   - Regular backups

3. **Logging:**
   - Avoid logging sensitive data
   - Implement log rotation
   - Secure log file permissions
   - Use centralized logging in production

4. **Deployment:**
   - Use non-root containers
   - Scan images for vulnerabilities
   - Enable health checks
   - Use network isolation

---

## Advanced Configuration

### Custom Cron Times

Edit `cron_setup.sh` to modify job schedules:

```bash
# Find these lines and modify:
local schedule="50 8 * * 1-5"  # 8:50 AM
local schedule="10 9 * * 1-5"  # 9:10 AM
local schedule="35 15 * * 1-5" # 3:35 PM
```

Cron format: `minute hour day month day_of_week command`

### Systemd Service (Linux)

If `setup.sh` creates a systemd service:

```bash
# Check status
sudo systemctl status algo-trading-engine

# View logs
sudo journalctl -u algo-trading-engine -f

# Restart
sudo systemctl restart algo-trading-engine

# Enable on boot
sudo systemctl enable algo-trading-engine
```

### Custom Paths

To use different paths, update `config/config.json`:

```json
"paths": {
  "ai_output_dir": "/custom/path/ai_outputs",
  "market_data_dir": "/custom/path/market_data",
  "log_dir": "/custom/path/logs"
}
```

Then create directories:
```bash
mkdir -p /custom/path/{ai_outputs,market_data,logs}
```

---

## Performance Tuning

### Database Optimization

```bash
# Update connection pool settings in config.json
# Increase for high-volume trading
"db_pool_size": 10,
"db_max_idle": 5
```

### Engine Optimization

```bash
# Polling interval (in config.json)
"polling_interval_minutes": 5  # Adjust based on latency requirements

# Market timeout
"market_timeout_seconds": 30
```

### Resource Allocation

**Docker:**
```yaml
trading-engine:
  mem_limit: 1g
  cpus: '1.0'
```

**Cloud Run:**
```bash
gcloud run deploy --memory 512Mi --cpu 1
```

---

## Support and Debugging

### Enable Debug Logging

Set log level in configuration:

```json
"log_level": "DEBUG"
```

### Collect Diagnostics

```bash
# Create diagnostic bundle
mkdir -p diagnostics
cp -r logs/ diagnostics/
cp config/config.json diagnostics/config.json.backup
./scripts/health_check.sh --verbose > diagnostics/health_check.txt
./scripts/health_check.sh --json > diagnostics/metrics.json

# Archive
tar -czf diagnostics-$(date +%Y%m%d_%H%M%S).tar.gz diagnostics/
```

### Getting Help

1. **Check logs:** `logs/setup_*.log`, `logs/engine_*.log`
2. **Run health check:** `./scripts/health_check.sh --verbose`
3. **Review configuration:** `config/config.json`
4. **Check database:** Connect to PostgreSQL and verify tables
5. **Monitor Docker:** `docker-compose logs -f`

---

## System Requirements

### Minimum

- **CPU:** 2 cores
- **RAM:** 2 GB
- **Disk:** 10 GB
- **Network:** Stable internet connection

### Recommended

- **CPU:** 4+ cores
- **RAM:** 4-8 GB
- **Disk:** 50+ GB (for market data)
- **Network:** 10+ Mbps

---

## File Structure

```
algoTradingAgent/
├── scripts/
│   ├── setup.sh                 # Main setup orchestrator
│   ├── install_dependencies.sh  # Dependency installer
│   ├── start_engine.sh          # Engine startup
│   ├── stop_engine.sh           # Engine shutdown
│   ├── health_check.sh          # Health monitoring
│   ├── cron_setup.sh            # Cron configuration
│   ├── docker_setup.sh          # Docker setup
│   ├── cloud_deploy.sh          # Cloud preparation
│   ├── is_trading_day.sh        # Trading day checker
│   ├── run_nightly.sh           # Nightly jobs
│   └── run_market.sh            # Market execution
├── config/
│   ├── config.json              # Main configuration
│   ├── holidays_2026.json       # Market holidays
│   └── dhan_instruments.json    # Broker instruments
├── logs/                        # Log directory
├── ai_outputs/                  # AI scores
├── market_data/                 # Market data cache
├── Dockerfile                   # Docker image definition
├── docker-compose.yml           # Docker orchestration
└── SETUP_SYSTEM.md             # This file
```

---

## Summary

This comprehensive setup system enables:

✓ **One-command initialization** - `./scripts/setup.sh`
✓ **Multi-environment support** - Local, server, Docker, cloud
✓ **Automated dependency management** - All tools installed automatically
✓ **Safety by default** - Live mode confirmations, validation checks
✓ **Smart scheduling** - Cron or systemd, trading-day aware
✓ **Comprehensive monitoring** - Health checks, logging, metrics
✓ **Cloud-ready** - AWS, GCP, Azure support with guides

Start with `./scripts/setup.sh` and follow the interactive prompts!
