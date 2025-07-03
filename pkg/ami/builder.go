// Package ami provides CloudWorkstation's AMI creation system.
package ami

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssm_types "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/google/uuid"
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
func (b *Builder) BuildAMI(ctx context.Context, request BuildRequest) (*BuildResult, error) {
	// Generate a unique build ID if not provided
	if request.BuildID == "" {
		request.BuildID = uuid.New().String()[:8]
	}
	
	// Start timing
	buildStart := time.Now()
	
	// Initialize result
	result := &BuildResult{
		TemplateID:    request.BuildID,
		TemplateName:  request.TemplateName,
		Region:        request.Region,
		Architecture:  request.Architecture,
		Status:        "in_progress",
		BuilderID:     "",
	}
	
	// 1. Launch a builder instance
	instanceID, err := b.launchBuilderInstance(ctx, request)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("failed to launch builder instance: %v", err)
		return result, err
	}
	result.BuilderID = instanceID
	
	// Ensure instance cleanup
	defer func() {
		if err := b.terminateInstance(context.Background(), instanceID); err != nil {
			// Log but don't fail - this is best effort cleanup
			fmt.Printf("Warning: failed to terminate builder instance %s: %v\n", instanceID, err)
		}
	}()
	
	// 2. Wait for instance to be ready
	if err := b.waitForInstanceReady(ctx, instanceID); err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("instance failed to become ready: %v", err)
		return result, err
	}
	
	// 3. Execute build steps
	validator := NewValidator(b.SSMClient, ValidatorOptions{
		FailFast:    false,
		LogProgress: true,
	})
	
	// Create a build log
	var buildLog strings.Builder
	buildLog.WriteString(fmt.Sprintf("Build started at %s\n", buildStart.Format(time.RFC3339)))
	buildLog.WriteString(fmt.Sprintf("Template: %s\n", request.TemplateName))
	buildLog.WriteString(fmt.Sprintf("Architecture: %s\n", request.Architecture))
	buildLog.WriteString(fmt.Sprintf("Region: %s\n\n", request.Region))
	
	// Print build info
	fmt.Printf("📋 Building AMI for %s template\n", request.TemplateName)
	fmt.Printf("🔧 Architecture: %s | Region: %s\n", request.Architecture, request.Region)
	
	// Execute each build step
	for i, step := range request.Template.BuildSteps {
		stepStart := time.Now()
		buildLog.WriteString(fmt.Sprintf("Step %d: %s\n", i+1, step.Name))
		
		// Print progress
		fmt.Printf("\n🔄 Step %d/%d: %s\n", i+1, len(request.Template.BuildSteps), step.Name)
		
		// Execute the build step
		output, err := b.executeStep(ctx, instanceID, step)
		if err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			result.Status = "failed"
			result.ErrorMessage = fmt.Sprintf("build step '%s' failed: %v", step.Name, err)
			buildLog.WriteString(fmt.Sprintf("FAILED: %v\nOutput:\n%s\n\n", err, output))
			result.Logs = buildLog.String()
			return result, err
		}
		
		stepDuration := time.Since(stepStart)
		fmt.Printf("✅ Completed in %s\n", stepDuration)
		buildLog.WriteString(fmt.Sprintf("SUCCESS (%s)\nOutput:\n%s\n\n", stepDuration, output))
	}
	
	// 4. Validate the build
	fmt.Printf("\n🔍 Running validation tests...\n")
	buildLog.WriteString("Validation:\n")
	if len(request.Template.Validation) > 0 {
		validationResult, err := validator.ValidateAMI(ctx, instanceID, request.Template.Validation)
		if err != nil {
			result.Status = "failed"
			result.ErrorMessage = fmt.Sprintf("validation failed: %v", err)
			buildLog.WriteString(fmt.Sprintf("FAILED: %v\n", err))
			result.Logs = buildLog.String()
			return result, err
		}
		
		if !validationResult.Successful {
			fmt.Printf("❌ Validation failed: %d/%d tests passed\n", 
				validationResult.SuccessfulTests, validationResult.TotalTests)
			result.Status = "failed"
			result.ErrorMessage = fmt.Sprintf("validation failed: %d/%d tests passed", 
				validationResult.SuccessfulTests, validationResult.TotalTests)
			buildLog.WriteString(validator.FormatValidationResult(validationResult))
			result.Logs = buildLog.String()
			result.ValidationLog = validator.FormatValidationResult(validationResult)
			return result, fmt.Errorf("validation failed")
		}
		
		fmt.Printf("✅ All validation tests passed (%d/%d)\n", 
			validationResult.SuccessfulTests, validationResult.TotalTests)
		buildLog.WriteString(fmt.Sprintf("SUCCESS: %d/%d tests passed\n\n", 
			validationResult.SuccessfulTests, validationResult.TotalTests))
		result.ValidationLog = validator.FormatValidationResult(validationResult)
	} else {
		buildLog.WriteString("No validation tests specified\n\n")
	}
	
	// 5. Create the AMI
	fmt.Printf("\n📸 Creating AMI...\n")
	if !request.DryRun {
		amiID, err := b.createAMI(ctx, instanceID, request)
		if err != nil {
			result.Status = "failed"
			result.ErrorMessage = fmt.Sprintf("failed to create AMI: %v", err)
			buildLog.WriteString(fmt.Sprintf("AMI Creation FAILED: %v\n", err))
			result.Logs = buildLog.String()
			return result, err
		}
		
		result.AMIID = amiID
		fmt.Printf("✅ AMI created successfully: %s\n", amiID)
		buildLog.WriteString(fmt.Sprintf("AMI Created: %s\n", amiID))
		
		// 6. Register AMI with registry
		if b.RegistryClient != nil {
			if err := b.RegistryClient.PublishAMI(ctx, result); err != nil {
				buildLog.WriteString(fmt.Sprintf("Registry Update FAILED: %v\n", err))
				// Continue anyway - this is non-fatal
			} else {
				fmt.Printf("📝 AMI registered in template registry\n")
				buildLog.WriteString("AMI registered in template registry\n")
			}
		}
	} else {
		// For dry run, use a dummy AMI ID
		result.AMIID = "ami-dryrun"
		fmt.Printf("✅ DRY RUN: AMI creation simulation successful\n")
		buildLog.WriteString("DRY RUN: AMI creation simulation successful\n")
		buildLog.WriteString("DRY RUN: Would create AMI with template settings\n")
		buildLog.WriteString("DRY RUN: Would register AMI in template registry\n")
	}
	
	// Finalize result
	buildDuration := time.Since(buildStart)
	result.Status = "success"
	result.BuildTime = buildDuration
	fmt.Printf("\n🎉 Build completed in %s\n", buildDuration)
	buildLog.WriteString(fmt.Sprintf("\nBuild completed in %s\n", buildDuration))
	result.Logs = buildLog.String()
	
	return result, nil
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
		return "", fmt.Errorf("subnet ID is required - specify with --subnet parameter")
	}
	
	// Use direct AWS CLI instead for launch - AWS SDK issues with subnet
	fmt.Printf("Using direct AWS CLI for launching instance...\n")
	
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
		Groups:                   []string{"sg-052d842a020512194"}, // Default security group
	}
	
	// Launch the instance
	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(baseAMI),
		InstanceType: types.InstanceType(instanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
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
		return "", fmt.Errorf("failed to launch instance: %w", err)
	}
	
	if len(result.Instances) == 0 {
		return "", fmt.Errorf("no instances were launched")
	}
	
	instanceID := *result.Instances[0].InstanceId
	fmt.Printf("Successfully launched instance %s\n", instanceID)
	
	return instanceID, nil
	
	// Use request subnet first if specified (command line parameter)
	var subnetToUse string
	if request.SubnetID != "" {
		subnetToUse = request.SubnetID
		fmt.Printf("Using subnet from request: %s\n", subnetToUse)
	} else if b.DefaultSubnet != "" {
		subnetToUse = b.DefaultSubnet
		fmt.Printf("Using default subnet: %s\n", subnetToUse)
	} else if request.DryRun {
		// For dry run mode, use a dummy subnet ID
		subnetToUse = "subnet-dummy"
	} else {
		return "", fmt.Errorf("subnet ID is required - specify with --subnet parameter")
	}
	
	// Set the subnet ID
	input.SubnetId = aws.String(subnetToUse)
	
	// Add security group if specified
	if b.SecurityGroupID != "" {
		fmt.Printf("Using security group: %s\n", b.SecurityGroupID)
		input.SecurityGroupIds = []string{b.SecurityGroupID}
	}
	
	if request.DryRun {
		// For dry run mode, handle anything special here
		// This block intentionally left mostly empty
	}
	
	// Add IAM profile if specified
	if iamInstanceProfile != nil {
		input.IamInstanceProfile = iamInstanceProfile
	}
	
	// Check VPC configuration if specified
	if request.VpcID != "" {
		fmt.Printf("Using VPC: %s\n", request.VpcID)
	}
	
	// Handle dry run specially
	if request.DryRun {
		// In dry run mode, just return a dummy instance ID
		return "i-dryruninstance", nil
	}
	
	// Launch the instance
	result, err := b.EC2Client.RunInstances(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to launch instance: %w", err)
	}
	
	if len(result.Instances) == 0 {
		return "", fmt.Errorf("no instances were launched")
	}
	
	instanceID := *result.Instances[0].InstanceId
	return instanceID, nil
}

