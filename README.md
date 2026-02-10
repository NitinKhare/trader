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

Copy the example config and fill in your Dhan API credentials:

```bash
cp config/config.json.example config/config.json
```

Then edit `config/config.json`:

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
# All Go tests (engine + internal packages)
go test ./...

# E2E paper mode tests (full pipeline: scores → strategy → risk → orders)
go test -v ./cmd/engine/ -run TestPaperMode

# Config validation tests (includes live mode safety checks)
go test -v ./internal/config/

# Broker tests (paper + Dhan mocked)
go test -v ./internal/broker/

# Strategy tests
go test -v ./internal/strategy/

# Verbose all
go test -v ./...
```

### E2E Test Coverage

| Test | What it verifies |
|------|-----------------|
| `TestPaperMode_EndToEnd` | Full buy pipeline: BULL regime + high-score stocks bought, low-score stocks skipped |
| `TestPaperMode_ExitMonitoring` | Positions exited when regime flips to BEAR |
| `TestPaperMode_RiskRejection` | Risk manager blocks buys beyond max position limit (5/5 filled, 6th and 7th rejected) |

Tests use `ForceRunMarketHourJobs()` to bypass the IST market-hours check, so they pass at any time of day.

## Project Structure

```
/cmd/engine/              Main engine entry point + E2E tests
/internal/
  /broker/                Broker interface + Dhan API v2 + Paper broker
  /strategy/              Strategy interface + Trend Following swing strategy
  /risk/                  Hard risk guardrails (cannot be overridden)
  /market/                Market calendar + data management + Dhan data provider
  /scheduler/             Nightly/market-hour/weekly job scheduler
  /storage/               Database interface + Postgres/TimescaleDB implementation
  /config/                Configuration loading, validation, live mode checks
  /webhook/               HTTP server for Dhan order postback notifications
/python_ai/
  /features/              Technical indicator calculations
  /scoring/               Stock scoring + market regime detection
  /models/                ML model placeholder (rule-based for v1)
  /backtests/             Backtest runner
  /data/                  Dhan data fetcher (Python)
  run_scoring.py          Nightly scoring pipeline entry point
/db/migrations/           SQL schema (Postgres + TimescaleDB)
/config/
  config.json.example     Config template (copy to config.json)
  config.json             System configuration (git-ignored, contains secrets)
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

## Engine Capabilities

### Dhan Broker Integration (Task 1)

Full implementation of the Dhan v2 API (`internal/broker/dhan.go`):

| Method | Endpoint | Description |
|--------|----------|-------------|
| `PlaceOrder` | `POST /v2/orders` | Place BUY/SELL orders (LIMIT, MARKET, SL, SL-M) |
| `CancelOrder` | `DELETE /v2/orders/{id}` | Cancel a pending order by ID |
| `GetOrderStatus` | `GET /v2/orders/{id}` | Check order fill status |
| `GetFunds` | `GET /v2/fundlimit` | Query available cash, margin, balance |
| `GetHoldings` | `GET /v2/holdings` | List current delivery holdings |
| `GetPositions` | `GET /v2/positions` | List intraday/overnight positions |

- **Paper broker** (`internal/broker/paper.go`): Simulates instant fills at order price with full fund/holding tracking. Used for testing without real money.
- **Broker registry**: `broker.New("dhan", configJSON)` auto-selects the right implementation.
- Instrument file for NSE ticker-to-securityId mapping.

### Market-Hour Trade Execution Engine (Task 2)

Two market-hour jobs run during NSE hours (9:15 AM - 3:30 PM IST):

**`execute_trades` job** — Entry pipeline:
1. Reads today's `market_regime.json` (BULL/SIDEWAYS/BEAR)
2. Reads today's `stock_scores.json` (ranked by AI composite score)
3. Sorts stocks by rank, loads candle history for each
4. Runs the trend-following strategy on each stock
5. Validates BUY intents through risk manager
6. Places approved LIMIT orders via the broker
7. Tracks and decrements available capital across multiple buys

**`monitor_exits` job** — Exit pipeline:
1. Gets current holdings from broker
2. Loads today's regime + scores for each held stock
3. Evaluates exit conditions (BEAR regime, trend collapse below 0.3)
4. Places SELL orders for positions that should exit

**Strategy: Trend-Following Swing** (`trend_follow_v1`):
- Entry: BULL regime, trend >= 0.6, breakout >= 0.5, liquidity >= 0.4, risk <= 0.5
- Exit: BEAR regime or trend drops below 0.3
- Sizing: ATR-based stop-loss (2x ATR), 2:1 risk-reward ratio

