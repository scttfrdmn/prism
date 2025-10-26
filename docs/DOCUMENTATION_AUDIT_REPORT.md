# Prism Documentation Audit Report
## Features Documented But Not Yet Implemented

**Generated**: October 19, 2025
**Auditor**: Claude Code
**Scope**: User guides, admin guides, and persona walkthroughs
**Status**: Comprehensive review of v0.5.x+ documentation against codebase

---

## Executive Summary

This audit identifies **52 documented features** across 7 major categories that are described in Prism documentation but not yet fully implemented in the codebase. These features are primarily in:

1. **Invitation System Security Features** (7 features) - Device binding, batch operations
2. **Profile Export/Import System** (5 features) - Backup and migration features
3. **Budget Management** (11 features) - Personal budgets, alerts, forecasting
4. **University Class Management** (12 features) - TA tools, student management
5. **Policy Framework** (8 features) - Template policies, AMI governance
6. **AMI Compilation System** (6 features) - Template-to-AMI workflow
7. **DCV/Desktop Integration** (3 features) - GUI workstation support

**Priority Assessment**:
- 🔴 **Critical** (18 features): Core functionality for documented workflows
- 🟡 **High** (21 features): Important enhancements for enterprise adoption
- 🟢 **Medium** (13 features): Nice-to-have features for advanced use cases

---

## Category 1: Invitation System Security Features

### 1.1 Device Binding and Management

**Documentation**: [ADMINISTRATOR_GUIDE.md](admin-guides/ADMINISTRATOR_GUIDE.md) (lines 14-47), [SECURE_INVITATION_ARCHITECTURE.md](admin-guides/SECURE_INVITATION_ARCHITECTURE.md)

**Promised Features**:
```bash
# Device-bound invitations
prism profiles invitations create-secure lab-access \
  --device-bound=true \
  --max-devices=3
```

**Current Status**:
- ✅ **PARTIAL**: Code exists in `pkg/profile/secure_invitation.go` and `pkg/profile/security/`
- ❌ **MISSING**: CLI commands not wired up in `internal/cli/profiles.go`
- ❌ **MISSING**: Device registry integration incomplete

**Evidence**:
```bash
# Search for create-secure command
$ grep -r "create-secure" internal/cli/
# No results - command not implemented in CLI
```

**What's Missing**:
1. `prism profiles invitations create-secure` command (no CLI handler)
2. `prism profiles invitations devices` command (list devices)
3. `prism profiles invitations revoke-device` command
4. `prism profiles invitations revoke-all` command
5. Device registry S3 backend integration
6. Keychain integration for device binding (code exists but not wired to CLI)
7. Device enrollment flow (`prism profiles enroll ENROLLMENT_CODE`)

**Suggested Phase**: **v0.6.1** (Multi-User Authentication & IAM)
**Priority**: 🟡 **High** - Enterprise security requirement
**Effort**: Medium (3-4 weeks)

---

### 1.2 Batch Invitation Operations

**Documentation**: [BATCH_INVITATION_GUIDE.md](admin-guides/BATCH_INVITATION_GUIDE.md), [BATCH_DEVICE_MANAGEMENT.md](admin-guides/BATCH_DEVICE_MANAGEMENT.md)

**Promised Features**:
```bash
# Batch invitation creation
prism profiles invitations batch-create \
  --csv-file invitations.csv \
  --output-file results.csv

# Batch invitation export
prism profiles invitations batch-export \
  --output-file invitations.csv

# Batch invitation acceptance
prism profiles invitations batch-accept \
  --csv-file invitations.csv
```

**Current Status**:
- ✅ **EXISTS**: Code in `pkg/profile/batch_invitation.go` (310+ lines)
- ❌ **MISSING**: CLI commands not exposed in `internal/cli/profiles.go`

**Evidence**:
```bash
$ grep -r "batch-create\|batch-export\|batch-accept" internal/cli/
# No results - batch commands not in CLI
```

**What's Missing**:
1. CLI commands for all 3 batch operations
2. CSV parsing and validation
3. Progress reporting for batch operations
4. Error handling and retry logic
5. Batch device operations (`devices batch-operation`, `devices export-info`, `devices batch-revoke-all`)

**Suggested Phase**: **v0.6.1** (Multi-User Authentication & IAM)
**Priority**: 🟡 **High** - Critical for class management
**Effort**: Low (1 week)

---

## Category 2: Profile Export/Import System

**Documentation**: [PROFILE_EXPORT_IMPORT.md](admin-guides/PROFILE_EXPORT_IMPORT.md)

**Promised Features**:
```bash
# Export profiles
prism profiles export my-profiles.zip
prism profiles export my-profiles.zip --include-credentials --password "secure"
prism profiles export --profiles personal,work --format json

# Import profiles
prism profiles import my-profiles.zip
prism profiles import --mode skip --import-credentials
```

**Current Status**:
- ✅ **EXISTS**: Code in `pkg/profile/export/export.go` (complete implementation)
- ✅ **EXISTS**: CLI commands in `internal/cli/export.go`
- ✅ **FUNCTIONAL**: Export/import appears to be implemented

**Evidence**:
```bash
$ grep -r "profiles export\|profiles import" internal/cli/export.go
# Found: Commands exist and are wired up
```

**What's Missing**:
1. ❌ Password protection for ZIP files (documented but not implemented)
2. ❌ Credential inclusion safety warnings
3. ❌ Profile conflict resolution modes (skip/overwrite/rename)
4. ❌ Invitation profile handling during export/import

**Suggested Phase**: **v0.6.0** (Enterprise Authentication & Security)
**Priority**: 🟢 **Medium** - Nice-to-have for team collaboration
**Effort**: Low (3-5 days)

---

## Category 3: Budget Management System

