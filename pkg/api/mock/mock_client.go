// Package mock provides mock implementations for CloudWorkstation API.
//
// This package contains mock implementations of the CloudWorkstation API interfaces
// for use in demos, testing, and development without requiring actual AWS credentials
// or resources. It simulates all API responses with realistic data.
//
// Usage:
//
//	// Create a mock client for demos or tests
//	client := mock.NewClient()
//	instances, err := client.ListInstances()
package mock

import (
	"fmt"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// MockClient provides a mock implementation of the CloudWorkstationAPI interface
type MockClient struct {
	Templates map[string]types.Template
	Instances map[string]types.Instance
	Volumes   map[string]types.EFSVolume
	Storage   map[string]types.EBSVolume
}

// Ensure MockClient implements CloudWorkstationAPI
var _ api.CloudWorkstationAPI = (*MockClient)(nil)

// NewClient creates a new mock client with pre-populated data
func NewClient() *MockClient {
	return &MockClient{
		Templates: loadMockTemplates(),
		Instances: loadMockInstances(),
		Volumes:   loadMockVolumes(),
		Storage:   loadMockStorage(),
	}
}

// loadMockTemplates creates realistic template data for demo purposes
func loadMockTemplates() map[string]types.Template {
	return map[string]types.Template{
		"basic-ubuntu": {
			Name:        "basic-ubuntu",
			Description: "Base Ubuntu 22.04 for general use",
			AMI: map[string]map[string]string{
				"us-east-1": {
					"x86_64": "ami-0123456789abcdef0",
					"arm64":  "ami-0abcdef0123456789",
				},
			},
			InstanceType: map[string]string{
				"x86_64": "t3.medium",
				"arm64":  "t4g.medium",
			},
			Ports: []int{22, 80, 443},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.0416,
				"arm64":  0.0336,
			},
		},
		"r-research": {
			Name:        "r-research",
			Description: "R and RStudio Server with common packages",
			AMI: map[string]map[string]string{
				"us-east-1": {
					"x86_64": "ami-0abcdef0123456789",
				},
			},
			InstanceType: map[string]string{
				"x86_64": "r5.large",
			},
			Ports: []int{22, 80, 443, 8787},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.126,
			},
		},
		"python-ml": {
			Name:        "python-ml",
			Description: "Python with ML frameworks and Jupyter",
			AMI: map[string]map[string]string{
				"us-east-1": {
					"x86_64": "ami-0123456789abcdef0",
					"arm64":  "ami-0abcdef0123456789",
				},
			},
			InstanceType: map[string]string{
				"x86_64": "g4dn.xlarge",
				"arm64":  "g5g.xlarge",
			},
			Ports: []int{22, 80, 443, 8888},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.526,
				"arm64":  0.42,
			},
		},
		"desktop-research": {
			Name:        "desktop-research",
			Description: "Ubuntu Desktop with research GUI applications",
			AMI: map[string]map[string]string{
				"us-east-1": {
					"x86_64": "ami-0abcdef0123456789",
				},
			},
			InstanceType: map[string]string{
				"x86_64": "t3.xlarge",
			},
			Ports: []int{22, 80, 443, 8443},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.1664,
			},
		},
		"data-science": {
			Name:        "data-science",
			Description: "Complete data science environment with R and Python",
			AMI: map[string]map[string]string{
				"us-east-1": {
					"x86_64": "ami-0123456789abcdef0",
				},
			},
			InstanceType: map[string]string{
				"x86_64": "r5.2xlarge",
			},
			Ports: []int{22, 80, 443, 8787, 8888},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.504,
			},
		},
	}
}

// loadMockInstances creates realistic instance data for demo purposes
func loadMockInstances() map[string]types.Instance {
	return map[string]types.Instance{
		"my-analysis": {
			ID:                 "i-0123456789abcdef0",
			Name:               "my-analysis",
			Template:           "r-research",
			PublicIP:           "54.84.123.45",
			PrivateIP:          "172.31.16.25",
			State:              "running",
			LaunchTime:         time.Now().Add(-24 * time.Hour),
			EstimatedDailyCost: 3.024,
			AttachedVolumes:    []string{"shared-data"},
		},
		"ml-training": {
			ID:                 "i-0abcdef0123456789",
			Name:               "ml-training",
			Template:           "python-ml",
			PublicIP:           "54.86.234.56",
			PrivateIP:          "172.31.32.67",
			State:              "stopped",
			LaunchTime:         time.Now().Add(-72 * time.Hour),
			EstimatedDailyCost: 12.624,
			AttachedEBSVolumes: []string{"training-data"},
		},
	}
}

