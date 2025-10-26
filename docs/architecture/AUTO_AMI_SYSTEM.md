# Auto-AMI Compilation & Update System

## Overview

The Auto-AMI system provides intelligent background compilation of popular templates and automatic rebuilding when base OS images are updated with security patches. This ensures optimal performance (fast launches) while maintaining security and reliability.

## Core Components

### **Auto-AMI Manager Architecture**

```go
// AutoAMIManager handles intelligent AMI lifecycle management
type AutoAMIManager struct {
    compiler           *TemplateCompiler
    usageTracker      *TemplateUsageTracker
    securityMonitor   *BaseImageMonitor
    costOptimizer     *CompilationCostOptimizer
    
    // Configuration
    popularityThreshold int           // Compilations triggered after N launches
    regionPriority     []string       // Regions to compile first
    scheduleWindow     string         // "02:00-06:00" for off-peak building
    securityWindow     time.Duration  // Max age before security rebuild (7 days)
    
    // State tracking
    compilationQueue   *CompilationQueue
    activeBuilds      map[string]*BuildStatus
    amiInventory      *AMIInventory
}

// BuildTrigger represents different reasons for AMI compilation
type BuildTrigger string
const (
    TriggerPopularity    BuildTrigger = "popularity"     // Usage-based auto-compilation
    TriggerSecurity      BuildTrigger = "security"      // Security patch available
    TriggerScheduled     BuildTrigger = "scheduled"     // Periodic rebuild
    TriggerManual        BuildTrigger = "manual"        // User-requested
    TriggerDependency    BuildTrigger = "dependency"    // Base AMI updated
)

// CompilationRequest enhanced with trigger information
type CompilationRequest struct {
    Template        *Template         `json:"template"`
    Trigger         BuildTrigger      `json:"trigger"`
    Priority        int              `json:"priority"`        // Higher = more urgent
    MaxCost         float64          `json:"max_cost"`        // Budget limit
    
    // Security update information
    BaseImageUpdates []BaseImageUpdate `json:"base_image_updates,omitempty"`
    SecurityPatches  []SecurityPatch   `json:"security_patches,omitempty"`
    
    // Scheduling
    ScheduledFor    time.Time        `json:"scheduled_for,omitempty"`
    Deadline        time.Time        `json:"deadline,omitempty"`
}

type BaseImageUpdate struct {
    ImageID          string    `json:"image_id"`
    PreviousImageID  string    `json:"previous_image_id"`
    UpdateType       string    `json:"update_type"`      // "security", "feature", "maintenance"
    SecurityLevel    string    `json:"security_level"`   // "critical", "high", "medium", "low"
    UpdatedAt        time.Time `json:"updated_at"`
    ChangelogURL     string    `json:"changelog_url,omitempty"`
}
```

### **Base Image Monitoring System**

