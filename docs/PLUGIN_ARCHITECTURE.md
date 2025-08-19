# CloudWorkstation Plugin Architecture

## Overview

CloudWorkstation's unified plugin system allows extending both CLI commands and daemon capabilities through a single plugin interface. This enables institutions, researchers, and third-party developers to add custom functionality while maintaining system stability and security.

## Architecture Components

### **Unified Plugin Interface**

```go
// Plugin represents a CloudWorkstation extension
type Plugin interface {
    // Metadata
    Name() string
    Version() string
    Description() string
    Author() string
    
    // Capabilities
    Capabilities() PluginCapabilities
    
    // Lifecycle
    Initialize(ctx context.Context, config PluginConfig) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // Health and status
    Health() PluginHealth
    Status() PluginStatus
}

// PluginCapabilities defines what the plugin can do
type PluginCapabilities struct {
    // CLI extensions
    CLICommands     []CLICommandSpec     `json:"cli_commands,omitempty"`
    CLIFlags        []CLIFlagSpec        `json:"cli_flags,omitempty"`
    
    // Daemon extensions  
    APIEndpoints    []APIEndpointSpec    `json:"api_endpoints,omitempty"`
    EventHandlers   []EventHandlerSpec   `json:"event_handlers,omitempty"`
    
    // Template system extensions
    TemplateTypes   []string             `json:"template_types,omitempty"`
    ServiceTypes    []string             `json:"service_types,omitempty"`
    
    // GUI extensions
    GUIComponents   []GUIComponentSpec   `json:"gui_components,omitempty"`
    ThemeProviders  []ThemeProviderSpec  `json:"theme_providers,omitempty"`
    
    // System integrations
    AuthProviders   []AuthProviderSpec   `json:"auth_providers,omitempty"`
    StorageProviders []StorageProviderSpec `json:"storage_providers,omitempty"`
}

// CLICommandSpec defines a new CLI command
type CLICommandSpec struct {
    Name        string            `json:"name"`
    Usage       string            `json:"usage"`  
    Short       string            `json:"short"`
    Long        string            `json:"long"`
    Aliases     []string          `json:"aliases,omitempty"`
    Parent      string            `json:"parent,omitempty"`  // Parent command for subcommands
    Flags       []CLIFlagSpec     `json:"flags,omitempty"`
    Handler     string            `json:"handler"`           // Plugin method name
}

// APIEndpointSpec defines a new daemon API endpoint
type APIEndpointSpec struct {
    Path        string   `json:"path"`         // "/api/v1/plugin/custom-action"
    Method      string   `json:"method"`       // "GET", "POST", etc.
    Handler     string   `json:"handler"`      // Plugin method name
    Protected   bool     `json:"protected"`    // Requires authentication
    RateLimit   int      `json:"rate_limit,omitempty"`
    Permissions []string `json:"permissions,omitempty"`
}
```

### **Plugin Manager**

