#!/bin/bash

# CloudWorkstation documentation standards checker
# Validates documentation consistency and standards

set -e

echo "📚 Checking CloudWorkstation documentation standards..."

# Color functions
red() { echo -e "\033[31m$*\033[0m"; }
green() { echo -e "\033[32m$*\033[0m"; }
yellow() { echo -e "\033[33m$*\033[0m"; }
blue() { echo -e "\033[34m$*\033[0m"; }

errors=0
warnings=0

check_file() {
    local file="$1"
    local description="$2"
    
    if [ -f "$file" ]; then
        green "✓ $description: $file"
    else
        red "✗ Missing $description: $file"
        ((errors++))
    fi
}

check_optional_file() {
    local file="$1"
    local description="$2"
    
    if [ -f "$file" ]; then
        green "✓ $description: $file"
    else
        yellow "! Optional $description not found: $file"
        ((warnings++))
    fi
}

echo ""
echo "Checking core documentation files..."

# Core documentation files
check_file "README.md" "Main README"
check_file "CHANGELOG.md" "Changelog"
check_file "LICENSE" "License"
check_file "CLAUDE.md" "Claude development context"

echo ""
echo "Checking user guides..."

# User documentation
check_file "docs/GETTING_STARTED.md" "Getting Started guide"
check_file "docs/TUI_USER_GUIDE.md" "TUI user guide"
check_file "docs/GUI_USER_GUIDE.md" "GUI user guide"

echo ""
echo "Checking technical documentation..."

# Technical documentation
check_file "TEMPLATE_SYSTEM.md" "Template system documentation"
check_file "docs/TEMPLATE_FORMAT.md" "Template format documentation"
check_file "docs/API_AUTHENTICATION.md" "API authentication guide"

echo ""
echo "Checking installation guides..."

# Installation documentation
check_file "INSTALL.md" "Installation guide"
check_file "docs/MACOS_DMG_INSTALLATION.md" "macOS DMG installation"
check_optional_file "docs/CHOCOLATEY_PACKAGE.md" "Chocolatey package documentation"
check_optional_file "docs/CONDA_PACKAGE.md" "Conda package documentation"

echo ""
echo "Checking README format..."

# Check README format
if grep -q "# CloudWorkstation" README.md; then
    green "✓ README has proper title"
else
    red "✗ README missing proper title"
    ((errors++))
fi

if grep -q "## Installation" README.md; then
    green "✓ README has installation section"
else
    yellow "! README missing installation section"
    ((warnings++))
fi

if grep -q "## Usage" README.md; then
    green "✓ README has usage section"
else
    yellow "! README missing usage section"
    ((warnings++))
fi

echo ""
echo "Checking CHANGELOG format..."

# Check CHANGELOG format
if grep -q "# Changelog" CHANGELOG.md; then
    green "✓ CHANGELOG has proper title"
else
    red "✗ CHANGELOG missing proper title"
    ((errors++))
fi

if grep -q "## \[" CHANGELOG.md; then
    green "✓ CHANGELOG has version entries"
else
    yellow "! CHANGELOG missing version entries"
    ((warnings++))
fi

echo ""
echo "Documentation standards check complete!"
echo ""

if [ $errors -eq 0 ]; then
    green "🎉 All required documentation files are present!"
else
    red "❌ Found $errors missing required documentation files"
fi

if [ $warnings -gt 0 ]; then
    yellow "⚠️  Found $warnings documentation warnings"
fi

echo ""
echo "Summary:"
echo "  Errors: $errors"
echo "  Warnings: $warnings"

if [ $errors -eq 0 ]; then
    exit 0
else
    exit 1
fi