# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## 📋 Quick Navigation

**Project Management** (Use GitHub!):
- 🎯 [GitHub Issues](https://github.com/scttfrdmn/prism/issues) - **Current work, bugs, features**
- 📊 [GitHub Projects](https://github.com/scttfrdmn/prism/projects) - **Roadmap and sprint planning**
- 🏁 [GitHub Milestones](https://github.com/scttfrdmn/prism/milestones) - **Phase tracking and progress**

**Essential Reading**:
- 👥 [USER_SCENARIOS/](USER_SCENARIOS/) - **5 persona walkthroughs (our north star)**
- 🎨 [UX Evaluation](architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md) - **Current UX issues and fixes**
- 🏛️ [VISION.md](VISION.md) - Long-term product vision
- 📐 [DESIGN_PRINCIPLES.md](DESIGN_PRINCIPLES.md) - Core design philosophy

**For Implementation**:
- 🏗️ [Architecture Docs](architecture/) - Technical architecture and system design
- 💻 [Development Guides](development/) - Setup, testing, code quality
- 📚 [User Guides](user-guides/) - End-user documentation (validate features against these)
- 👨‍💼 [Admin Guides](admin-guides/) - Administrator and institutional docs

---

## Project Overview

Prism is a command-line tool that provides academic researchers with pre-configured cloud workstations, eliminating the need for manual environment configuration.

**Current Version**: v0.5.7 (Released)
**Current Focus**: [Phase 5.0 UX Redesign - v0.5.8 and v0.5.9](ROADMAP.md#-current-focus-phase-50---ux-redesign) (HIGHEST PRIORITY)

---

## 🎯 Persona-Driven Development (CRITICAL)

Prism's feature development is guided by [5 persona walkthroughs](USER_SCENARIOS/) that represent real-world research scenarios. These scenarios are our **north star** for prioritization and decision-making.

### Before Implementing ANY Feature:

1. **Ask**: "Does this clearly improve one of the 5 persona workflows?"
2. **If yes**: Validate the feature makes the workflow simpler/faster/clearer
3. **If no**: Question whether it's the right priority

### The 5 Personas:

1. **[Solo Researcher](USER_SCENARIOS/01_SOLO_RESEARCHER_WALKTHROUGH.md)** - Individual research projects
2. **[Lab Environment](USER_SCENARIOS/02_LAB_ENVIRONMENT_WALKTHROUGH.md)** - Team collaboration
3. **[University Class](USER_SCENARIOS/03_UNIVERSITY_CLASS_WALKTHROUGH.md)** - Teaching & coursework
4. **[Conference Workshop](USER_SCENARIOS/04_CONFERENCE_WORKSHOP_WALKTHROUGH.md)** - Workshops & tutorials
5. **[Cross-Institutional Collaboration](USER_SCENARIOS/05_CROSS_INSTITUTIONAL_COLLABORATION_WALKTHROUGH.md)** - Multi-institution projects

These walkthroughs prioritize **usability and clarity of use** over technical sophistication. Features that add complexity without clear benefit to these scenarios should be deferred or redesigned.

---

## Core Design Principles

See [DESIGN_PRINCIPLES.md](DESIGN_PRINCIPLES.md) for full details. Key principles:

### 🎯 Default to Success
Every template must work out of the box in every supported region. No configuration required for basic usage.

### ⚡ Optimize by Default
Templates automatically choose the best instance size and type for their intended workload.

### 🔍 Transparent Fallbacks
When ideal configuration isn't available, users always know what changed and why.

### 💡 Helpful Warnings
Gentle guidance when users make suboptimal choices, with clear alternatives offered.

### 🚫 Zero Surprises
Users should never be surprised by what they get - clear communication about what's happening.

### 📈 Progressive Disclosure
Simple by default, detailed when needed. Power users can access advanced features without cluttering basic workflows.

---

## 🚀 Current Development Status

**Current Version**: v0.5.7 (Released)
**Current Milestone**: [Phase 5.0: UX Redesign](https://github.com/scttfrdmn/prism/milestone/1)

### Completed Phases
- ✅ Phase 1: Distributed Architecture
- ✅ Phase 2: Multi-Modal Access (CLI/TUI/GUI)
- ✅ Phase 3: Cost Optimization & Hibernation
- ✅ Phase 4: Enterprise Features (projects, budgets, collaboration)
- ✅ Phase 4.6: Cloudscape GUI Migration
- ✅ Phase 5A: Multi-User Foundation
- ✅ Phase 5B: Template Marketplace
- ✅ v0.5.7: Template Provisioning & Test Infrastructure

### Current Priority: Phase 5.0 UX Redesign

**Status**: 📋 PLANNED (v0.5.8 + v0.5.9 - December 2025 - January 2026)
**Priority**: 🔴 **CRITICAL - HIGHEST PRIORITY**

**Why This is Priority #1**:
- Current: 15-minute learning curve for first workspace (should be 30 seconds)
- Problem: 14 flat navigation items, advanced features too prominent
- Impact: New researchers face cognitive overload before basic tasks
- **Track progress**: [GitHub Milestones](https://github.com/scttfrdmn/prism/milestones)
- **See UX analysis**: [UX Evaluation](architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md)

### v0.5.8: Quick Start Experience (December 2025)
**Release Date**: Target December 13, 2025
**Release Plan**: [RELEASE_PLAN_v0.5.8.md](releases/RELEASE_PLAN_v0.5.8.md)
**Milestone**: [#2](https://github.com/scttfrdmn/prism/milestone/2) | **Status**: 📋 Planned

Features (2 weeks):
- [#15](https://github.com/scttfrdmn/prism/issues/15) - Rename "Instances" → "Workspaces"
- [#13](https://github.com/scttfrdmn/prism/issues/13) - Home Page with Quick Start wizard (GUI)
- [#17](https://github.com/scttfrdmn/prism/issues/17) - CLI `prism init` onboarding wizard

Success Metrics:
- ⏱️ Time to first workspace: 15min → 30sec
- 🎯 First-attempt success rate: >90%
- 😃 User confusion: Reduce by 70%

### v0.5.9: Navigation Restructure (January 2026)
**Release Date**: Target January 3, 2026
**Release Plan**: [RELEASE_PLAN_v0.5.9.md](releases/RELEASE_PLAN_v0.5.9.md)
**Milestone**: [#3](https://github.com/scttfrdmn/prism/milestone/3) | **Status**: 📋 Planned

Features (2 weeks):
- [#14](https://github.com/scttfrdmn/prism/issues/14) - Merge Terminal/WebView into Workspaces
- [#16](https://github.com/scttfrdmn/prism/issues/16) - Collapse Advanced Features under Settings
- [#18](https://github.com/scttfrdmn/prism/issues/18) - Unified Storage UI (EFS + EBS)
- [#19](https://github.com/scttfrdmn/prism/issues/19) - Integrate Budgets into Projects

Success Metrics:
- 🧭 Navigation complexity: 14 → 6 top-level items
- ⏱️ Time to find features: <10 seconds
- 😃 User confusion: Further 30% reduction
- 📱 Advanced feature discoverability: >95%

### v0.6.0: CLI Consistency + Enterprise (February 2026)
**Milestone**: [#4](https://github.com/scttfrdmn/prism/milestone/4) | **Status**: 📋 Planned

Features:
- [#20](https://github.com/scttfrdmn/prism/issues/20) - Consistent CLI Command Structure
- OAuth/OIDC integration for institutional SSO
- LDAP/Active Directory support
- Auto-update feature

---

## 🏗️ Architecture Overview

### Multi-Modal Access Strategy

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│ CLI Client  │  │ TUI Client  │  │ GUI Client  │
│ (cmd/cws)   │  │ (prism tui)   │  │ (cmd/cws-gui)│
└──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │                │                │
       └────────────────┼────────────────┘
                        │
                 ┌─────────────┐
                 │ Backend     │
                 │ Daemon      │
                 │ (cwsd:8947) │
                 └─────────────┘
```

**See** [GUI Architecture](architecture/GUI_ARCHITECTURE.md) for GUI details.

### Directory Structure

```
cmd/
├── cws/          # CLI client binary
├── cws-gui/      # GUI client binary (Wails v3-based)
└── cwsd/         # Backend daemon binary

pkg/
├── api/          # API client interface
├── daemon/       # Daemon core logic
├── aws/          # AWS operations
├── state/        # State management
├── project/      # Project & budget management
├── idle/         # Hibernation & cost optimization
├── profile/      # Enhanced profile system
├── research/     # Research user system (Phase 5A)
└── types/        # Shared types

internal/
├── cli/          # CLI application logic
├── tui/          # TUI application (BubbleTea-based)
└── gui/          # (GUI logic is in cmd/cws-gui/)
```

---

## 🧪 Development Workflow

### Building

```bash
# Build all components
make build

# Build specific components
go build -o bin/cws ./cmd/cws/        # CLI
go build -o bin/cwsd ./cmd/cwsd/      # Daemon
go build -o bin/cws-gui ./cmd/cws-gui/ # GUI

# Run tests
make test
```

### Running

```bash
# CLI interface - daemon auto-starts as needed
./bin/cws launch python-ml my-project

# TUI interface - daemon auto-starts as needed
./bin/cws tui

# GUI interface - daemon auto-starts as needed
./bin/cws-gui

# Manual daemon control (optional)
./bin/cwsd &                    # Start daemon manually
./bin/cws daemon stop           # Stop daemon
./bin/cws daemon status         # Check status
```

**See** [Development Setup](development/DEVELOPMENT_SETUP.md) for detailed setup instructions.

---

## 🧭 Key Implementation Guidelines

### 1. Validate Against Personas
Before implementing features, check if it improves one of the [5 persona workflows](USER_SCENARIOS/).

### 2. Follow Design Principles
See [DESIGN_PRINCIPLES.md](DESIGN_PRINCIPLES.md) - especially "Default to Success" and "Progressive Disclosure".

### 3. Maintain Multi-Modal Parity
Features must work across CLI, TUI, and GUI. See [Feature Parity Matrix](ROADMAP.md).

### 4. Focus on Usability First
Current priority is [Phase 5.0 UX Redesign](ROADMAP.md#-current-focus-phase-50---ux-redesign). Usability improvements take precedence over new features.

### 5. Use Existing Documentation
- Architecture questions: [architecture/](architecture/)
- User workflows: [USER_SCENARIOS/](USER_SCENARIOS/)
- Admin features: [admin-guides/](admin-guides/)
- Development: [development/](development/)

---

## 📚 Essential Documentation Map

**Strategic**:
- [ROADMAP.md](ROADMAP.md) - Current status and priorities
- [VISION.md](VISION.md) - Long-term product vision
- [USER_REQUIREMENTS.md](USER_REQUIREMENTS.md) - User research

**Personas & UX** (Highest Priority):
- [USER_SCENARIOS/](USER_SCENARIOS/) - 5 persona walkthroughs
- [UX Evaluation](architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md) - UX issues and fixes
- [GUI UX Design Review](architecture/GUI_UX_DESIGN_REVIEW.md)

**Architecture**:
- [GUI Architecture](architecture/GUI_ARCHITECTURE.md)
- [Daemon API Reference](architecture/DAEMON_API_REFERENCE.md)
- [Dual User Architecture](architecture/DUAL_USER_ARCHITECTURE.md)
- [Template Marketplace](architecture/TEMPLATE_MARKETPLACE_ARCHITECTURE.md)

**Development**:
- [Development Setup](development/DEVELOPMENT_SETUP.md)
- [Testing Guide](development/TESTING.md)
- [Code Quality](development/CODE_QUALITY_BEST_PRACTICES.md)
- [Release Process](development/RELEASE_PROCESS.md) - Legacy manual process
- [GoReleaser Release Process](development/GORELEASER_RELEASE_PROCESS.md) - **RECOMMENDED**: Automated release process

**User/Admin**:
- [User Guide v0.5.x](user-guides/USER_GUIDE_v0.5.x.md)
- [Administrator Guide](admin-guides/ADMINISTRATOR_GUIDE.md)
- [Troubleshooting](user-guides/TROUBLESHOOTING.md)

---

## 🎯 Quick Reference: Common Tasks

### Adding a New Feature
1. ✅ Does it improve a [persona workflow](USER_SCENARIOS/)?
2. ✅ Does it follow [design principles](DESIGN_PRINCIPLES.md)?
3. ✅ Check [UX evaluation](architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md) - does it address usability issues?
4. ✅ Implement in daemon (pkg/), then expose via API
5. ✅ Add to CLI (internal/cli/), TUI (internal/tui/), GUI (cmd/cws-gui/)
6. ✅ Update [ROADMAP.md](ROADMAP.md) status
7. ✅ Document in appropriate guide ([user-guides/](user-guides/) or [admin-guides/](admin-guides/))

### Fixing UX Issues
1. ✅ Check [UX evaluation](architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md) for prioritized fixes
2. ✅ Verify fix improves [persona workflows](USER_SCENARIOS/)
3. ✅ Update [ROADMAP.md](ROADMAP.md) Phase 5.0 checkboxes
4. ✅ Test against success metrics (time to first workspace, navigation complexity, etc.)

### Understanding Current State
1. ✅ Check [ROADMAP.md](ROADMAP.md) for current phase and status
2. ✅ Review [persona walkthroughs](USER_SCENARIOS/) to understand user needs
3. ✅ Read [UX evaluation](architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md) to understand pain points

### Creating a Release (GoReleaser)
**See [GoReleaser Release Process](development/GORELEASER_RELEASE_PROCESS.md) for complete documentation.**

Quick steps for creating a release:

1. ✅ Ensure version is synchronized (pkg/version/version.go, cmd/prism-gui/frontend/package.json)
2. ✅ Commit and push all changes to main branch
3. ✅ Create annotated git tag:
   ```bash
   git tag -a v0.5.8 -m "Release v0.5.8: [Brief Description]"
   git push origin v0.5.8
   ```
4. ✅ Set GitHub token:
   ```bash
   export GITHUB_TOKEN=$(gh auth token)
   ```
5. ✅ Run GoReleaser:
   ```bash
   goreleaser release --clean
   ```
6. ✅ Verify release: https://github.com/scttfrdmn/prism/releases/tag/v0.5.8

**What GoReleaser Does Automatically:**
- ✅ Builds binaries for all platforms (Linux, macOS, Windows; amd64, arm64)
- ✅ Creates archives (.tar.gz, .zip)
- ✅ Generates Linux packages (deb, rpm, apk)
- ✅ Creates GitHub release with all artifacts
- ✅ Updates Homebrew formula in scttfrdmn/homebrew-tap
- ✅ Updates Scoop manifest in scttfrdmn/scoop-bucket
- ✅ Calculates checksums

**Common Issues:**
- **GITHUB_TOKEN not set**: Run `export GITHUB_TOKEN=$(gh auth token)`
- **Dirty git state**: Commit all changes before running GoReleaser
- **Tag on wrong commit**: Delete and recreate tag on correct commit

---

## 📊 Success Metrics

**See [ROADMAP.md - Success Metrics](ROADMAP.md#-success-metrics) for current vs target state.**

Key metrics we're tracking:
- ⏱️ Time to first workspace launch
- 🧭 Navigation complexity (number of items)
- 🎯 CLI first-attempt success rate
- 😃 User confusion rate (% of support tickets)
- 🔧 Advanced feature discoverability

---

**Last Updated**: October 19, 2025
**Next Review**: End of Phase 5.0.1 (November 2025)

**For detailed roadmap and current priorities, see [ROADMAP.md](ROADMAP.md)**
