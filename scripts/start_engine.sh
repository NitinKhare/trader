#!/bin/bash
# start_engine.sh — Smart engine starter with validation and setup
#
# Features:
#   - Fetches market data for today
#   - Generates AI scores
#   - Verifies config.json is set to appropriate mode
#   - Checks if it's a trading day
#   - Starts the engine with proper logging
#   - Requires confirmation for live mode
#
# Usage:
#   ./scripts/start_engine.sh
#   ./scripts/start_engine.sh --force       # Skip confirmations
#   ./scripts/start_engine.sh --dry-run     # Test without trading

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
CONFIG="${PROJECT_ROOT}/config/config.json"
LOG_DIR="${PROJECT_ROOT}/logs"
ENGINE_BIN="${PROJECT_ROOT}/engine"

# Logging
ENGINE_LOG="${LOG_DIR}/engine_$(date +%Y%m%d_%H%M%S).log"
START_LOG="${LOG_DIR}/start_$(date +%Y%m%d_%H%M%S).log"

mkdir -p "${LOG_DIR}"

# Flags
FORCE=false
DRY_RUN=false

log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[${timestamp}] [${level}] ${message}" | tee -a "${START_LOG}"
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
    log "INFO" "$1"
}

error() {
    echo -e "${RED}✗ $1${NC}"
    log "ERROR" "$1"
}

error_exit() {
    error "$1"
    exit 1
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
        --dry-run)
            DRY_RUN=true
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
    echo "║          Algo Trading Engine - Smart Starter               ║"
    echo "║                                                            ║"
    if [ "${DRY_RUN}" = true ]; then
        echo "║          MODE: DRY RUN (No trades will be executed)      ║"
    fi
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Check prerequisites
check_prerequisites() {
    info "Checking prerequisites..."

    # Check configuration file
    if [ ! -f "${CONFIG}" ]; then
        error_exit "Configuration file not found: ${CONFIG}"
    fi
    success "Configuration file found"

    # Check if engine binary exists
    if [ ! -f "${ENGINE_BIN}" ]; then
        warn "Engine binary not found at ${ENGINE_BIN}"
        info "Attempting to build engine..."
        cd "${PROJECT_ROOT}"
        if command -v go >/dev/null 2>&1; then
            if go build -o engine ./cmd/engine; then
                success "Engine built successfully"
            else
                error_exit "Failed to build engine"
            fi
        else
            error_exit "Go is not installed and engine binary is missing"
        fi
    fi
    success "Engine binary ready"

    # Check log directory
    if [ ! -d "${LOG_DIR}" ]; then
        mkdir -p "${LOG_DIR}"
        success "Created log directory"
    fi
}

# Validate configuration
validate_configuration() {
    info "Validating configuration..."

    # Check for required config fields
    if ! grep -q '"active_broker"' "${CONFIG}"; then
        error_exit "Configuration incomplete: missing active_broker"
    fi

    if ! grep -q '"trading_mode"' "${CONFIG}"; then
        error_exit "Configuration incomplete: missing trading_mode"
    fi

    success "Configuration validation passed"

    # Extract trading mode
    local trading_mode=$(grep -o '"trading_mode"\s*:\s*"[^"]*"' "${CONFIG}" | sed 's/"trading_mode"\s*:\s*"\([^"]*\)".*/\1/')
    echo "Trading mode: ${trading_mode}"

    return 0
}

# Check if today is a trading day
is_trading_day() {
    info "Checking if today is a trading day..."

    local holidays_file="${PROJECT_ROOT}/config/holidays_2026.json"

    if [ ! -f "${holidays_file}" ]; then
        warn "Holidays file not found, assuming every weekday is a trading day"
        return 0
    fi

    local today=$(date +%Y-%m-%d)
    local day_of_week=$(date +%A)

    # Check if it's a weekend
    if [ "${day_of_week}" = "Saturday" ] || [ "${day_of_week}" = "Sunday" ]; then
        return 1
    fi

    # Check if it's a holiday (basic check)
    if grep -q "${today}" "${holidays_file}"; then
        return 1
    fi

    return 0
}

# Fetch market data
fetch_market_data() {
    info "Fetching market data for today..."

    local today=$(date +%Y-%m-%d)
    local market_data_dir="${PROJECT_ROOT}/market_data/${today}"

    if [ -d "${market_data_dir}" ]; then
        success "Market data already exists for today"
        return 0
    fi

    # Try to run the nightly fetch
    if [ -f "${SCRIPT_DIR}/run_nightly.sh" ]; then
        info "Running nightly data fetch..."
        if ! timeout 300 bash "${SCRIPT_DIR}/run_nightly.sh" >> "${START_LOG}" 2>&1; then
            warn "Nightly data fetch completed with warnings or timeout"
        else
            success "Market data fetched successfully"
        fi
    else
        warn "Nightly script not found, skipping data fetch"
    fi
}

# Generate AI scores
generate_ai_scores() {
    info "Checking AI scores for today..."

    local today=$(date +%Y-%m-%d)
    local ai_output_dir="${PROJECT_ROOT}/ai_outputs/${today}"

    if [ -d "${ai_output_dir}" ]; then
        if [ -f "${ai_output_dir}/stock_scores.json" ]; then
            success "AI scores already available for today"
            return 0
        fi
    fi

    warn "AI scores not yet generated for today"
    info "Scores will be generated during engine execution"
}

