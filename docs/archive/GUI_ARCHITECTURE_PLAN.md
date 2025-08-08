# CloudWorkstation GUI Architecture Plan

## Strategic Vision
Transform CloudWorkstation from a CLI-only tool into a comprehensive research platform accessible to non-technical users while maintaining power-user capabilities through progressive disclosure.

## Current vs Target Architecture

### Current (Monolithic)
```
┌─────────────────────┐
│     main.go         │
│  ┌───────────────┐  │
│  │ CLI Commands  │  │
│  │ AWS SDK       │  │
│  │ State Mgmt    │  │
│  │ JSON Storage  │  │
│  └───────────────┘  │
└─────────────────────┘
```

### Target (Distributed)
```
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│   GUI Client        │    │   CLI Client        │    │   Web Client        │
│ ┌─────────────────┐ │    │ ┌─────────────────┐ │    │ ┌─────────────────┐ │
│ │ Menubar/Tray    │ │    │ │ Command Parser  │ │    │ │ Dashboard       │ │
│ │ Dashboard       │ │◄──►│ │ API Wrapper     │ │◄──►│ │ (Future)        │ │
│ │ Notifications   │ │    │ │ Script Support  │ │    │ │                 │ │
│ └─────────────────┘ │    │ └─────────────────┘ │    │ └─────────────────┘ │
└─────────────────────┘    └─────────────────────┘    └─────────────────────┘
           │                           │                           │
           └───────────────────────────┼───────────────────────────┘
                                       │
                                       ▼
                          ┌─────────────────────┐
                          │  Backend Daemon     │
                          │ ┌─────────────────┐ │
                          │ │ REST/gRPC API   │ │
                          │ │ AWS SDK Ops     │ │
                          │ │ State Manager   │ │
                          │ │ Cost Tracker    │ │
                          │ │ Idle Monitor    │ │
                          │ │ Notifications   │ │
                          │ └─────────────────┘ │
                          └─────────────────────┘
```

## Component Architecture

### 1. Backend Daemon (`cwsd`)
**Core Service Layer**
```go
type CloudWorkstationDaemon struct {
    APIServer      *http.Server
    AWSManager     *aws.Manager
    StateManager   *state.Manager
    CostTracker    *cost.Tracker
    IdleMonitor    *idle.Monitor
    NotificationManager *notify.Manager
}
```

**Responsibilities:**
- REST/gRPC API server
- AWS operations (launch, stop, delete instances)
- State management and persistence
- Background monitoring (costs, idle detection)
- System notifications
- WebSocket for real-time updates

**API Endpoints:**
```
GET    /api/v1/instances          # List instances
POST   /api/v1/instances          # Launch instance
GET    /api/v1/instances/{id}     # Instance details
DELETE /api/v1/instances/{id}     # Delete instance
POST   /api/v1/instances/{id}/stop   # Stop instance
POST   /api/v1/instances/{id}/start  # Start instance

GET    /api/v1/templates          # List templates
GET    /api/v1/templates/{name}   # Template details

GET    /api/v1/volumes            # List volumes (EFS/EBS)
POST   /api/v1/volumes            # Create volume
DELETE /api/v1/volumes/{id}       # Delete volume

GET    /api/v1/costs              # Cost tracking
GET    /api/v1/status             # Daemon status

WebSocket: /api/v1/events         # Real-time updates
```

### 2. CLI Client (`cws`)
**Thin API Wrapper**
```go
type CLIClient struct {
    APIClient  *api.Client
    Config     *config.Config
    OutputFormat string // json, table, yaml
}
```

**Maintains Current Interface:**
```bash
# All existing commands work identically
cws launch r-research my-analysis
cws list
cws connect my-instance
cws volume create research-data
cws storage create ml-data L io2

# Plus new daemon management
cws daemon start
cws daemon stop  
cws daemon status
cws daemon logs
```

### 3. GUI Client (`cws-gui`)
**Progressive Disclosure Interface**

## GUI Design Philosophy

