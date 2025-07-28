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

## Current Phase: Advanced Research Features (Phase 3 IN PROGRESS)

**Phase 1 COMPLETED**: Distributed Architecture (daemon + CLI client)
**Phase 2 COMPLETED**: Multi-modal access with CLI/TUI/GUI parity  
**Phase 3 Sprint 1 COMPLETED**: Multi-package template system activation

**Sprint 1 Achievements**: 
- ✅ **Template System Integration**: Daemon exclusively uses unified YAML template system
- ✅ **Legacy Elimination**: Removed hardcoded template fallbacks completely
- ✅ **Multi-Package Foundation**: Established conda/spack/apt integration architecture
- ✅ **Template Scanning**: Robust directory scanning across multiple locations
- ✅ **API Compatibility**: Maintained full backward compatibility for all clients
- ✅ **Technical Debt Cleanup**: No more hardcoded template maintenance required

**Phase 3 Sprint 2 Goals**: Template optimization and advanced features  
- ✅ CLI `--with conda` package manager support (completed)
- ✅ Script generator template execution fixes (completed)
- Template validation and conda optimization
- Advanced template features (hibernation, cost optimization)
- Conda-based specialized research templates

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
├── profile/      # Enhanced profile system
└── types/        # Shared types

internal/
├── cli/          # CLI application logic
├── tui/          # TUI application (BubbleTea-based)
└── gui/          # (GUI logic is in cmd/cws-gui/)
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

### ✅ PHASE 3: Hibernation & Cost Optimization System

**Complete hibernation system implementation for intelligent cost optimization**

Successfully implemented comprehensive hibernation capabilities addressing CloudWorkstation's Phase 3 advanced features for cost optimization through intelligent instance state management.

#### Implementation Summary

**🏗️ Technical Architecture**:
- **AWS Hibernation Engine**: Full hibernation lifecycle with intelligent fallback to regular stop
- **Multi-Modal Integration**: REST API, GUI controls, and preparation for CLI commands
- **Smart Status Detection**: Automatic hibernation support detection with clear user feedback
- **Educational UI**: Smart confirmation dialogs explaining hibernation benefits vs regular operations

**🎯 Key Features Implemented**:
- ✅ **pkg/aws/manager.go**: `HibernateInstance()`, `ResumeInstance()`, `GetInstanceHibernationStatus()`
- ✅ **pkg/daemon/instance_handlers.go**: REST API endpoints for all hibernation operations
- ✅ **pkg/api/client/**: Complete API client integration with hibernation methods
- ✅ **pkg/types/runtime.go**: `HibernationStatus` type for comprehensive status tracking
- ✅ **cmd/cws-gui/main.go**: Smart hibernation controls with educational confirmation dialogs

**💡 Smart Fallback System**:
```go
// Hibernation with intelligent fallback
_, err = m.ec2.StopInstances(context.TODO(), &ec2.StopInstancesInput{
    InstanceIds: []string{instanceID},
    Hibernate:   aws.Bool(true),  // Falls back to regular stop if unsupported
})
```

**🎨 User Experience**:
- **GUI Integration**: Hibernation buttons appear contextually based on instance state
- **Educational Dialogs**: Clear explanations of hibernation benefits (RAM preservation, faster resume)
- **Transparent Fallbacks**: Users informed when hibernation unavailable with automatic fallback
- **Status Awareness**: Real-time hibernation support and state detection

**📊 Cost Optimization Impact**:
- **RAM State Preservation**: Instant resume from hibernated state vs cold boot
- **Compute Billing Stops**: No EC2 charges while hibernated (EBS storage continues)
- **Researcher-Friendly**: Maintains work session state for continuation without setup

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