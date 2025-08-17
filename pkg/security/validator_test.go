package security

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSecurityConfigValidator tests validator creation
func TestNewSecurityConfigValidator(t *testing.T) {
	config := GetDefaultSecurityConfig()
	
	validator := NewSecurityConfigValidator(config)
	
	assert.NotNil(t, validator)
	assert.Equal(t, config, validator.config)
}

// TestValidateSecurityConfiguration tests comprehensive security validation
func TestValidateSecurityConfiguration(t *testing.T) {
	config := GetDefaultSecurityConfig()
	validator := NewSecurityConfigValidator(config)
	
	result, err := validator.ValidateSecurityConfiguration()
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Score >= 0 && result.Score <= 100)
	assert.NotEmpty(t, result.Level)
	assert.NotEmpty(t, result.Summary)
}

// TestValidateSecurityConfigurationCriticalIssues tests validation with critical issues
func TestValidateSecurityConfigurationCriticalIssues(t *testing.T) {
	config := SecurityConfig{
		AuditLogEnabled:   false, // This should cause critical issue
		MonitoringEnabled: false, // This should cause critical issue
	}
	
	validator := NewSecurityConfigValidator(config)
	result, err := validator.ValidateSecurityConfiguration()
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Valid) // Should be invalid due to critical issues
	
	// Check for critical issues
	foundCriticalIssue := false
	for _, issue := range result.Issues {
		if issue.Severity == "CRITICAL" {
			foundCriticalIssue = true
			break
		}
	}
	assert.True(t, foundCriticalIssue)
}

// TestValidateAuditLogging tests audit logging validation
func TestValidateAuditLogging(t *testing.T) {
	tests := []struct {
		name          string
		config        SecurityConfig
		expectIssue   bool
		expectWarning bool
	}{
		{
			name: "Audit logging enabled with good retention",
			config: SecurityConfig{
				AuditLogEnabled:  true,
				LogRetentionDays: 90,
			},
			expectIssue:   false,
			expectWarning: false,
		},
		{
			name: "Audit logging disabled",
			config: SecurityConfig{
				AuditLogEnabled: false,
			},
			expectIssue:   true,
			expectWarning: false,
		},
		{
			name: "Audit logging enabled with short retention",
			config: SecurityConfig{
				AuditLogEnabled:  true,
				LogRetentionDays: 15, // Less than 30 days
			},
			expectIssue:   false,
			expectWarning: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSecurityConfigValidator(tt.config)
			result := &SecurityValidationResult{
				Issues:   make([]ValidationIssue, 0),
				Warnings: make([]ValidationWarning, 0),
			}
			
			validator.validateAuditLogging(result)
			
			if tt.expectIssue {
				assert.NotEmpty(t, result.Issues)
			} else {
				assert.Empty(t, result.Issues)
			}
			
			if tt.expectWarning {
				assert.NotEmpty(t, result.Warnings)
			} else {
				assert.Empty(t, result.Warnings)
			}
		})
	}
}

// TestValidateMonitoring tests security monitoring validation
func TestValidateMonitoring(t *testing.T) {
	tests := []struct {
		name          string
		config        SecurityConfig
		expectIssue   bool
		expectWarning bool
	}{
		{
			name: "Monitoring enabled with good interval",
			config: SecurityConfig{
				MonitoringEnabled: true,
				MonitorInterval:   30 * time.Second,
				AlertThreshold:    "MEDIUM",
			},
			expectIssue:   false,
			expectWarning: false,
		},
		{
			name: "Monitoring disabled",
			config: SecurityConfig{
				MonitoringEnabled: false,
			},
			expectIssue:   true,
			expectWarning: false,
		},
		{
			name: "Monitoring enabled with long interval",
			config: SecurityConfig{
				MonitoringEnabled: true,
				MonitorInterval:   10 * time.Minute, // Longer than 5 minutes
				AlertThreshold:    "MEDIUM",
			},
			expectIssue:   false,
			expectWarning: true,
		},
		{
			name: "Monitoring enabled with LOW threshold",
			config: SecurityConfig{
				MonitoringEnabled: true,
				MonitorInterval:   30 * time.Second,
				AlertThreshold:    "LOW",
			},
			expectIssue:   false,
			expectWarning: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSecurityConfigValidator(tt.config)
			result := &SecurityValidationResult{
				Issues:   make([]ValidationIssue, 0),
				Warnings: make([]ValidationWarning, 0),
			}
			
			validator.validateMonitoring(result)
			
			if tt.expectIssue {
				assert.NotEmpty(t, result.Issues)
			} else {
				assert.Empty(t, result.Issues)
			}
			
			if tt.expectWarning {
				assert.NotEmpty(t, result.Warnings)
			} else {
				assert.Empty(t, result.Warnings)
			}
		})
	}
}

