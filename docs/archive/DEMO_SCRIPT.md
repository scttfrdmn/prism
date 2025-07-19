# CloudWorkstation Demo Script

## Demo Overview: "From Zero to Research in Minutes"

**Duration:** 15 minutes
**Target Audience:** Academic researchers who need quick computational environments
**Goal:** Demonstrate how CloudWorkstation saves hours of setup time for research environments

## Setup Requirements

- Ensure AWS credentials are properly configured
- Pre-create templates with dependencies:
  - Base templates: `basic-ubuntu`, `desktop-research`
  - Application templates: `r-research`, `python-ml`, `data-science` 
- Pre-validate all commands to ensure they work
- Use a terminal with large, readable font (increase size)
- Prepare backup screenshots in case of connectivity issues

## Demo Flow

### 1. Introduction (2 minutes)

"Today I'm demonstrating CloudWorkstation, a tool designed to help researchers launch pre-configured computing environments in seconds rather than spending hours on setup. CloudWorkstation addresses the common pain points researchers face:

- Hours spent installing and configuring software
- Version conflicts between packages
- Differences between local development and production environments
- Limited access to specialized hardware like GPUs

With CloudWorkstation, you can go from zero to a fully functioning research environment with a single command. Let's see it in action."

### 2. Basic Usage (3 minutes)

"Let's start with the most basic use case. Imagine you're an R researcher who needs to analyze some data:"

```bash
cws launch r-research my-analysis
```

"This single command is doing several things:
- Finding the optimal instance type for R workloads (memory-optimized by default)
- Launching a pre-configured environment with R and RStudio Server
- Setting up common R packages for statistics and visualization
- Configuring security and networking automatically

[Show the launch progress]

"Notice how CloudWorkstation provides clear progress updates and cost estimates. This follows our design principle of 'Zero Surprises' - you always know exactly what's happening."

[Show the connection information]

"Now we have a fully configured R environment ready for work. If I open this URL, we get RStudio Server ready to use. No installation, no configuration hassles, just a working environment optimized for R research."

### 3. Template System (3 minutes)

"CloudWorkstation's power comes from its template system. Let's see what templates are available:"

```bash
cws ami template list
```

"These templates cover common research workflows. Let's look at the details of one:"

```bash
cws ami template info python-ml
```

"Each template has a semantic version number where:
- Major version changes (2.0.0) indicate breaking changes
- Minor version changes (1.1.0) add new features
- Patch version changes (1.0.1) fix bugs

We can compare versions to understand differences:"

```bash
cws ami template version compare 1.0.0 2.0.0
```

"This helps researchers understand what's changed between versions and make informed choices about which version to use."

### 4. Dependency Resolution (4 minutes)

"Templates in CloudWorkstation can depend on other templates. For example, the data-science template builds on both Python and R templates. Let's visualize these dependencies:"

```bash
cws ami template dependency graph data-science
```

"This shows the build order and relationships. Now, let's analyze the dependencies:"

```bash
cws ami template dependency analyze data-science
```

"We can see which dependencies are satisfied, which are missing, and if there are version conflicts."

"If we need to resolve missing dependencies:"

```bash
cws ami template dependency resolve data-science --fetch
```

"CloudWorkstation automatically fetches missing dependencies from the registry and resolves version conflicts to ensure everything works together correctly."

"This dependency system is what allows templates to be modular and composable while still ensuring everything works reliably together."

### 5. Launch with Advanced Options (2 minutes)

"While CloudWorkstation provides smart defaults, researchers can customize their environment. For example:"

```bash
cws launch data-science my-project --size L --region us-west-2 --spot
```

"The `--size L` flag requests a larger instance, suitable for bigger datasets."
"The `--region` flag allows selecting a specific AWS region, helpful when you need to be close to your data."
"The `--spot` flag uses AWS Spot instances for cost savings."

"Notice the transparent feedback about costs and any automatic fallbacks. For instance, if ARM instances aren't available in the selected region, CloudWorkstation will transparently fall back to x86 while explaining why."

### 6. Template Customization (Optional - 2 minutes)

"Researchers can also customize templates for their specific needs:"

```bash
cws ami template create my-custom-env --base python-ml
```

"This creates a new template based on the python-ml template. You can then add your specific packages and configurations."

"Templates can be shared with colleagues or the wider research community:"

```bash
cws ami template share my-custom-env
```

"This enables research reproducibility as others can use the exact same environment."

### 7. Closing (1 minute)

"As you've seen, CloudWorkstation can save researchers hours or even days of environment setup time while ensuring reproducible research environments. It follows key design principles:

- Default to Success: Templates just work out of the box
- Optimize by Default: Best instance types are pre-selected for each workload
- Transparent Fallbacks: Clear communication when the ideal isn't available
- Zero Surprises: Always know what you're getting and how much it costs
- Progressive Disclosure: Simple by default, detailed when needed

We're currently looking for research teams to pilot CloudWorkstation. If you're interested, please contact us to join the pilot program."

## Demo Tips

- When showing launch progress, emphasize how CloudWorkstation is handling the complexity
- Have a pre-launched instance ready as a backup
- Be prepared to explain how template versioning ensures reproducibility
- Highlight the cost transparency throughout the demo
- Emphasize how the dependency resolution system saves time troubleshooting conflicts

## Potential Questions and Answers

**Q: How much does CloudWorkstation cost?**
A: CloudWorkstation itself is open source and free. You pay only for the AWS resources you use, and we show clear cost estimates before launching.

**Q: Can I use my existing AWS account?**
A: Yes, CloudWorkstation uses your existing AWS credentials and resources.

**Q: What happens if I lose my internet connection to the workstation?**
A: Your work continues running on AWS. You can reconnect later and resume exactly where you left off.

**Q: Can I customize templates beyond the provided options?**
A: Absolutely. You can create custom templates, modify existing ones, and share them with colleagues.

**Q: How does CloudWorkstation compare to services like SageMaker?**
A: CloudWorkstation is more flexible, supports more research workflows beyond ML, is designed specifically for academic researchers, and provides optimized templates for various disciplines.

## Backup Plan

If the live demo experiences issues:

1. Switch to pre-recorded video demonstrating the same workflow
2. Use screenshots of each step in the presentation
3. Explain what would be happening at each step
4. Focus more on the architecture and benefits