```go
// BaseImageMonitor tracks upstream OS image updates
type BaseImageMonitor struct {
    ec2Client      *ec2.Client
    ssmClient      *ssm.Client
    
    // Monitored base images
    trackedImages  map[string]*TrackedImage
    updateChannel  chan BaseImageUpdate
    
    // Configuration
    scanInterval   time.Duration  // How often to check for updates
    securityWindow time.Duration  // Max age before forcing rebuild
}

type TrackedImage struct {
    Name            string                    `json:"name"`            // "ubuntu-22.04", "rocky-9"
    AWSImageFilter  *ec2.DescribeImagesInput `json:"aws_filter"`      // AWS filter for finding latest
    CurrentImageID  string                   `json:"current_image_id"`
    LastChecked     time.Time                `json:"last_checked"`
    
    // Templates using this base image
    DependentTemplates []string              `json:"dependent_templates"`
    
    // Update tracking
    LastUpdate      time.Time                `json:"last_update"`
    UpdateHistory   []BaseImageUpdate        `json:"update_history"`
}

// MonitorBaseImages continuously monitors for OS updates
func (monitor *BaseImageMonitor) MonitorBaseImages(ctx context.Context) {
    ticker := time.NewTicker(monitor.scanInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            monitor.checkForUpdates()
        }
    }
}

func (monitor *BaseImageMonitor) checkForUpdates() {
    for _, tracked := range monitor.trackedImages {
        latestImage, err := monitor.findLatestImage(tracked.AWSImageFilter)
        if err != nil {
            log.Printf("Error checking for updates to %s: %v", tracked.Name, err)
            continue
        }
        
        if latestImage.ImageId != tracked.CurrentImageID {
            // New base image available
            update := BaseImageUpdate{
                ImageID:         latestImage.ImageId,
                PreviousImageID: tracked.CurrentImageID,
                UpdateType:      monitor.classifyUpdate(latestImage),
                SecurityLevel:   monitor.assessSecurityLevel(latestImage),
                UpdatedAt:       time.Now(),
                ChangelogURL:    monitor.getChangelogURL(latestImage),
            }
            
            // Update tracking
            tracked.CurrentImageID = latestImage.ImageId
            tracked.LastUpdate = time.Now()
            tracked.UpdateHistory = append(tracked.UpdateHistory, update)
            
            // Trigger dependent template rebuilds
            monitor.updateChannel <- update
            
            log.Printf("Base image update detected: %s -> %s (%s security level)", 
                update.PreviousImageID, update.ImageID, update.SecurityLevel)
        }
    }
}
```

### **Intelligent Compilation Scheduling**

```go
// CompilationScheduler manages when and how to build AMIs
type CompilationScheduler struct {
    autoAMI        *AutoAMIManager
    costOptimizer  *CompilationCostOptimizer
    
    // Scheduling rules
    offPeakWindow  TimeWindow    // When to do bulk compilations
    urgentWindow   TimeWindow    // When to do urgent security builds
    budgetLimits   BudgetLimits  // Cost constraints
}

type TimeWindow struct {
    StartHour int    // 2 (2 AM)
    EndHour   int    // 6 (6 AM) 
    Timezone  string // "UTC", "America/New_York"
}

type BudgetLimits struct {
    DailyBudget    float64  // Maximum daily compilation cost
    MonthlyBudget  float64  // Maximum monthly compilation cost
    UrgentBudget   float64  // Emergency budget for critical security updates
}

// ScheduleCompilation determines optimal timing for AMI builds
func (scheduler *CompilationScheduler) ScheduleCompilation(req *CompilationRequest) (*ScheduledBuild, error) {
    now := time.Now()
    
    // Determine priority and urgency
    priority := scheduler.calculatePriority(req)
    urgency := scheduler.calculateUrgency(req)
    
    var scheduledTime time.Time
    var budgetSource string
    
    switch {
    case urgency == UrgencyCritical:
        // Critical security updates: build immediately
        scheduledTime = now
        budgetSource = "urgent"
        
    case urgency == UrgencyHigh:
        // High priority: build within next urgent window
        scheduledTime = scheduler.nextUrgentWindow()
        budgetSource = "daily"
        
    case priority >= PriorityHigh:
        // Popular templates: build during next off-peak window  
        scheduledTime = scheduler.nextOffPeakWindow()
        budgetSource = "daily"
        
    default:
        // Low priority: build when budget allows
        scheduledTime = scheduler.nextBudgetAvailableTime()
        budgetSource = "monthly"
    }
    
    return &ScheduledBuild{
        Request:       req,
        ScheduledFor:  scheduledTime,
        Priority:      priority,
        Urgency:       urgency,
        BudgetSource:  budgetSource,
        EstimatedCost: scheduler.estimateBuildCost(req),
    }, nil
}
```

### **Security-Driven Auto-Updates**

