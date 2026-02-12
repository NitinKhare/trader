#!/bin/bash
# cron_setup.sh — Configure cron jobs for automated trading tasks
#
# Sets up the following scheduled tasks:
#   - Pre-market data fetch (8:50 AM IST)
#   - Market open signal (9:10 AM IST)
#   - Market close cleanup (3:35 PM IST)
#   - Weekly analysis (Friday 4 PM IST)
#   - Hourly health checks during market hours
#
# Jobs only run on trading days (Mon-Fri, excluding holidays)
#
# Usage:
#   ./scripts/cron_setup.sh
#   ./scripts/cron_setup.sh --remove   # Remove all cron jobs
#   ./scripts/cron_setup.sh --list     # List current cron jobs

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
LOG_DIR="${PROJECT_ROOT}/logs"

CRON_LOG="${LOG_DIR}/cron_setup_$(date +%Y%m%d_%H%M%S).log"

mkdir -p "${LOG_DIR}"

# Flags
ACTION="add"

log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[${timestamp}] [${level}] ${message}" | tee -a "${CRON_LOG}"
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
        --remove)
            ACTION="remove"
            shift
            ;;
        --list)
            ACTION="list"
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
    echo "║     Algo Trading Agent - Cron Job Configuration            ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Create trading day check script
create_trading_day_check() {
    local check_script="${SCRIPT_DIR}/is_trading_day.sh"

    if [ -f "${check_script}" ]; then
        return
    fi

    cat > "${check_script}" << 'EOF'
#!/bin/bash
# is_trading_day.sh — Check if today is a trading day
# Returns 0 if trading day, 1 otherwise

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOLIDAYS_FILE="${PROJECT_ROOT}/config/holidays_2026.json"

# Get current date and day of week
TODAY=$(date +%Y-%m-%d)
DAY_OF_WEEK=$(date +%A)

# Check if weekend
if [ "${DAY_OF_WEEK}" = "Saturday" ] || [ "${DAY_OF_WEEK}" = "Sunday" ]; then
    exit 1
fi

# Check if holiday
if [ -f "${HOLIDAYS_FILE}" ] && grep -q "${TODAY}" "${HOLIDAYS_FILE}"; then
    exit 1
fi

exit 0
EOF

    chmod +x "${check_script}"
    success "Created trading day check script"
}

# Create pre-market data fetch job
create_premarket_job() {
    local job_name="trading-engine-premarket"
    local schedule="50 8 * * 1-5"  # 8:50 AM, Mon-Fri
    local command="${SCRIPT_DIR}/is_trading_day.sh && cd ${PROJECT_ROOT} && ${SCRIPT_DIR}/run_nightly.sh >> ${LOG_DIR}/premarket_\$(date +\\%Y\\%m\\%d).log 2>&1"

    info "Adding pre-market data fetch job..."
    info "Schedule: Daily at 8:50 AM (Mon-Fri only)"

    add_cron_job "${job_name}" "${schedule}" "${command}"
}

# Create market open job
create_market_open_job() {
    local job_name="trading-engine-market-open"
    local schedule="10 9 * * 1-5"  # 9:10 AM, Mon-Fri
    local command="${SCRIPT_DIR}/is_trading_day.sh && cd ${PROJECT_ROOT} && ${SCRIPT_DIR}/start_engine.sh --force >> ${LOG_DIR}/market_open_\$(date +\\%Y\\%m\\%d).log 2>&1"

    info "Adding market open job..."
    info "Schedule: Daily at 9:10 AM (Mon-Fri only)"

    add_cron_job "${job_name}" "${schedule}" "${command}"
}

# Create market close job
create_market_close_job() {
    local job_name="trading-engine-market-close"
    local schedule="35 15 * * 1-5"  # 3:35 PM, Mon-Fri
    local command="${SCRIPT_DIR}/is_trading_day.sh && cd ${PROJECT_ROOT} && ${SCRIPT_DIR}/stop_engine.sh >> ${LOG_DIR}/market_close_\$(date +\\%Y\\%m\\%d).log 2>&1"

    info "Adding market close cleanup job..."
    info "Schedule: Daily at 3:35 PM (Mon-Fri only)"

    add_cron_job "${job_name}" "${schedule}" "${command}"
}

# Create weekly analysis job
create_weekly_analysis_job() {
    local job_name="trading-engine-weekly-analysis"
    local schedule="0 16 * * 5"  # 4:00 PM Friday
    local command="cd ${PROJECT_ROOT} && python3 ${SCRIPT_DIR}/backtest_strategy.py >> ${LOG_DIR}/weekly_analysis_\$(date +\\%Y\\%m\\%d).log 2>&1"

    info "Adding weekly analysis job..."
    info "Schedule: Friday at 4:00 PM"

    add_cron_job "${job_name}" "${schedule}" "${command}"
}

# Create hourly health check job
create_health_check_job() {
    local job_name="trading-engine-health-check"
    local schedule="0 9-15 * * 1-5"  # Every hour from 9 AM to 3 PM, Mon-Fri
    local command="${SCRIPT_DIR}/is_trading_day.sh && ${SCRIPT_DIR}/health_check.sh --no-alert >> ${LOG_DIR}/health_check_\$(date +\\%Y\\%m\\%d).log 2>&1"

    info "Adding hourly health check job..."
    info "Schedule: Hourly from 9 AM to 3 PM (Mon-Fri only)"

    add_cron_job "${job_name}" "${schedule}" "${command}"
}

