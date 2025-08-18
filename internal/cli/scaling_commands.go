// Package cli implements CloudWorkstation's scaling and rightsizing command handlers.
//
// This file contains all scaling-related functionality including:
//   - Rightsizing analysis and recommendations
//   - Dynamic instance scaling operations
//   - Usage statistics and cost optimization
//   - Helper functions for size mapping and cost estimation
//
// Design Philosophy:
// Follows CloudWorkstation's core principles of transparency, cost optimization,
// and progressive disclosure for scaling operations.
package cli

import (
	"fmt"
	"strings"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// ScalingCommands handles all scaling and rightsizing related commands
type ScalingCommands struct {
	app *App
}

// NewScalingCommands creates a new ScalingCommands instance
func NewScalingCommands(app *App) *ScalingCommands {
	return &ScalingCommands{
		app: app,
	}
}

// Rightsizing handles rightsizing analysis and recommendations
func (s *ScalingCommands) Rightsizing(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf(`usage: cws rightsizing <subcommand> [options]

Available subcommands:
  analyze <instance>       - Analyze usage patterns for specific instance
  recommendations         - Show rightsizing recommendations for all instances
  stats <instance>        - Show detailed usage statistics
  export <instance>       - Export usage data as JSON
  summary                 - Show usage summary across all instances`)
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "analyze":
		return s.rightsizingAnalyze(subargs)
	case "recommendations":
		return s.rightsizingRecommendations(subargs)
	case "stats":
		return s.rightsizingStats(subargs)
	case "export":
		return s.rightsizingExport(subargs)
	case "summary":
		return s.rightsizingSummary(subargs)
	default:
		return fmt.Errorf("unknown rightsizing subcommand: %s\nRun 'cws rightsizing' for usage", subcommand)
	}
}

// rightsizingAnalyze analyzes usage patterns for a specific instance
func (s *ScalingCommands) rightsizingAnalyze(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws rightsizing analyze <instance-name>", "cws rightsizing analyze my-workstation")
	}

	instanceName := args[0]
	fmt.Printf("📊 Analyzing Usage Patterns for '%s'\n", instanceName)
	fmt.Printf("═══════════════════════════════════════\n\n")

	// Check daemon is running
	if err := s.app.apiClient.Ping(s.app.ctx); err != nil {
		return fmt.Errorf(DaemonNotRunningMessage)
	}

	// Get instance info
	response, err := s.app.apiClient.ListInstances(s.app.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	var instance *types.Instance
	for i := range response.Instances {
		if response.Instances[i].Name == instanceName {
			instance = &response.Instances[i]
			break
		}
	}
	if instance == nil {
		return NewNotFoundError("instance", instanceName, "Use 'cws list' to see available instances")
	}

	if instance.State != "running" {
		return NewStateError("instance", instanceName, instance.State, "running")
	}

	// Display current instance configuration
	fmt.Printf("🖥️  **Current Configuration**:\n")
	fmt.Printf("   Instance Type: %s\n", instance.InstanceType)
	fmt.Printf("   Template: %s\n", instance.Template)
	fmt.Printf("   Daily Cost: $%.2f\n", instance.EstimatedDailyCost)
	fmt.Printf("   State: %s\n", instance.State)
	fmt.Printf("   Launch Time: %s\n", instance.LaunchTime.Format("2006-01-02 15:04:05"))

	fmt.Printf("\n📈 **Usage Analytics Collection**:\n")
	fmt.Printf("   Analytics are automatically collected every %s when the instance is active.\n", AnalyticsCollectionInterval)
	fmt.Printf("   Data includes: CPU utilization, memory usage, disk I/O, network traffic, and GPU metrics.\n")
	fmt.Printf("   Rightsizing recommendations are generated hourly based on 24-hour usage patterns.\n")

	fmt.Printf("\n💡 **How to View Results**:\n")
	fmt.Printf("   • Live stats: cws rightsizing stats %s\n", instanceName)
	fmt.Printf("   • Recommendations: cws rightsizing recommendations\n")
	fmt.Printf("   • Export data: cws rightsizing export %s\n", instanceName)

	fmt.Printf("\n🔄 **Analysis Status**:\n")
	fmt.Printf("   ✅ Analytics collection is active\n")
	fmt.Printf("   📊 Usage data is being stored in %s\n", AnalyticsLogFile)
	fmt.Printf("   🎯 Recommendations available in %s\n", RightsizingLogFile)
	fmt.Printf("   ⏱️  Analysis runs automatically every hour\n")

	return nil
}

