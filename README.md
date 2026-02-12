# algoTradingAgent

AI-first automated swing trading system for the Indian equity cash market (NSE, delivery only).

Rule-based strategies enhanced by AI scoring. Nine independent strategies target different market conditions — trend following, mean reversion, breakout, momentum, VWAP reversion, EMA pullback, range breakout, MACD crossover, and Bollinger squeeze. Hard risk guardrails that cannot be overridden. Circuit breaker, trailing stop-loss, config hot-reload, and real-time postback processing. Dry-run mode for risk-free pipeline validation. Integrated Python AI scoring pipeline. Full backtesting, analytics, and live/paper execution in a single binary.

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
- [Circuit Breaker](#circuit-breaker)
- [Trailing Stop-Loss](#trailing-stop-loss)
- [Postback-Driven Position Updates](#postback-driven-position-updates)
- [Config Hot-Reload](#config-hot-reload)
- [Graceful Shutdown](#graceful-shutdown)
- [Dry-Run Mode](#dry-run-mode)
- [Dhan API Notes](#dhan-api-notes)

---

## Features

### Trading Engine
- **9 independent strategies** targeting different market conditions (trend follow, mean reversion, breakout, momentum, VWAP reversion, EMA pullback, opening range breakout, MACD crossover, Bollinger squeeze)
- **AI-driven stock scoring** with market regime detection (BULL / SIDEWAYS / BEAR)
- **Paper and live broker** support via a broker-agnostic interface
- **Dhan API v2 integration** for orders, holdings, funds, and historical data
- **Automated stop-loss orders** placed immediately after entry fill confirmation
- **Trailing stop-loss** adjusts SL upward as price moves in your favor to lock in profits
- **Order status polling** with configurable interval and timeout
- **Continuous market-hour polling** for trade execution and exit monitoring
- **Position persistence and reconciliation** across engine restarts
- **Dynamic capital tracking** from live broker balance
- **Postback-driven position updates** via Dhan webhook for real-time trade lifecycle management
- **Dry-run mode** validates the full pipeline without placing orders, prints a detailed report

### Risk Management
- **9 hard risk rules** that cannot be overridden by strategy or AI
- **Mandatory stop-loss** on every trade
- **Sector concentration limits** to prevent overexposure to a single sector
- **Max holding period** to force-exit stale positions
- **Daily loss circuit breaker** halts trading when loss threshold is breached
- **Circuit breaker** automatically halts trading on repeated order failures or API errors

### Analytics and Backtesting
- **Full performance analytics** with win rate, Sharpe ratio, max drawdown, profit factor
- **Per-strategy breakdown** to compare strategy performance side by side
- **Equity curve generation** for visual analysis
- **Backtest mode** runs strategies against historical AI outputs day by day
- **Backtest simulates** SL/target hits, max hold period, trailing SL, and sector limits

### Infrastructure
- **PostgreSQL/TimescaleDB** for trade, signal, candle, and log storage
- **Webhook server** for real-time Dhan order postback notifications
- **Postback-driven updates** process order fills, rejections, and cancellations in real-time
- **Job scheduler** with nightly, market-hour, and weekly job types
- **Graceful shutdown** waits for in-flight jobs to complete (30s timeout) before exiting
- **Config hot-reload** changes risk parameters without restarting the engine
- **Integrated Python AI pipeline** nightly mode executes Python scoring directly
- **Graceful degradation** -- engine works without DB, logs warnings

### Safety
- **Live mode double confirmation** requires both CLI flag and environment variable
- **Live mode config validation** enforces stricter limits on positions, risk, and capital deployment
- **Circuit breaker** with consecutive failure tracking, hourly sliding window, and cooldown auto-reset
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
    "max_hold_days": 15,
    "trailing_stop": {
      "enabled": true,
      "trail_pct": 1.5,
      "activation_pct": 2.0
    },
    "circuit_breaker": {
      "max_consecutive_failures": 3,
      "max_failures_per_hour": 10,
      "cooldown_minutes": 30
    }
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
    "log_dir": "./logs",
    "python_path": "python3",
    "stock_universe_file": "./config/stock_universe.json"
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

# Preview trades without placing orders
go run ./cmd/engine --mode dry-run
```

---

## Configuration

### Risk Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_risk_per_trade_pct` | float | 1.0 | Maximum risk per trade as % of capital |
| `max_open_positions` | int | 5 | Maximum concurrent open positions |
| `max_daily_loss_pct` | float | 3.0 | Maximum daily loss as % of capital (daily loss halt) |
| `max_capital_deployment_pct` | float | 80.0 | Maximum total capital deployed at once |
| `max_per_sector` | int | 2 | Maximum positions in the same sector (0 = disabled) |
| `max_hold_days` | int | 15 | Maximum days to hold a position (0 = disabled) |

### Trailing Stop Configuration

Nested under `risk.trailing_stop`:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | false | Enable trailing stop-loss adjustment |
| `trail_pct` | float | 1.5 | Trail distance as % below the high watermark |
| `activation_pct` | float | 2.0 | Minimum profit % before trailing activates (0 = immediate) |

### Circuit Breaker Configuration

Nested under `risk.circuit_breaker`:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_consecutive_failures` | int | 3 | Consecutive order failures before trip (0 = disabled) |
| `max_failures_per_hour` | int | 10 | Failures within a sliding 1-hour window before trip (0 = disabled) |
| `cooldown_minutes` | int | 30 | Minutes before auto-reset after trip (0 = manual reset only) |

### Paths Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `ai_output_dir` | string | -- | Directory where Python writes scoring outputs (required) |
| `market_data_dir` | string | -- | Directory for cached OHLCV market data |
| `log_dir` | string | -- | Directory for system logs |
| `python_path` | string | `"python3"` | Path to the Python interpreter for AI scoring |
| `stock_universe_file` | string | `"./config/stock_universe.json"` | Path to the stock universe JSON with sector mappings |

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
2. **run_ai_scoring** -- Executes the Python AI scoring pipeline (`python3 -m python_ai.run_scoring`), then verifies outputs exist
3. **generate_watchlist** -- Reads market regime, prepares next-day watchlist

### `--mode market`

Runs during market hours (9:15 AM - 3:30 PM IST). Two jobs run repeatedly at the configured polling interval:

1. **execute_trades** -- Reads AI scores and regime, checks circuit breaker status, runs all 4 strategies on ranked stocks, validates through risk manager, places approved orders, polls for fills, places automated stop-loss orders. Broker failures are tracked by the circuit breaker.
2. **monitor_exits** -- Checks open positions for max hold period violations, adjusts trailing stop-loss levels, runs strategy exit evaluation, places exit orders, cancels existing SL orders before exits.

Additional market-mode behaviors:
- **Circuit breaker** -- execute_trades is skipped entirely if the circuit breaker is tripped
- **Postback handler** -- If webhooks are enabled, order fill/reject/cancel events are processed in real-time
- **Config hot-reload** -- Risk parameters are watched for changes and applied without restart
- **Graceful shutdown** -- On SIGINT/SIGTERM, waits for in-flight jobs (up to 30s) before exiting

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

### `--mode dry-run`

Validates the full strategy + risk pipeline against today's AI outputs without placing any orders. Prints a detailed report showing which stocks would be bought, rejected, and why. See [Dry-Run Mode](#dry-run-mode) for full details.

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
| EMA | `CalculateEMA(candles, period)` | Exponential Moving Average (weights recent data more) |
| EMA Series | `CalculateEMASeries(candles, period)` | Full EMA series for each candle |
| VWAP | `CalculateVWAP(candles, period)` | Volume Weighted Average Price (fair value anchor) |
| MACD | `CalculateMACD(candles, fast, slow, signal)` | MACD line, signal line, and histogram |
| Bollinger Bands | `CalculateBollingerBands(candles, period, mult)` | Middle, upper, lower bands and bandwidth |
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

### 5. VWAP Reversion (`vwap_reversion_v1`)

Buys when price dips significantly below VWAP, expecting reversion to fair value. VWAP is a natural support level where institutional traders accumulate.

| | Criteria |
|---|---|
| **Entry** | BULL/SIDEWAYS regime, price >= 2% below VWAP(20), RSI(14) < 40, volatility <= 0.7, liquidity >= 0.5, risk <= 0.5 |
| **Stop Loss** | Entry - 1.5 x ATR(14) |
| **Target** | VWAP (fair value) |
| **Exit** | BEAR regime, price crosses above VWAP, RSI > 65 |
| **Sizing** | Risk-based |

### 6. EMA Pullback (`pullback_v1`)

Buys dips in an established uptrend. One of the most reliable swing setups — entering at a discount within a strong trend.

| | Criteria |
|---|---|
| **Entry** | BULL regime, trend >= 0.5, price > 50-EMA (uptrend), price within 1% of 20-EMA (pullback), RSI 40-60, risk <= 0.5 |
| **Stop Loss** | Entry - 2.0 x ATR(14) |
| **Target** | Entry + 2.5 x risk_per_share |
| **Exit** | BEAR regime, price below 50-EMA (uptrend broken), trend < 0.3 |
| **Sizing** | Risk-based |

### 7. Opening Range Breakout (`orb_v1`)

Identifies stocks consolidating in a tight range (ATR compression) and buys when they break out with volume. Tight consolidation precedes explosive directional moves.

| | Criteria |
|---|---|
| **Entry** | BULL regime, short ATR / long ATR < 0.6 (squeeze), price > consolidation high, volume > 1.3x avg, trend >= 0.4, breakout >= 0.5, risk <= 0.5 |
| **Stop Loss** | Consolidation low - 1.5 x short ATR |
| **Target** | Entry + 3.0 x risk_per_share |
| **Exit** | BEAR regime, price below entry (failed breakout), trend < 0.25 |
| **Sizing** | Risk-based |

### 8. MACD Crossover (`macd_crossover_v1`)

Buys on MACD bullish crossover (MACD line crosses above signal line), confirming a shift from bearish to bullish momentum. Standard parameters: 12/26/9.

| | Criteria |
|---|---|
| **Entry** | BULL regime, MACD > signal (fresh crossover), histogram > 0, trend >= 0.4, risk <= 0.5, liquidity >= 0.4 |
| **Stop Loss** | Entry - 2.0 x ATR(14) |
| **Target** | Entry + 2.0 x risk_per_share |
| **Exit** | BEAR regime, MACD bearish crossover (MACD < signal or histogram < 0), trend < 0.25 |
| **Sizing** | Risk-based |

### 9. Bollinger Band Squeeze (`bollinger_squeeze_v1`)

Identifies periods of extreme low volatility (narrow Bollinger Bands) that precede explosive moves. Buys when price breaks above the upper band after a squeeze.

| | Criteria |
|---|---|
| **Entry** | BULL/SIDEWAYS regime, prior bandwidth < 0.10 (squeeze), price > upper band, volume > 1.2x avg, trend >= 0.3, risk <= 0.5 |
| **Stop Loss** | Lower Bollinger Band |
| **Target** | Entry + 2.5 x risk_per_share |
| **Exit** | BEAR regime, price below middle band (momentum lost), trend < 0.2 |
| **Sizing** | Risk-based |

### Strategy Selection by Market Condition

| Regime | Trend Follow | Mean Reversion | Breakout | Momentum | VWAP Rev. | Pullback | ORB | MACD | Bollinger |
|--------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| BULL | Active | Active | Active | Active | Active | Active | Active | Active | Active |
| SIDEWAYS | -- | Active | -- | -- | Active | -- | -- | -- | Active |
| BEAR | Exit only | Exit only | Exit only | Exit only | Exit only | Exit only | Exit only | Exit only | Exit only |

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

After a BUY order is confirmed filled (via order status polling or webhook postback), the engine immediately places a SL-M (stop-loss market) sell order at the strategy's stop-loss price. The SL order ID is persisted in the database alongside the trade record. Before placing a strategy exit, the engine cancels the existing SL order to prevent a double sell.

When a webhook postback reports an SL order as TRADED, the engine automatically marks the position as closed in the database with `sl_hit` as the exit reason.

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
   - Adjusts trailing stop-loss using the candle's high (if enabled)
   - Checks if any open position's SL or target was hit using that day's candle (Low/High)
   - Checks max hold period and force-exits stale positions
4. At the end, closes any remaining open positions at the last known price
5. Generates a full analytics report

### Running a Backtest

```bash
# Ensure you have AI outputs and market data first
go run ./cmd/engine --mode backtest
```

The backtest uses the same config file as live trading, so risk parameters, sector limits, trailing stop-loss, and max hold days are applied consistently.

---

## Circuit Breaker

The circuit breaker (`internal/risk/circuit_breaker.go`) automatically halts trading when repeated failures are detected, preventing cascading losses from API outages or broker issues.

### How It Works

The circuit breaker monitors two failure signals:

1. **Consecutive failures** -- Tracks back-to-back order/API failures. Resets to zero on any success.
2. **Hourly failures** -- Sliding 1-hour window of all failures. Automatically prunes entries older than 1 hour.

When either threshold is breached, the circuit breaker **trips** and blocks all new trade execution. The `execute_trades` job checks `IsTripped()` at the very start and skips the entire run if tripped.

### Failure Sources

The following broker operations are wrapped with circuit breaker tracking:

| Operation | On Failure | On Success |
|-----------|-----------|------------|
| `GetFunds()` | `RecordFailure("funds_fetch_failed")` | `RecordSuccess()` |
| `GetHoldings()` | `RecordFailure("holdings_fetch_failed")` | `RecordSuccess()` |
| `PlaceOrder()` | `RecordFailure("order_place_failed")` | `RecordSuccess()` |
| Webhook REJECTED | `RecordFailure("order_rejected: ...")` | -- |

### Recovery

- **Auto-reset** -- If `cooldown_minutes > 0`, the circuit breaker automatically resets after the cooldown period. Each call to `IsTripped()` checks the cooldown.
- **Manual reset** -- Call `Reset()` programmatically, or restart the engine.
- **No auto-reset** -- If `cooldown_minutes = 0`, the circuit breaker stays tripped until manual reset or restart.

### Configuration

```json
"circuit_breaker": {
  "max_consecutive_failures": 3,
  "max_failures_per_hour": 10,
  "cooldown_minutes": 30
}
```

Set both thresholds to `0` to disable the circuit breaker entirely.

---

## Trailing Stop-Loss

The trailing stop-loss feature adjusts the stop-loss price upward as a position moves into profit, locking in gains while allowing room for normal price fluctuations.

### How It Works

1. **Activation** -- The trailing SL only activates once the position's unrealized profit reaches `activation_pct`. For example, if `activation_pct = 2.0`, the stock must be 2% above the entry price before trailing begins. Set to `0` for immediate activation.

2. **Trail calculation** -- Once active, the new SL is computed as:
   ```
   new_sl = current_high × (1 - trail_pct / 100)
   ```
   For example, with `trail_pct = 1.5` and a current high of ₹1000, the trailing SL is ₹985.

3. **Ratchet mechanism** -- The trailing SL only moves **up**, never down. If the computed new SL is below the current SL, it is ignored.

4. **Order replacement** -- Since the Dhan API does not support order modification, the engine cancels the existing SL-M order and places a new SL-M order at the higher price. Both the new SL price and the new SL order ID are persisted in the database.

### Where It Runs

- **Live/paper mode** -- Checked during `monitor_exits` on every polling cycle, after the max hold period check and before strategy exit evaluation.
- **Backtest mode** -- Applied using the candle's high price for each simulated day. The trailing SL is adjusted before SL/target hit checks, so tightened stops can trigger exits.

### Configuration

```json
"trailing_stop": {
  "enabled": true,
  "trail_pct": 1.5,
  "activation_pct": 2.0
}
```

| Field | Description |
|-------|-------------|
| `enabled` | Master switch for trailing stop-loss |
| `trail_pct` | Distance below the high watermark as a percentage (e.g., 1.5 = 1.5%) |
| `activation_pct` | Minimum profit % before trailing begins (0 = start immediately) |

---

## Postback-Driven Position Updates

When webhooks are enabled, the engine registers a postback handler that processes Dhan order notifications in real-time, replacing polling-based status checks for certain events.

### Events Handled

| Postback Status | Action |
|----------------|--------|
| **TRADED / COMPLETED** (entry order) | Auto-place SL-M order, update trade record with fill price |
| **TRADED / COMPLETED** (SL order) | Mark position as closed with `sl_hit` exit reason, remove from active tracking |
| **REJECTED** | Record failure in circuit breaker, log rejection reason |
| **CANCELLED** | Log cancellation, clean up tracking state |

### Order Matching

The postback handler identifies orders by matching the incoming order ID against:
1. `TradeRecord.OrderID` -- Entry buy orders
2. `TradeRecord.SLOrderID` -- Stop-loss sell orders

This matching uses thread-safe `GetByOrderID()` and `GetBySLOrderID()` methods on the trade context.

### Thread Safety

The trade context (`tradeContext`) is protected by a `sync.RWMutex` to allow concurrent access from:
- **Webhook HTTP handler** (goroutine per request) -- processes postbacks
- **Market-hour jobs** (scheduler goroutines) -- execute trades and monitor exits

All direct map accesses are replaced with thread-safe methods: `Get()`, `Set()`, `Delete()`, `GetByOrderID()`, `GetBySLOrderID()`, `Snapshot()`, `Len()`.

### Configuration

Enable webhooks in `config.json` and register the postback URL in your Dhan dashboard:

```json
"webhook": {
  "enabled": true,
  "port": 8080,
  "path": "/webhook/dhan/order"
}
```

Dhan will POST order status updates to `http://your-server:8080/webhook/dhan/order`.

---

## Config Hot-Reload

The config watcher monitors the configuration file for changes and applies updated risk parameters without restarting the engine.

### How It Works

1. **File polling** -- The watcher checks the config file's modification time every 5 seconds using `os.Stat()` (no external dependencies like `fsnotify`).
2. **Validation** -- Changed configs are fully parsed and validated. Invalid JSON or configs that fail validation are silently ignored (with a log warning).
3. **Risk-only changes** -- Only changes to `risk.*` fields trigger a reload. Changes to broker config, database URL, trading mode, or paths require a restart.
4. **Callback** -- On a valid risk change, the watcher notifies registered callbacks with both the old and new config, enabling the risk manager and circuit breaker to update their parameters atomically.

### What Gets Reloaded

| Field | Hot-Reloadable | Requires Restart |
|-------|:-:|:-:|
| `risk.max_risk_per_trade_pct` | Yes | -- |
| `risk.max_open_positions` | Yes | -- |
| `risk.max_daily_loss_pct` | Yes | -- |
| `risk.max_capital_deployment_pct` | Yes | -- |
| `risk.max_per_sector` | Yes | -- |
| `risk.max_hold_days` | Yes | -- |
| `risk.trailing_stop.*` | Yes | -- |
| `risk.circuit_breaker.*` | Yes | -- |
| `active_broker` | -- | Yes |
| `trading_mode` | -- | Yes |
| `database_url` | -- | Yes |
| `broker_config.*` | -- | Yes |
| `webhook.*` | -- | Yes |

### Usage

Simply edit `config.json` while the engine is running in market mode. Changes are picked up within 5 seconds. The engine logs which risk fields changed:

```
CONFIG RELOAD: risk config changed, applying updates
CONFIG RELOAD: max_open_positions: 5 → 8
CONFIG RELOAD: trailing_stop.trail_pct: 1.5 → 2.0
```

---

## Graceful Shutdown

The engine handles SIGINT (Ctrl+C) and SIGTERM signals to cleanly stop all operations and persist state before exiting.

### Shutdown Sequence

1. **Signal received** -- The signal handler sets the shutdown flag, stopping the scheduler loop.
2. **Wait for in-flight jobs** -- A `sync.WaitGroup` tracks active `execute_trades` and `monitor_exits` jobs. The engine waits up to **30 seconds** for them to complete.
3. **Webhook server shutdown** -- If the webhook server is running, it is gracefully shut down (finishes in-flight HTTP requests).
4. **Config watcher stop** -- The file watcher is stopped.
5. **Exit** -- The engine exits cleanly with all state persisted.

If in-flight jobs do not complete within 30 seconds, the engine logs a warning and exits anyway to avoid hanging indefinitely.

### What Gets Persisted

- All open trades remain in the database with their current state
- SL orders remain active at the broker (they are broker-managed, not engine-managed)
- On next startup, position reconciliation restores the full state

---

## Daily Workflow

```
Evening (after market close, 6-8 PM IST):
  1. Run: ./scripts/run_nightly.sh
     - Syncs market data from Dhan API (1 year of OHLCV for 50 stocks)
     - Runs AI scoring pipeline (regime + stock scores)
     - Generates next-day watchlist
  2. Review scores in ai_outputs/<date>/

Next morning (before or during market hours):
  3. (Optional) Preview trades: go run ./cmd/engine --mode dry-run
     - See exactly which stocks would be bought/rejected and why
     - No orders placed, no state changes

During market hours (9:15 AM - 3:30 PM IST):
  4. Run: ./scripts/run_market.sh
     - Executes pre-planned trades from all 4 strategies
     - Monitors open positions for exits (strategy + max hold period)
     - Adjusts trailing stop-loss as positions move into profit
     - Places automated stop-loss orders after fills
     - Processes postback notifications in real-time (if webhook enabled)
     - Circuit breaker halts trading on repeated failures
     - Continuously polls at configured interval

  5. (Optional) Adjust risk on the fly:
     - Edit config.json while the engine is running
     - Changes to risk parameters are picked up within 5 seconds
     - No restart needed

After market close:
  6. View analytics: go run ./cmd/engine --mode analytics
     - Win rate, Sharpe ratio, drawdown, strategy breakdown

Anytime:
  7. Check status: go run ./cmd/engine --mode status
  8. Run backtest: go run ./cmd/engine --mode backtest
  9. Dry-run preview: go run ./cmd/engine --mode dry-run
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
|   |   |-- watcher.go             Config hot-reload (stat-based file polling, risk-only changes)
|   |   |-- watcher_test.go        Config watcher tests
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
|   |   |-- circuit_breaker.go     Circuit breaker (consecutive + hourly failure tracking)
|   |   |-- circuit_breaker_test.go 10 tests (trip, reset, cooldown, config update)
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
|   |   |-- indicators.go          Shared indicators (ATR, RSI, SMA, EMA, VWAP, MACD, Bollinger, etc.)
|   |   |-- indicators_test.go     33 indicator tests
|   |   |-- trend_follow.go        Trend Following strategy
|   |   |-- trend_follow_test.go   Trend follow tests
|   |   |-- mean_reversion.go      Mean Reversion strategy
|   |   |-- mean_reversion_test.go 8 tests
|   |   |-- breakout.go            Breakout strategy
|   |   |-- breakout_test.go       8 tests
|   |   |-- momentum.go            Momentum strategy
|   |   |-- momentum_test.go       8 tests
|   |   |-- vwap.go                VWAP Reversion strategy
|   |   |-- vwap_test.go           5 tests
|   |   |-- pullback.go            EMA Pullback strategy
|   |   |-- pullback_test.go       5 tests
|   |   |-- orb.go                 Opening Range Breakout strategy
|   |   |-- orb_test.go            5 tests
|   |   |-- macd.go                MACD Crossover strategy
|   |   |-- macd_test.go           6 tests
|   |   |-- bollinger.go           Bollinger Band Squeeze strategy
|   |   |-- bollinger_test.go      5 tests
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
go test -v ./internal/risk/         # Risk manager (19 tests) + circuit breaker (10 tests)
go test -v ./internal/analytics/    # Analytics (12 tests)
go test -v ./internal/broker/       # Broker tests (paper + Dhan mocked)
go test -v ./internal/config/       # Config validation + config watcher (6 tests)
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
| `TestDryRun_PrintsReport` | Dry-run loads AI data, runs strategies + risk, prints report without placing orders |
| `TestDryRun_MissingAIOutputs` | Dry-run returns error when no AI outputs exist |
| `TestDryRun_MissingScores` | Dry-run returns error when scores file is missing |
| `TestRunPythonScoring_NotFound` | Python execution returns clear error when interpreter not found |
| `TestRunPythonScoring_ContextCancel` | Python execution respects context cancellation |

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

# Dry-run: preview trades without placing orders
go run ./cmd/engine --mode dry-run

# Custom config path
go run ./cmd/engine --config /path/to/config.json --mode market
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `config/config.json` | Path to configuration file |
| `--mode` | `status` | Run mode: `nightly`, `market`, `status`, `analytics`, `backtest`, `dry-run` |
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

## Dry-Run Mode

The dry-run mode (`--mode dry-run`) validates the full trading pipeline without placing any orders. It loads today's AI outputs, runs all 4 strategies, validates through the risk manager, and prints a detailed report.

### What It Does

1. Loads today's market regime and stock scores from `ai_outputs/{today}/`
2. Loads candle data from market data CSVs
3. Runs all 4 strategies (trend follow, mean reversion, breakout, momentum) on each scored stock
4. Validates BUY intents through the risk manager (including sector limits, capital deployment, daily loss)
5. Tracks simulated positions and capital so risk checks are realistic across stocks
6. Prints a formatted report and exits

### What It Does NOT Do

- No broker is initialized (no orders placed, not even paper orders)
- No database writes (no trade records, no logs)
- No state changes (no position tracking, no capital changes)
- No webhook server started

### Usage

```bash
# Run after nightly mode has produced AI outputs
go run ./cmd/engine --mode dry-run

# With custom config
go run ./cmd/engine --config /path/to/config.json --mode dry-run
```

### Sample Output

```
═══════════════════════════════════════════════════════════════════
                          DRY-RUN REPORT
═══════════════════════════════════════════════════════════════════
  Regime: BULL (confidence: 0.92)
  Capital: 500000.00

── APPROVED BUYS ─────────────────────────────────────────────────
  SYMBOL       STRATEGY              PRICE         SL     TARGET    QTY  ORDER VALUE     RISK AMT
  RELIANCE     trend_follow_v1     2450.00    2410.00    2530.00     20     49000.00       800.00
  TCS          momentum_v1         3800.00    3750.00    3925.00     13     49400.00       650.00

── REJECTED BUYS ─────────────────────────────────────────────────
  INFY         breakout_v1   price=1650.00 qty=30 value=49500.00
    → REJECTED: MAX_SECTOR_CONCENTRATION: already have 2 positions in sector IT

── SUMMARY ───────────────────────────────────────────────────────
  Stocks evaluated:     200
  Strategy BUY signals: 5
  Approved buys:        2
  Rejected buys:        3
  Skips:                195
  Capital deployed:     98400.00 (19.7%)
  Capital remaining:    401600.00
═══════════════════════════════════════════════════════════════════
```

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
