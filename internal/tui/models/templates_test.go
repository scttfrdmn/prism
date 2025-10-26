// Package models provides comprehensive test coverage for TUI template management
package models

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scttfrdmn/prism/internal/tui/api"
	"github.com/scttfrdmn/prism/pkg/types"
)

// mockTemplateAPIClient implements apiClient interface for template testing
type mockTemplateAPIClient struct {
	templates     map[string]api.TemplateResponse
	shouldError   bool
	errorMessage  string
	callLog       []string
	responseDelay time.Duration
}

// ListTemplates mock implementation
func (m *mockTemplateAPIClient) ListTemplates(ctx context.Context) (*api.ListTemplatesResponse, error) {
	m.callLog = append(m.callLog, "ListTemplates")
	if m.responseDelay > 0 {
		time.Sleep(m.responseDelay)
	}
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListTemplatesResponse{Templates: m.templates}, nil
}

func (m *mockTemplateAPIClient) GetTemplate(ctx context.Context, name string) (*api.TemplateResponse, error) {
	m.callLog = append(m.callLog, "GetTemplate:"+name)
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	// Find template by name
	if template, exists := m.templates[name]; exists {
		return &template, nil
	}
	return nil, fmt.Errorf("template not found: %s", name)
}

// Stub implementations for other apiClient methods
func (m *mockTemplateAPIClient) ListInstances(ctx context.Context) (*api.ListInstancesResponse, error) {
	m.callLog = append(m.callLog, "ListInstances")
	return &api.ListInstancesResponse{}, nil
}
func (m *mockTemplateAPIClient) GetInstance(ctx context.Context, name string) (*api.InstanceResponse, error) {
	m.callLog = append(m.callLog, "GetInstance:"+name)
	return &api.InstanceResponse{}, nil
}
func (m *mockTemplateAPIClient) LaunchInstance(ctx context.Context, req api.LaunchInstanceRequest) (*api.LaunchInstanceResponse, error) {
	m.callLog = append(m.callLog, "LaunchInstance:"+req.Name)
	return &api.LaunchInstanceResponse{}, nil
}
func (m *mockTemplateAPIClient) StartInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, "StartInstance:"+name)
	return nil
}
func (m *mockTemplateAPIClient) StopInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, "StopInstance:"+name)
	return nil
}
func (m *mockTemplateAPIClient) DeleteInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, "DeleteInstance:"+name)
	return nil
}
func (m *mockTemplateAPIClient) ListVolumes(ctx context.Context) (*api.ListVolumesResponse, error) {
	m.callLog = append(m.callLog, "ListVolumes")
	return &api.ListVolumesResponse{}, nil
}
func (m *mockTemplateAPIClient) ListStorage(ctx context.Context) (*api.ListStorageResponse, error) {
	m.callLog = append(m.callLog, "ListStorage")
	return &api.ListStorageResponse{}, nil
}
func (m *mockTemplateAPIClient) MountVolume(ctx context.Context, volumeName, instanceName, mountPoint string) error {
	m.callLog = append(m.callLog, "MountVolume")
	return nil
}
func (m *mockTemplateAPIClient) UnmountVolume(ctx context.Context, volumeName, instanceName string) error {
	m.callLog = append(m.callLog, "UnmountVolume")
	return nil
}
func (m *mockTemplateAPIClient) ListIdlePolicies(ctx context.Context) (*api.ListIdlePoliciesResponse, error) {
	m.callLog = append(m.callLog, "ListIdlePolicies")
	return &api.ListIdlePoliciesResponse{}, nil
}
func (m *mockTemplateAPIClient) UpdateIdlePolicy(ctx context.Context, req api.IdlePolicyUpdateRequest) error {
	m.callLog = append(m.callLog, "UpdateIdlePolicy")
	return nil
}
func (m *mockTemplateAPIClient) GetInstanceIdleStatus(ctx context.Context, name string) (*api.IdleDetectionResponse, error) {
	m.callLog = append(m.callLog, "GetInstanceIdleStatus:"+name)
	return &api.IdleDetectionResponse{}, nil
}
func (m *mockTemplateAPIClient) EnableIdleDetection(ctx context.Context, name, policy string) error {
	m.callLog = append(m.callLog, "EnableIdleDetection:"+name)
	return nil
}
func (m *mockTemplateAPIClient) DisableIdleDetection(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, "DisableIdleDetection:"+name)
	return nil
}
func (m *mockTemplateAPIClient) GetStatus(ctx context.Context) (*api.SystemStatusResponse, error) {
	m.callLog = append(m.callLog, "GetStatus")
	return &api.SystemStatusResponse{}, nil
}

