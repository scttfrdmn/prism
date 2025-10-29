# Release Plan: v0.5.8 - Quick Start Experience

**Release Version**: v0.5.8
**Release Date**: Target December 13, 2025
**Milestone**: [Phase 5.0: UX Redesign](https://github.com/scttfrdmn/prism/milestone/23)
**Status**: 🟡 IN PROGRESS (1 open issue remaining)

---

## 🎯 Release Theme: Quick Start Experience

v0.5.8 focuses on dramatically reducing the time and complexity for new researchers to launch their first workspace, transforming the initial user experience from confusing to effortless.

### Success Metrics

| Metric | Current (v0.5.7) | Target (v0.5.8) | Status |
|--------|------------------|-----------------|--------|
| ⏱️ Time to first workspace | 15 minutes | < 30 seconds | 🎯 Target |
| 🎯 First-attempt success rate | ~60% | > 90% | 🎯 Target |
| 😃 User confusion rate | ~40% | < 12% (70% reduction) | 🎯 Target |

---

## ✅ Completed Features (2/3)

### #15: Rename "Instances" → "Workspaces" ✅ CLOSED
**Impact**: Remove AWS jargon, use research-friendly terminology
**Status**: **COMPLETE** - All user-facing strings updated

**What Changed**:
- CLI commands: Display strings now show "workspace" (internal code still uses "instance")
- TUI interface: All references updated to "Workspaces"
- GUI interface: Navigation, dialogs, and help text updated
- Documentation: User guides reflect new terminology
- Error messages: Clear, friendly "workspace" language

**Backward Compatibility**: CLI commands remain the same (`prism list`, `prism launch`, etc.)

---

### #17: Add `prism init` Onboarding Wizard ✅ CLOSED
**Impact**: Interactive setup wizard removes configuration barriers
**Status**: **COMPLETE** - Full CLI onboarding implemented

**What Was Implemented**:
1. **AWS Configuration Validation**:
   - Detects existing AWS profiles or guides setup
   - Validates credentials with gentle error messages
   - Region selection with recommendations

2. **Research Area Selection**:
   - Domain-specific recommendations (ML, bioinformatics, data analysis, etc.)
   - Template suggestions based on selection
   - Resource sizing guidance

3. **Budget Configuration** (Optional):
   - Quick budget setup for grant-funded research
   - Cost estimation and recommendations
   - Optional hibernation policy configuration

4. **First Workspace Launch**:
   - Guided template selection
   - Pre-filled sensible defaults
   - Progress tracking with clear messaging

**Example Flow**:
```bash
$ prism init

Welcome to Prism! Let's get you set up in 30 seconds.

✓ AWS credentials detected (profile: research-lab)
✓ Region: us-west-2
✓ Research area: Machine Learning
  Recommended template: Python Machine Learning

Ready to launch your first workspace? [Y/n]

🚀 Launching "ml-workspace-1"...
✓ Workspace ready in 22 seconds!

Connect: prism connect ml-workspace-1
```

---

## 🚧 Remaining Work (1/3)

### #13: Home Page with Quick Start Wizard 🟡 IN PROGRESS
**Impact**: Professional first-time user experience in GUI
**Status**: **OPEN** - Requires implementation

**Scope**:
1. **Smart Home Page**:
   - **First-time users**: Quick Start wizard with onboarding flow
   - **Returning users**: Recent workspaces, quick actions, status overview
   - **Context-aware**: Different layouts based on user experience level

2. **Quick Start Wizard**:
   - Step-by-step guided flow
   - Template selection with visual previews
   - Resource configuration with smart defaults
   - Budget setup integration
   - Progress tracking with clear feedback

3. **Template Selection Interface**:
   - Visual cards with descriptions
   - Category filtering (ML, Data Science, Bioinformatics, etc.)
   - Complexity badges (Beginner, Intermediate, Advanced)
   - Cost estimates for each template

4. **Guided Configuration**:
   - Interactive form with helpful tooltips
   - Real-time validation and feedback
   - Pre-filled sensible defaults
   - Advanced options collapsed by default (Progressive Disclosure)

**Implementation Requirements**:
- Frontend: React + Cloudscape Design System components
- Integration: Use existing `prism launch` backend logic
- State management: User preferences and onboarding status
- Responsive design: Works on various screen sizes

**Estimated Effort**: 2-3 days
- Day 1: Home page layout and routing logic
- Day 2: Quick Start wizard implementation
- Day 3: Integration, testing, and polish

---

## 📋 Release Checklist

### Pre-Release
- [x] ~~Milestone cleanup: Remove unrelated issues from v0.5.8~~
- [x] ~~Ensure core features (#13, #15, #17) in correct milestone~~
- [x] ~~Create release plan document~~
- [ ] Complete #13 implementation
- [ ] Update CHANGELOG.md with v0.5.8 changes
- [ ] Update user guides with new terminology

### Testing
- [ ] CLI `prism init` wizard testing (various scenarios)
- [ ] GUI Quick Start wizard end-to-end testing
- [ ] Terminology consistency check (all "workspace" references)
- [ ] First-time user experience validation
- [ ] Success metrics verification

### Documentation
- [ ] Update CLI documentation with `prism init` guide
- [ ] Update GUI documentation with Quick Start wizard screenshots
- [ ] Create "Getting Started in 30 Seconds" tutorial
- [ ] Update terminology in all user guides

### Build & Release
- [ ] Version bump to v0.5.8 in `pkg/version/version.go`
- [ ] Version bump in `cmd/prism-gui/frontend/package.json`
- [ ] Run full build: `make build`
- [ ] Run tests: `make test`
- [ ] Create git tag: `git tag -a v0.5.8 -m "Release v0.5.8: Quick Start Experience"`
- [ ] Run GoReleaser: `goreleaser release --clean`
- [ ] Verify GitHub release created with all artifacts
- [ ] Test installation from Homebrew and Scoop
- [ ] Verify download links and checksums

### Post-Release
- [ ] Announce release in project channels
- [ ] Monitor for user feedback on Quick Start experience
- [ ] Track success metrics (time to first workspace, success rate)
- [ ] Create v0.5.9 milestone and plan next release

---

## 🎓 User Impact

### Before v0.5.8 (Current State)
New researchers face significant friction:
1. **AWS Terminology Barrier**: "Instances" and "EC2" confuse non-cloud users
2. **Configuration Complexity**: Manual AWS setup with unclear steps
3. **No Guidance**: Users left to figure out templates, sizing, and configuration
4. **Time to First Workspace**: 15+ minutes of confusion and trial-and-error

### After v0.5.8 (Target State)
Streamlined onboarding experience:
1. **Clear Language**: "Workspaces" aligns with research mental models
2. **Guided Setup**: `prism init` and GUI wizard handle configuration automatically
3. **Smart Defaults**: Templates pre-configured for research workflows
4. **30-Second Launch**: From zero to working workspace in < 30 seconds

### Example User Journey (v0.5.8)

```
New researcher (Dr. Sarah) needs ML environment:

BEFORE v0.5.8:
1. Install Prism
2. Read AWS setup documentation
3. Configure AWS credentials manually
4. Discover region concepts
5. Browse template list (confused by options)
6. Trial-and-error with launch command
7. Wait for provisioning (unsure if working)
8. Finally connect to workspace
⏱️ Total time: 15+ minutes, multiple failures

AFTER v0.5.8:
1. Install Prism
2. Run: `prism init`
3. Follow guided wizard (4 questions)
4. Confirm launch
5. Connect to workspace
⏱️ Total time: < 30 seconds, zero failures
```

---

## 🔄 Backward Compatibility

**100% Backward Compatible**

All existing CLI commands and API endpoints remain unchanged:
- `prism launch` - Still works (displays "workspace" in output)
- `prism list` - Still works (shows "workspaces" table)
- `prism connect` - Still works
- All project, budget, and storage commands - Unchanged

**Internal Code**: Variable names remain as `instance` to maintain consistency with AWS terminology in codebase.

---

## 📊 Release Statistics

- **Total Issues**: 3 (#13, #15, #17)
- **Closed Issues**: 2 (#15, #17)
- **Open Issues**: 1 (#13)
- **Completion**: 66.7%

**Development Timeline**:
- Week 1: Completed #15 (Terminology rename)
- Week 2: Completed #17 (CLI init wizard)
- Week 3: Target completion of #13 (GUI Quick Start)

---

## 🚀 Next Release: v0.5.9

**Theme**: Navigation Restructure
**Target**: January 3, 2026

Key features:
- Merge Terminal/WebView into Workspaces (Issue #14)
- Collapse Advanced Features under Settings (Issue #16)
- Unified Storage UI for EFS + EBS (Issue #18)

**Goal**: Reduce navigation complexity from 14 → 6 top-level items

---

## 📝 Notes

- v0.5.8 is the first release in Phase 5.0 (UX Redesign)
- Prioritizes usability and clarity over new features
- Addresses #1 pain point: time to first workspace
- Sets foundation for v0.5.9 navigation improvements

**Created**: 2025-10-28
**Last Updated**: 2025-10-28
**Owner**: Development Team