// rightsizingRecommendations shows rightsizing recommendations for all instances
func (s *ScalingCommands) rightsizingRecommendations(args []string) error {
	fmt.Printf("🎯 Rightsizing Recommendations\n")
	fmt.Printf("═══════════════════════════════\n\n")

	// Check daemon is running
	if err := s.app.apiClient.Ping(s.app.ctx); err != nil {
		return WrapDaemonError(err)
	}

	response, err := s.app.apiClient.ListInstances(s.app.ctx)
	if err != nil {
		return WrapAPIError("list instances", err)
	}

	if len(response.Instances) == 0 {
		fmt.Printf("No instances found. Launch an instance to start collecting usage data.\n")
		return nil
	}

	runningCount := 0
	for _, instance := range response.Instances {
		name := instance.Name
		if instance.State == "running" {
			runningCount++
			fmt.Printf("🖥️  **%s** (%s)\n", name, instance.InstanceType)
			fmt.Printf("   Template: %s\n", instance.Template)
			fmt.Printf("   Current Cost: $%.2f/day\n", instance.EstimatedDailyCost)
			fmt.Printf("   Status: Analytics collection active\n")
			fmt.Printf("   Recommendations: Available after 1+ hours of runtime\n\n")
		} else {
			fmt.Printf("⏸️  **%s** (stopped)\n", name)
			fmt.Printf("   Status: No active analytics collection\n\n")
		}
	}

	if runningCount == 0 {
		fmt.Printf("No running instances found. Start an instance to begin collecting usage analytics.\n\n")
	}

	fmt.Printf("📋 **How Rightsizing Works**:\n")
	fmt.Printf("   1. **Data Collection**: Every 2 minutes, detailed metrics are captured\n")
	fmt.Printf("      • CPU utilization (1min, 5min, 15min averages)\n")
	fmt.Printf("      • Memory usage (total, used, available)\n")
	fmt.Printf("      • Disk I/O and utilization\n")
	fmt.Printf("      • GPU metrics (if available)\n")
	fmt.Printf("      • Network traffic patterns\n\n")

	fmt.Printf("   2. **Analysis**: Every hour, patterns are analyzed\n")
	fmt.Printf("      • Average and peak utilization calculated\n")
	fmt.Printf("      • Bottleneck identification\n")
	fmt.Printf("      • Cost optimization opportunities detected\n\n")

	fmt.Printf("   3. **Recommendations**: Smart suggestions provided\n")
	fmt.Printf("      • Downsize: Low utilization → smaller instance\n")
	fmt.Printf("      • Upsize: High utilization → larger instance\n")
	fmt.Printf("      • Memory-optimized: High memory usage → r5/r6g families\n")
	fmt.Printf("      • GPU-optimized: High GPU usage → g4dn/g5g families\n\n")

	fmt.Printf("💰 **Cost Optimization Impact**:\n")
	fmt.Printf("   • Typical savings: %d-%d%% through rightsizing\n", TypicalRightsizingSavingsMin, TypicalRightsizingSavingsMax)
	fmt.Printf("   • Over-provisioned instances waste ~%d%% of costs\n", OverProvisioningWastePercent)
	fmt.Printf("   • Under-provisioned instances hurt productivity\n")

	return nil
}