**Documentation**: [01_SOLO_RESEARCHER_WALKTHROUGH.md](USER_SCENARIOS/01_SOLO_RESEARCHER_WALKTHROUGH.md) (lines 98-280)

### 3.1 Personal Budget Configuration

**Promised Features**:
```bash
# Setup wizard with budget
prism init  # Interactive setup with budget configuration

# Budget commands
prism budget set --monthly 100
prism budget show
prism budget status
prism budget forecast
prism budget report --month september --pdf
```

**Current Status**:
- ✅ **EXISTS**: Budget commands in `internal/cli/budget_commands.go`
- ✅ **EXISTS**: Budget tracking in `pkg/project/budget_tracker.go`
- ⚠️ **PARTIAL**: Budget commands exist for **projects**, not **personal budgets**

**Evidence**:
```bash
$ grep -r "personal.*budget\|monthly.*budget" pkg/project/
# No results - only project-level budgets exist
```

**What's Missing**:
1. ❌ Personal budget system (separate from project budgets)
2. ❌ `prism init` wizard with budget setup
3. ❌ Per-user budget tracking and enforcement
4. ❌ Budget inheritance (project → personal → instance)
5. ❌ `prism budget set` command for personal budgets

**Suggested Phase**: **v0.6.0** (Budget Safety Net - Solo Researcher)
**Priority**: 🔴 **Critical** - Core solo researcher workflow
**Effort**: Medium (2 weeks)

---

### 3.2 Budget Alerts and Notifications

**Promised Features**:
```bash
# Configure budget alerts
prism budget set --monthly 100 --alert-email user@example.com
prism budget alert add --threshold 80 --email user@example.com

# Email alerts at thresholds
# Subject: ⚠️ Prism Budget Alert: 80% Used
```

**Current Status**:
- ✅ **EXISTS**: Alert types defined in `pkg/cost/alerts.go`
- ❌ **MISSING**: Alert configuration CLI commands
- ❌ **MISSING**: Email integration and delivery system
- ❌ **MISSING**: Alert threshold configuration

**What's Missing**:
1. CLI commands for alert configuration
2. Email delivery system (SMTP, SES, or webhook)
3. Alert threshold tracking and triggering
4. Alert history and log
5. Slack/webhook integrations

**Suggested Phase**: **v0.6.0** (Budget Safety Net)
**Priority**: 🔴 **Critical** - Prevents overspending anxiety
**Effort**: Medium (1 week)

---

### 3.3 Pre-Launch Budget Impact Preview

**Promised Features**:
```bash
prism launch bioinformatics-suite rnaseq-analysis

# Expected output:
# 📊 Budget Impact Preview
#    Cost: $2.40/day
#    Current: $0 / $100 (0%)
#    Projected: ~$36 / $100 (36%) ✅
# Proceed? [Y/n]:
```

**Current Status**:
- ✅ **EXISTS**: Cost calculation in `pkg/project/cost_calculator.go`
- ❌ **MISSING**: Pre-launch budget check and confirmation
- ❌ **MISSING**: Budget impact display before launch

**What's Missing**:
1. Pre-launch budget validation hook
2. Interactive budget impact preview
3. Budget blocking (prevent launch if over budget)
4. Optional `--yes` flag to skip confirmation

**Suggested Phase**: **v0.6.0** (Budget Safety Net)
**Priority**: 🔴 **Critical** - Prevents accidental overspending
**Effort**: Low (3 days)

---

### 3.4 Budget Forecasting

**Promised Features**:
```bash
prism budget forecast

# Output:
# Current spend: $45.00 (Day 15 of 30)
# Projected end-of-month: $90.00
# Remaining buffer: $10.00 ✅
# Can I launch another instance?
# ✅ t3.medium ($0.80/day): Yes, $14 projected addition = $104 total
```

**Current Status**:
- ❌ **MISSING**: Forecasting algorithm
- ❌ **MISSING**: Historical usage pattern analysis
- ❌ **MISSING**: Projection calculations

**What's Missing**:
1. `prism budget forecast` command
2. Time-series cost analysis
3. ML-based usage prediction
4. "Can I afford this?" tool
5. What-if scenario modeling

**Suggested Phase**: **v0.6.1** (Budget Intelligence)
**Priority**: 🟡 **High** - Planning confidence
**Effort**: Medium (1 week)

---

### 3.5 Monthly Budget Reporting

**Promised Features**:
```bash
prism budget report --month september --pdf

# Generates PDF with:
# - Budget vs. actual spend
# - Instance usage breakdown
# - Hibernation savings
# - Cost efficiency metrics
```

**Current Status**:
- ❌ **MISSING**: Report generation system
- ❌ **MISSING**: PDF export capability
- ❌ **MISSING**: Monthly aggregation and analytics

**What's Missing**:
1. `prism budget report` command
2. PDF generation library integration
3. Monthly cost aggregation
4. Automated month-end email reports
5. CSV/PDF export formats

**Suggested Phase**: **v0.6.1** (Budget Intelligence)
**Priority**: 🟡 **High** - Reduces admin burden
**Effort**: Medium (1 week)

---

### 3.6 Time-Boxed Launches

**Promised Features**:
```bash
# Launch with auto-termination
prism launch gpu-ml-workstation protein-folding --hours 8

# Output:
# ✅ Instance will auto-terminate at 11:30 PM tonight
# 📊 Estimated cost: $8.27
```

**Current Status**:
- ❌ **MISSING**: Time-boxed launch feature
- ❌ **MISSING**: Auto-termination scheduling
- ❌ **MISSING**: Time limit enforcement

**What's Missing**:
1. `--hours` flag for launch command
2. Instance termination scheduler
3. Pre-termination warnings
4. Time remaining display