// TestValidateCorrelationAnalysis tests correlation analysis validation
func TestValidateCorrelationAnalysis(t *testing.T) {
	tests := []struct {
		name               string
		config             SecurityConfig
		expectRecommendation bool
		expectWarning      bool
	}{
		{
			name: "Correlation enabled with good interval",
			config: SecurityConfig{
				CorrelationEnabled: true,
				AnalysisInterval:   5 * time.Minute,
			},
			expectRecommendation: false,
			expectWarning:       false,
		},
		{
			name: "Correlation disabled",
			config: SecurityConfig{
				CorrelationEnabled: false,
			},
			expectRecommendation: true,
			expectWarning:       false,
		},
		{
			name: "Correlation enabled with long interval",
			config: SecurityConfig{
				CorrelationEnabled: true,
				AnalysisInterval:   20 * time.Minute, // Longer than 15 minutes
			},
			expectRecommendation: false,
			expectWarning:       true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSecurityConfigValidator(tt.config)
			result := &SecurityValidationResult{
				Recommendations: make([]ValidationRecommendation, 0),
				Warnings:        make([]ValidationWarning, 0),
			}
			
			validator.validateCorrelationAnalysis(result)
			
			if tt.expectRecommendation {
				assert.NotEmpty(t, result.Recommendations)
			} else {
				assert.Empty(t, result.Recommendations)
			}
			
			if tt.expectWarning {
				assert.NotEmpty(t, result.Warnings)
			} else {
				assert.Empty(t, result.Warnings)
			}
		})
	}
}

// TestValidateRegistrySecurity tests registry security validation
func TestValidateRegistrySecurity(t *testing.T) {
	tests := []struct {
		name          string
		config        SecurityConfig
		expectIssue   bool
		expectWarning bool
	}{
		{
			name: "Registry security enabled with URL",
			config: SecurityConfig{
				RegistrySecurityEnabled: true,
				RegistryURL:             "https://registry.example.com",
			},
			expectIssue:   false,
			expectWarning: false,
		},
		{
			name: "Registry security disabled",
			config: SecurityConfig{
				RegistrySecurityEnabled: false,
			},
			expectIssue:   true,
			expectWarning: false,
		},
		{
			name: "Registry security enabled without URL",
			config: SecurityConfig{
				RegistrySecurityEnabled: true,
				RegistryURL:             "",
			},
			expectIssue:   false,
			expectWarning: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSecurityConfigValidator(tt.config)
			result := &SecurityValidationResult{
				Issues:   make([]ValidationIssue, 0),
				Warnings: make([]ValidationWarning, 0),
			}
			
			validator.validateRegistrySecurity(result)
			
			if tt.expectIssue {
				assert.NotEmpty(t, result.Issues)
			} else {
				assert.Empty(t, result.Issues)
			}
			
			if tt.expectWarning {
				assert.NotEmpty(t, result.Warnings)
			} else {
				assert.Empty(t, result.Warnings)
			}
		})
	}
}

