// Package templates provides marketplace validation and security scanning for CloudWorkstation templates.
//
// The marketplace validator performs comprehensive security analysis, dependency checking,
// and quality validation to ensure templates meet community and institutional standards.
package templates

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// MarketplaceValidator provides comprehensive template validation for marketplace publication
type MarketplaceValidator struct {
	// Security scanners
	PackageScanner PackageSecurityScanner
	SecretsScanner SecretsScanner
	ConfigScanner  ConfigurationScanner

	// Quality checkers
	DependencyChecker DependencyChecker
	LicenseChecker    LicenseChecker

	// External integrations
	CVEDatabase     CVEDatabase
	LicenseRegistry LicenseRegistry

	// Validation configuration
	Config *ValidationConfig
}

// ValidationConfig defines validation rules and thresholds
type ValidationConfig struct {
	// Security thresholds
	MaxCriticalVulnerabilities int     `json:"max_critical_vulnerabilities"`
	MaxHighVulnerabilities     int     `json:"max_high_vulnerabilities"`
	MinSecurityScore           float64 `json:"min_security_score"`

	// Quality requirements
	RequireDescription   bool `json:"require_description"`
	RequireLicense       bool `json:"require_license"`
	RequireDocumentation bool `json:"require_documentation"`
	MinDescriptionLength int  `json:"min_description_length"`
	MaxLaunchTime        int  `json:"max_launch_time_minutes"`

	// Package validation
	AllowedPackageManagers []string `json:"allowed_package_managers"`
	BlockedPackages        []string `json:"blocked_packages"`
	RequirePackageVersions bool     `json:"require_package_versions"`

	// Content restrictions
	ForbiddenPatterns []string `json:"forbidden_patterns"`
	AllowedDomains    []string `json:"allowed_domains"`
	MaxTemplateSize   int64    `json:"max_template_size_bytes"`

	// Registry-specific rules
	RegistryType  RegistryType `json:"registry_type"`
	EnforcePolicy bool         `json:"enforce_policy"`
}

// MarketplaceValidationResult contains comprehensive validation results
type MarketplaceValidationResult struct {
	// Overall validation status
	Status      ValidationStatus `json:"status"`
	Score       float64          `json:"score"` // 0-100 overall quality score
	ValidatedAt time.Time        `json:"validated_at"`
	ValidatorID string           `json:"validator_id"`

	// Security analysis
	SecurityScan SecurityScanResult `json:"security_scan"`

	// Quality analysis
	QualityChecks []QualityCheck `json:"quality_checks"`

	// Dependency analysis
	Dependencies []DependencyAnalysis `json:"dependencies"`

	// Content analysis
	ContentAnalysis ContentAnalysis `json:"content_analysis"`

	// Recommendations
	Recommendations []Recommendation `json:"recommendations"`

	// Errors and warnings
	Errors   []MarketplaceValidationError   `json:"errors,omitempty"`
	Warnings []MarketplaceValidationWarning `json:"warnings,omitempty"`
}

// QualityCheck represents a specific quality validation check
type QualityCheck struct {
	Name       string         `json:"name"`
	Status     string         `json:"status"` // passed, failed, warning
	Score      float64        `json:"score"`  // 0-100
	Message    string         `json:"message"`
	Details    map[string]any `json:"details,omitempty"`
	Impact     string         `json:"impact"` // critical, high, medium, low
	ExecutedAt time.Time      `json:"executed_at"`
}

// DependencyAnalysis contains analysis of template dependencies
type DependencyAnalysis struct {
	Name             string            `json:"name"`
	Version          string            `json:"version,omitempty"`
	Type             string            `json:"type"` // package, template, system
	Source           string            `json:"source,omitempty"`
	SecurityFindings []SecurityFinding `json:"security_findings,omitempty"`
	LicenseInfo      LicenseInfo       `json:"license_info,omitempty"`
	Status           string            `json:"status"` // safe, vulnerable, unknown
}

