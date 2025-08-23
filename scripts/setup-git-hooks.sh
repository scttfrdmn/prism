#!/bin/bash
# Setup script for CloudWorkstation git hooks

echo "🔧 Setting up CloudWorkstation git hooks..."

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Configure git to use .githooks directory
git config core.hooksPath .githooks

# Make hooks executable
chmod +x "$PROJECT_ROOT/.githooks/pre-commit"
chmod +x "$PROJECT_ROOT/.githooks/pre-push"

echo -e "${GREEN}✅ Git hooks configured successfully!${NC}"
echo ""
echo "The following hooks are now active:"
echo "  • pre-commit: Quick tests (formatting, build, unit tests)"
echo "  • pre-push: Comprehensive tests (all tests, integration, E2E)"
echo ""
echo "To bypass hooks temporarily (not recommended):"
echo "  • Skip pre-commit: git commit --no-verify"
echo "  • Skip pre-push: git push --no-verify"
echo ""
echo -e "${YELLOW}Note: First push may take longer as it runs comprehensive tests${NC}"