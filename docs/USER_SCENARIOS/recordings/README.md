# CLI Terminal Recordings Guide

**Purpose**: Step-by-step instructions for recording Prism CLI workflows using asciinema

---

## 🎬 Quick Start

### Record Your First Workflow

```bash
# 1. Start recording
asciinema rec docs/USER_SCENARIOS/recordings/01-solo-researcher/cli-init-wizard.cast

# 2. Perform the workflow (type slowly, let progress complete)
prism init
# ... follow interactive prompts ...

# 3. Stop recording
# Press Ctrl+D or type: exit

# 4. Review the recording
asciinema play docs/USER_SCENARIOS/recordings/01-solo-researcher/cli-init-wizard.cast
```

---

## 📋 Solo Researcher Workflows (Test/Validation Phase)

### Recording 1: `prism init` Wizard ⭐ **START HERE**

**File**: `01-solo-researcher/cli-init-wizard.cast`
**Duration**: ~45 seconds
**Purpose**: Demonstrate interactive template selection and workspace launch

**Script to Follow**:
```bash
$ prism init

# Follow the prompts:
# - Template: Select "Bioinformatics Suite" (or python-ml)
# - Name: "sarahs-rnaseq" (persona-appropriate name)
# - Size: "M" (Medium, recommended)
# - Confirm: "y"
# - Wait for launch progress to complete (~30 seconds)
```

**Key Points to Show**:
- ✅ Interactive wizard interface
- ✅ Template selection with descriptions
- ✅ Cost estimates before launch
- ✅ Real-time progress indicators
- ✅ Connection details on success

---

### Recording 2: Daily Operations

**File**: `01-solo-researcher/cli-daily-operations.cast`
**Duration**: ~60 seconds
**Purpose**: Demonstrate common daily workspace management commands

**Script to Follow**:
```bash
# List workspaces
$ prism list

# Connect to workspace
$ prism connect sarahs-rnaseq
# (shows SSH command, then Ctrl+C to cancel actual connection)

# Stop workspace to save costs
$ prism stop sarahs-rnaseq

# List again to show stopped state
$ prism list

# Restart workspace
$ prism start sarahs-rnaseq
```

**Key Points to Show**:
- ✅ Workspace status table with costs
- ✅ Connect command output
- ✅ Cost savings message when stopping
- ✅ State transitions (running → stopped → running)

---

### Recording 3: Cost Tracking

**File**: `01-solo-researcher/cli-cost-tracking.cast`
**Duration**: ~30 seconds
**Purpose**: Demonstrate budget monitoring and cost visibility

**Script to Follow**:
```bash
# Show project costs
$ prism project costs

# Show workspace details with cost breakdown
$ prism list --verbose

# Show storage costs
$ prism storage list
```

**Key Points to Show**:
- ✅ Daily and monthly cost estimates
- ✅ Breakdown by service (compute vs storage)
- ✅ Budget tracking (if project configured)

---

## 🎯 Recording Standards

### Terminal Configuration

**Required Settings** (for consistency):
```
Terminal:    iTerm2 or macOS Terminal
Font:        Menlo 14pt (or Monaco 14pt)
Window Size: 120 columns × 30 rows
Theme:       Light background (better readability)
Shell:       Bash or Zsh (default)
```

**Set Window Size**:
```bash
# iTerm2: Cmd+, → Profiles → Window → Columns: 120, Rows: 30
# Terminal: Preferences → Profiles → Window → Columns: 120, Rows: 30
```

### Recording Guidelines

**DO ✅**:
- Type at normal conversational speed (not too fast)
- Let progress indicators complete fully
- Use realistic persona names (`sarahs-rnaseq`, not `test123`)
- Show complete workflows start-to-finish
- Include natural pauses (demonstrates real timing)
- Show success cases (happy path workflows)

