# Funds Logging Documentation

Your trading engine now automatically logs available funds from your Dhan broker account to the console on startup and during market-hour trading.

## What Gets Logged

### On Startup (All Modes)

When the engine starts, it fetches and logs your account funds:

```
[engine] 2026/02/13 01:43:45 main.go:126: [funds] available=₹500000.00 used_margin=₹0.00 total_balance=₹500000.00
```

**Fields:**
- **available** - Cash available for trading (can place new trades)
- **used_margin** - Margin being used by open positions
- **total_balance** - Total account balance

### During Trading (Market Mode)

Every time the engine runs jobs during market hours, it logs the current funds:

```
[engine] 2026/02/13 10:30:15 main.go:1480: [market-loop] [funds] available=₹450000.00 used_margin=₹50000.00 total=₹500000.00
```

This allows you to track fund changes as trades are executed.

## Why This Is Useful

✅ **Real-time monitoring** - See available capital before each trade cycle
✅ **Detect issues early** - If available funds suddenly drop, you'll know immediately
✅ **Verify live trading** - Confirm orders are actually being executed
✅ **Risk management** - Ensure you're not over-deployed
✅ **Audit trail** - All funds logged for record-keeping

## Paper Mode vs Live Mode

### Paper Mode (Simulated)
```
[engine] main.go:126: [funds] available=₹500000.00 used_margin=₹0.00 total_balance=₹500000.00
```
- Always shows configured capital
- Used margin stays at 0 (simulated orders don't use margin)
- Shows what would happen in real trading

### Live Mode (Real Broker)
```
[engine] main.go:126: [funds] available=₹450000.00 used_margin=₹50000.00 total_balance=₹500000.00
```
- Real funds from your Dhan account
- Shows actual margin used by positions
- Changes as orders are executed and positions held

## Example Log Output

### Startup (Paper Mode)
```
[engine] 2026/02/13 08:00:00 main.go:60: config loaded: broker=dhan mode=paper capital=500000.00
[engine] 2026/02/13 08:00:00 main.go:93: PAPER MODE — simulated orders only, no real money at risk
[engine] 2026/02/13 08:00:00 main.go:106: using PAPER broker
[engine] 2026/02/13 08:00:00 main.go:126: [funds] available=₹500000.00 used_margin=₹0.00 total_balance=₹500000.00
[engine] 2026/02/13 08:00:00 main.go:139: database connected — trade logging enabled
[engine] 2026/02/13 08:00:00 main.go:161: loaded 9 strategies
```

### Market Hour Trading
```
[engine] 2026/02/13 10:30:00 main.go:1475: [market-loop] running jobs at 10:30:00...
[engine] 2026/02/13 10:30:01 main.go:1480: [market-loop] [funds] available=₹480000.00 used_margin=₹20000.00 total=₹500000.00
[engine] 2026/02/13 10:30:05 main.go:1485: [market-loop] jobs completed successfully
```

## Interpreting Funds Information

### Healthy State (Paper Trading)
```
available=₹500000.00 used_margin=₹0.00 total_balance=₹500000.00
```
✅ No positions held
✅ Full capital available for trading
✅ Ready to place trades

### Active Trading (Live Mode)
```
available=₹450000.00 used_margin=₹50000.00 total=₹500000.00
```
✅ One or more positions held
✅ ₹50,000 tied up as margin
✅ ₹450,000 still available for new trades
✅ Equation: available + used_margin = total

### Warning Signs

**Insufficient Funds**
```
available=₹5,000.00 used_margin=₹495000.00 total_balance=₹500000.00
```
⚠️ Very little capital left for new trades
⚠️ Consider closing positions to free up margin

**Margin Exceeded**
```
available=₹-50,000.00 used_margin=₹550000.00 total_balance=₹500000.00
```
❌ Account is in negative (unlikely but possible)
❌ Broker may halt trading
❌ Urgent action needed

## Configuration

The funds logging is **automatic** and requires no configuration. It works with:

- **Paper broker** - Logs simulated capital
- **Live broker** - Logs real Dhan account funds
- **All modes** - status, backtest, market, live, etc.

## Technical Implementation

### Startup Logging
```go
// File: cmd/engine/main.go (line ~126)
if funds, err := activeBroker.GetFunds(ctx); err == nil {
    logger.Printf("[funds] available=₹%.2f used_margin=₹%.2f total_balance=₹%.2f",
        funds.AvailableCash, funds.UsedMargin, funds.TotalBalance)
}
```

### Market Loop Logging
```go
// File: cmd/engine/main.go (line ~1480)
if funds, err := b.GetFunds(timeoutCtx); err == nil {
    logger.Printf("[market-loop] [funds] available=₹%.2f used_margin=₹%.2f total=₹%.2f",
        funds.AvailableCash, funds.UsedMargin, funds.TotalBalance)
}
```

### Data Source
- **Broker**: Dhan's `/v2/fundlimit` API endpoint
- **Paper Broker**: In-memory simulation based on configured capital
- **Update Frequency**:
  - Once at startup
  - Every 5 minutes during market hours (configurable polling interval)

## Troubleshooting

### "Could not fetch available funds"
```
WARNING: could not fetch available funds: connection error
```

**Causes:**
- Dhan API down or unreachable
- Invalid credentials in config
- Network connectivity issue

**Solution:**
- Check your internet connection
- Verify Dhan credentials in config.json
- Check Dhan's status page

### Funds Not Changing in Live Mode
**Issue:** Running in live mode but funds look like paper trading

**Check:**
1. Verify `config.json` has `"trading_mode": "live"`
2. Confirm you're connecting to Dhan (not paper broker)
3. Check that orders are actually being placed (check order logs)

### Missing Funds Log at Startup
**Issue:** You don't see the funds log line

**Cause:** Error fetching funds (ignored silently)

**Solution:**
- Check if broker is initialized correctly
- Run in status mode to see all details
- Check for Dhan connection issues

## Example: Monitoring Live Trading

Here's how to use the funds log to monitor your live trading session:

1. **Start engine**
   ```bash
   go run ./cmd/engine --mode market
   ```

2. **Watch funds log**
   ```
   [engine] 10:30:00 [market-loop] [funds] available=₹500000.00 used_margin=₹0.00 total=₹500000.00
   [engine] 10:30:05 Executed BUY order: RELIANCE 10 @ ₹1000
   [engine] 10:35:00 [market-loop] [funds] available=₹490000.00 used_margin=₹10000.00 total=₹500000.00
   [engine] 10:45:00 Executed SELL order: RELIANCE 10 @ ₹1050 (P&L: ₹500)
   [engine] 10:45:05 [market-loop] [funds] available=₹500500.00 used_margin=₹0.00 total=₹500500.00
   ```

3. **Verify**
   - Funds decrease when you buy (used as margin)
   - Funds increase when you sell with profit
   - Funds decrease when you sell at loss
   - Total should match your account balance

## Summary

The engine now provides complete visibility into your available funds:

- ✅ Automatic on startup
- ✅ Logged every market cycle (5-minute intervals by default)
- ✅ Works for both paper and live trading
- ✅ Shows real Dhan account data in live mode
- ✅ Helps monitor capital allocation and risk

This keeps you informed of your account status at all times! 🚀