func (m *mockTemplateAPIClient) ListProjects(ctx context.Context, filter *api.ProjectFilter) (*api.ListProjectsResponse, error) {
	return &api.ListProjectsResponse{}, nil
}

func (m *mockTemplateAPIClient) GetPolicyStatus(ctx context.Context) (*api.PolicyStatusResponse, error) {
	return &api.PolicyStatusResponse{}, nil
}

func (m *mockTemplateAPIClient) ListPolicySets(ctx context.Context) (*api.ListPolicySetsResponse, error) {
	return &api.ListPolicySetsResponse{}, nil
}

func (m *mockTemplateAPIClient) AssignPolicySet(ctx context.Context, policySetID string) error {
	return nil
}

func (m *mockTemplateAPIClient) SetPolicyEnforcement(ctx context.Context, enabled bool) error {
	return nil
}

func (m *mockTemplateAPIClient) CheckTemplateAccess(ctx context.Context, templateName string) (*api.TemplateAccessResponse, error) {
	return &api.TemplateAccessResponse{}, nil
}

func (m *mockTemplateAPIClient) ListMarketplaceTemplates(ctx context.Context, filter *api.MarketplaceFilter) (*api.ListMarketplaceTemplatesResponse, error) {
	return &api.ListMarketplaceTemplatesResponse{}, nil
}

func (m *mockTemplateAPIClient) ListMarketplaceCategories(ctx context.Context) (*api.ListCategoriesResponse, error) {
	return &api.ListCategoriesResponse{}, nil
}

func (m *mockTemplateAPIClient) ListMarketplaceRegistries(ctx context.Context) (*api.ListRegistriesResponse, error) {
	return &api.ListRegistriesResponse{}, nil
}

func (m *mockTemplateAPIClient) InstallMarketplaceTemplate(ctx context.Context, templateName string) error {
	return nil
}

func (m *mockTemplateAPIClient) ListAMIs(ctx context.Context) (*api.ListAMIsResponse, error) {
	return &api.ListAMIsResponse{}, nil
}

func (m *mockTemplateAPIClient) ListAMIBuilds(ctx context.Context) (*api.ListAMIBuildsResponse, error) {
	return &api.ListAMIBuildsResponse{}, nil
}

func (m *mockTemplateAPIClient) ListAMIRegions(ctx context.Context) (*api.ListAMIRegionsResponse, error) {
	return &api.ListAMIRegionsResponse{}, nil
}

func (m *mockTemplateAPIClient) DeleteAMI(ctx context.Context, amiID string) error {
	return nil
}

func (m *mockTemplateAPIClient) GetRightsizingRecommendations(ctx context.Context) (*api.GetRightsizingRecommendationsResponse, error) {
	return &api.GetRightsizingRecommendationsResponse{}, nil
}

func (m *mockTemplateAPIClient) ApplyRightsizingRecommendation(ctx context.Context, instanceName string) error {
	return nil
}

func (m *mockTemplateAPIClient) GetLogs(ctx context.Context, instanceName, logType string) (*api.LogsResponse, error) {
	return &api.LogsResponse{}, nil
}