```go
// SecurityUpdateManager handles security-driven AMI rebuilds
type SecurityUpdateManager struct {
    monitor        *BaseImageMonitor
    scheduler      *CompilationScheduler
    notifier       *UpdateNotifier
    
    // Security policies
    criticalWindow time.Duration  // Max delay for critical updates (4 hours)
    highWindow     time.Duration  // Max delay for high updates (24 hours)
    mediumWindow   time.Duration  // Max delay for medium updates (7 days)
}

// ProcessSecurityUpdate handles incoming security updates
func (sum *SecurityUpdateManager) ProcessSecurityUpdate(update BaseImageUpdate) error {
    // Find all templates using this base image
    affectedTemplates := sum.findAffectedTemplates(update.ImageID)
    
    // Determine urgency based on security level
    urgency := sum.mapSecurityLevelToUrgency(update.SecurityLevel)
    
    // Create compilation requests for affected templates
    for _, template := range affectedTemplates {
        // Check if template has active AMIs that need updating
        activeAMIs := sum.findActiveAMIs(template.Name)
        if len(activeAMIs) == 0 {
            continue // No active AMIs to update
        }
        
        req := &CompilationRequest{
            Template:         template,
            Trigger:          TriggerSecurity,
            Priority:         sum.calculateSecurityPriority(update.SecurityLevel),
            MaxCost:          sum.getSecurityBudget(update.SecurityLevel),
            BaseImageUpdates: []BaseImageUpdate{update},
            
            // Security-driven scheduling
            Deadline: sum.calculateSecurityDeadline(update.SecurityLevel),
        }
        
        // Schedule compilation
        scheduled, err := sum.scheduler.ScheduleCompilation(req)
        if err != nil {
            return fmt.Errorf("failed to schedule security update for %s: %w", template.Name, err)
        }
        
        // Notify users about pending update
        sum.notifySecurityUpdate(template, update, scheduled)
        
        log.Printf("Scheduled security update for template %s (security level: %s, ETA: %v)", 
            template.Name, update.SecurityLevel, scheduled.ScheduledFor)
    }
    
    return nil
}

func (sum *SecurityUpdateManager) calculateSecurityDeadline(securityLevel string) time.Time {
    now := time.Now()
    
    switch securityLevel {
    case "critical":
        return now.Add(sum.criticalWindow)
    case "high":
        return now.Add(sum.highWindow)
    case "medium":
        return now.Add(sum.mediumWindow)
    default:
        return now.Add(30 * 24 * time.Hour) // 30 days for low priority
    }
}
```

## CLI Integration

### **Auto-AMI Management Commands**

```bash
# Configure auto-AMI settings
prism templates auto-ami configure
# Auto-AMI Configuration:
# ├── Popularity threshold: 5 launches
# ├── Off-peak window: 2:00-6:00 AM UTC
# ├── Security update window: 4 hours (critical), 24 hours (high)
# ├── Daily compilation budget: $25.00
# └── Monthly compilation budget: $500.00

# View auto-compilation status
prism templates auto-ami status
# AUTO-AMI STATUS
# 
# Popular Templates (auto-compilation enabled):
# ├── python-ml: ✓ Compiled (last: 2 days ago)
# ├── r-research: ⏳ Queued for tonight (2:30 AM)
# └── deep-learning-gpu: ⚠️ Pending security update (critical)
# 
# Recent Activity:
# ├── ubuntu-22.04 base image updated (security: high)
# ├── 3 templates scheduled for rebuild
# └── Estimated rebuild completion: 6:00 AM UTC

# Force immediate compilation (emergency)
prism templates auto-ami build python-ml --urgent --reason "critical-security-patch"
# Emergency compilation initiated for python-ml
# Trigger: critical-security-patch
# Estimated cost: $8.50
# ETA: 25 minutes
# Progress: aws.prism.cli/templates/build/python-ml-urgent-abc123

# Security update notifications
prism templates auto-ami security-status
# SECURITY UPDATE STATUS
# 
# Critical Updates Available:
# └── ubuntu-22.04: CVE-2024-1234 (kernel vulnerability)
#     ├── Affected templates: python-ml, r-research, ubuntu-basic
#     ├── Auto-rebuild scheduled: 30 minutes
#     └── Manual rebuild: prism templates auto-ami build --security-update
# 
# Recent Security Updates:
# ├── rocky-9: Security patches applied (completed 3 hours ago)  
# └── amazon-linux-2023: Maintenance update (completed yesterday)
```

