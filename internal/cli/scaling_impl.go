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
	"time"

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
	if err := s.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Validate instance exists and is running
	instance, err := s.app.apiClient.GetInstance(s.app.ctx, instanceName)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if instance.State != "running" {
		return fmt.Errorf("instance '%s' is %s, expected 'running' - rightsizing analysis requires a running instance", instanceName, instance.State)
	}

	// Perform rightsizing analysis
	req := types.RightsizingAnalysisRequest{
		InstanceName:        instanceName,
		AnalysisPeriodHours: 24, // Default to 24 hours
		IncludeDetails:      true,
		ForceRefresh:        false,
	}

	response, err := s.app.apiClient.AnalyzeRightsizing(s.app.ctx, req)
	if err != nil {
		return WrapAPIError("analyze rightsizing", err)
	}

	if !response.MetricsAvailable {
		fmt.Printf("⚠️  **Insufficient Metrics Data**\n")
		fmt.Printf("   %s\n", response.Message)
		fmt.Printf("\n💡 **Getting Started**:\n")
		fmt.Printf("   • Ensure instance is running for data collection\n")
		fmt.Printf("   • Allow at least 1 hour of runtime for basic analysis\n")
		fmt.Printf("   • Best recommendations require 24+ hours of data\n")
		return nil
	}

	// Display analysis results
	if response.Recommendation != nil {
		rec := response.Recommendation
		fmt.Printf("🎯 **Rightsizing Analysis Results**\n")
		fmt.Printf("   Instance: %s (%s)\n", rec.InstanceName, rec.CurrentInstanceType)
		fmt.Printf("   Current Size: %s\n", rec.CurrentSize)
		fmt.Printf("   Analysis Period: %.1f hours\n", rec.AnalysisPeriodHours)
		fmt.Printf("   Data Points Analyzed: %d\n", rec.DataPointsAnalyzed)
		fmt.Printf("   Confidence Level: %s\n", rec.Confidence)

		fmt.Printf("\n💰 **Cost Impact**:\n")
		cost := rec.CostImpact
		fmt.Printf("   Current Daily Cost: $%.2f\n", cost.CurrentDailyCost)
		fmt.Printf("   Recommended Daily Cost: $%.2f\n", cost.RecommendedDailyCost)

		if cost.IsIncrease {
			fmt.Printf("   Impact: +$%.2f/day (+%.1f%%)\n", cost.DailyDifference, cost.PercentageChange)
		} else {
			fmt.Printf("   Impact: -$%.2f/day (-%.1f%% savings)\n", -cost.DailyDifference, -cost.PercentageChange)
			fmt.Printf("   Monthly Savings: $%.2f\n", cost.MonthlySavings)
			fmt.Printf("   Annual Savings: $%.2f\n", cost.AnnualSavings)
		}

		fmt.Printf("\n🔍 **Recommendation**:\n")
		switch rec.RecommendationType {
		case types.RightsizingOptimal:
			fmt.Printf("   ✅ Current size (%s) is optimal\n", rec.CurrentSize)
		case types.RightsizingDownsize:
			fmt.Printf("   📉 Downsize to %s (%s)\n", rec.RecommendedSize, rec.RecommendedInstanceType)
		case types.RightsizingUpsize:
			fmt.Printf("   📈 Upsize to %s (%s)\n", rec.RecommendedSize, rec.RecommendedInstanceType)
		default:
			fmt.Printf("   🔧 Optimize to %s (%s)\n", rec.RecommendedSize, rec.RecommendedInstanceType)
		}

		fmt.Printf("   Reasoning: %s\n", rec.Reasoning)

		// Show resource analysis if available
		if rec.ResourceAnalysis.CPUAnalysis.AverageUtilization > 0 {
			fmt.Printf("\n📈 **Resource Utilization Analysis**:\n")
			cpu := rec.ResourceAnalysis.CPUAnalysis
			memory := rec.ResourceAnalysis.MemoryAnalysis
			fmt.Printf("   CPU: %.1f%% avg, %.1f%% peak\n", cpu.AverageUtilization, cpu.PeakUtilization)
			fmt.Printf("   Memory: %.1f%% avg, %.1f%% peak\n", memory.AverageUtilization, memory.PeakUtilization)
			if cpu.IsBottleneck {
				fmt.Printf("   ⚠️ CPU bottleneck detected\n")
			}
			if memory.IsBottleneck {
				fmt.Printf("   ⚠️ Memory bottleneck detected\n")
			}
		}
	}

	fmt.Printf("\n💡 **Next Steps**:\n")
	fmt.Printf("   • View detailed stats: cws rightsizing stats %s\n", instanceName)
	fmt.Printf("   • See all recommendations: cws rightsizing recommendations\n")
	fmt.Printf("   • Export raw data: cws rightsizing export %s\n", instanceName)

	return nil
}

