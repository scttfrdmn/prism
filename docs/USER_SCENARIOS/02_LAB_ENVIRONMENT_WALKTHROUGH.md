# Scenario 2: Research Lab with Hierarchical Budget Management

## Personas: The Smith Computational Biology Lab

### Dr. Patricia Smith (PI / Lab Director)
- **Role**: Principal Investigator
- **Responsibilities**: Oversees 3 grants, approves large purchases, monitors overall lab spend
- **Technical level**: Strategic oversight, delegates technical details
- **Concerns**: Stay within grant budgets, compliance, audit trails
- **Time constraints**: Very busy, needs dashboard views and exception alerts only

### Dr. Michael Torres (Senior Research Scientist)
- **Role**: Lab Manager / Senior Staff
- **Responsibilities**: Day-to-day lab operations, mentors junior staff, manages GPU cluster usage
- **Technical level**: Expert - can troubleshoot CloudWorkstation, optimizes costs
- **Concerns**: Efficient resource allocation, preventing grad student mistakes
- **Authority**: Can approve requests up to $500, launch any instance type

### Dr. Lisa Park (Postdoctoral Researcher)
- **Role**: Independent researcher with sub-grant
- **Responsibilities**: Leads protein folding project, manages 2 grad students
- **Technical level**: Advanced - comfortable with command line and cloud
- **Concerns**: Staying within her sub-budget ($800/month), finishing papers before fellowship ends
- **Authority**: Can launch CPU instances freely, needs approval for GPU

### James Wilson (Graduate Student - Year 4)
- **Role**: Ph.D. candidate working on RNA-seq analysis
- **Responsibilities**: Running experiments, learning computational methods
- **Technical level**: Intermediate - knows Python/R, learning cloud concepts
- **Concerns**: Not breaking anything, staying within allocated resources
- **Authority**: Can launch t3/r5 instances only, limited to 2 instances

### Maria Garcia (Graduate Student - Year 2)
- **Role**: Rotating student, new to lab
- **Responsibilities**: Learning pipelines, running established protocols
- **Technical level**: Beginner - just learned command line
- **Concerns**: Following instructions correctly, not wasting money
- **Authority**: Can launch single t3.medium instance, read-only access to shared data

---

## Lab Structure & Budget Allocation

### Grant Portfolio (Total: $4,500/month)

```
Smith Lab Organization
├── NIH Grant R01-2023 ($2,000/month)
│   ├── Dr. Torres (Lab Manager): $800/month - GPU cluster management
│   ├── James Wilson (Grad Student): $400/month - RNA-seq
│   └── Shared Resources: $800/month - EFS storage, collaborative workspaces
│
├── NSF Grant 2024-ML ($1,500/month)
│   ├── Dr. Lisa Park (Postdoc): $800/month - Protein folding lead
│   ├── Maria Garcia (Grad Student): $300/month - Learning project
│   └── Reserved: $400/month - Conference demos, visiting scholars
│
└── Discretionary Fund ($1,000/month)
    ├── Dr. Smith (PI): $500/month - Emergency overages, new projects
    └── Dr. Torres (Lab Manager): $500/month - Operational buffer
```

---

## Current State (v0.5.5): What Works Today

### ✅ Lab Setup (Phase 4 Complete)

#### Step 1: PI Creates Organization
```bash
# Dr. Smith creates lab organization
cws project create "Smith Lab" \
  --description "Computational Biology Research Group" \
  --owner patricia.smith@university.edu

# Output:
# ✅ Project created: Smith Lab (proj-abc123)
```

#### Step 2: Create Grant Projects
```bash
# NIH Grant project
cws project create "NIH-R01-2023" \
  --parent "Smith Lab" \
  --budget 2000 \
  --budget-period monthly \
  --description "RNA-seq and transcriptomics research"

# NSF Grant project
cws project create "NSF-2024-ML" \
  --parent "Smith Lab" \
  --budget 1500 \
  --budget-period monthly \
  --description "Machine learning for protein structure prediction"

# Discretionary
cws project create "Discretionary" \
  --parent "Smith Lab" \
  --budget 1000 \
  --budget-period monthly \
  --description "PI discretionary funds"
```

