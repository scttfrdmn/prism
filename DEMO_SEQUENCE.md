# CloudWorkstation v0.4.2 Demo Sequence

This demo showcases the key features of CloudWorkstation v0.4.2, highlighting the enterprise research management platform capabilities while demonstrating the simplicity that researchers love.

## Pre-Demo Setup

### 1. Environment Preparation
```bash
# Ensure development mode is enabled (avoids keychain prompts)
export CLOUDWORKSTATION_DEV=true

# Verify binaries are available
which cws cwsd cws-gui

# Check versions
cws --version
cwsd --version
```

### 2. Clean State (Optional)
```bash
# Stop any running daemon
cws daemon stop || true

# Clean previous state (optional - for fresh demo)
rm -rf ~/.cloudworkstation/state.json
```

## Demo Sequence: "From Researcher to Enterprise"

### Phase 1: Individual Researcher Experience (3 minutes)

#### 1.1 Quick Start - Launch First Workstation
```bash
# Start the daemon
cws daemon start

# Show available templates
cws templates

# Launch a Python ML workstation (demonstrates default-to-success)
cws launch "Python Machine Learning (Simplified)" ml-research

# Show running instances
cws list

# Get connection details
cws info ml-research
```

**Demo Points:**
- Zero configuration required
- Templates work out-of-the-box
- Smart instance sizing and regional fallbacks
- Clear connection information

#### 1.2 Template Inheritance Demo
```bash
# Show template details and inheritance
cws templates describe "Rocky Linux 9 + Conda Stack"

# Launch inherited template workstation
cws launch "Rocky Linux 9 + Conda Stack" data-analysis

# Compare with base template
cws templates describe "Rocky Linux 9 Base"
```

**Demo Points:**
- Template stacking architecture
- Inheritance merging (users, packages, services)
- Composition over duplication

#### 1.3 Multi-Modal Access
```bash
# CLI interface (already shown)
cws list

# Launch TUI for interactive management
cws tui
# Navigate: 1=Dashboard, 2=Instances, 3=Templates, 4=Storage
# Show real-time updates and keyboard navigation
# Exit TUI

# Launch GUI (if available)
cws-gui &
# Show system tray integration, tabbed interface, visual management
```

**Demo Points:**
- Same functionality across interfaces
- Real-time synchronization
- Professional user experience

### Phase 2: Cost Optimization Features (2 minutes)

#### 2.1 Manual Hibernation
```bash
# Show hibernation capabilities
cws hibernation-status ml-research

# Hibernate instance to save costs
cws hibernate ml-research

# Show state preservation
cws list

# Resume when needed
cws resume ml-research
```

#### 2.2 Automated Hibernation Policies
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
- Automated policies for hands-off optimization
- Session preservation through hibernation
- Cost transparency and audit trail

### Phase 3: Enterprise Features (4 minutes)

#### 3.1 Project-Based Organization
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

#### 3.2 Budget Management
```bash
# Set up budget tracking
cws project budget machine-learning-research set \
  --monthly-limit 500.00 \
  --alert-threshold 0.8

# Show real-time cost tracking
cws project cost machine-learning-research

# Show cost analytics
cws project cost machine-learning-research --breakdown
```

#### 3.3 Multi-User Collaboration
```bash
# Add team members (simulated)
cws project member add machine-learning-research \
  researcher@university.edu --role member

cws project member add machine-learning-research \
  advisor@university.edu --role admin

# Show project team
cws project members machine-learning-research
```

#### 3.4 Enterprise API Access
```bash
# Start daemon API (already running)
cws daemon status

# Show API capabilities (curl examples)
curl -s http://localhost:8947/api/v1/projects | jq .
curl -s http://localhost:8947/api/v1/instances | jq .
```

**Demo Points:**
- Grant-funded budget management
- Real-time cost tracking and alerts
- Role-based access control
- RESTful API for integration

### Phase 4: Advanced Features (2 minutes)

#### 4.1 Storage Management
```bash
# Show storage options
cws storage list

# Create shared storage
cws storage create shared-data --size 100GB --type efs

# Attach to instances
cws storage attach shared-data ml-research /mnt/shared
```

#### 4.2 Profile Management
```bash
# Show current profile
cws profile current

# List available profiles
cws profile list

# Show profile details
cws profile show research-profile
```

#### 4.3 System Health and Diagnostics
```bash
# Show system status
cws daemon status --detailed

# Health check
cws doctor

# Show comprehensive diagnostics
cws doctor --verbose
```

### Phase 5: Cleanup and Next Steps (1 minute)

#### 5.1 Resource Management
```bash
# Stop instances (hibernation preserves state)
cws hibernate ml-research
cws hibernate data-analysis

# Show cost savings
cws project cost machine-learning-research --savings

# Clean shutdown
cws daemon stop
```

#### 5.2 Installation Options
```bash
# Show installation methods
echo "Installation Options:"
echo "1. Homebrew Tap:"
echo "   brew tap scttfrdmn/cloudworkstation"
echo "   brew install cloudworkstation"
echo "2. GitHub Releases: Direct binary download"
echo "3. Source Build: Full GUI functionality"
```

## Demo Script Summary (12 minutes total)

### Key Messages:
1. **Simple for Individuals**: Zero-config templates, smart defaults, intuitive commands
2. **Powerful for Teams**: Project organization, budget management, collaboration
3. **Cost-Effective**: Hibernation, automated policies, transparent tracking
4. **Enterprise-Ready**: Role-based access, API integration, audit trails
5. **Multi-Modal**: CLI efficiency, TUI interactivity, GUI convenience

### Technical Highlights:
- Template inheritance system
- Cross-platform compatibility
- Real-time cost optimization
- Professional package management
- Comprehensive testing and reliability

### Business Value:
- Reduces research environment setup from hours to seconds
- Optimizes cloud costs through intelligent hibernation
- Scales from individual researchers to institutional deployments
- Maintains compliance through project budgets and audit trails

## Audience-Specific Variations:

### For Researchers:
Focus on Phase 1-2: Quick setup, templates, hibernation savings

### For IT/System Administrators:
Focus on Phase 3-4: Enterprise features, API integration, management

### For Budget/Finance Teams:
Focus on cost analytics, budget controls, hibernation savings

### For Technical Decision Makers:
Full sequence emphasizing scalability, integration, and ROI

## Demo Environment Requirements:
- macOS/Linux system with Homebrew
- CloudWorkstation v0.4.2 installed
- AWS credentials configured (for actual cloud operations)
- CLOUDWORKSTATION_DEV=true (for smooth demo experience)

## Recovery Commands (if demo issues arise):
```bash
# Reset daemon
cws daemon stop && sleep 2 && cws daemon start

# Clear stuck operations
cws project delete machine-learning-research --force

# Fresh state
rm ~/.cloudworkstation/state.json && cws daemon restart
```