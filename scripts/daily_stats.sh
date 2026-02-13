#!/bin/bash

# Daily Trading Statistics - Display in Terminal
# Usage: ./scripts/daily_stats.sh [optional-date: YYYY-MM-DD]

DATE=${1:-$(date +%Y-%m-%d)}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║           DAILY TRADING STATISTICS                         ║${NC}"
echo -e "${CYAN}║           Date: $DATE                            ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Get stats from database
STATS=$(psql -U algo -d algo_trading -h localhost -t -c "
SELECT
  COUNT(*) as total_trades,
  SUM(CASE WHEN exit_price > entry_price THEN 1 ELSE 0 END) as winning_trades,
  SUM(CASE WHEN exit_price < entry_price THEN 1 ELSE 0 END) as losing_trades,
  ROUND(SUM((exit_price - entry_price) * quantity), 2) as daily_pnl,
  ROUND(SUM(ABS(entry_price * quantity)), 2) as capital_used
FROM trades
WHERE DATE(exit_time AT TIME ZONE 'IST') = '$DATE' AND status = 'closed';
")

# Parse the results
IFS='|' read -r TOTAL_TRADES WINNING_TRADES LOSING_TRADES DAILY_PNL CAPITAL_USED <<< "$STATS"

# Trim whitespace
TOTAL_TRADES=$(echo $TOTAL_TRADES | xargs)
WINNING_TRADES=$(echo $WINNING_TRADES | xargs)
LOSING_TRADES=$(echo $LOSING_TRADES | xargs)
DAILY_PNL=$(echo $DAILY_PNL | xargs)
CAPITAL_USED=$(echo $CAPITAL_USED | xargs)

# Check if we have data
if [ -z "$TOTAL_TRADES" ] || [ "$TOTAL_TRADES" = "" ] || [ "$TOTAL_TRADES" = "0" ]; then
    echo -e "${YELLOW}No trades found for $DATE${NC}"
    echo ""
    exit 0
fi

# Format PNL color (green for profit, red for loss)
if (( $(echo "$DAILY_PNL < 0" | bc -l) )); then
    PNL_COLOR=$RED
else
    PNL_COLOR=$GREEN
fi

# Calculate win rate
if [ "$TOTAL_TRADES" -gt 0 ]; then
    WIN_RATE=$(echo "scale=1; ($WINNING_TRADES / $TOTAL_TRADES) * 100" | bc)
else
    WIN_RATE=0
fi

# Display summary
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}SUMMARY${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  ${YELLOW}Total Trades:${NC}      ${GREEN}${TOTAL_TRADES}${NC}"
echo -e "  ${YELLOW}Winning Trades:${NC}    ${GREEN}${WINNING_TRADES}${NC}"
echo -e "  ${YELLOW}Losing Trades:${NC}     ${RED}${LOSING_TRADES}${NC}"
echo -e "  ${YELLOW}Win Rate:${NC}          ${GREEN}${WIN_RATE}%${NC}"
echo ""
echo -e "  ${YELLOW}Daily P&L:${NC}         ${PNL_COLOR}₹${DAILY_PNL}${NC}"
echo -e "  ${YELLOW}Capital Used:${NC}      ${CYAN}₹${CAPITAL_USED}${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Show detailed trades
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}DETAILED TRADES${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

psql -U algo -d algo_trading -h localhost -c "
SELECT
  symbol,
  quantity,
  entry_price,
  exit_price,
  ROUND((exit_price - entry_price) * quantity, 2) as pnl,
  (exit_time AT TIME ZONE 'IST')::time as exit_time
FROM trades
WHERE DATE(exit_time AT TIME ZONE 'IST') = '$DATE' AND status = 'closed'
ORDER BY exit_time DESC;
"

echo ""

# Show open positions
OPEN_POSITIONS=$(psql -U algo -d algo_trading -h localhost -t -c "
SELECT COUNT(*) FROM trades WHERE status = 'open';
")

OPEN_POSITIONS=$(echo $OPEN_POSITIONS | xargs)

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}OPEN POSITIONS${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ "$OPEN_POSITIONS" -eq 0 ]; then
    echo -e "  ${GREEN}No open positions${NC}"
else
    echo -e "  ${YELLOW}Open Positions: ${GREEN}${OPEN_POSITIONS}${NC}"
    echo ""
    psql -U algo -d algo_trading -h localhost -c "
SELECT
  symbol,
  quantity,
  entry_price,
  ROUND((entry_price * quantity), 2) as capital_deployed,
  stop_loss,
  target
FROM trades
WHERE status = 'open'
ORDER BY entry_time DESC;
"
fi

echo ""
echo -e "${CYAN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                    END OF REPORT                           ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════════════════════╝${NC}"
