#!/bin/bash
# setup.sh — Comprehensive setup system for the trading engine
#
# This is the main entry point for setting up the trading engine.
# It detects the OS, deployment type, and guides the user through setup.
#
# Usage:
#   ./scripts/setup.sh
#
# Features:
#   - Detects OS (Linux/Mac/Windows via WSL)
#   - Interactive setup for local vs cloud deployment
#   - Installs all dependencies
#   - Sets up database
#   - Configures cron jobs or systemd services
#   - Creates health monitoring scripts
#   - Validates configuration

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Setup directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOG_DIR="${PROJECT_ROOT}/logs"
CONFIG_DIR="${PROJECT_ROOT}/config"

# Logging setup
SETUP_LOG="${LOG_DIR}/setup_$(date +%Y%m%d_%H%M%S).log"
mkdir -p "${LOG_DIR}"

# Logging function
log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[${timestamp}] [${level}] ${message}" | tee -a "${SETUP_LOG}"
}

# Error handling
error_exit() {
    log "ERROR" "$1"
    exit 1
}

# Success message
success() {
    echo -e "${GREEN}✓ $1${NC}"
    log "INFO" "$1"
}

# Warning message
warn() {
    echo -e "${YELLOW}⚠ $1${NC}"
    log "WARN" "$1"
}

# Info message
info() {
    echo -e "${BLUE}ℹ $1${NC}"
    log "INFO" "$1"
}

# Interactive prompt with default value
prompt() {
    local prompt_text=$1
    local default=$2
    local response=""

    read -p "$(echo -e ${BLUE}$prompt_text${NC}) [$default]: " response
    echo "${response:-$default}"
}

# Yes/No prompt
confirm() {
    local prompt_text=$1
    local response=""

    while true; do
        read -p "$(echo -e ${BLUE}$prompt_text${NC}) (yes/no): " response
        case "${response,,}" in
            yes|y) return 0 ;;
            no|n) return 1 ;;
            *) echo "Please answer yes or no" ;;
        esac
    done
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "Linux" ;;
        Darwin*)    echo "MacOS" ;;
        MINGW*|MSYS*) echo "Windows" ;;
        *)          echo "Unknown" ;;
    esac
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Print header
print_header() {
    clear
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║     Algo Trading Agent - Comprehensive Setup System        ║"
    echo "║                                                            ║"
    echo "║     This will configure your trading engine for:           ║"
    echo "║     - Dependencies installation                            ║"
    echo "║     - Database setup                                       ║"
    echo "║     - Deployment configuration                            ║"
    echo "║     - Automated job scheduling                            ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Main setup flow
main() {
    print_header

    info "Starting setup process..."
    info "Log file: ${SETUP_LOG}"
    log "INFO" "Setup started on $(date)"

    # Detect OS
    local OS=$(detect_os)
    info "Detected OS: ${OS}"

    if [ "${OS}" = "Unknown" ]; then
        error_exit "Unsupported operating system"
    fi

    # Check if scripts exist
    if [ ! -f "${SCRIPT_DIR}/install_dependencies.sh" ]; then
        error_exit "install_dependencies.sh not found in scripts directory"
    fi

    # Step 1: Dependency Installation
    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}Step 1: Dependency Installation${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"

    if confirm "Install dependencies (Go, Python, PostgreSQL client, Docker)?"; then
        chmod +x "${SCRIPT_DIR}/install_dependencies.sh"
        if ! "${SCRIPT_DIR}/install_dependencies.sh"; then
            error_exit "Dependency installation failed"
        fi
    else
        warn "Skipping dependency installation. Make sure all dependencies are installed."
    fi

    # Step 2: Validate Configuration
    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}Step 2: Configuration Validation${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"

    validate_configuration

    # Step 3: Database Setup
    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}Step 3: Database Setup${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"

    setup_database

    # Step 4: Determine deployment type
    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}Step 4: Deployment Type Selection${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"

    local deployment_type=$(select_deployment_type)
    info "Selected deployment type: ${deployment_type}"

    # Step 5: Setup based on deployment type
    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}Step 5: Deployment Configuration${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"

    case "${deployment_type}" in
        "local")
            setup_local_deployment "${OS}"
            ;;
        "server")
            setup_server_deployment "${OS}"
            ;;
        "docker")
            setup_docker_deployment
            ;;
        "cloud")
            setup_cloud_deployment
            ;;
    esac

    # Step 6: Health check
    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}Step 6: Health Check${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"

    run_health_check

    # Final summary
    print_summary "${deployment_type}"
}

