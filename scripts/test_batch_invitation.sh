#!/bin/bash
# Test script for batch invitation functionality

set -e

# Color codes for better readability
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting batch invitation system tests${NC}"

# Create temporary test directory
TEST_DIR=$(mktemp -d)
echo "Using temporary directory: ${TEST_DIR}"

# Create test CSV file
CSV_FILE="${TEST_DIR}/test-invitations.csv"
cat > "${CSV_FILE}" << EOF
Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Test User 1,read_only,30,no,no,yes,1
Test User 2,read_write,60,no,no,yes,2
Test Admin,admin,90,yes,no,yes,3
EOF

echo -e "${GREEN}Created test CSV file with 3 invitations${NC}"

# Create output file path
OUTPUT_FILE="${TEST_DIR}/results.csv"

# Run unit tests
echo -e "${YELLOW}Running unit tests for batch invitation system${NC}"
go test -v ./pkg/profile/batch_invitation_test.go ./pkg/profile/invitation.go ./pkg/profile/secure_invitation.go ./pkg/profile/batch_invitation.go
echo -e "${GREEN}Unit tests completed${NC}"

# Run CLI integration tests
echo -e "${YELLOW}Running CLI integration tests${NC}"
go test -v ./internal/cli/batch_invitation_test.go
echo -e "${GREEN}CLI integration tests completed${NC}"

# Check if we can run live tests (requires AWS credentials)
if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
    echo -e "${YELLOW}Skipping live tests (AWS credentials not found)${NC}"
else
    echo -e "${YELLOW}Running live batch invitation tests${NC}"

    # Build the binary if needed
    if [ ! -f "./bin/cws" ]; then
        echo "Building cws binary..."
        go build -o ./bin/cws ./cmd/cws
    fi

    # Run batch creation test
    echo "Testing batch invitation creation..."
    ./bin/cws profiles invitations batch-create \
        --csv-file "${CSV_FILE}" \
        --output-file "${OUTPUT_FILE}"

    # Check if output file was created
    if [ -f "${OUTPUT_FILE}" ]; then
        echo -e "${GREEN}Successfully created batch invitations and exported results${NC}"
        echo "Result summary:"
        head -n 5 "${OUTPUT_FILE}"
        echo "..."
    else
        echo -e "${RED}Failed to create output file${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}Live tests completed${NC}"
fi

# Run integration tests in isolated environment
echo -e "${YELLOW}Running integration tests in isolated environment${NC}"
CI=true go test -v ./pkg/profile/batch_invitation_integration_test.go ./pkg/profile/invitation.go ./pkg/profile/secure_invitation.go ./pkg/profile/batch_invitation.go
echo -e "${GREEN}Integration tests completed${NC}"

echo -e "${GREEN}All batch invitation tests completed successfully${NC}"

# Clean up
rm -rf "${TEST_DIR}"
echo "Cleaned up test directory"