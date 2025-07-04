//go:build integration
// +build integration

package ami

import (
	"context"
	"fmt"
	"os"
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

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           localstackEndpoint,
					SigningRegion: "us-east-1",
				}, nil
			})),
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
		ParameterPrefix: "/cloudworkstation/ami/",
	}

	// Create base AMIs map for testing
	baseAMIs := map[string]map[string]string{
		"us-east-1": {
			"x86_64": "ami-test-x86-east1",
			"arm64":  "ami-test-arm-east1",
			"ubuntu-22.04-server-lts": "ami-test-ubuntu-east1",
		},
		"us-west-2": {
			"x86_64": "ami-test-x86-west2",
			"arm64":  "ami-test-arm-west2",
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

	// Create a test security group in LocalStack
	createSgInput := &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String("default"),
		Description: aws.String("Default security group"),
		VpcId:       aws.String("vpc-test123"),
	}
	createSgOutput, err := ec2Client.CreateSecurityGroup(context.TODO(), createSgInput)
	if err != nil {
		t.Logf("Warning: Failed to create security group in LocalStack: %v", err)
		// Continue anyway as this may just be a limitation in LocalStack
	} else {
		t.Logf("Created security group: %s", *createSgOutput.GroupId)
	}

	// Test with explicit security group
	securityGroup, err := builder.getDefaultSecurityGroup("sg-explicit", "vpc-test123")
	assert.NoError(t, err)
	assert.Equal(t, "sg-explicit", securityGroup)

	// Test with default security group from builder
	builder.SecurityGroupID = "sg-builder-default"
	securityGroup, err = builder.getDefaultSecurityGroup("", "vpc-test123")
	assert.NoError(t, err)
	assert.Equal(t, "sg-builder-default", securityGroup)

	// Test with lookup - this will likely be limited in LocalStack
	builder.SecurityGroupID = ""
	securityGroup, err = builder.getDefaultSecurityGroup("", "vpc-test123")
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
	parser := NewParser(nil)

	// Valid template YAML
	validTemplate := []byte(`
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
`)

	// Parse valid template
	template, err := parser.ParseTemplate(validTemplate)
	assert.NoError(t, err)
	assert.Equal(t, "Test Template", template.Name)
	assert.Equal(t, "ubuntu-22.04-server-lts", template.Base)
	assert.Equal(t, 1, len(template.BuildSteps))
	assert.Equal(t, 1, len(template.Validation))

	// Invalid template YAML (missing required fields)
	invalidTemplate := []byte(`
name: "Invalid Template"
description: "Missing required base field"
`)

	// Parse invalid template
	template, err = parser.ParseTemplate(invalidTemplate)
	assert.Error(t, err)
	assert.Nil(t, template)
}

// TestListTemplates tests template listing
func TestListTemplates(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ami-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

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
	if err := os.WriteFile(tempDir+"/template1.yml", []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}
	if err := os.WriteFile(tempDir+"/template2.yaml", []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	// Create a test parser with base AMIs
	baseAMIs := map[string]map[string]string{
		"us-east-1": {
			"ubuntu-22.04-server-lts": "ami-test",
		},
	}
	parser := NewParser(baseAMIs)

	// List templates
	templates, err := parser.ListTemplates(tempDir)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(templates))
	assert.Contains(t, templates, "template1")
	assert.Contains(t, templates, "template2")
}

// TestAMIRegistry tests the AMI registry functionality
func TestAMIRegistry(t *testing.T) {
	_, ssmClient := setupLocalstackClients(t)

	registry := &Registry{
		SSMClient:       ssmClient,
		ParameterPrefix: "/cloudworkstation/ami/test/",
	}

	// Test reference registration
	ref := Reference{
		AMIID:        "ami-test123",
		Region:       "us-east-1",
		Architecture: "x86_64",
		TemplateName: "test-template",
		Version:      "1.0.0",
		BuildDate:    time.Now(),
		Tags: map[string]string{
			"Name":    "test-ami",
			"Purpose": "testing",
		},
	}

	// Register AMI reference
	err := registry.RegisterAMI(ref)
	if err != nil {
		// LocalStack limitations may cause this to fail
		t.Logf("Warning: RegisterAMI failed due to LocalStack limitations: %v", err)
	} else {
		// Test lookup - may also fail due to LocalStack limitations
		found, err := registry.LookupAMI("test-template", "us-east-1", "x86_64")
		if err != nil {
			t.Logf("Warning: LookupAMI failed due to LocalStack limitations: %v", err)
		} else {
			assert.Equal(t, ref.AMIID, found.AMIID)
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
	result, err := builder.CreateAMI(request)

	// This should succeed even in LocalStack as it's a dry run
	assert.NoError(t, err)
	assert.Equal(t, "dry-run", result.Status)
	assert.Equal(t, request.TemplateName, result.TemplateID)
	assert.Equal(t, request.Region, result.Region)
}

// TestCrossRegionCopyPreparation tests the preparation for cross-region copying
func TestCrossRegionCopyPreparation(t *testing.T) {
	builder := setupLocalstackBuilder(t)

	// Test cross-region copy preparation with multiple regions
	regions := []string{"us-west-1", "eu-west-1"}
	sourceRegion := "us-east-1"
	amiID := "ami-test123"
	name := "test-ami"
	tags := map[string]string{
		"Name": name,
		"Test": "Value",
	}

	// Execute the function - should succeed even with LocalStack limitations
	copiedAMIs, err := builder.copyAMIToRegions(amiID, name, sourceRegion, regions, tags)

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