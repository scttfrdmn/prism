# Scenario 4: Conference Workshop

## Persona: Dr. Alex Rivera - Workshop Instructor

**Background**:
- Assistant Professor, Machine Learning researcher
- Accepted to teach 3-hour workshop at NeurIPS 2025
- Workshop: "Hands-on Deep Learning with PyTorch"
- Expected attendance: 40-60 participants (international)
- Budget: $200 from conference organizers (one-time allocation)
- **Critical constraint**: Must work perfectly on first try - no second chances

**Pain Points**:
- Participants arrive with varying laptop configurations (Windows/Mac/Linux)
- Limited time for troubleshooting (workshop starts in 90 minutes)
- Need identical environments for all participants to follow along
- Budget must cover entire workshop duration + buffer
- International participants in multiple timezones for pre-workshop prep
- **Must auto-terminate** - can't rely on participants to clean up afterwards

**Workshop Structure**:
- **Week before**: Send invitation links to registered participants
- **Day before**: Early access for testing (24-hour window)
- **Workshop day**: 3-hour hands-on session
- **Auto-cleanup**: Terminate all instances 3 hours after workshop ends

---

## Current State (v0.5.5): What Works Today

### ✅ Pre-Workshop Setup (1 Week Before)

```bash
# Alex sets up workshop environment
cws profile create neurips-workshop --aws-profile alex-research --region us-west-2

# Create template-restricted project for workshop
cws project create neurips-dl-workshop \
  --budget 200 \
  --description "NeurIPS 2025: Deep Learning Workshop" \
  --alert-threshold 80

# Generate batch invitations for 60 participants
cat > workshop_participants.csv << EOF
Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Participant_01,read_only,7,no,no,yes,2
Participant_02,read_only,7,no,no,yes,2
Participant_03,read_only,7,no,no,yes,2
[... 57 more participants ...]
EOF

# Create invitations with basic policy restrictions
cws profiles invitations batch-create \
  --csv-file workshop_participants.csv \
  --output-file invitation_codes.csv \
  --include-encoded
```

**Current capabilities**:
- ✅ Batch invitation generation (60 participants in seconds)
- ✅ Time-boxed access (7-day expiration)
- ✅ **Access extension**: Can extend for additional day(s) so participants can continue working
- ✅ Device binding (prevents casual sharing)
- ✅ Budget allocation ($200 total)
- ✅ Basic policy restrictions (template whitelist)

💡 **Workshop Extension Example**: After 3-hour workshop ends, Alex can extend access for 24 hours:
```bash
cws profiles invitations extend neurips-workshop --add-days 1
# All 60 participants get automatic 24-hour extension
# Great for: Homework completion, extended tutorials, follow-up work
```

### ✅ Day Before Workshop (Early Access Testing)

```bash
# Participants receive email with invitation link
# They accept invitation and test their environment

# Participant workflow:
cws profiles invitations accept <INVITATION-CODE> neurips-workshop
cws launch pytorch-ml workshop-test --size S

# Alex monitors early access
cws project instances neurips-dl-workshop
# Output:
# ✅ 12 participants tested successfully
# ⚠️  3 participants having issues (Alex contacts them)
# 💰 Current spend: $4.20 (within budget)
```

---

## ⚠️ Current Pain Points: What Doesn't Work

### ❌ Problem 1: No Automatic Instance Termination

**Scenario**: Workshop ends at 3:00 PM, instances should terminate at 6:00 PM

**What should happen** (MISSING):
```bash
# Alex launches instances with auto-terminate timer
cws launch pytorch-ml workshop-instance --hours 6

# CloudWorkstation output:
# ✅ Instance launching: workshop-instance
# ⏰ Auto-terminate scheduled: 6 hours from now (6:00 PM)
# 📊 Cost for 6 hours: $3.20
# 🔔 Warning will be sent 30 minutes before termination
```

