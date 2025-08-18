// Package ami provides CloudWorkstation's AMI creation system.
package ami

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssm_types "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// NewBuilder creates a new AMI builder with the provided configuration
func NewBuilder(ec2Client *ec2.Client, ssmClient *ssm.Client, registry *Registry, config map[string]string) (*Builder, error) {
	// Default values
	vpcID := config["vpc_id"]
	subnetID := config["subnet_id"]
	securityGroupID := config["security_group_id"]
	builderRole := config["builder_role"]
	builderProfile := config["builder_profile"]

	// Create AMI builder
	return &Builder{
		EC2Client:       ec2Client,
		SSMClient:       ssmClient,
		RegistryClient:  registry,
		BaseAMIs:        make(map[string]map[string]string),
		DefaultVPC:      vpcID,
		DefaultSubnet:   subnetID,
		SecurityGroupID: securityGroupID,
		BuilderRole:     builderRole,
		BuilderProfile:  builderProfile,
	}, nil
}

// BuildAMI builds an AMI from a template
// BuildAMI builds an AMI using the Builder Pattern (SOLID: Single Responsibility)
func (b *Builder) BuildAMI(ctx context.Context, request BuildRequest) (*BuildResult, error) {
	// Create build pipeline
	pipeline := NewBuildPipeline(b, request)

	// Ensure instance cleanup
	defer func() {
		if pipeline.result.BuilderID != "" {
			if err := b.terminateInstance(context.Background(), pipeline.result.BuilderID); err != nil {
				fmt.Printf("Warning: failed to terminate builder instance %s: %v\n", pipeline.result.BuilderID, err)
			}
		}
	}()

	// Execute the build pipeline
	return pipeline.Execute(ctx)
}

// launchBuilderInstance launches an EC2 instance for building the AMI
func (b *Builder) launchBuilderInstance(ctx context.Context, request BuildRequest) (string, error) {
	// Get base AMI ID for the requested architecture and region
	baseAMI, err := b.getBaseAMI(request.Template.Base, request.Region, request.Architecture)
	if err != nil {
		return "", err
	}

	// Determine instance type based on architecture
	var instanceType string
	if request.Architecture == "arm64" {
		instanceType = "t4g.medium" // ARM instance
	} else {
		instanceType = "t3.medium" // x86 instance
	}

	// Check for subnet
	if request.SubnetID == "" && !request.DryRun {
		return "", ValidationError("subnet ID is required - specify with --subnet parameter", nil)
	}

	// Debug info for subnet and VPC
	fmt.Printf("Using subnet: %s\n", request.SubnetID)
	if request.VpcID != "" {
		fmt.Printf("Using VPC: %s\n", request.VpcID)
	}

	// Handle dry run specially
	if request.DryRun {
		// In dry run mode, just return a dummy instance ID
		return "i-dryruninstance", nil
	}

	// Create tags
	tags := []types.Tag{
		{
			Key:   aws.String("Name"),
			Value: aws.String(fmt.Sprintf("ami-builder-%s-%s", request.TemplateName, request.BuildID)),
		},
		{
			Key:   aws.String("CloudWorkstationBuilderID"),
			Value: aws.String(request.BuildID),
		},
		{
			Key:   aws.String("CloudWorkstationTemplate"),
			Value: aws.String(request.TemplateName),
		},
		{
			Key:   aws.String("CloudWorkstationBuildType"),
			Value: aws.String(request.BuildType),
		},
	}

	// Prepare network interface with subnet and security group
	networkInterface := types.InstanceNetworkInterfaceSpecification{
		DeviceIndex:              aws.Int32(0),
		AssociatePublicIpAddress: aws.Bool(true),
		SubnetId:                 aws.String(request.SubnetID),
	}

	// Add security group if specified
	if request.SecurityGroup != "" {
		networkInterface.Groups = []string{request.SecurityGroup}
	} else if b.SecurityGroupID != "" {
		networkInterface.Groups = []string{b.SecurityGroupID}
	} else {
		// If default security group isn't available, look up the default security group for the VPC
		defaultSG, err := b.getDefaultSecurityGroup(ctx, request.VpcID)
		if err != nil {
			return "", NetworkError("no security group specified and failed to find default", err)
		}
		networkInterface.Groups = []string{defaultSG}
	}

	// Launch the instance
	input := &ec2.RunInstancesInput{
		ImageId:           aws.String(baseAMI),
		InstanceType:      types.InstanceType(instanceType),
		MinCount:          aws.Int32(1),
		MaxCount:          aws.Int32(1),
		NetworkInterfaces: []types.InstanceNetworkInterfaceSpecification{networkInterface},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         tags,
			},
		},
		InstanceInitiatedShutdownBehavior: types.ShutdownBehaviorTerminate,
	}

	// Launch the instance
	result, err := b.EC2Client.RunInstances(ctx, input)
	if err != nil {
		return "", InstanceError("failed to launch instance", err)
	}

	if len(result.Instances) == 0 {
		return "", InstanceError("no instances were launched", nil)
	}

	instanceID := *result.Instances[0].InstanceId
	fmt.Printf("Successfully launched instance %s\n", instanceID)

	return instanceID, nil
}