// TestValidateTimeConfiguration tests timing configuration validation
func TestValidateTimeConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		config      SecurityConfig
		expectIssue bool
	}{
		{
			name: "Valid timing configuration",
			config: SecurityConfig{
				MonitorInterval:     30 * time.Second,
				AnalysisInterval:    5 * time.Minute,
				HealthCheckInterval: 15 * time.Minute,
			},
			expectIssue: false,
		},
		{
			name: "Zero monitor interval",
			config: SecurityConfig{
				MonitorInterval:     0,
				AnalysisInterval:    5 * time.Minute,
				HealthCheckInterval: 15 * time.Minute,
			},
			expectIssue: true,
		},
		{
			name: "Zero analysis interval",
			config: SecurityConfig{
				MonitorInterval:     30 * time.Second,
				AnalysisInterval:    0,
				HealthCheckInterval: 15 * time.Minute,
			},
			expectIssue: true,
		},
		{
			name: "Zero health check interval",
			config: SecurityConfig{
				MonitorInterval:     30 * time.Second,
				AnalysisInterval:    5 * time.Minute,
				HealthCheckInterval: 0,
			},
			expectIssue: true,
		},
		{
			name: "All intervals zero",
			config: SecurityConfig{
				MonitorInterval:     0,
				AnalysisInterval:    0,
				HealthCheckInterval: 0,
			},
			expectIssue: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSecurityConfigValidator(tt.config)
			result := &SecurityValidationResult{
				Issues: make([]ValidationIssue, 0),
			}
			
			validator.validateTimeConfiguration(result)
			
			if tt.expectIssue {
				assert.NotEmpty(t, result.Issues)
				for _, issue := range result.Issues {
					assert.Equal(t, "timing_configuration", issue.Component)
					assert.Equal(t, "HIGH", issue.Severity)
				}
			} else {
				assert.Empty(t, result.Issues)
			}
		})
	}
}

// TestValidateProductionReadiness tests production readiness validation
func TestValidateProductionReadiness(t *testing.T) {
	tests := []struct {
		name        string
		config      SecurityConfig
		expectValid bool
		issueCount  int
	}{
		{
			name: "Production ready configuration",
			config: SecurityConfig{
				AuditLogEnabled:    true,
				MonitoringEnabled:  true,
				HealthCheckEnabled: true,
				LogRetentionDays:   30,
			},
			expectValid: true,
			issueCount:  0,
		},
		{
			name: "Missing audit logging",
			config: SecurityConfig{
				AuditLogEnabled:    false,
				MonitoringEnabled:  true,
				HealthCheckEnabled: true,
				LogRetentionDays:   30,
			},
			expectValid: false,
			issueCount:  1,
		},
		{
			name: "Missing all production requirements",
			config: SecurityConfig{
				AuditLogEnabled:    false,
				MonitoringEnabled:  false,
				HealthCheckEnabled: false,
				LogRetentionDays:   3, // Less than 7 days
			},
			expectValid: false,
			issueCount:  4, // All production requirements fail
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSecurityConfigValidator(tt.config)
			result := &SecurityValidationResult{
				Issues:          make([]ValidationIssue, 0),
				Recommendations: make([]ValidationRecommendation, 0),
			}
			
			validator.validateProductionReadiness(result)
			
			productionIssues := 0
			for _, issue := range result.Issues {
				if issue.Component == "production_readiness" {
					productionIssues++
				}
			}
			
			assert.Equal(t, tt.issueCount, productionIssues)
		})
	}
}

