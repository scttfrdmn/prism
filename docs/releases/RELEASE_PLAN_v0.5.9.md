# Prism v0.5.9 Release Plan: Navigation Restructure

**Release Date**: Target January 2026
**Focus**: Reduce cognitive load through progressive disclosure and clean navigation

## 🎯 Release Goals

### Primary Objective
Restructure navigation to reduce complexity from 14 flat items to 6 organized sections by:
1. Merging related functionality (Terminal/WebView → Workspaces)
2. Collapsing advanced features under Settings
3. Unifying storage interfaces (EFS + EBS)
4. Integrating budgets into Projects

### Success Metrics
- 🧭 Navigation complexity: 14 → 6 top-level items
- ⏱️ Time to find advanced features: <10 seconds
- 😃 User confusion: Further 30% reduction
- 📱 Advanced feature discoverability: >95%

---

## 📦 Features & Issues

### 1. Issue #14: Merge Terminal/WebView into Workspaces
**Priority**: P0 (Core navigation simplification)
**Effort**: Large (3-4 days)
**Impact**: Critical (Reduces navigation by 2 items)

**Current State**:
```
Navigation:
- Workspaces (list view)
- Terminal (SSH connection)
- WebView (Jupyter/RStudio access)
```

**Proposed State**:
```
Navigation:
- Workspaces (integrated view)
  └─ Each workspace card shows:
     - Status, cost, uptime
     - Quick actions: Connect (SSH/Web), Stop, Terminate
     - Expandable: Connection details, logs, metrics
```

**Implementation**:

#### Phase 1: Workspace Detail Component
```typescript
// cmd/prism-gui/frontend/src/components/WorkspaceDetail.tsx
interface WorkspaceDetailProps {
  workspace: Workspace;
  onConnect: (method: 'ssh' | 'web') => void;
  onAction: (action: 'stop' | 'start' | 'hibernate' | 'terminate') => void;
}

Sections:
- Header: Name, status badge, actions dropdown
- Connection Options:
  - SSH: Command to copy, or click to launch terminal
  - Web Services: Jupyter/RStudio/VSCode buttons
  - Port forwarding: List of exposed ports
- Details Accordion:
  - Configuration (template, size, region)
  - Cost (current session, lifetime, estimate)
  - Storage (attached EFS/EBS volumes)
  - Logs (recent activity, errors)
  - Metrics (CPU, memory, network - if available)
```

#### Phase 2: Integrated Terminal Component
```typescript
// cmd/prism-gui/frontend/src/components/IntegratedTerminal.tsx
Features:
- Embedded xterm.js terminal
- SSH connection via WebSocket (through daemon)
- Tab support for multiple workspaces
- Copy/paste support
- Search in terminal
- Download/upload file shortcuts
```

#### Phase 3: Web Service Integration
```typescript
// cmd/prism-gui/frontend/src/components/WebServicePanel.tsx
Features:
- Detect available services (Jupyter, RStudio, VSCode)
- Embedded iframe for web UIs (optional)
- External link option
- Connection status indicators
- Port forwarding setup
```

**API Requirements**:
- GET `/api/v1/workspaces/{id}/services` - List available web services
- POST `/api/v1/workspaces/{id}/terminal` - Initialize terminal session
- WebSocket `/ws/terminal/{session-id}` - Terminal I/O stream
- GET `/api/v1/workspaces/{id}/ports` - List port forwarding

**Migration**:
- Remove "Terminal" navigation item
- Remove "WebView" navigation item
- Update all links to point to unified Workspaces tab
- Add migration notice for users

**Testing**:
- [ ] Workspace list shows all workspaces correctly
- [ ] Connection options work (SSH, Jupyter, RStudio)
- [ ] Terminal embedded successfully
- [ ] Web services detect and connect
- [ ] Multiple workspace tabs work
- [ ] All personas can complete workflows

---

### 2. Issue #16: Collapse Advanced Features under Settings
**Priority**: P0 (Navigation simplification)
**Effort**: Medium (2-3 days)
**Impact**: Critical (Reduces navigation by 4-5 items)

**Current Navigation (14 items)**:
```
- Home
- Workspaces
- Templates
- Terminal
- WebView
- Storage (EFS)
- Volumes (EBS)
- AMI Management
- Projects
- Budgets
- Idle Detection
- Profiles
- Users
- Settings
```

