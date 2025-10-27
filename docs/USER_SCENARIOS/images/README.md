# Persona Walkthrough Screenshots

This directory contains screenshots for the persona walkthroughs, organized by persona number.

## Directory Structure

```
images/
├── 01-solo-researcher/       # Solo researcher screenshots
├── 02-lab-environment/        # Lab environment screenshots
├── 03-university-class/       # University class screenshots
├── 04-conference-workshop/    # Conference workshop screenshots
├── 05-cross-institutional/    # Cross-institutional screenshots
├── 06-nih-cui/                # NIH CUI compliance screenshots
├── 07-nih-hipaa/              # NIH HIPAA compliance screenshots
└── 08-institutional-it/       # Institutional IT screenshots
```

## Screenshot Guidelines

### Naming Convention
Use descriptive names that indicate what's shown:
- `cli-init-wizard-step1.png` - CLI command screenshots
- `gui-template-gallery.png` - GUI interface screenshots
- `tui-dashboard.png` - TUI interface screenshots

### Technical Requirements
- **Format**: PNG (lossless)
- **Size**: Target <500KB per image
- **Resolution**: 1920x1080 for GUI, 120x40 for terminal
- **Optimization**: Use ImageOptim or similar before committing

### Content Guidelines
- Use realistic data (e.g., "cancer-research" not "test-123")
- Show v0.5.8 "workspace" terminology
- Include relevant UI context (menus, tabs, status bars)
- Ensure text is readable at 800px width

## Adding Screenshots to Persona Documents

### Basic Image
```markdown
![CLI Quick Start Wizard](images/01-solo-researcher/cli-init-wizard.png)
```

### Image with Caption
```markdown
<p align="center">
  <img src="images/01-solo-researcher/gui-template-gallery.png" alt="GUI Template Gallery" width="800">
  <br>
  <em>Professional template selection with Cloudscape Cards and Badges</em>
</p>
```

## Capturing Screenshots

### CLI Screenshots (Terminal)
```bash
# macOS built-in (interactive window selection)
screencapture -w -o cli-init-wizard.png

# iTerm2 built-in
# Press ⌘+⇧+S to save screenshot
```

### GUI Screenshots (Prism Desktop App)
```bash
# macOS built-in (window screenshot)
screencapture -w -o gui-template-gallery.png

# Or use ⌘+⇧+4, then press Space to capture window
```

### TUI Screenshots (Terminal)
Same as CLI screenshots above.

## See Also
- [Screenshot Plan](../SCREENSHOT_PLAN.md) - Detailed plan for screenshot capture
- [GUI User Guide](../../user-guides/GUI_USER_GUIDE.md) - GUI documentation
- [TUI User Guide](../../user-guides/TUI_USER_GUIDE.md) - TUI documentation
