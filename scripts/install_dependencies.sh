#!/bin/bash
# install_dependencies.sh — Install all required dependencies
#
# Installs Go, Python3, PostgreSQL client, and Docker (optional)
# Detects OS and uses appropriate package managers
#
# Usage:
#   ./scripts/install_dependencies.sh
#
# Supported OS: macOS, Linux (Ubuntu/Debian/CentOS/RHEL)

set -euo pipefail

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOG_DIR="${PROJECT_ROOT}/logs"
INSTALL_LOG="${LOG_DIR}/install_$(date +%Y%m%d_%H%M%S).log"

mkdir -p "${LOG_DIR}"

log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[${timestamp}] [${level}] ${message}" | tee -a "${INSTALL_LOG}"
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
    log "INFO" "$1"
}

warn() {
    echo -e "${YELLOW}⚠ $1${NC}"
    log "WARN" "$1"
}

error() {
    echo -e "${RED}✗ $1${NC}"
    log "ERROR" "$1"
}

error_exit() {
    error "$1"
    exit 1
}

info() {
    echo -e "${BLUE}ℹ $1${NC}"
    log "INFO" "$1"
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)
            if [ -f /etc/os-release ]; then
                . /etc/os-release
                echo "${ID}"
            else
                echo "unknown"
            fi
            ;;
        Darwin*)
            echo "macos"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Print header
print_header() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║      Algo Trading Agent - Dependency Installation          ║"
    echo "║                                                            ║"
    echo "║  This script will install:                                 ║"
    echo "║  - Go (1.20+)                                              ║"
    echo "║  - Python3 (3.8+)                                          ║"
    echo "║  - PostgreSQL client                                       ║"
    echo "║  - Docker (optional)                                       ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Install Go
install_go() {
    info "Installing Go..."

    if command_exists go; then
        local go_version=$(go version | awk '{print $3}')
        success "Go ${go_version} is already installed"
        return 0
    fi

    local OS=$(uname -s)
    local ARCH=$(uname -m)

    # Convert architecture names
    case "${ARCH}" in
        x86_64)  ARCH="amd64" ;;
        arm64)   ARCH="arm64" ;;
        *)       error_exit "Unsupported architecture: ${ARCH}" ;;
    esac

    # Download Go
    local go_version="1.24.1"
    local go_url="https://go.dev/dl/go${go_version}.${OS,,}-${ARCH}.tar.gz"
    local go_tar="/tmp/go.tar.gz"

    info "Downloading Go ${go_version}..."
    if curl -L -o "${go_tar}" "${go_url}" 2>/dev/null; then
        info "Extracting Go..."
        sudo tar -C /usr/local -xzf "${go_tar}"
        rm "${go_tar}"

        # Add to PATH
        export PATH="${PATH}:/usr/local/go/bin"
        echo 'export PATH="${PATH}:/usr/local/go/bin"' >> ~/.bashrc 2>/dev/null || true
        echo 'export PATH="${PATH}:/usr/local/go/bin"' >> ~/.zshrc 2>/dev/null || true

        success "Go installed successfully"
    else
        error_exit "Failed to download Go"
    fi
}

# Install Python3
install_python3() {
    info "Installing Python3..."

    if command_exists python3; then
        local python_version=$(python3 --version | awk '{print $2}')
        success "Python3 ${python_version} is already installed"
        return 0
    fi

    local OS=$(detect_os)

    case "${OS}" in
        ubuntu|debian)
            info "Installing Python3 on Debian/Ubuntu..."
            sudo apt-get update
            sudo apt-get install -y python3 python3-pip python3-venv
            success "Python3 installed"
            ;;
        rhel|centos|fedora)
            info "Installing Python3 on RHEL/CentOS/Fedora..."
            sudo yum install -y python3 python3-pip
            success "Python3 installed"
            ;;
        macos)
            info "Installing Python3 on macOS..."
            if command_exists brew; then
                brew install python3
            else
                error_exit "Homebrew not found. Install Homebrew first: https://brew.sh"
            fi
            success "Python3 installed"
            ;;
        *)
            error_exit "Unsupported OS for automatic Python3 installation: ${OS}"
            ;;
    esac
}

# Install PostgreSQL client
install_postgresql_client() {
    info "Installing PostgreSQL client..."

    if command_exists psql; then
        local pg_version=$(psql --version | awk '{print $3}')
        success "PostgreSQL client ${pg_version} is already installed"
        return 0
    fi

    local OS=$(detect_os)

    case "${OS}" in
        ubuntu|debian)
            info "Installing PostgreSQL client on Debian/Ubuntu..."
            sudo apt-get update
            sudo apt-get install -y postgresql-client
            success "PostgreSQL client installed"
            ;;
        rhel|centos|fedora)
            info "Installing PostgreSQL client on RHEL/CentOS/Fedora..."
            sudo yum install -y postgresql
            success "PostgreSQL client installed"
            ;;
        macos)
            info "Installing PostgreSQL client on macOS..."
            if command_exists brew; then
                brew install postgresql
            else
                error_exit "Homebrew not found. Install Homebrew first: https://brew.sh"
            fi
            success "PostgreSQL client installed"
            ;;
        *)
            error_exit "Unsupported OS for automatic PostgreSQL installation: ${OS}"
            ;;
    esac
}