### Order Postback Webhook (Task 3)

HTTP server (`internal/webhook/`) receives real-time order status updates from Dhan:

- Listens on configurable port/path (default: `:8080/webhook/dhan/order`)
- Parses Dhan postback JSON (TRANSIT, PENDING, TRADED, REJECTED, CANCELLED, EXPIRED)
- Maps to broker-agnostic `OrderUpdate` type
- Multiple callback handlers via `OnOrderUpdate()`
- Graceful shutdown

Enable in config:
```json
"webhook": {
  "enabled": true,
  "port": 8080,
  "path": "/webhook/dhan/order"
}
```

### Trade Logging to PostgreSQL (Task 4)

All trade actions persisted to Postgres (`internal/storage/postgres.go`):

| Table | Contents |
|-------|----------|
| `trade_records` | Open/closed trades: entry/exit price, P&L, strategy ID, status |
| `trade_logs` | Every engine decision: `BUY_PLACED`, `BUY_REJECTED`, `EXIT_PLACED`, etc. |
| `signal_records` | Strategy signals with approval/rejection details |
| `ai_scores` | AI stock scores per date (for backtesting) |
| `candles` | OHLCV market data (TimescaleDB hypertable for time-series queries) |

- Engine works without DB (logs a warning, continues without persistence)
- Connection pooling: pgx driver, 10 max / 5 idle connections
- DB errors are logged but never crash the engine

### Live Mode Switch with Safety (Task 6)

Multiple safety layers prevent accidental live trading:

**Double confirmation at startup** — both required or engine exits:
```bash
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live
```

**Live mode config validation** (`internal/config/config.go`):

| Rule | Live Mode Limit | Paper Mode |
|------|-----------------|------------|
| `broker_config[active_broker]` | Required | Not required |
| `max_open_positions` | <= 5 | No limit |
| `max_risk_per_trade_pct` | <= 2% | No limit |
| `max_capital_deployment_pct` | <= 70% | No limit |
| `database_url` | Required (trades must be logged) | Required |

Missing either `--confirm-live` flag or `ALGO_LIVE_CONFIRMED=true` env var prints a safety banner:
```
  ╔═══════════════════════════════════════════════════════════╗
  ║                    ⚠  LIVE MODE BLOCKED  ⚠                ║
  ╠═══════════════════════════════════════════════════════════╣
  ║  Live trading requires TWO explicit confirmations:       ║
  ║  1. CLI flag:   --confirm-live                            ║
  ║  2. Env var:    ALGO_LIVE_CONFIRMED=true                  ║
  ╚═══════════════════════════════════════════════════════════╝
```

### Risk Management

All trade intents pass through the risk manager (`internal/risk/risk.go`) before execution:

| Rule | Description |
|------|-------------|
| Mandatory stop loss | Every BUY must have a stop loss price |
| Max risk per trade | Single trade cannot risk more than N% of capital |
| Max open positions | Hard limit on concurrent positions |
| Max daily loss | Trading halts if daily loss exceeds threshold |
| Max capital deployment | Limits total capital deployed at once |
| Sufficient capital | Order value cannot exceed available cash |

EXIT intents always pass. SKIP/HOLD intents bypass validation.

## CLI Reference

### Commands

```bash
# Check system status (market hours, funds, mode)
go run ./cmd/engine --mode status

# Run nightly jobs (data sync, AI scoring, watchlist)
go run ./cmd/engine --mode nightly

# Run market-hour jobs in paper mode (default)
go run ./cmd/engine --mode market

# Run in live mode (double confirmation required)
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live

# Custom config path
go run ./cmd/engine --config /path/to/config.json --mode market
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `config/config.json` | Path to configuration file |
| `--mode` | `status` | Run mode: `nightly`, `market`, or `status` |
| `--confirm-live` | `false` | Required safety flag to enable live trading |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ALGO_TRADING_MODE` | Override config `trading_mode` (`paper` or `live`) |
| `ALGO_DATABASE_URL` | Override config `database_url` |
| `ALGO_ACTIVE_BROKER` | Override config `active_broker` |
| `ALGO_LIVE_CONFIRMED` | Must be `true` (with `--confirm-live`) to start live mode |

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
- Data API subscription required (Rs 499/month) — subscribe at Profile → DhanHQ Trading APIs
