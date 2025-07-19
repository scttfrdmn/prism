# CloudWorkstation 0.4.0 Roadmap

## Overview

This document outlines the development plan for CloudWorkstation 0.4.0, which introduces a Terminal User Interface (TUI) as its headline feature. The TUI implementation provides an intermediate option between the existing command-line interface and the planned GUI, offering visual management capabilities while remaining in the terminal environment.

## Core Features

CloudWorkstation 0.4.0 focuses on three primary goals:

1. **Terminal User Interface**: Interactive, text-based graphical interface using the BubbleTea framework
2. **Enhanced Visual Management**: Dashboard view, template browser, and interactive components
3. **Interface Mode Flexibility**: Seamless switching between CLI and TUI modes

## Development Phases

### Phase 1: Initial TUI Integration (2-3 weeks)

This phase focuses on integrating the existing TUI code from the `feature/tui-implementation` branch and updating it to work with the latest 0.3.0 features.

**Tasks:**
- Merge the `feature/tui-implementation` branch into a new `feature/v0-4-0` branch
- Update TUI components to work with v0.3.0 API changes
- Implement missing screens for all v0.3.0 features:
  - Idle detection management views
  - Multi-repository template browsing
  - Enhanced template details (reflecting new template format)
- Ensure all existing TUI views work with the current API

**Success Criteria:**
- TUI starts and displays dashboard correctly
- Template browsing works with updated template format
- No regressions in existing functionality

### Phase 2: TUI Enhancements (2-3 weeks)

Building on the core integration, this phase adds new TUI-specific features to improve user experience and functionality.

**Tasks:**
- Implement interactive instance management:
  - Start/stop/delete operations directly from dashboard
  - SSH/connection info with clickable links (using OSC 8 hyperlinks)
- Add storage management views:
  - EFS volume visualization 
  - EBS volume management with size visualization
  - Storage attachments to instances
- Improve navigation and keyboard shortcuts:
  - Global help overlay (press '?')
  - Context-sensitive actions based on current selection
  - Tab-based navigation between sections

**Success Criteria:**
- All operations available in CLI also work in TUI
- Storage visualization accurately displays volumes and attachments
- Help system clearly explains all available commands and shortcuts

### Phase 3: CLI-TUI Integration (1-2 weeks)

This phase focuses on creating a seamless experience between the CLI and TUI interfaces, allowing users to switch easily between them.

**Tasks:**
- Add mode switching capabilities:
  - `cws tui` command to launch TUI interface
  - Option to launch specific TUI views directly (e.g., `cws tui templates`)
  - Exit TUI back to CLI with state preserved
- Support command output in both modes:
  - Allow operations to return to CLI or remain in TUI
  - Add configuration preference for default interface
- Implement config system for TUI preferences

**Success Criteria:**
- Seamless transition between CLI and TUI modes
- State preservation when switching interfaces
- User preferences for interface mode properly saved

### Phase 4: Testing & Documentation (1-2 weeks)

This phase ensures the TUI implementation is thoroughly tested and well-documented.

**Tasks:**
- Create TUI-specific tests:
  - Component rendering tests
  - Navigation flow tests 
  - Data display verification
  - Cross-platform compatibility
- Update documentation:
  - Add TUI usage guide
  - Update README with TUI screenshots
  - Include keyboard shortcut reference
  - Document all TUI views and features

**Success Criteria:**
- Test coverage meets project standards (85%+ overall, 80%+ per file)
- Documentation covers all TUI features and usage patterns
- README updated with TUI information and screenshots

### Phase 5: Final Integration & Release (1 week)

This phase completes the integration and prepares for release.

**Tasks:**
- Final QA testing across platforms
- Performance optimization for large instance/template lists
- Create release notes highlighting TUI as major new feature
- Tag v0.4.0 release

**Success Criteria:**
- All tests pass on all supported platforms
- TUI performs well with large numbers of instances and templates
- Release notes clearly explain new features and improvements

## Design Principles

The TUI implementation will follow CloudWorkstation's established design principles:

### 🎯 **Default to Success**
- TUI should work without configuration
- All views should have reasonable default sizes and layouts
- Fallbacks for terminals with limited capabilities

### ⚡ **Optimize by Default**
- Efficient rendering for low-latency connections
- Smart caching to minimize API calls
- Responsive layouts that adapt to terminal size

### 🔍 **Transparent Fallbacks**
- Clear feedback when terminal doesn't support features
- Graceful degradation for limited color support
- Alternative navigation for terminals without mouse support

### 💡 **Helpful Guidance**
- Context-sensitive help throughout the interface
- Clear explanations of available actions
- Visual indicators for current state and operations

### 🚫 **Zero Surprises**
- Consistent behavior across views
- Preview changes before applying
- Clear confirmation for destructive actions

### 📈 **Progressive Disclosure**
- Simple views by default, detailed on demand
- Keyboard shortcuts for power users
- Advanced options accessible but not cluttering the interface

## Timeline

- **Weeks 1-3**: Phase 1 - Initial TUI Integration
- **Weeks 4-6**: Phase 2 - TUI Enhancements
- **Weeks 7-8**: Phase 3 - CLI-TUI Integration
- **Weeks 9-10**: Phase 4 - Testing & Documentation
- **Week 11**: Phase 5 - Final Integration & Release

## Future Considerations

The TUI implementation lays groundwork for future developments:

1. **GUI Integration**: The component architecture can inform GUI development
2. **Custom Views**: TUI layout engine could support user-customizable dashboards
3. **Plugin System**: Extend TUI with custom views for specific research domains
4. **Remote Management**: TUI could be used for remote instance management via SSH

## Resources

- BubbleTea Framework: https://github.com/charmbracelet/bubbletea
- LipGloss Styling: https://github.com/charmbracelet/lipgloss
- Bubble Components: https://github.com/charmbracelet/bubbles