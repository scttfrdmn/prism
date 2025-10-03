// Package templates provides AMI configuration validation for the Universal AMI System
package templates

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// AMIConfigValidator validates AMI configuration in templates
type AMIConfigValidator struct {
	// Known AWS regions for validation
	ValidRegions map[string]bool

	// Known AWS account IDs for trusted sources
	TrustedAccounts map[string]string // account_id -> description

	// AMI ID pattern validation
	AMIPattern *regexp.Regexp
}

// NewAMIConfigValidator creates a new AMI configuration validator
func NewAMIConfigValidator() *AMIConfigValidator {
	// Common AWS regions (this would be loaded from configuration in production)
	validRegions := map[string]bool{
		"us-east-1":      true,
		"us-east-2":      true,
		"us-west-1":      true,
		"us-west-2":      true,
		"ca-central-1":   true,
		"eu-west-1":      true,
		"eu-west-2":      true,
		"eu-west-3":      true,
		"eu-central-1":   true,
		"eu-north-1":     true,
		"ap-south-1":     true,
		"ap-southeast-1": true,
		"ap-southeast-2": true,
		"ap-northeast-1": true,
		"ap-northeast-2": true,
		"ap-northeast-3": true,
		"sa-east-1":      true,
	}

	// Trusted AWS accounts (CloudWorkstation, major vendors, etc.)
	trustedAccounts := map[string]string{
		"099720109477":     "Canonical (Ubuntu)",
		"137112412989":     "Amazon Web Services",
		"309956199498":     "Red Hat",
		"679593333241":     "MathWorks",
		"aws-marketplace":  "AWS Marketplace",
		"cloudworkstation": "CloudWorkstation Community",
	}

	// AMI ID regex pattern
	amiPattern := regexp.MustCompile(`^ami-[a-f0-9]{8,17}$`)

	return &AMIConfigValidator{
		ValidRegions:    validRegions,
		TrustedAccounts: trustedAccounts,
		AMIPattern:      amiPattern,
	}
}

// ValidateAMIConfig validates an AMI configuration and returns validation errors
func (v *AMIConfigValidator) ValidateAMIConfig(config *AMIConfig) []TemplateValidationError {
	var errors []TemplateValidationError

	// Validate AMI strategy
	if config.Strategy != "" {
		errors = append(errors, v.validateAMIStrategy(config.Strategy)...)
	}

	// Validate direct AMI mappings
	if config.AMIMappings != nil {
		errors = append(errors, v.validateAMIMappings(config.AMIMappings)...)
	}

	// Validate AMI search configuration
	if config.AMISearch != nil {
		errors = append(errors, v.validateAMISearch(config.AMISearch)...)
	}

	// Validate marketplace search configuration
	if config.MarketplaceSearch != nil {
		errors = append(errors, v.validateMarketplaceSearch(config.MarketplaceSearch)...)
	}

	// Validate fallback configuration
	errors = append(errors, v.validateFallbackConfig(config)...)

	// Validate optimization settings
	errors = append(errors, v.validateOptimizationConfig(config)...)

	// Validate legacy compatibility
	if config.AMIs != nil {
		errors = append(errors, v.validateLegacyAMIs(config.AMIs)...)
	}

	return errors
}

// validateAMIStrategy validates the AMI deployment strategy
func (v *AMIConfigValidator) validateAMIStrategy(strategy AMIStrategy) []TemplateValidationError {
	var errors []TemplateValidationError

	switch strategy {
	case AMIStrategyPreferred, AMIStrategyRequired, AMIStrategyFallback:
		// Valid strategies
	case "":
		// Empty is valid (defaults to preferred)
	default:
		errors = append(errors, TemplateValidationError{
			Field:   "ami_config.strategy",
			Message: fmt.Sprintf("invalid AMI strategy '%s', must be one of: ami_preferred, ami_required, ami_fallback", strategy),
		})
	}

	return errors
}

