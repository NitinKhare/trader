#!/bin/bash
# setup_db.sh — Initialize the database.
#
# Creates the database and runs migrations.
# Requires Postgres and TimescaleDB to be installed.
#
# Usage:
#   ./scripts/setup_db.sh [database_url]

set -euo pipefail

DB_URL="${1:-postgres://localhost:5432/algo_trading?sslmode=disable}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=== Database Setup ==="
echo "Database: ${DB_URL}"

# Create database if it doesn't exist.
DB_NAME=$(echo "${DB_URL}" | sed -n 's|.*/\([^?]*\).*|\1|p')
echo "Creating database: ${DB_NAME}"
createdb "${DB_NAME}" 2>/dev/null || echo "Database ${DB_NAME} already exists"

# Run migrations.
echo "Running migrations..."
for migration in "${PROJECT_ROOT}"/db/migrations/*.sql; do
    echo "  Applying: $(basename "${migration}")"
    psql "${DB_URL}" -f "${migration}"
done

echo "=== Database Setup Complete ==="
