package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// RightsizingCobraCommands provides Cobra-based rightsizing command structure
type RightsizingCobraCommands struct {
	app *App
}

// NewRightsizingCobraCommands creates rightsizing commands
func NewRightsizingCobraCommands(app *App) *RightsizingCobraCommands {
	return &RightsizingCobraCommands{app: app}
}

// CreateRightsizingCommand creates the main rightsizing command with subcommands
func (r *RightsizingCobraCommands) CreateRightsizingCommand() *cobra.Command {
	rightsizingCmd := &cobra.Command{
		Use:   "rightsizing",
		Short: "Analyze and optimize instance sizes",
		Long: `Analyze usage patterns and provide rightsizing recommendations for cost optimization.

CloudWorkstation rightsizing provides intelligent instance sizing recommendations based on
real usage metrics, helping you optimize costs while maintaining performance.

Key capabilities:
• Instance-specific analysis with detailed resource utilization
• Fleet-wide optimization recommendations
• Cost impact analysis with potential savings calculations
• Resource usage patterns and bottleneck detection
• Export capabilities for reporting and integration`,
	}

	// Add subcommands
	rightsizingCmd.AddCommand(r.createAnalyzeCommand())
	rightsizingCmd.AddCommand(r.createRecommendationsCommand())
	rightsizingCmd.AddCommand(r.createStatsCommand())
	rightsizingCmd.AddCommand(r.createSummaryCommand())
	rightsizingCmd.AddCommand(r.createExportCommand())

	return rightsizingCmd
}

// createAnalyzeCommand creates the analyze subcommand
func (r *RightsizingCobraCommands) createAnalyzeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze <instance-name>",
		Short: "Analyze instance rightsizing recommendations",
		Long: `Perform detailed rightsizing analysis for a specific instance.

This command analyzes CPU, memory, storage, and network utilization patterns
to provide intelligent sizing recommendations with cost impact analysis.

Examples:
  cws rightsizing analyze my-workstation
  cws rightsizing analyze gpu-training --period 168  # 1 week analysis
  cws rightsizing analyze my-server --refresh         # Force refresh metrics`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Convert Cobra flags to args format expected by existing method
			analysisArgs := []string{"analyze", args[0]}

			if period, _ := cmd.Flags().GetFloat64("period"); period > 0 {
				analysisArgs = append(analysisArgs, "--period", fmt.Sprintf("%.1f", period))
			}
			if refresh, _ := cmd.Flags().GetBool("refresh"); refresh {
				analysisArgs = append(analysisArgs, "--refresh")
			}
			if format, _ := cmd.Flags().GetString("format"); format != "" {
				analysisArgs = append(analysisArgs, "--format", format)
			}

			return r.app.Rightsizing(analysisArgs)
		},
	}

	cmd.Flags().Float64("period", 24, "Analysis period in hours (default 24)")
	cmd.Flags().Bool("refresh", false, "Force refresh metrics data")
	cmd.Flags().String("format", "table", "Output format: table, json, yaml")

	return cmd
}

// createRecommendationsCommand creates the recommendations subcommand
func (r *RightsizingCobraCommands) createRecommendationsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recommendations",
		Short: "List all rightsizing recommendations",
		Long: `Display rightsizing recommendations for all instances in your environment.

This command provides a fleet-wide view of rightsizing opportunities with
potential cost savings and optimization recommendations.

Examples:
  cws rightsizing recommendations
  cws rightsizing recommendations --format json
  cws rightsizing recommendations --savings-only      # Show only cost-saving opportunities`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Convert flags to args format
			recArgs := []string{"recommendations"}

			if format, _ := cmd.Flags().GetString("format"); format != "" {
				recArgs = append(recArgs, "--format", format)
			}
			if savingsOnly, _ := cmd.Flags().GetBool("savings-only"); savingsOnly {
				recArgs = append(recArgs, "--savings-only")
			}
			if sortBy, _ := cmd.Flags().GetString("sort-by"); sortBy != "" {
				recArgs = append(recArgs, "--sort-by", sortBy)
			}

			return r.app.Rightsizing(recArgs)
		},
	}

	cmd.Flags().String("format", "table", "Output format: table, json, yaml")
	cmd.Flags().Bool("savings-only", false, "Show only instances with cost-saving opportunities")
	cmd.Flags().String("sort-by", "savings", "Sort by: savings, utilization, confidence, name")

	return cmd
}

