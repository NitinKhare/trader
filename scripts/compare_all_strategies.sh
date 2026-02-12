#!/bin/bash
#
# Compare all strategies in sequence
#
# Usage:
#   ./scripts/compare_all_strategies.sh
#   ./scripts/compare_all_strategies.sh --json
#

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

STRATEGIES=(
    "TrendFollow"
    "MeanReversion"
    "Breakout"
    "Momentum"
    "VWAPReversion"
    "Pullback"
    "ORB"
    "MACDCrossover"
    "BollingerSqueeze"
)

OUTPUT_JSON=${1:-""}

cd "$REPO_ROOT"

echo "📊 Strategy Comparison Report"
echo "════════════════════════════════════════════════════════════════"
echo ""
echo "Testing all 9 strategies individually..."
echo ""

declare -A results

for strategy in "${STRATEGIES[@]}"; do
    echo "🔄 Testing $strategy..."

    output=$(bash scripts/test_single_strategy.sh "$strategy" --backtest 2>&1)

    # Extract metrics
    total_trades=$(echo "$output" | grep "Total trades:" | grep -oE '[0-9]+' | head -1)
    winning=$(echo "$output" | grep "Winning trades:" | grep -oE '[0-9]+' | head -1)
    win_pct=$(echo "$output" | grep "Winning trades:" | grep -oE '[0-9]+\.[0-9]+%' | head -1)
    pnl=$(echo "$output" | grep "Total P&L:" | grep -oE '₹[^[:space:]]+')
    profit_factor=$(echo "$output" | grep "Profit factor:" | grep -oE '[0-9]+\.[0-9]+')
    sharpe=$(echo "$output" | grep "Sharpe ratio:" | grep -oE '[0-9\.-]+' | head -1)

    results[$strategy]="$total_trades|$winning|$win_pct|$pnl|$profit_factor|$sharpe"

    echo "  ✅ Complete"
    echo ""
done

# Display results table
echo ""
echo "📈 Results Summary"
echo "════════════════════════════════════════════════════════════════"
echo ""
printf "%-20s | %8s | %8s | %10s | %10s | %10s\n" \
    "Strategy" "Trades" "Win %" "P&L" "P-Factor" "Sharpe"
echo "────────────────────┼──────────┼──────────┼────────────┼────────────┼──────────"

# Sort by P&L (descending)
for strategy in "${STRATEGIES[@]}"; do
    IFS='|' read -r trades wins win_pct pnl pf sharpe <<< "${results[$strategy]}"

    # Extract numeric values for sorting
    pnl_num=$(echo "$pnl" | grep -oE '[0-9-]+' | head -1)

    printf "%-20s | %8s | %8s | %10s | %10s | %10s\n" \
        "$strategy" "$trades" "$win_pct" "$pnl" "$pf" "$sharpe"
done

echo "════════════════════════════════════════════════════════════════"
echo ""
echo "✅ Strategy comparison complete!"
echo ""
echo "💡 Recommendations:"
echo "   - Deploy strategies with Sharpe ratio > 1.0"
echo "   - Avoid strategies with negative P&L"
echo "   - Consider profit factor > 1.2 as minimum threshold"
echo ""
