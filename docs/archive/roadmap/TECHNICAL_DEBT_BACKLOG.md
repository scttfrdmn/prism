# Technical Debt and Enhancement Backlog

**Last Updated**: October 25, 2025
**Status**: Active tracking of deferred implementations

This document tracks features that were intentionally deferred during development, marked as design decisions rather than immediate TODO items. These represent real work that should be scheduled for future releases.

---

## 🎯 Current Focus: v0.5.6 UX Redesign (Q4 2025 - Q1 2026)

### Phase 5.0.1: Quick Wins (2 weeks - Due: November 15, 2025)
**Milestone**: [#2](https://github.com/scttfrdmn/cloudworkstation/milestone/2)

#### 🏠 #13: Home Page with Quick Start Wizard
**Location**: `cmd/cws-gui/frontend/src/` (new HomePage component)
**Priority**: Critical
**Effort**: 3-4 days
**Implementation**:
- Create HomePage.tsx with context-aware landing page
- New user view: Quick Start buttons, educational content
- Returning user view: Recent workspaces, recommendations, quick stats
- Integration points: Dashboard navigation, first-run detection

#### 🔀 #14: Merge Terminal/WebView into Workspaces
**Location**: `cmd/cws-gui/frontend/src/`, TUI navigation, CLI help
**Priority**: Critical
**Effort**: 2-3 days
**Implementation**:
- Move Terminal/WebView from top-level nav to contextual actions in Workspaces
- GUI: Add "Open Terminal" / "Open Web UI" buttons to workspace action dropdown
- TUI: Remove pages 4-5, add actions to instance detail view
- Update navigation: 14 items → 12 items (step toward 6)

#### 📝 #15: Rename "Instances" → "Workspaces"
**Location**: All interfaces (CLI/TUI/GUI), docs, code comments
**Priority**: Critical
**Effort**: 4-5 days (requires comprehensive rename across codebase)
**Implementation**:
- Code: Update all user-facing strings, preserve internal "instance" in API/types
- GUI: Navigation item, page titles, dialogs, help text
- TUI: Page headers, help text, status messages
- CLI: Command help text, output formatting, error messages
- Docs: Update all user-facing documentation (ROADMAP, USER_GUIDE, etc.)

#### ⚙️ #16: Collapse Advanced Features Under Settings
**Location**: `cmd/cws-gui/frontend/src/App.tsx` navigation
**Priority**: Critical
**Effort**: 2-3 days
**Implementation**:
- Move AMI, Rightsizing, Policy, Idle Detection, Logs under Settings → Advanced
- Create Settings submenu structure
- Update navigation: 12 items → 7 items (Settings + 6 collapsed items)
- Role-based visibility: Hide admin features from non-admin users

#### ✅ #17: `cws init` Onboarding Wizard **(COMPLETE - October 25, 2025)**
**Status**: ✅ Complete - PR #70 merged
**Location**: `internal/cli/init_wizard.go`
**Implemented**:
- ✅ 5-step interactive wizard: AWS config, research area, budget, hibernation, templates
- ✅ AWS credential detection and validation
- ✅ Research area selection with personalized recommendations
- ✅ Optional budget configuration
- ✅ Hibernation policy setup (gpu/batch/balanced)
- ✅ Template recommendations based on domain

**Deferred Enhancements** (moved to technical debt):
- [ ] **Auto-run on first command**: Automatically prompt `cws init` on first use if not initialized
  - Priority: Medium
  - Effort: 1-2 days
  - Implementation: Check `IsInitialized()` in main command handler, prompt user to run init
  - Benefits: Zero-friction onboarding, no manual init required

- [ ] **Skip/customize options**: Add `--skip` flag and `CWS_SKIP_INIT=1` env var
  - Priority: Low
  - Effort: 0.5 days
  - Implementation: Add flag parsing, env var check in init wizard
  - Benefits: Advanced users can bypass wizard for CI/CD environments

- [ ] **TUI init equivalent**: Simplified onboarding in terminal interface
  - Priority: Low
  - Effort: 2-3 days
  - Implementation: BubbleTea-based init flow in TUI
  - Benefits: Consistent experience across CLI/TUI

**Phase 5.0.1 Total Effort**: 14-19 days

### Phase 5.0.2: Information Architecture (4 weeks - Due: December 15, 2025)
**Milestone**: [#3](https://github.com/scttfrdmn/cloudworkstation/milestone/3)

#### 💾 #18: Unified Storage UI (EFS + EBS)
**Location**: `cmd/cws-gui/frontend/src/pages/Storage.tsx`
**Priority**: High
**Effort**: 5-6 days
**Implementation**:
- Single unified storage page with tabbed interface (EFS Volumes | EBS Volumes)
- Consistent operations: Create, Attach, Detach, Delete for both types
- Clear visual distinction: Shared (EFS) vs Dedicated (EBS)
- Merge `volume` and `storage` CLI commands into `cws storage`

#### 💰 #19: Integrate Budgets into Projects
**Location**: `cmd/cws-gui/frontend/src/pages/Projects.tsx`, Budget page removal
**Priority**: High
**Effort**: 4-5 days
**Implementation**:
- Remove standalone Budget navigation item
- Add Budget tab to Project detail view
- Show budget in project list table as additional column
- Update CLI: Budget operations under `cws project budget`
- Navigation: 7 items → 6 items (final target)

**Phase 5.0.2 Total Effort**: 9-11 days

### Phase 5.0.3: CLI Consistency (2 weeks - Due: December 31, 2025)
**Milestone**: [#4](https://github.com/scttfrdmn/cloudworkstation/milestone/4)

#### ⌨️ #20: Consistent CLI Command Structure
**Location**: `internal/cli/app.go`, all command files
**Priority**: High
**Effort**: 8-10 days
**Implementation**:
- Unified storage commands: `cws storage create/attach/detach/list/delete`
- Consistent workspace operations: `cws workspace start/stop/hibernate/resize`
- Predictable patterns: `cws <noun> <verb> <target> [options]`
- Enhanced tab completion with hierarchical structure
- Backward compatibility aliases for deprecated commands

**Phase 5.0.3 Total Effort**: 8-10 days

### Template Provisioning Enhancements (Parallel to Phase 5.0)
**Milestone**: [#13](https://github.com/scttfrdmn/cloudworkstation/milestone/13)

#### 📦 #30: SSM File Operations for Large File Transfer
**Location**: `pkg/aws/ssm_transfer.go` (new)
**Priority**: Medium
**Effort**: 5-6 days

#### ☁️ #64: S3-Backed File Transfer with Progress Reporting
**Location**: `pkg/storage/s3_transfer.go` (new)
**Priority**: Medium
**Effort**: 6-7 days

#### 🗂️ #31: Template Asset Management System
**Location**: `pkg/templates/assets.go` (new)
**Priority**: Medium
**Effort**: 4-5 days

**Template Provisioning Total Effort**: 15-18 days

**v0.5.6 Total Estimated Effort**: 46-58 days (9-12 weeks with parallel development)

---

## 🔄 Deferred Items

### 🏷️ #65: Project Rename (DEFERRED - Final Name TBD)
**Location**: Entire codebase, documentation, GitHub repository, domain
**Status**: **ON HOLD** - Awaiting final project name decision (may not be "CloudWorkspaces")
**Effort**: 4-5 days (when resumed)
**Implementation**:
- **Go Module**: Update module path from `github.com/scttfrdmn/cloudworkstation` to new name
- **All Imports**: Update ~100+ import statements across all `.go` files
- **State Migration**: Add code to move `~/.cloudworkstation` → new config directory
- **User-Facing Text**: Update all CLI/TUI/GUI strings, help text, error messages
- **Documentation**: Update all markdown files, mkdocs.yml, links
- **Binary Names**: Keep `cws`, `cwsd`, `cws-gui` (abbreviation works for both!)
- **GitHub**: Rename repository (redirects handled automatically)
- **Domain Migration**: If cloudworkspaces.io is chosen, update mkdocs.yml, docs/CNAME
- **DNS Setup**: Point new domain to GitHub Pages, configure SSL
- **Old Domain**: Set up 301 redirect (6-12 months)
- **Testing**: Verify migration path, all interfaces, documentation links, domain
**Rationale**: Aligns with "Instances → Workspaces" terminology change, better describes product
**Note**: All planning work preserved for when final name is selected

---

## ✅ Completed Items (October 17, 2025)

### ✅ 0. SSH Readiness Progress Reporting - COMPLETED
**Location**: `pkg/aws/manager.go:572-589`
**Completed**: October 17, 2025
**Implementation**:
- ✅ Status message feedback with emoji indicators (⏳, →, ✓, ⚠️, ✅)
- ✅ `waitForInstanceReadyWithProgress()` integrated into launch flow with progress callback
- ✅ Real-time feedback for instance_ready and ssh_ready stages
- ✅ Graceful error handling with user-friendly messages
**Remaining Work for Full Implementation** (moved to Future Enhancements):
- Thread ProgressReporter through launch orchestration flow for GUI/TUI
- Stream progress updates from daemon to CLI via WebSocket or SSE
- Update TUI to show launch progress with real-time updates
- Update GUI to show launch progress with real-time updates

### ✅ 1. IAM Instance Profile Validation - COMPLETED (Enhanced)
**Location**: `pkg/aws/manager.go:1663-1794`
**Completed**: October 17, 2025
**Implementation**:
- ✅ IAM client added to Manager struct (line 44)
- ✅ IAM client initialized in NewManager (lines 104, 122)
- ✅ Real `GetInstanceProfile()` API call implemented
- ✅ **Auto-creation of CloudWorkstation-Instance-Profile** if it doesn't exist:
  - Creates IAM role with EC2 trust relationship
  - Attaches AmazonSSMManagedInstanceCore for SSM access
  - Creates inline policy for autonomous idle detection (EC2 self-management)
  - Tags resources as ManagedBy: CloudWorkstation
- ✅ Graceful fallback when user lacks IAM permissions (logs warning, continues without IAM)
- ✅ Zero-configuration SSM access for users with IAM permissions
**Bonus Deliverables**:
- 📄 Complete IAM permissions documentation (docs/AWS_IAM_PERMISSIONS.md)
- 📄 Ready-to-apply IAM policy JSON (docs/cloudworkstation-iam-policy.json)
- 🔧 Interactive IAM setup script (scripts/setup-iam-permissions.sh)
**Note**: IAM profile auto-creation provides zero-configuration SSM and autonomous features

---

## High Priority Items

### 2. AWS Quota Management and Intelligent Availability Handling
**Location**: `pkg/aws/manager.go` (new quota monitoring module needed)
**Current Behavior**: No quota awareness, generic launch failures, no AZ failover
**Impact**: Users surprised by quota limits, confusing errors, failed launches in unavailable AZs
**Problem Statement**:
Researchers often don't understand AWS service quotas (vCPU limits, instance type limits per region/AZ). When launches fail, error messages are cryptic. CloudWorkStation should be intelligent about AWS capacity and gracefully handle failures.

**Implementation Needed**:

1. **Quota Awareness** (`pkg/aws/quota_manager.go`):
   - Query AWS Service Quotas API for current limits
   - Track usage against quotas (vCPUs, instances per type, storage)
   - Pre-launch validation: "You have 12/20 vCPUs used, this t3.xlarge (4 vCPU) will use 16/20"
   - Proactive warnings: "Warning: You're at 90% of your vCPU quota in us-west-2"
   - CLI command: `cws admin quota show --region us-west-2`

2. **Quota Increase Assistance** (`pkg/aws/quota_requests.go`):
   - Detect quota-related launch failures
   - Explain what quota was hit and why
   - Provide pre-filled quota increase request URL
   - CLI command: `cws admin quota request --instance-type p3.2xlarge --reason "ML research workload"`
   - Guide users through AWS Support Center quota request process

3. **Intelligent AZ Failover** (`pkg/aws/availability_manager.go`):
   - Detect `InsufficientInstanceCapacity` errors
   - Automatically retry in different AZ within same region
   - User-friendly message: "Instance type unavailable in us-west-2a, trying us-west-2b..."
   - Track AZ health per instance type (recent success rates)
   - Prefer AZs with recent successful launches

4. **AWS Health Dashboard Integration** (`pkg/aws/health_monitor.go`):
   - Monitor AWS Health API for service events
   - Detect regional outages, degraded performance, scheduled maintenance
   - Proactive notifications: "⚠️ AWS reports degraded EC2 performance in us-east-1"
   - Block launches to affected regions with clear explanations
   - Auto-suggest alternative regions with healthy status
   - CLI command: `cws admin aws-health --all-regions`

5. **Capacity Planning** (`pkg/aws/capacity_planner.go`):
   - Analyze historical launch patterns
   - Recommend regions/AZs with best availability for instance types
   - Warn about high-demand instance types (GPU, large memory)
   - Suggest spot instances when on-demand capacity is constrained

**User Experience Examples**:

```bash
# Pre-launch quota check
$ cws launch gpu-ml-workstation my-training --size XL
⚠️  Quota Check Failed
    Instance type: p3.8xlarge (32 vCPUs, 4 GPUs)
    Current usage: 24/32 vCPUs in us-west-2
    After launch: 56/32 vCPUs ❌ (24 over limit)

    You need to request a vCPU quota increase:
    1. Visit: https://console.aws.amazon.com/servicequotas/home/services/ec2/quotas/L-1216C47A
    2. Request new limit: 64 vCPUs (allows 2 simultaneous p3.8xlarge)
    3. Typical approval time: 24-48 hours

    Alternative: Launch p3.2xlarge instead? (8 vCPUs, 1 GPU) [Y/n]

# Intelligent AZ failover
$ cws launch bioinformatics-suite genome-analysis
✅ Launching r5.4xlarge in us-west-2a...
⚠️  InsufficientInstanceCapacity in us-west-2a
🔄 Retrying in us-west-2b...
✅ Successfully launched in us-west-2b!
🔗 SSH ready in ~90 seconds...

# AWS Health monitoring
$ cws launch python-ml earthquake-prediction --region us-east-1
⚠️  AWS Health Alert: Degraded EC2 Performance in us-east-1
    Event: API_ISSUE (started 15 minutes ago)
    Impact: Elevated launch failures and delayed instance starts
    Status: AWS engineers investigating

    Recommendation: Use us-west-2 (healthy) or wait ~30 minutes
    Launch anyway? [y/N]

# Quota status command
$ cws admin quota show
📊 AWS Service Quotas - us-west-2

vCPU Limits:
  Standard (A, C, D, H, I, M, R, T, Z): 24/32 (75% used) ⚠️
  GPU (P, G, Inf, DL, Trn):             0/8 (0% used) ✅

Instance Type Limits:
  p3.2xlarge: 0/2 available ✅
  r5.xlarge:  3/5 available ⚠️ (approaching limit)
  t3.medium:  8/20 available ✅

Storage:
  EBS General Purpose (gp3): 3.2TB / 50TB ✅

Recommendations:
  ⚠️  Consider requesting vCPU increase for compute-intensive work
  ✅ GPU quota sufficient for current workload
```

**Target Release**: v0.6.0 (Q2 2026)
**Effort**: Large (2-3 weeks for full implementation)
**Priority**: High - dramatically improves user experience and reduces support burden

**Dependencies**:
- AWS Service Quotas API
- AWS Health API (requires Business or Enterprise Support for full access)
- EC2 DescribeInstanceTypeOfferings for AZ availability

**Related Issues**:
- Reduces "why did my launch fail?" support tickets
- Makes CloudWorkStation intelligent about AWS constraints
- Enables researchers to self-service quota increases
- Proactive problem prevention vs reactive error handling

---

### 3. Multi-User Authentication System
**Location**: `pkg/daemon/middleware.go:103`
**Current Behavior**: Uses bearer token directly as user ID without validation
**Impact**: No real authentication for institutional deployments
**Implementation Needed**:
- Add OAuth/OIDC provider integration (Google, Microsoft, institutional SSO)
- Implement LDAP/Active Directory authentication
- Add SAML support for enterprise SSO
- Create token validation and session management
- Implement role-based access control (RBAC)
- Add user permission system
**Target Release**: v0.6.0 - v0.7.0
**Effort**: Large (2-3 weeks)
**Priority**: High - critical for institutional deployments

### 4. SSM File Operations Support
**Location**: `pkg/daemon/template_application_handlers.go:287`
**Current Behavior**: SSM executor created with nil clients for file operations
**Impact**: `CopyFile()` and `GetFile()` methods not functional
**Implementation Needed**:
- Pass real SSM client to SystemsManagerExecutor
- Add S3 client and bucket configuration
- Implement file upload/download via S3 + SSM
- Add file transfer progress reporting
**Target Release**: v0.5.6
**Effort**: Medium (3-5 days)
**Priority**: Medium - needed for advanced template provisioning

---

## User Experience Items

### 11. Auto-Update Feature
**Location**: New module `pkg/update/`
**Current Behavior**: No version detection, users manually check for updates
**Impact**: Users run outdated versions, miss bug fixes and features
**Implementation Needed**:
- **Phase 1 (v0.6.0)**: Version detection and notifications (2-3 days)
  - Query GitHub Releases API for latest version
  - Add `cws version --check-update` command
  - Show startup notifications in CLI/TUI/GUI
  - Cache update checks (24-hour TTL)
- **Phase 2 (v0.6.1)**: Assisted platform-specific updates (3-4 days)
  - `cws update` command with platform detection
  - Homebrew, apt, manual install support
  - Checksum verification and rollback
- **Phase 3 (v0.7.0)**: Background auto-update system (4-5 days)
  - Daemon-managed background checks
  - Auto-download with user permission
  - Release channel support (stable/beta/dev)
**Target Release**: v0.6.0 (Phase 1), v0.6.1 (Phase 2), v0.7.0 (Phase 3)
**Effort**: Large (9-12 days total across 3 phases)
**Priority**: High - improves version adoption and user experience
**GitHub Issue**: #61

### 12. GUI System Tray/Menu Bar and Auto-Start
**Location**: `cmd/cws-gui/main.go` (Wails 3 integration)
**Current Behavior**: No system tray, no auto-start, GUI must stay visible
**Impact**: Reduced convenience compared to other dev tools (Docker Desktop, VS Code)
**Implementation Needed**:
- **Phase 1 (v0.6.0)**: System tray/menu bar integration (3-4 days)
  - Native macOS menu bar, Windows system tray, Linux tray
  - Context menu with common actions (Quick Launch, My Workspaces, Cost Summary)
  - Minimize to tray instead of taskbar
  - Native notifications for events
- **Phase 2 (v0.6.0)**: Auto-start on login (2-3 days)
  - macOS Launch Agent creation
  - Windows Registry startup
  - Linux XDG autostart
  - Settings UI for auto-start preferences
  - `--minimized` flag support
- **Phase 3 (v0.6.1)**: Advanced tray features (2-3 days)
  - Quick Launch from system tray
  - Intelligent notifications (launch complete, budget alerts, idle warnings)
  - Context-aware menu (recent templates, instances needing attention)
**Target Release**: v0.6.0 (Phases 1-2), v0.6.1 (Phase 3)
**Effort**: Large (7-10 days total across 3 phases)
**Priority**: High - significantly improves GUI usability and convenience
**GitHub Issue**: #62

### 13. EC2 Capacity Blocks for ML Support
**Location**: New module `pkg/aws/capacity_blocks.go`
**Current Behavior**: No support for reserved GPU capacity blocks
**Impact**: Researchers can't guarantee GPU availability for scheduled training runs
**Problem**: Large ML workloads need guaranteed capacity for multi-day training jobs. On-demand launches can fail with InsufficientInstanceCapacity, especially for high-end GPU instances (P5, P4d, Trn1). Capacity Blocks solve this by allowing advance reservation (up to 8 weeks) with guaranteed capacity and discounted pricing.
**Implementation Needed**:
- **Phase 1 (v0.7.0)**: Capacity Block Discovery (4-5 days)
  - API: DescribeCapacityBlockOfferings
  - CLI: `cws capacity-blocks search` with cost comparison
  - Search by instance type, count, duration (1-14 days), date range
- **Phase 2 (v0.7.0)**: Reservation Purchase (3-4 days)
  - API: PurchaseCapacityBlock (upfront payment, immutable)
  - CLI: `cws capacity-blocks purchase` with budget integration
  - State management for reservations
  - Confirmation dialog with warnings (cannot cancel)
- **Phase 3 (v0.7.1)**: Scheduled Launch (5-6 days)
  - Launch instances using capacity reservation ID
  - Pre-launch validation (time window, capacity availability)
  - **Scheduling Options**: Daemon-based (simple) or Lambda-based (reliable)
  - Auto-launch when reservation becomes active
- **Phase 4 (v0.7.1)**: Management & Monitoring (3-4 days)
  - List reservations (active, scheduled, past)
  - Utilization tracking (X/N instances used)
  - Cost analytics integration
  - Underutilization warnings
- **Phase 5 (v0.7.2)**: GUI Integration (4-5 days)
  - Visual calendar picker for date selection
  - Reservation dashboard with timeline view
  - Cost comparison charts
  - Scheduled launch interface
**Key Challenges**:
- **Immutability**: Reservations cannot be modified or canceled after purchase
- **Upfront Costs**: Full payment charged within 12 hours (budget planning required)
- **Scheduling**: Reliable auto-launch requires Lambda/EventBridge (daemon-based option for simplicity)
- **Utilization**: Users must maximize capacity usage to justify cost
**Technical Details**:
- Supported: p6-b200, p5, p5e, p5en, p4d, p4de, trn1, trn2 instances
- Duration: 1-14 days (1-day increments), up to 182 days (7-day increments)
- Cluster sizes: 1-64 instances (512 GPUs or 1024 Trainium chips)
- Pricing: Upfront with ~10-20% discount vs on-demand
- Advance booking: Up to 8 weeks before start date
**Target Release**: v0.7.0 (Phases 1-2), v0.7.1 (Phases 3-4), v0.7.2 (Phase 5)
**Effort**: Large (19-24 days total across 5 phases)
**Priority**: Medium - valuable for large ML workloads, institutional deployments
**GitHub Issue**: #63

---

## Medium Priority Items

### 5. TUI Project Member Management
**Location**: `internal/tui/models/projects.go:313`
**Current Behavior**: Shows member count, directs to CLI for details
**Impact**: TUI provides overview only, no direct member management
**Implementation Needed**:
- Add paginated member list view
- Implement member addition dialog
- Add member removal confirmation
- Create role change interface
- Add member search/filter
**Target Release**: v0.6.1
**Effort**: Medium (4-6 days)
**Priority**: Medium - improves TUI completeness

### 6. TUI Project Instance List
**Location**: `internal/tui/models/projects.go:339`
**Current Behavior**: Shows instance count, directs to CLI or main instance view
**Impact**: No project-filtered instance view in TUI
**Implementation Needed**:
- Add project-filtered instance API call
- Create project instance list view
- Implement instance actions from project view
- Add instance search/filter within project
**Target Release**: v0.6.1
**Effort**: Medium (3-4 days)
**Priority**: Low - main instance view already provides this functionality

### 7. TUI Cost Breakdown Display
**Location**: `internal/tui/models/budget.go:352`
**Current Behavior**: Shows placeholder text, directs to CLI
**Impact**: No detailed cost breakdown in TUI
**Implementation Needed**:
- Call cost breakdown API endpoint
- Parse and format cost data for TUI
- Add service breakdown visualization
- Implement cost trend chart (if TUI space permits)
**Target Release**: v0.6.1
**Effort**: Small (2-3 days)
**Priority**: Low - CLI provides excellent cost breakdown already

### 8. TUI Hibernation Savings Display
**Location**: `internal/tui/models/budget.go:414`
**Current Behavior**: Shows placeholder text, directs to CLI
**Impact**: No hibernation savings visualization in TUI
**Implementation Needed**:
- Call hibernation savings API endpoint
- Calculate savings by project
- Display savings trends over time
- Add savings forecast
**Target Release**: v0.6.1
**Effort**: Small (2-3 days)
**Priority**: Low - savings API fully functional in CLI

---

## Low Priority Items

### 9. TUI Project Creation Dialog
**Location**: `internal/tui/models/projects.go:433`
**Current Behavior**: Returns error, directs to CLI
**Impact**: Project creation must be done via CLI
**Implementation Needed**:
- Design TUI form dialog for project creation
- Add input validation (name, owner email, description)
- Implement project creation API call from TUI
- Add success/error feedback
- Handle edge cases (duplicate names, invalid owners)
**Target Release**: v0.7.0
**Effort**: Medium (3-4 days)
**Priority**: Low - CLI provides excellent UX for complex input

### 10. TUI Budget Creation Dialog
**Location**: `internal/tui/models/budget.go:467`
**Current Behavior**: Returns error, directs to CLI
**Impact**: Budget creation must be done via CLI
**Implementation Needed**:
- Design multi-step TUI form for budget creation
- Add amount, period, and limit inputs
- Implement alert configuration in TUI
- Add automated action configuration
- Create budget creation API call
**Target Release**: v0.7.0
**Effort**: Large (5-7 days)
**Priority**: Low - CLI with flags provides superior UX for complex configuration

---

## Architectural Improvements

### 11. Cobra Flag System Integration
**Location**: `internal/cli/instance_impl.go:267`
**Current Behavior**: Manual flag parsing for direct API usage
**Impact**: Duplicate flag parsing logic for backwards compatibility
**Implementation Needed**:
- Deprecate direct API usage path
- Migrate all callers to Cobra commands
- Remove manual flag parsing
- Clean up legacy API compatibility layer
**Target Release**: v0.8.0
**Effort**: Medium (4-5 days)
**Priority**: Low - current dual-path works correctly

---

## Implementation Strategy

### Phase Assignments

**v0.5.6 (Q1 2026)**: Storage & Template Enhancements
- Item #3: SSM File Operations Support

**v0.6.0 (Q2 2026)**: Security, Authentication & User Experience
- Item #2: AWS Quota Management and Intelligent Availability Handling
- Item #3: Multi-User Authentication System (Phase 1)
- Item #11: Auto-Update Feature (Phase 1: Version Detection)
- Item #12: GUI System Tray/Menu Bar and Auto-Start

**v0.6.1 (Q2 2026)**: TUI Polish & Advanced Features
- Item #5: TUI Project Member Management
- Item #6: TUI Project Instance List
- Item #7: TUI Cost Breakdown Display
- Item #8: TUI Hibernation Savings Display
- Item #11: Auto-Update Feature (Phase 2: Assisted Update)
- Item #12: GUI System Tray (Phase 3: Advanced Features)

**v0.7.0 (Q3 2026)**: Advanced UI Features & GPU Scheduling
- Item #3: Multi-User Authentication System (Phase 2)
- Item #9: TUI Project Creation Dialog
- Item #10: TUI Budget Creation Dialog
- Item #11: Auto-Update Feature (Phase 3: Background Updates)
- Item #13: EC2 Capacity Blocks for ML (Phases 1-2: Discovery & Reservation)

**v0.7.1 (Q3 2026)**: Capacity Block Scheduling & Management
- Item #13: EC2 Capacity Blocks for ML (Phases 3-4: Scheduled Launch & Management)

**v0.7.2 (Q4 2026)**: GUI Enhancements
- Item #13: EC2 Capacity Blocks for ML (Phase 5: GUI Integration)

**v0.8.0 (Q4 2026)**: Code Cleanup & Modernization
- Item #4: Cobra Flag System Integration

---

## Tracking Metrics

- **Total Items**: 14 (added Capacity Blocks for ML)
- **Completed**: 2 items (SSH Readiness Progress, IAM Instance Profile Validation)
- **Remaining**: 12 items
- **High Priority**: 4 items (AWS Quota Management, Multi-User Auth, Auto-Update, GUI System Tray)
- **Medium Priority**: 5 items (includes Capacity Blocks for ML)
- **Low Priority**: 3 items
- **Estimated Remaining Effort**: 16-18 weeks of development time (added 4 weeks for Capacity Blocks)

---

## Notes

1. **All items are intentional deferrals**, not bugs or broken functionality
2. **Current implementations work correctly** - these are enhancements
3. **Priority based on user impact** and institutional deployment needs
4. **Effort estimates** assume single developer, may be parallelized
5. **Target releases** are tentative and may shift based on priorities

---

## Review Schedule

This backlog should be reviewed:
- **Monthly**: Reprioritize based on user feedback
- **Quarterly**: Adjust target releases based on roadmap
- **Before each release**: Confirm items for upcoming version
