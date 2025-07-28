# Instance-to-AMI Workflow Design

## Overview

Design and implement the ability to save a running, customized CloudWorkstation instance as a reusable AMI template. This enables researchers to preserve their exact environment configuration for reuse and sharing.

## User Story

**Researcher Workflow:**
1. Launch initial environment: `cws launch python-ml my-research`  
2. Customize environment (install packages, configure tools, add data)
3. Save customized environment: `cws save my-research my-custom-ml-env`
4. Reuse saved environment: `cws launch my-custom-ml-env new-project`
5. Share with colleagues: `cws template share my-custom-ml-env`

## Technical Requirements

### 1. CLI Command: `cws save <instance-name> <template-name>`

**Command Signature:**
```bash
cws save my-instance my-template [options]
  --description "Description of custom template"
  --region us-west-2                    # Copy to specific regions  
  --public                              # Make template publicly shareable
  --project my-project                  # Associate with project (Phase 4)
```

**Implementation Steps:**
1. Validate instance exists and is running
2. Stop instance temporarily (required for consistent AMI)
3. Create AMI using AWS CreateImage API  
4. Restart instance automatically
5. Register AMI as new template in system
6. Copy AMI to additional regions if requested

### 2. AMI Creation from Instance

**New Function in `pkg/ami/builder.go`:**
```go
func (b *Builder) CreateAMIFromInstance(ctx context.Context, request InstanceSaveRequest) (*BuildResult, error)
```

**InstanceSaveRequest Structure:**
```go
type InstanceSaveRequest struct {
    InstanceName     string            // CloudWorkstation instance name
    TemplateName     string            // Name for new template
    Description      string            // Template description  
    CopyToRegions    []string          // Regions to copy AMI
    Tags             map[string]string // Custom tags
    ProjectID        string            // Associated project (Phase 4)
    Public           bool              // Allow public sharing
}
```

### 3. Template Registration

**Integration with Template System:**
- Register saved AMI in template registry (`pkg/ami/registry.go`)
- Create template definition file (YAML) for the saved environment
- Enable template to be used with `cws launch` command
- Support template versioning for incremental updates

**Generated Template Structure:**
```yaml
name: "my-custom-ml-env"
description: "Custom ML environment with TensorFlow 2.x and Jupyter extensions"
source: "saved-from-instance"  # Indicates origin
original_template: "python-ml" # Base template used
saved_from: "my-research"      # Original instance name
saved_by: "researcher@university.edu"
saved_date: "2024-01-15T10:30:00Z"

# AMI mappings (automatically populated)
ami_config:
  amis:
    us-east-1:
      x86_64: "ami-abc123def456"
      arm64: "ami-def456ghi789"
```

### 4. Instance State Management

**Safe AMI Creation Process:**
1. **Check instance state** - ensure instance is running and healthy
2. **Create snapshot warning** - inform user about temporary stop
3. **Stop instance gracefully** - `cws stop instance-name`  
4. **Create AMI** - AWS CreateImage with proper naming and tagging
5. **Monitor AMI creation** - wait for AMI to be available
6. **Restart instance** - `cws start instance-name`
7. **Register template** - add to CloudWorkstation template system

**Error Handling:**
- If AMI creation fails, restart instance immediately
- If instance fails to restart, provide troubleshooting guidance
- Preserve original instance state regardless of save operation outcome

### 5. Template Usage Integration

**Seamless Template Integration:**
Once saved, custom templates work exactly like built-in templates:

```bash
# Use saved template like any other
cws launch my-custom-ml-env new-research --size L
cws templates                           # Shows custom templates  
cws template info my-custom-ml-env      # Shows template details
```

**Template Metadata:**
- Source tracking (saved from which instance)
- Creation timestamp and creator
- Base template information
- Customization notes
- Usage statistics

### 6. Project Integration (Phase 4)

**Project-Based Template Management:**
```bash
# Associate saved template with project
cws save my-instance my-template --project brain-imaging-study

# List project templates
cws project templates brain-imaging-study

# Share project template with team members
cws project share-template brain-imaging-study my-template
```

## Implementation Plan

### Phase 1: Core Functionality
1. **CLI Command** - Add `cws save` command to CLI interface
2. **AMI Creation** - Implement instance-to-AMI conversion  
3. **Template Registration** - Register saved AMIs as templates
4. **Basic Testing** - Ensure save/launch workflow works

### Phase 2: Enhanced Features  
1. **Multi-Region Support** - Copy AMIs to multiple regions
2. **Template Sharing** - Enable community sharing of custom templates
3. **Versioning** - Support incremental saves and template versions
4. **Metadata Tracking** - Track template lineage and usage

### Phase 3: Project Integration
1. **Project Association** - Link saved templates to specific projects
2. **Team Sharing** - Share templates within project teams
3. **Access Control** - Manage who can use/modify custom templates
4. **Cost Tracking** - Track AMI storage costs in project budgets

## Benefits

### For Individual Researchers
- **Environment Preservation** - Never lose a perfectly configured setup
- **Rapid Deployment** - Launch complex environments in seconds
- **Experimentation Safety** - Try changes knowing you can return to known-good state

### For Research Teams  
- **Environment Sharing** - Share exact configurations with colleagues
- **Reproducible Research** - Ensure consistent environments across team
- **Onboarding** - New team members get productive environments instantly

### For Institutions
- **Standardization** - Create approved environments for specific research domains
- **Cost Control** - Reuse environments instead of rebuilding repeatedly  
- **Compliance** - Maintain approved software configurations

## Example Workflows

### Workflow 1: Individual Environment Preservation
```bash
# Start with base template
cws launch python-ml earthquake-analysis

# Researcher customizes over several days:
# - Installs specific seismic analysis packages
# - Configures Jupyter with custom kernels  
# - Adds research datasets
# - Optimizes performance settings

# Save the customized environment
cws save earthquake-analysis seismic-ml-environment \
  --description "ML environment optimized for seismic data analysis"

# Launch new projects from saved environment
cws launch seismic-ml-environment aftershock-prediction --size GPU-L
```

### Workflow 2: Team Environment Sharing
```bash
# Lead researcher creates and saves custom environment
cws save my-genomics-work team-genomics-env --public \
  --description "Genomics pipeline with BWA, GATK, and R Bioconductor"

# Team members use the shared environment
cws launch team-genomics-env variant-calling
cws launch team-genomics-env population-analysis  
```

### Workflow 3: Course Environment Distribution
```bash
# Professor creates course environment
cws save ml-course-prep cs229-environment \
  --description "Stanford CS229 Machine Learning Course Environment"

# Students launch identical environments
cws launch cs229-environment assignment-1
cws launch cs229-environment final-project
```

This feature transforms CloudWorkstation from a template-based system to a **living research platform** where environments evolve and can be preserved at any point in their lifecycle.