**Proposed Navigation (6 items)**:
```
- Home
- Workspaces (merged Terminal/WebView)
- Templates
- Storage (unified EFS+EBS)
- Projects (integrated Budgets)
- Settings
  ├─ Profiles
  ├─ Users
  ├─ Advanced
  │  ├─ AMI Management
  │  ├─ Idle Detection  
  │  └─ System
  └─ About
```

**Implementation**:

#### Phase 1: Settings Navigation Structure
```typescript
// cmd/prism-gui/frontend/src/pages/Settings.tsx
interface SettingsPage {
  sections: SettingsSection[];
}

interface SettingsSection {
  id: string;
  label: string;
  icon: string;
  component: React.ComponentType;
  visible: (user: User) => boolean;  // Role-based visibility
}

Sections:
- General: App preferences, theme, language
- Profiles: AWS profile management
- Users: Research user management (admin only)
- Advanced:
  - AMI Management (admin only)
  - Idle Detection: Configure hibernation policies
  - System: Daemon settings, logs, diagnostics
- About: Version, docs links, support
```

#### Phase 2: Role-Based Visibility
```typescript
// Only show admin sections to admins
function isVisible(section: SettingsSection, user: User): boolean {
  switch (section.id) {
    case 'ami':
    case 'users':
      return user.isAdmin;
    default:
      return true;
  }
}
```

#### Phase 3: Cloudscape SideNavigation
```typescript
// Use Cloudscape SideNavigation component
<SideNavigation
  activeHref={activeSection}
  header={{ text: "Settings", href: "/settings" }}
  items={[
    { type: "link", text: "General", href: "/settings/general" },
    { type: "link", text: "Profiles", href: "/settings/profiles" },
    { 
      type: "expandable-link-group",
      text: "Advanced",
      items: [
        { type: "link", text: "AMI Management", href: "/settings/ami" },
        { type: "link", text: "Idle Detection", href: "/settings/idle" },
        { type: "link", text: "System", href: "/settings/system" }
      ]
    }
  ]}
/>
```

**Migration**:
- Keep all functionality, just reorganize
- Add breadcrumbs for deep settings
- Add search within settings
- Preserve bookmarks/direct links

**Testing**:
- [ ] All settings accessible
- [ ] Role-based visibility works
- [ ] Breadcrumb navigation correct
- [ ] Search finds settings
- [ ] Existing URLs redirect properly

---

### 3. Issue #18: Unified Storage UI (EFS + EBS)
**Priority**: P1 (Simplification)
**Effort**: Medium (2-3 days)
**Impact**: High (Reduces navigation by 1 item)

**Current State**:
- Storage: EFS (shared filesystems)
- Volumes: EBS (block storage)

**Proposed State**:
- Storage: Unified view of all storage
  - Types: Shared (EFS), Workspace (EBS), S3 (future)
  - Filters: By type, by workspace, by project

**Implementation**:

```typescript
// cmd/prism-gui/frontend/src/pages/Storage.tsx
interface StorageItem {
  id: string;
  name: string;
  type: 'efs' | 'ebs' | 's3';  // Extensible for future
  size: number;
  cost: number;
  attachedTo?: string[];  // Workspace IDs
  status: 'available' | 'in-use' | 'creating' | 'error';
}

Features:
- Tabbed view: All | EFS | EBS | S3 (future)
- Table with filters and search
- Actions: Create, attach, detach, resize, delete
- Cost breakdown by type
- Usage charts (storage over time)
```

**Cloudscape Components**:
- `Tabs` - Storage type switching
- `Table` - Unified storage list
- `PropertyFilter` - Advanced filtering
- `Modal` - Create/edit dialogs

**API Integration**:
- GET `/api/v1/storage` - Unified storage list
- POST `/api/v1/storage` - Create (type-specific)
- PUT `/api/v1/storage/{id}/attach` - Attach to workspace
- DELETE `/api/v1/storage/{id}` - Delete storage

**Testing**:
- [ ] All storage types visible
- [ ] Create EFS filesystem
- [ ] Create EBS volume
- [ ] Attach/detach operations
- [ ] Cost calculations correct
- [ ] Filters work properly