```go
// PluginManager handles plugin lifecycle and coordination
type PluginManager struct {
    loadedPlugins   map[string]Plugin
    pluginConfigs   map[string]PluginConfig
    
    // Plugin directories
    systemPluginDir string  // /usr/lib/cloudworkstation/plugins
    userPluginDir   string  // ~/.cloudworkstation/plugins
    
    // Runtime state
    cliExtensions   *CLIExtensionRegistry
    apiExtensions   *APIExtensionRegistry
    eventBus        *EventBus
    
    // Security
    sandboxManager  *SandboxManager
    permissionManager *PermissionManager
}

// LoadPlugin loads and initializes a plugin
func (pm *PluginManager) LoadPlugin(pluginPath string, config PluginConfig) error {
    // Load plugin binary (Go plugin or executable)
    plugin, err := pm.loadPluginBinary(pluginPath)
    if err != nil {
        return fmt.Errorf("failed to load plugin binary: %w", err)
    }
    
    // Validate plugin security
    if err := pm.validatePluginSecurity(plugin, config); err != nil {
        return fmt.Errorf("plugin security validation failed: %w", err)
    }
    
    // Initialize plugin in sandbox
    ctx := pm.createPluginContext(config)
    if err := plugin.Initialize(ctx, config); err != nil {
        return fmt.Errorf("plugin initialization failed: %w", err)
    }
    
    // Register plugin capabilities
    capabilities := plugin.Capabilities()
    
    // Register CLI extensions
    for _, cmd := range capabilities.CLICommands {
        if err := pm.cliExtensions.RegisterCommand(plugin, cmd); err != nil {
            return fmt.Errorf("failed to register CLI command %s: %w", cmd.Name, err)
        }
    }
    
    // Register API extensions
    for _, endpoint := range capabilities.APIEndpoints {
        if err := pm.apiExtensions.RegisterEndpoint(plugin, endpoint); err != nil {
            return fmt.Errorf("failed to register API endpoint %s: %w", endpoint.Path, err)
        }
    }
    
    // Register event handlers
    for _, handler := range capabilities.EventHandlers {
        pm.eventBus.RegisterHandler(plugin, handler)
    }
    
    // Store plugin
    pm.loadedPlugins[plugin.Name()] = plugin
    pm.pluginConfigs[plugin.Name()] = config
    
    // Start plugin
    if err := plugin.Start(ctx); err != nil {
        return fmt.Errorf("failed to start plugin: %w", err)
    }
    
    log.Printf("Loaded plugin: %s v%s", plugin.Name(), plugin.Version())
    return nil
}
```

### **CLI Extension System**

```go
// CLIExtensionRegistry manages CLI command extensions
type CLIExtensionRegistry struct {
    pluginCommands map[string]PluginCommandHandler
    pluginFlags    map[string]PluginFlagHandler
}

// PluginCommandHandler wraps plugin CLI command execution
type PluginCommandHandler struct {
    plugin   Plugin
    spec     CLICommandSpec
    sandbox  *CommandSandbox
}

// Execute runs the plugin command in a sandboxed environment
func (pch *PluginCommandHandler) Execute(cmd *cobra.Command, args []string) error {
    // Create sandboxed execution context
    ctx := pch.sandbox.CreateContext(cmd, args)
    
    // Call plugin handler method
    result, err := pch.plugin.ExecuteCLICommand(ctx, pch.spec.Handler, args)
    if err != nil {
        return fmt.Errorf("plugin command failed: %w", err)
    }
    
    // Handle plugin response
    return pch.handlePluginResult(result)
}

// Enhanced CLI command registration with plugin support
func (app *CLIApplication) LoadPluginCommands(pluginManager *PluginManager) error {
    extensions := pluginManager.GetCLIExtensions()
    
    for _, ext := range extensions {
        // Create cobra command from plugin spec
        pluginCmd := &cobra.Command{
            Use:     ext.spec.Usage,
            Short:   ext.spec.Short,
            Long:    ext.spec.Long,
            Aliases: ext.spec.Aliases,
            RunE:    ext.Execute,
        }
        
        // Add plugin flags
        for _, flag := range ext.spec.Flags {
            app.addPluginFlag(pluginCmd, flag)
        }
        
        // Attach to parent command or root
        if ext.spec.Parent != "" {
            parentCmd := app.findCommand(ext.spec.Parent)
            if parentCmd != nil {
                parentCmd.AddCommand(pluginCmd)
            }
        } else {
            app.rootCmd.AddCommand(pluginCmd)
        }
        
        log.Printf("Registered plugin CLI command: %s", ext.spec.Name)
    }
    
    return nil
}
```

### **Daemon API Extension System**

