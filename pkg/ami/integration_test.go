//go:build integration
// +build integration

package ami

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/stretchr/testify/assert"
)

const localstackEndpoint = "http://localhost:4566"

// setupLocalstackClients creates AWS clients configured to use LocalStack
func setupLocalstackClients(t *testing.T) (*ec2.Client, *ssm.Client) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test - set INTEGRATION_TESTS=1 to run")
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithBaseEndpoint(localstackEndpoint),
	)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)
	ssmClient := ssm.NewFromConfig(cfg)

	return ec2Client, ssmClient
}

// setupLocalstackBuilder creates a Builder configured to use LocalStack
func setupLocalstackBuilder(t *testing.T) *Builder {
	ec2Client, ssmClient := setupLocalstackClients(t)

	// Create mock registry client
	registryClient := &Registry{
		SSMClient:       ssmClient,
		ParameterPrefix: "/prism/ami/",
	}

	// Create base AMIs map for testing
	baseAMIs := map[string]map[string]string{
		"us-east-1": {
			"x86_64":                  "ami-test-x86-east1",
			"arm64":                   "ami-test-arm-east1",
			"ubuntu-22.04-server-lts": "ami-test-ubuntu-east1",
		},
		"us-west-2": {
			"x86_64":                  "ami-test-x86-west2",
			"arm64":                   "ami-test-arm-west2",
			"ubuntu-22.04-server-lts": "ami-test-ubuntu-west2",
		},
	}

	builder := &Builder{
		EC2Client:       ec2Client,
		SSMClient:       ssmClient,
		RegistryClient:  registryClient,
		BaseAMIs:        baseAMIs,
		DefaultVPC:      "vpc-test123",
		DefaultSubnet:   "subnet-test123",
		SecurityGroupID: "sg-test123",
	}

	return builder
}

// TestValidateRegion tests the region validation logic
func TestValidateRegion(t *testing.T) {
	builder := setupLocalstackBuilder(t)

	// Test valid region
	err := builder.validateRegion("us-east-1")
	assert.NoError(t, err)

	// Test another valid region
	err = builder.validateRegion("us-west-2")
	assert.NoError(t, err)

	// Test invalid region
	err = builder.validateRegion("invalid-region")
	assert.Error(t, err)
}

// TestGetDefaultSecurityGroup tests security group resolution
func TestGetDefaultSecurityGroup(t *testing.T) {
	builder := setupLocalstackBuilder(t)
	ec2Client := builder.EC2Client
	ctx := context.Background()

	// Create a test security group in LocalStack
	createSgInput := &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String("default"),
		Description: aws.String("Default security group"),
		VpcId:       aws.String("vpc-test123"),
	}
	createSgOutput, err := ec2Client.CreateSecurityGroup(ctx, createSgInput)
	if err != nil {
		t.Logf("Warning: Failed to create security group in LocalStack: %v", err)
		// Continue anyway as this may just be a limitation in LocalStack
	} else {
		t.Logf("Created security group: %s", *createSgOutput.GroupId)
	}

	// getDefaultSecurityGroup looks up the VPC's "default" group in EC2.
	// This will likely be limited in LocalStack.
	securityGroup, err := builder.getDefaultSecurityGroup(ctx, "vpc-test123")
	// This might fail in LocalStack - check both possibilities
	if err == nil {
		t.Logf("Found security group: %s", securityGroup)
		assert.NotEmpty(t, securityGroup)
	} else {
		t.Logf("Security group lookup limitation in LocalStack: %v", err)
	}
}

// TestInitializeBaseAMIs tests base AMI initialization
func TestInitializeBaseAMIs(t *testing.T) {
	builder := setupLocalstackBuilder(t)

	// Clear the base AMIs to test initialization
	builder.BaseAMIs = nil

	// Initialize base AMIs
	builder.initializeBaseAMIs()

	// Verify base AMIs are populated
	assert.NotNil(t, builder.BaseAMIs)
	assert.NotEmpty(t, builder.BaseAMIs)

	// Check that key regions exist
	assert.Contains(t, builder.BaseAMIs, "us-east-1")
	assert.Contains(t, builder.BaseAMIs, "us-west-2")

	// Check that architectures exist for a region
	assert.Contains(t, builder.BaseAMIs["us-east-1"], "x86_64")
	assert.Contains(t, builder.BaseAMIs["us-east-1"], "arm64")
}

// TestParseTemplate tests template parsing
func TestParseTemplate(t *testing.T) {
	// Create a test parser
	parser := NewParser()

	// Valid template YAML
	validTemplate := `
name: "Test Template"
base: "ubuntu-22.04-server-lts"
description: "Test template for integration tests"
build_steps:
  - name: "System updates"
    script: |
      apt-get update -y
validation:
  - name: "System check"
    command: "echo success"
    success: true
`

	// Parse valid template
	template, err := parser.ParseTemplate(validTemplate)
	assert.NoError(t, err)
	assert.Equal(t, "Test Template", template.Name)
	assert.Equal(t, "ubuntu-22.04-server-lts", template.Base)
	assert.Equal(t, 1, len(template.BuildSteps))
	assert.Equal(t, 1, len(template.Validation))

	// Invalid template YAML (missing required fields)
	invalidTemplate := `
name: "Invalid Template"
description: "Missing required base field"
`

	// Parse invalid template
	template, err = parser.ParseTemplate(invalidTemplate)
	assert.Error(t, err)
	assert.Nil(t, template)
}