---

### 4. Issue #19: Integrate Budgets into Projects
**Priority**: P1 (Simplification)
**Effort**: Small (1-2 days)
**Impact**: Medium (Reduces navigation by 1 item)

**Current State**:
- Projects: Project management
- Budgets: Budget management (separate)

**Proposed State**:
- Projects: Integrated view
  - Each project has optional budget
  - Budget shown in project details
  - Budget alerts in project card

**Implementation**:

```typescript
// cmd/prism-gui/frontend/src/pages/Projects.tsx
interface ProjectCard {
  project: Project;
  budget?: Budget;
  currentSpend: number;
  workspaces: number;
  members: number;
}

Features:
- Project card shows budget status
- Budget indicator (progress bar)
- Alert badges for budget warnings
- Budget tab in project details
```

**Project Detail Tabs**:
```
- Overview (description, status, members)
- Workspaces (list of project workspaces)
- Budget (budget config, spending history, alerts)
- Settings (permissions, policies)
```

**API Changes** (if needed):
- Include budget in project response
- Aggregate spending by project
- Budget alerts API

**Migration**:
- Remove "Budgets" navigation item
- Redirect /budgets → /projects?tab=budget
- Update documentation

**Testing**:
- [ ] Projects show budget status
- [ ] Budget configuration works
- [ ] Budget alerts display
- [ ] Spending aggregation correct
- [ ] All personas can manage budgets

---

## 📅 Implementation Schedule

### Week 3 (Dec 16-20)
**Day 1-3**: Issue #14 - Merge Terminal/WebView
- Implement workspace detail component
- Add connection options
- Integrate terminal (xterm.js)
- Add web service detection

**Day 4-5**: Issue #18 - Unified Storage UI
- Design unified storage component
- Implement storage table with tabs
- Add filters and actions
- Test EFS/EBS operations

### Week 4 (Dec 23-27)
**Day 1-3**: Issue #16 - Settings restructure
- Reorganize navigation structure
- Implement settings side navigation
- Add role-based visibility
- Test all settings accessible

**Day 4**: Issue #19 - Integrate Budgets
- Add budget to project cards
- Implement budget detail tab
- Remove budgets navigation
- Test budget workflows

**Day 5**: Testing & polish
- Extended persona walkthroughs
- Navigation flow testing
- Performance testing
- Bug fixes

---

## 🧪 Extended Persona Walkthroughs

Continue extending walkthroughs from v0.5.8:

### Workspace Management Scenarios

**Scenario 1**: Multi-workspace research project
- Launch 2 workspaces (data processing + analysis)
- Connect to both via integrated terminal
- Share files via EFS storage
- Monitor costs across both workspaces
- Stop/hibernate when not in use

**Scenario 2**: Collaborative lab environment
- Create project with budget
- Launch shared workspace
- Add lab members to project
- Connect via web services (Jupyter)
- Monitor project spending
- Adjust budget as needed

**Scenario 3**: Class instruction
- Create class project
- Launch template workspaces for students
- Monitor all workspace status
- Connect to student workspace to help
- Manage costs within class budget
- Cleanup at end of semester

---

## 🔍 Testing Strategy

### Navigation Testing
- [ ] All 6 top-level items accessible
- [ ] Settings sections organized correctly
- [ ] Advanced features findable
- [ ] Role-based visibility works
- [ ] Breadcrumbs accurate
- [ ] Mobile navigation responsive

### Integration Testing
- [ ] Workspace connections work (SSH, web)
- [ ] Storage operations complete
- [ ] Project/budget integration seamless
- [ ] Settings changes apply correctly
- [ ] Terminal sessions stable

### Performance Testing
- [ ] Navigation transitions smooth (<300ms)
- [ ] Workspace list loads quickly (100+ workspaces)
- [ ] Terminal responsive (low latency)
- [ ] Storage table renders fast (50+ items)

### Usability Testing (Persona-Based)
- [ ] Solo Researcher: Find and use advanced features
- [ ] Lab Environment: Manage multiple workspaces
- [ ] University Class: Monitor class project budget
- [ ] All users: Navigate new structure intuitively

---

## 📚 Documentation Updates

### New Documentation
- [ ] New navigation guide
- [ ] Integrated workspace management tutorial
- [ ] Unified storage guide

