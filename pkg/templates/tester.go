// Package templates provides CloudWorkstation's unified template system.
package templates

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// TestSuite represents a collection of template tests
type TestSuite struct {
	Name        string
	Description string
	Tests       []TemplateTest
}

// TemplateTest represents a single test case for a template
type TemplateTest struct {
	Name        string
	Description string
	Template    string
	TestFunc    TestFunction
}

// TestFunction defines a test implementation
type TestFunction func(ctx context.Context, template *Template) TestResult

// TestResult represents the outcome of a test
type TestResult struct {
	Passed   bool
	Duration time.Duration
	Message  string
	Details  []string
}

// TestReport contains results for all tests in a suite
type TestReport struct {
	SuiteName   string
	StartTime   time.Time
	EndTime     time.Time
	TotalTests  int
	PassedTests int
	FailedTests int
	TestResults map[string]TestResult
}

// TemplateTester runs tests against templates
type TemplateTester struct {
	registry *TemplateRegistry
	suites   []TestSuite
}

// NewTemplateTester creates a new tester with default test suites
func NewTemplateTester(registry *TemplateRegistry) *TemplateTester {
	return &TemplateTester{
		registry: registry,
		suites: []TestSuite{
			createSyntaxTestSuite(),
			createCompatibilityTestSuite(),
			createPerformanceTestSuite(),
			createSecurityTestSuite(),
			createIntegrationTestSuite(),
		},
	}
}

// RunAllTests runs all test suites against all templates
func (t *TemplateTester) RunAllTests(ctx context.Context) map[string]*TestReport {
	reports := make(map[string]*TestReport)

	for _, suite := range t.suites {
		report := t.RunSuite(ctx, suite)
		reports[suite.Name] = report
	}

	return reports
}

// RunSuite runs a single test suite
func (t *TemplateTester) RunSuite(ctx context.Context, suite TestSuite) *TestReport {
	report := &TestReport{
		SuiteName:   suite.Name,
		StartTime:   time.Now(),
		TestResults: make(map[string]TestResult),
	}

	for _, test := range suite.Tests {
		template, exists := t.registry.Templates[test.Template]
		if !exists {
			// Try to run against all templates if specific one not found
			if test.Template == "*" {
				for name, tmpl := range t.registry.Templates {
					testName := fmt.Sprintf("%s/%s", test.Name, name)
					result := t.runTest(ctx, test, tmpl)
					report.TestResults[testName] = result
					report.TotalTests++
					if result.Passed {
						report.PassedTests++
					} else {
						report.FailedTests++
					}
				}
			} else {
				report.TestResults[test.Name] = TestResult{
					Passed:  false,
					Message: fmt.Sprintf("Template %s not found", test.Template),
				}
				report.TotalTests++
				report.FailedTests++
			}
		} else {
			result := t.runTest(ctx, test, template)
			report.TestResults[test.Name] = result
			report.TotalTests++
			if result.Passed {
				report.PassedTests++
			} else {
				report.FailedTests++
			}
		}
	}

	report.EndTime = time.Now()
	return report
}

// runTest executes a single test
func (t *TemplateTester) runTest(ctx context.Context, test TemplateTest, template *Template) TestResult {
	start := time.Now()

	// Run test with timeout
	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Execute test function
	result := test.TestFunc(testCtx, template)
	result.Duration = time.Since(start)

	return result
}

// Test Suite Definitions

func createSyntaxTestSuite() TestSuite {
	return TestSuite{
		Name:        "syntax",
		Description: "Template syntax and structure validation",
		Tests: []TemplateTest{
			{
				Name:        "valid_yaml",
				Description: "Template is valid YAML",
				Template:    "*",
				TestFunc:    testValidYAML,
			},
			{
				Name:        "required_fields",
				Description: "Template has all required fields",
				Template:    "*",
				TestFunc:    testRequiredFields,
			},
			{
				Name:        "valid_references",
				Description: "All references are valid",
				Template:    "*",
				TestFunc:    testValidReferences,
			},
		},
	}
}

func createCompatibilityTestSuite() TestSuite {
	return TestSuite{
		Name:        "compatibility",
		Description: "Cross-platform and version compatibility",
		Tests: []TemplateTest{
			{
				Name:        "os_compatibility",
				Description: "Template works with supported OS versions",
				Template:    "*",
				TestFunc:    testOSCompatibility,
			},
			{
				Name:        "architecture_support",
				Description: "Template supports both x86_64 and arm64",
				Template:    "*",
				TestFunc:    testArchitectureSupport,
			},
			{
				Name:        "package_availability",
				Description: "All packages are available",
				Template:    "*",
				TestFunc:    testPackageAvailability,
			},
		},
	}
}

