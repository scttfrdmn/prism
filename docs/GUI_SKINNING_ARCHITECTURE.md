# GUI Skinning & Theming Architecture

## Overview

CloudWorkstation's GUI skinning system allows institutions and users to customize the interface appearance, branding, and behavior while maintaining core functionality. This enables institutional branding, accessibility customization, and specialized research workflows.

## Architecture Components

### **Theme System Structure**

```
themes/
├── default/                    # Built-in CloudWorkstation theme
│   ├── theme.json             # Theme configuration
│   ├── colors.json            # Color palette
│   ├── fonts.json             # Typography settings
│   ├── icons/                 # Icon assets
│   ├── layouts/               # Layout definitions
│   └── components/            # Component styling
│
├── university-brand/          # Institutional theme example
│   ├── theme.json
│   ├── colors.json
│   ├── assets/
│   │   ├── logo.png
│   │   ├── background.jpg
│   │   └── favicon.ico
│   └── custom-components/     # Override components
│
└── accessibility/             # High-contrast accessibility theme
    ├── theme.json
    ├── colors.json
    └── accessibility-settings.json
```

### **Theme Configuration Model**

```go
// ThemeManager handles theme loading and application
type ThemeManager struct {
    currentTheme    *Theme
    availableThemes map[string]*Theme
    customAssets    map[string][]byte
    
    // Theme directories
    systemThemePath string  // Built-in themes
    userThemePath   string  // User-installed themes
}

// Theme represents a complete GUI theme
type Theme struct {
    Metadata      ThemeMetadata      `json:"metadata"`
    Colors        ColorPalette       `json:"colors"`
    Typography    Typography         `json:"typography"`
    Layout        LayoutConfig       `json:"layout"`
    Components    ComponentOverrides `json:"components"`
    Assets        AssetReferences    `json:"assets"`
    Behavior      BehaviorConfig     `json:"behavior"`
}

type ThemeMetadata struct {
    Name         string `json:"name"`
    Version      string `json:"version"`
    Author       string `json:"author"`
    Description  string `json:"description"`
    Institution  string `json:"institution,omitempty"`
    
    // Compatibility and requirements
    MinCWSVersion   string   `json:"min_cws_version"`
    SupportedOS     []string `json:"supported_os"`
    RequiredPlugins []string `json:"required_plugins,omitempty"`
}
```

### **Color Palette System**

```json
// colors.json - Institutional branding example
{
  "metadata": {
    "name": "University Research Theme",
    "institution": "Research University"
  },
  
  "palette": {
    // Primary institutional colors
    "primary": "#003366",        // University blue
    "secondary": "#CC9900",      // University gold
    "accent": "#0066CC",         // Accent blue
    
    // Semantic colors
    "success": "#28A745",        // Green for successful operations
    "warning": "#FFC107",        // Yellow for warnings
    "error": "#DC3545",          // Red for errors
    "info": "#17A2B8",           // Blue for information
    
    // Interface colors
    "background": "#FFFFFF",     // Main background
    "surface": "#F8F9FA",        // Card/panel background
    "surface_dark": "#E9ECEF",   // Darker surface variant
    
    // Text colors  
    "text_primary": "#212529",   // Primary text
    "text_secondary": "#6C757D", // Secondary text
    "text_disabled": "#ADB5BD",  // Disabled text
    "text_on_primary": "#FFFFFF", // Text on primary color
    
    // Status indicator colors
    "status_running": "#28A745",    // Running instances
    "status_stopped": "#6C757D",    // Stopped instances  
    "status_error": "#DC3545",      // Error states
    "status_pending": "#FFC107",    // Pending/transitioning
    
    // Cost/budget colors
    "cost_low": "#28A745",         // Low cost (green)
    "cost_medium": "#FFC107",      // Medium cost (yellow)
    "cost_high": "#FD7E14",        // High cost (orange)
    "cost_critical": "#DC3545"     // Critical cost (red)
  },
  
  // Dark mode variants
  "dark_palette": {
    "background": "#1A1A1A",
    "surface": "#2D2D2D",
    "text_primary": "#FFFFFF",
    "text_secondary": "#B0B0B0"
  }
}
```

### **Component Override System**

