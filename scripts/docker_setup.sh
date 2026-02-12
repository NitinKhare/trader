#!/bin/bash
# docker_setup.sh — Create Docker deployment configuration
#
# Creates:
#   - Dockerfile for building the trading engine image
#   - docker-compose.yml for orchestration
#   - Volume mounts for config and logs
#   - Environment configuration
#
# Usage:
#   ./scripts/docker_setup.sh
#   ./scripts/docker_setup.sh --build   # Build image after setup

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

DOCKER_LOG="${LOG_DIR}/docker_setup_$(date +%Y%m%d_%H%M%S).log"

mkdir -p "${LOG_DIR}"

# Flags
BUILD_IMAGE=false

log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[${timestamp}] [${level}] ${message}" | tee -a "${DOCKER_LOG}"
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
        --build)
            BUILD_IMAGE=true
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
    echo "║     Algo Trading Agent - Docker Setup                      ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Check Docker
check_docker() {
    info "Checking Docker installation..."

    if ! command -v docker &> /dev/null; then
        error_exit "Docker is not installed"
    fi

    if ! command -v docker-compose &> /dev/null; then
        warn "docker-compose not found, will use 'docker compose' (docker plugin)"
    fi

    success "Docker is installed"
}

# Create Dockerfile
create_dockerfile() {
    info "Creating Dockerfile..."

    local dockerfile="${PROJECT_ROOT}/Dockerfile"

    if [ -f "${dockerfile}" ]; then
        warn "Dockerfile already exists, skipping"
        return
    fi

    cat > "${dockerfile}" << 'EOF'
# Multi-stage build for Algo Trading Engine

# Build stage
FROM golang:1.24.1-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make bash postgresql-client

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd ./cmd
COPY internal ./internal

# Build the engine
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o engine ./cmd/engine

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    postgresql-client \
    bash \
    curl \
    tzdata \
    python3 \
    py3-pip \
    && rm -rf /var/cache/apk/*

# Install Python dependencies
RUN pip install --no-cache-dir \
    pandas \
    numpy \
    scikit-learn \
    requests \
    python-dotenv

# Set working directory
WORKDIR /app

# Copy engine binary from builder
COPY --from=builder /build/engine .

# Copy scripts
COPY scripts ./scripts
COPY config ./config
COPY python_ai ./python_ai

# Create necessary directories
RUN mkdir -p logs ai_outputs market_data

# Set executable permissions
RUN chmod +x ./engine && chmod +x ./scripts/*.sh

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Environment variables
ENV LOG_DIR=/app/logs
ENV AI_OUTPUT_DIR=/app/ai_outputs
ENV MARKET_DATA_DIR=/app/market_data

# Expose port for webhook
EXPOSE 8080

# Default command
CMD ["./engine", "--config", "config/config.json", "--mode", "market"]
EOF

    success "Created Dockerfile"
}

# Create docker-compose.yml
create_docker_compose() {
    info "Creating docker-compose.yml..."

    local compose_file="${PROJECT_ROOT}/docker-compose.yml"

    if [ -f "${compose_file}" ]; then
        warn "docker-compose.yml already exists, skipping"
        return
    fi

    cat > "${compose_file}" << 'EOF'
version: '3.8'

services:
  # PostgreSQL database
  postgres:
    image: postgres:15-alpine
    container_name: algo-trading-db
    environment:
      POSTGRES_DB: algo_trading
      POSTGRES_USER: algo
      POSTGRES_PASSWORD: algo123
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./db/migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U algo"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - trading-network

  # Trading engine
  trading-engine:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: algo-trading-engine
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://algo:algo123@postgres:5432/algo_trading?sslmode=disable
      LOG_DIR: /app/logs
      AI_OUTPUT_DIR: /app/ai_outputs
      MARKET_DATA_DIR: /app/market_data
    volumes:
      # Config mount
      - ./config:/app/config:ro
      # Logs mount
      - ./logs:/app/logs
      # AI outputs mount
      - ./ai_outputs:/app/ai_outputs
      # Market data mount
      - ./market_data:/app/market_data
    ports:
      - "8080:8080"
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "bash", "-c", "ps aux | grep engine | grep -v grep"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks:
      - trading-network

  # Optional: Prometheus for monitoring (if you add monitoring)
  # prometheus:
  #   image: prom/prometheus:latest
  #   container_name: algo-trading-prometheus
  #   volumes:
  #     - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
  #     - prometheus_data:/prometheus
  #   ports:
  #     - "9090:9090"
  #   networks:
  #     - trading-network

  # Optional: Grafana for dashboards (if you add monitoring)
  # grafana:
  #   image: grafana/grafana:latest
  #   container_name: algo-trading-grafana
  #   environment:
  #     GF_SECURITY_ADMIN_PASSWORD: admin
  #   ports:
  #     - "3000:3000"
  #   volumes:
  #     - grafana_data:/var/lib/grafana
  #   networks:
  #     - trading-network

volumes:
  postgres_data:
  # prometheus_data:
  # grafana_data:

networks:
  trading-network:
    driver: bridge
EOF

    success "Created docker-compose.yml"
}

# Create .dockerignore
create_dockerignore() {
    info "Creating .dockerignore..."

    local dockerignore="${PROJECT_ROOT}/.dockerignore"

    if [ -f "${dockerignore}" ]; then
        warn ".dockerignore already exists, skipping"
        return
    fi

    cat > "${dockerignore}" << 'EOF'
.git
.gitignore
.claude
.env
.env.local
*.log
node_modules
vendor
.DS_Store
*.swp
*.swo
*.bak
.vscode
.idea
build/
dist/
*.exe
*.dylib
.gradle
.pytest_cache
__pycache__
*.pyc
.python-version
venv/
env/
EOF

    success "Created .dockerignore"
}

# Create docker.env file
create_docker_env() {
    info "Creating docker.env..."

    local docker_env="${PROJECT_ROOT}/docker.env"

    if [ -f "${docker_env}" ]; then
        warn "docker.env already exists, skipping"
        return
    fi

    cat > "${docker_env}" << 'EOF'
# Docker environment variables for trading engine

# Database configuration
POSTGRES_DB=algo_trading
POSTGRES_USER=algo
POSTGRES_PASSWORD=algo123
DATABASE_URL=postgres://algo:algo123@postgres:5432/algo_trading?sslmode=disable

# Engine configuration
LOG_LEVEL=INFO
POLLING_INTERVAL_MINUTES=5

# Volume mounts
LOG_DIR=/app/logs
AI_OUTPUT_DIR=/app/ai_outputs
MARKET_DATA_DIR=/app/market_data
EOF

    success "Created docker.env"
}

# Create docker build/run scripts
create_docker_scripts() {
    info "Creating Docker helper scripts..."

    # Docker build script
    local build_script="${SCRIPT_DIR}/docker_build.sh"

    cat > "${build_script}" << 'EOF'
#!/bin/bash
# docker_build.sh — Build Docker image

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Building Docker image..."
cd "${PROJECT_ROOT}"

if docker build -t algo-trading-engine:latest -f Dockerfile .; then
    echo "✓ Docker image built successfully"
    docker images | grep algo-trading-engine
else
    echo "✗ Failed to build Docker image"
    exit 1
fi
EOF

    chmod +x "${build_script}"
    success "Created docker_build.sh"

    # Docker run script
    local run_script="${SCRIPT_DIR}/docker_run.sh"

    cat > "${run_script}" << 'EOF'
#!/bin/bash
# docker_run.sh — Run trading engine in Docker

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Starting Docker containers..."
cd "${PROJECT_ROOT}"

docker-compose up -d

echo "✓ Containers started"
echo ""
echo "View logs:"
echo "  docker-compose logs -f"
echo ""
echo "Stop containers:"
echo "  docker-compose down"
EOF

    chmod +x "${run_script}"
    success "Created docker_run.sh"
}

# Build Docker image
build_docker_image() {
    if [ "${BUILD_IMAGE}" != true ]; then
        return
    fi

    info "Building Docker image..."

    cd "${PROJECT_ROOT}"

    if ! docker build -t algo-trading-engine:latest -f Dockerfile .; then
        error_exit "Failed to build Docker image"
    fi

    success "Docker image built successfully"
    docker images | grep algo-trading-engine || true
}

# Print summary
print_summary() {
    echo ""
    echo -e "${GREEN}"
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║              DOCKER SETUP COMPLETED                        ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    echo ""
    echo "Created files:"
    echo "  - ${PROJECT_ROOT}/Dockerfile"
    echo "  - ${PROJECT_ROOT}/docker-compose.yml"
    echo "  - ${PROJECT_ROOT}/.dockerignore"
    echo "  - ${PROJECT_ROOT}/docker.env"
    echo "  - ${SCRIPT_DIR}/docker_build.sh"
    echo "  - ${SCRIPT_DIR}/docker_run.sh"
    echo ""

    echo "Next steps:"
    echo "1. Build the Docker image:"
    echo "   cd ${PROJECT_ROOT}"
    echo "   docker build -t algo-trading-engine:latest ."
    echo ""
    echo "2. Start containers:"
    echo "   docker-compose up -d"
    echo ""
    echo "3. Monitor logs:"
    echo "   docker-compose logs -f trading-engine"
    echo ""
    echo "4. Stop containers:"
    echo "   docker-compose down"
    echo ""
    echo "5. Check container status:"
    echo "   docker-compose ps"
    echo ""
    echo "Setup log: ${DOCKER_LOG}"
    echo ""
}

# Main flow
main() {
    print_header

    log "INFO" "Docker setup started"

    # Check Docker
    check_docker

    echo ""

    # Create files
    create_dockerfile
    create_docker_compose
    create_dockerignore
    create_docker_env
    create_docker_scripts

    echo ""

    # Build if requested
    build_docker_image

    print_summary

    log "INFO" "Docker setup completed successfully"
}

# Entry point
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
