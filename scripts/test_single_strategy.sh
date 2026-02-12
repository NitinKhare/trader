#!/bin/bash
#
# Test a single strategy in isolation
#
# This script temporarily modifies main.go to run only the specified strategy,
# then restores it after testing.
#
# Usage:
#   ./scripts/test_single_strategy.sh TrendFollow
#   ./scripts/test_single_strategy.sh Breakout --backtest
#   ./scripts/test_single_strategy.sh --list
#

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MAIN_GO="$REPO_ROOT/cmd/engine/main.go"
MAIN_GO_BACKUP="$REPO_ROOT/cmd/engine/main.go.backup"

STRATEGIES=(
    "TrendFollow:strategy.NewTrendFollowStrategy(cfg.Risk),"
    "MeanReversion:strategy.NewMeanReversionStrategy(cfg.Risk),"
    "Breakout:strategy.NewBreakoutStrategy(cfg.Risk),"
    "Momentum:strategy.NewMomentumStrategy(cfg.Risk),"
    "VWAPReversion:strategy.NewVWAPReversionStrategy(cfg.Risk),"
    "Pullback:strategy.NewPullbackStrategy(cfg.Risk),"
    "ORB:strategy.NewORBStrategy(cfg.Risk),"
    "MACDCrossover:strategy.NewMACDCrossoverStrategy(cfg.Risk),"
    "BollingerSqueeze:strategy.NewBollingerSqueezeStrategy(cfg.Risk),"
)

cleanup() {
    if [[ -f "$MAIN_GO_BACKUP" ]]; then
        mv "$MAIN_GO_BACKUP" "$MAIN_GO"
        echo "✅ Restored original main.go"
    fi
}

trap cleanup EXIT

list_strategies() {
    echo "Available Strategies:"
    echo "===================="
    for item in "${STRATEGIES[@]}"; do
        name="${item%%:*}"
        echo "  - $name"
    done
}

if [[ "$1" == "--list" ]]; then
    list_strategies
    exit 0
fi

if [[ -z "$1" ]]; then
    echo "Usage: $0 [STRATEGY_NAME|--list] [--backtest]"
    echo ""
    list_strategies
    exit 1
fi

STRATEGY_NAME="$1"
MODE="status"

if [[ "$2" == "--backtest" ]]; then
    MODE="backtest"
elif [[ "$3" == "--backtest" ]]; then
    MODE="backtest"
fi

# Find the strategy
STRATEGY_CODE=""
for item in "${STRATEGIES[@]}"; do
    name="${item%%:*}"
    if [[ "$name" == "$STRATEGY_NAME" ]]; then
        STRATEGY_CODE="${item#*:}"
        break
    fi
done

if [[ -z "$STRATEGY_CODE" ]]; then
    echo "❌ Unknown strategy: $STRATEGY_NAME"
    echo ""
    list_strategies
    exit 1
fi

# Backup and modify main.go
echo "🔧 Modifying main.go to use only $STRATEGY_NAME strategy..."
cp "$MAIN_GO" "$MAIN_GO_BACKUP"

# Create a temporary Python script to modify the file
python3 << PYTHON_EOF
import re

strategy_name = "$STRATEGY_NAME"
strategy_code = "$STRATEGY_CODE"
main_go = "$MAIN_GO"

with open(main_go, "r") as f:
    content = f.read()

# Find the strategies block and replace it
# Pattern: strategies := []strategy.Strategy{ ... }
pattern = r'strategies := \[\]strategy\.Strategy\{[^}]+\}'
replacement = f'''strategies := []strategy.Strategy{{
\t\t{strategy_code}
\t}}'''

content = re.sub(pattern, replacement, content, flags=re.DOTALL)

with open(main_go, "w") as f:
    f.write(content)

print(f"Modified strategies list to use only: {strategy_name}")
PYTHON_EOF

echo ""
echo "🚀 Running backtest with only $STRATEGY_NAME strategy..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

cd "$REPO_ROOT"
go run ./cmd/engine --mode "$MODE"

echo ""
echo "✅ Strategy backtest complete!"
echo "💡 Original main.go has been restored"