#### Step 3: Add Lab Members with Roles
```bash
# Add senior staff with Admin role
cws project member add "NIH-R01-2023" \
  --email michael.torres@university.edu \
  --role admin \
  --budget-allocation 800

cws project member add "NSF-2024-ML" \
  --email lisa.park@university.edu \
  --role admin \
  --budget-allocation 800

# Add graduate students with Member role
cws project member add "NIH-R01-2023" \
  --email james.wilson@university.edu \
  --role member \
  --budget-allocation 400

cws project member add "NSF-2024-ML" \
  --email maria.garcia@university.edu \
  --role member \
  --budget-allocation 300
```

#### Step 4: Configure Budget Alerts
```bash
# Alert PI at 75% and 90% of each project
cws project budget alert add "NIH-R01-2023" \
  --threshold 75 \
  --email patricia.smith@university.edu

cws project budget alert add "NIH-R01-2023" \
  --threshold 90 \
  --email patricia.smith@university.edu,michael.torres@university.edu

# Same for other projects...
```

### ✅ Daily Lab Operations (What Works)

#### Scenario: James (Grad Student) Runs RNA-seq Pipeline
```bash
# James launches instance
cws launch bioinformatics-suite rnaseq-sample-42 \
  --project "NIH-R01-2023" \
  --size M

# CloudWorkstation output:
# ✅ Instance launching: rnaseq-sample-42
# 📊 Cost: $2.40/day (r5.xlarge)
# 💰 Project budget: $245 / $400 (61% used this month)
# 🔗 SSH ready in ~90 seconds...

# James works for 4 hours, then stops
cws stop rnaseq-sample-42

# Cost tracking automatically updated
cws project cost show "NIH-R01-2023"

# Output:
# 💰 Project: NIH-R01-2023 Budget Status
#    Monthly budget: $2,000.00
#    Current spend: $1,245.80 (62%)
#
#    By member:
#    - michael.torres: $720.50 / $800.00 (90%)
#    - james.wilson: $245.30 / $400.00 (61%)
#    - Shared resources: $280.00 / $800.00 (35%)
```

#### Scenario: Lab Manager Monitors Usage
```bash
# Dr. Torres checks overall lab status
cws project list --tree

# Output:
# Smith Lab
# ├── NIH-R01-2023: $1,245 / $2,000 (62%) ✅
# │   ├── michael.torres: $720 / $800 (90%) ⚠️
# │   ├── james.wilson: $245 / $400 (61%) ✅
# │   └── shared: $280 / $800 (35%) ✅
# ├── NSF-2024-ML: $980 / $1,500 (65%) ✅
# │   ├── lisa.park: $650 / $800 (81%) ⚠️
# │   └── maria.garcia: $130 / $300 (43%) ✅
# └── Discretionary: $50 / $1,000 (5%) ✅
#
# Total: $2,275 / $4,500 (51%) ✅
```

---

## ⚠️ Current Pain Points: What Doesn't Work

### ❌ Problem 1: No Sub-Budget Hierarchy
**Scenario**: Dr. Park wants to allocate her $800 between her own work and grad student Maria

**What should work** (MISSING):
```bash
# Dr. Park creates sub-budgets from her allocation
cws project budget allocate "NSF-2024-ML" \
  --member lisa.park \
  --subdivide \
  --personal 500 \
  --delegate maria.garcia 300

# Result should be:
# NSF-2024-ML: $800 allocated to lisa.park
# ├── lisa.park (personal): $500
# └── maria.garcia (supervised by lisa.park): $300
```

**Current limitation**: Flat budget allocation only - no delegation
**Workaround**: Manual tracking in spreadsheet, trust system
**Impact**: Dr. Park can't manage her sub-team independently

### ❌ Problem 2: No Approval Workflows
**Scenario**: Maria (beginner grad student) tries to launch expensive GPU instance