// ContentAnalysis provides analysis of template content and metadata
type ContentAnalysis struct {
	// Template structure
	TemplateSize     int64   `json:"template_size_bytes"`
	ComplexityScore  float64 `json:"complexity_score"`
	MaintenanceScore float64 `json:"maintenance_score"`

	// Content quality
	DocumentationScore   float64 `json:"documentation_score"`
	DescriptionQuality   string  `json:"description_quality"` // excellent, good, adequate, poor
	MetadataCompleteness float64 `json:"metadata_completeness"`

	// Technical analysis
	PackageAnalysis       PackageAnalysis       `json:"package_analysis"`
	ConfigurationAnalysis ConfigurationAnalysis `json:"configuration_analysis"`

	// Security content analysis
	SecretsFound       []SecretFinding `json:"secrets_found,omitempty"`
	SuspiciousPatterns []PatternMatch  `json:"suspicious_patterns,omitempty"`
}

// PackageAnalysis analyzes packages and their security implications
type PackageAnalysis struct {
	TotalPackages        int                 `json:"total_packages"`
	KnownVulnerabilities []VulnerabilityInfo `json:"known_vulnerabilities,omitempty"`
	UnknownPackages      []string            `json:"unknown_packages,omitempty"`
	OutdatedPackages     []OutdatedPackage   `json:"outdated_packages,omitempty"`
	LicenseConflicts     []LicenseConflict   `json:"license_conflicts,omitempty"`
}

// ConfigurationAnalysis analyzes template configuration security
type ConfigurationAnalysis struct {
	SecurityMisconfigurations []SecurityMisconfiguration `json:"security_misconfigurations,omitempty"`
	BestPracticeViolations    []BestPracticeViolation    `json:"best_practice_violations,omitempty"`
	NetworkSecurityIssues     []NetworkSecurityIssue     `json:"network_security_issues,omitempty"`
	PrivilegeEscalations      []PrivilegeEscalation      `json:"privilege_escalations,omitempty"`
}

// Supporting types for detailed analysis
type VulnerabilityInfo struct {
	CVEID       string  `json:"cve_id"`
	Package     string  `json:"package"`
	Version     string  `json:"version"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	FixVersion  string  `json:"fix_version,omitempty"`
	Score       float64 `json:"cvss_score,omitempty"`
}

type OutdatedPackage struct {
	Package        string `json:"package"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	SecurityRisk   string `json:"security_risk"` // high, medium, low
}

type LicenseInfo struct {
	License      string   `json:"license"`
	Compatible   bool     `json:"compatible"`
	Restrictions []string `json:"restrictions,omitempty"`
}

type LicenseConflict struct {
	Package1 string `json:"package1"`
	License1 string `json:"license1"`
	Package2 string `json:"package2"`
	License2 string `json:"license2"`
	Conflict string `json:"conflict_reason"`
}

type SecretFinding struct {
	Type     string  `json:"type"`     // api_key, password, private_key, etc.
	Location string  `json:"location"` // field or section where found
	Severity string  `json:"severity"` // critical, high, medium
	Masked   string  `json:"masked"`   // partially masked secret for identification
	Entropy  float64 `json:"entropy"`  // randomness score
}

type PatternMatch struct {
	Pattern     string `json:"pattern"`
	Location    string `json:"location"`
	Match       string `json:"match"`
	Risk        string `json:"risk"` // high, medium, low
	Description string `json:"description"`
}

type SecurityMisconfiguration struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Remediation string `json:"remediation"`
}

type BestPracticeViolation struct {
	Practice   string `json:"practice"`
	Violation  string `json:"violation"`
	Impact     string `json:"impact"`
	Suggestion string `json:"suggestion"`
}

type NetworkSecurityIssue struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Risk        string `json:"risk"`
	Ports       []int  `json:"ports,omitempty"`
}

type PrivilegeEscalation struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Vector      string `json:"vector"`
}

type Recommendation struct {
	Type        string `json:"type"`     // security, quality, performance, maintenance
	Priority    string `json:"priority"` // critical, high, medium, low
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Effort      string `json:"effort"` // low, medium, high
}

type MarketplaceValidationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
	Value   any    `json:"value,omitempty"`
}

type MarketplaceValidationWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
	Impact  string `json:"impact"` // low, medium, high
}

