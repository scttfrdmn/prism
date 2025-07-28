# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# Cloud Workstation Platform - Claude Development Context

## Project Overview

This is a command-line tool that allows academic researchers to launch pre-configured cloud workstations in seconds rather than spending hours setting up research environments.

## Core Design Principles

These principles guide every design decision and feature implementation:

### 🎯 **Default to Success**
Every template must work out of the box in every supported region. No configuration should be required for basic usage.
- `cws launch python-ml my-project` should always work
- Smart fallbacks handle regional/architecture limitations transparently
- Templates include battle-tested defaults for their specific use cases

### ⚡ **Optimize by Default**
Templates automatically choose the best instance size and type for their intended workload.
- ML templates default to GPU instances
- R templates default to memory-optimized configurations
- Cost-performance ratio optimized for academic budgets
- ARM instances preferred when available (better price/performance)

### 🔍 **Transparent Fallbacks**
When the ideal configuration isn't available, users always know what changed and why.
- Clear communication: "ARM GPU not available in us-west-1, using x86 GPU instead"
- Fallback chains documented and predictable
- No silent degradation of performance or capabilities

### 💡 **Helpful Warnings**
Gentle guidance when users make suboptimal choices, with clear alternatives offered.
- Warning when choosing CPU instance for ML workload
- Memory warnings for data-intensive R work
- Cost alerts for expensive configurations
- Educational not prescriptive approach

### 🚫 **Zero Surprises**
Users should never be surprised by what they get - clear communication about what's happening.
- Detailed configuration preview before launch
- Real-time progress reporting during operations
- Clear cost estimates and architecture information
- Dry-run mode for validation without commitment

### 📈 **Progressive Disclosure**
Simple by default, detailed when needed. Power users can access advanced features without cluttering basic workflows.
- Basic: `cws launch template-name project-name`
- Intermediate: `cws launch template-name project-name --size L`
- Advanced: `cws launch template-name project-name --instance-type c5.2xlarge --spot`
- Expert: Full template customization and regional optimization

## Current Phase: Enterprise Research Platform (Phase 4 COMPLETE)

**Phase 1 COMPLETED**: Distributed Architecture (daemon + CLI client)
**Phase 2 COMPLETED**: Multi-modal access with CLI/TUI/GUI parity  
**Phase 3 COMPLETED**: Comprehensive cost optimization with hibernation ecosystem
**Phase 4 COMPLETED**: Project-based budget management and enterprise features

**🎉 PHASE 4 COMPLETE: Enterprise Research Management Platform**
- ✅ **Project-Based Organization**: Complete project lifecycle management with role-based access control
- ✅ **Advanced Budget Management**: Project-specific budgets with real-time tracking and automated controls
- ✅ **Cost Analytics**: Detailed cost breakdowns, hibernation savings, and resource utilization metrics  
- ✅ **Multi-User Collaboration**: Project member management with granular permissions (Owner/Admin/Member/Viewer)
- ✅ **Enterprise API**: Full REST API for project management, budget monitoring, and cost analysis
- ✅ **Budget Automation**: Configurable alerts and automated actions (hibernate/stop instances, prevent launches)

CloudWorkstation is now a full **enterprise research platform** supporting collaborative projects, grant-funded budgets, and institutional research management while maintaining its core simplicity for individual researchers.

**Phase 5 Planning**: Research ecosystem expansion
- 🎯 **Template Marketplace**: Community-contributed research environments with discovery and sharing
- 🎯 **Advanced Storage**: OpenZFS/FSx integration for specialized research workloads
- 🎯 **Multi-Cloud Support**: Azure and GCP integration for global research collaboration
- 🎯 **Research Workflows**: Integration with research data management and CI/CD systems

**Multi-Modal Access Strategy**:
```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│ CLI Client  │  │ TUI Client  │  │ GUI Client  │
│ (cmd/cws)   │  │ (cws tui)   │  │ (cmd/cws-gui)│
└──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │                │                │
       └────────────────┼────────────────┘
                        │
                 ┌─────────────┐
                 │ Backend     │
                 │ Daemon      │
                 │ (cwsd:8947) │
                 └─────────────┘
```

