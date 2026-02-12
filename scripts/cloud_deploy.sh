#!/bin/bash
# cloud_deploy.sh — Prepare code for cloud deployment
#
# Features:
#   - Prepares code for cloud (removes local paths, sensitive data)
#   - Creates deployment package
#   - Generates deployment instructions for AWS/GCP/Azure
#   - Creates infrastructure-as-code templates
#
# Usage:
#   ./scripts/cloud_deploy.sh
#   ./scripts/cloud_deploy.sh --package  # Create deployment package
#   ./scripts/cloud_deploy.sh --aws      # Generate AWS deployment guide
#   ./scripts/cloud_deploy.sh --gcp      # Generate GCP deployment guide
#   ./scripts/cloud_deploy.sh --azure    # Generate Azure deployment guide

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

CLOUD_LOG="${LOG_DIR}/cloud_deploy_$(date +%Y%m%d_%H%M%S).log"
DEPLOY_DIR="${PROJECT_ROOT}/cloud-deployment-$(date +%Y%m%d_%H%M%S)"

mkdir -p "${LOG_DIR}"
mkdir -p "${DEPLOY_DIR}"

# Flags
ACTION="prepare"

log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[${timestamp}] [${level}] ${message}" | tee -a "${CLOUD_LOG}"
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
        --package)
            ACTION="package"
            shift
            ;;
        --aws)
            ACTION="aws"
            shift
            ;;
        --gcp)
            ACTION="gcp"
            shift
            ;;
        --azure)
            ACTION="azure"
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
    echo "║   Algo Trading Agent - Cloud Deployment Preparation       ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Prepare code for cloud
prepare_code() {
    info "Preparing code for cloud deployment..."

    mkdir -p "${DEPLOY_DIR}/src"

    # Copy source code (exclude sensitive files)
    info "Copying source files..."
    cp -r "${PROJECT_ROOT}/cmd" "${DEPLOY_DIR}/src/"
    cp -r "${PROJECT_ROOT}/internal" "${DEPLOY_DIR}/src/"
    cp -r "${PROJECT_ROOT}/scripts" "${DEPLOY_DIR}/src/"
    cp -r "${PROJECT_ROOT}/db" "${DEPLOY_DIR}/src/"
    cp -r "${PROJECT_ROOT}/python_ai" "${DEPLOY_DIR}/src/"
    cp "${PROJECT_ROOT}/go.mod" "${DEPLOY_DIR}/src/"
    cp "${PROJECT_ROOT}/go.sum" "${DEPLOY_DIR}/src/"
    cp "${PROJECT_ROOT}/Dockerfile" "${DEPLOY_DIR}/src/" 2>/dev/null || true
    cp "${PROJECT_ROOT}/docker-compose.yml" "${DEPLOY_DIR}/src/" 2>/dev/null || true

    success "Source code copied"

    # Create cloud-ready config template
    info "Creating configuration templates..."
    mkdir -p "${DEPLOY_DIR}/config-templates"

    cat > "${DEPLOY_DIR}/config-templates/config.template.json" << 'EOF'
{
  "active_broker": "dhan",
  "trading_mode": "paper",
  "capital": 500000.00,
  "risk": {
    "max_risk_per_trade_pct": 1.0,
    "max_open_positions": 5,
    "max_daily_loss_pct": 3.0,
    "max_capital_deployment_pct": 70.0
  },
  "paths": {
    "ai_output_dir": "/app/ai_outputs",
    "market_data_dir": "/app/market_data",
    "log_dir": "/app/logs"
  },
  "broker_config": {
    "dhan": {
      "client_id": "${DHAN_CLIENT_ID}",
      "access_token": "${DHAN_ACCESS_TOKEN}",
      "base_url": "https://api.dhan.co",
      "instrument_file": "/app/config/dhan_instruments.json"
    }
  },
  "database_url": "${DATABASE_URL}",
  "market_calendar_path": "/app/config/holidays_2026.json",
  "polling_interval_minutes": 5,
  "webhook": {
    "enabled": false,
    "port": 8080,
    "path": "/webhook/dhan/order"
  }
}
EOF

    success "Configuration template created"
}