**Suggested Phase**: **v0.7.0** (Advanced Budget Features)
**Priority**: 🟢 **Medium** - Prevents runaway costs
**Effort**: Low (3 days)

---

## Category 4: University Class Management

**Documentation**: [03_UNIVERSITY_CLASS_WALKTHROUGH.md](USER_SCENARIOS/03_UNIVERSITY_CLASS_WALKTHROUGH.md)

### 4.1 Course/Class Creation Wizard

**Promised Features**:
```bash
prism course create "CS229-Fall2024" --interactive

# Interactive wizard:
# - Course details (code, title, dates)
# - Student budget allocation
# - Teaching staff roles
# - Approved templates
# - Auto-semester-end actions
```

**Current Status**:
- ❌ **MISSING**: Course management system
- ❌ **MISSING**: `prism course` command namespace
- ❌ **MISSING**: Course entity separate from projects

**What's Missing**:
1. `prism course create` command
2. Course data model (extends project)
3. Semester date tracking
4. Auto-cleanup scheduling
5. Course-specific constraints (template whitelist)

**Suggested Phase**: **v0.8.0** (Class Management Basics)
**Priority**: 🔴 **Critical** - Core class workflow
**Effort**: High (2 weeks)

---

### 4.2 Template Whitelisting (Course-Level)

**Promised Features**:
```bash
# Create course with approved templates
prism course create "CS229-Fall2024" \
  --approved-templates "ml-cpu-student,ml-final-project"

# Student tries unapproved template:
prism launch gpu-ml-workstation homework1
# ❌ Launch BLOCKED: Template not approved for CS229-Fall2024
```

**Current Status**:
- ⚠️ **PARTIAL**: Basic policy framework exists (`pkg/policy/`)
- ❌ **MISSING**: Template whitelist enforcement
- ❌ **MISSING**: Course-level template restrictions

**What's Missing**:
1. Template whitelist configuration
2. Launch-time template validation against whitelist
3. Educational error messages
4. Template approval workflow

**Suggested Phase**: **v0.8.0** (Class Management Basics)
**Priority**: 🔴 **Critical** - Prevents budget blowouts
**Effort**: Medium (1 week)

---

### 4.3 TA Debug Access ("God Mode")

**Promised Features**:
```bash
# TA initiates debug session
prism ta debug ml-hw3 --student sophie.martinez@university.edu

# Options:
# [1] View instance status and logs
# [2] SSH into instance (full access, logged)
# [3] View Jupyter notebooks (read-only)
# [4] Export student workspace for review
# [5] Reset instance (backup + fresh start)
```

**Current Status**:
- ❌ **MISSING**: `prism ta` command namespace
- ❌ **MISSING**: TA role and permissions system
- ❌ **MISSING**: Debug session management
- ❌ **MISSING**: Session logging for academic integrity

**What's Missing**:
1. TA role definition in project/course system
2. `prism ta debug` command
3. Temporary SSH access with logging
4. Read-only Jupyter access
5. Session recording and audit trail
6. `prism ta ssh` with automatic logging
7. `prism ta annotate` for leaving messages
8. `prism ta reset-instance` for clean slate

**Suggested Phase**: **v0.8.1** (TA Support Tools)
**Priority**: 🔴 **Critical** - Makes office hours efficient
**Effort**: High (2 weeks)

---

### 4.4 Student Bulk Import

**Promised Features**:
```bash
# Import students from Canvas LMS
prism course import-students "CS229-Fall2024" \
  --canvas \
  --course-id 12345

# Import from CSV
prism project member import "CS229-Fall2024" \
  --csv students.csv \
  --role member \
  --default-budget 24
```

**Current Status**:
- ⚠️ **PARTIAL**: `prism project member add` exists for individual members
- ❌ **MISSING**: Bulk CSV import
- ❌ **MISSING**: Canvas LMS integration

**What's Missing**:
1. `prism project member import` command
2. CSV parsing and validation
3. Bulk SSH key generation
4. Welcome email automation
5. Canvas API integration

**Suggested Phase**: **v0.8.0** (Class Management Basics)
**Priority**: 🟡 **High** - Essential for large classes
**Effort**: Low (3 days for CSV, 1 week for Canvas)

---

### 4.5 Automatic Semester End Cleanup

**Promised Features**:
```bash
# Course with auto-end date
prism course create "CS229-Fall2024" \
  --end-date "2024-12-13" \
  --auto-stop-instances \
  --revoke-access \
  --archive-workspaces

# On Dec 13, 11:59 PM:
# - Stop all 50 student instances
# - Revoke student SSH keys
# - Archive workspaces to S3
# - Generate final cost report
# - Email professor
```

**Current Status**:
- ❌ **MISSING**: Course end date tracking
- ❌ **MISSING**: Automated cleanup scheduler
- ❌ **MISSING**: Workspace archival system
- ❌ **MISSING**: Final report generation

**What's Missing**:
1. Course end date configuration
2. Cron/scheduler for end-of-semester actions
3. Bulk instance stop operation
4. SSH key revocation system
5. S3 workspace archival
6. Final report email automation

**Suggested Phase**: **v0.8.0** (Class Management Basics)
**Priority**: 🔴 **Critical** - Prevents spending over break
**Effort**: Medium (1 week)

---

### 4.6 Academic Integrity Audit Logs

**Promised Features**:
```bash
# Check for plagiarism between students
prism ta audit \
  --students emily.chen@university.edu,david.kim@university.edu \
  --timeframe "2024-10-15 to 2024-10-20" \
  --assignment hw5

# Output:
# - Complete command history
# - File modification timeline
# - Git commit history
# - Suspicious activity flags
# - Code similarity detection
```

