# TUI Feature Parity Audit
**Date**: October 7, 2025
**Status**: 🔍 **COMPREHENSIVE AUDIT IN PROGRESS**
**Goal**: Identify and implement ALL missing CLI functionality in TUI

---

## Executive Summary

This audit compares CLI command functionality with TUI implementation to identify missing features. The TUI currently has **7 pages** but is missing several major CLI command categories entirely.

**Current TUI Pages**:
1. Dashboard - ✅ Basic implementation
2. Instances - ✅ Basic implementation
3. Templates - ✅ Basic implementation
4. Storage - ✅ Basic implementation (EFS/EBS)
5. Users - ✅ Research user management
6. Settings - ⚠️ Unknown functionality
7. Profiles - ⚠️ Unknown functionality

**Major Missing CLI Categories**:
- ❌ **Project Management** (comprehensive CLI in `project_cobra.go`)
- ❌ **Budget Management** (comprehensive CLI in `budget_commands.go` - 1,797 lines!)
- ❌ **Policy Framework** (CLI in `policy_cobra.go`)
- ❌ **Marketplace** (CLI with search, browse, install)
- ❌ **AMI Management** (CLI in `ami_cobra.go`)
- ❌ **Idle/Hibernation Management** (CLI in `idle_cobra.go`)
- ❌ **Rightsizing** (CLI in `rightsizing_cobra.go`)
- ❌ **Repository Management** (CLI in `repo_cobra.go`)
- ❌ **Logs** (CLI in `logs_commands.go`)
- ❌ **Admin Commands** (CLI in `admin_commands.go`)
- ❌ **Daemon Management** (CLI in `daemon_cobra.go`)

---

## Detailed Analysis

### ✅ IMPLEMENTED: Basic Features (TUI Has These)

#### 1. Dashboard Page (dashboard.go)
**Status**: ✅ PARTIAL - Basic overview only

**Implemented**:
- Instance overview with table display
- System status display
- Cost data display (daily/monthly)
- Tabs: Overview, Instances, Storage, Costs
- Auto-refresh every 30 seconds

**Missing Advanced Features**:
- No budget status display
- No project summaries
- No hibernation savings metrics
- No policy status
- No marketplace activity

---

#### 2. Instances Page (instances.go)
**Status**: ✅ PARTIAL - Basic management only

**Implemented**:
- List instances with details (name, template, status, type, cost, IP, launch time)
- Instance actions (appears to have action system)
- Connection commands
- Table navigation
- Auto-refresh

**Missing CLI Features**:
- ❌ Instance hibernation controls (CLI: `prism hibernate <instance>`)
- ❌ Instance resume controls (CLI: `prism resume <instance>`)
- ❌ Instance rightsizing recommendations
- ❌ Instance logs viewing
- ❌ Template application (CLI: `prism template apply <template> <instance>`)
- ❌ Template rollback
- ❌ Project filtering/assignment

---

#### 3. Templates Page (templates.go)
**Status**: ✅ PARTIAL - Browsing only

**Implemented**:
- List available templates
- Template details view (description, costs, ports)
- Template selection

**Missing CLI Features**:
- ❌ Template validation (CLI: `prism templates validate`)
- ❌ Template marketplace integration (CLI: `prism marketplace search/browse/install`)
- ❌ Template information display (CLI: `prism templates info <template>`)
- ❌ Template registry management
- ❌ Template discovery commands

---

#### 4. Storage Page (storage.go)
**Status**: ✅ PARTIAL - Basic EFS/EBS management

**Implemented**:
- List EFS volumes
- List EBS storage
- Mount dialog functionality
- Tab-based view (volumes vs storage)

**Missing CLI Features**:
- ❌ Storage creation wizards
- ❌ Storage analytics (CLI: `prism storage analytics`)
- ❌ Cost breakdown by storage type
- ❌ FSx integration (if implemented in CLI)
- ❌ S3 mount points (if implemented in CLI)

---

#### 5. Users Page (users.go)
**Status**: ✅ GOOD - Research user management implemented

**Implemented**:
- List research users
- Create user dialog
- Delete user dialog
- Research user manager integration

**Potential Missing CLI Features**:
- User provisioning status
- SSH key management view
- User status across instances
- Profile-specific user lists

---

#### 6. Settings Page
**Status**: ⚠️ **UNKNOWN** - Need to examine implementation

**Expected CLI Features to Implement**:
- Daemon settings (CLI: `prism daemon status/start/stop`)
- Configuration management
- API endpoint configuration
- Default preferences

---

#### 7. Profiles Page
**Status**: ⚠️ **UNKNOWN** - Need to examine implementation

