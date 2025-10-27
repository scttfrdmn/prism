# Prism v0.5.8 Release Plan: Quick Start Experience

**Release Date**: Target December 2025
**Focus**: First-time user experience - zero to running workspace in <30 seconds

## 🎯 Release Goals

### Primary Objective
Transform the first-time user experience from 15 minutes to 30 seconds by implementing:
1. Intuitive Quick Start wizard (GUI)
2. Guided CLI onboarding (`prism init`)
3. Consistent "Workspaces" terminology

### Success Metrics
- ⏱️ Time to first workspace launch: 15min → 30sec
- 🎯 First-attempt success rate: >90%
- 😃 User confusion (support tickets): Reduce by 70%

---

## 📦 Features & Issues

### 1. Issue #15: Rename "Instances" → "Workspaces"
**Priority**: P0 (Foundation for other changes)
**Effort**: Small (2-3 hours)
**Impact**: High (Better mental model)

**Implementation**:
- [ ] Update all GUI labels, titles, navigation items
- [ ] Update CLI help text, command descriptions
- [ ] Update API documentation
- [ ] Update user-facing documentation
- [ ] Search for "instance" in codebase, replace with "workspace" where user-facing

**Files to Update**:
- `cmd/prism-gui/frontend/src/**/*.tsx` - All React components
- `internal/cli/*.go` - CLI help text
- `docs/user-guides/**` - All user documentation
- Keep internal code using "instance" (AWS terminology)

**Testing**:
- Visual inspection of all GUI pages
- CLI help text review
- Documentation consistency check

---

### 2. Issue #13: Home Page with Quick Start Wizard (GUI)
**Priority**: P0 (Core feature)
**Effort**: Large (3-4 days)
**Impact**: Critical (Main UX improvement)

**Design**: See wireframes in `/tmp/quick_start_wireframes.md`

**Implementation**:

#### Phase 1: Home Page Component
```typescript
// cmd/prism-gui/frontend/src/pages/Home.tsx
interface HomePageProps {
  onQuickStart: () => void;
  recentWorkspaces: Workspace[];
  systemStatus: SystemStatus;
}

Components:
- Hero section with Quick Start CTA
- Recent workspaces grid (last 5)
- System status cards (running, costs, alerts)
- Quick actions (launch from template, connect to existing)
```

#### Phase 2: Quick Start Wizard
```typescript
// cmd/prism-gui/frontend/src/components/QuickStartWizard.tsx
Steps:
1. Welcome & Template Selection
   - Categories: ML/AI, Data Science, Web Dev, Bioinformatics, Custom
   - Template cards with icons, descriptions, tags
   - Search and filter

2. Basic Configuration
   - Workspace name (required, validated)
   - Size: S/M/L/XL (with cost estimates)
   - Optional: Advanced settings accordion

3. Review & Launch
   - Summary of selections
   - Estimated monthly cost
   - Launch button (disabled until valid)

4. Progress & Success
   - Real-time progress indicator
   - Log streaming (optional expansion)
   - Connection details on success
   - Quick action: Connect now / View workspace
```

#### Phase 3: Cloudscape Components
- `Wizard` - Multi-step flow
- `Cards` - Template selection
- `Form` - Configuration inputs
- `ProgressBar` - Launch progress
- `Alert` - Success/error messages
- `Button` - Primary actions

**API Integration**:
- GET `/api/v1/templates` - List templates with categories
- POST `/api/v1/workspaces` - Launch workspace
- GET `/api/v1/workspaces/{id}/status` - Poll launch status
- GET `/api/v1/workspaces` - Recent workspaces

**Testing**:
- [ ] Solo Researcher: Launch Python ML workspace in <30sec
- [ ] Lab Environment: Launch shared workspace
- [ ] University Class: Launch from custom template
- [ ] All template categories work
- [ ] Error handling (invalid names, AWS errors)
- [ ] Progress indicators accurate

---