**What should happen** (MISSING):
```bash
# Maria attempts GPU launch
cws launch gpu-ml-workstation protein-experiment --project "NSF-2024-ML"

# CloudWorkstation should prompt:
# ⚠️  APPROVAL REQUIRED: GPU Instance Launch
#
#    Requested by: maria.garcia@university.edu
#    Instance: p3.2xlarge ($24.80/day)
#    Project: NSF-2024-ML
#    Your budget: $130 / $300 (43%)
#
#    This instance exceeds your authority level.
#    Approval request sent to:
#    - Dr. Lisa Park (lisa.park@university.edu) - Project lead
#    - Dr. Michael Torres (michael.torres@university.edu) - Lab manager
#
#    Request ID: req-xyz789
#    Status: Pending approval (will notify via email)
#
#    You can check status with: cws approval status req-xyz789

# Dr. Park receives email:
# Subject: Approval Request: GPU Instance Launch (Maria Garcia)
#
# Maria Garcia has requested approval to launch:
# - Instance: p3.2xlarge (1 GPU, $24.80/day)
# - Project: NSF-2024-ML
# - Justification: "Need GPU for protein folding simulation homework"
# - Estimated cost: $24.80 (8 hour time limit requested)
#
# Maria's budget: $130 / $300 (43% used)
# Project budget: $980 / $1,500 (65% used)
#
# Approve or deny: cws approval review req-xyz789
```

**Current state**: No approval system - relies on role-based restrictions only
**Workaround**: Maria asks in Slack, someone with admin role launches for her
**Impact**: Bypasses audit trails, confusion about who launched what

### ❌ Problem 3: No Time-Boxed Collaborator Access
**Scenario**: Visiting scholar Dr. Kim joins for 3-month collaboration

**What should work** (MISSING):
```bash
# Dr. Smith grants temporary access
cws project member add "NIH-R01-2023" \
  --email dr.kim@external.edu \
  --role member \
  --budget-allocation 200 \
  --start-date 2024-06-01 \
  --end-date 2024-08-31 \
  --auto-revoke \
  --notify-before-expiry 7days

# Result:
# ✅ Temporary member added: dr.kim@external.edu
#    Access: June 1 - August 31, 2024 (90 days)
#    Budget: $200/month
#    Auto-revoke: September 1, 2024 at 00:00 UTC
#    Reminder: August 25, 2024 (7 days before)

# On August 25, both Dr. Kim and Dr. Smith receive email:
# Subject: Collaborator Access Expiring Soon
#
# Dr. Kim's access to project "NIH-R01-2023" expires in 7 days.
#
# Current usage:
# - Instances: 1 active (rnaseq-collaboration)
# - Spend: $180 / $200 (90%)
#
# Actions:
# 1. Extend access: cws project member extend dr.kim@external.edu --days 30
# 2. Let expire: Instances will be stopped, data archived on Sep 1
# 3. Convert to permanent: cws project member permanent dr.kim@external.edu

# On September 1 at 00:00 UTC (auto-revoke):
# - Dr. Kim's instances automatically stopped
# - SSH keys revoked from all project instances
# - EFS home directory archived to S3
# - Email sent to both parties confirming revocation
```

**Current state**: Manual member management - no expiration dates
**Workaround**: Calendar reminders, manual revocation
**Impact**: Forgotten temp users accumulate, security risk, budget waste

### ❌ Problem 4: No Resource Quotas by Role
**Scenario**: Grad students should have instance limits to prevent mistakes

**What should work** (MISSING):
```bash
# PI configures role-based quotas
cws project policy create "NIH-R01-2023" \
  --role member \
  --max-instances 2 \
  --max-instance-cost 5.00/day \
  --allowed-instance-types "t3.*,r5.large,r5.xlarge" \
  --blocked-instance-types "p3.*,p4.*"  # No GPUs

# Maria tries to launch 3rd instance
cws launch bioinformatics-suite experiment-3 --project "NIH-R01-2023"

# CloudWorkstation output:
# ❌ Launch failed: Quota exceeded
#
#    Your quota (Member role):
#    - Instances: 2 / 2 (100%)
#    - Current instances:
#      1. rnaseq-analysis (running)
#      2. protein-prep (stopped)
#
#    To launch another instance:
#    1. Stop or delete an existing instance
#    2. Request quota increase from lab manager
#
#    Contact: michael.torres@university.edu

# Maria tries GPU instance
cws launch gpu-ml-workstation experiment-gpu --project "NIH-R01-2023"

# CloudWorkstation output:
# ❌ Launch failed: Instance type not allowed
#
#    p3.2xlarge is not permitted for Member role.
#    Allowed instance types: t3.*, r5.large, r5.xlarge
#
#    For GPU access, request approval from:
#    - Dr. Michael Torres (Lab Manager)
#    - Dr. Patricia Smith (PI)
```

**Current state**: Basic role permissions only (owner/admin/member/viewer)
**Workaround**: Trust-based system, post-incident corrections
**Impact**: Accidental expensive launches, budget surprises

