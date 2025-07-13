# CloudWorkstation Terminal User Interface (TUI) Guide

## Overview

CloudWorkstation's Terminal User Interface (TUI) provides an intuitive, keyboard-driven interface for managing your cloud workstations. It offers all the functionality of the command-line interface in a visual format that's easy to navigate and use.

## Getting Started

To launch the TUI, run:

```bash
cws tui
```

This will open the dashboard view showing a summary of your CloudWorkstation resources.

## Navigation

The TUI is organized into multiple views, with a tab bar at the top for navigation:

- **Dashboard**: Overview of your workstations and resources
- **Instances**: Manage your running and stopped workstations
- **Templates**: Browse and launch available workstation templates
- **Storage**: Manage EFS volumes and EBS storage
- **Settings**: Configure CloudWorkstation preferences

### Basic Controls

- **Tab Navigation**: Use <kbd>←</kbd> and <kbd>→</kbd> arrow keys or <kbd>Tab</kbd>/<kbd>Shift+Tab</kbd> to move between tabs
- **List Navigation**: Use <kbd>↑</kbd> and <kbd>↓</kbd> arrow keys to navigate lists
- **Selection**: Press <kbd>Enter</kbd> to select an item or activate a control
- **Help**: Press <kbd>?</kbd> to show available keyboard shortcuts for the current view
- **Quit**: Press <kbd>Ctrl+C</kbd> or <kbd>q</kbd> to exit the TUI

### Search

In list views (Instances, Templates, Storage), you can search by pressing <kbd>/</kbd> to activate the search box. Type your search query and results will filter as you type. Press <kbd>Esc</kbd> to cancel search mode.

## Dashboard View

The Dashboard provides an overview of your CloudWorkstation resources:

- **Instance Summary**: Count of running and stopped instances
- **Cost Summary**: Daily and monthly estimated costs
- **Recent Activity**: Latest actions performed
- **Quick Actions**: Common operations accessible with keyboard shortcuts

### Dashboard Controls

- <kbd>r</kbd>: Refresh dashboard data
- <kbd>→</kbd>: Navigate to Instances view

## Instances View

The Instances view shows all your workstations with their current status:

- **Name**: Instance name
- **Template**: The template used to create the instance
- **State**: Current state (running, stopped, etc.)
- **IP**: Public IP address (if available)
- **Cost/Day**: Estimated daily cost

### Instance Controls

- <kbd>↑</kbd>/<kbd>↓</kbd>: Navigate instance list
- <kbd>Enter</kbd>: Show instance details
- <kbd>s</kbd>: Start selected instance
- <kbd>p</kbd>: Stop selected instance
- <kbd>c</kbd>: Connect to selected instance (shows connection options)
- <kbd>d</kbd>: Delete selected instance (with confirmation)
- <kbd>r</kbd>: Refresh instance list
- <kbd>/</kbd>: Search instances

## Templates View

The Templates view allows you to browse and launch available workstation templates:

- **Name**: Template name
- **Description**: Brief description of the template
- **Architecture**: Supported architectures (x86_64, arm64)
- **Cost**: Estimated hourly/daily cost

### Template Controls

- <kbd>↑</kbd>/<kbd>↓</kbd>: Navigate template list
- <kbd>Enter</kbd>: View template details
- <kbd>l</kbd>: Launch selected template (opens launch dialog)
- <kbd>r</kbd>: Refresh template list
- <kbd>/</kbd>: Search templates

### Launch Dialog

When launching a template, you'll be prompted to configure:

1. **Instance Name**: Enter a name for your workstation
2. **Size**: Select from XS, S, M, L, XL (determines instance type)
3. **Advanced Options**: Toggle to show/hide advanced settings
   - Instance Type: Override automatic selection
   - Spot Instance: Use spot pricing for lower cost (with interruption risk)
   - Volumes: Attach EFS volumes
   - Storage: Add EBS storage volumes

## Storage View

The Storage view manages your persistent storage resources:

### EFS Volumes Tab