# Validate configuration
validate_configuration() {
    info "Validating configuration..."

    if [ ! -f "${CONFIG_DIR}/config.json" ]; then
        error_exit "config.json not found at ${CONFIG_DIR}/config.json"
    fi

    success "Configuration file found"

    # Check for required fields
    local client_id=$(grep -o '"client_id"\s*:\s*"[^"]*"' "${CONFIG_DIR}/config.json" | head -1)
    local access_token=$(grep -o '"access_token"\s*:\s*"[^"]*"' "${CONFIG_DIR}/config.json" | head -1)

    if [ -z "${client_id}" ] || [ -z "${access_token}" ]; then
        warn "Dhan API credentials not found in config.json"
        info "You need to add Dhan API credentials manually:"
        echo "  1. Go to https://dhanhq.co/ and get your credentials"
        echo "  2. Update ${CONFIG_DIR}/config.json with:"
        echo "     - client_id"
        echo "     - access_token"

        if ! confirm "Continue without validating credentials?"; then
            error_exit "Setup cancelled"
        fi
    else
        success "Dhan API credentials found in config"
    fi

    # Check trading mode
    local trading_mode=$(grep -o '"trading_mode"\s*:\s*"[^"]*"' "${CONFIG_DIR}/config.json" | head -1)
    if echo "${trading_mode}" | grep -q "live"; then
        warn "WARNING: Trading mode is set to LIVE"
        warn "Make sure you understand the risks of live trading"
        if ! confirm "Continue with LIVE trading mode?"; then
            error_exit "Setup cancelled"
        fi
    else
        success "Trading mode is set to paper/backtest"
    fi
}

# Setup database
setup_database() {
    info "Setting up database..."

    # Check if PostgreSQL client is available
    if ! command_exists psql; then
        error_exit "PostgreSQL client (psql) not found. Install PostgreSQL first."
    fi

    # Get database URL from config or use default
    local db_url=$(grep -o '"database_url"\s*:\s*"[^"]*"' "${CONFIG_DIR}/config.json" | sed 's/"database_url"\s*:\s*"\([^"]*\)".*/\1/')

    if [ -z "${db_url}" ]; then
        warn "Database URL not found in config.json. Using default."
        db_url="postgres://algo:algo123@localhost:5432/algo_trading?sslmode=disable"
    fi

    info "Database URL: ${db_url}"

    if confirm "Run database setup now?"; then
        chmod +x "${SCRIPT_DIR}/setup_db.sh"
        if "${SCRIPT_DIR}/setup_db.sh" "${db_url}"; then
            success "Database setup completed"
        else
            warn "Database setup failed or already exists"
        fi
    else
        warn "Skipping database setup. Run './scripts/setup_db.sh' later."
    fi
}

# Select deployment type
select_deployment_type() {
    echo ""
    echo "Select deployment type:"
    echo "  1) Local machine (for development/testing)"
    echo "  2) Server (Linux with systemd)"
    echo "  3) Docker (containerized)"
    echo "  4) Cloud (AWS/GCP/Azure)"
    echo ""

    local choice=$(prompt "Enter your choice" "1")

    case "${choice}" in
        1) echo "local" ;;
        2) echo "server" ;;
        3) echo "docker" ;;
        4) echo "cloud" ;;
        *)
            warn "Invalid choice, defaulting to local"
            echo "local"
            ;;
    esac
}