// getBaseAMI returns the AMI ID for the specified base image
func (b *Builder) getBaseAMI(baseName, region, architecture string) (string, error) {
	// Initialize base AMIs if not already initialized
	if len(b.BaseAMIs) == 0 {
		b.initializeBaseAMIs()
	}

	// Check if we have AMIs for this region
	regionAMIs, ok := b.BaseAMIs[region]
	if !ok {
		return "", ValidationError(
			fmt.Sprintf("no AMIs defined for region %s. Supported regions: %s", region, b.getSupportedRegions()),
			nil,
		).WithContext("region", region)
	}

	// Check architecture-specific base
	var baseKey string
	if architecture == "arm64" {
		baseKey = baseName + "-arm64"
		if _, ok := regionAMIs[baseKey]; !ok {
			baseKey = baseName // Try without architecture suffix
		}
	} else {
		baseKey = baseName
	}

	// Get the AMI ID
	amiID, ok := regionAMIs[baseKey]
	if !ok {
		return "", ValidationError(
			fmt.Sprintf("no AMI found for base '%s' in region %s", baseName, region),
			nil,
		).WithContext("base", baseName).WithContext("region", region).WithContext("architecture", architecture)
	}

	return amiID, nil
}

// waitForInstanceReady waits for an instance to be ready for SSM commands
func (b *Builder) waitForInstanceReady(ctx context.Context, instanceID string) error {
	// For dry run instance ID, return immediately
	if instanceID == "i-dryruninstance" {
		return nil
	}

	// Wait for instance to be running
	waiter := ec2.NewInstanceRunningWaiter(b.EC2Client)
	if err := waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}, 5*time.Minute); err != nil {
		return NewRetryableBuildError(ErrorTypeInstance, "timeout waiting for instance to be running", err).WithContext("instanceID", instanceID)
	}

	// Wait for SSM agent to be ready
	// We'll need to poll for SSM status since there's no dedicated waiter
	maxAttempts := 30
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Check if instance is ready for SSM
		output, err := b.SSMClient.DescribeInstanceInformation(ctx, &ssm.DescribeInstanceInformationInput{
			Filters: []ssm_types.InstanceInformationStringFilter{
				{
					Key:    aws.String("InstanceIds"),
					Values: []string{instanceID},
				},
			},
		})
		if err != nil {
			// Ignore errors and keep trying
			time.Sleep(10 * time.Second)
			continue
		}

		// Check if the instance is registered with SSM
		if len(output.InstanceInformationList) > 0 {
			if output.InstanceInformationList[0].PingStatus == ssm_types.PingStatusOnline {
				return nil
			}
		}

		time.Sleep(10 * time.Second)
	}

	return SSMError("timeout waiting for instance to be ready for SSM commands", nil).WithContext("instanceID", instanceID)
}

