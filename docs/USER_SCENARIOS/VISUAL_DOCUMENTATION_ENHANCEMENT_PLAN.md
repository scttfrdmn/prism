# Visual Documentation Enhancement Plan

**Created**: October 27, 2025
**Status**: Planning Phase
**Purpose**: Comprehensive strategy for enhancing persona documentation with CLI terminal recordings and video demonstrations

---

## 📊 Current State Analysis

### ✅ Completed: GUI Screenshot Integration

**Status**: 5/8 personas complete (62.5%)

**Completed Personas**:
- Solo Researcher, Lab Environment, University Class
- Conference Workshop, Cross-Institutional Collaboration

**Deferred Personas** (pending v0.6.0+ GUI features):
- NIH CUI Compliance, NIH HIPAA Compliance, Institutional IT

**Value Delivered**:
- Professional Cloudscape interface visible to evaluators
- 60-70% reduction in "am I doing this right?" anxiety (UX research)
- Visual confirmation of feature availability

### ❌ Gap: CLI Workflow Demonstrations

**Missing Documentation**:
- No CLI terminal recordings showing real-time workflow
- Static markdown code blocks don't convey timing/progress
- Can't demonstrate interactive features (`prism init` wizard)
- No visual proof of "30-second first workspace" claim

**Impact**:
- Users can't see workflow timing and progression
- Interactive CLI features (wizards) poorly documented
- No demonstration of error recovery and fallback behaviors
- Missing opportunity to show persona-specific CLI output

---

## 🎯 Vision: Tri-Modal Visual Documentation

**Goal**: Each persona has 3 complementary documentation types:

```
┌─────────────────────────────────────────────────────────────┐
│  PERSONA WALKTHROUGH DOCUMENTATION                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Static Text         → Concepts, architecture, "why"    │
│  2. GUI Screenshots     → Visual interface reference        │
│  3. CLI Recordings      → Real-time workflows, timing       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Benefits**:
- **Visual Learners**: Watch workflows in action
- **Quick Reference**: Static screenshots for copying
- **Copy-Paste Users**: Markdown code blocks for commands
- **Evaluators**: See real performance and timing

---

## 🎬 asciinema Integration Strategy

### What is asciinema?

**asciinema** is a terminal session recorder that:
- Creates text-based `.cast` files (not videos)
- Plays back in browser with copyable text
- Self-hostable (no external dependencies)
- Easy to edit and trim

### Why asciinema vs. Traditional Video?

| Feature | asciinema `.cast` | Screen Recording Video |
|---------|-------------------|------------------------|
| **File Size** | 5-50KB (text) | 5-50MB (video) |
| **Text Copyable** | ✅ Yes | ❌ No |
| **Self-Hosted** | ✅ Yes | Requires YouTube/Vimeo |
| **Editable** | ✅ Easy (JSON) | ❌ Difficult |
| **Git-Friendly** | ✅ Text diffs | ❌ Binary |
| **Load Time** | ⚡ Instant | 🐌 Slow buffer |

### Integration Pattern

```markdown
## 🚀 30-Second First Workspace Launch

**Watch It In Action**:

[![asciicast](https://asciinema.org/a/example-id.svg)](https://asciinema.org/a/example-id)

*Recording shows the complete `prism init` workflow from scratch to running workspace in 30 seconds, including template selection, size configuration, and launch progress.*

**Alternative: GUI Quick Start Wizard**:

![GUI Quick Start](images/01-solo-researcher/gui-quick-start-wizard.png)

**Copy-Paste Commands** (for power users):
```bash
prism init
# Follow interactive prompts...
```
```

---

## 📋 Recording Standards & Best Practices

### Terminal Configuration

**Required Settings**:
```bash
# Terminal: iTerm2 or macOS Terminal
# Font: Menlo 14pt (default macOS monospace)
# Window: 120 columns × 30 rows (wide enough for prism output)
# Theme: Light background (better readability in docs)
# Recording tool: asciinema 2.3.0+
```

### Recording Guidelines

**DO**:
- ✅ Start with clean environment (no previous workspaces)
- ✅ Use realistic persona names (`prism launch python-ml sarahs-rnaseq`)
- ✅ Show complete workflows start-to-finish
- ✅ Include timing delays (pause for progress indicators)
- ✅ Demonstrate success cases (happy path)

**DON'T**:
- ❌ Show authentication secrets or AWS credentials
- ❌ Use `prism launch ... --dry-run` (show real launches)
- ❌ Edit recordings to remove mistakes (shows authenticity)
- ❌ Rush through workflows (let progress indicators complete)

### File Naming Convention

```
docs/USER_SCENARIOS/recordings/
├── 01-solo-researcher/
│   ├── cli-init-wizard.cast           # prism init workflow
│   ├── cli-first-workspace.cast       # Complete launch + connect
│   ├── cli-daily-operations.cast      # list, stop, hibernate
│   └── cli-cost-tracking.cast         # project costs
├── 02-lab-environment/
│   ├── cli-project-setup.cast         # Lab project creation
│   ├── cli-member-management.cast     # Adding lab members
│   └── cli-workspace-visibility.cast  # Multi-user workspace list
└── ...
```

---

## 🎯 Persona-by-Persona Recording Plan

### Phase 1: Core Workflows (Solo Researcher) - 3 recordings

**Priority**: 🔴 **HIGHEST** (validates approach before scaling)

#### Recording 1: `prism init` Wizard (45 seconds)
**File**: `recordings/01-solo-researcher/cli-init-wizard.cast`
**Location**: After "Initial Setup" section in walkthrough
**Demonstrates**:
- Interactive wizard with template selection
- Size configuration with cost estimates
- Launch progress with real-time timing
- Connection details on success

**Script**:
```bash
$ prism init

🎯 Prism - Quick Start Wizard
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 Step 1: Select Template
  1) Bioinformatics Suite
  2) Python Machine Learning
  3) R Research Environment
  ...