### **User Notifications and Control**

```bash
# Get notifications about AMI updates
prism notifications list --type ami-updates
# AMI UPDATE NOTIFICATIONS
# 
# ⚠️  Security Update Available (2 hours ago)
#     Base image: ubuntu-22.04 
#     Security level: HIGH
#     Affected templates: python-ml, ubuntu-basic
#     Auto-rebuild: Tonight at 2:30 AM
#     Manual rebuild: prism templates auto-ami build --security-update
# 
# ✓  Auto-compilation Complete (5 hours ago)
#     Template: r-research
#     Trigger: popularity (launched 8 times this week)
#     New AMI: ami-0abc123def (us-west-2)
#     Launch time improvement: 6 minutes → 45 seconds

# User preferences for auto-updates
prism templates auto-ami preferences
# AUTO-AMI PREFERENCES
# 
# Security Updates:
# ├── Critical: Auto-rebuild immediately ✓
# ├── High: Auto-rebuild within 24 hours ✓  
# ├── Medium: Auto-rebuild within 7 days ✓
# └── Low: Manual approval required ✓
# 
# Popularity-based Compilation:
# ├── Enable for frequently used templates ✓
# ├── Threshold: 5 launches per week
# └── Off-peak building only ✓
# 
# Budget Controls:
# ├── Daily compilation budget: $25.00
# └── Approve emergency security builds ✓

# Override auto-AMI for specific template
prism templates auto-ami disable python-ml --reason "prefer-fresh-builds"
prism templates auto-ami enable python-ml --popularity-threshold 3
```

## Integration Examples

### **Educational Institution Configuration**

```yaml
# Auto-AMI policy for university deployment
auto_ami_policy:
  # Semester preparation
  seasonal_precompilation:
    enabled: true
    schedule: "2 weeks before semester start"
    templates: ["python-basic", "r-stats", "java-dev", "data-science"]
    
  # Security update policy  
  security_updates:
    critical_window: "1 hour"     # Rebuild critical updates within 1 hour
    high_window: "6 hours"        # High priority updates within 6 hours  
    medium_window: "24 hours"     # Medium updates daily
    
    # Notification preferences
    notify_instructors: true      # Alert course instructors
    notify_students: false        # Don't alarm students
    
  # Budget controls
  daily_budget: 100.00           # Higher budget during semester prep
  emergency_budget: 250.00       # For critical security updates
  
  # Timing optimization
  build_window: "01:00-05:00"    # Build during minimal usage
  semester_prep_window: "summer" # Heavy building during summer break
```

### **Research Lab Configuration**

```yaml
# Auto-AMI policy for research lab
auto_ami_policy:
  # Research-focused settings
  popularity_threshold: 3        # Lower threshold for research templates
  
  # Security prioritization
  security_updates:
    critical_window: "immediate"  # Zero tolerance for critical vulnerabilities
    high_window: "2 hours"       # Fast response for research security
    
  # Cost optimization for grant funding
  budget_tracking: true         # Track compilation costs against grants
  prefer_off_peak: true         # Minimize costs during off-peak hours
  
  # Research continuity
  preserve_running_instances: true  # Don't force restart running research
  notify_before_rebuild: true       # Warn researchers about updates
```

This auto-AMI system ensures Prism environments remain secure and performant while minimizing disruption to research workflows. The intelligent scheduling and cost optimization make it practical for both educational institutions and research organizations with varying budget constraints and security requirements.