// Package security provides configuration validation for production deployments
package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile/security"
)

// SecurityValidationResult represents the result of security configuration validation
type SecurityValidationResult struct {
	Valid         bool                      `json:"valid"`
	Score         int                       `json:"score"` // 0-100 security score
	Level         SecurityLevel             `json:"level"`
	Issues        []ValidationIssue         `json:"issues"`
	Warnings      []ValidationWarning       `json:"warnings"`
	Recommendations []ValidationRecommendation `json:"recommendations"`
	Summary       string                    `json:"summary"`
}

// ValidationIssue represents a security configuration issue that must be fixed
type ValidationIssue struct {
	Component   string `json:"component"`
	Issue       string `json:"issue"`
	Severity    string `json:"severity"`
	Impact      string `json:"impact"`
	Resolution  string `json:"resolution"`
}

// ValidationWarning represents a security configuration warning
type ValidationWarning struct {
	Component   string `json:"component"`
	Warning     string `json:"warning"`
	Recommendation string `json:"recommendation"`
}

// ValidationRecommendation represents a security improvement recommendation
type ValidationRecommendation struct {
	Component   string `json:"component"`
	Recommendation string `json:"recommendation"`
	Priority    string `json:"priority"`
	Benefit     string `json:"benefit"`
}

// SecurityLevel represents the overall security level
type SecurityLevel string

const (
	SecurityLevelBasic      SecurityLevel = "BASIC"
	SecurityLevelStandard   SecurityLevel = "STANDARD"
	SecurityLevelHardened   SecurityLevel = "HARDENED"
	SecurityLevelEnterprise SecurityLevel = "ENTERPRISE"
)

// SecurityConfigValidator validates security configurations for production deployment
type SecurityConfigValidator struct {
	config SecurityConfig
}

// NewSecurityConfigValidator creates a new security configuration validator
func NewSecurityConfigValidator(config SecurityConfig) *SecurityConfigValidator {
	return &SecurityConfigValidator{
		config: config,
	}
}

// ValidateSecurityConfiguration performs comprehensive security configuration validation
func (v *SecurityConfigValidator) ValidateSecurityConfiguration() (*SecurityValidationResult, error) {
	result := &SecurityValidationResult{
		Valid:           true,
		Issues:          make([]ValidationIssue, 0),
		Warnings:        make([]ValidationWarning, 0),
		Recommendations: make([]ValidationRecommendation, 0),
	}

	// Validate core security components
	v.validateAuditLogging(result)
	v.validateMonitoring(result)
	v.validateCorrelationAnalysis(result)
	v.validateRegistrySecurity(result)
	v.validateKeychainProvider(result)
	v.validateHealthChecks(result)
	v.validateTimeConfiguration(result)
	v.validateProductionReadiness(result)

	// Calculate overall security score and level
	result.Score = v.calculateSecurityScore(result)
	result.Level = v.determineSecurityLevel(result.Score)
	result.Summary = v.generateSummary(result)

	// Mark as invalid if there are critical issues
	for _, issue := range result.Issues {
		if issue.Severity == "CRITICAL" {
			result.Valid = false
			break
		}
	}

	return result, nil
}

// validateAuditLogging validates audit logging configuration
func (v *SecurityConfigValidator) validateAuditLogging(result *SecurityValidationResult) {
	if !v.config.AuditLogEnabled {
		result.Issues = append(result.Issues, ValidationIssue{
			Component:  "audit_logging",
			Issue:      "Audit logging is disabled",
			Severity:   "CRITICAL",
			Impact:     "No audit trail for security events and compliance",
			Resolution: "Enable audit logging in security configuration",
		})
		return
	}

	// Check log retention policy
	if v.config.LogRetentionDays < 30 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component:      "audit_logging",
			Warning:        fmt.Sprintf("Log retention period is only %d days", v.config.LogRetentionDays),
			Recommendation: "Consider increasing retention to at least 90 days for compliance",
		})
	}

	// Check log directory permissions
	homeDir, err := os.UserHomeDir()
	if err == nil {
		auditDir := filepath.Join(homeDir, ".cloudworkstation", "security", "audit")
		if info, err := os.Stat(auditDir); err == nil {
			if info.Mode().Perm() != 0700 {
				result.Issues = append(result.Issues, ValidationIssue{
					Component:  "audit_logging",
					Issue:      "Audit log directory has incorrect permissions",
					Severity:   "HIGH",
					Impact:     "Potential unauthorized access to sensitive audit logs",
					Resolution: "Set audit directory permissions to 0700 (owner read/write only)",
				})
			}
		}
	}
}