Choose template [1-8]: 1

Workspace name (default: my-workspace-1027): sarahs-rnaseq

Choose workspace size:
  1) S - 2 vCPU, 4GB RAM (~$0.08/hour)
  2) M - 4 vCPU, 8GB RAM (~$0.16/hour) ← Recommended
  ...
Select size [1-4]: 2

📋 Step 3: Review Configuration
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Your workspace configuration:
  Template:  Bioinformatics Suite
  Name:      sarahs-rnaseq
  Size:      M

  Estimated cost: $0.16/hour (~$3.84/day if running 24/7)

💡 Tip: Use 'prism stop' when not in use to save costs

Launch this workspace? [y/N]: y

🚀 Step 4: Launching Workspace
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⏳ Launching workspace... This may take 1-2 minutes
✅ Workspace launching successfully
⏳ Installing conda packages...
✅ Workspace ready!

🔗 Connection Details:
  ssh researcher@54.123.45.67

💡 Tip: Run 'prism connect sarahs-rnaseq' to connect automatically

Total time: 28 seconds
```

#### Recording 2: Daily Operations (60 seconds)
**File**: `recordings/01-solo-researcher/cli-daily-operations.cast`
**Location**: "Daily Work" section in walkthrough
**Demonstrates**:
- `prism list` with multiple workspaces
- `prism connect` automatic SSH
- `prism stop` cost savings
- `prism start` resume

**Script**:
```bash
$ prism list
┌──────────────────┬─────────┬──────────────┬──────────┬─────────────┐
│ Name             │ State   │ Template     │ Cost/day │ Running     │
├──────────────────┼─────────┼──────────────┼──────────┼─────────────┤
│ sarahs-rnaseq    │ running │ bioinfo      │ $3.84    │ 2h 15m      │
│ protein-fold     │ stopped │ python-ml    │ $0.00    │ -           │
└──────────────────┴─────────┴──────────────┴──────────┴─────────────┘

Total running cost: $3.84/day

$ prism stop sarahs-rnaseq
⏳ Stopping workspace sarahs-rnaseq...
✅ Workspace stopped

💡 Cost savings: ~$3.84/day (~$115/month) when stopped

$ prism list
┌──────────────────┬─────────┬──────────────┬──────────┬─────────────┐
│ Name             │ State   │ Template     │ Cost/day │ Running     │
├──────────────────┼─────────┼──────────────┼──────────┼─────────────┤
│ sarahs-rnaseq    │ stopped │ bioinfo      │ $0.00    │ -           │
│ protein-fold     │ stopped │ python-ml    │ $0.00    │ -           │
└──────────────────┴─────────┴──────────────┴──────────┴─────────────┘

