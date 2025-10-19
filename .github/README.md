# GitHub Project Management Structure

This directory contains CloudWorkstation's GitHub configuration for project management, issue tracking, and automation.

## 📋 Issue Templates

Located in `.github/ISSUE_TEMPLATE/`:

- **🐛 Bug Report** (`bug_report.yml`) - Report bugs or unexpected behavior
- **✨ Feature Request** (`feature_request.yml`) - Suggest new features
- **🎨 UX Improvement** (`ux_improvement.yml`) - Suggest usability improvements
- **📚 Documentation** (`documentation.yml`) - Report documentation issues
- **🔧 Technical Debt** (`technical_debt.yml`) - Report code that needs refactoring

## 🏷️ Labels

Defined in `.github/labels.yml`:

### Type Labels
- `bug` - Something isn't working
- `enhancement` - New feature or request
- `ux-improvement` - Usability or UX improvement
- `documentation` - Documentation improvements
- `technical-debt` - Code refactoring needed

### Priority Labels
- `priority: critical` - Blocking work or severe impact
- `priority: high` - Should be addressed soon
- `priority: medium` - Important but not urgent
- `priority: low` - Nice to have

### Area Labels
- `area: cli` - Command-line interface
- `area: gui` - Desktop GUI application
- `area: tui` - Terminal interface
- `area: daemon` - Backend daemon
- `area: templates` - Template system
- `area: aws` - AWS integration
- `area: build` - Build system, CI/CD
- `area: tests` - Testing infrastructure

### Persona Labels
- `persona: solo-researcher` - Benefits solo researcher
- `persona: lab-environment` - Benefits lab collaboration
- `persona: university-class` - Benefits teaching
- `persona: conference-workshop` - Benefits workshops
- `persona: cross-institutional` - Benefits multi-institution

### Status Labels
- `triage` - Needs initial review
- `needs-info` - Waiting for more information
- `blocked` - Blocked by dependency
- `ready` - Ready to be worked on
- `in-progress` - Currently being worked on
- `in-review` - In code review
- `awaiting-merge` - Approved and ready

### Resolution Labels
- `duplicate` - Already exists
- `wontfix` - Will not be worked on
- `invalid` - Not applicable
- `works-as-designed` - Behavior is intentional

### Special Labels
- `good first issue` - Good for newcomers
- `help wanted` - Extra attention needed
- `breaking-change` - Breaks backward compatibility
- `security` - Security-related
- `performance` - Performance optimization

### Phase Labels
- `phase: 5.0-ux-redesign` - Part of Phase 5.0 UX Redesign
- `phase: 5.1-universal-ami` - Part of Phase 5.1
- `phase: 5.2-marketplace` - Part of Phase 5.2
- And more...

## 🎯 Setup Instructions

### Initial Setup

Run the setup script to create labels and milestones:

```bash
./scripts/setup-github-project.sh
```

This will:
1. Sync labels from `.github/labels.yml`
2. Create milestones for each development phase
3. Display next steps for manual configuration

### Migrate Roadmap to Issues

Run the migration script to create issues from ROADMAP.md:

```bash
./scripts/create-roadmap-issues.sh
```

This will:
1. Create GitHub issues for Phase 5.0.1 (Quick Wins)
2. Create GitHub issues for Phase 5.0.2 (Information Architecture)
3. Create GitHub issues for Phase 5.0.3 (CLI Consistency)
4. Assign proper labels and milestones

## 📊 GitHub Projects Board

After running the setup scripts, manually create a GitHub Projects board:

1. Go to: https://github.com/scttfrdmn/cloudworkstation/projects
2. Click "New project"
3. Choose "Board" view
4. Add columns:
   - **Backlog** - Triaged but not prioritized
   - **Ready** - Prioritized, ready to work on
   - **In Progress** - Currently being worked on
   - **Review** - In code review
   - **Done** - Completed (auto-archive after 2 weeks)

## 🔄 Workflow

### For Contributors

1. **Find Work**: Browse [GitHub Issues](https://github.com/scttfrdmn/cloudworkstation/issues) or [Projects Board](https://github.com/scttfrdmn/cloudworkstation/projects)
2. **Pick an Issue**: Choose from "Ready" column or issues labeled `good first issue`
3. **Create Branch**: `git checkout -b feature/issue-123-description`
4. **Make Changes**: Implement the feature or fix
5. **Create PR**: Use the pull request template, reference the issue
6. **Review**: Address code review feedback
7. **Merge**: Maintainer merges when approved

### For Maintainers

1. **Triage**: Review new issues, add labels and milestones
2. **Prioritize**: Move issues to "Ready" column when prioritized
3. **Assign**: Assign issues to team members or self
4. **Review**: Review PRs, provide feedback, approve
5. **Merge**: Merge approved PRs, close related issues
6. **Release**: Tag releases when milestones are complete

## 🎯 Best Practices

### Issue Management

- **Be Specific**: Use descriptive titles and detailed descriptions
- **Link to Personas**: Always indicate which persona(s) benefit
- **Add Context**: Include screenshots, logs, command output
- **Break Down Large Issues**: Split epics into smaller, actionable tasks
- **Keep Updated**: Comment with progress updates

### Pull Requests

- **Reference Issues**: Use "Closes #123" or "Relates to #456"
- **Follow Template**: Fill out all sections of the PR template
- **Keep Focused**: One feature or fix per PR
- **Add Tests**: Include tests for new functionality
- **Update Docs**: Update documentation when changing behavior

### Labels

- **Use Multiple Labels**: Combine type, priority, area, and persona labels
- **Update as Needed**: Add `blocked` or `needs-info` when status changes
- **Remove Triage**: Remove `triage` label after initial review

## 📚 Resources

- [GitHub Issues](https://github.com/scttfrdmn/cloudworkstation/issues) - Bug reports, feature requests, tasks
- [GitHub Discussions](https://github.com/scttfrdmn/cloudworkstation/discussions) - Q&A, ideas, show and tell
- [GitHub Projects](https://github.com/scttfrdmn/cloudworkstation/projects) - Development roadmap and sprint planning
- [GitHub Milestones](https://github.com/scttfrdmn/cloudworkstation/milestones) - Phase tracking and progress
- [Persona Walkthroughs](../docs/USER_SCENARIOS/) - Real-world use cases
- [UX Evaluation](../docs/architecture/UX_EVALUATION_AND_RECOMMENDATIONS.md) - Usability improvements
- [Design Principles](../docs/DESIGN_PRINCIPLES.md) - Core philosophy