// loadMockVolumes creates realistic EFS volume data
func loadMockVolumes() map[string]types.EFSVolume {
	return map[string]types.EFSVolume{
		"shared-data": {
			Name:            "shared-data",
			FileSystemId:    "fs-0123456789abcdef0",
			Region:          "us-east-1",
			CreationTime:    time.Now().Add(-30 * 24 * time.Hour),
			MountTargets:    []string{"fsmt-0123456789abcdef0"},
			State:           "available",
			PerformanceMode: "generalPurpose",
			ThroughputMode:  "bursting",
			EstimatedCostGB: 0.30,
			SizeBytes:       int64(10 * 1024 * 1024 * 1024), // 10 GB
		},
		"research-data": {
			Name:            "research-data",
			FileSystemId:    "fs-0abcdef0123456789",
			Region:          "us-east-1",
			CreationTime:    time.Now().Add(-15 * 24 * time.Hour),
			MountTargets:    []string{"fsmt-0abcdef0123456789"},
			State:           "available",
			PerformanceMode: "maxIO",
			ThroughputMode:  "provisioned",
			EstimatedCostGB: 0.33,
			SizeBytes:       int64(50 * 1024 * 1024 * 1024), // 50 GB
		},
	}
}

// loadMockStorage creates realistic EBS volume data
func loadMockStorage() map[string]types.EBSVolume {
	return map[string]types.EBSVolume{
		"training-data": {
			Name:            "training-data",
			VolumeID:        "vol-0123456789abcdef0",
			Region:          "us-east-1",
			CreationTime:    time.Now().Add(-45 * 24 * time.Hour),
			State:           "available",
			VolumeType:      "gp3",
			SizeGB:          1000,
			IOPS:            3000,
			Throughput:      125,
			EstimatedCostGB: 0.08,
			AttachedTo:      "ml-training",
		},
		"analysis-storage": {
			Name:            "analysis-storage",
			VolumeID:        "vol-0abcdef0123456789",
			Region:          "us-east-1",
			CreationTime:    time.Now().Add(-10 * 24 * time.Hour),
			State:           "available",
			VolumeType:      "gp3",
			SizeGB:          500,
			IOPS:            3000,
			Throughput:      125,
			EstimatedCostGB: 0.08,
			AttachedTo:      "",
		},
	}
}