### Level 1: Menubar/System Tray (Always Visible)
```
┌─────────────────────────────────────┐
│ 🔬 CloudWorkstation            💰$12.45 │
├─────────────────────────────────────┤
│ 🟢 r-analysis (running)             │
│ 🟡 ml-training (stopped)            │
│ 🔴 gpu-workstation (idle 45m)       │
├─────────────────────────────────────┤
│ ⚡ Quick Launch                     │
│   📊 R Research Environment         │
│   🐍 Python Data Science           │
│   🧠 ML Training                    │
├─────────────────────────────────────┤
│ 📋 Open Dashboard                   │
│ ⚙️  Preferences                     │
│ ❓ Help                             │
└─────────────────────────────────────┘
```

**Features:**
- Live cost display in menubar
- Instance status at a glance
- One-click launch for common templates
- Notifications (idle instances, budget alerts)
- Quick access to full dashboard

### Level 2: Dashboard (Main Window)
```
┌──────────────────────────────────────────────────────────────┐
│ CloudWorkstation Dashboard                          💰 $12.45/day │
├──────────────────────────────────────────────────────────────┤
│ 🚀 Quick Launch                                               │
│ ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌──────────┐   │
│ │📊 R Research│ │🐍 Python ML │ │🧠 GPU Train │ │+ Custom  │   │
│ │Launch in 2m │ │Launch in 1m │ │Launch in 5m │ │Template  │   │
│ └────────────┘ └────────────┘ └────────────┘ └──────────┘   │
├──────────────────────────────────────────────────────────────┤
│ 📋 Active Instances                                           │
│ ┌─────────────────────────────────────────────────────────┐  │
│ │ r-analysis    🟢 Running   m5.large    $2.30/day  [Connect]│  │
│ │ ml-training   🟡 Stopped   p3.xlarge   $0/day     [Start] │  │
│ │ gpu-work      🔴 Idle 45m  g4dn.xl     $13.70/day [Stop]  │  │
│ └─────────────────────────────────────────────────────────┘  │
├──────────────────────────────────────────────────────────────┤
│ 💰 Budget Status                     📊 Usage This Month      │
│ Project: Genomics Research            ├── Compute: $234       │
│ Budget: $500/month                    ├── Storage: $45        │
│ Used: $345 (69%) ✅                  └── Network: $12        │
│ Remaining: $155                                               │
└──────────────────────────────────────────────────────────────┘
```

### Level 3: Advanced Configuration (When Needed)
```
┌──────────────────────────────────────────────────────────────┐
│ Launch Research Environment                                   │
├──────────────────────────────────────────────────────────────┤
│ Template: [Neuroimaging Research     ▼]                      │
│           FSL + AFNI + ANTs + Neuroglancer                   │
│                                                              │
│ Instance Size: [Medium (4 vCPU, 16GB RAM)  ▼] $4.61/day     │
│               Recommended for neuroimaging                   │
│                                                              │
│ Storage (Optional):                                          │
│ ☐ EFS Volume:     [research-data         ▼] (shared)        │
│ ☐ EBS Volume:     [Large (2TB)          ▼] $195/month       │
│                                                              │
│ Advanced Options: [▼ Show]                                   │
│ ┌────────────────────────────────────────────────────────┐   │
│ │ Region:          [us-west-2           ▼]               │   │
│ │ Architecture:    [ARM64 (20% cheaper) ▼]               │   │
│ │ Package Manager: [Smart Default       ▼]               │   │
│ │ VPC:             [Default             ▼]               │   │
│ │ Security Group:  [research-sg         ▼]               │   │
│ └────────────────────────────────────────────────────────┘   │
│                                                              │
│ Estimated Cost: $4.61/day + $195/month storage              │
│                                                              │
│           [Cancel]                    [Launch Environment]   │
└──────────────────────────────────────────────────────────────┘
```

## Technology Stack Recommendations

### Option 1: Go + Fyne (Recommended)
**Pros:**
- Single language (Go) for entire stack
- True native performance
- Excellent system tray support
- Small binary size
- Cross-platform

**Cons:**
- Less flexible UI compared to web technologies

```go
// Example Fyne implementation
import "fyne.io/fyne/v2/app"
import "fyne.io/fyne/v2/driver/desktop"

type CloudWorkstationApp struct {
    app    fyne.App
    window fyne.Window
    systray desktop.App
    apiClient *api.Client
}
```

### Option 2: Go + Wails
**Pros:**
- Go backend with web frontend
- Rich UI capabilities
- Native packaging
- Good performance