**Current workaround**: Alex must manually stop 60 instances or rely on participants
**Risk**: If forgotten, $200 budget exhausted in 3 days

### ❌ Problem 2: No Template Whitelisting at Invitation Level

**Scenario**: Participants should ONLY be able to launch PyTorch ML template

**What should happen** (MISSING):
```bash
# Create invitations with template restrictions
cws profiles invitations batch-create \
  --csv-file participants.csv \
  --template-whitelist "PyTorch Machine Learning" \
  --max-instance-type "t3.medium" \
  --output-file invitations.csv

# When participant tries wrong template:
participant$ cws launch gpu-ml-workstation expensive-instance
# ❌ Error: Template 'gpu-ml-workstation' not allowed by your invitation policy
#    Allowed templates: ["PyTorch Machine Learning"]
#
#    This is a workshop environment with restricted templates.
#    Please use: cws launch "PyTorch Machine Learning" my-instance
```

**Current workaround**: Trust participants + budget alerts
**Risk**: Single participant launches GPU instance → $600/day → budget blown in 8 hours

### ❌ Problem 3: No Bulk Launch for Pre-Provisioning

**Scenario**: Workshop starts at 9:00 AM, Alex wants all environments ready at 8:45 AM

**What should happen** (MISSING):
```bash
# Night before workshop: Pre-provision all instances
cws project bulk-launch neurips-dl-workshop \
  --template "PyTorch Machine Learning" \
  --count 60 \
  --name-pattern "workshop-{01-60}" \
  --start-time "2025-12-08T08:45:00" \
  --terminate-hours 6

# Output:
# 🚀 Scheduling 60 instance launches for Dec 8, 8:45 AM
# 📊 Estimated cost: $192.00 (within $200 budget ✅)
# ⏰ All instances will auto-terminate at 2:45 PM (3-hour workshop)
#
# 💡 Effective Cost Analysis:
#    24/7 assumption: $2.40/hour × 60 instances × 24 hours = $3,456
#    Actual workshop cost: $2.40/hour × 60 instances × 3 hours = $432
#    Your cost with auto-terminate: $192 (early terminations banked immediately)
#    Savings: $240 banked in real-time as participants finish early!
#
# Instance name assignments:
# - Participant_01 → workshop-01
# - Participant_02 → workshop-02
# ...

# 8:45 AM on workshop day - all instances auto-launch
# 9:00 AM - participants arrive, instances are ready
```

> **💡 GUI Note**: Workshop scheduling available in GUI Projects tab with calendar view - *coming soon in v0.6.0*

**Current workaround**: Participants launch on-demand (slow, error-prone)
**Impact**: First 30 minutes wasted on environment setup

### ❌ Problem 4: No Real-Time Workshop Dashboard

**Scenario**: During workshop, Alex needs to see participant progress at a glance

**What should happen** (MISSING):
```bash
cws workshop dashboard neurips-dl-workshop

# Terminal dashboard (live updates):
# ┌─────────────────────────────────────────────────────────┐
# │ NeurIPS DL Workshop - Live Dashboard                   │
# │                                                         │
# │ Participants:     58 / 60 active                       │
# │ Instances:        58 running, 2 stopped                │
# │ Avg Uptime:       1h 23m (82 compute hours total)     │
# │                                                         │
# │ Budget:          $38.40 / $200.00 (19%) ✅            │
# │ Available:       $161.60 (real-time as terminations happen) │
# │ Effective cost:  $0.47/hour (vs $2.40/hour 24/7)     │
# │                                                         │
# │ 💡 Real-time banking: 2 early finishers already banked $4.80! │
# │ Time Remaining:   1h 37m until auto-terminate          │
# │                                                         │
# │ Participants Needing Help:                             │
# │   ⚠️  workshop-27: Instance stopped (needs restart)    │
# │   ⚠️  workshop-43: High error rate (check logs)        │
# │                                                         │
# │ Cost by Status:                                         │
# │   Running:  $38.40/hr (58 instances)                   │
# │   Stopped:  $0.00/hr (2 instances)                     │
# │                                                         │
# │ Refresh: Every 30s | Press 'q' to quit                 │
# └─────────────────────────────────────────────────────────┘
```

