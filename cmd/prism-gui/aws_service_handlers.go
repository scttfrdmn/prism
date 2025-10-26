package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/scttfrdmn/prism/pkg/profile"
)

// AWS service constants
const (
	awsServiceBraket     = "braket"
	awsServiceSageMaker  = "sagemaker"
	awsServiceConsole    = "console"
	awsServiceCloudShell = "cloudshell"
	awsEmbeddingMode     = "iframe"
)

// AWS service connection handlers for embedded access

// OpenAWSService provides generic AWS service access
func (s *PrismService) OpenAWSService(ctx context.Context, service string, region string) (*ConnectionConfig, error) {
	// Generate federated token for AWS service access
	token, err := s.generateServiceToken(ctx, service, region)
	if err != nil {
		return nil, fmt.Errorf("failed to generate service token: %w", err)
	}

	// Build service-specific configuration
	config := &ConnectionConfig{
		ID:            fmt.Sprintf("aws-%s-%s-%d", service, region, time.Now().Unix()),
		Type:          ConnectionTypeAWS,
		AWSService:    service,
		Region:        region,
		ProxyURL:      s.buildAWSServiceURL(service, region, token),
		AuthToken:     token,
		EmbeddingMode: s.getServiceEmbeddingMode(service),
		Title:         s.buildServiceTitle(service, region),
		Status:        "connecting",
		Metadata: map[string]interface{}{
			"service_type": "aws",
			"launch_time":  time.Now().Format(time.RFC3339),
		},
	}

	return config, nil
}

// OpenBraketConsole opens Amazon Braket quantum computing console
func (s *PrismService) OpenBraketConsole(ctx context.Context, region string) (*ConnectionConfig, error) {
	config, err := s.OpenAWSService(ctx, awsServiceBraket, region)
	if err != nil {
		return nil, err
	}

	// Braket-specific metadata
	config.Metadata["quantum_devices"] = []string{"sv1", "ionq", "rigetti"}
	config.Metadata["service_description"] = "Amazon Braket quantum computing platform"
	config.Title = fmt.Sprintf("⚛️ Braket (%s)", region)

	return config, nil
}

// OpenSageMakerStudio opens SageMaker Studio for ML development
func (s *PrismService) OpenSageMakerStudio(ctx context.Context, region string) (*ConnectionConfig, error) {
	config, err := s.OpenAWSService(ctx, awsServiceSageMaker, region)
	if err != nil {
		return nil, err
	}

	// SageMaker-specific metadata
	config.Metadata["service_description"] = "SageMaker Studio ML development environment"
	config.Metadata["notebook_support"] = true
	config.Title = fmt.Sprintf("🤖 SageMaker (%s)", region)

	return config, nil
}

// OpenAWSConsole opens AWS Management Console
func (s *PrismService) OpenAWSConsole(ctx context.Context, service string, region string) (*ConnectionConfig, error) {
	config, err := s.OpenAWSService(ctx, awsServiceConsole, region)
	if err != nil {
		return nil, err
	}

	// Console-specific metadata
	config.Metadata["console_service"] = service
	config.Metadata["service_description"] = "AWS Management Console"
	config.Title = fmt.Sprintf("🎛️ Console (%s)", region)

	return config, nil
}

// OpenCloudShell opens AWS CloudShell terminal
func (s *PrismService) OpenCloudShell(ctx context.Context, region string) (*ConnectionConfig, error) {
	config, err := s.OpenAWSService(ctx, awsServiceCloudShell, region)
	if err != nil {
		return nil, err
	}

	// CloudShell-specific metadata
	config.Metadata["terminal_type"] = awsServiceCloudShell
	config.Metadata["persistent_storage"] = true
	config.Metadata["service_description"] = "AWS CloudShell browser-based terminal"
	config.Title = fmt.Sprintf("🖥️ CloudShell (%s)", region)

	return config, nil
}

// Helper functions for AWS service integration

// generateServiceToken creates a federated token for AWS service access
func (s *PrismService) generateServiceToken(ctx context.Context, service string, region string) (string, error) {
	// Use the existing AWS configuration from Prism
	cfg, err := s.getAWSConfig(ctx, region)
	if err != nil {
		return "", fmt.Errorf("failed to get AWS config: %w", err)
	}

	// Create STS client for token generation
	stsClient := sts.NewFromConfig(cfg)

	// Generate federation token with appropriate permissions
	input := &sts.GetFederationTokenInput{
		Name:            aws.String(fmt.Sprintf("Prism-%s", service)),
		DurationSeconds: aws.Int32(3600), // 1 hour token
		Policy:          aws.String(s.getServicePolicy(service)),
	}

	result, err := stsClient.GetFederationToken(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to generate federation token: %w", err)
	}

	// Create AWS federation token for console access
	sessionJSON := map[string]string{
		"sessionId":    *result.Credentials.AccessKeyId,
		"sessionKey":   *result.Credentials.SecretAccessKey,
		"sessionToken": *result.Credentials.SessionToken,
	}

	sessionData, err := json.Marshal(sessionJSON)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session data: %w", err)
	}

	return base64.StdEncoding.EncodeToString(sessionData), nil
}

