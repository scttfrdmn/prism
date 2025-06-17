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

## Applying Design Principles in Development

### Code Quality Standards
- **Error Messages**: Must be actionable and educational, not technical jargon
- **Default Values**: Every configuration option must have a sensible default
- **Validation**: Fail fast with clear explanations and suggested fixes
- **Documentation**: Examples should show the simplest possible usage first

### Feature Development Priority
1. **Core User Journey**: Template + name → working instance
2. **Smart Defaults**: Minimize required configuration
3. **Transparent Feedback**: Users understand what's happening
4. **Graceful Degradation**: Fallbacks that maintain functionality
5. **Advanced Features**: Power user options that don't complicate basics

### Testing Philosophy
- **Happy Path First**: Basic workflows must always work
- **Regional Coverage**: Test fallbacks across all supported regions
- **Architecture Coverage**: Ensure ARM/x86 parity where possible
- **Cost Validation**: Verify pricing estimates are accurate
- **User Journey Testing**: End-to-end workflows from researcher perspective

## Current Phase: MVP (Phase 1)

**Goal**: Ultra-simple, working tool that provides immediate value

**Scope**: 
- Single Go file implementation
- Hard-coded templates (no YAML files)
- Basic commands: launch, list, connect, stop, delete
- AWS only
- Public instances (no VPN yet)
- JSON state file

**NOT in MVP**:
- Multi-user support
- VPN/private networking
- Budget management
- Configuration sync
- GUI
- Template validation
- Multiple cloud providers

## Architecture Decisions

### Simplicity First
- Single `main.go` file with all functionality
- Hard-coded AMI IDs for templates
- No databases - just JSON state file
- No complex abstractions - direct AWS SDK calls

### Templates (Stackable Architecture)

**Base Templates** (Foundation layers):
- `basic-ubuntu`: Plain Ubuntu 22.04 for general use
- `desktop-research`: Ubuntu Desktop + NICE DCV + research GUI applications
- `gpu-workstation`: NVIDIA drivers + CUDA + NICE DCV + ML/rendering tools

**Application Stacks** (Can be layered on base templates):
- `r-research`: R + RStudio Server + common packages
- `python-research`: Python + Jupyter + data science stack
- `scivis`: Scientific visualization + ParaView + VisIt + VTK
- `gis-research`: QGIS + GRASS + PostGIS + R spatial + Python geospatial
- `cuda-ml`: CUDA + cuDNN + PyTorch + TensorFlow + Jupyter + multi-GPU support
- `neuroimaging`: FSL + AFNI + ANTs + MRtrix + Neuroglancer
- `bioinformatics`: BWA + GATK + Samtools + R Bioconductor + Galaxy
- `cad-engineering`: CAD software + NVIDIA Omniverse + NICE DCV for 3D work

**Multi-Stack Architecture (User Choice)**:
```bash
# Simple templates (CloudWorkstation chooses best approach)
cws launch neuroimaging my-brain-analysis       # Smart defaults
cws launch cuda-ml gpu-training                 # Optimized for use case

# Power users can specify approach when needed
cws launch neuroimaging my-workstation --with spack
cws launch python-ml my-project --with conda
cws launch bioinformatics pipeline --with docker
cws launch desktop-research+custom:my-env workstation --with apptainer

# Mix approaches based on what works best
cws launch desktop-research my-workstation
# ↳ GUI apps: Native installation (best performance)
# ↳ Python environments: Conda (familiar to most researchers)  
# ↳ HPC software: Spack (when available)
# ↳ Web services: Docker (when appropriate)
```

**Behind-the-Scenes Intelligence**:
- **Smart Defaults**: CloudWorkstation picks the best tool for each component
- **Hidden Complexity**: Researchers see simple templates, not package managers
- **Flexible Override**: Power users can specify preferences when needed
- **Progressive Disclosure**: Start simple, access advanced features when ready
- **Workflow Awareness**: Different defaults for HPC vs desktop vs cloud-native workflows

