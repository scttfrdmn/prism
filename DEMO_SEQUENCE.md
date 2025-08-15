# CloudWorkstation v0.4.2 Demo Sequence

This demo showcases CloudWorkstation v0.4.2 from installation to actual usage, demonstrating the complete workflow that researchers experience from setup to connecting to their workstation.

## Phase 1: Installation (2 minutes)

### 1.1 Installation Options
```bash
# Option 1: Homebrew Tap (Recommended)
brew tap scttfrdmn/cloudworkstation
brew install cloudworkstation

# Option 2: GitHub Releases (Alternative)
# Download from https://github.com/scttfrdmn/cloudworkstation/releases

# Option 3: Source Build (Full Features including GUI)
# git clone && make build
```

### 1.2 Initial Setup
```bash
# Verify installation
cws --version
cwsd --version

# Configure AWS credentials (required for cloud operations)
aws configure

# Set development mode (optional - avoids keychain prompts)
export CLOUDWORKSTATION_DEV=true
```

**Demo Points:**
- Professional package management via Homebrew
- Multiple installation options for different needs
- Simple setup process

## Phase 2: First Workstation Launch (3 minutes)

### 2.1 Daemon and Templates
```bash
# Start the daemon
cws daemon start

# Show available templates
cws templates list

# Show template details with cost estimation
cws templates info "Python Machine Learning (Simplified)"
```

### 2.2 Launch and Connect
```bash
# Launch a Python ML workstation
cws launch "Python Machine Learning (Simplified)" ml-research

# Show running instances
cws list

# Get connection details
cws info ml-research

# Connect to your workstation (KEY STEP)
cws connect ml-research

# Inside workstation: show environment
whoami
ls -la
conda list | head -10
jupyter --version

# Exit from workstation
exit
```

**Demo Points:**
- Zero configuration required
- Templates work out-of-the-box
- Direct SSH connection to workstation
- Pre-configured research environment ready to use

## Phase 3: Template Inheritance Demo (2 minutes)

### 3.1 Stacked Templates
```bash
# Show template inheritance
cws templates info "Rocky Linux 9 + Conda Stack"

# Launch inherited template workstation
cws launch "Rocky Linux 9 + Conda Stack" data-analysis

# Compare with base template
cws templates info "Rocky Linux 9 Base"

# Connect to new workstation
cws connect data-analysis

# Inside workstation: show inherited environment
whoami  # shows rocky user (from base)
su - datascientist  # shows datascientist user (from stack)
conda --version  # shows conda (from stack)
exit
exit
```

**Demo Points:**
- Template stacking architecture
- Inheritance merging (users, packages, services)
- Composition over duplication
- Multiple users and environments in single workstation

## Phase 4: Multi-Modal Access (2 minutes)

### 4.1 Different Interfaces
```bash
# CLI interface (already shown)
cws list

# Launch TUI for interactive management
cws tui
# Navigate: 1=Dashboard, 2=Instances, 3=Templates, 4=Storage
# Show real-time updates and keyboard navigation
# Exit TUI

# Launch GUI (if available from source build)
cws-gui &
# Show system tray integration, tabbed interface, visual management

# API access
curl -s http://localhost:8947/api/v1/instances | jq '.[] | .name'
```

**Demo Points:**
- Same functionality across interfaces
- Real-time synchronization
- Professional user experience
- RESTful API for integration

## Phase 5: Cost Optimization Features (2 minutes)

### 5.1 Manual Hibernation
```bash
# Show hibernation capabilities
cws hibernation-status ml-research

# Hibernate instance to save costs (preserves state)
cws hibernate ml-research

# Show state preservation
cws list

# Resume when needed
cws resume ml-research

# Reconnect after resume
cws connect ml-research
# Environment is preserved exactly as left
exit
```

### 5.2 Automated Hibernation Policies
```bash
# Show available hibernation policies
cws idle profile list

# Apply cost-optimized policy
cws idle instance ml-research --profile cost-optimized

# Show policy configuration
cws idle profile show cost-optimized

# View hibernation history/audit trail
cws idle history
```

**Demo Points:**
- Manual control for immediate savings
- Session preservation through hibernation (work state maintained)
- Automated policies for hands-off optimization
- Cost transparency and audit trail

## Phase 6: Enterprise Features (3 minutes)

### 6.1 Project-Based Organization
```bash
# Create a research project
cws project create "machine-learning-research" \
  --description "Deep learning model development project" \
  --budget-limit 500.00

# Show project details
cws project show machine-learning-research

# Associate instances with project
cws project assign machine-learning-research ml-research
cws project assign machine-learning-research data-analysis
```