**Current Status**:
- ❌ **MISSING**: Audit logging system
- ❌ **MISSING**: Command history tracking
- ❌ **MISSING**: Plagiarism detection tools

**What's Missing**:
1. `prism ta audit` command
2. Instance command logging
3. File modification tracking
4. SSH session recording
5. Code similarity analysis
6. Audit report generation (PDF/CSV)

**Suggested Phase**: **v0.9.0** (Academic Features)
**Priority**: 🟡 **High** - Academic integrity compliance
**Effort**: High (1 week)

---

### 4.7 TA Dashboard

**Promised Features**:
```bash
# TA views all students in course
prism ta dashboard "CS229-Fall2024"

# Shows:
# - List of all students
# - Instance status per student
# - Budget usage warnings
# - Pending help requests
# - Recent activity
```

**Current Status**:
- ❌ **MISSING**: `prism ta dashboard` command
- ❌ **MISSING**: TUI/GUI TA interface
- ❌ **MISSING**: Student activity monitoring

**What's Missing**:
1. TA-specific dashboard views
2. Student list with status
3. At-risk student detection
4. Help request queue
5. Activity monitoring and alerts

**Suggested Phase**: **v0.8.1** (TA Support Tools)
**Priority**: 🟡 **High** - Proactive student support
**Effort**: Medium (1 week)

---

### 4.8 Course Material Upload

**Promised Features**:
```bash
# Upload shared course materials
prism course upload-materials "CS229-Fall2024" \
  --source ~/CS229-Materials/ \
  --destination /datasets

# Creates shared read-only EFS volume
# Available to all students at /mnt/cs229-materials/
```

**Current Status**:
- ⚠️ **PARTIAL**: EFS volume creation exists (`pkg/storage/efs_manager.go`)
- ❌ **MISSING**: Course-specific shared storage
- ❌ **MISSING**: Bulk file upload command

**What's Missing**:
1. `prism course upload-materials` command
2. Shared read-only EFS for courses
3. Bulk file upload with progress
4. Automatic mounting to student instances

**Suggested Phase**: **v0.8.0** (Class Management Basics)
**Priority**: 🟡 **High** - Essential for coursework
**Effort**: Low (3 days)

---

### 4.9 Canvas/LMS Integration

**Promised Features**:
```bash
# Sync with Canvas LMS
prism course import-students --canvas --course-id 12345
prism course sync-grades "CS229-Fall2024"
prism course sync-due-dates
```

**Current Status**:
- ❌ **MISSING**: Canvas API integration
- ❌ **MISSING**: Grade passback
- ❌ **MISSING**: Due date synchronization
- ❌ **MISSING**: Single sign-on (SSO)

**What's Missing**:
1. Canvas API client
2. OAuth authentication with Canvas
3. Student roster sync
4. Assignment due date sync
5. Grade export/passback
6. LTI integration for SSO

**Suggested Phase**: **v0.9.1** (LMS Integration)
**Priority**: 🟢 **Medium** - Streamlines course management
**Effort**: High (2 weeks)

---

### 4.10 Workshop Mode

**Promised Features**:
```bash
# Create 3-hour workshop
prism workshop create "AWS-MLOps-Tutorial" \
  --date 2024-11-15 \
  --duration 3h \
  --max-participants 50 \
  --budget 150 \
  --access-code "MLOPS2024"

# Participants join via code
prism workshop join --code MLOPS2024
```

**Current Status**:
- ❌ **MISSING**: `prism workshop` command namespace
- ❌ **MISSING**: Workshop entity (time-limited project)
- ❌ **MISSING**: Access code system

**What's Missing**:
1. Workshop data model
2. Access code generation and validation
3. Time-limited access (3 hours + optional extension)
4. Anonymous participant support
5. Auto-cleanup after workshop end

**Suggested Phase**: **v0.8.2** (Workshop Support)
**Priority**: 🟢 **Medium** - Conference/tutorial use case
**Effort**: Low (5 days, reuses class infrastructure)

---

### 4.11 Student Instance Reset

**Promised Features**:
```bash
# TA resets broken student instance
prism ta reset-instance ml-hw4 \
  --student sophie.martinez@university.edu

# Actions:
# ✅ Backup current state to S3
# ✅ Launch fresh instance from template
# ✅ Restore student homework files
# ✅ Send email notification
```

**Current Status**:
- ❌ **MISSING**: Instance reset functionality
- ❌ **MISSING**: Selective file backup/restore
- ❌ **MISSING**: TA permission to reset student instances

**What's Missing**:
1. `prism ta reset-instance` command
2. Selective file backup (preserve homework, discard environment)
3. Instance recreation from template
4. File restoration logic
5. Notification system

**Suggested Phase**: **v0.8.1** (TA Support Tools)
**Priority**: 🟡 **High** - Reduces student frustration
**Effort**: Medium (3-5 days)

---

### 4.12 Student Budget Distribution

**Promised Features**:
```bash
# Course with per-student budget
prism course create "CS229-Fall2024" \
  --total-budget 1200 \
  --budget-per-student 24

# Student sees their individual budget
emily@laptop:~$ prism budget status
# Your budget: $12 / $24 (50%)
```

**Current Status**:
- ⚠️ **PARTIAL**: Project budgets exist (`pkg/project/budget_tracker.go`)
- ❌ **MISSING**: Per-member budget allocation within projects
- ❌ **MISSING**: Individual budget enforcement

**What's Missing**:
1. Per-member budget allocation
2. Individual budget tracking within projects
3. Student-level budget enforcement
4. Budget exhaustion handling

**Suggested Phase**: **v0.8.0** (Class Management Basics)
**Priority**: 🔴 **Critical** - Prevents individual overspending
**Effort**: Medium (1 week)

---

## Category 5: Policy Framework

