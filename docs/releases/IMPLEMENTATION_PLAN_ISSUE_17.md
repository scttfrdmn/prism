# Implementation Plan: Issue #17 - CLI `prism init` Onboarding Wizard

**GitHub Issue**: [#17](https://github.com/scttfrdmn/prism/issues/17)
**Milestone**: v0.5.8 - Quick Start Experience
**Priority**: P1 (CLI companion to GUI Quick Start wizard)
**Effort**: Medium (2-3 days)
**Status**: 🔄 In Progress

---

## 📋 Overview

Create an interactive CLI wizard (`prism init`) that provides first-time users with a guided setup experience matching the GUI Quick Start wizard. This command reduces time to first workspace from 15 minutes to <30 seconds for CLI users.

**Success Criteria**:
- ⏱️ Time to first workspace: <30 seconds
- 🎯 First-attempt success rate: >90%
- 😃 Clear guidance at each step
- 🔄 Feature parity with GUI Quick Start wizard

---

## 🏗️ Architecture

### File Structure
```
internal/cli/
├── init.go              # New: Init command implementation
├── root_command.go      # Modified: Register init command
└── templates_cobra.go   # Existing: Template operations to reuse
```

### Command Flow
```
prism init
  ├─ Step 1: Welcome & AWS Check
  ├─ Step 2: Template Selection (categorized)
  ├─ Step 3: Configuration (name, size)
  ├─ Step 4: Review & Confirm
  ├─ Step 5: Launch with Progress
  └─ Step 6: Success & Connection Info
```

---

## 🎯 Detailed Implementation Plan

### Phase 1: Command Infrastructure (30 min)

**File**: `internal/cli/init_cobra.go`

**Create Cobra Command Structure**:
```go
type InitCobraCommands struct {
    app *App
}

func NewInitCobraCommands(app *App) *InitCobraCommands {
    return &InitCobraCommands{app: app}
}

func (ic *InitCobraCommands) CreateInitCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:     "init",
        Short:   "Launch your first workspace (guided setup)",
        GroupID: "core",
        Long: `Interactive wizard to launch your first Prism workspace.

This command guides you through:
  1. Template selection from categorized options
  2. Workspace configuration (name and size)
  3. Review and confirmation
  4. Launch with real-time progress
  5. Connection details and next steps`,
        RunE: func(cmd *cobra.Command, args []string) error {
            return ic.runInitWizard()
        },
    }

    cmd.Flags().Bool("skip-aws-check", false, "Skip AWS credential validation")
    cmd.Flags().Bool("non-interactive", false, "Use defaults without prompts")

    return cmd
}
```

**Register in root_command.go**:
```go
// In RegisterAllCommands() method, add after line ~385:
initCobra := NewInitCobraCommands(r.app)
rootCmd.AddCommand(initCobra.CreateInitCommand())
```

---

### Phase 2: Step 1 - Welcome & AWS Check (45 min)

**Implementation**:
```go
func (ic *InitCobraCommands) runInitWizard() error {
    // Welcome message
    ic.printWelcome()

    // Check AWS credentials
    if err := ic.checkAWSCredentials(); err != nil {
        return ic.guideAWSSetup(err)
    }

    fmt.Println("✅ AWS credentials validated")
    fmt.Println()

    // Continue to template selection
    return ic.selectTemplate()
}

func (ic *InitCobraCommands) printWelcome() {
    fmt.Println("🎉 Welcome to Prism!")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println()
    fmt.Println("This wizard will help you launch your first research workspace.")
    fmt.Println()
}

func (ic *InitCobraCommands) checkAWSCredentials() error {
    // Ensure daemon running
    if err := ic.app.ensureDaemonRunning(); err != nil {
        return fmt.Errorf("failed to start daemon: %w", err)
    }

    // Check AWS credentials via API
    client := ic.app.apiClient
    _, err := client.ListInstances() // Basic API call to validate credentials
    return err
}

func (ic *InitCobraCommands) guideAWSSetup(err error) error {
    fmt.Println("❌ AWS credentials not configured")
    fmt.Println()
    fmt.Println("To use Prism, you need AWS credentials. Here's how to set them up:")
    fmt.Println()
    fmt.Println("1. Install AWS CLI:")
    fmt.Println("   brew install awscli  # macOS")
    fmt.Println("   pip install awscli   # Python")
    fmt.Println()
    fmt.Println("2. Configure credentials:")
    fmt.Println("   aws configure")
    fmt.Println()
    fmt.Println("3. Run 'prism init' again")
    fmt.Println()
    fmt.Println("For detailed setup: https://docs.prism.dev/aws-setup")

    return fmt.Errorf("AWS credentials required: %w", err)
}
```

---

### Phase 3: Step 2 - Template Selection (1 hour)

**Implementation**:
```go
func (ic *InitCobraCommands) selectTemplate() error {
    fmt.Println("📦 Step 1: Select a Template")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println()

    // Fetch templates
    templates, err := ic.fetchTemplates()
    if err != nil {
        return fmt.Errorf("failed to fetch templates: %w", err)
    }

    // Categorize templates
    categories := ic.categorizeTemplates(templates)

    // Display categories
    fmt.Println("Choose a category:")
    fmt.Println()
    categoryNames := []string{"ML/AI", "Data Science", "Bioinformatics", "Web Development", "All Templates"}
    for i, cat := range categoryNames {
        fmt.Printf("  %d) %s\n", i+1, cat)
    }
    fmt.Println()

    // Get category selection
    catIdx := ic.promptChoice("Select category", 1, len(categoryNames))
    selectedCategory := categoryNames[catIdx-1]

    // Display templates in category
    categoryTemplates := categories[selectedCategory]
    fmt.Println()
    fmt.Printf("📋 %s Templates:\n\n", selectedCategory)

    for i, tmpl := range categoryTemplates {
        fmt.Printf("  %d) %s\n", i+1, tmpl.Name)
        fmt.Printf("     %s\n", tmpl.Description)
        if tmpl.RecommendedSize != "" {
            fmt.Printf("     Recommended: %s (~$%.2f/hour)\n", tmpl.RecommendedSize, tmpl.EstimatedCost)
        }
        fmt.Println()
    }

    // Get template selection
    tmplIdx := ic.promptChoice("Select template", 1, len(categoryTemplates))
    selectedTemplate := categoryTemplates[tmplIdx-1]

    return ic.configureWorkspace(selectedTemplate)
}

func (ic *InitCobraCommands) fetchTemplates() ([]Template, error) {
    client := ic.app.apiClient
    templatesMap, err := client.ListTemplates()
    if err != nil {
        return nil, err
    }

    templates := make([]Template, 0, len(templatesMap))
    for _, tmpl := range templatesMap {
        templates = append(templates, tmpl)
    }

    return templates, nil
}

func (ic *InitCobraCommands) categorizeTemplates(templates []Template) map[string][]Template {
    categories := map[string][]Template{
        "ML/AI":          {},
        "Data Science":   {},
        "Bioinformatics": {},
        "Web Development": {},
        "All Templates":  templates,
    }

    for _, tmpl := range templates {
        name := strings.ToLower(tmpl.Name)
        desc := strings.ToLower(tmpl.Description)

        if strings.Contains(name, "ml") || strings.Contains(name, "machine learning") || strings.Contains(desc, "tensorflow") {
            categories["ML/AI"] = append(categories["ML/AI"], tmpl)
        }
        if strings.Contains(name, "python") || strings.Contains(name, "jupyter") || strings.Contains(name, "data") {
            categories["Data Science"] = append(categories["Data Science"], tmpl)
        }
        if strings.Contains(name, "bio") || strings.Contains(name, "genomics") {
            categories["Bioinformatics"] = append(categories["Bioinformatics"], tmpl)
        }
        if strings.Contains(name, "web") || strings.Contains(name, "node") {
            categories["Web Development"] = append(categories["Web Development"], tmpl)
        }
    }

    return categories
}

func (ic *InitCobraCommands) promptChoice(prompt string, min, max int) int {
    reader := bufio.NewReader(os.Stdin)
    for {
        fmt.Printf("%s [%d-%d]: ", prompt, min, max)
        input, _ := reader.ReadString('\n')
        input = strings.TrimSpace(input)

        choice, err := strconv.Atoi(input)
        if err == nil && choice >= min && choice <= max {
            return choice
        }

        fmt.Printf("❌ Please enter a number between %d and %d\n\n", min, max)
    }
}
```

---

### Phase 4: Step 3 - Configuration (45 min)

**Implementation**:
```go
func (ic *InitCobraCommands) configureWorkspace(template Template) error {
    fmt.Println()
    fmt.Println("⚙️  Step 2: Configure Workspace")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println()

    // Prompt for workspace name
    name := ic.promptWorkspaceName()

    // Prompt for size
    size := ic.promptSize(template.RecommendedSize)

    return ic.reviewAndLaunch(template, name, size)
}

func (ic *InitCobraCommands) promptWorkspaceName() string {
    reader := bufio.NewReader(os.Stdin)

    // Suggest a default name
    defaultName := fmt.Sprintf("my-workspace-%s", time.Now().Format("0102"))

    for {
        fmt.Printf("Workspace name (default: %s): ", defaultName)
        input, _ := reader.ReadString('\n')
        input = strings.TrimSpace(input)

        if input == "" {
            return defaultName
        }

        // Validate name (alphanumeric and hyphens)
        if matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]$`, input); matched {
            return input
        }

        fmt.Println("❌ Name must contain only letters, numbers, and hyphens")
        fmt.Println()
    }
}