// LaunchInstance simulates launching a new instance
func (m *MockClient) LaunchInstance(req types.LaunchRequest) (*types.LaunchResponse, error) {
	// Validate template exists
	if _, exists := m.Templates[req.Template]; !exists {
		return nil, fmt.Errorf("template not found: %s", req.Template)
	}
	
	// Create a unique instance ID
	instanceID := fmt.Sprintf("i-%s-%d", req.Name, time.Now().Unix())
	publicIP := "54.84.123.45" // Mock IP
	
	// Select default cost based on template
var costPerHour float64
	
	template := m.Templates[req.Template]
	if template.InstanceType["arm64"] != "" {
		// Prefer ARM for cost savings
		costPerHour = template.EstimatedCostPerHour["arm64"]
	} else {
		costPerHour = template.EstimatedCostPerHour["x86_64"]
	}
	
	// Apply size adjustments if specified
	if req.Size != "" {
		switch req.Size {
		case "XS":
			costPerHour *= 0.5
		case "S":
			costPerHour *= 0.75
		case "M":
			// Base size, no adjustment
		case "L":
			costPerHour *= 2.0
		case "XL":
			costPerHour *= 4.0
		case "GPU-S":
			costPerHour = 0.75
		case "GPU-M":
			costPerHour = 1.5
		case "GPU-L":
			costPerHour = 3.0
		}
	}
	
	// Apply spot discount if requested
	if req.Spot {
		costPerHour *= 0.3 // 70% discount for spot
	}
	
	// Calculate daily cost
	dailyCost := costPerHour * 24
	
	// Create instance
	instance := types.Instance{
		ID:                 instanceID,
		Name:               req.Name,
		Template:           req.Template,
		State:              "running",
		LaunchTime:         time.Now(),
		PublicIP:           publicIP,
		PrivateIP:          "172.31.16." + fmt.Sprint(time.Now().Second()),
		EstimatedDailyCost: dailyCost,
	}
	
	// Add volumes if specified
	if len(req.Volumes) > 0 {
		instance.AttachedVolumes = req.Volumes
	}
	
	// Add EBS volumes if specified
	if len(req.EBSVolumes) > 0 {
		instance.AttachedEBSVolumes = req.EBSVolumes
	}
	
	// Store instance if not dry run
	if !req.DryRun {
		m.Instances[req.Name] = instance
	}
	
	// Create connection info based on template
	var connectionInfo string
	switch req.Template {
	case "r-research":
		connectionInfo = fmt.Sprintf("RStudio Server: http://%s:8787", publicIP)
	case "python-ml":
		connectionInfo = fmt.Sprintf("JupyterLab: http://%s:8888", publicIP)
	case "desktop-research":
		connectionInfo = fmt.Sprintf("NICE DCV: https://%s:8443", publicIP)
	default:
		connectionInfo = fmt.Sprintf("ssh ubuntu@%s", publicIP)
	}
	
	// Build response
	resp := &types.LaunchResponse{
		Instance:       instance,
		Message:        fmt.Sprintf("Successfully launched %s instance '%s'", req.Template, req.Name),
		EstimatedCost:  fmt.Sprintf("$%.2f/day", dailyCost),
		ConnectionInfo: connectionInfo,
	}
	
	// Simulate delay for more realistic response
	time.Sleep(500 * time.Millisecond)
	
	return resp, nil
}

// ListInstances returns mock instances
func (m *MockClient) ListInstances() (*types.ListResponse, error) {
	instances := make([]types.Instance, 0, len(m.Instances))
	totalCost := 0.0
	
	for _, instance := range m.Instances {
		instances = append(instances, instance)
		if instance.State == "running" {
			totalCost += instance.EstimatedDailyCost
		}
	}
	
	return &types.ListResponse{
		Instances: instances,
		TotalCost: totalCost,
	}, nil
}

// GetInstance returns a specific instance
func (m *MockClient) GetInstance(name string) (*types.Instance, error) {
	if instance, exists := m.Instances[name]; exists {
		return &instance, nil
	}
	return nil, fmt.Errorf("instance not found: %s", name)
}

// DeleteInstance simulates deleting an instance
func (m *MockClient) DeleteInstance(name string) error {
	if _, exists := m.Instances[name]; !exists {
		return fmt.Errorf("instance not found: %s", name)
	}
	
	delete(m.Instances, name)
	return nil
}

// StopInstance simulates stopping an instance
func (m *MockClient) StopInstance(name string) error {
	if instance, exists := m.Instances[name]; exists {
		instance.State = "stopped"
		m.Instances[name] = instance
		return nil
	}
	return fmt.Errorf("instance not found: %s", name)
}

// StartInstance simulates starting an instance
func (m *MockClient) StartInstance(name string) error {
	if instance, exists := m.Instances[name]; exists {
		instance.State = "running"
		m.Instances[name] = instance
		return nil
	}
	return fmt.Errorf("instance not found: %s", name)
}

// ConnectInstance returns mock connection information
func (m *MockClient) ConnectInstance(name string) (string, error) {
	if instance, exists := m.Instances[name]; exists {
		switch m.Templates[instance.Template].Name {
		case "r-research":
			return fmt.Sprintf("RStudio Server: http://%s:8787 (username: rstudio, password: cloudworkstation)", instance.PublicIP), nil
		case "python-ml":
			return fmt.Sprintf("JupyterLab: http://%s:8888 (token: cloudworkstation)", instance.PublicIP), nil
		case "desktop-research":
			return fmt.Sprintf("NICE DCV: https://%s:8443 (username: ubuntu, password: cloudworkstation)", instance.PublicIP), nil
		default:
			return fmt.Sprintf("ssh ubuntu@%s", instance.PublicIP), nil
		}
	}
	return "", fmt.Errorf("instance not found: %s", name)
}

