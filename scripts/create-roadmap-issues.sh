#!/bin/bash
# Create GitHub Issues from ROADMAP.md
# This script migrates roadmap items to properly labeled GitHub issues

set -e

REPO="scttfrdmn/cloudworkstation"

echo "🚀 Creating GitHub Issues from ROADMAP.md"
echo "=========================================="
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
# Phase 5.0.1: Quick Wins (2 weeks)
# ============================================================================

echo "📋 Creating Phase 5.0.1 issues (Quick Wins)..."
echo ""

gh issue create --repo "$REPO" \
  --title "[UX] Home Page with Quick Start Wizard" \
  --label "ux-improvement,priority: critical,area: gui,phase: 5.0-ux-redesign" \
  --milestone "Phase 5.0.1: Quick Wins" \
  --body "## Description

Create a smart Home page that serves as the entry point for all users with contextual guidance.

## Requirements

### First-Time Users
- Quick Start guide walkthrough
- Template selection wizard
- AWS setup validation

### Returning Users
- Recent activity summary
- Recent workspaces (quick reconnect)
- Recommended actions based on state

### Context-Aware Features
- Show relevant features based on user state
- Highlight unused features with educational tooltips
- Cost alerts and budget status (if configured)

## Success Metrics

- **Impact**: 90% reduction in \"what do I do first?\" confusion
- **Measurement**: Time to first workspace launch
- **Target**: 15 minutes → 2 minutes (87% improvement)

## Persona Impact

- ✅ Solo Researcher - Reduced onboarding friction
- ✅ Lab Environment - Team members get started faster
- ✅ University Class - Students can begin coursework immediately
- ✅ Conference Workshop - Attendees ready in minutes

## Implementation Notes

