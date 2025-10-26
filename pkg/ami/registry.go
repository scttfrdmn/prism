// Package ami provides Prism's AMI creation system.
package ami

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// NewRegistry creates a new AMI registry
func NewRegistry(ssmClient *ssm.Client, paramPrefix string) *Registry {
	if paramPrefix == "" {
		paramPrefix = "/cloudworkstation/amis"
	}
	return &Registry{
		SSMClient:       ssmClient,
		ParameterPrefix: paramPrefix,
	}
}

// PublishAMI registers a new AMI in the registry
func (r *Registry) PublishAMI(ctx context.Context, buildResult *BuildResult) error {
	// Create AMI reference
	amiRef := Reference{
		AMIID:        buildResult.AMIID,
		Region:       buildResult.Region,
		Architecture: buildResult.Architecture,
		TemplateName: buildResult.TemplateName,
		Version:      buildResult.Version,
		BuildDate:    buildResult.BuildTime,
		Tags:         make(map[string]string),
	}

	// Serialize to JSON
	amiJSON, err := json.Marshal(amiRef)
	if err != nil {
		return fmt.Errorf("failed to marshal AMI reference: %w", err)
	}

	// Calculate parameter path
	paramPath := r.getParameterPath(buildResult.TemplateName, buildResult.Region, buildResult.Architecture)

	// Store in SSM Parameter Store
	_, err = r.SSMClient.PutParameter(ctx, &ssm.PutParameterInput{
		Name:      aws.String(paramPath),
		Type:      types.ParameterTypeString,
		Value:     aws.String(string(amiJSON)),
		Overwrite: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("failed to publish AMI to SSM: %w", err)
	}

	return nil
}

// LookupAMI finds the latest AMI for a template, region, and architecture
func (r *Registry) LookupAMI(ctx context.Context, templateName, region, architecture string) (*Reference, error) {
	// Calculate parameter path
	paramPath := r.getParameterPath(templateName, region, architecture)

	// Retrieve from SSM Parameter Store
	param, err := r.SSMClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name: aws.String(paramPath),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to lookup AMI in registry: %w", err)
	}

	// Deserialize JSON
	var amiRef Reference
	if err := json.Unmarshal([]byte(*param.Parameter.Value), &amiRef); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AMI reference: %w", err)
	}

	return &amiRef, nil
}

// ListTemplateAMIs returns all AMIs for a specific template
func (r *Registry) ListTemplateAMIs(ctx context.Context, templateName string) ([]*Reference, error) {
	// Calculate parameter path prefix
	pathPrefix := fmt.Sprintf("%s/%s/", r.ParameterPrefix, templateName)

	// List all parameters with this prefix
	params, err := r.SSMClient.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
		Path:           aws.String(pathPrefix),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(false),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list AMIs for template: %w", err)
	}

	// Parse each parameter
	var results []*Reference
	for _, param := range params.Parameters {
		var amiRef Reference
		if err := json.Unmarshal([]byte(*param.Value), &amiRef); err != nil {
			return nil, fmt.Errorf("failed to unmarshal AMI reference: %w", err)
		}
		results = append(results, &amiRef)
	}

	return results, nil
}

// ListTemplates returns all templates that have AMIs in the registry
func (r *Registry) ListTemplates(ctx context.Context) ([]string, error) {
	// List parameters at top level
	params, err := r.SSMClient.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
		Path:           aws.String(r.ParameterPrefix),
		Recursive:      aws.Bool(false),
		WithDecryption: aws.Bool(false),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	// Extract template names from paths
	templates := make(map[string]bool)
	for _, param := range params.Parameters {
		parts := strings.Split(*param.Name, "/")
		if len(parts) > 0 {
			templateName := parts[len(parts)-1]
			templates[templateName] = true
		}
	}

	// Convert map keys to slice
	var result []string
	for template := range templates {
		result = append(result, template)
	}

	return result, nil
}

// DeleteAMI removes an AMI from the registry
func (r *Registry) DeleteAMI(ctx context.Context, templateName, region, architecture string) error {
	// Calculate parameter path
	paramPath := r.getParameterPath(templateName, region, architecture)

	// Delete from SSM Parameter Store
	_, err := r.SSMClient.DeleteParameter(ctx, &ssm.DeleteParameterInput{
		Name: aws.String(paramPath),
	})
	if err != nil {
		return fmt.Errorf("failed to delete AMI from registry: %w", err)
	}

	return nil
}

// getParameterPath calculates the SSM parameter path for an AMI
func (r *Registry) getParameterPath(templateName, region, architecture string) string {
	return fmt.Sprintf("%s/%s/%s/%s", r.ParameterPrefix, templateName, region, architecture)
}