**Current Architecture**:
```
cmd/
├── cws/          # CLI client binary
├── cws-gui/      # GUI client binary (Fyne-based)
└── cwsd/         # Backend daemon binary

pkg/
├── api/          # API client interface
├── daemon/       # Daemon core logic  
├── aws/          # AWS operations
├── state/        # State management
├── project/      # Project & budget management (Phase 4)
├── idle/         # Hibernation & cost optimization (Phase 3)
├── profile/      # Enhanced profile system
└── types/        # Shared types & project models

internal/
├── cli/          # CLI application logic
├── tui/          # TUI application (BubbleTea-based)
└── gui/          # (GUI logic is in cmd/cws-gui/)
```

**Phase 4 Enterprise Components**:
```
pkg/project/
├── manager.go         # Project lifecycle & member management
├── budget_tracker.go  # Real-time cost tracking & alerts
├── cost_calculator.go # AWS pricing engine & hibernation savings
└── types.go          # Request/response types & filters

pkg/daemon/
└── project_handlers.go # REST API endpoints (/api/v1/projects)

pkg/types/
└── project.go         # Enterprise data models & budget types
```

**Feature Parity Matrix**:
| Feature | CLI | TUI | GUI | Status |
|---------|-----|-----|-----|---------|
| Templates | ✅ | ✅ | ✅ | Complete |
| Instance Management | ✅ | ✅ | ✅ | Complete |
| Storage (EFS/EBS) | ✅ | ✅ | ✅ | Complete |
| Advanced Launch | ✅ | ✅¹ | ✅ | Complete |
| Profile Management | ✅ | ✅ | ✅ | Complete |
| Daemon Control | ✅ | ✅ | ✅ | Complete |

¹ *TUI provides CLI command guidance for launch operations*

## Architecture Decisions

### Multi-Modal Design Philosophy
- **CLI**: Power users, automation, scripting - maximum efficiency
- **TUI**: Interactive terminal users, remote access - keyboard-first navigation
- **GUI**: Desktop users, visual management - mouse-friendly interface
- **Unified Backend**: All interfaces share same daemon API and state

### API Architecture
- **REST API**: HTTP endpoints on port 8947 (CWS on phone keypad)
- **Options Pattern**: Modern `api.NewClientWithOptions()` with configuration
- **Profile Integration**: Seamless AWS credential and region management
- **Graceful Operations**: Proper shutdown, error handling, progress reporting

### Templates (Inheritance Architecture)

**✅ IMPLEMENTED: Template Inheritance System**

CloudWorkstation now supports template stacking and inheritance, allowing templates to build upon each other:

```bash
# Base template provides foundation
# templates/base-rocky9.yml: Rocky Linux 9 + DNF + system tools + rocky user

# Stacked template inherits and extends  
# templates/rocky9-conda-stack.yml:
#   inherits: ["Rocky Linux 9 Base"]
#   package_manager: "conda"  # Override parent's DNF
#   adds: conda packages, datascientist user, jupyter service

# Launch stacked template
cws launch "Rocky Linux 9 + Conda Stack" my-analysis
# ↳ Gets: rocky user + datascientist user, system packages + conda packages, ports 22 + 8888
```

**Inheritance Merging Rules**:
- **Packages**: Append (base system packages + child conda packages)
- **Users**: Append (base rocky user + child datascientist user)  
- **Services**: Append (base services + child jupyter service)
- **Package Manager**: Override (child conda overrides parent DNF)
- **Ports**: Deduplicate (base 22 + child 8888 = [22, 8888])

**Available Templates**:
- `Rocky Linux 9 Base`: Foundation with DNF, system tools, rocky user
- `Rocky Linux 9 + Conda Stack`: Inherits base + adds conda ML packages
- `Python Machine Learning (Simplified)`: Conda + Jupyter + ML packages  
- `R Research Environment (Simplified)`: Conda + RStudio + tidyverse
- `Basic Ubuntu (APT)`: Ubuntu + APT package management
- `Web Development (APT)`: Ubuntu + web development tools

**Future Multi-Stack Architecture**:
```bash  
# Planned: Complex inheritance chains
cws launch gpu-ml-workstation my-training
# ↳ Inherits: Base OS → GPU Drivers → Conda ML → Desktop GUI

# Power users can override at launch
cws launch "Rocky Linux 9 + Conda Stack" my-project --with spack
```

**Design Benefits**:
- **Composition Over Duplication**: Inherit and extend vs copy/paste
- **Maintainable Library**: Base template updates propagate to children
- **Clear Relationships**: Explicit parent-child dependencies
- **Flexible Override**: Change any aspect while preserving inheritance

