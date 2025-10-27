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
- [#19](https://github.com/scttfrdmn/prism/issues/19) - Integrate Budgets into Projects

#### Success Metrics
- 🧭 Navigation complexity: 14 → 6 top-level items
- ⏱️ Time to find features: <10 seconds
- 😃 User confusion: Further 30% reduction
- 📱 Advanced feature discoverability: >95%

**Implementation Schedule**: 2 weeks (Dec 16-27, 2025)
- Week 1: Terminal/WebView merge + Unified Storage
- Week 2: Settings restructure + Budget integration + testing

**Status**: 📋 Planned

---

### v0.6.0 (Q2 2026): Enterprise Authentication + Advanced Features
**Release Date**: Target February 2026
**Focus**: Enterprise-ready authentication and advanced enterprise features

#### CLI Consistency Improvements
**Milestone**: [Phase 5.0.3: CLI Consistency](https://github.com/scttfrdmn/prism/milestone/4)

- [#20](https://github.com/scttfrdmn/prism/issues/20) - Consistent CLI Command Structure
- Unified storage commands (`prism storage` replacing `volume` + `storage`)
- Predictable command patterns
- Enhanced tab completion

#### Enterprise Authentication
- OAuth/OIDC integration (Google, Microsoft, institutional SSO)
- LDAP/Active Directory support
- SAML support for enterprise SSO
- Token validation and session management
- Role-based access control (RBAC) foundation

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
| Integrate Budgets into Projects | 📋 Planned | [#3](https://github.com/scttfrdmn/prism/milestone/3) | [#19](https://github.com/scttfrdmn/prism/issues/19) |

### v0.6.0: CLI Consistency + Enterprise (February 2026)

| Feature | Status | Milestone | Issues |
|---------|--------|-----------|--------|
| Consistent CLI Commands | 📋 Planned | [#4](https://github.com/scttfrdmn/prism/milestone/4) | [#20](https://github.com/scttfrdmn/prism/issues/20) |
| OAuth/OIDC Integration | 📋 Planned | TBD | Coming Soon |
| LDAP/Active Directory | 📋 Planned | TBD | Coming Soon |

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