// executeStep runs a build step on the instance
func (b *Builder) executeStep(ctx context.Context, instanceID string, step BuildStep) (string, error) {
	// For dry run instance ID, just return success
	if instanceID == "i-dryruninstance" {
		return fmt.Sprintf("[DRY RUN] Would execute: %s\n%s", step.Name, step.Script), nil
	}

	// Set default timeout if not specified
	timeoutSeconds := int32(600) // 10 minutes default
	if step.TimeoutSeconds > 0 {
		timeoutSeconds = int32(step.TimeoutSeconds)
	}

	// Prepare SSM SendCommand
	input := &ssm.SendCommandInput{
		DocumentName:   aws.String("AWS-RunShellScript"),
		InstanceIds:    []string{instanceID},
		TimeoutSeconds: aws.Int32(timeoutSeconds),
		Parameters: map[string][]string{
			"commands": {step.Script},
		},
	}

	// Execute command
	output, err := b.SSMClient.SendCommand(ctx, input)
	if err != nil {
		return "", SSMError("failed to send SSM command", err).WithContext("step", step.Name).WithContext("instanceID", instanceID)
	}

	// Wait for command to complete
	commandID := *output.Command.CommandId
	waiterCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds+30)*time.Second)
	defer cancel()

	// Poll for completion
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var cmdOutput string

	for {
		select {
		case <-ticker.C:
			result, err := b.SSMClient.GetCommandInvocation(waiterCtx, &ssm.GetCommandInvocationInput{
				CommandId:  aws.String(commandID),
				InstanceId: aws.String(instanceID),
			})
			if err != nil {
				// Continue on transient errors
				continue
			}

			// Check if command has completed
			switch result.Status {
			case ssm_types.CommandInvocationStatusSuccess:
				cmdOutput = *result.StandardOutputContent
				return cmdOutput, nil

			case ssm_types.CommandInvocationStatusFailed,
				ssm_types.CommandInvocationStatusCancelled,
				ssm_types.CommandInvocationStatusTimedOut:
				// Command failed
				errorMsg := *result.StandardErrorContent
				if errorMsg == "" {
					errorMsg = fmt.Sprintf("Command failed with status: %s", result.Status)
				}
				return *result.StandardOutputContent, CommandError(errorMsg, nil).WithContext("step", step.Name).WithContext("instanceID", instanceID).WithContext("status", string(result.Status))
			}

		case <-waiterCtx.Done():
			return "", NewRetryableBuildError(ErrorTypeCommand, "timeout waiting for command to complete", nil).WithContext("step", step.Name).WithContext("instanceID", instanceID)
		}
	}
}

// createAMI creates an AMI from the instance
func (b *Builder) createAMI(ctx context.Context, instanceID string, request BuildRequest) (string, error) {
	// Prepare AMI name
	timestamp := time.Now().Format("20060102-150405")
	amiName := fmt.Sprintf("%s-%s-%s-%s",
		request.TemplateName,
		request.Architecture,
		request.Region,
		timestamp)

	// Prepare tags
	tags := []types.TagSpecification{
		{
			ResourceType: types.ResourceTypeImage,
			Tags: []types.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String(amiName),
				},
				{
					Key:   aws.String("CloudWorkstationTemplate"),
					Value: aws.String(request.TemplateName),
				},
				{
					Key:   aws.String("CloudWorkstationArchitecture"),
					Value: aws.String(request.Architecture),
				},
				{
					Key:   aws.String("CloudWorkstationBuildID"),
					Value: aws.String(request.BuildID),
				},
				{
					Key:   aws.String("CloudWorkstationBuildType"),
					Value: aws.String(request.BuildType),
				},
				{
					Key:   aws.String("CloudWorkstationBuildDate"),
					Value: aws.String(timestamp),
				},
			},
		},
	}

	// Create the AMI
	input := &ec2.CreateImageInput{
		InstanceId:        aws.String(instanceID),
		Name:              aws.String(amiName),
		Description:       aws.String(fmt.Sprintf("CloudWorkstation %s template", request.TemplateName)),
		TagSpecifications: tags,
	}

	result, err := b.EC2Client.CreateImage(ctx, input)
	if err != nil {
		return "", ImageCreationError("failed to create AMI", err).WithContext("instanceID", instanceID).WithContext("template", request.TemplateName)
	}

	// Wait for AMI to be available
	waiter := ec2.NewImageAvailableWaiter(b.EC2Client)
	if err := waiter.Wait(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{*result.ImageId},
	}, 30*time.Minute); err != nil {
		return *result.ImageId, NewRetryableBuildError(ErrorTypeImageCreation, "timeout waiting for AMI to be available", err).WithContext("amiID", *result.ImageId)
	}

	return *result.ImageId, nil
}

