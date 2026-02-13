# CRITICAL BUG REPORT: Trades Showing Zero P&L (Entry Price = Exit Price)

## Issue Description
Trades are being recorded with exit price = entry price, resulting in 0 P&L for all closed trades.

Example from daily-stats output:
```
Symbol       Quantity   Entry Price  Exit Price   P&L          Exit Time
APOLLOHOSP   10         7507.00      7507.00      0.00         06:28:58
SBIN         57         1182.90      1182.90      0.00         06:28:58
TITAN        20         4249.10      4249.10      0.00         06:28:58
```

## Root Cause Analysis

### Where the Bug Is
There are TWO locations in `cmd/engine/main.go` where trades are closed incorrectly:

#### Location 1: execute_trades job (Line ~772)
```go
case strategy.ActionExit:
    // ... place exit order ...
    resp, err := b.PlaceOrder(ctx, order)

    // IMMEDIATE PROBLEM: Closes trade right after placing exit order
    closeTradeRecord(ctx, store, logger, tc, intent.Symbol, intent.Price, intent.Reason)
```

#### Location 2: monitor_exits job (Line ~905)
```go
// For max hold period exits
resp, err := b.PlaceOrder(ctx, exitOrder)
if err == nil {
    // IMMEDIATE PROBLEM: Closes trade right after placing exit order
    closeTradeRecord(ctx, store, logger, tc, h.Symbol, exitPrice, "max_hold_period")
}
```

### Why This Is Wrong

1. **Order Placement != Order Execution**
   - Placing an exit order (especially limit orders) doesn't mean it was filled
   - In paper mode: Orders might fill at entry price or market price, not the limit order price
   - In live mode: Limit orders may not fill for hours or may not fill at all

2. **Using Wrong Price**
   - `intent.Price` = the TARGET/LIMIT price set at strategy decision time
   - Actual exit price = where the order actually fills (market price at that moment)
   - These are often different!

3. **Immediate Closure Problem**
   - Code closes trade immediately after placing order
   - Should wait for order to actually fill
   - Then query the actual fill price

### The Real Issue in Paper Mode

In paper mode, when an exit order is placed:
- Paper broker likely simulates a fill immediately
- BUT it may fill at average holding price (7507.00) instead of limit price
- Or it fills instantly at current market price = entry price (no slippage in simulator)
- Trade gets closed with entry price as exit price = 0 P&L

## Solution

### Required Changes

The trade closing logic must be refactored to:

1. **Do NOT close trade immediately when exit order is placed**
   - Just log that exit order was placed
   - Store the exit order ID in the trade record

2. **Add Order Status Polling**
   - Check order status in the next monitoring cycle
   - Verify if order was FILLED, PENDING, or REJECTED

3. **Get Actual Fill Price from Order Details**
   - Call broker.GetOrderStatus() or similar to get filled price
   - Use actual fill price, NOT the limit order price

4. **THEN Close Trade with Real Price**
   - Only mark trade as closed when order shows FILLED status
   - Use the actual filled price from broker response

### Code Changes Needed

#### Step 1: Update Trade Model
Add `exit_order_id` field to track which order is closing the trade.

#### Step 2: Modify execute_trades Job
**BEFORE (WRONG):**
```go
case strategy.ActionExit:
    // ... create order ...
    resp, err := b.PlaceOrder(ctx, order)
    if err == nil {
        closeTradeRecord(ctx, store, logger, tc, intent.Symbol, intent.Price, intent.Reason)
    }
```

**AFTER (CORRECT):**
```go
case strategy.ActionExit:
    // ... create order ...
    resp, err := b.PlaceOrder(ctx, order)
    if err == nil {
        // Store exit order ID, but don't close trade yet
        if tc != nil {
            if trade, ok := tc.Get(intent.Symbol); ok {
                trade.ExitOrderID = resp.OrderID
                trade.ExitReason = intent.Reason
                // DO NOT call closeTradeRecord here!
                logger.Printf("  %s: EXIT ORDER PLACED (will monitor for fill) order_id=%s",
                    intent.Symbol, resp.OrderID)
            }
        }
    }
```

#### Step 3: Modify monitor_exits Job
Add new logic to check if exit orders have been filled:

```go
// NEW: Check if any exit orders have been filled
for _, h := range holdings {
    if tc != nil {
        if trade, ok := tc.Get(h.Symbol); ok && trade.ExitOrderID != "" {
            // Check if exit order was filled
            orderStatus, err := b.GetOrderStatus(ctx, trade.ExitOrderID)
            if err == nil && orderStatus.Status == "COMPLETED" {
                // Order filled! Get actual fill price
                actualFillPrice := orderStatus.FilledPrice
                if actualFillPrice == 0 {
                    actualFillPrice = orderStatus.Price
                }

                // NOW close the trade with REAL fill price
                closeTradeRecord(ctx, store, logger, tc, h.Symbol, actualFillPrice, trade.ExitReason)
            }
        }
    }
}
```

#### Step 4: Remove Premature Trade Closure
Remove or comment out the line in monitor_exits that closes trades immediately:
```go
// DELETE THIS:
// closeTradeRecord(ctx, store, logger, tc, h.Symbol, exitPrice, "max_hold_period")

// REPLACE WITH:
if tc != nil {
    if trade, ok := tc.Get(h.Symbol); ok {
        trade.ExitOrderID = resp.OrderID
        trade.ExitReason = "max_hold_period"
        // Will be closed when order fills
    }
}
```

## Verification Steps

After making changes:

1. **Run Paper Mode Trading**
   ```bash
   go run ./cmd/engine --mode market
   ```

2. **Monitor Logs**
   ```bash
   tail -f logs/engine_*.log | grep -E "EXIT|FILLED|order"
   ```

3. **Check Results**
   ```bash
   ./daily-stats
   ```

4. **Verify Exit Prices Are Different**
   - Should see exit prices that differ from entry prices
   - P&L should show actual profits/losses, not 0
   - Win/loss distribution should match trade outcomes

## Impact

- **Severity**: CRITICAL - All P&L calculations are wrong
- **Scope**: All exit trades (strategies and manual exits)
- **Affected Modes**: Both paper and live
- **Data Impact**: Need to clean up existing trades with zero P&L before re-running

## Next Steps

1. Fix the code as per solution above
2. Rebuild engine: `go build -o engine ./cmd/engine`
3. Clean database: `./clear-trades --confirm`
4. Regenerate AI scores for today: `python3 -m python_ai.run_scoring ...`
5. Run fresh trading session: `go run ./cmd/engine --mode market`
6. Verify P&L shows actual profits/losses
