# CloudWorkstation Demo Implementation Plan

## Overview

This document outlines a comprehensive plan to address the issues identified in the demo testing phase and ensure a successful user recruitment demonstration. The plan prioritizes critical fixes while providing fallback options in case certain issues cannot be resolved in time.

## Implementation Timeline

### Phase 1: Core Infrastructure Fixes (Days 1-2)

**Goal:** Fix critical issues that prevent basic functionality

#### 1.1 Fix Daemon Port Configuration

```go
// In cmd/cwsd/main.go
func main() {
    portFlag := flag.String("port", "8080", "Port to listen on")
    helpFlag := flag.Bool("help", false, "Show help")
    versionFlag := flag.Bool("version", false, "Show version")
    flag.Parse()
    
    // Handle flags
    if *helpFlag {
        flag.Usage()
        return
    }
    
    if *versionFlag {
        fmt.Printf("CloudWorkstation v%s [%s]\n", version, runtime.Version())
        return
    }
    
    // Configure port
    port := *portFlag
    log.Printf("CloudWorkstation Daemon v%s starting...\n", version)
    
    // Try to start on the specified port, fall back to auto-detection
    server := server.New(port)
    err := server.Start()
    if err != nil {
        if strings.Contains(err.Error(), "bind: address already in use") {
            // Try finding an available port
            for p := 8081; p < 8100; p++ {
                alternatePort := fmt.Sprintf("%d", p)
                log.Printf("Port %s in use, trying port %s\n", port, alternatePort)
                server = server.New(alternatePort)
                err = server.Start()
                if err == nil {
                    break
                }
            }
        }
        
        if err != nil {
            log.Fatalf("Server failed: %v", err)
        }
    }
}
```

#### 1.2 Fix CLI Compilation Errors

```go
// In internal/cli/app.go
type App struct {
    version   string
    apiClient api.CloudWorkstationAPI
    ctx       context.Context // Add context field
}

// Update NewApp constructor
func NewApp(version string) *App {
    return &App{
        version:   version,
        apiClient: api.NewClient(""), // Uses default localhost:8080
        ctx:       context.Background(),
    }
}
```

- Remove unused imports in template_version_search.go
- Fix unused variable issues in template_validate.go
- Remove unused label in template_version_search.go

#### 1.3 Basic Mock Mode Framework

```go
// In pkg/api/mock_client.go
type MockClient struct {
    Templates map[string]*types.Template
    Instances map[string]*types.Instance
}

func NewMockClient() *MockClient {
    return &MockClient{
        Templates: loadMockTemplates(),
        Instances: loadMockInstances(),
    }
}

// In cmd/cws/main.go
func main() {
    versionFlag := flag.Bool("version", false, "Show version information")
    demoFlag := flag.Bool("demo", false, "Run in demo mode with mock data")
    flag.Parse()
    
    // Create app with appropriate client
    var app *cli.App
    if *demoFlag {
        app = cli.NewAppWithClient(version, api.NewMockClient())
        fmt.Println("⚠️  Running in DEMO mode with mock data")
    } else {
        app = cli.NewApp(version)
    }
    
    // Run the app
    if err := app.Run(flag.Args()); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### Phase 2: Mock Data Implementation (Days 3-4)

**Goal:** Create realistic mock responses for core demo commands

#### 2.1 Template Listing and Info

```go
func loadMockTemplates() map[string]*types.Template {
    return map[string]*types.Template{
        "basic-ubuntu": {
            Name:        "basic-ubuntu",
            Version:     "1.0.0",
            Description: "Base Ubuntu 22.04 template",
        },
        "desktop-research": {
            Name:        "desktop-research", 
            Version:     "1.0.0",
            Description: "Ubuntu Desktop with research tools",
        },
        "r-research": {
            Name:        "r-research",
            Version:     "1.0.0",
            Description: "R and RStudio Server with common packages",
            Dependencies: []types.TemplateDependency{
                {Name: "basic-ubuntu", Version: "1.0.0", VersionOperator: ">="},
            },
        },
        "python-ml": {
            Name:        "python-ml",
            Version:     "2.1.1",
            Description: "Python with ML frameworks and Jupyter",
            Dependencies: []types.TemplateDependency{
                {Name: "basic-ubuntu", Version: "1.0.0", VersionOperator: ">="},
            },
        },
        "data-science": {
            Name:        "data-science",
            Version:     "1.0.0",
            Description: "Complete data science environment with R and Python",
            Dependencies: []types.TemplateDependency{
                {Name: "r-research", Version: "1.0.0", VersionOperator: ">="},
                {Name: "python-ml", Version: "2.0.0", VersionOperator: ">="},
                {Name: "desktop-research", Version: "1.0.0", VersionOperator: ">="},
            },
        },
    }
}

