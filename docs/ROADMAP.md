# CloudWorkstation Roadmap

**Current Version**: v0.5.5
**Last Updated**: October 19, 2025

This document tracks CloudWorkstation's development roadmap, current status, and upcoming priorities.

---

## 🎯 Guiding Principle: Persona-Driven Development

All feature development is validated against [5 persona walkthroughs](USER_SCENARIOS/) representing real research workflows:

1. **Solo Researcher** - Individual research projects
2. **Lab Environment** - Team collaboration
3. **University Class** - Teaching & coursework
4. **Conference Workshop** - Workshops & tutorials
5. **Cross-Institutional Collaboration** - Multi-institution projects

**Before implementing any feature**: Does it clearly improve one of these workflows?

---

## ✅ Completed Phases

### Phase 1: Distributed Architecture ✅ **COMPLETE**
- Daemon + CLI client architecture
- REST API on port 8947
- Multi-client support (CLI, TUI, GUI can connect simultaneously)

### Phase 2: Multi-Modal Access ✅ **COMPLETE**
- CLI (command-line power users)
- TUI (terminal interface with BubbleTea)
- GUI (desktop app with Wails v3 + Cloudscape)
- Feature parity across all interfaces

### Phase 3: Cost Optimization ✅ **COMPLETE**
- Hibernation system (manual + automated)
- Idle detection with configurable policies
- Spot instance support
- Cost estimation and tracking

### Phase 4: Enterprise Features ✅ **COMPLETE**
- Project-based organization
- Budget management with alerts
- Multi-user collaboration
- Role-based access control (Owner/Admin/Member/Viewer)
- Cost analytics and reporting

### Phase 4.6: Professional GUI ✅ **COMPLETE** (September 2025)
- Cloudscape Design System migration
- 60+ AWS-native components
- Professional template selection
- Enterprise-grade instance management
- WCAG AA accessibility

### Phase 5A: Multi-User Foundation ✅ **COMPLETE** (September 2025)
- Dual user system (system users + research users)
- UID/GID consistency across instances
- SSH key management (Ed25519 + RSA)
- EFS home directory integration
- CLI/TUI research user management
- Policy framework foundation

### Phase 5B: Template Marketplace ✅ **COMPLETE** (October 2025)
- Multi-registry system with authentication
- Template discovery and search
- Security validation and quality analysis
- Ratings, badges, and verification
- Dependency management

---

## 🚀 Current Focus: Phase 5.0 - UX Redesign

**Status**: 🟡 **PLANNING** (v0.5.6 - Q4 2025)
**Priority**: 🔴 **CRITICAL - HIGHEST PRIORITY**

### Why UX is Priority #1

Based on [Expert UX Evaluation](architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md):

**Current Problems**:
- ❌ 15-minute learning curve for first workspace (should be 30 seconds)
- ❌ 14 flat navigation items (cognitive overload)
- ❌ Advanced features (AMI, Rightsizing) compete with basic workflows
- ❌ No clear starting point ("what do I do first?")
- ❌ Storage confusion (EFS vs EBS, "volume" vs "storage")

**Impact on Personas**:
- Solo Researcher spends 15min before first launch
- Lab PI can't find budget management
- IT Admin can't discover policy features
- All personas face unnecessary complexity

### Phase 5.0.1: Quick Wins (2 weeks) - **IN PROGRESS**

High-impact, low-effort improvements:

- [ ] **Home Page** with Quick Start wizard
  - First-time users: Quick Start guide
  - Returning users: Recent activity + recommended actions
  - Context-aware: Show relevant features based on user state
  - **Impact**: 90% reduction in "what do I do first?" confusion

- [ ] **Merge Terminal/WebView** into Workspaces
  - Remove from navigation (not destinations)
  - Add as contextual dropdown on each workspace
  - Support multiple terminals simultaneously
  - **Impact**: 14% navigation complexity reduction

- [ ] **Rename "Instances" → "Workspaces"**
  - Researcher-friendly terminology
  - Update CLI, GUI, TUI, and all documentation
  - **Impact**: Clearer mental model for non-technical users