- **Name**: Volume name
- **Size**: Current size in GB
- **State**: Available, creating, etc.
- **Mount Target**: Network location for mounting

#### EFS Controls

- <kbd>↑</kbd>/<kbd>↓</kbd>: Navigate volume list
- <kbd>Enter</kbd>: View volume details
- <kbd>c</kbd>: Create new volume
- <kbd>d</kbd>: Delete selected volume (with confirmation)
- <kbd>r</kbd>: Refresh volume list

### EBS Storage Tab

- **Name**: Storage volume name
- **Size**: Size in GB
- **Type**: gp3, io2, etc.
- **State**: Available, in-use, etc.
- **Attached To**: Instance name (if attached)

#### EBS Controls

- <kbd>↑</kbd>/<kbd>↓</kbd>: Navigate storage list
- <kbd>Enter</kbd>: View storage details
- <kbd>c</kbd>: Create new storage volume
- <kbd>a</kbd>: Attach selected volume to an instance
- <kbd>d</kbd>: Detach selected volume
- <kbd>x</kbd>: Delete selected volume (with confirmation)
- <kbd>r</kbd>: Refresh storage list

## Settings View

The Settings view allows you to configure CloudWorkstation preferences:

- **AWS Profile**: Select AWS profile to use
- **AWS Region**: Select default region
- **Theme**: Toggle between light and dark mode
- **Registry**: Enable/disable AMI registry lookup

### Settings Controls

- <kbd>Tab</kbd>/<kbd>Shift+Tab</kbd>: Navigate between settings
- <kbd>Enter</kbd>: Select/edit a setting
- <kbd>↑</kbd>/<kbd>↓</kbd>: Change selected option
- <kbd>s</kbd>: Save changes

## Theme Switching

CloudWorkstation TUI supports both light and dark themes:

- **Dark Theme**: Default theme optimized for low-light environments
- **Light Theme**: High-contrast theme for bright environments

To toggle between themes:
1. Navigate to the Settings view
2. Select the Theme setting
3. Press <kbd>Enter</kbd> to toggle between Dark and Light
4. Press <kbd>s</kbd> to save your preference

## Notifications

The TUI displays notifications for important events:

- **Success**: Green notifications for completed operations
- **Error**: Red notifications for failed operations
- **Info**: Blue notifications for information messages

Notifications appear at the top of the screen and automatically dismiss after a few seconds.

## Advanced Features

### Keyboard Shortcuts Reference

Press <kbd>?</kbd> in any view to show a contextual help dialog with all available keyboard shortcuts.

### Search Functionality

- The search feature (<kbd>/</kbd>) supports partial matching and is case-insensitive.
- Search results update in real-time as you type.
- Press <kbd>Esc</kbd> to clear the search and show all items.

### Progress Indicators

Long-running operations (launch, delete, etc.) display progress indicators to keep you informed:

- Spinner animation during the operation
- Percentage complete (when available)
- Operation-specific status messages

## Troubleshooting

### Common Issues

1. **TUI doesn't display properly**:
   - Ensure your terminal supports TrueColor and Unicode
   - Try resizing your terminal window
   - Check if TERM environment variable is set correctly

2. **Operation fails with error**:
   - Check AWS credentials are configured correctly
   - Verify you have necessary permissions in AWS
   - Check daemon is running with `cws daemon status`

3. **TUI appears frozen**:
   - Some operations may take time to complete
   - Look for progress indicators at the bottom of the screen
   - Press <kbd>Ctrl+C</kbd> to exit if unresponsive

### Logs

The TUI logs errors and important events to:
```
~/.cloudworkstation/logs/tui.log
```

Examine this file for detailed error information when troubleshooting.

## Integration with CLI

The TUI works alongside the CLI, sharing the same state file and daemon process. Changes made in one interface will be reflected in the other.

To switch between interfaces:
- Exit TUI with <kbd>q</kbd>, then use CLI commands
- From CLI, run `cws tui` to launch the TUI again