// validateAMIMappings validates direct AMI mappings
func (v *AMIConfigValidator) validateAMIMappings(mappings map[string]string) []TemplateValidationError {
	var errors []TemplateValidationError

	for region, amiID := range mappings {
		// Validate region
		if !v.ValidRegions[region] {
			errors = append(errors, TemplateValidationError{
				Field:   fmt.Sprintf("ami_config.ami_mappings.%s", region),
				Message: fmt.Sprintf("unknown AWS region '%s'", region),
			})
		}

		// Validate AMI ID format
		if !v.AMIPattern.MatchString(amiID) {
			errors = append(errors, TemplateValidationError{
				Field:   fmt.Sprintf("ami_config.ami_mappings.%s", region),
				Message: fmt.Sprintf("invalid AMI ID format '%s', must match pattern 'ami-xxxxxxxxx'", amiID),
			})
		}
	}

	return errors
}

// validateAMISearch validates dynamic AMI search configuration
func (v *AMIConfigValidator) validateAMISearch(search *AMISearchConfig) []TemplateValidationError {
	var errors []TemplateValidationError

	// Owner is required for AMI search
	if search.Owner == "" {
		errors = append(errors, TemplateValidationError{
			Field:   "ami_config.ami_search.owner",
			Message: "owner is required for AMI search",
		})
	}

	// Name pattern is required
	if search.NamePattern == "" {
		errors = append(errors, TemplateValidationError{
			Field:   "ami_config.ami_search.name_pattern",
			Message: "name_pattern is required for AMI search",
		})
	}

	// Validate architecture values
	for i, arch := range search.Architecture {
		if arch != "x86_64" && arch != "arm64" {
			errors = append(errors, TemplateValidationError{
				Field:   fmt.Sprintf("ami_config.ami_search.architecture[%d]", i),
				Message: fmt.Sprintf("invalid architecture '%s', must be 'x86_64' or 'arm64'", arch),
			})
		}
	}

	// Validate creation date format
	if search.MinCreationDate != "" {
		if _, err := time.Parse("2006-01-02", search.MinCreationDate); err != nil {
			errors = append(errors, TemplateValidationError{
				Field:   "ami_config.ami_search.min_creation_date",
				Message: fmt.Sprintf("invalid date format '%s', must be YYYY-MM-DD", search.MinCreationDate),
			})
		}
	}

	// Validate required tags
	for key, value := range search.RequiredTags {
		if key == "" {
			errors = append(errors, TemplateValidationError{
				Field:   "ami_config.ami_search.required_tags",
				Message: "tag key cannot be empty",
			})
		}
		if value == "" {
			errors = append(errors, TemplateValidationError{
				Field:   fmt.Sprintf("ami_config.ami_search.required_tags.%s", key),
				Message: "tag value cannot be empty",
			})
		}
	}

	return errors
}

// validateMarketplaceSearch validates marketplace search configuration
func (v *AMIConfigValidator) validateMarketplaceSearch(search *MarketplaceSearchConfig) []TemplateValidationError {
	var errors []TemplateValidationError

	// Product code is required for marketplace search
	if search.ProductCode == "" {
		errors = append(errors, TemplateValidationError{
			Field:   "ami_config.marketplace_search.product_code",
			Message: "product_code is required for marketplace search",
		})
	}

	// Validate version constraint format (basic validation)
	if search.VersionConstraint != "" {
		// Simple validation for semver constraints
		validConstraintPattern := regexp.MustCompile(`^(>=|>|<=|<|=)?\d+\.\d+\.\d+$`)
		if !validConstraintPattern.MatchString(search.VersionConstraint) {
			errors = append(errors, TemplateValidationError{
				Field:   "ami_config.marketplace_search.version_constraint",
				Message: fmt.Sprintf("invalid version constraint format '%s', must be semver with optional operator (e.g., '>=2.0.0')", search.VersionConstraint),
			})
		}
	}

	return errors
}