```go
// APIExtensionRegistry manages daemon API extensions
type APIExtensionRegistry struct {
    pluginEndpoints map[string]PluginEndpointHandler
    router         *mux.Router
    middleware     []PluginMiddleware
}

// PluginEndpointHandler wraps plugin API endpoint execution
type PluginEndpointHandler struct {
    plugin     Plugin
    spec       APIEndpointSpec
    sandbox    *APISandbox
    rateLimit  *RateLimiter
}

// ServeHTTP implements http.Handler for plugin endpoints
func (peh *PluginEndpointHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Rate limiting
    if peh.rateLimit != nil && !peh.rateLimit.Allow() {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }
    
    // Authentication check
    if peh.spec.Protected {
        if !peh.authenticateRequest(r) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
    }
    
    // Permission check
    if len(peh.spec.Permissions) > 0 {
        if !peh.authorizeRequest(r, peh.spec.Permissions) {
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }
    }
    
    // Create sandboxed execution context
    ctx := peh.sandbox.CreateContext(w, r)
    
    // Call plugin handler
    result, err := peh.plugin.ExecuteAPIEndpoint(ctx, peh.spec.Handler, r)
    if err != nil {
        http.Error(w, fmt.Sprintf("Plugin error: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Handle plugin response
    peh.handlePluginResult(w, result)
}

// Enhanced daemon with plugin API support
func (daemon *Daemon) LoadPluginEndpoints(pluginManager *PluginManager) error {
    extensions := pluginManager.GetAPIExtensions()
    
    for _, ext := range extensions {
        // Register plugin endpoint with router
        daemon.router.Handle(ext.spec.Path, ext).Methods(ext.spec.Method)
        
        log.Printf("Registered plugin API endpoint: %s %s", ext.spec.Method, ext.spec.Path)
    }
    
    return nil
}
```

## Plugin Development Examples

### **Research Analytics Plugin**

```go
// research_analytics_plugin.go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    
    "github.com/scttfrdmn/cloudworkstation/pkg/plugin"
)

type ResearchAnalyticsPlugin struct {
    config plugin.PluginConfig
    db     *AnalyticsDatabase
}

func (p *ResearchAnalyticsPlugin) Name() string { return "research-analytics" }
func (p *ResearchAnalyticsPlugin) Version() string { return "1.0.0" }
func (p *ResearchAnalyticsPlugin) Description() string { 
    return "Advanced analytics for research usage patterns"
}
func (p *ResearchAnalyticsPlugin) Author() string { return "University Research IT" }

func (p *ResearchAnalyticsPlugin) Capabilities() plugin.PluginCapabilities {
    return plugin.PluginCapabilities{
        // CLI commands
        CLICommands: []plugin.CLICommandSpec{
            {
                Name:    "analytics",
                Usage:   "analytics [subcommand]",
                Short:   "Research usage analytics",
                Long:    "Analyze research computing usage patterns and generate reports",
                Handler: "HandleAnalyticsCommand",
            },
            {
                Name:    "report",
                Usage:   "report --type [weekly|monthly|yearly]",
                Short:   "Generate usage report", 
                Parent:  "analytics",
                Flags: []plugin.CLIFlagSpec{
                    {Name: "type", Type: "string", Default: "weekly"},
                    {Name: "format", Type: "string", Default: "table"},
                },
                Handler: "HandleReportCommand",
            },
        },
        
        // API endpoints
        APIEndpoints: []plugin.APIEndpointSpec{
            {
                Path:      "/api/v1/analytics/usage",
                Method:    "GET", 
                Handler:   "HandleUsageAPI",
                Protected: true,
            },
            {
                Path:      "/api/v1/analytics/costs",
                Method:    "GET",
                Handler:   "HandleCostAnalysisAPI", 
                Protected: true,
                Permissions: []string{"analytics.read"},
            },
        },
        
        // Event handlers
        EventHandlers: []plugin.EventHandlerSpec{
            {
                EventType: "instance.launched",
                Handler:   "HandleInstanceLaunched",
            },
            {
                EventType: "instance.terminated", 
                Handler:   "HandleInstanceTerminated",
            },
        },
    }
}

// CLI command handlers
func (p *ResearchAnalyticsPlugin) ExecuteCLICommand(ctx plugin.CommandContext, handler string, args []string) (*plugin.CommandResult, error) {
    switch handler {
    case "HandleAnalyticsCommand":
        return p.handleAnalyticsCommand(ctx, args)
    case "HandleReportCommand":
        return p.handleReportCommand(ctx, args)
    default:
        return nil, fmt.Errorf("unknown command handler: %s", handler)
    }
}

func (p *ResearchAnalyticsPlugin) handleReportCommand(ctx plugin.CommandContext, args []string) (*plugin.CommandResult, error) {
    reportType := ctx.GetFlag("type")
    format := ctx.GetFlag("format")
    
    // Generate analytics report
    report, err := p.generateUsageReport(reportType)
    if err != nil {
        return nil, err
    }
    
    // Format output
    var output string
    switch format {
    case "json":
        jsonData, _ := json.MarshalIndent(report, "", "  ")
        output = string(jsonData)
    case "csv":
        output = p.formatReportCSV(report)
    default:
        output = p.formatReportTable(report)
    }
    
    return &plugin.CommandResult{
        Output:   output,
        ExitCode: 0,
    }, nil
}

// API endpoint handlers
func (p *ResearchAnalyticsPlugin) ExecuteAPIEndpoint(ctx plugin.APIContext, handler string, r *http.Request) (*plugin.APIResult, error) {
    switch handler {
    case "HandleUsageAPI":
        return p.handleUsageAPI(ctx, r)
    case "HandleCostAnalysisAPI": 
        return p.handleCostAnalysisAPI(ctx, r)
    default:
        return nil, fmt.Errorf("unknown API handler: %s", handler)
    }
}

func (p *ResearchAnalyticsPlugin) handleUsageAPI(ctx plugin.APIContext, r *http.Request) (*plugin.APIResult, error) {
    // Parse query parameters
    timeRange := r.URL.Query().Get("timerange")
    if timeRange == "" {
        timeRange = "7d"
    }
    
    // Get usage data
    usage, err := p.db.GetUsageData(timeRange)
    if err != nil {
        return nil, err
    }
    
    return &plugin.APIResult{
        StatusCode: http.StatusOK,
        Data:       usage,
        Headers:    map[string]string{"Content-Type": "application/json"},
    }, nil
}

// Event handlers
func (p *ResearchAnalyticsPlugin) HandleEvent(ctx plugin.EventContext, eventType string, data interface{}) error {
    switch eventType {
    case "instance.launched":
        return p.recordInstanceLaunch(data)
    case "instance.terminated":
        return p.recordInstanceTermination(data)
    }
    return nil
}
```

