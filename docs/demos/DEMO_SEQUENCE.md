# Prism v0.4.5 Demo Sequence

This demo showcases Prism v0.4.5 from installation to actual usage, demonstrating the complete workflow that researchers experience from setup to connecting to their workstation.

## Phase 1: Installation (2 minutes)

### 1.1 Installation Options
```bash
# Option 1: Homebrew Tap (Recommended)
brew tap scttfrdmn/prism
brew install prism

# Option 2: GitHub Releases (Alternative)
# Download from https://github.com/scttfrdmn/prism/releases

# Option 3: Source Build (Full Features including GUI)
# git clone && make build
```

### 1.2 Initial Setup
```bash
# Verify installation
prism --version
cwsd --version

# Configure AWS credentials (required for cloud operations)
aws configure --profile aws  # Use your preferred profile name

# AWS Setup Note: This demo uses Prism's built-in profile system (recommended).
# For alternative methods and detailed setup, see AWS_SETUP_GUIDE.md

# Set development mode BEFORE starting daemon (avoids keychain prompts)
export CLOUDWORKSTATION_DEV=true

# Start daemon for profile management
prism daemon start

# Create Prism profile (RECOMMENDED METHOD)
prism profiles add personal my-research --aws-profile aws --region us-west-2

# Activate your profile
prism profiles switch aws

# Verify profile is active
prism profiles current
```

**Demo Points:**
- Professional package management via Homebrew
- Multiple installation options for different needs
- Prism profiles eliminate need for environment variables
- Simple setup process with persistent configuration

## Phase 2: First Workstation Launch (3 minutes)

### 2.1 Templates
```bash
# Show available templates (daemon already running from setup)
prism templates list

# Show template details with cost estimation
prism templates info "Python Machine Learning (Simplified)"
```

### 2.2 Launch and Connect
```bash
# Launch a Python ML workstation
prism launch "Python Machine Learning (Simplified)" ml-research

# Show running instances
prism list

# Get connection details
prism info ml-research

# Connect to your workstation (KEY STEP)
prism connect ml-research

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
prism templates info "Rocky Linux 9 + Conda Stack"

# Launch inherited template workstation
prism launch "Rocky Linux 9 + Conda Stack" data-analysis

# Compare with base template
prism templates info "Rocky Linux 9 Base"

# Connect to new workstation
prism connect data-analysis

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
prism list

# Launch TUI for interactive management
prism tui
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
prism hibernation-status ml-research

# Hibernate instance to save costs (preserves state)
prism hibernate ml-research

# Show state preservation
prism list

# Resume when needed
prism resume ml-research

# Reconnect after resume
prism connect ml-research
# Environment is preserved exactly as left
exit
```

### 5.2 Automated Hibernation Policies
```bash
# Show available hibernation policies
prism idle profile list

# Apply cost-optimized policy
prism idle instance ml-research --profile cost-optimized

# Show policy configuration
prism idle profile show cost-optimized

# View hibernation history/audit trail
prism idle history
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
prism project create "machine-learning-research" \
  --description "Deep learning model development project" \
  --budget-limit 500.00

# Show project details
prism project show machine-learning-research

# Associate instances with project
prism project assign machine-learning-research ml-research
prism project assign machine-learning-research data-analysis
```

### 6.2 Budget Management & Collaboration
```bash
# Set up budget tracking
prism project budget machine-learning-research set \
  --monthly-limit 500.00 \
  --alert-threshold 0.8

# Add team members (simulated)
prism project member add machine-learning-research \
  researcher@university.edu --role member

# Show real-time cost tracking
prism project cost machine-learning-research --breakdown

# Show project team
prism project members machine-learning-research
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
prism storage list

# Create shared storage
prism storage create shared-data --size 100GB --type efs

# Attach to instances
prism storage attach shared-data ml-research /mnt/shared

# Connect and verify storage
prism connect ml-research
df -h | grep /mnt/shared  # Show mounted storage
echo "test data" > /mnt/shared/test.txt
exit
```

### 7.2 System Health and Diagnostics
```bash
# Health check
prism doctor

# Show system status
prism daemon status --detailed

# Profile management
prism profiles current
prism profiles list

# Profile switching demonstration
prism profiles add personal demo-profile --aws-profile aws --region us-east-1
prism profiles switch demo-profile
prism profiles current
prism profiles switch aws  # Switch back to main profile
```

**Demo Points:**
- Persistent shared storage between workstations
- System health monitoring
- Prism profile management with easy switching
- No environment variables needed for AWS configuration

## Phase 8: Cleanup and Next Steps (1 minute)

### 8.1 Resource Management
```bash
# Show final project cost summary
prism project cost machine-learning-research --savings

# Hibernate instances (preserves state for next session)
prism hibernate ml-research
prism hibernate data-analysis

# Final status check
prism list

# Clean shutdown
prism daemon stop
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
- **Connection Workflow**: `prism launch` → `prism connect` → research environment ready
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
- **Prism**: v0.4.2 binaries available
- **AWS**: Credentials configured using Prism profiles (see [AWS_SETUP_GUIDE.md](AWS_SETUP_GUIDE.md))
  ```bash
  aws configure --profile aws  # Configure AWS CLI
  prism profiles add personal my-research --aws-profile aws --region us-west-2
  prism profiles switch aws      # Activate Prism profile
  ```
- **Development Mode**: `export CLOUDWORKSTATION_DEV=true` for smooth keychain experience
- **Network**: Internet access for AWS API calls and package downloads

## Recovery Commands (if demo issues arise):
```bash
# Reset daemon
prism daemon stop && sleep 2 && prism daemon start

# Clear stuck operations  
prism project delete machine-learning-research --force

# Fresh state
rm ~/.prism/state.json && prism daemon restart

# Emergency cleanup
prism list  # identify running instances
prism hibernate <instance-name>  # hibernate all instances
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
