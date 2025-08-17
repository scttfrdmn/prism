// Package ami provides CloudWorkstation's AMI creation system.
package ami

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// TemplateRegistryEntry represents a shared template in the registry
type TemplateRegistryEntry struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Version      string            `json:"version"`
	PublishedAt  time.Time         `json:"published_at"`
	PublishedBy  string            `json:"published_by"`
	Architecture string            `json:"architecture,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
	Format       string            `json:"format"` // yaml, json, etc.
	TemplateData string            `json:"template_data"`
}

// PublishTemplate publishes a template to the registry
//
// This method serializes a template and stores it in SSM Parameter Store
// with associated metadata for discovery and sharing.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - templateName: Name of the template to share
//   - templateData: Serialized template data (YAML or JSON)
//   - format: Format of the template data (yaml, json)
//   - metadata: Additional metadata for the template
//
// Returns:
//   - error: Any sharing errors
func (r *Registry) PublishTemplate(ctx context.Context, templateName, templateData, format string, metadata map[string]string) error {
	// Create registry entry
	entry := TemplateRegistryEntry{
		Name:         templateName,
		Version:      "1.0.0", // Default version
		PublishedAt:  time.Now(),
		Format:       format,
		TemplateData: templateData,
		Tags:         make(map[string]string),
	}

	// Add metadata if provided
	if metadata != nil {
		if desc, ok := metadata["description"]; ok {
			entry.Description = desc
		}
		if version, ok := metadata["version"]; ok {
			entry.Version = version
		}
		if publisher, ok := metadata["publisher"]; ok {
			entry.PublishedBy = publisher
		}
		if arch, ok := metadata["architecture"]; ok {
			entry.Architecture = arch
		}

		// Add all other metadata as tags
		for k, v := range metadata {
			if k != "description" && k != "version" && k != "publisher" && k != "architecture" {
				entry.Tags[k] = v
			}
		}
	}

	// Serialize registry entry
	entryData, err := json.Marshal(entry)
	if err != nil {
		return RegistryError("failed to marshal template registry entry", err).
			WithContext("template_name", templateName)
	}

	// Parameter name format: /prefix/templates/template-name/latest
	paramName := fmt.Sprintf("%s/templates/%s/latest", r.ParameterPrefix, templateName)

	// Store in SSM Parameter Store
	_, err = r.SSMClient.PutParameter(ctx, &ssm.PutParameterInput{
		Name:      aws.String(paramName),
		Type:      types.ParameterTypeString,
		Value:     aws.String(string(entryData)),
		Overwrite: aws.Bool(true),
		Tags: []types.Tag{
			{
				Key:   aws.String("CloudWorkstationTemplate"),
				Value: aws.String("true"),
			},
			{
				Key:   aws.String("Name"),
				Value: aws.String(templateName),
			},
			{
				Key:   aws.String("PublishedAt"),
				Value: aws.String(time.Now().Format(time.RFC3339)),
			},
		},
	})

	if err != nil {
		return RegistryError("failed to publish template to registry", err).
			WithContext("template_name", templateName).
			WithContext("parameter_name", paramName)
	}

	// Also store versioned copy
	paramVersioned := fmt.Sprintf("%s/templates/%s/%s", r.ParameterPrefix, templateName, entry.Version)
	_, err = r.SSMClient.PutParameter(ctx, &ssm.PutParameterInput{
		Name:  aws.String(paramVersioned),
		Type:  types.ParameterTypeString,
		Value: aws.String(string(entryData)),
		Tags: []types.Tag{
			{
				Key:   aws.String("CloudWorkstationTemplate"),
				Value: aws.String("true"),
			},
			{
				Key:   aws.String("Name"),
				Value: aws.String(templateName),
			},
			{
				Key:   aws.String("Version"),
				Value: aws.String(entry.Version),
			},
		},
	})

	if err != nil {
		return RegistryError("failed to publish versioned template to registry", err).
			WithContext("template_name", templateName).
			WithContext("parameter_name", paramVersioned)
	}

	return nil
}

// ListSharedTemplates lists templates available in the registry
//
// This method queries SSM Parameter Store for templates shared in the registry.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//
// Returns:
//   - map[string]*TemplateRegistryEntry: Map of template names to registry entries
//   - error: Any listing errors
func (r *Registry) ListSharedTemplates(ctx context.Context) (map[string]*TemplateRegistryEntry, error) {
	result := make(map[string]*TemplateRegistryEntry)

	// Parameter path for templates: /prefix/templates/
	path := fmt.Sprintf("%s/templates/", r.ParameterPrefix)

	// Get parameters by path
	input := &ssm.GetParametersByPathInput{
		Path:           aws.String(path),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(false),
	}

	paginator := ssm.NewGetParametersByPathPaginator(r.SSMClient, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return result, RegistryError("failed to list shared templates", err).
				WithContext("parameter_path", path)
		}

		for _, param := range page.Parameters {
			// Only process latest versions
			if strings.Contains(*param.Name, "/latest") {
				// Extract template name from path
				namePath := strings.TrimPrefix(*param.Name, path)
				templateName := strings.Split(namePath, "/")[0]

				// Parse registry entry
				var entry TemplateRegistryEntry
				if err := json.Unmarshal([]byte(*param.Value), &entry); err != nil {
					fmt.Printf("Warning: Failed to parse registry entry for %s: %v\n", templateName, err)
					continue
				}

				// Add to result
				result[templateName] = &entry
			}
		}
	}

	return result, nil
}

// GetSharedTemplate gets a template from the registry
//
// This method retrieves a template from SSM Parameter Store.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - templateName: Name of the template to retrieve
//   - version: Optional version to retrieve ("" for latest)
//
// Returns:
//   - *TemplateRegistryEntry: The template registry entry
//   - error: Any retrieval errors
func (r *Registry) GetSharedTemplate(ctx context.Context, templateName, version string) (*TemplateRegistryEntry, error) {
	// Parameter name format: /prefix/templates/template-name/latest or /prefix/templates/template-name/version
	paramSuffix := "latest"
	if version != "" {
		paramSuffix = version
	}

	paramName := fmt.Sprintf("%s/templates/%s/%s", r.ParameterPrefix, templateName, paramSuffix)

	// Get parameter
	param, err := r.SSMClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(paramName),
		WithDecryption: aws.Bool(false),
	})

	if err != nil {
		return nil, RegistryError(fmt.Sprintf("template '%s' not found in registry", templateName), err).
			WithContext("template_name", templateName).
			WithContext("parameter_name", paramName)
	}

	// Parse registry entry
	var entry TemplateRegistryEntry
	if err := json.Unmarshal([]byte(*param.Parameter.Value), &entry); err != nil {
		return nil, RegistryError("failed to parse registry entry", err).
			WithContext("template_name", templateName).
			WithContext("parameter_name", paramName)
	}

	return &entry, nil
}

// ListSharedTemplateVersions lists all versions of a template in the registry
//
// This method queries SSM Parameter Store for all versions of a specific template.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - templateName: Name of the template to list versions for
//
// Returns:
//   - []string: List of version strings (e.g., "1.0.0", "1.1.0")
//   - error: Any listing errors
func (r *Registry) ListSharedTemplateVersions(ctx context.Context, templateName string) ([]string, error) {
	result := []string{}

	// Parameter path for template versions: /prefix/templates/template-name/
	paramPath := fmt.Sprintf("%s/templates/%s/", r.ParameterPrefix, templateName)

	// Get parameters by path
	input := &ssm.GetParametersByPathInput{
		Path:      aws.String(paramPath),
		Recursive: aws.Bool(true),
	}

	paginator := ssm.NewGetParametersByPathPaginator(r.SSMClient, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return result, RegistryError("failed to list template versions", err).
				WithContext("template_name", templateName).
				WithContext("parameter_path", paramPath)
		}

		for _, param := range page.Parameters {
			// Extract version from path
			pathParts := strings.Split(*param.Name, "/")
			version := pathParts[len(pathParts)-1]

			// Skip "latest" as it's not a real version
			if version != "latest" {
				result = append(result, version)
			}
		}
	}

	// Sort versions (TODO: use semantic versioning for proper sorting)

	return result, nil
}

// DeleteSharedTemplate deletes a template from the registry
//
// This method removes a template from SSM Parameter Store.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - templateName: Name of the template to delete
//   - version: Optional version to delete ("" for all versions)
//
// Returns:
//   - error: Any deletion errors
func (r *Registry) DeleteSharedTemplate(ctx context.Context, templateName, version string) error {
	// Parameter path for template: /prefix/templates/template-name
	paramPath := fmt.Sprintf("%s/templates/%s", r.ParameterPrefix, templateName)

	if version == "" {
		// Delete all versions - list parameters by path
		input := &ssm.GetParametersByPathInput{
			Path:      aws.String(paramPath),
			Recursive: aws.Bool(true),
		}

		paginator := ssm.NewGetParametersByPathPaginator(r.SSMClient, input)

		var paramNames []string
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return RegistryError("failed to list template parameters", err).
					WithContext("template_name", templateName).
					WithContext("parameter_path", paramPath)
			}

			for _, param := range page.Parameters {
				paramNames = append(paramNames, *param.Name)
			}
		}

		// Delete all parameters
		for _, name := range paramNames {
			_, err := r.SSMClient.DeleteParameter(ctx, &ssm.DeleteParameterInput{
				Name: aws.String(name),
			})
			if err != nil {
				// Log error but continue with other parameters
				fmt.Printf("Warning: Failed to delete parameter %s: %v\n", name, err)
			}
		}
	} else {
		// Delete specific version
		paramName := fmt.Sprintf("%s/%s", paramPath, version)
		_, err := r.SSMClient.DeleteParameter(ctx, &ssm.DeleteParameterInput{
			Name: aws.String(paramName),
		})
		if err != nil {
			return RegistryError(fmt.Sprintf("failed to delete template version '%s'", version), err).
				WithContext("template_name", templateName).
				WithContext("version", version).
				WithContext("parameter_name", paramName)
		}
	}

	return nil
}