### State Management
Simple JSON file at `~/.cloudworkstation/state.json`:
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
      "estimated_daily_cost": 2.40
    }
  }
}
```

## Development Principles

1. **Working over perfect**: Ship working MVP quickly
2. **Simple over complex**: Avoid abstractions until needed
3. **Direct over generic**: Direct AWS calls, no provider abstraction yet
4. **Manual over automated**: Manual AMI creation acceptable for MVP

## Future Phases (Post-MVP)

- Phase 2: Template library expansion
- Phase 3: Private networking with VPN
- Phase 4: Multi-user collaboration  
- Phase 5: Budget management
- Phase 6: Configuration sync

## Testing Strategy

MVP: Manual testing only
- Test each command works
- Test state persistence
- Test AWS integration
- Test on different platforms

## AMI Strategy

**MVP**: Manual AMI creation
1. Launch base Ubuntu instance
2. Install research software manually
3. Create AMI manually
4. Hard-code AMI ID in templates

**Future**: Automated AMI building with Packer

## Code Structure

```
main.go                 # Everything in one file
├── main()             # CLI argument parsing
├── handleLaunch()     # Launch instances
├── handleList()       # List instances
├── handleConnect()    # SSH/RDP connection
├── handleStop()       # Stop instances
├── handleDelete()     # Terminate instances
├── Template struct    # Template definition
├── Instance struct    # Instance state
├── State struct       # Application state
└── AWS helper funcs   # AWS SDK wrappers
```

## Success Criteria

- Launches R environment in < 1 minute
- Researcher can immediately use RStudio Server
- Reliable state management
- Clear cost information
- Works on macOS, Linux, Windows

## Common Issues to Watch

1. **AWS credentials**: Ensure clear error messages for auth issues
2. **Region consistency**: Use consistent AWS region
3. **State file corruption**: Handle JSON parsing errors gracefully
4. **Instance launch failures**: Clear error messages and cleanup
5. **Cross-platform paths**: Handle Windows vs Unix paths correctly

## Next Development Session Focus

If continuing development:
1. Add error handling and validation
2. Improve connection helpers (automatic SSH/browser opening)
3. Add more templates
4. Better cost tracking
5. Windows support testing

## Research User Feedback Integration

Key points to validate with users:
- Template selection covers their needs
- Launch time is acceptable  
- Connection process is smooth
- Cost visibility is sufficient
- Command interface is intuitive

## Development Commands

### Building and Testing
```bash
# Build the application
make build
# or: go build -o cws main.go

# Run tests
make test
# or: go test ./...

# Development build and run
./dev.sh build
./dev.sh run <args>

# Cross-compile for all platforms
make cross-compile

# Clean build artifacts
make clean
```

### Quick Development Workflow
```bash
# Build and test a basic command
./dev.sh build && ./cws templates

# Test launch (requires AWS credentials)
make test-launch
```

## Key Implementation Details

### Template System (main.go:48-112)
- Templates are hard-coded Go structs in the `templates` map
- Each template includes AMI ID, instance type, user data script, and cost estimates
- User data scripts are base64-encoded and executed on instance launch

### State Management (main.go:498-547)
- State file location: `~/.cloudworkstation/state.json`
- State is loaded/saved for each command execution
- Graceful handling of missing or corrupted state files

### AWS Integration
- Uses AWS SDK v2 with default credential chain
- Single EC2 client initialized in main()
- Direct EC2 API calls without abstraction layer
- Error handling focuses on clear user messages

### Command Structure
All commands follow pattern: `handleXXX()` functions called from main switch statement
- `handleLaunch()`: Creates EC2 instance with template configuration
- `handleList()`: Shows instances with live AWS state lookup
- `handleConnect()`: Provides connection info (SSH/web URLs)
- `handleStop()/handleStart()`: EC2 instance lifecycle management
- `handleDelete()`: Terminates instance with confirmation prompt

### Dependencies
- Go 1.24.4+ required
- AWS SDK v2 for EC2 operations
- No external CLI dependencies

### Configuration Management
The application supports persistent configuration via `~/.cloudworkstation/state.json`:

```bash
# Set AWS profile and region
cws config profile my-profile
cws config region us-west-2

# View current configuration
cws config show

