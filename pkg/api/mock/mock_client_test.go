// Package mock provides comprehensive test coverage for the MockClient implementation
package mock

import (
	"context"
	"testing"

	"github.com/scttfrdmn/cloudworkstation/pkg/project"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockClientCreation validates basic MockClient instantiation
func TestMockClientCreation(t *testing.T) {
	client := NewClient()
	require.NotNil(t, client)
	assert.NotNil(t, client.Templates)
	assert.NotNil(t, client.Instances)
	assert.NotNil(t, client.Volumes)
	assert.NotNil(t, client.Storage)

	// Validate pre-populated data
	assert.Greater(t, len(client.Templates), 0, "Mock client should have pre-loaded templates")
	assert.Greater(t, len(client.Instances), 0, "Mock client should have pre-loaded instances")
	assert.Greater(t, len(client.Volumes), 0, "Mock client should have pre-loaded volumes")
	assert.Greater(t, len(client.Storage), 0, "Mock client should have pre-loaded storage")
}

// TestMockTemplateData validates template data consistency
func TestMockTemplateData(t *testing.T) {
	client := NewClient()

	// Test expected templates are present
	expectedTemplates := []string{
		"basic-ubuntu", "r-research", "python-ml", "desktop-research", "data-science",
	}

	for _, templateName := range expectedTemplates {
		template, exists := client.Templates[templateName]
		assert.True(t, exists, "Expected template %s should exist", templateName)
		assert.NotEmpty(t, template.Name)
		assert.NotEmpty(t, template.Description)
		assert.NotNil(t, template.AMI)
		assert.NotNil(t, template.InstanceType)
		assert.NotNil(t, template.EstimatedCostPerHour)
		assert.Greater(t, len(template.Ports), 0)
	}
}

// TestMockInstanceData validates instance data consistency
func TestMockInstanceData(t *testing.T) {
	client := NewClient()

	for name, instance := range client.Instances {
		assert.Equal(t, name, instance.Name)
		assert.NotEmpty(t, instance.ID)
		assert.NotEmpty(t, instance.Template)
		assert.NotEmpty(t, instance.State)
		assert.Greater(t, instance.HourlyRate, 0.0)
		assert.NotEmpty(t, instance.PublicIP)
		assert.NotEmpty(t, instance.PrivateIP)

		// Validate template reference exists
		_, exists := client.Templates[instance.Template]
		assert.True(t, exists, "Instance %s references non-existent template %s", name, instance.Template)
	}
}

// TestLaunchInstanceFunctionality tests the comprehensive launch workflow
func TestLaunchInstanceFunctionality(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	tests := []struct {
		name        string
		request     types.LaunchRequest
		expectError bool
		description string
	}{
		{
			name: "valid launch request",
			request: types.LaunchRequest{
				Template: "basic-ubuntu",
				Name:     "test-instance",
				Size:     "M",
				Spot:     false,
				DryRun:   false,
			},
			expectError: false,
			description: "Basic valid launch should succeed",
		},
		{
			name: "spot instance with cost savings",
			request: types.LaunchRequest{
				Template: "python-ml",
				Name:     "ml-spot-instance",
				Size:     "L",
				Spot:     true,
				DryRun:   false,
			},
			expectError: false,
			description: "Spot instances should have cost savings applied",
		},
		{
			name: "dry run request",
			request: types.LaunchRequest{
				Template: "r-research",
				Name:     "dry-run-test",
				Size:     "S",
				DryRun:   true,
			},
			expectError: false,
			description: "Dry run should not create actual instance",
		},
		{
			name: "invalid template",
			request: types.LaunchRequest{
				Template: "non-existent-template",
				Name:     "test-invalid",
			},
			expectError: true,
			description: "Invalid template should return error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialInstanceCount := len(client.Instances)

			response, err := client.LaunchInstance(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.Message)
				assert.NotEmpty(t, response.EstimatedCost)
				assert.Equal(t, tt.request.Template, response.Instance.Template)
				assert.Equal(t, tt.request.Name, response.Instance.Name)

				if !tt.request.DryRun {
					// Verify instance was actually created
					assert.Equal(t, initialInstanceCount+1, len(client.Instances))
					instance, exists := client.Instances[tt.request.Name]
					assert.True(t, exists)
					assert.Equal(t, "running", instance.State)
				} else {
					// Dry run should not create instance
					assert.Equal(t, initialInstanceCount, len(client.Instances))
				}

				// Test spot pricing calculation
				if tt.request.Spot {
					template := client.Templates[tt.request.Template]
					baseCost := template.EstimatedCostPerHour["x86_64"]
					if template.EstimatedCostPerHour["arm64"] > 0 {
						baseCost = template.EstimatedCostPerHour["arm64"] // ARM is preferred
					}

					// Apply size multiplier first, then spot discount
					sizeMultiplier := 1.0
					if tt.request.Size == "L" {
						sizeMultiplier = 2.0
					}
					adjustedCost := baseCost * sizeMultiplier
					expectedSpotCost := adjustedCost * 0.3 // 70% discount
					assert.InDelta(t, expectedSpotCost, response.Instance.HourlyRate, 0.1)
				}
			}
		})
	}
}

