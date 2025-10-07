# Page snapshot

```yaml
- main:
  - navigation:
    - button
  - navigation "Breadcrumbs":
    - list:
      - listitem:
        - link "CloudWorkstation":
          - /url: "#/"
      - listitem:
        - link "Research Templates" [disabled]
  - heading "Research Templates" [level=1]
  - combobox "Filter templates"
  - group:
    - text: 🤖
    - heading "Python Machine Learning" [level=3]
    - text: Popular
    - radio
    - text: Complete ML environment with TensorFlow, PyTorch, and Jupyter Machine Learning moderate Pre-tested Launch Time ~2 min Cost $0.48/hour 📊
    - heading "R Research Environment" [level=3]
    - text: Popular
    - radio
    - text: Statistical computing with R, RStudio, and tidyverse packages Data Science simple Pre-tested Launch Time ~3 min Cost $0.24/hour 🖥️
    - heading "Rocky Linux 9 Base" [level=3]
    - radio
    - text: Enterprise Linux foundation for custom research environments Base System simple Pre-tested Launch Time ~1 min Cost $0.12/hour
  - complementary:
    - button
```