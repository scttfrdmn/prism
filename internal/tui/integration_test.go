// Package tui provides CloudWorkstation's terminal user interface implementation.
//
// This file contains integration tests between TUI components and the daemon API.
package tui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scttfrdmn/cloudworkstation/internal/tui/models"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/daemon"
	"github.com/scttfrdmn/cloudworkstation/pkg/state"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTUIModelIntegration tests the integration between TUI models and the daemon API
func TestTUIModelIntegration(t *testing.T) {
	// Create a test state manager
	stateManager, err := state.NewManager(":memory:", false)
	require.NoError(t, err)

	// Create test instance
	testInstance := types.Instance{
		ID:               "i-12345678",
		Name:             "test-instance",
		Template:         "r-research",
		PublicIP:         "1.2.3.4",
		State:            "running",
		EstimatedCostDay: 1.15,
		LaunchTime:       "2024-07-01T12:00:00Z",
	}
	err = stateManager.SaveInstance(testInstance)
	require.NoError(t, err)

	// Create test templates
	templates := map[string]types.Template{
		"r-research": {
			Name:        "R Research Environment",
			Description: "R + RStudio Server + tidyverse packages for statistical analysis",
			InstanceType: map[string]string{
				"x86_64": "t3.medium",
				"arm64":  "t4g.medium",
			},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.0464,
				"arm64":  0.0368,
			},
		},
		"python-research": {
			Name:        "Python Research Environment",
			Description: "Python + Jupyter + data science packages",
			InstanceType: map[string]string{
				"x86_64": "t3.medium",
				"arm64":  "t4g.medium",
			},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.0464,
				"arm64":  0.0368,
			},
		},
	}

	// Create test server with daemon handler
	server := daemon.NewServer(stateManager, nil, daemon.Options{})
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r)
	}))
	defer testServer.Close()

	// Create API client pointing to test server
	apiClient := api.NewClient(testServer.URL)

	// Test Instances model
	t.Run("InstancesModel", func(t *testing.T) {
		ctx := context.Background()
		model := models.NewInstancesModel(apiClient)

		// Test loading instances
		err = model.LoadInstances(ctx)
		require.NoError(t, err)
		require.Len(t, model.Instances, 1, "Should load one instance")
		assert.Equal(t, "test-instance", model.Instances[0].Name)
		assert.Equal(t, "running", model.Instances[0].State)
		assert.Equal(t, "r-research", model.Instances[0].Template)

		// Test stopping an instance
		model.Selected = 0
		err = model.StopSelected(ctx)
		require.NoError(t, err)

		// Refresh instances and verify state change
		err = model.LoadInstances(ctx)
		require.NoError(t, err)
		assert.Equal(t, "stopped", model.Instances[0].State)
	})

	// Test Templates model
	t.Run("TemplatesModel", func(t *testing.T) {
		ctx := context.Background()
		model := models.NewTemplatesModel(apiClient)

		// Create a mock client that returns our test templates
		mockClient := &mockTemplatesClient{
			templates: templates,
		}
		model.Client = mockClient

		// Test loading templates
		err = model.LoadTemplates(ctx)
		require.NoError(t, err)
		require.Len(t, model.Templates, 2, "Should load two templates")
		assert.Contains(t, model.TemplateNames, "r-research")
		assert.Contains(t, model.TemplateNames, "python-research")
	})

	// Test Dashboard model
	t.Run("DashboardModel", func(t *testing.T) {
		ctx := context.Background()
		model := models.NewDashboardModel(apiClient)

		// Test loading data
		err = model.LoadData(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, model.Stats.RunningInstances)
		assert.Equal(t, 0, model.Stats.StoppedInstances)
		assert.InDelta(t, 1.15, model.Stats.TotalDailyCost, 0.01)
	})
}

// mockTemplatesClient is a mock implementation of the templates API
type mockTemplatesClient struct {
	templates map[string]types.Template
}