```go
// ComponentOverrides allows custom component implementations
type ComponentOverrides struct {
    InstanceCard    *ComponentConfig `json:"instance_card,omitempty"`
    TemplateGrid    *ComponentConfig `json:"template_grid,omitempty"`
    StatusBar       *ComponentConfig `json:"status_bar,omitempty"`
    NavigationPanel *ComponentConfig `json:"navigation_panel,omitempty"`
    CostDisplay     *ComponentConfig `json:"cost_display,omitempty"`
}

type ComponentConfig struct {
    CustomImplementation string                 `json:"custom_implementation,omitempty"` // Go plugin path
    StyleOverrides      map[string]interface{} `json:"style_overrides,omitempty"`
    BehaviorOverrides   map[string]interface{} `json:"behavior_overrides,omitempty"`
    LayoutOverrides     map[string]interface{} `json:"layout_overrides,omitempty"`
}
```

### **Institutional Branding Example**

```json
// theme.json - University of Research branding
{
  "metadata": {
    "name": "University of Research Theme", 
    "version": "1.0.0",
    "author": "UofR IT Department",
    "institution": "University of Research",
    "description": "Official CloudWorkstation theme with university branding"
  },
  
  "branding": {
    "institution_name": "University of Research",
    "logo_path": "assets/logo.png",
    "favicon_path": "assets/favicon.ico", 
    "background_image": "assets/campus-background.jpg",
    "website_url": "https://research.university.edu",
    
    // Header customization
    "header_text": "Research Computing Platform",
    "subtitle": "University of Research - IT Services",
    
    // Footer customization  
    "footer_links": [
      {"text": "IT Help Desk", "url": "https://help.university.edu"},
      {"text": "Research Computing", "url": "https://rc.university.edu"},
      {"text": "Privacy Policy", "url": "https://university.edu/privacy"}
    ],
    
    // Contact information
    "support_email": "cloudworkstation@university.edu",
    "support_phone": "+1-555-HELP-RC"
  },
  
  "layout": {
    "show_institution_logo": true,
    "header_height": 80,
    "sidebar_width": 280,
    "compact_mode": false,
    
    // Custom dashboard layout for research workflows
    "dashboard_layout": "research_focused",
    "quick_actions": [
      "launch_template",
      "manage_budgets", 
      "view_shared_storage",
      "contact_support"
    ]
  }
}
```

## Implementation Architecture

### **Theme Loading and Management**

```go
// Enhanced GUI main with theme system
func main() {
    // Initialize theme manager
    themeManager := NewThemeManager()
    
    // Load user preferences or institutional default
    selectedTheme := loadThemePreference()
    if selectedTheme == "" {
        selectedTheme = detectInstitutionalTheme() // Auto-detect from domain/config
    }
    
    err := themeManager.LoadTheme(selectedTheme)
    if err != nil {
        log.Printf("Failed to load theme %s, falling back to default: %v", selectedTheme, err)
        themeManager.LoadTheme("default")
    }
    
    // Create themed application
    app := app.NewWithTheme(themeManager.GetCurrentTheme())
    
    // Apply institutional branding
    if branding := themeManager.GetBranding(); branding != nil {
        app.SetIcon(branding.Logo)
        app.SetTitle(branding.ApplicationTitle)
    }
    
    // Load themed components
    mainWindow := createThemedMainWindow(themeManager)
    
    mainWindow.ShowAndRun()
}

// createThemedMainWindow creates main window with theme applied
func createThemedMainWindow(themeManager *ThemeManager) *wails.App {
    theme := themeManager.GetCurrentTheme()
    
    // Create Wails application with themed properties
    app := wails.New(wails.Options{
        Title:  theme.Branding.ApplicationTitle,
        Width:  theme.Layout.DefaultWidth,
        Height: theme.Layout.DefaultHeight,
    })
    
    // Apply institutional branding through CSS and web assets
    if theme.Branding.BackgroundImage != "" {
        // Set background via CSS variables or web assets
        app.SetCSS(generateThemedCSS(theme))
    }
    
    return app
}
```

### **Component Theming System**