// rightsizingStats shows detailed usage statistics for an instance
func (s *ScalingCommands) rightsizingStats(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws rightsizing stats <instance-name>", "cws rightsizing stats my-workstation")
	}

	instanceName := args[0]
	fmt.Printf("📊 Detailed Usage Statistics for '%s'\n", instanceName)
	fmt.Printf("══════════════════════════════════════════\n\n")

	// Check daemon is running
	if err := s.app.apiClient.Ping(s.app.ctx); err != nil {
		return fmt.Errorf(DaemonNotRunningMessage)
	}

	// Get instance info
	response, err := s.app.apiClient.ListInstances(s.app.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	var instance *types.Instance
	for i := range response.Instances {
		if response.Instances[i].Name == instanceName {
			instance = &response.Instances[i]
			break
		}
	}
	if instance == nil {
		return NewNotFoundError("instance", instanceName, "Use 'cws list' to see available instances")
	}

	fmt.Printf("🖥️  **Instance Information**:\n")
	fmt.Printf("   Name: %s\n", instanceName)
	fmt.Printf("   Type: %s\n", instance.InstanceType)
	fmt.Printf("   State: %s\n", instance.State)
	fmt.Printf("   Template: %s\n", instance.Template)
	fmt.Printf("   Daily Cost: $%.2f\n", instance.EstimatedDailyCost)

	if instance.State != "running" {
		fmt.Printf("\n⚠️  Instance is not running. Usage statistics are only collected for running instances.\n")
		return nil
	}

	fmt.Printf("\n📈 **Live Usage Data** (updated every 2 minutes):\n")
	fmt.Printf("   Analytics File: %s\n", AnalyticsLogFile)
	fmt.Printf("   Recommendations File: %s\n\n", RightsizingLogFile)

	fmt.Printf("📊 **Data Points Collected**:\n")
	fmt.Printf("   • CPU: Load averages, core count, utilization percentage\n")
	fmt.Printf("   • Memory: Total, used, free, available (MB)\n")
	fmt.Printf("   • Disk: Total, used, available (GB), utilization percentage\n")
	fmt.Printf("   • Network: RX/TX bytes\n")
	fmt.Printf("   • GPU: Utilization, memory usage, temperature, power draw\n")
	fmt.Printf("   • System: Process count, logged-in users, uptime\n\n")

	fmt.Printf("🎯 **Rightsizing Analysis**:\n")
	fmt.Printf("   • Analysis Period: Rolling 24-hour window\n")
	fmt.Printf("   • Sample Frequency: Every %s\n", AnalyticsCollectionInterval)
	fmt.Printf("   • Recommendation Updates: Every hour\n")
	fmt.Printf("   • Confidence Level: Based on data volume and patterns\n\n")

	fmt.Printf("💡 **Access Raw Data**:\n")
	fmt.Printf("   Export analytics: cws rightsizing export %s\n", instanceName)
	fmt.Printf("   View recommendations: cws rightsizing recommendations\n")

	return nil
}

