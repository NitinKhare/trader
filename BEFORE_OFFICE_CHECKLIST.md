# ✅ BEFORE LEAVING FOR OFFICE - VERIFICATION CHECKLIST

## 1. DATABASE VERIFICATION
```bash
# Verify migration applied
go run scripts/run_migration.go -file db/migrations/005_add_order_fill_tracking.sql
# Expected: ✓ Migration applied successfully (or already applied)
```

## 2. BUILD VERIFICATION
```bash
# Rebuild engine with new database methods
go build -o engine ./cmd/engine
# Expected: No errors, exits cleanly
```

## 3. DATA CLEANUP
```bash
# Remove all stale data from previous tests
./clear-trades --confirm
# Expected: "Clean slate ready!"
```

## 4. VERIFY CLEAN STATE
```bash
# Check no trades remain
./daily-stats
# Expected output:
# "No trades found for 2026-02-13"
# "No open positions"
```

## 5. FRESH RUN TEST (3 minutes)
```bash
# Start engine
./engine --mode market &
ENGINE_PID=$!

# Let it run
sleep 180

# Stop it
kill $ENGINE_PID
wait $ENGINE_PID 2>/dev/null

# Check results
./daily-stats
```

## 6. EXPECTED RESULTS AFTER FIX

### ✅ GOOD SIGNS (Fix is working):
- [ ] Trades show in DETAILED TRADES section
- [ ] Exit prices are DIFFERENT from entry prices
- [ ] P&L shows REAL values (not all zeros)
- [ ] OPEN POSITIONS match trades waiting for exits
- [ ] No trade appears in both CLOSED and OPEN
- [ ] Exit times show actual execution times

### ❌ BAD SIGNS (Problems remain):
- [ ] All exit prices = entry prices (0 P&L)
- [ ] Trades appear in both CLOSED and OPEN sections
- [ ] Exit times all show same timestamp
- [ ] Daily P&L = ₹0 (this is NOT right unless no exits)

## 7. READY TO DEPLOY?

Run this verification:
```bash
# 1. Check engine runs
./engine --mode market &
sleep 10

# 2. Verify trading logs appear
tail logs/engine_*.log | grep -E "BUY ORDER|exit order queued"

# 3. Stop engine
kill %1

# 4. Check database state
./daily-stats

# 5. Verify clean (trades don't appear closed prematurely)
```

## 8. OFFICE SETUP

Before leaving:
```bash
# 1. Final cleanup
./clear-trades --confirm

# 2. Verify clean
./daily-stats

# 3. Plug in power adapter
# (critical!)

# 4. Start engine with persistence
caffeinate -i ./engine --mode market &

# 5. Verify it's running
sleep 5
ps aux | grep engine | grep -v grep

# 6. Can now close terminal and leave
# Engine will keep running via caffeinate
```

## 9. MONITOR FROM OFFICE

```bash
# SSH in from office
ssh your-mac.local

# Check if engine still running
ps aux | grep engine

# View latest logs
tail -100 logs/engine_*.log

# Check daily stats
./daily-stats

# Everything looking good? You're done!
```

## ✅ GO/NO-GO DECISION

**GO** if:
- ✅ Migration applied without errors
- ✅ Engine builds without errors
- ✅ Database cleaned successfully
- ✅ Fresh run shows trades with real exit prices
- ✅ No duplicate trades
- ✅ P&L shows real values (not all zeros)

**NO-GO** if:
- ❌ Exit prices still = entry prices
- ❌ Duplicate trades appear
- ❌ All P&L = 0
- ❌ Trades appear in both CLOSED and OPEN

---

**Last check time**: _______________
**Status**: [ ] GO [ ] NO-GO
**Notes**: ____________________

