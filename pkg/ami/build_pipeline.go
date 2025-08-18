package ami

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/google/uuid"
)

// BuildPipeline represents the AMI build pipeline using Builder Pattern (SOLID)
type BuildPipeline struct {
	builder    *Builder
	request    BuildRequest
	result     *BuildResult
	buildLog   strings.Builder
	buildStart time.Time
}

// PipelineStep represents a single step in the build pipeline
type PipelineStep interface {
	Execute(ctx context.Context, pipeline *BuildPipeline) error
	GetName() string
}

// InstanceSaveStep represents a single step in the instance save pipeline
type InstanceSaveStep interface {
	Execute(ctx context.Context, pipeline *InstanceSavePipeline) error
	GetName() string
}

// NewBuildPipeline creates a new build pipeline
func NewBuildPipeline(builder *Builder, request BuildRequest) *BuildPipeline {
	// Initialize build result
	result := &BuildResult{
		TemplateID:   request.BuildID,
		TemplateName: request.TemplateName,
		Region:       request.Region,
		Architecture: request.Architecture,
		Status:       "in_progress",
		BuilderID:    "",
		CopiedAMIs:   make(map[string]string),
	}

	// Generate unique build ID if not provided
	if request.BuildID == "" {
		request.BuildID = uuid.New().String()[:8]
		result.TemplateID = request.BuildID
	}

	pipeline := &BuildPipeline{
		builder:    builder,
		request:    request,
		result:     result,
		buildStart: time.Now(),
	}

	// Initialize build log
	pipeline.buildLog.WriteString(fmt.Sprintf("Build started at %s\n", pipeline.buildStart.Format(time.RFC3339)))
	pipeline.buildLog.WriteString(fmt.Sprintf("Template: %s\n", request.TemplateName))
	pipeline.buildLog.WriteString(fmt.Sprintf("Architecture: %s\n", request.Architecture))
	pipeline.buildLog.WriteString(fmt.Sprintf("Region: %s\n\n", request.Region))

	return pipeline
}

// Execute runs the complete build pipeline
func (p *BuildPipeline) Execute(ctx context.Context) (*BuildResult, error) {
	// Create build steps in order
	steps := []PipelineStep{
		&ValidationStep{},
		&BaseAMIStep{},
		&BuilderInstanceStep{},
		&BuildExecutionStep{},
		&ValidationExecutionStep{},
		&AMICreationStep{},
		&AMICopyStep{},
		&CleanupStep{},
	}

	// Execute each step
	for _, step := range steps {
		fmt.Printf("\n🔄 %s\n", step.GetName())

		if err := step.Execute(ctx, p); err != nil {
			p.result.Status = "failed"
			p.result.ErrorMessage = err.Error()
			p.result.Logs = p.buildLog.String()
			return p.result, err
		}
	}

	// Build completed successfully
	buildDuration := time.Since(p.buildStart)
	p.result.Status = "completed"
	p.result.BuildDuration = buildDuration
	p.result.Logs = p.buildLog.String()

	fmt.Printf("\n🎉 Build completed successfully in %s\n", buildDuration)
	return p.result, nil
}

// LogStep logs a step result
func (p *BuildPipeline) LogStep(stepName string, success bool, duration time.Duration, output string, err error) {
	if success {
		p.buildLog.WriteString(fmt.Sprintf("SUCCESS (%s)\nOutput:\n%s\n\n", duration, output))
	} else {
		p.buildLog.WriteString(fmt.Sprintf("FAILED: %v\nOutput:\n%s\n\n", err, output))
	}
}

// InstanceSavePipeline represents the instance save pipeline using Builder Pattern (SOLID)
type InstanceSavePipeline struct {
	builder         *Builder
	request         InstanceSaveRequest
	result          *BuildResult
	saveLog         strings.Builder
	saveStart       time.Time
	instanceDetails *ec2.DescribeInstancesOutput
}

// NewInstanceSavePipeline creates a new instance save pipeline
func NewInstanceSavePipeline(builder *Builder, request InstanceSaveRequest) *InstanceSavePipeline {
	buildID := fmt.Sprintf("save-%s-%d", request.InstanceName, time.Now().Unix())

	result := &BuildResult{
		TemplateID:   buildID,
		TemplateName: request.TemplateName,
		Status:       "in_progress",
		BuilderID:    request.InstanceID,
		CopiedAMIs:   make(map[string]string),
	}

	pipeline := &InstanceSavePipeline{
		builder:   builder,
		request:   request,
		result:    result,
		saveStart: time.Now(),
	}

	// Initialize save log
	pipeline.saveLog.WriteString(fmt.Sprintf("Instance save started at %s\n", pipeline.saveStart.Format(time.RFC3339)))
	pipeline.saveLog.WriteString(fmt.Sprintf("Instance: %s\n", request.InstanceName))
	pipeline.saveLog.WriteString(fmt.Sprintf("Target Template: %s\n\n", request.TemplateName))

	return pipeline
}