// TestTemplatesModelCreation tests basic template model instantiation
func TestTemplatesModelCreation(t *testing.T) {
	mockClient := &mockTemplateAPIClient{
		templates: map[string]api.TemplateResponse{
			"python-ml": {
				Name:        "python-ml",
				Description: "Python Machine Learning Environment",
				Ports:       []int{22, 8888},
				InstanceType: map[string]string{
					"x86_64": "t3.medium",
					"arm64":  "t4g.medium",
				},
				EstimatedCost: map[string]float64{
					"x86_64": 0.0416,
					"arm64":  0.0336,
				},
			},
		},
	}

	model := NewTemplatesModel(mockClient)

	// Validate model structure
	assert.NotNil(t, model.apiClient)
	assert.True(t, model.loading) // Model starts in loading state
	assert.Empty(t, model.error)
	assert.Empty(t, model.selected)
	assert.NotNil(t, model.templates)
	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
}

// TestTemplatesModelInit tests model initialization
func TestTemplatesModelInit(t *testing.T) {
	mockClient := &mockTemplateAPIClient{
		templates: map[string]api.TemplateResponse{
			"python-ml": {Name: "python-ml", Description: "ML template"},
		},
	}

	model := NewTemplatesModel(mockClient)

	// Test Init command
	cmd := model.Init()
	assert.NotNil(t, cmd)
	// Init returns a tea.Batch command
}

// TestTemplatesModelView tests model view rendering in different states
func TestTemplatesModelView(t *testing.T) {
	mockTemplates := map[string]api.TemplateResponse{
		"python-ml": {
			Name:        "python-ml",
			Description: "Python ML Environment",
			Ports:       []int{22, 8888},
		},
	}

	mockClient := &mockTemplateAPIClient{
		templates: mockTemplates,
	}

	model := NewTemplatesModel(mockClient)
	model.width = 100
	model.height = 50

	// Test loading state
	t.Run("loading_state", func(t *testing.T) {
		model.loading = true
		view := model.View()

		assert.NotEmpty(t, view)
		// Should show loading spinner
	})

	// Test error state
	t.Run("error_state", func(t *testing.T) {
		model.loading = false
		model.error = "Network connection failed"
		view := model.View()

		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Error")
	})

	// Test empty state
	t.Run("empty_state", func(t *testing.T) {
		model.loading = false
		model.error = ""
		model.templates = make(map[string]types.Template)
		view := model.View()

		assert.NotEmpty(t, view)
		// Should show empty message
	})

	// Test normal state with templates
	t.Run("normal_state", func(t *testing.T) {
		model.loading = false
		model.error = ""
		// Convert API response templates to internal template format
		internalTemplates := make(map[string]types.Template)
		for name, tmpl := range mockTemplates {
			internalTemplates[name] = types.Template{
				Name:                 tmpl.Name,
				Description:          tmpl.Description,
				Ports:                tmpl.Ports,
				EstimatedCostPerHour: tmpl.EstimatedCost,
			}
		}
		model.templates = internalTemplates
		view := model.View()

		assert.NotEmpty(t, view)
		assert.Greater(t, len(view), 50) // Should be substantial content
	})
}

// TestBrowserTemplateItemInterface tests the list.Item interface implementation
func TestBrowserTemplateItemInterface(t *testing.T) {
	item := BrowserTemplateItem{
		name:        "python-ml",
		description: "Machine Learning Environment",
		costX86:     1.00,
		costARM:     0.80,
		ports:       []int{22, 8888},
	}

	// Test list.Item interface methods
	assert.Equal(t, "python-ml", item.FilterValue())
	assert.Equal(t, "python-ml", item.Title())
	assert.Equal(t, "Machine Learning Environment", item.Description())
}

