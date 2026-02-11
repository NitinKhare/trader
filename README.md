# algoTradingAgent

AI-first automated swing trading system for the Indian equity cash market (NSE, delivery only).

Rule-based strategies enhanced by AI scoring. Four independent strategies target different market conditions. Hard risk guardrails that cannot be overridden. Full backtesting, analytics, and live/paper execution in a single binary.

**Philosophy: AI advises, rules decide. Safety over profit.**

---

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Modes of Operation](#modes-of-operation)
- [Strategies](#strategies)
- [Risk Management](#risk-management)
- [Performance Analytics](#performance-analytics)
- [Backtesting](#backtesting)
- [Daily Workflow](#daily-workflow)
- [Project Structure](#project-structure)
- [Running Tests](#running-tests)
- [CLI Reference](#cli-reference)
- [Dhan API Notes](#dhan-api-notes)

---

## Features

### Trading Engine
- **4 independent strategies** targeting different market conditions (trend follow, mean reversion, breakout, momentum)
- **AI-driven stock scoring** with market regime detection (BULL / SIDEWAYS / BEAR)
- **Paper and live broker** support via a broker-agnostic interface
- **Dhan API v2 integration** for orders, holdings, funds, and historical data
- **Automated stop-loss orders** placed immediately after entry fill confirmation
- **Order status polling** with configurable interval and timeout
- **Continuous market-hour polling** for trade execution and exit monitoring
- **Position persistence and reconciliation** across engine restarts
- **Dynamic capital tracking** from live broker balance

### Risk Management
- **9 hard risk rules** that cannot be overridden by strategy or AI
- **Mandatory stop-loss** on every trade
- **Sector concentration limits** to prevent overexposure to a single sector
- **Max holding period** to force-exit stale positions
- **Daily loss circuit breaker** halts trading when loss threshold is breached

### Analytics and Backtesting
- **Full performance analytics** with win rate, Sharpe ratio, max drawdown, profit factor
- **Per-strategy breakdown** to compare strategy performance side by side
- **Equity curve generation** for visual analysis
- **Backtest mode** runs strategies against historical AI outputs day by day
- **Backtest simulates** SL/target hits, max hold period, and sector limits

### Infrastructure
- **PostgreSQL/TimescaleDB** for trade, signal, candle, and log storage
- **Webhook server** for real-time Dhan order postback notifications
- **Job scheduler** with nightly, market-hour, and weekly job types
- **Signal handling** (SIGINT/SIGTERM) for graceful shutdown
- **Graceful degradation** -- engine works without DB, logs warnings

### Safety
- **Live mode double confirmation** requires both CLI flag and environment variable
- **Live mode config validation** enforces stricter limits on positions, risk, and capital deployment
- **Every trade decision is logged** with full explainability (scores, reason, regime)

---

## Architecture

```
                     ┌─────────────────────────────────────────┐
                     │           Python AI Layer               │
                     │  (Feature engineering, scoring, regime) │
                     └───────────────┬─────────────────────────┘
                                     │ JSON files (ai_outputs/)
                                     v
┌──────────────┐    ┌────────────────────────────────────────────┐
│  Dhan API    │───>│             Go Trading Engine              │
│  (data)      │    │                                            │
└──────────────┘    │  ┌──────────┐  ┌──────────┐  ┌─────────┐  │
                    │  │Strategies│─>│   Risk   │─>│ Broker  │  │
                    │  │  (x4)    │  │ Manager  │  │Interface│  │
                    │  └──────────┘  └──────────┘  └────┬────┘  │
┌──────────────┐    │  ┌──────────┐  ┌──────────┐       │       │
│  PostgreSQL  │<───│  │Analytics │  │Scheduler │       │       │
│  TimescaleDB │    │  └──────────┘  └──────────┘       │       │
└──────────────┘    └───────────────────────────────┬────┘───────┘
                                                    │
                              ┌─────────────────────┴────────────┐
                              │                                  │
                         ┌────v────┐                      ┌──────v─────┐
                         │  Paper  │                      │   Dhan     │
                         │ Broker  │                      │ Live API   │
                         └─────────┘                      └────────────┘
```

- **AI advises, rules decide** -- AI generates scores, strategies apply deterministic rules
- **Broker-agnostic** -- switch brokers by changing config, not code
- **Market data and broker are separate** -- data fetching is independent of order execution
- **Safety over profit** -- mandatory stop loss, hard risk limits, prefer not trading over bad trades
- **Explainable** -- every trade has a reason, every decision is logged

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.24+ | Core engine, strategies, risk, broker, analytics |
| Python | 3.10+ | AI scoring, feature engineering |
| PostgreSQL | 14+ | Trade/signal/candle storage |
| TimescaleDB | 2.x | Time-series extension for Postgres |
| Docker | 20+ | (Optional) Run Postgres/TimescaleDB in container |

---

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

docker cp db/migrations/002_add_order_tracking.sql algo_trading_db:/tmp/tracking.sql
docker exec algo_trading_db psql -U algo -d algo_trading -f /tmp/tracking.sql
```

**Option B: Local Postgres**

```bash
./scripts/setup_db.sh "postgres://user:pass@localhost:5432/algo_trading?sslmode=disable"
```

### 4. Configure the engine

Copy the example config and fill in your details:

```bash
cp config/config.json.example config/config.json
```

Edit `config/config.json`:

```json
{
  "active_broker": "dhan",
  "trading_mode": "paper",
  "capital": 500000.00,
  "risk": {
    "max_risk_per_trade_pct": 1.0,
    "max_open_positions": 5,
    "max_daily_loss_pct": 3.0,
    "max_capital_deployment_pct": 80.0,
    "max_per_sector": 2,
    "max_hold_days": 15
  },
  "broker_config": {
    "dhan": {
      "client_id": "YOUR_DHAN_CLIENT_ID",
      "access_token": "YOUR_DHAN_ACCESS_TOKEN",
      "base_url": "https://api.dhan.co",
      "instrument_file": "./config/dhan_instruments.json"
    }
  },
  "database_url": "postgres://algo:algo123@localhost:5432/algo_trading?sslmode=disable",
  "market_calendar_path": "./config/holidays_2026.json",
  "paths": {
    "ai_output_dir": "./ai_outputs",
    "market_data_dir": "./market_data",
    "log_dir": "./logs"
  },
  "webhook": {
    "enabled": false,
    "port": 8080,
    "path": "/webhook/dhan/order"
  }
}
```

Get Dhan API credentials from [web.dhan.co](https://web.dhan.co) > Access DhanHQ APIs. Access tokens expire every 24 hours.

Alternatively, set environment variables:

```bash
export DHAN_CLIENT_ID="your_client_id"
export DHAN_ACCESS_TOKEN="your_access_token"
```

### 5. Build the instrument mapping (one-time)

Maps NSE ticker symbols to Dhan's numeric security IDs:

```bash
python3 scripts/fetch_instruments.py
```

Generates `config/dhan_instruments.json`. Re-run if the NIFTY 50 composition changes.

### 6. Fetch market data

```bash
# Python script (standalone, good for initial backfill)
python3 -m python_ai.data.fetch_dhan --days-back 365

# Or via the Go engine nightly mode
go run ./cmd/engine --config config/config.json --mode nightly
```

### 7. Run AI scoring

```bash
python3 -m python_ai.run_scoring \
  --date 2026-02-11 \
  --output-dir ./ai_outputs \
  --data-dir ./market_data \
  --universe-file ./config/stock_universe.json
```

### 8. Run the engine

```bash
# Paper mode (default, no real money)
go run ./cmd/engine --mode market

# Check system status
go run ./cmd/engine --mode status

# View performance analytics
go run ./cmd/engine --mode analytics

# Run backtest against historical data
go run ./cmd/engine --mode backtest
```

---

## Configuration

### Risk Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_risk_per_trade_pct` | float | 1.0 | Maximum risk per trade as % of capital |
| `max_open_positions` | int | 5 | Maximum concurrent open positions |
| `max_daily_loss_pct` | float | 3.0 | Maximum daily loss as % of capital (circuit breaker) |
| `max_capital_deployment_pct` | float | 80.0 | Maximum total capital deployed at once |
| `max_per_sector` | int | 2 | Maximum positions in the same sector (0 = disabled) |
| `max_hold_days` | int | 15 | Maximum days to hold a position (0 = disabled) |

### Sector Mapping

The stock universe (`config/stock_universe.json`) maps each NIFTY 50 stock to its sector:

```json
{
  "stocks": [
    {"symbol": "RELIANCE", "sector": "OIL_GAS"},
    {"symbol": "TCS", "sector": "IT"},
    {"symbol": "HDFCBANK", "sector": "BANKING"}
  ],
  "symbols": ["RELIANCE", "TCS", "HDFCBANK"]
}
```

Sectors covered: BANKING, IT, PHARMA, OIL_GAS, AUTO, FMCG, METALS, CEMENT, POWER, TELECOM, CONGLOMERATE, FINANCE, CONSUMER, INFRA, HEALTHCARE.

The sector map is loaded at engine startup and passed to the risk manager for concentration checks.

---

## Modes of Operation

### `--mode status`

Prints current system state: market hours, trading day status, next session time, available funds, active mode.

### `--mode nightly`

Runs after market close (6-8 PM IST). Three sequential jobs:

1. **sync_market_data** -- Fetches 1 year of OHLCV data from Dhan API for all NIFTY 50 stocks, exports as CSVs
2. **run_ai_scoring** -- Verifies AI scoring outputs exist for today
3. **generate_watchlist** -- Reads market regime, prepares next-day watchlist

### `--mode market`

Runs during market hours (9:15 AM - 3:30 PM IST). Two jobs run repeatedly at the configured polling interval:

1. **execute_trades** -- Reads AI scores and regime, runs all 4 strategies on ranked stocks, validates through risk manager, places approved orders, polls for fills, places automated stop-loss orders
2. **monitor_exits** -- Checks open positions for max hold period violations, runs strategy exit evaluation, places exit orders, cancels existing SL orders before exits

Supports continuous polling (`polling_interval_minutes > 0`) or single execution (`polling_interval_minutes = 0`).

### `--mode analytics`

Loads all closed trades from the database and prints a comprehensive performance report with win rate, Sharpe ratio, max drawdown, profit factor, hold time stats, and per-strategy breakdown.

### `--mode backtest`

Runs all strategies against historical AI outputs day by day:

1. Scans `ai_outputs/` for date folders with regime and scores files
2. Iterates chronologically, loading regime + scores for each day
3. Filters candle data up to each date (no look-ahead bias)
4. Runs all 4 strategies, validates through risk manager with sector limits
5. Simulates fills at day's close, checks SL/target hits from candle data
6. Applies max hold period, closes remaining positions at backtest end
7. Generates full analytics report

---

## Strategies

All strategies implement the same `Strategy` interface: stateless, deterministic, pure functions. AI scores advise, strategies apply rules. Each produces a `TradeIntent` that must pass risk management before becoming an order.

### Shared Technical Indicators

All strategies share a common set of indicators (`internal/strategy/indicators.go`):

| Indicator | Function | Description |
|-----------|----------|-------------|
| ATR | `CalculateATR(candles, period)` | Average True Range (volatility measure) |
| RSI | `CalculateRSI(candles, period)` | Relative Strength Index (momentum oscillator, 0-100) |
| SMA | `CalculateSMA(candles, period)` | Simple Moving Average |
| ROC | `CalculateROC(candles, period)` | Rate of Change (% price momentum) |
| Highest High | `HighestHigh(candles, period)` | Highest high over N periods |
| Lowest Low | `LowestLow(candles, period)` | Lowest low over N periods |
| Avg Volume | `AverageVolume(candles, period)` | Average volume over N periods |

### 1. Trend Follow (`trend_follow_v1`)

Targets strong uptrends in BULL markets.

| | Criteria |
|---|---|
| **Entry** | BULL regime, confidence >= 0.6, trend >= 0.6, breakout >= 0.5, liquidity >= 0.4, risk <= 0.5 |
| **Stop Loss** | Entry - 2.0 x ATR(14) |
| **Target** | Entry + 2.0 x risk (2:1 reward-risk) |
| **Exit** | BEAR regime or trend drops below 0.3 |
| **Sizing** | Risk-based: `max_risk_per_trade_pct / risk_per_share` |

### 2. Mean Reversion (`mean_reversion_v1`)

Buys oversold stocks in non-bear markets, targeting a bounce back to the mean.

| | Criteria |
|---|---|
| **Entry** | BULL/SIDEWAYS regime, trend < 0.4, RSI(14) < 35, price < 20-SMA, risk <= 0.6, liquidity >= 0.4 |
| **Stop Loss** | Entry - 1.5 x ATR(14) |
| **Target** | 20-day SMA (the mean) |
| **Exit** | BEAR regime, price > SMA, RSI > 65, or trend > 0.7 |
| **Sizing** | Risk-based |

### 3. Breakout (`breakout_v1`)

Buys on high-volume resistance breaks in BULL markets.

| | Criteria |
|---|---|
| **Entry** | BULL regime, breakout quality >= 0.7, price > 20-day high, volume > 1.5x average, trend >= 0.5, risk <= 0.4 |
| **Stop Loss** | Resistance level - 1.5 x ATR(14) |
| **Target** | Entry + 3.0 x risk_per_share |
| **Exit** | BEAR regime, trend < 0.3, price below entry (failed breakout) |
| **Sizing** | Risk-based |

### 4. Momentum (`momentum_v1`)

Buys top-ranked stocks with strong momentum in BULL markets.

| | Criteria |
|---|---|
| **Entry** | BULL regime, rank <= 5, trend >= 0.7, ROC(10) > 5%, breakout >= 0.6, liquidity >= 0.6, risk <= 0.3 |
| **Stop Loss** | Entry - 2.5 x ATR(14) |
| **Target** | Entry + 2.5 x risk_per_share |
| **Exit** | BEAR regime, ROC negative, rank > 10, or trend < 0.5 |
| **Sizing** | Risk-based |

### Strategy Selection by Market Condition

| Regime | Trend Follow | Mean Reversion | Breakout | Momentum |
|--------|:---:|:---:|:---:|:---:|
| BULL | Active | Active (oversold only) | Active | Active |
| SIDEWAYS | -- | Active (oversold only) | -- | -- |
| BEAR | Exit only | Exit only | Exit only | Exit only |

---

## Risk Management

The risk manager (`internal/risk/risk.go`) is the final gatekeeper before any order is placed. Risk rules are implemented in Go and **cannot be overridden** by strategy or AI.

### Risk Rules

| Rule | Description | Rejection Code |
|------|-------------|----------------|
| Mandatory stop loss | Every BUY must have a stop-loss price set | `MANDATORY_STOP_LOSS` |
| Valid stop loss | Stop loss must be below entry price for BUY orders | `INVALID_STOP_LOSS` |
| Max risk per trade | Single trade risk cannot exceed N% of total capital | `MAX_RISK_PER_TRADE` |
| Max open positions | Hard limit on concurrent positions | `MAX_OPEN_POSITIONS` |
| No duplicate positions | Cannot open two positions in the same stock | `DUPLICATE_POSITION` |
| Daily loss circuit breaker | Trading halts if daily loss (realized + unrealized) exceeds threshold | `MAX_DAILY_LOSS` |
| Max capital deployment | Total deployed capital cannot exceed N% of total capital | `MAX_CAPITAL_DEPLOYMENT` |
| Sufficient capital | Order value must not exceed available cash | `INSUFFICIENT_CAPITAL` |
| Sector concentration | Cannot hold more than N positions in the same sector | `MAX_SECTOR_CONCENTRATION` |

EXIT intents always pass risk checks. SKIP and HOLD intents bypass validation entirely.

### Max Holding Period

Positions held longer than `max_hold_days` are force-exited during the `monitor_exits` job. This check runs **before** strategy evaluation, so time exits override strategy HOLD decisions. The exit reason is logged as `max_hold_period`.

### Automated Stop-Loss Orders

After a BUY order is confirmed filled (via order status polling), the engine immediately places a SL-M (stop-loss market) sell order at the strategy's stop-loss price. The SL order ID is persisted in the database alongside the trade record. Before placing a strategy exit, the engine cancels the existing SL order to prevent a double sell.

### Dynamic Capital Updates

On each trading run, the risk manager's capital base is updated from the live broker's total balance. This ensures risk percentages automatically adjust when money is added to or withdrawn from the account.

### Position Reconciliation

On startup, the engine loads open trades from the database and reconciles them with actual broker holdings:

- **In DB but not in broker** -- Marked as closed (`external_close`) -- likely manually sold or stopped out
- **In broker but not in DB** -- Logged as warning -- position lacks SL/target context
- **Quantity mismatch** -- Logged as warning -- partial close detected

---

## Performance Analytics

The analytics package (`internal/analytics/`) computes comprehensive performance metrics from closed trade records.

### Metrics

| Category | Metrics |
|----------|---------|
| **Trade Summary** | Total trades, winning trades, losing trades, win rate (%) |
| **Profit & Loss** | Total P&L, average P&L, gross profit, gross loss, profit factor |
| **Risk** | Max drawdown (absolute and %), Sharpe ratio (annualized with sqrt(252)) |
| **Hold Time** | Average hold days, min hold days, max hold days |
| **Strategy Breakdown** | Per-strategy: trades, win rate, P&L, avg hold time |

### Usage

```bash
# From live/paper trades in the database
go run ./cmd/engine --mode analytics

# From backtest (automatically printed at the end)
go run ./cmd/engine --mode backtest
```

The report is printed in a formatted text format with INR currency values.

---

## Backtesting

The backtest mode runs the full strategy + risk pipeline against historical data without placing real orders.

### How It Works

1. Scans `ai_outputs/` for date directories containing both `market_regime.json` and `stock_scores.json`
2. Sorts dates chronologically and iterates day by day
3. For each day:
   - Loads the AI regime and scores
   - Filters candle history up to that date (prevents look-ahead bias)
   - Runs all 4 strategies on each scored stock
   - Validates BUY intents through the risk manager (including sector limits)
   - Simulates fills at the day's close price
   - Checks if any open position's SL or target was hit using that day's candle (Low/High)
   - Checks max hold period and force-exits stale positions
4. At the end, closes any remaining open positions at the last known price
5. Generates a full analytics report

### Running a Backtest

```bash
# Ensure you have AI outputs and market data first
go run ./cmd/engine --mode backtest
```

The backtest uses the same config file as live trading, so risk parameters, sector limits, and max hold days are applied consistently.

---

## Daily Workflow

```
Evening (after market close, 6-8 PM IST):
  1. Run: ./scripts/run_nightly.sh
     - Syncs market data from Dhan API (1 year of OHLCV for 50 stocks)
     - Runs AI scoring pipeline (regime + stock scores)
     - Generates next-day watchlist
  2. Review scores in ai_outputs/<date>/

Next morning (during market hours, 9:15 AM - 3:30 PM IST):
  3. Run: ./scripts/run_market.sh
     - Executes pre-planned trades from all 4 strategies
     - Monitors open positions for exits (strategy + max hold period)
     - Places automated stop-loss orders after fills
     - Continuously polls at configured interval

After market close:
  4. View analytics: go run ./cmd/engine --mode analytics
     - Win rate, Sharpe ratio, drawdown, strategy breakdown

Anytime:
  5. Check status: go run ./cmd/engine --mode status
  6. Run backtest: go run ./cmd/engine --mode backtest
```

---

## Project Structure

```
algoTradingAgent/
|-- cmd/engine/
|   |-- main.go                    Entry point, job orchestration, all 5 modes
|   |-- main_test.go               E2E tests (paper mode full pipeline)
|
|-- internal/
|   |-- analytics/
|   |   |-- analytics.go           Performance metrics, Sharpe, drawdown, reporting
|   |   |-- analytics_test.go      12 tests
|   |
|   |-- broker/
|   |   |-- broker.go              Broker interface (PlaceOrder, GetHoldings, GetFunds, etc.)
|   |   |-- dhan.go                Dhan API v2 implementation
|   |   |-- dhan_test.go           Dhan API tests with mocked HTTP
|   |   |-- paper.go               Paper broker (simulated fills, fund tracking)
|   |   |-- paper_test.go          Paper broker tests
|   |
|   |-- config/
|   |   |-- config.go              Config loading, validation, live mode safety checks
|   |   |-- config_test.go         Config validation tests
|   |
|   |-- market/
|   |   |-- calendar.go            IST market hours, holidays, trading day checks
|   |   |-- calendar_test.go       Calendar tests
|   |   |-- data.go                Market data management, CSV export
|   |   |-- dhan_data.go           Dhan historical data provider
|   |   |-- dhan_data_test.go      Data provider tests
|   |
|   |-- risk/
|   |   |-- risk.go                Risk manager (9 rules including sector concentration)
|   |   |-- risk_test.go           19 tests (all rules, sector limits, capital updates)
|   |
|   |-- scheduler/
|   |   |-- scheduler.go           Job scheduler (nightly, market-hour, weekly)
|   |
|   |-- storage/
|   |   |-- storage.go             Store interface (trades, logs, signals, candles, analytics queries)
|   |   |-- postgres.go            PostgreSQL/TimescaleDB implementation
|   |   |-- storage_test.go        Storage tests
|   |
|   |-- strategy/
|   |   |-- strategy.go            Strategy interface, types (TradeIntent, StockScore, Candle, etc.)
|   |   |-- indicators.go          Shared indicators (ATR, RSI, SMA, ROC, HighestHigh, etc.)
|   |   |-- indicators_test.go     18 indicator tests
|   |   |-- trend_follow.go        Trend Following strategy
|   |   |-- trend_follow_test.go   Trend follow tests
|   |   |-- mean_reversion.go      Mean Reversion strategy
|   |   |-- mean_reversion_test.go 8 tests
|   |   |-- breakout.go            Breakout strategy
|   |   |-- breakout_test.go       8 tests
|   |   |-- momentum.go            Momentum strategy
|   |   |-- momentum_test.go       8 tests
|   |
|   |-- webhook/
|       |-- webhook.go             Dhan order postback HTTP server
|       |-- webhook_test.go        Webhook tests
|
|-- python_ai/
|   |-- features/                  Technical indicator calculations
|   |-- scoring/                   Stock scoring + market regime detection
|   |-- models/                    ML model placeholder (rule-based for v1)
|   |-- backtests/                 Python-side backtest runner
|   |-- data/                      Dhan data fetcher (Python)
|   |-- run_scoring.py             Nightly scoring pipeline entry point
|
|-- db/migrations/
|   |-- 001_initial_schema.sql     Base schema (trades, logs, signals, candles)
|   |-- 002_add_order_tracking.sql Order ID and SL order tracking columns
|
|-- config/
|   |-- config.json.example        Config template (copy to config.json)
|   |-- config.json                System config (git-ignored, contains secrets)
|   |-- holidays_2026.json         NSE exchange holidays
|   |-- stock_universe.json        NIFTY 50 stocks with sector mapping (50 stocks, 15 sectors)
|   |-- dhan_instruments.json      Ticker to Dhan security ID mapping
|
|-- scripts/
|   |-- run_nightly.sh             Full nightly pipeline
|   |-- run_market.sh              Market-hour execution
|   |-- setup_db.sh                Database initialization
|   |-- fetch_instruments.py       Generate Dhan instrument mapping
|
|-- ai_outputs/                    Generated AI scores by date (git-ignored)
|-- market_data/                   OHLCV CSV files (git-ignored)
```

---

## Running Tests

```bash
# All tests (10 packages)
go test ./...

# With verbose output
go test -v ./...

# Individual packages
go test -v ./internal/strategy/     # 4 strategies + indicators (42+ tests)
go test -v ./internal/risk/         # Risk manager (19 tests)
go test -v ./internal/analytics/    # Analytics (12 tests)
go test -v ./internal/broker/       # Broker tests (paper + Dhan mocked)
go test -v ./internal/config/       # Config validation
go test -v ./cmd/engine/            # E2E paper mode tests

# E2E test: full pipeline (scores -> strategy -> risk -> orders)
go test -v ./cmd/engine/ -run TestPaperMode
```

### E2E Test Coverage

| Test | What It Verifies |
|------|-----------------|
| `TestPaperMode_EndToEnd` | Full buy pipeline: BULL regime + high-score stocks bought, low-score stocks skipped |
| `TestPaperMode_ExitMonitoring` | Positions exited when regime flips to BEAR |
| `TestPaperMode_RiskRejection` | Risk manager blocks buys beyond max position limit |
| `TestPaperMode_DynamicCapital` | Capital updates from broker balance affect risk limits |
| `TestPaperMode_SLOrderPlacement` | Stop-loss orders placed after entry fill confirmation |
| `TestPaperMode_ContinuousPolling` | Continuous market-hour polling works correctly |

Tests use `ForceRunMarketHourJobs()` to bypass the IST market-hours check, so they pass at any time of day.

---

## CLI Reference

### Commands

```bash
# Check system status
go run ./cmd/engine --mode status

# Run nightly pipeline
go run ./cmd/engine --mode nightly

# Run market-hour trading (paper mode)
go run ./cmd/engine --mode market

# Run market-hour trading (live mode, double confirmation)
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live

# View performance analytics from closed trades
go run ./cmd/engine --mode analytics

# Run backtest against historical data
go run ./cmd/engine --mode backtest

# Custom config path
go run ./cmd/engine --config /path/to/config.json --mode market
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `config/config.json` | Path to configuration file |
| `--mode` | `status` | Run mode: `nightly`, `market`, `status`, `analytics`, `backtest` |
| `--confirm-live` | `false` | Required safety flag to enable live trading |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ALGO_TRADING_MODE` | Override config `trading_mode` (`paper` or `live`) |
| `ALGO_DATABASE_URL` | Override config `database_url` |
| `ALGO_ACTIVE_BROKER` | Override config `active_broker` |
| `ALGO_LIVE_CONFIRMED` | Must be `true` (with `--confirm-live`) to start live mode |
| `DHAN_CLIENT_ID` | Override Dhan client ID |
| `DHAN_ACCESS_TOKEN` | Override Dhan access token |

---

## Dhan API Notes

- Historical data endpoint: `POST /v2/charts/historical`
- Rate limit: 10 requests/second (auto-throttled)
- Max 90 days per request (auto-chunked for longer periods)
- Access tokens expire every 24 hours -- refresh at [web.dhan.co](https://web.dhan.co)
- Postback URL available for order status webhooks (configured in Dhan dashboard)
- Data API subscription required (Rs 499/month) -- subscribe at Profile > DhanHQ Trading APIs

### Dhan Broker Methods

| Method | Endpoint | Description |
|--------|----------|-------------|
| `PlaceOrder` | `POST /v2/orders` | Place BUY/SELL orders (LIMIT, MARKET, SL, SL-M) |
| `CancelOrder` | `DELETE /v2/orders/{id}` | Cancel a pending order by ID |
| `GetOrderStatus` | `GET /v2/orders/{id}` | Check order fill status |
| `GetFunds` | `GET /v2/fundlimit` | Query available cash, margin, balance |
| `GetHoldings` | `GET /v2/holdings` | List current delivery holdings |
| `GetPositions` | `GET /v2/positions` | List intraday/overnight positions |

### Database Tables

| Table | Contents |
|-------|----------|
| `trades` | Open/closed trades: entry/exit price, P&L, strategy ID, SL order ID, status |
| `trade_logs` | Every engine decision: `BUY_PLACED`, `BUY_REJECTED`, `EXIT_PLACED`, etc. |
| `signal_records` | Strategy signals with approval/rejection details |
| `ai_scores` | AI stock scores per date (for backtesting) |
| `candles` | OHLCV market data (TimescaleDB hypertable for time-series queries) |

### Live Mode Safety

Both of the following are required to start live trading, or the engine exits with a safety banner:

```bash
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live
```

Live mode config validation enforces:

| Rule | Limit |
|------|-------|
| Broker config | Must exist for the active broker |
| Max open positions | <= 5 |
| Max risk per trade | <= 2% |
| Max capital deployment | <= 70% |
| Database URL | Required (all trades must be logged) |