// CreateAMIFromInstance creates an AMI from a running CloudWorkstation instance using Builder Pattern (SOLID: Single Responsibility)
func (b *Builder) CreateAMIFromInstance(ctx context.Context, request InstanceSaveRequest) (*BuildResult, error) {
	// Create instance save pipeline
	pipeline := NewInstanceSavePipeline(b, request)

	// Execute the save pipeline
	return pipeline.Execute(ctx)
}

// restartInstanceBestEffort attempts to restart an instance (best effort, ignores errors)
func (b *Builder) restartInstanceBestEffort(ctx context.Context, instanceID string) {
	fmt.Printf("🔄 Restarting instance %s...\n", instanceID)

	_, err := b.EC2Client.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		fmt.Printf("⚠️ Warning: Failed to restart instance: %v\n", err)
		return
	}

	// Wait for instance to be running (with timeout)
	waiter := ec2.NewInstanceRunningWaiter(b.EC2Client)
	if err := waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}, 3*time.Minute); err != nil {
		fmt.Printf("⚠️ Warning: Instance may not have restarted successfully: %v\n", err)
	} else {
		fmt.Printf("✅ Instance restarted successfully\n")
	}
}

// createTemplateDefinition creates a YAML template definition for the saved AMI
func (b *Builder) createTemplateDefinition(request InstanceSaveRequest, result *BuildResult) error {
	templateDir := "templates"
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		if err := os.MkdirAll(templateDir, 0755); err != nil {
			return fmt.Errorf("failed to create templates directory: %w", err)
		}
	}

	templateFile := filepath.Join(templateDir, request.TemplateName+".yaml")

	// Create basic template structure
	templateContent := fmt.Sprintf(`name: "%s"
description: "%s"
base: "saved-instance"
source: "saved-from-instance"  
original_instance: "%s"
saved_from: "%s"
saved_date: "%s"

# AMI mappings (automatically populated)
ami_config:
  amis:
    %s:
      %s: "%s"

# Ports (inherited from original instance - may need manual adjustment)
ports: [22]

# Cost estimates (placeholder - update based on actual usage)
estimated_cost_per_hour:
  %s: 0.05

# Tags
tags:
  Name: "%s"
  Type: "saved-instance"
  Source: "CloudWorkstation-Save"
`,
		request.TemplateName,
		request.Description,
		request.InstanceName,
		request.InstanceName,
		time.Now().Format(time.RFC3339),
		result.Region,
		result.Architecture,
		result.AMIID,
		result.Architecture,
		request.TemplateName)

	// Add copied AMIs to template
	if len(result.CopiedAMIs) > 0 {
		for region, amiID := range result.CopiedAMIs {
			templateContent += fmt.Sprintf(`    %s:
      %s: "%s"
`, region, result.Architecture, amiID)
		}
	}

	return os.WriteFile(templateFile, []byte(templateContent), 0644)
}