// createStatsCommand creates the stats subcommand
func (r *RightsizingCobraCommands) createStatsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats <instance-name>",
		Short: "Show detailed rightsizing statistics",
		Long: `Display comprehensive rightsizing statistics for a specific instance.

This command provides detailed resource utilization statistics, performance
metrics, and comprehensive analysis for informed rightsizing decisions.

Examples:
  cws rightsizing stats my-workstation
  cws rightsizing stats gpu-server --detailed
  cws rightsizing stats my-analysis --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			statsArgs := []string{"stats", args[0]}

			if format, _ := cmd.Flags().GetString("format"); format != "" {
				statsArgs = append(statsArgs, "--format", format)
			}
			if detailed, _ := cmd.Flags().GetBool("detailed"); detailed {
				statsArgs = append(statsArgs, "--detailed")
			}

			return r.app.Rightsizing(statsArgs)
		},
	}

	cmd.Flags().String("format", "table", "Output format: table, json, yaml")
	cmd.Flags().Bool("detailed", false, "Show detailed metrics and analysis")

	return cmd
}

// createSummaryCommand creates the summary subcommand
func (r *RightsizingCobraCommands) createSummaryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Show fleet-wide rightsizing summary",
		Long: `Display a comprehensive fleet-wide rightsizing summary.

This command provides high-level insights across your entire instance fleet,
including total potential savings, resource utilization trends, and optimization
opportunities.

Examples:
  cws rightsizing summary
  cws rightsizing summary --detailed
  cws rightsizing summary --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			summaryArgs := []string{"summary"}

			if format, _ := cmd.Flags().GetString("format"); format != "" {
				summaryArgs = append(summaryArgs, "--format", format)
			}
			if detailed, _ := cmd.Flags().GetBool("detailed"); detailed {
				summaryArgs = append(summaryArgs, "--detailed")
			}

			return r.app.Rightsizing(summaryArgs)
		},
	}

	cmd.Flags().String("format", "table", "Output format: table, json, yaml")
	cmd.Flags().Bool("detailed", false, "Show detailed fleet analysis")

	return cmd
}

// createExportCommand creates the export subcommand
func (r *RightsizingCobraCommands) createExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <instance-name>",
		Short: "Export rightsizing metrics data",
		Long: `Export detailed metrics data for external analysis or reporting.

This command exports raw metrics data in various formats for integration
with external monitoring systems, reporting tools, or custom analysis.

Examples:
  cws rightsizing export my-workstation
  cws rightsizing export gpu-server --format csv --output metrics.csv
  cws rightsizing export my-analysis --period 168  # Export 1 week of data`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			exportArgs := []string{"export", args[0]}

			if format, _ := cmd.Flags().GetString("format"); format != "" {
				exportArgs = append(exportArgs, "--format", format)
			}
			if output, _ := cmd.Flags().GetString("output"); output != "" {
				exportArgs = append(exportArgs, "--output", output)
			}
			if period, _ := cmd.Flags().GetFloat64("period"); period > 0 {
				exportArgs = append(exportArgs, "--period", fmt.Sprintf("%.1f", period))
			}

			return r.app.Rightsizing(exportArgs)
		},
	}

	cmd.Flags().String("format", "json", "Export format: json, csv, yaml")
	cmd.Flags().String("output", "", "Output file path (default: stdout)")
	cmd.Flags().Float64("period", 24, "Data export period in hours")

	return cmd
}