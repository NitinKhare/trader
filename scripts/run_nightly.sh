#!/bin/bash
# run_nightly.sh — Nightly pipeline: data sync + AI scoring + engine jobs.
#
# This script is the single entry point for the nightly cycle.
# It runs after market close (typically 6-8 PM IST).
#
# Steps:
#   1. Sync market data from Dhan API (Go engine nightly job)
#   2. Run Python AI scoring pipeline
#   3. Verify AI outputs and generate watchlist (Go engine nightly job)
#
# The Go engine's nightly mode handles step 1 and 3. Step 2 is the Python scoring.
#
# Usage:
#   ./scripts/run_nightly.sh [YYYY-MM-DD]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default to today's date.
DATE="${1:-$(date +%Y-%m-%d)}"
CONFIG="${PROJECT_ROOT}/config/config.json"

echo "=== Nightly Pipeline: ${DATE} ==="
echo "Project root: ${PROJECT_ROOT}"
echo ""

# Step 1: Sync market data + prepare (Go engine handles Dhan API fetch).
echo "--- Step 1: Market Data Sync (Dhan API) ---"
cd "${PROJECT_ROOT}"
go run ./cmd/engine --config "${CONFIG}" --mode nightly

echo ""

# Step 2: Run AI scoring on the freshly synced data.
echo "--- Step 2: AI Scoring ---"
python3 -m python_ai.run_scoring \
    --date "${DATE}" \
    --output-dir ./ai_outputs \
    --data-dir ./market_data \
    --universe-file ./config/stock_universe.json

echo ""
echo "=== Nightly Pipeline Complete ==="