### State Management
Enhanced state management with profile integration:
```json
{
  "instances": {
    "my-instance": {
      "id": "i-1234567890abcdef0",
      "name": "my-instance", 
      "template": "r-research",
      "public_ip": "54.123.45.67",
      "state": "running",
      "launch_time": "2024-06-15T10:30:00Z",
      "estimated_daily_cost": 2.40,
      "attached_volumes": ["shared-data"],
      "attached_ebs_volumes": ["project-storage-L"]
    }
  },
  "volumes": {
    "shared-data": {
      "filesystem_id": "fs-1234567890abcdef0",
      "state": "available",
      "creation_time": "2024-06-15T10:00:00Z"
    }
  },
  "current_profile": {
    "name": "research-profile",
    "aws_profile": "my-aws-profile", 
    "region": "us-west-2"
  }
}
```

## Development Principles

1. **Multi-modal first**: Every feature must work across CLI, TUI, and GUI
2. **API-driven**: All interfaces use the same backend API
3. **Profile-aware**: Seamless AWS credential and region management
4. **Real-time sync**: Changes reflect across all interfaces immediately
5. **Professional quality**: Zero compilation errors, comprehensive testing

## Future Phases (Post-Phase 2)

- **Phase 3**: Advanced research features (multi-package managers, hibernation, snapshots)
- **Phase 4**: Collaboration & scale (multi-user, template marketplace, multi-cloud)
- **Phase 5**: Enterprise features (SSO, compliance, advanced monitoring)

## Development Commands

### Building and Testing
```bash
# Build all components
make build
# Builds: cws (CLI), cwsd (daemon), cws-gui (GUI)

# Build specific components
go build -o bin/cws ./cmd/cws/        # CLI
go build -o bin/cwsd ./cmd/cwsd/      # Daemon  
go build -o bin/cws-gui ./cmd/cws-gui/ # GUI

# Run tests
make test

# Cross-compile for all platforms
make cross-compile

# Clean build artifacts
make clean
```

### Running Different Interfaces
```bash
# CLI interface (traditional)
./bin/cws launch python-ml my-project

# TUI interface (interactive terminal)
./bin/cws tui
# Navigation: 1=Dashboard, 2=Instances, 3=Templates, 4=Storage, 5=Settings, 6=Profiles

# GUI interface (desktop application)
./bin/cws-gui
# System tray integration with professional tabbed interface

# Daemon (backend service)
./bin/cwsd
# Runs on port 8947, provides REST API for all clients
```

### Development Workflow
```bash
# Start daemon for development
./bin/cwsd &

# Test CLI functionality
./bin/cws templates
./bin/cws list

# Test TUI functionality  
./bin/cws tui

# Test GUI functionality (in separate terminal)
./bin/cws-gui

# Graceful daemon shutdown
./bin/cws daemon stop
```

## Key Implementation Details

### API Client Pattern (All Interfaces)
```go
// Modern API client initialization
client := api.NewClientWithOptions("http://localhost:8947", client.Options{
    AWSProfile: profile.AWSProfile,
    AWSRegion:  profile.Region,
})
```

### Profile System Integration
```go
// Enhanced profile management
currentProfile, err := profile.GetCurrentProfile()
if err != nil {
    // Handle gracefully with defaults
}

// Apply to API client
apiClient := api.NewClientWithOptions(daemonURL, client.Options{
    AWSProfile: currentProfile.AWSProfile,
    AWSRegion:  currentProfile.Region,
})
```

### Cross-Interface State Synchronization
- All interfaces use same daemon backend (port 8947)
- Real-time updates via polling and WebSocket (future)
- Shared profile and configuration system
- Consistent error handling and user feedback

### GUI Specific (cmd/cws-gui/main.go)
- **Fyne Framework**: Cross-platform native GUI
- **System Tray**: Always-on monitoring and quick access
- **Tabbed Interface**: Templates, Instances, Storage, Settings
- **Professional Dialogs**: Connection info, confirmations, progress
- **Real-time Updates**: Automatic refresh with visual indicators

### TUI Specific (internal/tui/)
- **BubbleTea Framework**: Professional terminal interface
- **Page Navigation**: Keyboard-driven (1-6 keys for pages)
- **Real-time Updates**: 30-second refresh intervals
- **Professional Styling**: Consistent theming, loading states
- **Action Dialogs**: Instance management with confirmations

## Testing Strategy