// Execute runs the complete instance save pipeline
func (p *InstanceSavePipeline) Execute(ctx context.Context) (*BuildResult, error) {
	// Ensure instance restart on exit (best effort)
	defer func() {
		p.builder.restartInstanceBestEffort(ctx, p.request.InstanceID)
	}()

	// Create save steps in order
	steps := []InstanceSaveStep{
		&InstanceDetailsStep{},
		&InstanceStopStep{},
		&InstanceAMICreationStep{},
		&InstanceAMICopyStep{},
		&TemplateDefinitionStep{},
		&RegistryPublishStep{},
	}

	// Execute each step
	for _, step := range steps {
		fmt.Printf("\n🔄 %s\n", step.GetName())

		if err := step.Execute(ctx, p); err != nil {
			p.result.Status = "failed"
			p.result.ErrorMessage = err.Error()
			p.result.Logs = p.saveLog.String()
			return p.result, err
		}
	}

	// Save completed successfully
	saveDuration := time.Since(p.saveStart)
	p.result.Status = "success"
	p.result.BuildDuration = saveDuration
	p.result.BuildTime = time.Now()
	p.result.Logs = p.saveLog.String()

	fmt.Printf("\n🎉 Instance saved as AMI successfully in %s\n", saveDuration)
	return p.result, nil
}

// LogSaveStep logs a save step result
func (p *InstanceSavePipeline) LogSaveStep(stepName string, success bool, duration time.Duration, output string, err error) {
	if success {
		p.saveLog.WriteString(fmt.Sprintf("%s - SUCCESS (%s)\nOutput:\n%s\n\n", stepName, duration, output))
	} else {
		p.saveLog.WriteString(fmt.Sprintf("%s - FAILED: %v\nOutput:\n%s\n\n", stepName, err, output))
	}
}

// Build Steps Implementation

// ValidationStep validates the build request
type ValidationStep struct{}

func (v *ValidationStep) GetName() string {
	return "Validating build request"
}

func (v *ValidationStep) Execute(ctx context.Context, pipeline *BuildPipeline) error {
	// Validate region
	if err := pipeline.builder.validateRegion(pipeline.request.Region); err != nil {
		return err
	}

	// Validate target regions for copying
	for _, region := range pipeline.request.CopyToRegions {
		if err := pipeline.builder.validateRegion(region); err != nil {
			return ValidationError("invalid target region for copying", err).WithContext("region", region)
		}
	}

	fmt.Printf("✅ Request validation completed\n")
	return nil
}

// BaseAMIStep retrieves and validates the base AMI
type BaseAMIStep struct{}

func (b *BaseAMIStep) GetName() string {
	return "Retrieving base AMI"
}

func (b *BaseAMIStep) Execute(ctx context.Context, pipeline *BuildPipeline) error {
	baseAMI, err := pipeline.builder.getBaseAMI(
		pipeline.request.Template.Base,
		pipeline.request.Region,
		pipeline.request.Architecture,
	)
	if err != nil {
		return fmt.Errorf("failed to get base AMI: %w", err)
	}

	pipeline.result.SourceAMI = baseAMI
	fmt.Printf("✅ Base AMI: %s\n", baseAMI)
	return nil
}

// BuilderInstanceStep launches the builder instance
type BuilderInstanceStep struct{}

func (b *BuilderInstanceStep) GetName() string {
	return "Launching builder instance"
}

func (b *BuilderInstanceStep) Execute(ctx context.Context, pipeline *BuildPipeline) error {
	instanceID, err := pipeline.builder.launchBuilderInstance(ctx, pipeline.request)
	if err != nil {
		return fmt.Errorf("failed to launch builder instance: %w", err)
	}

	pipeline.result.BuilderID = instanceID
	fmt.Printf("✅ Builder instance: %s\n", instanceID)

	// Wait for instance to be ready
	if err := pipeline.builder.waitForInstanceReady(ctx, instanceID); err != nil {
		return fmt.Errorf("instance failed to become ready: %w", err)
	}

	fmt.Printf("✅ Instance ready for build\n")
	return nil
}

// BuildExecutionStep executes all build steps
type BuildExecutionStep struct{}

