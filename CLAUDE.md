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

### Templates
Templates are hard-coded structs with pre-built AMI IDs:
- `r-research`: R + RStudio Server + common packages
- `python-research`: Python + Jupyter + data science stack
- `basic-ubuntu`: Plain Ubuntu for general use

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

### Phase 2 Improvements
- **Asynchronous Launch Mode**: Background instance launches with status polling
- **Dynamic AMI Discovery**: Automatic lookup of latest Ubuntu AMIs by region
- **Enhanced Template Management**: User-defined custom templates
- **Better Error Recovery**: Automatic retry and rollback mechanisms

### Phase 3 Scalability
- **Multi-Instance Management**: Launch and manage instance groups
- **Resource Scheduling**: Automatic start/stop based on schedules
- **Cost Optimization**: Spot instance support and cost alerts
- **Template Marketplace**: Shared community templates

### User Experience Improvements
- **Progress Indicators**: All long-running operations now show detailed progress
- **Dry-run Mode**: Validate configurations without launching (`--dry-run`)
- **Comprehensive Testing**: `cws test` validates AWS connectivity and permissions
- **Architecture Awareness**: Automatic ARM/x86 instance selection based on local architecture
- **T-Shirt Sizing**: Intuitive XS-XXL sizing with GPU options
- **Smart Template Defaults**: Templates specify optimal configurations automatically

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