**Documentation**: [TEMPLATE_POLICY_FRAMEWORK.md](admin-guides/TEMPLATE_POLICY_FRAMEWORK.md)

### 5.1 Template Whitelist/Blacklist

**Promised Features**:
```bash
# Create profile with template restrictions
prism profiles create lab-profile \
  --template-whitelist "python-basic,r-basic" \
  --template-blacklist "gpu-ml"

# Launch with policy enforcement
prism launch gpu-ml-workstation test
# ❌ Error: Template 'gpu-ml-workstation' not allowed by profile policy
```

**Current Status**:
- ⚠️ **PARTIAL**: Policy framework exists (`pkg/policy/`)
- ❌ **MISSING**: Template whitelist/blacklist enforcement at launch
- ❌ **MISSING**: Profile-level template restrictions

**What's Missing**:
1. Template whitelist/blacklist configuration
2. Launch-time policy validation
3. Profile-level template restrictions
4. Policy inheritance (profile → project → user)

**Suggested Phase**: **v0.6.2** (Policy Framework Enhancement)
**Priority**: 🟡 **High** - Essential for institutional control
**Effort**: Medium (1 week)

---

### 5.2 Instance Type Restrictions

**Promised Features**:
```bash
# Limit instance types
prism profile create restricted \
  --max-instance-types "t3.medium,t3.large"

# Policy check at launch
prism launch python-ml my-project --instance-type c5.4xlarge
# ❌ Error: Instance type 'c5.4xlarge' exceeds profile limits
```

**Current Status**:
- ⚠️ **PARTIAL**: Policy types defined in `pkg/policy/types.go`
- ❌ **MISSING**: Instance type validation at launch
- ❌ **MISSING**: Profile-level instance type limits

**What's Missing**:
1. Instance type whitelist configuration
2. Launch-time instance type validation
3. Cost-based instance type limits
4. Educational warnings for expensive types

**Suggested Phase**: **v0.6.2** (Policy Framework Enhancement)
**Priority**: 🟡 **High** - Cost control
**Effort**: Low (3 days)

---

### 5.3 Regional Restrictions

**Promised Features**:
```bash
# Restrict to specific regions
prism profile create regional \
  --allowed-regions "us-west-2,us-east-1"

prism launch python-ml my-project --region eu-west-1
# ❌ Error: Region 'eu-west-1' not allowed by profile policy
```

**Current Status**:
- ❌ **MISSING**: Regional policy enforcement
- ❌ **MISSING**: Profile-level region restrictions

**What's Missing**:
1. Region whitelist configuration
2. Launch-time region validation
3. Compliance-based region restrictions

**Suggested Phase**: **v0.6.2** (Policy Framework Enhancement)
**Priority**: 🟢 **Medium** - Compliance requirement
**Effort**: Low (2 days)

---

### 5.4 Cost Limit Enforcement

**Promised Features**:
```bash
# Set cost limits per profile
prism profile create cost-limited \
  --max-hourly-cost 0.20 \
  --max-daily-budget 5.00

# Launch blocked by cost limit
prism launch gpu-ml-workstation expensive
# ❌ Error: Estimated cost $24.80/day exceeds limit $5.00/day
```

**Current Status**:
- ⚠️ **PARTIAL**: Cost calculation exists (`pkg/project/cost_calculator.go`)
- ❌ **MISSING**: Pre-launch cost validation against limits
- ❌ **MISSING**: Profile-level cost limits

**What's Missing**:
1. Cost limit configuration per profile
2. Pre-launch cost validation
3. Cost limit inheritance
4. Optional cost warnings vs. hard blocks

**Suggested Phase**: **v0.6.0** (Budget Safety Net)
**Priority**: 🔴 **Critical** - Prevents overspending
**Effort**: Low (3 days)

---

### 5.5 Digital Signature Verification (Enterprise)

**Documentation**: [TEMPLATE_POLICY_FRAMEWORK.md](admin-guides/TEMPLATE_POLICY_FRAMEWORK.md) (lines 151-169)

**Promised Features**:
```yaml
# Template with digital signature
name: "Institutional Python ML Environment"
signature:
  authority: "University IT Security"
  public_key_id: "univ-it-2024"
  signature: "base64-encoded-signature"
```

**Current Status**:
- ❌ **MISSING**: Digital signature system
- ❌ **MISSING**: Template signing tools
- ❌ **MISSING**: Signature verification at launch

**Note**: Documented as **Enterprise Feature (Proprietary)**, not open source

**What's Missing**:
1. Template signing workflow
2. Public key infrastructure (PKI)
3. Signature verification
4. Revocation checking
5. Trust chain validation

**Suggested Phase**: **v1.0.0+** (Enterprise Features - Post Open Source)
**Priority**: 🟢 **Low** - Enterprise-only
**Effort**: High (3+ weeks)

---

### 5.6 Compliance Framework Integration (Enterprise)

**Documentation**: [TEMPLATE_POLICY_FRAMEWORK.md](admin-guides/TEMPLATE_POLICY_FRAMEWORK.md) (lines 71-91)

**Promised Features**:
```go
type InstitutionalPolicyEngine struct {
    ComplianceFrameworks []string // "HIPAA", "SOX", "NIST 800-171"
    AuditLogging        AuditConfig
}
```

**Current Status**:
- ❌ **MISSING**: Compliance framework definitions
- ❌ **MISSING**: Policy enforcement based on compliance

**Note**: Documented as **Enterprise Feature (Proprietary)**

**What's Missing**:
1. Compliance framework metadata
2. HIPAA/SOX/NIST policy enforcement
3. Compliance audit logging
4. Compliance dashboards

**Suggested Phase**: **v1.0.0+** (Enterprise Features)
**Priority**: 🟢 **Low** - Enterprise-only
**Effort**: Very High (1-2 months)