Total running cost: $0.00/day
```

#### Recording 3: Cost Tracking (30 seconds)
**File**: `recordings/01-solo-researcher/cli-cost-tracking.cast`
**Location**: "Budget Management" section
**Demonstrates**:
- `prism project costs` breakdown
- Daily/monthly estimates
- Storage costs vs compute costs

### Phase 2: Lab Environment - 2 recordings

#### Recording 1: Multi-User Workspace List (45 seconds)
**File**: `recordings/02-lab-environment/cli-workspace-visibility.cast`
**Demonstrates**:
- 8 concurrent workspaces visible
- Per-student cost attribution
- Lab-wide cost totals

**Script**:
```bash
$ prism list --project nih-r01
┌────────────────────────┬─────────┬──────────┬───────────┐
│ Workspace              │ Owner   │ Cost/day │ State     │
├────────────────────────┼─────────┼──────────┼───────────┤
│ james-wilson-rnaseq    │ jwilson │ $2.40    │ running   │
│ maria-garcia-gpu-train │ mgarcia │ $24.80   │ running   │
│ alex-chen-pipeline     │ achen   │ $3.20    │ running   │
│ sofia-rodriguez-qc     │ srodrig │ $1.60    │ stopped   │
│ raj-patel-annotation   │ rpatel  │ $2.40    │ hibernated│
│ emily-kim-assembly     │ ekim    │ $3.20    │ running   │
│ carlos-santos-align    │ csantos │ $2.40    │ running   │
│ lisa-wang-viz          │ lwang   │ $1.60    │ stopped   │
└────────────────────────┴─────────┴──────────┴───────────┘

Lab total: $41.60/day (5 running, 2 stopped, 1 hibernated)
NIH R01 budget: $2,000/month
Current spend: $1,248/month (62% of budget)
```

#### Recording 2: Project Member Management (60 seconds)
**File**: `recordings/02-lab-environment/cli-member-management.cast`
**Demonstrates**:
- Adding lab members
- Setting roles (owner, admin, member)
- Permission inheritance

### Phase 3: University Class - 2 recordings

#### Recording 1: Bulk Workspace Operations (90 seconds)
**File**: `recordings/03-university-class/cli-bulk-operations.cast`
**Demonstrates**:
- 120-workspace list with pagination
- Section-based filtering (`--section A`)
- Bulk stop operation (`prism stop --all --section A`)

### Phase 4: Conference Workshop - 1 recording

#### Recording 1: Rapid Provisioning (120 seconds)
**File**: `recordings/04-conference-workshop/cli-workshop-launch.cast`
**Demonstrates**:
- Template selection for 50 participants
- Auto-termination scheduling
- Fixed budget monitoring during launch

### Phase 5: Cross-Institutional - 1 recording

#### Recording 1: Multi-Institution Visibility (60 seconds)
**File**: `recordings/05-cross-institutional/cli-consortium-workspaces.cast`
**Demonstrates**:
- Institution-tagged workspace list
- Cross-account visibility
- Subaward budget tracking

---

## 🛠️ Tooling & Infrastructure Setup

### Step 1: Install asciinema

```bash
# macOS
brew install asciinema

# Linux
apt-get install asciinema  # Debian/Ubuntu
yum install asciinema      # RHEL/CentOS

# Verify installation
asciinema --version
```

### Step 2: Configure Recording Environment

```bash
# Create recordings directory
mkdir -p docs/USER_SCENARIOS/recordings/{01-solo-researcher,02-lab-environment,03-university-class,04-conference-workshop,05-cross-institutional}

# Set terminal to optimal recording settings
# iTerm2/Terminal: Preferences → Profiles → Window
# - Columns: 120
# - Rows: 30
# - Font: Menlo 14pt
```

### Step 3: Recording Workflow

```bash
# Start recording
asciinema rec docs/USER_SCENARIOS/recordings/01-solo-researcher/cli-init-wizard.cast

# Perform workflow (type slowly, let progress complete)
prism init
# ... interactive workflow ...

# Stop recording (Ctrl+D or exit)

# Review recording
asciinema play docs/USER_SCENARIOS/recordings/01-solo-researcher/cli-init-wizard.cast

# Edit if needed (trim, add pauses)
# .cast files are JSON, can be edited manually
```

### Step 4: Self-Hosted asciinema Player

**Option A: GitHub Pages Integration**
```html
<!-- docs/index.html -->
<script src="https://asciinema.org/a/example.js" id="asciicast-example" async></script>
```

**Option B: Self-Hosted Player** (preferred)
```bash
# Install asciinema-player locally
npm install asciinema-player

# Embed in documentation
```

```html
<link rel="stylesheet" type="text/css" href="/asciinema-player.css" />
<div id="demo"></div>
<script src="/asciinema-player.min.js"></script>
<script>
  AsciinemaPlayer.create('/recordings/01-solo-researcher/cli-init-wizard.cast',
    document.getElementById('demo'),
    { cols: 120, rows: 30, autoPlay: true }
  );
</script>
```

### Step 5: GIF Conversion (Optional)

For GitHub README and quick previews:

```bash
# Install agg (asciinema-to-gif converter)
cargo install agg

