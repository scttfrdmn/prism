#!/bin/bash
# Version validation script for Prism
# Ensures all version numbers across the codebase are synchronized
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Extract version from various sources
VERSION_GO=$(grep -m 1 'Version = ' pkg/version/version.go | sed 's/.*"\(.*\)".*/\1/')
VERSION_PACKAGE_JSON=$(grep -m 1 '"version":' cmd/prism-gui/frontend/package.json | sed 's/.*: *"\(.*\)".*/\1/')
VERSION_GIT_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "none")

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔍 Version Validation Report"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Found versions:"
echo "  pkg/version/version.go:              $VERSION_GO"
echo "  cmd/prism-gui/frontend/package.json: $VERSION_PACKAGE_JSON"
echo "  Latest git tag:                    $VERSION_GIT_TAG"
echo ""

# Check if all versions match
VERSIONS_MATCH=true

if [ "$VERSION_GO" != "$VERSION_PACKAGE_JSON" ]; then
    echo -e "${RED}✗ ERROR: Go version ($VERSION_GO) != package.json version ($VERSION_PACKAGE_JSON)${NC}"
    VERSIONS_MATCH=false
fi

# Extract version without 'v' prefix for comparison
VERSION_GIT_TAG_CLEAN=$(echo "$VERSION_GIT_TAG" | sed 's/^v//')

if [ "$VERSION_GO" != "$VERSION_GIT_TAG_CLEAN" ] && [ "$VERSION_GIT_TAG" != "none" ]; then
    echo -e "${YELLOW}⚠ WARNING: Go version ($VERSION_GO) != git tag ($VERSION_GIT_TAG_CLEAN)${NC}"
    echo -e "${YELLOW}  This is OK if you haven't created a git tag for the latest release yet.${NC}"
fi

echo ""

if [ "$VERSIONS_MATCH" = true ]; then
    echo -e "${GREEN}✓ All code versions are synchronized${NC}"
    echo ""
    exit 0
else
    echo -e "${RED}✗ Version mismatch detected!${NC}"
    echo ""
    echo "To fix:"
    echo "  1. Update pkg/version/version.go to set Version = \"X.Y.Z\""
    echo "  2. Update cmd/prism-gui/frontend/package.json to set \"version\": \"X.Y.Z\""
    echo "  3. Run this script again to verify"
    echo ""
    exit 1
fi