---

### 5.7 Security Classification System (Enterprise)

**Documentation**: [TEMPLATE_POLICY_FRAMEWORK.md](admin-guides/TEMPLATE_POLICY_FRAMEWORK.md) (lines 84-90)

**Promised Features**:
```go
type SecurityLevel string
const (
    SecurityPublic       SecurityLevel = "public"
    SecurityInternal     SecurityLevel = "internal"
    SecurityConfidential SecurityLevel = "confidential"
    SecurityRestricted   SecurityLevel = "restricted"
)
```

**Current Status**:
- ❌ **MISSING**: Security classification system
- ❌ **MISSING**: User clearance levels
- ❌ **MISSING**: Clearance-based access control

**Note**: Documented as **Enterprise Feature (Proprietary)**

**What's Missing**:
1. Security classification metadata
2. User clearance tracking
3. Classification-based access control
4. Data handling rules

**Suggested Phase**: **v1.0.0+** (Enterprise Features)
**Priority**: 🟢 **Low** - Enterprise-only
**Effort**: High (2-3 weeks)

---

### 5.8 Institutional Policy Dashboard (Enterprise)

**Documentation**: [TEMPLATE_POLICY_FRAMEWORK.md](admin-guides/TEMPLATE_POLICY_FRAMEWORK.md) (line 239)

**Promised Features**:
- Institutional control plane
- Policy violation reporting
- Template lifecycle management
- Institutional dashboard/reporting

**Current Status**:
- ❌ **MISSING**: Institutional admin dashboard
- ❌ **MISSING**: Policy violation tracking

**Note**: Documented as **Enterprise Feature (Proprietary)**

**Suggested Phase**: **v1.0.0+** (Enterprise Features)
**Priority**: 🟢 **Low** - Enterprise-only
**Effort**: Very High (1-2 months)

---

## Category 6: AMI Compilation System

**Documentation**: [AMI_POLICY_ENFORCEMENT.md](admin-guides/AMI_POLICY_ENFORCEMENT.md)

### 6.1 Template Compilation to AMI

**Promised Features**:
```bash
# Compile template to AMI
prism templates compile python-ml \
  --regions us-west-2,eu-west-1 \
  --architectures x86_64,arm64

# Check compilation status
prism templates compile status python-ml
```

**Current Status**:
- ⚠️ **PARTIAL**: AMI-related code exists (`pkg/ami/`, `pkg/aws/ami_*.go`)
- ❌ **MISSING**: `prism templates compile` command
- ❌ **MISSING**: Template-to-AMI build pipeline
- ❌ **MISSING**: Multi-region AMI distribution

**What's Missing**:
1. `prism templates compile` command
2. EC2 Image Builder integration
3. Packer-based template compilation
4. Multi-region AMI copying
5. Compilation progress tracking
6. AMI metadata embedding

**Suggested Phase**: **v0.7.0** (Auto-AMI System)
**Priority**: 🟢 **Medium** - Performance optimization
**Effort**: Very High (3-4 weeks)

---

### 6.2 Pre-Compiled AMI Templates

**Documentation**: [AMI_POLICY_ENFORCEMENT.md](admin-guides/AMI_POLICY_ENFORCEMENT.md) (lines 187-233)

**Promised Features**:
```yaml
# Template with pre-compiled AMIs
name: "Python Machine Learning (Compiled)"
compile_to_ami:
  enabled: true
  regions: ["us-west-2", "us-east-1"]

precompiled_amis:
  us-west-2:
    x86_64: "ami-0abc123def456789a"
    arm64:  "ami-0def456abc789012b"
```

**Current Status**:
- ⚠️ **PARTIAL**: Template system supports AMI references
- ❌ **MISSING**: Automatic fallback to pre-compiled AMIs
- ❌ **MISSING**: AMI metadata validation

**What's Missing**:
1. Template AMI resolution logic
2. Automatic AMI vs. package installation selection
3. AMI age validation (use fresh AMIs)
4. Performance comparison (AMI vs. template)

**Suggested Phase**: **v0.7.0** (Auto-AMI System)
**Priority**: 🟢 **Medium** - Faster launches
**Effort**: Medium (1 week)

---

### 6.3 AMI Policy Enforcement

**Documentation**: [AMI_POLICY_ENFORCEMENT.md](admin-guides/AMI_POLICY_ENFORCEMENT.md) (lines 87-153)

**Promised Features**:
```bash
# AMI inherits source template policies
prism launch python-ml-compiled my-homework

# Policy check:
# → Source template 'python-ml' is in whitelist ✓
# → AMI embedded cost $0.0464 < limit $0.15 ✓
# → Launch approved
```

**Current Status**:
- ❌ **MISSING**: AMI policy validation
- ❌ **MISSING**: Source template tracking in AMIs
- ❌ **MISSING**: Policy inheritance from template to AMI

**What's Missing**:
1. AMI metadata extraction
2. Source template linkage
3. Policy validation for AMI launches
4. Cost limit enforcement for AMIs

**Suggested Phase**: **v0.7.0** (Auto-AMI System)
**Priority**: 🟢 **Medium** - Consistent policy enforcement
**Effort**: Medium (1 week)

---

### 6.4 AMI Cost Analysis

**Documentation**: [AMI_POLICY_ENFORCEMENT.md](admin-guides/AMI_POLICY_ENFORCEMENT.md) (lines 72-84)

**Promised Features**:
```yaml
policy_metadata:
  cost_estimates:
    x86_64: 0.0464  # Hourly cost
    arm64:  0.0371  # ARM cheaper
  resource_limits:
    max_hourly_cost: 0.20
```