// validateMonitoring validates security monitoring configuration
func (v *SecurityConfigValidator) validateMonitoring(result *SecurityValidationResult) {
	if !v.config.MonitoringEnabled {
		result.Issues = append(result.Issues, ValidationIssue{
			Component:  "security_monitoring",
			Issue:      "Security monitoring is disabled",
			Severity:   "CRITICAL",
			Impact:     "No real-time threat detection or security alerting",
			Resolution: "Enable security monitoring in configuration",
		})
		return
	}

	// Check monitoring interval
	if v.config.MonitorInterval > 5*time.Minute {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component:      "security_monitoring",
			Warning:        fmt.Sprintf("Monitor interval is %v, which may be too infrequent", v.config.MonitorInterval),
			Recommendation: "Consider reducing monitor interval to 30 seconds for better threat detection",
		})
	}

	// Check alert threshold
	if v.config.AlertThreshold == "LOW" {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component:      "security_monitoring",
			Warning:        "Alert threshold is set to LOW, which may generate too many false positives",
			Recommendation: "Consider using MEDIUM or HIGH threshold for production",
		})
	}
}

// validateCorrelationAnalysis validates event correlation configuration
func (v *SecurityConfigValidator) validateCorrelationAnalysis(result *SecurityValidationResult) {
	if !v.config.CorrelationEnabled {
		result.Recommendations = append(result.Recommendations, ValidationRecommendation{
			Component:      "correlation_analysis",
			Recommendation: "Enable security event correlation for advanced threat detection",
			Priority:       "MEDIUM",
			Benefit:        "Improved detection of complex attack patterns and behavioral anomalies",
		})
		return
	}

	// Check analysis interval
	if v.config.AnalysisInterval > 15*time.Minute {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component:      "correlation_analysis",
			Warning:        fmt.Sprintf("Analysis interval is %v, which may delay threat detection", v.config.AnalysisInterval),
			Recommendation: "Consider reducing analysis interval to 5 minutes or less",
		})
	}
}

// validateRegistrySecurity validates registry security configuration
func (v *SecurityConfigValidator) validateRegistrySecurity(result *SecurityValidationResult) {
	if !v.config.RegistrySecurityEnabled {
		result.Issues = append(result.Issues, ValidationIssue{
			Component:  "registry_security",
			Issue:      "Registry security is disabled",
			Severity:   "HIGH",
			Impact:     "Invitation registry communication is not cryptographically secured",
			Resolution: "Enable registry security for HMAC signing and certificate pinning",
		})
		return
	}

	// Check registry URL configuration
	if v.config.RegistryURL == "" {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component:      "registry_security",
			Warning:        "Registry URL is not configured",
			Recommendation: "Configure registry URL for secure remote registry communication",
		})
	}
}

// validateKeychainProvider validates keychain provider security
func (v *SecurityConfigValidator) validateKeychainProvider(result *SecurityValidationResult) {
	// Test keychain provider availability
	if err := security.ValidateKeychainProvider(); err != nil {
		result.Issues = append(result.Issues, ValidationIssue{
			Component:  "keychain_provider",
			Issue:      fmt.Sprintf("Keychain provider validation failed: %v", err),
			Severity:   "HIGH",
			Impact:     "Compromised secret storage and credential management",
			Resolution: "Fix keychain provider configuration or install required components",
		})
		return
	}

	// Check keychain provider type
	keychainInfo, err := security.GetKeychainInfo()
	if err == nil {
		if !keychainInfo.Native {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Component:      "keychain_provider",
				Warning:        "Using fallback file-based storage instead of native keychain",
				Recommendation: "Install and configure native keychain for better security",
			})
		}

		// Check security level
		if strings.Contains(strings.ToLower(keychainInfo.SecurityLevel), "fallback") {
			result.Recommendations = append(result.Recommendations, ValidationRecommendation{
				Component:      "keychain_provider",
				Recommendation: "Upgrade to hardware-backed keychain storage",
				Priority:       "HIGH",
				Benefit:        "Enhanced protection against credential theft and tampering",
			})
		}
	}
}

// validateHealthChecks validates health check configuration
func (v *SecurityConfigValidator) validateHealthChecks(result *SecurityValidationResult) {
	if !v.config.HealthCheckEnabled {
		result.Recommendations = append(result.Recommendations, ValidationRecommendation{
			Component:      "health_checks",
			Recommendation: "Enable automated security health checks",
			Priority:       "MEDIUM",
			Benefit:        "Proactive detection of security component failures",
		})
		return
	}

	// Check health check interval
	if v.config.HealthCheckInterval > 30*time.Minute {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component:      "health_checks",
			Warning:        fmt.Sprintf("Health check interval is %v, which may be too infrequent", v.config.HealthCheckInterval),
			Recommendation: "Consider reducing health check interval to 15 minutes or less",
		})
	}
}