// TestInstanceManagementWorkflow tests the complete instance lifecycle
func TestInstanceManagementWorkflow(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Launch instance
	launchReq := types.LaunchRequest{
		Template: "basic-ubuntu",
		Name:     "lifecycle-test",
		Size:     "M",
	}

	response, err := client.LaunchInstance(ctx, launchReq)
	require.NoError(t, err)
	require.NotNil(t, response)

	instanceName := response.Instance.Name

	// Test GetInstance
	instance, err := client.GetInstance(ctx, instanceName)
	require.NoError(t, err)
	assert.Equal(t, "running", instance.State)

	// Test StopInstance
	err = client.StopInstance(ctx, instanceName)
	require.NoError(t, err)
	instance, _ = client.GetInstance(ctx, instanceName)
	assert.Equal(t, "stopped", instance.State)

	// Test StartInstance
	err = client.StartInstance(ctx, instanceName)
	require.NoError(t, err)
	instance, _ = client.GetInstance(ctx, instanceName)
	assert.Equal(t, "running", instance.State)

	// Test HibernateInstance
	err = client.HibernateInstance(ctx, instanceName)
	require.NoError(t, err)
	instance, _ = client.GetInstance(ctx, instanceName)
	assert.Equal(t, "hibernated", instance.State)

	// Test ResumeInstance
	err = client.ResumeInstance(ctx, instanceName)
	require.NoError(t, err)
	instance, _ = client.GetInstance(ctx, instanceName)
	assert.Equal(t, "running", instance.State)

	// Test hibernation status
	status, err := client.GetInstanceHibernationStatus(ctx, instanceName)
	require.NoError(t, err)
	assert.True(t, status.HibernationSupported)
	assert.False(t, status.PossiblyHibernated) // Should be false since we resumed

	// Test ListInstances
	listResp, err := client.ListInstances(ctx)
	require.NoError(t, err)
	assert.Greater(t, len(listResp.Instances), 0)

	// Find our instance in the list
	found := false
	for _, inst := range listResp.Instances {
		if inst.Name == instanceName {
			found = true
			break
		}
	}
	assert.True(t, found, "Launched instance should appear in list")

	// Test DeleteInstance
	err = client.DeleteInstance(ctx, instanceName)
	require.NoError(t, err)

	// Verify deletion
	_, err = client.GetInstance(ctx, instanceName)
	assert.Error(t, err, "Getting deleted instance should return error")
}

// TestVolumeManagementWorkflow tests EFS volume operations
func TestVolumeManagementWorkflow(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	volumeName := "test-volume"

	// Create volume
	createReq := types.VolumeCreateRequest{
		Name:            volumeName,
		Region:          "us-east-1",
		PerformanceMode: "generalPurpose",
		ThroughputMode:  "bursting",
	}

	volume, err := client.CreateVolume(ctx, createReq)
	require.NoError(t, err)
	assert.Equal(t, volumeName, volume.Name)
	assert.Equal(t, "available", volume.State)

	// Test GetVolume
	retrievedVolume, err := client.GetVolume(ctx, volumeName)
	require.NoError(t, err)
	assert.Equal(t, volumeName, retrievedVolume.Name)

	// Test ListVolumes
	volumes, err := client.ListVolumes(ctx)
	require.NoError(t, err)
	assert.Greater(t, len(volumes), 0)

	// Test volume attachment (requires instance)
	launchReq := types.LaunchRequest{
		Template: "basic-ubuntu",
		Name:     "volume-test-instance",
	}
	launchResp, err := client.LaunchInstance(ctx, launchReq)
	require.NoError(t, err)

	err = client.AttachVolume(ctx, volumeName, launchResp.Instance.Name)
	require.NoError(t, err)

	// Verify attachment
	instance, err := client.GetInstance(ctx, launchResp.Instance.Name)
	require.NoError(t, err)
	assert.Contains(t, instance.AttachedVolumes, volumeName)

	// Test detachment
	err = client.DetachVolume(ctx, volumeName)
	require.NoError(t, err)

	// Test DeleteVolume
	err = client.DeleteVolume(ctx, volumeName)
	require.NoError(t, err)

	// Verify deletion
	_, err = client.GetVolume(ctx, volumeName)
	assert.Error(t, err)

	// Cleanup instance
	_ = client.DeleteInstance(ctx, launchResp.Instance.Name)
}