# Setup local deployment
setup_local_deployment() {
    local os=$1
    info "Setting up local deployment for ${os}..."

    # Create necessary directories
    mkdir -p "${LOG_DIR}"
    mkdir -p "${PROJECT_ROOT}/ai_outputs"
    mkdir -p "${PROJECT_ROOT}/market_data"

    success "Created necessary directories"

    # Create start script
    info "Creating start script..."
    chmod +x "${SCRIPT_DIR}/start_engine.sh"
    success "Start script ready at ${SCRIPT_DIR}/start_engine.sh"

    # Create stop script
    chmod +x "${SCRIPT_DIR}/stop_engine.sh"
    success "Stop script ready at ${SCRIPT_DIR}/stop_engine.sh"

    # Offer to setup cron for local
    echo ""
    if confirm "Setup cron jobs for automated scheduling?"; then
        chmod +x "${SCRIPT_DIR}/cron_setup.sh"
        if "${SCRIPT_DIR}/cron_setup.sh"; then
            success "Cron jobs configured"
        else
            warn "Cron job setup failed"
        fi
    fi

    # Create health check script
    info "Creating health check script..."
    chmod +x "${SCRIPT_DIR}/health_check.sh"
    success "Health check script ready at ${SCRIPT_DIR}/health_check.sh"
}

# Setup server deployment
setup_server_deployment() {
    local os=$1
    info "Setting up server deployment for ${os}..."

    if [ "${os}" != "Linux" ]; then
        warn "Server deployment is optimized for Linux. You are on ${os}."
    fi

    # Create necessary directories
    mkdir -p "${LOG_DIR}"
    mkdir -p "${PROJECT_ROOT}/ai_outputs"
    mkdir -p "${PROJECT_ROOT}/market_data"

    success "Created necessary directories"

    # Offer to create systemd service
    if [ "${os}" = "Linux" ] && command_exists systemctl; then
        echo ""
        if confirm "Create systemd service for continuous operation?"; then
            create_systemd_service
            success "Systemd service created"
        fi
    fi

    # Setup cron for scheduled jobs
    echo ""
    if confirm "Setup cron jobs for scheduled tasks?"; then
        chmod +x "${SCRIPT_DIR}/cron_setup.sh"
        if "${SCRIPT_DIR}/cron_setup.sh"; then
            success "Cron jobs configured"
        else
            warn "Cron job setup failed"
        fi
    fi

    # Create health monitoring
    chmod +x "${SCRIPT_DIR}/health_check.sh"
    success "Health check script ready at ${SCRIPT_DIR}/health_check.sh"
}

# Create systemd service
create_systemd_service() {
    info "Creating systemd service..."

    local service_file="/tmp/algo-trading-engine.service"

    cat > "${service_file}" << 'EOF'
[Unit]
Description=Algo Trading Engine
After=network.target

[Service]
Type=simple
User=%USER%
WorkingDirectory=%WORK_DIR%
ExecStart=%BIN_PATH%/scripts/start_engine.sh
Restart=always
RestartSec=10
StandardOutput=append:%LOG_DIR%/engine.log
StandardError=append:%LOG_DIR%/engine.log

[Install]
WantedBy=multi-user.target
EOF

    # Replace placeholders
    sed -i.bak "s|%USER%|${USER}|g" "${service_file}"
    sed -i.bak "s|%WORK_DIR%|${PROJECT_ROOT}|g" "${service_file}"
    sed -i.bak "s|%BIN_PATH%|${PROJECT_ROOT}|g" "${service_file}"
    sed -i.bak "s|%LOG_DIR%|${LOG_DIR}|g" "${service_file}"

    info "Service file location: ${service_file}"
    info "To install, run:"
    echo "  sudo cp ${service_file} /etc/systemd/system/"
    echo "  sudo systemctl daemon-reload"
    echo "  sudo systemctl enable algo-trading-engine"
    echo "  sudo systemctl start algo-trading-engine"
}

