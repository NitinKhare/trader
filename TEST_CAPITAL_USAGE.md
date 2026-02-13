# 🧪 Test Capital Usage Tracking

## Quick Test (5 minutes)

### Step 1: Build the engine
```bash
cd /Users/nitinkhare/Downloads/algoTradingAgent
go build -o engine ./cmd/engine
```

Expected output:
```
(no errors)
```

### Step 2: Run a backtest
```bash
./engine --mode backtest 2>&1 | head -50
```

Expected output:
```
[backtest] starting backtest run...
[backtest] found XX trading days with AI data: YYYY-MM-DD to YYYY-MM-DD
[backtest] starting capital: ₹1,000,000.00

... trading happens ...

[backtest] completed: XX closed trades, X still open at end
[backtest] capital usage summary:
  Starting Capital:    ₹1,000,000.00
  Maximum Used:        ₹XXX,XXX.XX (XX.X% of starting)
  Final Capital:       ₹1,XXX,XXX.XX
  Total P&L:           ₹XX,XXX.XX (X.XX%)
```

### Step 3: Verify the four metrics appear
```bash
./engine --mode backtest 2>&1 | grep -A4 "capital usage summary"
```

Expected output:
```
[backtest] capital usage summary:
  Starting Capital:    ₹1,000,000.00
  Maximum Used:        ₹850,000.00 (85.0% of starting)
  Final Capital:       ₹1,050,000.00
  Total P&L:           ₹50,000.00 (5.00%)
```

✅ **If you see all four metrics, the feature is working!**

---

## Detailed Test (15 minutes)

### Test Case 1: Capital Tracking Works

```bash
./engine --mode backtest > backtest_output.txt 2>&1
```

Check the file:
```bash
cat backtest_output.txt | grep -E "(starting capital|Maximum Used|Final Capital|Total P&L)"
```

Expected:
```
[backtest] starting capital: ₹1,000,000.00
[backtest] capital usage summary:
  Starting Capital:    ₹1,000,000.00
  Maximum Used:        ₹850,000.00 (85.0% of starting)
  Final Capital:       ₹1,050,000.00
  Total P&L:           ₹50,000.00 (5.00%)
```

✅ Check: All four metrics present

### Test Case 2: Starting Capital Matches Config

Check your config:
```bash
grep '"capital"' config.json
```

Example output:
```json
"capital": 1000000,
```

Now check backtest output:
```bash
./engine --mode backtest 2>&1 | grep "starting capital"
```

Example output:
```
[backtest] starting capital: ₹1,000,000.00
```

✅ Check: Values match (1000000 in config = ₹1,000,000.00 in logs)

### Test Case 3: Maximum Used is Reasonable

Run backtest:
```bash
./engine --mode backtest 2>&1 | grep "Maximum Used"
```

Example output:
```
Maximum Used:        ₹850,000.00 (85.0% of starting)
```

✅ Check: Percentage is between 30% and 100%
   - Under 30% = Not deploying enough capital
   - Over 100% = Impossible (check for bug)

### Test Case 4: Final Capital Makes Sense

Get the values:
```bash
./engine --mode backtest 2>&1 | grep -E "(Starting Capital|Final Capital|Total P&L)"
```

Verify math:
```
Starting:  ₹1,000,000
P&L:       ₹50,000
Expected Final: ₹1,050,000

Check if it matches logged Final Capital: ₹1,050,000 ✅
```

✅ Check: Final Capital = Starting Capital + Total P&L

### Test Case 5: Percentage Calculations

Get the output:
```bash
./engine --mode backtest 2>&1 | tail -20
```

Manual calculation:
```
Maximum Used % = 850,000 / 1,000,000 × 100 = 85%
Total P&L % = 50,000 / 1,000,000 × 100 = 5%

Compare with logged values ✅
```

✅ Check: Percentages are mathematically correct

---

## Integration Test (30 minutes)

### Full Workflow Test

```bash
# 1. Clean database
./clear-trades --confirm
echo "✅ Database cleaned"

# 2. Run backtest with capital tracking
./engine --mode backtest > results.txt 2>&1
echo "✅ Backtest complete"

# 3. Extract capital summary
grep -A4 "capital usage summary" results.txt
echo "✅ Capital summary logged"

# 4. Verify numbers are reasonable
if grep -q "Maximum Used.*[0-9].*%" results.txt; then
    echo "✅ Percentage calculations present"
fi

# 5. Check daily stats
./daily-stats
echo "✅ Daily stats running"
```

Expected:
```
✅ Database cleaned
✅ Backtest complete
✅ Capital summary logged
✅ Percentage calculations present
✅ Daily stats running
```

---

## Performance Test (Optional)

### Check Log Output Clarity

```bash
./engine --mode backtest 2>&1 | grep backtest
```