- [ ] **Collapse Advanced Features**
  - Move AMI, Rightsizing, Idle Detection under Settings > Advanced
  - Collapsed by default (expand when needed)
  - **Impact**: 64% reduction in cognitive load

- [ ] **Add `cws init` Wizard**
  - Interactive first-time setup (AWS, budget, hibernation, templates)
  - Learns user's research area for recommendations
  - **Impact**: 15min → 2min onboarding (87% improvement)

**Timeline**: 2 weeks
**Success Metric**: Time to first workspace reduced from 15min to 2min

### Phase 5.0.2: Information Architecture (4 weeks) - **PLANNED**

Restructure navigation for task-oriented workflows:

- [ ] **Unified Storage UI**
  - Single "Storage" page with tabs: Shared (EFS) / Private (EBS)
  - Educational tooltips explaining differences
  - Contextual actions for each storage type
  - **Impact**: Eliminates #1 user confusion

- [ ] **Integrate Budgets into Projects**
  - Budget as tab in Project detail view
  - Remove separate "Budget" navigation item
  - Show per-collaborator spending
  - **Impact**: Makes project budgets discoverable

- [ ] **Reorganize Navigation** (14 items → 5 items)
  ```
  Current (14 items):           Proposed (5 items):
  ├─ Dashboard                 ├─ 🏠 Home (smart landing)
  ├─ Templates                 ├─ 🚀 Workspaces (core workflow)
  ├─ Instances                 ├─ 📊 My Work (storage, costs, logs)
  ├─ Terminal                  ├─ 👥 Collaboration (projects, sharing)
  ├─ Web View                  └─ ⚙️  Settings & Admin (advanced features)
  ├─ Storage
  ├─ Projects
  ├─ Users
  ├─ Budget
  ├─ AMI
  ├─ Rightsizing
  ├─ Policy
  ├─ Marketplace
  ├─ Idle Detection
  └─ Logs
      Settings
  ```

- [ ] **Role-Based Visibility**
  - Admin features (Policy, User Management) only visible to admins
  - PI features (Budget Management) only visible to project owners
  - Reduces clutter for solo researchers

- [ ] **Context-Aware Recommendations**
  - Home page analyzes user state
  - Suggests: "Budget at 80%, review spending"
  - Suggests: "Instance hibernated 5 days, delete?"
  - Proactive cost optimization guidance

**Timeline**: 4 weeks
**Success Metric**: Navigation reduced from 14 to 5 items (64% reduction)

### Phase 5.0.3: CLI Consistency (2 weeks) - **PLANNED**

Restructure CLI commands for predictable patterns:

- [ ] **Consistent Command Structure**
  ```
  Current (scattered):         Proposed (grouped):
  ├─ cws hibernate             ├─ cws launch
  ├─ cws start                 ├─ cws connect
  ├─ cws volume                ├─ cws list
  ├─ cws storage               ├─ cws stop
  ├─ cws ami                   ├─ cws delete
  ├─ cws marketplace           │
  ├─ cws research-user         ├─ cws workspace <action>
  ├─ cws idle                  ├─ cws storage <action>
  └─ ...40+ commands           ├─ cws templates <action>
                               ├─ cws collab <action>
                               ├─ cws admin <action>
                               └─ cws config <action>
  ```

- [ ] **Unified Storage Commands**
  - Merge `cws volume` and `cws storage` into `cws storage`
  - Use `--type efs|ebs` flag for clarity
  - Backward compatibility warnings for deprecated commands

- [ ] **Predictable Patterns**
  - Verb-noun-object everywhere: `cws storage create`, `cws workspace hibernate`
  - Consistent help text format
  - Tab completion support

**Timeline**: 2 weeks
**Success Metric**: CLI first-attempt success from 35% to 85% (143% improvement)

---

## 📅 Upcoming Phases (Post-UX Redesign)

### Phase 5.1: Universal AMI System (v0.5.7 - Q1 2026)

**Status**: 🟢 **PLANNED**

- Universal AMI reference in templates
- Auto-compilation based on template popularity
- Cross-region AMI copying
- Performance: 30-second launches vs 5-8 minute provisioning

**Persona Benefit**: All personas get faster workspace launches