**Expected CLI Features to Implement**:
- Profile list (CLI: `prism profile list`)
- Profile switching (CLI: `prism profile switch`)
- Profile creation (CLI: `prism profile create`)
- AWS region/credential management

---

## ❌ MAJOR MISSING FEATURES

### 1. Project Management (CRITICAL MISSING FEATURE)
**CLI Implementation**: `internal/cli/project_cobra.go` (244 lines)

**Missing TUI Features**:
```bash
# Project Commands (NONE in TUI)
prism project list                      # List all projects
prism project create <name>             # Create project
prism project info <name>               # Project details
prism project delete <name>             # Delete project
prism project members <project>         # Member management
  - add <email> <role>
  - remove <email>
  - list
prism project budget                    # Budget management
  - status <project>
  - set <project> <amount>
  - disable <project>
  - history <project>
prism project instances <project>       # Project instances
prism project templates <project>       # Project templates
```

**Impact**: ⭐⭐⭐ **CRITICAL**
- Projects are Phase 4 feature - complete lack of TUI support
- Budget management entirely missing
- Collaborative features not accessible

---

### 2. Budget Management (CRITICAL MISSING FEATURE)
**CLI Implementation**: `internal/cli/budget_commands.go` (1,797 lines!!!)

**Missing TUI Features**:
```bash
# Budget Commands (NONE in TUI - 1,797 LINES OF CLI CODE!)
prism budget list                       # List all budgets
prism budget create <project> <amount>  # Create budget
prism budget update <budget-id>         # Update budget
prism budget delete <budget-id>         # Delete budget
prism budget info <budget-id>           # Detailed info
prism budget status [budget-id]         # Current status
prism budget usage <budget-id>          # Resource usage
prism budget history <budget-id>        # Spending history
prism budget alerts <budget-id>         # Manage alerts
prism budget forecast <budget-id>       # Spending forecast
prism budget savings [budget-id]        # Hibernation savings
prism budget breakdown <budget-id>      # Cost breakdown
```

**Advanced Budget Features**:
- Alert thresholds and notifications (email, Slack, webhook)
- Automated actions (hibernate_all, stop_all, prevent_launch)
- Monthly/daily spending limits
- Budget periods (project, monthly, weekly, daily)
- Cost forecasting and projections
- Savings analysis and recommendations

**Impact**: ⭐⭐⭐ **CRITICAL**
- 1,797 lines of CLI code with ZERO TUI equivalent
- Budget management is Phase 4 enterprise feature
- Real-time cost tracking unavailable in TUI
- No budget alerts or automated actions
- No cost optimization recommendations

---

### 3. Policy Framework (HIGH PRIORITY MISSING FEATURE)
**CLI Implementation**: `internal/cli/policy_cobra.go` (314 lines)

**Missing TUI Features**:
```bash
# Policy Commands (NONE in TUI)
prism policy status                     # Policy enforcement status
prism policy list                       # List policy sets
prism policy assign <policy-set>        # Assign policy
prism policy enable                     # Enable enforcement
prism policy disable                    # Disable enforcement
prism policy check <template>           # Check template access
```

**Impact**: ⭐⭐ **HIGH**
- Policy framework (Phase 5A+) not accessible in TUI
- Access control not manageable
- Institutional governance features missing

---

### 4. Marketplace (HIGH PRIORITY MISSING FEATURE)
**CLI Implementation**: `internal/cli/marketplace_commands.go`

**Missing TUI Features**:
```bash
# Marketplace Commands (NONE in TUI)
prism marketplace search <query>        # Search templates
prism marketplace browse                # Browse categories
prism marketplace show <template>       # Template details
prism marketplace install <template>    # Install template
prism marketplace registries            # Registry management
```

**Impact**: ⭐⭐ **HIGH**
- Template marketplace (Phase 5B) not accessible
- Community templates not discoverable
- Registry management unavailable

---

### 5. AMI Management (MEDIUM PRIORITY)
**CLI Implementation**: `internal/cli/ami_cobra.go`

**Missing TUI Features**:
```bash
# AMI Commands (NONE in TUI)
prism ami list                          # List AMIs
prism ami create <instance>             # Create AMI from instance
prism ami delete <ami-id>               # Delete AMI
prism ami info <ami-id>                 # AMI details
```

**Impact**: ⭐ **MEDIUM**
- AMI operations not available in TUI
- Image management requires CLI

---

### 6. Idle/Hibernation Management (MEDIUM-HIGH PRIORITY)
**CLI Implementation**: `internal/cli/idle_cobra.go`