### ❌ Problem 5: No Grant Period Management
**Scenario**: NIH grant ends June 30 - need to freeze project and generate final report

**What should work** (MISSING):
```bash
# Dr. Smith configures grant end date
cws project configure "NIH-R01-2023" \
  --end-date 2024-06-30 \
  --freeze-after-end \
  --final-report-email patricia.smith@university.edu

# June 30, 2024 at 23:59 (automatic actions):
# 1. All running instances stopped
# 2. No new launches allowed
# 3. Project marked as "Archived"
# 4. Final cost report generated

# Email sent to Dr. Smith:
# Subject: Project Archived: NIH-R01-2023 Final Report
#
# The NIH-R01-2023 project has been automatically archived as of June 30, 2024.
#
# Final Statistics:
# - Total spend (12 months): $23,450 / $24,000 budget (97.7%)
# - Unused budget: $550
# - Total compute hours: 14,520
# - Hibernation savings: $4,230 (15%)
#
# Active resources at archive time:
# - Instances: 4 (all stopped automatically)
# - EFS volumes: 2 (maintained for 90-day archive period)
#
# Data Archive:
# - EFS snapshots: s3://smith-lab-archives/NIH-R01-2023/
# - Instance configs: s3://smith-lab-archives/NIH-R01-2023/instances.json
# - Cost reports: s3://smith-lab-archives/NIH-R01-2023/reports/
#
# Next steps:
# 1. Review final report (attached PDF)
# 2. Data will be archived to S3 and EFS volumes deleted after 90 days
# 3. To restore project: cws project restore NIH-R01-2023

# Generate grant office report
cws project report "NIH-R01-2023" \
  --start 2023-07-01 \
  --end 2024-06-30 \
  --format pdf \
  --template nih-final-report \
  --output ~/Desktop/NIH-R01-2023-final.pdf

# Report includes:
# - Monthly spend breakdown
# - Cost by resource type (compute, storage, network)
# - Per-member usage and efficiency
# - Hibernation/cost optimization summary
# - Instance type distribution
# - Peak usage periods
# - Compliance: All expenses within approved budget
```

**Current state**: Manual project closure, no automated archiving
**Workaround**: PI tracks grant dates in calendar, manually stops instances
**Impact**: Forgotten projects continue spending, archiving is ad-hoc

---

## 🎯 Ideal Future State: Complete Lab Walkthrough

### Month 0: Lab Setup (Full Configuration)

```bash
# Dr. Smith (PI) initial setup
cws init --org-mode

# Interactive org setup:
#
# 🏛️  Organization Setup
#
#    Organization name: Smith Computational Biology Lab
#    Primary contact: patricia.smith@university.edu
#    Institution: University Research Computing
#    Department: Molecular Biology
#
#    Billing configuration:
#    AWS Account: 123456789012
#    Cost center: BIO-COMP-001
#    Grant codes: [Will configure per-project]
#
# ✅ Organization created!

# Create projects with full configuration
cws project create "NIH-R01-2023" \
  --budget 2000 \
  --period monthly \
  --start-date 2023-07-01 \
  --end-date 2024-06-30 \
  --grant-code "1R01GM123456-01" \
  --auto-freeze-at-end \
  --alert-thresholds 50,75,90,95 \
  --approval-required-over 10.00/day

# Configure role-based policies
cws project policy create "NIH-R01-2023" \
  --role admin \
  --max-instances 10 \
  --max-daily-cost 100 \
  --approval-threshold 50/day

cws project policy create "NIH-R01-2023" \
  --role member \
  --max-instances 2 \
  --max-daily-cost 10 \
  --approval-threshold 5/day \
  --allowed-types "t3.*,r5.*" \
  --blocked-types "p3.*,p4.*,x2.*"

# Add lab members with detailed configuration
cws project member add "NIH-R01-2023" \
  --email michael.torres@university.edu \
  --role admin \
  --budget 800 \
  --notify-at 75,90 \
  --allow-subdelegation

cws project member add "NIH-R01-2023" \
  --email james.wilson@university.edu \
  --role member \
  --budget 400 \
  --supervisor michael.torres@university.edu \
  --notify-at 80 \
  --onboarding-template "grad-student-rna-seq"
```

### Month 1-3: Normal Operations with Approval Workflow

