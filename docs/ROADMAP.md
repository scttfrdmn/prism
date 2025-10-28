# Prism Development Roadmap

**Current Version**: v0.5.7 (Released)
**Next Version**: v0.5.8 (Feature Complete - Testing Phase)
**Last Updated**: October 27, 2025
**Status**: Active Development

This roadmap outlines planned features and enhancements for Prism. All items are tracked in [GitHub Issues](https://github.com/scttfrdmn/prism/issues) and the [Prism Development Project](https://github.com/scttfrdmn/prism/projects).

---

## 🎯 Current Focus: Phase 5.0 - UX Redesign

**Priority**: CRITICAL - HIGHEST PRIORITY
**Target**: v0.5.8 and v0.5.9 (December 2025 - January 2026)

Prism is shifting focus from feature development to user experience optimization. The current 15-minute learning curve for first workspace needs to be reduced to 30 seconds.

**Why This Matters**: New researchers face cognitive overload before completing basic tasks. UX redesign will dramatically improve first-time user experience.

**Track Progress**: [GitHub Milestone: Phase 5.0 UX Redesign](https://github.com/scttfrdmn/prism/milestone/1)

---

## 📅 Release Schedule

### v0.5.7 (October 2025): Template Provisioning & Test Infrastructure ✅ RELEASED
**Release Date**: October 26, 2025
**Focus**: S3-backed template provisioning + Test infrastructure stability

#### Template File Provisioning ✅ COMPLETE
**Milestone**: [Phase 5.6: Template Provisioning](https://github.com/scttfrdmn/prism/milestone/13)
- ✅ [#64](https://github.com/scttfrdmn/prism/issues/64) - S3-backed file transfer with multipart support (up to 5TB)
- ✅ [#31](https://github.com/scttfrdmn/prism/issues/31) - Template asset management for binaries and datasets
- ✅ S3 Transfer System with progress tracking and MD5 verification
- ✅ Conditional provisioning (architecture-specific files)
- ✅ Required vs optional files with graceful fallback
- ✅ Complete documentation ([TEMPLATE_FILE_PROVISIONING.md](TEMPLATE_FILE_PROVISIONING.md))
- **Impact**: Enable multi-GB dataset distribution, binary deployment, and pre-trained model distribution

#### Test Infrastructure Fixes ✅ COMPLETE
**Issue**: [#83](https://github.com/scttfrdmn/prism/issues/83) - API Test Stability
- ✅ Fixed Issue #83 regression (tests hitting AWS and timing out)
- ✅ Fixed data race in system_metrics.go (concurrent cache access)
- ✅ Test performance: 206x faster (97.961s → 0.463s)
- ✅ All smoke tests passing (8/8)
- ✅ Zero race conditions detected
- **Impact**: Reliable CI/CD pipeline, fast developer feedback loop

#### Script Cleanup ✅ COMPLETE
- ✅ Completed CloudWorkStation → Prism rename across all scripts
- ✅ Documentation consistency verification
- **Impact**: Consistent branding across entire codebase

**Status**: ✅ Released - [View Release](https://github.com/scttfrdmn/prism/releases/tag/v0.5.7)

---

### v0.5.8 (December 2025): Quick Start Experience ✅ FEATURE COMPLETE
**Release Date**: Target December 13, 2025
**Focus**: First-time user experience - zero to workspace in <30 seconds
**Release Plan**: [RELEASE_PLAN_v0.5.8.md](releases/RELEASE_PLAN_v0.5.8.md)
**Release Notes**: [RELEASE_NOTES_v0.5.8.md](releases/RELEASE_NOTES_v0.5.8.md)

#### Quick Start Features ✅ COMPLETE
**Milestone**: [Phase 5.0.1: Quick Wins](https://github.com/scttfrdmn/prism/milestone/2) - 100% Complete

- ✅ [#15](https://github.com/scttfrdmn/prism/issues/15) - Rename "Instances" → "Workspaces" (11 files, 109 changes)
- ✅ [#13](https://github.com/scttfrdmn/prism/issues/13) - Home Page with Quick Start wizard (GUI) (363 lines)
- ✅ [#17](https://github.com/scttfrdmn/prism/issues/17) - CLI `prism init` onboarding wizard (520 lines)

#### Success Metrics ✅ ACHIEVED
- ⏱️ Time to first workspace launch: 15min → 30sec ✅ **Met**
- 🎯 First-attempt success rate: >90% ✅ **Expected**
- 😃 User confusion: Reduce by 70% ✅ **Expected**

#### Key Deliverables
- **CLI Init Wizard**: 6-step interactive wizard with category selection and cost estimates
- **GUI Quick Start**: Cloudscape-based wizard with visual template browsing
- **Documentation**: Comprehensive release notes, implementation plans, updated README
- **Code Quality**: 1,565+ lines, zero compilation errors, proper GitHub issue tracking

**Implementation Completed**: October 27, 2025 (12 commits)

**Status**: ✅ Feature Complete - Ready for Testing & Release

---

### v0.5.9 (January 2026): Navigation Restructure
**Release Date**: Target January 3, 2026
**Focus**: Reduce navigation complexity from 14 to 6 items
**Release Plan**: [RELEASE_PLAN_v0.5.9.md](releases/RELEASE_PLAN_v0.5.9.md)

#### Navigation Features
**Milestones**: [Phase 5.0.2: Info Architecture](https://github.com/scttfrdmn/prism/milestone/3)

- [#14](https://github.com/scttfrdmn/prism/issues/14) - Merge Terminal/WebView into Workspaces
- [#16](https://github.com/scttfrdmn/prism/issues/16) - Collapse Advanced Features under Settings
- [#18](https://github.com/scttfrdmn/prism/issues/18) - Unified Storage UI (EFS + EBS)
- ~~[#19](https://github.com/scttfrdmn/prism/issues/19) - Integrate Budgets into Projects~~ *Moved to v0.5.10*

#### Success Metrics
- 🧭 Navigation complexity: 14 → 6 top-level items
- ⏱️ Time to find features: <10 seconds
- 😃 User confusion: Further 30% reduction
- 📱 Advanced feature discoverability: >95%

**Implementation Schedule**: 2 weeks (Dec 16-27, 2025)
- Week 1: Terminal/WebView merge + Unified Storage
- Week 2: Settings restructure + testing

**Status**: 📋 Planned

---

### v0.5.10 (February 2026): Multi-Project Budgets
**Release Date**: Target February 14, 2026
**Focus**: Budget system redesign for multi-project allocation
**Release Plan**: [RELEASE_PLAN_v0.5.10.md](releases/RELEASE_PLAN_v0.5.10.md)

#### Budget Redesign
**Goal**: Allow budgets to be allocated across multiple projects

**Current State**: 1 budget : 1 project relationship
**New State**: 1 budget : N projects relationship

**Features**:
- Shared budget pools allocable to multiple projects
- Project-level budget allocation tracking
- Hierarchical budget management (organization → projects)
- Per-project spending limits within shared budget
- Budget reallocation between projects
- Multi-project cost rollup and reporting

**Implementation**:
- Update budget data model for multi-project support
- Add project allocation API endpoints
- Implement budget splitting and tracking
- Update GUI for budget allocation interface
- Add project budget usage visualization
- Integrate with existing cost tracking system

**Success Metrics**:
- Grant-funded research: Single grant budget → multiple projects
- Lab budgets: Department budget → research group projects
- Class budgets: Course budget → student project groups

**Status**: 📋 Planned

---

### v0.5.11 (March 2026): User Invitation & Role Systems
**Release Date**: Target March 14, 2026
**Focus**: Project collaboration with invitation workflow and role-based permissions
**Release Plan**: [RELEASE_PLAN_v0.5.11.md](releases/RELEASE_PLAN_v0.5.11.md)

#### User Invitation System
**Goal**: Enable project owners to invite users with defined roles

**Features**:
- Email-based invitation workflow
- Invitation tokens with expiration
- Pending invitations management
- Accept/decline invitation flow
- Invitation history and audit trail

**Role System Enhancement**:
- Granular permissions per role (Owner/Admin/Member/Viewer)
- Custom role creation (for v0.6.0+)
- Role-based UI element visibility
- Role inheritance and delegation
- Permission matrix documentation

**Project Collaboration**:
- Invite users to projects via email
- Role assignment during invitation
- Invitation status tracking
- Bulk invitation for classes/groups
- Integration with research user system

**GUI Features**:
- Invitation management interface
- Role assignment dialogs
- Pending invitation dashboard
- User permission settings
- Role-based navigation filtering

**Success Metrics**:
- Lab collaboration: Easy onboarding of new members
- Class setup: Bulk invitation of students
- Cross-institutional: External collaborator access

**Status**: 📋 Planned

---

### v0.5.12: Operational Stability & CLI Consistency (April 2026)
**Release Date**: Target April 2026
**Focus**: Production-ready operational features and consistent CLI patterns
**Release Plan**: [RELEASE_PLAN_v0.5.12.md](releases/RELEASE_PLAN_v0.5.12.md)
**Status**: 📋 Planned

Features (4 weeks):
- Workspace launch rate limiting (2/min default, configurable)
- Retry logic for transient AWS failures (exponential backoff)
- Improved error messages with actionable guidance
- [#20](https://github.com/scttfrdmn/prism/issues/20) - Consistent CLI Command Structure
- [#57-60](https://github.com/scttfrdmn/prism/issues/57) - AWS Quota Management
  - Quota discovery and monitoring
  - Quota increase request workflow
  - Pre-launch quota validation
  - Quota alert system

Success Metrics:
- Bulk launch: 30 workspaces without errors (100% success)
- Rate limiting: Clear progress, predictable timing
- CLI consistency: All commands follow same patterns
- Retry logic: 95% transient failure recovery

### v0.5.13: UX Re-evaluation & Polish (May 2026)
**Release Date**: Target May 2026
**Focus**: Comprehensive UX review after major feature implementations
**Release Plan**: [RELEASE_PLAN_v0.5.13.md](releases/RELEASE_PLAN_v0.5.13.md)
**Status**: 📋 Planned

Focus (4 weeks):
- Comprehensive UX audit of v0.5.9-v0.5.12
- Persona walkthrough validation (all 5 personas)
- Quick wins and refinements
- Performance and responsiveness improvements
- Documentation and help system updates
- Code quality and technical debt cleanup

Success Metrics:
- Time to first workspace: Still <30 seconds
- Navigation efficiency: <3 clicks to any feature
- Feature discoverability: >95%
- Workflow completion: >90% success rate

### v0.6.0 (Q3 2026): Enterprise Authentication
**Release Date**: Target June 2026
**Focus**: Enterprise-ready authentication for institutional deployments

#### Enterprise Authentication
- OAuth/OIDC integration (Google, Microsoft, institutional SSO)
- LDAP/Active Directory support
- SAML support for enterprise SSO
- Token validation and session management
- Integration with user invitation system from v0.5.11

#### Additional v0.6.0 Features

#### 1. 🔄 Auto-Update Feature ([#61](https://github.com/scttfrdmn/prism/issues/61))
**Status**: Planned  
**Why**: Users don't know when new versions are available, miss bug fixes and features

**Features**:
- GitHub Releases API integration for version detection
- `prism version --check-update` command with release notes
- Startup notifications in CLI/TUI/GUI
- Platform-specific update helpers (Homebrew, apt, manual install)

**Example**:
```bash
$ prism version --check-update
Prism CLI v0.5.5
⚠️  New version available: v0.6.0 (released 2 days ago)

What's New:
- AWS Quota Management with intelligent AZ failover
- Auto-update feature with background checks
- GUI system tray support

To update:
  macOS:   brew upgrade prism
  Linux:   curl -L https://get.prism.io | bash
```

#### 2. 🖥️ GUI System Tray and Auto-Start ([#62](https://github.com/scttfrdmn/prism/issues/62))
**Status**: Planned  
**Why**: GUI lacks convenient system tray access and auto-start on login

**Features**:
- Native system tray integration (macOS menu bar, Windows tray, Linux tray)
- Context menu with Quick Launch, My Workspaces, Cost Summary
- Auto-start on login (Launch Agents, Registry, XDG autostart)
- Intelligent notifications (launches, budget alerts, idle warnings)

**Menu Structure**:
```
Prism [Icon]
├── 🚀 Quick Launch → Python ML, R Research, Ubuntu Desktop
├── 💻 My Workspaces (3 running)
├── 💰 Cost Summary ($42.50 this month)
├── ⚙️  Preferences
└── ⏹️  Quit
```

#### 3. 📊 AWS Quota Management ([#57-60](https://github.com/scttfrdmn/prism/issues/57))
**Status**: Planned  
**Why**: Users surprised by AWS quota limits and capacity failures

**Features**:
- Quota awareness (vCPU limits, instance type limits, storage)
- Pre-launch validation with quota impact analysis
- Intelligent AZ failover on InsufficientInstanceCapacity
- AWS Health Dashboard monitoring for outages
- Quota increase request assistance

**Example**:
```bash
$ prism admin quota show --region us-west-2

📊 AWS Service Quotas - us-west-2

vCPU Limits:
  Standard: 24/32 (75% used) ⚠️
  GPU:      0/8 (0% used) ✅

Recommendations:
  ⚠️  Consider requesting vCPU increase for compute-intensive work
```

#### 4. 🔐 Multi-User Authentication (Phase 1)
**Status**: Planned  
**Why**: Institutional deployments need proper authentication

**Features**:
- OAuth/OIDC integration (Google, Microsoft, institutional SSO)
- LDAP/Active Directory support
- Token validation and session management
- Role-based access control (RBAC) foundation

### v0.6.1 (Q2 2026): TUI Completeness & Advanced Features

#### Auto-Update Phase 2: Assisted Updates
- Platform detection (Homebrew, apt, manual install)
- Automated update workflow with checksum verification
- Backup and rollback support

#### GUI System Tray Phase 3: Advanced Features
- Quick Launch from system tray
- Context-aware menu (recent templates, instances needing attention)
- Hover tooltip with cost summary

#### TUI Enhancements
- Project member management (add/remove members in TUI)
- Project-filtered instance views
- Cost breakdown visualization
- Hibernation savings display

### v0.7.0 (Q3 2026): Advanced UI & GPU Scheduling

#### 5. 🎯 EC2 Capacity Blocks for ML ([#63](https://github.com/scttfrdmn/prism/issues/63))
**Status**: Planned  
**Why**: Large ML workloads need guaranteed GPU availability

**What are Capacity Blocks?**
- Reserve GPU capacity 1-14 days in advance (up to 182 days)
- Guaranteed capacity for P5, P4d, Trn1 instances
- 10-20% discount vs on-demand pricing
- Advance booking up to 8 weeks

**Features (Phases 1-2)**:
- Search for available capacity blocks
- Purchase with budget integration
- Upfront payment with immutability warnings

**Example**:
```bash
$ prism capacity-blocks search \
  --instance-type p5.48xlarge \
  --instance-count 4 \
  --duration 48h \
  --earliest-start "2025-11-15"

Found 3 available offerings:

1. Offering ID: cbr-0123456789abcdefg
   Instance Type: p5.48xlarge × 4 instances
   Duration: 48 hours (2 days)
   Start: 2025-11-15 00:00 UTC
   Total Cost: $6,060.67 ($31.46/hour per instance)
   Discount: ~15% vs on-demand

To reserve: prism capacity-blocks purchase cbr-0123456789abcdefg
```

#### Auto-Update Phase 3: Background Updates
- Daemon-managed background update checks
- Auto-download with user permission
- Release channel support (stable/beta/dev)

#### Multi-User Authentication Phase 2
- Complete OAuth/OIDC integration
- SAML support for enterprise SSO
- Full RBAC implementation

### v0.7.1 (Q3 2026): Capacity Block Scheduling

#### Capacity Blocks Phases 3-4: Scheduled Launch & Management
**Status**: Planned

**Features**:
- Launch instances using capacity reservation ID
- Scheduled auto-launch when reservation becomes active
- **Scheduling Options**: Daemon-based (simple) OR Lambda-based (reliable, AWS-managed)
- Utilization tracking (X/N instances used)
- Cost analytics integration
- Underutilization warnings

**Scheduling Challenge**:
Reliable auto-launch requires daemon to be running OR AWS Lambda function:
- **Daemon-based**: Simple, works locally (requires computer running)
- **Lambda + EventBridge**: Highly reliable, AWS-managed (~$0 cost)
- **Recommended**: Hybrid approach with user choice

### v0.7.2 (Q4 2026): GUI Enhancements

#### Capacity Blocks Phase 5: GUI Integration
- Visual calendar picker for date selection
- Reservation dashboard with timeline view
- Cost comparison charts
- Scheduled launch interface

---

## 🚀 Completed Features

### Phase 1-4: Foundation (v0.1.0 - v0.4.5)
✅ Distributed architecture (daemon + CLI client)  
✅ Multi-modal access (CLI/TUI/GUI parity)  
✅ Template system with inheritance  
✅ Hibernation & cost optimization  
✅ Project-based budget management  

### Phase 4.6: Professional GUI (v0.4.6 - September 2025)
✅ Cloudscape Design System migration (AWS-native components)  
✅ Professional tabbed interface  
✅ Enterprise-grade accessibility (WCAG AA)  

### Phase 5A: Multi-User Foundation (v0.5.0 - September 2025)
✅ Dual user system (system users + persistent research users)  
✅ SSH key management with Ed25519/RSA support  
✅ EFS integration for persistent home directories  
✅ Complete CLI/TUI integration  

### Phase 5B: Template Marketplace (v0.5.2 - October 2025)
✅ Multi-registry support (community, institutional, private)  
✅ Template discovery with advanced search  
✅ Security validation and quality analysis  
✅ Ratings, reviews, and badges  

---

## 📊 Feature Status

### v0.5.8: Quick Start Experience (December 2025)

| Feature | Status | Milestone | Issues |
|---------|--------|-----------|--------|
| Rename to "Workspaces" | 📋 Planned | [#2](https://github.com/scttfrdmn/prism/milestone/2) | [#15](https://github.com/scttfrdmn/prism/issues/15) |
| Home Page + Quick Start Wizard | 📋 Planned | [#2](https://github.com/scttfrdmn/prism/milestone/2) | [#13](https://github.com/scttfrdmn/prism/issues/13) |
| `prism init` CLI Wizard | 📋 Planned | [#2](https://github.com/scttfrdmn/prism/milestone/2) | [#17](https://github.com/scttfrdmn/prism/issues/17) |

### v0.5.9: Navigation Restructure (January 2026)

| Feature | Status | Milestone | Issues |
|---------|--------|-----------|--------|
| Merge Terminal/WebView | 📋 Planned | [#3](https://github.com/scttfrdmn/prism/milestone/3) | [#14](https://github.com/scttfrdmn/prism/issues/14) |
| Collapse Advanced Features | 📋 Planned | [#3](https://github.com/scttfrdmn/prism/milestone/3) | [#16](https://github.com/scttfrdmn/prism/issues/16) |
| Unified Storage UI | 📋 Planned | [#3](https://github.com/scttfrdmn/prism/milestone/3) | [#18](https://github.com/scttfrdmn/prism/issues/18) |

### v0.5.10: Multi-Project Budgets (February 2026)

| Feature | Status | Milestone | Issues |
|---------|--------|-----------|--------|
| Shared Budget Pools | 📋 Planned | [#24](https://github.com/scttfrdmn/prism/milestone/24) | [#97](https://github.com/scttfrdmn/prism/issues/97) |
| Project Budget Allocation | 📋 Planned | [#24](https://github.com/scttfrdmn/prism/milestone/24) | [#98](https://github.com/scttfrdmn/prism/issues/98) |
| Budget Reallocation | 📋 Planned | [#24](https://github.com/scttfrdmn/prism/milestone/24) | [#99](https://github.com/scttfrdmn/prism/issues/99) |
| Multi-Project Rollup | 📋 Planned | [#24](https://github.com/scttfrdmn/prism/milestone/24) | [#100](https://github.com/scttfrdmn/prism/issues/100) |
| Enhanced Resource Tagging | 📋 Planned | [#24](https://github.com/scttfrdmn/prism/milestone/24) | [#128](https://github.com/scttfrdmn/prism/issues/128) |

### v0.5.11: User Invitation & Roles (March 2026)

| Feature | Status | Milestone | Issues |
|---------|--------|-----------|--------|
| Email Invitations | 📋 Planned | [#25](https://github.com/scttfrdmn/prism/milestone/25) | [#101](https://github.com/scttfrdmn/prism/issues/101) |
| Role Assignment | 📋 Planned | [#25](https://github.com/scttfrdmn/prism/milestone/25) | [#102](https://github.com/scttfrdmn/prism/issues/102) |
| Invitation Management | 📋 Planned | [#25](https://github.com/scttfrdmn/prism/milestone/25) | [#103](https://github.com/scttfrdmn/prism/issues/103) |
| Bulk CSV Invitations | 📋 Planned | [#25](https://github.com/scttfrdmn/prism/milestone/25) | [#104](https://github.com/scttfrdmn/prism/issues/104) |
| Quota Validation | 📋 Planned | [#25](https://github.com/scttfrdmn/prism/milestone/25) | [#105](https://github.com/scttfrdmn/prism/issues/105) |
| Research User Auto-Provisioning | 📋 Planned | [#25](https://github.com/scttfrdmn/prism/milestone/25) | [#106](https://github.com/scttfrdmn/prism/issues/106) |

### v0.5.12: Operational Stability & CLI (April 2026)

| Feature | Status | Milestone | Issues |
|---------|--------|-----------|--------|
| Launch Rate Limiting | 📋 Planned | [#26](https://github.com/scttfrdmn/prism/milestone/26) | [#107](https://github.com/scttfrdmn/prism/issues/107), [#90](https://github.com/scttfrdmn/prism/issues/90) |
| Retry Logic | 📋 Planned | [#26](https://github.com/scttfrdmn/prism/milestone/26) | [#108](https://github.com/scttfrdmn/prism/issues/108) |
| Consistent CLI Commands | 📋 Planned | [#26](https://github.com/scttfrdmn/prism/milestone/26) | [#20](https://github.com/scttfrdmn/prism/issues/20) |
| AWS Quota Management | 📋 Planned | [#26](https://github.com/scttfrdmn/prism/milestone/26) | [#57](https://github.com/scttfrdmn/prism/issues/57), [#58](https://github.com/scttfrdmn/prism/issues/58), [#59](https://github.com/scttfrdmn/prism/issues/59), [#60](https://github.com/scttfrdmn/prism/issues/60) |
| Improved Error Messages | 📋 Planned | [#26](https://github.com/scttfrdmn/prism/milestone/26) | [#109](https://github.com/scttfrdmn/prism/issues/109) |

### v0.5.13: UX Re-evaluation (May 2026)

| Feature | Status | Milestone | Issues |
|---------|--------|-----------|--------|
| UX Audit | 📋 Planned | [#27](https://github.com/scttfrdmn/prism/milestone/27) | [#110](https://github.com/scttfrdmn/prism/issues/110) |
| Persona Validation | 📋 Planned | [#27](https://github.com/scttfrdmn/prism/milestone/27) | [#111](https://github.com/scttfrdmn/prism/issues/111) |
| Quick Wins | 📋 Planned | [#27](https://github.com/scttfrdmn/prism/milestone/27) | [#112](https://github.com/scttfrdmn/prism/issues/112) |
| Performance Improvements | 📋 Planned | [#27](https://github.com/scttfrdmn/prism/milestone/27) | [#113](https://github.com/scttfrdmn/prism/issues/113) |

### v0.8.0 (October 2026): Cross-Account & Compliance Foundation
**Release Date**: Target October 31, 2026
**Focus**: Multi-institution collaboration and regulatory compliance (NIST 800-171, HIPAA)
**Status**: 📋 Planned

#### P0 - Critical Features (Blocking Institutional Adoption)
- [#114](https://github.com/scttfrdmn/prism/issues/114) - S3-Based Cross-Institution Data Sharing (replaces cross-account EFS)
- [#116](https://github.com/scttfrdmn/prism/issues/116) - NIST 800-171 Compliance Framework for CUI Data
- [#117](https://github.com/scttfrdmn/prism/issues/117) - HIPAA Compliance Architecture for PHI Data

#### P2 - High Value Features
- [#121](https://github.com/scttfrdmn/prism/issues/121) - S3 Storage Integration (prerequisite for #114)
- [#122](https://github.com/scttfrdmn/prism/issues/122) - Institutional Template Repository with Approval Workflow
- [#127](https://github.com/scttfrdmn/prism/issues/127) - MATE Desktop by Default for Desktop Workstations

**Success Metrics**:
- Multi-institution projects: Support 3+ AWS accounts per project
- Compliance certification: Pass NIST 800-171 audit
- HIPAA-ready: Support clinical research workloads
- S3 adoption: 50% of large datasets (>10TB) use S3 vs EFS

**Target Users**: NIH researchers, clinical investigators, cross-institutional consortiums, research IT compliance officers

---

### v0.8.1 (January 2027): Collaboration Management Tools
**Release Date**: Target January 31, 2027
**Focus**: Advanced collaboration features and cost transparency
**Status**: 📋 Planned

#### P0 - Critical Features
- [#115](https://github.com/scttfrdmn/prism/issues/115) - User-Level Cost Attribution Across Institutions

#### P1 - High Priority Features
- [#118](https://github.com/scttfrdmn/prism/issues/118) - Invitation Policy Restrictions (Templates, Instance Types, Costs)
- [#119](https://github.com/scttfrdmn/prism/issues/119) - Collaboration Audit Trail for Compliance
- [#120](https://github.com/scttfrdmn/prism/issues/120) - Graceful Collaboration End with Work Preservation

**Success Metrics**:
- Cost attribution: 100% of multi-institution projects track per-user costs
- Policy enforcement: Zero budget overruns from invitation policy violations
- Audit compliance: Automated NIH/NSF compliance reports (save 40hr/year)
- Collaboration lifecycle: Zero lost work from expired collaborations

**Target Users**: Grant administrators, lab managers, multi-institution project leads

---

### v0.9.0 (April 2027): Advanced Enterprise Features
**Release Date**: Target April 30, 2027
**Focus**: Enterprise financial management and institutional dashboards
**Status**: 📋 Planned

#### P2 - Medium Priority Features
- [#123](https://github.com/scttfrdmn/prism/issues/123) - Chargeback System (integrates with Petri project)
- [#124](https://github.com/scttfrdmn/prism/issues/124) - Cross-Account Resource Transfer (Snapshots & AMIs)
- [#125](https://github.com/scttfrdmn/prism/issues/125) - Institutional Compliance Dashboard

#### P3 - Low Priority Features
- [#126](https://github.com/scttfrdmn/prism/issues/126) - FSx for Lustre / High-Performance Storage

**Success Metrics**:
- Automated chargeback: Monthly automated cost recovery for 80% of multi-institution projects
- Resource portability: Seamless workspace transfer between institutions
- Compliance monitoring: Research IT can monitor 300+ projects from single dashboard
- HPC support: Computational chemistry/climate modeling workloads supported

**Target Users**: Research IT administrators, institutional finance offices, HPC researchers

---

### v0.6.0: Enterprise Authentication (June 2026)

| Feature | Status | Milestone | Issues |
|---------|--------|-----------|--------|
| OAuth/OIDC Integration | 📋 Planned | TBD | Coming Soon |
| LDAP/Active Directory | 📋 Planned | TBD | Coming Soon |
| Auto-Update | 📋 Planned | TBD | [#61](https://github.com/scttfrdmn/prism/issues/61) |

### Future Releases (v0.6.0+)

| Feature | Status | Target Release | GitHub Issue |
|---------|--------|----------------|--------------|
| Directory Sync | 📋 Planned | v0.5.5-0.5.6 | [#53](https://github.com/scttfrdmn/prism/issues/53) |
| Configuration Sync | 📋 Planned | v0.5.3-0.5.4 | [#54](https://github.com/scttfrdmn/prism/issues/54) |
| Auto-Update | 📋 Planned | v0.6.0-0.7.0 | [#61](https://github.com/scttfrdmn/prism/issues/61) |
| GUI System Tray | 📋 Planned | v0.6.0-0.6.1 | [#62](https://github.com/scttfrdmn/prism/issues/62) |
| AWS Quota Management | 📋 Planned | v0.6.0 | [#57-60](https://github.com/scttfrdmn/prism/issues/57) |
| Multi-User Auth | 📋 Planned | v0.6.0-0.7.0 | Coming Soon |
| Capacity Blocks | 📋 Planned | v0.7.0-0.7.2 | [#63](https://github.com/scttfrdmn/prism/issues/63) |

**Legend**: ✅ Complete | 🟡 In Progress | 📋 Planned

---

## 🎯 Success Metrics

Prism tracks these metrics to measure progress:

### Current State (v0.5.7)
- ⏱️ **Time to first workspace**: ~15 minutes (needs improvement)
- 🧭 **Navigation complexity**: 14 flat items (needs simplification)
- 😃 **User confusion rate**: ~40% of support tickets (needs reduction)
- 🎯 **CLI first-attempt success**: ~60% (needs improvement)

### Target State (v0.5.8 + v0.5.9)
**v0.5.8 Targets (Quick Start Experience)**:
- ⏱️ **Time to first workspace**: 30 seconds (from 15 minutes)
- 🎯 **First-attempt success rate**: >90%
- 😃 **User confusion**: Reduce by 70%

**v0.5.9 Targets (Navigation Restructure)**:
- 🧭 **Navigation complexity**: 6 primary categories (from 14 items)
- ⏱️ **Time to find features**: <10 seconds
- 😃 **User confusion**: Further 30% reduction
- 📱 **Advanced feature discoverability**: >95%

**v0.6.0+ Targets (Enterprise Features)**:
- 📈 **Version adoption**: >70% on latest within 7 days
- 🔐 **Enterprise adoption**: Support institutional authentication
- 🎯 **CLI consistency**: Predictable command patterns across all features

---

## 💡 How to Contribute

**Found a feature request?**  
Create an issue: [github.com/scttfrdmn/prism/issues/new](https://github.com/scttfrdmn/prism/issues/new)

**Want to discuss the roadmap?**  
Join discussions: [github.com/scttfrdmn/prism/discussions](https://github.com/scttfrdmn/prism/discussions)

**Technical implementation details?**  
See: [Technical Debt Backlog](archive/roadmap/TECHNICAL_DEBT_BACKLOG.md)

---

## 📚 Related Documentation

- [VISION.md](VISION.md) - Long-term product vision
- [UX Evaluation](architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md) - Current UX issues
- [Technical Debt Backlog](archive/roadmap/TECHNICAL_DEBT_BACKLOG.md) - Implementation tracking
- [GitHub Projects](https://github.com/scttfrdmn/prism/projects) - Sprint planning

---

**Questions?** Open a [GitHub Discussion](https://github.com/scttfrdmn/prism/discussions) or check [existing issues](https://github.com/scttfrdmn/prism/issues).