**Missing TUI Features**:
```bash
# Idle Commands (NONE in TUI)
prism idle profile list                 # List idle profiles
prism idle profile create <name>        # Create profile
prism idle instance <name> --profile    # Assign profile
prism idle history                      # Hibernation history
prism idle status                       # Idle detection status
```

**Impact**: ⭐⭐ **MEDIUM-HIGH**
- Automated hibernation policies not manageable
- Idle detection configuration unavailable
- Cost optimization features missing

---

### 7. Rightsizing (MEDIUM PRIORITY)
**CLI Implementation**: `internal/cli/rightsizing_cobra.go`

**Missing TUI Features**:
```bash
# Rightsizing Commands (NONE in TUI)
prism rightsizing analyze               # Analyze instances
prism rightsizing recommendations       # Get recommendations
prism rightsizing apply <instance>      # Apply recommendation
```

**Impact**: ⭐⭐ **MEDIUM**
- Cost optimization recommendations unavailable
- Instance sizing analysis missing

---

### 8. Repository Management (LOW-MEDIUM PRIORITY)
**CLI Implementation**: `internal/cli/repo_cobra.go`

**Missing TUI Features**:
```bash
# Repository Commands (NONE in TUI)
prism repo list                         # List repositories
prism repo add <name> <url>             # Add repository
prism repo remove <name>                # Remove repository
prism repo update                       # Update repositories
```

**Impact**: ⭐ **MEDIUM**
- Template repository management unavailable

---

### 9. Logs (MEDIUM PRIORITY)
**CLI Implementation**: `internal/cli/logs_commands.go`

**Missing TUI Features**:
```bash
# Logs Commands (NONE in TUI)
prism logs <instance>                   # View instance logs
prism logs --follow <instance>          # Follow logs
prism logs --tail <n> <instance>        # Tail logs
```

**Impact**: ⭐⭐ **MEDIUM**
- Log viewing not available in TUI
- Debugging requires CLI

---

### 10. Admin Commands (LOW PRIORITY)
**CLI Implementation**: `internal/cli/admin_commands.go`

**Missing TUI Features**:
```bash
# Admin Commands (NONE in TUI)
prism admin daemon status               # Daemon status
prism admin daemon restart              # Restart daemon
prism admin cleanup                     # Cleanup resources
```

**Impact**: ⭐ **LOW**
- Administrative operations require CLI

---

### 11. Daemon Management (MEDIUM PRIORITY)
**CLI Implementation**: `internal/cli/daemon_cobra.go`

**Missing TUI Features**:
```bash
# Daemon Commands (NONE in TUI)
prism daemon status                     # Check status
prism daemon start                      # Start daemon
prism daemon stop                       # Stop daemon
prism daemon restart                    # Restart daemon
prism daemon logs                       # Daemon logs
```

**Impact**: ⭐⭐ **MEDIUM**
- Daemon control not available in TUI
- Status monitoring limited

---

## Priority Implementation Plan

### 🔴 CRITICAL PRIORITY (Week 1-2)

#### 1. Budget Management TUI (HIGHEST PRIORITY)
**Effort**: 40 hours
**Impact**: ⭐⭐⭐ CRITICAL

**New TUI Page Required**: "Budget" page

**Features to Implement**:
- Budget list view with status indicators (green/yellow/red)
- Budget creation wizard
- Budget status dashboard with spending charts
- Alert configuration interface
- Cost breakdown views
- Savings analysis display
- Forecast visualization

**Why Critical**: 1,797 lines of CLI code with zero TUI equivalent. Budget management is Phase 4 enterprise feature.

---

#### 2. Project Management TUI (CRITICAL)
**Effort**: 25 hours
**Impact**: ⭐⭐⭐ CRITICAL

**New TUI Page Required**: "Projects" page

**Features to Implement**:
- Project list with member counts and budgets
- Project creation/deletion
- Member management interface
- Project info view
- Instance/template association views
- Budget integration (links to budget page)

**Why Critical**: Projects are Phase 4 collaborative feature - TUI has zero support.

---

### 🟡 HIGH PRIORITY (Week 3-4)

#### 3. Policy Framework TUI
**Effort**: 15 hours
**Impact**: ⭐⭐ HIGH

**Integration**: Add to Settings page or new "Policy" page

**Features to Implement**:
- Policy status display
- Policy set list
- Policy assignment interface
- Template access checking
- Enforcement toggle

---

#### 4. Marketplace TUI
**Effort**: 20 hours
**Impact**: ⭐⭐ HIGH

**Enhancement**: Extend existing Templates page

