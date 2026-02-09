#!/bin/bash
# run_market.sh — Market-hour trading execution.
#
# This script runs during market hours (9:15 AM - 3:30 PM IST).
# It executes pre-planned trades and monitors exits.
#
# Usage:
#   ./scripts/run_market.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

CONFIG="${PROJECT_ROOT}/config/config.json"

echo "=== Market Hour Execution ==="
cd "${PROJECT_ROOT}"
go run ./cmd/engine --config "${CONFIG}" --mode market
echo "=== Market Hour Execution Complete ==="