### Phase 5.2: Template Marketplace Enhancement (v0.5.8 - Q1 2026)

**Status**: 🟢 **PLANNED** (Foundation complete in Phase 5B)

- Decentralized repository system
- Community + institutional + commercial templates
- Repository authentication (SSH, tokens, OAuth)
- BYOL licensing for commercial software (MATLAB, Stata, etc.)

**Persona Benefit**: University Class, Conference Workshop (community templates)

### Phase 5.3: Configuration & Directory Sync (v0.5.9 - Q2 2026)

**Status**: 🟢 **PLANNED**

- Template-based config sync (RStudio, Jupyter, VS Code)
- EFS bidirectional directory sync
- Conflict resolution for simultaneous edits
- Cross-platform support (macOS, Linux, Windows)

**Persona Benefit**: Solo Researcher, Lab Environment (seamless workflow)

### Phase 5.4: AWS Research Services (v0.5.10 - Q2 2026)

**Status**: 🟢 **PLANNED**

- EMR Studio integration (big data analytics)
- SageMaker Studio Lab (educational ML)
- Amazon Braket (quantum computing)
- Unified web service framework

**Persona Benefit**: Cross-Institutional Collaboration (advanced research tools)

### Phase 6: Extensibility & Ecosystem (v0.6.0 - Q3 2026)

**Status**: 🟢 **PLANNED**

- Plugin architecture (CLI + daemon)
- Auto-AMI system with popularity-driven compilation
- GUI skinning & institutional branding
- Web services integration framework

**Persona Benefit**: University Class, IT Admins (institutional customization)

---

## 🎯 Success Metrics

### Current State (Before Phase 5.0)
- ⏱️ Time to first workspace: **15 minutes**
- 🧭 Navigation complexity: **14 items**
- 🎯 CLI first-attempt success: **35%**
- 😕 User confusion rate: **40% of support tickets**
- 🔧 Advanced feature discovery: **<5% use AMI/Rightsizing**

### Target State (After Phase 5.0)
- ⏱️ Time to first workspace: **2 minutes** (87% improvement)
- 🧭 Navigation complexity: **5 items** (64% reduction)
- 🎯 CLI first-attempt success: **85%** (143% improvement)
- 😃 User confusion rate: **<10% of support tickets** (75% improvement)
- 🔧 Advanced feature discovery: **Available when needed, not intrusive**

---

## 📊 Development Velocity

**Recent Completions** (September-October 2025):
- ✅ Phase 4.6: Cloudscape GUI migration (2 weeks)
- ✅ Phase 5A: Multi-user foundation (3 weeks)
- ✅ Phase 5B: Template marketplace (2 weeks)

**Average Velocity**: 2-3 weeks per major phase

**Phase 5.0 Estimate**: 8 weeks total
- Quick Wins: 2 weeks
- Information Architecture: 4 weeks
- CLI Consistency: 2 weeks

---

## 🔗 Related Documentation

- [CLAUDE.md](CLAUDE.md) - Development context for AI assistants
- [VISION.md](VISION.md) - Long-term product vision
- [USER_REQUIREMENTS.md](USER_REQUIREMENTS.md) - User research and requirements
- [UX Evaluation](architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md) - Detailed UX analysis
- [Persona Walkthroughs](USER_SCENARIOS/) - Real-world usage scenarios

---

## 💡 How to Use This Roadmap

**For Contributors**:
1. Check **Current Focus** section for highest priority work
2. Validate features against **Persona Walkthroughs**
3. Review **Success Metrics** to understand impact
4. See **Related Documentation** for detailed context

**For Researchers** (Curious about upcoming features):
1. See **Upcoming Phases** for what's coming next
2. Check **Success Metrics** for expected improvements
3. Review **Persona Walkthroughs** to see your workflow represented

**For PIs/Admins** (Planning adoption):
1. **Current Version** shows production-ready features
2. **Phase 5.0** timing indicates when UX improvements arrive
3. **Phase 6** shows institutional customization timeline

---

**Last Updated**: October 19, 2025
**Next Review**: End of Phase 5.0.1 (November 2025)