// ListTemplates returns mock templates
func (m *MockClient) ListTemplates() (map[string]types.Template, error) {
	return m.Templates, nil
}

// GetTemplate returns a specific template
func (m *MockClient) GetTemplate(name string) (*types.Template, error) {
	if template, exists := m.Templates[name]; exists {
		return &template, nil
	}
	return nil, fmt.Errorf("template not found: %s", name)
}

// CreateVolume simulates creating an EFS volume
func (m *MockClient) CreateVolume(req types.VolumeCreateRequest) (*types.EFSVolume, error) {
	if _, exists := m.Volumes[req.Name]; exists {
		return nil, fmt.Errorf("volume already exists: %s", req.Name)
	}
	
	// Create mock volume with realistic values
	volume := types.EFSVolume{
		Name:            req.Name,
		FileSystemId:    fmt.Sprintf("fs-%s-%d", req.Name, time.Now().Unix()),
		Region:          req.Region,
		CreationTime:    time.Now(),
		MountTargets:    []string{fmt.Sprintf("fsmt-%s-%d", req.Name, time.Now().Unix())},
		State:           "available",
		PerformanceMode: req.PerformanceMode,
		ThroughputMode:  req.ThroughputMode,
		EstimatedCostGB: 0.30, // $0.30/GB is typical EFS cost
		SizeBytes:       0,    // New volume starts empty
	}
	
	// Use defaults if not specified
	if volume.PerformanceMode == "" {
		volume.PerformanceMode = "generalPurpose"
	}
	
	if volume.ThroughputMode == "" {
		volume.ThroughputMode = "bursting"
	}
	
	if volume.Region == "" {
		volume.Region = "us-east-1"
	}
	
	// Store volume
	m.Volumes[req.Name] = volume
	
	// Simulate delay
	time.Sleep(300 * time.Millisecond)
	
	return &volume, nil
}

// ListVolumes returns all mock EFS volumes
func (m *MockClient) ListVolumes() (map[string]types.EFSVolume, error) {
	return m.Volumes, nil
}

// GetVolume returns a specific EFS volume
func (m *MockClient) GetVolume(name string) (*types.EFSVolume, error) {
	if volume, exists := m.Volumes[name]; exists {
		return &volume, nil
	}
	return nil, fmt.Errorf("volume not found: %s", name)
}

// DeleteVolume simulates deleting an EFS volume
func (m *MockClient) DeleteVolume(name string) error {
	if _, exists := m.Volumes[name]; !exists {
		return fmt.Errorf("volume not found: %s", name)
	}
	
	delete(m.Volumes, name)
	return nil
}