```go
// ThemedInstanceCard creates instance card with applied theme
func ThemedInstanceCard(instance *types.Instance, theme *Theme) *ComponentData {
    // Check for custom component override
    if override := theme.Components.InstanceCard; override != nil && override.CustomImplementation != "" {
        return loadCustomComponent(override.CustomImplementation, instance)
    }
    
    // Use themed default implementation with web-based rendering
    cardData := &ComponentData{
        Name:   instance.Name,
        Status: string(instance.State),
        Cost:   fmt.Sprintf("$%.3f/hour", instance.HourlyRate),
    }
    
    // Apply theme colors through CSS classes
    cardData.StatusColor = getStatusColorClass(instance.State, theme)
    cardData.CostColor = getCostColorClass(instance.HourlyRate, theme)
    cardData.ThemeClass = theme.Components.InstanceCard.StyleOverrides["card_class"]
    
    return cardData
}

// getStatusColorClass returns themed CSS class for instance status
func getStatusColorClass(status types.InstanceState, theme *Theme) string {
    switch status {
    case types.InstanceStateRunning:
        return "status-running"
    case types.InstanceStateStopped:
        return "status-stopped"
    case types.InstanceStateError:
        return "status-error"
    default:
        return "status-pending"
    }
}
```

## Theme Distribution and Management

### **Theme Installation**

```bash
# Install institutional theme
cws gui theme install university-research-theme.cwstheme

# List available themes
cws gui theme list
# THEME                     VERSION   AUTHOR              STATUS
# default                  1.0.0     CloudWorkstation    Active
# university-research      1.2.0     UofR IT             Available
# accessibility-high       1.0.0     CloudWorkstation    Available

# Switch theme
cws gui theme set university-research
# Applied theme: University of Research
# Restart GUI to see changes? [y/N] y

# Create theme from current customizations  
cws gui theme export my-custom-theme
# Exported theme to: ~/.cloudworkstation/themes/my-custom-theme/
```

### **Theme Package Format**

```
university-theme.cwstheme (ZIP format)
├── manifest.json          # Theme metadata and dependencies
├── theme.json            # Main theme configuration
├── colors.json           # Color palette
├── assets/
│   ├── logo.png
│   ├── favicon.ico
│   └── backgrounds/
└── components/           # Optional custom components
    ├── instance-card.so  # Compiled Go plugin
    └── cost-display.so
```

## Accessibility and Customization

### **High-Contrast Accessibility Theme**

```json
{
  "metadata": {
    "name": "High Contrast Accessibility",
    "description": "High contrast theme for visual accessibility"
  },
  
  "accessibility": {
    "high_contrast": true,
    "large_fonts": true,
    "reduced_motion": true,
    "screen_reader_optimized": true
  },
  
  "colors": {
    "background": "#000000",
    "surface": "#1A1A1A", 
    "text_primary": "#FFFFFF",
    "text_secondary": "#CCCCCC",
    "primary": "#00FF00",      // High contrast green
    "error": "#FF0000",       // Pure red for visibility
    "success": "#00FF00"      // Pure green for visibility
  },
  
  "typography": {
    "base_size": 16,          // Larger base font
    "scale_factor": 1.25,     // Larger scaling
    "font_weight": "bold"     // Bolder text
  }
}
```

### **Research Workflow Optimization Theme**

```json
{
  "metadata": {
    "name": "Research Workflow Optimized",
    "description": "Optimized for computational research workflows"
  },
  
  "layout": {
    "dashboard_layout": "research_compact",
    "quick_actions": [
      "launch_gpu_template",
      "check_budget_status",
      "mount_shared_storage", 
      "view_running_jobs"
    ],
    
    // Prioritize information density for researchers
    "show_detailed_costs": true,
    "show_instance_specs": true,
    "show_utilization_graphs": true,
    "compact_template_grid": true
  },
  
  "behavior": {
    "auto_refresh_interval": 30,    // Faster updates for active research
    "default_view": "instances",    // Start with instances, not dashboard
    "confirm_destructive_actions": true,
    "show_advanced_options": true   // Expose advanced features by default
  }
}
```

## Integration with Institutional Identity

### **Automatic Theme Detection**

```go
// Auto-detect institutional theme based on user context
func detectInstitutionalTheme() string {
    // Check for institutional configuration
    if domain := getCurrentUserDomain(); domain != "" {
        if theme := lookupInstitutionalTheme(domain); theme != "" {
            return theme
        }
    }
    
    // Check for profile-based theme
    if profile := getCurrentProfile(); profile != nil {
        if profile.InstitutionalTheme != "" {
            return profile.InstitutionalTheme
        }
    }
    
    // Check environment variable
    if theme := os.Getenv("CWS_THEME"); theme != "" {
        return theme
    }
    
    return "default"
}
```

This GUI skinning system allows institutions to maintain their brand identity while providing researchers with familiar, accessible interfaces optimized for their specific workflows.