#### Week 1: James (Grad Student) Regular Work
```bash
# James launches standard analysis instance
cws launch bioinformatics-suite rnaseq-batch-1 --project "NIH-R01-2023"

# Auto-approved (within authority):
# ✅ Instance launching: rnaseq-batch-1 (r5.xlarge, $2.40/day)
# 📊 Your budget: $45 / $400 (11%)
# ⚙️  Hibernation: lab-standard (20min idle)
```

#### Week 2: Maria Requests GPU (Approval Flow)
```bash
# Maria needs GPU for first time
cws launch gpu-ml-workstation protein-hw --project "NSF-2024-ML"

# Approval required (exceeds authority):
# ⚠️  GPU Instance Approval Required
#
#    Requested: p3.2xlarge ($24.80/day, 1 GPU)
#    Your role: Member (max $5/day without approval)
#
#    Approval request created: req-202406-015
#    Notified: lisa.park@university.edu (Project lead)
#              michael.torres@university.edu (Lab manager)
#
#    Include justification: (optional but recommended)

# Maria adds context
cws approval comment req-202406-015 \
  "Need GPU for deep learning homework (Biophysics 601). Estimated 4 hours. Will use time limit."

# Dr. Park receives Slack notification (integration):
# 📋 Approval Request from Maria Garcia
#    Instance: p3.2xlarge ($24.80/day)
#    Justification: "Deep learning homework..."
#    Budget impact: $10 (4hr time limit)
#    Approve: /cws approve req-202406-015
#    Deny: /cws deny req-202406-015

# Dr. Park approves with modifications
cws approval approve req-202406-015 \
  --max-hours 6 \
  --note "Approved for homework. Auto-terminate after 6h. Come to my office if you need more time."

# Maria receives notification
# ✅ Approval granted: req-202406-015
#    Instance: p3.2xlarge
#    Time limit: 6 hours (auto-terminate at 4:30 PM today)
#    Notes from Dr. Park: "Approved for homework..."
#
#    Launch with: cws launch --approval req-202406-015

cws launch --approval req-202406-015

# Instance launches with enforced limits:
# ✅ Launching: protein-hw (p3.2xlarge)
# ⏰ Auto-terminate: 4:30 PM (6 hours)
# 📊 Estimated cost: $6.20
```

#### Week 3: Dr. Torres Manages Lab Resources
```bash
# Morning dashboard check
cws project dashboard "Smith Lab"

# Output (TUI dashboard):
# ╔══════════════════════════════════════════════════════════════╗
# ║ Smith Lab Dashboard - June 2024                              ║
# ╟──────────────────────────────────────────────────────────────╢
# ║ Total Budget: $4,500/month | Spent: $2,340 (52%) ✅         ║
# ║ Active Instances: 7 | Hibernated: 3                          ║
# ║ Pending Approvals: 2 | Budget Alerts: 1                     ║
# ╠══════════════════════════════════════════════════════════════╣
# ║                                                              ║
# ║ Projects:                                                    ║
# ║ ├─ NIH-R01-2023: $1,250 / $2,000 (63%) ⚠️ (Alert: M.Torres) ║
# ║ │  ├─ M.Torres: $740 / $800 (93%) ⚠️                        ║
# ║ │  └─ J.Wilson: $280 / $400 (70%) ✅                        ║
# ║ ├─ NSF-2024-ML: $890 / $1,500 (59%) ✅                      ║
# ║ └─ Discretionary: $200 / $1,000 (20%) ✅                    ║
# ║                                                              ║
# ║ Pending Approvals:                                           ║
# ║ 1. req-202406-018: James Wilson - GPU (p3.2xlarge)          ║
# ║    Justification: "Benchmarking new pipeline"                ║
# ║    [A]pprove  [D]eny  [M]ore info                           ║
# ║ 2. req-202406-019: External: dr.kim@external.edu            ║
# ║    Temporary access request (3 months)                       ║
# ║    [R]eview  [S]kip                                         ║
# ╚══════════════════════════════════════════════════════════════╝

# Dr. Torres reviews James' GPU request
cws approval show req-202406-018

# Details:
# Approval Request: req-202406-018
# Requested by: James Wilson (james.wilson@university.edu)
# Instance: p3.2xlarge ($24.80/day)
# Project: NIH-R01-2023
# Time: June 15, 2024 at 9:30 AM
#
# Justification:
# "Need to benchmark new RNA-seq pipeline with deep learning step.
#  Comparing CPU vs GPU performance for paper revision.
#  Estimated 12 hours of testing."
#
# Budget Analysis:
# James' budget: $280 / $400 (70%)
# Cost impact: ~$12.40 (12 hours)
# After approval: $292.40 / $400 (73%) ✅
#
# Previous GPU usage: 2 times (both approved, well-utilized)
# Recommendation: ✅ Low risk, reasonable justification

# Approve with time limit
cws approval approve req-202406-018 \
  --max-hours 12 \
  --note "Approved for benchmarking. Please document results for lab meeting."

# Dr. Torres handles temporary collaborator
cws approval review req-202406-019

# Temporary Access Request: req-202406-019
# Requested by: Dr. Patricia Smith (PI)
# New member: Dr. Kim (dr.kim@external.edu)
# Project: NIH-R01-2023
# Duration: 3 months (July 1 - Sept 30, 2024)
# Budget allocation: $300/month
# Justification: "Collaboration on RNA-editing project, visiting scholar"
#
# This requires PI approval (>$200/month allocation)
# Status: Pending patricia.smith@university.edu

# Dr. Torres adds recommendation
cws approval comment req-202406-019 \
  "Dr. Kim has good track record from previous collaboration. Recommend approval with standard member permissions."
```

