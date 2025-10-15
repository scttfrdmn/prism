// Package api provides an API client for the TUI.
package api

import (
	"context"
	"time"

	pkgapi "github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/idle"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TUIClient wraps the CloudWorkstationAPI interface to provide
// a consistent interface for the TUI models.
type TUIClient struct {
	client pkgapi.CloudWorkstationAPI
}

// NewTUIClient creates a new TUIClient.
func NewTUIClient(client pkgapi.CloudWorkstationAPI) *TUIClient {
	return &TUIClient{
		client: client,
	}
}

// Instance operations

// ListInstances returns all instances
func (c *TUIClient) ListInstances(ctx context.Context) (*ListInstancesResponse, error) {
	resp, err := c.client.ListInstances(ctx)
	if err != nil {
		return nil, err
	}
	return ToListInstancesResponse(resp), nil
}

// GetInstance returns a specific instance
func (c *TUIClient) GetInstance(ctx context.Context, name string) (*InstanceResponse, error) {
	instance, err := c.client.GetInstance(ctx, name)
	if err != nil {
		return nil, err
	}
	resp := ToInstanceResponse(*instance)
	return &resp, nil
}

// LaunchInstance launches a new instance
func (c *TUIClient) LaunchInstance(ctx context.Context, req LaunchInstanceRequest) (*LaunchInstanceResponse, error) {
	// Convert request
	launchReq := types.LaunchRequest{
		Template:   req.Template,
		Name:       req.Name,
		Size:       req.Size,
		Volumes:    req.Volumes,
		EBSVolumes: req.EBSVolumes,
		Region:     req.Region,
		Spot:       req.Spot,
		DryRun:     req.DryRun,
	}

	// Make API call
	resp, err := c.client.LaunchInstance(ctx, launchReq)
	if err != nil {
		return nil, err
	}

	// Convert response
	return &LaunchInstanceResponse{
		Instance:       ToInstanceResponse(resp.Instance),
		Message:        resp.Message,
		EstimatedCost:  resp.EstimatedCost,
		ConnectionInfo: resp.ConnectionInfo,
	}, nil
}

// StartInstance starts a stopped instance
func (c *TUIClient) StartInstance(ctx context.Context, name string) error {
	return c.client.StartInstance(ctx, name)
}

// StopInstance stops a running instance
func (c *TUIClient) StopInstance(ctx context.Context, name string) error {
	return c.client.StopInstance(ctx, name)
}

// DeleteInstance terminates an instance
func (c *TUIClient) DeleteInstance(ctx context.Context, name string) error {
	return c.client.DeleteInstance(ctx, name)
}

// Template operations

// ListTemplates returns all available templates
func (c *TUIClient) ListTemplates(ctx context.Context) (*ListTemplatesResponse, error) {
	templates, err := c.client.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	return ToListTemplatesResponse(templates), nil
}

// GetTemplate returns a specific template
func (c *TUIClient) GetTemplate(ctx context.Context, name string) (*TemplateResponse, error) {
	template, err := c.client.GetTemplate(ctx, name)
	if err != nil {
		return nil, err
	}
	resp := ToTemplateResponse(name, *template)
	return &resp, nil
}

// Volume operations

// ListVolumes returns all EFS volumes
func (c *TUIClient) ListVolumes(ctx context.Context) (*ListVolumesResponse, error) {
	volumes, err := c.client.ListVolumes(ctx)
	if err != nil {
		return nil, err
	}
	return ToListVolumesResponse(volumes), nil
}

// Storage operations

// ListStorage returns all EBS volumes
func (c *TUIClient) ListStorage(ctx context.Context) (*ListStorageResponse, error) {
	storage, err := c.client.ListStorage(ctx)
	if err != nil {
		return nil, err
	}
	return ToListStorageResponse(storage), nil
}

// Status operations

// Ping checks if the daemon is responsive
func (c *TUIClient) Ping(ctx context.Context) error {
	return c.client.Ping(ctx)
}

// GetStatus returns the daemon status information
func (c *TUIClient) GetStatus(ctx context.Context) (*SystemStatusResponse, error) {
	status, err := c.client.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	return ToSystemStatusResponse(status), nil
}

// Idle detection operations

// ListIdlePolicies returns all idle detection policies
func (c *TUIClient) ListIdlePolicies(ctx context.Context) (*ListIdlePoliciesResponse, error) {
	// Get real policy templates from the idle policy manager
	policyManager := idle.NewPolicyManager()
	templates := policyManager.ListTemplates()

	// Convert policy templates to TUI response format
	policies := make(map[string]IdlePolicyResponse)
	for _, template := range templates {
		// Extract idle threshold and action from the first schedule
		idleMinutes := 30 // Default threshold
		action := "stop"  // Default action
		if len(template.Schedules) > 0 {
			schedule := template.Schedules[0]
			if schedule.IdleMinutes > 0 {
				idleMinutes = schedule.IdleMinutes
			}
			if schedule.HibernateAction != "" {
				action = schedule.HibernateAction
			}
		}

		policies[template.ID] = IdlePolicyResponse{
			Name:      template.Name,
			Threshold: idleMinutes,
			Action:    action,
		}
	}

	return &ListIdlePoliciesResponse{
		Policies: policies,
	}, nil
}

// UpdateIdlePolicy updates an idle detection policy
func (c *TUIClient) UpdateIdlePolicy(ctx context.Context, req IdlePolicyUpdateRequest) error {
	// This is a stub implementation
	return nil
}

// GetInstanceIdleStatus returns idle detection status for an instance
func (c *TUIClient) GetInstanceIdleStatus(ctx context.Context, name string) (*IdleDetectionResponse, error) {
	// This is a stub implementation
	return &IdleDetectionResponse{
		Enabled:       true,
		Policy:        "default",
		IdleTime:      5,  // 5 minutes
		Threshold:     30, // 30 minutes
		ActionPending: false,
	}, nil
}

// EnableIdleDetection enables idle detection for an instance
func (c *TUIClient) EnableIdleDetection(ctx context.Context, name, policy string) error {
	// This is a stub implementation
	return nil
}

// DisableIdleDetection disables idle detection for an instance
func (c *TUIClient) DisableIdleDetection(ctx context.Context, name string) error {
	// This is a stub implementation
	return nil
}

// Volume mount/unmount operations

// MountVolume mounts an EFS volume to an instance
func (c *TUIClient) MountVolume(ctx context.Context, volumeName, instanceName, mountPoint string) error {
	return c.client.MountVolume(ctx, volumeName, instanceName, mountPoint)
}

// UnmountVolume unmounts an EFS volume from an instance
func (c *TUIClient) UnmountVolume(ctx context.Context, volumeName, instanceName string) error {
	return c.client.UnmountVolume(ctx, volumeName, instanceName)
}

// Project operations (Phase 4 Enterprise)

// ListProjects returns all projects with optional filtering
func (c *TUIClient) ListProjects(ctx context.Context, filter *ProjectFilter) (*ListProjectsResponse, error) {
	// The underlying client's ListProjects method will be called
	// For now, return empty list if method doesn't exist yet
	// This allows TUI to compile and we'll implement the backend later
	return &ListProjectsResponse{
		Projects: []ProjectResponse{},
	}, nil
}

// Policy operations (Phase 5A+)

// GetPolicyStatus returns the current policy enforcement status
func (c *TUIClient) GetPolicyStatus(ctx context.Context) (*PolicyStatusResponse, error) {
	// For now, return default disabled status
	// Backend integration will provide real data
	return &PolicyStatusResponse{
		Enabled:          false,
		AssignedPolicies: []string{},
		Message:          "Policy enforcement is currently disabled (default allow)",
		StatusIcon:       "⚪",
	}, nil
}

// ListPolicySets returns all available policy sets
func (c *TUIClient) ListPolicySets(ctx context.Context) (*ListPolicySetsResponse, error) {
	// For now, return sample policy sets
	// Backend integration will provide real data
	return &ListPolicySetsResponse{
		PolicySets: []PolicySetResponse{
			{
				ID:          "student",
				Description: "Student access - basic templates only",
				PolicyCount: 3,
				Status:      "Available",
			},
			{
				ID:          "researcher",
				Description: "Researcher access - all research templates",
				PolicyCount: 5,
				Status:      "Available",
			},
			{
				ID:          "admin",
				Description: "Admin access - full system access",
				PolicyCount: 10,
				Status:      "Available",
			},
		},
	}, nil
}

// AssignPolicySet assigns a policy set to the current user
func (c *TUIClient) AssignPolicySet(ctx context.Context, policySetID string) error {
	// Backend integration will handle actual assignment
	return nil
}

// SetPolicyEnforcement enables or disables policy enforcement
func (c *TUIClient) SetPolicyEnforcement(ctx context.Context, enabled bool) error {
	// Backend integration will handle enforcement toggle
	return nil
}

// CheckTemplateAccess checks if the user has access to a template
func (c *TUIClient) CheckTemplateAccess(ctx context.Context, templateName string) (*TemplateAccessResponse, error) {
	// For now, return allowed by default
	// Backend integration will provide real access control
	return &TemplateAccessResponse{
		Allowed:         true,
		TemplateName:    templateName,
		Reason:          "Policy enforcement disabled - default allow",
		MatchedPolicies: []string{},
		Suggestions:     []string{},
	}, nil
}

// Marketplace operations (Phase 5B)

// ListMarketplaceTemplates returns all marketplace templates with optional filtering
func (c *TUIClient) ListMarketplaceTemplates(ctx context.Context, filter *MarketplaceFilter) (*ListMarketplaceTemplatesResponse, error) {
	// For now, return sample templates
	// Backend integration will provide real marketplace data
	return &ListMarketplaceTemplatesResponse{
		Templates: []MarketplaceTemplateResponse{
			{
				Name:         "Python Data Science",
				Publisher:    "Community",
				Category:     "Data Science",
				Description:  "Complete Python data science environment with pandas, numpy, scikit-learn",
				Rating:       4.8,
				RatingCount:  156,
				Downloads:    2341,
				Verified:     true,
				Keywords:     []string{"python", "data-science", "ml", "pandas"},
				License:      "MIT",
				Registry:     "community",
				RegistryType: "community",
			},
			{
				Name:         "R Statistical Analysis",
				Publisher:    "Community",
				Category:     "Statistics",
				Description:  "R environment with tidyverse, ggplot2, and statistical packages",
				Rating:       4.6,
				RatingCount:  89,
				Downloads:    1523,
				Verified:     true,
				Keywords:     []string{"r", "statistics", "tidyverse", "ggplot2"},
				License:      "MIT",
				Registry:     "community",
				RegistryType: "community",
			},
			{
				Name:         "Deep Learning GPU",
				Publisher:    "Community",
				Category:     "Machine Learning",
				Description:  "PyTorch and TensorFlow with CUDA support for GPU acceleration",
				Rating:       4.9,
				RatingCount:  234,
				Downloads:    4567,
				Verified:     true,
				Keywords:     []string{"deep-learning", "gpu", "pytorch", "tensorflow"},
				License:      "Apache-2.0",
				Registry:     "community",
				RegistryType: "community",
			},
			{
				Name:         "Bioinformatics Toolkit",
				Publisher:    "Institutional",
				Category:     "Bioinformatics",
				Description:  "Comprehensive bioinformatics tools including BLAST, EMBOSS, and more",
				Rating:       4.7,
				RatingCount:  67,
				Downloads:    890,
				Verified:     false,
				Keywords:     []string{"bioinformatics", "genomics", "blast"},
				License:      "GPL-3.0",
				Registry:     "institutional",
				RegistryType: "institutional",
			},
			{
				Name:         "Web Development Stack",
				Publisher:    "Community",
				Category:     "Development",
				Description:  "Node.js, React, and modern web development tools",
				Rating:       4.5,
				RatingCount:  123,
				Downloads:    3456,
				Verified:     true,
				Keywords:     []string{"web", "nodejs", "react", "javascript"},
				License:      "MIT",
				Registry:     "community",
				RegistryType: "community",
			},
		},
	}, nil
}

// ListMarketplaceCategories returns all marketplace categories
func (c *TUIClient) ListMarketplaceCategories(ctx context.Context) (*ListCategoriesResponse, error) {
	// For now, return sample categories
	// Backend integration will provide real category data
	return &ListCategoriesResponse{
		Categories: []CategoryResponse{
			{
				Name:          "Data Science",
				Description:   "Python and R environments for data analysis and visualization",
				TemplateCount: 15,
			},
			{
				Name:          "Machine Learning",
				Description:   "Deep learning frameworks and GPU-accelerated environments",
				TemplateCount: 12,
			},
			{
				Name:          "Bioinformatics",
				Description:   "Genomics, proteomics, and computational biology tools",
				TemplateCount: 8,
			},
			{
				Name:          "Development",
				Description:   "General software development environments",
				TemplateCount: 20,
			},
			{
				Name:          "Statistics",
				Description:   "Statistical analysis and modeling environments",
				TemplateCount: 10,
			},
		},
	}, nil
}

// ListMarketplaceRegistries returns all configured template registries
func (c *TUIClient) ListMarketplaceRegistries(ctx context.Context) (*ListRegistriesResponse, error) {
	// For now, return sample registries
	// Backend integration will provide real registry data
	return &ListRegistriesResponse{
		Registries: []RegistryResponse{
			{
				Name:          "Community Registry",
				Type:          "community",
				URL:           "https://registry.cloudworkstation.org",
				TemplateCount: 65,
				Status:        "active",
			},
			{
				Name:          "Institutional Registry",
				Type:          "institutional",
				URL:           "https://institutional.cloudworkstation.org",
				TemplateCount: 23,
				Status:        "active",
			},
			{
				Name:          "Official Registry",
				Type:          "official",
				URL:           "https://official.cloudworkstation.org",
				TemplateCount: 10,
				Status:        "active",
			},
		},
	}, nil
}

// InstallMarketplaceTemplate installs a template from the marketplace
func (c *TUIClient) InstallMarketplaceTemplate(ctx context.Context, templateName string) error {
	// Backend integration will handle actual installation
	return nil
}

// AMI Management operations

// ListAMIs returns all available AMIs
func (c *TUIClient) ListAMIs(ctx context.Context) (*ListAMIsResponse, error) {
	// For now, return sample AMIs
	// Backend integration will provide real AMI data
	return &ListAMIsResponse{
		AMIs: []AMIResponse{
			{
				ID:           "ami-0123456789abcdef0",
				TemplateName: "python-ml",
				Region:       "us-west-2",
				State:        "available",
				Architecture: "x86_64",
				SizeGB:       12.5,
				Description:  "Python ML environment with TensorFlow and PyTorch",
				CreatedAt:    time.Now().Add(-7 * 24 * time.Hour),
			},
			{
				ID:           "ami-0abcdef123456789",
				TemplateName: "r-research",
				Region:       "us-west-2",
				State:        "available",
				Architecture: "arm64",
				SizeGB:       10.2,
				Description:  "R research environment with tidyverse",
				CreatedAt:    time.Now().Add(-14 * 24 * time.Hour),
			},
			{
				ID:           "ami-0fedcba987654321",
				TemplateName: "python-ml",
				Region:       "us-east-1",
				State:        "available",
				Architecture: "x86_64",
				SizeGB:       12.5,
				Description:  "Python ML environment with TensorFlow and PyTorch",
				CreatedAt:    time.Now().Add(-3 * 24 * time.Hour),
			},
		},
	}, nil
}

// ListAMIBuilds returns all AMI build jobs
func (c *TUIClient) ListAMIBuilds(ctx context.Context) (*ListAMIBuildsResponse, error) {
	// For now, return sample builds
	// Backend integration will provide real build data
	return &ListAMIBuildsResponse{
		Builds: []AMIBuildResponse{
			{
				ID:           "build-0123456789",
				TemplateName: "deep-learning-gpu",
				Status:       "in_progress",
				Progress:     65,
				CurrentStep:  "Installing CUDA drivers",
				StartedAt:    time.Now().Add(-25 * time.Minute),
			},
			{
				ID:           "build-0987654321",
				TemplateName: "web-development",
				Status:       "completed",
				Progress:     100,
				StartedAt:    time.Now().Add(-2 * time.Hour),
			},
		},
	}, nil
}

// ListAMIRegions returns AMI regional coverage
func (c *TUIClient) ListAMIRegions(ctx context.Context) (*ListAMIRegionsResponse, error) {
	// For now, return sample regions
	// Backend integration will provide real regional data
	return &ListAMIRegionsResponse{
		Regions: []AMIRegionResponse{
			{Name: "us-east-1", AMICount: 5},
			{Name: "us-east-2", AMICount: 3},
			{Name: "us-west-1", AMICount: 2},
			{Name: "us-west-2", AMICount: 8},
			{Name: "eu-west-1", AMICount: 4},
			{Name: "eu-central-1", AMICount: 3},
			{Name: "ap-northeast-1", AMICount: 2},
			{Name: "ap-southeast-1", AMICount: 1},
		},
	}, nil
}

// DeleteAMI deletes an AMI
func (c *TUIClient) DeleteAMI(ctx context.Context, amiID string) error {
	// Backend integration will handle actual deletion
	return nil
}

// Rightsizing operations

// GetRightsizingRecommendations returns rightsizing recommendations for all instances
func (c *TUIClient) GetRightsizingRecommendations(ctx context.Context) (*GetRightsizingRecommendationsResponse, error) {
	// For now, return sample recommendations
	// Backend integration will provide real recommendations based on CloudWatch metrics
	return &GetRightsizingRecommendationsResponse{
		Recommendations: []RightsizingRecommendation{
			{
				InstanceName:      "ml-workstation",
				CurrentType:       "m5.2xlarge",
				RecommendedType:   "m5.xlarge",
				CPUUtilization:    25.3,
				MemoryUtilization: 35.7,
				CurrentCost:       292.80,
				RecommendedCost:   146.40,
				MonthlySavings:    146.40,
				SavingsPercentage: 50.0,
				Confidence:        "high",
				Reason:            "Consistent low CPU and memory utilization over 30 days",
			},
			{
				InstanceName:      "data-analysis",
				CurrentType:       "r5.4xlarge",
				RecommendedType:   "r5.2xlarge",
				CPUUtilization:    18.5,
				MemoryUtilization: 42.3,
				CurrentCost:       604.80,
				RecommendedCost:   302.40,
				MonthlySavings:    302.40,
				SavingsPercentage: 50.0,
				Confidence:        "high",
				Reason:            "Memory-optimized instance underutilized, can safely downsize",
			},
			{
				InstanceName:      "gpu-training",
				CurrentType:       "p3.2xlarge",
				RecommendedType:   "g4dn.xlarge",
				CPUUtilization:    45.2,
				MemoryUtilization: 55.8,
				CurrentCost:       2196.00,
				RecommendedCost:   394.20,
				MonthlySavings:    1801.80,
				SavingsPercentage: 82.0,
				Confidence:        "medium",
				Reason:            "GPU usage patterns suggest g4dn instance type is sufficient",
			},
		},
	}, nil
}

// ApplyRightsizingRecommendation applies a rightsizing recommendation
func (c *TUIClient) ApplyRightsizingRecommendation(ctx context.Context, instanceName string) error {
	// Backend integration will handle actual resize operation
	return nil
}

// Logs operations

// GetLogs retrieves logs for an instance
func (c *TUIClient) GetLogs(ctx context.Context, instanceName, logType string) (*LogsResponse, error) {
	// For now, return sample logs
	// Backend integration will fetch real logs from CloudWatch or EC2
	sampleLogs := []string{
		"[2025-10-07 10:23:45] Instance starting...",
		"[2025-10-07 10:23:46] Loading kernel modules",
		"[2025-10-07 10:23:47] Mounting filesystems",
		"[2025-10-07 10:23:48] Starting network services",
		"[2025-10-07 10:23:49] Applying cloud-init configuration",
		"[2025-10-07 10:23:50] Installing packages: python3, pip, numpy",
		"[2025-10-07 10:24:15] Package installation complete",
		"[2025-10-07 10:24:16] Running post-install scripts",
		"[2025-10-07 10:24:20] CloudWorkstation initialization complete",
		"[2025-10-07 10:24:21] Instance ready for SSH connections",
	}

	if logType == "cloud-init" {
		sampleLogs = []string{
			"Cloud-init v. 23.1.2 running 'init' at Mon, 07 Oct 2025 10:23:45 +0000",
			"Cloud-init v. 23.1.2 running 'modules:config' at Mon, 07 Oct 2025 10:23:48 +0000",
			"Running module package-update-upgrade-install",
			"Running module runcmd",
			"Running module scripts-user",
			"Cloud-init v. 23.1.2 running 'modules:final' at Mon, 07 Oct 2025 10:24:15 +0000",
			"Cloud-init v. 23.1.2 finished at Mon, 07 Oct 2025 10:24:20 +0000. DataSource: DataSourceEc2",
		}
	}

	return &LogsResponse{
		Lines: sampleLogs,
	}, nil
}
