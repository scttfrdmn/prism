# Page snapshot

```yaml
- banner:
  - heading "☁️ CloudWorkstation" [level=1]
  - button "🌙"
  - button "⚙️"
- main:
  - heading "Quick Start" [level=2]
  - paragraph: Choose a template and launch your research environment
  - text: Complexity Level
  - button "All"
  - button "🟢 Simple"
  - button "🟡 Moderate"
  - button "🟠 Advanced"
  - button "🔴 Complex"
  - text: Research Domain
  - button "All Domains"
  - button "🤖 ML"
  - button "📊 Data"
  - button "🧬 Bio"
  - button "🌐 Web"
  - button "🖥️ Base"
  - text: Sort By
  - combobox:
    - option "Most Popular" [selected]
    - option "Complexity (Simple → Complex)"
    - option "Research Domain"
    - option "Cost (Low → High)"
    - option "Launch Time (Fast → Slow)"
  - paragraph: Failed to load templates
  - text: Please check if the daemon is running
  - button "Retry"
- navigation:
  - button "🚀 Quick Start"
  - button "💻 My Instances"
  - button "🖥️ Remote Desktop"
- text: Connected to daemon
```