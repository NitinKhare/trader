#!/bin/bash
# run_nightly.sh — Nightly pipeline: data sync + AI scoring + watchlist.
#
# This script is the single entry point for the nightly cycle.
# It runs after market close (typically 6-8 PM IST).
#
# The Go engine's nightly mode handles the full pipeline:
#   1. Sync market data from Dhan API
#   2. Execute Python AI scoring pipeline
#   3. Verify AI outputs and generate watchlist
#
# Usage:
#   ./scripts/run_nightly.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONFIG="${PROJECT_ROOT}/config/config.json"

echo "=== Nightly Pipeline ==="
echo "Project root: ${PROJECT_ROOT}"
echo ""

cd "${PROJECT_ROOT}"
go run ./cmd/engine --config "${CONFIG}" --mode nightly

echo ""
echo "=== Nightly Pipeline Complete ==="