// Interface definitions for pluggable scanners
type PackageSecurityScanner interface {
	ScanPackages(ctx context.Context, packages PackageDefinitions) (*PackageAnalysis, error)
}

type SecretsScanner interface {
	ScanForSecrets(ctx context.Context, content string) ([]SecretFinding, error)
}

type ConfigurationScanner interface {
	ScanConfiguration(ctx context.Context, template *Template) (*ConfigurationAnalysis, error)
}

type DependencyChecker interface {
	CheckDependencies(ctx context.Context, dependencies []TemplateDependency) ([]DependencyAnalysis, error)
}

type LicenseChecker interface {
	ValidateLicenses(ctx context.Context, template *Template) ([]LicenseConflict, error)
}

type CVEDatabase interface {
	QueryVulnerabilities(ctx context.Context, packages []string) ([]VulnerabilityInfo, error)
}

type LicenseRegistry interface {
	GetLicenseInfo(ctx context.Context, license string) (*LicenseInfo, error)
}

// NewMarketplaceValidator creates a new marketplace validator with default configuration
func NewMarketplaceValidator() *MarketplaceValidator {
	return &MarketplaceValidator{
		Config: &ValidationConfig{
			MaxCriticalVulnerabilities: 0,
			MaxHighVulnerabilities:     2,
			MinSecurityScore:           70.0,
			RequireDescription:         true,
			RequireLicense:             true,
			RequireDocumentation:       true,
			MinDescriptionLength:       50,
			MaxLaunchTime:              30,
			AllowedPackageManagers:     []string{"apt", "dnf", "conda", "spack"},
			RequirePackageVersions:     false,
			MaxTemplateSize:            1024 * 1024, // 1MB
			EnforcePolicy:              true,
		},
	}
}

// ValidateTemplate performs comprehensive validation of a template for marketplace publication
func (v *MarketplaceValidator) ValidateTemplate(ctx context.Context, template *Template) (*MarketplaceValidationResult, error) {
	result := &MarketplaceValidationResult{
		ValidatedAt: time.Now(),
		ValidatorID: fmt.Sprintf("marketplace-validator-v1.0-%x", sha256.Sum256([]byte(template.Name))),
		Status:      ValidationTesting,
	}

	// Validate template structure and metadata
	if err := v.validateTemplateStructure(template, result); err != nil {
		return result, fmt.Errorf("template structure validation failed: %w", err)
	}

	// Security scanning
	if err := v.performSecurityScan(ctx, template, result); err != nil {
		return result, fmt.Errorf("security scan failed: %w", err)
	}

	// Quality analysis
	if err := v.performQualityAnalysis(ctx, template, result); err != nil {
		return result, fmt.Errorf("quality analysis failed: %w", err)
	}

	// Dependency analysis
	if err := v.analyzeDependencies(ctx, template, result); err != nil {
		return result, fmt.Errorf("dependency analysis failed: %w", err)
	}

	// Content analysis
	if err := v.analyzeContent(ctx, template, result); err != nil {
		return result, fmt.Errorf("content analysis failed: %w", err)
	}

	// Generate recommendations
	v.generateRecommendations(template, result)

	// Calculate overall score and status
	v.calculateScore(result)
	v.determineStatus(result)

	return result, nil
}