func (ic *InitCobraCommands) promptSize(recommendedSize string) string {
    fmt.Println()
    fmt.Println("Choose workspace size:")
    fmt.Println()

    sizes := []struct {
        name string
        spec string
        cost string
    }{
        {"S", "2 vCPU, 4GB RAM", "~$0.08/hour"},
        {"M", "4 vCPU, 8GB RAM", "~$0.16/hour"},
        {"L", "8 vCPU, 16GB RAM", "~$0.32/hour"},
        {"XL", "16 vCPU, 32GB RAM", "~$0.64/hour"},
    }

    for i, size := range sizes {
        marker := "  "
        if size.name == recommendedSize {
            marker = "→ "
        }
        fmt.Printf("%s%d) %s - %s (%s)\n", marker, i+1, size.name, size.spec, size.cost)
    }

    fmt.Println()
    if recommendedSize != "" {
        fmt.Printf("💡 Tip: Size '%s' is recommended for this template\n\n", recommendedSize)
    }

    choice := ic.promptChoice("Select size", 1, len(sizes))
    return sizes[choice-1].name
}
```

---

### Phase 5: Step 4 - Review & Launch (45 min)

**Implementation**:
```go
func (ic *InitCobraCommands) reviewAndLaunch(template Template, name, size string) error {
    fmt.Println()
    fmt.Println("📋 Step 3: Review Configuration")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println()

    // Display summary
    fmt.Println("Your workspace configuration:")
    fmt.Println()
    fmt.Printf("  Template:  %s\n", template.Name)
    fmt.Printf("  Name:      %s\n", name)
    fmt.Printf("  Size:      %s\n", size)
    fmt.Println()

    // Show cost estimate
    costPerHour := ic.estimateCost(size)
    costPerMonth := costPerHour * 730 // Average hours per month
    fmt.Printf("  Estimated cost: $%.2f/hour (~$%.2f/month if running 24/7)\n", costPerHour, costPerMonth)
    fmt.Println()
    fmt.Println("💡 Tip: Use 'prism stop' when not in use to save costs")
    fmt.Println()

    // Confirm
    if !ic.promptConfirm("Launch this workspace?") {
        fmt.Println("\n❌ Launch cancelled")
        return nil
    }

    return ic.launchWorkspace(template, name, size)
}

