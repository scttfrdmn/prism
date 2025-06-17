# CloudWorkstation GUI Architecture

## Overview

The CloudWorkstation GUI is a modern, single-page application built with Go and Fyne that provides a clean, organized interface for managing cloud research environments. It follows contemporary design principles with no popup windows and a dashboard-centric approach.

## Design Philosophy

### Single-Page Application (SPA)
- **No popup windows** - All interactions happen within the main window
- **Inline notifications** - Feedback appears as dismissible cards at the top
- **Content switching** - Navigation changes the main content area
- **Consistent layout** - Sidebar navigation with main content area

### Modern Visual Design
- **Card-based layouts** for organized information presentation
- **Grid systems** for consistent spacing and alignment
- **Typography hierarchy** with proper bold headers and text styling
- **Icon integration** using Fyne's built-in theme system
- **Visual status indicators** with color-coded state icons

## Architecture Components

```
┌─────────────┬──────────────────────────────────┐
│   Sidebar   │           Main Content           │
│  (20% width)│          (80% width)             │
├─────────────┼──────────────────────────────────┤
│ App Info    │   ┌─ Notifications (inline) ─┐   │
│ - Logo      │   │ ✅ Success/Error alerts   │   │
│ - Version   │   │ ℹ️  Info messages         │   │
│ - Cost      │   │ ❌ Error notifications    │   │
│             │   └──────────────────────────┘   │
├─────────────┤                                  │
│ Navigation  │     📊 Dynamic Content Area      │
│ 🏠 Dashboard│     ┌─────────────────────────┐   │
│ 💻 Instances│     │  Dashboard / Instances  │   │
│ 📋 Templates│     │  Templates / Storage    │   │
│ 💾 Storage  │     │  Billing / Settings     │   │
│ 💰 Billing  │     └─────────────────────────┘   │
│ ⚙️ Settings │                                  │
├─────────────┤                                  │
│ Quick Actions│                                  │
│ - R Env     │                                  │
│ - Python ML │                                  │
│ - Ubuntu    │                                  │
├─────────────┤                                  │
│ Status      │                                  │
│ - Connection│                                  │
│ - Health    │                                  │
└─────────────┴──────────────────────────────────┘
```

## Navigation Sections

### 🏠 Dashboard (Primary)
**Purpose**: Overview and quick actions
**Features**:
- Overview cards (active instances, daily cost, totals)
- Quick launch form with template/name/size selection
- Recent instances list with management shortcuts
- Real-time cost and status updates

### 💻 Instances 
**Purpose**: Complete instance management
**Features**:
- Detailed instance cards with full information
- State-aware action buttons (Connect/Start/Stop/Delete)
- Visual status indicators with color coding
- Launch new instance shortcut

### 📋 Templates
**Purpose**: Template discovery and launching
**Features**:
- Visual template gallery with descriptions
- Pre-configured environment details
- One-click template launching
- Future: Custom template creation

### 💾 Storage
**Purpose**: Volume and storage management
**Features**:
- EFS volume management (future)
- EBS volume operations (future)
- Storage cost tracking (future)
- Attachment/detachment workflows (future)

### 💰 Billing
**Purpose**: Cost monitoring and control
**Features**:
- Current cost breakdown
- Daily/monthly estimates
- Running vs total instance costs
- Advanced billing features (future)

### ⚙️ Settings
**Purpose**: Application configuration
**Features**:
- Daemon connection settings
- Connection testing
- Application information
- About and help links

## Backend Integration

### API Client Architecture
```go
type CloudWorkstationGUI struct {
    apiClient api.CloudWorkstationAPI  // Interface to daemon
    // ... UI components
}

// Daemon connection
apiClient: api.NewClient("http://localhost:8080")
```

### Real-time Data Flow
```
User Action → GUI Handler → API Client → HTTP Request
     ↓
Daemon REST API → AWS SDK → Cloud Operation
     ↓
Response → GUI Update → Notification → Refresh
```

### Supported Operations
- ✅ **Instance Lifecycle**: Launch, start, stop, delete
- ✅ **Template Management**: List, select, quick launch
- ✅ **Connection Info**: SSH details and access
- ✅ **Status Monitoring**: Real-time state and cost updates
- ✅ **Health Checks**: Daemon connectivity and error handling

