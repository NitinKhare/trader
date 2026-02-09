# algoTradingAgent

AI-first swing trading system for Indian equity cash market (NSE, delivery only).

Rule-based strategies enhanced by AI scoring. Broker-agnostic. Local-first. Safety over profit.

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.24+ | Core engine, strategy, risk, broker |
| Python | 3.10+ | AI scoring, feature engineering, backtesting |
| PostgreSQL | 14+ | Trade/signal/candle storage |
| TimescaleDB | 2.x | Time-series extension for Postgres |
| Docker | 20+ | (Optional) Run Postgres/TimescaleDB in container |

## Quick Start

### 1. Clone and enter the project

```bash
cd /path/to/algoTradingAgent
```

### 2. Install Python dependencies

```bash
pip3 install -r python_ai/requirements.txt
```

### 3. Set up the database

**Option A: Docker (recommended)**

```bash
docker volume create algo_trading_pgdata

docker run -d \
  --name algo_trading_db \
  -p 5432:5432 \
  -e POSTGRES_USER=algo \
  -e POSTGRES_PASSWORD=algo123 \
  -e POSTGRES_DB=algo_trading \
  -v algo_trading_pgdata:/var/lib/postgresql/data \
  timescale/timescaledb:latest-pg16
```

Run migrations:

```bash
docker cp db/migrations/001_initial_schema.sql algo_trading_db:/tmp/schema.sql
docker exec algo_trading_db psql -U algo -d algo_trading -f /tmp/schema.sql
```

**Option B: Local Postgres**

```bash
./scripts/setup_db.sh "postgres://user:pass@localhost:5432/algo_trading?sslmode=disable"
```

### 4. Configure Dhan API credentials

Edit `config/config.json` and fill in your Dhan API credentials:

```json
{
  "active_broker": "dhan",
  "trading_mode": "paper",
  "capital": 500000.00,
  "broker_config": {
    "dhan": {
      "client_id": "YOUR_DHAN_CLIENT_ID",
      "access_token": "YOUR_DHAN_ACCESS_TOKEN",
      "base_url": "https://api.dhan.co",
      "instrument_file": "./config/dhan_instruments.json"
    }
  }
}
```

