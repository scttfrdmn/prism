# Prism GUI Navigation Guide

**Version**: v0.5.9+
**Last Updated**: October 27, 2025

## Overview

The Prism GUI features a simplified, intuitive navigation structure designed to help researchers focus on core tasks while keeping advanced features easily accessible.

## Navigation Structure

### Main Navigation (9 Items)

The main navigation sidebar provides access to core Prism features:

1. **🏠 Dashboard** - Overview and quick actions
2. **📋 Templates** - Browse and launch research environment templates
3. **💻 My Workspaces** - Manage your running cloud workspaces
4. **🖥️ Terminal** - Direct terminal access to workspaces
5. **🌐 Web Services** - Access web-based services (Jupyter, RStudio, etc.)
6. **---** *(divider)*
7. **💾 Storage** - Manage EFS and EBS volumes
8. **📊 Projects** - Project-based collaboration and budgets
9. **---** *(divider)*
10. **⚙️ Settings** - Configuration and advanced features

### Settings Internal Navigation

The Settings section uses an internal side navigation to organize configuration and advanced features:

#### General Settings
- System status and health
- Daemon configuration
- Auto-refresh intervals
- Default workspace sizes
- AWS profile and region information
- Feature toggles
- Debug and troubleshooting tools

#### Profiles
- AWS profile management
- Region configuration
- Credential validation
- Profile switching

#### Users
- Research user management
- SSH key management
- User provisioning
- Multi-user collaboration setup

#### Advanced *(Expandable Section)*
- **AMI Management** - Custom AMI creation and optimization
- **Rightsizing** - Instance sizing recommendations
- **Policy Framework** - Institutional governance and access control
- **Template Marketplace** - Community template discovery and sharing
- **Idle Detection** - Automated hibernation policies and cost optimization
- **Logs Viewer** - System logs and diagnostics

## Navigation Changes in v0.5.9

### What Changed

**Renamed**:
- "Research Templates" → "Templates" (simplified for clarity)

**Removed from Main Navigation** (moved to Settings > Advanced):
- Users
- AMI Management
- Rightsizing
- Policy Framework
- Template Marketplace
- Idle Detection
- Logs Viewer

**Result**:
- Main navigation reduced from 15 items → 9 items (40% reduction)
- Advanced features remain fully accessible via Settings
- Clearer focus on core research workflow

### Why These Changes

**Problem**: The previous navigation had 15 flat items, causing cognitive overload for new users trying to perform basic tasks like launching their first workspace.

**Solution**: Progressive disclosure - show the most common features upfront, keep advanced features easily accessible but not prominent.

**Benefits**:
- ⚡ Faster time to first workspace launch
- 🎯 Reduced cognitive load for beginners
- 💡 Advanced features still discoverable by experienced users
- 📈 Clearer learning path for new researchers

## Using the Settings Section

### Accessing General Settings

1. Click **Settings** in the main navigation
2. The General section loads by default
3. Configure system preferences, view status, and manage features

### Accessing Advanced Features

1. Click **Settings** in the main navigation
2. Click **Advanced** in the Settings side navigation
3. Expand to see 6 advanced features:
   - AMI Management
   - Rightsizing
   - Policy Framework
   - Template Marketplace
   - Idle Detection
   - Logs Viewer
4. Click any feature to access its full interface

### Quick Tips

- **Keyboard users**: Navigate Settings using Tab and Arrow keys
- **Mouse users**: Click any section in the side nav to switch views
- **Looking for a feature?**: Check Settings > Advanced first
- **Need help?**: Settings > General has troubleshooting links

## Common Workflows

### First-Time User Journey

1. **Dashboard** - See overview and system status
2. **Templates** - Browse available research environments
3. **Templates** - Click "Launch" on desired template
4. **My Workspaces** - See your running workspace
5. **Terminal** or **Web Services** - Connect to your workspace

### Managing Cost Optimization

1. **Settings** → **Advanced** → **Idle Detection**
2. Configure hibernation policies
3. View automated cost savings
4. Monitor idle workspace detection

### Creating Custom Templates

1. **My Workspaces** - Launch and configure a workspace
2. **Settings** → **Advanced** → **AMI Management**
3. Create AMI from configured workspace
4. Share via **Settings** → **Advanced** → **Template Marketplace**

### Multi-User Research Projects

1. **Projects** - Create project and set budget
2. **Settings** → **Users** - Add research users
3. **Projects** - Invite collaborators
4. **Settings** → **Advanced** → **Policy Framework** - Set access controls

## Accessibility

The navigation system is fully accessible:

- **Keyboard Navigation**: Tab, Arrow keys, Enter
- **Screen Readers**: Proper ARIA labels and semantic HTML
- **Focus Indicators**: Clear visual feedback
- **High Contrast**: Settings side navigation uses clear borders
- **Responsive**: Works on all screen sizes

## Troubleshooting

**Can't find a feature?**
- Check Settings > Advanced section
- Use browser Find (Ctrl/Cmd+F) to search page
- Consult this guide's "What Changed" section

**Settings side navigation not responding?**
- Check browser console for JavaScript errors
- Refresh the page (Ctrl/Cmd+R)
- Clear browser cache and reload

**Need more help?**
- Settings > General > Troubleshooting section
- GitHub Issues: https://github.com/scttfrdmn/prism/issues
- Documentation: https://github.com/scttfrdmn/prism/tree/main/docs

## Future Enhancements

Planned improvements for Settings navigation:

- **Search**: Quick search within Settings
- **Favorites**: Pin frequently-used advanced features
- **Keyboard Shortcuts**: Direct access to sections (e.g., Ctrl+, for Settings)
- **Breadcrumbs**: Show current location within Settings

## Related Documentation

- [GUI Architecture](../architecture/GUI_ARCHITECTURE.md)
- [User Guide v0.5.x](USER_GUIDE_v0.5.x.md)
- [GUI UX Design Review](../architecture/GUI_UX_DESIGN_REVIEW.md)
- [Release Plan v0.5.9](../releases/RELEASE_PLAN_v0.5.9.md)

---

**Navigation Philosophy**: *"Simple by default, detailed when needed. Progressive disclosure ensures core features are prominent while advanced capabilities remain accessible."*