// Implement ListTemplates
func (m *MockClient) ListTemplates() (*types.TemplateListResponse, error) {
    templates := make([]*types.Template, 0, len(m.Templates))
    for _, t := range m.Templates {
        templates = append(templates, t)
    }
    return &types.TemplateListResponse{Templates: templates}, nil
}

// Implement GetTemplate
func (m *MockClient) GetTemplate(name string) (*types.Template, error) {
    if template, ok := m.Templates[name]; ok {
        return template, nil
    }
    return nil, fmt.Errorf("template not found: %s", name)
}
```

#### 2.2 Version Comparison Data

```go
// Pre-define version differences for the demo
var versionDifferences = map[string]map[string][]string{
    "python-ml": {
        "1.0.0_2.0.0": {
            "Updated Python from 3.8 to 3.10",
            "Upgraded PyTorch from 1.8 to 2.0",
            "Added CUDA 11.7 support",
            "Improved GPU detection and configuration",
            "Breaking change: Removed legacy ML libraries",
        },
    },
}

func (m *MockClient) CompareVersions(template, version1, version2 string) (*types.VersionCompareResult, error) {
    key := fmt.Sprintf("%s_%s", version1, version2)
    
    // Get pre-defined differences
    differences, ok := versionDifferences[template][key]
    if !ok {
        // Generate generic differences if not pre-defined
        differences = []string{
            fmt.Sprintf("Updated %s core components", template),
            "Improved performance and stability",
        }
    }
    
    // Parse versions
    v1, _ := ami.NewVersionInfo(version1)
    v2, _ := ami.NewVersionInfo(version2)
    
    // Determine result
    var result string
    if v1.IsGreaterThan(v2) {
        result = fmt.Sprintf("%s is greater than %s", version1, version2)
    } else if v2.IsGreaterThan(v1) {
        result = fmt.Sprintf("%s is less than %s", version1, version2)
    } else {
        result = fmt.Sprintf("%s is equal to %s", version1, version2)
    }
    
    return &types.VersionCompareResult{
        Result: result,
        VersionOne: version1,
        VersionTwo: version2,
        Differences: differences,
        MajorOne: v1.Major,
        MajorTwo: v2.Major,
        MinorOne: v1.Minor,
        MinorTwo: v2.Minor,
        PatchOne: v1.Patch,
        PatchTwo: v2.Patch,
    }, nil
}
```

#### 2.3 Dependency Graph and Resolution

```go
func (m *MockClient) GetDependencyGraph(templateName string) (*types.DependencyGraphResponse, error) {
    if template, ok := m.Templates[templateName]; !ok {
        return nil, fmt.Errorf("template not found: %s", templateName)
    }
    
    // Pre-defined dependency graphs for demo templates
    graphs := map[string][]string{
        "r-research":     {"basic-ubuntu", "r-research"},
        "python-ml":      {"basic-ubuntu", "python-ml"},
        "data-science":   {"basic-ubuntu", "r-research", "python-ml", "desktop-research", "data-science"},
    }
    
    graph, ok := graphs[templateName]
    if !ok {
        graph = []string{templateName}
    }
    
    return &types.DependencyGraphResponse{
        TemplateName: templateName,
        BuildOrder:   graph,
    }, nil
}

