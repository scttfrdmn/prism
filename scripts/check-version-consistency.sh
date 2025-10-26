#!/bin/bash
# Check version consistency across all components
#
# This script ensures that:
# 1. pkg/version/version.go has the definitive version
# 2. All package.json files match that version
# 3. All built binaries report the correct version
# 4. CLAUDE.md roadmap matches the version

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Extract version from pkg/version/version.go
CODE_VERSION=$(grep -E '^\s*Version\s*=\s*"' pkg/version/version.go | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$CODE_VERSION" ]; then
    echo -e "${RED}❌ Could not extract version from pkg/version/version.go${NC}"
    exit 1
fi

echo -e "${GREEN}📦 Source code version: $CODE_VERSION${NC}"
echo ""

# Track failures
FAILURES=0

# Check package.json files
echo "Checking package.json files..."
for pkg_json in frontend/package.json cmd/cws-gui/frontend/package.json; do
    if [ -f "$pkg_json" ]; then
        PKG_VERSION=$(grep -E '^\s*"version":\s*"' "$pkg_json" | sed -E 's/.*"version":\s*"([^"]+)".*/\1/')
        if [ "$PKG_VERSION" = "$CODE_VERSION" ]; then
            echo -e "  ${GREEN}✓${NC} $pkg_json: $PKG_VERSION"
        else
            echo -e "  ${RED}✗${NC} $pkg_json: $PKG_VERSION (expected $CODE_VERSION)"
            FAILURES=$((FAILURES + 1))
        fi
    fi
done
echo ""

# Check CLAUDE.md for version references
echo "Checking CLAUDE.md documentation..."
if grep -q "v0\\.5\\.6\\|0\\.5\\.6\\|Phase 5" CLAUDE.md; then
    echo -e "  ${GREEN}✓${NC} CLAUDE.md references current version context"
else
    echo -e "  ${YELLOW}⚠${NC}  CLAUDE.md may need version updates"
fi
echo ""

# Check built binaries (if they exist)
echo "Checking built binaries..."

check_binary_version() {
    local binary=$1
    local name=$2

    if [ -f "$binary" ]; then
        # Run binary with --version and extract version
        BIN_VERSION=$($binary --version 2>&1 | head -1 | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | sed 's/v//')

        if [ -z "$BIN_VERSION" ]; then
            echo -e "  ${RED}✗${NC} $name: Could not extract version"
            FAILURES=$((FAILURES + 1))
        elif [ "$BIN_VERSION" = "$CODE_VERSION" ]; then
            echo -e "  ${GREEN}✓${NC} $name: v$BIN_VERSION"
        else
            echo -e "  ${RED}✗${NC} $name: v$BIN_VERSION (expected $CODE_VERSION) - STALE BINARY!"
            FAILURES=$((FAILURES + 1))
        fi
    else
        echo -e "  ${YELLOW}⚠${NC}  $name: binary not found (run 'make build')"
    fi
}

check_binary_version "bin/prismd" "Daemon (prismd)"
check_binary_version "bin/prism" "CLI (prism)"

# GUI version check - Wails binary doesn't support --version flag
# Check if GUI binary exists and verify against package.json
if [ -f "bin/cws-gui" ]; then
    # GUI version comes from package.json - try both possible locations
    GUI_PKG_JSON=""
    if [ -f "cmd/prism-gui/frontend/package.json" ]; then
        GUI_PKG_JSON="cmd/prism-gui/frontend/package.json"
    elif [ -f "cmd/cws-gui/frontend/package.json" ]; then
        GUI_PKG_JSON="cmd/cws-gui/frontend/package.json"
    fi

    if [ -n "$GUI_PKG_JSON" ]; then
        PKG_VERSION=$(grep '"version"' "$GUI_PKG_JSON" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        if [ "$PKG_VERSION" = "$CODE_VERSION" ]; then
            echo -e "  ${GREEN}✓${NC} GUI (cws-gui): v$PKG_VERSION (from package.json)"
        else
            echo -e "  ${RED}✗${NC} GUI (cws-gui): package.json has v$PKG_VERSION (expected $CODE_VERSION)"
            FAILURES=$((FAILURES + 1))
        fi
    else
        echo -e "  ${YELLOW}⚠${NC}  GUI (cws-gui): package.json not found"
    fi
else
    echo -e "  ${YELLOW}⚠${NC}  GUI (cws-gui): binary not found (run 'make build')"
fi

echo ""

# Check for legacy stale binaries
echo "Checking for legacy/stale binaries..."
STALE_BINARIES=0
for legacy in bin/cws bin/cwsd; do
    if [ -f "$legacy" ]; then
        TIMESTAMP=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M" "$legacy" 2>/dev/null || stat -c "%y" "$legacy" 2>/dev/null | cut -d' ' -f1-2)
        echo -e "  ${YELLOW}⚠${NC}  $legacy exists (last modified: $TIMESTAMP) - consider removing"
        STALE_BINARIES=$((STALE_BINARIES + 1))
    fi
done

if [ $STALE_BINARIES -eq 0 ]; then
    echo -e "  ${GREEN}✓${NC} No legacy binaries found"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ $FAILURES -eq 0 ]; then
    echo -e "${GREEN}✅ All version checks passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ $FAILURES version inconsistencies found${NC}"
    echo ""
    echo "To fix:"
    echo "  1. Update version in pkg/version/version.go"
    echo "  2. Update package.json files to match"
    echo "  3. Rebuild binaries: make clean && make build"
    echo "  4. Update CLAUDE.md if needed"
    exit 1
fi