### 6.2 Budget Management & Collaboration
```bash
# Set up budget tracking
cws project budget machine-learning-research set \
  --monthly-limit 500.00 \
  --alert-threshold 0.8

# Add team members (simulated)
cws project member add machine-learning-research \
  researcher@university.edu --role member

# Show real-time cost tracking
cws project cost machine-learning-research --breakdown

# Show project team
cws project members machine-learning-research
```

**Demo Points:**
- Grant-funded budget management
- Real-time cost tracking and alerts
- Role-based access control
- Project-based resource organization

## Phase 7: Storage & Advanced Features (2 minutes)

### 7.1 Storage Management
```bash
# Show storage options
cws storage list

# Create shared storage
cws storage create shared-data --size 100GB --type efs

# Attach to instances
cws storage attach shared-data ml-research /mnt/shared

# Connect and verify storage
cws connect ml-research
df -h | grep /mnt/shared  # Show mounted storage
echo "test data" > /mnt/shared/test.txt
exit
```

### 7.2 System Health and Diagnostics
```bash
# Health check
cws doctor

# Show system status
cws daemon status --detailed

# Profile management
cws profile current
cws profile list
```

**Demo Points:**
- Persistent shared storage between workstations
- System health monitoring
- Profile-based AWS credential management

## Phase 8: Cleanup and Next Steps (1 minute)

### 8.1 Resource Management
```bash
# Show final project cost summary
cws project cost machine-learning-research --savings

# Hibernate instances (preserves state for next session)
cws hibernate ml-research
cws hibernate data-analysis

# Final status check
cws list

# Clean shutdown
cws daemon stop
```

**Demo Points:**
- Cost savings through hibernation
- State preservation for future work sessions
- Clean resource management

## Demo Summary: Complete Workflow (15 minutes total)

### Complete User Journey:
1. **Installation** → Professional Homebrew tap installation
2. **Launch** → Zero-config template selection and workstation creation
3. **Connect** → Direct SSH access to pre-configured research environment
4. **Work** → Full research environment with conda, jupyter, tools ready to use
5. **Collaborate** → Template inheritance, shared storage, project organization
6. **Optimize** → Hibernation for cost savings while preserving work state
7. **Scale** → Enterprise features for team collaboration and budget management
8. **Integrate** → Multi-modal access and REST API for workflow integration

### Key Technical Demonstrations:
- **Template Inheritance**: Rocky Linux 9 Base → Rocky Linux 9 + Conda Stack
- **Connection Workflow**: `cws launch` → `cws connect` → research environment ready
- **State Preservation**: Hibernation maintains exact work environment for resume
- **Multi-Modal Access**: Same functionality across CLI, TUI, GUI, and API
- **Enterprise Features**: Project budgets, team collaboration, cost tracking

### Business Value Demonstrated:
- **Setup Time**: From hours to seconds for research environments
- **Cost Optimization**: Hibernation policies with session state preservation
- **Scalability**: Individual researchers to institutional deployments
- **Compliance**: Project budgets, audit trails, role-based access control

## Audience-Specific Variations:

### For Researchers (Focus: Phases 1-3, 5):
- Installation and first connection experience
- Template inheritance for custom environments  
- Cost optimization through hibernation

### For IT/System Administrators (Focus: Phases 4, 6-7):
- Multi-modal access and API integration
- Enterprise project management and user roles
- Storage management and system diagnostics

### For Budget/Finance Teams (Focus: Phases 5-6):
- Cost optimization features and hibernation savings
- Project budget management and real-time tracking
- Cost analytics and audit trails

### For Technical Decision Makers (Full Sequence):
- Complete workflow demonstrating scalability
- Enterprise integration capabilities
- ROI through setup time reduction and cost optimization

## Demo Environment Requirements:
- **System**: macOS/Linux with Homebrew installed
- **CloudWorkstation**: v0.4.2 binaries available
- **AWS**: Credentials configured (`aws configure`) for actual cloud operations
- **Development Mode**: `export CLOUDWORKSTATION_DEV=true` for smooth keychain experience
- **Network**: Internet access for AWS API calls and package downloads

## Recovery Commands (if demo issues arise):
```bash
# Reset daemon
cws daemon stop && sleep 2 && cws daemon start

# Clear stuck operations  
cws project delete machine-learning-research --force

# Fresh state
rm ~/.cloudworkstation/state.json && cws daemon restart

# Emergency cleanup
cws list  # identify running instances
cws hibernate <instance-name>  # hibernate all instances
```

## Quick Demo Checklist:
- [ ] Installation via Homebrew tap
- [ ] Template selection and launch
- [ ] **Connect to workstation** (KEY STEP)
- [ ] Template inheritance demonstration
- [ ] Hibernation with state preservation
- [ ] Project and budget management
- [ ] Multi-modal access (CLI, TUI, API)
- [ ] Storage attachment and verification
- [ ] Clean resource management