#!/bin/bash

# Batch create GitHub issues for all persona feature gaps
# Based on comprehensive audit of USER_SCENARIOS documentation

cd "$(dirname "$0")/.." || exit 1

echo "Creating GitHub issues for persona feature gaps..."
echo "This will create 57 issues (one already created: #137)"
echo ""
read -p "Continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

# Track created issues
CREATED=0

# SOLO RESEARCHER GAPS (10 remaining)

gh issue create --title "📧 Budget Alert Notifications at Thresholds" --label "enhancement" --body "## Problem
Researchers need email notifications at budget thresholds (50%, 75%, 90%, 100%) to prevent overspending.

## Persona: Solo Researcher
**Priority**: 🔴 Critical | **Phase**: v0.6.0

## Feature
\`\`\`bash
prism budget set 500 --alert-at 50,75,90,100
# Email sent at each threshold with remaining budget
\`\`\`

**Related**: #137, #136"
CREATED=$((CREATED + 1))

gh issue create --title "💵 Pre-Launch Cost Preview & Confirmation" --label "enhancement" --body "## Problem
Researchers need to see total cost impact before launching expensive instances.

## Persona: Solo Researcher
**Priority**: 🟡 High | **Phase**: v0.6.0

## Feature
\`\`\`bash
prism launch gpu-workstation name
# Shows: Estimated daily cost: \$73.44, proceed? (y/N)
\`\`\`

**Related**: #137, #136"
CREATED=$((CREATED + 1))

gh issue create --title "📊 Budget Forecasting & Affordability Tool" --label "enhancement" --body "## Problem
Researchers need \"can I afford this?\" tool to plan workspace usage.

## Persona: Solo Researcher
**Priority**: 🟡 High | **Phase**: v0.6.1

## Feature
\`\`\`bash
prism budget forecast --workspace-count 3 --days 7
# Shows: Can afford 3 workspaces for 4.2 days (budget exhausted Oct 15)
\`\`\`

**Related**: #137, #136"
CREATED=$((CREATED + 1))

gh issue create --title "📈 Monthly Budget Reports & PDF Export" --label "enhancement" --body "## Problem
Researchers need monthly cost reports for grant administrators.

## Persona: Solo Researcher
**Priority**: 🟡 High | **Phase**: v0.6.1

## Feature
\`\`\`bash
prism budget report --month september --pdf
# Generates detailed cost breakdown with charts
\`\`\`

**Related**: #137, #136"
CREATED=$((CREATED + 1))

gh issue create --title "💡 Cost Optimization Recommendations Engine" --label "enhancement" --body "## Problem
Researchers don't know about cheaper alternatives (spot instances, reserved capacity).

## Persona: Solo Researcher
**Priority**: 🟢 Medium | **Phase**: v0.6.2

## Feature
\`\`\`bash
prism optimize analyze
# Suggests: Switch to spot instances → save 70% (\$210/month)
\`\`\`

**Related**: #137"
CREATED=$((CREATED + 1))

gh issue create --title "🔄 Budget Rollover to Next Month" --label "enhancement" --body "## Problem
Unused monthly budget should roll over to next month for grant-funded researchers.

## Persona: Solo Researcher
**Priority**: 🟢 Medium | **Phase**: v0.6.2

## Feature
\`\`\`bash
prism budget set 500 --rollover
# Unused \$50 from September → October budget = \$550
\`\`\`

**Related**: #137"
CREATED=$((CREATED + 1))

gh issue create --title "📅 Multi-Month Budget Allocation for Grants" --label "enhancement" --body "## Problem
Grant-funded researchers receive lump sums (e.g., \$6000 for 6 months) and need multi-month budgeting.

## Persona: Solo Researcher
**Priority**: 🟢 Medium | **Phase**: v0.7.0

## Feature
\`\`\`bash
prism budget set 6000 --months 6 --start-date 2026-01-01
# Tracks spending across 6-month grant period
\`\`\`

**Related**: #137"
CREATED=$((CREATED + 1))

gh issue create --title "🤝 Budget Sharing Between Researchers" --label "enhancement" --body "## Problem
PIs need to share budget allocations with postdocs and students.

## Persona: Solo Researcher + Lab Environment
**Priority**: 🟢 Medium | **Phase**: v0.7.0

## Feature
\`\`\`bash
prism budget share postdoc@university.edu --amount 50
# Transfers \$50 from PI budget to postdoc
\`\`\`

**Related**: #137, Lab Environment features"
CREATED=$((CREATED + 1))

gh issue create --title "⏱️ Time-Boxed Workspace Launches" --label "enhancement" --body "## Problem
Researchers want workspaces to auto-terminate after fixed time (e.g., 8-hour analysis run).

## Persona: Solo Researcher + Workshop
**Priority**: 🟢 Medium | **Phase**: v0.7.0

## Feature
\`\`\`bash
prism launch template name --hours 8
# Workspace auto-terminates after 8 hours
\`\`\`

**Related**: #135 (auto-terminate), #137"
CREATED=$((CREATED + 1))

gh issue create --title "🔍 Cost Optimization Usage Pattern Analyzer" --label "enhancement" --body "## Problem
Researchers don't know if their usage patterns are cost-efficient.

## Persona: Solo Researcher
**Priority**: 🟢 Low | **Phase**: v0.7.1

## Feature
\`\`\`bash
prism optimize analyze-usage
# Analyzes: You use GPU workspaces for 2hrs/day → spot instances save 65%
\`\`\`

**Related**: #137"
CREATED=$((CREATED + 1))

# LAB ENVIRONMENT GAPS (12 issues)

gh issue create --title "🏛️ Hierarchical Sub-Budget Delegation" --label "enhancement" --body "## Problem
Lab PIs need to allocate sub-budgets to postdocs, who then allocate to students.

## Persona: Lab Environment
**Priority**: 🔴 Critical | **Phase**: v0.7.0

## Feature
\`\`\`bash
# PI allocates to postdoc
prism budget allocate postdoc@lab.edu --amount 200

# Postdoc allocates to students
prism budget allocate student1@lab.edu --amount 50
\`\`\`

**Related**: #137, Lab collaboration"
CREATED=$((CREATED + 1))

gh issue create --title "✋ Approval Workflows for GPU/Expensive Resources" --label "enhancement" --body "## Problem
Lab PIs need request/approve/deny workflow for expensive GPU instances.

## Persona: Lab Environment
**Priority**: 🔴 Critical | **Phase**: v0.7.0

## Feature
\`\`\`bash
# Student requests GPU
prism request gpu-workstation --reason \"Training neural network\"

# PI reviews and approves
prism approvals list
prism approve <request-id>
\`\`\`

**Related**: Lab management, budget controls"
CREATED=$((CREATED + 1))

gh issue create --title "⏳ Time-Boxed Collaborator Access with Auto-Revoke" --label "enhancement" --body "## Problem
Lab collaborations need automatic access expiry with pre-expiry warnings.

## Persona: Lab Environment + Cross-Institutional
**Priority**: 🟡 High | **Phase**: v0.7.0

## Feature
\`\`\`bash
prism collab add researcher@partner.edu --expires 90d
# Auto-revoke after 90 days, warnings at 30d/7d/1d
\`\`\`

**Related**: Cross-institutional collaboration"
CREATED=$((CREATED + 1))

gh issue create --title "📏 Resource Quotas by Role (Admin/Member/Viewer)" --label "enhancement" --body "## Problem
Lab admins need role-based quotas: students limited to 2 instances, postdocs unlimited.

## Persona: Lab Environment + University Class
**Priority**: 🟡 High | **Phase**: v0.7.0

## Feature
\`\`\`bash
prism role set student --max-instances 2 --max-daily-cost 10
prism role set postdoc --max-instances unlimited
\`\`\`

**Related**: Role-based access control"
CREATED=$((CREATED + 1))

gh issue create --title "📅 Grant Period Management & Auto-Freeze" --label "enhancement" --body "## Problem
Labs need grant period tracking with auto-freeze when grant ends.

## Persona: Lab Environment
**Priority**: 🟡 High | **Phase**: v0.7.0

## Feature
\`\`\`bash
prism project create grant-r01 --start 2026-01-01 --end 2026-12-31
# Auto-freeze project on 2026-12-31, generate final report
\`\`\`

**Related**: Project lifecycle management"
CREATED=$((CREATED + 1))

gh issue create --title "📊 Centralized Approval Dashboard for PIs" --label "enhancement" --body "## Problem
PIs need dashboard to see all pending requests across lab members.

## Persona: Lab Environment
**Priority**: 🟢 Medium | **Phase**: v0.7.1

## Feature
\`\`\`bash
prism approvals dashboard
# Shows: 3 pending GPU requests, 2 budget increase requests
\`\`\`

**Related**: Approval workflows, TUI/GUI integration"
CREATED=$((CREATED + 1))

gh issue create --title "🎓 New Member Onboarding Templates" --label "enhancement" --body "## Problem
Labs need template-based provisioning for new members (workspace + budget + access).

## Persona: Lab Environment
**Priority**: 🟢 Medium | **Phase**: v0.7.1

## Feature
\`\`\`bash
prism member add newstudent@lab.edu --template grad-student
# Auto-provisions: \$50 budget, 2 instance limit, data access
\`\`\`

**Related**: User onboarding, templates"
CREATED=$((CREATED + 1))

gh issue create --title "💸 Dynamic Budget Reallocation Interface" --label "enhancement" --body "## Problem
Lab budgets shift mid-semester and need easy reallocation.

## Persona: Lab Environment
**Priority**: 🟢 Medium | **Phase**: v0.7.1

## Feature
\`\`\`bash
prism budget reallocate --from postdoc1 --to postdoc2 --amount 100
# Moves \$100 allocation between members
\`\`\`

**Related**: Budget management, lab flexibility"
CREATED=$((CREATED + 1))

gh issue create --title "🤝 Cross-Project Budget Sharing (\"Borrow from Discretionary\")" --label "enhancement" --body "## Problem
Labs need temporary budget sharing between projects for urgent work.

## Persona: Lab Environment
**Priority**: 🟢 Medium | **Phase**: v0.7.2

## Feature
\`\`\`bash
prism budget borrow --from discretionary --to urgent-analysis --amount 200 --repay-by 2026-11-30
\`\`\`

**Related**: Budget flexibility"
CREATED=$((CREATED + 1))

gh issue create --title "🚨 Emergency Budget Overage with PI Approval" --label "enhancement" --body "## Problem
Critical analysis needs exceed budget and require emergency PI approval.

## Persona: Lab Environment
**Priority**: 🟢 Medium | **Phase**: v0.7.2

## Feature
\`\`\`bash
prism budget request-emergency --amount 500 --reason \"Critical deadline\"
# PI receives urgent approval request
\`\`\`

**Related**: Approval workflows, budget management"
CREATED=$((CREATED + 1))

gh issue create --title "🔍 Compliance Audit Trail for Labs" --label "enhancement" --body "## Problem
Labs need compliance-ready audit logs for NIH/NSF reporting.

## Persona: Lab Environment + NIH Compliance
**Priority**: 🟢 Medium | **Phase**: v0.8.0

## Feature
\`\`\`bash
prism audit report --format nih --year 2026
# Generates: User access logs, cost attribution, data access
\`\`\`

**Related**: Compliance, NIH requirements"
CREATED=$((CREATED + 1))

gh issue create --title "📈 Lab Usage Analytics Dashboard" --label "enhancement" --body "## Problem
Lab PIs need visibility into member usage patterns and efficiency.

## Persona: Lab Environment
**Priority**: 🟢 Low | **Phase**: v0.8.1

## Feature
\`\`\`bash
prism lab analytics
# Shows: Member activity, cost efficiency, underutilized resources
\`\`\`

**Related**: Lab management, cost optimization"
CREATED=$((CREATED + 1))

# UNIVERSITY CLASS GAPS (16 issues)

gh issue create --title "🔧 Admin Support Access for Classes & Workshops (dedicated support user)" --label "enhancement" --body "## Problem
Instructors/TAs/workshop leaders need privileged access to participant workspaces for troubleshooting, with full audit logging. Applies to both University Classes and Conference Workshops.

## Persona: University Class + Conference Workshop
**Priority**: 🔴 Critical | **Phase**: v0.8.0

## Feature

### Dedicated Support User System

Each class/workshop gets a dedicated \`support\` user account with elevated privileges:

\`\`\`bash
# Create class with support access enabled
prism class create CS229 --enable-support-access

# Support user automatically created: support@cs229.prism
# Credentials provided to instructor/TAs

# Instructor grants TA support access
prism support grant ta@university.edu --class CS229

# TA accesses student workspace via support user
prism support access student@uni.edu workspace-name --reason \"Help with numpy error\"
# → SSH session as 'support' user with sudo, all commands logged

# All support access logged with full audit trail
prism audit support-access --user student@uni.edu --class CS229
# Shows: Who accessed, when, what commands, duration, reason

# Workshop variant:
prism workshop create neurips-dl --enable-support-access
prism support access participant@conference.org workshop-instance --reason \"Connection issue\"
\`\`\`

### Technical Implementation

1. **Dedicated Support User** on each workspace:
   - Username: \`support\`
   - Sudo access for system fixes
   - SSH key managed by Prism
   - No home directory persistence (ephemeral)

2. **Access Control**:
   - Instructor designates who has support access (TAs, co-instructors)
   - Support users log access reason (required)
   - Time-limited sessions (e.g., 30min max)

3. **Audit Logging**:
   - Every support command logged to central audit log
   - Academic integrity compliance (plagiarism investigations)
   - Accessible to professor for review

4. **Permissions**:
   - View all files
   - Execute commands as any user (sudo)
   - Modify system configuration
   - Cannot delete audit logs

**Related**: Academic integrity, workshop support, debugging, audit logging

**Use Cases**:
- University Class: TA helps student debug broken environment
- Workshop: Instructor fixes participant's connection issue during live event
- Lab: PI troubleshoots postdoc's workspace remotely

**Security**: Full audit trail prevents abuse, reason required for compliance"
CREATED=$((CREATED + 1))

gh issue create --title "📚 Template Whitelisting for Class Assignments" --label "enhancement" --body "## Problem
Professors need to restrict students to assignment-specific templates only.

## Persona: University Class
**Priority**: 🔴 Critical | **Phase**: v0.8.0

## Feature
\`\`\`bash
prism class create CS229 --allowed-templates \"Python Data Science\"
# Students can ONLY launch whitelisted template
\`\`\`

**Related**: Academic control, cost management"
CREATED=$((CREATED + 1))

gh issue create --title "🎓 Auto Semester-End Cleanup & Archiving" --label "enhancement" --body "## Problem
Classes need automatic workspace cleanup when semester ends.

## Persona: University Class
**Priority**: 🔴 Critical | **Phase**: v0.8.0

## Feature
\`\`\`bash
prism class create CS229 --end-date 2026-05-15
# Auto-cleanup workspaces on end date, preserve student data for 30 days
\`\`\`

**Related**: Course lifecycle, data preservation"
CREATED=$((CREATED + 1))

gh issue create --title "💰 Per-Student Budget Isolation & Enforcement" --label "enhancement" --body "## Problem
Each student gets fixed budget (\$20/semester) - prevent exceeding individual allocation.

## Persona: University Class
**Priority**: 🔴 Critical | **Phase**: v0.8.0

## Feature
\`\`\`bash
prism class budget --per-student 20
# Each student gets isolated \$20 budget, hard cap enforced
\`\`\`

**Related**: #137 (budget caps), academic fairness"
CREATED=$((CREATED + 1))

gh issue create --title "🔄 Student Workspace Reset & Backup" --label "enhancement" --body "## Problem
Students break their environments and need fresh start with backup of old work.

## Persona: University Class
**Priority**: 🔴 Critical | **Phase**: v0.8.0

## Feature
\`\`\`bash
# TA resets broken student workspace
prism student reset student@uni.edu workspace-name --backup
# Creates snapshot backup, launches fresh workspace
\`\`\`

**Related**: Student support, TA tools"
CREATED=$((CREATED + 1))

gh issue create --title "🔒 Academic Integrity Audit Logs" --label "enhancement" --body "## Problem
Professors need audit logs for plagiarism investigations (SSH sessions, file access).

## Persona: University Class
**Priority**: 🔴 Critical | **Phase**: v0.8.0

## Feature
\`\`\`bash
prism audit academic-integrity --student student@uni.edu --start 2026-03-01
# Shows: SSH sessions, file modifications, TA access
\`\`\`

**Related**: Compliance, TA access logging"
CREATED=$((CREATED + 1))

gh issue create --title "📊 Bulk Student Import from Canvas/Blackboard" --label "enhancement" --body "## Problem
Professors need to import 50+ students from LMS CSV roster.

## Persona: University Class
**Priority**: 🟡 High | **Phase**: v0.8.0

## Feature
\`\`\`bash
prism class import-students --csv canvas-roster.csv
# Auto-creates accounts, assigns budgets, provisions workspaces
\`\`\`

**Related**: LMS integration, student onboarding"
CREATED=$((CREATED + 1))

gh issue create --title "📁 Shared Course Materials on Read-Only EFS" --label "enhancement" --body "## Problem
Professors need read-only shared storage for datasets/assignments (all students access).

## Persona: University Class
**Priority**: 🟡 High | **Phase**: v0.8.0

## Feature
\`\`\`bash
prism class storage create course-materials --read-only
# Mounted at /course-materials on all student workspaces
\`\`\`

**Related**: Storage management, course delivery"
CREATED=$((CREATED + 1))

gh issue create --title "📊 TA Dashboard for Student Progress Monitoring" --label "enhancement" --body "## Problem
TAs need centralized view of all student workspaces and progress.

## Persona: University Class
**Priority**: 🟢 Medium | **Phase**: v0.8.1

## Feature
\`\`\`bash
prism ta dashboard
# Shows: 50 students, 3 need help, 12 idle workspaces, budget status
\`\`\`

**Related**: TA tools, TUI integration"
CREATED=$((CREATED + 1))

gh issue create --title "📈 Grade Correlation Analytics (Usage vs Performance)" --label "enhancement" --body "## Problem
Professors want to see correlation between workspace usage and student grades.

## Persona: University Class
**Priority**: 🟢 Medium | **Phase**: v0.8.1

## Feature
\`\`\`bash
prism class analytics --grades grades.csv
# Shows: Students with <5hr usage scored 15% lower on average
\`\`\`

**Related**: Academic research, educational analytics"
CREATED=$((CREATED + 1))

gh issue create --title "🎓 Canvas/Blackboard Grade Passback Integration" --label "enhancement" --body "## Problem
Automated assignment completion detection → LMS grade submission.

## Persona: University Class
**Priority**: 🟢 Low | **Phase**: v0.9.0

## Feature
\`\`\`bash
prism class setup-grading --lms canvas --assignment-id 12345
# Detects notebook completion, submits grade to Canvas
\`\`\`

**Related**: LMS integration, automated grading"
CREATED=$((CREATED + 1))

gh issue create --title "💬 In-Workspace TA Messaging System" --label "enhancement" --body "## Problem
TAs need to send messages directly to student workspaces (appears in terminal/Jupyter).

## Persona: University Class
**Priority**: 🟢 Low | **Phase**: v0.9.0

## Feature
\`\`\`bash
prism student message student@uni.edu \"Office hours today 3-5 PM\"
# Message appears in student terminal on next login
\`\`\`

**Related**: Student communication, TA tools"
CREATED=$((CREATED + 1))

gh issue create --title "🔄 Automated Student Workspace Provisioning" --label "enhancement" --body "## Problem
When student joins class, auto-provision workspace with course template.

## Persona: University Class
**Priority**: 🟢 Medium | **Phase**: v0.8.0

## Feature
\`\`\`bash
prism class auto-provision --template \"CS229 Python\" --budget 20
# New students get workspace immediately on roster addition
\`\`\`

**Related**: Student onboarding, automation"
CREATED=$((CREATED + 1))

gh issue create --title "📊 Student Usage Report for Professors" --label "enhancement" --body "## Problem
Professors need semester-end report of student workspace usage for assessment.

## Persona: University Class
**Priority**: 🟢 Medium | **Phase**: v0.8.1

## Feature
\`\`\`bash
prism class report --semester fall-2026 --pdf
# Generates: Usage by student, completion rates, cost analysis
\`\`\`

**Related**: Academic reporting"
CREATED=$((CREATED + 1))

gh issue create --title "🎓 Assignment Deadline Integration" --label "enhancement" --body "## Problem
Workspaces should warn students about approaching assignment deadlines.

## Persona: University Class
**Priority**: 🟢 Low | **Phase**: v0.9.0

## Feature
\`\`\`bash
prism class assignment add --due 2026-10-15 --workspace-template assignment-3
# Workspace shows countdown, reminds student of deadline
\`\`\`

**Related**: LMS integration"
CREATED=$((CREATED + 1))

gh issue create --title "📝 Plagiarism Detection via File Access Patterns" --label "enhancement" --body "## Problem
Detect suspicious activity: multiple students accessing same files simultaneously.

## Persona: University Class
**Priority**: 🟢 Low | **Phase**: v0.9.0

## Feature
\`\`\`bash
prism audit suspicious-activity
# Flags: 3 students modified identical files within 5 minutes
\`\`\`

**Related**: Academic integrity"
CREATED=$((CREATED + 1))

# CONFERENCE WORKSHOP GAPS (9 remaining - #135 already created)

gh issue create --title "🔐 Template Whitelisting in Workshop Invitations" --label "enhancement" --body "## Problem
Workshop participants should ONLY launch whitelisted template (prevent expensive GPU launches).

## Persona: Conference Workshop
**Priority**: 🔴 Critical | **Phase**: v0.7.0

## Feature
\`\`\`bash
prism workshop invitations create --template-whitelist \"PyTorch ML\" --max-type t3.medium
# Participants blocked from launching other templates
\`\`\`

**Related**: #135 (auto-terminate), workshop cost control"
CREATED=$((CREATED + 1))

gh issue create --title "🔒 Policy-Restricted Workshop Invitations" --label "enhancement" --body "## Problem
Workshop invitations need embedded policy restrictions (templates, instance types, costs).

## Persona: Conference Workshop
**Priority**: 🔴 Critical | **Phase**: v0.7.0

## Feature
\`\`\`bash
prism invitations create-workshop \\
  --template-whitelist \"PyTorch ML\" \\
  --max-instance-type t3.medium \\
  --max-hourly-cost 0.10 \\
  --auto-terminate-hours 6
\`\`\`

**Related**: #135, invitation system"
CREATED=$((CREATED + 1))

gh issue create --title "🚀 Bulk Workspace Pre-Provisioning for Workshops" --label "enhancement" --body "## Problem
Workshop needs all 60 workspaces ready 15 minutes before start time.

## Persona: Conference Workshop
**Priority**: 🟡 High | **Phase**: v0.7.1

## Feature
\`\`\`bash
prism workshop provision --count 60 --start-time \"2026-12-08T08:45:00\"
# All workspaces launch automatically at specified time
\`\`\`

**Related**: #135 (auto-terminate), workshop logistics"
CREATED=$((CREATED + 1))

gh issue create --title "📊 Live Workshop Dashboard (Participant Monitoring)" --label "enhancement" --body "## Problem
Workshop instructors need real-time view of all participant workspaces during event.

## Persona: Conference Workshop
**Priority**: 🟡 High | **Phase**: v0.7.1

## Feature
\`\`\`bash
prism workshop dashboard --live
# Shows: 58/60 active, 2 stopped, budget status, time remaining
\`\`\`

**Related**: #135, real-time monitoring, TUI"
CREATED=$((CREATED + 1))

gh issue create --title "📥 Bulk Participant Work Download/Export" --label "enhancement" --body "## Problem
Workshop instructors need to preserve all participant notebooks after event ends.

## Persona: Conference Workshop
**Priority**: 🟢 Medium | **Phase**: v0.7.2

## Feature
\`\`\`bash
prism workshop export-all workshop-id --output-dir ./participant-work/
# Downloads all participant notebooks as ZIP files
\`\`\`

**Related**: Data preservation"
CREATED=$((CREATED + 1))

gh issue create --title "🧪 Pre-Workshop Testing Period (24h Early Access)" --label "enhancement" --body "## Problem
Workshop participants need 24-hour testing window before event to catch issues.

## Persona: Conference Workshop
**Priority**: 🟢 Medium | **Phase**: v0.7.2

## Feature
\`\`\`bash
prism workshop early-access enable --duration 24h
# Participants can test environment day before workshop
\`\`\`

**Related**: Workshop preparation"
CREATED=$((CREATED + 1))

gh issue create --title "💾 Snapshot Quick-Save for Participants" --label "enhancement" --body "## Problem
Workshop participants need one-click save of their work before auto-termination.

## Persona: Conference Workshop
**Priority**: 🟢 Medium | **Phase**: v0.7.2

## Feature
\`\`\`bash
# Before auto-terminate, prompt participant:
# Save your work? (y/N): y
prism snapshot create workshop-instance my-workshop-work
\`\`\`

**Related**: #135 (auto-terminate), data preservation"
CREATED=$((CREATED + 1))

gh issue create --title "📋 Workshop Configuration Templates" --label "enhancement" --body "## Problem
Workshop instructors need reusable workshop configurations for recurring events.

## Persona: Conference Workshop
**Priority**: 🟢 Medium | **Phase**: v0.8.0

## Feature
\`\`\`bash
prism workshop template save neurips-dl-2026
# Next year: prism workshop create --from-template neurips-dl-2026
\`\`\`

**Related**: Workshop reusability"
CREATED=$((CREATED + 1))

gh issue create --title "📈 Participant Progress Tracking During Workshop" --label "enhancement" --body "## Problem
Workshop instructors need to identify struggling participants during live event.

## Persona: Conference Workshop
**Priority**: 🟢 Medium | **Phase**: v0.7.1

## Feature
\`\`\`bash
prism workshop progress
# Shows: 12 high engagement, 3 stuck on section 2, 1 idle
\`\`\`

**Related**: Live monitoring, participant support"
CREATED=$((CREATED + 1))

# CROSS-INSTITUTIONAL COLLABORATION GAPS (9 issues)

gh issue create --title "🌐 Cross-Account EFS Access via Access Points" --label "enhancement" --body "## Problem
Multi-institutional projects need shared EFS storage across different AWS accounts.

## Persona: Cross-Institutional Collaboration
**Priority**: 🔴 Critical | **Phase**: v0.8.0

## Feature
\`\`\`bash
prism storage share --with partner-account@aws --type efs --access-point ap-123
# Partner mounts shared EFS from their AWS account
\`\`\`

**Related**: Multi-account AWS, data sharing"
CREATED=$((CREATED + 1))

gh issue create --title "💰 Cost Attribution by Collaborator" --label "enhancement" --body "## Problem
Multi-institution projects need to track which collaborator launched what for chargeback.

## Persona: Cross-Institutional Collaboration
**Priority**: 🔴 Critical | **Phase**: v0.8.0

## Feature
\`\`\`bash
prism project costs --by-user
# Shows: Institution A: \$1,250, Institution B: \$875, Institution C: \$425
\`\`\`

**Related**: Cost tracking, chargeback"
CREATED=$((CREATED + 1))

gh issue create --title "🔐 Invitation Policy Restrictions (Template/Instance Limits)" --label "enhancement" --body "## Problem
Cross-institutional invitations need embedded restrictions (templates, instance types, max cost).

## Persona: Cross-Institutional Collaboration + Workshop
**Priority**: 🔴 Critical | **Phase**: v0.8.0

## Feature
\`\`\`bash
prism invitations create \\
  --template-whitelist \"Shared Analysis\" \\
  --max-instance-type m5.xlarge \\
  --max-daily-cost 50
\`\`\`

**Related**: Policy enforcement, invitation system"
CREATED=$((CREATED + 1))

gh issue create --title "🔍 Collaboration Audit Trail for Compliance" --label "enhancement" --body "## Problem
Multi-institutional projects need compliance audit logs for all collaborator actions.

## Persona: Cross-Institutional Collaboration
**Priority**: 🟡 High | **Phase**: v0.8.1

## Feature
\`\`\`bash
prism audit collaboration --project multi-site-grant --year 2026
# Generates: User access, data access, cost attribution, compliance report
\`\`\`

**Related**: Compliance, NIH reporting"
CREATED=$((CREATED + 1))

gh issue create --title "⏰ Graceful Collaboration End (Warnings + Work Preservation)" --label "enhancement" --body "## Problem
Collaborations end abruptly - need 30d/7d/1d warnings and data export tools.

## Persona: Cross-Institutional Collaboration
**Priority**: 🟡 High | **Phase**: v0.8.1

## Feature
\`\`\`bash
prism collaboration end --project collab-2026 --date 2026-12-31
# Auto-sends warnings 30d/7d/1d before, provides export tools
\`\`\`

**Related**: Collaboration lifecycle"
CREATED=$((CREATED + 1))

gh issue create --title "💸 Automated Chargeback to Collaborator Accounts" --label "enhancement" --body "## Problem
Multi-institutional projects need monthly invoices sent to each institution's billing.

## Persona: Cross-Institutional Collaboration
**Priority**: 🟢 Medium | **Phase**: v0.8.2

## Feature
\`\`\`bash
prism project chargeback setup --institution-a billing@institution-a.edu
# Auto-sends monthly invoices with cost breakdown
\`\`\`

**Related**: Cost attribution, institutional billing"
CREATED=$((CREATED + 1))

gh issue create --title "💾 Cross-Account Snapshot Transfer for Work Preservation" --label "enhancement" --body "## Problem
Collaborators need to transfer workspace snapshots to their home institution's AWS account.

## Persona: Cross-Institutional Collaboration
**Priority**: 🟢 Medium | **Phase**: v0.8.2

## Feature
\`\`\`bash
prism snapshot transfer my-workspace --to-account partner-aws-account
# Copies AMI snapshot to partner's AWS account
\`\`\`

**Related**: Data portability, cross-account AWS"
CREATED=$((CREATED + 1))

gh issue create --title "📊 Collaboration Health Dashboard (Proactive Monitoring)" --label "enhancement" --body "## Problem
Multi-site projects need proactive monitoring: budget trends, usage patterns, expiry warnings.

## Persona: Cross-Institutional Collaboration
**Priority**: 🟢 Medium | **Phase**: v0.8.2

## Feature
\`\`\`bash
prism collaboration health --project multi-site
# Shows: Budget on track, expiring in 45 days, Institution B inactive
\`\`\`

**Related**: Project health, monitoring"
CREATED=$((CREATED + 1))

gh issue create --title "📋 Multi-Site Project Templates for Faster Setup" --label "enhancement" --body "## Problem
Recurring multi-institutional projects need reusable setup templates.

## Persona: Cross-Institutional Collaboration
**Priority**: 🟢 Medium | **Phase**: v0.8.3

## Feature
\`\`\`bash
prism project create --from-template multi-site-collab-2025
# Reuses: Budget structure, invitation policies, access controls
\`\`\`

**Related**: Project templates, reusability"
CREATED=$((CREATED + 1))

echo ""
echo "✅ Created $CREATED GitHub issues for persona feature gaps"
echo ""
echo "Summary by persona:"
echo "  Solo Researcher: 10 issues"
echo "  Lab Environment: 12 issues"
echo "  University Class: 16 issues"
echo "  Conference Workshop: 9 issues"
echo "  Cross-Institutional: 9 issues"
echo "  Total: 56 new issues (+ 1 already created = 57 total)"
echo ""
echo "View all issues: https://github.com/scttfrdmn/prism/issues"