func (m *mockTemplatesClient) ListTemplates(ctx context.Context) (map[string]types.Template, error) {
	return m.templates, nil
}

func (m *mockTemplatesClient) GetTemplate(ctx context.Context, name string) (*types.Template, error) {
	template, ok := m.templates[name]
	if !ok {
		return nil, api.NewAPIError("template not found", http.StatusNotFound)
	}
	return &template, nil
}

// Test_TUIEndToEndWorkflow tests a full workflow across multiple TUI components
func Test_TUIEndToEndWorkflow(t *testing.T) {
	// Create a test state manager
	stateManager, err := state.NewManager(":memory:", false)
	require.NoError(t, err)

	// Create test server with daemon handler
	server := daemon.NewServer(stateManager, nil, daemon.Options{})
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r)
	}))
	defer testServer.Close()

	// Create API client pointing to test server
	apiClient := api.NewClient(testServer.URL)
	ctx := context.Background()

	// 1. Start with empty system
	dashModel := models.NewDashboardModel(apiClient)
	err = dashModel.LoadData(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, dashModel.Stats.RunningInstances)
	assert.Equal(t, 0, dashModel.Stats.StoppedInstances)

	// 2. Create a mock template
	template := types.Template{
		Name:        "test-template",
		Description: "Test template for integration tests",
		InstanceType: map[string]string{
			"x86_64": "t3.small",
			"arm64":  "t4g.small",
		},
		EstimatedCostPerHour: map[string]float64{
			"x86_64": 0.0208,
			"arm64":  0.0168,
		},
	}

	// Save template to state
	err = stateManager.SaveState(types.State{
		Templates: map[string]types.Template{
			"test-template": template,
		},
	})
	require.NoError(t, err)

	// 3. Launch an instance (simulate the API action directly since we can't fully test the launch workflow)
	instance := types.Instance{
		ID:               "i-workflow123",
		Name:             "workflow-test",
		Template:         "test-template",
		PublicIP:         "5.6.7.8",
		State:            "running",
		EstimatedCostDay: 0.50,
		LaunchTime:       "2024-07-11T10:00:00Z",
	}
	err = stateManager.SaveInstance(instance)
	require.NoError(t, err)

	// 4. Verify in instances model
	instModel := models.NewInstancesModel(apiClient)
	err = instModel.LoadInstances(ctx)
	require.NoError(t, err)
	require.Len(t, instModel.Instances, 1)
	assert.Equal(t, "workflow-test", instModel.Instances[0].Name)

	// 5. Check dashboard again to verify counts updated
	err = dashModel.LoadData(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, dashModel.Stats.RunningInstances)
	assert.Equal(t, 0, dashModel.Stats.StoppedInstances)
	assert.InDelta(t, 0.50, dashModel.Stats.TotalDailyCost, 0.01)

	// 6. Stop the instance
	instModel.Selected = 0
	err = instModel.StopSelected(ctx)
	require.NoError(t, err)

	// 7. Verify instance state changed
	err = instModel.LoadInstances(ctx)
	require.NoError(t, err)
	assert.Equal(t, "stopped", instModel.Instances[0].State)

	// 8. Check dashboard again to verify counts updated
	err = dashModel.LoadData(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, dashModel.Stats.RunningInstances)
	assert.Equal(t, 1, dashModel.Stats.StoppedInstances)

	// 9. Delete the instance
	instModel.Selected = 0
	err = instModel.DeleteSelected(ctx)
	require.NoError(t, err)

	// 10. Verify instance was removed
	err = instModel.LoadInstances(ctx)
	require.NoError(t, err)
	assert.Len(t, instModel.Instances, 0)

	// 11. Check dashboard again to verify counts updated
	err = dashModel.LoadData(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, dashModel.Stats.RunningInstances)
	assert.Equal(t, 0, dashModel.Stats.StoppedInstances)
	assert.InDelta(t, 0.0, dashModel.Stats.TotalDailyCost, 0.01)
}