# Setup Docker deployment
setup_docker_deployment() {
    info "Setting up Docker deployment..."

    if ! command_exists docker; then
        error_exit "Docker is not installed. Install Docker first."
    fi

    success "Docker is installed"

    if confirm "Create Dockerfile and docker-compose.yml?"; then
        chmod +x "${SCRIPT_DIR}/docker_setup.sh"
        if "${SCRIPT_DIR}/docker_setup.sh"; then
            success "Docker setup completed"
        else
            error_exit "Docker setup failed"
        fi
    fi
}

# Setup Cloud deployment
setup_cloud_deployment() {
    info "Setting up cloud deployment preparation..."

    if confirm "Prepare code for cloud deployment (AWS/GCP/Azure)?"; then
        chmod +x "${SCRIPT_DIR}/cloud_deploy.sh"
        if "${SCRIPT_DIR}/cloud_deploy.sh"; then
            success "Cloud deployment package created"
        else
            warn "Cloud deployment preparation failed"
        fi
    fi
}

# Run health check
run_health_check() {
    info "Running health check..."

    # Check Go installation
    if command_exists go; then
        local go_version=$(go version | awk '{print $3}')
        success "Go ${go_version} is installed"
    else
        warn "Go is not installed"
    fi

    # Check Python installation
    if command_exists python3; then
        local python_version=$(python3 --version | awk '{print $2}')
        success "Python ${python_version} is installed"
    else
        warn "Python3 is not installed"
    fi

    # Check PostgreSQL client
    if command_exists psql; then
        success "PostgreSQL client is installed"
    else
        warn "PostgreSQL client is not installed"
    fi

    # Check configuration
    if [ -f "${CONFIG_DIR}/config.json" ]; then
        success "Configuration file exists"
    else
        warn "Configuration file not found"
    fi

    # Check database connectivity
    local db_url=$(grep -o '"database_url"\s*:\s*"[^"]*"' "${CONFIG_DIR}/config.json" | sed 's/"database_url"\s*:\s*"\([^"]*\)".*/\1/')
    if [ -n "${db_url}" ] && command_exists psql; then
        if timeout 5 psql "${db_url}" -c "SELECT 1" >/dev/null 2>&1; then
            success "Database connection successful"
        else
            warn "Database connection failed"
        fi
    fi
}

# Print summary
print_summary() {
    local deployment_type=$1

    echo ""
    echo -e "${GREEN}"
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║                    SETUP COMPLETED                         ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    echo ""
    echo "Deployment Type: ${deployment_type}"
    echo "Project Root: ${PROJECT_ROOT}"
    echo "Log Directory: ${LOG_DIR}"
    echo "Setup Log: ${SETUP_LOG}"
    echo ""

    echo "Next steps:"
    echo ""

    case "${deployment_type}" in
        "local")
            echo "1. Start the engine:"
            echo "   ./scripts/start_engine.sh"
            echo ""
            echo "2. Monitor health:"
            echo "   ./scripts/health_check.sh"
            echo ""
            echo "3. Stop the engine:"
            echo "   ./scripts/stop_engine.sh"
            ;;
        "server")
            echo "1. Check systemd service (if created):"
            echo "   sudo systemctl status algo-trading-engine"
            echo ""
            echo "2. Check cron jobs:"
            echo "   crontab -l"
            echo ""
            echo "3. Monitor health:"
            echo "   ./scripts/health_check.sh"
            ;;
        "docker")
            echo "1. Build and run Docker container:"
            echo "   docker-compose up -d"
            echo ""
            echo "2. Check logs:"
            echo "   docker-compose logs -f"
            echo ""
            echo "3. Stop container:"
            echo "   docker-compose down"
            ;;
        "cloud")
            echo "1. Review cloud deployment package"
            echo "2. Follow cloud provider's deployment instructions"
            echo "3. Deploy to AWS/GCP/Azure"
            ;;
    esac

    echo ""
    echo -e "${YELLOW}Important:${NC}"
    echo "- Ensure config/config.json has correct API credentials"
    echo "- Verify trading mode (live/paper) before running"
    echo "- Always test with paper trading first"
    echo "- Monitor logs regularly: tail -f ${LOG_DIR}/*.log"
    echo ""
    success "Setup completed successfully!"
}

# Entry point
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