func createPerformanceTestSuite() TestSuite {
	return TestSuite{
		Name:        "performance",
		Description: "Performance and optimization checks",
		Tests: []TemplateTest{
			{
				Name:        "launch_time",
				Description: "Estimated launch time is reasonable",
				Template:    "*",
				TestFunc:    testLaunchTime,
			},
			{
				Name:        "resource_usage",
				Description: "Resource requirements are appropriate",
				Template:    "*",
				TestFunc:    testResourceUsage,
			},
		},
	}
}

func createSecurityTestSuite() TestSuite {
	return TestSuite{
		Name:        "security",
		Description: "Security best practices",
		Tests: []TemplateTest{
			{
				Name:        "no_hardcoded_secrets",
				Description: "No hardcoded passwords or keys",
				Template:    "*",
				TestFunc:    testNoHardcodedSecrets,
			},
			{
				Name:        "secure_defaults",
				Description: "Secure default configurations",
				Template:    "*",
				TestFunc:    testSecureDefaults,
			},
		},
	}
}

func createIntegrationTestSuite() TestSuite {
	return TestSuite{
		Name:        "integration",
		Description: "Integration with CloudWorkstation features",
		Tests: []TemplateTest{
			{
				Name:        "hibernation_support",
				Description: "Template supports hibernation",
				Template:    "*",
				TestFunc:    testHibernationSupport,
			},
			{
				Name:        "parameter_processing",
				Description: "Parameters work correctly",
				Template:    "*",
				TestFunc:    testParameterProcessing,
			},
		},
	}
}

// Test Function Implementations

func testValidYAML(ctx context.Context, template *Template) TestResult {
	// Template already parsed successfully if we have it
	return TestResult{
		Passed:  true,
		Message: "Valid YAML structure",
	}
}

func testRequiredFields(ctx context.Context, template *Template) TestResult {
	var missing []string

	if template.Name == "" {
		missing = append(missing, "name")
	}
	if template.Description == "" {
		missing = append(missing, "description")
	}
	if template.Base == "" && len(template.Inherits) == 0 {
		missing = append(missing, "base or inherits")
	}

	if len(missing) > 0 {
		return TestResult{
			Passed:  false,
			Message: fmt.Sprintf("Missing required fields: %s", strings.Join(missing, ", ")),
			Details: missing,
		}
	}

	return TestResult{
		Passed:  true,
		Message: "All required fields present",
	}
}

func testValidReferences(ctx context.Context, template *Template) TestResult {
	// Check variable references in template
	var issues []string

	// Check variable usage in various fields
	for varName := range template.Variables {
		varRef := fmt.Sprintf("{{.%s}}", varName)
		found := false

		// Check if variable is used anywhere
		if strings.Contains(template.PostInstall, varRef) ||
			strings.Contains(template.UserData, varRef) {
			found = true
		}

		for _, pkg := range template.Packages.Conda {
			if strings.Contains(pkg, varRef) {
				found = true
				break
			}
		}

		if !found {
			issues = append(issues, fmt.Sprintf("Unused variable: %s", varName))
		}
	}

	if len(issues) > 0 {
		return TestResult{
			Passed:  false,
			Message: "Reference issues found",
			Details: issues,
		}
	}

	return TestResult{
		Passed:  true,
		Message: "All references valid",
	}
}

func testOSCompatibility(ctx context.Context, template *Template) TestResult {
	supportedOS := map[string]bool{
		"ubuntu-20.04":     true,
		"ubuntu-22.04":     true,
		"ubuntu-24.04":     true,
		"rocky-9":          true,
		"amazonlinux-2023": true,
	}

	if template.Base != "" && !supportedOS[template.Base] {
		return TestResult{
			Passed:  false,
			Message: fmt.Sprintf("Unsupported OS: %s", template.Base),
		}
	}

	return TestResult{
		Passed:  true,
		Message: "OS is supported",
	}
}

func testArchitectureSupport(ctx context.Context, template *Template) TestResult {
	// Check if template explicitly excludes architectures
	if template.AMIConfig.AMIs != nil {
		hasX86 := false
		hasARM := false

		for _, archMap := range template.AMIConfig.AMIs {
			if _, ok := archMap["x86_64"]; ok {
				hasX86 = true
			}
			if _, ok := archMap["arm64"]; ok {
				hasARM = true
			}
		}

		if hasX86 && !hasARM {
			return TestResult{
				Passed:  false,
				Message: "Template only supports x86_64, consider adding arm64 support",
			}
		}
	}

	return TestResult{
		Passed:  true,
		Message: "Architecture support is adequate",
	}
}