// TestStorageManagementWorkflow tests EBS storage operations
func TestStorageManagementWorkflow(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	storageName := "test-storage"

	// Test different size configurations
	sizeTests := []struct {
		size         string
		expectedSize int32
	}{
		{"XS", 100},
		{"S", 250},
		{"M", 500},
		{"L", 1000},
		{"XL", 2000},
		{"XXL", 4000},
	}

	for _, tt := range sizeTests {
		t.Run("size_"+tt.size, func(t *testing.T) {
			testStorageName := storageName + "-" + tt.size

			createReq := types.StorageCreateRequest{
				Name:       testStorageName,
				Size:       tt.size,
				VolumeType: "gp3",
				Region:     "us-east-1",
			}

			storage, err := client.CreateStorage(ctx, createReq)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSize, storage.SizeGB)
			assert.Equal(t, "gp3", storage.VolumeType)

			// Cleanup
			_ = client.DeleteStorage(ctx, testStorageName)
		})
	}
}

// TestTemplateOperations tests template-related functionality
func TestTemplateOperations(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test ListTemplates
	templates, err := client.ListTemplates(ctx)
	require.NoError(t, err)
	assert.Greater(t, len(templates), 0)

	// Test GetTemplate for each template
	for templateName := range templates {
		template, err := client.GetTemplate(ctx, templateName)
		require.NoError(t, err, "Failed to get template %s", templateName)
		assert.Equal(t, templateName, template.Name)
		assert.NotEmpty(t, template.Description)
	}

	// Test invalid template
	_, err = client.GetTemplate(ctx, "non-existent")
	assert.Error(t, err)
}

// TestProjectManagementOperations tests project-related mock functionality
func TestProjectManagementOperations(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test CreateProject - simplified to avoid complex budget types
	createReq := project.CreateProjectRequest{
		Name:        "Test Project",
		Description: "A test project",
		// Skip budget for now to avoid type complexity
	}

	proj, err := client.CreateProject(ctx, createReq)
	require.NoError(t, err)
	assert.Equal(t, createReq.Name, proj.Name)
	assert.Equal(t, createReq.Description, proj.Description)

	// Test ListProjects
	listResp, err := client.ListProjects(ctx, nil)
	require.NoError(t, err)
	assert.Greater(t, len(listResp.Projects), 0)

	// Test GetProject
	retrievedProject, err := client.GetProject(ctx, proj.ID)
	require.NoError(t, err)
	assert.Equal(t, proj.ID, retrievedProject.ID)

	// Test UpdateProject
	newName := "Updated Project Name"
	updateReq := project.UpdateProjectRequest{
		Name: &newName,
	}
	updatedProj, err := client.UpdateProject(ctx, proj.ID, updateReq)
	require.NoError(t, err)
	assert.Equal(t, newName, updatedProj.Name)

	// Test project budget operations
	budgetStatus, err := client.GetProjectBudgetStatus(ctx, proj.ID)
	require.NoError(t, err)
	assert.True(t, budgetStatus.BudgetEnabled)
	assert.Greater(t, budgetStatus.TotalBudget, 0.0)

	// Test DeleteProject
	err = client.DeleteProject(ctx, proj.ID)
	assert.NoError(t, err)
}

// TestIdlePolicyOperations tests idle policy functionality
func TestIdlePolicyOperations(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test ListIdlePolicies
	policies, err := client.ListIdlePolicies(ctx)
	require.NoError(t, err)
	assert.Greater(t, len(policies), 0)

	// Test GetIdlePolicy - use policy ID, not name
	for _, policy := range policies {
		retrievedPolicy, err := client.GetIdlePolicy(ctx, policy.ID)
		require.NoError(t, err, "Failed to get policy %s", policy.ID)
		assert.Equal(t, policy.ID, retrievedPolicy.ID)
	}

	// Test policy operations on instance
	launchReq := types.LaunchRequest{
		Template: "basic-ubuntu",
		Name:     "policy-test-instance",
	}
	launchResp, err := client.LaunchInstance(ctx, launchReq)
	require.NoError(t, err)

	instanceID := launchResp.Instance.Name

	// Test ApplyIdlePolicy
	err = client.ApplyIdlePolicy(ctx, instanceID, "balanced")
	assert.NoError(t, err)

	// Test GetInstanceIdlePolicies
	instancePolicies, err := client.GetInstanceIdlePolicies(ctx, instanceID)
	require.NoError(t, err)
	assert.NotNil(t, instancePolicies)

	// Test RecommendIdlePolicy
	recommendation, err := client.RecommendIdlePolicy(ctx, instanceID)
	require.NoError(t, err)
	assert.NotNil(t, recommendation)

	// Cleanup
	client.DeleteInstance(ctx, instanceID)
}