### Month 11: Grant Period Ending

```bash
# May 1 (2 months before end): Automated warning
# Email to Dr. Smith:
#
# Subject: Project Ending Soon: NIH-R01-2023 (60 days)
#
# Your project "NIH-R01-2023" will end on June 30, 2024 (60 days).
#
# Current status:
# - Budget: $22,340 / $24,000 (93%)
# - Remaining: $1,660 for 60 days
# - Active instances: 6
# - EFS volumes: 2 (1.2 TB)
#
# Recommended actions:
# 1. Plan data archival strategy
# 2. Complete pending experiments
# 3. Generate preliminary reports
# 4. Consider requesting no-cost extension if needed
#
# Archive checklist: cws project archive-plan NIH-R01-2023

# Dr. Smith reviews archive plan
cws project archive-plan "NIH-R01-2023"

# Archive Plan: NIH-R01-2023
# End date: June 30, 2024 (60 days)
#
# Current resources:
# - 6 active instances → Will auto-stop June 30 23:59
# - 2 EFS volumes (1.2 TB) → Will be snapshotted and archived to S3
# - 4 EBS volumes (500 GB) → Will be snapshotted
#
# Data archival:
# - EFS snapshots: $12/month for 7 years (compliance)
# - S3 Deep Archive: $3/month
# - Total archive cost: $15/month
#
# Member access:
# - 4 members will lose project access
# - Research user accounts: Preserved for 1 year
# - SSH keys: Revoked from project instances
#
# Reports generated:
# - Final cost report (PDF)
# - Member activity report
# - Resource utilization summary
# - Grant compliance documentation
#
# Timeline:
# May 30: Warning email to all members (30 days before)
# June 15: Final warning (15 days before)
# June 30: Auto-archive and freeze
#
# Approve plan? [y/N]: y

# June 30, 11:59 PM: Automated archival
# - All instances stopped
# - EFS snapshots created
# - Data archived to S3
# - Final reports generated
# - Project marked "Archived"

# July 1: Dr. Smith receives final report
cws project report "NIH-R01-2023" --final

# NIH R01-2023 Final Report
# Grant Period: July 1, 2023 - June 30, 2024
#
# Budget Performance:
# - Total budget: $24,000.00
# - Total spent: $23,450.20
# - Unused: $549.80 (2.3%)
# - Efficiency: 97.7% ✅
#
# Resource Utilization:
# - Total compute hours: 14,520
# - Average cost/hour: $1.61
# - Hibernation savings: $4,230 (15%)
# - Peak month: December 2023 ($2,340)
#
# By Member:
# ├─ Michael Torres: $9,340 / $9,600 (97%)
# │  └─ Efficiency: Excellent
# ├─ James Wilson: $4,250 / $4,800 (89%)
# │  └─ Efficiency: Good
# └─ Shared Resources: $9,860 / $9,600 (103%) ⚠️
#    └─ Note: Overage covered by unused member allocations
#
# Compliance:
# ✅ All expenses within approved budget
# ✅ No unauthorized resource types
# ✅ Audit trail complete (14,520 logged events)
# ✅ Data archived per university policy
```