// validateTemplateStructure performs basic template structure validation
func (v *MarketplaceValidator) validateTemplateStructure(template *Template, result *MarketplaceValidationResult) error {
	// Required fields validation
	if template.Name == "" {
		result.Errors = append(result.Errors, MarketplaceValidationError{
			Code:    "MISSING_NAME",
			Message: "Template name is required",
			Field:   "name",
		})
	}

	if template.Description == "" {
		result.Errors = append(result.Errors, MarketplaceValidationError{
			Code:    "MISSING_DESCRIPTION",
			Message: "Template description is required",
			Field:   "description",
		})
	} else if len(template.Description) < v.Config.MinDescriptionLength {
		result.Warnings = append(result.Warnings, MarketplaceValidationWarning{
			Code:    "SHORT_DESCRIPTION",
			Message: fmt.Sprintf("Description is shorter than recommended minimum of %d characters", v.Config.MinDescriptionLength),
			Field:   "description",
			Impact:  "medium",
		})
	}

	// Validate package manager
	if template.PackageManager != "" {
		found := false
		for _, allowed := range v.Config.AllowedPackageManagers {
			if template.PackageManager == allowed {
				found = true
				break
			}
		}
		if !found {
			result.Warnings = append(result.Warnings, MarketplaceValidationWarning{
				Code:    "UNSUPPORTED_PACKAGE_MANAGER",
				Message: fmt.Sprintf("Package manager %s is not in the allowed list", template.PackageManager),
				Field:   "package_manager",
				Impact:  "medium",
			})
		}
	}

	// Validate launch time estimate
	if template.EstimatedLaunchTime > v.Config.MaxLaunchTime {
		result.Warnings = append(result.Warnings, MarketplaceValidationWarning{
			Code:    "LONG_LAUNCH_TIME",
			Message: fmt.Sprintf("Estimated launch time of %d minutes exceeds recommended maximum of %d minutes", template.EstimatedLaunchTime, v.Config.MaxLaunchTime),
			Field:   "estimated_launch_time",
			Impact:  "low",
		})
	}

	return nil
}

// performSecurityScan conducts comprehensive security analysis
func (v *MarketplaceValidator) performSecurityScan(ctx context.Context, template *Template, result *MarketplaceValidationResult) error {
	securityScan := &SecurityScanResult{
		Status:   "pending",
		ScanDate: time.Now(),
		Scanner:  "marketplace-security-scanner-v1.0",
		Findings: []SecurityFinding{},
	}

	// Package security scanning
	if v.PackageScanner != nil {
		packageAnalysis, err := v.PackageScanner.ScanPackages(ctx, template.Packages)
		if err != nil {
			return fmt.Errorf("package scanning failed: %w", err)
		}

		// Convert package vulnerabilities to security findings
		for _, vuln := range packageAnalysis.KnownVulnerabilities {
			finding := SecurityFinding{
				Severity:    vuln.Severity,
				Category:    "vulnerability",
				Description: fmt.Sprintf("Package %s %s has known vulnerability %s", vuln.Package, vuln.Version, vuln.CVEID),
				CVEID:       vuln.CVEID,
			}
			if vuln.FixVersion != "" {
				finding.Remediation = fmt.Sprintf("Update to version %s or later", vuln.FixVersion)
			}
			securityScan.Findings = append(securityScan.Findings, finding)
		}
	}

	// Secrets scanning
	if v.SecretsScanner != nil {
		templateContent := fmt.Sprintf("%+v", template)
		secrets, err := v.SecretsScanner.ScanForSecrets(ctx, templateContent)
		if err != nil {
			return fmt.Errorf("secrets scanning failed: %w", err)
		}

		for _, secret := range secrets {
			finding := SecurityFinding{
				Severity:    secret.Severity,
				Category:    "secret",
				Description: fmt.Sprintf("Potential %s found in %s", secret.Type, secret.Location),
				Remediation: "Remove or properly secure sensitive information",
			}
			securityScan.Findings = append(securityScan.Findings, finding)
		}
	}

	// Configuration security scanning
	if v.ConfigScanner != nil {
		configAnalysis, err := v.ConfigScanner.ScanConfiguration(ctx, template)
		if err != nil {
			return fmt.Errorf("configuration scanning failed: %w", err)
		}

		for _, misconfig := range configAnalysis.SecurityMisconfigurations {
			finding := SecurityFinding{
				Severity:    misconfig.Severity,
				Category:    "misconfiguration",
				Description: misconfig.Description,
				Remediation: misconfig.Remediation,
			}
			securityScan.Findings = append(securityScan.Findings, finding)
		}
	}

	// Check for forbidden patterns
	templateStr := fmt.Sprintf("%+v", template)
	for _, pattern := range v.Config.ForbiddenPatterns {
		if matched, _ := regexp.MatchString(pattern, templateStr); matched {
			finding := SecurityFinding{
				Severity:    "high",
				Category:    "policy_violation",
				Description: fmt.Sprintf("Template contains forbidden pattern: %s", pattern),
				Remediation: "Remove the prohibited content",
			}
			securityScan.Findings = append(securityScan.Findings, finding)
		}
	}

	// Validate external URLs
	allURLs := append(template.LearningResources, template.Maintainer)
	if template.Marketplace != nil {
		allURLs = append(allURLs, template.Marketplace.SourceURL, template.Marketplace.DocumentationURL)
	}

	for _, urlStr := range allURLs {
		if urlStr == "" {
			continue
		}
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			continue
		}

		if len(v.Config.AllowedDomains) > 0 {
			allowed := false
			for _, domain := range v.Config.AllowedDomains {
				if strings.HasSuffix(parsedURL.Hostname(), domain) {
					allowed = true
					break
				}
			}
			if !allowed {
				finding := SecurityFinding{
					Severity:    "medium",
					Category:    "external_reference",
					Description: fmt.Sprintf("Reference to non-whitelisted domain: %s", parsedURL.Hostname()),
					Remediation: "Use only approved domains for external references",
				}
				securityScan.Findings = append(securityScan.Findings, finding)
			}
		}
	}

	// Calculate security score
	securityScan.Score = v.calculateSecurityScore(securityScan.Findings)

	// Determine overall security status
	criticalCount := 0
	highCount := 0
	for _, finding := range securityScan.Findings {
		switch finding.Severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		}
	}

	if criticalCount > v.Config.MaxCriticalVulnerabilities {
		securityScan.Status = "failed"
	} else if highCount > v.Config.MaxHighVulnerabilities {
		securityScan.Status = "warning"
	} else {
		securityScan.Status = "passed"
	}

	result.SecurityScan = *securityScan
	return nil
}