# Create deployment package
create_deployment_package() {
    info "Creating deployment package..."

    # Create README for cloud deployment
    cat > "${DEPLOY_DIR}/CLOUD_DEPLOYMENT.md" << 'EOF'
# Cloud Deployment Guide

This package contains the Algo Trading Engine code prepared for cloud deployment.

## Prerequisites

- Docker installed locally (for building the image)
- Cloud provider account (AWS/GCP/Azure)
- Docker registry access (Docker Hub, ECR, GCR, or ACR)
- PostgreSQL database (managed service or self-hosted)

## Step 1: Build Docker Image

```bash
cd src
docker build -t algo-trading-engine:latest .
```

## Step 2: Push to Registry

### For Docker Hub:
```bash
docker tag algo-trading-engine:latest yourusername/algo-trading-engine:latest
docker push yourusername/algo-trading-engine:latest
```

### For AWS ECR:
```bash
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <your-ecr-uri>
docker tag algo-trading-engine:latest <your-ecr-uri>/algo-trading-engine:latest
docker push <your-ecr-uri>/algo-trading-engine:latest
```

### For GCP GCR:
```bash
gcloud auth configure-docker
docker tag algo-trading-engine:latest gcr.io/your-project/algo-trading-engine:latest
docker push gcr.io/your-project/algo-trading-engine:latest
```

## Step 3: Configure Environment

Update `config-templates/config.template.json` with:
- `DHAN_CLIENT_ID`: Your Dhan broker client ID
- `DHAN_ACCESS_TOKEN`: Your Dhan broker API token
- `DATABASE_URL`: Your cloud database connection string

## Step 4: Deploy to Cloud

Follow the provider-specific guides in this directory:
- `AWS_DEPLOYMENT.md` - AWS deployment using ECS/Fargate
- `GCP_DEPLOYMENT.md` - Google Cloud deployment using Cloud Run/GKE
- `AZURE_DEPLOYMENT.md` - Azure deployment using ACI/AKS

## Configuration Management

Use your cloud provider's secrets management:
- **AWS**: AWS Secrets Manager or Parameter Store
- **GCP**: Secret Manager
- **Azure**: Key Vault

## Monitoring

Each guide includes monitoring setup for:
- Application logs
- Container health
- Database connectivity
- Trading metrics

## Support

For issues or questions:
1. Check the logs in your cloud provider's console
2. Review the troubleshooting section in the provider guide
3. Ensure all environment variables are correctly set
EOF

    success "Cloud deployment guide created"
}