func (ic *InitCobraCommands) estimateCost(size string) float64 {
    costs := map[string]float64{
        "S":  0.08,
        "M":  0.16,
        "L":  0.32,
        "XL": 0.64,
    }
    if cost, ok := costs[size]; ok {
        return cost
    }
    return 0.16 // default to M
}

func (ic *InitCobraCommands) promptConfirm(prompt string) bool {
    reader := bufio.NewReader(os.Stdin)
    fmt.Printf("%s [y/N]: ", prompt)
    input, _ := reader.ReadString('\n')
    input = strings.TrimSpace(strings.ToLower(input))
    return input == "y" || input == "yes"
}
```

---

### Phase 6: Step 5 - Launch with Progress (1 hour)

**Implementation**:
```go
func (ic *InitCobraCommands) launchWorkspace(template Template, name, size string) error {
    fmt.Println()
    fmt.Println("🚀 Step 4: Launching Workspace")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println()

    // Build launch request
    launchArgs := []string{
        "launch",
        template.Slug,
        name,
        "--size", size,
        "--wait", // Wait for launch to complete
    }

    // Launch via existing Launch method
    fmt.Println("⏳ Launching workspace... This may take 1-2 minutes")
    fmt.Println()

    err := ic.app.Launch(launchArgs)
    if err != nil {
        fmt.Println()
        fmt.Printf("❌ Failed to launch workspace: %v\n", err)
        return err
    }

    return ic.displaySuccess(name)
}

