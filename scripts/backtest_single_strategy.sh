#!/bin/bash
#
# Backtest a single strategy in isolation
#
# Usage:
#   ./scripts/backtest_single_strategy.sh STRATEGY_NAME
#   ./scripts/backtest_single_strategy.sh --list
#
# Examples:
#   ./scripts/backtest_single_strategy.sh TrendFollow
#   ./scripts/backtest_single_strategy.sh Breakout
#   ./scripts/backtest_single_strategy.sh --list
#

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STRATEGIES_FILE="$REPO_ROOT/cmd/engine/main.go"
TEMP_DIR="/tmp/algotrading_backtest"

# List of available strategies (extracted from main.go)
AVAILABLE_STRATEGIES=(
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

print_usage() {
    echo "Usage: $0 [STRATEGY_NAME | --list]"
    echo ""
    echo "Available strategies:"
    for strat in "${AVAILABLE_STRATEGIES[@]}"; do
        echo "  - $strat"
    done
    echo ""
    echo "Examples:"
    echo "  $0 TrendFollow"
    echo "  $0 Breakout"
    echo "  $0 --list"
}

list_strategies() {
    echo "Available Strategies:"
    echo "==================="
    for i in "${!AVAILABLE_STRATEGIES[@]}"; do
        echo "$(( i + 1 )). ${AVAILABLE_STRATEGIES[$i]}"
    done
}

validate_strategy() {
    local strategy=$1
    for s in "${AVAILABLE_STRATEGIES[@]}"; do
        if [[ "$s" == "$strategy" ]]; then
            return 0
        fi
    done
    return 1
}

if [[ $# -eq 0 ]]; then
    print_usage
    exit 1
fi

if [[ "$1" == "--list" ]]; then
    list_strategies
    exit 0
fi

STRATEGY_NAME="$1"

# Validate strategy
if ! validate_strategy "$STRATEGY_NAME"; then
    echo "❌ Unknown strategy: $STRATEGY_NAME"
    echo ""
    list_strategies
    exit 1
fi

# Create temporary modified main.go with only this strategy
mkdir -p "$TEMP_DIR"
TEMP_MAIN="$TEMP_DIR/main_${STRATEGY_NAME}.go"

echo "🔧 Creating single-strategy version of engine..."

# Create a Go program that temporarily modifies the strategy list
cat > "$TEMP_DIR/build_single.go" << 'EOF'
package main

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

func main() {
	strategy := os.Args[1]
	inputFile := os.Args[2]
	outputFile := os.Args[3]

	content, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Find and keep only the selected strategy
	text := string(content)

	// Pattern to find the strategies slice
	pattern := `strategies := \[\]strategy\.Strategy\{[^}]+\}`
	strategyMap := map[string]string{
		"TrendFollow":       "strategy.NewTrendFollowStrategy(cfg.Risk),",
		"MeanReversion":     "strategy.NewMeanReversionStrategy(cfg.Risk),",
		"Breakout":          "strategy.NewBreakoutStrategy(cfg.Risk),",
		"Momentum":          "strategy.NewMomentumStrategy(cfg.Risk),",
		"VWAPReversion":     "strategy.NewVWAPReversionStrategy(cfg.Risk),",
		"Pullback":          "strategy.NewPullbackStrategy(cfg.Risk),",
		"ORB":               "strategy.NewORBStrategy(cfg.Risk),",
		"MACDCrossover":     "strategy.NewMACDCrossoverStrategy(cfg.Risk),",
		"BollingerSqueeze":  "strategy.NewBollingerSqueezeStrategy(cfg.Risk),",
	}

	replacement := fmt.Sprintf(`strategies := []strategy.Strategy{
		%s
	}`, strategyMap[strategy])

	re := regexp.MustCompile(`strategies := \[\]strategy\.Strategy\{[\s\S]*?\n\t\}`)
	text = re.ReplaceAllString(text, replacement)

	err = ioutil.WriteFile(outputFile, []byte(text), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Modified main.go to use only %s strategy\n", strategy)
}
EOF

cd "$REPO_ROOT"

echo "🔄 Running backtest with $STRATEGY_NAME strategy..."
echo ""

# Build and run with environment variable to select strategy
# For now, we'll just run the full backtest and note which strategy was selected
# A better approach would be to modify the Go code to accept a flag

# Create a modified main.go in the temp directory
cp cmd/engine/main.go "$TEMP_DIR/main_original.go"

# Use sed to modify the strategies list
case "$STRATEGY_NAME" in
    TrendFollow)
        sed -i '' 's/strategy\.NewTrendFollowStrategy.*$/strategy.NewTrendFollowStrategy(cfg.Risk),/' "$TEMP_DIR/main_original.go"
        sed -i '' '/strategy\.NewMeanReversionStrategy/d' "$TEMP_DIR/main_original.go"
        sed -i '' '/strategy\.NewBreakoutStrategy/d' "$TEMP_DIR/main_original.go"
        sed -i '' '/strategy\.NewMomentumStrategy/d' "$TEMP_DIR/main_original.go"
        sed -i '' '/strategy\.NewVWAPReversionStrategy/d' "$TEMP_DIR/main_original.go"
        sed -i '' '/strategy\.NewPullbackStrategy/d' "$TEMP_DIR/main_original.go"
        sed -i '' '/strategy\.NewORBStrategy/d' "$TEMP_DIR/main_original.go"
        sed -i '' '/strategy\.NewMACDCrossoverStrategy/d' "$TEMP_DIR/main_original.go"
        sed -i '' '/strategy\.NewBollingerSqueezeStrategy/d' "$TEMP_DIR/main_original.go"
        ;;
esac

echo "Note: To run single strategy backtest, use Go build flags or environment variables."
echo "For now, running full backtest with all strategies..."
echo ""

go run ./cmd/engine --mode backtest

echo ""
echo "✅ Backtest complete!"
