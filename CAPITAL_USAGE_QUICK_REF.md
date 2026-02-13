# 💰 Capital Usage - Quick Reference

## What Gets Logged in Backtest

```
[backtest] starting capital: ₹1,000,000.00
[backtest] capital usage summary:
  Starting Capital:    ₹1,000,000.00
  Maximum Used:        ₹850,000.00 (85.0% of starting)
  Final Capital:       ₹1,050,000.00
  Total P&L:           ₹50,000.00 (5.00%)
```

## Key Metrics

| Metric | What It Means | Good Range |
|--------|---------------|-----------|
| **Starting Capital** | Total capital for backtest | From config.json |
| **Maximum Used** | Peak capital deployed | 50-80% of starting |
| **% of Starting** | How much of total was used | 50-80% ✅ |
| **Final Capital** | Ending capital after all trades | > Starting ✅ |
| **Total P&L** | Profit or loss | +1% to +5% monthly ✅ |

## Capital Calculation

### Daily Used Capital
```
Capital Used = Sum of (Entry Price × Quantity) for all open positions
```

**Example:**
- Position 1: ₹5000 × 20 = ₹100,000
- Position 2: ₹1000 × 50 = ₹50,000
- **Total Used: ₹150,000**

### Final Capital
```
Final = Starting + Sum of all P&L
```

**Example:**
- Starting: ₹1,000,000
- Trade 1: +₹2,000
- Trade 2: -₹1,000
- Trade 3: +₹49,000
- **Final: ₹1,050,000**

### P&L Percentage
```
Return % = (Final - Starting) / Starting × 100
```

**Example:**
- (1,050,000 - 1,000,000) / 1,000,000 × 100 = **5.0%**

## Go-Live Checklist

Before switching from backtest to live:

- [ ] Maximum Used between 50-80%?
- [ ] Final Capital > Starting Capital?
- [ ] P&L positive and sustainable?
- [ ] Win rate > 40%?
- [ ] Profit factor > 1.0?

If all ✅, you can go live!

## Quick Commands

```bash
# Run backtest and see capital logs
./engine --mode backtest

# Just see capital summary
./engine --mode backtest | grep -A4 "capital usage"

# Run and save to file
./engine --mode backtest > backtest_results.txt 2>&1

# Dry run (no trades placed)
./engine --mode dry-run
```

## Common Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| Max used < 30% | Few positions opening | Lower entry scores |
| Max used > 90% | Too many positions | Increase position limit |
| Final < Starting | Losing money | Check strategy signals |
| P&L = 0% | No trades or breakeven | Verify AI outputs exist |

## Capital Constraints

These are enforced **during backtesting**:

```
✅ Max 1% risk per trade
✅ Max 5 concurrent positions
✅ No over-leveraging (cost ≤ available capital)
✅ Max 70% capital deployment
```

## Expected Performance

### Conservative Strategy
- Maximum Used: 50-60%
- Monthly Return: 1-3%
- Risk: Low

### Balanced Strategy
- Maximum Used: 60-75%
- Monthly Return: 3-8%
- Risk: Medium

### Aggressive Strategy
- Maximum Used: 75-90%
- Monthly Return: 8-15%
- Risk: High

---

**Remember:** Always backtest thoroughly before going live! 📊
