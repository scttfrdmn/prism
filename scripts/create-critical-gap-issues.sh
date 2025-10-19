#!/bin/bash
set -e

REPO="scttfrdmn/cloudworkstation"

echo "🚀 Creating GitHub issues for critical documentation gaps..."
echo ""

# Create milestones for new phases
echo "📦 Creating milestones for gap-filling phases..."

create_milestone_if_needed() {
    local title="$1"
    local description="$2"

    if gh api "repos/$REPO/milestones" --jq ".[].title" | grep -q "^$title$"; then
        echo "  ⏭️  Milestone '$title' already exists"
    else
        gh api "repos/$REPO/milestones" -f title="$title" -f description="$description"
        echo "  ✅ Created milestone: $title"
    fi
}

create_milestone_if_needed "Phase 0.6.0: Budget Safety Net" "Personal budgets, alerts, pre-launch checks for solo researchers"
create_milestone_if_needed "Phase 0.7.0: Class Management Basics" "Course creation, template whitelisting, student management"
create_milestone_if_needed "Phase 0.7.1: TA Support Tools" "TA debug access, instance reset, student support"
create_milestone_if_needed "Phase 0.8.0: Invitation Security" "Device binding, batch operations, secure invitations"
create_milestone_if_needed "Phase 0.9.0: DCV Desktop Integration" "NICE DCV GUI workstations, desktop templates"

echo ""
echo "📦 Creating Phase 0.6.0: Budget Safety Net issues..."

gh issue create \
  --repo "$REPO" \
  --title "[Budget] Personal Budget System for Solo Researchers" \
  --body "## Summary
Implement personal budget system separate from project budgets to support solo researcher workflow.

## Problem
**Current State**: Only project-level budgets exist. Solo researchers can't set personal monthly budgets.

