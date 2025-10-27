# Persona Walkthrough Screenshot Plan

**Created**: October 2025
**Purpose**: Visual enhancement of persona documents to improve engagement and clarity

---

## 📸 Screenshot Strategy

Adding screenshots to persona walkthroughs will:
- **Reduce cognitive load** - Visual examples are easier to understand than text alone
- **Build confidence** - Users see what to expect before trying commands
- **Showcase UX improvements** - Demonstrate v0.5.8 Quick Start wizard and workspace terminology
- **Appeal to visual learners** - 65% of people are visual learners

---

## 🎯 High-Priority Screenshots

### 1. Solo Researcher Walkthrough (01_SOLO_RESEARCHER_WALKTHROUGH.md)

**Critical Screenshots**:
1. **CLI Quick Start Wizard** (`prism init`) - Line 87
   - Screenshot: 6-step wizard flow showing template selection
   - Shows: Category selection → Template choice → Size selection → Name input → Confirmation
   - Impact: Demonstrates 30-second time-to-first-workspace

2. **GUI Template Gallery** - Line 150
   - Screenshot: GUI template selection with Cards and Badges
   - Shows: Cloudscape-based professional interface
   - Impact: Shows multi-modal access (CLI vs GUI)

3. **Terminal Connection** - Line 200
   - Screenshot: `prism connect` output showing SSH command
   - Shows: Clear connection instructions
   - Impact: Reduces "what do I do next?" confusion

4. **Workspace List** - Line 250
   - Screenshot: `prism list` showing running workspaces with costs
   - Shows: Real-time cost tracking
   - Impact: Demonstrates cost transparency

**Image Paths**:
```
docs/USER_SCENARIOS/images/01-solo-researcher/
├── cli-init-wizard-step1-categories.png
├── cli-init-wizard-step2-templates.png
├── cli-init-wizard-step3-size.png
├── cli-init-wizard-step4-name.png
├── cli-init-wizard-step5-confirmation.png
├── cli-init-wizard-step6-launching.png
├── gui-template-gallery.png
├── cli-connect-output.png
└── cli-list-workspaces.png
```

---

### 2. University Class Walkthrough (03_UNIVERSITY_CLASS_WALKTHROUGH.md)

**Critical Screenshots**:
1. **GUI Quick Start Wizard** - For student onboarding
   - Screenshot: 4-step visual wizard in Cloudscape
   - Shows: Student-friendly visual interface
   - Impact: Reduces instructor support burden

2. **TUI Dashboard** - For TA management
   - Screenshot: TUI with keyboard navigation
   - Shows: Alternative interface for terminal-comfortable users
   - Impact: Demonstrates interface flexibility

**Image Paths**:
```
docs/USER_SCENARIOS/images/03-university-class/
├── gui-quick-start-step1.png
├── gui-quick-start-step2.png
├── gui-quick-start-step3.png
├── gui-quick-start-step4.png
├── tui-dashboard-overview.png
└── tui-workspace-management.png
```

---

### 3. Conference Workshop Walkthrough (04_CONFERENCE_WORKSHOP_WALKTHROUGH.md)

**Critical Screenshots**:
1. **Bulk Workspace Launch** - Workshop preparation
   - Screenshot: Instructor launching multiple identical environments
   - Shows: Scalability for events
   - Impact: Demonstrates institutional capabilities

---

## 📋 Screenshot Capture Checklist

### CLI Screenshots (Terminal)
- [ ] Use consistent terminal theme (dark background, good contrast)
- [ ] Capture full command + output
- [ ] Show real data (not Lorem Ipsum)
- [ ] Highlight new v0.5.8 "workspace" terminology

**Tools**:
- macOS: `screencapture -w -o screenshot.png` (interactive window selection)
- iTerm2: Built-in screenshot feature (⌘+⇧+S)

**Commands to capture**:
```bash
# Must be running to capture:
prism init                          # Quick Start wizard
prism list                          # Workspace list with costs
prism connect my-workspace          # Connection instructions
prism templates                     # Template gallery in CLI
```

### GUI Screenshots (Desktop App)
- [ ] Capture at 1920x1080 resolution (standard)
- [ ] Show full window including title bar
- [ ] Demonstrate Cloudscape design system
- [ ] Include relevant data (project names, costs, templates)

**Screens to capture**:
- Home page with Quick Start wizard
- Template gallery (Templates tab)
- Workspace management (Instances tab)
- Project budget dashboard

### TUI Screenshots (Terminal)
- [ ] Use same terminal theme as CLI
- [ ] Show full screen layout
- [ ] Capture keyboard shortcuts in footer
- [ ] Show navigation tabs (1-6)