// validateTimeConfiguration validates timing-related security configurations
func (v *SecurityConfigValidator) validateTimeConfiguration(result *SecurityValidationResult) {
	// Check for reasonable timing values
	timingIssues := make([]string, 0)

	if v.config.MonitorInterval <= 0 {
		timingIssues = append(timingIssues, "monitor interval is invalid")
	}
	if v.config.AnalysisInterval <= 0 {
		timingIssues = append(timingIssues, "analysis interval is invalid")
	}
	if v.config.HealthCheckInterval <= 0 {
		timingIssues = append(timingIssues, "health check interval is invalid")
	}

	if len(timingIssues) > 0 {
		result.Issues = append(result.Issues, ValidationIssue{
			Component:  "timing_configuration",
			Issue:      "Invalid timing configuration: " + strings.Join(timingIssues, ", "),
			Severity:   "HIGH",
			Impact:     "Security components may not function properly",
			Resolution: "Set valid positive durations for all timing configurations",
		})
	}
}

// validateProductionReadiness validates overall production readiness
func (v *SecurityConfigValidator) validateProductionReadiness(result *SecurityValidationResult) {
	// Check for minimum production requirements
	productionRequirements := []struct {
		condition bool
		component string
		issue     string
	}{
		{v.config.AuditLogEnabled, "production_readiness", "Audit logging must be enabled for production"},
		{v.config.MonitoringEnabled, "production_readiness", "Security monitoring must be enabled for production"},
		{v.config.HealthCheckEnabled, "production_readiness", "Health checks must be enabled for production"},
		{v.config.LogRetentionDays >= 7, "production_readiness", "Log retention must be at least 7 days for production"},
	}

	for _, req := range productionRequirements {
		if !req.condition {
			result.Issues = append(result.Issues, ValidationIssue{
				Component:  req.component,
				Issue:      req.issue,
				Severity:   "CRITICAL",
				Impact:     "System is not ready for production deployment",
				Resolution: "Enable all required production security features",
			})
		}
	}

	// NIST 800-171 compliance validation
	v.validateNIST800171Compliance(result)

	// Add production-specific recommendations
	if v.config.RegistrySecurityEnabled && v.config.CorrelationEnabled {
		result.Recommendations = append(result.Recommendations, ValidationRecommendation{
			Component:      "production_readiness",
			Recommendation: "Consider implementing additional monitoring and alerting integration",
			Priority:       "LOW",
			Benefit:        "Enhanced operational visibility and incident response",
		})
	}
}

