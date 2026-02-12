#!/bin/bash
# health_check.sh — Monitor engine health and system status
#
# Features:
#   - Checks if engine is running
#   - Verifies recent logs
#   - Checks database connection
#   - Validates configuration
#   - Reports issues
#   - Can run as a cron job or manually
#
# Usage:
#   ./scripts/health_check.sh
#   ./scripts/health_check.sh --verbose    # Detailed output
#   ./scripts/health_check.sh --json       # JSON output for monitoring systems

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
CONFIG="${PROJECT_ROOT}/config/config.json"
LOG_DIR="${PROJECT_ROOT}/logs"

HEALTH_LOG="${LOG_DIR}/health_check_$(date +%Y%m%d_%H%M%S).log"

mkdir -p "${LOG_DIR}"

# Flags
VERBOSE=false
JSON_OUTPUT=false
ALERT_ON_FAILURE=true

# Health status
OVERALL_STATUS="OK"
ISSUES=()
WARNINGS=()

log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[${timestamp}] [${level}] ${message}" >> "${HEALTH_LOG}"
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
    log "INFO" "$1"
}

error_check() {
    echo -e "${RED}✗ $1${NC}"
    OVERALL_STATUS="FAILED"
    ISSUES+=("$1")
    log "ERROR" "$1"
}

warn_check() {
    echo -e "${YELLOW}⚠ $1${NC}"
    if [ "${OVERALL_STATUS}" != "FAILED" ]; then
        OVERALL_STATUS="WARNING"
    fi
    WARNINGS+=("$1")
    log "WARN" "$1"
}

info_check() {
    echo -e "${BLUE}ℹ $1${NC}"
    log "INFO" "$1"
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --verbose)
            VERBOSE=true
            shift
            ;;
        --json)
            JSON_OUTPUT=true
            shift
            ;;
        --no-alert)
            ALERT_ON_FAILURE=false
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
    if [ "${JSON_OUTPUT}" = false ]; then
        clear
        echo -e "${BLUE}"
        echo "╔════════════════════════════════════════════════════════════╗"
        echo "║     Algo Trading Engine - Health Check Report              ║"
        echo "║     $(date '+%Y-%m-%d %H:%M:%S')                                          ║"
        echo "╚════════════════════════════════════════════════════════════╝"
        echo -e "${NC}"
        echo ""
    fi
}

# Check engine process
check_engine_process() {
    info_check "Checking engine process..."

    if pgrep -f "${ENGINE_BIN}" > /dev/null 2>&1; then
        local pids=$(pgrep -f "${ENGINE_BIN}")
        success "Engine is running (PID: ${pids})"

        if [ "${VERBOSE}" = true ]; then
            for pid in ${pids}; do
                ps -p "${pid}" -o pid,cmd,etime,lstart
            done
        fi
    else
        error_check "Engine is not running"
    fi
}

# Check engine logs
check_engine_logs() {
    info_check "Checking recent engine logs..."

    if [ ! -d "${LOG_DIR}" ] || [ -z "$(ls -1 "${LOG_DIR}"/*engine*.log 2>/dev/null)" ]; then
        warn_check "No engine logs found"
        return
    fi

    local latest_log=$(ls -1t "${LOG_DIR}"/engine*.log 2>/dev/null | head -1)

    if [ -z "${latest_log}" ]; then
        warn_check "No recent engine logs found"
        return
    fi

    # Check log modification time
    local log_age_seconds=$(( $(date +%s) - $(stat -f%m "${latest_log}" 2>/dev/null || stat -c%Y "${latest_log}") ))
    local log_age_minutes=$(( log_age_seconds / 60 ))

    if [ ${log_age_minutes} -lt 5 ]; then
        success "Recent logs found (${log_age_minutes} minutes old)"
    elif [ ${log_age_minutes} -lt 60 ]; then
        warn_check "Logs are ${log_age_minutes} minutes old"
    else
        warn_check "Logs are ${log_age_minutes} minutes old (may indicate engine not running)"
    fi

    # Check for errors in logs
    if [ "${VERBOSE}" = true ] && [ -f "${latest_log}" ]; then
        echo ""
        info_check "Last 10 log lines:"
        tail -10 "${latest_log}" | sed 's/^/  /'
        echo ""
    fi

    # Check for critical errors
    if grep -q "ERROR\|FATAL\|PANIC" "${latest_log}" 2>/dev/null; then
        error_count=$(grep -c "ERROR\|FATAL\|PANIC" "${latest_log}")
        warn_check "Found ${error_count} error(s) in recent logs"

        if [ "${VERBOSE}" = true ]; then
            echo ""
            info_check "Error lines:"
            grep "ERROR\|FATAL\|PANIC" "${latest_log}" | tail -5 | sed 's/^/  /'
            echo ""
        fi
    else
        success "No critical errors in recent logs"
    fi
}

