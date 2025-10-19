#!/bin/bash
# Setup GitHub Labels from .github/labels.yml
# Compatible with older gh CLI versions

set -e

REPO="scttfrdmn/cloudworkstation"

echo "🚀 Creating GitHub Labels for $REPO"
echo "===================================="
echo ""

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI (gh) is not installed"
    echo "Install it with: brew install gh"
    exit 1
fi

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo "❌ Not authenticated with GitHub CLI"
    echo "Run: gh auth login"
    exit 1
fi

echo "✅ GitHub CLI is installed and authenticated"
echo ""

# Function to create or update a label
create_label() {
    local name="$1"
    local color="$2"
    local description="$3"

    # Check if label exists
    if gh label list --repo "$REPO" | grep -q "^$name"; then
        echo "  ⏭️  Label '$name' already exists"
    else
        gh label create "$name" --repo "$REPO" --color "$color" --description "$description" 2>/dev/null && \
            echo "  ✅ Created label: $name" || \
            echo "  ⚠️  Failed to create: $name"
    fi
}

echo "📋 Creating labels..."
echo ""

# Type Labels
echo "Creating type labels..."
create_label "bug" "d73a4a" "Something isn't working correctly"
create_label "enhancement" "a2eeef" "New feature or request"
create_label "ux-improvement" "bfdadc" "Usability or user experience improvement"
create_label "documentation" "0075ca" "Improvements or additions to documentation"
create_label "technical-debt" "fbca04" "Code refactoring or improvement needed"

# Priority Labels
echo ""
echo "Creating priority labels..."
create_label "priority: critical" "b60205" "Highest priority - blocking work or severe impact"
create_label "priority: high" "d93f0b" "High priority - should be addressed soon"
create_label "priority: medium" "fbca04" "Medium priority - important but not urgent"
create_label "priority: low" "0e8a16" "Low priority - nice to have"

# Area Labels
echo ""
echo "Creating area labels..."
create_label "area: cli" "c5def5" "Command-line interface (cmd/cws, internal/cli)"
create_label "area: gui" "c5def5" "Desktop GUI application (cmd/cws-gui)"
create_label "area: tui" "c5def5" "Terminal interface (internal/tui)"
create_label "area: daemon" "c5def5" "Backend daemon (cmd/cwsd, pkg/daemon)"
create_label "area: templates" "c5def5" "Template system and marketplace"
create_label "area: aws" "c5def5" "AWS integration (pkg/aws)"
create_label "area: build" "c5def5" "Build system, CI/CD, packaging"
create_label "area: tests" "c5def5" "Testing infrastructure and test coverage"

# Persona Labels
echo ""
echo "Creating persona labels..."
create_label "persona: solo-researcher" "e99695" "Benefits solo researcher workflow"
create_label "persona: lab-environment" "e99695" "Benefits lab collaboration workflow"
create_label "persona: university-class" "e99695" "Benefits teaching/coursework workflow"
create_label "persona: conference-workshop" "e99695" "Benefits workshop/tutorial workflow"
create_label "persona: cross-institutional" "e99695" "Benefits multi-institution collaboration"

# Status Labels
echo ""
echo "Creating status labels..."
create_label "triage" "ededed" "Needs initial review and prioritization"
create_label "needs-info" "d876e3" "Waiting for more information from reporter"
create_label "blocked" "b60205" "Blocked by external dependency or other issue"
create_label "ready" "0e8a16" "Ready to be worked on"
create_label "in-progress" "fbca04" "Currently being worked on"
create_label "in-review" "fef2c0" "In code review"
create_label "awaiting-merge" "c2e0c6" "Approved and ready to merge"

# Resolution Labels
echo ""
echo "Creating resolution labels..."
create_label "duplicate" "cfd3d7" "This issue or PR already exists"
create_label "wontfix" "ffffff" "This will not be worked on"
create_label "invalid" "e4e669" "This doesn't seem right or is not applicable"
create_label "works-as-designed" "fef2c0" "Behavior is intentional"

# Special Labels
echo ""
echo "Creating special labels..."
create_label "good first issue" "7057ff" "Good for newcomers to the project"
create_label "help wanted" "008672" "Extra attention is needed from community"
create_label "breaking-change" "d73a4a" "Changes that break backward compatibility"
create_label "security" "ee0701" "Security-related issue or improvement"
create_label "performance" "1d76db" "Performance optimization or issue"
create_label "dependencies" "0366d6" "Pull requests that update a dependency file"

# Phase Labels
echo ""
echo "Creating phase labels..."
create_label "phase: 5.0-ux-redesign" "bfd4f2" "Part of Phase 5.0 UX Redesign"
create_label "phase: 5.1-universal-ami" "bfd4f2" "Part of Phase 5.1 Universal AMI System"
create_label "phase: 5.2-marketplace" "bfd4f2" "Part of Phase 5.2 Template Marketplace Enhancement"
create_label "phase: 5.3-config-sync" "bfd4f2" "Part of Phase 5.3 Configuration & Directory Sync"
create_label "phase: 5.4-aws-services" "bfd4f2" "Part of Phase 5.4 AWS Research Services"
create_label "phase: 6.0-extensibility" "bfd4f2" "Part of Phase 6 Extensibility & Ecosystem"

echo ""
echo "✅ Labels created successfully!"
echo ""
echo "View labels: https://github.com/$REPO/labels"
echo ""