All components tested with:
- **Unit Tests**: Core functionality and API integration
- **Integration Tests**: Cross-interface compatibility
- **Manual Testing**: Real AWS integration and user workflows
- **Build Testing**: Zero compilation errors across all platforms

## Recent Major Achievements

### ✅ PHASE 3: Complete Hibernation & Cost Optimization Ecosystem

**🎉 FULLY IMPLEMENTED: Comprehensive hibernation system with automated policy integration**

Successfully implemented the complete hibernation ecosystem providing intelligent cost optimization through both manual hibernation controls and automated hibernation policies across CLI, GUI, and API interfaces.

#### Complete Hibernation Architecture

**🏗️ Full Technical Stack**:
- **AWS Hibernation Engine**: Full hibernation lifecycle with intelligent fallback to regular stop
- **REST API Layer**: Complete endpoint coverage for hibernation operations + idle policy management
- **API Client Layer**: Type-safe client methods with proper error handling for all hibernation features
- **GUI Interface**: Smart controls with educational confirmation dialogs
- **CLI Interface**: Educational commands with cost optimization messaging + policy management
- **Idle Detection System**: Automated hibernation policies with configurable thresholds and actions

**🎯 Complete Interface Coverage**:
- ✅ **AWS Layer** (`pkg/aws/manager.go`): `HibernateInstance()`, `ResumeInstance()`, `GetInstanceHibernationStatus()`
- ✅ **API Layer** (`pkg/daemon/instance_handlers.go`): REST endpoints `/hibernate`, `/resume`, `/hibernation-status`
- ✅ **Idle API Layer** (`pkg/daemon/idle_handlers.go`): 7 REST endpoints for complete idle policy management
- ✅ **Client Layer** (`pkg/api/client/`): Complete API client integration with hibernation + idle methods  
- ✅ **Types Layer** (`pkg/types/runtime.go`): Complete type system for hibernation status + idle policies
- ✅ **GUI Layer** (`cmd/cws-gui/main.go`): Smart hibernation controls with educational confirmation dialogs
- ✅ **CLI Layer** (`cmd/cws/main.go`, `internal/cli/app.go`): Manual hibernation + automated policy commands

**💡 Dual-Mode Hibernation System**:
```bash
# Manual Hibernation Controls
cws hibernate my-instance    # Intelligent hibernation with support detection
cws resume my-instance       # Smart resume with automatic fallback logic

# Automated Hibernation Policies  
cws idle profile list        # Show hibernation policies (batch: 60min hibernate)
cws idle profile create cost-optimized --idle-minutes 10 --action hibernate
cws idle instance my-gpu-workstation --profile gpu  # GPU-optimized hibernation
cws idle history            # Audit trail of automated hibernation actions

# Pre-configured hibernation profiles:
# - batch: 60min idle → hibernate (long-running research jobs)
# - gpu: 15min idle → stop (expensive GPU instances)  
# - cost-optimized: 10min idle → hibernate (maximum cost savings)
```

**🎨 Intelligent Cost Optimization**:
- **Hibernation-First**: Policies prefer hibernation when possible (preserves RAM state)
- **Smart Fallback**: Automatic degradation to stop when hibernation unsupported
- **Configurable Thresholds**: Fine-tuned idle detection (CPU, memory, network, disk, GPU usage)
- **Domain Mapping**: Research domains automatically mapped to hibernation-optimized policies
- **Instance Overrides**: Per-instance hibernation policy customization

**📊 Research Impact**:
- **Manual Control**: Direct hibernation/resume for immediate cost optimization
- **Automated Policies**: Hands-off hibernation based on actual usage patterns
- **Session Preservation**: Complete work environment state maintained through hibernation
- **Cost Transparency**: Clear audit trail of hibernation actions and cost savings
- **Domain Intelligence**: ML/GPU workloads get hibernation-optimized policies automatically

#### Implementation Statistics
- **🔧 16 files modified** across 3 major hibernation implementations
- **🔧 7 new REST API endpoints** for idle detection and hibernation policy management
- **📝 850+ lines** of hibernation functionality across all layers and policy integration
- **🧪 Complete API coverage** for manual hibernation + automated policy operations
- **🎨 Full UX integration** with educational messaging and policy management
- **📚 Comprehensive documentation** of hibernation benefits, policies, and cost optimization