**Documented Promise** (01_SOLO_RESEARCHER_WALKTHROUGH.md):
\`\`\`bash
cws budget set --monthly 100
cws budget show
cws budget status
\`\`\`

**Impact**: Scenario 1 (Solo Researcher) workflow is blocked. Researchers can't manage personal spending.

## Implementation Tasks
- [ ] Personal budget data model (separate from project budgets)
- [ ] Budget storage in user/profile state
- [ ] \`cws budget set --monthly AMOUNT\` command
- [ ] \`cws budget show\` command
- [ ] \`cws budget status\` command
- [ ] Budget inheritance: personal → instance
- [ ] Per-user budget tracking in cost calculator

## Persona Impact
- **Solo Researcher**: Core workflow - personal budget management
- **Lab Environment**: Optional personal budgets within project allocation

## Success Metrics
- Solo researchers can set monthly personal budgets
- Budget status shows current spend vs limit
- Commands match documented examples

## Documentation
- [01_SOLO_RESEARCHER_WALKTHROUGH.md](docs/USER_SCENARIOS/01_SOLO_RESEARCHER_WALKTHROUGH.md) lines 98-280
- [DOCUMENTATION_AUDIT_REPORT.md](docs/DOCUMENTATION_AUDIT_REPORT.md) Category 3.1

## Related
- Audit finding: Category 3.1 Personal Budget Configuration
- Priority: 🔴 Critical
- Effort: Medium (2 weeks)" \
  --milestone "Phase 0.6.0: Budget Safety Net" \
  --label "enhancement,priority: critical,area: budget,phase: 0.6.0-budget-safety"

gh issue create \
  --repo "$REPO" \
  --title "[Budget] Budget Alerts and Email Notifications" \
  --body "## Summary
Send email alerts when users approach or exceed budget thresholds to prevent overspending anxiety.

## Problem
**Current State**: No alert system. Users discover overspending after the fact.

**Documented Promise** (01_SOLO_RESEARCHER_WALKTHROUGH.md):
\`\`\`bash
cws budget set --monthly 100 --alert-email user@example.com
cws budget alert add --threshold 80 --email user@example.com

# Receives email: ⚠️ CloudWorkstation Budget Alert: 80% Used
\`\`\`

**Impact**: Researchers stress about unexpected cloud bills. No proactive warnings.

## Implementation Tasks
- [ ] Alert configuration system (thresholds, recipients)
- [ ] \`cws budget alert add/list/remove\` commands
- [ ] Email delivery integration (AWS SES preferred)
- [ ] Alert threshold tracking (50%, 80%, 90%, 100%)
- [ ] Alert history and deduplication
- [ ] Email templates (warning, exceeded)
- [ ] Optional Slack/webhook integrations

## Persona Impact
- **Solo Researcher**: Reduces spending anxiety
- **Lab Environment**: Team budget alerts
- **University Class**: Instructor budget monitoring

## Success Metrics
- Email sent at 80% budget threshold
- Clear actionable guidance in alert emails
- Alert history viewable via CLI

## Documentation
- [01_SOLO_RESEARCHER_WALKTHROUGH.md](docs/USER_SCENARIOS/01_SOLO_RESEARCHER_WALKTHROUGH.md)
- [DOCUMENTATION_AUDIT_REPORT.md](docs/DOCUMENTATION_AUDIT_REPORT.md) Category 3.2

## Related
- Audit finding: Category 3.2 Budget Alerts
- Priority: 🔴 Critical
- Effort: Medium (1 week)" \
  --milestone "Phase 0.6.0: Budget Safety Net" \
  --label "enhancement,priority: critical,area: budget,phase: 0.6.0-budget-safety"

gh issue create \
  --repo "$REPO" \
  --title "[Budget] Pre-Launch Budget Impact Preview" \
  --body "## Summary
Show budget impact before launching instances to prevent accidental overspending.

## Problem
**Current State**: No budget check before launch. Users discover cost impact after launch.

**Documented Promise** (01_SOLO_RESEARCHER_WALKTHROUGH.md):
\`\`\`bash
cws launch bioinformatics-suite rnaseq-analysis

# Expected output:
# 📊 Budget Impact Preview
#    Cost: \$2.40/day
#    Current: \$45 / \$100 (45%)
#    Projected: ~\$81 / \$100 (81%) ✅
# Proceed? [Y/n]:
\`\`\`

**Impact**: Researchers launch expensive instances without realizing cost impact.

## Implementation Tasks
- [ ] Pre-launch budget validation hook
- [ ] Cost projection calculation
- [ ] Interactive confirmation prompt
- [ ] Budget impact display (current + projected)
- [ ] Optional \`--yes\` flag to skip confirmation
- [ ] Budget blocking (prevent launch if over budget)
- [ ] \`--force\` override for emergency launches

## Persona Impact
- **Solo Researcher**: Prevents accidental budget exhaustion
- **University Class**: Students see cost before launching
- **Conference Workshop**: Attendees understand cost impact

## Success Metrics
- Budget impact shown before every launch
- Clear visual indicators (✅ ⚠️ ❌)
- Launches blocked when over budget (with override)

## Documentation
- [01_SOLO_RESEARCHER_WALKTHROUGH.md](docs/USER_SCENARIOS/01_SOLO_RESEARCHER_WALKTHROUGH.md)
- [DOCUMENTATION_AUDIT_REPORT.md](docs/DOCUMENTATION_AUDIT_REPORT.md) Category 3.3

## Related
- Audit finding: Category 3.3 Pre-Launch Budget Impact
- Priority: 🔴 Critical
- Effort: Low (3 days)" \
  --milestone "Phase 0.6.0: Budget Safety Net" \
  --label "enhancement,priority: critical,area: budget,phase: 0.6.0-budget-safety"

gh issue create \
  --repo "$REPO" \
  --title "[Budget] Budget Forecasting and Planning Tool" \
  --body "## Summary
Forecast end-of-month spending and help researchers plan within budget constraints.

## Problem
**Current State**: No forecasting. Researchers can't predict if they'll stay within budget.

**Documented Promise** (01_SOLO_RESEARCHER_WALKTHROUGH.md):
\`\`\`bash
cws budget forecast

# Output:
# Current spend: \$45.00 (Day 15 of 30)
# Projected end-of-month: \$90.00
# Remaining buffer: \$10.00 ✅
# Can I launch another instance?
# ✅ t3.medium (\$0.80/day): Yes, \$14 projected = \$104 total ⚠️
\`\`\`

**Impact**: Researchers can't confidently plan launches or know if they can afford experiments.

## Implementation Tasks
- [ ] \`cws budget forecast\` command
- [ ] Time-series cost analysis
- [ ] Linear projection algorithm
- [ ] \"Can I afford this?\" calculator
- [ ] What-if scenario modeling
- [ ] Historical usage pattern analysis
- [ ] Instance cost rate lookup

## Persona Impact
- **Solo Researcher**: Confident planning within budget
- **Lab Environment**: Team resource planning
- **University Class**: Help students plan projects

## Success Metrics
- Accurate month-end projections (±10%)
- Clear affordability guidance
- Helpful what-if scenarios

## Documentation
- [01_SOLO_RESEARCHER_WALKTHROUGH.md](docs/USER_SCENARIOS/01_SOLO_RESEARCHER_WALKTHROUGH.md)
- [DOCUMENTATION_AUDIT_REPORT.md](docs/DOCUMENTATION_AUDIT_REPORT.md) Category 3.4

## Related
- Audit finding: Category 3.4 Budget Forecasting
- Priority: 🟡 High
- Effort: Medium (1 week)" \
  --milestone "Phase 0.6.0: Budget Safety Net" \
  --label "enhancement,priority: high,area: budget,phase: 0.6.0-budget-safety"

echo "✅ Phase 0.6.0 issues created"
echo ""

echo "📦 Creating Phase 0.7.0: Class Management Basics issues..."

gh issue create \
  --repo "$REPO" \
  --title "[Class] Course Creation and Management System" \
  --body "## Summary
Implement course/class entity as specialized project type with education-specific features.

## Problem
**Current State**: No course management. Instructors use generic projects without education features.

**Documented Promise** (03_UNIVERSITY_CLASS_WALKTHROUGH.md):
\`\`\`bash
cws course create \"CS229-Fall2024\" --interactive

# Interactive wizard:
# - Course details (code, title, dates)
# - Student budget allocation
# - Teaching staff roles
# - Approved templates
# - Auto-semester-end actions
\`\`\`

**Impact**: Scenario 3 (University Class) workflow completely blocked.

## Implementation Tasks
- [ ] Course data model (extends project with semester dates)
- [ ] \`cws course create\` command with interactive wizard
- [ ] \`cws course list\` command
- [ ] \`cws course show COURSE_ID\` with student list
- [ ] \`cws course update\` command
- [ ] \`cws course close\` for semester end
- [ ] Semester date tracking (start, end, grace period)
- [ ] Auto-cleanup scheduling for semester end
- [ ] Course-specific metadata (department, credits, level)

## Persona Impact
- **University Class**: Core workflow - course creation and management
- **Conference Workshop**: Workshop sessions as short \"courses\"

## Success Metrics
- Instructors can create courses via wizard
- Course details match academic calendar
- Auto-cleanup prevents leftover instances

## Documentation
- [03_UNIVERSITY_CLASS_WALKTHROUGH.md](docs/USER_SCENARIOS/03_UNIVERSITY_CLASS_WALKTHROUGH.md)
- [DOCUMENTATION_AUDIT_REPORT.md](docs/DOCUMENTATION_AUDIT_REPORT.md) Category 4.1

## Related
- Audit finding: Category 4.1 Course Creation Wizard
- Priority: 🔴 Critical
- Effort: High (2 weeks)" \
  --milestone "Phase 0.7.0: Class Management Basics" \
  --label "enhancement,priority: critical,area: education,phase: 0.7.0-class-mgmt"

gh issue create \
  --repo "$REPO" \
  --title "[Class] Template Whitelisting for Courses" \
  --body "## Summary
Enforce template whitelists at course level to prevent students from launching unapproved expensive instances.

## Problem
**Current State**: Students can launch any template. No cost controls for classes.

**Documented Promise** (03_UNIVERSITY_CLASS_WALKTHROUGH.md):
\`\`\`bash
cws course create \"CS229-Fall2024\" \\
  --approved-templates \"ml-cpu-student,ml-final-project\"

# Student tries unapproved template:
cws launch gpu-ml-workstation homework1
# ❌ Launch BLOCKED: Template not approved for CS229-Fall2024
\`\`\`

**Impact**: Students accidentally launch expensive GPU instances. Budget disasters.

## Implementation Tasks
- [ ] Template whitelist configuration in course
- [ ] Launch-time template validation
- [ ] Educational error messages with approved alternatives
- [ ] \`cws course templates add/remove\` commands
- [ ] \`cws course templates list\` command
- [ ] Template approval workflow for instructors
- [ ] GUI template selector filtered by course
- [ ] Override capability for instructors

## Persona Impact
- **University Class**: Prevents budget disasters
- **Conference Workshop**: Limit attendees to workshop templates

## Success Metrics
- Student launches blocked when template not approved
- Clear guidance on approved templates
- Zero unauthorized GPU launches

## Documentation
- [03_UNIVERSITY_CLASS_WALKTHROUGH.md](docs/USER_SCENARIOS/03_UNIVERSITY_CLASS_WALKTHROUGH.md)
- [DOCUMENTATION_AUDIT_REPORT.md](docs/DOCUMENTATION_AUDIT_REPORT.md) Category 4.2

## Related
- Audit finding: Category 4.2 Template Whitelisting
- Priority: 🔴 Critical
- Effort: Medium (1 week)" \
  --milestone "Phase 0.7.0: Class Management Basics" \
  --label "enhancement,priority: critical,area: education,area: policy,phase: 0.7.0-class-mgmt"

gh issue create \
  --repo "$REPO" \
  --title "[Class] Student Bulk Management and Budget Distribution" \
  --body "## Summary
Import student rosters and distribute budgets across class members efficiently.

## Problem
**Current State**: No bulk student management. Manual one-by-one addition.

**Documented Promise** (03_UNIVERSITY_CLASS_WALKTHROUGH.md):
\`\`\`bash
# Import student roster
cws course students import CS229-Fall2024 roster.csv

# roster.csv:
# email,name,budget
# alice@university.edu,Alice Chen,50
# bob@university.edu,Bob Smith,50

# Distribute equal budgets
cws course budget distribute CS229-Fall2024 --amount 50 --per-student
\`\`\`

**Impact**: Instructors waste hours adding students manually.

## Implementation Tasks
- [ ] \`cws course students import\` command
- [ ] CSV parsing with validation
- [ ] Bulk user creation/invitation
- [ ] Per-student budget allocation
- [ ] \`cws course budget distribute\` command
- [ ] Student list export
- [ ] Email notifications to students
- [ ] Enrollment confirmation tracking

## Persona Impact
- **University Class**: Efficient course setup
- **Conference Workshop**: Bulk attendee management

## Success Metrics
- Import 100 students in <30 seconds
- Budgets distributed correctly
- Students receive welcome emails

## Documentation
- [03_UNIVERSITY_CLASS_WALKTHROUGH.md](docs/USER_SCENARIOS/03_UNIVERSITY_CLASS_WALKTHROUGH.md)
- [DOCUMENTATION_AUDIT_REPORT.md](docs/DOCUMENTATION_AUDIT_REPORT.md) Category 4.4

## Related
- Audit finding: Category 4.4 Student Bulk Management
- Priority: 🔴 Critical
- Effort: Medium (1 week)" \
  --milestone "Phase 0.7.0: Class Management Basics" \
  --label "enhancement,priority: critical,area: education,phase: 0.7.0-class-mgmt"

echo "✅ Phase 0.7.0 issues created"
echo ""

echo "📦 Creating Phase 0.7.1: TA Support Tools issues..."

gh issue create \
  --repo "$REPO" \
  --title "[TA] Debug Access (God Mode) for Student Support" \
  --body "## Summary
Enable TAs to debug student instances for office hours and grading support.

## Problem
**Current State**: TAs can't access student instances. Students must share screenshots or error messages.

**Documented Promise** (03_UNIVERSITY_CLASS_WALKTHROUGH.md):
\`\`\`bash
cws ta debug ml-hw3 --student sophie.martinez@university.edu

# Options:
# [1] View instance status and logs
# [2] SSH into instance (full access, logged)
# [3] View Jupyter notebooks (read-only)
# [4] Export student workspace for review
# [5] Reset instance (backup + fresh start)
\`\`\`

**Impact**: Office hours inefficient. TAs can't see what students see.

## Implementation Tasks
- [ ] TA role definition in course/project system
- [ ] \`cws ta debug\` command with interactive menu
- [ ] Temporary SSH access with audit logging
- [ ] Read-only Jupyter notebook viewer
- [ ] Workspace export for grading
- [ ] \`cws ta ssh\` command with automatic logging
- [ ] Session recording for academic integrity
- [ ] \`cws ta annotate\` for leaving messages
- [ ] Access revocation after session

## Persona Impact
- **University Class**: Efficient office hours and debugging
- **Lab Environment**: Senior researchers help junior members

## Success Metrics
- TAs can access student instances securely
- All sessions logged for audit
- 50% reduction in \"can you see my screen\" questions

## Documentation
- [03_UNIVERSITY_CLASS_WALKTHROUGH.md](docs/USER_SCENARIOS/03_UNIVERSITY_CLASS_WALKTHROUGH.md)
- [DOCUMENTATION_AUDIT_REPORT.md](docs/DOCUMENTATION_AUDIT_REPORT.md) Category 4.3

## Related
- Audit finding: Category 4.3 TA Debug Access
- Priority: 🔴 Critical
- Effort: High (2 weeks)" \
  --milestone "Phase 0.7.1: TA Support Tools" \
  --label "enhancement,priority: critical,area: education,security,phase: 0.7.1-ta-tools"

gh issue create \
  --repo "$REPO" \
  --title "[TA] Instance Reset Capability for Fresh Starts" \
  --body "## Summary
Allow TAs to reset student instances to clean state while preserving backup for review.

## Problem
**Current State**: No reset capability. Students stuck with corrupted instances.

**Documented Promise** (03_UNIVERSITY_CLASS_WALKTHROUGH.md):
\`\`\`bash
cws ta reset-instance ml-hw3 --student sophie.martinez@university.edu

# Actions:
# 1. Create snapshot backup of current state
# 2. Stop instance
# 3. Re-provision from template
# 4. Notify student via email
# 5. Store backup for 7 days
\`\`\`

**Impact**: Students with broken environments can't continue work.

## Implementation Tasks
- [ ] \`cws ta reset-instance\` command
- [ ] Automatic snapshot creation
- [ ] Instance re-provisioning from template
- [ ] Student notification system
- [ ] Backup retention (7 days default)
- [ ] Optional student-initiated reset
- [ ] Reset history tracking
- [ ] Backup restoration if needed

## Persona Impact
- **University Class**: Students get fresh starts quickly
- **Conference Workshop**: Reset between sessions

## Success Metrics
- Reset completes in <5 minutes
- Student receives notification
- Backup available for review

## Documentation
- [03_UNIVERSITY_CLASS_WALKTHROUGH.md](docs/USER_SCENARIOS/03_UNIVERSITY_CLASS_WALKTHROUGH.md)
- [DOCUMENTATION_AUDIT_REPORT.md](docs/DOCUMENTATION_AUDIT_REPORT.md) Category 4.3

## Related
- Audit finding: Category 4.3 TA Debug Access (Reset capability)
- Priority: 🔴 Critical
- Effort: Medium (1 week)" \
  --milestone "Phase 0.7.1: TA Support Tools" \
  --label "enhancement,priority: critical,area: education,phase: 0.7.1-ta-tools"

echo "✅ Phase 0.7.1 issues created"
echo ""

echo "📦 Creating Phase 0.8.0: Invitation Security issues..."

gh issue create \
  --repo "$REPO" \
  --title "[Invitation] Wire CLI Commands to Existing Security Code" \
  --body "## Summary
Expose existing invitation security features (device binding, batch operations) via CLI commands.

## Problem
**Current State**: Security code exists in \`pkg/profile/security/\` but no CLI interface.

**Documented Promise** (ADMINISTRATOR_GUIDE.md, BATCH_INVITATION_GUIDE.md):
\`\`\`bash
# Device-bound invitations
cws profiles invitations create-secure lab-access \\
  --device-bound=true \\
  --max-devices=3

# Batch operations
cws profiles invitations batch-create --csv-file invitations.csv
cws profiles invitations batch-export --output-file invitations.csv
\`\`\`

**Impact**: Enterprise security features documented but inaccessible.

## Implementation Tasks
- [ ] \`cws profiles invitations create-secure\` CLI handler
- [ ] \`cws profiles invitations devices\` command (list devices)
- [ ] \`cws profiles invitations revoke-device\` command
- [ ] \`cws profiles invitations revoke-all\` command
- [ ] \`cws profiles invitations batch-create\` command
- [ ] \`cws profiles invitations batch-export\` command
- [ ] \`cws profiles invitations batch-accept\` command
- [ ] Wire existing \`pkg/profile/security/\` code to CLI
- [ ] Integration tests for all commands

## Persona Impact
- **Lab Environment**: Secure team invitations
- **University Class**: Batch student invitations
- **Cross-Institutional**: Device-bound multi-institution access

## Success Metrics
- All documented commands work
- Device binding enforced
- Batch operations complete successfully

## Documentation
- [ADMINISTRATOR_GUIDE.md](docs/admin-guides/ADMINISTRATOR_GUIDE.md)
- [BATCH_INVITATION_GUIDE.md](docs/admin-guides/BATCH_INVITATION_GUIDE.md)
- [DOCUMENTATION_AUDIT_REPORT.md](docs/DOCUMENTATION_AUDIT_REPORT.md) Category 1.1, 1.2

## Related
- Audit finding: Category 1 Invitation Security (code exists, CLI missing)
- Priority: 🟡 High
- Effort: Medium (3 weeks)" \
  --milestone "Phase 0.8.0: Invitation Security" \
  --label "enhancement,priority: high,area: auth,security,phase: 0.8.0-invitations"

echo "✅ Phase 0.8.0 issues created"
echo ""

echo "📦 Creating Phase 0.9.0: DCV Desktop Integration issues..."

gh issue create \
  --repo "$REPO" \
  --title "[DCV] NICE DCV Desktop Workstation Support" \
  --body "## Summary
Enable GUI desktop workstations via NICE DCV for visualization, graphical tools, and desktop applications.

## Problem
**Current State**: DCV proxy code exists in daemon but no desktop templates or CLI commands.

**Documented Promise** (Historical RELEASE_NOTES.md, archived docs):
\`\`\`bash
# Launch desktop workstation
cws launch desktop-research bio-viz

# Connect to desktop
cws dcv connect bio-viz
# Opens DCV client with desktop session
\`\`\`

**Impact**: Researchers needing GUI tools (RStudio IDE, MATLAB GUI, visualization) can't use CloudWorkstation.

## Implementation Tasks
- [ ] Desktop-enabled templates (Ubuntu Desktop + NICE DCV)
- [ ] DCV server provisioning in templates
- [ ] \`cws dcv\` command namespace
- [ ] \`cws dcv connect\` command (opens DCV client)
- [ ] \`cws dcv status\` command
- [ ] DCV port tunneling and security
- [ ] Browser-based DCV access (optional)
- [ ] Password/key management for DCV sessions
- [ ] GPU-accelerated desktop support

## Persona Impact
- **Solo Researcher**: GUI tools (RStudio IDE, MATLAB, ImageJ)
- **Lab Environment**: Shared desktop workstations
- **University Class**: Student desktop sessions for GUI courses

## Success Metrics
- Desktop templates launch successfully
- DCV sessions connect reliably
- GPU acceleration works for visualization

## Documentation
- [RELEASE_NOTES.md](docs/releases/RELEASE_NOTES.md) (historical reference)
- Archive: [AMI_BUILDER_IMPLEMENTATION.md](docs/archive/old-implementation/AMI_BUILDER_IMPLEMENTATION.md)
- [DOCUMENTATION_AUDIT_REPORT.md](docs/DOCUMENTATION_AUDIT_REPORT.md) Category 7

## Related
- Audit finding: Category 7 DCV/Desktop Integration
- Priority: 🟡 High
- Effort: High (3 weeks)" \
  --milestone "Phase 0.9.0: DCV Desktop Integration" \
  --label "enhancement,priority: high,area: templates,phase: 0.9.0-dcv"

echo "✅ Phase 0.9.0 issues created"
echo ""

echo "🎉 All critical gap issues created successfully!"
echo ""
echo "📊 Summary:"
echo "  - Phase 0.6.0: Budget Safety Net (4 issues)"
echo "  - Phase 0.7.0: Class Management Basics (3 issues)"
echo "  - Phase 0.7.1: TA Support Tools (2 issues)"
echo "  - Phase 0.8.0: Invitation Security (1 issue)"
echo "  - Phase 0.9.0: DCV Desktop Integration (1 issue)"
echo ""
echo "Total: 11 new critical issues created"
echo ""
echo "View issues: https://github.com/$REPO/issues"
echo "View audit report: docs/DOCUMENTATION_AUDIT_REPORT.md"
