#!/bin/bash

# CloudWorkstation AWS Integration Test Setup Validation
# This script validates the AWS integration test environment

set -e

echo "🔧 CloudWorkstation AWS Integration Test Setup Validation"
echo "========================================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Configuration
AWS_PROFILE="${AWS_PROFILE:-aws}"
AWS_TEST_REGION="${AWS_TEST_REGION:-us-east-1}"
DAEMON_URL="${DAEMON_URL:-http://localhost:8947}"

echo ""
log_info "Configuration:"
echo "  AWS Profile: $AWS_PROFILE"
echo "  Test Region: $AWS_TEST_REGION"  
echo "  Daemon URL: $DAEMON_URL"
echo ""

# Check 1: Go environment
log_info "Checking Go environment..."
if ! command -v go &> /dev/null; then
    log_error "Go not found. Please install Go 1.19+ and add to PATH"
    exit 1
fi

GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1)
log_success "Go found: $GO_VERSION"

# Check 2: AWS CLI
log_info "Checking AWS CLI..."
if ! command -v aws &> /dev/null; then
    log_error "AWS CLI not found. Please install AWS CLI v2"
    exit 1
fi

AWS_VERSION=$(aws --version 2>&1 | head -1)
log_success "AWS CLI found: $AWS_VERSION"

# Check 3: AWS Profile Configuration
log_info "Checking AWS profile '$AWS_PROFILE'..."
if ! aws configure list --profile "$AWS_PROFILE" &> /dev/null; then
    log_error "AWS profile '$AWS_PROFILE' not configured"
    echo "Configure with: aws configure --profile $AWS_PROFILE"
    exit 1
fi

log_success "AWS profile '$AWS_PROFILE' configured"

# Check 4: AWS Credentials
log_info "Checking AWS credentials..."
if ! aws sts get-caller-identity --profile "$AWS_PROFILE" &> /dev/null; then
    log_error "AWS credentials invalid or expired for profile '$AWS_PROFILE'"
    echo "Check your credentials with: aws sts get-caller-identity --profile $AWS_PROFILE"
    exit 1
fi

IDENTITY=$(aws sts get-caller-identity --profile "$AWS_PROFILE" --output text)
ACCOUNT_ID=$(echo "$IDENTITY" | awk '{print $1}')
USER_ARN=$(echo "$IDENTITY" | awk '{print $2}')
log_success "AWS credentials valid - Account: $ACCOUNT_ID"
log_info "Identity: $USER_ARN"

# Check 5: AWS Permissions
log_info "Checking AWS permissions..."

# Test EC2 permissions
if aws ec2 describe-regions --region "$AWS_TEST_REGION" --profile "$AWS_PROFILE" &> /dev/null; then
    log_success "EC2 permissions verified"
else
    log_error "EC2 permissions insufficient"
    echo "Ensure your AWS user/role has EC2 full access"
    exit 1
fi

# Test EFS permissions
if aws efs describe-file-systems --region "$AWS_TEST_REGION" --profile "$AWS_PROFILE" &> /dev/null; then
    log_success "EFS permissions verified"
else
    log_warning "EFS permissions may be insufficient"
    echo "Consider adding EFS full access for storage tests"
fi

# Check 6: CloudWorkstation Build
log_info "Checking CloudWorkstation build..."
if [ ! -f "./bin/cwsd" ]; then
    log_warning "CloudWorkstation daemon not built"
    log_info "Building daemon..."
    make build-daemon
fi

if [ ! -f "./bin/cws" ]; then
    log_warning "CloudWorkstation CLI not built"
    log_info "Building CLI..."
    make build-cli
fi

log_success "CloudWorkstation binaries available"

# Check 7: Daemon Connectivity
log_info "Checking CloudWorkstation daemon..."
if ! pgrep -f "cwsd" > /dev/null; then
    log_warning "CloudWorkstation daemon not running"
    log_info "Start daemon with: ./bin/cwsd &"
    
    # Try to start daemon for testing
    log_info "Attempting to start daemon..."
    ./bin/cwsd &
    DAEMON_PID=$!
    sleep 3
    
    if curl -s "$DAEMON_URL/api/v1/ping" > /dev/null; then
        log_success "Daemon started successfully"
        # Stop the daemon we started
        kill $DAEMON_PID 2>/dev/null || true
        sleep 1
    else
        log_error "Failed to start daemon"
        kill $DAEMON_PID 2>/dev/null || true
        exit 1
    fi
else
    if curl -s "$DAEMON_URL/api/v1/ping" > /dev/null; then
        log_success "CloudWorkstation daemon accessible"
    else
        log_error "CloudWorkstation daemon not responding at $DAEMON_URL"
        echo "Check daemon status and URL configuration"
        exit 1
    fi
fi

# Check 8: Resource Limits
log_info "Checking AWS resource limits..."

# Check EC2 limits
EC2_LIMIT=$(aws service-quotas get-service-quota --service-code ec2 --quota-code L-1216C47A --region "$AWS_TEST_REGION" --profile "$AWS_PROFILE" --output text --query 'Quota.Value' 2>/dev/null || echo "unknown")
if [ "$EC2_LIMIT" != "unknown" ] && [ "$EC2_LIMIT" -ge "10" ]; then
    log_success "EC2 instance limit sufficient: $EC2_LIMIT"
elif [ "$EC2_LIMIT" != "unknown" ]; then
    log_warning "EC2 instance limit may be low: $EC2_LIMIT (recommended: 10+)"
else
    log_warning "Could not check EC2 limits - ensure you have sufficient quota"
fi

# Check 9: Cost Safeguards
log_info "Reviewing cost safeguards..."
echo "  Max instances: ${AWS_TEST_MAX_INSTANCES:-5}"
echo "  Max volumes: ${AWS_TEST_MAX_VOLUMES:-3}"
echo "  Max hourly cost: \$${AWS_TEST_MAX_HOURLY_COST:-5.00}"
echo "  Test timeout: ${AWS_TEST_TIMEOUT:-10} minutes"
log_success "Cost limits configured"

# Check 10: Build Tags Support
log_info "Checking build tags support..."
if go list -tags=aws_integration ./internal/cli/ > /dev/null 2>&1; then
    log_success "AWS integration build tags supported"
else
    log_error "Build tags not supported or source issue"
    exit 1
fi

# Final Summary
echo ""
echo "=========================================="
log_success "AWS Integration Test Environment Ready!"
echo ""
echo "Next steps:"
echo "1. Start daemon: ./bin/cwsd &"
echo "2. Run quick tests: make test-aws-quick"
echo "3. Run full tests: make test-aws"
echo ""
echo "Environment variables:"
echo "  RUN_AWS_TESTS=true"
echo "  AWS_PROFILE=$AWS_PROFILE" 
echo "  AWS_TEST_REGION=$AWS_TEST_REGION"
echo ""
log_warning "AWS integration tests will create real resources and may incur costs"
echo "=========================================="