// validateFallbackConfig validates fallback configuration
func (v *AMIConfigValidator) validateFallbackConfig(config *AMIConfig) []TemplateValidationError {
	var errors []TemplateValidationError

	// Validate fallback strategy
	if config.FallbackStrategy != "" {
		switch config.FallbackStrategy {
		case "script_provisioning", "error", "cross_region":
			// Valid fallback strategies
		default:
			errors = append(errors, TemplateValidationError{
				Field:   "ami_config.fallback_strategy",
				Message: fmt.Sprintf("invalid fallback strategy '%s', must be one of: script_provisioning, error, cross_region", config.FallbackStrategy),
			})
		}
	}

	// Validate fallback timeout format
	if config.FallbackTimeout != "" {
		if _, err := time.ParseDuration(config.FallbackTimeout); err != nil {
			errors = append(errors, TemplateValidationError{
				Field:   "ami_config.fallback_timeout",
				Message: fmt.Sprintf("invalid timeout format '%s', must be a valid duration (e.g., '10m', '30s')", config.FallbackTimeout),
			})
		}
	}

	return errors
}

// validateOptimizationConfig validates optimization configuration
func (v *AMIConfigValidator) validateOptimizationConfig(config *AMIConfig) []TemplateValidationError {
	var errors []TemplateValidationError

	// Validate preferred architecture
	if config.PreferredArchitecture != "" {
		if config.PreferredArchitecture != "x86_64" && config.PreferredArchitecture != "arm64" {
			errors = append(errors, TemplateValidationError{
				Field:   "ami_config.preferred_architecture",
				Message: fmt.Sprintf("invalid architecture '%s', must be 'x86_64' or 'arm64'", config.PreferredArchitecture),
			})
		}
	}

	// Validate instance family preferences
	validFamilies := map[string]bool{
		"t3": true, "t3a": true, "t4g": true,
		"m5": true, "m5a": true, "m5n": true, "m6i": true, "m6a": true,
		"c5": true, "c5n": true, "c6i": true, "c6a": true,
		"r5": true, "r5a": true, "r6i": true,
		"p3": true, "p4": true, "g4dn": true, "g5": true,
	}

	for i, family := range config.InstanceFamilyPreference {
		if !validFamilies[family] {
			errors = append(errors, TemplateValidationError{
				Field:   fmt.Sprintf("ami_config.instance_family_preference[%d]", i),
				Message: fmt.Sprintf("unknown instance family '%s'", family),
			})
		}
	}

	return errors
}

// validateLegacyAMIs validates legacy AMI configuration for backwards compatibility
func (v *AMIConfigValidator) validateLegacyAMIs(amis map[string]map[string]string) []TemplateValidationError {
	var errors []TemplateValidationError

	for region, archMap := range amis {
		// Validate region
		if !v.ValidRegions[region] {
			errors = append(errors, TemplateValidationError{
				Field:   fmt.Sprintf("ami_config.amis.%s", region),
				Message: fmt.Sprintf("unknown AWS region '%s'", region),
			})
		}

		// Validate architecture mappings
		for arch, amiID := range archMap {
			if arch != "x86_64" && arch != "arm64" {
				errors = append(errors, TemplateValidationError{
					Field:   fmt.Sprintf("ami_config.amis.%s.%s", region, arch),
					Message: fmt.Sprintf("invalid architecture '%s', must be 'x86_64' or 'arm64'", arch),
				})
			}

			if !v.AMIPattern.MatchString(amiID) {
				errors = append(errors, TemplateValidationError{
					Field:   fmt.Sprintf("ami_config.amis.%s.%s", region, arch),
					Message: fmt.Sprintf("invalid AMI ID format '%s', must match pattern 'ami-xxxxxxxxx'", amiID),
				})
			}
		}
	}

	return errors
}

// ValidateTemplateAMIConfig validates AMI configuration in the context of a complete template
func (v *AMIConfigValidator) ValidateTemplateAMIConfig(template *Template) []TemplateValidationError {
	var errors []TemplateValidationError

	if !v.hasAMIConfig(template) {
		return errors
	}

	// Validate the base AMI configuration
	errors = append(errors, v.ValidateAMIConfig(&template.AMIConfig)...)

	// Template-specific validations
	errors = append(errors, v.validateTemplateAMIStrategy(template)...)
	errors = append(errors, v.validatePackageManagerCompatibility(template)...)
	errors = append(errors, v.validateSSHUserRequirement(template)...)

	return errors
}