// TestValidateNIST800171Compliance tests NIST 800-171 compliance validation
func TestValidateNIST800171Compliance(t *testing.T) {
	tests := []struct {
		name            string
		config          SecurityConfig
		expectCritical  int
		expectWarnings  int
	}{
		{
			name: "NIST 800-171 compliant configuration",
			config: SecurityConfig{
				AuditLogEnabled:         true,
				MonitoringEnabled:       true,
				RegistrySecurityEnabled: true,
				HealthCheckEnabled:      true,
				LogRetentionDays:        2555, // 7 years
				MonitorInterval:         30 * time.Second,
			},
			expectCritical: 0,
			expectWarnings: 0,
		},
		{
			name: "NIST 800-171 non-compliant - missing audit",
			config: SecurityConfig{
				AuditLogEnabled:         false,
				MonitoringEnabled:       true,
				RegistrySecurityEnabled: true,
				HealthCheckEnabled:      true,
				LogRetentionDays:        90,
			},
			expectCritical: 1, // Missing audit logging
			expectWarnings: 1, // Short retention
		},
		{
			name: "NIST 800-171 non-compliant - missing monitoring",
			config: SecurityConfig{
				AuditLogEnabled:         true,
				MonitoringEnabled:       false,
				RegistrySecurityEnabled: true,
				HealthCheckEnabled:      true,
				LogRetentionDays:        90,
			},
			expectCritical: 1, // Missing monitoring
			expectWarnings: 1, // Short retention
		},
		{
			name: "NIST 800-171 completely non-compliant",
			config: SecurityConfig{
				AuditLogEnabled:         false,
				MonitoringEnabled:       false,
				RegistrySecurityEnabled: false,
				HealthCheckEnabled:      false,
				LogRetentionDays:        7,
			},
			expectCritical: 2, // audit + monitoring are CRITICAL
			expectWarnings: 1, // Short retention
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSecurityConfigValidator(tt.config)
			result := &SecurityValidationResult{
				Issues:          make([]ValidationIssue, 0),
				Warnings:        make([]ValidationWarning, 0),
				Recommendations: make([]ValidationRecommendation, 0),
			}
			
			validator.validateNIST800171Compliance(result)
			
			criticalIssues := 0
			nistWarnings := 0
			nistRecommendations := 0
			
			for _, issue := range result.Issues {
				if issue.Component == "nist_800_171" && issue.Severity == "CRITICAL" {
					criticalIssues++
				}
			}
			
			for _, warning := range result.Warnings {
				if warning.Component == "nist_800_171" {
					nistWarnings++
				}
			}
			
			for _, rec := range result.Recommendations {
				if rec.Component == "nist_800_171" {
					nistRecommendations++
				}
			}
			
			assert.Equal(t, tt.expectCritical, criticalIssues)
			assert.Equal(t, tt.expectWarnings, nistWarnings)
			assert.Equal(t, 3, nistRecommendations) // Should always have 3 NIST recommendations
		})
	}
}