### Updated Documentation
- [ ] Getting Started (new navigation)
- [ ] User Guide v0.5.x (navigation changes)
- [ ] Administrator Guide (settings location)
- [ ] Troubleshooting (updated paths)

### Migration Guide
- [ ] v0.5.8 → v0.5.9 navigation changes
- [ ] Old URL redirects
- [ ] Feature location mapping

### Persona Walkthrough Updates ✨
**Critical**: All 5 persona walkthroughs must be updated to reflect v0.5.9 features:

**a) Implemented Features Integration**:
- [ ] Update navigation paths (14 → 6 items)
- [ ] Document integrated Terminal/WebView in Workspaces tab
- [ ] Show unified Storage UI (EFS + EBS in one place)
- [ ] Update budget management in Projects context
- [ ] Document advanced features under Settings section
- [ ] Update all navigation references (old → new paths)

**b) GUI Screenshots**:
- [ ] New 6-item navigation bar (Home, Workspaces, Templates, Storage, Projects, Settings)
- [ ] Integrated workspace view with Terminal/WebView tabs
- [ ] Unified Storage UI showing both EFS and EBS
- [ ] Project details with integrated budget tab
- [ ] Settings page with Advanced submenu (AMI, Idle Detection, System)
- [ ] Role-based visibility (admin vs non-admin views)

**Persona-Specific Updates**:
1. **Solo Researcher**: Show integrated terminal access from workspace view
2. **Lab Environment**: Document unified storage for team file sharing
3. **University Class**: Show project budgets integrated with class projects
4. **Conference Workshop**: Simplified navigation for workshop participants
5. **Cross-Institutional**: Advanced features accessibility under Settings

**Before/After Comparisons**:
- [ ] Navigation: 14-item sidebar → 6-item organized navigation
- [ ] Workspace access: Separate Terminal/WebView pages → Integrated tabs
- [ ] Storage management: Two pages → Unified tabbed view
- [ ] Budget tracking: Separate Budgets page → Integrated in Projects

---

## 🚀 Release Criteria

### Must Have (Blocking)
- ✅ Issue #14 complete (merged Terminal/WebView)
- ✅ Issue #16 complete (Settings restructure)
- ✅ Issue #18 complete (unified Storage)
- ✅ Issue #19 complete (integrated Budgets)
- ✅ All persona tests pass
- ✅ Navigation flows intuitive
- ✅ No regressions
- ✅ Documentation updated

### Nice to Have (Non-Blocking)
- Enhanced terminal features (tabs, split panes)
- Storage usage charts
- Project dashboard improvements
- Video walkthroughs

---

## 📊 Success Metrics (Post-Release)

Track for 2 weeks after release:

1. **Navigation Complexity**
   - Achieved: 14 → 6 top-level items
   - Measure: Item count in main navigation

2. **Time to Find Features**
   - Target: <10 seconds for any feature
   - Measure: User testing, analytics
   - Track: Settings clicks, search usage

3. **Advanced Feature Usage**
   - Target: >95% discoverability
   - Measure: Features accessed / total users
   - Track: AMI management, Idle detection usage

4. **User Confusion**
   - Target: 30% further reduction
   - Measure: Support tickets, forum questions
   - Track: "Where is X?" questions

5. **Navigation Efficiency**
   - Target: Fewer clicks to complete tasks
   - Measure: Click-through analysis
   - Track: Path to workspace connection, storage creation

---

## 🔄 Backward Compatibility

### URL Redirects
```
Old URL                  → New URL
/terminal               → /workspaces?tab=terminal
/webview                → /workspaces?tab=services
/volumes                → /storage?type=ebs
/ami                    → /settings/advanced/ami
/idle                   → /settings/advanced/idle
/budgets                → /projects?tab=budgets
```

### API Compatibility
- All existing APIs remain functional
- New unified endpoints added
- Deprecation notices for old endpoints
- Migration guide for API consumers

---

## 🔗 Related Documents

- v0.5.8 Release Plan: `/tmp/v0.5.8_plan.md`
- UX Evaluation: `docs/architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md`
- GUI Architecture: `docs/architecture/GUI_ARCHITECTURE.md`
- Navigation Design Patterns: Cloudscape docs