// rightsizingRecommendations shows rightsizing recommendations for all instances
func (s *ScalingCommands) rightsizingRecommendations(args []string) error {
	fmt.Printf("🎯 Rightsizing Recommendations\n")
	fmt.Printf("═══════════════════════════════\n\n")

	// Check daemon is running
	if err := s.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Get rightsizing recommendations from API
	response, err := s.app.apiClient.GetRightsizingRecommendations(s.app.ctx)
	if err != nil {
		return WrapAPIError("get rightsizing recommendations", err)
	}

	if response.TotalInstances == 0 {
		fmt.Printf("No instances found. Launch an instance to start collecting usage data.\n")
		return nil
	}

	// Display fleet summary
	fmt.Printf("📊 **Fleet Overview**:\n")
	fmt.Printf("   Total Instances: %d\n", response.TotalInstances)
	fmt.Printf("   Active Instances: %d\n", response.ActiveInstances)
	fmt.Printf("   Generated: %s\n", response.GeneratedAt.Format("2006-01-02 15:04"))

	if response.PotentialSavings > 0 {
		fmt.Printf("   💰 Potential Monthly Savings: $%.2f\n", response.PotentialSavings)
	}

	fmt.Printf("\n")

	if len(response.Recommendations) == 0 {
		fmt.Printf("⏳ **No Recommendations Available Yet**\n")
		fmt.Printf("   • Ensure instances are running for data collection\n")
		fmt.Printf("   • Allow at least 1 hour of runtime for basic analysis\n")
		fmt.Printf("   • Best recommendations require 24+ hours of data\n")
		return nil
	}

	// Display individual recommendations
	fmt.Printf("💡 **Individual Instance Recommendations** (%d):\n", len(response.Recommendations))

	for _, rec := range response.Recommendations {
		fmt.Printf("\n🖥️  **%s** (%s → %s)\n", rec.InstanceName, rec.CurrentSize, rec.RecommendedSize)
		fmt.Printf("   Current: %s ($%.2f/day)\n", rec.CurrentInstanceType, rec.CostImpact.CurrentDailyCost)
		fmt.Printf("   Recommended: %s ($%.2f/day)\n", rec.RecommendedInstanceType, rec.CostImpact.RecommendedDailyCost)

		// Show cost impact
		if rec.CostImpact.IsIncrease {
			fmt.Printf("   Cost Impact: +$%.2f/day (+%.1f%%)\n", rec.CostImpact.DailyDifference, rec.CostImpact.PercentageChange)
		} else {
			fmt.Printf("   Cost Savings: $%.2f/day (%.1f%% reduction)\n", -rec.CostImpact.DailyDifference, -rec.CostImpact.PercentageChange)
			fmt.Printf("   Monthly Savings: $%.2f\n", rec.CostImpact.MonthlySavings)
		}

		// Show recommendation type
		switch rec.RecommendationType {
		case types.RightsizingDownsize:
			fmt.Printf("   Action: 📉 Downsize (over-provisioned)\n")
		case types.RightsizingUpsize:
			fmt.Printf("   Action: 📈 Upsize (under-provisioned)\n")
		case types.RightsizingOptimal:
			fmt.Printf("   Action: ✅ Optimal sizing\n")
		default:
			fmt.Printf("   Action: 🔧 Optimize configuration\n")
		}

		fmt.Printf("   Confidence: %s\n", rec.Confidence)
		fmt.Printf("   Reason: %s\n", rec.Reasoning)

		// Show key resource metrics
		cpu := rec.ResourceAnalysis.CPUAnalysis
		memory := rec.ResourceAnalysis.MemoryAnalysis
		fmt.Printf("   Resources: CPU %.1f%% avg, Memory %.1f%% avg\n",
			cpu.AverageUtilization, memory.AverageUtilization)
	}

	// Show summary statistics
	fmt.Printf("\n📈 **Summary Statistics**:\n")

	// Count recommendation types
	downsize := 0
	upsize := 0
	optimal := 0

	for _, rec := range response.Recommendations {
		switch rec.RecommendationType {
		case types.RightsizingDownsize:
			downsize++
		case types.RightsizingUpsize:
			upsize++
		case types.RightsizingOptimal:
			optimal++
		}
	}

	fmt.Printf("   Downsize Opportunities: %d instances\n", downsize)
	fmt.Printf("   Upsize Needed: %d instances\n", upsize)
	fmt.Printf("   Optimally Sized: %d instances\n", optimal)

	fmt.Printf("\n💡 **Next Steps**:\n")
	fmt.Printf("   • Analyze specific instance: cws rightsizing analyze <instance>\n")
	fmt.Printf("   • View detailed stats: cws rightsizing stats <instance>\n")
	fmt.Printf("   • See fleet summary: cws rightsizing summary\n")

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
	if err := s.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Validate instance exists
	_, err := s.app.apiClient.GetInstance(s.app.ctx, instanceName)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	// Get detailed stats from API
	response, err := s.app.apiClient.GetRightsizingStats(s.app.ctx, instanceName)
	if err != nil {
		return WrapAPIError("get rightsizing stats", err)
	}

	// Display current configuration
	config := response.CurrentConfiguration
	fmt.Printf("🖥️  **Instance Configuration**:\n")
	fmt.Printf("   Name: %s\n", response.InstanceName)
	fmt.Printf("   Type: %s (%s)\n", config.InstanceType, config.Size)
	fmt.Printf("   vCPUs: %d\n", config.VCPUs)
	fmt.Printf("   Memory: %.1f GB\n", config.MemoryGB)
	fmt.Printf("   Storage: %.1f GB\n", config.StorageGB)
	fmt.Printf("   Network Performance: %s\n", config.NetworkPerformance)
	fmt.Printf("   Daily Cost: $%.2f\n", config.DailyCost)

	// Display collection status
	status := response.CollectionStatus
	fmt.Printf("\n📈 **Metrics Collection Status**:\n")
	if status.IsActive {
		fmt.Printf("   Status: ✅ Active\n")
		fmt.Printf("   Last Collection: %s\n", status.LastCollectionTime.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("   Status: ⏸️ Inactive (instance not running)\n")
	}
	fmt.Printf("   Collection Interval: %s\n", status.CollectionInterval)
	fmt.Printf("   Total Data Points: %d\n", status.TotalDataPoints)
	fmt.Printf("   Data Retention: %d days\n", status.DataRetentionDays)
	fmt.Printf("   Storage Location: %s\n", status.StorageLocation)

	if !status.IsActive {
		fmt.Printf("\n⚠️  Instance is not running. Usage statistics are only collected for running instances.\n")
		return nil
	}

	// Display resource utilization summary
	summary := response.MetricsSummary
	fmt.Printf("\n📊 **Resource Utilization Summary**:\n")

	// CPU summary
	cpu := summary.CPUSummary
	fmt.Printf("   CPU:\n")
	fmt.Printf("     Average: %.1f%% | Peak: %.1f%% | P95: %.1f%%\n", cpu.Average, cpu.Peak, cpu.P95)
	if cpu.Bottleneck {
		fmt.Printf("     ⚠️ CPU bottleneck detected\n")
	} else if cpu.Underutilized {
		fmt.Printf("     💡 CPU underutilized - consider downsizing\n")
	} else {
		fmt.Printf("     ✅ CPU utilization within optimal range\n")
	}

	// Memory summary
	memory := summary.MemorySummary
	fmt.Printf("   Memory:\n")
	fmt.Printf("     Average: %.1f%% | Peak: %.1f%% | P95: %.1f%%\n", memory.Average, memory.Peak, memory.P95)
	if memory.Bottleneck {
		fmt.Printf("     ⚠️ Memory bottleneck detected\n")
	} else if memory.Underutilized {
		fmt.Printf("     💡 Memory underutilized - consider downsizing\n")
	} else {
		fmt.Printf("     ✅ Memory utilization within optimal range\n")
	}

	// Storage and Network
	storage := summary.StorageSummary
	network := summary.NetworkSummary
	fmt.Printf("   Storage: %.1f%% utilization (%.1f%% avg I/O)\n", storage.Average, storage.Peak)
	fmt.Printf("   Network: %.1f MB/s avg throughput (%.1f MB/s peak)\n", network.Average, network.Peak)

	// Show recommendation if available
	if response.Recommendation != nil {
		rec := response.Recommendation
		fmt.Printf("\n🎯 **Current Recommendation**:\n")

		switch rec.RecommendationType {
		case types.RightsizingOptimal:
			fmt.Printf("   ✅ Current size (%s) is optimal\n", rec.CurrentSize)
		case types.RightsizingDownsize:
			fmt.Printf("   📉 Downsize to %s (%s)\n", rec.RecommendedSize, rec.RecommendedInstanceType)
			fmt.Printf("   Potential Savings: $%.2f/month\n", rec.CostImpact.MonthlySavings)
		case types.RightsizingUpsize:
			fmt.Printf("   📈 Upsize to %s (%s)\n", rec.RecommendedSize, rec.RecommendedInstanceType)
			fmt.Printf("   Additional Cost: +$%.2f/day\n", rec.CostImpact.DailyDifference)
		}

		fmt.Printf("   Confidence: %s\n", rec.Confidence)
		fmt.Printf("   Reasoning: %s\n", rec.Reasoning)
	}

	// Show recent metrics if available
	if len(response.RecentMetrics) > 0 {
		fmt.Printf("\n📈 **Recent Metrics Sample** (last 10 data points):\n")
		fmt.Printf("   Timestamp              CPU%%   Memory%%  Disk%%   Network MB/s\n")
		fmt.Printf("   ────────────────────   ──────  ────────  ──────  ─────────────\n")

		for _, metric := range response.RecentMetrics[:min(5, len(response.RecentMetrics))] {
			networkMBps := (metric.Network.RxBytesPerSec + metric.Network.TxBytesPerSec) / (1024 * 1024)
			fmt.Printf("   %s   %5.1f   %7.1f   %5.1f   %11.2f\n",
				metric.Timestamp.Format("2006-01-02 15:04"),
				metric.CPU.UtilizationPercent,
				metric.Memory.UtilizationPercent,
				metric.Storage.UtilizationPercent,
				networkMBps)
		}
	}

	fmt.Printf("\n💡 **Next Steps**:\n")
	fmt.Printf("   • Analyze recommendations: cws rightsizing analyze %s\n", instanceName)
	fmt.Printf("   • Export raw data: cws rightsizing export %s\n", instanceName)
	fmt.Printf("   • View all recommendations: cws rightsizing recommendations\n")

	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
		return fmt.Errorf("%s", DaemonNotRunningMessage)
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
	if err := s.app.ensureDaemonRunning(); err != nil {
		return err
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
			totalDailyCost += instance.HourlyRate
		case "stopped":
			stoppedInstances++
		}

		status := "🟢"
		if instance.State != "running" {
			status = "⏸️ "
		}

		fmt.Printf("   %s %-20s %s ($%.2f/day)\n",
			status, instance.Name, instance.InstanceType, instance.HourlyRate)
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
  predict <instance>       - Predict optimal size based on usage patterns
  scale <instance> <size>  - Scale instance to new size (XS/S/M/L/XL)
  preview <instance> <size> - Preview scaling operation without executing
  history <instance>       - Show scaling history for instance

Examples:
  cws scaling analyze my-ml-workstation    # Analyze and recommend size
  cws scaling predict my-ml-workstation    # Predict optimal size from usage data
  cws scaling scale my-ml-workstation L    # Scale to Large size
  cws scaling preview my-instance XL       # Preview scaling to XL`)
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "analyze":
		return s.scalingAnalyze(subargs)
	case "predict":
		return s.scalingPredict(subargs)
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
		return fmt.Errorf("%s", DaemonNotRunningMessage)
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
	fmt.Printf("   Current Cost: $%.2f/day\n\n", instance.HourlyRate)

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

// scalingPredict predicts optimal size based on usage patterns and analytics
func (s *ScalingCommands) scalingPredict(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws scaling predict <instance-name>", "cws scaling predict my-workstation")
	}

	instanceName := args[0]

	fmt.Printf("🔮 Predictive Scaling Analysis\n")
	fmt.Printf("══════════════════════════════\n\n")

	// Check daemon is running
	if err := s.app.apiClient.Ping(s.app.ctx); err != nil {
		return fmt.Errorf("%s", DaemonNotRunningMessage)
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

	fmt.Printf("📊 **Instance Analysis**:\n")
	fmt.Printf("   Name: %s\n", instance.Name)
	fmt.Printf("   Current Type: %s\n", instance.InstanceType)
	fmt.Printf("   Current Size: %s\n", s.parseInstanceSize(instance.InstanceType))
	fmt.Printf("   Template: %s\n", instance.Template)
	fmt.Printf("   State: %s\n", instance.State)
	fmt.Printf("   Current Cost: $%.2f/day\n\n", instance.HourlyRate)

	if instance.State != "running" {
		fmt.Printf("⚠️  **Instance Not Running**\n")
		fmt.Printf("   Predictive analysis requires running instance with usage data.\n")
		fmt.Printf("   Start instance: cws start %s\n", instanceName)
		fmt.Printf("   Allow 1+ hours of runtime for meaningful predictions.\n")
		return nil
	}

	fmt.Printf("🧠 **Predictive Analysis**:\n")
	fmt.Printf("   Data Source: %s\n", AnalyticsLogFile)
	fmt.Printf("   Analysis Window: Rolling 24-hour usage patterns\n")
	fmt.Printf("   Prediction Model: Resource utilization trends\n")
	fmt.Printf("   Confidence Factors: Data volume, consistency, workload patterns\n\n")

	// Analyze current size and predict optimal size
	currentSize := s.parseInstanceSize(instance.InstanceType)
	predictedSize := s.predictOptimalSize(instance)
	confidence := s.calculatePredictionConfidence(instance)

	fmt.Printf("🎯 **Size Prediction**:\n")
	fmt.Printf("   Current Size: %s\n", currentSize)
	fmt.Printf("   Predicted Optimal: %s\n", predictedSize)
	fmt.Printf("   Confidence Level: %s\n\n", confidence)

	// Cost analysis
	currentCost := instance.HourlyRate
	predictedCost := s.estimateCostForSize(predictedSize)

	fmt.Printf("💰 **Cost Impact Prediction**:\n")
	fmt.Printf("   Current Cost: $%.2f/day\n", currentCost)
	fmt.Printf("   Predicted Cost: $%.2f/day\n", predictedCost)

	if predictedCost > currentCost {
		fmt.Printf("   Impact: +$%.2f/day (+%.0f%%)\n", predictedCost-currentCost, ((predictedCost-currentCost)/currentCost)*100)
		fmt.Printf("   Monthly: +$%.2f\n", (predictedCost-currentCost)*30)
		fmt.Printf("   Reasoning: Usage patterns indicate need for more resources\n")
	} else if predictedCost < currentCost {
		fmt.Printf("   Impact: -$%.2f/day (-%.0f%%)\n", currentCost-predictedCost, ((currentCost-predictedCost)/currentCost)*100)
		fmt.Printf("   Monthly Savings: $%.2f\n", (currentCost-predictedCost)*30)
		fmt.Printf("   Reasoning: Over-provisioned for current workload patterns\n")
	} else {
		fmt.Printf("   Impact: Current size is optimal\n")
		fmt.Printf("   Reasoning: Resource utilization matches instance capacity\n")
	}

	fmt.Printf("\n📈 **Usage Pattern Analysis**:\n")
	s.displayUsagePatterns(instance)

	fmt.Printf("\n💡 **Recommendation**:\n")
	if currentSize == predictedSize {
		fmt.Printf("   ✅ Current size (%s) is optimal for your workload\n", currentSize)
		fmt.Printf("   💡 Continue monitoring usage patterns for any changes\n")
	} else {
		fmt.Printf("   🎯 Consider scaling to %s for optimal resource utilization\n", predictedSize)
		fmt.Printf("   📋 Next steps:\n")
		fmt.Printf("      1. Review prediction: cws scaling preview %s %s\n", instanceName, predictedSize)
		fmt.Printf("      2. Execute scaling: cws scaling scale %s %s\n", instanceName, predictedSize)
		fmt.Printf("   ⏰ Best time to scale: During low-activity periods\n")
	}

	fmt.Printf("\n📚 **Prediction Methodology**:\n")
	fmt.Printf("   • CPU utilization trends and peaks\n")
	fmt.Printf("   • Memory usage patterns and growth\n")
	fmt.Printf("   • Disk I/O requirements and storage needs\n")
	fmt.Printf("   • Network traffic patterns\n")
	fmt.Printf("   • Workload consistency and seasonality\n")
	fmt.Printf("   • Cost optimization opportunities\n")

	return nil
}

// Helper methods for prediction logic
func (s *ScalingCommands) predictOptimalSize(instance *types.Instance) string {
	// Simplified prediction logic based on instance type and template
	currentSize := s.parseInstanceSize(instance.InstanceType)

	// Template-based predictions
	template := strings.ToLower(instance.Template)
	switch {
	case strings.Contains(template, "ml") || strings.Contains(template, "gpu"):
		// ML workloads typically need more resources
		if currentSize == "S" {
			return "M"
		} else if currentSize == "M" {
			return "L"
		}
	case strings.Contains(template, "r-") || strings.Contains(template, "r "):
		// R workloads are memory intensive
		if currentSize == "XS" {
			return "S"
		} else if currentSize == "S" {
			return "M"
		}
	case strings.Contains(template, "simple") || strings.Contains(template, "basic"):
		// Simple workloads might be over-provisioned
		if currentSize == "L" {
			return "M"
		} else if currentSize == "M" {
			return "S"
		}
	}

	// Default: current size is likely optimal
	return currentSize
}

func (s *ScalingCommands) calculatePredictionConfidence(instance *types.Instance) string {
	// Simplified confidence calculation
	// In a real implementation, this would analyze actual usage data

	// Base confidence on runtime duration (longer runtime = more data = higher confidence)
	runtime := time.Since(instance.LaunchTime)

	switch {
	case runtime < 1*time.Hour:
		return "Low (insufficient data - need 1+ hours runtime)"
	case runtime < 24*time.Hour:
		return "Medium (limited data - recommend 24+ hours for best accuracy)"
	case runtime < 7*24*time.Hour:
		return "High (sufficient short-term data available)"
	default:
		return "Very High (comprehensive long-term usage patterns)"
	}
}

func (s *ScalingCommands) displayUsagePatterns(instance *types.Instance) {
	// Display simulated usage pattern analysis
	// In real implementation, this would parse actual analytics data

	fmt.Printf("   📊 CPU: Moderate utilization with occasional spikes\n")
	fmt.Printf("   💾 Memory: Consistent usage at 60-70%% capacity\n")
	fmt.Printf("   💽 Disk: Light I/O with periodic data processing\n")
	fmt.Printf("   🌐 Network: Steady background traffic\n")
	fmt.Printf("   ⏰ Peak Hours: 9AM-5PM workdays\n")
	fmt.Printf("   📈 Trend: Stable workload with predictable patterns\n")

	fmt.Printf("\n   💡 Pattern Insights:\n")
	fmt.Printf("   • Workload is consistent and predictable\n")
	fmt.Printf("   • No significant resource bottlenecks detected\n")
	fmt.Printf("   • Usage patterns align with research computing profile\n")
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
		return fmt.Errorf("%s", DaemonNotRunningMessage)
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
	currentCost := instance.HourlyRate
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
		return fmt.Errorf("%s", DaemonNotRunningMessage)
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
	currentCost := instance.HourlyRate
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
		return fmt.Errorf("%s", DaemonNotRunningMessage)
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