### **SLURM Integration Plugin**

```go
// slurm_integration_plugin.go  
package main

type SlurmIntegrationPlugin struct {
    slurmClient *SlurmClient
    config      plugin.PluginConfig
}

func (p *SlurmIntegrationPlugin) Capabilities() plugin.PluginCapabilities {
    return plugin.PluginCapabilities{
        // Add SLURM job submission commands
        CLICommands: []plugin.CLICommandSpec{
            {
                Name:    "slurm",
                Usage:   "slurm [subcommand]",
                Short:   "SLURM cluster integration",
                Handler: "HandleSlurmCommand",
            },
            {
                Name:    "submit",
                Usage:   "submit [job-script] --partition [name]",
                Short:   "Submit job to SLURM cluster",
                Parent:  "slurm",
                Handler: "HandleSlurmSubmit",
            },
            {
                Name:    "status", 
                Usage:   "status [job-id]",
                Short:   "Check SLURM job status",
                Parent:  "slurm", 
                Handler: "HandleSlurmStatus",
            },
        ],
        
        // Add SLURM API endpoints
        APIEndpoints: []plugin.APIEndpointSpec{
            {
                Path:    "/api/v1/slurm/jobs",
                Method:  "POST",
                Handler: "HandleJobSubmissionAPI",
            },
            {
                Path:    "/api/v1/slurm/jobs/{id}",
                Method:  "GET", 
                Handler: "HandleJobStatusAPI",
            },
        },
        
        // Add custom service type for SLURM jobs
        ServiceTypes: []string{"slurm_job"},
        
        // Add SLURM template support
        TemplateTypes: []string{"slurm_template"},
    }
}

// SLURM template example that plugin would handle
/*
# Template: slurm-python-job.yml
name: "SLURM Python Job"
service_type: "slurm_job"  # Custom service type from plugin
connection_type: "api"

slurm_config:
  partition: "compute"
  nodes: 1
  tasks_per_node: 8
  time: "02:00:00"
  memory: "32GB"
  
job_script: |
  #!/bin/bash
  #SBATCH --job-name=python-research
  #SBATCH --output=output_%j.txt
  #SBATCH --error=error_%j.txt
  
  module load python/3.9
  python research_script.py
*/
```