// TestStatusAndUtilityOperations tests daemon status and utility functions
func TestStatusAndUtilityOperations(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test Ping
	err := client.Ping(ctx)
	assert.NoError(t, err)

	// Test GetStatus
	status, err := client.GetStatus(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, status.Version)
	assert.Equal(t, "running", status.Status)

	// Test Shutdown (mock doesn't actually shut down)
	err = client.Shutdown(ctx)
	assert.NoError(t, err)

	// Test MakeRequest
	response, err := client.MakeRequest("GET", "/test", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, response)
}

// TestCostCalculationAccuracy tests the cost calculation logic in mock client
func TestCostCalculationAccuracy(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test size multipliers
	sizeTests := []struct {
		size       string
		multiplier float64
		isGPU      bool
	}{
		{"XS", 0.5, false},
		{"S", 0.75, false},
		{"M", 1.0, false},
		{"L", 2.0, false},
		{"XL", 4.0, false},
		{"GPU-S", 0.75, true}, // Absolute cost
		{"GPU-M", 1.5, true},  // Absolute cost
		{"GPU-L", 3.0, true},  // Absolute cost
	}

	for _, tt := range sizeTests {
		t.Run("size_"+tt.size, func(t *testing.T) {
			launchReq := types.LaunchRequest{
				Template: "python-ml",
				Name:     "cost-test-" + tt.size,
				Size:     tt.size,
				DryRun:   true,
			}

			response, err := client.LaunchInstance(ctx, launchReq)
			require.NoError(t, err)

			template := client.Templates["python-ml"]
			baseCost := template.EstimatedCostPerHour["arm64"] // ARM preferred

			expectedCost := baseCost * tt.multiplier
			if tt.isGPU {
				expectedCost = tt.multiplier // Absolute cost for GPU
			}

			assert.InDelta(t, expectedCost, response.Instance.HourlyRate, 0.01,
				"Cost calculation incorrect for size %s", tt.size)
		})
	}
}

// TestEdgeCasesAndErrorHandling tests error scenarios and edge cases
func TestEdgeCasesAndErrorHandling(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test operations on non-existent instances
	err := client.StopInstance(ctx, "non-existent")
	assert.Error(t, err)

	err = client.StartInstance(ctx, "non-existent")
	assert.Error(t, err)

	err = client.HibernateInstance(ctx, "non-existent")
	assert.Error(t, err)

	err = client.ResumeInstance(ctx, "non-existent")
	assert.Error(t, err)

	// Test duplicate volume creation
	volumeReq := types.VolumeCreateRequest{Name: "duplicate-test"}
	_, err = client.CreateVolume(ctx, volumeReq)
	require.NoError(t, err)

	_, err = client.CreateVolume(ctx, volumeReq)
	assert.Error(t, err, "Creating duplicate volume should fail")

	// Test duplicate storage creation
	storageReq := types.StorageCreateRequest{Name: "duplicate-storage-test"}
	_, err = client.CreateStorage(ctx, storageReq)
	require.NoError(t, err)

	_, err = client.CreateStorage(ctx, storageReq)
	assert.Error(t, err, "Creating duplicate storage should fail")

	// Test attachment to non-existent instance
	err = client.AttachVolume(ctx, "duplicate-test", "non-existent")
	assert.Error(t, err)

	err = client.AttachStorage(ctx, "duplicate-storage-test", "non-existent")
	assert.Error(t, err)
}

// TestTemplateConnectionInfo validates connection information generation
func TestTemplateConnectionInfo(t *testing.T) {
	client := NewClient()

	connectionTests := []struct {
		template     string
		expectsHTTP  bool
		expectedPort string
	}{
		{"r-research", true, "8787"},
		{"python-ml", true, "8888"},
		{"desktop-research", true, "8443"},
		{"basic-ubuntu", false, ""},
	}

	for _, tt := range connectionTests {
		t.Run(tt.template, func(t *testing.T) {
			launchReq := types.LaunchRequest{
				Template: tt.template,
				Name:     "connection-test",
				DryRun:   true,
			}

			response, err := client.LaunchInstance(context.Background(), launchReq)
			require.NoError(t, err)

			if tt.expectsHTTP {
				assert.Contains(t, response.ConnectionInfo, tt.expectedPort)
				assert.Contains(t, response.ConnectionInfo, "http")
			} else {
				assert.Contains(t, response.ConnectionInfo, "ssh")
			}
		})
	}
}