# Check configuration
check_configuration() {
    info_check "Checking configuration..."

    if [ ! -f "${CONFIG}" ]; then
        error_check "Configuration file not found: ${CONFIG}"
        return
    fi
    success "Configuration file exists"

    # Check for required fields
    if ! grep -q '"active_broker"' "${CONFIG}"; then
        error_check "Configuration missing: active_broker"
        return
    fi

    if ! grep -q '"trading_mode"' "${CONFIG}"; then
        error_check "Configuration missing: trading_mode"
        return
    fi

    success "Configuration has required fields"

    # Check trading mode
    local trading_mode=$(grep -o '"trading_mode"\s*:\s*"[^"]*"' "${CONFIG}" | sed 's/"trading_mode"\s*:\s*"\([^"]*\)".*/\1/')
    info_check "Trading mode: ${trading_mode}"

    # Get capital info
    local capital=$(grep -o '"capital"\s*:\s*[^,]*' "${CONFIG}" | head -1 | sed 's/"capital"\s*:\s*//')
    if [ -n "${capital}" ]; then
        info_check "Configured capital: ${capital}"
    fi
}

# Check database connection
check_database() {
    info_check "Checking database connection..."

    if ! command -v psql &> /dev/null; then
        warn_check "PostgreSQL client (psql) not installed, skipping database check"
        return
    fi

    local db_url=$(grep -o '"database_url"\s*:\s*"[^"]*"' "${CONFIG}" | sed 's/"database_url"\s*:\s*"\([^"]*\)".*/\1/')

    if [ -z "${db_url}" ]; then
        warn_check "Database URL not found in configuration"
        return
    fi

    if timeout 5 psql "${db_url}" -c "SELECT 1" > /dev/null 2>&1; then
        success "Database connection successful"

        if [ "${VERBOSE}" = true ]; then
            # Get database info
            psql "${db_url}" -c "SELECT version();" 2>/dev/null | head -3
        fi
    else
        error_check "Database connection failed"
    fi
}

# Check disk space
check_disk_space() {
    info_check "Checking disk space..."

    local disk_usage=$(df -h "${PROJECT_ROOT}" | tail -1 | awk '{print $5}' | sed 's/%//')

    if [ "${disk_usage}" -lt 80 ]; then
        success "Disk usage: ${disk_usage}%"
    elif [ "${disk_usage}" -lt 90 ]; then
        warn_check "Disk usage: ${disk_usage}% (warning)"
    else
        error_check "Disk usage: ${disk_usage}% (critical)"
    fi
}

# Check system resources
check_system_resources() {
    info_check "Checking system resources..."

    local cpu_usage=$(ps aux | grep "${ENGINE_BIN}" | grep -v grep | awk '{sum+=$3} END {print sum}')
    if [ -z "${cpu_usage}" ]; then
        cpu_usage="0"
    fi

    local mem_usage=$(ps aux | grep "${ENGINE_BIN}" | grep -v grep | awk '{sum+=$4} END {print sum}')
    if [ -z "${mem_usage}" ]; then
        mem_usage="0"
    fi

    info_check "Engine CPU usage: ${cpu_usage}%"
    info_check "Engine memory usage: ${mem_usage}%"

    if [ "${cpu_usage%.*}" -gt 50 ]; then
        warn_check "High CPU usage detected"
    fi

    if [ "${mem_usage%.*}" -gt 30 ]; then
        warn_check "High memory usage detected"
    fi
}