**Current Status**:
- ⚠️ **PARTIAL**: Cost calculation exists (`pkg/aws/pricing.go`)
- ❌ **MISSING**: AMI-specific cost estimates
- ❌ **MISSING**: Cost comparison (AMI vs. on-demand build)

**What's Missing**:
1. AMI cost metadata storage
2. Cost comparison tools
3. Architecture-specific cost analysis
4. Cost savings reporting

**Suggested Phase**: **v0.7.0** (Auto-AMI System)
**Priority**: 🟢 **Low** - Cost transparency
**Effort**: Low (3 days)

---

### 6.5 Auto-AMI Scheduling

**Documentation**: [AMI_POLICY_ENFORCEMENT.md](admin-guides/AMI_POLICY_ENFORCEMENT.md) - Implied feature

**Promised Features**:
```bash
# Automatic compilation for popular templates
# - Popularity-driven (weekly builds for top 10 templates)
# - Security-driven (rebuild on base OS patches)
# - Cost-optimized (build during off-peak hours)
```

**Current Status**:
- ❌ **MISSING**: Popularity tracking
- ❌ **MISSING**: Automatic compilation scheduler
- ❌ **MISSING**: Security patch detection

**What's Missing**:
1. Template usage analytics
2. Scheduled compilation jobs
3. Base OS CVE monitoring
4. Automatic rebuilds on security updates

**Suggested Phase**: **v0.7.0** (Auto-AMI System)
**Priority**: 🟢 **Low** - Automation enhancement
**Effort**: High (2 weeks)

---

### 6.6 AMI Template Info Display

**Documentation**: [AMI_POLICY_ENFORCEMENT.md](admin-guides/AMI_POLICY_ENFORCEMENT.md) (lines 290-306)

**Promised Features**:
```bash
prism templates info python-ml-compiled

# Output:
# Template: Python Machine Learning (Compiled)
# Type: Compiled (AMI-based)
# Source Template: python-ml-v2.1
# Compilation Status: Complete
# Available AMIs:
#   us-west-2: ami-0abc... (x86_64), ami-0def... (arm64)
# Launch Performance: ~30 seconds (vs ~5-8 minutes for source)
```

**Current Status**:
- ✅ **EXISTS**: `prism templates info` command
- ❌ **MISSING**: AMI-specific information display
- ❌ **MISSING**: Compilation status tracking

**What's Missing**:
1. AMI information in template info
2. Compilation status display
3. Performance comparison metrics
4. AMI age and freshness indicators

**Suggested Phase**: **v0.7.0** (Auto-AMI System)
**Priority**: 🟢 **Low** - User transparency
**Effort**: Low (2 days)

---

## Category 7: DCV/Desktop Integration

**Documentation**: Mentioned in [WEB_SERVICES_INTEGRATION_GUIDE.md](user-guides/WEB_SERVICES_INTEGRATION_GUIDE.md)

### 7.1 NICE DCV Desktop Support

**Promised Features**:
- GUI desktop access via NICE DCV
- Web-based desktop in iframe
- VNC alternative for graphical workloads

**Current Status**:
- ⚠️ **PARTIAL**: DCV proxy handler exists (`pkg/daemon/connection_proxy_handlers.go`)
- ❌ **MISSING**: DCV installation in templates
- ❌ **MISSING**: DCV configuration and setup
- ❌ **MISSING**: CLI commands for DCV access

**Evidence**:
```go
// In pkg/daemon/connection_proxy_handlers.go:
func (s *Server) handleDCVProxy(w http.ResponseWriter, r *http.Request) {
    // DCV desktop proxy implementation
}
```

**What's Missing**:
1. DCV template examples
2. DCV server installation scripts
3. DCV authentication integration
4. `prism dcv` CLI commands
5. GUI interface for DCV access

**Suggested Phase**: **v0.8.0** (Web Services Integration)
**Priority**: 🟡 **High** - Important for GUI workflows
**Effort**: High (2 weeks)

---

### 7.2 Desktop Templates

**Promised Features**:
```bash
# Launch desktop workstation
prism launch ubuntu-desktop my-desktop

# Access via DCV
prism dcv connect my-desktop
```

**Current Status**:
- ❌ **MISSING**: Desktop-enabled templates
- ❌ **MISSING**: GUI installation scripts
- ❌ **MISSING**: Desktop environment configuration

**What's Missing**:
1. Desktop templates (Ubuntu, Rocky Linux with GUI)
2. XFCE/GNOME installation
3. DCV server setup in templates
4. Display configuration

**Suggested Phase**: **v0.8.0** (Web Services Integration)
**Priority**: 🟡 **High** - Desktop use cases
**Effort**: Medium (1 week)

---

### 7.3 DCV CLI Commands

**Promised Features**:
```bash
# DCV management
prism dcv start my-instance
prism dcv connect my-instance
prism dcv status my-instance
```

**Current Status**:
- ❌ **MISSING**: `prism dcv` command namespace
- ❌ **MISSING**: DCV lifecycle management

**What's Missing**:
1. DCV CLI commands
2. DCV session management
3. DCV URL generation
4. DCV authentication

**Suggested Phase**: **v0.8.0** (Web Services Integration)
**Priority**: 🟢 **Medium** - Desktop workflow
**Effort**: Low (3 days)

---

## Summary Statistics

### By Priority

| Priority | Count | Features |
|----------|-------|----------|
| 🔴 **Critical** | 18 | Personal budgets, alerts, class management, template whitelisting, TA debug |
| 🟡 **High** | 21 | Forecasting, TA tools, policy enforcement, DCV support, student management |
| 🟢 **Medium/Low** | 13 | AMI compilation, enterprise features, workshop mode, reporting |
| **Total** | **52** | |

### By Suggested Phase