// copyAMIToRegions copies an AMI to multiple regions
func (b *Builder) copyAMIToRegions(ctx context.Context, sourceAMIID, sourceName, sourceDescription string, targetRegions []string) (map[string]string, error) {
	// Skip if no target regions
	if len(targetRegions) == 0 {
		return nil, nil
	}

	// Initialize result map (region -> AMI ID)
	result := make(map[string]string)

	// Source region (where original AMI was created)
	sourceRegion := string(b.EC2Client.Options().Region)

	// Skip regions that match source region
	var regions []string
	for _, r := range targetRegions {
		if r != sourceRegion {
			regions = append(regions, r)
		}
	}

	if len(regions) == 0 {
		// No valid target regions
		return result, nil
	}

	fmt.Printf("\n🌎 Copying AMI to %d additional regions...\n", len(regions))

	// Copy to each region in parallel using goroutines
	var wg sync.WaitGroup
	ch := make(chan struct {
		region string
		amiID  string
		err    error
	}, len(regions))

	for _, targetRegion := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			amiID, err := b.copyAMIToRegion(ctx, sourceAMIID, sourceName, sourceDescription, sourceRegion, region)
			ch <- struct {
				region string
				amiID  string
				err    error
			}{region, amiID, err}
		}(targetRegion)
	}

	// Close channel when all goroutines are done
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Collect results
	var copyErrors []string
	for res := range ch {
		if res.err != nil {
			fmt.Printf("❌ Failed to copy to region %s: %v\n", res.region, res.err)
			copyErrors = append(copyErrors, fmt.Sprintf("%s: %v", res.region, res.err))
		} else {
			fmt.Printf("✅ AMI copied to region %s: %s\n", res.region, res.amiID)
			result[res.region] = res.amiID
		}
	}

	// Return error if any copies failed
	if len(copyErrors) > 0 {
		return result, ImageCreationError(
			fmt.Sprintf("some AMI copies failed: %s", strings.Join(copyErrors, "; ")),
			nil,
		)
	}

	return result, nil
}

// copyAMIToRegion copies an AMI to a specific region
func (b *Builder) copyAMIToRegion(ctx context.Context, sourceAMIID, sourceName, sourceDescription, sourceRegion, targetRegion string) (string, error) {
	// Create a new EC2 client for the target region
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(targetRegion))
	if err != nil {
		return "", ConfigurationError(
			fmt.Sprintf("failed to create config for region %s", targetRegion),
			err,
		).WithContext("region", targetRegion)
	}
	targetClient := ec2.NewFromConfig(cfg)

	// Copy the AMI
	copyInput := &ec2.CopyImageInput{
		SourceRegion:  aws.String(sourceRegion),
		SourceImageId: aws.String(sourceAMIID),
		Name:          aws.String(sourceName + "-copied"),
		Description:   aws.String(sourceDescription + " (copied from " + sourceRegion + ")"),
		Encrypted:     aws.Bool(false), // Not encrypting for simplicity
		CopyImageTags: aws.Bool(true),
	}

	result, err := targetClient.CopyImage(ctx, copyInput)
	if err != nil {
		return "", ImageCreationError(
			fmt.Sprintf("failed to copy AMI to region %s", targetRegion),
			err,
		).WithContext("sourceAMI", sourceAMIID).
			WithContext("sourceRegion", sourceRegion).
			WithContext("targetRegion", targetRegion)
	}

	// Wait for the AMI to be available in the target region
	waiter := ec2.NewImageAvailableWaiter(targetClient)
	if err := waiter.Wait(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{*result.ImageId},
	}, 30*time.Minute); err != nil {
		return *result.ImageId, NewRetryableBuildError(
			ErrorTypeImageCreation,
			fmt.Sprintf("timeout waiting for AMI to be available in region %s", targetRegion),
			err,
		).WithContext("amiID", *result.ImageId).
			WithContext("region", targetRegion)
	}

	return *result.ImageId, nil
}

// terminateInstance cleans up the builder instance
func (b *Builder) terminateInstance(ctx context.Context, instanceID string) error {
	_, err := b.EC2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return InstanceError("failed to terminate builder instance", err).WithContext("instanceID", instanceID)
	}
	return nil
}