func (b *BuildExecutionStep) GetName() string {
	return "Executing build steps"
}

func (b *BuildExecutionStep) Execute(ctx context.Context, pipeline *BuildPipeline) error {
	fmt.Printf("📋 Executing %d build steps\n", len(pipeline.request.Template.BuildSteps))

	for i, step := range pipeline.request.Template.BuildSteps {
		stepStart := time.Now()
		pipeline.buildLog.WriteString(fmt.Sprintf("Step %d: %s\n", i+1, step.Name))

		fmt.Printf("  🔄 Step %d/%d: %s\n", i+1, len(pipeline.request.Template.BuildSteps), step.Name)

		output, err := pipeline.builder.executeStep(ctx, pipeline.result.BuilderID, step)
		stepDuration := time.Since(stepStart)

		if err != nil {
			fmt.Printf("  ❌ Failed: %v\n", err)
			pipeline.LogStep(step.Name, false, stepDuration, output, err)
			return fmt.Errorf("build step '%s' failed: %w", step.Name, err)
		}

		fmt.Printf("  ✅ Completed in %s\n", stepDuration)
		pipeline.LogStep(step.Name, true, stepDuration, output, nil)
	}

	return nil
}

// ValidationExecutionStep runs validation tests
type ValidationExecutionStep struct{}

func (v *ValidationExecutionStep) GetName() string {
	return "Running validation tests"
}

func (v *ValidationExecutionStep) Execute(ctx context.Context, pipeline *BuildPipeline) error {
	pipeline.buildLog.WriteString("Validation:\n")

	if len(pipeline.request.Template.Validation) == 0 {
		pipeline.buildLog.WriteString("No validation tests specified\n\n")
		fmt.Printf("⚠️  No validation tests specified\n")
		return nil
	}

	validator := NewValidator(pipeline.builder.SSMClient, ValidatorOptions{
		FailFast:    false,
		LogProgress: true,
	})

	validationResult, err := validator.ValidateAMI(pipeline.result.BuilderID, &pipeline.request.Template)
	if err != nil {
		pipeline.buildLog.WriteString(fmt.Sprintf("FAILED: %v\n", err))
		return fmt.Errorf("validation failed: %w", err)
	}

	if !validationResult.Successful {
		fmt.Printf("❌ Validation failed: %d/%d tests passed\n",
			validationResult.SuccessfulTests, validationResult.TotalTests)
		pipeline.buildLog.WriteString(validator.FormatValidationResult(validationResult))
		pipeline.result.ValidationLog = validator.FormatValidationResult(validationResult)
		return ValidationError("AMI validation failed", nil).
			WithContext("successful_tests", fmt.Sprintf("%d", validationResult.SuccessfulTests)).
			WithContext("total_tests", fmt.Sprintf("%d", validationResult.TotalTests))
	}

	fmt.Printf("✅ All validation tests passed (%d/%d)\n",
		validationResult.SuccessfulTests, validationResult.TotalTests)
	pipeline.buildLog.WriteString(fmt.Sprintf("SUCCESS: %d/%d tests passed\n\n",
		validationResult.SuccessfulTests, validationResult.TotalTests))
	pipeline.result.ValidationLog = validator.FormatValidationResult(validationResult)

	return nil
}

// AMICreationStep creates the AMI from the builder instance
type AMICreationStep struct{}

func (a *AMICreationStep) GetName() string {
	return "Creating AMI"
}

func (a *AMICreationStep) Execute(ctx context.Context, pipeline *BuildPipeline) error {
	if pipeline.request.DryRun {
		fmt.Printf("🏁 Dry run complete - skipping AMI creation\n")
		pipeline.result.Status = "dry_run_complete"
		return nil
	}

	amiID, err := pipeline.builder.createAMI(ctx, pipeline.result.BuilderID, pipeline.request)
	if err != nil {
		return fmt.Errorf("failed to create AMI: %w", err)
	}

	pipeline.result.AMIID = amiID
	fmt.Printf("✅ AMI created: %s\n", amiID)
	return nil
}

// AMICopyStep copies the AMI to target regions
type AMICopyStep struct{}

func (a *AMICopyStep) GetName() string {
	return "Copying AMI to target regions"
}