---

## 📋 Feature Gap Analysis: Lab Environment

### Critical Missing Features

| Feature | Priority | User Impact | Blocks Scenario | Effort |
|---------|----------|-------------|-----------------|--------|
| **Hierarchical Sub-Budgets** | 🔴 Critical | PI can't delegate | Postdoc managing students | High |
| **Approval Workflows** | 🔴 Critical | No request/review process | Grad students, GPU access | High |
| **Time-Boxed Access** | 🟡 High | Manual collaborator mgmt | Visiting scholars | Medium |
| **Resource Quotas by Role** | 🟡 High | No instance limits | Prevent mistakes | Medium |
| **Grant Period Management** | 🟡 High | Manual project closure | End-of-grant chaos | Medium |
| **Approval Dashboard** | 🟢 Medium | Requests via email | Centralized management | Low |

### Key Workflow Gaps

| Workflow | Current State | Desired State | Priority |
|----------|---------------|---------------|----------|
| **New member onboarding** | Manual setup | Template-based provisioning | Medium |
| **Budget reallocation** | Manual tracking | Dynamic reallocation UI | Low |
| **Cross-project sharing** | Not supported | "Borrow from Discretionary" | Low |
| **Emergency overages** | No mechanism | PI emergency approval | High |
| **Audit trail** | Basic logs | Compliance-ready reports | High |

---

## 🎯 Priority Recommendations: Lab Environment

### Phase 1: Approval & Hierarchy (v0.7.0)
**Target**: Labs can delegate and approve resource requests

1. **Approval Workflow System** (3 weeks)
   - Request/approve/deny infrastructure
   - Email + CLI + TUI approval interface
   - Time-limited approvals
   - Audit trail

2. **Hierarchical Sub-Budgets** (2 weeks)
   - Budget delegation (postdoc allocates to students)
   - Nested budget tracking
   - Rollup reporting

3. **Resource Quotas** (1 week)
   - Per-role instance limits
   - Instance type restrictions
   - Cost-per-day caps

### Phase 2: Lab Management (v0.7.1)
**Target**: PIs can oversee labs with minimal effort

4. **Lab Dashboard** (2 weeks)
   - Organization-wide view
   - Pending approvals
   - Budget alerts
   - Active instances by project

5. **Time-Boxed Membership** (1 week)
   - Start/end dates for members
   - Auto-revoke on expiry
   - Pre-expiry warnings

### Phase 3: Grant Lifecycle (v0.8.0)
**Target**: Complete grant period management

6. **Project Lifecycle Management** (2 weeks)
   - Project start/end dates
   - Auto-freeze on end date
   - Data archival workflows

7. **Compliance Reporting** (1 week)
   - Grant-ready final reports
   - Audit trail exports
   - Cost allocation reports

---

## Success Metrics: Lab Environment

### PI Perspective (Dr. Smith)
- ✅ **Peace of Mind**: "I get alerts before problems, not after"
- ✅ **Time Savings**: "No more monthly budget spreadsheets - 2 hours/month saved"
- ✅ **Compliance**: "Grant office reports generate automatically"
- ✅ **Delegation**: "My postdocs manage their teams independently"

### Lab Manager Perspective (Dr. Torres)
- ✅ **Control**: "I can review and approve expensive requests in 30 seconds"
- ✅ **Visibility**: "Dashboard shows entire lab status at a glance"
- ✅ **Prevention**: "Grad students can't accidentally launch $500/day instances"

### Graduate Student Perspective (James & Maria)
- ✅ **Clarity**: "I always know my remaining budget"
- ✅ **Confidence**: "Approval process is fast, not bureaucratic"
- ✅ **Learning**: "I understand cloud costs better now"

### Technical Metrics
- 95% of approvals processed within 2 hours
- 98% of projects stay within budget
- 100% of grant-end dates trigger automated archival
- Average PI time managing lab: < 30 min/week

---

## Next Steps

1. **User Research**: Interview 3 PIs about current budget management pain
2. **Approval UI Mockups**: Design approval dashboard and email templates
3. **Technical Design**: Hierarchical budget schema, approval state machine
4. **Pilot Program**: Deploy with 1-2 friendly labs for feedback

**Estimated Timeline**: Approval & Hierarchy (Phase 1) → 6 weeks of development