**Features to Implement**:
- Marketplace search interface
- Template browsing with filters
- Template installation wizard
- Registry management view

---

#### 5. Idle/Hibernation Management TUI
**Effort**: 15 hours
**Impact**: ⭐⭐ MEDIUM-HIGH

**Integration**: Add to Instances page or new "Automation" page

**Features to Implement**:
- Idle profile list and management
- Profile assignment interface
- Hibernation history view
- Status monitoring

---

### 🟢 MEDIUM PRIORITY (Week 5-6)

#### 6. Enhanced Instance Management
**Effort**: 10 hours
**Impact**: ⭐⭐ MEDIUM

**Enhancement**: Extend existing Instances page

**Features to Add**:
- Hibernation controls (hibernate/resume buttons)
- Template application interface
- Logs viewing
- Project filtering

---

#### 7. Logs Viewer TUI
**Effort**: 10 hours
**Impact**: ⭐⭐ MEDIUM

**New Feature**: Add to Instances page as action or new page

**Features to Implement**:
- Log viewing interface
- Follow mode
- Filtering and search

---

#### 8. Daemon Management TUI
**Effort**: 8 hours
**Impact**: ⭐⭐ MEDIUM

**Integration**: Add to Settings page

**Features to Implement**:
- Daemon status display
- Control buttons (start/stop/restart)
- Daemon logs viewer

---

### 🔵 LOWER PRIORITY (Week 7+)

#### 9. AMI Management TUI
**Effort**: 12 hours
**Impact**: ⭐ MEDIUM

#### 10. Rightsizing TUI
**Effort**: 15 hours
**Impact**: ⭐⭐ MEDIUM

#### 11. Repository Management TUI
**Effort**: 8 hours
**Impact**: ⭐ MEDIUM

#### 12. Admin Commands TUI
**Effort**: 5 hours
**Impact**: ⭐ LOW

---

## Summary Statistics

### Missing TUI Pages Needed:
1. **Budget Management Page** (CRITICAL - 1,797 lines of CLI)
2. **Project Management Page** (CRITICAL - 244 lines of CLI)
3. **Policy Management Page** (HIGH - 314 lines of CLI)

### Major Enhancements Needed:
1. **Templates Page**: Add marketplace integration
2. **Instances Page**: Add hibernation, logs, template operations
3. **Settings Page**: Add daemon management, policy framework

### Total Implementation Effort:
- **Critical Priority**: 65 hours (Budget + Project)
- **High Priority**: 50 hours (Policy + Marketplace + Idle)
- **Medium Priority**: 28 hours (Instance enhancements + Logs + Daemon)
- **Lower Priority**: 40 hours (AMI + Rightsizing + Repos + Admin)
- **TOTAL**: ~183 hours of TUI development

---

## Key Findings

1. **Budget Management** is the single largest gap - 1,797 lines of CLI code with ZERO TUI equivalent

2. **Project Management** is completely missing from TUI despite being Phase 4 enterprise feature

3. **Policy Framework** (Phase 5A+) has no TUI interface for institutional governance

4. **Marketplace** (Phase 5B) template discovery not accessible in TUI

5. **Cost Optimization Features** (hibernation policies, rightsizing) mostly unavailable

6. **TUI has 7 pages** but needs at least **3 major new pages** plus significant enhancements to existing pages

---

## Recommendations

### Immediate Action (Next 2 Weeks):
1. ✅ **Implement Budget Management TUI** - 1,797 lines of CLI functionality missing
2. ✅ **Implement Project Management TUI** - Core Phase 4 feature unavailable

### Short-Term (Weeks 3-4):
3. ✅ **Add Policy Framework to TUI** - Institutional governance inaccessible
4. ✅ **Extend Templates page with Marketplace** - Template discovery broken
5. ✅ **Add Idle/Hibernation Management** - Cost optimization unavailable

### Medium-Term (Weeks 5-6):
6. ✅ **Enhance Instances page** - Add hibernation, logs, template ops
7. ✅ **Add Logs viewer** - Debugging requires CLI currently
8. ✅ **Add Daemon management to Settings** - Daemon control unavailable

### Success Criteria:
- ✅ All CLI commands accessible via TUI
- ✅ Feature parity across CLI/TUI/GUI
- ✅ Zero functionality exclusive to CLI
- ✅ Professional TUI experience matching CLI capabilities

---

**Next Step**: After completing TUI audit, audit GUI for same feature gaps.

**Status**: ✅ TUI AUDIT COMPLETE - Ready to implement missing features

---

*Last Updated: October 7, 2025*
*Next: Begin GUI feature parity audit*