# Architecture detection
cws arch
```

**Configuration Priority** (highest to lowest):
1. Environment variables (`AWS_PROFILE`, `AWS_REGION`, `AWS_DEFAULT_REGION`)
2. Application configuration (`cws config profile/region`)
3. AWS profile defaults
4. AWS SDK defaults

**Region Requirement**: A region must be configured either through:
- `cws config region <region>`
- Environment variables (`AWS_REGION` or `AWS_DEFAULT_REGION`)
- AWS profile default region

## Roadmap & Future Enhancements

### Phase 2: Enhanced Storage & Environments (In Progress)
- **✅ EFS Volume Management**: Complete lifecycle with automatic mounting
- **✅ EBS Secondary Volumes**: T-shirt sizing (XS-XL) with gp3/io2 support
- **✅ Volume Integration**: Launch instances with attached storage
- **🚧 Multi-Stack Templates**: Spack-based scientific software environments
- **🚧 Desktop Environments**: NICE DCV for GUI applications
- **🚧 Idle Detection**: Smart cost controls with desktop activity monitoring

### Phase 3: Advanced Research Features
- **Multi-Package Manager Support**: Spack + Conda + Docker with smart defaults
- **Granular Budget Tracking**: Instance-level cost monitoring with project budgets
- **Hibernation Support**: Cost-optimized pause/resume with EBS preservation  
- **Snapshot Management**: EFS/EBS snapshots for reproducible research
- **Specialized Templates**: Scientific visualization, GIS, CUDA ML, neuroimaging
- **Local SSD Support**: i3/i4i instances for ultra-high performance workloads

### Phase 4: Collaboration & Scale
- **Multi-User Projects**: Shared instances and collaborative workspaces
- **Template Marketplace**: Community-contributed research environments
- **Resource Scheduling**: Automatic start/stop based on schedules and budgets
- **OpenZFS/FSx Integration**: Advanced storage for specialized workloads
- **Multi-Cloud Support**: AWS + Azure + GCP for global research collaboration

### User Experience Improvements (Implemented)
- **✅ Progress Indicators**: All long-running operations show detailed progress
- **✅ Dry-run Mode**: Validate configurations without launching (`--dry-run`)
- **✅ Comprehensive Testing**: `cws test` validates AWS connectivity and permissions
- **✅ Architecture Awareness**: Automatic ARM/x86 instance selection based on local architecture
- **✅ Smart Template Defaults**: Templates specify optimal configurations automatically
- **✅ Volume Management**: Complete EFS/EBS lifecycle with budget tracking

## Design Principle Applications

### How Principles Guide Feature Decisions

**Example: T-Shirt Sizing Implementation**
- ✅ **Default to Success**: Every template specifies a working default size
- ✅ **Optimize by Default**: ML templates default to GPU sizes, R templates to memory-optimized
- ✅ **Transparent Fallbacks**: "GPU-S not available in region, using GPU-XS instead"  
- ✅ **Helpful Warnings**: "Size S has no GPU - consider GPU-S for ML workloads"
- ✅ **Zero Surprises**: Dry-run shows exact instance type and cost before launch
- ✅ **Progressive Disclosure**: `cws launch template name` (simple) → `--size L` (intermediate) → `--instance-type c5.large` (advanced)

**Example: Regional Availability**
- ✅ **Default to Success**: Templates work in all supported regions via fallbacks
- ✅ **Transparent Fallbacks**: Clear explanation when ARM GPU falls back to x86
- ✅ **Helpful Warnings**: Suggest better regions for GPU-intensive workloads
- ✅ **Zero Surprises**: `cws validate template` shows what will actually launch

### Anti-Patterns to Avoid
- ❌ **Complex Configuration**: Requiring multiple flags for basic usage
- ❌ **Silent Failures**: Falling back without explanation
- ❌ **Technical Jargon**: Error messages mentioning instance families
- ❌ **Surprising Costs**: Launching expensive instances without clear warning
- ❌ **Feature Creep**: Adding advanced options to basic commands

## Recent Major Enhancements

### Storage Revolution (Phase 2)
CloudWorkstation now provides enterprise-grade storage management that rivals dedicated cloud platforms:

**EFS Integration:**
- Complete lifecycle management (create, attach, detach, delete)
- Automatic mounting with proper permissions
- Cross-instance data sharing
- Safe deletion with mount target cleanup

**EBS Secondary Volumes:**
- T-shirt sizing (XS=100GB to XL=4TB) with transparent pricing
- Smart performance configuration (gp3 vs io2)
- Multiple volumes per instance support
- Automatic formatting and mounting

**Launch-Time Integration:**
```bash
# Simple usage with powerful backend
cws launch r-research data-analysis --volume shared-data --storage L
# ↳ Launches with EFS volume + 2TB EBS volume, fully configured
```

### Multi-Stack Template Architecture
Designed comprehensive stackable template system supporting multiple package managers:

**Smart Defaults:**
- GUI applications: Native installation (performance)
- Python environments: Conda (familiarity)
- HPC software: Spack (optimization)
- Web services: Docker (isolation)

**Progressive Disclosure:**
```bash
# Simple (90% of users)
cws launch neuroimaging brain-analysis

