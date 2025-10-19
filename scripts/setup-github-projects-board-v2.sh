#!/bin/bash
# Setup GitHub Projects V2 Board using GraphQL API
# This creates a user-level project board and adds repository issues to it

set -e

REPO_OWNER="scttfrdmn"
REPO_NAME="cloudworkstation"

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
# Step 1: Get User ID
# ============================================================================

echo "📋 Step 1: Getting user ID..."
USER_ID=$(gh api graphql -f query='
  query {
    viewer {
      id
      login
    }
  }
' --jq '.data.viewer.id')

USER_LOGIN=$(gh api graphql -f query='
  query {
    viewer {
      login
    }
  }
' --jq '.data.viewer.login')

echo "✅ User: $USER_LOGIN"
echo "✅ User ID: $USER_ID"
echo ""

# ============================================================================
# Step 2: Create Project
# ============================================================================

echo "📋 Step 2: Creating project 'CloudWorkstation Development'..."

PROJECT_RESPONSE=$(gh api graphql -f query='
  mutation($ownerId: ID!) {
    createProjectV2(input: {
      ownerId: $ownerId
      title: "CloudWorkstation Development"
    }) {
      projectV2 {
        id
        number
        url
      }
    }
  }
' -f ownerId="$USER_ID" 2>&1)

PROJECT_ID=$(echo "$PROJECT_RESPONSE" | jq -r '.data.createProjectV2.projectV2.id' 2>/dev/null)
PROJECT_NUMBER=$(echo "$PROJECT_RESPONSE" | jq -r '.data.createProjectV2.projectV2.number' 2>/dev/null)
PROJECT_URL=$(echo "$PROJECT_RESPONSE" | jq -r '.data.createProjectV2.projectV2.url' 2>/dev/null)

if [ -z "$PROJECT_ID" ] || [ "$PROJECT_ID" = "null" ]; then
    echo "⚠️  Project may already exist, trying to find it..."

    # List existing projects
    EXISTING_PROJECT=$(gh api graphql -f query='
      query($login: String!) {
        user(login: $login) {
          projectsV2(first: 20) {
            nodes {
              id
              title
              number
              url
            }
          }
        }
      }
    ' -f login="$USER_LOGIN" --jq '.data.user.projectsV2.nodes[] | select(.title == "CloudWorkstation Development")')

    if [ -n "$EXISTING_PROJECT" ]; then
        PROJECT_ID=$(echo "$EXISTING_PROJECT" | jq -r '.id')
        PROJECT_NUMBER=$(echo "$EXISTING_PROJECT" | jq -r '.number')
        PROJECT_URL=$(echo "$EXISTING_PROJECT" | jq -r '.url')
        echo "✅ Found existing project"
    else
        echo "❌ Failed to create or find project"
        echo "Error response: $PROJECT_RESPONSE"
        exit 1
    fi
fi

echo "✅ Project ID: $PROJECT_ID"
echo "✅ Project URL: $PROJECT_URL"
echo ""

# ============================================================================
# Step 3: Get Status Field ID
# ============================================================================

echo "📋 Step 3: Getting status field configuration..."

STATUS_FIELD=$(gh api graphql -f query='
  query($projectId: ID!) {
    node(id: $projectId) {
      ... on ProjectV2 {
        fields(first: 20) {
          nodes {
            ... on ProjectV2SingleSelectField {
              id
              name
              options {
                id
                name
              }
            }
          }
        }
      }
    }
  }
' -f projectId="$PROJECT_ID" --jq '.data.node.fields.nodes[] | select(.name == "Status")')

STATUS_FIELD_ID=$(echo "$STATUS_FIELD" | jq -r '.id')
TODO_ID=$(echo "$STATUS_FIELD" | jq -r '.options[] | select(.name == "Todo") | .id')
IN_PROGRESS_ID=$(echo "$STATUS_FIELD" | jq -r '.options[] | select(.name == "In Progress") | .id')
DONE_ID=$(echo "$STATUS_FIELD" | jq -r '.options[] | select(.name == "Done") | .id')

echo "✅ Status field ID: $STATUS_FIELD_ID"
echo "   Todo: $TODO_ID"
echo "   In Progress: $IN_PROGRESS_ID"
echo "   Done: $DONE_ID"
echo ""

# ============================================================================
# Step 4: Add Issues to Project
# ============================================================================

echo "📋 Step 4: Adding issues to project..."
echo ""

# Get repository ID
REPO_ID=$(gh api graphql -f query='
  query($owner: String!, $name: String!) {
    repository(owner: $owner, name: $name) {
      id
    }
  }
' -f owner="$REPO_OWNER" -f name="$REPO_NAME" --jq '.data.repository.id')

ADDED_COUNT=0
FAILED_COUNT=0

# Add issue IDs for issues #13-#20
for issue_num in {13..20}; do
    echo "  Processing issue #$issue_num..."

    # Get issue global ID
    ISSUE_ID=$(gh api graphql -f query='
      query($owner: String!, $name: String!, $issueNumber: Int!) {
        repository(owner: $owner, name: $name) {
          issue(number: $issueNumber) {
            id
            title
          }
        }
      }
    ' -f owner="$REPO_OWNER" -f name="$REPO_NAME" -F issueNumber="$issue_num" --jq '.data.repository.issue.id' 2>/dev/null)

    if [ -z "$ISSUE_ID" ] || [ "$ISSUE_ID" = "null" ]; then
        echo "    ⚠️  Issue #$issue_num not found, skipping..."
        FAILED_COUNT=$((FAILED_COUNT + 1))
        continue
    fi

    # Add issue to project
    ADD_RESPONSE=$(gh api graphql -f query='
      mutation($projectId: ID!, $contentId: ID!) {
        addProjectV2ItemById(input: {
          projectId: $projectId
          contentId: $contentId
        }) {
          item {
            id
          }
        }
      }
    ' -f projectId="$PROJECT_ID" -f contentId="$ISSUE_ID" 2>&1)

    ITEM_ID=$(echo "$ADD_RESPONSE" | jq -r '.data.addProjectV2ItemById.item.id' 2>/dev/null)

    if [ -n "$ITEM_ID" ] && [ "$ITEM_ID" != "null" ]; then
        echo "    ✅ Added to project"
        ADDED_COUNT=$((ADDED_COUNT + 1))

        # Set status based on issue number
        # Issues #13-17 are Phase 5.0.1 (mark as "In Progress" for now, will be "Ready")
        # Issues #18-20 are Phase 5.0.2+ (mark as "Todo" for Backlog)
        if [ "$issue_num" -le 17 ]; then
            STATUS_ID="$IN_PROGRESS_ID"
            STATUS_NAME="In Progress"
        else
            STATUS_ID="$TODO_ID"
            STATUS_NAME="Todo"
        fi

        # Update item status
        UPDATE_RESPONSE=$(gh api graphql -f query='
          mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $value: String!) {
            updateProjectV2ItemFieldValue(input: {
              projectId: $projectId
              itemId: $itemId
              fieldId: $fieldId
              value: {
                singleSelectOptionId: $value
              }
            }) {
              projectV2Item {
                id
              }
            }
          }
        ' -f projectId="$PROJECT_ID" -f itemId="$ITEM_ID" -f fieldId="$STATUS_FIELD_ID" -f value="$STATUS_ID" 2>&1)

        if echo "$UPDATE_RESPONSE" | jq -e '.data.updateProjectV2ItemFieldValue.projectV2Item.id' &>/dev/null; then
            echo "    ✅ Set status to '$STATUS_NAME'"
        else
            echo "    ⚠️  Could not set status"
        fi
    else
        echo "    ⚠️  Failed to add to project"
        echo "    Response: $ADD_RESPONSE"
        FAILED_COUNT=$((FAILED_COUNT + 1))
    fi

    echo ""
done

# ============================================================================
# Summary
# ============================================================================

echo "=================================================="
echo "✅ GitHub Projects Board Setup Complete!"
echo "=================================================="
echo ""
echo "Project URL: $PROJECT_URL"
echo ""
echo "✅ Created/found project 'CloudWorkstation Development'"
echo "✅ Added $ADDED_COUNT issues to project"
if [ $FAILED_COUNT -gt 0 ]; then
    echo "⚠️  Failed to add $FAILED_COUNT issues"
fi
echo ""
echo "📝 Next steps (manual customization in web UI):"
echo "1. Go to: $PROJECT_URL"
echo "2. Click 'Settings' (⚙️) in top-right"
echo "3. Under 'Status' field options:"
echo "   a. Rename 'Todo' → 'Backlog'"
echo "   b. Add 'Ready' option (drag between Backlog and In Progress)"
echo "   c. Add 'Review' option (drag between In Progress and Done)"
echo "4. Move Phase 5.0.1 issues (#13-17) from 'In Progress' to 'Ready'"
echo "5. Move Phase 5.0.2+ issues (#18-20) from 'Todo' to 'Backlog'"
echo ""
echo "🎯 Priority order in 'Ready' column:"
echo "   #13 - Home Page with Quick Start Wizard"
echo "   #14 - Merge Terminal/WebView into Workspaces"
echo "   #15 - Rename 'Instances' → 'Workspaces'"
echo "   #16 - Collapse Advanced Features"
echo "   #17 - Add 'cws init' Wizard"
echo ""
