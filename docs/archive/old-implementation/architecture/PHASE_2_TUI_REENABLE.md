# Phase 2 TUI Re-enablement Plan

## Overview

The TUI (Terminal User Interface) has a complete implementation but is currently disabled due to API client integration issues. This document outlines the plan to re-enable the TUI and achieve full CLI/TUI/GUI parity.

## Current Status

### ✅ Complete Implementation Exists
- Full BubbleTea-based TUI in `internal/tui/`
- Page-based navigation (Dashboard, Instances, Templates, Storage, Settings, Profiles)
- Professional components (tables, tabs, spinners, status bars)
- Real-time data refresh system (30-second intervals)
- Dedicated TUI API client architecture

### ⚠️ Known Issues
- API client integration disabled due to modernization lag
- Mock client dependencies need removal
- Three pages have placeholder implementations
- Needs alignment with Phase 2 feature enhancements

## Implementation Plan

### Task 1: API Client Modernization (30 minutes)
**Objective**: Update TUI API client to use modernized daemon integration

**Files to Update**:
- `internal/tui/app.go` - Remove mock dependencies, enable real API client
- `internal/tui/api/client.go` - Update to use `api.NewClientWithOptions()` pattern
- `internal/tui/models/*.go` - Fix API response handling

**Implementation Steps**:
```go
// Replace in internal/tui/app.go:
func NewApp() *App {
    apiClient := api.NewClientWithOptions("http://localhost:8947", client.Options{
        AWSProfile: currentProfile.AWSProfile,
        AWSRegion:  currentProfile.Region,
    })
    
    return &App{
        apiClient: api.NewTUIClient(apiClient),
        program:   nil,
    }
}
```

### Task 2: Missing Page Implementations (1 hour)
**Objective**: Complete placeholder pages to match GUI functionality

**Pages to Implement**:

#### Instances Page
- Dynamic instance loading from API
- Lifecycle management (start/stop/delete)
- Connection information display
- Real-time status updates

#### Storage Page  
- EFS volume management (create/delete/attach/detach)
- EBS volume management with T-shirt sizing
- Volume attachment to instances
- Storage cost tracking

#### Settings Page
- Daemon status monitoring
- Profile management integration
- Configuration options
- System diagnostics

### Task 3: CLI/TUI/GUI Parity Alignment (1-2 hours)
**Objective**: Ensure all three interfaces have identical functionality

**Parity Requirements**:

| Feature | CLI Command | GUI Section | TUI Page | Status |
|---------|-------------|-------------|----------|---------|
| Templates | `prism templates` | Templates tab | Templates page | ⚠️ Update needed |
| Launch Basic | `prism launch` | Launch dialog | Launch workflow | ⚠️ Update needed |
| Launch Advanced | `prism launch --volume` | Advanced launch | Advanced options | ❌ Missing |
| Instance List | `prism list` | Instances tab | Instances page | ⚠️ Update needed |
| Instance Control | `prism start/stop` | Instance dialogs | Instance actions | ❌ Missing |
| Storage EFS | `prism volumes` | Storage EFS tab | Storage page | ❌ Missing |
| Storage EBS | `prism ebs-volumes` | Storage EBS tab | Storage page | ❌ Missing |
| Profile Mgmt | `prism config` | Settings profiles | Settings page | ⚠️ Update needed |
| Daemon Control | `prism daemon` | Settings daemon | Settings page | ❌ Missing |

### Task 4: Advanced Features Integration
**Objective**: Bring TUI up to Phase 2 feature level

**New Features to Add**:
- Volume attachment during launch
- Networking configuration (VPC/subnet)
- Spot instance options
- Dry run validation
- Enhanced cost tracking
- Real-time daemon monitoring

## Implementation Details

### API Client Pattern
```go
// Modernized TUI API client initialization
type TUIApp struct {
    apiClient   *api.Client
    currentProfile *profile.Profile
}

func (app *TUIApp) initializeAPI() error {
    profile, err := profile.GetCurrentProfile()
    if err != nil {
        return err
    }
    
    app.apiClient = api.NewClientWithOptions("http://localhost:8947", client.Options{
        AWSProfile: profile.AWSProfile,
        AWSRegion:  profile.Region,
    })
    
    return nil
}
```

### Navigation Enhancement
```go
// Enhanced keyboard navigation for feature parity
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "l": // Launch instance
            return m.showLaunchDialog()
        case "v": // Volume management  
            m.currentPage = StoragePage
        case "d": // Daemon status
            return m.showDaemonStatus()
        }
    }
}
```

### Real-time Updates
```go
// Enhanced refresh system for live data
func (m DashboardModel) startRealTimeUpdates() tea.Cmd {
    return tea.Every(10*time.Second, func(t time.Time) tea.Msg {
        return RefreshMsg{
            Instances: true,
            Storage:   true,
            Daemon:    true,
        }
    })
}
```

## Success Criteria

### Phase 2 TUI Completion Requirements
1. **✅ All placeholder pages implemented**
2. **✅ API client fully integrated and functional**
3. **✅ Feature parity with CLI and GUI achieved**
4. **✅ Advanced launch options available**
5. **✅ Real-time monitoring functional**
6. **✅ Professional error handling and user feedback**

### User Experience Goals
- **Zero Learning Curve**: Users familiar with CLI/GUI can immediately use TUI
- **Performance**: Sub-second response times for all operations
- **Reliability**: Graceful handling of daemon disconnections
- **Accessibility**: Full keyboard navigation with clear visual feedback

## Timeline Estimate

**Total Implementation Time**: 2.5-3 hours

- **API Client Modernization**: 30 minutes
- **Missing Pages**: 1 hour  
- **Parity Alignment**: 1-1.5 hours
- **Testing & Polish**: 30 minutes

## Post-Implementation

### Testing Strategy
1. **Functional Testing**: Verify each TUI page matches CLI/GUI behavior
2. **Integration Testing**: Confirm API client works with all daemon endpoints
3. **User Experience Testing**: Ensure navigation and workflows are intuitive
4. **Performance Testing**: Validate refresh rates and responsiveness

### Documentation Updates
- Update `CLAUDE.md` to reflect TUI re-enablement
- Add TUI usage examples to user documentation
- Document keyboard shortcuts and navigation patterns

## Technical Notes

### Dependencies
- BubbleTea framework (already integrated)
- Lipgloss styling (already configured)  
- Prism API client (needs modernization)
- Profile system integration (already available)

### Architecture Benefits
- **Consistent State**: All interfaces share same daemon backend
- **Unified Configuration**: Profile system works across CLI/TUI/GUI
- **Real-time Sync**: Changes in one interface reflect in others
- **Progressive Disclosure**: TUI provides middle ground between CLI simplicity and GUI richness

This plan positions the TUI as a first-class interface alongside CLI and GUI, completing the Prism multi-modal access strategy established in Phase 2.