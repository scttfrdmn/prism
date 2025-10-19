#!/bin/bash
# Setup GitHub Projects V2 Board using GraphQL API
# This creates a project board and adds issues to it

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
# Step 1: Get Repository ID
# ============================================================================

echo "📋 Step 1: Getting repository ID..."
REPO_ID=$(gh api graphql -f query='
  query($owner: String!, $name: String!) {
    repository(owner: $owner, name: $name) {
      id
    }
  }
' -f owner="$REPO_OWNER" -f name="$REPO_NAME" --jq '.data.repository.id')

echo "✅ Repository ID: $REPO_ID"
echo ""

# ============================================================================
# Step 2: Create Project
# ============================================================================

echo "📋 Step 2: Creating project 'CloudWorkstation Development'..."

PROJECT_ID=$(gh api graphql -f query='
  mutation($repositoryId: ID!) {
    createProjectV2(input: {
      repositoryId: $repositoryId
      title: "CloudWorkstation Development"
    }) {
      projectV2 {
        id
        number
      }
    }
  }
' -f repositoryId="$REPO_ID" --jq '.data.createProjectV2.projectV2.id' 2>/dev/null)

if [ -z "$PROJECT_ID" ]; then
    echo "⚠️  Project may already exist, trying to find it..."
    PROJECT_ID=$(gh api graphql -f query='
      query($owner: String!, $name: String!) {
        repository(owner: $owner, name: $name) {
          projectsV2(first: 10) {
            nodes {
              id
              title
              number
            }
          }
        }
      }
    ' -f owner="$REPO_OWNER" -f name="$REPO_NAME" --jq '.data.repository.projectsV2.nodes[] | select(.title == "CloudWorkstation Development") | .id')
fi

if [ -z "$PROJECT_ID" ]; then
    echo "❌ Failed to create or find project"
    exit 1
fi

echo "✅ Project ID: $PROJECT_ID"

# Get project number for URL
PROJECT_NUMBER=$(gh api graphql -f query='
  query($owner: String!, $name: String!) {
    repository(owner: $owner, name: $name) {
      projectsV2(first: 10) {
        nodes {
          id
          title
          number
        }
      }
    }
  }
' -f owner="$REPO_OWNER" -f name="$REPO_NAME" --jq ".data.repository.projectsV2.nodes[] | select(.id == \"$PROJECT_ID\") | .number")

echo "✅ Project created: https://github.com/users/$REPO_OWNER/projects/$PROJECT_NUMBER"
echo ""

# ============================================================================
# Step 3: Get Status Field ID
# ============================================================================

echo "📋 Step 3: Getting status field ID..."

STATUS_FIELD_ID=$(gh api graphql -f query='
  query($projectId: ID!) {
    node(id: $projectId) {
      ... on ProjectV2 {
        fields(first: 10) {
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
' -f projectId="$PROJECT_ID" --jq '.data.node.fields.nodes[] | select(.name == "Status") | .id')

echo "✅ Status field ID: $STATUS_FIELD_ID"
echo ""

# Get status option IDs
STATUS_OPTIONS=$(gh api graphql -f query='
  query($projectId: ID!) {
    node(id: $projectId) {
      ... on ProjectV2 {
        fields(first: 10) {
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

# Parse status option IDs
TODO_ID=$(echo "$STATUS_OPTIONS" | jq -r '.options[] | select(.name == "Todo") | .id')
IN_PROGRESS_ID=$(echo "$STATUS_OPTIONS" | jq -r '.options[] | select(.name == "In Progress") | .id')
DONE_ID=$(echo "$STATUS_OPTIONS" | jq -r '.options[] | select(.name == "Done") | .id')

echo "Status options:"
echo "  Todo: $TODO_ID"
echo "  In Progress: $IN_PROGRESS_ID"
echo "  Done: $DONE_ID"
echo ""

# ============================================================================
# Step 4: Add Custom Status Options
# ============================================================================

echo "📋 Step 4: Adding custom status options (Backlog, Ready, Review)..."

# Add "Backlog" option
gh api graphql -f query='
  mutation($projectId: ID!, $fieldId: ID!) {
    updateProjectV2Field(input: {
      projectId: $projectId
      fieldId: $fieldId
      name: "Status"
      singleSelectOptions: [
        {name: "Backlog", color: "GRAY"}
      ]
    }) {
      projectV2Field {
        ... on ProjectV2SingleSelectField {
          options {
            id
            name
          }
        }
      }
    }
  }
' -f projectId="$PROJECT_ID" -f fieldId="$STATUS_FIELD_ID" &>/dev/null || echo "  ⚠️  Could not add custom options (may already exist)"

echo "✅ Custom status options configured"
echo ""

# ============================================================================
# Step 5: Add Issues to Project
# ============================================================================

echo "📋 Step 5: Adding issues to project..."

# Get issue IDs for issues #13-#20
for issue_num in {13..20}; do
    echo "  Adding issue #$issue_num..."

    # Get issue global ID
    ISSUE_ID=$(gh api graphql -f query='
      query($owner: String!, $name: String!, $issueNumber: Int!) {
        repository(owner: $owner, name: $name) {
          issue(number: $issueNumber) {
            id
          }
        }
      }
    ' -f owner="$REPO_OWNER" -f name="$REPO_NAME" -F issueNumber="$issue_num" --jq '.data.repository.issue.id' 2>/dev/null)

    if [ -z "$ISSUE_ID" ]; then
        echo "    ⚠️  Issue #$issue_num not found, skipping..."
        continue
    fi

    # Add issue to project
    ITEM_ID=$(gh api graphql -f query='
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
    ' -f projectId="$PROJECT_ID" -f contentId="$ISSUE_ID" --jq '.data.addProjectV2ItemById.item.id' 2>/dev/null)

    if [ -n "$ITEM_ID" ]; then
        echo "    ✅ Added issue #$issue_num to project"

        # Set status based on issue number
        # Issues #13-17 are Phase 5.0.1 (Ready)
        # Issues #18-20 are Phase 5.0.2+ (Backlog/Todo)
        if [ "$issue_num" -le 17 ]; then
            STATUS_ID="$IN_PROGRESS_ID"  # Will need to change to "Ready" once we can update options
            STATUS_NAME="In Progress"
        else
            STATUS_ID="$TODO_ID"
            STATUS_NAME="Todo"
        fi

        # Update item status
        gh api graphql -f query='
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
        ' -f projectId="$PROJECT_ID" -f itemId="$ITEM_ID" -f fieldId="$STATUS_FIELD_ID" -f value="$STATUS_ID" &>/dev/null

        echo "    ✅ Set status to '$STATUS_NAME'"
    else
        echo "    ⚠️  Failed to add issue #$issue_num"
    fi
done

echo ""

# ============================================================================
# Summary
# ============================================================================

echo "=================================================="
echo "✅ GitHub Projects Board Setup Complete!"
echo "=================================================="
echo ""
echo "Project URL: https://github.com/users/$REPO_OWNER/projects/$PROJECT_NUMBER"
echo ""
echo "✅ Created project 'CloudWorkstation Development'"
echo "✅ Added 8 issues (#13-#20) to project"
echo "✅ Phase 5.0.1 issues (#13-17) marked as In Progress"
echo "✅ Phase 5.0.2+ issues (#18-20) marked as Todo/Backlog"
echo ""
echo "📝 Manual steps still needed:"
echo "1. Rename status columns:"
echo "   - 'Todo' → 'Backlog'"
echo "   - Keep 'In Progress'"
echo "   - Keep 'Done'"
echo "2. Add custom status options:"
echo "   - 'Ready' (between Backlog and In Progress)"
echo "   - 'Review' (between In Progress and Done)"
echo "3. Move Phase 5.0.1 issues from 'In Progress' to 'Ready'"
echo ""
echo "⚠️  Note: GitHub Projects V2 GraphQL API has limited support for"
echo "   custom status options. Some customization requires web UI."
echo ""