// validateNIST800171Compliance validates NIST 800-171 compliance requirements
func (v *SecurityConfigValidator) validateNIST800171Compliance(result *SecurityValidationResult) {
	// NIST 800-171 requires comprehensive audit logging (AU.2.041, AU.2.042)
	if !v.config.AuditLogEnabled {
		result.Issues = append(result.Issues, ValidationIssue{
			Component:  "nist_800_171",
			Issue:      "NIST 800-171 AU.2.041: Audit record generation not enabled",
			Severity:   "CRITICAL",
			Impact:     "Non-compliance with federal security requirements for CUI",
			Resolution: "Enable comprehensive audit logging for NIST 800-171 compliance",
		})
	}

	// NIST 800-171 requires extended retention for federal contracts
	if v.config.LogRetentionDays < 2555 { // 7 years
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component:      "nist_800_171",
			Warning:        fmt.Sprintf("NIST 800-171 AU: Log retention is %d days, federal contracts require 7 years (2555 days)", v.config.LogRetentionDays),
			Recommendation: "Set log retention to 2555+ days for federal compliance requirements",
		})
	}

	// NIST 800-171 requires continuous monitoring (SI.2.214)
	if !v.config.MonitoringEnabled {
		result.Issues = append(result.Issues, ValidationIssue{
			Component:  "nist_800_171",
			Issue:      "NIST 800-171 SI.2.214: Security event monitoring not enabled",
			Severity:   "CRITICAL",
			Impact:     "Cannot detect security events as required for CUI protection",
			Resolution: "Enable continuous security monitoring for NIST 800-171 compliance",
		})
	}

	// NIST 800-171 requires encryption for CUI (SC.2.179)
	if !v.config.RegistrySecurityEnabled {
		result.Issues = append(result.Issues, ValidationIssue{
			Component:  "nist_800_171",
			Issue:      "NIST 800-171 SC.2.179: Encryption not fully enabled for CUI protection",
			Severity:   "HIGH",
			Impact:     "CUI data may not be adequately protected in transit",
			Resolution: "Enable registry security for complete encryption coverage",
		})
	}

	// NIST 800-171 requires security assessment and continuous monitoring
	if !v.config.HealthCheckEnabled {
		result.Issues = append(result.Issues, ValidationIssue{
			Component:  "nist_800_171",
			Issue:      "NIST 800-171 CA.2.157: Continuous monitoring not implemented",
			Severity:   "HIGH",
			Impact:     "Cannot maintain security posture assessment as required",
			Resolution: "Enable health checks for continuous security posture monitoring",
		})
	}

	// NIST 800-171 configuration management requirements (CM.2.061, CM.2.062)
	if v.config.MonitorInterval > 5*time.Minute {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component:      "nist_800_171",
			Warning:        "NIST 800-171 CM.2.062: Configuration monitoring interval may be too long for change detection",
			Recommendation: "Reduce monitoring interval to 5 minutes or less for timely change detection",
		})
	}

	// NIST 800-171 recommendations for enhanced security
	nistRecommendations := []ValidationRecommendation{
		{
			Component:      "nist_800_171",
			Recommendation: "Implement formal System Security Plan (SSP) documentation",
			Priority:       "HIGH",
			Benefit:        "Required documentation for NIST 800-171 compliance and assessment",
		},
		{
			Component:      "nist_800_171",
			Recommendation: "Develop Plan of Action & Milestones (POA&M) for any security gaps",
			Priority:       "HIGH",
			Benefit:        "Structured approach to address compliance gaps and maintain certification",
		},
		{
			Component:      "nist_800_171",
			Recommendation: "Establish incident response procedures per NIST 800-171 IR controls",
			Priority:       "MEDIUM",
			Benefit:        "Systematic response to security events as required for CUI protection",
		},
	}

	result.Recommendations = append(result.Recommendations, nistRecommendations...)
}

// calculateSecurityScore calculates overall security score (0-100)
func (v *SecurityConfigValidator) calculateSecurityScore(result *SecurityValidationResult) int {
	score := 100

	// Deduct points for issues
	for _, issue := range result.Issues {
		switch issue.Severity {
		case "CRITICAL":
			score -= 25
		case "HIGH":
			score -= 15
		case "MEDIUM":
			score -= 10
		case "LOW":
			score -= 5
		}
	}

	// Deduct points for warnings
	score -= len(result.Warnings) * 3

	// Bonus points for enabled features
	enabledFeatures := 0
	if v.config.AuditLogEnabled {
		enabledFeatures++
	}
	if v.config.MonitoringEnabled {
		enabledFeatures++
	}
	if v.config.CorrelationEnabled {
		enabledFeatures++
	}
	if v.config.RegistrySecurityEnabled {
		enabledFeatures++
	}
	if v.config.HealthCheckEnabled {
		enabledFeatures++
	}

	// Each enabled feature adds 2 points, max 10 points
	score += enabledFeatures * 2

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// determineSecurityLevel determines security level based on score
func (v *SecurityConfigValidator) determineSecurityLevel(score int) SecurityLevel {
	switch {
	case score >= 90:
		return SecurityLevelEnterprise
	case score >= 75:
		return SecurityLevelHardened
	case score >= 50:
		return SecurityLevelStandard
	default:
		return SecurityLevelBasic
	}
}

// generateSummary generates a human-readable summary
func (v *SecurityConfigValidator) generateSummary(result *SecurityValidationResult) string {
	criticalIssues := 0
	highIssues := 0
	
	for _, issue := range result.Issues {
		switch issue.Severity {
		case "CRITICAL":
			criticalIssues++
		case "HIGH":
			highIssues++
		}
	}

	if criticalIssues > 0 {
		return fmt.Sprintf("Security configuration has %d critical issues that must be resolved before production deployment", criticalIssues)
	}

	if highIssues > 0 {
		return fmt.Sprintf("Security configuration has %d high-priority issues that should be resolved", highIssues)
	}

	if len(result.Warnings) > 0 {
		return fmt.Sprintf("Security configuration is functional but has %d warnings to consider", len(result.Warnings))
	}

	return "Security configuration meets production standards"
}

// ValidateCurrentConfiguration validates the current security configuration
func ValidateCurrentConfiguration() (*SecurityValidationResult, error) {
	// Get default configuration for validation
	config := GetDefaultSecurityConfig()
	
	validator := NewSecurityConfigValidator(config)
	return validator.ValidateSecurityConfiguration()
}