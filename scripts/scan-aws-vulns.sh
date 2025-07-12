#!/bin/bash
# scan-aws-vulns.sh - Focus on AWS SDK vulnerabilities which are the most critical

set -e

echo "🔎 Scanning AWS SDK dependencies for vulnerabilities..."

# Get all AWS SDK dependencies
aws_deps=$(grep -E "github.com/aws/aws-sdk-go-v2" go.mod | awk '{print $1}')

if [ -z "$aws_deps" ]; then
  echo "❌ No AWS SDK dependencies found in go.mod"
  exit 1
fi

# Download AWS SDK dependencies explicitly
echo "📥 Downloading AWS SDK dependencies..."
for dep in $aws_deps; do
  echo "   - $dep"
  go mod download $dep
done

# Generate temporary go file for scanning
tmp_file=$(mktemp)
echo "package main

import (
" > $tmp_file

for dep in $aws_deps; do
  echo "  _ \"$dep\"" >> $tmp_file
done

echo ")

func main() {}" >> $tmp_file

# Run govulncheck on just the AWS dependencies
echo "🔒 Scanning AWS dependencies..."
if govulncheck $tmp_file 2>/dev/null; then
  echo "✅ No vulnerabilities found in AWS SDK dependencies"
else
  echo "⚠️ Potential vulnerabilities in AWS SDK dependencies"
fi

# Clean up
rm $tmp_file

echo "✨ AWS SDK vulnerability scan complete"