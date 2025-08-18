// Security CLI commands for CloudWorkstation
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// Security command handles all security-related operations
func (a *App) SecurityCommand() *cobra.Command {
	securityCmd := &cobra.Command{
		Use:   "security",
		Short: "Security management commands",
		Long:  `Manage CloudWorkstation security features including monitoring, audit logs, and health checks.`,
	}

	// Security status command
	securityCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show security status",
		Long:  `Display comprehensive security status including monitoring, audit logging, and system health.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.SecurityStatus()
		},
	})

	// Security health command
	securityCmd.AddCommand(&cobra.Command{
		Use:   "health",
		Short: "Check security health",
		Long:  `Perform comprehensive security health check and display results.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.SecurityHealth()
		},
	})

	// Security dashboard command
	securityCmd.AddCommand(&cobra.Command{
		Use:   "dashboard",
		Short: "Show security dashboard",
		Long:  `Display real-time security dashboard with threat analysis and recommendations.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.SecurityDashboard()
		},
	})

	// Security correlations command
	securityCmd.AddCommand(&cobra.Command{
		Use:   "correlations",
		Short: "Show security correlations",
		Long:  `Display recent security event correlations and threat analysis.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.SecurityCorrelations()
		},
	})

	// Security keychain command
	securityCmd.AddCommand(&cobra.Command{
		Use:   "keychain",
		Short: "Show keychain information",
		Long:  `Display keychain provider information and diagnostics.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.SecurityKeychain()
		},
	})

	// Security config command
	securityCmd.AddCommand(&cobra.Command{
		Use:   "config",
		Short: "Show security configuration",
		Long:  `Display current security configuration and settings.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.SecurityConfig()
		},
	})

	// AWS compliance commands
	awsComplianceCmd := &cobra.Command{
		Use:   "compliance",
		Short: "AWS compliance validation",
		Long:  `Validate CloudWorkstation against AWS Artifact compliance reports and Service Control Policies.`,
	}

	// AWS compliance validate command
	awsComplianceCmd.AddCommand(&cobra.Command{
		Use:   "validate <framework>",
		Short: "Validate compliance framework",
		Long: `Validate CloudWorkstation against specific compliance framework using AWS Artifact reports.
		
Available frameworks:
  nist-800-171    NIST 800-171 (CUI Protection)
  nist-800-53     NIST 800-53 (Federal/Healthcare Controls)
  soc-2           SOC 2 Type II 
  hipaa           HIPAA (Healthcare - includes NIST 800-53)
  fedramp         FedRAMP (Federal - based on NIST 800-53)
  iso-27001       ISO 27001 (Information Security)
  pci-dss         PCI DSS (Payment Card)
  gdpr            GDPR (Privacy)
  itar            ITAR (Export Control - requires GovCloud)
  ear             EAR (Export Administration)
  cmmc            CMMC (Defense Contractors)
  cmmc-l1         CMMC Level 1 (Basic Cyber Hygiene)
  cmmc-l2         CMMC Level 2 (Intermediate + NIST 800-171)
  cmmc-l3         CMMC Level 3 (Expert + NIST 800-53)
  ferpa           FERPA (Student Privacy in Education)`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.ValidateAWSCompliance(args[0])
		},
	})

	// AWS compliance report command
	awsComplianceCmd.AddCommand(&cobra.Command{
		Use:   "report <framework>",
		Short: "Generate compliance report",
		Long:  `Generate detailed compliance report with AWS Artifact alignment and gap analysis.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.GenerateComplianceReport(args[0])
		},
	})

	// AWS SCP validation command
	awsComplianceCmd.AddCommand(&cobra.Command{
		Use:   "scp <framework>",
		Short: "Validate Service Control Policies",
		Long:  `Check if required Service Control Policies are in place for compliance framework.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.ValidateSCPs(args[0])
		},
	})

	// List supported frameworks
	awsComplianceCmd.AddCommand(&cobra.Command{
		Use:   "frameworks",
		Short: "List supported frameworks",
		Long:  `Display all supported compliance frameworks with descriptions.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.ListComplianceFrameworks()
		},
	})

	securityCmd.AddCommand(awsComplianceCmd)

	return securityCmd
}

// SecurityStatus displays comprehensive security status
func (a *App) SecurityStatus() error {
	fmt.Println("🔒 CloudWorkstation Security Status")
	fmt.Println("═══════════════════════════════════")

	// Make API request to security status endpoint
	resp, err := a.apiClient.MakeRequest("GET", "/api/v1/security/status", nil)
	if err != nil {
		return fmt.Errorf("failed to get security status: %w", err)
	}

	var status map[string]interface{}
	if err := json.Unmarshal(resp, &status); err != nil {
		return fmt.Errorf("failed to parse security status: %w", err)
	}

	// Display security status in user-friendly format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

	_, _ = fmt.Fprintf(w, "Status:\t%v\n", getStatusValue(status, "enabled"))
	_, _ = fmt.Fprintf(w, "Running:\t%v\n", getStatusValue(status, "running"))

	if lastCheck, ok := status["last_health_check"].(string); ok && lastCheck != "" {
		if t, err := time.Parse(time.RFC3339, lastCheck); err == nil {
			_, _ = fmt.Fprintf(w, "Last Health Check:\t%s\n", t.Format("2006-01-02 15:04:05"))
		}
	}

	// Show configuration
	if config, ok := status["configuration"].(map[string]interface{}); ok {
		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "Configuration:")
		_, _ = fmt.Fprintf(w, "  Audit Logging:\t%v\n", config["audit_log_enabled"])
		_, _ = fmt.Fprintf(w, "  Monitoring:\t%v\n", config["monitoring_enabled"])
		_, _ = fmt.Fprintf(w, "  Correlation Analysis:\t%v\n", config["correlation_enabled"])
		_, _ = fmt.Fprintf(w, "  Registry Security:\t%v\n", config["registry_security_enabled"])
	}

	// Show keychain info if available
	if keychain, ok := status["keychain_info"].(map[string]interface{}); ok {
		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "Keychain:")
		_, _ = fmt.Fprintf(w, "  Provider:\t%s\n", keychain["provider"])
		_, _ = fmt.Fprintf(w, "  Native:\t%v\n", keychain["native"])
		_, _ = fmt.Fprintf(w, "  Security Level:\t%s\n", keychain["security_level"])
	}

	return nil
}

// SecurityHealth performs and displays security health check
func (a *App) SecurityHealth() error {
	fmt.Println("🏥 Security Health Check")
	fmt.Println("═══════════════════════")

	// Trigger health check
	_, err := a.apiClient.MakeRequest("POST", "/api/v1/security/health", nil)
	if err != nil {
		return fmt.Errorf("failed to trigger health check: %w", err)
	}

	// Get health status
	resp, err := a.apiClient.MakeRequest("GET", "/api/v1/security/health", nil)
	if err != nil {
		return fmt.Errorf("failed to get health status: %w", err)
	}

	var health map[string]interface{}
	if err := json.Unmarshal(resp, &health); err != nil {
		return fmt.Errorf("failed to parse health status: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

	_, _ = fmt.Fprintln(w, "Component\tStatus\tDetails")
	_, _ = fmt.Fprintln(w, "─────────\t──────\t───────")

	// Display system health
	if systemHealth, ok := health["system_health"].(map[string]interface{}); ok {
		_, _ = fmt.Fprintf(w, "Keychain\t%s\t\n", getHealthStatus(systemHealth["keychain_status"]))
		_, _ = fmt.Fprintf(w, "Encryption\t%s\t\n", getHealthStatus(systemHealth["encryption_status"]))
		_, _ = fmt.Fprintf(w, "File Integrity\t%s\t\n", getHealthStatus(systemHealth["file_integrity"]))
		_, _ = fmt.Fprintf(w, "Device Binding\t%s\t\n", getHealthStatus(systemHealth["device_binding"]))
		_, _ = fmt.Fprintf(w, "Audit Logging\t%s\t\n", getHealthStatus(systemHealth["audit_logging"]))
	}

	fmt.Println("\n✅ Health check completed")
	return nil
}

// SecurityDashboard displays the security dashboard
func (a *App) SecurityDashboard() error {
	fmt.Println("📊 Security Dashboard")
	fmt.Println("════════════════════")

	resp, err := a.apiClient.MakeRequest("GET", "/api/v1/security/dashboard", nil)
	if err != nil {
		return fmt.Errorf("failed to get security dashboard: %w", err)
	}

	var dashboard map[string]interface{}
	if err := json.Unmarshal(resp, &dashboard); err != nil {
		return fmt.Errorf("failed to parse dashboard: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

	// Overall status
	_, _ = fmt.Fprintf(w, "Status:\t%s\n", dashboard["status"])
	_, _ = fmt.Fprintf(w, "Threat Level:\t%s\n", dashboard["threat_level"])
	_, _ = fmt.Fprintf(w, "Security Score:\t%v/100\n", dashboard["security_score"])

	// Active alerts
	if alerts, ok := dashboard["active_alerts"].([]interface{}); ok {
		_, _ = fmt.Fprintf(w, "Active Alerts:\t%d\n", len(alerts))
	}

	// Metrics
	if metrics, ok := dashboard["metrics"].(map[string]interface{}); ok {
		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "Metrics:")
		_, _ = fmt.Fprintf(w, "  Total Events:\t%v\n", metrics["total_events"])
		_, _ = fmt.Fprintf(w, "  Failed Attempts:\t%v\n", metrics["failed_attempts"])
		_, _ = fmt.Fprintf(w, "  Successful Operations:\t%v\n", metrics["successful_operations"])
		_, _ = fmt.Fprintf(w, "  Tamper Attempts:\t%v\n", metrics["tamper_attempts"])
	}

	// Recommendations
	if recs, ok := dashboard["recommendations"].([]interface{}); ok && len(recs) > 0 {
		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "Recommendations:")
		for _, rec := range recs {
			_, _ = fmt.Fprintf(w, "  • %s\n", rec)
		}
	}

	return nil
}

// SecurityCorrelations displays security event correlations
func (a *App) SecurityCorrelations() error {
	fmt.Println("🔗 Security Event Correlations")
	fmt.Println("═════════════════════════════")

	resp, err := a.apiClient.MakeRequest("GET", "/api/v1/security/correlations", nil)
	if err != nil {
		return fmt.Errorf("failed to get correlations: %w", err)
	}

	var correlationData map[string]interface{}
	if err := json.Unmarshal(resp, &correlationData); err != nil {
		return fmt.Errorf("failed to parse correlations: %w", err)
	}

	correlations, ok := correlationData["correlations"].([]interface{})
	if !ok {
		fmt.Println("No correlations available")
		return nil
	}

	if len(correlations) == 0 {
		fmt.Println("No recent security correlations found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

	_, _ = fmt.Fprintln(w, "Pattern\tType\tRisk Score\tEvents\tTimestamp")
	_, _ = fmt.Fprintln(w, "───────\t────\t──────────\t──────\t─────────")

	for _, corr := range correlations {
		if correlation, ok := corr.(map[string]interface{}); ok {
			pattern := getStringValue(correlation, "pattern")
			corrType := getStringValue(correlation, "correlation_type")
			riskScore := getIntValue(correlation, "risk_score")
			eventCount := 0
			if events, ok := correlation["events"].([]interface{}); ok {
				eventCount = len(events)
			}

			timestamp := ""
			if ts, ok := correlation["timestamp"].(string); ok {
				if t, err := time.Parse(time.RFC3339, ts); err == nil {
					timestamp = t.Format("15:04:05")
				}
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n",
				pattern, corrType, riskScore, eventCount, timestamp)
		}
	}

	return nil
}

// SecurityKeychain displays keychain information and diagnostics
func (a *App) SecurityKeychain() error {
	fmt.Println("🔐 Keychain Information")
	fmt.Println("══════════════════════")

	resp, err := a.apiClient.MakeRequest("GET", "/api/v1/security/keychain", nil)
	if err != nil {
		return fmt.Errorf("failed to get keychain info: %w", err)
	}

	var keychainData map[string]interface{}
	if err := json.Unmarshal(resp, &keychainData); err != nil {
		return fmt.Errorf("failed to parse keychain info: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

	// Display keychain info
	if info, ok := keychainData["info"].(map[string]interface{}); ok {
		_, _ = fmt.Fprintln(w, "Provider Information:")
		_, _ = fmt.Fprintf(w, "  Provider:\t%s\n", info["provider"])
		_, _ = fmt.Fprintf(w, "  Platform:\t%s\n", info["platform"])
		_, _ = fmt.Fprintf(w, "  Native:\t%v\n", info["native"])
		_, _ = fmt.Fprintf(w, "  Available:\t%v\n", info["available"])
		_, _ = fmt.Fprintf(w, "  Security Level:\t%s\n", info["security_level"])

		if fallback, ok := info["fallback_reason"].(string); ok && fallback != "" {
			_, _ = fmt.Fprintf(w, "  Fallback Reason:\t%s\n", fallback)
		}
	}

	// Display diagnostics
	if diagnostics, ok := keychainData["diagnostics"].(map[string]interface{}); ok {
		if issues, ok := diagnostics["issues"].([]interface{}); ok && len(issues) > 0 {
			_, _ = fmt.Fprintln(w, "")
			_, _ = fmt.Fprintln(w, "Issues:")
			for _, issue := range issues {
				_, _ = fmt.Fprintf(w, "  ⚠️ %s\n", issue)
			}
		}

		if warnings, ok := diagnostics["warnings"].([]interface{}); ok && len(warnings) > 0 {
			_, _ = fmt.Fprintln(w, "")
			_, _ = fmt.Fprintln(w, "Warnings:")
			for _, warning := range warnings {
				_, _ = fmt.Fprintf(w, "  ⚠️ %s\n", warning)
			}
		}

		if recommendations, ok := diagnostics["recommendations"].([]interface{}); ok && len(recommendations) > 0 {
			_, _ = fmt.Fprintln(w, "")
			_, _ = fmt.Fprintln(w, "Recommendations:")
			for _, rec := range recommendations {
				_, _ = fmt.Fprintf(w, "  💡 %s\n", rec)
			}
		}
	}

	return nil
}

// SecurityConfig displays security configuration
func (a *App) SecurityConfig() error {
	fmt.Println("⚙️ Security Configuration")
	fmt.Println("════════════════════════")

	resp, err := a.apiClient.MakeRequest("GET", "/api/v1/security/config", nil)
	if err != nil {
		return fmt.Errorf("failed to get security config: %w", err)
	}

	var configData map[string]interface{}
	if err := json.Unmarshal(resp, &configData); err != nil {
		return fmt.Errorf("failed to parse security config: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

	_, _ = fmt.Fprintf(w, "Security Enabled:\t%v\n", configData["enabled"])
	_, _ = fmt.Fprintf(w, "Security Running:\t%v\n", configData["running"])

	if config, ok := configData["configuration"].(map[string]interface{}); ok {
		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "Component Configuration:")
		_, _ = fmt.Fprintf(w, "  Audit Logging:\t%v\n", config["audit_log_enabled"])
		_, _ = fmt.Fprintf(w, "  Monitoring:\t%v\n", config["monitoring_enabled"])
		_, _ = fmt.Fprintf(w, "  Correlation Analysis:\t%v\n", config["correlation_enabled"])
		_, _ = fmt.Fprintf(w, "  Registry Security:\t%v\n", config["registry_security_enabled"])
		_, _ = fmt.Fprintf(w, "  Health Checks:\t%v\n", config["health_check_enabled"])

		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "Timing Configuration:")
		_, _ = fmt.Fprintf(w, "  Monitor Interval:\t%s\n", config["monitor_interval"])
		_, _ = fmt.Fprintf(w, "  Analysis Interval:\t%s\n", config["analysis_interval"])
		_, _ = fmt.Fprintf(w, "  Health Check Interval:\t%s\n", config["health_check_interval"])

		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "Alert Configuration:")
		_, _ = fmt.Fprintf(w, "  Alert Threshold:\t%s\n", config["alert_threshold"])
		_, _ = fmt.Fprintf(w, "  Log Retention Days:\t%v\n", config["log_retention_days"])
	}

	return nil
}

// Helper functions

func getStatusValue(status map[string]interface{}, key string) string {
	if val, ok := status[key]; ok {
		return fmt.Sprintf("%v", val)
	}
	return "unknown"
}

func getHealthStatus(status interface{}) string {
	if s, ok := status.(string); ok {
		switch s {
		case "OK":
			return "✅ OK"
		case "WARNING":
			return "⚠️ WARNING"
		case "ERROR", "COMPROMISED":
			return "❌ ERROR"
		default:
			return s
		}
	}
	return "unknown"
}

func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getIntValue(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	if val, ok := m[key].(int); ok {
		return val
	}
	return 0
}

// AWS Compliance Methods

// ValidateAWSCompliance validates CloudWorkstation against AWS compliance framework using Strategy Pattern (SOLID: Single Responsibility)
func (a *App) ValidateAWSCompliance(framework string) error {
	// Create and execute compliance validation command
	validator := NewComplianceValidator(a.apiClient)
	return validator.Validate(framework)
}

// GenerateComplianceReport generates detailed compliance report
func (a *App) GenerateComplianceReport(framework string) error {
	fmt.Printf("📊 Generating AWS Compliance Report: %s\n", strings.ToUpper(framework))
	fmt.Println("════════════════════════════════════════════")

	resp, err := a.apiClient.MakeRequest("GET", fmt.Sprintf("/api/v1/security/compliance/report/%s", framework), nil)
	if err != nil {
		return fmt.Errorf("failed to generate compliance report: %w", err)
	}

	var report map[string]interface{}
	if err := json.Unmarshal(resp, &report); err != nil {
		return fmt.Errorf("failed to parse compliance report: %w", err)
	}

	// Display comprehensive report
	fmt.Printf("Framework: %s\n", report["framework"])
	fmt.Printf("AWS Compliance: %v\n", report["aws_compliant"])
	fmt.Printf("Overall Score: %v/100\n", report["compliance_score"])
	fmt.Printf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// Display detailed sections
	if sections, ok := report["sections"].([]interface{}); ok {
		for _, section := range sections {
			if sectionData, ok := section.(map[string]interface{}); ok {
				fmt.Printf("=== %s ===\n", sectionData["title"])
				if content, ok := sectionData["content"].(string); ok {
					fmt.Println(content)
				}
				fmt.Println()
			}
		}
	}

	fmt.Println("Report generation completed ✅")
	return nil
}

// ValidateSCPs validates Service Control Policies
func (a *App) ValidateSCPs(framework string) error {
	fmt.Printf("🛡️ Validating Service Control Policies: %s\n", strings.ToUpper(framework))
	fmt.Println("══════════════════════════════════════════════")

	resp, err := a.apiClient.MakeRequest("GET", fmt.Sprintf("/api/v1/security/compliance/scp/%s", framework), nil)
	if err != nil {
		return fmt.Errorf("failed to validate SCPs: %w", err)
	}

	var scpStatus map[string]interface{}
	if err := json.Unmarshal(resp, &scpStatus); err != nil {
		return fmt.Errorf("failed to parse SCP status: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

	// Display required SCPs
	if requiredSCPs, ok := scpStatus["required_scps"].([]interface{}); ok && len(requiredSCPs) > 0 {
		_, _ = fmt.Fprintln(w, "Required SCPs:")
		_, _ = fmt.Fprintln(w, "Policy\tStatus\tDescription")
		_, _ = fmt.Fprintln(w, "──────\t──────\t───────────")

		implementedSCPs := make(map[string]bool)
		if implemented, ok := scpStatus["implemented_scps"].([]interface{}); ok {
			for _, scp := range implemented {
				if scpName, ok := scp.(string); ok {
					implementedSCPs[scpName] = true
				}
			}
		}

		for _, scp := range requiredSCPs {
			if scpName, ok := scp.(string); ok {
				status := "❌ Missing"
				if implementedSCPs[scpName] {
					status = "✅ Implemented"
				}

				description := getSCPDescription(scpName)
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", scpName, status, description)
			}
		}
	}

	// Display implementation recommendations
	if gaps, ok := scpStatus["gaps"].([]interface{}); ok && len(gaps) > 0 {
		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "Implementation Recommendations:")
		for _, gap := range gaps {
			if gapData, ok := gap.(map[string]interface{}); ok {
				_, _ = fmt.Fprintf(w, "• %s\n", gapData["remediation"])
			}
		}
	}

	return nil
}

// ListComplianceFrameworks lists supported compliance frameworks
func (a *App) ListComplianceFrameworks() error {
	fmt.Println("📋 Supported AWS Compliance Frameworks")
	fmt.Println("═════════════════════════════════════")

	frameworks := []struct {
		Key         string
		Name        string
		Description string
		Scope       string
	}{
		{"nist-800-171", "NIST 800-171", "Protecting Controlled Unclassified Information", "Federal Contracts, CUI"},
		{"nist-800-53", "NIST 800-53", "Security Controls for Federal Information Systems", "Healthcare, Federal (HIPAA/FedRAMP)"},
		{"soc-2", "SOC 2 Type II", "Security, Availability, and Confidentiality", "Service Organizations"},
		{"hipaa", "HIPAA", "Health Insurance Portability and Accountability Act", "Healthcare (uses NIST 800-53)"},
		{"fedramp", "FedRAMP", "Federal Risk and Authorization Management Program", "Cloud Services (uses NIST 800-53)"},
		{"iso-27001", "ISO 27001", "Information Security Management Systems", "International Standard"},
		{"pci-dss", "PCI DSS", "Payment Card Industry Data Security Standard", "Payment Processing"},
		{"gdpr", "GDPR", "General Data Protection Regulation", "EU Privacy Protection"},
		{"itar", "ITAR", "International Traffic in Arms Regulations", "Defense Export Control (GovCloud Only)"},
		{"ear", "EAR", "Export Administration Regulations", "Dual-Use Technology Export Control"},
		{"cmmc", "CMMC", "Cybersecurity Maturity Model Certification", "Defense Industrial Base"},
		{"cmmc-l1", "CMMC Level 1", "Basic Cyber Hygiene", "Defense DIB Basic"},
		{"cmmc-l2", "CMMC Level 2", "Intermediate Cyber Hygiene", "Defense DIB + CUI (NIST 800-171)"},
		{"cmmc-l3", "CMMC Level 3", "Expert Cyber Hygiene", "Defense DIB Advanced (NIST 800-53)"},
		{"fisma", "FISMA", "Federal Information Security Modernization Act", "Federal Information Systems"},
		{"dfars", "DFARS", "Defense Federal Acquisition Regulation Supplement", "Defense Contractors"},
		{"ferpa", "FERPA", "Family Educational Rights and Privacy Act", "Student Privacy in Education"},
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

	_, _ = fmt.Fprintln(w, "Framework\tName\tScope")
	_, _ = fmt.Fprintln(w, "─────────\t────\t─────")

	for _, f := range frameworks {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", f.Key, f.Name, f.Scope)
	}

	fmt.Println("\nUsage:")
	fmt.Println("  cws security compliance validate <framework>")
	fmt.Println("  cws security compliance report <framework>")
	fmt.Println("  cws security compliance scp <framework>")

	return nil
}

// Helper functions for compliance display

func getComplianceStatusIcon(status string) string {
	switch strings.ToUpper(status) {
	case "CERTIFIED", "AUTHORIZED", "COMPLIANT":
		return "✅"
	case "ELIGIBLE", "REVIEW_REQUIRED":
		return "⚠️"
	case "NOT_CERTIFIED", "NON_COMPLIANT":
		return "❌"
	default:
		return "❓"
	}
}

func getSeverityIcon(severity string) string {
	switch strings.ToUpper(severity) {
	case "CRITICAL":
		return "🚨"
	case "HIGH":
		return "⚠️"
	case "MEDIUM":
		return "⚡"
	case "LOW":
		return "💡"
	default:
		return "❓"
	}
}

func getPriorityIcon(priority string) string {
	switch strings.ToUpper(priority) {
	case "CRITICAL":
		return "🚨"
	case "HIGH":
		return "⚠️"
	case "MEDIUM":
		return "⚡"
	case "LOW":
		return "💡"
	default:
		return "❓"
	}
}

func getSCPDescription(scpName string) string {
	descriptions := map[string]string{
		"DenyRootUserAccess":          "Prevents root user console access",
		"RequireMFAForConsoleAccess":  "Enforces MFA for AWS console login",
		"EnforceSSLOnlyRequests":      "Requires HTTPS/TLS for all requests",
		"RestrictRegionAccess":        "Limits access to approved AWS regions",
		"DenyUnencryptedStorage":      "Prevents unencrypted storage resources",
		"DenyPublicS3Buckets":         "Blocks public S3 bucket creation",
		"EnforceVPCEndpoints":         "Requires VPC endpoints for AWS services",
		"RequireMFAForAllAccess":      "Enforces MFA for all AWS access",
		"DenyNonGovCloudRegions":      "Restricts access to GovCloud regions only",
		"EnforceFIPS140-2":            "Requires FIPS 140-2 compliant encryption",
		"RequireCloudTrailEncryption": "Mandates encrypted CloudTrail logs",
		"DenyPublicAMISharing":        "Prevents public AMI sharing",
	}

	if desc, exists := descriptions[scpName]; exists {
		return desc
	}
	return "Security control policy"
}

// Compliance Validation Strategy Pattern Implementation (SOLID: Single Responsibility + Open/Closed)

// ComplianceValidator handles AWS compliance validation using Strategy Pattern (SOLID: Single Responsibility)
type ComplianceValidator struct {
	apiClient        interface{}
	headerService    *ComplianceHeaderService
	dataService      *ComplianceDataService
	overviewRenderer *ComplianceOverviewRenderer
	servicesRenderer *ComplianceServicesRenderer
	gapRenderer      *ComplianceGapRenderer
	actionsRenderer  *ComplianceActionsRenderer
}

// NewComplianceValidator creates a new compliance validator
func NewComplianceValidator(apiClient interface{}) *ComplianceValidator {
	return &ComplianceValidator{
		apiClient:        apiClient,
		headerService:    NewComplianceHeaderService(),
		dataService:      NewComplianceDataService(apiClient),
		overviewRenderer: NewComplianceOverviewRenderer(),
		servicesRenderer: NewComplianceServicesRenderer(),
		gapRenderer:      NewComplianceGapRenderer(),
		actionsRenderer:  NewComplianceActionsRenderer(),
	}
}

// Validate validates compliance for the given framework using Strategy Pattern
func (v *ComplianceValidator) Validate(framework string) error {
	// Display header
	v.headerService.DisplayHeader(framework)

	// Retrieve compliance data
	complianceStatus, err := v.dataService.GetComplianceData(framework)
	if err != nil {
		return err
	}

	// Create tabwriter for formatted output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

	// Render different sections using Strategy Pattern
	v.overviewRenderer.Render(w, complianceStatus)
	v.servicesRenderer.Render(w, complianceStatus)
	v.gapRenderer.Render(w, complianceStatus)
	v.actionsRenderer.Render(w, complianceStatus)

	return nil
}

// ComplianceHeaderService handles header display using Strategy Pattern (SOLID: Single Responsibility)
type ComplianceHeaderService struct{}

func NewComplianceHeaderService() *ComplianceHeaderService {
	return &ComplianceHeaderService{}
}

func (s *ComplianceHeaderService) DisplayHeader(framework string) {
	fmt.Printf("🔍 Validating AWS Compliance: %s\n", strings.ToUpper(framework))
	fmt.Println("═══════════════════════════════════════")
}

// ComplianceDataService handles data retrieval using Strategy Pattern (SOLID: Single Responsibility)
type ComplianceDataService struct {
	apiClient interface{}
}

func NewComplianceDataService(apiClient interface{}) *ComplianceDataService {
	return &ComplianceDataService{apiClient: apiClient}
}

func (s *ComplianceDataService) GetComplianceData(framework string) (map[string]interface{}, error) {
	// Make API request for AWS compliance validation
	if requester, ok := s.apiClient.(interface{ MakeRequest(string, string, interface{}) ([]byte, error) }); ok {
		resp, err := requester.MakeRequest("POST", fmt.Sprintf("/api/v1/security/compliance/validate/%s", framework), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to validate AWS compliance: %w", err)
		}

		var complianceStatus map[string]interface{}
		if err := json.Unmarshal(resp, &complianceStatus); err != nil {
			return nil, fmt.Errorf("failed to parse compliance status: %w", err)
		}

		return complianceStatus, nil
	}

	return nil, fmt.Errorf("API client does not support MakeRequest")
}

// ComplianceOverviewRenderer renders overview section using Strategy Pattern (SOLID: Single Responsibility)
type ComplianceOverviewRenderer struct{}

func NewComplianceOverviewRenderer() *ComplianceOverviewRenderer {
	return &ComplianceOverviewRenderer{}
}

func (r *ComplianceOverviewRenderer) Render(w *tabwriter.Writer, complianceStatus map[string]interface{}) {
	// Display compliance overview
	_, _ = fmt.Fprintf(w, "Framework:\t%s\n", complianceStatus["framework"])
	_, _ = fmt.Fprintf(w, "AWS Compliant:\t%v\n", complianceStatus["aws_compliant"])

	if reportID, ok := complianceStatus["artifact_report_id"].(string); ok && reportID != "" {
		_, _ = fmt.Fprintf(w, "AWS Artifact Report:\t%s\n", reportID)
	}

	if lastUpdated, ok := complianceStatus["last_updated"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastUpdated); err == nil {
			_, _ = fmt.Fprintf(w, "Last Updated:\t%s\n", t.Format("2006-01-02 15:04:05"))
		}
	}
}

// ComplianceServicesRenderer renders AWS services section using Strategy Pattern (SOLID: Single Responsibility)
type ComplianceServicesRenderer struct{}

func NewComplianceServicesRenderer() *ComplianceServicesRenderer {
	return &ComplianceServicesRenderer{}
}

func (r *ComplianceServicesRenderer) Render(w *tabwriter.Writer, complianceStatus map[string]interface{}) {
	services, ok := complianceStatus["aws_services"].([]interface{})
	if !ok || len(services) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "AWS Service Compliance:")
	_, _ = fmt.Fprintln(w, "Service\tStatus\tRegions")
	_, _ = fmt.Fprintln(w, "───────\t──────\t───────")

	for _, svc := range services {
		if service, ok := svc.(map[string]interface{}); ok {
			r.renderServiceRow(w, service)
		}
	}
}

func (r *ComplianceServicesRenderer) renderServiceRow(w *tabwriter.Writer, service map[string]interface{}) {
	serviceName := getStringValue(service, "service_name")
	status := getStringValue(service, "compliance_status")
	regions := r.formatRegions(service)
	statusIcon := getComplianceStatusIcon(status)
	_, _ = fmt.Fprintf(w, "%s\t%s %s\t%s\n", serviceName, statusIcon, status, regions)
}

func (r *ComplianceServicesRenderer) formatRegions(service map[string]interface{}) string {
	regionList, ok := service["certified_regions"].([]interface{})
	if !ok {
		return ""
	}

	regionStrings := make([]string, len(regionList))
	for i, r := range regionList {
		regionStrings[i] = fmt.Sprintf("%v", r)
	}

	if len(regionStrings) > 3 {
		return fmt.Sprintf("%s... (%d total)", strings.Join(regionStrings[:3], ", "), len(regionStrings))
	}
	return strings.Join(regionStrings, ", ")
}

// ComplianceGapRenderer renders gap analysis section using Strategy Pattern (SOLID: Single Responsibility)
type ComplianceGapRenderer struct{}

func NewComplianceGapRenderer() *ComplianceGapRenderer {
	return &ComplianceGapRenderer{}
}

func (r *ComplianceGapRenderer) Render(w *tabwriter.Writer, complianceStatus map[string]interface{}) {
	gaps, ok := complianceStatus["gap_analysis"].([]interface{})
	if !ok || len(gaps) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Gap Analysis:")
	_, _ = fmt.Fprintln(w, "Control\tSeverity\tRemediation")
	_, _ = fmt.Fprintln(w, "───────\t────────\t───────────")

	for _, gap := range gaps {
		if gapData, ok := gap.(map[string]interface{}); ok {
			r.renderGapRow(w, gapData)
		}
	}
}

func (r *ComplianceGapRenderer) renderGapRow(w *tabwriter.Writer, gapData map[string]interface{}) {
	control := getStringValue(gapData, "control")
	severity := getStringValue(gapData, "severity")
	remediation := getStringValue(gapData, "remediation")
	severityIcon := getSeverityIcon(severity)
	_, _ = fmt.Fprintf(w, "%s\t%s %s\t%s\n", control, severityIcon, severity, remediation)
}

// ComplianceActionsRenderer renders recommended actions section using Strategy Pattern (SOLID: Single Responsibility)
type ComplianceActionsRenderer struct{}

func NewComplianceActionsRenderer() *ComplianceActionsRenderer {
	return &ComplianceActionsRenderer{}
}

func (r *ComplianceActionsRenderer) Render(w *tabwriter.Writer, complianceStatus map[string]interface{}) {
	recs, ok := complianceStatus["recommended_actions"].([]interface{})
	if !ok || len(recs) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Recommended Actions:")

	for _, rec := range recs {
		if recommendation, ok := rec.(map[string]interface{}); ok {
			r.renderRecommendationRow(w, recommendation)
		}
	}
}

func (r *ComplianceActionsRenderer) renderRecommendationRow(w *tabwriter.Writer, recommendation map[string]interface{}) {
	priority := getStringValue(recommendation, "priority")
	action := getStringValue(recommendation, "action")
	awsService := getStringValue(recommendation, "aws_service")
	priorityIcon := getPriorityIcon(priority)

	if awsService != "" {
		_, _ = fmt.Fprintf(w, "%s %s [%s]:\t%s\n", priorityIcon, priority, awsService, action)
	} else {
		_, _ = fmt.Fprintf(w, "%s %s:\t%s\n", priorityIcon, priority, action)
	}
}