# Check live mode safety
check_live_mode() {
    local trading_mode=$(grep -o '"trading_mode"\s*:\s*"[^"]*"' "${CONFIG}" | sed 's/"trading_mode"\s*:\s*"\([^"]*\)".*/\1/')

    if [ "${trading_mode}" != "live" ]; then
        info "Trading mode: ${trading_mode} (not live)"
        return 0
    fi

    # Live mode checks
    echo ""
    echo -e "${RED}════════════════════════════════════════════════════════════${NC}"
    echo -e "${RED}WARNING: LIVE TRADING MODE ENABLED${NC}"
    echo -e "${RED}════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo "You are about to start the engine in LIVE TRADING MODE."
    echo "This means real money will be used for trading."
    echo ""

    # Get capital from config
    local capital=$(grep -o '"capital"\s*:\s*[^,]*' "${CONFIG}" | head -1 | sed 's/"capital"\s*:\s*//')
    echo "Configured capital: ${capital}"
    echo ""

    # Get max daily loss from config
    local max_daily_loss=$(grep -o '"max_daily_loss_pct"\s*:\s*[^,}]*' "${CONFIG}" | head -1 | sed 's/"max_daily_loss_pct"\s*:\s*//')
    echo "Max daily loss allowed: ${max_daily_loss}%"
    echo ""

    if [ "${FORCE}" != true ]; then
        warn "LIVE MODE: Requires explicit confirmation"
        echo ""
        read -p "$(echo -e ${RED}Type 'I UNDERSTAND THE RISKS' to proceed:${NC}) " confirmation
        if [ "${confirmation}" != "I UNDERSTAND THE RISKS" ]; then
            error_exit "Live mode confirmation failed. Operation cancelled."
        fi
    else
        success "Live mode confirmed (forced)"
    fi

    success "Live mode safety checks passed"
}

# Start the engine
start_trading_engine() {
    info "Starting trading engine..."

    local mode="market"
    if [ "${DRY_RUN}" = true ]; then
        mode="dry-run"
    fi

    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}Engine Output (Log: ${ENGINE_LOG})${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo ""

    cd "${PROJECT_ROOT}"

    # Run engine
    if [ "${DRY_RUN}" = true ]; then
        "${ENGINE_BIN}" --config "${CONFIG}" --mode "${mode}" --confirm-live 2>&1 | tee -a "${ENGINE_LOG}"
    else
        local trading_mode=$(grep -o '"trading_mode"\s*:\s*"[^"]*"' "${CONFIG}" | sed 's/"trading_mode"\s*:\s*"\([^"]*\)".*/\1/')

        if [ "${trading_mode}" = "live" ]; then
            "${ENGINE_BIN}" --config "${CONFIG}" --mode "${mode}" --confirm-live 2>&1 | tee -a "${ENGINE_LOG}"
        else
            "${ENGINE_BIN}" --config "${CONFIG}" --mode "${mode}" 2>&1 | tee -a "${ENGINE_LOG}"
        fi
    fi
}

# Print summary
print_summary() {
    echo ""
    echo -e "${GREEN}"
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║                  ENGINE STARTED                            ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    echo ""
    echo "Engine Log: ${ENGINE_LOG}"
    echo "Start Log: ${START_LOG}"
    echo ""
    echo "To monitor:"
    echo "  tail -f ${ENGINE_LOG}"
    echo ""
    echo "To stop engine:"
    echo "  ./scripts/stop_engine.sh"
    echo ""
}

# Handle errors
trap 'error_exit "Engine startup failed"' ERR

# Main flow
main() {
    print_header

    log "INFO" "Engine start process initiated"

    # Check if already running
    if pgrep -f "${ENGINE_BIN}" > /dev/null 2>&1; then
        warn "Engine appears to be already running"
        read -p "$(echo -e ${YELLOW}Continue anyway?${NC}) (yes/no): " continue_choice
        if [[ "${continue_choice,,}" != "yes" && "${continue_choice,,}" != "y" ]]; then
            error_exit "Operation cancelled"
        fi
    fi

    # Run checks
    check_prerequisites
    validate_configuration

    echo ""

    # Check if trading day
    if ! is_trading_day; then
        warn "Today is not a trading day (weekend or holiday)"
        read -p "$(echo -e ${YELLOW}Continue anyway?${NC}) (yes/no): " continue_choice
        if [[ "${continue_choice,,}" != "yes" && "${continue_choice,,}" != "y" ]]; then
            error_exit "Operation cancelled"
        fi
    else
        success "Today is a trading day"
    fi

    echo ""

    # Fetch data
    fetch_market_data

    echo ""

    # Check AI scores
    generate_ai_scores

    echo ""

    # Safety check for live mode
    check_live_mode

    echo ""

    # Confirm before starting
    if [ "${FORCE}" != true ] && [ "${DRY_RUN}" != true ]; then
        read -p "$(echo -e ${BLUE}Proceed with engine startup?${NC}) (yes/no): " startup_choice
        if [[ "${startup_choice,,}" != "yes" && "${startup_choice,,}" != "y" ]]; then
            error_exit "Startup cancelled"
        fi
    fi

    echo ""

    # Start engine
    start_trading_engine

    print_summary

    log "INFO" "Engine started successfully"
}

# Entry point
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