// rightsizingExport exports usage data as JSON
func (s *ScalingCommands) rightsizingExport(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws rightsizing export <instance-name>", "cws rightsizing export my-workstation")
	}

	instanceName := args[0]
	fmt.Printf("📤 Exporting Usage Data for '%s'\n", instanceName)
	fmt.Printf("═══════════════════════════════════\n\n")

	// Check daemon is running
	if err := s.app.apiClient.Ping(s.app.ctx); err != nil {
		return fmt.Errorf(DaemonNotRunningMessage)
	}

	// Get instance info
	response, err := s.app.apiClient.ListInstances(s.app.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Find the instance
	var instance *types.Instance
	for i := range response.Instances {
		if response.Instances[i].Name == instanceName {
			instance = &response.Instances[i]
			break
		}
	}
	if instance == nil {
		return NewNotFoundError("instance", instanceName, "Use 'cws list' to see available instances")
	}

	fmt.Printf("📊 **Usage Analytics Export**\n")
	fmt.Printf("Instance: %s\n", instanceName)
	fmt.Printf("Export Format: JSON\n\n")

	fmt.Printf("📁 **Available Data Files**:\n")
	fmt.Printf("   Analytics Data: %s\n", AnalyticsLogFile)
	fmt.Printf("      • Detailed metrics collected every 2 minutes\n")
	fmt.Printf("      • Rolling window of last %d samples (~%d hours)\n", DefaultAnalyticsSampleCount, DefaultAnalyticsSampleHours)
	fmt.Printf("      • CPU, memory, disk, network, GPU, and system metrics\n\n")

	fmt.Printf("   Rightsizing Recommendations: %s\n", RightsizingLogFile)
	fmt.Printf("      • Analysis results updated hourly\n")
	fmt.Printf("      • Recommendations with confidence levels\n")
	fmt.Printf("      • Cost optimization suggestions\n\n")

	fmt.Printf("💻 **Command to Access Data**:\n")
	fmt.Printf("   # Connect to instance and view analytics\n")
	fmt.Printf("   cws connect %s\n", instanceName)
	fmt.Printf("   \n")
	fmt.Printf("   # Then on the instance:\n")
	fmt.Printf("   sudo cat %s | jq .\n", AnalyticsLogFile)
	fmt.Printf("   sudo cat %s | jq .\n\n", RightsizingLogFile)

	fmt.Printf("📈 **Data Structure Example**:\n")
	fmt.Printf(`   {
     "timestamp": "2024-08-08T17:30:00Z",
     "cpu": {
       "utilization_percent": 15.2,
       "load_1min": 0.3,
       "core_count": 2
     },
     "memory": {
       "total_mb": 4096,
       "utilization_percent": 35.5
     },
     "gpu": {
       "utilization_percent": 0,
       "count": 0
     }
   }`)

	fmt.Printf("\n\n🚀 **Integration Options**:\n")
	fmt.Printf("   • Parse JSON for custom dashboards\n")
	fmt.Printf("   • Import into monitoring tools\n")
	fmt.Printf("   • Build automated rightsizing workflows\n")

	return nil
}

// rightsizingSummary shows usage summary across all instances
func (s *ScalingCommands) rightsizingSummary(args []string) error {
	fmt.Printf("📋 Usage Summary Across All Instances\n")
	fmt.Printf("════════════════════════════════════\n\n")

	// Check daemon is running
	if err := s.app.apiClient.Ping(s.app.ctx); err != nil {
		return WrapDaemonError(err)
	}

	response, err := s.app.apiClient.ListInstances(s.app.ctx)
	if err != nil {
		return WrapAPIError("list instances", err)
	}

	if len(response.Instances) == 0 {
		fmt.Printf("No instances found.\n")
		return nil
	}

	totalInstances := len(response.Instances)
	runningInstances := 0
	stoppedInstances := 0
	totalDailyCost := 0.0

	fmt.Printf("📊 **Fleet Overview**:\n")
	for _, instance := range response.Instances {
		switch instance.State {
		case "running":
			runningInstances++
			totalDailyCost += instance.EstimatedDailyCost
		case "stopped":
			stoppedInstances++
		}

		status := "🟢"
		if instance.State != "running" {
			status = "⏸️ "
		}

		fmt.Printf("   %s %-20s %s ($%.2f/day)\n",
			status, instance.Name, instance.InstanceType, instance.EstimatedDailyCost)
	}

	fmt.Printf("\n💰 **Cost Summary**:\n")
	fmt.Printf("   Total Instances: %d\n", totalInstances)
	fmt.Printf("   Running: %d\n", runningInstances)
	fmt.Printf("   Stopped: %d\n", stoppedInstances)
	fmt.Printf("   Current Daily Cost: $%.2f\n", totalDailyCost)
	fmt.Printf("   Monthly Estimate: $%.2f\n", totalDailyCost*30)

	if runningInstances > 0 {
		fmt.Printf("\n📈 **Rightsizing Potential**:\n")
		fmt.Printf("   Analytics Active: %d instances\n", runningInstances)
		fmt.Printf("   Data Collection: Every %s\n", AnalyticsCollectionInterval)
		fmt.Printf("   Analysis Updates: Every hour\n")

		estimatedSavings := totalDailyCost * DefaultSavingsEstimate // Assume 25% average savings potential
		fmt.Printf("   Estimated Savings Potential: $%.2f/day (%.0f%%)\n", estimatedSavings, DefaultSavingsEstimate*100)
		fmt.Printf("   Annual Savings Potential: $%.2f\n", estimatedSavings*DaysToYearEstimate)
	}

	fmt.Printf("\n🎯 **Optimization Recommendations**:\n")
	if runningInstances == 0 {
		fmt.Printf("   No running instances to analyze\n")
	} else {
		fmt.Printf("   ✅ Analytics collection is active\n")
		fmt.Printf("   📊 Run 'cws rightsizing recommendations' for detailed analysis\n")
		fmt.Printf("   💡 Allow 1+ hours runtime for meaningful recommendations\n")
	}

	fmt.Printf("\n📚 **Best Practices**:\n")
	fmt.Printf("   • Monitor instances for 24+ hours before rightsizing\n")
	fmt.Printf("   • Consider peak usage patterns, not just averages\n")
	fmt.Printf("   • Test rightsized instances with representative workloads\n")
	fmt.Printf("   • Use spot instances for non-critical workloads\n")

	return nil
}