- Implement in GUI (cmd/cws-gui) first
- Consider TUI equivalent (simplified dashboard)
- Update CLI to suggest \`cws tui\` or \`cws-gui\` for first-time users

## Related

- Part of [UX Evaluation Recommendations](https://github.com/scttfrdmn/prism/blob/main/docs/architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md)
- Addresses: Navigation complexity, onboarding friction
" || true

gh issue create --repo "$REPO" \
  --title "[UX] Merge Terminal/WebView into Workspaces" \
  --label "ux-improvement,priority: critical,area: gui,phase: 5.0-ux-redesign" \
  --milestone "Phase 5.0.1: Quick Wins" \
  --body "## Description

Terminal and WebView should not be top-level navigation items. They are contextual actions on workspaces, not destinations.

## Requirements

### Navigation Changes
- Remove \"Terminal\" and \"Web View\" from main navigation
- Add contextual dropdown/button on each workspace row
- Support multiple terminal sessions simultaneously
- Support opening multiple web services

### User Experience
- Click workspace → Actions → \"Open Terminal\" or \"Open Web Service\"
- Terminal opens in system terminal (existing behavior)
- Web services open in browser (existing behavior)
- Keep history of recent connections for quick access

## Success Metrics

- **Impact**: 14% navigation complexity reduction (14 items → 12 items)
- **Measurement**: Navigation items count
- **UX Improvement**: Actions are contextual to workspaces

## Persona Impact

- ✅ All personas - Clearer navigation, less cognitive load
- ✅ Solo Researcher - Faster workspace access
- ✅ Lab Environment - Clearer workspace management

## Implementation Notes

- Update GUI navigation (cmd/cws-gui/frontend)
- Update TUI navigation (internal/tui)
- CLI already works this way (\`cws connect\`, \`cws webview\`)

## Related

- Part of [UX Evaluation Recommendations](https://github.com/scttfrdmn/prism/blob/main/docs/architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md)
- Addresses: Navigation complexity
" || true

gh issue create --repo "$REPO" \
  --title "[UX] Rename \"Instances\" → \"Workspaces\"" \
  --label "ux-improvement,priority: critical,area: gui,area: cli,area: tui,phase: 5.0-ux-redesign" \
  --milestone "Phase 5.0.1: Quick Wins" \
  --body "## Description

\"Instances\" is AWS jargon. \"Workspaces\" is researcher-friendly terminology that better describes what users are creating.

## Requirements

### Code Changes
- Update all user-facing strings in GUI, TUI, CLI
- Update command help text
- Update API responses (user-facing fields only)
- Keep internal variable names as \`instance\` (technical accuracy)

### Documentation Changes
- Update all user guides
- Update persona walkthroughs
- Update troubleshooting docs
- Add note about terminology for AWS-familiar users

### Backward Compatibility
- CLI commands remain the same (\`cws launch\`, \`cws list\`, etc.)
- API endpoints remain the same (internal technical names)
- Only display strings change

## Success Metrics

- **Impact**: Clearer mental model for non-technical users
- **Measurement**: User feedback, support ticket reduction
- **Target**: Eliminate \"What's an instance?\" questions

## Persona Impact

- ✅ Solo Researcher - More approachable terminology
- ✅ University Class - Students understand immediately
- ✅ Conference Workshop - No AWS knowledge required
- ⚠️ IT Admins - Clarify relationship to EC2 instances in docs

## Implementation Notes

- This is primarily a string replacement task
- Create glossary explaining terminology mapping (for AWS-familiar users)
- Consider adding \`--instance-id\` flag aliases for AWS power users

## Related

- Part of [UX Evaluation Recommendations](https://github.com/scttfrdmn/prism/blob/main/docs/architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md)
- Addresses: Terminology accessibility
" || true

gh issue create --repo "$REPO" \
  --title "[UX] Collapse Advanced Features Under Settings" \
  --label "ux-improvement,priority: critical,area: gui,phase: 5.0-ux-redesign" \
  --milestone "Phase 5.0.1: Quick Wins" \
  --body "## Description

AMI Management, Rightsizing, and Idle Detection are advanced features that should not compete with core workflows in main navigation.

## Requirements

### Navigation Restructure
- Move AMI, Rightsizing, Idle Detection to Settings > Advanced
- Keep Settings as top-level item
- Add \"Advanced\" section within Settings (collapsed by default)
- Add badges showing feature status (e.g., \"5 idle policies active\")

### Discoverability
- Add educational tooltip: \"Advanced features for power users\"
- Show feature benefits when hovering/expanding
- Link from relevant contexts (e.g., \"Optimize costs\" → Idle Detection)

### Progressive Disclosure
- Beginners see clean navigation
- Power users can expand Advanced section
- Contextual hints guide users to advanced features when relevant

## Success Metrics

- **Impact**: 64% reduction in cognitive load (14 items → 5 items)
- **Measurement**: Navigation items count
- **Target**: 14 navigation items → 5 navigation items

## Persona Impact

- ✅ Solo Researcher - Less overwhelming interface
- ✅ University Class - Students focus on core workflow
- ⚠️ IT Admins - Ensure advanced features remain discoverable

## Implementation Notes

- Update GUI navigation structure (cmd/cws-gui/frontend)
- Update TUI navigation (internal/tui)
- CLI remains unchanged (advanced commands still available)
- Add contextual links from dashboard (e.g., cost alerts → Idle Detection)

## Related

- Part of [UX Evaluation Recommendations](https://github.com/scttfrdmn/prism/blob/main/docs/architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md)
- Addresses: Navigation complexity, progressive disclosure
" || true

gh issue create --repo "$REPO" \
  --title "[UX] Add 'cws init' Onboarding Wizard" \
  --label "enhancement,priority: critical,area: cli,phase: 5.0-ux-redesign" \
  --milestone "Phase 5.0.1: Quick Wins" \
  --body "## Description

Create an interactive first-time setup wizard that guides users through CloudWorkstation configuration.

## Requirements

### Interactive Setup Flow

\`\`\`
\$ cws init

Welcome to CloudWorkstation!
This wizard will help you set up your research environment.

1. AWS Configuration
   ✓ AWS credentials detected (~/.aws/credentials)
   ✓ Default region: us-west-2

2. Research Area
   What type of research do you do?
   > Machine Learning / Data Science
     Bioinformatics / Genomics
     Social Science / Statistics
     Other

3. Budget (Optional)
   Would you like to set a monthly budget?
   > Yes, set a budget alert
     No, I'll monitor manually

   Budget: \$100/month

4. Hibernation (Recommended)
   Automatically hibernate idle workspaces to save costs?
   > Yes, use 'balanced' policy (30 min idle)
     Customize idle detection
     No, I'll manage manually

5. Templates
   Based on your research area, we recommend:
   - Python Machine Learning (Simplified)
   - R Research Environment (Simplified)

   Download recommended templates? [Y/n]: y

✅ Setup complete! Launch your first workspace:
   cws launch \"Python Machine Learning (Simplified)\" my-first-project
\`\`\`

### Features
- AWS credential detection and validation
- Research area selection (maps to idle policies and template recommendations)
- Optional budget configuration
- Hibernation policy setup
- Template discovery and installation
- Profile creation (if using multiple AWS accounts)

### Post-Setup
- Save configuration to ~/.cloudworkstation/config.yaml
- Show next steps (launch first workspace)
- Link to relevant documentation based on research area

## Success Metrics

- **Impact**: 15min → 2min onboarding (87% improvement)
- **Measurement**: Time from installation to first workspace launch
- **Target**: First-time users productive in under 2 minutes

## Persona Impact

- ✅ Solo Researcher - Guided setup, less overwhelm
- ✅ Lab Environment - Consistent configuration across team
- ✅ University Class - Students ready immediately
- ✅ Conference Workshop - Attendees set up in minutes

## Implementation Notes

- Implement as \`cws init\` command in internal/cli/
- Use interactive prompts (survey library or similar)
- Auto-run on first \`cws\` command if ~/.cloudworkstation/ doesn't exist
- Allow skipping with \`cws init --skip\` or \`PRISM_SKIP_INIT=1\`

## Related

- Part of [UX Evaluation Recommendations](https://github.com/scttfrdmn/prism/blob/main/docs/architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md)
- Addresses: Onboarding friction, first-time user experience
- See: [Zero-Setup Guide](https://github.com/scttfrdmn/prism/blob/main/docs/user-guides/ZERO_SETUP_GUIDE.md)
" || true

echo "✅ Phase 5.0.1 issues created"
echo ""

# ============================================================================
# Phase 5.0.2: Information Architecture (4 weeks)
# ============================================================================

echo "📋 Creating Phase 5.0.2 issues (Information Architecture)..."
echo ""

gh issue create --repo "$REPO" \
  --title "[UX] Unified Storage UI (EFS + EBS)" \
  --label "ux-improvement,priority: high,area: gui,phase: 5.0-ux-redesign" \
  --milestone "Phase 5.0.2: Information Architecture" \
  --body "## Description

Users are confused by \"Storage\" vs \"Volume\" terminology. Create single Storage UI with clear tabs explaining differences.

## Requirements

### UI Structure
- Single \"Storage\" page in navigation
- Two tabs: \"Shared (EFS)\" and \"Private (EBS)\"
- Educational tooltips explaining when to use each
- Unified actions (create, attach, delete)

### Educational Content
- **Shared (EFS)**: For data shared across multiple workspaces
- **Private (EBS)**: For workspace-specific data

### Contextual Actions
- \"Attach to workspace\" → shows compatible workspaces
- \"Create from template\" → suggests size based on use case
- Cost comparison (EFS vs EBS pricing)

## Success Metrics

- **Impact**: Eliminates #1 user confusion
- **Measurement**: Support tickets about storage
- **Target**: 50% reduction in storage-related questions

## Persona Impact

- ✅ Solo Researcher - Understands storage options immediately
- ✅ Lab Environment - Shared EFS for collaboration is clear
- ✅ University Class - Students know where to put coursework

## Related

- Part of [UX Evaluation Recommendations](https://github.com/scttfrdmn/prism/blob/main/docs/architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md)
- Addresses: Storage confusion, terminology clarity
" || true

gh issue create --repo "$REPO" \
  --title "[UX] Integrate Budgets into Projects" \
  --label "ux-improvement,priority: high,area: gui,phase: 5.0-ux-redesign" \
  --milestone "Phase 5.0.2: Information Architecture" \
  --body "## Description

Budget should be a tab within Project detail view, not a separate navigation item.

## Requirements

### UI Changes
- Remove \"Budget\" from main navigation
- Add \"Budget\" tab to Project detail view
- Show per-collaborator spending breakdown
- Show workspace-level costs within project

### Workflow Integration
- Project creation wizard can optionally set budget
- Budget alerts show in project dashboard
- Cost forecasting based on project workspace usage

## Success Metrics

- **Impact**: Makes project budgets discoverable
- **Measurement**: Budget feature usage (currently <5%)
- **Target**: 30%+ of projects have budgets configured

## Persona Impact

- ✅ Lab Environment - PI sets budget per project/grant
- ✅ University Class - Instructor sets budget per semester
- ✅ Cross-Institutional - Multi-institution budget tracking

## Related

- Part of [UX Evaluation Recommendations](https://github.com/scttfrdmn/prism/blob/main/docs/architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md)
- Addresses: Feature discoverability, information architecture
" || true

echo "✅ Phase 5.0.2 issues created"
echo ""

# ============================================================================
# Phase 5.0.3: CLI Consistency
# ============================================================================

echo "📋 Creating Phase 5.0.3 issues (CLI Consistency)..."
echo ""

gh issue create --repo "$REPO" \
  --title "[UX] Consistent CLI Command Structure" \
  --label "ux-improvement,priority: high,area: cli,phase: 5.0-ux-redesign" \
  --milestone "Phase 5.0.3: CLI Consistency" \
  --body "## Description

Restructure CLI commands into consistent, predictable groups following verb-noun-object pattern.

## Current Problems

- 40+ flat commands with inconsistent patterns
- \`cws hibernate\` vs \`cws volume create\` (inconsistent verb placement)
- \`cws storage\` and \`cws volume\` (confusing overlap)
- No clear command grouping

## Proposed Structure

\`\`\`
cws workspace <action>   # Workspace management
  launch, list, stop, start, delete, hibernate, resume, connect, webview

cws storage <action>     # Unified storage (EFS + EBS)
  list, create, attach, detach, delete
  --type efs|ebs         # Specify storage type

cws templates <action>   # Template management
  list, info, install, validate, discover, search

cws collab <action>      # Collaboration (projects, users)
  project, user, share

cws admin <action>       # Advanced/admin features
  policy, ami, rightsizing, idle, daemon

cws config <action>      # Configuration
  init, profile, set
\`\`\`

## Requirements

- Consistent verb-noun-object pattern everywhere
- Group related commands under namespaces
- Maintain backward compatibility (deprecated command warnings)
- Tab completion support
- Consistent help text formatting

## Success Metrics

- **Impact**: CLI first-attempt success from 35% to 85%
- **Measurement**: Command error rates, user feedback
- **Target**: Predictable, discoverable command structure

## Persona Impact

- ✅ All personas - More predictable CLI
- ✅ Solo Researcher - Easier to remember commands
- ✅ IT Admins - Scriptable, consistent automation

## Implementation Notes

- Phase 1: Add new grouped commands alongside existing
- Phase 2: Deprecation warnings for old commands
- Phase 3: Remove old commands in v1.0.0 (breaking change)

## Related

- Part of [UX Evaluation Recommendations](https://github.com/scttfrdmn/prism/blob/main/docs/architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md)
- Addresses: CLI consistency, discoverability
- Technical debt: Cobra migration (see #TBD)
" || true

echo "✅ Phase 5.0.3 issues created"
echo ""

echo "=================================================="
echo "✅ GitHub Issues Created from ROADMAP.md"
echo "=================================================="
echo ""
echo "View all issues: https://github.com/$REPO/issues"
echo "View Phase 5.0.1 milestone: https://github.com/$REPO/milestone/2"
echo ""
echo "Next steps:"
echo "1. Review and adjust issue priorities"
echo "2. Assign issues to team members"
echo "3. Add issues to GitHub Projects board"
echo "4. Archive ROADMAP.md to docs/archive/planning/"
echo ""