Get your credentials from [web.dhan.co](https://web.dhan.co) → Access DhanHQ APIs. Note: access tokens expire every 24 hours.

Alternatively, set environment variables:

```bash
export DHAN_CLIENT_ID="your_client_id"
export DHAN_ACCESS_TOKEN="your_access_token"
```

### 5. Build the instrument mapping (one-time)

This maps NSE ticker symbols to Dhan's numeric security IDs:

```bash
python3 scripts/fetch_instruments.py
```

Generates `config/dhan_instruments.json`. Re-run periodically if the NIFTY 50 composition changes.

### 6. Fetch market data from Dhan

**Option A: Python script (standalone, good for initial backfill)**

```bash
# Fetch 1 year of history for the entire universe
python3 -m python_ai.data.fetch_dhan --days-back 365

# Fetch specific symbols only
python3 -m python_ai.data.fetch_dhan --symbols RELIANCE,TCS,INFY --days-back 90

# Fetch with custom output directory
python3 -m python_ai.data.fetch_dhan --output-dir ./market_data --days-back 365
```

**Option B: Go engine nightly mode (automated, part of the pipeline)**

```bash
go run ./cmd/engine --config config/config.json --mode nightly
```

This automatically fetches data from Dhan, exports CSVs, and verifies AI outputs.

Both options save CSV files to `market_data/SYMBOL.csv`.

### 7. Run the nightly pipeline

Run after market close (6-8 PM IST). This syncs data, scores stocks, and prepares the watchlist:

```bash
./scripts/run_nightly.sh
```

Or run each step separately:

```bash
# Step 1: Sync market data + nightly engine jobs
go run ./cmd/engine --config config/config.json --mode nightly

# Step 2: AI scoring
python3 -m python_ai.run_scoring \
  --date 2026-02-08 \
  --output-dir ./ai_outputs \
  --data-dir ./market_data \
  --universe-file ./config/stock_universe.json
```

### 8. Run market-hour execution

Run during market hours (9:15 AM - 3:30 PM IST). Executes pre-planned trades and monitors exits:

```bash
./scripts/run_market.sh
```

Or directly:

```bash
go run ./cmd/engine --config config/config.json --mode market
```

### 9. Check system status

```bash
go run ./cmd/engine --config config/config.json --mode status
```

Shows: market open/closed, trading day, next session time, available funds, active mode.

## Running Tests

```bash
# All Go tests
go test ./internal/...

# Verbose output
go test -v ./internal/...
```

Covers: strategies, risk rules, broker (paper), config validation, market calendar, Dhan data provider (mocked).

## Project Structure

```
/cmd/engine/              Main engine entry point
/internal/
  /broker/                Broker interface + Paper broker + Dhan stub
  /strategy/              Strategy interface + Trend Following strategy
  /risk/                  Hard risk guardrails (cannot be overridden)
  /market/                Market calendar + data management + Dhan data provider
  /scheduler/             Nightly/market-hour/weekly job scheduler
  /storage/               Database interface + Postgres implementation
  /config/                Configuration loading and validation
/python_ai/
  /features/              Technical indicator calculations
  /scoring/               Stock scoring + market regime detection
  /models/                ML model placeholder (rule-based for v1)
  /backtests/             Backtest runner
  /data/                  Dhan data fetcher (Python)
  run_scoring.py          Nightly scoring pipeline entry point
/db/migrations/           SQL schema (Postgres + TimescaleDB)
/config/
  config.json             System configuration
  holidays_2026.json      NSE exchange holidays
  stock_universe.json     NIFTY 50 stock list
  dhan_instruments.json   Ticker → Dhan securityId mapping
/scripts/
  run_nightly.sh          Full nightly pipeline
  run_market.sh           Market-hour execution
  setup_db.sh             Database initialization
  fetch_instruments.py    Generate Dhan instrument mapping
/ai_outputs/              Generated AI scores (git-ignored)
/market_data/             OHLCV CSV files (git-ignored)
```

## Daily Workflow

```
Evening (after market close, 6-8 PM IST):
  1. Run: ./scripts/run_nightly.sh
     - Syncs market data from Dhan API
     - Runs AI scoring pipeline
     - Generates next-day watchlist
  2. Review scores in ai_outputs/<date>/

Next morning (during market hours, 9:15 AM - 3:30 PM IST):
  3. Run: ./scripts/run_market.sh
     - Executes pre-planned trades
     - Monitors open positions for exits

Check anytime:
  4. Run: go run ./cmd/engine --config config/config.json --mode status
```

## Architecture

```
Dhan API ──> Go DhanDataProvider ──> CSV files (market_data/)
                                         |
                                    Python AI Layer (offline scoring)
                                         |
                                    JSON/Parquet files (ai_outputs/)
                                         |
                                    Go Engine (strategies + risk)
                                         |
                                    Broker API (paper or live)
```

- **AI advises, rules decide** — AI generates scores, strategies apply deterministic rules
- **Broker-agnostic** — switch brokers by changing config, not code
- **Market data ≠ broker** — data fetching is separate from order execution
- **Safety over profit** — mandatory stop loss, hard risk limits, prefer not trading over bad trades
- **Explainable** — every trade has a reason, every decision is logged

## Dhan API Notes

- Historical data endpoint: `POST /v2/charts/historical`
- Rate limit: 10 requests/second (auto-throttled)
- Max 90 days per request (auto-chunked)
- Access tokens expire every 24 hours — refresh at [web.dhan.co](https://web.dhan.co)
- Postback URL available for order status webhooks (configured in Dhan dashboard)
# trader