func (v *AMIConfigValidator) hasAMIConfig(template *Template) bool {
	config := &template.AMIConfig
	return config.Strategy != "" ||
		config.AMIMappings != nil ||
		config.AMISearch != nil ||
		config.MarketplaceSearch != nil
}

func (v *AMIConfigValidator) validateTemplateAMIStrategy(template *Template) []TemplateValidationError {
	var errors []TemplateValidationError

	if template.AMIConfig.Strategy == AMIStrategyRequired {
		if !v.hasResolutionMethod(&template.AMIConfig) {
			errors = append(errors, TemplateValidationError{
				Field:   "ami_config",
				Message: "when strategy is 'ami_required', at least one AMI resolution method must be provided (ami_mappings, ami_search, marketplace_search, or legacy amis)",
			})
		}
	}

	return errors
}

func (v *AMIConfigValidator) hasResolutionMethod(config *AMIConfig) bool {
	return (config.AMIMappings != nil && len(config.AMIMappings) > 0) ||
		config.AMISearch != nil ||
		config.MarketplaceSearch != nil ||
		(config.AMIs != nil && len(config.AMIs) > 0)
}

func (v *AMIConfigValidator) validatePackageManagerCompatibility(template *Template) []TemplateValidationError {
	var errors []TemplateValidationError

	if v.hasScriptBasedPackageManager(template) && template.AMIConfig.Strategy == AMIStrategyRequired {
		errors = append(errors, TemplateValidationError{
			Field:   "ami_config.strategy",
			Message: "template cannot have both 'ami_required' strategy and script-based package manager, use 'ami_preferred' or 'ami_fallback'",
		})
	}

	return errors
}

func (v *AMIConfigValidator) hasScriptBasedPackageManager(template *Template) bool {
	return template.PackageManager != "" && template.PackageManager != "ami"
}

func (v *AMIConfigValidator) validateSSHUserRequirement(template *Template) []TemplateValidationError {
	var errors []TemplateValidationError

	if v.hasAMIMappings(&template.AMIConfig) && template.AMIConfig.SSHUser == "" {
		errors = append(errors, TemplateValidationError{
			Field:   "ami_config.ssh_user",
			Message: "ssh_user must be specified when using AMI mappings",
		})
	}

	return errors
}

func (v *AMIConfigValidator) hasAMIMappings(config *AMIConfig) bool {
	return config.AMIMappings != nil && len(config.AMIMappings) > 0
}

// GetAMIConfigSummary returns a human-readable summary of AMI configuration
func (v *AMIConfigValidator) GetAMIConfigSummary(config *AMIConfig) string {
	if config == nil {
		return "No AMI configuration"
	}

	var parts []string

	// Strategy
	strategy := config.Strategy
	if strategy == "" {
		strategy = AMIStrategyPreferred // Default
	}
	parts = append(parts, fmt.Sprintf("Strategy: %s", strategy))

	// Resolution methods
	var methods []string
	if len(config.AMIMappings) > 0 {
		methods = append(methods, fmt.Sprintf("%d direct mappings", len(config.AMIMappings)))
	}
	if config.AMISearch != nil {
		methods = append(methods, "dynamic search")
	}
	if config.MarketplaceSearch != nil {
		methods = append(methods, "marketplace search")
	}
	if len(config.AMIs) > 0 {
		totalLegacy := 0
		for _, archMap := range config.AMIs {
			totalLegacy += len(archMap)
		}
		methods = append(methods, fmt.Sprintf("%d legacy mappings", totalLegacy))
	}

	if len(methods) > 0 {
		parts = append(parts, fmt.Sprintf("Methods: %s", strings.Join(methods, ", ")))
	}

	// Optimization
	if config.PreferredArchitecture != "" {
		parts = append(parts, fmt.Sprintf("Preferred arch: %s", config.PreferredArchitecture))
	}

	return strings.Join(parts, "; ")
}