**DON'T ❌**:
- Don't show AWS credentials or secrets
- Don't edit out minor typos (shows authenticity)
- Don't rush through progress bars
- Don't use `--dry-run` mode (show real launches)
- Don't skip output (let commands complete)

### File Naming Convention

```
01-solo-researcher/
├── cli-init-wizard.cast           # First workspace launch
├── cli-daily-operations.cast      # Common management commands
├── cli-cost-tracking.cast         # Budget monitoring
└── README.md                       # This file (optional per-persona)
```

---

## 🛠️ asciinema Commands Reference

### Recording

```bash
# Start recording
asciinema rec <filename>.cast

# Start with specific terminal size
asciinema rec -c 120 -r 30 <filename>.cast

# Record with idle time limit (auto-pause if no activity)
asciinema rec --idle-time-limit 2 <filename>.cast
```

### Playback

```bash
# Play recording
asciinema play <filename>.cast

# Play at 2x speed
asciinema play -s 2 <filename>.cast

# Play at 0.5x speed (slow motion)
asciinema play -s 0.5 <filename>.cast
```

### Editing

```bash
# .cast files are JSON - can be edited manually
# Example: Trim first 5 seconds
# Edit the .cast file and remove events before 5.0 timestamp
```

### Uploading (Optional)

```bash
# Upload to asciinema.org (for sharing/embedding)
asciinema upload <filename>.cast

# Note: For production, we'll self-host using asciinema-player
```

---

## 📊 Quality Checklist

Before committing a recording, verify:

- [ ] **Duration**: Reasonable length (30-90 seconds for most workflows)
- [ ] **Completeness**: Workflow shows start → finish
- [ ] **Timing**: Progress indicators complete naturally
- [ ] **Visibility**: Text is readable at default font size
- [ ] **Authenticity**: Shows real commands and real timing
- [ ] **Privacy**: No credentials or secrets visible
- [ ] **Context**: Persona-appropriate names and scenarios

---

## 🔄 Re-recording

If you need to re-record:

```bash
# Simply record again to overwrite
asciinema rec docs/USER_SCENARIOS/recordings/01-solo-researcher/cli-init-wizard.cast

# Or delete first
rm docs/USER_SCENARIOS/recordings/01-solo-researcher/cli-init-wizard.cast
asciinema rec docs/USER_SCENARIOS/recordings/01-solo-researcher/cli-init-wizard.cast
```

---

## 📝 Integration into Documentation

Once recordings are complete, integrate into the Solo Researcher walkthrough:

### Markdown Pattern

```markdown
## 🚀 30-Second First Workspace Launch

**Watch It In Action**:

<script id="asciicast-cli-init-wizard"
  src="https://asciinema.org/a/example.js"
  async>
</script>

*Recording shows the complete `prism init` workflow from scratch to running
workspace in 30 seconds, including template selection, size configuration,
and launch progress with real timing.*

**Alternative: GUI Quick Start Wizard**:

![GUI Quick Start](images/01-solo-researcher/gui-quick-start-wizard.png)

**Copy-Paste Commands** (for power users):
```bash
prism init
```
```

---

## 🚀 Ready to Record!

**Your Next Steps**:

1. **Configure Terminal**: Set to 120×30, Menlo 14pt, light theme
2. **Start Prism Daemon**: Ensure `prism` command is available and daemon running
3. **Record Workflow 1**: `prism init` wizard (validation test)
4. **Review Recording**: `asciinema play cli-init-wizard.cast`
5. **Iterate if Needed**: Re-record until satisfied with timing and flow
6. **Commit**: Add `.cast` files to git and integrate into documentation

**Questions?**: See [VISUAL_DOCUMENTATION_ENHANCEMENT_PLAN.md](../VISUAL_DOCUMENTATION_ENHANCEMENT_PLAN.md) for detailed strategy and best practices.

---

**Last Updated**: October 27, 2025
**Status**: Infrastructure ready, awaiting recordings
**Next**: Record Solo Researcher workflows for validation