// CreateStorage simulates creating an EBS volume
func (m *MockClient) CreateStorage(req types.StorageCreateRequest) (*types.EBSVolume, error) {
	if _, exists := m.Storage[req.Name]; exists {
		return nil, fmt.Errorf("storage volume already exists: %s", req.Name)
	}
	
	// Determine size based on t-shirt sizing
	sizeGB := int32(100) // default XS
	
	switch req.Size {
	case "XS":
		sizeGB = 100
	case "S":
		sizeGB = 250
	case "M":
		sizeGB = 500
	case "L":
		sizeGB = 1000
	case "XL":
		sizeGB = 2000
	case "XXL":
		sizeGB = 4000
	default:
		// Try to parse as specific GB size
		var size int
		if _, err := fmt.Sscanf(req.Size, "%d", &size); err == nil {
			sizeGB = int32(size)
		}
	}
	
	// Set defaults for volume type
	volumeType := req.VolumeType
	if volumeType == "" {
		volumeType = "gp3"
	}
	
	// Set IOPS and throughput based on volume type
	var iops, throughput int32
	
	switch volumeType {
	case "gp3":
		iops = 3000 // Default for gp3
		throughput = 125
	case "io2":
		iops = 16000
		throughput = 500
	}
	
	// Calculate cost based on volume type and size
	var costPerGB float64
	
	switch volumeType {
	case "gp3":
		costPerGB = 0.08
	case "io2":
		costPerGB = 0.125
	default:
		costPerGB = 0.10
	}
	
	// Create storage volume
	volume := types.EBSVolume{
		Name:            req.Name,
		VolumeID:        fmt.Sprintf("vol-%s-%d", req.Name, time.Now().Unix()),
		Region:          req.Region,
		CreationTime:    time.Now(),
		State:           "available",
		VolumeType:      volumeType,
		SizeGB:          sizeGB,
		IOPS:            iops,
		Throughput:      throughput,
		EstimatedCostGB: costPerGB,
	}
	
	if volume.Region == "" {
		volume.Region = "us-east-1"
	}
	
	// Store volume
	m.Storage[req.Name] = volume
	
	// Simulate delay
	time.Sleep(500 * time.Millisecond)
	
	return &volume, nil
}

// ListStorage returns all mock EBS volumes
func (m *MockClient) ListStorage() (map[string]types.EBSVolume, error) {
	return m.Storage, nil
}

// GetStorage returns a specific EBS volume
func (m *MockClient) GetStorage(name string) (*types.EBSVolume, error) {
	if volume, exists := m.Storage[name]; exists {
		return &volume, nil
	}
	return nil, fmt.Errorf("storage volume not found: %s", name)
}

// DeleteStorage simulates deleting an EBS volume
func (m *MockClient) DeleteStorage(name string) error {
	if _, exists := m.Storage[name]; !exists {
		return fmt.Errorf("storage volume not found: %s", name)
	}
	
	delete(m.Storage, name)
	return nil
}

// AttachStorage simulates attaching an EBS volume to an instance
func (m *MockClient) AttachStorage(ctx context.Context, volumeName, instanceName string) error {
	volume, exists := m.Storage[volumeName]
	if !exists {
		return fmt.Errorf("storage volume not found: %s", volumeName)
	}
	
	if _, exists := m.Instances[instanceName]; !exists {
		return fmt.Errorf("instance not found: %s", instanceName)
	}
	
	// Update volume attachment
	volume.AttachedTo = instanceName
	volume.State = "in-use"
	m.Storage[volumeName] = volume
	
	// Update instance
	instance := m.Instances[instanceName]
	instance.AttachedEBSVolumes = append(instance.AttachedEBSVolumes, volumeName)
	m.Instances[instanceName] = instance
	
	return nil
}

// DetachStorage simulates detaching an EBS volume
func (m *MockClient) DetachStorage(volumeName string) error {
	volume, exists := m.Storage[volumeName]
	if !exists {
		return fmt.Errorf("storage volume not found: %s", volumeName)
	}
	
	// Skip if not attached
	if volume.AttachedTo == "" {
		return nil
	}
	
	// Update instance if it exists
	if instance, exists := m.Instances[volume.AttachedTo]; exists {
		// Remove volume from instance
		updatedVolumes := []string{}
		for _, v := range instance.AttachedEBSVolumes {
			if v != volumeName {
				updatedVolumes = append(updatedVolumes, v)
			}
		}
		instance.AttachedEBSVolumes = updatedVolumes
		m.Instances[volume.AttachedTo] = instance
	}
	
	// Update volume
	volume.AttachedTo = ""
	volume.State = "available"
	m.Storage[volumeName] = volume
	
	return nil
}

// GetStatus returns mock daemon status
func (m *MockClient) GetStatus() (*types.DaemonStatus, error) {
	return &types.DaemonStatus{
		Version:       "0.1.0",
		Status:        "running",
		StartTime:     time.Now().Add(-24 * time.Hour),
		ActiveOps:     0,
		TotalRequests: 42,
		ErrorCount:    2,
		AWSRegion:     "us-east-1",
	}, nil
}

// Ping simulates a health check
func (m *MockClient) Ping() error {
	return nil
}