Should show:
```
[backtest] starting backtest run...
[backtest] found 252 trading days with AI data: 2024-01-01 to 2024-12-31
[backtest] starting capital: ₹1,000,000.00
... (trading logs) ...
[backtest] completed: 145 closed trades, 3 still open at end
[backtest] capital usage summary:
  Starting Capital:    ₹1,000,000.00
  Maximum Used:        ₹850,000.00 (85.0% of starting)
  Final Capital:       ₹1,050,000.00
  Total P&L:           ₹50,000.00 (5.00%)
```

✅ Check: Clear, well-formatted output

---

## Troubleshooting

### Issue: "No capital usage summary shown"

**Solution 1:** Make sure backtest is running
```bash
./engine --mode backtest 2>&1 | tail -50
```

**Solution 2:** Check if AI outputs exist
```bash
ls ai_outputs/
```

If empty, run nightly pipeline first:
```bash
python3 -m python_ai.run_scoring --date $(date +%Y-%m-%d)
```

**Solution 3:** Look for errors
```bash
./engine --mode backtest 2>&1 | grep -i error
```

### Issue: "Maximum Used is 0%"

**Problem:** No positions were opened
**Solutions:**
- Check if market regime is BULL
- Check AI score thresholds
- Run longer backtest period

### Issue: "Final Capital less than Starting"

**Problem:** Losing money
**Solutions:**
- This is normal in backtests
- Check if strategy is actually profitable
- Review individual trade results

### Issue: "Percentages don't add up"

**Problem:** Math error in calculation
**Solutions:**
- Check decimal precision (₹XXX,XXX.XX format)
- Verify formula: (Value / StartCapital) × 100
- Compare manual calc to logged value

---

## Expected Results for Different Strategies

### Conservative Strategy
```
Starting Capital:    ₹1,000,000.00
Maximum Used:        ₹500,000.00 (50.0% of starting)
Final Capital:       ₹1,020,000.00
Total P&L:           ₹20,000.00 (2.00%)
```
✅ Safe, consistent

### Balanced Strategy
```
Starting Capital:    ₹1,000,000.00
Maximum Used:        ₹750,000.00 (75.0% of starting)
Final Capital:       ₹1,050,000.00
Total P&L:           ₹50,000.00 (5.00%)
```
✅ Good utilization

### Aggressive Strategy
```
Starting Capital:    ₹1,000,000.00
Maximum Used:        ₹900,000.00 (90.0% of starting)
Final Capital:       ₹1,090,000.00
Total P&L:           ₹90,000.00 (9.00%)
```
✅ High risk, high reward

---

## Validation Checklist

Run through this before going live:

- [ ] Code builds without errors: `go build`
- [ ] Backtest runs: `./engine --mode backtest`
- [ ] Capital metrics appear in output
- [ ] Starting Capital matches config.json
- [ ] Maximum Used is 30-100%
- [ ] Final Capital > Starting Capital (if profitable)
- [ ] Percentages calculate correctly
- [ ] Daily stats work: `./daily-stats`
- [ ] Log files created: `logs/engine_*.log`
- [ ] Can pipe output: `./engine --mode backtest | grep capital`

---

## Quick Verification Script

Save this as `test_capital_usage.sh`:

```bash
#!/bin/bash

echo "🧪 Testing Capital Usage Tracking..."
echo ""

# Build
echo "1️⃣  Building engine..."
go build -o engine ./cmd/engine
if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi
echo "✅ Build successful"
echo ""

# Run backtest
echo "2️⃣  Running backtest..."
./engine --mode backtest > /tmp/backtest_out.txt 2>&1
if [ $? -ne 0 ]; then
    echo "❌ Backtest failed!"
    exit 1
fi
echo "✅ Backtest complete"
echo ""

# Check for metrics
echo "3️⃣  Checking for capital metrics..."
if grep -q "starting capital" /tmp/backtest_out.txt; then
    echo "✅ Starting capital logged"
else
    echo "❌ Starting capital missing"
    exit 1
fi

if grep -q "Maximum Used" /tmp/backtest_out.txt; then
    echo "✅ Maximum Used logged"
else
    echo "❌ Maximum Used missing"
    exit 1
fi

if grep -q "Final Capital" /tmp/backtest_out.txt; then
    echo "✅ Final Capital logged"
else
    echo "❌ Final Capital missing"
    exit 1
fi

if grep -q "Total P&L" /tmp/backtest_out.txt; then
    echo "✅ Total P&L logged"
else
    echo "❌ Total P&L missing"
    exit 1
fi

echo ""
echo "4️⃣  Capital Usage Summary:"
grep -A4 "capital usage summary" /tmp/backtest_out.txt
echo ""
echo "✅ All checks passed! Capital usage tracking is working!"
```

Run it:
```bash
chmod +x test_capital_usage.sh
./test_capital_usage.sh
```

---

## Summary

The capital usage tracking feature is working if you see:

1. ✅ Starting Capital: Matches config.json
2. ✅ Maximum Used: Between 30-100%
3. ✅ Final Capital: Greater than starting (if profitable)
4. ✅ Total P&L: Shows profit/loss and percentage

**Test now:** `./engine --mode backtest`

**Good luck!** 🚀