func (a *AMICopyStep) Execute(ctx context.Context, pipeline *BuildPipeline) error {
	if pipeline.request.DryRun || len(pipeline.request.CopyToRegions) == 0 {
		return nil // Skip if dry run or no target regions
	}

	fmt.Printf("🔄 Copying AMI to %d regions...\n", len(pipeline.request.CopyToRegions))

	// Get AMI details for copying
	amiDetails, err := pipeline.builder.EC2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{pipeline.result.AMIID},
	})
	if err != nil {
		return fmt.Errorf("failed to get AMI details for copying: %w", err)
	}
	if len(amiDetails.Images) == 0 {
		return fmt.Errorf("AMI %s not found for copying", pipeline.result.AMIID)
	}

	sourceAMI := amiDetails.Images[0]
	sourceRegion := pipeline.request.Region

	for _, targetRegion := range pipeline.request.CopyToRegions {
		copiedAMI, err := pipeline.builder.copyAMIToRegion(ctx,
			pipeline.result.AMIID,
			*sourceAMI.Name,
			*sourceAMI.Description,
			sourceRegion,
			targetRegion)
		if err != nil {
			return fmt.Errorf("failed to copy AMI to %s: %w", targetRegion, err)
		}

		pipeline.result.CopiedAMIs[targetRegion] = copiedAMI
		fmt.Printf("✅ Copied to %s: %s\n", targetRegion, copiedAMI)
	}

	return nil
}

// CleanupStep performs cleanup operations
type CleanupStep struct{}

func (c *CleanupStep) GetName() string {
	return "Cleaning up resources"
}

func (c *CleanupStep) Execute(ctx context.Context, pipeline *BuildPipeline) error {
	// Cleanup is handled by the defer in the main BuildAMI function
	// This step exists for completeness and future extension
	fmt.Printf("✅ Cleanup scheduled\n")
	return nil
}

// Instance Save Pipeline Steps Implementation

// InstanceDetailsStep gets instance details and sets up the result
type InstanceDetailsStep struct{}

func (i *InstanceDetailsStep) GetName() string {
	return "Getting instance details"
}

func (i *InstanceDetailsStep) Execute(ctx context.Context, pipeline *InstanceSavePipeline) error {
	p := pipeline

	instanceDetails, err := p.builder.EC2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{p.request.InstanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(instanceDetails.Reservations) == 0 || len(instanceDetails.Reservations[0].Instances) == 0 {
		return fmt.Errorf("instance %s not found", p.request.InstanceID)
	}

	p.instanceDetails = instanceDetails
	instance := instanceDetails.Reservations[0].Instances[0]
	p.result.Architecture = string(instance.Architecture)
	p.result.Region = string(p.builder.EC2Client.Options().Region)

	fmt.Printf("✅ Instance: %s | Architecture: %s | Region: %s\n",
		p.request.InstanceID, p.result.Architecture, p.result.Region)
	return nil
}

// InstanceStopStep stops the instance for consistent AMI creation
type InstanceStopStep struct{}

func (i *InstanceStopStep) GetName() string {
	return "Stopping instance for consistent AMI creation"
}

func (i *InstanceStopStep) Execute(ctx context.Context, pipeline *InstanceSavePipeline) error {
	p := pipeline

	_, err := p.builder.EC2Client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{p.request.InstanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	// Wait for instance to be stopped
	waiter := ec2.NewInstanceStoppedWaiter(p.builder.EC2Client)
	if err := waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{p.request.InstanceID},
	}, 5*time.Minute); err != nil {
		return fmt.Errorf("timeout waiting for instance to stop: %w", err)
	}

	fmt.Printf("✅ Instance stopped\n")
	return nil
}

// InstanceAMICreationStep creates the AMI from stopped instance
type InstanceAMICreationStep struct{}

func (i *InstanceAMICreationStep) GetName() string {
	return "Creating AMI from instance"
}

func (i *InstanceAMICreationStep) Execute(ctx context.Context, pipeline *InstanceSavePipeline) error {
	p := pipeline

	timestamp := time.Now().Format("20060102-150405")
	amiName := fmt.Sprintf("%s-%s-%s-%s",
		p.request.TemplateName,
		p.result.Architecture,
		p.result.Region,
		timestamp)

	// Create AMI with proper tagging
	tags := i.buildInstanceSaveTags(p.request, amiName, timestamp)

	createImageResult, err := p.builder.EC2Client.CreateImage(ctx, &ec2.CreateImageInput{
		InstanceId:        aws.String(p.request.InstanceID),
		Name:              aws.String(amiName),
		Description:       aws.String(p.request.Description),
		TagSpecifications: tags,
	})
	if err != nil {
		return fmt.Errorf("failed to create AMI: %w", err)
	}

	amiID := *createImageResult.ImageId
	p.result.AMIID = amiID
	fmt.Printf("✅ AMI creation started: %s\n", amiID)

	// Wait for AMI to be available
	fmt.Printf("⏳ Waiting for AMI to be available...\n")
	amiWaiter := ec2.NewImageAvailableWaiter(p.builder.EC2Client)
	if err := amiWaiter.Wait(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{amiID},
	}, 30*time.Minute); err != nil {
		return fmt.Errorf("timeout waiting for AMI to be available: %w", err)
	}

	fmt.Printf("✅ AMI is now available\n")
	return nil
}