# Power users (when needed)
cws launch neuroimaging brain-analysis --python-with spack --fsl-with native
```

**NICE DCV Integration:**
- Hardware-accelerated remote desktop
- Perfect for GUI-heavy research applications
- Superior to RDP/VNC for scientific visualization
- Automatic desktop idle detection for cost control

### Advanced Features Designed
**Budget & Cost Management:**
- Instance-level cost tracking with persistent storage awareness
- Multi-month project budgets vs traditional monthly AWS budgets
- Proactive cost controls with research-aware idle detection
- EBS/EFS costs continue when instances stopped (properly tracked)

**Research-Specific Templates:**
- Scientific visualization (ParaView + VisIt + VTK)
- GIS research (QGIS + GRASS + PostGIS)
- CUDA ML (optimized PyTorch + TensorFlow stack)
- Neuroimaging (FSL + AFNI + ANTs + Neuroglancer)
- All with desktop GUI support via NICE DCV

This phase establishes CloudWorkstation as a comprehensive research computing platform, not just a simple VM launcher.

## GUI Development Strategy

### Strategic Pivot to Multi-Modal Access
CloudWorkstation will evolve from CLI-only to support multiple interface modes while maintaining its core design principles:

**Target Interfaces:**
- **CLI**: Power users, automation, scripting (maintain current functionality)
- **GUI**: Non-technical researchers, visual management, always-on monitoring
- **Web** (future): Browser-based access, collaboration features

### Architecture Transformation Required
Current monolithic `main.go` must split into distributed architecture:

```
Current:                    Target:
┌─────────────┐            ┌─────────────┐  ┌─────────────┐
│   main.go   │    →       │ GUI Client  │  │ CLI Client  │
│ Everything  │            │(menubar/tray)│  │ (current)   │
└─────────────┘            └──────┬──────┘  └──────┬──────┘
                                  │                │
                                  └────────┬───────┘
                                          │
                                   ┌─────────────┐
                                   │ Backend     │
                                   │ Daemon      │
                                   │ (API server)│
                                   └─────────────┘
```

### Progressive Disclosure in GUI
Following CloudWorkstation's core principles:

**Level 1 - Menubar/System Tray:**
- Instance status at a glance
- One-click launch common templates
- Cost monitoring ($12.45/day visible)
- Notifications (idle instances, budget alerts)

**Level 2 - Dashboard Window:**
- Visual instance management
- Template selection with descriptions
- Budget tracking with project context
- Quick access to common operations

**Level 3 - Advanced Configuration:**
- Full CLI-equivalent options
- Custom template creation
- Advanced storage configuration
- Multi-project management

### Key Design Benefits
- **Accessibility**: Non-technical researchers can use CloudWorkstation
- **Always-On Monitoring**: Menubar shows costs/status continuously  
- **Proactive Notifications**: Idle detection, budget warnings
- **Progressive Disclosure**: Simple by default, advanced when needed
- **Unified Experience**: GUI and CLI share same backend/state

### Implementation Phases
1. **Phase 1**: Split architecture (daemon + API + CLI client)
2. **Phase 2**: Basic GUI (menubar + simple dashboard)
3. **Phase 3**: Advanced GUI (full feature parity)
4. **Phase 4**: Polish + ecosystem integration

This GUI strategy will dramatically expand CloudWorkstation's reach to researchers who prefer visual interfaces while maintaining the power and flexibility that technical users require.