// performQualityAnalysis analyzes template quality metrics
func (v *MarketplaceValidator) performQualityAnalysis(ctx context.Context, template *Template, result *MarketplaceValidationResult) error {
	checks := []QualityCheck{}

	// Documentation completeness check
	docScore := v.calculateDocumentationScore(template)
	docCheck := QualityCheck{
		Name:       "Documentation Completeness",
		Score:      docScore,
		ExecutedAt: time.Now(),
	}

	if docScore >= 80 {
		docCheck.Status = "passed"
		docCheck.Message = "Comprehensive documentation provided"
		docCheck.Impact = "low"
	} else if docScore >= 60 {
		docCheck.Status = "warning"
		docCheck.Message = "Documentation could be more comprehensive"
		docCheck.Impact = "medium"
	} else {
		docCheck.Status = "failed"
		docCheck.Message = "Insufficient documentation"
		docCheck.Impact = "high"
	}

	checks = append(checks, docCheck)

	// Metadata completeness check
	metadataScore := v.calculateMetadataScore(template)
	metadataCheck := QualityCheck{
		Name:       "Metadata Completeness",
		Score:      metadataScore,
		ExecutedAt: time.Now(),
	}

	if metadataScore >= 80 {
		metadataCheck.Status = "passed"
		metadataCheck.Message = "Complete metadata provided"
	} else {
		metadataCheck.Status = "warning"
		metadataCheck.Message = "Some metadata fields are missing"
		metadataCheck.Impact = "medium"
	}

	checks = append(checks, metadataCheck)

	// Template complexity analysis
	complexityScore := v.calculateComplexityScore(template)
	complexityCheck := QualityCheck{
		Name:       "Template Complexity",
		Score:      complexityScore,
		ExecutedAt: time.Now(),
		Status:     "passed",
		Message:    fmt.Sprintf("Complexity level: %s", template.Complexity),
		Impact:     "low",
	}

	checks = append(checks, complexityCheck)

	result.QualityChecks = checks
	return nil
}