# Create AWS deployment guide
create_aws_guide() {
    info "Creating AWS deployment guide..."

    cat > "${DEPLOY_DIR}/AWS_DEPLOYMENT.md" << 'EOF'
# AWS Deployment Guide

This guide covers deploying the Algo Trading Engine on AWS.

## Option 1: ECS Fargate (Recommended)

### Prerequisites
- AWS account with appropriate permissions
- AWS CLI installed and configured
- Docker image pushed to ECR

### Step 1: Create ECR Repository

```bash
aws ecr create-repository \
    --repository-name algo-trading-engine \
    --region us-east-1
```

### Step 2: Create CloudFormation Stack

```bash
aws cloudformation create-stack \
    --stack-name algo-trading-engine \
    --template-body file://aws-ecs-template.yml \
    --parameters \
        ParameterKey=DockerImage,ParameterValue=<your-ecr-uri>/algo-trading-engine:latest \
        ParameterKey=DatabaseURL,ParameterValue=<your-rds-url> \
        ParameterKey=DhanClientId,ParameterValue=<your-client-id> \
        ParameterKey=DhanAccessToken,ParameterValue=<your-token> \
    --capabilities CAPABILITY_IAM
```

### Step 3: Configure RDS

```bash
aws rds create-db-instance \
    --db-instance-identifier algo-trading-db \
    --db-instance-class db.t3.micro \
    --engine postgres \
    --master-username algo \
    --master-user-password <secure-password> \
    --allocated-storage 20 \
    --storage-type gp2
```

### Step 4: Set Secrets in Secrets Manager

```bash
aws secretsmanager create-secret \
    --name trading-engine/config \
    --secret-string '{"dhan_client_id":"...","dhan_token":"..."}'
```

## Option 2: Lambda + EventBridge (Serverless)

### Step 1: Package Lambda Function

```bash
zip -r lambda-package.zip src/cmd/engine
```

### Step 2: Create Lambda Function

```bash
aws lambda create-function \
    --function-name algo-trading-engine \
    --zip-file fileb://lambda-package.zip \
    --handler main \
    --runtime go1.x \
    --timeout 300
```

### Step 3: Create EventBridge Rules

```bash
# Pre-market fetch (8:50 AM)
aws events put-rule \
    --name algo-trading-premarket \
    --schedule-expression "cron(50 8 ? * MON-FRI *)"
```

## Monitoring and Logs

### CloudWatch Logs

```bash
aws logs create-log-group --log-group-name /aws/ecs/algo-trading-engine

aws logs create-log-stream \
    --log-group-name /aws/ecs/algo-trading-engine \
    --log-stream-name trading-engine
```

### CloudWatch Alarms

```bash
aws cloudwatch put-metric-alarm \
    --alarm-name trading-engine-unhealthy \
    --alarm-description "Alert when engine is unhealthy" \
    --metric-name TaskCount \
    --namespace AWS/ECS \
    --statistic Average \
    --period 300 \
    --threshold 0 \
    --comparison-operator LessThanOrEqualToThreshold
```

## Cost Optimization

- Use Fargate Spot Instances for non-critical runs
- Scale down during non-market hours
- Use RDS Multi-AZ only for production
- Set CloudWatch log retention to 7 days

## Security

- Enable VPC for isolation
- Use Security Groups to restrict access
- Store secrets in Secrets Manager
- Enable CloudTrail for audit logging
- Use IAM roles with minimal permissions

EOF

    success "AWS deployment guide created"
}

