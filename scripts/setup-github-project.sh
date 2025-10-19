#!/bin/bash
# Setup GitHub Project Management Structure
# This script creates labels, milestones, and project boards for CloudWorkstation

set -e

REPO="scttfrdmn/cloudworkstation"

echo "🚀 Setting up GitHub Project Management for $REPO"
echo "=================================================="
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

# ============================================================================
# Step 1: Create Labels
# ============================================================================

echo "📋 Step 1: Creating labels..."
echo ""
echo "Run: ./scripts/setup-github-labels.sh to create labels"
echo "(Skipping label creation in this script for compatibility)"
echo ""

# ============================================================================
# Step 2: Create Milestones
# ============================================================================

echo "📋 Step 2: Creating milestones..."
echo ""

# Function to create milestone if it doesn't exist
create_milestone() {
    local title="$1"
    local due_date="$2"
    local description="$3"

    # Check if milestone exists
    if gh api "repos/$REPO/milestones" --jq ".[].title" | grep -q "^$title$"; then
        echo "  ⏭️  Milestone '$title' already exists"
    else
        if [ -n "$due_date" ]; then
            gh api "repos/$REPO/milestones" -f title="$title" -f description="$description" -f due_on="$due_date"
        else
            gh api "repos/$REPO/milestones" -f title="$title" -f description="$description"
        fi
        echo "  ✅ Created milestone: $title"
    fi
}

# Phase 5.0: UX Redesign (Q4 2025)
create_milestone \
    "Phase 5.0: UX Redesign" \
    "2025-12-31T23:59:59Z" \
    "Critical UX improvements: Home page, navigation restructure, CLI consistency. Target: Reduce onboarding from 15min to 2min, navigation from 14 to 5 items."

# Phase 5.0.1: Quick Wins
create_milestone \
    "Phase 5.0.1: Quick Wins" \
    "2025-11-15T23:59:59Z" \
    "High-impact, low-effort UX improvements (2 weeks): Home page, merge Terminal/WebView, rename Instances→Workspaces, collapse advanced features, add cws init wizard."

# Phase 5.0.2: Information Architecture
create_milestone \
    "Phase 5.0.2: Information Architecture" \
    "2025-12-15T23:59:59Z" \
    "Navigation restructure (4 weeks): Unified storage UI, integrate budgets into projects, reorganize navigation (14→5 items), role-based visibility, context-aware recommendations."

# Phase 5.0.3: CLI Consistency
create_milestone \
    "Phase 5.0.3: CLI Consistency" \
    "2025-12-31T23:59:59Z" \
    "CLI command restructure (2 weeks): Consistent command structure, unified storage commands, predictable patterns, tab completion."

# Phase 5.1: Universal AMI System
create_milestone \
    "Phase 5.1: Universal AMI System" \
    "2026-03-31T23:59:59Z" \
    "Universal AMI reference, auto-compilation, cross-region copying. Target: 30-second launches vs 5-8 minute provisioning."

# Phase 5.2: Template Marketplace Enhancement
create_milestone \
    "Phase 5.2: Template Marketplace Enhancement" \
    "2026-03-31T23:59:59Z" \
    "Decentralized repositories, community/institutional templates, authentication, BYOL licensing for commercial software."

# Phase 5.3: Configuration & Directory Sync
create_milestone \
    "Phase 5.3: Configuration & Directory Sync" \
    "2026-06-30T23:59:59Z" \
    "Template-based config sync, EFS bidirectional sync, conflict resolution, cross-platform support."

# Phase 5.4: AWS Research Services
create_milestone \
    "Phase 5.4: AWS Research Services" \
    "2026-06-30T23:59:59Z" \
    "EMR Studio integration, SageMaker Studio Lab, Amazon Braket, unified web service framework."

# Phase 6.0: Extensibility & Ecosystem
create_milestone \
    "Phase 6.0: Extensibility & Ecosystem" \
    "2026-09-30T23:59:59Z" \
    "Plugin architecture, auto-AMI system, GUI skinning, web services integration framework."

echo ""
echo "✅ Milestones created successfully"
echo ""

# ============================================================================
# Step 3: Summary
# ============================================================================

echo "=================================================="
echo "✅ GitHub Project Setup Complete!"
echo "=================================================="
echo ""
echo "Next steps:"
echo "1. Review labels: https://github.com/$REPO/labels"
echo "2. Review milestones: https://github.com/$REPO/milestones"
echo "3. Create GitHub Projects board manually (gh CLI doesn't support ProjectsV2 yet)"
echo "   - Go to: https://github.com/$REPO/projects"
echo "   - Create new project (Board view)"
echo "   - Add columns: Backlog, Ready, In Progress, Review, Done"
echo "4. Migrate ROADMAP.md items to issues with proper labels and milestones"
echo ""
echo "Recommended board configuration:"
echo "  - Backlog: Triaged but not yet prioritized"
echo "  - Ready: Prioritized and ready to work on"
echo "  - In Progress: Currently being worked on"
echo "  - Review: In code review"
echo "  - Done: Completed (auto-archive after 2 weeks)"
echo ""