### 3. Issue #17: CLI `prism init` Onboarding Wizard
**Priority**: P1 (CLI companion to GUI)
**Effort**: Medium (2-3 days)
**Impact**: High (CLI users get same experience)

**Design**: Interactive CLI wizard matching GUI flow

**Implementation**:

```go
// internal/cli/init.go
func InitCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "init",
        Short: "Launch your first Prism workspace (guided setup)",
        Long: `Interactive wizard to help you launch your first research workspace.
        
This command will guide you through:
  1. Template selection
  2. Workspace configuration
  3. Launch and connection`,
        RunE: runInit,
    }
}

func runInit(cmd *cobra.Command, args []string) error {
    // Step 1: Welcome & AWS check
    fmt.Println("🎉 Welcome to Prism!")
    if err := checkAWSCredentials(); err != nil {
        return guideAWSSetup(err)
    }
    
    // Step 2: Template selection
    templates := fetchTemplates()
    selected := interactiveTemplateSelect(templates)
    
    // Step 3: Configuration
    name := promptWorkspaceName()
    size := promptSize(selected.RecommendedSize)
    advanced := promptAdvancedOptions()
    
    // Step 4: Review & confirm
    displaySummary(selected, name, size, advanced)
    if !confirm("Launch this workspace?") {
        return nil
    }
    
    // Step 5: Launch & progress
    workspace := launchWithProgress(selected, name, size, advanced)
    
    // Step 6: Success
    displayConnectionInfo(workspace)
    offerNextSteps(workspace)
    
    return nil
}
```

**Interactive Elements**:
- Use `survey` library for prompts
- Template selection: Arrow key navigation with descriptions
- Size selection: Radio buttons with cost estimates
- Progress: Spinner with status updates
- Colors: Green for success, yellow for warnings, red for errors

**Features**:
- Auto-detect AWS profile and region
- Validate inputs before launch
- Show estimated costs
- Offer to save preferences for next time
- Guide to `prism connect` after launch

**Testing**:
- [ ] Solo Researcher: First-time setup
- [ ] Existing user: Quick re-launch
- [ ] No AWS credentials: Helpful error
- [ ] Invalid inputs: Clear validation
- [ ] Network errors: Graceful handling

---

## 🧪 Extended Persona Walkthroughs

### Extending Walkthroughs: Post-Launch Activities

For v0.5.8, extend each persona walkthrough to include:

1. **Connection & Access**
   - SSH connection process
   - Web service access (Jupyter, RStudio)
   - File transfer (upload/download)

2. **Actual Work**
   - Run example analysis
   - Install additional packages
   - Save work/results

3. **Workspace Management**
   - Stop/start workspace
   - Monitor costs
   - Share with collaborators (if applicable)

4. **Cleanup**
   - Save important files to persistent storage
   - Terminate workspace
   - Verify costs

**Example Extended Walkthrough** (Solo Researcher):

```markdown
### Post-Launch: Python ML Analysis

**Scenario**: Train a simple ML model on sample data

1. **Connect to Workspace** (<30 seconds)
   - GUI: Click "Connect" → Opens Jupyter in browser
   - CLI: `prism connect my-ml-workspace --jupyter`
   - Verify: Jupyter interface loads, conda env active

2. **Prepare Environment** (2 minutes)
   - Create new notebook
   - Verify pre-installed packages (pandas, scikit-learn, torch)
   - Upload sample dataset (iris.csv)

3. **Run Analysis** (5 minutes)
   - Load data with pandas
   - Train simple classifier
   - Evaluate model
   - Save results to /data (EFS - persists)

4. **Manage Workspace** (1 minute)
   - Check running time and cost
   - Download trained model
   - Stop workspace (or leave running)

5. **Resume Later** (next day)
   - Start workspace
   - Reconnect to Jupyter
   - Verify /data files persist
   - Continue work

**Total Time**: ~30 minutes of productive work
**Cost**: $0.50 for 2 hours (with hibernation)
```