# Create GCP deployment guide
create_gcp_guide() {
    info "Creating GCP deployment guide..."

    cat > "${DEPLOY_DIR}/GCP_DEPLOYMENT.md" << 'EOF'
# Google Cloud Platform (GCP) Deployment Guide

This guide covers deploying the Algo Trading Engine on GCP.

## Option 1: Cloud Run (Recommended)

### Prerequisites
- GCP account with appropriate permissions
- gcloud CLI installed and configured
- Docker image pushed to GCR

### Step 1: Create GCR Repository

```bash
gcloud container images list --repository=gcr.io/your-project
```

### Step 2: Build and Push Image

```bash
gcloud builds submit --tag gcr.io/your-project/algo-trading-engine:latest
```

### Step 3: Deploy to Cloud Run

```bash
gcloud run deploy algo-trading-engine \
    --image gcr.io/your-project/algo-trading-engine:latest \
    --platform managed \
    --region us-central1 \
    --memory 512Mi \
    --cpu 1 \
    --timeout 3600 \
    --set-env-vars DATABASE_URL=<your-cloudsql-connection> \
    --set-env-vars DHAN_CLIENT_ID=<your-client-id> \
    --set-env-vars DHAN_ACCESS_TOKEN=<your-token>
```

### Step 4: Set Up Cloud SQL

```bash
gcloud sql instances create algo-trading-db \
    --database-version=POSTGRES_13 \
    --tier=db-f1-micro \
    --region=us-central1
```

## Option 2: GKE (Kubernetes)

### Step 1: Create Cluster

```bash
gcloud container clusters create algo-trading-cluster \
    --zone us-central1-a \
    --num-nodes 3 \
    --machine-type n1-standard-1
```

### Step 2: Create Deployment

```bash
kubectl create deployment algo-trading-engine \
    --image=gcr.io/your-project/algo-trading-engine:latest
```

### Step 3: Expose Service

```bash
kubectl expose deployment algo-trading-engine \
    --type=LoadBalancer \
    --port=8080
```

## Option 3: Compute Engine (VMs)

### Step 1: Create VM Instance

```bash
gcloud compute instances create algo-trading-vm \
    --image-family=ubuntu-2004-lts \
    --image-project=ubuntu-os-cloud \
    --machine-type=e2-medium \
    --zone=us-central1-a
```

### Step 2: SSH into Instance

```bash
gcloud compute ssh algo-trading-vm --zone=us-central1-a
```

### Step 3: Install and Run

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Pull and run container
docker pull gcr.io/your-project/algo-trading-engine:latest
docker run -d \
    -e DATABASE_URL=<your-url> \
    -e DHAN_CLIENT_ID=<your-id> \
    -e DHAN_ACCESS_TOKEN=<your-token> \
    gcr.io/your-project/algo-trading-engine:latest
```

## Monitoring

### Cloud Logging

```bash
gcloud logging create-sink algo-trading-logs \
    logging.googleapis.com/projects/your-project/logs/algo-trading
```

### Cloud Monitoring

```bash
gcloud monitoring dashboards create --config-from-file dashboard.json
```

## Cost Optimization

- Use Cloud Run for periodic jobs (most cost-effective)
- Use Autopilot for GKE to optimize resource usage
- Set up preemptible nodes for non-critical workloads
- Use Committed Use Discounts for long-running services

EOF

    success "GCP deployment guide created"
}

# Create Azure deployment guide
create_azure_guide() {
    info "Creating Azure deployment guide..."

    cat > "${DEPLOY_DIR}/AZURE_DEPLOYMENT.md" << 'EOF'
# Azure Deployment Guide

This guide covers deploying the Algo Trading Engine on Microsoft Azure.

## Option 1: Container Instances (Recommended)

### Prerequisites
- Azure account with appropriate permissions
- Azure CLI installed and configured
- Docker image pushed to ACR

### Step 1: Create Container Registry

```bash
az acr create \
    --resource-group algo-trading \
    --name algotrading \
    --sku Basic
```

### Step 2: Build and Push Image

```bash
az acr build \
    --registry algotrading \
    --image algo-trading-engine:latest \
    --file Dockerfile .
```

### Step 3: Deploy Container Instance

```bash
az container create \
    --resource-group algo-trading \
    --name algo-trading-engine \
    --image algotrading.azurecr.io/algo-trading-engine:latest \
    --cpu 1 \
    --memory 1.5 \
    --environment-variables \
        DATABASE_URL="<your-connection-string>" \
        DHAN_CLIENT_ID="<your-client-id>" \
        DHAN_ACCESS_TOKEN="<your-token>"
```

### Step 4: Create PostgreSQL Database

```bash
az postgres server create \
    --resource-group algo-trading \
    --name algo-trading-db \
    --location eastus \
    --admin-user algo \
    --admin-password <secure-password> \
    --sku-name B_Gen5_1
```

## Option 2: App Service

### Step 1: Create App Service Plan

```bash
az appservice plan create \
    --name algo-trading-plan \
    --resource-group algo-trading \
    --sku B1 \
    --is-linux
```

### Step 2: Create Web App

```bash
az webapp create \
    --resource-group algo-trading \
    --plan algo-trading-plan \
    --name algo-trading-app \
    --deployment-container-image-name algotrading.azurecr.io/algo-trading-engine:latest
```

## Option 3: Kubernetes Service (AKS)

### Step 1: Create AKS Cluster

```bash
az aks create \
    --resource-group algo-trading \
    --name algo-trading-cluster \
    --node-count 3 \
    --vm-set-type VirtualMachineScaleSets \
    --load-balancer-sku standard
```

### Step 2: Get Credentials

```bash
az aks get-credentials \
    --resource-group algo-trading \
    --name algo-trading-cluster
```

### Step 3: Deploy to AKS

```bash
kubectl create deployment algo-trading-engine \
    --image=algotrading.azurecr.io/algo-trading-engine:latest

kubectl expose deployment algo-trading-engine \
    --port=8080 \
    --type=LoadBalancer
```

## Monitoring with Application Insights

```bash
az monitor app-insights component create \
    --app algo-trading-insights \
    --location eastus \
    --resource-group algo-trading \
    --application-type web
```

## Security

- Use Azure Key Vault for secrets management
- Enable managed identity for container authentication
- Use Virtual Networks to isolate resources
- Enable Azure Monitor for comprehensive logging

## Cost Optimization

- Use Container Instances for one-off jobs
- Scale down during non-market hours
- Use Spot VMs for AKS nodes
- Set up auto-scaling based on metrics

EOF

    success "Azure deployment guide created"
}

# Create infrastructure-as-code templates
create_iac_templates() {
    info "Creating infrastructure-as-code templates..."

    mkdir -p "${DEPLOY_DIR}/iac"

    # Terraform template for AWS
    cat > "${DEPLOY_DIR}/iac/aws-main.tf" << 'EOF'
terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# ECR Repository
resource "aws_ecr_repository" "trading_engine" {
  name                 = "algo-trading-engine"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}

# RDS PostgreSQL Database
resource "aws_db_instance" "trading_db" {
  identifier          = "algo-trading-db"
  engine              = "postgres"
  engine_version      = "13.7"
  instance_class      = var.db_instance_class
  allocated_storage   = 20
  storage_type        = "gp2"
  username            = "algo"
  password            = var.db_password
  skip_final_snapshot = true
}

# ECS Cluster
resource "aws_ecs_cluster" "trading_cluster" {
  name = "algo-trading-cluster"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}

# CloudWatch Log Group
resource "aws_cloudwatch_log_group" "trading_logs" {
  name              = "/aws/ecs/algo-trading-engine"
  retention_in_days = 7
}

output "ecr_repository_url" {
  value = aws_ecr_repository.trading_engine.repository_url
}

output "rds_endpoint" {
  value = aws_db_instance.trading_db.endpoint
}
EOF

    success "Created Terraform template for AWS"

    # Variables
    cat > "${DEPLOY_DIR}/iac/variables.tf" << 'EOF'
variable "aws_region" {
  description = "AWS region"
  default     = "us-east-1"
}

variable "db_instance_class" {
  description = "RDS instance class"
  default     = "db.t3.micro"
}

variable "db_password" {
  description = "RDS password"
  sensitive   = true
}
EOF

    success "Created variables file"
}

# Print summary
print_summary() {
    echo ""
    echo -e "${GREEN}"
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║        CLOUD DEPLOYMENT PREPARATION COMPLETED              ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    echo ""
    echo "Deployment package location: ${DEPLOY_DIR}"
    echo ""
    echo "Contents:"
    echo "  src/                         - Source code ready for deployment"
    echo "  config-templates/            - Configuration templates"
    echo "  iac/                         - Infrastructure-as-code templates"
    echo "  CLOUD_DEPLOYMENT.md          - General cloud deployment guide"

    case "${ACTION}" in
        aws)
            echo "  AWS_DEPLOYMENT.md            - AWS-specific deployment guide"
            ;;
        gcp)
            echo "  GCP_DEPLOYMENT.md            - GCP-specific deployment guide"
            ;;
        azure)
            echo "  AZURE_DEPLOYMENT.md          - Azure-specific deployment guide"
            ;;
        *)
            echo "  AWS_DEPLOYMENT.md            - AWS-specific deployment guide"
            echo "  GCP_DEPLOYMENT.md            - GCP-specific deployment guide"
            echo "  AZURE_DEPLOYMENT.md          - Azure-specific deployment guide"
            ;;
    esac

    echo ""
    echo "Next steps:"
    echo "1. Review the deployment guides in ${DEPLOY_DIR}"
    echo "2. Update configuration templates with your API credentials"
    echo "3. Build and push Docker image to your cloud registry"
    echo "4. Deploy using the provider-specific instructions"
    echo ""
    echo "Setup log: ${CLOUD_LOG}"
    echo ""
}

# Main flow
main() {
    print_header

    log "INFO" "Cloud deployment preparation started with action: ${ACTION}"

    # Prepare code
    prepare_code

    echo ""

    # Create deployment package
    create_deployment_package

    echo ""

    # Create provider-specific guides
    case "${ACTION}" in
        aws|prepare)
            create_aws_guide
            create_gcp_guide
            create_azure_guide
            ;;
        gcp)
            create_gcp_guide
            ;;
        azure)
            create_azure_guide
            ;;
    esac

    echo ""

    # Create IaC templates
    create_iac_templates

    print_summary

    log "INFO" "Cloud deployment preparation completed successfully"
}

# Entry point
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