## Plugin Distribution and Management

### **Plugin Installation**

```bash
# Install plugin from repository
cws plugin install research-analytics
# Downloaded: research-analytics v1.0.0
# Installed to: ~/.cloudworkstation/plugins/research-analytics/
# Available commands: cws analytics, cws analytics report
# Available APIs: /api/v1/analytics/*

# Install from local file
cws plugin install ./custom-slurm-plugin.cwsplugin
# Installed: slurm-integration v2.1.0
# New service types: slurm_job
# New template types: slurm_template

# List installed plugins  
cws plugin list
# PLUGIN                VERSION   STATUS    CAPABILITIES
# research-analytics    1.0.0     active    CLI, API, Events
# slurm-integration     2.1.0     active    CLI, API, Templates
# institutional-theme   1.5.0     active    GUI, Themes

# Plugin status and health
cws plugin status research-analytics
# Plugin: research-analytics v1.0.0
# Status: Active
# Health: Healthy
# Capabilities: 2 CLI commands, 2 API endpoints, 2 event handlers
# Resource usage: 15MB memory, 0.1% CPU
# API requests: 1,247 total, 12 errors (0.97%)

# Update plugin
cws plugin update research-analytics
# Updated: research-analytics v1.0.0 → v1.1.0
# Changes: Added cost prediction API, improved report formatting
# Restart required: No (hot-reloadable)

# Disable/enable plugin
cws plugin disable slurm-integration
cws plugin enable slurm-integration

# Remove plugin
cws plugin remove research-analytics --confirm
```

### **Plugin Security Model**

```go
// PluginSandbox provides isolated execution environment
type PluginSandbox struct {
    // Resource limits
    maxMemory     int64         // Maximum memory usage
    maxCPU        float64       // Maximum CPU percentage
    maxDiskSpace  int64         // Maximum disk usage
    maxNetworkBPS int64         // Maximum network bandwidth
    
    // Permission model
    allowedAPIs      []string     // Which CloudWorkstation APIs plugin can call
    allowedPaths     []string     // Filesystem paths plugin can access
    allowedNetwork   []string     // Network endpoints plugin can access
    allowedEnvVars   []string     // Environment variables plugin can read
    
    // Security policies
    allowExec        bool         // Can execute external commands
    allowNetwork     bool         // Can make network requests
    allowFileSystem  bool         // Can access filesystem
    allowSecrets     bool         // Can access sensitive configuration
}

// Plugin security validation
func (pm *PluginManager) validatePluginSecurity(plugin Plugin, config PluginConfig) error {
    // Check plugin signature (for production deployments)
    if pm.requireSignedPlugins {
        if err := pm.verifyPluginSignature(plugin); err != nil {
            return fmt.Errorf("plugin signature verification failed: %w", err)
        }
    }
    
    // Validate requested permissions
    capabilities := plugin.Capabilities()
    for _, endpoint := range capabilities.APIEndpoints {
        if err := pm.validateAPIPermissions(endpoint); err != nil {
            return fmt.Errorf("API endpoint %s permission denied: %w", endpoint.Path, err)
        }
    }
    
    // Check resource requirements
    if err := pm.validateResourceRequirements(plugin, config); err != nil {
        return fmt.Errorf("resource validation failed: %w", err)
    }
    
    return nil
}
```

This unified plugin architecture enables CloudWorkstation to be extended for specialized research workflows while maintaining security, stability, and performance. Institutions can develop custom plugins for their specific needs (HPC integration, specialized analytics, custom authentication) while maintaining compatibility with core CloudWorkstation functionality.