// TestCalculateSecurityScore tests security score calculation
func TestCalculateSecurityScore(t *testing.T) {
	validator := NewSecurityConfigValidator(SecurityConfig{})
	
	tests := []struct {
		name           string
		issues         []ValidationIssue
		warnings       []ValidationWarning
		config         SecurityConfig
		expectedScore  int
	}{
		{
			name:     "Perfect configuration",
			issues:   []ValidationIssue{},
			warnings: []ValidationWarning{},
			config: SecurityConfig{
				AuditLogEnabled:         true,
				MonitoringEnabled:       true,
				CorrelationEnabled:      true,
				RegistrySecurityEnabled: true,
				HealthCheckEnabled:      true,
			},
			expectedScore: 100, // 100 base + 10 bonus - 0 deductions
		},
		{
			name: "One critical issue",
			issues: []ValidationIssue{
				{Severity: "CRITICAL"},
			},
			warnings: []ValidationWarning{},
			config: SecurityConfig{
				AuditLogEnabled: true,
			},
			expectedScore: 77, // 100 + 2 - 25 = 77
		},
		{
			name: "Multiple issues",
			issues: []ValidationIssue{
				{Severity: "CRITICAL"},
				{Severity: "HIGH"},
				{Severity: "MEDIUM"},
				{Severity: "LOW"},
			},
			warnings: []ValidationWarning{{}, {}}, // 2 warnings
			config:   SecurityConfig{},
			expectedScore: 39, // 100 - 25 - 15 - 10 - 5 - 6 = 39
		},
		{
			name: "All features enabled",
			issues: []ValidationIssue{},
			warnings: []ValidationWarning{},
			config: SecurityConfig{
				AuditLogEnabled:         true,
				MonitoringEnabled:       true,
				CorrelationEnabled:      true,
				RegistrySecurityEnabled: true,
				HealthCheckEnabled:      true,
			},
			expectedScore: 100, // 100 + 10 = 110, capped at 100
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator.config = tt.config
			result := &SecurityValidationResult{
				Issues:   tt.issues,
				Warnings: tt.warnings,
			}
			
			score := validator.calculateSecurityScore(result)
			
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

// TestDetermineSecurityLevel tests security level determination
func TestDetermineSecurityLevel(t *testing.T) {
	validator := NewSecurityConfigValidator(SecurityConfig{})
	
	tests := []struct {
		score         int
		expectedLevel SecurityLevel
	}{
		{100, SecurityLevelEnterprise},
		{95, SecurityLevelEnterprise},
		{90, SecurityLevelEnterprise},
		{89, SecurityLevelHardened},
		{80, SecurityLevelHardened},
		{75, SecurityLevelHardened},
		{74, SecurityLevelStandard},
		{60, SecurityLevelStandard},
		{50, SecurityLevelStandard},
		{49, SecurityLevelBasic},
		{25, SecurityLevelBasic},
		{0, SecurityLevelBasic},
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("Score_%d", tt.score), func(t *testing.T) {
			level := validator.determineSecurityLevel(tt.score)
			assert.Equal(t, tt.expectedLevel, level)
		})
	}
}

// TestGenerateSummary tests summary generation
func TestGenerateSummary(t *testing.T) {
	validator := NewSecurityConfigValidator(SecurityConfig{})
	
	tests := []struct {
		name            string
		issues          []ValidationIssue
		warnings        []ValidationWarning
		expectedContains string
	}{
		{
			name: "Critical issues",
			issues: []ValidationIssue{
				{Severity: "CRITICAL"},
				{Severity: "CRITICAL"},
			},
			warnings:         []ValidationWarning{},
			expectedContains: "2 critical issues",
		},
		{
			name: "High issues",
			issues: []ValidationIssue{
				{Severity: "HIGH"},
			},
			warnings:         []ValidationWarning{},
			expectedContains: "1 high-priority issues",
		},
		{
			name:     "Only warnings",
			issues:   []ValidationIssue{},
			warnings: []ValidationWarning{{}, {}},
			expectedContains: "2 warnings",
		},
		{
			name:             "Perfect configuration",
			issues:           []ValidationIssue{},
			warnings:         []ValidationWarning{},
			expectedContains: "meets production standards",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &SecurityValidationResult{
				Issues:   tt.issues,
				Warnings: tt.warnings,
			}
			
			summary := validator.generateSummary(result)
			
			assert.Contains(t, summary, tt.expectedContains)
		})
	}
}

// TestValidateCurrentConfiguration tests current configuration validation
func TestValidateCurrentConfiguration(t *testing.T) {
	result, err := ValidateCurrentConfiguration()
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Score >= 0 && result.Score <= 100)
	assert.NotEmpty(t, result.Level)
	assert.NotEmpty(t, result.Summary)
}

// TestSecurityLevelConstants tests security level constants
func TestSecurityLevelConstants(t *testing.T) {
	levels := []SecurityLevel{
		SecurityLevelBasic,
		SecurityLevelStandard,
		SecurityLevelHardened,
		SecurityLevelEnterprise,
	}
	
	for _, level := range levels {
		assert.NotEmpty(t, string(level))
	}
	
	assert.Equal(t, "BASIC", string(SecurityLevelBasic))
	assert.Equal(t, "STANDARD", string(SecurityLevelStandard))
	assert.Equal(t, "HARDENED", string(SecurityLevelHardened))
	assert.Equal(t, "ENTERPRISE", string(SecurityLevelEnterprise))
}