**Screens to capture**:
- Dashboard (Tab 1)
- Workspace management (Tab 2)
- Template browser (Tab 3)
- Storage management (Tab 4)

---

## 🖼️ Markdown Syntax for Screenshots

### Basic Image
```markdown
![Alt text description](../images/01-solo-researcher/cli-init-wizard.png)
```

### Image with Caption
```markdown
<p align="center">
  <img src="../images/01-solo-researcher/gui-template-gallery.png" alt="GUI Template Gallery" width="800">
  <br>
  <em>Professional template selection with Cloudscape Cards and Badges</em>
</p>
```

### Side-by-Side Comparison
```markdown
| CLI Interface | GUI Interface |
|---------------|---------------|
| ![CLI](../images/cli-example.png) | ![GUI](../images/gui-example.png) |
| Terminal-based command | Visual point-and-click |
```

---

## 📊 Priority Order

### Phase 1: Essential Screenshots (Week 1)
1. ✅ **Solo Researcher** - CLI Quick Start wizard (highest impact)
2. ✅ **Solo Researcher** - GUI template gallery
3. ✅ **University Class** - GUI Quick Start wizard
4. ✅ **Solo Researcher** - Workspace list with costs

### Phase 2: Enhanced Context (Week 2)
5. ⏳ **University Class** - TUI dashboard
6. ⏳ **Lab Environment** - Cost tracking dashboard
7. ⏳ **Conference Workshop** - Bulk launch workflow
8. ⏳ **Institutional IT** - Admin policy dashboard

### Phase 3: Advanced Features (Week 3)
9. ⏳ **Cross-Institutional** - Multi-account collaboration
10. ⏳ **HIPAA Compliance** - Audit trail and compliance reports
11. ⏳ **CUI Compliance** - Policy enforcement

---

## 🎨 Screenshot Best Practices

### Consistency
- **Terminal theme**: Consistent across all CLI/TUI screenshots
- **Window size**: 1920x1080 for GUI, 120x40 for terminal
- **Font size**: Readable at documentation width (~800px)

### Content
- **Use realistic data**: "cancer-research" not "test-project-123"
- **Show context**: Include relevant UI chrome (menus, tabs, status bars)
- **Highlight new features**: v0.5.8 workspace terminology, Quick Start wizard

### Technical
- **Format**: PNG (lossless, good for screenshots)
- **Optimization**: Compress with tools like ImageOptim or pngcrush
- **Size**: Target <500KB per image (for fast page loads)
- **Alt text**: Descriptive for accessibility

---

## 🚀 Implementation Steps

### Step 1: Create Image Directory Structure
```bash
mkdir -p docs/USER_SCENARIOS/images/{01-solo-researcher,02-lab-environment,03-university-class,04-conference-workshop,05-cross-institutional,06-nih-cui,07-nih-hipaa,08-institutional-it}
```

### Step 2: Capture Screenshots
- Launch Prism GUI and capture key screens
- Run CLI commands and capture terminal output
- Launch TUI and capture navigation flows

### Step 3: Add to Persona Documents
- Insert images at strategic points in walkthroughs
- Add captions explaining what's shown
- Create side-by-side CLI/GUI comparisons

### Step 4: Update README
- Add "Screenshots" section to each persona document header
- Link to screenshot plan for contributors

---

## 📝 Maintenance Plan

### When to Update Screenshots:
- **Major UI changes** (e.g., navigation restructure in v0.5.9)
- **New features** (e.g., Quick Start wizard in v0.5.8)
- **Branding updates** (logo changes, color scheme updates)
- **Bug fixes** affecting visual appearance

### Automation Opportunities:
- **Automated screenshot testing**: Playwright or Cypress for GUI
- **CLI output capture**: Script-based command execution with output capture
- **Version badges**: Automatically add version watermarks to screenshots

---

## ✅ Success Metrics

Track these metrics after adding screenshots:
- **User engagement**: Time spent on persona pages
- **Support reduction**: Fewer "how do I..." questions
- **Onboarding speed**: Time to first workspace launch for new users
- **Documentation quality**: User feedback on clarity and helpfulness

---

## 📚 Related Documentation

- [GUI User Guide](../user-guides/GUI_USER_GUIDE.md) - GUI screenshots and visual guides
- [TUI User Guide](../user-guides/TUI_USER_GUIDE.md) - Terminal interface screenshots
- [Getting Started](../user-guides/GETTING_STARTED.md) - Beginner-friendly visual walkthrough

---

**Next Steps**: Capture Phase 1 screenshots (CLI Quick Start wizard + GUI template gallery)
