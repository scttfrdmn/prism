# CloudWorkstation Terminal User Interface (TUI) Plan

## Overview

For CloudWorkstation 0.2.0, we plan to implement a full Terminal User Interface (TUI) using the BubbleTea framework. This will provide an interactive, user-friendly interface within the terminal, bridging the gap between our current CLI and the planned GUI.

## Motivation

1. **Enhanced User Experience**: Offer a more intuitive, interactive experience while maintaining terminal workflow
2. **Progressive Enhancement**: Step between CLI and full GUI, consistent with our "Progressive Disclosure" principle
3. **Speed & Efficiency**: Faster development cycle than GUI with immediate user value
4. **Remote Usability**: Fully functional over SSH connections (unlike GUI)

## Technical Stack

- **Framework**: [BubbleTea](https://github.com/charmbracelet/bubbletea) - Go TUI framework
- **Components**:
  - [Bubbles](https://github.com/charmbracelet/bubbles) - Common BubbleTea components
  - [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions
  - [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering

## Architecture

```
cmd/
├── cws/          # CLI client (existing)
├── cws-tui/      # New TUI client
└── cwsd/         # Backend daemon (existing)

internal/
├── cli/          # CLI logic (existing)
└── tui/          # New TUI logic
    ├── app.go    # Main TUI application
    ├── models/   # BubbleTea models
    │   ├── dashboard.go
    │   ├── instances.go
    │   ├── templates.go
    │   └── volumes.go
    ├── components/ # Reusable UI components
    │   ├── table.go
    │   ├── spinner.go
    │   ├── progress.go
    │   └── form.go
    └── styles/   # UI styling
        └── theme.go
```

## Feature Set

### Dashboard View
- System status overview
- Active instances summary
- Cost tracking
- Quick action buttons

### Instance Management
- Interactive instance list with filtering
- Visual instance lifecycle state tracking
- Detailed instance view with metrics
- Action menu for instance operations

### Template Browser
- Visual template catalog with descriptions
- Template details view
- Launch form with parameter configuration
- Size selection with cost comparison

### Storage Management
- Volume listing with usage statistics
- Interactive volume creation form
- Attachment operations UI
- Volume details view

### System Features
- Keyboard shortcuts with help overlay
- Status bar with system information
- Notification system for long-running operations
- Context-sensitive help

## User Experience Goals

1. **Consistency**: Maintain consistent navigation patterns throughout
2. **Visibility**: Always show current status and available actions
3. **Feedback**: Provide immediate feedback for actions
4. **Efficiency**: Optimize for keyboard navigation and shortcuts
5. **Discoverability**: Make features discoverable through visual cues

## Mockups

### Main Dashboard
```
┌─ CloudWorkstation ─────────────────────────────────────────┐
│                                                            │
│  Dashboard   Instances   Templates   Storage    Settings   │
│                                                            │
├────────────────────────────────────────────────────────────┤
│ System Status: ● RUNNING                                   │
│ Region: us-west-2                                          │
│                                                            │
├─ Active Instances ─────────┬─ Daily Cost ──────────────────┤
│                            │                               │
│ ● python-ml    running     │ Current: $12.45               │
│ ● r-analysis   running     │ Projected: $348.60/month      │
│ ○ bioinfo      stopped     │                               │
│                            │                               │
├─ Quick Launch ─────────────┴───────────────────────────────┤
│                                                            │
│  [Python ML]  [R Research]  [Ubuntu Desktop]  [Custom...]  │
│                                                            │
└────────────────────────────────────────────────────────────┘
 q:Quit  h:Help  1:Dashboard  2:Instances  3:Templates
```

### Instance View
```
┌─ Instances ──────────────────────────────────────────────────┐
│                                                              │
│  Filter: [_____________]      Sort: Name ▼      [+ Launch]   │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│  NAME         TEMPLATE        STATUS    IP           COST/DAY│
│ ▶python-ml    python-research RUNNING   34.215.3.41  $8.64   │
│  r-analysis   r-research      RUNNING   18.236.76.22 $3.81   │
│  bioinfo      bioinformatics  STOPPED   -            $0.00   │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│ python-ml                                                    │
│ Instance Type: g4dn.xlarge (GPU, 4 vCPU, 16 GB RAM)         │
│ Launch Time: 2023-07-09 15:42:21                            │
│ Public IP: 34.215.3.41                                       │
│                                                              │
│ Volumes:                                                     │
│  • /home/ubuntu (50GB)                                       │
│  • /data (EFS: shared-data)                                  │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│ [Connect] [Stop] [Resize] [Delete]                           │
└──────────────────────────────────────────────────────────────┘
 ↑/↓:Select  Enter:Action  q:Back  r:Refresh
```

## Development Plan

1. **Phase 1: Basic Framework (2 weeks)**
   - Set up BubbleTea application structure
   - Implement core navigation and layout
   - Create theme and styling system
   - Add API client integration

2. **Phase 2: Main Views (2 weeks)**
   - Dashboard with system summary
   - Instance listing and management
   - Template browser and selection
   - Storage management interfaces

3. **Phase 3: Interactive Features (2 weeks)**
   - Form components for data entry
   - Keyboard shortcuts and help system
   - Long-running operation handling
   - Notifications and status updates

4. **Phase 4: Testing & Polish (1 week)**
   - Component tests with teatest
   - Cross-platform testing
   - Performance optimization
   - Documentation and user guides

## Testing Strategy

- **Unit Tests**: Test individual TUI components
- **Component Tests**: Test model updates and view rendering
- **Integration Tests**: Test API client integration
- **User Testing**: Gather feedback on workflows and usability

## Success Criteria

1. Complete all key workflows currently in CLI version
2. Intuitive navigation with minimal learning curve
3. Performance comparable to CLI for common operations
4. High user satisfaction in initial testing
5. Comprehensive test coverage for all components

## Future Extensions

- **Plugin System**: Allow custom views and commands
- **Theme Customization**: User-defined color schemes
- **Remote Management**: Control remote instances
- **Advanced Visualization**: Charts and graphs for cost/usage
- **Session Management**: Save and restore workspace state
