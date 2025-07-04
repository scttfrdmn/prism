#!/bin/bash
# Test script for the AMI builder integration tests

set -e

# Start LocalStack if not running
echo "Starting LocalStack for integration testing..."
docker-compose -f docker-compose.test.yml up -d localstack

# Wait for LocalStack to be ready
echo "Waiting for LocalStack to be ready..."
timeout=30
while [ $timeout -gt 0 ]; do
  if curl -s http://localhost:4566/health | grep -q '"ready": true'; then
    echo "LocalStack is ready!"
    break
  fi
  echo "Waiting for LocalStack... ($timeout seconds left)"
  sleep 2
  timeout=$((timeout - 2))
done

if [ $timeout -le 0 ]; then
  echo "ERROR: LocalStack failed to start within the timeout period"
  docker-compose -f docker-compose.test.yml logs localstack
  docker-compose -f docker-compose.test.yml down
  exit 1
fi

# Run AMI builder integration tests
echo "Running AMI builder integration tests..."
INTEGRATION_TESTS=1 go test -v -tags=integration ./pkg/ami

# Run AWS integration tests (includes multi-region support)
echo "Running AWS package integration tests..."
INTEGRATION_TESTS=1 go test -v -tags=integration ./pkg/aws

# Collect and display test coverage
echo "Collecting test coverage..."
INTEGRATION_TESTS=1 go test -v -tags=integration -coverprofile=ami_coverage.out ./pkg/ami
go tool cover -func=ami_coverage.out | grep "total"

# Generate HTML coverage report
go tool cover -html=ami_coverage.out -o ami_coverage.html
echo "Coverage report generated: ami_coverage.html"

# Shut down LocalStack
echo "Shutting down LocalStack..."
docker-compose -f docker-compose.test.yml down

echo "AMI builder tests completed successfully!"