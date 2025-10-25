# CloudWorkStation Development Roadmap

**Current Version**: v0.5.5  
**Last Updated**: October 20, 2025  
**Status**: Active Development

This roadmap outlines planned features and enhancements for CloudWorkStation. All items are tracked in [GitHub Issues](https://github.com/scttfrdmn/cloudworkstation/issues) and the [CloudWorkStation Development Project](https://github.com/scttfrdmn/cloudworkstation/projects).

---

## 🎯 Current Focus: Phase 5.0 - UX Redesign

**Priority**: CRITICAL - HIGHEST PRIORITY
**Target**: v0.5.6 (Q4 2025)

CloudWorkStation is shifting focus from feature development to user experience optimization. The current 15-minute learning curve for first workspace needs to be reduced to 30 seconds.

### Phase 5.0.1: Quick Wins (2 weeks - Due: November 15, 2025)
**Milestone**: [Phase 5.0.1: Quick Wins](https://github.com/scttfrdmn/cloudworkstation/milestone/2)

High-impact, low-effort improvements:
- [#13](https://github.com/scttfrdmn/cloudworkstation/issues/13) - Home Page with Quick Start wizard
- [#14](https://github.com/scttfrdmn/cloudworkstation/issues/14) - Merge Terminal/WebView into Workspaces
- ✅ [#15](https://github.com/scttfrdmn/cloudworkstation/issues/15) - Rename "Instances" → "Workspaces" **(COMPLETE)**
- [#16](https://github.com/scttfrdmn/cloudworkstation/issues/16) - Collapse Advanced Features under Settings
- ✅ [#17](https://github.com/scttfrdmn/cloudworkstation/issues/17) - Add `cws init` onboarding wizard **(COMPLETE)**

**Deferred**:
- ~~[#65](https://github.com/scttfrdmn/cloudworkstation/issues/65) - Project rename~~ *(deferred - final name TBD)*

### Phase 5.0.2: Information Architecture (4 weeks - Due: December 15, 2025)
**Milestone**: [Phase 5.0.2: Information Architecture](https://github.com/scttfrdmn/cloudworkstation/milestone/3)

Navigation and structural improvements:
- [#18](https://github.com/scttfrdmn/cloudworkstation/issues/18) - Unified Storage UI (EFS + EBS)
- [#19](https://github.com/scttfrdmn/cloudworkstation/issues/19) - Integrate Budgets into Projects
- Navigation reorganization (14 → 6 items)
- Role-based visibility (hide admin features from non-admins)
- Context-aware recommendations

### Phase 5.0.3: CLI Consistency (2 weeks - Due: December 31, 2025)
**Milestone**: [Phase 5.0.3: CLI Consistency](https://github.com/scttfrdmn/cloudworkstation/milestone/4)

Command structure improvements:
- [#20](https://github.com/scttfrdmn/cloudworkstation/issues/20) - Consistent CLI Command Structure
- Unified storage commands (`cws storage` replacing `volume` + `storage`)
- Predictable command patterns
- Enhanced tab completion

**Why This Matters**: New researchers face cognitive overload before completing basic tasks. UX redesign will dramatically improve first-time user experience.

**Track Progress**: [GitHub Milestone: Phase 5.0 UX Redesign](https://github.com/scttfrdmn/cloudworkstation/milestone/1)

---

## 📅 Release Schedule

### v0.5.6 (Q1 2026): UX Redesign + Storage & Template Enhancements
**Release Date**: January 15, 2026
**Focus**: User experience transformation + Advanced provisioning capabilities

#### UX Redesign Components
**Milestones**: [5.0.1](https://github.com/scttfrdmn/cloudworkstation/milestone/2), [5.0.2](https://github.com/scttfrdmn/cloudworkstation/milestone/3), [5.0.3](https://github.com/scttfrdmn/cloudworkstation/milestone/4)
- Complete Phase 5.0.1, 5.0.2, 5.0.3 (detailed above)
- Home page, navigation restructure, CLI consistency
- **Impact**: Reduce onboarding from 15min to 30sec

#### Storage & Template Enhancements
**Milestone**: [Phase 5.6: Template Provisioning](https://github.com/scttfrdmn/cloudworkstation/milestone/13)
- [#30](https://github.com/scttfrdmn/cloudworkstation/issues/30) - SSM File Operations for Large File Transfer
- [#64](https://github.com/scttfrdmn/cloudworkstation/issues/64) - S3-backed file transfer with progress reporting
- [#31](https://github.com/scttfrdmn/cloudworkstation/issues/31) - Template asset management for binaries and configuration files
- **Impact**: Enable multi-GB template provisioning with progress tracking

### v0.6.0 (Q2 2026): Security, Authentication & User Experience
**Major Release - Enterprise Ready**

#### 1. 🔄 Auto-Update Feature ([#61](https://github.com/scttfrdmn/cloudworkstation/issues/61))
**Status**: Planned  
**Why**: Users don't know when new versions are available, miss bug fixes and features

**Features**:
- GitHub Releases API integration for version detection
- `cws version --check-update` command with release notes
- Startup notifications in CLI/TUI/GUI
- Platform-specific update helpers (Homebrew, apt, manual install)

**Example**:
```bash
$ cws version --check-update
CloudWorkStation CLI v0.5.5
⚠️  New version available: v0.6.0 (released 2 days ago)

What's New:
- AWS Quota Management with intelligent AZ failover
- Auto-update feature with background checks
- GUI system tray support

To update:
  macOS:   brew upgrade cloudworkstation
  Linux:   curl -L https://get.cloudworkstation.io | bash
```

#### 2. 🖥️ GUI System Tray and Auto-Start ([#62](https://github.com/scttfrdmn/cloudworkstation/issues/62))
**Status**: Planned  
**Why**: GUI lacks convenient system tray access and auto-start on login

**Features**:
- Native system tray integration (macOS menu bar, Windows tray, Linux tray)
- Context menu with Quick Launch, My Workspaces, Cost Summary
- Auto-start on login (Launch Agents, Registry, XDG autostart)
- Intelligent notifications (launches, budget alerts, idle warnings)

**Menu Structure**:
```
CloudWorkStation [Icon]
├── 🚀 Quick Launch → Python ML, R Research, Ubuntu Desktop
├── 💻 My Workspaces (3 running)
├── 💰 Cost Summary ($42.50 this month)
├── ⚙️  Preferences
└── ⏹️  Quit
```

#### 3. 📊 AWS Quota Management ([#57-60](https://github.com/scttfrdmn/cloudworkstation/issues/57))
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
$ cws admin quota show --region us-west-2

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

#### 5. 🎯 EC2 Capacity Blocks for ML ([#63](https://github.com/scttfrdmn/cloudworkstation/issues/63))
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
$ cws capacity-blocks search \
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

To reserve: cws capacity-blocks purchase cbr-0123456789abcdefg
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

### v0.5.6 Components (Q4 2025 - Q1 2026)

| Feature | Status | Milestone | Issues |
|---------|--------|-----------|--------|
| **Phase 5.0.1: Quick Wins** | 🟡 In Progress (2/5 complete) | [#2](https://github.com/scttfrdmn/cloudworkstation/milestone/2) | [#13](https://github.com/scttfrdmn/cloudworkstation/issues/13), [#14](https://github.com/scttfrdmn/cloudworkstation/issues/14), ~~[#15](https://github.com/scttfrdmn/cloudworkstation/issues/15)~~, [#16](https://github.com/scttfrdmn/cloudworkstation/issues/16), ~~[#17](https://github.com/scttfrdmn/cloudworkstation/issues/17)~~ |
| Home Page + Quick Start | 🟡 In Progress | [#2](https://github.com/scttfrdmn/cloudworkstation/milestone/2) | [#13](https://github.com/scttfrdmn/cloudworkstation/issues/13) |
| Merge Terminal/WebView | 🟡 In Progress | [#2](https://github.com/scttfrdmn/cloudworkstation/milestone/2) | [#14](https://github.com/scttfrdmn/cloudworkstation/issues/14) |
| Rename to "Workspaces" | ✅ Complete | [#2](https://github.com/scttfrdmn/cloudworkstation/milestone/2) | [#15](https://github.com/scttfrdmn/cloudworkstation/issues/15) |
| Collapse Advanced Features | 🟡 In Progress | [#2](https://github.com/scttfrdmn/cloudworkstation/milestone/2) | [#16](https://github.com/scttfrdmn/cloudworkstation/issues/16) |
| `cws init` Wizard | ✅ Complete | [#2](https://github.com/scttfrdmn/cloudworkstation/milestone/2) | [#17](https://github.com/scttfrdmn/cloudworkstation/issues/17) |
| **Phase 5.0.2: Info Architecture** | 📋 Planned | [#3](https://github.com/scttfrdmn/cloudworkstation/milestone/3) | [#18](https://github.com/scttfrdmn/cloudworkstation/issues/18), [#19](https://github.com/scttfrdmn/cloudworkstation/issues/19) |
| Unified Storage UI | 📋 Planned | [#3](https://github.com/scttfrdmn/cloudworkstation/milestone/3) | [#18](https://github.com/scttfrdmn/cloudworkstation/issues/18) |
| Integrate Budgets into Projects | 📋 Planned | [#3](https://github.com/scttfrdmn/cloudworkstation/milestone/3) | [#19](https://github.com/scttfrdmn/cloudworkstation/issues/19) |
| **Phase 5.0.3: CLI Consistency** | 📋 Planned | [#4](https://github.com/scttfrdmn/cloudworkstation/milestone/4) | [#20](https://github.com/scttfrdmn/cloudworkstation/issues/20) |
| Consistent CLI Commands | 📋 Planned | [#4](https://github.com/scttfrdmn/cloudworkstation/milestone/4) | [#20](https://github.com/scttfrdmn/cloudworkstation/issues/20) |
| **Template Provisioning** | 📋 Planned | [#13](https://github.com/scttfrdmn/cloudworkstation/milestone/13) | [#30](https://github.com/scttfrdmn/cloudworkstation/issues/30), [#31](https://github.com/scttfrdmn/cloudworkstation/issues/31), [#64](https://github.com/scttfrdmn/cloudworkstation/issues/64) |
| SSM File Operations | 📋 Planned | [#13](https://github.com/scttfrdmn/cloudworkstation/milestone/13) | [#30](https://github.com/scttfrdmn/cloudworkstation/issues/30) |
| S3 File Transfer + Progress | 📋 Planned | [#13](https://github.com/scttfrdmn/cloudworkstation/milestone/13) | [#64](https://github.com/scttfrdmn/cloudworkstation/issues/64) |
| Template Asset Management | 📋 Planned | [#13](https://github.com/scttfrdmn/cloudworkstation/milestone/13) | [#31](https://github.com/scttfrdmn/cloudworkstation/issues/31) |

### Future Releases (v0.6.0+)

| Feature | Status | Target Release | GitHub Issue |
|---------|--------|----------------|--------------|
| Directory Sync | 📋 Planned | v0.5.5-0.5.6 | [#53](https://github.com/scttfrdmn/cloudworkstation/issues/53) |
| Configuration Sync | 📋 Planned | v0.5.3-0.5.4 | [#54](https://github.com/scttfrdmn/cloudworkstation/issues/54) |
| Auto-Update | 📋 Planned | v0.6.0-0.7.0 | [#61](https://github.com/scttfrdmn/cloudworkstation/issues/61) |
| GUI System Tray | 📋 Planned | v0.6.0-0.6.1 | [#62](https://github.com/scttfrdmn/cloudworkstation/issues/62) |
| AWS Quota Management | 📋 Planned | v0.6.0 | [#57-60](https://github.com/scttfrdmn/cloudworkstation/issues/57) |
| Multi-User Auth | 📋 Planned | v0.6.0-0.7.0 | Coming Soon |
| Capacity Blocks | 📋 Planned | v0.7.0-0.7.2 | [#63](https://github.com/scttfrdmn/cloudworkstation/issues/63) |

**Legend**: ✅ Complete | 🟡 In Progress | 📋 Planned

---

## 🎯 Success Metrics

CloudWorkStation tracks these metrics to measure progress:

### Current State (v0.5.5)
- ⏱️ **Time to first workspace**: ~15 minutes (needs improvement)
- 🧭 **Navigation complexity**: 14 flat items (needs simplification)
- 😃 **User confusion rate**: ~40% of support tickets (needs reduction)
- 🎯 **CLI first-attempt success**: ~60% (needs improvement)

### Target State (v0.6.0)
- ⏱️ **Time to first workspace**: 30 seconds
- 🧭 **Navigation complexity**: 6 primary categories
- 😃 **User confusion rate**: <15% of support tickets
- 🎯 **CLI first-attempt success**: >85%
- 📈 **Version adoption**: >70% on latest within 7 days

---

## 💡 How to Contribute

**Found a feature request?**  
Create an issue: [github.com/scttfrdmn/cloudworkstation/issues/new](https://github.com/scttfrdmn/cloudworkstation/issues/new)

**Want to discuss the roadmap?**  
Join discussions: [github.com/scttfrdmn/cloudworkstation/discussions](https://github.com/scttfrdmn/cloudworkstation/discussions)

**Technical implementation details?**  
See: [Technical Debt Backlog](archive/roadmap/TECHNICAL_DEBT_BACKLOG.md)

---

## 📚 Related Documentation

- [VISION.md](VISION.md) - Long-term product vision
- [UX Evaluation](architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md) - Current UX issues
- [Technical Debt Backlog](archive/roadmap/TECHNICAL_DEBT_BACKLOG.md) - Implementation tracking
- [GitHub Projects](https://github.com/scttfrdmn/cloudworkstation/projects) - Sprint planning

---

**Questions?** Open a [GitHub Discussion](https://github.com/scttfrdmn/cloudworkstation/discussions) or check [existing issues](https://github.com/scttfrdmn/cloudworkstation/issues).