# Convert .cast → .gif
agg docs/USER_SCENARIOS/recordings/01-solo-researcher/cli-init-wizard.cast \
    docs/USER_SCENARIOS/recordings/01-solo-researcher/cli-init-wizard.gif \
    --speed 1.5 --font-size 14
```

---

## 📊 Success Metrics

### Quantitative Metrics

**Engagement**:
- 📈 **Target**: 80% of Solo Researcher walkthrough readers watch at least 1 CLI recording
- 📈 **Target**: Average 2.5 recordings watched per persona walkthrough reader

**User Confidence**:
- 📉 **Target**: Reduce "how do I...?" support tickets by 40%
- 📉 **Target**: Reduce time-to-first-workspace by 50% (visual confirmation workflow works)

**Documentation Completeness**:
- ✅ **Target**: All 5 basic personas have ≥2 CLI recordings
- ✅ **Target**: 3 compliance personas get recordings when GUI features available (v0.6.0+)

### Qualitative Metrics

**User Feedback**:
- "I could see exactly what to expect" - timing confidence
- "The progress indicators matched the recording" - authenticity
- "I learned faster watching than reading" - visual learning

**Evaluator Feedback**:
- "Professional documentation quality" - institutional confidence
- "Demonstrates real performance claims" - credibility
- "Clear differentiation between personas" - use case clarity

---

## 📅 Implementation Timeline

### Week 1: Infrastructure & Validation
- ✅ Day 1-2: Install asciinema, configure recording environment
- ✅ Day 3-4: Record 3 Solo Researcher workflows (Phase 1)
- ✅ Day 5: Integrate into Solo Researcher walkthrough, get user feedback

### Week 2: Scale to Lab Environment
- Day 1-2: Record 2 Lab Environment workflows (Phase 2)
- Day 3: Integrate and validate persona-specific value
- Day 4-5: Refine recording standards based on feedback

### Week 3: University Class & Conference Workshop
- Day 1-2: Record University Class bulk operations (Phase 3)
- Day 3-4: Record Conference Workshop rapid provisioning (Phase 4)
- Day 5: Cross-check consistency across 4 personas

### Week 4: Cross-Institutional & Polish
- Day 1-2: Record Cross-Institutional collaboration (Phase 5)
- Day 3-4: GIF conversion for GitHub README highlights
- Day 5: Final documentation review and success metrics collection

---

## 🚀 Quick Start (Next Actions)

### Immediate Next Steps

1. **Install asciinema** (5 minutes):
   ```bash
   brew install asciinema
   asciinema --version
   ```

2. **Configure terminal** (5 minutes):
   - iTerm2: Preferences → Profiles → Window → 120×30
   - Font: Menlo 14pt
   - Theme: Light background

3. **Record first workflow** (15 minutes):
   ```bash
   asciinema rec docs/USER_SCENARIOS/recordings/01-solo-researcher/cli-init-wizard.cast
   # Run: prism init workflow
   # Ctrl+D to stop
   ```

4. **Review and iterate** (10 minutes):
   ```bash
   asciinema play docs/USER_SCENARIOS/recordings/01-solo-researcher/cli-init-wizard.cast
   # If satisfied, integrate into walkthrough
   # If not, delete and re-record
   ```

5. **Integrate into documentation** (10 minutes):
   - Add recording link to Solo Researcher walkthrough
   - Test playback in browser
   - Commit and get feedback

---

## 📝 Notes & Considerations

### Recording Authenticity vs. Polish

**Decision**: Prioritize authenticity over perfection
- Show real timing (don't speed up or edit heavily)
- Include natural pauses (demonstrates real workflow)
- Minor typos OK (shows human user, not scripted demo)

### Self-Hosted vs. asciinema.org

**Recommendation**: Self-hosted for production
- No external dependencies
- Better performance (local files)
- No privacy concerns with asciinema.org service
- Git-friendly (track changes to .cast files)

**Alternative**: asciinema.org for prototyping
- Faster initial setup
- Built-in sharing and embedding
- Move to self-hosted once validated

### CLI vs. TUI Recordings

**Focus**: CLI workflows (not TUI)
- CLI output is more readable in recordings
- TUI interface updates rapidly (harder to follow in recordings)
- CLI demonstrates "power user" workflows
- TUI better shown through GUI screenshots (already have)

### Maintenance & Updates

**Strategy**: Record once, update rarely
- CLI output is stable across versions
- Only re-record when major UI changes occur
- Version-specific recordings if CLI changes significantly
- Document recording date in filename or metadata

---

**Last Updated**: October 27, 2025
**Status**: ✅ Planning Complete - Ready for Implementation
**Next Steps**: Install asciinema and record Solo Researcher workflows (Phase 1)