func (m *MockClient) ResolveDependencies(templateName string, fetchMissing bool) (*types.DependencyResolveResponse, error) {
    if _, ok := m.Templates[templateName]; !ok {
        return nil, fmt.Errorf("template not found: %s", templateName)
    }
    
    // Get dependency graph
    graphResp, _ := m.GetDependencyGraph(templateName)
    
    // Create mock resolved dependencies
    resolved := make(map[string]*types.ResolvedDependency)
    for _, dep := range graphResp.BuildOrder {
        if dep == templateName {
            continue // Skip the target template
        }
        
        resolved[dep] = &types.ResolvedDependency{
            Name:       dep,
            Version:    m.Templates[dep].Version,
            IsOptional: false,
            Status:     "satisfied",
        }
    }
    
    // If fetchMissing, simulate fetching one dependency
    fetched := []string{}
    if fetchMissing && templateName == "data-science" {
        fetched = append(fetched, "python-ml")
        resolved["python-ml"].Status = "satisfied"
    }
    
    return &types.DependencyResolveResponse{
        TemplateName:   templateName,
        Dependencies:   resolved,
        FetchedTemplates: fetched,
        BuildOrder:     graphResp.BuildOrder,
    }, nil
}
```

#### 2.4 Instance Launch Mock

```go
func (m *MockClient) LaunchInstance(req types.LaunchRequest) (*types.LaunchResponse, error) {
    // Create a mock instance
    instanceID := fmt.Sprintf("%s-%s", req.Template, time.Now().Format("20060102150405"))
    publicIP := "54.84.123.45" // Mock IP
    
    // Simulate different settings based on template
    var instanceType, estimatedCost, connectionInfo string
    switch req.Template {
    case "r-research":
        instanceType = "r5.large"
        estimatedCost = "$3.65 per day"
        connectionInfo = fmt.Sprintf("RStudio Server: http://%s:8787", publicIP)
    case "python-ml":
        instanceType = "g4dn.xlarge"
        estimatedCost = "$7.80 per day"
        connectionInfo = fmt.Sprintf("JupyterLab: http://%s:8888", publicIP)
    case "data-science":
        instanceType = "m5.2xlarge"
        if req.Spot {
            estimatedCost = "$6.72 per day (spot pricing)"
        } else {
            estimatedCost = "$22.32 per day"
        }
        connectionInfo = fmt.Sprintf("Multiple services at http://%s", publicIP)
    default:
        instanceType = "t3.medium"
        estimatedCost = "$1.15 per day"
        connectionInfo = fmt.Sprintf("SSH: ssh ubuntu@%s", publicIP)
    }
    
    // Create instance
    instance := &types.Instance{
        ID:          instanceID,
        Name:        req.Name,
        Template:    req.Template,
        State:       "running",
        LaunchTime:  time.Now(),
        PublicIP:    publicIP,
        InstanceType: instanceType,
    }
    
    // Add to mock data
    m.Instances[instanceID] = instance
    
    // Simulate delay for launch
    time.Sleep(500 * time.Millisecond)
    
    return &types.LaunchResponse{
        InstanceID:    instanceID,
        Message:       fmt.Sprintf("Successfully launched %s instance '%s'", req.Template, req.Name),
        EstimatedCost: estimatedCost,
        ConnectionInfo: connectionInfo,
    }, nil
}
```

### Phase 3: CLI Command Implementation (Days 5-6)

**Goal:** Implement essential CLI commands for the demo

#### 3.1 Fix CLI Command Structure

```go
// In internal/cli/app.go
func (a *App) Run(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("no command specified")
    }
    
    cmd := args[0]
    cmdArgs := args[1:]
    
    switch cmd {
    case "ami":
        return a.handleAMI(cmdArgs)
    case "launch":
        return a.Launch(cmdArgs)
    case "list":
        return a.List(cmdArgs)
    case "connect":
        return a.Connect(cmdArgs)
    case "stop":
        return a.Stop(cmdArgs)
    case "start":
        return a.Start(cmdArgs)
    case "delete":
        return a.Delete(cmdArgs)
    case "version":
        fmt.Printf("CloudWorkstation CLI v%s\n", a.version)
        return nil
    case "help", "--help", "-h":
        return a.printHelp()
    default:
        return fmt.Errorf("unknown command: %s", cmd)
    }
}
```

#### 3.2 Template Management Commands

```go
func (a *App) handleAMI(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("missing ami subcommand")
    }
    
    subcmd := args[0]
    subargs := args[1:]
    
    switch subcmd {
    case "template":
        return a.handleTemplate(subargs)
    // Other AMI commands...
    default:
        return fmt.Errorf("unknown ami subcommand: %s", subcmd)
    }
}