> **💡 GUI Note**: Live workshop dashboard available in GUI with real-time participant status - *coming soon in v0.6.0*

**Current workaround**: Manual `cws list` + `cws project instances` polling
**Impact**: Can't proactively help struggling participants

### ❌ Problem 5: No Post-Workshop Data Preservation

**Scenario**: Participants want to keep their workshop code after instances terminate

**What should happen** (MISSING):
```bash
# 30 minutes before auto-terminate, participants receive email:
#
# Subject: ⏰ Workshop Instance Terminating in 30 Minutes
#
# Your workshop instance will terminate at 6:00 PM (in 30 minutes).
#
# To preserve your work:
#
# 1. Download your notebook:
#    cws download workshop-instance ~/workshop-code.zip
#
# 2. Or snapshot your instance:
#    cws snapshot create workshop-instance my-workshop-work
#    (This will create a personal AMI - $2.50/month storage)
#
# After termination, you can recreate your environment:
#    cws launch-from-snapshot my-workshop-work restored-env

# Bulk download (instructor):
cws workshop export-all neurips-dl-workshop \
  --output-dir ./participant-work/ \
  --format zip

# Creates:
# ./participant-work/
#   ├── workshop-01.zip (Participant_01's notebooks)
#   ├── workshop-02.zip (Participant_02's notebooks)
#   ...
```

