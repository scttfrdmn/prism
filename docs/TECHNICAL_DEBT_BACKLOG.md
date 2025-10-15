# Technical Debt and Enhancement Backlog

**Last Updated**: October 15, 2025
**Status**: Active tracking of deferred implementations

This document tracks features that were intentionally deferred during development, marked as design decisions rather than immediate TODO items. These represent real work that should be scheduled for future releases.

---

## High Priority Items

### 1. IAM Instance Profile Validation
**Location**: `pkg/aws/manager.go:1545`
**Current Behavior**: Always returns `false` for painless onboarding
**Impact**: Templates can specify IAM profiles but they're not validated before launch
**Implementation Needed**:
- Add IAM client to Manager struct
- Implement `GetInstanceProfile()` API call
- Add profile validation before instance launch
- Handle graceful fallback when profile doesn't exist
**Target Release**: v0.6.0
**Effort**: Medium (2-3 days)
**Priority**: High - enables secure AWS resource access

### 2. Multi-User Authentication System
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

### 3. SSM File Operations Support
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

### 4. TUI Project Member Management
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

### 5. TUI Project Instance List
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

### 6. TUI Cost Breakdown Display
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

### 7. TUI Hibernation Savings Display
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

### 8. TUI Project Creation Dialog
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

### 9. TUI Budget Creation Dialog
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

### 10. Cobra Flag System Integration
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

- **Total Items**: 10
- **High Priority**: 3 items
- **Medium Priority**: 4 items
- **Low Priority**: 3 items
- **Estimated Total Effort**: 8-10 weeks of development time

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
