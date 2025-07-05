# CloudWorkstation Demo Walkthrough

This step-by-step guide demonstrates CloudWorkstation's capabilities with detailed explanations of each command and expected outputs. Use this as a reference when practicing the demo.

## Part 1: Basic Launch Workflow

### Command: Checking Available Templates
```bash
cws ami template list
```

**Expected Output:**
```
📋 Available templates:

NAME              VERSION    DESCRIPTION                                         
basic-ubuntu      1.0.0      Base Ubuntu 22.04 template                          
desktop-research  1.0.0      Ubuntu Desktop with research tools                  
r-research        1.0.0      R and RStudio Server with common packages           
python-ml         2.1.1      Python with ML frameworks and Jupyter               
data-science      1.0.0      Complete data science environment with R and Python 
```

**Explanation Points:**
- CloudWorkstation provides a variety of templates for different research needs
- Each template has a specific version (following semantic versioning)
- Templates build on each other in a modular way

### Command: Launching a Simple R Environment
```bash
cws launch r-research my-analysis
```

**Expected Output:**
```
🚀 Launching r-research instance 'my-analysis'...
💻 Using instance type: r5.large (optimized for R workloads)
📊 Estimated cost: $3.65 per day

⏳ Creating instance... (this may take 1-2 minutes)
  ✓ Creating security group
  ✓ Launching EC2 instance
  ✓ Waiting for instance to be ready
  ✓ Configuring RStudio Server

✅ Launch successful!

🔗 Connection details for 'my-analysis':
  • RStudio Server: http://54.84.123.45:8787 (user: rstudio, password: see below)
  • SSH: ssh -i ~/.ssh/id_rsa rstudio@54.84.123.45
  • Password: cloudws-12345

💾 Connection details saved to ~/.cloudworkstation/instances/my-analysis.json
```

**Explanation Points:**
- CloudWorkstation automatically selects an R-optimized instance (r5.large)
- Clear cost estimate is shown before proceeding
- Security and setup are handled automatically
- Connection information is displayed and saved for future reference

## Part 2: Template Information and Versioning

### Command: Exploring Template Details
```bash
cws ami template info python-ml
```

**Expected Output:**
```
📄 Template: python-ml (version 2.1.1)

Description:
  Python environment with machine learning frameworks and Jupyter Notebook

Base: basic-ubuntu (version >= 1.0.0)

Contents:
  • Python 3.10 with virtual environments
  • PyTorch 2.0 with CUDA support
  • TensorFlow 2.12
  • Jupyter Notebook and JupyterLab
  • Common ML libraries (scikit-learn, pandas, numpy)
  • GPU acceleration when available

Dependencies:
  • basic-ubuntu (version >= 1.0.0)

Build Steps: 10
Validation Steps: 5
Last Updated: 2023-07-15
```

**Explanation Points:**
- Templates include detailed information about their contents
- Dependencies are clearly listed with version constraints
- The template includes build and validation steps
- Template versioning tracks changes over time

### Command: Comparing Template Versions
```bash
cws ami template version compare python-ml 1.0.0 2.0.0
```

**Expected Output:**
```
🔍 Comparing versions: 1.0.0 vs 2.0.0

Result: 1.0.0 is less than 2.0.0

Breakdown:
  Major: 1 vs 2
  Minor: 0 vs 0
  Patch: 0 vs 0

Key differences:
  • Updated Python from 3.8 to 3.10
  • Upgraded PyTorch from 1.8 to 2.0
  • Added CUDA 11.7 support
  • Improved GPU detection and configuration
  • Breaking change: Removed legacy ML libraries
```

**Explanation Points:**
- Semantic versioning helps understand the magnitude of changes
- Major version increase (1.x to 2.x) indicates breaking changes
- The comparison shows exactly what changed between versions
- This helps researchers choose the appropriate version for their needs

## Part 3: Dependency Management

### Command: Visualizing Dependencies
```bash
cws ami template dependency graph data-science
```

**Expected Output:**
```
📋 Build order for template 'data-science':

1. basic-ubuntu
2. r-research
3. python-ml
4. desktop-research
5. data-science (target template)

digraph G {
  rankdir="LR";
  node [shape=box, style=filled, fillcolor=lightblue];
  "basic-ubuntu";
  "r-research";
  "python-ml";
  "desktop-research";
  "data-science" [fillcolor=lightgreen, fontcolor=black];

  "basic-ubuntu" -> "r-research";
  "basic-ubuntu" -> "python-ml";
  "basic-ubuntu" -> "desktop-research";
  "r-research" -> "data-science";
  "python-ml" -> "data-science";
  "desktop-research" -> "data-science";
}
```

**Explanation Points:**
- The dependency graph shows how templates build on each other
- Build order ensures dependencies are satisfied before building
- This visualization helps understand template relationships
- The graph can be exported in DOT format for visualization tools

### Command: Analyzing Dependencies
```bash
cws ami template dependency analyze data-science
```