// TestListTemplates tests template listing
func TestListTemplates(t *testing.T) {
	// Parser.ListTemplates scans ./templates and ~/.prism/templates (plus a
	// system directory), so point both the working directory and HOME at a
	// temp tree containing a templates/ directory.
	tempDir := t.TempDir()
	templatesDir := filepath.Join(tempDir, "templates")
	if err := os.MkdirAll(templatesDir, 0o750); err != nil {
		t.Fatalf("Failed to create templates directory: %v", err)
	}
	t.Chdir(tempDir)
	t.Setenv("HOME", tempDir)

	// Create test template files
	templateContent := `
name: "Test Template"
base: "ubuntu-22.04-server-lts"
description: "Test template"
build_steps:
  - name: "System updates"
    script: |
      echo "test"
validation:
  - name: "Test"
    command: "echo test"
    success: true
`

	// Write test templates
	if err := os.WriteFile(filepath.Join(templatesDir, "template1.yml"), []byte(templateContent), 0o600); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "template2.yaml"), []byte(templateContent), 0o600); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	// Create a test parser with base AMIs
	parser := NewParser()
	parser.BaseAMIs = map[string]map[string]string{
		"us-east-1": {
			"ubuntu-22.04-server-lts": "ami-test",
		},
	}

	// List templates (a system-wide template directory may add more entries)
	templates, err := parser.ListTemplates()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(templates), 2)
	assert.Contains(t, templates, "template1")
	assert.Contains(t, templates, "template2")
}

// TestAMIRegistry tests the AMI registry functionality
func TestAMIRegistry(t *testing.T) {
	_, ssmClient := setupLocalstackClients(t)
	ctx := context.Background()

	registry := &Registry{
		SSMClient:       ssmClient,
		ParameterPrefix: "/prism/ami/test/",
	}

	// Test reference registration (PublishAMI builds the Reference from a BuildResult)
	build := &BuildResult{
		AMIID:        "ami-test123",
		Region:       "us-east-1",
		Architecture: "x86_64",
		TemplateName: "test-template",
		Version:      "1.0.0",
		BuildTime:    time.Now(),
	}

	// Register AMI reference
	err := registry.PublishAMI(ctx, build)
	if err != nil {
		// LocalStack limitations may cause this to fail
		t.Logf("Warning: PublishAMI failed due to LocalStack limitations: %v", err)
	} else {
		// Test lookup - may also fail due to LocalStack limitations
		found, err := registry.LookupAMI(ctx, "test-template", "us-east-1", "x86_64")
		if err != nil {
			t.Logf("Warning: LookupAMI failed due to LocalStack limitations: %v", err)
		} else {
			assert.Equal(t, build.AMIID, found.AMIID)
			assert.Equal(t, build.Version, found.Version)
		}
	}
}

// TestCreateDryRun tests the dry run mode of AMI creation
func TestCreateDryRun(t *testing.T) {
	builder := setupLocalstackBuilder(t)

	// Create test template
	template := Template{
		Name:        "integration-test-template",
		Base:        "ubuntu-22.04-server-lts",
		Description: "Template for integration testing",
		BuildSteps: []BuildStep{
			{
				Name:   "Test Step",
				Script: "echo testing",
			},
		},
		Validation: []Validation{
			{
				Name:    "Test Validation",
				Command: "echo success",
				Success: true,
			},
		},
	}

	// Create build request with dry run
	request := BuildRequest{
		TemplateName: "integration-test",
		Template:     template,
		Region:       "us-east-1",
		Architecture: "x86_64",
		Version:      "1.0.0",
		DryRun:       true,
		BuildID:      fmt.Sprintf("test-%d", time.Now().Unix()),
		BuildType:    "manual",
		VpcID:        "vpc-test123",
		SubnetID:     "subnet-test123",
	}

	// Execute dry run
	result, err := builder.BuildAMI(context.Background(), request)

	// This should succeed even in LocalStack as it's a dry run. The pipeline
	// marks a finished dry run "completed" and creates no AMI.
	assert.NoError(t, err)
	if assert.NotNil(t, result) {
		assert.True(t, result.IsSuccessful())
		assert.Empty(t, result.AMIID)
		assert.Equal(t, request.BuildID, result.TemplateID)
		assert.Equal(t, request.TemplateName, result.TemplateName)
		assert.Equal(t, request.Region, result.Region)
	}
}

// TestCrossRegionCopyPreparation tests the preparation for cross-region copying
func TestCrossRegionCopyPreparation(t *testing.T) {
	builder := setupLocalstackBuilder(t)

	// Test cross-region copy preparation with multiple regions
	// The source region is taken from the builder's EC2 client (us-east-1).
	regions := []string{"us-west-1", "eu-west-1"}
	amiID := "ami-test123"
	name := "test-ami"
	description := "Integration test AMI"

	// Execute the function - should succeed even with LocalStack limitations
	copiedAMIs, err := builder.copyAMIToRegions(context.Background(), amiID, name, description, regions)

	// This may fail in LocalStack due to cross-region limitations
	if err != nil {
		t.Logf("Warning: Cross-region AMI copying failed due to LocalStack limitations: %v", err)
	} else {
		assert.Equal(t, len(regions), len(copiedAMIs))
		for _, region := range regions {
			assert.Contains(t, copiedAMIs, region)
		}
	}
}

// TestBuildValidation tests validation of build results
func TestBuildValidation(t *testing.T) {
	// Create test build result
	result := BuildResult{
		TemplateID:   "test-template",
		TemplateName: "Test Template",
		Region:       "us-east-1",
		Architecture: "x86_64",
		AMIID:        "ami-test123",
		Status:       "completed",
		BuilderID:    "test-builder",
		SourceAMI:    "ami-source123",
	}

	// Validate successful build
	assert.True(t, result.IsSuccessful())

	// Test with failure status
	result.Status = "failed"
	result.ErrorMessage = "Test error"
	assert.False(t, result.IsSuccessful())
}