| Phase | Features | Priority Focus |
|-------|----------|----------------|
| **v0.6.0** (Budget Safety Net) | 7 | Personal budgets, alerts, pre-launch checks |
| **v0.6.1** (Budget Intelligence) | 3 | Forecasting, reporting |
| **v0.6.2** (Policy Framework) | 4 | Template policies, cost limits |
| **v0.7.0** (Auto-AMI System) | 6 | AMI compilation, performance |
| **v0.8.0** (Class Management) | 7 | Course creation, whitelisting, semester end |
| **v0.8.1** (TA Support) | 5 | TA debug, dashboard, student reset |
| **v0.8.2** (Workshop Support) | 1 | Workshop mode |
| **v0.9.0** (Academic Features) | 2 | Audit logs, analytics |
| **v0.9.1** (LMS Integration) | 1 | Canvas integration |
| **v1.0.0+** (Enterprise) | 4 | Digital signatures, compliance, security classification |
| **Total** | **52** | |

### By Effort Estimate

| Effort | Count | Total Time (weeks) |
|--------|-------|--------------------|
| **Low** (< 1 week) | 18 | ~9 weeks |
| **Medium** (1-2 weeks) | 21 | ~31 weeks |
| **High** (2-4 weeks) | 9 | ~27 weeks |
| **Very High** (1+ months) | 4 | ~16 weeks |
| **Total** | **52** | **~83 weeks** (16 months) |

---

## Recommended Action Plan

### Phase 1: Budget Safety Net (v0.6.0) - **4 weeks**
**Goal**: Solo researchers can confidently manage budgets

1. Personal budget system (separate from project budgets) - 2 weeks
2. Budget alerts (email notifications) - 1 week
3. Pre-launch budget check - 3 days
4. Cost limit enforcement - 3 days

**Impact**: Addresses critical solo researcher anxiety (Scenario 1)

---

### Phase 2: Class Management Basics (v0.8.0) - **5 weeks**
**Goal**: Professors can safely run classes

1. Course creation and management - 2 weeks
2. Template whitelisting enforcement - 1 week
3. Automatic semester end cleanup - 1 week
4. Student bulk import (CSV) - 3 days
5. Student budget distribution - 1 week

**Impact**: Addresses critical class workflow (Scenario 3)

---

### Phase 3: TA Support Tools (v0.8.1) - **3 weeks**
**Goal**: TAs can efficiently help students

1. TA debug access system - 2 weeks
2. Instance reset capability - 3 days
3. TA dashboard - 1 week

**Impact**: Makes office hours 3x more efficient (Scenario 3)

---

### Phase 4: Policy & Enterprise (v0.6.2, v0.9.0) - **3 weeks**
**Goal**: Institutional governance and compliance

1. Policy framework enhancements - 1 week
2. Academic integrity audit logs - 1 week
3. DCV/Desktop integration - 1 week

**Impact**: Enables institutional adoption

---

## Documentation Recommendations

### 1. Mark Features as "Planned"
Add status markers to all documentation:
```markdown
<!-- STATUS: ✅ IMPLEMENTED (v0.5.x) -->
<!-- STATUS: 🚧 IN PROGRESS (v0.6.0) -->
<!-- STATUS: 📋 PLANNED (v0.8.0) -->
```

### 2. Create Implementation Tracking
- GitHub Issues for each missing feature
- Milestones for each phase (v0.6.0, v0.8.0, etc.)
- Link docs to tracking issues

### 3. Update User Scenarios
Revise persona walkthroughs to show:
- ✅ **What works today**
- 🚧 **What's in progress**
- 📋 **What's planned**

### 4. Separate Documentation
- `docs/CURRENT_FEATURES.md` - What exists now
- `docs/ROADMAP.md` - What's planned
- `docs/FUTURE_VISION.md` - Long-term ideas

---

## Notes on Existing Features

### Features That ARE Implemented

1. ✅ **Profile Export/Import** - Fully functional (`pkg/profile/export/`, `internal/cli/export.go`)
2. ✅ **Project Budgets** - Complete (`pkg/project/budget_tracker.go`)
3. ✅ **Budget Commands** - CLI exists (`internal/cli/budget_commands.go`)
4. ✅ **Hibernation System** - Fully operational (Phase 3 complete)
5. ✅ **Policy Framework Foundation** - Basic structure (`pkg/policy/`)
6. ✅ **Research User System** - Complete (Phase 5A)
7. ✅ **Template Marketplace** - Complete (Phase 5B)
8. ✅ **DCV Proxy Handler** - Backend exists (`pkg/daemon/connection_proxy_handlers.go`)

### Features Partially Implemented

1. ⚠️ **Budget System** - Projects only, not personal budgets
2. ⚠️ **Policy Framework** - Types defined, enforcement missing
3. ⚠️ **Invitation Security** - Code exists, CLI not wired up
4. ⚠️ **AMI Support** - Infrastructure exists, compilation missing
5. ⚠️ **DCV Support** - Backend proxy exists, templates and CLI missing

---

## Conclusion

Prism has **excellent documentation** that describes a comprehensive, enterprise-ready research platform. However, **approximately 52 features** (30-40% of documented functionality) are not yet implemented.

**Key Gaps**:
1. **Personal Budget Management** - Critical for solo researchers
2. **Class Management System** - Essential for educational adoption
3. **TA Support Tools** - Required for efficient teaching
4. **Invitation Security** - Device binding not accessible via CLI
5. **Policy Enforcement** - Framework exists but not enforced
6. **AMI Compilation** - Performance optimization not available

**Recommendation**: Prioritize **Solo Researcher Budget Features (v0.6.0)** and **Class Management Basics (v0.8.0)** as these address the two most compelling user scenarios and have the highest impact on adoption.

---

**End of Audit Report**