## User Experience Design

### Notification System
```go
// Inline notifications replace all popup dialogs
func (g *CloudWorkstationGUI) showNotification(type, title, message)
- Success: Green with checkmark icon
- Error: Red with error icon  
- Info: Blue with info icon
- Auto-dismiss after 5 seconds
- Manual dismiss with × button
```

### Loading States
```go
// Non-blocking operations with visual feedback
g.launchForm.launchBtn.SetText("Launching...")
g.launchForm.launchBtn.Disable()

// Background API calls with animations
go func() {
    response, err := g.apiClient.LaunchInstance(req)
    // Update UI on main thread
}()
```

### Form Validation
- Inline validation without disrupting workflow
- Clear error messages in notification area
- Required field highlighting
- Smart defaults for improved UX

## State Management

### Data Synchronization
```go
type CloudWorkstationGUI struct {
    // Data state
    instances     []types.Instance
    templates     map[string]types.Template
    totalCost     float64
    lastUpdate    time.Time
    
    // Background refresh every 30 seconds
    refreshTicker *time.Ticker
}
```

### Form State
```go
// Persistent form state across navigation
launchForm struct {
    templateSelect *widget.Select
    nameEntry     *widget.Entry
    sizeSelect    *widget.Select
    launchBtn     *widget.Button
}
```

## Visual Design System

### Color Coding
- 🟢 **Running**: Green circle - instance is active
- 🟡 **Stopped**: Yellow circle - instance is stopped
- 🟠 **Pending**: Orange circle - transitional states
- 🔴 **Terminated**: Red circle - instance destroyed
- ⚫ **Unknown**: Black circle - unknown state

### Typography
- **Bold headers** for section titles and primary information
- **Regular text** for descriptions and secondary information
- **Italic text** for placeholder and helper text
- **Monospace** for technical details (IDs, commands)

### Layout Principles
- **Card containers** for grouped information
- **Grid layouts** for consistent spacing
- **Separators** for visual hierarchy
- **Spacers** for flexible positioning

## Performance Considerations

### Efficient Updates
- **Selective rendering** - Only update changed content areas
- **Background operations** - Non-blocking API calls
- **Smart refresh** - Avoid unnecessary re-renders
- **Lazy loading** - Load content on demand

### Memory Management
- **Resource cleanup** - Proper disposal of timers and resources
- **Event handling** - Efficient callback management
- **State optimization** - Minimal data retention

## Future Enhancements

### Visual Improvements
- **Dark mode support** - Theme switching capability
- **Custom icons** - CloudWorkstation branded iconography  
- **Enhanced animations** - Smooth transitions and loading states
- **Responsive design** - Better window resizing behavior

### Functionality Expansion
- **Advanced filtering** - Search and filter instances/templates
- **Bulk operations** - Multi-select for batch actions
- **Activity timeline** - History of operations and changes
- **Usage analytics** - Charts and graphs for usage patterns

### Integration Features
- **Keyboard shortcuts** - Power user productivity features
- **Export capabilities** - Data export and reporting
- **Collaboration tools** - Share workstations and templates
- **Integration hooks** - External tool connections

## Development Guidelines

### Code Organization
```
cmd/cws-gui/main.go
├── Application setup and initialization
├── Navigation and layout management
├── View creation functions (Dashboard, Instances, etc.)
├── Event handlers for user interactions
├── API integration and data management
└── Utility functions and helpers
```

### Best Practices
- **Single responsibility** - Each function has a clear purpose
- **Consistent naming** - Follow Go and Fyne conventions
- **Error handling** - Graceful degradation with user feedback
- **Documentation** - Clear comments for complex logic

### Testing Strategy
- **Manual testing** - User workflow verification
- **Integration testing** - API connectivity validation
- **Visual testing** - Layout and design verification
- **Performance testing** - Responsiveness under load

This GUI architecture provides a solid foundation for a modern, user-friendly cloud workstation management interface that scales with user needs and maintains excellent usability throughout the application lifecycle.