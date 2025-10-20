# Technical Debt and Enhancement Backlog

**Last Updated**: October 17, 2025
**Status**: Active tracking of deferred implementations

This document tracks features that were intentionally deferred during development, marked as design decisions rather than immediate TODO items. These represent real work that should be scheduled for future releases.

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

**v0.6.0 (Q2 2026)**: Security & Authentication
- Item #1: IAM Instance Profile Validation
- Item #2: Multi-User Authentication System (Phase 1)

**v0.6.1 (Q2 2026)**: TUI Polish
- Item #4: TUI Project Member Management
- Item #5: TUI Project Instance List
- Item #6: TUI Cost Breakdown Display
- Item #7: TUI Hibernation Savings Display

**v0.7.0 (Q3 2026)**: Advanced UI Features
- Item #2: Multi-User Authentication System (Phase 2)
- Item #8: TUI Project Creation Dialog
- Item #9: TUI Budget Creation Dialog

**v0.8.0 (Q4 2026)**: Code Cleanup & Modernization
- Item #10: Cobra Flag System Integration

---

## Tracking Metrics

- **Total Items**: 11
- **Completed**: 2 items (SSH Readiness Progress, IAM Instance Profile Validation)
- **Remaining**: 9 items
- **High Priority**: 2 items (down from 3)
- **Medium Priority**: 4 items
- **Low Priority**: 3 items
- **Estimated Remaining Effort**: 8-9 weeks of development time (down from 9-11 weeks)

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