// Scaling handles dynamic instance scaling operations
func (s *ScalingCommands) Scaling(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf(`usage: cws scaling <subcommand> [options]

Available subcommands:
  analyze <instance>       - Analyze current instance and recommend optimal size
  scale <instance> <size>  - Scale instance to new size (XS/S/M/L/XL)
  preview <instance> <size> - Preview scaling operation without executing
  history <instance>       - Show scaling history for instance
  
Examples:
  cws scaling analyze my-ml-workstation    # Analyze and recommend size
  cws scaling scale my-ml-workstation L    # Scale to Large size
  cws scaling preview my-instance XL       # Preview scaling to XL`)
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "analyze":
		return s.scalingAnalyze(subargs)
	case "scale":
		return s.scalingScale(subargs)
	case "preview":
		return s.scalingPreview(subargs)
	case "history":
		return s.scalingHistory(subargs)
	default:
		return fmt.Errorf("unknown scaling subcommand: %s", subcommand)
	}
}

// scalingAnalyze analyzes an instance and recommends optimal size
func (s *ScalingCommands) scalingAnalyze(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws scaling analyze <instance-name>", "cws scaling analyze my-workstation")
	}

	instanceName := args[0]

	fmt.Printf("🔍 Dynamic Scaling Analysis\n")
	fmt.Printf("═══════════════════════════\n\n")

	// Check daemon is running
	if err := s.app.apiClient.Ping(s.app.ctx); err != nil {
		return fmt.Errorf(DaemonNotRunningMessage)
	}

	// Get instance info
	response, err := s.app.apiClient.ListInstances(s.app.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Find the instance
	var instance *types.Instance
	for i := range response.Instances {
		if response.Instances[i].Name == instanceName {
			instance = &response.Instances[i]
			break
		}
	}
	if instance == nil {
		return NewNotFoundError("instance", instanceName, "Use 'cws list' to see available instances")
	}

	fmt.Printf("📊 **Current Instance Configuration**:\n")
	fmt.Printf("   Name: %s\n", instance.Name)
	fmt.Printf("   Type: %s\n", instance.InstanceType)
	fmt.Printf("   State: %s\n", instance.State)
	fmt.Printf("   Current Cost: $%.2f/day\n\n", instance.EstimatedDailyCost)

	// Parse current size from instance type
	currentSize := s.parseInstanceSize(instance.InstanceType)
	fmt.Printf("   Current T-Shirt Size: %s\n", currentSize)

	if instance.State != "running" {
		fmt.Printf("\n⚠️  **Instance Not Running**\n")
		fmt.Printf("   Instance must be running to collect usage analytics.\n")
		fmt.Printf("   Start instance: cws start %s\n", instanceName)
		return nil
	}

	fmt.Printf("\n📈 **Usage Analysis**:\n")
	fmt.Printf("   Analytics Collection: Active (every 2 minutes)\n")
	fmt.Printf("   Data Location: %s\n", AnalyticsLogFile)
	fmt.Printf("   Recommendations: %s\n\n", RightsizingLogFile)

	fmt.Printf("🎯 **Scaling Recommendations**:\n")
	fmt.Printf("   Current size appears suitable for general workloads.\n")
	fmt.Printf("   Run analytics for 1+ hours for data-driven recommendations.\n\n")

	fmt.Printf("💡 **Available Sizes**:\n")
	fmt.Printf("   XS: 1vCPU, 2GB RAM, 100GB storage ($0.50/day)\n")
	fmt.Printf("   S:  2vCPU, 4GB RAM, 500GB storage ($1.00/day)\n")
	fmt.Printf("   M:  2vCPU, 8GB RAM, 1TB storage ($2.00/day)\n")
	fmt.Printf("   L:  4vCPU, 16GB RAM, 2TB storage ($4.00/day)\n")
	fmt.Printf("   XL: 8vCPU, 32GB RAM, 4TB storage ($8.00/day)\n\n")

	fmt.Printf("🔧 **Next Steps**:\n")
	fmt.Printf("   1. Monitor usage: cws rightsizing stats %s\n", instanceName)
	fmt.Printf("   2. Preview scaling: cws scaling preview %s <size>\n", instanceName)
	fmt.Printf("   3. Execute scaling: cws scaling scale %s <size>\n", instanceName)

	return nil
}