// analyzeDependencies performs dependency analysis
func (v *MarketplaceValidator) analyzeDependencies(ctx context.Context, template *Template, result *MarketplaceValidationResult) error {
	dependencies := []DependencyAnalysis{}

	if template.Marketplace != nil {
		for _, dep := range template.Marketplace.Dependencies {
			analysis := DependencyAnalysis{
				Name:    dep.Name,
				Version: dep.Version,
				Type:    dep.Type,
				Source:  dep.Source,
				Status:  "unknown",
			}

			// TODO: Implement actual dependency validation
			// This would involve checking if the dependency exists,
			// scanning it for vulnerabilities, validating licenses, etc.

			analysis.Status = "safe" // Placeholder
			dependencies = append(dependencies, analysis)
		}
	}

	result.Dependencies = dependencies
	return nil
}

// analyzeContent performs comprehensive content analysis
func (v *MarketplaceValidator) analyzeContent(ctx context.Context, template *Template, result *MarketplaceValidationResult) error {
	analysis := ContentAnalysis{
		DocumentationScore:   v.calculateDocumentationScore(template),
		MetadataCompleteness: v.calculateMetadataScore(template),
		ComplexityScore:      v.calculateComplexityScore(template),
	}

	// Determine description quality
	descLen := len(template.Description)
	if descLen >= 200 {
		analysis.DescriptionQuality = "excellent"
	} else if descLen >= 100 {
		analysis.DescriptionQuality = "good"
	} else if descLen >= 50 {
		analysis.DescriptionQuality = "adequate"
	} else {
		analysis.DescriptionQuality = "poor"
	}

	// Package analysis would be performed here
	analysis.PackageAnalysis = PackageAnalysis{
		TotalPackages: len(template.Packages.System) + len(template.Packages.Conda) + len(template.Packages.Spack) + len(template.Packages.Pip),
	}

	result.ContentAnalysis = analysis
	return nil
}

// generateRecommendations creates actionable recommendations
func (v *MarketplaceValidator) generateRecommendations(template *Template, result *MarketplaceValidationResult) {
	recommendations := []Recommendation{}

	// Security recommendations
	criticalFindings := 0
	for _, finding := range result.SecurityScan.Findings {
		if finding.Severity == "critical" {
			criticalFindings++
		}
	}

	if criticalFindings > 0 {
		recommendations = append(recommendations, Recommendation{
			Type:        "security",
			Priority:    "critical",
			Title:       "Address Critical Security Issues",
			Description: fmt.Sprintf("Template has %d critical security findings that must be resolved", criticalFindings),
			Action:      "Review and fix all critical security findings before publication",
			Effort:      "high",
		})
	}

	// Quality recommendations
	if result.ContentAnalysis.DescriptionQuality == "poor" {
		recommendations = append(recommendations, Recommendation{
			Type:        "quality",
			Priority:    "high",
			Title:       "Improve Template Description",
			Description: "Template description is too brief and lacks detail",
			Action:      "Expand description to at least 50 characters with clear use case information",
			Effort:      "low",
		})
	}

	// Documentation recommendations
	if result.ContentAnalysis.DocumentationScore < 60 {
		recommendations = append(recommendations, Recommendation{
			Type:        "quality",
			Priority:    "medium",
			Title:       "Add Learning Resources",
			Description: "Template would benefit from additional documentation links",
			Action:      "Add relevant tutorials, documentation, and learning resources",
			Effort:      "medium",
		})
	}

	// Sort recommendations by priority
	sort.Slice(recommendations, func(i, j int) bool {
		priority := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1}
		return priority[recommendations[i].Priority] > priority[recommendations[j].Priority]
	})

	result.Recommendations = recommendations
}

// calculateScore computes overall template score
func (v *MarketplaceValidator) calculateScore(result *MarketplaceValidationResult) {
	// Weight different aspects of the score
	securityWeight := 0.4
	qualityWeight := 0.3
	documentationWeight := 0.2
	metadataWeight := 0.1

	score := 0.0

	// Security score
	score += result.SecurityScan.Score * securityWeight

	// Average quality score
	qualityScore := 0.0
	if len(result.QualityChecks) > 0 {
		for _, check := range result.QualityChecks {
			qualityScore += check.Score
		}
		qualityScore /= float64(len(result.QualityChecks))
	}
	score += qualityScore * qualityWeight

	// Documentation and metadata scores
	score += result.ContentAnalysis.DocumentationScore * documentationWeight
	score += result.ContentAnalysis.MetadataCompleteness * metadataWeight

	result.Score = score
}

