// Package ami provides CloudWorkstation's AMI creation system.
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

// getVersionedParameterPath calculates the SSM parameter path for a versioned AMI
func (r *Registry) getVersionedParameterPath(templateName, region, architecture, version string) string {
	return fmt.Sprintf("%s/%s/%s/%s/version/%s", r.ParameterPrefix, templateName, region, architecture, version)
}

// LookupVersionedAMI finds a specific version of an AMI for a template, region, and architecture
func (r *Registry) LookupVersionedAMI(ctx context.Context, templateName, region, architecture, version string) (*Reference, error) {
	// Calculate parameter path
	paramPath := r.getVersionedParameterPath(templateName, region, architecture, version)

	// Retrieve from SSM Parameter Store
	param, err := r.SSMClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name: aws.String(paramPath),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to lookup versioned AMI in registry: %w", err)
	}

	// Deserialize JSON
	var amiRef Reference
	if err := json.Unmarshal([]byte(*param.Parameter.Value), &amiRef); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AMI reference: %w", err)
	}

	return &amiRef, nil
}

// ListVersions returns all versions available for a specific template, region, and architecture
func (r *Registry) ListVersions(ctx context.Context, templateName, region, architecture string) ([]string, error) {
	// Calculate parameter path prefix for versions
	pathPrefix := fmt.Sprintf("%s/%s/%s/%s/version/", r.ParameterPrefix, templateName, region, architecture)

	// List all parameters with this prefix
	params, err := r.SSMClient.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
		Path:           aws.String(pathPrefix),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(false),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list versions for template: %w", err)
	}

	// Extract version from path
	versions := []string{}
	for _, param := range params.Parameters {
		parts := strings.Split(*param.Name, "/")
		if len(parts) > 0 {
			version := parts[len(parts)-1]
			versions = append(versions, version)
		}
	}

	return versions, nil
}

// PublishVersionedAMI registers a versioned AMI in the registry
func (r *Registry) PublishVersionedAMI(ctx context.Context, buildResult *BuildResult) error {
	// Skip if no version is specified
	if buildResult.Version == "" {
		return nil
	}

	// Create AMI reference
	amiRef := Reference{
		AMIID:        buildResult.AMIID,
		Region:       buildResult.Region,
		Architecture: buildResult.Architecture,
		TemplateName: buildResult.TemplateName,
		Version:      buildResult.Version,
		BuildDate:    buildResult.BuildTime.UTC(),
		Tags:         make(map[string]string),
	}

	// Serialize to JSON
	amiJSON, err := json.Marshal(amiRef)
	if err != nil {
		return fmt.Errorf("failed to marshal versioned AMI reference: %w", err)
	}

	// Calculate parameter path for versioned AMI
	paramPath := r.getVersionedParameterPath(
		buildResult.TemplateName,
		buildResult.Region,
		buildResult.Architecture,
		buildResult.Version,
	)

	// Store in SSM Parameter Store
	_, err = r.SSMClient.PutParameter(ctx, &ssm.PutParameterInput{
		Name:      aws.String(paramPath),
		Type:      types.ParameterTypeString,
		Value:     aws.String(string(amiJSON)),
		Overwrite: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("failed to publish versioned AMI to SSM: %w", err)
	}

	return nil
}

// GetTemplateLatestVersion gets the latest version of a template from the registry
func (r *Registry) GetTemplateLatestVersion(ctx context.Context, templateName string) (string, error) {
	// Get shared template with latest version
	entry, err := r.GetSharedTemplate(ctx, templateName, "")
	if err != nil {
		return "", fmt.Errorf("failed to get latest template version: %w", err)
	}

	return entry.Version, nil
}

// CompareVersions compares two version strings using semantic versioning rules
func CompareVersions(version1, version2 string) (int, error) {
	// Parse versions
	v1, err := NewVersionInfo(version1)
	if err != nil {
		return 0, fmt.Errorf("invalid first version: %w", err)
	}

	v2, err := NewVersionInfo(version2)
	if err != nil {
		return 0, fmt.Errorf("invalid second version: %w", err)
	}

	// Compare
	if v1.IsGreaterThan(v2) {
		return 1, nil
	} else if v2.IsGreaterThan(v1) {
		return -1, nil
	}
	return 0, nil
}

// GetLatestAMIByVersion gets the latest AMI for a template based on semantic versioning
func (r *Registry) GetLatestAMIByVersion(ctx context.Context, templateName, region, architecture string) (*Reference, error) {
	// Get all versions
	versions, err := r.ListVersions(ctx, templateName, region, architecture)
	if err != nil {
		// If no versions found, fall back to latest
		return r.LookupAMI(ctx, templateName, region, architecture)
	}

	if len(versions) == 0 {
		// If no versions found, fall back to latest
		return r.LookupAMI(ctx, templateName, region, architecture)
	}

	// Find latest version using semantic versioning
	latestVersion := versions[0]
	for _, version := range versions[1:] {
		comp, err := CompareVersions(version, latestVersion)
		if err != nil {
			// Skip invalid versions
			continue
		}

		if comp > 0 {
			latestVersion = version
		}
	}

	// Lookup AMI with latest version
	return r.LookupVersionedAMI(ctx, templateName, region, architecture, latestVersion)
}