# Add cron job
add_cron_job() {
    local name=$1
    local schedule=$2
    local command=$3

    # Get current crontab
    local current_crontab=$(crontab -l 2>/dev/null || echo "")

    # Check if job already exists
    if echo "${current_crontab}" | grep -q "${name}"; then
        warn "Job '${name}' already exists in crontab"
        return
    fi

    # Add job with comment for identification
    local new_crontab="${current_crontab}
# ${name}
${schedule} ${command}"

    # Install new crontab
    echo "${new_crontab}" | crontab -

    success "Added cron job: ${name}"
}

# Remove cron jobs
remove_cron_jobs() {
    info "Removing all trading engine cron jobs..."

    local current_crontab=$(crontab -l 2>/dev/null || echo "")

    if [ -z "${current_crontab}" ]; then
        warn "No crontab found"
        return
    fi

    # Remove lines containing trading-engine jobs
    local new_crontab=$(echo "${current_crontab}" | grep -v "trading-engine" | sed '/^$/N;/^\n$/!P;D')

    if [ -z "${new_crontab}" ]; then
        # Empty crontab
        crontab -r 2>/dev/null || true
        success "All trading engine cron jobs removed (crontab cleared)"
    else
        # Install modified crontab
        echo "${new_crontab}" | crontab -
        success "All trading engine cron jobs removed"
    fi
}

# List cron jobs
list_cron_jobs() {
    info "Current cron jobs:"

    local current_crontab=$(crontab -l 2>/dev/null || echo "")

    if [ -z "${current_crontab}" ]; then
        echo "No crontab found"
        return
    fi

    echo ""
    echo "${current_crontab}" | grep -A1 "trading-engine" | sed 's/^/  /'
    echo ""
}

# Validate cron setup
validate_cron() {
    info "Validating cron configuration..."

    local current_crontab=$(crontab -l 2>/dev/null || echo "")

    if [ -z "${current_crontab}" ]; then
        warn "No crontab found"
        return
    fi

    local job_count=$(echo "${current_crontab}" | grep -c "trading-engine" || echo "0")

    if [ "${job_count}" -gt 0 ]; then
        success "Found ${job_count} trading engine cron job(s)"
    else
        warn "No trading engine cron jobs found"
    fi
}

# Test cron job
test_cron_job() {
    info "Testing cron job execution..."

    if [ -f "${SCRIPT_DIR}/health_check.sh" ]; then
        info "Running health check test..."
        if bash "${SCRIPT_DIR}/health_check.sh" --no-alert >> "${LOG_DIR}/cron_test.log" 2>&1; then
            success "Cron job test passed"
        else
            warn "Cron job test completed with warnings"
        fi
    fi
}

# Print summary
print_summary() {
    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}Cron Configuration Summary${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo ""

    case "${ACTION}" in
        add)
            echo "Scheduled Jobs:"
            echo "  1. Pre-market data fetch      - 8:50 AM (Mon-Fri)"
            echo "  2. Market open signal         - 9:10 AM (Mon-Fri)"
            echo "  3. Market close cleanup       - 3:35 PM (Mon-Fri)"
            echo "  4. Weekly analysis            - Friday 4:00 PM"
            echo "  5. Hourly health checks       - 9 AM-3 PM (Mon-Fri)"
            echo ""
            echo "All jobs include trading day validation"
            echo ""
            echo "View current cron jobs:"
            echo "  crontab -l"
            echo ""
            echo "View logs:"
            echo "  tail -f ${LOG_DIR}/premarket_*.log"
            echo "  tail -f ${LOG_DIR}/market_open_*.log"
            echo "  tail -f ${LOG_DIR}/market_close_*.log"
            ;;
        remove)
            echo "All trading engine cron jobs have been removed."
            echo ""
            echo "View crontab:"
            echo "  crontab -l"
            ;;
        list)
            echo "Current cron configuration:"
            list_cron_jobs
            ;;
    esac

    echo "Log file: ${CRON_LOG}"
    echo ""
}

# Main flow
main() {
    print_header

    log "INFO" "Cron setup started with action: ${ACTION}"

    case "${ACTION}" in
        add)
            echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
            echo -e "${BLUE}Adding Cron Jobs${NC}"
            echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
            echo ""

            # Create trading day check script first
            create_trading_day_check
            echo ""

            # Add jobs
            create_premarket_job
            echo ""

            create_market_open_job
            echo ""

            create_market_close_job
            echo ""

            create_weekly_analysis_job
            echo ""

            create_health_check_job
            echo ""

            # Validate
            validate_cron
            echo ""

            # Test
            test_cron_job
            ;;

        remove)
            echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
            echo -e "${BLUE}Removing Cron Jobs${NC}"
            echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
            echo ""

            remove_cron_jobs
            echo ""

            validate_cron
            ;;

        list)
            list_cron_jobs
            ;;
    esac

    print_summary

    log "INFO" "Cron setup completed successfully"
}

# Entry point
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