// determineStatus sets the overall validation status
func (v *MarketplaceValidator) determineStatus(result *MarketplaceValidationResult) {
	// Check for blocking errors
	if len(result.Errors) > 0 {
		result.Status = ValidationFailed
		return
	}

	// Check security status
	if result.SecurityScan.Status == "failed" {
		result.Status = ValidationFailed
		return
	}

	// Check quality thresholds
	if result.Score < v.Config.MinSecurityScore {
		result.Status = ValidationTesting
		return
	}

	// Check for high-impact warnings
	hasHighImpactWarnings := false
	for _, warning := range result.Warnings {
		if warning.Impact == "high" {
			hasHighImpactWarnings = true
			break
		}
	}

	if hasHighImpactWarnings || result.SecurityScan.Status == "warning" {
		result.Status = ValidationTesting
		return
	}

	result.Status = ValidationValidated
}

// Helper methods for score calculations
func (v *MarketplaceValidator) calculateSecurityScore(findings []SecurityFinding) float64 {
	if len(findings) == 0 {
		return 100.0
	}

	// Severity weights
	weights := map[string]float64{
		"critical": -25.0,
		"high":     -10.0,
		"medium":   -3.0,
		"low":      -1.0,
		"info":     -0.1,
	}

	score := 100.0
	for _, finding := range findings {
		if weight, exists := weights[finding.Severity]; exists {
			score += weight
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (v *MarketplaceValidator) calculateDocumentationScore(template *Template) float64 {
	score := 0.0
	maxScore := 100.0

	// Basic description (20 points)
	if template.Description != "" {
		score += 20.0
		// Bonus for detailed description
		if len(template.Description) >= 100 {
			score += 10.0
		}
	}

	// Long description (15 points)
	if template.LongDescription != "" {
		score += 15.0
	}

	// Prerequisites (15 points)
	if len(template.Prerequisites) > 0 {
		score += 15.0
	}

	// Learning resources (20 points)
	if len(template.LearningResources) > 0 {
		score += 20.0
	}

	// Maintainer info (10 points)
	if template.Maintainer != "" {
		score += 10.0
	}

	// Version info (10 points)
	if template.Version != "" {
		score += 10.0
	}

	// Documentation URL (10 points)
	if template.Marketplace != nil && template.Marketplace.DocumentationURL != "" {
		score += 10.0
	}

	return (score / maxScore) * 100.0
}

func (v *MarketplaceValidator) calculateMetadataScore(template *Template) float64 {
	score := 0.0
	fields := 0

	// Required fields
	if template.Name != "" {
		score += 10
		fields++
	}
	if template.Description != "" {
		score += 10
		fields++
	}
	if template.Category != "" {
		score += 10
		fields++
	}
	if template.Domain != "" {
		score += 10
		fields++
	}
	if template.Complexity != "" {
		score += 10
		fields++
	}
	if template.Maintainer != "" {
		score += 10
		fields++
	}
	if template.Version != "" {
		score += 10
		fields++
	}
	if template.Base != "" {
		score += 10
		fields++
	}
	if len(template.Tags) > 0 {
		score += 10
		fields++
	}
	if !template.LastUpdated.IsZero() {
		score += 10
		fields++
	}

	if fields == 0 {
		return 0
	}

	return score / float64(fields)
}

func (v *MarketplaceValidator) calculateComplexityScore(template *Template) float64 {
	score := 50.0 // Base score

	// Package complexity
	totalPackages := len(template.Packages.System) + len(template.Packages.Conda) + len(template.Packages.Spack) + len(template.Packages.Pip)
	if totalPackages > 50 {
		score += 20
	} else if totalPackages > 20 {
		score += 10
	}

	// Service complexity
	if len(template.Services) > 3 {
		score += 15
	} else if len(template.Services) > 0 {
		score += 5
	}

	// User configuration
	if len(template.Users) > 1 {
		score += 10
	}

	// Research user integration
	if template.ResearchUser != nil {
		score += 5
	}

	if score > 100 {
		score = 100
	}

	return score
}