// getBaseAMI returns the AMI ID for the specified base image
func (b *Builder) getBaseAMI(baseName, region, architecture string) (string, error) {
	// For now, use hardcoded mapping - in the future this would be dynamic
	// Example Ubuntu 22.04 LTS AMIs
	if b.BaseAMIs == nil || len(b.BaseAMIs) == 0 {
		// Initialize with default Ubuntu 22.04 AMIs
		b.BaseAMIs = map[string]map[string]string{
			"us-east-1": {
				"ubuntu-22.04-server-lts": "ami-02029c87fa31fb148", // x86_64
				"ubuntu-22.04-server-lts-arm64": "ami-050499786ebf55a6a", // arm64
			},
			"us-west-2": {
				"ubuntu-22.04-server-lts": "ami-016d360a89daa11ba", // x86_64
				"ubuntu-22.04-server-lts-arm64": "ami-09f6c9efbf93542be", // arm64
			},
		}
	}
	
	// Check if we have AMIs for this region
	regionAMIs, ok := b.BaseAMIs[region]
	if !ok {
		return "", fmt.Errorf("no AMIs defined for region %s", region)
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
		return "", fmt.Errorf("no AMI found for base '%s' in region %s", baseName, region)
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
		return fmt.Errorf("timeout waiting for instance to be running: %w", err)
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
	
	return fmt.Errorf("timeout waiting for instance to be ready for SSM commands")
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
		return "", fmt.Errorf("failed to send SSM command: %w", err)
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
				return *result.StandardOutputContent, fmt.Errorf(errorMsg)
			}
			
		case <-waiterCtx.Done():
			return "", fmt.Errorf("timeout waiting for command to complete")
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
		InstanceId:  aws.String(instanceID),
		Name:        aws.String(amiName),
		Description: aws.String(fmt.Sprintf("CloudWorkstation %s template", request.TemplateName)),
		TagSpecifications: tags,
	}
	
	result, err := b.EC2Client.CreateImage(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to create AMI: %w", err)
	}
	
	// Wait for AMI to be available
	waiter := ec2.NewImageAvailableWaiter(b.EC2Client)
	if err := waiter.Wait(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{*result.ImageId},
	}, 30*time.Minute); err != nil {
		return *result.ImageId, fmt.Errorf("timeout waiting for AMI to be available: %w", err)
	}
	
	return *result.ImageId, nil
}

// terminateInstance cleans up the builder instance
func (b *Builder) terminateInstance(ctx context.Context, instanceID string) error {
	_, err := b.EC2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	return err
}