func (a *App) handleTemplate(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("missing template subcommand")
    }
    
    subcmd := args[0]
    subargs := args[1:]
    
    switch subcmd {
    case "list":
        return a.handleTemplateList(subargs)
    case "info":
        return a.handleTemplateInfo(subargs)
    case "version":
        return a.handleTemplateVersion(subargs)
    case "dependency":
        return a.handleTemplateDependency(subargs)
    // Other template commands...
    default:
        return fmt.Errorf("unknown template subcommand: %s", subcmd)
    }
}
```

#### 3.3 Implement Core Commands

Focus on implementing these key commands for the demo:
- `cws ami template list`
- `cws ami template info <template>`
- `cws ami template version compare <v1> <v2>`
- `cws ami template dependency graph <template>`
- `cws ami template dependency resolve <template>`
- `cws launch <template> <name>`

### Phase 4: Finalization and Testing (Day 7)

**Goal:** Polish the demo and prepare contingency plans

#### 4.1 Full Demo Testing

- Run through the entire demo script with mock data
- Test each command for proper output formatting
- Verify timing of each demo section

#### 4.2 Prepare Backup Materials

- Create screenshots of expected outputs
- Record a video walkthrough of the demo
- Prepare slides for any features not fully implemented

#### 4.3 Fallback Script

Create a shell script that can simulate the demo if needed:

```bash
#!/bin/bash

# Demo script for CloudWorkstation

function show_command() {
    echo -e "\n\$ $1"
    sleep 1
}

function run_command() {
    show_command "$1"
    eval "$1"
    sleep 2
}

echo "CloudWorkstation Demo"
echo "====================="

# Template listing
show_command "cws ami template list"
cat <<EOF
📋 Available templates:

NAME              VERSION    DESCRIPTION                                         
basic-ubuntu      1.0.0      Base Ubuntu 22.04 template                          
desktop-research  1.0.0      Ubuntu Desktop with research tools                  
r-research        1.0.0      R and RStudio Server with common packages           
python-ml         2.1.1      Python with ML frameworks and Jupyter               
data-science      1.0.0      Complete data science environment with R and Python 
EOF

sleep 3

# More commands...
```

## Contingency Plans

### If Daemon Issues Persist

1. Modify the demo to use only the fallback script
2. Explain that the daemon is typically running as a service
3. Show pre-recorded video of daemon interaction

### If CLI Commands Aren't Ready

1. Use the fallback script to simulate command outputs
2. Focus on the architecture and design principles
3. Show mockups of the planned interface

### If Time Runs Short

1. Focus on implementing just 2-3 core commands:
   - Template listing
   - Dependency visualization
   - Basic launch simulation
2. Use slides to explain more complex features

## Resource Allocation

### Developer Assignments

1. **Backend Developer:**
   - Fix daemon port configuration
   - Implement mock registry interface

2. **Frontend Developer:**
   - Fix CLI compilation issues
   - Implement formatted outputs for demo commands

3. **DevOps/Demo Lead:**
   - Create fallback scripts
   - Prepare demo environment
   - Test full demo flow

## Conclusion

By following this implementation plan, we can ensure a successful demonstration of CloudWorkstation's capabilities, even if not all features are fully implemented. The key is to focus on showcasing the core value proposition: simplifying research environment setup through templates and dependency management.

The mock mode implementation will allow us to demonstrate the system's capabilities reliably without depending on actual AWS resources, while the fallback mechanisms ensure we can deliver a compelling demo even if technical issues arise.