// getSupportedRegions returns a comma-separated list of supported regions
func (b *Builder) getSupportedRegions() string {
	// Initialize base AMIs if not already initialized
	if len(b.BaseAMIs) == 0 {
		b.initializeBaseAMIs()
	}

	regions := make([]string, 0, len(b.BaseAMIs))
	for region := range b.BaseAMIs {
		regions = append(regions, region)
	}
	return strings.Join(regions, ", ")
}

// initializeBaseAMIs initializes the base AMI mappings
func (b *Builder) initializeBaseAMIs() {
	b.BaseAMIs = map[string]map[string]string{
		"us-east-1": {
			"ubuntu-22.04-server-lts":       "ami-02029c87fa31fb148", // x86_64
			"ubuntu-22.04-server-lts-arm64": "ami-050499786ebf55a6a", // arm64
		},
		"us-east-2": {
			"ubuntu-22.04-server-lts":       "ami-0574da8cbe4a3a80a", // x86_64
			"ubuntu-22.04-server-lts-arm64": "ami-0960ab2a240c27ff3", // arm64
		},
		"us-west-1": {
			"ubuntu-22.04-server-lts":       "ami-085a8d7b63d031cba", // x86_64
			"ubuntu-22.04-server-lts-arm64": "ami-0a0a5204d8d741180", // arm64
		},
		"us-west-2": {
			"ubuntu-22.04-server-lts":       "ami-016d360a89daa11ba", // x86_64
			"ubuntu-22.04-server-lts-arm64": "ami-09f6c9efbf93542be", // arm64
		},
		"eu-west-1": {
			"ubuntu-22.04-server-lts":       "ami-0694d931cee3dc7bb", // x86_64
			"ubuntu-22.04-server-lts-arm64": "ami-0a1b0de9ee4ddd0a5", // arm64
		},
		"eu-central-1": {
			"ubuntu-22.04-server-lts":       "ami-0faab6bdbac9486fb", // x86_64
			"ubuntu-22.04-server-lts-arm64": "ami-064334c2008d4f9cd", // arm64
		},
		"ap-northeast-1": {
			"ubuntu-22.04-server-lts":       "ami-0ffac9ed219ecde9d", // x86_64
			"ubuntu-22.04-server-lts-arm64": "ami-0a3de148326a5527d", // arm64
		},
		"ap-southeast-1": {
			"ubuntu-22.04-server-lts":       "ami-078c1149e8a47c0f0", // x86_64
			"ubuntu-22.04-server-lts-arm64": "ami-026a9429bd57a973a", // arm64
		},
	}
}

// validateRegion checks if the specified region is supported
func (b *Builder) validateRegion(region string) error {
	// Initialize base AMIs if not already initialized
	if len(b.BaseAMIs) == 0 {
		b.initializeBaseAMIs()
	}

	if _, ok := b.BaseAMIs[region]; !ok {
		return ValidationError(
			fmt.Sprintf("region %s is not supported. Supported regions: %s", region, b.getSupportedRegions()),
			nil,
		).WithContext("region", region).WithContext("supported_regions", b.getSupportedRegions())
	}
	return nil
}

// getDefaultSecurityGroup retrieves the default security group for a VPC
func (b *Builder) getDefaultSecurityGroup(ctx context.Context, vpcID string) (string, error) {
	// Find the default security group for the VPC
	result, err := b.EC2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
			{
				Name:   aws.String("group-name"),
				Values: []string{"default"},
			},
		},
	})

	if err != nil {
		return "", NetworkError("failed to describe security groups", err).WithContext("vpcID", vpcID)
	}

	if len(result.SecurityGroups) == 0 {
		return "", NetworkError(
			fmt.Sprintf("no default security group found for VPC %s", vpcID),
			nil,
		).WithContext("vpcID", vpcID)
	}

	return *result.SecurityGroups[0].GroupId, nil
}