**Current workaround**: Participants manually SCP files (most don't)
**Impact**: Lost learning artifacts, can't reproduce workshop results

---

## 🎯 Ideal Future State: Complete Workshop Walkthrough

### Week Before Workshop: Setup with Auto-Terminate

```bash
# Create workshop project with aggressive cost controls
cws project create neurips-dl-workshop \
  --budget 200 \
  --hard-cap \
  --alert-threshold 50,75,90 \
  --description "NeurIPS 2025 Workshop: Deep Learning with PyTorch"

# Create policy-restricted invitations
cws profiles invitations batch-create-workshop \
  --csv-file participants.csv \
  --template-whitelist "PyTorch Machine Learning" \
  --max-instance-type "t3.medium" \
  --max-hourly-cost 0.10 \
  --valid-days 7 \
  --auto-terminate-hours 6 \
  --output-file invitation_codes.csv

# CloudWorkstation output:
# 📧 Generated 60 workshop invitations
#    - Valid for 7 days (expires Dec 9, 2025)
#    - Template restricted: "PyTorch Machine Learning" only
#    - Max instance: t3.medium ($0.0416/hr)
#    - Auto-terminate: 6 hours after launch
#    - Device limit: 2 devices per participant
#
# 📊 Projected costs:
#    - Per participant: $3.20 (6 hours × $0.0416/hr × 1.3 buffer)
#    - Total if all 60 launch: $192.00 ✅ (within $200 budget)
#
# ✅ Invitations saved to: invitation_codes.csv
#
# Next steps:
#   1. Email invitation codes to participants
#   2. Enable early access (optional): cws workshop early-access enable
#   3. Monitor signups: cws workshop participants neurips-dl-workshop

# Email invitation codes to participants
cws workshop email-invitations \
  --csv-file invitation_codes.csv \
  --template workshop_welcome.html \
  --subject "NeurIPS 2025: Deep Learning Workshop Access"
```

### Day Before Workshop: Early Access Testing

```bash
# Enable early access window (24 hours before workshop)
cws workshop early-access neurips-dl-workshop \
  --enable \
  --duration 24h \
  --test-mode

# Participants who test early (optional for them):
participant$ cws profiles invitations accept <CODE> neurips-workshop
participant$ cws launch "PyTorch Machine Learning" test-env --hours 2
# (Automatically terminates after 2 hours)

# Alex monitors early access
cws workshop participants neurips-dl-workshop

# Output:
# 📊 Early Access Status (24 hours before workshop)
#
# Accepted Invitations: 58 / 60 (97%)
# Tested Environment:   15 / 58 (26%)
#
# ✅ Ready: 15 participants (tested successfully)
# 🟡 Accepted but not tested: 43 participants
# ❌ Not yet accepted: 2 participants
#    - Participant_23: Invitation sent, not accepted
#    - Participant_47: Invitation sent, not accepted
#
# 💰 Early access cost: $3.20 (15 participants × $0.21/test)
# 📧 Reminder emails:
#    - Send reminder to 43 accepted-not-tested? [Y/n]: y
#    - Send urgent reminder to 2 not-accepted? [Y/n]: y
```

### Workshop Day: Smooth Execution

**8:45 AM - Pre-provisioning (optional)**:
```bash
# Option A: Let participants launch on-demand (default)
# - Slower but gives participants control
# - Launch time: ~2 minutes per instance

# Option B: Pre-provision all instances (advanced)
cws workshop bulk-provision neurips-dl-workshop \
  --template "PyTorch Machine Learning" \
  --size S \
  --auto-terminate-hours 6

# Output:
# 🚀 Provisioning 58 instances for accepted participants...
# ⏰ Auto-terminate: 6 hours from now (2:45 PM)
#
# Progress: [████████████████████] 58/58 complete (3m 12s)
#
# ✅ All instances ready!
# 💰 Current cost: $0.22 (15 minutes of provisioning)
# 📧 Email sent to all participants with connection info
```

**9:00 AM - Workshop begins**:
```bash
# Alex opens live dashboard in separate terminal
cws workshop dashboard neurips-dl-workshop --live

# Participants launch (if not pre-provisioned):
participant$ cws launch "PyTorch Machine Learning" workshop-instance
# ✅ Instance ready in 90 seconds!
# 📓 Jupyter Lab: http://54.123.45.67:8888 (token: abc123)
# ⏰ Instance will auto-terminate at 3:00 PM (6 hours)
# 💡 To save your work: cws download workshop-instance ~/my-work.zip
```

**10:30 AM - Participant needs help**:
```bash
# Dashboard shows participant_27 with high error rate
# Alex remotely debugs (with participant permission):
alex$ cws workshop debug neurips-dl-workshop workshop-27

# Options:
# 1. View Jupyter logs
# 2. View terminal output
# 3. SSH access (requires participant approval)
# 4. Reset notebook kernel
# 5. Restart instance

# Alex selects option 1, identifies issue, helps participant
```

**2:30 PM - 30 minutes before auto-terminate**:
```bash
# All participants automatically receive email + terminal notification:
#
# ⏰ Your workshop instance will terminate in 30 minutes!
#
# Save your work now:
#   cws download workshop-instance ~/neurips-workshop.zip
#
# Or create a snapshot to continue later:
#   cws snapshot create workshop-instance my-dl-work
#   (Costs $2.50/month, can recreate anytime)

# Participants who want to continue (personal budget):
participant$ cws snapshot create workshop-instance my-workshop
# ✅ Snapshot created: my-workshop
# 💰 Storage cost: $2.50/month (personal account)
#
# To recreate:
#   cws launch-from-snapshot my-workshop continued-work
```

**3:00 PM - Workshop ends, auto-terminate begins**:
```bash
# CloudWorkstation automatically:
# 1. Sends final warning (5 minutes before)
# 2. Terminates all instances at 3:00 PM sharp
# 3. Generates cost report
# 4. Archives workshop data (optional)

# Alex receives final report:
cws workshop report neurips-dl-workshop --export-pdf

# Output:
# 📊 NeurIPS 2025 Deep Learning Workshop - Final Report
#
# Participants:     58 / 60 registered (97%)
# Active instances: 58 instances for 3 hours
# Total uptime:     174 instance-hours
#
# Budget:
#   Allocated: $200.00
#   Spent:     $187.45 ✅ (within budget)
#   Saved:     $12.55 (available for next workshop - rollover enabled)
#
#   💡 Effective Cost Analysis:
#      24/7 assumption: $2.40/hr × 58 instances × 24 hours = $3,345.60
#      Actual workshop: $2.40/hr × 58 instances × 3 hours = $418.00
#      Your actual cost: $187.45 (early terminations banked immediately!)
#      Real-time banking: Every participant who finished early freed budget
#
#   Breakdown:
#     - Instance compute: $172.90 (58 × 3hrs × $0.99/hr)
#     - Early access:     $3.20 (15 tests)
#     - Pre-provisioning: $0.22 (15min buffer)
#     - Storage:          $11.13 (EBS, snapshots)
#
#   💡 Cloud vs Traditional:
#      Conference room PCs: $60,000 upfront + maintenance
#      CloudWorkstation: $187.45 for 3 hours of actual use
#      You only paid for compute time, not ownership!
#
# Participant Engagement:
#   - High engagement: 42 participants (72%)
#   - Medium engagement: 12 participants (21%)
#   - Low engagement: 4 participants (7%)
#
# Data Preservation:
#   - Snapshots created: 12 participants
#   - Downloads completed: 31 participants
#   - No action: 15 participants (work lost)
#
# ✅ All instances terminated successfully
# 💰 No ongoing costs
# 📧 Post-workshop survey sent to all participants
```

> **💡 GUI Note**: Workshop reports with charts and PDF export available in GUI Reports tab - *coming soon in v0.6.0*

---

## 📋 Feature Gap Analysis

### Critical Missing Features (Blockers)

| Feature | Priority | User Impact | Current Workaround | Effort |
|---------|----------|-------------|-------------------|--------|
| **Auto-Terminate Timer** | 🔴 Critical | Prevents budget overruns | Manual cleanup | Medium |
| **Template Whitelisting in Invitations** | 🔴 Critical | Prevents expensive launches | Trust + alerts | Low |
| **Policy-Restricted Invitations** | 🔴 Critical | Enforces workshop constraints | Manual monitoring | Medium |
| **Bulk Instance Provisioning** | 🟡 High | Saves 30min setup time | On-demand launch | Medium |
| **Workshop Dashboard** | 🟡 High | Real-time participant monitoring | Manual polling | High |

### Nice-to-Have Features (Enhancers)

| Feature | Priority | User Impact | Benefit |
|---------|----------|-------------|---------|
| **Participant Progress Tracking** | 🟢 Medium | Identify struggling participants | Proactive help |
| **Bulk Download/Export** | 🟢 Medium | Preserve participant work | Learning continuity |
| **Pre-Workshop Testing** | 🟢 Medium | Catch issues early | Smoother workshop |
| **Snapshot Quick-Save** | 🟢 Low | Easy work preservation | Student satisfaction |
| **Workshop Templates** | 🟢 Low | Reusable configurations | Faster setup |

---

## 🎯 Priority Recommendations

### Phase 1: Workshop Safety Net (v0.7.0)
**Target**: Workshops can run without budget disasters

1. **Auto-Terminate Timer** (1 week)
   - `cws launch template name --hours 6`
   - Countdown warnings at 30min, 5min
   - Graceful termination with EBS preservation

2. **Invitation Policy Restrictions** (1 week)
   - Template whitelist in invitation tokens
   - Instance type restrictions
   - Hourly cost limits
   - Policy validation before launch

3. **Workshop Project Type** (3 days)
   - `cws project create workshop --type workshop`
   - Built-in auto-terminate defaults
   - Aggressive budget alerts
   - One-time budget (no rollover)

### Phase 2: Workshop Management Tools (v0.7.1)
**Target**: Instructors can manage workshops effectively

4. **Workshop Dashboard** (1 week)
   - Live participant status
   - Real-time budget tracking
   - Problem detection (stopped instances, errors)
   - Terminal-based (TUI) interface

5. **Bulk Provisioning** (1 week)
   - Pre-launch instances for all participants
   - Scheduled start time
   - Coordinated auto-terminate
   - Assignment to invitation tokens

### Phase 3: Workshop Polish (v0.8.0+)
**Target**: Professional workshop experience

6. **Work Preservation** (3 days)
   - One-click download before terminate
   - Quick snapshot creation
   - Bulk export for instructors

7. **Workshop Templates** (3 days)
   - Reusable workshop configurations
   - Import participant list
   - One-command workshop setup

---

## Success Metrics

### User Satisfaction (Alex's Perspective)
- ✅ **Reliability**: "Zero budget disasters - workshop stayed under $200"
- ✅ **Ease of Setup**: "60 participants onboarded in 15 minutes"
- ✅ **Peace of Mind**: "Auto-terminate means I can focus on teaching, not cleanup"
- ✅ **Participant Success**: "97% completion rate - everyone could follow along"

### Technical Metrics
- Auto-terminate prevents 100% of budget overruns
- Workshop setup time: < 30 minutes (vs 2+ hours manual)
- Participant environment ready: < 2 minutes (vs 30+ minutes with local install)
- Zero instances left running post-workshop

### Business Impact
- **Conference Adoption**: "CloudWorkstation workshops" become a standard
- **Reduced Support**: Instructors handle workshops independently
- **Positive Reviews**: "Best hands-on workshop I've attended!" - Participants
- **Academic Reputation**: CloudWorkstation seen as workshop-ready platform

---

## Key Differences from University Class Scenario

| Aspect | Workshop (3 hours) | Class (15 weeks) |
|--------|-------------------|------------------|
| **Duration** | Single 3-hour session | 15-week semester |
| **Preparation** | 1 week (must be perfect) | 2-4 weeks (iterate) |
| **Budget** | One-time $200 | Semester $1,200 with rollover |
| **Access** | 6-hour window + cleanup | Ongoing with extensions |
| **Cleanup** | Immediate auto-terminate | Gradual semester-end |
| **Support** | On-site only (3 hours) | Office hours + TAs |
| **Participants** | 40-60 attendees | 50 students |
| **TA Structure** | None or single assistant | Head TA + multiple TAs |
| **Failure Cost** | Workshop disaster | Grade assignment issues |

---

## Reusable Infrastructure from Class Scenario

✅ **Already applicable**:
- Batch invitation system
- Device binding security
- Budget allocation and tracking
- Template restrictions via policy

🔧 **Needs adaptation**:
- Time limits: 6 hours vs 15 weeks
- Budget model: One-time vs recurring
- Auto-cleanup: Immediate vs gradual
- Support structure: Self-service vs TA hierarchy

---

## Next Steps

1. **Validate with Real Workshop Instructors**: Interview 2-3 conference workshop presenters
2. **Prototype Auto-Terminate**: Implement basic time-limited launches
3. **Design Workshop Dashboard**: Mock up live monitoring interface
4. **Implementation Plan**: Break down into 2-week sprints

**Estimated Timeline**: Workshop Safety Net (Phase 1) → 3 weeks of development

---

## Comparison: Workshop vs Class

**Similarities**:
- Batch user onboarding
- Template standardization
- Budget constraints
- Time-boxed access

**Critical Differences**:
```
Workshop = "High-stakes, single-shot performance"
Class = "Ongoing management with iteration opportunities"

Workshop auto-terminate = "6 hours hard deadline"
Class semester end = "Graceful 2-week wind-down"

Workshop budget = "$200 total, must not exceed"
Class budget = "$1,200 with weekly monitoring and adjustments"
```

**When to use each**:
- **Workshop project**: Single-day events, tutorials, short courses
- **Class project**: Semester-long courses, research bootcamps, training programs