**Expected Output:**
```
🔍 Analyzing dependencies for template 'data-science'

📊 Dependency Analysis Summary:
  Total dependencies:   3
  Satisfied:            3
  Missing (required):   0
  Missing (optional):   0
  Version mismatch:     0

✅ Template is buildable - all required dependencies are satisfied

Dependencies:
DEPENDENCY        VERSION   STATUS       OPTIONAL   NOTES
basic-ubuntu      1.0.0     ✅ satisfied  false       
r-research        1.0.0     ✅ satisfied  false       
python-ml         2.0.0     ✅ satisfied  false       
```

**Explanation Points:**
- The analyze command checks if all dependencies are satisfied
- It shows a summary of dependency status
- Green checkmarks indicate satisfied dependencies
- Status can also show missing or mismatched versions

### Command: Resolving Dependencies
```bash
cws ami template dependency resolve data-science --fetch
```

**Expected Output:**
```
🔍 Resolving dependencies for template 'data-science' (with fetching)

📋 Resolved dependencies for 'data-science':

DEPENDENCY        VERSION   STATUS       OPTIONAL   NOTES
basic-ubuntu      1.0.0     ✅ satisfied  false       
r-research        1.0.0     ✅ satisfied  false       
python-ml         2.1.1     ✅ satisfied  false      fetched from registry

📦 Build Order:
  1. basic-ubuntu
  2. r-research
  3. python-ml
  4. data-science (target template)

✅ All dependencies resolved successfully
```

**Explanation Points:**
- The resolve command checks and fetches missing dependencies
- It automatically retrieves the appropriate versions from the registry
- The "fetched from registry" note shows which templates were retrieved
- This ensures all dependencies are available before building

## Part 4: Advanced Launch Options

### Command: Launch with Customizations
```bash
cws launch data-science research-project --size L --region us-west-2 --spot
```

**Expected Output:**
```
🚀 Launching data-science instance 'research-project'...

💻 Selected configuration:
  • Instance type: m5.2xlarge (8 vCPU, 32GB RAM)
  • Region: us-west-2 (Oregon)
  • Spot instance: Yes (70% cost savings)
  • Storage: 100GB system disk
  
📊 Estimated costs:
  • Hourly: $0.28 (spot price, regular price: $0.93)
  • Daily: $6.72
  • Monthly: ~$201.60
  
⏳ Creating instance... (this may take 2-3 minutes)
  ✓ Validating template dependencies
  ✓ Creating security group
  ✓ Requesting spot instance
  ✓ Waiting for instance to be ready
  ✓ Configuring environment
  
✅ Launch successful!

🔗 Connection details for 'research-project':
  • JupyterLab: http://34.217.45.123:8888 (token: see below)
  • RStudio Server: http://34.217.45.123:8787 (user: rstudio, password: see below)
  • SSH: ssh -i ~/.ssh/id_rsa ubuntu@34.217.45.123
  • Password/Token: cloudws-67890

💾 Connection details saved to ~/.cloudworkstation/instances/research-project.json
```

**Explanation Points:**
- Advanced options allow customizing the environment
- The size flag (L) selects a larger instance with more resources
- Region selection allows placing instances closer to data or team
- Spot instances provide significant cost savings
- Transparent cost information helps with budgeting
- Multiple access methods (JupyterLab, RStudio, SSH) for flexibility

## Part 5: Template Customization

### Command: Creating a Custom Template
```bash
cws ami template create genomics-analysis --base python-ml
```

**Expected Output:**
```
📝 Creating new template 'genomics-analysis' based on 'python-ml'...

✅ Template created successfully!
📄 Template file: ~/.cloudworkstation/templates/genomics-analysis.yaml

Next steps:
1. Edit the template to add your genomics packages
2. Validate the template: cws ami template validate genomics-analysis
3. Build the template: cws ami template build genomics-analysis
```

**Explanation Points:**
- Templates can be customized for specific research domains
- The base template provides a starting point with core functionality
- The template is stored as a YAML file that can be edited
- Validation ensures the template will build correctly

### Command: Sharing a Template
```bash
cws ami template share genomics-analysis
```

**Expected Output:**
```
🌐 Sharing template 'genomics-analysis' (version 1.0.0)...
📤 Uploading template to registry...
✅ Template 'genomics-analysis' successfully shared

The template is now available to other CloudWorkstation users:
- They can discover it with: cws ami template search genomics
- They can import it with: cws ami template import-shared genomics-analysis
```

**Explanation Points:**
- Templates can be shared with colleagues or the community
- This enables reproducible research environments
- Others can discover and use your specialized templates
- Version tracking ensures others get the exact environment you used

## Summary of Key Benefits to Highlight

1. **Time Saving**
   - From hours/days of setup to minutes
   - No manual software installation or configuration

2. **Research Reproducibility**
   - Versioned templates ensure consistent environments
   - Shareable configurations for collaboration

3. **Cost Transparency**
   - Clear cost estimates before launching
   - Options like spot instances for budget optimization

4. **Automated Dependency Management**
   - No more version conflicts or dependency hell
   - Automatic resolution of software requirements

5. **Flexibility**
   - Support for various research workflows
   - Customizable templates for specific domains
   - Multiple access methods (web, SSH, etc.)