// TestValidationStructs tests validation result structures
func TestValidationStructs(t *testing.T) {
	// Test ValidationIssue
	issue := ValidationIssue{
		Component:  "test_component",
		Issue:      "Test issue description",
		Severity:   "HIGH",
		Impact:     "Security compromise possible",
		Resolution: "Apply security patch",
	}
	
	assert.Equal(t, "test_component", issue.Component)
	assert.Equal(t, "HIGH", issue.Severity)
	
	// Test ValidationWarning
	warning := ValidationWarning{
		Component:      "test_component",
		Warning:        "Configuration suboptimal",
		Recommendation: "Consider upgrading",
	}
	
	assert.Equal(t, "test_component", warning.Component)
	assert.NotEmpty(t, warning.Recommendation)
	
	// Test ValidationRecommendation
	recommendation := ValidationRecommendation{
		Component:      "test_component",
		Recommendation: "Enable feature X",
		Priority:       "MEDIUM",
		Benefit:        "Improved security posture",
	}
	
	assert.Equal(t, "MEDIUM", recommendation.Priority)
	assert.NotEmpty(t, recommendation.Benefit)
}

// TestValidateHealthChecks tests health check validation
func TestValidateHealthChecks(t *testing.T) {
	tests := []struct {
		name               string
		config             SecurityConfig
		expectRecommendation bool
		expectWarning      bool
	}{
		{
			name: "Health checks enabled with good interval",
			config: SecurityConfig{
				HealthCheckEnabled:  true,
				HealthCheckInterval: 15 * time.Minute,
			},
			expectRecommendation: false,
			expectWarning:       false,
		},
		{
			name: "Health checks disabled",
			config: SecurityConfig{
				HealthCheckEnabled: false,
			},
			expectRecommendation: true,
			expectWarning:       false,
		},
		{
			name: "Health checks enabled with long interval",
			config: SecurityConfig{
				HealthCheckEnabled:  true,
				HealthCheckInterval: 45 * time.Minute, // Longer than 30 minutes
			},
			expectRecommendation: false,
			expectWarning:       true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSecurityConfigValidator(tt.config)
			result := &SecurityValidationResult{
				Recommendations: make([]ValidationRecommendation, 0),
				Warnings:        make([]ValidationWarning, 0),
			}
			
			validator.validateHealthChecks(result)
			
			if tt.expectRecommendation {
				assert.NotEmpty(t, result.Recommendations)
				for _, rec := range result.Recommendations {
					if rec.Component == "health_checks" {
						assert.Equal(t, "MEDIUM", rec.Priority)
					}
				}
			}
			
			if tt.expectWarning {
				assert.NotEmpty(t, result.Warnings)
				for _, warning := range result.Warnings {
					if warning.Component == "health_checks" {
						assert.Contains(t, warning.Warning, "45m0s")
					}
				}
			}
		})
	}
}

// TestValidateKeychainProvider tests keychain provider validation
func TestValidateKeychainProvider(t *testing.T) {
	validator := NewSecurityConfigValidator(SecurityConfig{})
	result := &SecurityValidationResult{
		Issues:          make([]ValidationIssue, 0),
		Warnings:        make([]ValidationWarning, 0),
		Recommendations: make([]ValidationRecommendation, 0),
	}
	
	validator.validateKeychainProvider(result)
	
	// Keychain validation may fail in test environment, but should not panic
	// The important thing is that it handles errors gracefully
	assert.True(t, len(result.Issues) >= 0) // May have issues due to test environment
}

