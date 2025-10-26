//go:build aws_integration
// +build aws_integration

// Package cli AWS integration test configuration
package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// AWSTestConfig holds configuration for AWS integration tests
type AWSTestConfig struct {
	Enabled       bool
	AWSProfile    string
	Region        string
	Timeout       time.Duration
	MaxInstances  int
	MaxVolumes    int
	MaxHourlyCost float64
	DaemonURL     string
	TestPrefix    string
}

// LoadAWSTestConfig loads configuration from environment variables
func LoadAWSTestConfig() *AWSTestConfig {
	config := &AWSTestConfig{
		// Default values
		Enabled:       false,
		AWSProfile:    "aws",
		Region:        "us-east-1",
		Timeout:       10 * time.Minute,
		MaxInstances:  5,
		MaxVolumes:    3,
		MaxHourlyCost: 5.0,
		DaemonURL:     "http://localhost:8947",
		TestPrefix:    "cwstest",
	}

	// Load from environment
	if os.Getenv("RUN_AWS_TESTS") == "true" {
		config.Enabled = true
	}

	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		config.AWSProfile = profile
	}

	if region := os.Getenv("AWS_TEST_REGION"); region != "" {
		config.Region = region
	}

	if timeoutStr := os.Getenv("AWS_TEST_TIMEOUT"); timeoutStr != "" {
		if minutes, err := strconv.Atoi(timeoutStr); err == nil {
			config.Timeout = time.Duration(minutes) * time.Minute
		}
	}

	if maxInstStr := os.Getenv("AWS_TEST_MAX_INSTANCES"); maxInstStr != "" {
		if maxInst, err := strconv.Atoi(maxInstStr); err == nil {
			config.MaxInstances = maxInst
		}
	}

	if maxVolStr := os.Getenv("AWS_TEST_MAX_VOLUMES"); maxVolStr != "" {
		if maxVol, err := strconv.Atoi(maxVolStr); err == nil {
			config.MaxVolumes = maxVol
		}
	}

	if maxCostStr := os.Getenv("AWS_TEST_MAX_HOURLY_COST"); maxCostStr != "" {
		if maxCost, err := strconv.ParseFloat(maxCostStr, 64); err == nil {
			config.MaxHourlyCost = maxCost
		}
	}

	if daemonURL := os.Getenv(DaemonURLEnvVar); daemonURL != "" {
		config.DaemonURL = daemonURL
	}

	if prefix := os.Getenv("AWS_TEST_PREFIX"); prefix != "" {
		config.TestPrefix = prefix
	}

	return config
}

// Validate checks if the AWS test configuration is valid
func (c *AWSTestConfig) Validate() error {
	if !c.Enabled {
		return fmt.Errorf("AWS tests not enabled - set RUN_AWS_TESTS=true")
	}

	if c.AWSProfile == "" {
		return fmt.Errorf("AWS profile not specified - set AWS_PROFILE")
	}

	if c.Region == "" {
		return fmt.Errorf("AWS region not specified")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("invalid timeout: %v", c.Timeout)
	}

	if c.MaxInstances <= 0 {
		return fmt.Errorf("invalid max instances: %d", c.MaxInstances)
	}

	if c.MaxVolumes <= 0 {
		return fmt.Errorf("invalid max volumes: %d", c.MaxVolumes)
	}

	if c.MaxHourlyCost <= 0 {
		return fmt.Errorf("invalid max hourly cost: %.2f", c.MaxHourlyCost)
	}

	return nil
}

// String returns a string representation of the configuration
func (c *AWSTestConfig) String() string {
	var sb strings.Builder
	sb.WriteString("AWS Integration Test Configuration:\n")
	sb.WriteString(fmt.Sprintf("  Enabled: %v\n", c.Enabled))
	sb.WriteString(fmt.Sprintf("  AWS Profile: %s\n", c.AWSProfile))
	sb.WriteString(fmt.Sprintf("  Region: %s\n", c.Region))
	sb.WriteString(fmt.Sprintf("  Timeout: %s\n", c.Timeout))
	sb.WriteString(fmt.Sprintf("  Max Instances: %d\n", c.MaxInstances))
	sb.WriteString(fmt.Sprintf("  Max Volumes: %d\n", c.MaxVolumes))
	sb.WriteString(fmt.Sprintf("  Max Hourly Cost: $%.2f\n", c.MaxHourlyCost))
	sb.WriteString(fmt.Sprintf("  Daemon URL: %s\n", c.DaemonURL))
	sb.WriteString(fmt.Sprintf("  Test Prefix: %s\n", c.TestPrefix))
	return sb.String()
}

// GetResourceName generates a unique test resource name
func (c *AWSTestConfig) GetResourceName(testID, resourceType string) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-%s-%s-%s", c.TestPrefix, testID, resourceType, timestamp)
}

// IsAWSProfileTest returns true if testing against the AWS profile
func (c *AWSTestConfig) IsAWSProfileTest() bool {
	return strings.ToLower(c.AWSProfile) == "aws"
}

// GetEstimatedCost returns estimated hourly cost for a resource type
func (c *AWSTestConfig) GetEstimatedCost(resourceType string) float64 {
	switch resourceType {
	case "t3.nano":
		return 0.0052 // $0.0052/hour
	case "t3.micro":
		return 0.0104 // $0.0104/hour
	case "t3.small":
		return 0.0208 // $0.0208/hour
	case "ebs-gp3-100gb":
		return 0.08 / 24 / 30 // ~$0.08/month
	case "efs":
		return 0.30 / 24 / 30 // ~$0.30/month per GB
	default:
		return 0.05 // Default estimate
	}
}

// GetSafeInstanceTypes returns list of cost-effective instance types for testing
func (c *AWSTestConfig) GetSafeInstanceTypes() []string {
	return []string{
		"t3.nano",  // $0.0052/hour - cheapest
		"t3.micro", // $0.0104/hour - free tier eligible
		"t3.small", // $0.0208/hour - small workloads
	}
}

// GetTestTags returns standard tags for test resources
func (c *AWSTestConfig) GetTestTags() map[string]string {
	return map[string]string{
		"CreatedBy":   "PrismIntegrationTest",
		"TestPrefix":  c.TestPrefix,
		"Environment": "test",
		"AutoCleanup": "true",
		"CostCenter":  "integration-testing",
		"MaxLifespan": "2h", // Resources should be cleaned up within 2 hours
	}
}