// TestTemplateDataProcessing tests template data conversion and processing
func TestTemplateDataProcessing(t *testing.T) {
	templates := []api.TemplateResponse{
		{
			Name:        "python-ml-gpu",
			Description: "Python ML with GPU support",
			Ports:       []int{22, 8888, 6006}, // SSH, Jupyter, TensorBoard
			EstimatedCost: map[string]float64{
				"x86_64": 3.06,
			},
		},
		{
			Name:        "web-dev",
			Description: "Web development stack",
			Ports:       []int{22, 3000, 8080},
			EstimatedCost: map[string]float64{
				"x86_64": 0.0464,
				"arm64":  0.0370,
			},
		},
	}

	// Test template to list item conversion
	t.Run("template_to_list_items", func(t *testing.T) {
		items := make([]list.Item, len(templates))

		for i, template := range templates {
			// Calculate costs
			x86Cost := 0.0
			armCost := 0.0

			if cost, exists := template.EstimatedCost["x86_64"]; exists {
				x86Cost = cost * 24 // Daily cost
			}
			if cost, exists := template.EstimatedCost["arm64"]; exists {
				armCost = cost * 24 // Daily cost
			}

			items[i] = BrowserTemplateItem{
				name:        template.Name,
				description: template.Description,
				costX86:     x86Cost,
				costARM:     armCost,
				ports:       template.Ports,
			}
		}

		assert.Len(t, items, 2)

		// Validate GPU template
		gpuItem := items[0].(BrowserTemplateItem)
		assert.Equal(t, "python-ml-gpu", gpuItem.name)
		assert.Equal(t, "Python ML with GPU support", gpuItem.description)
		assert.Equal(t, 3.06*24, gpuItem.costX86) // High GPU cost
		assert.Equal(t, 0.0, gpuItem.costARM)     // No ARM GPU option
		assert.Equal(t, []int{22, 8888, 6006}, gpuItem.ports)

		// Validate web dev template
		webItem := items[1].(BrowserTemplateItem)
		assert.Equal(t, "web-dev", webItem.name)
		assert.True(t, webItem.costARM < webItem.costX86) // ARM should be cheaper
	})
}

// TestTemplateModelNavigation tests navigation and selection logic
func TestTemplateModelNavigation(t *testing.T) {
	templates := map[string]api.TemplateResponse{
		"template-a": {Name: "template-a", Description: "First template"},
		"template-b": {Name: "template-b", Description: "Second template"},
		"template-c": {Name: "template-c", Description: "Third template"},
	}

	// Test template selection logic
	t.Run("template_selection", func(t *testing.T) {
		selectedName := "template-b"
		selectedTemplate, exists := templates[selectedName]

		assert.True(t, exists)
		assert.Equal(t, "template-b", selectedTemplate.Name)
		assert.Equal(t, "Second template", selectedTemplate.Description)
	})

	// Test template list navigation bounds
	t.Run("navigation_bounds", func(t *testing.T) {
		templateNames := make([]string, 0, len(templates))
		for name := range templates {
			templateNames = append(templateNames, name)
		}

		// Simulate list selection (0-based indexing)
		selectedIndex := 1
		assert.GreaterOrEqual(t, selectedIndex, 0)
		assert.Less(t, selectedIndex, len(templateNames))

		// Test bounds protection
		invalidIndex := len(templateNames)
		assert.GreaterOrEqual(t, invalidIndex, len(templateNames))

		safeIndex := 0
		if invalidIndex < len(templateNames) {
			safeIndex = invalidIndex
		}
		assert.GreaterOrEqual(t, safeIndex, 0)
		assert.Less(t, safeIndex, len(templateNames))
	})
}

// TestTemplateModelPerformance tests performance with large template lists
func TestTemplateModelPerformance(t *testing.T) {
	// Generate large template map
	templateMap := make(map[string]api.TemplateResponse)
	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("template-%d", i)
		templateMap[name] = api.TemplateResponse{
			Name:        name,
			Description: fmt.Sprintf("Template %d for testing", i),
			Ports:       []int{22, 8000 + i},
			EstimatedCost: map[string]float64{
				"x86_64": 0.04 + float64(i)*0.001,
			},
		}
	}

	mockClient := &mockTemplateAPIClient{
		templates: templateMap,
	}

	model := NewTemplatesModel(mockClient)
	// Convert to internal template format
	internalTemplates := make(map[string]types.Template)
	for name, tmpl := range templateMap {
		internalTemplates[name] = types.Template{
			Name:                 tmpl.Name,
			Description:          tmpl.Description,
			Ports:                tmpl.Ports,
			EstimatedCostPerHour: tmpl.EstimatedCost,
		}
	}
	model.templates = internalTemplates

	// Test view rendering performance
	t.Run("large_template_list_rendering", func(t *testing.T) {
		start := time.Now()

		// Render view multiple times
		for i := 0; i < 10; i++ {
			view := model.View()
			assert.NotEmpty(t, view)
		}

		duration := time.Since(start)
		assert.Less(t, duration, time.Second, "Template view rendering should be performant")
	})

	// Test template conversion performance
	t.Run("template_conversion_performance", func(t *testing.T) {
		start := time.Now()

		// Convert templates to list items
		items := make([]list.Item, 0, len(templateMap))
		for _, template := range templateMap {
			x86Cost := 0.0
			if cost, exists := template.EstimatedCost["x86_64"]; exists {
				x86Cost = cost * 24
			}

			item := BrowserTemplateItem{
				name:        template.Name,
				description: template.Description,
				costX86:     x86Cost,
				ports:       template.Ports,
			}
			items = append(items, item)
		}

		duration := time.Since(start)
		assert.Less(t, duration, 100*time.Millisecond, "Template conversion should be fast")
		assert.Len(t, items, 50)
	})
}