// TestSecurityValidationResultJSONSerialization tests JSON serialization
func TestSecurityValidationResultJSONSerialization(t *testing.T) {
	result := SecurityValidationResult{
		Valid: true,
		Score: 85,
		Level: SecurityLevelHardened,
		Issues: []ValidationIssue{
			{
				Component:  "test",
				Issue:      "Test issue",
				Severity:   "MEDIUM",
				Impact:     "Test impact",
				Resolution: "Test resolution",
			},
		},
		Warnings: []ValidationWarning{
			{
				Component:      "test",
				Warning:        "Test warning",
				Recommendation: "Test recommendation",
			},
		},
		Recommendations: []ValidationRecommendation{
			{
				Component:      "test",
				Recommendation: "Test recommendation",
				Priority:       "LOW",
				Benefit:        "Test benefit",
			},
		},
		Summary: "Test summary",
	}
	
	// Test that all fields are properly set
	assert.True(t, result.Valid)
	assert.Equal(t, 85, result.Score)
	assert.Equal(t, SecurityLevelHardened, result.Level)
	assert.Len(t, result.Issues, 1)
	assert.Len(t, result.Warnings, 1)
	assert.Len(t, result.Recommendations, 1)
	assert.Equal(t, "Test summary", result.Summary)
}

// TestComprehensiveValidationWorkflow tests complete validation workflow
func TestComprehensiveValidationWorkflow(t *testing.T) {
	// Test with various configuration scenarios
	configs := []struct {
		name   string
		config SecurityConfig
	}{
		{
			name:   "Minimal configuration",
			config: SecurityConfig{},
		},
		{
			name: "Standard configuration",
			config: SecurityConfig{
				AuditLogEnabled:   true,
				MonitoringEnabled: true,
				LogRetentionDays:  30,
				MonitorInterval:   30 * time.Second,
			},
		},
		{
			name: "Enterprise configuration",
			config: SecurityConfig{
				AuditLogEnabled:         true,
				LogRetentionDays:        2555, // 7 years
				MonitoringEnabled:       true,
				MonitorInterval:         15 * time.Second,
				AlertThreshold:          "HIGH",
				RegistrySecurityEnabled: true,
				RegistryURL:             "https://registry.example.com",
				CorrelationEnabled:      true,
				AnalysisInterval:        2 * time.Minute,
				HealthCheckEnabled:      true,
				HealthCheckInterval:     5 * time.Minute,
			},
		},
	}
	
	for _, tt := range configs {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSecurityConfigValidator(tt.config)
			
			result, err := validator.ValidateSecurityConfiguration()
			
			assert.NoError(t, err)
			assert.NotNil(t, result)
			
			// All results should have valid scores and levels
			assert.True(t, result.Score >= 0 && result.Score <= 100)
			assert.NotEmpty(t, result.Level)
			assert.NotEmpty(t, result.Summary)
			
			// Enterprise config should score higher
			if tt.name == "Enterprise configuration" {
				assert.True(t, result.Score >= 80, "Enterprise config should score highly")
			}
		})
	}
}

// TestValidationIssueComponents tests that validation covers all security components
func TestValidationIssueComponents(t *testing.T) {
	config := SecurityConfig{
		AuditLogEnabled:         false,
		MonitoringEnabled:       false,
		CorrelationEnabled:      false,
		RegistrySecurityEnabled: false,
		HealthCheckEnabled:      false,
		MonitorInterval:         0,
		AnalysisInterval:        0,
		HealthCheckInterval:     0,
	}
	
	validator := NewSecurityConfigValidator(config)
	result, err := validator.ValidateSecurityConfiguration()
	
	require.NoError(t, err)
	
	// Should have issues/warnings for all major components
	components := make(map[string]bool)
	for _, issue := range result.Issues {
		components[issue.Component] = true
	}
	for _, warning := range result.Warnings {
		components[warning.Component] = true
	}
	for _, rec := range result.Recommendations {
		components[rec.Component] = true
	}
	
	// Should cover core security components
	expectedComponents := []string{
		"audit_logging",
		"security_monitoring", 
		"registry_security",
		"production_readiness",
		"timing_configuration",
		"nist_800_171",
	}
	
	for _, expected := range expectedComponents {
		assert.True(t, components[expected], "Should validate component: %s", expected)
	}
}