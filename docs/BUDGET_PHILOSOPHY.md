# Budget System Philosophy and Conceptual Model

**Document Purpose**: Explain the "why" and "when" behind Prism's v0.5.10 multi-budget system design, helping institutional administrators and research leaders understand the conceptual model before diving into technical implementation.

**Target Audience**: Principal Investigators, Institutional Administrators, Grant Managers, Department Heads

---

## Table of Contents

- [Overview: The Two-Tier Budget System](#overview-the-two-tier-budget-system)
- [Core Concepts](#core-concepts)
- [Design Philosophy](#design-philosophy)
- [Real-World Use Cases](#real-world-use-cases)
- [Comparison to Other Models](#comparison-to-other-models)
- [Mental Models for Different Users](#mental-models-for-different-users)
- [Design Trade-offs](#design-trade-offs)

---

## Overview: The Two-Tier Budget System

Prism's v0.5.10 budget system enables flexible, many-to-many relationships between funding sources and research projects through a **two-tier architecture**:

```
┌─────────────────────┐
│  Budget Pools       │  ← Funding sources (grants, departments, credits)
│  (Tier 1)           │
└─────────┬───────────┘
          │
          │ Many-to-Many
          │ Relationships
          │
┌─────────▼───────────┐
│  Allocations        │  ← The connective tissue
│  (Tier 2)           │
└─────────┬───────────┘
          │
          │
          │
┌─────────▼───────────┐
│  Projects           │  ← Research projects with workspaces
│  (Research Work)    │
└─────────────────────┘
```

**Key Insight**: The system separates **funding sources** (Budget Pools) from **spending contexts** (Projects) with **Allocations** as the flexible connector, enabling the same budget to fund multiple projects OR the same project to receive funding from multiple sources.

---

## Core Concepts

### 1. Budget Pools (Tier 1)

**What**: A funding source with a total amount, time period, and tracking of allocated vs spent amounts.

**Examples**:
- "NSF Grant CISE-2024-12345" ($50,000, 3 years)
- "CS Department Q1 2026 Budget" ($10,000, quarterly)
- "AWS Research Credits Promo 2026" ($5,000, 1 year)

**Purpose**: Represent real-world funding sources that need to be managed, tracked, and reported on independently.

**Characteristics**:
- Have their own lifecycle (start date, end date)
- Track total amount, allocated amount, spent amount
- Can fund zero, one, or many projects
- Created and managed by budget administrators (PIs, department heads)

### 2. Project Budget Allocations (Tier 2)

**What**: A specific amount of money from a Budget Pool allocated to a specific Project, with independent tracking, alerts, and backup funding options.

**Examples**:
- "$15,000 from NSF Grant → Climate Model Project"
- "$5,000 from Department Budget → Student Pilot Studies"
- "$2,000 from AWS Credits → ML Training Project"

**Purpose**: Enable fine-grained control over how much of each budget can be spent by each project, with project-specific spending limits and backup funding.

**Characteristics**:
- Links ONE budget pool to ONE project
- Has its own allocated amount (subset of budget pool)
- Tracks spending independently
- Can have project-specific alert thresholds
- Can specify backup funding source for exhaustion handling
- The many-to-many magic happens here

### 3. Projects (Research Work)

**What**: A collection of workspaces, volumes, and users organized around a research goal, with optional default allocation for frictionless resource launching.

**Examples**:
- "Climate Model Development" (3 workspaces, 2 volumes)
- "Student Pilot Studies" (10 small workspaces for coursework)
- "GPU Training Experiments" (1 p3.8xlarge, 5TB storage)

**Purpose**: Organize resources and collaborators around research objectives, with automatic funding selection for simplified workflows.

**Characteristics**:
- Can have zero, one, or many funding allocations
- Has a DefaultAllocationID for automatic funding selection
- Members launch workspaces without thinking about funding
- Owner/Admins manage which budget sources fund the project

---

## Design Philosophy

### Why the Two-Tier System?

**Problem**: Research funding is complex and doesn't fit simple 1:1 relationships.

**Real-World Scenarios That Drove Design**:

1. **One Grant, Many Projects**:
   - NSF grant funds PI's 3 separate research projects
   - Each project needs its own spending limit
   - Grant requires per-project cost reporting
   - **Solution**: 1 Budget Pool → 3 Allocations → 3 Projects

2. **Multi-Source Project**:
   - Large project funded by NSF ($50K) + DOE ($30K) + Institution ($10K)
   - Need to track spending per funding source
   - Each source has different reporting requirements
   - **Solution**: 3 Budget Pools → 3 Allocations → 1 Project

3. **Department-Wide Budget**:
   - CS Department Q1 budget funds 20 student projects
   - Each student project has $500 spending limit
   - Department needs to track total spending and per-project breakdown
   - **Solution**: 1 Budget Pool → 20 Allocations → 20 Projects

**Design Decision**: A two-tier system with Allocations as the flexible connector enables ALL these scenarios without forcing users into a rigid structure.

### Why Not Simple 1:1 Budgets?

**Previous Prism versions** (pre-v0.5.10) had simple 1:1 budgets:
- Each project had exactly one budget
- Budget was tightly coupled to project

**Limitations**:
- ❌ Couldn't allocate one grant across multiple projects
- ❌ Couldn't combine multiple funding sources for one project
- ❌ Couldn't enforce per-project limits within shared budget
- ❌ No way to handle budget transitions (grant exhaustion → backup funding)

**v0.5.10 Solution**: Many-to-many relationships via Allocations enable flexible real-world funding structures while maintaining clear accounting.

### Separation of Concerns

**Key Principle**: Budget management is separate from project management.

**Why This Matters**:

| Concern | Who Manages | What They Control |
|---------|-------------|-------------------|
| **Budget Pools** | PIs, Grant Managers | Total funding, period, reporting |
| **Allocations** | PIs, Project Owners | Per-project limits, backup funding |
| **Projects** | Project Owners, Admins | Workspaces, members, resources |
| **Resource Launch** | Project Members | Launch workspaces (funding automatic) |

**Benefit**: Each role focuses on their domain without needing to understand the full complexity:
- **Students** launch workspaces → funding handled automatically via DefaultAllocation
- **Project Owners** manage project resources → budget selected once during setup
- **PIs** manage grant allocations → oversee spending across research group
- **Admins** monitor institution-wide spending → roll up costs by budget/project/user

---

## Design Decisions & Rationale

### 1. Default Allocation Model

**Design**: Projects have a `DefaultAllocationID` that's automatically used when members launch workspaces without specifying funding.

**Why**:
- **Reduces friction**: Students don't need to understand funding complexity
- **PI control**: PI sets default once, students just use project
- **Flexibility preserved**: Advanced users can still specify `--funding` if needed

**Example Workflow**:
```bash
# PI sets up project with default funding
prism allocation create \
  --budget "NSF Grant" \
  --project "Climate Model" \
  --amount 15000 \
  --default

# Student just launches workspace (funding automatic)
prism workspace launch python-ml my-analysis --project "Climate Model"
# → Automatically uses NSF Grant allocation

# Advanced: Explicitly use different funding
prism workspace launch python-ml gpu-training \
  --project "Climate Model" \
  --funding "AWS Research Credits"
```

### 2. Backup Funding Philosophy

**Design**: Allocations can specify a `BackupAllocationID` that's automatically used when the primary allocation is exhausted.

**Why This Matters**: Research doesn't stop when a grant runs out - researchers need bridge funding, emergency reserves, or department safety nets.

**Philosophy**: Continuity over disruption.

**Without Backup Funding**:
1. Primary allocation exhausts at 2 AM during GPU training run
2. Prism hibernates all project workspaces
3. Student loses 6 hours of training progress
4. Email sent to PI (who's asleep)
5. PI manually switches funding next morning
6. Student restarts training (additional cost)

**With Backup Funding**:
1. Primary allocation exhausts at 2 AM during GPU training
2. Prism automatically switches to backup (department emergency fund)
3. Training continues uninterrupted
4. Email sent to PI (handled at normal hours)
5. PI reviews backup usage and reallocates if needed
6. Research productivity maintained

**Real-World Use Cases**:
- **Grant Transitions**: Bridge between NIH R01 ending and renewal
- **Cost Overruns**: Unexpected compute needs beyond original allocation
- **Multi-Phase Projects**: Finish Phase 1 with backup while Phase 2 grant pending

**Configuration**:
```bash
prism allocation create \
  --budget "NSF Grant" \
  --project "Climate Model" \
  --amount 15000 \
  --backup "Department Emergency Fund"
```

### 3. Reallocation Audit Trail

**Design**: Every reallocation (moving funds between allocations) requires a reason and creates an immutable audit record.

**Why**:
- **Grant Compliance**: NSF/NIH require justification for budget changes
- **Institutional Oversight**: Administrators need to understand spending patterns
- **Retrospective Analysis**: "Why did we move $5K from Project A to Project B?"
- **Dispute Resolution**: Clear history if questions arise during audits

**Example**:
```bash
prism allocation reallocate \
  --from "NSF Grant → ML Project" \
  --to "NSF Grant → Climate Project" \
  --amount 5000 \
  --reason "ML project completed ahead of schedule, climate project needs GPU resources"
```

**Audit Trail**:
```
Reallocation #47: ML Project → Climate Project
Amount: $5,000
Date: 2026-01-15 14:23 UTC
By: PI Sarah Chen (sarah@university.edu)
Reason: ML project completed ahead of schedule, climate project needs GPU resources

Budget Impact:
  NSF Grant CISE-2024-12345:
    - ML Project allocation: $20,000 → $15,000 (-$5,000)
    - Climate Project allocation: $15,000 → $20,000 (+$5,000)
    - Total budget unchanged: $50,000
```

### 4. Real-Time Tracking

**Design**: Spending is tracked immediately when resources launch/resize/stop, not in batch overnight jobs.

**Why**:
- **Prevents Over-Spending**: Know instantly if allocation is exhausted
- **Pre-Launch Validation**: Block launches that would exceed budget
- **Real-Time Dashboards**: Current spending visible without delay
- **Immediate Alerts**: Notification sent when threshold crossed, not hours later

**Technical Implementation**: Cost tracking updates on every state change:
- Instance launched → add hourly cost × uptime
- Instance stopped → stop accumulating compute cost (storage continues)
- Volume attached → add daily storage cost
- Instance terminated → remove from active cost tracking

**Benefit**: PIs and administrators have immediate visibility into spending, enabling proactive budget management instead of reactive responses to monthly AWS bills.

---

## Real-World Use Cases

### Use Case 1: Single Grant Funding Multiple Projects

**Scenario**: PI receives NSF CISE grant ($50,000, 3 years) funding 3 distinct research projects with different collaborators and cost profiles.

**Requirements**:
- Track spending per project for progress reports
- Enforce per-project spending limits ($20K, $20K, $10K)
- Prevent one project from consuming entire grant
- Reallocate funds between projects as priorities shift

**Prism Solution**:
```bash
# 1. Create budget pool (grant)
prism budget create \
  --name "NSF CISE-2024-12345" \
  --amount 50000 \
  --period "2024-03-01 to 2027-02-28"

# 2. Allocate to projects with limits
prism allocation create --budget "NSF CISE-2024-12345" \
  --project "ML Optimization" --amount 20000 --default

prism allocation create --budget "NSF CISE-2024-12345" \
  --project "Climate Modeling" --amount 20000 --default

prism allocation create --budget "NSF CISE-2024-12345" \
  --project "Student Pilots" --amount 10000 --default

# 3. Projects launch independently
# Each project's spending tracked separately
# Grant report shows per-project breakdown
```

**Result**:
- Project 1 cannot spend more than $20K (enforced)
- Each project's spending tracked independently
- PI can reallocate if one project needs more budget
- NSF report shows clear per-project cost attribution

### Use Case 2: Multi-Source Project Funding

**Scenario**: Large climate modeling project funded by NSF ($50K) + DOE ($30K) + University matching funds ($10K). Each funding source has different reporting and compliance requirements.

**Requirements**:
- Track spending per funding source
- Meet NSF quarterly reporting deadlines
- Meet DOE annual reporting requirements
- Track university match spending for institutional records

**Prism Solution**:
```bash
# 1. Create budget pools for each source
prism budget create --name "NSF Climate Grant" --amount 50000
prism budget create --name "DOE Energy Research" --amount 30000
prism budget create --name "University Matching" --amount 10000

# 2. Allocate all three to single project
prism allocation create --budget "NSF Climate Grant" \
  --project "Climate Modeling" --amount 50000 --default

prism allocation create --budget "DOE Energy Research" \
  --project "Climate Modeling" --amount 30000

prism allocation create --budget "University Matching" \
  --project "Climate Modeling" --amount 10000

# 3. Launch resources with explicit funding selection
prism workspace launch python-ml climate-sim-01 \
  --project "Climate Modeling" \
  --funding "NSF Climate Grant"  # Use NSF funds

prism workspace launch python-ml climate-sim-02 \
  --project "Climate Modeling" \
  --funding "DOE Energy Research"  # Use DOE funds
```

**Result**:
- Single project with multiple funding sources
- Spending tracked per funding source for reporting
- Can exhaust one source and continue with others
- Clear attribution for compliance and audits

### Use Case 3: Department-Wide Student Budget

**Scenario**: CS Department allocates $10,000 quarterly budget for 20 student pilot projects in Advanced ML course. Each student project should have $500 spending limit.

**Requirements**:
- Each student project gets equal allocation
- Department tracks total spending vs budget
- Prevent individual student from exceeding $500
- Roll up costs for quarterly department reporting

**Prism Solution**:
```bash
# 1. Create department budget pool
prism budget create \
  --name "CS Dept Q1 2026 Student Computing" \
  --amount 10000 \
  --period "2026-01-01 to 2026-03-31"

# 2. Create allocations for each student project
for i in {1..20}; do
  prism allocation create \
    --budget "CS Dept Q1 2026 Student Computing" \
    --project "student-ml-project-$i" \
    --amount 500 \
    --default
done

# 3. Students launch workspaces (funding automatic)
# Student 5 launches workspace
prism workspace launch python-ml analysis --project "student-ml-project-5"
# → Automatically charged to their $500 allocation

# 4. Department administrator monitors total spending
prism budget show "CS Dept Q1 2026 Student Computing"
# Shows: 20 allocations, total spent, per-project breakdown
```

**Result**:
- Each student has $500 spending limit (enforced)
- Department sees total spending in real-time
- Per-project breakdown for grading/assessment
- Quarterly report shows total department compute costs

### Use Case 4: Grant Transition with Backup Funding

**Scenario**: PI's NIH R01 grant ends March 31st, but renewal doesn't start until April 15th. Department provides 2-week bridge funding ($2,000) to cover storage and minimal compute during gap.

**Requirements**:
- Continue research during grant transition
- Storage costs must not be interrupted (data loss risk)
- Minimal compute for urgent analysis
- Track bridge funding separately for institutional records

**Prism Solution**:
```bash
# 1. Set up backup funding for seamless transition
prism allocation create \
  --budget "Department Bridge Funding" \
  --project "Genomics Research" \
  --amount 2000

prism allocation update \
  --id "alloc-nih-r01-genomics" \
  --backup "Department Bridge Funding"

# 2. Primary allocation exhausts on March 30th
# → Prism automatically switches to backup
# → Email sent to PI: "Primary exhausted, using backup"
# → Research continues uninterrupted

# 3. Renewal grant starts April 15th
prism allocation create \
  --budget "NIH R01 Renewal 2026" \
  --project "Genomics Research" \
  --amount 75000 \
  --default

# 4. Update resources to use renewal funding
prism allocation switch \
  --project "Genomics Research" \
  --from "Department Bridge Funding" \
  --to "NIH R01 Renewal 2026"
```

**Result**:
- No research disruption during grant transition
- Bridge funding tracked separately for department
- Clear audit trail: Primary → Backup → Renewal
- Storage costs maintained throughout transition

---

## Comparison to Other Models

### Prism v0.5.10 vs Simple Per-Project Budgets

| Feature | Simple Budgets (pre-v0.5.10) | Prism v0.5.10 Multi-Budget |
|---------|------------------------------|----------------------------|
| **Grant → Projects** | 1 grant = 1 project only | 1 grant → many projects |
| **Project Funding** | Single budget source | Multiple budget sources |
| **Spending Limits** | Project-wide limit only | Per-allocation limits |
| **Budget Transitions** | Manual switch with downtime | Automatic backup funding |
| **Cost Attribution** | Single source only | Per-source tracking |
| **Reallocation** | Not possible | Between allocations with audit trail |
| **Department Budgets** | Difficult to manage | Native support |

**When Simple Budgets Work**: Solo researcher, single grant, one project.

**When Multi-Budget Essential**: Research groups, multiple grants, institutional deployments, teaching courses.

### Prism vs AWS Cost Allocation Tags

| Feature | AWS Cost Allocation Tags | Prism Multi-Budget System |
|---------|-------------------------|---------------------------|
| **Granularity** | Tag-based grouping after spending | Pre-allocation with enforcement |
| **Enforcement** | Retrospective only | Real-time pre-launch validation |
| **Spending Limits** | No built-in limits | Per-allocation limits enforced |
| **Budget Exhaustion** | Manual response to alerts | Automatic backup funding |
| **Reporting** | Monthly AWS Cost Explorer | Real-time dashboard |
| **Grant Compliance** | Manual tag → grant mapping | Native budget → project tracking |

**AWS Tags Alone**: Good for cost visibility, insufficient for budget enforcement and research workflows.

**Prism Advantage**: Real-time enforcement + automatic handling + research-optimized workflows.

### Prism vs Traditional Grant Accounting Systems

| Feature | University Grant System | Prism Budget System |
|---------|------------------------|---------------------|
| **Scope** | All grant expenses | Cloud computing only |
| **Granularity** | Purchase orders, invoices | Per-resource tracking |
| **Enforcement** | Monthly reconciliation | Real-time pre-launch |
| **Researcher Visibility** | Limited/none | Real-time dashboard |
| **Budget Reallocation** | Formal paperwork, weeks | Instant with audit trail |
| **Multi-Source Projects** | Complex manual tracking | Native many-to-many support |

**Complementary Systems**: Prism provides real-time cloud budget management that feeds into traditional grant accounting for institutional records.

---

## Mental Models for Different Users

### For Students/Researchers (Project Members)

**Your Mental Model**: "I just use my project. Funding is handled."

**What You See**:
```bash
# Launch workspace - funding automatic
prism workspace launch python-ml my-analysis --project "Climate Model"

# Don't need to think about:
# - Which grant is paying
# - How much budget is left
# - Budget allocation policies
```

**What You Need to Know**:
- Your project name
- What template you need
- When to stop instances to save costs

**What You Don't Need to Know**:
- Budget pool vs allocation concepts
- Which funding source your project uses
- How backup funding works

**Benefit**: Frictionless research computing without budget anxiety.

### For PIs (Project Owners)

**Your Mental Model**: "I manage grant allocations across my research group."

**What You See**:
```bash
# Create budget from grant
prism budget create --name "NSF CISE Grant" --amount 50000

# Allocate to projects with limits
prism allocation create \
  --budget "NSF CISE Grant" \
  --project "Climate Model" \
  --amount 20000 \
  --backup "Department Emergency" \
  --default

# Monitor spending
prism budget status "NSF CISE Grant"
# → Shows: Per-project breakdown, remaining funds, spending trends
```

**What You Need to Know**:
- How much each project should be allocated
- When to reallocate funds between projects
- Setting up backup funding for transitions
- Monitoring spending trends

**What Prism Handles for You**:
- Real-time spending tracking
- Budget exhaustion handling
- Per-project enforcement
- Audit trail for grant reporting

**Benefit**: Focus on research priorities, not budget micromanagement.

### For Institutional Admins (Department Heads, Grant Managers)

**Your Mental Model**: "I oversee all research spending and ensure compliance."

**What You See**:
```bash
# Overview of all institutional budgets
prism budget list --all
# → Shows: All grants, department budgets, credit programs

# Per-budget details
prism budget show "NSF CISE Grant"
# → Projects funded, allocations, spending, timeline

# Roll-up reports
prism budget report --department "Computer Science" --quarter Q1-2026
# → Total spending, per-PI breakdown, per-grant attribution

# Compliance exports
prism budget export --budget "NSF CISE Grant" --format nsf-report
# → Grant-compliant cost report for NSF submission
```

**What You Need to Know**:
- Which PIs have which grants
- Department-wide spending trends
- Compliance and reporting requirements
- Budget policies and spending limits

**What Prism Handles for You**:
- Real-time institutional spending visibility
- Per-grant cost tracking and attribution
- Audit trails for compliance
- Automated alerts for budget issues
- Grant-specific reporting formats

**Benefit**: Institution-wide visibility with minimal administrative overhead.

---

## Design Trade-offs

### Flexibility vs Simplicity

**Trade-off**: Two-tier system (Budget → Allocation → Project) is more complex than simple 1:1 budgets.

**Why We Chose Flexibility**:
- Real-world research funding is inherently complex
- Forcing simple 1:1 model creates workarounds (multiple Prism profiles, manual tracking)
- Complexity hidden from end users (students) via DefaultAllocation
- PIs dealing with grants already understand this funding structure

**Mitigation**:
- Default allocation makes simple case simple: Students never see complexity
- Documentation provides clear mental models for each user role
- GUI wizard-based setup for common scenarios

### Real-Time Tracking vs Performance

**Trade-off**: Real-time cost tracking adds latency to every instance state change.

**Why We Chose Real-Time**:
- Budget exhaustion must be detected immediately to prevent over-spending
- Pre-launch validation requires current spending data
- Researchers need instant feedback on budget status
- Academic budgets are typically small enough for real-time performance

**Mitigation**:
- Cached spending amounts updated incrementally (not full recalculation)
- Async notification system doesn't block operations
- Background job handles complex analytics (trends, predictions)

### Many-to-Many vs Simple Hierarchy

**Trade-off**: Many-to-many relationships increase database complexity and potential for configuration errors.

**Why We Chose Many-to-Many**:
- Multi-source projects are common in large research efforts
- Department-wide budgets fund many independent projects
- Grant transitions require backup funding from different sources
- Simple hierarchy forces unnatural workarounds

**Mitigation**:
- Strong validation prevents orphaned allocations
- Clear error messages for misconfiguration
- GUI prevents invalid allocation relationships
- Documentation includes common patterns and anti-patterns

---

## Summary: The Prism Budget Philosophy

**Core Principle**: Budget management should reflect the **real complexity of research funding** while **hiding that complexity from researchers** who just want to do science.

**Key Design Tenets**:

1. **Flexible Relationships**: Many-to-many via allocations enables real-world funding structures
2. **Separation of Concerns**: Budget management separate from project management
3. **Frictionless for Users**: Default allocations eliminate funding decisions for most launches
4. **Real-Time Enforcement**: Prevent over-spending before it happens
5. **Continuity Over Disruption**: Backup funding prevents research interruption
6. **Audit Trail**: Every budget change tracked for compliance and retrospection
7. **Multi-Level Visibility**: Students see simplicity, PIs see control, admins see oversight

**Result**: A budget system that's as flexible as academic funding requires, as simple as researchers need, and as powerful as administrators demand.

---

## Related Documentation

- **BUDGET_PHILOSOPHY.md**: Temporal surplus/deficit tracking and burst budgeting
- **RESOURCE_TAGGING.md**: AWS cost allocation and zombie resource cleanup
- **User Guide** (planned): Step-by-step budget setup and management
- **API Reference**: REST API endpoints for budget operations
- **Release Plan**: docs/releases/RELEASE_PLAN_v0.5.10.md

---

## Questions? Feedback?

This document describes the **philosophy and conceptual model** of Prism's budget system. For implementation details, see the related documentation above.

**Found this helpful?** Share feedback via GitHub issues or contact the Prism team.

**Missing a use case?** We'd love to hear about your institution's funding structure - open an issue to discuss.

---


## The Core Value Proposition

**Prism's budget banking system is the killer feature that makes cloud computing viable for academic research.**

Unlike traditional owned hardware (fixed amortization costs), cloud computing allows researchers to "bank" unused budget for future bursting. This document explains how Prism leverages this fundamental cloud advantage to eliminate the primary hesitation academics have about AWS.

---

## The Problem Prism Solves

Academic researchers receive grant funding with fixed budgets over specific time periods (quarterly, annually, or grant-specific periods). They face several challenges:

1. **Budget Shortfall Fear**: "What if I run out of budget before the period ends?"
2. **Committed Costs Anxiety**: "Storage costs continue even when instances are stopped"
3. **Unpredictable Workloads**: Research requires bursting (intensive computation) followed by quiet periods (data analysis)
4. **Grant Reporting Requirements**: Need to demonstrate responsible budget management

Traditional cloud billing makes these problems worse because researchers lack visibility into their spending trajectory and can't easily predict whether they'll have budget through the end of their grant period.

---

## The Budget Banking Solution

### Core Concept: Surplus Accumulation

When you stop an instance, you don't just reduce your burn rate - you **bank the difference** between actual spend and target spend. This creates a **surplus** that can be used for bursting above the target rate when needed.

### Mathematical Foundation

```
Budget Period: $10,000 over 90 days (quarterly grant)
Target Burn Rate: $111.11/day

Week 1-2 (14 days): Light usage
- Running: 1× t3.medium ($0.04/hr) + 100GB EBS ($0.10/day) = $14.54/day
- Target: $111.11/day × 14 days = $1,555.54
- Actual: $14.54/day × 14 days = $203.56
- SURPLUS BANKED: $1,351.98

Week 3-4 (14 days): Data processing
- Running: 3× c5.4xlarge ($0.68/hr) + 500GB EBS ($0.50/day) = $49.34/day
- Target: $111.11/day × 14 days = $1,555.54
- Actual: $49.34/day × 14 days = $690.76
- ADDITIONAL SURPLUS: $864.78
- TOTAL SURPLUS: $2,216.76

Week 5 (7 days): BURST - GPU Training
- Running: 2× p3.2xlarge ($3.06/hr) + 1TB EBS ($1.00/day) = $147.88/day
- Target: $111.11/day × 7 days = $777.77
- Actual: $147.88/day × 7 days = $1,035.16
- Surplus used: $257.39
- REMAINING SURPLUS: $1,959.37

Week 6-13 (56 days): Analysis & writing
- Stopped: 0× instances + 1TB EBS ($1.00/day) = $1.00/day
- Target: $111.11/day × 56 days = $6,222.16
- Actual: $1.00/day × 56 days = $56.00
- MASSIVE SURPLUS: $6,166.16
- TOTAL SURPLUS: $8,125.53

Quarter End Budget Status:
- Total Spent: $1,985.48
- Total Budget: $10,000
- Remaining: $8,014.52
- Researcher stayed WELL under budget
- Storage costs fully covered
- GPU burst easily accommodated
```

### Key Advantages

**1. Burst Capability**
- Bank budget during quiet periods (writing, data analysis)
- Burst during compute-intensive periods (model training, simulations)
- No anxiety about temporary overspending

**2. Storage Cost Protection**
- EBS, EFS, S3 costs continue regardless of instance state
- Surplus ensures storage always covered
- Optional "cushion" provides safety buffer beyond period end

**3. Shortfall Prevention**
- Real-time visibility: "Are you on track?"
- Predictive analytics: "Will you hit zero before period end?"
- Early warning: "Adjust usage now to avoid shortfall"

**4. Cloud Advantage Realization**
- Unlike owned hardware (use or lose), cloud lets you bank savings
- Stopped instance = banked budget for future use
- Aligns perfectly with research workflow patterns

---

## The Safety Net: Budget Cushion

Researchers can optionally configure a "cushion" - reserved budget beyond the period end date to ensure storage continuity even if compute budget exhausted.

```
Budget: $10,000 for 90 days
Cushion: 14 days (2 weeks)
Storage Costs: $5/day (EBS + EFS + S3)

Calculation:
- Reserve for cushion: $5/day × 14 days = $70
- Available for period: $10,000 - $70 = $9,930
- Target burn rate: $9,930 / 90 days = $110.33/day

If budget exhausted on day 90:
- Cushion covers storage for days 91-104
- Researcher has 2 weeks to secure additional funding
- No data loss, no emergency scrambling
```

### Cushion Use Cases

1. **Grant Transitions**: Bridge funding between grant periods
2. **Emergency Reserve**: Unexpected expenses in final weeks
3. **Storage Continuity**: Ensure data accessible during extension requests
4. **Peace of Mind**: "Safety blanket" for budget anxiety

---

## Academic-Specific Features

### GDEW (Global Data Egress Waiver) Integration

AWS provides academic researchers with automatic credits for up to 15% of monthly spend on egress charges. For 99% of research workloads, this effectively eliminates egress costs.

```
Monthly Spend: $3,333 (one month of quarterly budget)
Egress Charges: $250 (data downloads, inter-region transfers)
GDEW Credit: min($250, $3,333 × 0.15) = min($250, $500) = $250
Net Egress Cost: $0

Effective Impact:
- Egress fully covered by GDEW
- No impact on research budget
- Researcher can freely download results
```

**Prism GDEW Tracking**:
- Optional checkbox: "I have AWS GDEW"
- Automatic monthly tracking window
- Real-time estimation of available credits
- Alerts when approaching 15% threshold
- Proactive guidance: "Egress on track to be covered"

### Discount & Credit Support

**Enterprise Discount Programs (EDP)**:
- Across-the-board percentage discount
- Applied to all services
- Discoverable via AWS Cost Explorer API

**Private Pricing Agreements (PPA)**:
- Service-specific discounts (often S3, EC2 Reserved Instances)
- Custom pricing negotiated with AWS
- Applied automatically in billing

**Time-Limited Credits**:
- Research grants, promotional credits, support credits
- Have start date, expiration date, remaining balance
- Discoverable via AWS Billing API
- Track utilization: "You have $500 in credits expiring next month"

---

## Implementation Priorities

### Phase 1: Foundation (v0.5.9)
1. Budget period tracking (start date, end date, total budget)
2. Target burn rate calculation
3. Surplus/deficit tracking
4. Basic "on track" indicator

### Phase 2: Storage Costs (v0.5.10)
1. EBS volume cost tracking (independent of instances)
2. EFS filesystem usage monitoring
3. S3 bucket storage tracking
4. Committed costs calculation

### Phase 3: Analytics & Prediction (v0.5.11)
1. Burn rate trend analysis
2. Shortfall prediction engine
3. Surplus projection
4. Budget cushion configuration

### Phase 4: Academic Features (v0.5.12)
1. GDEW tracking and estimation
2. Discount/credit discovery and application
3. Calendar month alignment
4. Grant reporting exports

### Phase 5: Alerting & Integration (v0.5.13)
1. Unified alerting system (apprise-go integration)
2. Alert templates (burn rate, surplus, shortfall, GDEW)
3. Multi-channel delivery (email, Slack, MS Teams, etc.)
4. Customizable thresholds

---

## Why This Matters

**Quote from the creator:**

> "This is SO CORE to this whole project it is hard to convey to you. This alone breaks the hesitation people have with the cloud (AWS). Plus, if you add in an optional cushion... folks have a safety blanket."

Budget banking transforms cloud computing from a source of anxiety ("Am I spending too much?") to a source of empowerment ("I have surplus to burst when I need it"). It's the difference between:

- **Traditional approach**: "I'm afraid to launch a GPU instance because I might run out of budget"
- **Prism approach**: "I have $2,000 in surplus, so I can safely burst for 3 days of GPU training"

This psychological shift - from fear to confidence - is what makes Prism viable for academic research.

---

## Technical Architecture

See related issues for implementation details:
- #TBD: Budget Period & Burn Rate Foundation
- #TBD: Surplus/Banking Calculation Engine
- #TBD: EBS Cost Tracking
- #TBD: EFS Cost Tracking
- #TBD: S3 Cost Tracking & Egress
- #TBD: GDEW Integration
- #TBD: Discount & Credit Discovery
- #TBD: Predictive Analytics & Shortfall Detection
- #TBD: Unified Alerting System (apprise-go)

---

## Appendix: Real-World Example

**Dr. Sarah Chen - Computational Biology Research**

Grant: NIH R01 ($15,000 AWS budget for 12 months)

**Month 1-3: Dataset preparation**
- 1× t3.large running 24/7 ($60/month)
- 2TB EBS storage ($200/month)
- Target: $1,250/month
- Actual: $260/month
- **Surplus banked: $2,970**

**Month 4: Model training burst**
- 4× p3.8xlarge running 10 days ($5,000)
- 5TB EBS temporary storage ($500)
- Target: $1,250
- Actual: $5,500
- **Surplus used: $4,250**
- **Remaining surplus: +$720** (still in surplus!)

**Month 5-12: Analysis & writing**
- All instances stopped
- 2TB EBS storage only ($200/month × 8 = $1,600)
- Target: $1,250/month × 8 = $10,000
- Actual: $1,600
- **Additional surplus: $8,400**

**End of grant:**
- Total spent: $7,360
- Total budget: $15,000
- **Under budget by $7,640 (51%)**
- Researcher demonstrated responsible budget management
- All data preserved on EBS throughout entire period
- GPU burst completed without anxiety
- Grant renewal shows efficient resource utilization

**Without budget banking visibility:**
- Dr. Chen would have hesitated to launch GPUs
- Training might have been delayed or compromised
- Anxiety about storage costs throughout quiet periods
- Uncertainty about budget status until monthly bill

**With Prism budget banking:**
- Confidence to burst when needed
- Clear visibility into surplus
- No anxiety during quiet periods
- Real-time "on track" status
- Professional grant reporting

This is the transformation Prism enables.