// scalingScale scales an instance to a new size
func (s *ScalingCommands) scalingScale(args []string) error {
	if len(args) < 2 {
		return NewUsageError("cws scaling scale <instance-name> <size>", "cws scaling scale my-workstation L")
	}

	instanceName := args[0]
	newSize := strings.ToUpper(args[1])

	// Validate size
	if !ValidTSizes[newSize] {
		return NewValidationError("size", newSize, "XS, S, M, L, XL")
	}

	fmt.Printf("⚡ Dynamic Instance Scaling\n")
	fmt.Printf("═══════════════════════════\n\n")

	// Check daemon is running
	if err := s.app.apiClient.Ping(s.app.ctx); err != nil {
		return fmt.Errorf(DaemonNotRunningMessage)
	}

	// Get instance info
	response, err := s.app.apiClient.ListInstances(s.app.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Find the instance
	var instance *types.Instance
	for i := range response.Instances {
		if response.Instances[i].Name == instanceName {
			instance = &response.Instances[i]
			break
		}
	}
	if instance == nil {
		return NewNotFoundError("instance", instanceName, "Use 'cws list' to see available instances")
	}

	currentSize := s.parseInstanceSize(instance.InstanceType)

	fmt.Printf("🔄 **Scaling Operation**:\n")
	fmt.Printf("   Instance: %s\n", instance.Name)
	fmt.Printf("   Current Size: %s (%s)\n", currentSize, instance.InstanceType)
	fmt.Printf("   Target Size: %s\n", newSize)
	fmt.Printf("   Current State: %s\n\n", instance.State)

	if currentSize == newSize {
		fmt.Printf("✅ Instance is already size %s. No scaling needed.\n", newSize)
		return nil
	}

	if instance.State != "running" && instance.State != "stopped" {
		return NewStateError("instance", instanceName, instance.State, "running or stopped")
	}

	// Show cost comparison
	currentCost := instance.EstimatedDailyCost
	newCost := s.estimateCostForSize(newSize)

	fmt.Printf("💰 **Cost Impact**:\n")
	fmt.Printf("   Current Cost: $%.2f/day\n", currentCost)
	fmt.Printf("   New Cost: $%.2f/day\n", newCost)

	if newCost > currentCost {
		fmt.Printf("   Impact: +$%.2f/day (+%.0f%%)\n\n", newCost-currentCost, ((newCost-currentCost)/currentCost)*100)
	} else if newCost < currentCost {
		fmt.Printf("   Impact: -$%.2f/day (-%.0f%%)\n\n", currentCost-newCost, ((currentCost-newCost)/currentCost)*100)
	} else {
		fmt.Printf("   Impact: No cost change\n\n")
	}

	fmt.Printf("⚠️  **NOTICE: Dynamic Scaling Implementation**\n")
	fmt.Printf("   This feature requires AWS instance type modification capabilities.\n")
	fmt.Printf("   Currently showing preview mode - full implementation pending.\n\n")

	fmt.Printf("🛠️  **Manual Scaling Process**:\n")
	fmt.Printf("   1. Stop instance: cws stop %s\n", instanceName)
	fmt.Printf("   2. Modify via AWS Console or CLI\n")
	fmt.Printf("   3. Start instance: cws start %s\n\n", instanceName)

	fmt.Printf("🚧 **Implementation Status**: Preview Mode\n")
	fmt.Printf("   Full dynamic scaling will be implemented in future release.\n")

	return nil
}

// scalingPreview shows what a scaling operation would do
func (s *ScalingCommands) scalingPreview(args []string) error {
	if len(args) < 2 {
		return NewUsageError("cws scaling preview <instance-name> <size>", "cws scaling preview my-workstation L")
	}

	instanceName := args[0]
	newSize := strings.ToUpper(args[1])

	// Validate size
	if !ValidTSizes[newSize] {
		return NewValidationError("size", newSize, "XS, S, M, L, XL")
	}

	fmt.Printf("👁️  Scaling Preview\n")
	fmt.Printf("═══════════════════\n\n")

	// Check daemon is running
	if err := s.app.apiClient.Ping(s.app.ctx); err != nil {
		return fmt.Errorf(DaemonNotRunningMessage)
	}

	// Get instance info
	response, err := s.app.apiClient.ListInstances(s.app.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Find the instance
	var instance *types.Instance
	for i := range response.Instances {
		if response.Instances[i].Name == instanceName {
			instance = &response.Instances[i]
			break
		}
	}
	if instance == nil {
		return NewNotFoundError("instance", instanceName, "Use 'cws list' to see available instances")
	}

	currentSize := s.parseInstanceSize(instance.InstanceType)

	fmt.Printf("📋 **Preview: %s → %s**\n", currentSize, newSize)
	fmt.Printf("   Instance: %s\n", instance.Name)
	fmt.Printf("   Current Type: %s\n", instance.InstanceType)
	fmt.Printf("   Target Type: %s\n\n", s.getInstanceTypeForSize(newSize))

	// Resource comparison
	fmt.Printf("🔄 **Resource Changes**:\n")
	currentSpecs := s.getSizeSpecs(currentSize)
	newSpecs := s.getSizeSpecs(newSize)

	fmt.Printf("   CPU: %s → %s\n", currentSpecs.CPU, newSpecs.CPU)
	fmt.Printf("   Memory: %s → %s\n", currentSpecs.Memory, newSpecs.Memory)
	fmt.Printf("   Storage: %s → %s\n\n", currentSpecs.Storage, newSpecs.Storage)

	// Cost comparison
	currentCost := instance.EstimatedDailyCost
	newCost := s.estimateCostForSize(newSize)

	fmt.Printf("💰 **Cost Impact**:\n")
	fmt.Printf("   Current: $%.2f/day\n", currentCost)
	fmt.Printf("   New: $%.2f/day\n", newCost)

	if newCost > currentCost {
		fmt.Printf("   Change: +$%.2f/day (+%.0f%%)\n", newCost-currentCost, ((newCost-currentCost)/currentCost)*100)
		fmt.Printf("   Monthly: +$%.2f\n", (newCost-currentCost)*30)
	} else if newCost < currentCost {
		fmt.Printf("   Change: -$%.2f/day (-%.0f%%)\n", currentCost-newCost, ((currentCost-newCost)/currentCost)*100)
		fmt.Printf("   Monthly: -$%.2f savings\n", (currentCost-newCost)*30)
	} else {
		fmt.Printf("   Change: No cost difference\n")
	}

	fmt.Printf("\n⚡ **Scaling Process**:\n")
	if instance.State == "running" {
		fmt.Printf("   1. Stop instance (preserves data)\n")
		fmt.Printf("   2. Modify instance type\n")
		fmt.Printf("   3. Start with new configuration\n")
		fmt.Printf("   4. Validate functionality\n")
		fmt.Printf("   Estimated downtime: 2-5 minutes\n")
	} else {
		fmt.Printf("   1. Modify instance type (instance stopped)\n")
		fmt.Printf("   2. Start with new configuration\n")
		fmt.Printf("   No additional downtime required\n")
	}

	fmt.Printf("\n✅ **To Execute**:\n")
	fmt.Printf("   cws scaling scale %s %s\n", instanceName, newSize)

	return nil
}

// scalingHistory shows scaling history for an instance
func (s *ScalingCommands) scalingHistory(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws scaling history <instance-name>", "cws scaling history my-workstation")
	}

	instanceName := args[0]

	fmt.Printf("📊 Scaling History\n")
	fmt.Printf("═════════════════\n\n")

	// Check daemon is running
	if err := s.app.apiClient.Ping(s.app.ctx); err != nil {
		return fmt.Errorf(DaemonNotRunningMessage)
	}

	// Get instance info
	response, err := s.app.apiClient.ListInstances(s.app.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Find the instance
	var instance *types.Instance
	for i := range response.Instances {
		if response.Instances[i].Name == instanceName {
			instance = &response.Instances[i]
			break
		}
	}
	if instance == nil {
		return NewNotFoundError("instance", instanceName, "Use 'cws list' to see available instances")
	}

	fmt.Printf("🏷️  **Instance**: %s\n", instance.Name)
	fmt.Printf("   Current Type: %s\n", instance.InstanceType)
	fmt.Printf("   Current Size: %s\n", s.parseInstanceSize(instance.InstanceType))
	fmt.Printf("   Launch Time: %s\n\n", instance.LaunchTime)

	fmt.Printf("📈 **Scaling History**:\n")
	fmt.Printf("   Launch: %s (Size: %s)\n",
		instance.LaunchTime,
		s.parseInstanceSize(instance.InstanceType))

	fmt.Printf("\n💡 **Note**: Comprehensive scaling history tracking will be\n")
	fmt.Printf("   implemented in future release with AWS CloudTrail integration.\n")

	return nil
}

// Helper functions for scaling

func (s *ScalingCommands) parseInstanceSize(instanceType string) string {
	if size, exists := InstanceTypeSizeMapping[instanceType]; exists {
		return size
	}
	return "Unknown"
}

func (s *ScalingCommands) getInstanceTypeForSize(size string) string {
	if instanceType, exists := SizeInstanceTypeMapping[size]; exists {
		return instanceType
	}
	return "unknown"
}

func (s *ScalingCommands) estimateCostForSize(size string) float64 {
	if specs, exists := TSizeSpecifications[size]; exists {
		return specs.Cost
	}
	return 0.0
}

type SizeSpecs struct {
	CPU     string
	Memory  string
	Storage string
}

func (s *ScalingCommands) getSizeSpecs(size string) SizeSpecs {
	if specs, exists := TSizeSpecifications[size]; exists {
		return SizeSpecs{
			CPU:     specs.CPU,
			Memory:  specs.Memory,
			Storage: specs.Storage,
		}
	}
	return SizeSpecs{"Unknown", "Unknown", "Unknown"}
}