# Install Docker
install_docker() {
    info "Installing Docker..."

    if command_exists docker; then
        local docker_version=$(docker --version | awk '{print $3}' | sed 's/,//')
        success "Docker ${docker_version} is already installed"
        return 0
    fi

    local OS=$(detect_os)

    case "${OS}" in
        ubuntu|debian)
            info "Installing Docker on Debian/Ubuntu..."
            sudo apt-get update
            sudo apt-get install -y docker.io docker-compose
            sudo usermod -aG docker "${USER}"
            success "Docker installed"
            warn "You may need to log out and back in for group changes to take effect"
            ;;
        rhel|centos|fedora)
            info "Installing Docker on RHEL/CentOS/Fedora..."
            sudo yum install -y docker docker-compose
            sudo systemctl start docker
            sudo systemctl enable docker
            sudo usermod -aG docker "${USER}"
            success "Docker installed"
            warn "You may need to log out and back in for group changes to take effect"
            ;;
        macos)
            info "Installing Docker on macOS..."
            if command_exists brew; then
                brew install docker docker-compose
            else
                error_exit "Homebrew not found. Install Homebrew first: https://brew.sh"
            fi
            success "Docker installed"
            ;;
        *)
            error_exit "Unsupported OS for automatic Docker installation: ${OS}"
            ;;
    esac
}

# Install Python dependencies
install_python_dependencies() {
    info "Installing Python dependencies..."

    if [ ! -f "${PROJECT_ROOT}/python_ai/requirements.txt" ]; then
        warn "requirements.txt not found at ${PROJECT_ROOT}/python_ai/"
        return 0
    fi

    if command_exists pip3; then
        info "Installing from requirements.txt..."
        pip3 install --upgrade pip
        pip3 install -r "${PROJECT_ROOT}/python_ai/requirements.txt"
        success "Python dependencies installed"
    else
        error "pip3 not found, skipping Python dependencies"
    fi
}

# Install Go dependencies
install_go_dependencies() {
    info "Installing Go dependencies..."

    if [ ! -f "${PROJECT_ROOT}/go.mod" ]; then
        warn "go.mod not found"
        return 0
    fi

    cd "${PROJECT_ROOT}"
    if command_exists go; then
        go mod download
        go mod tidy
        success "Go dependencies installed"
    else
        error "Go not found, cannot install Go dependencies"
    fi
}

# Build the engine
build_engine() {
    info "Building trading engine..."

    if [ ! -f "${PROJECT_ROOT}/cmd/engine/main.go" ]; then
        warn "Engine source not found"
        return 0
    fi

    cd "${PROJECT_ROOT}"
    if command_exists go; then
        info "Building engine binary..."
        if go build -o engine ./cmd/engine; then
            success "Engine built successfully"
            ls -lh "${PROJECT_ROOT}/engine"
        else
            error "Failed to build engine"
            return 1
        fi
    else
        error "Go not found, cannot build engine"
        return 1
    fi
}

# Validate installations
validate_installations() {
    info "Validating installations..."

    local missing=()

    if ! command_exists go; then
        missing+=("Go")
    else
        success "Go is installed: $(go version)"
    fi

    if ! command_exists python3; then
        missing+=("Python3")
    else
        success "Python3 is installed: $(python3 --version)"
    fi

    if ! command_exists psql; then
        missing+=("PostgreSQL client")
    else
        success "PostgreSQL client is installed"
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        error "Missing dependencies: ${missing[*]}"
        return 1
    fi

    success "All required dependencies are installed"
    return 0
}

# Main installation flow
main() {
    print_header

    log "INFO" "Dependency installation started"

    # Detect OS
    local OS=$(detect_os)
    info "Detected OS: ${OS}"

    if [ "${OS}" = "unknown" ]; then
        error_exit "Unsupported operating system"
    fi

    # Install dependencies
    install_go
    install_python3
    install_postgresql_client

    # Optional Docker installation
    echo ""
    read -p "$(echo -e ${BLUE}Install Docker?${NC}) (yes/no) [no]: " install_docker_choice
    if [[ "${install_docker_choice,,}" == "yes" || "${install_docker_choice,,}" == "y" ]]; then
        install_docker
    else
        info "Skipping Docker installation"
    fi

    # Install language dependencies
    echo ""
    install_python_dependencies
    install_go_dependencies

    # Build engine
    echo ""
    if validate_installations; then
        echo ""
        build_engine
    else
        warn "Some dependencies are still missing"
    fi

    # Print summary
    echo ""
    echo -e "${GREEN}"
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║              INSTALLATION COMPLETED                        ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    info "Installation log: ${INSTALL_LOG}"
    log "INFO" "Dependency installation completed successfully"
}

# Entry point
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