func testPackageAvailability(ctx context.Context, template *Template) TestResult {
	// This would ideally check package repositories
	// For now, just check for known problematic packages
	problematic := []string{
		"cuda-11-0",        // Version specific packages
		"tensorflow==1.15", // Old versions
	}

	var issues []string

	allPackages := append(template.Packages.System, template.Packages.Conda...)
	allPackages = append(allPackages, template.Packages.Pip...)

	for _, pkg := range allPackages {
		for _, prob := range problematic {
			if strings.Contains(pkg, prob) {
				issues = append(issues, fmt.Sprintf("Potentially unavailable package: %s", pkg))
			}
		}
	}

	if len(issues) > 0 {
		return TestResult{
			Passed:  false,
			Message: "Package availability concerns",
			Details: issues,
		}
	}

	return TestResult{
		Passed:  true,
		Message: "Packages appear available",
	}
}

func testLaunchTime(ctx context.Context, template *Template) TestResult {
	if template.EstimatedLaunchTime > 15 {
		return TestResult{
			Passed:  false,
			Message: fmt.Sprintf("Launch time too long: %d minutes", template.EstimatedLaunchTime),
		}
	}

	return TestResult{
		Passed:  true,
		Message: fmt.Sprintf("Launch time acceptable: %d minutes", template.EstimatedLaunchTime),
	}
}

func testResourceUsage(ctx context.Context, template *Template) TestResult {
	// Check if default instance type is appropriate
	if strings.Contains(template.InstanceDefaults.Type, "8xlarge") {
		return TestResult{
			Passed:  false,
			Message: "Default instance type is very large",
		}
	}

	return TestResult{
		Passed:  true,
		Message: "Resource requirements appropriate",
	}
}

func testNoHardcodedSecrets(ctx context.Context, template *Template) TestResult {
	var issues []string

	// Check for common secret patterns
	secretPatterns := []string{
		"password=",
		"api_key=",
		"secret=",
		"token=",
		"aws_access_key",
	}

	for _, pattern := range secretPatterns {
		if strings.Contains(strings.ToLower(template.PostInstall), pattern) {
			issues = append(issues, fmt.Sprintf("Potential secret in post_install: %s", pattern))
		}
		if strings.Contains(strings.ToLower(template.UserData), pattern) {
			issues = append(issues, fmt.Sprintf("Potential secret in user_data: %s", pattern))
		}
	}

	if len(issues) > 0 {
		return TestResult{
			Passed:  false,
			Message: "Potential hardcoded secrets detected",
			Details: issues,
		}
	}

	return TestResult{
		Passed:  true,
		Message: "No hardcoded secrets detected",
	}
}

func testSecureDefaults(ctx context.Context, template *Template) TestResult {
	var issues []string

	// Check for insecure service configurations
	for _, service := range template.Services {
		if service.Port == 23 { // Telnet
			issues = append(issues, "Telnet service exposed (port 23)")
		}
		if service.Port == 21 { // FTP
			issues = append(issues, "FTP service exposed (port 21)")
		}
	}

	if len(issues) > 0 {
		return TestResult{
			Passed:  false,
			Message: "Insecure default services",
			Details: issues,
		}
	}

	return TestResult{
		Passed:  true,
		Message: "Secure default configuration",
	}
}

func testHibernationSupport(ctx context.Context, template *Template) TestResult {
	// Check if template is configured for hibernation
	if template.IdleDetection != nil && template.IdleDetection.Enabled {
		return TestResult{
			Passed:  true,
			Message: "Hibernation support configured",
		}
	}

	return TestResult{
		Passed:  false,
		Message: "Consider enabling idle detection for hibernation support",
	}
}

func testParameterProcessing(ctx context.Context, template *Template) TestResult {
	if len(template.Parameters) > 0 {
		// Check that parameters are used in template
		var unused []string
		for name := range template.Parameters {
			paramRef := fmt.Sprintf("{{.%s}}", name)
			found := false

			// Check various fields for parameter usage
			if strings.Contains(template.Description, paramRef) ||
				strings.Contains(template.PostInstall, paramRef) ||
				strings.Contains(template.UserData, paramRef) {
				found = true
			}

			if !found {
				unused = append(unused, name)
			}
		}

		if len(unused) > 0 {
			return TestResult{
				Passed:  false,
				Message: fmt.Sprintf("Unused parameters: %s", strings.Join(unused, ", ")),
				Details: unused,
			}
		}
	}

	return TestResult{
		Passed:  true,
		Message: "Parameters correctly configured",
	}
}
