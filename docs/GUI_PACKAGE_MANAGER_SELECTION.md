# GUI Package Manager Selection

## Overview

The CloudWorkstation GUI now includes package manager selection in the launch form, allowing users to override template defaults with their preferred package manager.

## Implementation

### User Interface

The package manager selection is integrated into the basic launch form alongside template, name, and size selection. It includes:

1. **Dropdown Selection**: Choose from supported package managers:
   - `Default` - Let template choose optimal package manager
   - `conda` - Cross-platform package manager for data science
   - `apt` - Native Ubuntu/Debian package manager
   - `dnf` - Red Hat/Fedora package manager
   - `spack` - HPC and scientific computing packages
   - `ami` - Use pre-built AMI packages

2. **Contextual Help**: Dynamic help text that updates based on selection:
   - Explains the purpose and use case for each package manager
   - Provides guidance for optimal selection
   - Helps users understand the implications of their choice

### Technical Implementation

**Form Structure**:
```go
launchForm struct {
    templateSelect    *widget.Select
    nameEntry         *widget.Entry
    sizeSelect        *widget.Select
    packageMgrSelect  *widget.Select  // New field
    packageMgrHelp    *widget.Label   // New help text
    launchBtn         *widget.Button
    // ... other fields
}
```

**Launch Request Integration**:
```go
// Package manager selection is added to launch request
if g.launchForm.packageMgrSelect.Selected != "" && 
   g.launchForm.packageMgrSelect.Selected != "Default" {
    req.PackageManager = g.launchForm.packageMgrSelect.Selected
}
```

### User Experience

**Default Behavior**:
- Defaults to "Default" selection
- Shows help text: "Let template choose optimal package manager for the workload"
- When "Default" is selected, no package manager override is sent in the request

**Override Behavior**:
- User selects specific package manager (e.g., "conda")
- Help text updates: "Best for Python data science and R packages. Cross-platform package manager."
- Selected package manager is included in launch request via `--with` parameter equivalent

### Help Text System

The help system provides contextual guidance:

```go
func (g *CloudWorkstationGUI) updatePackageManagerHelp(selected string) {
    switch selected {
    case "conda":
        helpText = "Best for Python data science and R packages. Cross-platform package manager."
    case "apt":
        helpText = "Native Ubuntu/Debian package manager. System-level packages."
    case "dnf":
        helpText = "Red Hat/Fedora package manager. Newer replacement for yum."
    case "spack":
        helpText = "HPC and scientific computing packages. Optimized builds."
    case "ami":
        helpText = "Use pre-built AMI with packages already installed."
    case "Default":
        helpText = "Let template choose optimal package manager for the workload."
    }
}
```

## Integration with Template System

This GUI enhancement integrates seamlessly with the template inheritance system:

1. **Template Defaults**: Templates specify their optimal package manager
2. **User Override**: GUI allows users to override template defaults
3. **Inheritance Respect**: Package manager overrides work with inherited templates
4. **Validation**: Invalid package managers are caught by the template validation system

## Example Usage

**Default Usage**:
1. User selects template: "Rocky Linux 9 + Conda Stack"
2. Package manager shows: "Default" (uses template's conda specification)
3. Result: Instance launches with conda as specified in template

**Override Usage**:
1. User selects template: "Rocky Linux 9 + Conda Stack"
2. User changes package manager to: "spack"
3. Help text shows: "HPC and scientific computing packages. Optimized builds."
4. Result: Instance launches with spack instead of template's conda

## Benefits

### For Researchers
- **Visual Selection**: Clear dropdown with descriptive help text
- **Informed Decisions**: Contextual help explains each package manager
- **Flexibility**: Override template defaults when needed
- **Progressive Disclosure**: Advanced feature available without cluttering basic UI

### For Power Users
- **Override Capability**: Change package managers for specialized workflows
- **Clear Feedback**: Help text explains implications of selections
- **Integration**: Works with template inheritance and validation systems

### For System Reliability
- **Validation**: Package manager selections validated by existing template system
- **Consistency**: GUI and CLI use same package manager options
- **Error Prevention**: Invalid selections prevented by validation system

## Future Enhancements

The package manager selection system provides a foundation for additional GUI enhancements:

1. **Smart Recommendations**: Show recommended package managers based on template content
2. **Template Dependencies**: Display package manager compatibility warnings
3. **Performance Hints**: Show expected performance characteristics for different managers
4. **Cost Estimates**: Display cost implications of different package manager choices

This implementation provides researchers with the flexibility to customize their environments while maintaining CloudWorkstation's design principles of simplicity with progressive disclosure.