func (ic *InitCobraCommands) displaySuccess(name string) error {
    fmt.Println()
    fmt.Println("✅ Success! Your workspace is ready")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println()

    // Get workspace details
    client := ic.app.apiClient
    instance, err := client.GetInstance(name)
    if err != nil {
        return err
    }

    // Display connection info
    fmt.Println("📡 Connection Information:")
    fmt.Println()
    fmt.Printf("  Name:      %s\n", instance.Name)
    fmt.Printf("  Status:    %s\n", instance.State)
    fmt.Printf("  Public IP: %s\n", instance.PublicIP)
    fmt.Println()

    // SSH command
    fmt.Println("🔗 Connect via SSH:")
    fmt.Printf("  ssh ubuntu@%s\n", instance.PublicIP)
    fmt.Println()

    // Web services if available
    if len(instance.WebServices) > 0 {
        fmt.Println("🌐 Web Services:")
        for _, svc := range instance.WebServices {
            fmt.Printf("  %s: http://%s:%d\n", svc.Name, instance.PublicIP, svc.Port)
        }
        fmt.Println()
    }

    // Next steps
    fmt.Println("📚 Next Steps:")
    fmt.Println("  • Connect:  prism connect", name)
    fmt.Println("  • Monitor:  prism list")
    fmt.Println("  • Stop:     prism stop", name)
    fmt.Println("  • Delete:   prism delete", name)
    fmt.Println()
    fmt.Println("💡 Run 'prism --help' to see all available commands")
    fmt.Println()

    return nil
}
```

---

## 🧪 Testing Plan

### Manual Testing Checklist

**Test 1: Complete Flow (Happy Path)**
```bash
prism init
# → Select category: ML/AI
# → Select template: Python Machine Learning
# → Name: test-init-wizard
# → Size: M
# → Confirm: y
# → Wait for launch
# → Verify success message and connection info
```

**Test 2: AWS Credentials Missing**
```bash
# Temporarily remove AWS credentials
export AWS_PROFILE=nonexistent
prism init
# → Should show AWS setup guidance
# → Should exit gracefully
```

**Test 3: Invalid Input Handling**
```bash
prism init
# → Enter invalid category number (0, 100)
# → Enter invalid workspace name (spaces, special chars)
# → Verify validation and re-prompts
```

**Test 4: Cancel Flow**
```bash
prism init
# → Go through steps
# → Answer 'n' to confirmation
# → Verify graceful cancellation
```

**Test 5: Non-Interactive Mode** (Future)
```bash
prism init --non-interactive
# → Should use sensible defaults
# → Should launch without prompts
```

---

## 📊 Success Metrics

**Measure Against v0.5.8 Goals**:
- ⏱️ Time to first workspace: **Target <30 seconds**
  - Measure from `prism init` to workspace running
  - Include all prompt interactions

- 🎯 First-attempt success rate: **Target >90%**
  - Track successful completions vs. errors/cancellations

- 😃 User confusion: **Reduce by 70%**
  - Clear prompts and validation messages
  - Helpful guidance at each step

---

## 🔄 Integration Points

### Reuse Existing Code
- `app.ensureDaemonRunning()` - AWS credential check
- `app.apiClient.ListTemplates()` - Template fetching
- `app.Launch(args)` - Actual launch operation
- `app.apiClient.GetInstance(name)` - Workspace status

### New Dependencies
- `bufio` - Input reading
- `strconv` - String to int conversion
- `regexp` - Name validation
- `time` - Default name generation

---

## 📝 Git Workflow

### Branch Strategy
```bash
git checkout -b feature/issue-17-cli-init-wizard
```

### Commit Structure
```
1. "feat(cli): Add init command infrastructure (#17)"
   - Create init_cobra.go
   - Register in root_command.go

