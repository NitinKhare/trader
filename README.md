# 🚀 Algorithmic Trading Agent - Complete System

A fully-functional algorithmic trading system with AI stock scoring, multi-strategy evaluation, real-time position management, and complete interactive dashboard.

---

## 📋 Table of Contents

1. [Quick Start](#quick-start)
2. [System Overview](#system-overview)
3. [Architecture](#architecture)
4. [Dashboard Setup](#dashboard-setup)
5. [REST API Reference](#rest-api-reference)
6. [WebSocket Reference](#websocket-reference)
7. [Trading Engine](#trading-engine)
8. [Daily Workflow](#daily-workflow)
9. [Risk Management](#risk-management)
10. [Troubleshooting](#troubleshooting)
11. [Commands Reference](#commands-reference)
12. [FAQs](#faqs)

---

## 🎯 Quick Start

### The Essentials

```bash
# 1. Generate AI scores (Morning, before 9:15 AM)
python3 -m python_ai.run_scoring --date today --output-dir ./ai_outputs --data-dir ./market_data

# 2. Start trading engine
go run ./cmd/engine --mode market

# 3. Start dashboard backend (in another terminal)
./dashboard --port 8081

# 4. Start dashboard frontend (in another terminal)
cd web && npm run dev

# 5. Monitor
tail -f logs/engine_*.log
# And open http://localhost:3000/dashboard in your browser
```

### End of Day - Check Results

```bash
./daily-stats
```

---

## 📊 System Overview

### What This System Does

- **AI Stock Scoring**: Loads/generates daily scores for 55+ stocks
- **Multi-Strategy Trading**: Evaluates 9 different strategies on each stock
- **Position Management**: Automatically sizes and manages positions
- **Risk Control**: Enforces position limits, capital deployment limits, daily loss limits
- **Real-Time Dashboard**: Monitor all metrics, trades, and positions in real-time
- **Complete Logging**: Audit trail of every decision
- **Daily Reports**: `./daily-stats` shows trades, P&L, capital usage

### Three Complete Subsystems

**1. Trading Engine (Go)**
- Loads AI stock scores
- Evaluates 9 strategies
- Places orders via Dhan API
- Logs all activity
- Tracks P&L in database

**2. Dashboard Backend (Go)**
- REST API (5 endpoints)
- WebSocket streaming (5-second intervals)
- PostgreSQL event listener
- Multi-client broadcaster

**3. Dashboard Frontend (React)**
- Real-time metrics display
- Equity curve visualization
- Open positions tracking
- Connection status monitoring
- Responsive design (mobile, tablet, desktop)

### Key Metrics

| Metric | Value | Notes |
|--------|-------|-------|
| **Trading Mode** | Paper or Live | Configurable in `config/config.json` |
| **Active Strategies** | 9 | Trend follow, momentum, mean reversion, etc. |
| **Stock Universe** | 64 stocks | NSE large-cap and mid-cap |
| **Max Risk Per Trade** | 1.0% | Configurable |
| **Max Open Positions** | 5 | Configurable |
| **Max Capital Deployment** | 70% | Live mode |
| **Max Daily Loss** | 3.0% | Configurable |
| **Dashboard Update Interval** | 5 seconds | Real-time via WebSocket |
| **Market Hours** | 9:15 AM - 3:30 PM IST | NSE trading hours |

---

## 🏗️ Architecture

### Complete System Diagram

```
┌─────────────────────────────────────┐
│  Browser (http://localhost:3000)    │
│                                     │
│  ┌──────────────────────────────┐  │
│  │  React Dashboard             │  │
│  │  • MetricsCard               │  │
│  │  • EquityCurveChart          │  │
│  │  • PositionsTable            │  │
│  │  • StatusIndicator           │  │
│  └──────────────────────────────┘  │
└──────────────┬──────────────────────┘
               │
        ┌──────┴──────┐
        ↓             ↓
    REST API    WebSocket
   (Initial)   (Real-time)
        │             │
        └──────┬──────┘
               ↓
┌──────────────────────────────────────┐
│  Go Backend (http://localhost:8081)  │
│  • HTTP Server (REST API)            │
│  • WebSocket Handler                 │
│  • Broadcaster (multi-client)        │
│  • PostgreSQL Event Listener         │
└──────────────┬──────────────────────┘
               │
        ┌──────┴──────┐
        ↓             ↓
  PostgreSQL    Dhan API
   Database    (Trading)
        │             │
        └──────┬──────┘
               ↓
        ┌─────────────┐
        │ Trading     │
        │ Engine (Go) │
        └─────────────┘
```

### Data Flow

**Initial Dashboard Load:**
1. User opens http://localhost:3000/dashboard
2. React fetches 4 REST endpoints in parallel from Go backend
3. Display metrics with loading skeleton
4. Connect to WebSocket for real-time updates

**Real-Time Updates (every 5 seconds):**
1. Go backend fetches fresh metrics from database
2. Broadcasts to all connected WebSocket clients
3. React receives update, updates state
4. Components re-render with smooth animations

**Event-Driven Triggers:**
1. Trading engine executes trade
2. Database updated
3. PostgreSQL fires NOTIFY event
4. Event listener catches notification
5. Fresh metrics fetched and broadcast immediately

---

## 📈 Dashboard Setup

### Prerequisites

```bash
# Node.js 18+
node --version

# Go 1.20+
go version

# PostgreSQL 14+
psql --version
```

### Build Dashboard (Backend)

```bash
# Already compiled as: ./dashboard (15 MB)
# To rebuild from source:
cd /Users/nitinkhare/Downloads/algoTradingAgent
go build -o dashboard ./cmd/dashboard
```

### Setup Frontend

```bash
cd /Users/nitinkhare/Downloads/algoTradingAgent/web

# Install dependencies
npm install

# Configure environment (create .env.local)
cat > .env.local << EOF
NEXT_PUBLIC_API_URL=http://localhost:8081
NEXT_PUBLIC_WS_URL=ws://localhost:8081
EOF

# Verify build compiles
npm run build
```

### Run Complete System

**Terminal 1 - Start Backend:**
```bash
cd /Users/nitinkhare/Downloads/algoTradingAgent
./dashboard --port 8081

# Expected output:
# [dashboard] broadcaster: started
# [dashboard] event listener: started
# [dashboard] periodic broadcast: started
# [dashboard] dashboard API starting on port 8081
```

**Terminal 2 - Start Frontend:**
```bash
cd /Users/nitinkhare/Downloads/algoTradingAgent/web
npm run dev

# Expected output:
# ▲ Next.js 14.1.4
# - Local:         http://localhost:3000
# ✓ Ready in 747ms
```

**Terminal 3 - View Dashboard:**
```bash
# Open in browser:
# http://localhost:3000/dashboard
# OR
# http://localhost:3000/docs (for documentation)
# OR
# http://localhost:3000 (for home page)
```

### Dashboard Features

**1. Metrics Card**
- Total P&L (absolute and percentage)
- Win rate (% of winning trades)
- Profit factor (gross profit / gross loss)
- Sharpe ratio (risk-adjusted returns)
- Max drawdown (peak to trough decline)
- Available capital (remaining deployable capital)
- All updated every 5 seconds

**2. Equity Curve Chart**
- Time-series visualization of account growth
- Drawdown overlay showing peak-to-current decline
- Interactive tooltips with exact values
- Smooth animations on updates

**3. Open Positions Table**
- Stock symbol and quantity
- Entry price and entry time
- Stop-loss and target profit levels
- Unrealized P&L (absolute and percentage)
- Color-coded: Green (profit), Red (loss)
- Updates in real-time

**4. Status Indicator**
- Trading mode badge (PAPER or LIVE)
- API connection status (green/red dot)
- WebSocket connection status
- Last update timestamp
- Auto-reconnect attempts on disconnect

**5. Documentation (Interactive)**
- 13 comprehensive sections
- Quick Start, System Overview, Architecture
- Complete API Reference
- Troubleshooting guides
- Performance metrics
- Deployment instructions
- FAQs and best practices
- Accessible at /docs route

---

## 🔌 REST API Reference

### Base URL
```
http://localhost:8081
```

### Endpoints

#### 1. GET /api/metrics
Returns current trading metrics

**Response:**
```json
{
  "total_pnl": 5420.50,
  "total_pnl_percent": 1.084,
  "win_rate": 0.60,
  "profit_factor": 1.83,
  "drawdown": 12500.25,
  "drawdown_percent": 2.5,
  "sharpe_ratio": 1.42,
  "total_trades": 25,
  "winning_trades": 15,
  "losing_trades": 10,
  "avg_pnl": 216.82,
  "gross_profit": 15320.00,
  "gross_loss": 8380.00,
  "avg_hold_days": 2.5,
  "initial_capital": 500000.00,
  "final_capital": 505420.50,
  "timestamp": "2026-02-14T15:45:30Z"
}
```

#### 2. GET /api/positions/open
Returns currently open positions

**Response:**
```json
[
  {
    "id": 1,
    "symbol": "TITAN",
    "quantity": 20,
    "entry_price": 4249.10,
    "entry_time": "2026-02-14T10:30:15Z",
    "stop_loss": 4009.00,
    "target": 4550.00,
    "strategy_id": "trend_follow_1",
    "unrealized_pnl": 820.00,
    "unrealized_pnl_percent": 0.96
  }
]
```

#### 3. GET /api/charts/equity
Returns equity curve data for charting

**Response:**
```json
[
  {
    "date": "2026-02-01",
    "equity": 500000.00,
    "drawdown": 0,
    "drawdown_percent": 0
  },
  {
    "date": "2026-02-02",
    "equity": 502150.50,
    "drawdown": 0,
    "drawdown_percent": 0
  },
  {
    "date": "2026-02-03",
    "equity": 498500.00,
    "drawdown": 3650.50,
    "drawdown_percent": 0.73
  }
]
```

#### 4. GET /api/status
Returns system status

**Response:**
```json
{
  "trading_mode": "paper",
  "api_status": "connected",
  "websocket_status": "connected",
  "database_status": "connected",
  "last_update": "2026-02-14T15:45:30Z",
  "uptime_seconds": 3600
}
```

#### 5. GET /health
Health check endpoint

**Response:**
```json
{
  "status": "healthy"
}
```

---

## ⚡ WebSocket Reference

### Connection

```javascript
const ws = new WebSocket('ws://localhost:8081/ws');
```

### Message Format

**Server sends metrics every 5 seconds:**
```json
{
  "type": "metrics",
  "data": {
    "total_pnl": 5420.50,
    "total_pnl_percent": 1.084,
    "win_rate": 0.60,
    "profit_factor": 1.83,
    "sharpe_ratio": 1.42,
    "timestamp": "2026-02-14T15:45:30Z"
  },
  "timestamp": "2026-02-14T15:45:30Z"
}
```

### Features

- **Auto-reconnect**: Exponential backoff (5 max attempts)
- **Heartbeat**: 30-second ping/pong to detect stale connections
- **Multi-client**: Broadcasts to all connected clients simultaneously
- **Non-blocking**: Skips slow clients to avoid blocking updates
- **Error handling**: Graceful disconnect with automatic reconnection

### Testing

```bash
# Install wscat globally
npm install -g wscat

# Connect to WebSocket
wscat -c ws://localhost:8081/ws

# You'll see metrics updates every 5 seconds
```

---

## 💹 Trading Engine

### Daily Workflow

**Morning (8:45 AM - 9:10 AM)**

```bash
# 1. Generate AI scores
python3 -m python_ai.run_scoring --date today --output-dir ./ai_outputs --data-dir ./market_data

# 2. Start trading engine
go run ./cmd/engine --mode market
```

**During Trading (9:15 AM - 3:30 PM)**

```bash
# Monitor in separate terminal
tail -f logs/engine_*.log
```

**End of Day (After 3:30 PM)**

```bash
# Get summary statistics
./daily-stats
```

### Strategies (9 Total)

| # | Strategy | Entry Signal | Exit Signal |
|---|----------|--------------|-------------|
| 1 | Trend Follow | Strong uptrend (0.6+) | Trend weakens |
| 2 | Mean Reversion | High RSI oversold | Price crosses SMA(20) |
| 3 | Breakout | Price > 20-day high | Breakout fails |
| 4 | Momentum | Top 5 ranked stocks | Rank falls outside top 10 |
| 5 | VWAP Reversion | Price > VWAP by 2% | Reverts to VWAP |
| 6 | Pullback | Consolidation after uptrend | Tight ATR range breaks |
| 7 | ORB | Breakout from opening range | Range becomes too wide |
| 8 | MACD Crossover | MACD crosses above signal | Bearish crossover |
| 9 | Bollinger Squeeze | Breakout above upper band | Squeeze compression |

### Deployment Modes

**Paper Mode (Safe Testing)**
```bash
# config/config.json: "trading_mode": "paper"
go run ./cmd/engine --mode market
```
- Use for testing strategies without real money risk
- Verifying system functionality
- Learning how it works

**Live Mode (Real Trading)**
```bash
# config/config.json: "trading_mode": "live"
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live
```
- Real capital trading
- Requires valid Dhan API credentials
- Dual safety confirmation gates

**Backtest Mode (Historical Analysis)**
```bash
go run ./cmd/engine --mode backtest --start-date 2026-02-01 --end-date 2026-02-13
```
- Analyze historical strategy performance
- No market interaction

---

## 🛡️ Risk Management

### Position Limits (in config/config.json)

```json
{
  "max_risk_per_trade_pct": 1.0,        // Risk max 1% per trade
  "max_open_positions": 5,               // Never hold > 5 positions
  "max_capital_deployment_pct": 70.0,   // Never deploy > 70% capital
  "max_daily_loss_pct": 3.0             // Stop trading if lose 3% today
}
```

### How It Works

1. **Trade Sizing**: quantity = risk_budget / stop_loss_distance
2. **Position Limits**: Reject if would exceed max positions or capital
3. **Daily Limits**: Track daily P&L, stop trading if hit 3% loss
4. **Stop-Loss**: Automatic stop-loss order on every position
5. **Take-Profit**: Profit target on every position

### Example Calculation

```
Capital: ₹500,000
Max risk per trade: 1% = ₹5,000

Trade: TITAN @ ₹4249
Stop loss: ₹4009 (3.2% below entry)
Risk per share: ₹240

Quantity = ₹5,000 / ₹240 = ~20 shares
Capital used: 20 × ₹4249 = ₹84,980
```

---

## 🔧 Configuration

### Edit: config/config.json

```json
{
  "active_broker": "dhan",
  "trading_mode": "paper",              // "paper" or "live"
  "capital": 500000.00,                 // Your trading capital

  "risk": {
    "max_risk_per_trade_pct": 1.0,
    "max_open_positions": 5,
    "max_daily_loss_pct": 3.0,
    "max_capital_deployment_pct": 70.0
  },

  "broker_config": {
    "dhan": {
      "client_id": "your_id",
      "access_token": "your_token"
    }
  },

  "database_url": "postgres://algo:algo123@localhost:5432/algo_trading",
  "polling_interval_minutes": 5
}
```

---

## 📊 Daily Stats Command

### The Command

```bash
./daily-stats
```

### What It Shows

```
╔════════════════════════════════════════════════════════════╗
║           DAILY TRADING STATISTICS                         ║
║           Date: 2026-02-14                                 ║
╚════════════════════════════════════════════════════════════╝

SUMMARY
  Total Trades:      25
  Winning Trades:    15
  Losing Trades:     10
  Win Rate:          60.0%

  Daily P&L:         ₹5,420.00
  Capital Used:      ₹180,500.00

DETAILED TRADES
Symbol    Quantity  Entry Price  Exit Price   P&L       Exit Time
──────────────────────────────────────────────────────────────
TITAN     20        4249.10      4249.10      820.00    06:18:37
SBIN      57        1182.90      1182.90      1840.00   06:18:37
M&M       9         3674.90      3674.90      620.00    06:18:37

OPEN POSITIONS
(Shows positions still held with stop loss and targets)
```

---

## 🐛 Troubleshooting

### Dashboard Won't Load

**Check backend is running:**
```bash
curl http://localhost:8081/health
# Should return: {"status": "healthy"}
```

**Check frontend is running:**
```bash
curl http://localhost:3000
# Should return HTML of home page
```

**Check WebSocket connection:**
```bash
npm install -g wscat
wscat -c ws://localhost:8081/ws
# Should receive metrics every 5 seconds
```

### Engine Won't Start

```bash
# Check PostgreSQL is running
psql -U algo -d algo_trading -h localhost -c "SELECT 1;"

# Check config file syntax
cat config/config.json

# Check API credentials are valid
# (Try simple API call to Dhan with your token)
```

### No Trades Being Placed

**Likely causes:**
- Capital too low for stock prices
- All positions at max
- Daily loss limit hit
- No valid signals from strategies

**Check:**
```bash
tail -f logs/engine_*.log
grep -i "skip\|reject" logs/engine_*.log | head -20
```

### WebSocket Disconnect

- Check backend is still running: `curl http://localhost:8081/health`
- Check no firewall blocking port 8081
- Browser will auto-reconnect with exponential backoff
- Status indicator shows connection state

### Database Connection Error

```bash
# Make sure PostgreSQL is running
brew services start postgresql  # macOS
systemctl start postgresql      # Linux

# Verify credentials
psql -U algo -d algo_trading -h localhost
```

---

## 📚 Commands Reference

### Dashboard Commands

```bash
# Start Backend
./dashboard --port 8081

# Start Frontend (in web/ directory)
npm run dev

# Build Frontend
npm run build

# Run Production
npm start
```

### Trading Engine Commands

```bash
# Generate AI Scores
python3 -m python_ai.run_scoring --date today --output-dir ./ai_outputs --data-dir ./market_data

# Run Engine (Paper)
go run ./cmd/engine --mode market

# Run Engine (Live)
ALGO_LIVE_CONFIRMED=true go run ./cmd/engine --mode market --confirm-live

# Status Check
go run ./cmd/engine --mode status

# Backtest
go run ./cmd/engine --mode backtest --start-date 2026-02-01 --end-date 2026-02-13
```

### Monitoring

```bash
# Watch logs in real-time
tail -f logs/engine_*.log

# See only trades
grep "BUY ORDER PLACED\|EXIT ORDER PLACED" logs/engine_*.log

# Count today's trades
grep -c "BUY ORDER PLACED" logs/engine_$(date +%Y-%m-%d)*.log

# Get daily stats
./daily-stats

# Specific date
./daily-stats -date 2026-02-13
```

### Database Queries

```bash
# Query trades for specific date
psql -U algo -d algo_trading -h localhost << 'EOF'
SELECT symbol, COUNT(*) as trades,
       ROUND(SUM((exit_price - entry_price) * quantity), 2) as pnl
FROM trades
WHERE DATE(exit_time AT TIME ZONE 'IST') = '2026-02-14'
  AND status = 'closed'
GROUP BY symbol
ORDER BY pnl DESC;
EOF
```

### Build Commands

```bash
# Rebuild dashboard
go build -o dashboard ./cmd/dashboard

# Rebuild daily stats
go build -o daily-stats ./cmd/daily-stats

# Rebuild trading engine
go build -o engine ./cmd/engine
```

---

## ❓ FAQs

### Dashboard & Monitoring

**Q: How do I access the dashboard?**
A: Open http://localhost:3000/dashboard in your browser (after starting backend and frontend)

**Q: How often do metrics update?**
A: Every 5 seconds via WebSocket real-time updates

**Q: What if the WebSocket disconnects?**
A: React automatically reconnects with exponential backoff (up to 5 attempts). Status indicator shows connection state.

**Q: Can I view historical data?**
A: Yes, equity curve chart shows all historical data. Select date ranges by clicking on chart.

**Q: Is the dashboard mobile-friendly?**
A: Yes, fully responsive design works on mobile, tablet, and desktop.

### Trading System

**Q: How do I know how many trades were taken?**
A: Run `./daily-stats` - shows total trades, wins/losses, and P&L

**Q: Can I trade with real money?**
A: Yes, switch `trading_mode` to "live" in config.json and use real Dhan credentials

**Q: What happens if I lose more than 3% in a day?**
A: Engine automatically stops trading that day (configurable safety feature)

**Q: Can I run this 24/5 on the cloud?**
A: Yes, deploy to AWS/GCP/Azure with continuous runtime

**Q: What if the engine crashes?**
A: Use systemd on Linux or launchd on macOS - auto-restart enabled

**Q: How do I backtest?**
A: `go run ./cmd/engine --mode backtest --start-date 2026-02-01 --end-date 2026-02-13`

**Q: Which stocks can I trade?**
A: All 64 NSE large-cap and mid-cap stocks in the configured universe

**Q: Can I modify strategies?**
A: Yes, each strategy is in `internal/strategy/` with clear entry/exit rules

---

## 📂 Project Structure

```
algoTradingAgent/
├── cmd/
│   ├── dashboard/           # Dashboard backend (Go)
│   │   ├── main.go
│   │   ├── response.go
│   │   └── websocket.go
│   ├── engine/              # Trading engine (Go)
│   └── daily-stats/         # Statistics CLI
│
├── internal/
│   ├── dashboard/           # Dashboard internal packages
│   │   ├── broadcaster.go
│   │   └── events.go
│   ├── broker/              # Broker APIs (Dhan, paper)
│   ├── strategy/            # All 9 trading strategies
│   ├── risk/                # Risk management
│   ├── storage/             # Database operations
│   └── ...
│
├── web/                     # React Frontend
│   ├── app/
│   │   ├── layout.tsx
│   │   ├── page.tsx
│   │   ├── dashboard/
│   │   │   └── page.tsx
│   │   └── docs/
│   │       └── page.tsx
│   ├── hooks/
│   │   ├── useAPI.ts
│   │   ├── useWebSocket.ts
│   │   └── useMetrics.ts
│   ├── types/
│   │   └── api.ts
│   ├── utils/
│   │   ├── formatting.ts
│   │   └── constants.ts
│   ├── styles/
│   │   └── globals.css
│   ├── package.json
│   ├── tsconfig.json
│   ├── next.config.ts
│   └── .env.local
│
├── python_ai/               # AI stock scoring system
├── config/
│   ├── config.json          # Main configuration
│   └── instruments.json     # Dhan instruments
│
├── db/
│   └── migrations/          # Database schema
│
├── market_data/             # CSV files with OHLCV data
├── ai_outputs/              # Daily AI scores
├── logs/                    # Engine logs
├── scripts/                 # Utility scripts
│
├── go.mod                   # Go dependencies
├── go.sum                   # Go checksums
├── dashboard                # Compiled binary (15 MB)
├── daily-stats              # Compiled binary
└── README.md                # This file
```

---

## ✅ System Status

| Component | Status | Notes |
|-----------|--------|-------|
| **Database** | ✅ | PostgreSQL + TimescaleDB |
| **Trading Engine** | ✅ | Real-time, all strategies |
| **Dashboard Backend** | ✅ | REST API + WebSocket |
| **Dashboard Frontend** | ✅ | React + TypeScript |
| **Logging** | ✅ | File + console |
| **Daily Stats** | ✅ | Complete reporting |
| **Risk Management** | ✅ | All limits enforced |
| **Paper Trading** | ✅ | ₹500K simulated |
| **Live Trading** | ✅ | Ready for real capital |
| **Documentation** | ✅ | Complete + interactive |

---

## 🚀 Next Steps

### Short Term
1. **Test Dashboard**: Verify all metrics display correctly
2. **Paper Trade 3-5 Days**: Test strategies without risk
3. **Review Daily Stats**: Monitor performance

### Medium Term
1. **Switch to Live Mode** when confident
2. **Start with Small Capital** and increase gradually
3. **Monitor Daily Results** and adjust parameters

### Long Term
1. **Deploy to Cloud** for 24/5 trading
2. **Optimize Strategies** based on live data
3. **Scale Capital** as system proves profitable

---

## 📞 Support

### View Logs
```bash
tail -f logs/engine_*.log
```

### Check Daily Performance
```bash
./daily-stats
```

### Access Interactive Documentation
```bash
# Open in browser:
http://localhost:3000/docs
```

### Database Status
```bash
psql -U algo -d algo_trading -h localhost -c "SELECT COUNT(*) FROM trades;"
```

---

## 🎯 Key Points

- ✅ **Complete System**: Everything from AI to trading to monitoring
- ✅ **Real-Time Dashboard**: See all metrics updated every 5 seconds
- ✅ **Safe**: Dual confirmation for live trading, position limits, daily loss limits
- ✅ **Automated**: Strategies run 24/5 once deployed
- ✅ **Documented**: Comprehensive guides and interactive documentation
- ✅ **Production Ready**: Tested, debugged, and fully functional

---

## 📝 License

Proprietary - For personal use only

---

**Last Updated**: 2026-02-14
**System Status**: ✅ Production Ready

For questions or issues:
- Check logs: `tail -f logs/engine_*.log`
- View dashboard: `http://localhost:3000`
- Check docs: `http://localhost:3000/docs`
- Get daily stats: `./daily-stats`

Happy Trading! 🚀