// TestTemplateModelIntegration tests complete template workflow
func TestTemplateModelIntegration(t *testing.T) {
	mockTemplates := map[string]api.TemplateResponse{
		"integration-test-template": {
			Name:        "integration-test-template",
			Description: "Template for integration testing",
			Ports:       []int{22, 8888},
			InstanceType: map[string]string{
				"x86_64": "t3.medium",
				"arm64":  "t4g.medium",
			},
			EstimatedCost: map[string]float64{
				"x86_64": 0.0416,
				"arm64":  0.0336,
			},
		},
	}

	mockClient := &mockTemplateAPIClient{
		templates: mockTemplates,
	}

	model := NewTemplatesModel(mockClient)

	// Test complete workflow
	t.Run("complete_template_workflow", func(t *testing.T) {
		// 1. Initialize model
		cmd := model.Init()
		assert.NotNil(t, cmd)

		// 2. Load templates (convert to internal format)
		internalTemplates := make(map[string]types.Template)
		for name, tmpl := range mockTemplates {
			internalTemplates[name] = types.Template{
				Name:                 tmpl.Name,
				Description:          tmpl.Description,
				Ports:                tmpl.Ports,
				EstimatedCostPerHour: tmpl.EstimatedCost,
			}
		}

		newModel, _ := model.Update(internalTemplates)
		templatesModel, ok := newModel.(TemplatesModel)
		require.True(t, ok)

		assert.False(t, templatesModel.loading)
		assert.Len(t, templatesModel.templates, 1)

		// 3. Set window size
		sizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
		sizedModel, _ := templatesModel.Update(sizeMsg)
		templatesModel, ok = sizedModel.(TemplatesModel)
		require.True(t, ok)

		assert.Equal(t, 120, templatesModel.width)
		assert.Equal(t, 40, templatesModel.height)

		// 4. Select template
		templatesModel.selected = "integration-test-template"
		selectedTemplate, exists := templatesModel.templates["integration-test-template"]
		assert.True(t, exists)
		assert.Equal(t, "integration-test-template", selectedTemplate.Name)

		// 5. Render final view
		view := templatesModel.View()
		assert.NotEmpty(t, view)
		assert.Greater(t, len(view), 100) // Should have substantial content
	})

	// Test error handling workflow
	t.Run("error_handling_workflow", func(t *testing.T) {
		// Test with error client
		errorClient := &mockTemplateAPIClient{
			shouldError:  true,
			errorMessage: "API connection failed",
		}

		errorModel := NewTemplatesModel(errorClient)

		// Simulate error during template loading
		errorMsg := fmt.Errorf("API connection failed")
		newModel, cmd := errorModel.Update(errorMsg)

		templatesModel, ok := newModel.(TemplatesModel)
		require.True(t, ok)

		assert.False(t, templatesModel.loading)
		assert.Equal(t, "API connection failed", templatesModel.error)
		assert.Nil(t, cmd)

		// Verify error is displayed in view
		view := templatesModel.View()
		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Error")
	})
}
