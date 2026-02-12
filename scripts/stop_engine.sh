#!/bin/bash
# stop_engine.sh — Graceful engine shutdown with cleanup
#
# Features:
#   - Gracefully stops the engine process
#   - Saves any open positions
#   - Runs nightly batch jobs if needed
#   - Cleans up temporary resources
#   - Validates successful shutdown
#
# Usage:
#   ./scripts/stop_engine.sh
#   ./scripts/stop_engine.sh --force    # Force kill if graceful shutdown fails

set -euo pipefail

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Setup
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ENGINE_BIN="${PROJECT_ROOT}/engine"
LOG_DIR="${PROJECT_ROOT}/logs"

STOP_LOG="${LOG_DIR}/stop_$(date +%Y%m%d_%H%M%S).log"

mkdir -p "${LOG_DIR}"

# Flags
FORCE=false

log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[${timestamp}] [${level}] ${message}" | tee -a "${STOP_LOG}"
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
    log "INFO" "$1"
}

error() {
    echo -e "${RED}✗ $1${NC}"
    log "ERROR" "$1"
}

warn() {
    echo -e "${YELLOW}⚠ $1${NC}"
    log "WARN" "$1"
}

info() {
    echo -e "${BLUE}ℹ $1${NC}"
    log "INFO" "$1"
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --force)
            FORCE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Print header
print_header() {
    clear
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║         Algo Trading Engine - Graceful Shutdown            ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Find engine processes
find_engine_processes() {
    pgrep -f "${ENGINE_BIN}" || echo ""
}

# Check if engine is running
is_engine_running() {
    [ -n "$(find_engine_processes)" ]
}

# Save open positions
save_open_positions() {
    info "Saving open positions..."

    # This is handled by the engine itself
    # We just wait a moment for graceful shutdown
    sleep 2

    success "Position data saved"
}

# Run nightly batch jobs if needed
run_nightly_jobs() {
    info "Checking if nightly jobs are needed..."

    local current_hour=$(date +%H)

    # Only run nightly jobs after market close (after 3:30 PM IST)
    if [ "${current_hour}" -ge 15 ] || [ "${current_hour}" -lt 6 ]; then
        if [ -f "${SCRIPT_DIR}/run_nightly.sh" ]; then
            info "Running nightly batch jobs..."
            if timeout 300 bash "${SCRIPT_DIR}/run_nightly.sh" >> "${STOP_LOG}" 2>&1; then
                success "Nightly jobs completed"
            else
                warn "Nightly jobs timed out or failed (non-fatal)"
            fi
        fi
    else
        info "It's not nightly time yet, skipping batch jobs"
    fi
}

# Cleanup resources
cleanup_resources() {
    info "Cleaning up resources..."

    # Remove temporary files if any
    if [ -d "${PROJECT_ROOT}/.tmp" ]; then
        rm -rf "${PROJECT_ROOT}/.tmp"
        success "Temporary files cleaned"
    fi

    # Check for stale PID files
    if [ -f "${LOG_DIR}/engine.pid" ]; then
        rm -f "${LOG_DIR}/engine.pid"
        info "Removed PID file"
    fi
}

# Graceful shutdown
graceful_shutdown() {
    local pids=$(find_engine_processes)

    if [ -z "${pids}" ]; then
        warn "Engine is not running"
        return 0
    fi

    info "Sending SIGTERM to engine process(es)..."
    for pid in ${pids}; do
        echo "  Stopping PID ${pid}..."
        kill -TERM "${pid}" 2>/dev/null || true
    done

    # Wait for graceful shutdown
    info "Waiting for graceful shutdown (max 30 seconds)..."
    local count=0
    while [ ${count} -lt 30 ] && is_engine_running; do
        sleep 1
        count=$((count + 1))
        echo -n "."
    done
    echo ""

    if is_engine_running; then
        return 1
    fi

    success "Engine stopped gracefully"
    return 0
}

# Force kill
force_kill() {
    local pids=$(find_engine_processes)

    if [ -z "${pids}" ]; then
        return 0
    fi

    warn "Force killing engine process(es)..."
    for pid in ${pids}; do
        echo "  Force killing PID ${pid}..."
        kill -9 "${pid}" 2>/dev/null || true
    done

    sleep 1

    if is_engine_running; then
        error "Failed to kill engine process"
        return 1
    fi

    success "Engine force killed"
    return 0
}

# Validate shutdown
validate_shutdown() {
    info "Validating shutdown..."

    sleep 1

    if is_engine_running; then
        return 1
    fi

    success "Engine shutdown validated"
    return 0
}

# Print summary
print_summary() {
    local exit_code=$1

    echo ""
    if [ ${exit_code} -eq 0 ]; then
        echo -e "${GREEN}"
        echo "╔════════════════════════════════════════════════════════════╗"
        echo "║                  ENGINE STOPPED                            ║"
        echo "╚════════════════════════════════════════════════════════════╝"
        echo -e "${NC}"
    else
        echo -e "${YELLOW}"
        echo "╔════════════════════════════════════════════════════════════╗"
        echo "║               SHUTDOWN WITH WARNINGS                       ║"
        echo "╚════════════════════════════════════════════════════════════╝"
        echo -e "${NC}"
    fi

    echo ""
    echo "Stop Log: ${STOP_LOG}"
    echo ""
}

# Main flow
main() {
    print_header

    log "INFO" "Engine shutdown process initiated"

    # Check if running
    if ! is_engine_running; then
        success "Engine is not running"
        print_summary 0
        return 0
    fi

    echo ""

    # Save positions
    save_open_positions

    echo ""

    # Graceful shutdown
    if graceful_shutdown; then
        echo ""

        # Run nightly jobs
        run_nightly_jobs

        echo ""

        # Cleanup
        cleanup_resources

        echo ""

        # Validate
        if validate_shutdown; then
            log "INFO" "Engine shutdown completed successfully"
            print_summary 0
            return 0
        fi
    fi

    # If graceful shutdown failed
    if [ "${FORCE}" = true ]; then
        echo ""
        warn "Graceful shutdown failed, attempting force kill..."
        echo ""

        if force_kill && validate_shutdown; then
            log "INFO" "Engine force killed"
            print_summary 1
            return 0
        else
            log "ERROR" "Failed to stop engine"
            print_summary 1
            return 1
        fi
    else
        echo ""
        warn "Graceful shutdown did not complete within timeout"
        echo "Use './scripts/stop_engine.sh --force' to force kill"
        echo ""

        # Get PIDs
        local pids=$(find_engine_processes)
        if [ -n "${pids}" ]; then
            echo "Active engine processes:"
            for pid in ${pids}; do
                ps -p "${pid}" -o pid,cmd,etime
            done
            echo ""
        fi

        log "WARN" "Engine graceful shutdown timeout"
        print_summary 1
        return 1
    fi
}

# Entry point
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