**Cons:**
- More complex build process
- Larger binary size

### Option 3: Tauri + Rust
**Pros:**
- Smallest binary size
- Excellent performance
- Modern web UI
- Strong security model

**Cons:**
- Different language from backend
- Steeper learning curve

## Implementation Phases

### Phase 1: Architecture Split (4-6 weeks)
**Goals:** Split monolithic application without changing functionality

**Tasks:**
1. **Extract Backend Daemon**
   - Move AWS operations to daemon service
   - Implement REST API
   - Background state management
   - System service integration (systemd/launchd/Windows Service)

2. **Refactor CLI Client** 
   - Thin wrapper around API calls
   - Maintain all existing commands
   - Add daemon management commands
   - Backward compatibility mode (embedded daemon)

3. **Testing & Migration**
   - Comprehensive API testing
   - CLI compatibility testing
   - Migration path for existing users
   - Performance testing

### Phase 2: Basic GUI (4-6 weeks)
**Goals:** Simple menubar/system tray with core functionality

**Features:**
- System tray with instance status
- Quick launch for common templates
- Instance list with basic controls (start/stop/connect)
- Cost display
- Basic notifications

**User Experience:**
```bash
# Installation
brew install cloudworkstation-gui  # or equivalent
# Auto-starts daemon, adds to menubar

# Usage
Click menubar → "Launch Python Environment" → Running in 2 minutes
```

### Phase 3: Advanced GUI (6-8 weeks)
**Goals:** Full-featured dashboard with progressive disclosure

**Features:**
- Comprehensive dashboard
- Advanced template configuration
- Volume management
- Budget tracking and alerts
- Settings and preferences
- Help and documentation integration

### Phase 4: Polish & Ecosystem (4-6 weeks)
**Goals:** Production-ready release with ecosystem integrations

**Features:**
- Auto-updates
- Crash reporting
- Usage analytics (opt-in)
- Integration with IDEs (VS Code extension)
- Template marketplace
- Multi-user support

## User Experience Scenarios

### Scenario 1: Non-Technical Researcher
**Dr. Sarah (Biology Professor)**
```
1. Installs CloudWorkstation via university software center
2. Sees menubar icon, clicks "Launch R Environment"
3. Gets notification "R environment ready" in 3 minutes
4. Clicks "Connect" → RStudio opens in browser
5. Does research, gets notification "Instance idle 30 min"
6. Clicks "Yes, stop instance" → costs stop
```

### Scenario 2: Power User
**Mike (Computational Scientist)**
```
1. Uses CLI for scripting: cws launch neuroimaging-stack brain-analysis --storage XL
2. Uses GUI for monitoring: glances at menubar for costs, instance status
3. Gets budget alert in GUI: "Project 80% of monthly budget"
4. Uses GUI dashboard to review and optimize costs
5. Uses CLI for automation: cws template export my-custom-env
```

### Scenario 3: Mixed Team
**Research Lab with varied technical skills**
```
1. PI uses GUI for oversight: budget tracking, team instance monitoring
2. Grad students use GUI for common tasks: launch standard environments
3. Postdocs use CLI for advanced workflows: custom templates, automation
4. All see shared project costs and status in GUI dashboard
```

## Migration Strategy

### Backward Compatibility
- CLI maintains identical interface
- Existing scripts continue to work
- Gradual migration path
- Embedded daemon mode for simple setups

### Installation Options
```bash
# Traditional CLI-only (current users)
brew install cloudworkstation

# Full GUI installation (new users)  
brew install cloudworkstation-gui

# Enterprise deployment
cloudworkstation deploy --multi-user --ldap-auth
```

## Success Metrics

### Adoption Metrics
- GUI vs CLI usage ratio
- Time to first successful launch (goal: <5 minutes)
- Non-technical user retention
- Support ticket reduction

### User Experience Metrics
- Click-to-running-instance time (goal: <3 minutes)
- Cost surprise incidents (goal: <1% of launches)
- Idle instance detection effectiveness (goal: >95% catch rate)

This architecture maintains CloudWorkstation's core simplicity while dramatically expanding accessibility. The progressive disclosure ensures non-technical users aren't overwhelmed while power users retain full capabilities.

Ready to proceed with Phase 1 implementation?