// buildAWSServiceURL constructs the appropriate URL for AWS service access
func (s *PrismService) buildAWSServiceURL(service string, region string, token string) string {
	switch service {
	case awsServiceBraket:
		return fmt.Sprintf("%s/aws-proxy/braket?region=%s&token=%s", s.daemonURL, region, url.QueryEscape(token))
	case awsServiceSageMaker:
		return fmt.Sprintf("%s/aws-proxy/sagemaker?region=%s&token=%s", s.daemonURL, region, url.QueryEscape(token))
	case awsServiceConsole:
		return fmt.Sprintf("%s/aws-proxy/console?region=%s&token=%s", s.daemonURL, region, url.QueryEscape(token))
	case awsServiceCloudShell:
		return fmt.Sprintf("%s/aws-proxy/cloudshell?region=%s&token=%s", s.daemonURL, region, url.QueryEscape(token))
	default:
		return fmt.Sprintf("%s/aws-proxy/%s?region=%s&token=%s", s.daemonURL, service, region, url.QueryEscape(token))
	}
}

// getServiceEmbeddingMode determines the best embedding approach for each service
//
//nolint:unparam // Future extensibility for different embedding modes
func (s *PrismService) getServiceEmbeddingMode(service string) string {
	switch service {
	case awsServiceBraket:
		return awsEmbeddingMode // Braket console works well in iframe
	case awsServiceSageMaker:
		return awsEmbeddingMode // SageMaker Studio supports iframe embedding
	case awsServiceConsole:
		return awsEmbeddingMode // AWS Console can be embedded with proper auth
	case awsServiceCloudShell:
		return awsEmbeddingMode // CloudShell has iframe support
	default:
		return awsEmbeddingMode
	}
}

// buildServiceTitle creates a human-readable title for the service tab
func (s *PrismService) buildServiceTitle(service string, region string) string {
	serviceNames := map[string]string{
		awsServiceBraket:     "Amazon Braket",
		awsServiceSageMaker:  "SageMaker Studio",
		awsServiceConsole:    "AWS Console",
		awsServiceCloudShell: "CloudShell",
	}

	if name, exists := serviceNames[service]; exists {
		return fmt.Sprintf("%s (%s)", name, region)
	}
	return fmt.Sprintf("%s (%s)", service, region)
}

// getServicePolicy returns IAM policy for specific AWS services
func (s *PrismService) getServicePolicy(service string) string {
	policies := map[string]string{
		awsServiceBraket: `{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Action": [
						"braket:*",
						"s3:GetObject",
						"s3:PutObject",
						"s3:ListBucket"
					],
					"Resource": "*"
				}
			]
		}`,
		awsServiceSageMaker: `{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Action": [
						"sagemaker:*",
						"s3:GetObject",
						"s3:PutObject",
						"s3:ListBucket",
						"iam:PassRole"
					],
					"Resource": "*"
				}
			]
		}`,
		awsServiceConsole: `{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Action": [
						"ec2:Describe*",
						"s3:List*",
						"s3:Get*",
						"iam:ListRoles",
						"cloudwatch:GetMetricData"
					],
					"Resource": "*"
				}
			]
		}`,
		awsServiceCloudShell: `{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Action": [
						"cloudshell:*",
						"ec2:Describe*",
						"s3:List*"
					],
					"Resource": "*"
				}
			]
		}`,
	}

	if policy, exists := policies[service]; exists {
		return policy
	}

	// Default minimal policy
	return `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Action": [
					"ec2:Describe*"
				],
				"Resource": "*"
			}
		]
	}`
}

// getAWSConfig returns AWS configuration for the specified region
// Uses the Prism profile system for consistent credential management
//
//nolint:unparam // Error return reserved for future auth validation
func (s *PrismService) getAWSConfig(ctx context.Context, region string) (aws.Config, error) {
	// Get current profile from the Prism profile system
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		// Fallback to default config if profile manager unavailable
		return config.LoadDefaultConfig(ctx, config.WithRegion(region))
	}

	currentProfile, err := profileManager.GetCurrentProfile()
	if err != nil {
		// Fallback to default config if no current profile
		return config.LoadDefaultConfig(ctx, config.WithRegion(region))
	}

	// Load AWS config using the profile's AWS profile name and region
	cfgOpts := []func(*config.LoadOptions) error{}

	// Use profile's AWS profile name if specified
	if currentProfile.AWSProfile != "" {
		cfgOpts = append(cfgOpts, config.WithSharedConfigProfile(currentProfile.AWSProfile))
	}

	// Prefer specified region parameter, fallback to profile's region
	if region != "" {
		cfgOpts = append(cfgOpts, config.WithRegion(region))
	} else if currentProfile.Region != "" {
		cfgOpts = append(cfgOpts, config.WithRegion(currentProfile.Region))
	}

	return config.LoadDefaultConfig(ctx, cfgOpts...)
}