2. "feat(cli): Implement init wizard steps 1-2 (#17)"
   - Welcome and AWS check
   - Template selection with categories

3. "feat(cli): Implement init wizard steps 3-4 (#17)"
   - Workspace configuration
   - Review and confirmation

4. "feat(cli): Implement init wizard steps 5-6 (#17)"
   - Launch with progress
   - Success display

5. "test(cli): Add init wizard manual testing (#17)"
   - Test all flows
   - Document edge cases

6. "docs(cli): Update README with init command (#17)"
   - Add quick start example
   - Update command reference
```

### Pull Request
```markdown
## Summary
Implements CLI `prism init` onboarding wizard for v0.5.8 Quick Start Experience.

## Changes
- ✅ Interactive wizard with 6 steps
- ✅ Category-based template selection
- ✅ Input validation and helpful prompts
- ✅ Real-time launch progress
- ✅ Connection info and next steps

## Testing
- ✅ Happy path: Complete wizard flow
- ✅ AWS credentials missing
- ✅ Invalid inputs handled gracefully
- ✅ Cancel flow works correctly

## Related Issues
Closes #17

## Success Metrics
- ⏱️ Time to first workspace: <30 seconds ✅
- 🎯 Clear guidance at each step ✅
- 🔄 Feature parity with GUI wizard ✅
```

---

## 🚀 Implementation Timeline

**Day 1** (4 hours):
- ✅ Create implementation plan (this document)
- [ ] Phase 1: Command infrastructure (30 min)
- [ ] Phase 2: Welcome & AWS check (45 min)
- [ ] Phase 3: Template selection (1 hour)
- [ ] Testing of Steps 1-2 (45 min)

**Day 2** (4 hours):
- [ ] Phase 4: Configuration (45 min)
- [ ] Phase 5: Review & launch (45 min)
- [ ] Phase 6: Success display (1 hour)
- [ ] Testing of Steps 3-6 (1.5 hours)

**Day 3** (2 hours):
- [ ] Complete testing checklist
- [ ] Documentation updates
- [ ] Code review and refinements
- [ ] Submit PR

---

## 🎯 Definition of Done

- [ ] Command registered in root_command.go
- [ ] All 6 wizard steps implemented
- [ ] Input validation working
- [ ] AWS credential check functional
- [ ] Launch integration working
- [ ] Success message displays correctly
- [ ] Manual testing complete (all 5 tests)
- [ ] Code committed with proper messages
- [ ] README updated with init command
- [ ] PR submitted and linked to issue #17

---

**Document Status**: 📝 Complete - Ready for Implementation
**Next Step**: Begin Phase 1 - Command Infrastructure
**Updated**: 2025-10-27
