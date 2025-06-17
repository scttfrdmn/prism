# Cloud Workstation Platform - Claude Development Context

## Project Overview

This is a command-line tool that allows academic researchers to launch pre-configured cloud workstations in seconds rather than spending hours setting up research environments.

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