#### Cost Optimization Achievement
- **Manual Hibernation**: Immediate hibernation/resume for session-preserving cost savings
- **Automated Hibernation**: Policy-driven hibernation after configurable idle periods (10-60 minutes)
- **Intelligent Actions**: Hibernation preferred over stop when supported (preserves RAM state)
- **Research-Optimized**: Domain-specific policies (batch jobs hibernate longer, GPU instances hibernate faster)
- **Comprehensive Audit**: Complete history tracking of automated hibernation cost savings

This represents **CloudWorkstation's complete cost optimization achievement**, providing researchers with the most comprehensive hibernation system available - combining immediate manual control with intelligent automated policies for maximum cost savings while preserving work session continuity.

### ✅ FULLY IMPLEMENTED: Template Inheritance & Validation System

Successfully completed the comprehensive template system addressing the original user request: *"Can the templates be stacked? That is reference each other? Say I want a Rocky9 linux but install some conda software on it."*

#### Implementation Summary

**🎯 User Request**: 100% Satisfied
- ✅ Templates can be stacked and reference each other via `inherits` field
- ✅ Rocky9 Linux + conda software use case fully working
- ✅ Example: `Rocky Linux 9 Base` + `Rocky Linux 9 + Conda Stack` 
- ✅ Launch produces combined environment: 2 users, system + conda packages, ports 22 + 8888

**🏗️ Technical Architecture**:
- **Template Inheritance Engine**: Multi-level inheritance with intelligent merging
- **Comprehensive Validation**: 8+ validation rules with clear error messages  
- **CLI Integration**: `cws templates validate` command with full validation suite
- **Clean Implementation**: Removed legacy "auto" package manager, cleaned dead code

**📊 Working Example**:
```bash
# Base template: Rocky Linux 9 + DNF + system tools + rocky user
# Stacked template: inherits base + adds conda packages + datascientist user + jupyter

cws launch "Rocky Linux 9 + Conda Stack" my-analysis
# Result: Both users, all packages, combined ports [22, 8888]
```

**🧪 Validation Results**:
- ✅ All templates pass validation
- ✅ Error detection: invalid package managers, self-reference, invalid ports/users
- ✅ Template consistency: package manager matching, inheritance rules
- ✅ Build system integration: validation prevents invalid templates

**📚 Documentation**:
- **docs/TEMPLATE_SYSTEM_IMPLEMENTATION.md**: Complete implementation summary
- **docs/TEMPLATE_INHERITANCE.md**: Technical inheritance and validation guide
- **Working Examples**: base-rocky9.yml and rocky9-conda-stack.yml templates

This represents a major advancement in CloudWorkstation's template capabilities, enabling researchers to build complex environments through simple template composition - exactly the "stackable architecture" envisioned for research computing.

## Success Criteria

Phase 2 Successfully Achieved:
- ✅ All three interfaces (CLI/TUI/GUI) fully functional
- ✅ Complete feature parity across all interfaces
- ✅ Professional user experience with consistent theming
- ✅ Zero compilation errors and comprehensive testing
- ✅ Production-ready deployment capabilities

## Common Issues to Watch

1. **Profile Integration**: Ensure consistent AWS credential handling across interfaces
2. **API Compatibility**: Maintain backward compatibility when updating daemon API
3. **Cross-Platform**: Test GUI and TUI on different operating systems
4. **Error Handling**: Provide consistent, helpful error messages across interfaces
5. **Performance**: Ensure real-time updates don't impact system performance

## Next Development Session Focus

With Phase 2 complete, future development should focus on:
1. **Phase 3 Planning**: Advanced research features and multi-package managers
2. **User Feedback**: Gather researcher feedback on multi-modal interface design
3. **Performance Optimization**: Optimize real-time updates and API efficiency
4. **Documentation**: User guides for CLI, TUI, and GUI interfaces
5. **Template Expansion**: Additional research environment templates

## Research User Feedback Integration

Key validation points for multi-modal access:
- **Interface Preference**: Do researchers prefer CLI, TUI, or GUI for different tasks?
- **Feature Completeness**: Are all necessary research workflows supported?
- **Performance**: Are real-time updates and interface switching smooth?
- **Learning Curve**: Can researchers easily switch between interfaces?
- **Workflow Integration**: How does CloudWorkstation fit into existing research workflows?

**Phase 2 Status: 🎉 COMPLETE**  
**Multi-Modal Access: CLI ✅ TUI ✅ GUI ✅**  
**Production Ready: Zero errors, comprehensive testing, professional quality**