package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// AWS service connection handlers for embedded access

// OpenAWSService provides generic AWS service access
func (s *CloudWorkstationService) OpenAWSService(ctx context.Context, service string, region string) (*ConnectionConfig, error) {
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
func (s *CloudWorkstationService) OpenBraketConsole(ctx context.Context, region string) (*ConnectionConfig, error) {
	config, err := s.OpenAWSService(ctx, "braket", region)
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
func (s *CloudWorkstationService) OpenSageMakerStudio(ctx context.Context, region string) (*ConnectionConfig, error) {
	config, err := s.OpenAWSService(ctx, "sagemaker", region)
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
func (s *CloudWorkstationService) OpenAWSConsole(ctx context.Context, service string, region string) (*ConnectionConfig, error) {
	config, err := s.OpenAWSService(ctx, "console", region)
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
func (s *CloudWorkstationService) OpenCloudShell(ctx context.Context, region string) (*ConnectionConfig, error) {
	config, err := s.OpenAWSService(ctx, "cloudshell", region)
	if err != nil {
		return nil, err
	}

	// CloudShell-specific metadata
	config.Metadata["terminal_type"] = "cloudshell"
	config.Metadata["persistent_storage"] = true
	config.Metadata["service_description"] = "AWS CloudShell browser-based terminal"
	config.Title = fmt.Sprintf("🖥️ CloudShell (%s)", region)

	return config, nil
}

// Helper functions for AWS service integration

// generateServiceToken creates a federated token for AWS service access
func (s *CloudWorkstationService) generateServiceToken(ctx context.Context, service string, region string) (string, error) {
	// Use the existing AWS configuration from CloudWorkstation
	cfg, err := s.getAWSConfig(ctx, region)
	if err != nil {
		return "", fmt.Errorf("failed to get AWS config: %w", err)
	}

	// Create STS client for token generation
	stsClient := sts.NewFromConfig(cfg)

	// Generate federation token with appropriate permissions
	input := &sts.GetFederationTokenInput{
		Name:            aws.String(fmt.Sprintf("CloudWorkstation-%s", service)),
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
func (s *CloudWorkstationService) buildAWSServiceURL(service string, region string, token string) string {
	switch service {
	case "braket":
		return fmt.Sprintf("%s/aws-proxy/braket?region=%s&token=%s", s.daemonURL, region, url.QueryEscape(token))
	case "sagemaker":
		return fmt.Sprintf("%s/aws-proxy/sagemaker?region=%s&token=%s", s.daemonURL, region, url.QueryEscape(token))
	case "console":
		return fmt.Sprintf("%s/aws-proxy/console?region=%s&token=%s", s.daemonURL, region, url.QueryEscape(token))
	case "cloudshell":
		return fmt.Sprintf("%s/aws-proxy/cloudshell?region=%s&token=%s", s.daemonURL, region, url.QueryEscape(token))
	default:
		return fmt.Sprintf("%s/aws-proxy/%s?region=%s&token=%s", s.daemonURL, service, region, url.QueryEscape(token))
	}
}

// getServiceEmbeddingMode determines the best embedding approach for each service
func (s *CloudWorkstationService) getServiceEmbeddingMode(service string) string {
	switch service {
	case "braket":
		return "iframe" // Braket console works well in iframe
	case "sagemaker":
		return "iframe" // SageMaker Studio supports iframe embedding
	case "console":
		return "iframe" // AWS Console can be embedded with proper auth
	case "cloudshell":
		return "iframe" // CloudShell has iframe support
	default:
		return "iframe"
	}
}

// buildServiceTitle creates a human-readable title for the service tab
func (s *CloudWorkstationService) buildServiceTitle(service string, region string) string {
	serviceNames := map[string]string{
		"braket":     "Amazon Braket",
		"sagemaker":  "SageMaker Studio",
		"console":    "AWS Console",
		"cloudshell": "CloudShell",
	}

	if name, exists := serviceNames[service]; exists {
		return fmt.Sprintf("%s (%s)", name, region)
	}
	return fmt.Sprintf("%s (%s)", service, region)
}

// getServicePolicy returns IAM policy for specific AWS services
func (s *CloudWorkstationService) getServicePolicy(service string) string {
	policies := map[string]string{
		"braket": `{
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
		"sagemaker": `{
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
		"console": `{
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
		"cloudshell": `{
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
func (s *CloudWorkstationService) getAWSConfig(ctx context.Context, region string) (aws.Config, error) {
	// This should use the same AWS configuration logic as the rest of CloudWorkstation
	// For now, return a basic configuration - this will need to be integrated
	// with the existing AWS profile and credential management system

	return aws.Config{
		Region: region,
		// Additional AWS configuration will be added based on existing CloudWorkstation setup
	}, nil
}