func (i *InstanceAMICreationStep) buildInstanceSaveTags(request InstanceSaveRequest, amiName, timestamp string) []types.TagSpecification {
	nameKey := "Name"
	templateKey := "CloudWorkstationTemplate"
	sourceKey := "CloudWorkstationSource"
	savedFromKey := "CloudWorkstationSavedFrom"
	savedDateKey := "CloudWorkstationSavedDate"
	sourceValue := "saved-instance"

	tags := []types.Tag{
		{Key: aws.String(nameKey), Value: aws.String(amiName)},
		{Key: aws.String(templateKey), Value: aws.String(request.TemplateName)},
		{Key: aws.String(sourceKey), Value: aws.String(sourceValue)},
		{Key: aws.String(savedFromKey), Value: aws.String(request.InstanceName)},
		{Key: aws.String(savedDateKey), Value: aws.String(timestamp)},
	}

	// Add custom tags from request
	for key, value := range request.Tags {
		tags = append(tags, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	return []types.TagSpecification{
		{
			ResourceType: types.ResourceTypeImage,
			Tags:         tags,
		},
	}
}

// InstanceAMICopyStep copies the AMI to target regions
type InstanceAMICopyStep struct{}

func (i *InstanceAMICopyStep) GetName() string {
	return "Copying AMI to target regions"
}

func (i *InstanceAMICopyStep) Execute(ctx context.Context, pipeline *InstanceSavePipeline) error {
	p := pipeline

	if len(p.request.CopyToRegions) == 0 {
		return nil // Skip if no target regions
	}

	fmt.Printf("🌍 Copying AMI to additional regions...\n")

	// Get AMI details for copying
	image, err := p.builder.EC2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{p.result.AMIID},
	})
	if err != nil || len(image.Images) == 0 {
		return fmt.Errorf("unable to get AMI details for copying: %w", err)
	}

	amiDetails := image.Images[0]
	copiedAMIs, err := p.builder.copyAMIToRegions(ctx,
		p.result.AMIID,
		*amiDetails.Name,
		*amiDetails.Description,
		p.request.CopyToRegions)

	if err != nil {
		fmt.Printf("⚠️ Some AMI copies failed: %v\n", err)
	} else if len(copiedAMIs) > 0 {
		p.result.CopiedAMIs = copiedAMIs
		fmt.Printf("✅ AMI copied to %d regions\n", len(copiedAMIs))
	}

	return nil
}

// TemplateDefinitionStep creates the template definition file
type TemplateDefinitionStep struct{}

func (t *TemplateDefinitionStep) GetName() string {
	return "Creating template definition"
}

func (t *TemplateDefinitionStep) Execute(ctx context.Context, pipeline *InstanceSavePipeline) error {
	p := pipeline

	if err := p.builder.createTemplateDefinition(p.request, p.result); err != nil {
		fmt.Printf("⚠️ Warning: Failed to create template definition: %v\n", err)
	} else {
		fmt.Printf("✅ Template definition created\n")
	}
	return nil
}

// RegistryPublishStep publishes the AMI to the registry
type RegistryPublishStep struct{}

func (r *RegistryPublishStep) GetName() string {
	return "Publishing to registry"
}

func (r *RegistryPublishStep) Execute(ctx context.Context, pipeline *InstanceSavePipeline) error {
	p := pipeline

	if p.builder.RegistryClient == nil {
		return nil // Skip if no registry
	}

	if err := p.builder.RegistryClient.PublishAMI(ctx, p.result); err != nil {
		fmt.Printf("⚠️ Warning: Failed to register AMI in registry: %v\n", err)
	} else {
		fmt.Printf("✅ AMI registered in template registry\n")
	}

	// Register copied AMIs if any
	for copyRegion, copiedID := range p.result.CopiedAMIs {
		copiedResult := *p.result
		copiedResult.AMIID = copiedID
		copiedResult.Region = copyRegion

		if err := p.builder.RegistryClient.PublishAMI(ctx, &copiedResult); err != nil {
			fmt.Printf("⚠️ Failed to register copied AMI in region %s: %v\n", copyRegion, err)
		}
	}

	return nil
}
