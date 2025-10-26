#!/bin/bash
# Setup GitHub Projects V2 Board
# Requires GitHub CLI with 'project' scope

set -e

REPO_OWNER="scttfrdmn"
REPO_NAME="prism"

echo "🚀 Setting up GitHub Projects Board"
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

# ============================================================================
# Check for Project Scope
# ============================================================================

echo "📋 Checking for required 'project' scope..."

TEST_QUERY=$(gh api graphql -f query='
  query {
    viewer {
      login
    }
  }
' 2>&1)

# Try to check if we have project scope by attempting a simple project query
SCOPE_TEST=$(gh api graphql -f query='
  query {
    viewer {
      projectsV2(first: 1) {
        totalCount
      }
    }
  }
' 2>&1)

if echo "$SCOPE_TEST" | grep -q "INSUFFICIENT_SCOPES"; then
    echo "❌ GitHub CLI token missing 'project' scope!"
    echo ""
    echo "To fix this, run:"
    echo ""
    echo "  gh auth refresh -s project"
    echo ""
    echo "This will prompt you to authorize the 'project' scope."
    echo "After authorizing, run this script again."
    echo ""
    exit 1
fi

echo "✅ Project scope is available"
echo ""

# ============================================================================
# Manual Instructions
# ============================================================================

echo "=================================================="
echo "GitHub Projects V2 Setup Instructions"
echo "=================================================="
echo ""
echo "Unfortunately, due to API limitations and scope requirements,"
echo "it's simpler to create the Projects board through the web UI."
echo ""
echo "📋 Follow these steps:"
echo ""
echo "1. Go to: https://github.com/$REPO_OWNER?tab=projects"
echo ""
echo "2. Click 'New project' button"
echo ""
echo "3. Select 'Board' template"
echo ""
echo "4. Name it: 'CloudWorkstation Development'"
echo ""
echo "5. Click 'Create project'"
echo ""
echo "6. Customize the board:"
echo "   - Rename 'Todo' → 'Backlog'"
echo "   - Add 'Ready' column (after Backlog)"
echo "   - Keep 'In Progress'"
echo "   - Add 'Review' column (after In Progress)"
echo "   - Keep 'Done'"
echo ""
echo "7. Add issues to the board:"
echo "   - Click '+ Add item' in Backlog"
echo "   - Search for issues #13-#20"
echo "   - Or bulk-add from: https://github.com/$REPO_OWNER/$REPO_NAME/issues"
echo ""
echo "8. Organize issues:"
echo "   - Move #13-#17 to 'Ready' column (Phase 5.0.1)"
echo "   - Keep #18-#20 in 'Backlog' (Phase 5.0.2+)"
echo ""
echo "9. Link project to repository:"
echo "   - In project settings, link to $REPO_OWNER/$REPO_NAME"
echo ""
echo "=================================================="
echo ""
echo "🎯 Priority order for 'Ready' column:"
echo "   #13 - Home Page with Quick Start Wizard"
echo "   #14 - Merge Terminal/WebView into Workspaces"
echo "   #15 - Rename 'Instances' → 'Workspaces'"
echo "   #16 - Collapse Advanced Features"
echo "   #17 - Add 'cws init' Wizard"
echo ""
echo "📚 All issues already have:"
echo "   ✅ Proper labels (ux-improvement, priority, area, persona)"
echo "   ✅ Assigned to milestones (Phase 5.0.1, 5.0.2, 5.0.3)"
echo "   ✅ Detailed requirements and success metrics"
echo ""
echo "This should take about 5 minutes through the web UI."
echo "=================================================="
echo ""