# Check dependencies
check_dependencies() {
    info_check "Checking dependencies..."

    local missing=()

    if ! command -v go &> /dev/null; then
        missing+=("Go")
    fi

    if ! command -v python3 &> /dev/null; then
        missing+=("Python3")
    fi

    if ! command -v psql &> /dev/null; then
        missing+=("PostgreSQL client")
    fi

    if [ ${#missing[@]} -eq 0 ]; then
        success "All required dependencies are installed"
    else
        warn_check "Missing dependencies: ${missing[*]}"
    fi
}

# Check AI outputs
check_ai_outputs() {
    info_check "Checking AI outputs..."

    local today=$(date +%Y-%m-%d)
    local ai_dir="${PROJECT_ROOT}/ai_outputs/${today}"

    if [ -d "${ai_dir}" ]; then
        if [ -f "${ai_dir}/stock_scores.json" ]; then
            success "AI scores available for today"
        else
            warn_check "AI scores directory exists but stock_scores.json not found"
        fi
    else
        warn_check "No AI outputs for today yet"
    fi
}

# Generate JSON output
output_json() {
    cat << EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "status": "${OVERALL_STATUS}",
  "issues": [
    $(printf '"%s"' "${ISSUES[@]}" | paste -sd ',' -)
  ],
  "warnings": [
    $(printf '"%s"' "${WARNINGS[@]}" | paste -sd ',' -)
  ],
  "details": {
    "engine_running": $(pgrep -f "${ENGINE_BIN}" > /dev/null 2>&1 && echo "true" || echo "false"),
    "config_valid": $([ -f "${CONFIG}" ] && echo "true" || echo "false"),
    "database_connected": "unknown"
  }
}
EOF
}

# Print summary
print_summary() {
    if [ "${JSON_OUTPUT}" = true ]; then
        output_json
        return
    fi

    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}Summary${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo ""

    echo -n "Overall Status: "
    case "${OVERALL_STATUS}" in
        OK)
            echo -e "${GREEN}OK${NC}"
            ;;
        WARNING)
            echo -e "${YELLOW}WARNING${NC}"
            ;;
        FAILED)
            echo -e "${RED}FAILED${NC}"
            ;;
    esac

    echo ""

    if [ ${#ISSUES[@]} -gt 0 ]; then
        echo -e "${RED}Issues (${#ISSUES[@]}):${NC}"
        printf '  - %s\n' "${ISSUES[@]}"
        echo ""
    fi

    if [ ${#WARNINGS[@]} -gt 0 ]; then
        echo -e "${YELLOW}Warnings (${#WARNINGS[@]}):${NC}"
        printf '  - %s\n' "${WARNINGS[@]}"
        echo ""
    fi

    echo "Health Check Log: ${HEALTH_LOG}"
    echo ""

    # Alert if needed
    if [ "${ALERT_ON_FAILURE}" = true ] && [ "${OVERALL_STATUS}" != "OK" ]; then
        echo -e "${YELLOW}⚠ ALERT: System health issues detected${NC}"
        echo "Review logs for details: ${HEALTH_LOG}"
    fi
}

# Main flow
main() {
    print_header

    log "INFO" "Health check started"

    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}Engine Status${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo ""

    check_engine_process
    echo ""
    check_engine_logs

    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}System Status${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo ""

    check_configuration
    echo ""
    check_database
    echo ""
    check_dependencies
    echo ""
    check_ai_outputs
    echo ""
    check_disk_space
    echo ""
    check_system_resources

    echo ""

    print_summary

    log "INFO" "Health check completed with status: ${OVERALL_STATUS}"

    # Exit code based on status
    case "${OVERALL_STATUS}" in
        OK)
            exit 0
            ;;
        WARNING)
            exit 1
            ;;
        FAILED)
            exit 2
            ;;
    esac
}

# Entry point
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