---

## 📅 Implementation Schedule

### Week 1 (Dec 2-6)
**Day 1-2**: Issue #15 - Instances → Workspaces rename
- Update GUI components
- Update CLI help text
- Update documentation
- Test and commit

**Day 3-5**: Issue #13 - Design & start implementation
- Create detailed wireframes
- Implement Home page component
- Start Quick Start wizard (template selection)

### Week 2 (Dec 9-13)
**Day 1-3**: Issue #13 - Complete wizard implementation
- Configuration step
- Review & launch
- Progress tracking
- Success screen

**Day 4-5**: Issue #17 - CLI `prism init`
- Implement interactive wizard
- Match GUI flow
- Test with all personas

**Testing**: Extended persona walkthroughs

---

## 🔍 Testing Strategy

### Pre-Release Testing

**1. Functionality Testing**
- [ ] All wizard steps work correctly
- [ ] All template categories functional
- [ ] Progress indicators accurate
- [ ] Error handling comprehensive
- [ ] CLI/GUI feature parity

**2. Persona Validation** (Extended Walkthroughs)
- [ ] Solo Researcher: ML workflow end-to-end
- [ ] Lab Environment: Shared workspace workflow
- [ ] University Class: Student onboarding
- [ ] Conference Workshop: Quick launch scenario
- [ ] Cross-Institutional: Multi-profile usage

**3. Performance Testing**
- [ ] Wizard loads in <2 seconds
- [ ] Template list renders quickly (50+ templates)
- [ ] Launch initiates immediately
- [ ] Progress updates every 5 seconds

**4. Accessibility Testing**
- [ ] Keyboard navigation works
- [ ] Screen reader friendly
- [ ] Color contrast sufficient
- [ ] Focus indicators visible

**5. Regression Testing**
- [ ] Existing features still work
- [ ] Templates tab unchanged
- [ ] Workspaces tab functional
- [ ] API backward compatible

---

## 📚 Documentation Updates

### New Documentation
- [ ] Quick Start guide (GUI wizard)
- [ ] CLI `prism init` tutorial
- [ ] Extended persona walkthroughs (with post-launch)

### Updated Documentation
- [ ] Getting Started guide
- [ ] User Guide v0.5.x
- [ ] Troubleshooting (new wizard errors)
- [ ] Architecture docs (new components)

---

## 🚀 Release Criteria

### Must Have (Blocking)
- ✅ Issue #15 complete (rename)
- ✅ Issue #13 complete (Quick Start GUI)
- ✅ Issue #17 complete (CLI init)
- ✅ All persona tests pass
- ✅ No critical bugs
- ✅ Documentation updated

### Nice to Have (Non-Blocking)
- Extended persona walkthroughs with real data
- Video tutorials
- Blog post announcement
- User testing with real researchers

---

## 📊 Success Metrics (Post-Release)

Track for 2 weeks after release:

1. **Time to First Workspace**
   - Target: <30 seconds
   - Measure: Wizard start → workspace ready
   - Track: GUI analytics

2. **First-Attempt Success Rate**
   - Target: >90%
   - Measure: Successful launches / total attempts
   - Track: Error logs

3. **Support Ticket Volume**
   - Target: 70% reduction in "getting started" tickets
   - Measure: Support ticket categories
   - Track: GitHub issues, user feedback

4. **Wizard Completion Rate**
   - Target: >85% complete wizard
   - Measure: Started / completed
   - Track: GUI analytics

5. **CLI `init` Usage**
   - Target: 40% of new users use `prism init`
   - Measure: Command usage stats
   - Track: Telemetry (opt-in)

---

## 🔗 Related Documents

- UX Evaluation: `docs/architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md`
- Persona Walkthroughs: `docs/USER_SCENARIOS/`
- GUI Architecture: `docs/architecture/GUI_ARCHITECTURE.md`
- Design Principles: `docs/DESIGN_PRINCIPLES.md`

