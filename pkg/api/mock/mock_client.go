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
	"context"
	"fmt"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/idle"
	"github.com/scttfrdmn/cloudworkstation/pkg/project"
	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
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
var _ client.CloudWorkstationAPI = (*MockClient)(nil)

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
			ID:              "i-0123456789abcdef0",
			Name:            "my-analysis",
			Template:        "r-research",
			PublicIP:        "54.84.123.45",
			PrivateIP:       "172.31.16.25",
			State:           "running",
			LaunchTime:      time.Now().Add(-24 * time.Hour),
			HourlyRate:      0.126,
			CurrentSpend:    3.024,
			AttachedVolumes: []string{"shared-data"},
		},
		"ml-training": {
			ID:                 "i-0abcdef0123456789",
			Name:               "ml-training",
			Template:           "python-ml",
			PublicIP:           "54.86.234.56",
			PrivateIP:          "172.31.32.67",
			State:              "stopped",
			LaunchTime:         time.Now().Add(-72 * time.Hour),
			HourlyRate:         0.526,
			CurrentSpend:       12.624,
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
func (m *MockClient) LaunchInstance(ctx context.Context, req types.LaunchRequest) (*types.LaunchResponse, error) {
	if err := m.validateLaunchRequest(req); err != nil {
		return nil, err
	}

	instanceID := fmt.Sprintf("i-%s-%d", req.Name, time.Now().Unix())
	publicIP := "54.84.123.45" // Mock IP

	costPerHour := m.calculateInstanceCost(req)
	instance := m.buildInstance(req, instanceID, publicIP, costPerHour)

	if !req.DryRun {
		m.Instances[req.Name] = instance
	}

	resp := m.buildLaunchResponse(instance, req, costPerHour, publicIP)

	// Simulate delay for more realistic response
	time.Sleep(500 * time.Millisecond)

	return resp, nil
}

func (m *MockClient) validateLaunchRequest(req types.LaunchRequest) error {
	if _, exists := m.Templates[req.Template]; !exists {
		return fmt.Errorf("template not found: %s", req.Template)
	}
	return nil
}

func (m *MockClient) calculateInstanceCost(req types.LaunchRequest) float64 {
	template := m.Templates[req.Template]
	costPerHour := m.getBaseCost(template)
	costPerHour = m.applySizeAdjustments(costPerHour, req.Size)

	if req.Spot {
		costPerHour *= 0.3 // 70% discount for spot
	}

	return costPerHour
}

func (m *MockClient) getBaseCost(template types.Template) float64 {
	if template.InstanceType["arm64"] != "" {
		// Prefer ARM for cost savings
		return template.EstimatedCostPerHour["arm64"]
	}
	return template.EstimatedCostPerHour["x86_64"]
}

func (m *MockClient) applySizeAdjustments(baseCost float64, size string) float64 {
	sizeMultipliers := map[string]float64{
		"XS":    0.5,
		"S":     0.75,
		"M":     1.0, // Base size, no adjustment
		"L":     2.0,
		"XL":    4.0,
		"GPU-S": 0.75, // Absolute cost for GPU instances
		"GPU-M": 1.5,
		"GPU-L": 3.0,
	}

	if multiplier, exists := sizeMultipliers[size]; exists {
		if size == "GPU-S" || size == "GPU-M" || size == "GPU-L" {
			return multiplier // Absolute cost for GPU
		}
		return baseCost * multiplier
	}

	return baseCost
}

func (m *MockClient) buildInstance(req types.LaunchRequest, instanceID, publicIP string, costPerHour float64) types.Instance {
	dailyCost := costPerHour * 24

	instance := types.Instance{
		ID:           instanceID,
		Name:         req.Name,
		Template:     req.Template,
		State:        "running",
		LaunchTime:   time.Now(),
		PublicIP:     publicIP,
		PrivateIP:    "172.31.16." + fmt.Sprint(time.Now().Second()),
		HourlyRate:   dailyCost / 24.0,
		CurrentSpend: dailyCost,
	}

	if len(req.Volumes) > 0 {
		instance.AttachedVolumes = req.Volumes
	}

	if len(req.EBSVolumes) > 0 {
		instance.AttachedEBSVolumes = req.EBSVolumes
	}

	return instance
}

func (m *MockClient) buildLaunchResponse(instance types.Instance, req types.LaunchRequest, costPerHour float64, publicIP string) *types.LaunchResponse {
	dailyCost := costPerHour * 24
	connectionInfo := m.getConnectionInfo(req.Template, publicIP)

	return &types.LaunchResponse{
		Instance:       instance,
		Message:        fmt.Sprintf("Successfully launched %s instance '%s'", req.Template, req.Name),
		EstimatedCost:  fmt.Sprintf("$%.2f/day", dailyCost),
		ConnectionInfo: connectionInfo,
	}
}

func (m *MockClient) getConnectionInfo(template, publicIP string) string {
	connectionInfoMap := map[string]string{
		"r-research":       fmt.Sprintf("RStudio Server: http://%s:8787", publicIP),
		"python-ml":        fmt.Sprintf("JupyterLab: http://%s:8888", publicIP),
		"desktop-research": fmt.Sprintf("NICE DCV: https://%s:8443", publicIP),
	}

	if info, exists := connectionInfoMap[template]; exists {
		return info
	}

	return fmt.Sprintf("ssh ubuntu@%s", publicIP)
}

// ListInstances returns mock instances
func (m *MockClient) ListInstances(ctx context.Context) (*types.ListResponse, error) {
	instances := make([]types.Instance, 0, len(m.Instances))
	totalCost := 0.0

	for _, instance := range m.Instances {
		instances = append(instances, instance)
		if instance.State == "running" {
			totalCost += instance.CurrentSpend
		}
	}

	return &types.ListResponse{
		Instances: instances,
		TotalCost: totalCost,
	}, nil
}

// GetInstance returns a specific instance
func (m *MockClient) GetInstance(ctx context.Context, name string) (*types.Instance, error) {
	if instance, exists := m.Instances[name]; exists {
		return &instance, nil
	}
	return nil, fmt.Errorf("instance not found: %s", name)
}

// DeleteInstance simulates deleting an instance
func (m *MockClient) DeleteInstance(ctx context.Context, name string) error {
	if _, exists := m.Instances[name]; !exists {
		return fmt.Errorf("instance not found: %s", name)
	}

	delete(m.Instances, name)
	return nil
}

// StopInstance simulates stopping an instance
func (m *MockClient) StopInstance(ctx context.Context, name string) error {
	if instance, exists := m.Instances[name]; exists {
		instance.State = "stopped"
		m.Instances[name] = instance
		return nil
	}
	return fmt.Errorf("instance not found: %s", name)
}

// StartInstance simulates starting an instance
func (m *MockClient) StartInstance(ctx context.Context, name string) error {
	if instance, exists := m.Instances[name]; exists {
		instance.State = "running"
		m.Instances[name] = instance
		return nil
	}
	return fmt.Errorf("instance not found: %s", name)
}

// HibernateInstance simulates hibernating an instance
func (m *MockClient) HibernateInstance(ctx context.Context, name string) error {
	instance, exists := m.Instances[name]
	if !exists {
		return fmt.Errorf("instance not found: %s", name)
	}

	instance.State = "hibernated"
	m.Instances[name] = instance
	return nil
}

// ResumeInstance simulates resuming an instance from hibernation
func (m *MockClient) ResumeInstance(ctx context.Context, name string) error {
	instance, exists := m.Instances[name]
	if !exists {
		return fmt.Errorf("instance not found: %s", name)
	}

	instance.State = "running"
	m.Instances[name] = instance
	return nil
}

// GetInstanceHibernationStatus returns hibernation status (mock)
func (m *MockClient) GetInstanceHibernationStatus(ctx context.Context, name string) (*types.HibernationStatus, error) {
	instance, exists := m.Instances[name]
	if !exists {
		return nil, fmt.Errorf("instance not found: %s", name)
	}

	return &types.HibernationStatus{
		HibernationSupported: true,
		IsHibernated:         instance.State == "hibernated",
		InstanceName:         name,
	}, nil
}

// ConnectInstance returns mock connection information (deprecated, use GetInstance instead)
func (m *MockClient) ConnectInstance(ctx context.Context, name string) (string, error) {
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
func (m *MockClient) ListTemplates(ctx context.Context) (map[string]types.Template, error) {
	return m.Templates, nil
}

// GetTemplate returns a specific template
func (m *MockClient) GetTemplate(ctx context.Context, name string) (*types.Template, error) {
	if template, exists := m.Templates[name]; exists {
		return &template, nil
	}
	return nil, fmt.Errorf("template not found: %s", name)
}

// CreateVolume simulates creating an EFS volume
func (m *MockClient) CreateVolume(ctx context.Context, req types.VolumeCreateRequest) (*types.EFSVolume, error) {
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
func (m *MockClient) ListVolumes(ctx context.Context) ([]types.EFSVolume, error) {
	volumes := make([]types.EFSVolume, 0, len(m.Volumes))
	for _, volume := range m.Volumes {
		volumes = append(volumes, volume)
	}
	return volumes, nil
}

// GetVolume returns a specific EFS volume
func (m *MockClient) GetVolume(ctx context.Context, name string) (*types.EFSVolume, error) {
	if volume, exists := m.Volumes[name]; exists {
		return &volume, nil
	}
	return nil, fmt.Errorf("volume not found: %s", name)
}

// DeleteVolume simulates deleting an EFS volume
func (m *MockClient) DeleteVolume(ctx context.Context, name string) error {
	if _, exists := m.Volumes[name]; !exists {
		return fmt.Errorf("volume not found: %s", name)
	}

	delete(m.Volumes, name)
	return nil
}

// CreateStorage simulates creating an EBS volume
func (m *MockClient) CreateStorage(ctx context.Context, req types.StorageCreateRequest) (*types.EBSVolume, error) {
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
func (m *MockClient) ListStorage(ctx context.Context) ([]types.EBSVolume, error) {
	volumes := make([]types.EBSVolume, 0, len(m.Storage))
	for _, volume := range m.Storage {
		volumes = append(volumes, volume)
	}
	return volumes, nil
}

// GetStorage returns a specific EBS volume
func (m *MockClient) GetStorage(ctx context.Context, name string) (*types.EBSVolume, error) {
	if volume, exists := m.Storage[name]; exists {
		return &volume, nil
	}
	return nil, fmt.Errorf("storage volume not found: %s", name)
}

// DeleteStorage simulates deleting an EBS volume
func (m *MockClient) DeleteStorage(ctx context.Context, name string) error {
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
func (m *MockClient) DetachStorage(ctx context.Context, volumeName string) error {
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

// Status methods moved to bottom of file to avoid duplicates

// AttachVolume simulates attaching an EFS volume to an instance
func (m *MockClient) AttachVolume(ctx context.Context, volumeName, instanceName string) error {
	_, exists := m.Volumes[volumeName]
	if !exists {
		return fmt.Errorf("volume not found: %s", volumeName)
	}

	if _, exists := m.Instances[instanceName]; !exists {
		return fmt.Errorf("instance not found: %s", instanceName)
	}

	// Update instance
	instance := m.Instances[instanceName]
	instance.AttachedVolumes = append(instance.AttachedVolumes, volumeName)
	m.Instances[instanceName] = instance

	return nil
}

// DetachVolume simulates detaching an EFS volume from an instance
func (m *MockClient) DetachVolume(ctx context.Context, volumeName string) error {
	if _, exists := m.Volumes[volumeName]; !exists {
		return fmt.Errorf("volume not found: %s", volumeName)
	}

	// Update instances that have this volume attached
	for instName, instance := range m.Instances {
		updatedVolumes := []string{}
		for _, vol := range instance.AttachedVolumes {
			if vol != volumeName {
				updatedVolumes = append(updatedVolumes, vol)
			}
		}
		instance.AttachedVolumes = updatedVolumes
		m.Instances[instName] = instance
	}

	return nil
}

// GetRegistryStatus returns the status of the AMI registry
func (m *MockClient) GetRegistryStatus(ctx context.Context) (*client.RegistryStatusResponse, error) {
	return &client.RegistryStatusResponse{
		Active:        true,
		TemplateCount: 5,
		AMICount:      15,
		Status:        "operational",
	}, nil
}

// SetRegistryStatus enables or disables the AMI registry
func (m *MockClient) SetRegistryStatus(ctx context.Context, enabled bool) error {
	// In a mock client, we just return success
	return nil
}

// LookupAMI finds an AMI for a specific template in a region
func (m *MockClient) LookupAMI(ctx context.Context, templateName, region, arch string) (*client.AMIReferenceResponse, error) {
	template, exists := m.Templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	regionAMIs, exists := template.AMI[region]
	if !exists {
		return nil, fmt.Errorf("template not available in region %s", region)
	}

	amiID, exists := regionAMIs[arch]
	if !exists {
		return nil, fmt.Errorf("template not available for architecture %s in region %s", arch, region)
	}

	return &client.AMIReferenceResponse{
		AMIID:        amiID,
		Region:       region,
		Architecture: arch,
		TemplateName: templateName,
		Version:      "1.0.0",
		BuildDate:    time.Now().Add(-24 * time.Hour),
		Status:       "available",
	}, nil
}

// ListTemplateAMIs lists all AMIs available for a template across regions
func (m *MockClient) ListTemplateAMIs(ctx context.Context, templateName string) ([]client.AMIReferenceResponse, error) {
	template, exists := m.Templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	var result []client.AMIReferenceResponse

	for region, archMap := range template.AMI {
		for arch, amiID := range archMap {
			result = append(result, client.AMIReferenceResponse{
				AMIID:        amiID,
				Region:       region,
				Architecture: arch,
				TemplateName: templateName,
				Version:      "1.0.0",
				BuildDate:    time.Now().Add(-24 * time.Hour),
				Status:       "available",
			})
		}
	}

	return result, nil
}

// SetOptions sets client configuration options
func (m *MockClient) SetOptions(options client.Options) {
	// Mock client ignores options but implements the interface
}

// Idle detection operations - Mock implementations

// Legacy idle detection methods removed - using new hibernation policy system

// GetIdlePendingActions returns mock pending idle actions
func (m *MockClient) GetIdlePendingActions(ctx context.Context) ([]types.IdleState, error) {
	return []types.IdleState{}, nil
}

// ExecuteIdleActions executes pending idle actions (mock)
func (m *MockClient) ExecuteIdleActions(ctx context.Context) (*types.IdleExecutionResponse, error) {
	return &types.IdleExecutionResponse{
		Executed: 0,
		Errors:   []string{},
		Total:    0,
	}, nil
}

// GetIdleHistory returns mock idle history
func (m *MockClient) GetIdleHistory(ctx context.Context) ([]types.IdleHistoryEntry, error) {
	return []types.IdleHistoryEntry{}, nil
}

// Project management operations - Mock implementations

// CreateProject creates a new project (mock)
func (m *MockClient) CreateProject(ctx context.Context, req project.CreateProjectRequest) (*types.Project, error) {
	// Convert budget request to budget type
	var budget types.ProjectBudget
	if req.Budget != nil {
		budget.MonthlyLimit = req.Budget.MonthlyLimit
		budget.AlertThresholds = req.Budget.AlertThresholds
	}

	return &types.Project{
		ID:          "mock-project-123",
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now(),
		Budget:      &budget,
		Owner:       "mock-user",
	}, nil
}

// ListProjects lists projects (mock)
func (m *MockClient) ListProjects(ctx context.Context, filter *project.ProjectFilter) (*project.ProjectListResponse, error) {
	return &project.ProjectListResponse{
		Projects: []project.ProjectSummary{
			{
				ID:              "mock-project-123",
				Name:            "Mock Research Project",
				Owner:           "mock-user",
				Status:          "active",
				MemberCount:     1,
				ActiveInstances: 2,
				TotalCost:       150.25,
				CreatedAt:       time.Now().Add(-24 * time.Hour),
				LastActivity:    time.Now().Add(-1 * time.Hour),
			},
		},
		TotalCount:    1,
		FilteredCount: 1,
	}, nil
}

// GetProject gets a project by ID (mock)
func (m *MockClient) GetProject(ctx context.Context, projectID string) (*types.Project, error) {
	return &types.Project{
		ID:          projectID,
		Name:        "Mock Research Project",
		Description: "A sample project for testing",
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		Budget:      &types.ProjectBudget{MonthlyLimit: &[]float64{500.0}[0]},
		Owner:       "mock-user",
	}, nil
}

// UpdateProject updates a project (mock)
func (m *MockClient) UpdateProject(ctx context.Context, projectID string, req project.UpdateProjectRequest) (*types.Project, error) {
	name := "Mock Research Project"
	description := "A sample project for testing"

	if req.Name != nil {
		name = *req.Name
	}
	if req.Description != nil {
		description = *req.Description
	}

	return &types.Project{
		ID:          projectID,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		Budget:      &types.ProjectBudget{MonthlyLimit: &[]float64{500.0}[0]},
		Owner:       "mock-user",
	}, nil
}

// DeleteProject deletes a project (mock)
func (m *MockClient) DeleteProject(ctx context.Context, projectID string) error {
	return nil
}

// AddProjectMember adds a member to a project (mock)
func (m *MockClient) AddProjectMember(ctx context.Context, projectID string, req project.AddMemberRequest) error {
	return nil
}

// UpdateProjectMember updates a project member (mock)
func (m *MockClient) UpdateProjectMember(ctx context.Context, projectID, userID string, req project.UpdateMemberRequest) error {
	return nil
}

// RemoveProjectMember removes a member from a project (mock)
func (m *MockClient) RemoveProjectMember(ctx context.Context, projectID, userID string) error {
	return nil
}

// GetProjectMembers gets project members (mock)
func (m *MockClient) GetProjectMembers(ctx context.Context, projectID string) ([]types.ProjectMember, error) {
	return []types.ProjectMember{
		{
			UserID:  "mock-user",
			Role:    "owner",
			AddedAt: time.Now().Add(-24 * time.Hour),
		},
	}, nil
}

// GetProjectBudgetStatus gets project budget status (mock)
func (m *MockClient) GetProjectBudgetStatus(ctx context.Context, projectID string) (*project.BudgetStatus, error) {
	return &project.BudgetStatus{
		ProjectID:             projectID,
		BudgetEnabled:         true,
		TotalBudget:           1000.0,
		SpentAmount:           150.25,
		RemainingBudget:       849.75,
		SpentPercentage:       0.15,
		ProjectedMonthlySpend: 280.50,
		ActiveAlerts:          []string{},
		TriggeredActions:      []string{},
		LastUpdated:           time.Now(),
	}, nil
}

// GetProjectCostBreakdown gets project cost breakdown (mock)
func (m *MockClient) GetProjectCostBreakdown(ctx context.Context, projectID string, start, end time.Time) (*types.ProjectCostBreakdown, error) {
	return &types.ProjectCostBreakdown{
		ProjectID:     projectID,
		TotalCost:     150.25,
		InstanceCosts: []types.InstanceCost{},
		StorageCosts:  []types.StorageCost{},
		PeriodStart:   start,
		PeriodEnd:     end,
		GeneratedAt:   time.Now(),
	}, nil
}

// GetProjectResourceUsage gets project resource usage (mock)
func (m *MockClient) GetProjectResourceUsage(ctx context.Context, projectID string, duration time.Duration) (*types.ProjectResourceUsage, error) {
	return &types.ProjectResourceUsage{
		ProjectID:         projectID,
		ActiveInstances:   2,
		TotalInstances:    5,
		TotalStorage:      100.0,
		ComputeHours:      48.5,
		IdleSavings:       25.50,
		MeasurementPeriod: duration,
	}, nil
}

// GetCostTrends returns cost trend data for analysis (mock)
func (m *MockClient) GetCostTrends(ctx context.Context, projectID, period string) (map[string]interface{}, error) {
	// Generate mock cost trend data based on period
	dailyData := []map[string]interface{}{
		{"date": "2025-10-01", "cost": 45.50, "instances": 3},
		{"date": "2025-10-02", "cost": 52.30, "instances": 4},
		{"date": "2025-10-03", "cost": 48.75, "instances": 3},
		{"date": "2025-10-04", "cost": 51.20, "instances": 4},
		{"date": "2025-10-05", "cost": 44.80, "instances": 3},
		{"date": "2025-10-06", "cost": 47.90, "instances": 3},
		{"date": "2025-10-07", "cost": 53.40, "instances": 4},
	}

	weeklyData := []map[string]interface{}{
		{"week": "Week 1", "cost": 320.50, "instances": 3},
		{"week": "Week 2", "cost": 355.30, "instances": 4},
		{"week": "Week 3", "cost": 298.75, "instances": 3},
		{"week": "Week 4", "cost": 340.20, "instances": 4},
	}

	monthlyData := []map[string]interface{}{
		{"month": "July 2025", "cost": 1250.00, "instances": 3},
		{"month": "August 2025", "cost": 1420.50, "instances": 4},
		{"month": "September 2025", "cost": 1380.75, "instances": 3},
		{"month": "October 2025", "cost": 1510.30, "instances": 4},
	}

	var trendsData []map[string]interface{}
	switch period {
	case "daily":
		trendsData = dailyData
	case "weekly":
		trendsData = weeklyData
	case "monthly":
		trendsData = monthlyData
	default:
		trendsData = dailyData
	}

	return map[string]interface{}{
		"project_id":     projectID,
		"period":         period,
		"trends":         trendsData,
		"total_cost":     1510.30,
		"average_cost":   377.58,
		"trend":          "increasing",
		"percent_change": 3.2,
	}, nil
}

// Status operations - Mock implementations

// GetStatus returns daemon status (mock)
func (m *MockClient) GetStatus(ctx context.Context) (*types.DaemonStatus, error) {
	return &types.DaemonStatus{
		Version:   "0.4.1",
		Status:    "running",
		StartTime: time.Now().Add(-2*time.Hour - 30*time.Minute),
	}, nil
}

// Ping pings the daemon (mock)
func (m *MockClient) Ping(ctx context.Context) error {
	return nil
}

// Shutdown shuts down the daemon (mock)
func (m *MockClient) Shutdown(ctx context.Context) error {
	return nil
}

// MakeRequest makes a generic API request (mock)
func (m *MockClient) MakeRequest(method, path string, body interface{}) ([]byte, error) {
	return []byte(`{"status": "mock response"}`), nil
}

// Template application operations - Mock implementations

// ApplyTemplate applies a template to an instance (mock)
func (m *MockClient) ApplyTemplate(ctx context.Context, req templates.ApplyRequest) (*templates.ApplyResponse, error) {
	return &templates.ApplyResponse{
		Success:            true,
		Message:            "Template applied successfully (mock)",
		PackagesInstalled:  5,
		ServicesConfigured: 2,
		UsersCreated:       1,
		RollbackCheckpoint: "checkpoint-1",
		Warnings:           []string{},
		ExecutionTime:      30 * time.Second,
	}, nil
}

// DiffTemplate shows differences between template and instance state (mock)
func (m *MockClient) DiffTemplate(ctx context.Context, req templates.DiffRequest) (*templates.TemplateDiff, error) {
	return &templates.TemplateDiff{
		PackagesToInstall:   []templates.PackageDiff{},
		PackagesToRemove:    []templates.PackageDiff{},
		ServicesToConfigure: []templates.ServiceDiff{},
		ServicesToStop:      []templates.ServiceDiff{},
		UsersToCreate:       []templates.UserDiff{},
		UsersToModify:       []templates.UserDiff{},
		PortsToOpen:         []int{},
		ConflictsFound:      []templates.ConflictDiff{},
	}, nil
}

// GetInstanceLayers gets applied template layers for an instance (mock)
func (m *MockClient) GetInstanceLayers(ctx context.Context, instanceID string) ([]templates.AppliedTemplate, error) {
	return []templates.AppliedTemplate{
		{
			Name:               "base-ubuntu",
			AppliedAt:          time.Now().Add(-1 * time.Hour),
			PackageManager:     "apt",
			PackagesInstalled:  []string{"curl", "wget", "git"},
			ServicesConfigured: []string{"ssh"},
			UsersCreated:       []string{"ubuntu"},
			RollbackCheckpoint: "checkpoint-1",
		},
	}, nil
}

// RollbackInstance rolls back template changes (mock)
func (m *MockClient) RollbackInstance(ctx context.Context, req types.RollbackRequest) error {
	// Mock implementation - just return success
	return nil
}

// MountVolume mounts a volume to an instance (mock)
func (m *MockClient) MountVolume(ctx context.Context, instanceID, volumeID, mountPoint string) error {
	// Mock implementation - just return success
	return nil
}

// UnmountVolume unmounts a volume from an instance (mock)
func (m *MockClient) UnmountVolume(ctx context.Context, instanceID, mountPoint string) error {
	// Mock implementation - just return success
	return nil
}

// Idle policy operations

// ListIdlePolicies returns available idle policies (mock)
func (m *MockClient) ListIdlePolicies(ctx context.Context) ([]*idle.PolicyTemplate, error) {
	manager := idle.NewPolicyManager()
	return manager.ListTemplates(), nil
}

// GetIdlePolicy returns a specific idle policy (mock)
func (m *MockClient) GetIdlePolicy(ctx context.Context, name string) (*idle.PolicyTemplate, error) {
	manager := idle.NewPolicyManager()
	return manager.GetTemplate(name)
}

// ApplyIdlePolicy applies an idle policy to an instance (mock)
func (m *MockClient) ApplyIdlePolicy(ctx context.Context, instanceID, policyName string) error {
	// Mock implementation - just return success
	return nil
}

// RemoveIdlePolicy removes an idle policy from an instance (mock)
func (m *MockClient) RemoveIdlePolicy(ctx context.Context, instanceID, policyName string) error {
	// Mock implementation - just return success
	return nil
}

// GetInstanceIdlePolicies returns policies applied to an instance (mock)
func (m *MockClient) GetInstanceIdlePolicies(ctx context.Context, instanceID string) ([]*idle.PolicyTemplate, error) {
	// Return empty list for mock
	return []*idle.PolicyTemplate{}, nil
}

// RecommendIdlePolicy recommends a policy for an instance (mock)
func (m *MockClient) RecommendIdlePolicy(ctx context.Context, instanceID string) (*idle.PolicyTemplate, error) {
	manager := idle.NewPolicyManager()
	return manager.GetTemplate("balanced")
}

// GetIdleSavingsReport returns idle savings report (mock)
func (m *MockClient) GetIdleSavingsReport(ctx context.Context, period string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"total_saved":               150.00,
		"projected_monthly_savings": 450.00,
		"idle_hours":                720,
	}, nil
}

// AssignPolicySet assigns a policy set to the current user (mock)
func (m *MockClient) AssignPolicySet(ctx context.Context, policySet string) (*client.PolicyAssignResponse, error) {
	return &client.PolicyAssignResponse{
		Success:           true,
		Message:           "Policy set assigned successfully (mock)",
		AssignedPolicySet: policySet,
		EnforcementStatus: "active",
	}, nil
}

// CheckTemplateAccess checks if a template is accessible under current policies (mock)
func (m *MockClient) CheckTemplateAccess(ctx context.Context, templateName string) (*client.PolicyCheckResponse, error) {
	return &client.PolicyCheckResponse{
		Allowed:         true,
		TemplateName:    templateName,
		Reason:          "Template access allowed (mock)",
		MatchedPolicies: []string{"default"},
		Suggestions:     []string{},
	}, nil
}

// GetPolicyStatus returns the current policy enforcement status (mock)
func (m *MockClient) GetPolicyStatus(ctx context.Context) (*client.PolicyStatusResponse, error) {
	return &client.PolicyStatusResponse{
		Enabled:          true,
		Status:           "active",
		StatusIcon:       "✅",
		AssignedPolicies: []string{"default", "basic-access"},
		Message:          "Policy enforcement is active (mock)",
	}, nil
}

// ListPolicySets returns available policy sets (mock)
func (m *MockClient) ListPolicySets(ctx context.Context) (*client.PolicySetsResponse, error) {
	return &client.PolicySetsResponse{
		PolicySets: map[string]client.PolicySetInfo{
			"default": {
				ID:          "default",
				Name:        "Default Policy Set",
				Description: "Standard template and resource access policies",
				Policies:    5,
				Status:      "active",
				Tags:        map[string]string{"environment": "development"},
			},
			"restricted": {
				ID:          "restricted",
				Name:        "Restricted Access",
				Description: "Limited template access for basic users",
				Policies:    3,
				Status:      "available",
				Tags:        map[string]string{"security": "high"},
			},
		},
	}, nil
}

// SetPolicyEnforcement enables or disables policy enforcement (mock)
func (m *MockClient) SetPolicyEnforcement(ctx context.Context, enabled bool) (*client.PolicyEnforcementResponse, error) {
	status := "disabled"
	if enabled {
		status = "enabled"
	}

	return &client.PolicyEnforcementResponse{
		Success: true,
		Message: fmt.Sprintf("Policy enforcement %s successfully (mock)", status),
		Enabled: enabled,
		Status:  status,
	}, nil
}

// Universal AMI System operations - Mock implementations (Phase 5.1 Week 2)

// ResolveAMI resolves AMI for a template (mock)
func (m *MockClient) ResolveAMI(ctx context.Context, templateName string, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"template_name":                templateName,
		"target_region":                "us-east-1",
		"resolution_method":            "fallback_script",
		"ami_id":                       "",
		"launch_time_estimate_seconds": 355,
		"cost_savings":                 0.0,
		"warning":                      "No AMI configuration found, using script provisioning",
	}, nil
}

// TestAMIAvailability tests AMI availability across regions (mock)
func (m *MockClient) TestAMIAvailability(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	templateName := request["template_name"].(string)
	return map[string]interface{}{
		"template_name":     templateName,
		"overall_status":    "passed",
		"tested_at":         time.Now(),
		"total_regions":     4,
		"available_regions": 4,
		"region_results": map[string]interface{}{
			"us-east-1":  map[string]interface{}{"status": "passed"},
			"us-west-2":  map[string]interface{}{"status": "passed"},
			"eu-west-1":  map[string]interface{}{"status": "passed"},
			"ap-south-1": map[string]interface{}{"status": "passed"},
		},
	}, nil
}

// GetAMICosts provides cost analysis for AMI vs script deployment (mock)
func (m *MockClient) GetAMICosts(ctx context.Context, templateName string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"template_name":      templateName,
		"region":             "us-east-1",
		"recommendation":     "neutral",
		"reasoning":          "Both AMI and script provisioning have similar cost/benefit profiles",
		"ami_launch_cost":    0.0336,
		"ami_storage_cost":   0.8000,
		"ami_setup_cost":     0.0003,
		"script_launch_cost": 0.0336,
		"script_setup_cost":  0.0033,
		"script_setup_time":  5,
		"break_even_point":   2.7,
		"cost_savings_1h":    0.0000,
		"cost_savings_8h":    0.0000,
		"time_savings":       5,
	}, nil
}

// PreviewAMIResolution shows what would happen during AMI resolution (mock)
func (m *MockClient) PreviewAMIResolution(ctx context.Context, templateName string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"template_name":                templateName,
		"target_region":                "us-east-1",
		"resolution_method":            "fallback_script",
		"launch_time_estimate_seconds": 355,
		"fallback_chain":               []string{"no_ami_config", "script_fallback"},
		"warning":                      "No AMI available, would use script provisioning",
	}, nil
}

// AMI Creation methods (mock)

// CreateAMI creates an AMI from a running instance (mock)
func (m *MockClient) CreateAMI(ctx context.Context, request types.AMICreationRequest) (map[string]interface{}, error) {
	return map[string]interface{}{
		"creation_id":                  fmt.Sprintf("ami-creation-%s-12345", request.TemplateName),
		"ami_id":                       "ami-mock12345678901234",
		"template_name":                request.TemplateName,
		"instance_id":                  request.InstanceID,
		"target_regions":               request.MultiRegion,
		"status":                       "pending",
		"message":                      "AMI creation initiated successfully",
		"estimated_completion_minutes": 12,
		"storage_cost":                 8.50,
		"creation_cost":                0.025,
	}, nil
}

// GetAMIStatus checks the status of AMI creation (mock)
func (m *MockClient) GetAMIStatus(ctx context.Context, creationID string) (map[string]interface{}, error) {
	// Simulate different stages based on creation ID
	progress := 75
	status := "in_progress"

	return map[string]interface{}{
		"creation_id":                  creationID,
		"ami_id":                       "ami-mock12345678901234",
		"status":                       status,
		"progress":                     progress,
		"message":                      "AMI creation in progress - creating snapshot",
		"estimated_completion_minutes": 3,
		"elapsed_time_minutes":         9,
		"storage_cost":                 8.50,
		"creation_cost":                0.025,
	}, nil
}

// ListUserAMIs lists AMIs created by the user (mock)
func (m *MockClient) ListUserAMIs(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"user_amis": []map[string]interface{}{
			{
				"ami_id":        "ami-mock12345678901234",
				"name":          "my-custom-python-env",
				"description":   "Custom Python ML environment with PyTorch",
				"architecture":  "x86_64",
				"owner":         "123456789012",
				"creation_date": "2024-12-01T15:30:00Z",
				"public":        false,
				"tags": map[string]string{
					"CloudWorkstation": "true",
					"Template":         "python-ml",
					"Creator":          "researcher",
				},
			},
			{
				"ami_id":        "ami-mock98765432109876",
				"name":          "genomics-pipeline-v2",
				"description":   "Optimized genomics analysis pipeline",
				"architecture":  "arm64",
				"owner":         "123456789012",
				"creation_date": "2024-11-30T14:20:00Z",
				"public":        true,
				"tags": map[string]string{
					"CloudWorkstation": "true",
					"Template":         "bioinformatics",
					"Creator":          "researcher",
					"Community":        "published",
				},
			},
		},
		"total_count":  2,
		"storage_cost": 17.00,
	}, nil
}

// Template Marketplace operations - Mock implementations (Phase 5.2)

// SearchMarketplace searches the template marketplace (mock)
func (m *MockClient) SearchMarketplace(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"templates": []map[string]interface{}{
			{
				"id":           "marketplace-template-1",
				"name":         "Deep Learning GPU",
				"description":  "Optimized deep learning environment with GPU support",
				"category":     "Machine Learning",
				"author":       "community-user",
				"downloads":    1250,
				"rating":       4.8,
				"last_updated": "2024-11-15",
				"verified":     true,
			},
		},
		"total_results": 1,
		"query":         params,
	}, nil
}

// GetMarketplaceTemplate gets a specific marketplace template (mock)
func (m *MockClient) GetMarketplaceTemplate(ctx context.Context, templateID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"id":           templateID,
		"name":         "Deep Learning GPU",
		"description":  "Optimized deep learning environment with GPU support and PyTorch",
		"category":     "Machine Learning",
		"author":       "community-user",
		"downloads":    1250,
		"rating":       4.8,
		"last_updated": "2024-11-15",
		"verified":     true,
		"readme":       "# Deep Learning GPU Template\nThis template provides...",
		"installation": "Automated installation via CloudWorkstation marketplace",
	}, nil
}

// PublishMarketplaceTemplate publishes a template to the marketplace (mock)
func (m *MockClient) PublishMarketplaceTemplate(ctx context.Context, template map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"success":     true,
		"template_id": "marketplace-template-new-123",
		"message":     "Template published successfully to marketplace",
		"status":      "pending_review",
	}, nil
}

// AddMarketplaceReview adds a review to a marketplace template (mock)
func (m *MockClient) AddMarketplaceReview(ctx context.Context, templateID string, review map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"success":   true,
		"review_id": "review-123",
		"message":   "Review added successfully",
		"rating":    review["rating"],
	}, nil
}

// ForkMarketplaceTemplate forks a marketplace template (mock)
func (m *MockClient) ForkMarketplaceTemplate(ctx context.Context, templateID string, options map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"success":           true,
		"forked_template":   "forked-template-456",
		"message":           "Template forked successfully",
		"original_template": templateID,
	}, nil
}

// GetMarketplaceFeatured gets featured marketplace templates (mock)
func (m *MockClient) GetMarketplaceFeatured(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"featured_templates": []map[string]interface{}{
			{
				"id":          "featured-1",
				"name":        "Data Science Complete",
				"description": "Complete data science environment with R, Python, and Jupyter",
				"rating":      4.9,
				"downloads":   5000,
				"featured":    true,
			},
		},
		"total_count": 1,
	}, nil
}

// GetMarketplaceTrending gets trending marketplace templates (mock)
func (m *MockClient) GetMarketplaceTrending(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"trending_templates": []map[string]interface{}{
			{
				"id":            "trending-1",
				"name":          "Quantum Computing",
				"description":   "Quantum computing research environment with Qiskit",
				"rating":        4.7,
				"downloads":     750,
				"weekly_growth": 45,
			},
		},
		"total_count": 1,
	}, nil
}

// AnalyzeRightsizing provides instance rightsizing analysis (mock)
func (m *MockClient) AnalyzeRightsizing(ctx context.Context, req types.RightsizingAnalysisRequest) (*types.RightsizingAnalysisResponse, error) {
	return &types.RightsizingAnalysisResponse{}, nil
}

// GetRightsizingRecommendations returns rightsizing recommendations (mock)
func (m *MockClient) GetRightsizingRecommendations(ctx context.Context) (*types.RightsizingRecommendationsResponse, error) {
	return &types.RightsizingRecommendationsResponse{}, nil
}

// GetRightsizingStats returns instance rightsizing statistics (mock)
func (m *MockClient) GetRightsizingStats(ctx context.Context, instanceID string) (*types.RightsizingStatsResponse, error) {
	return &types.RightsizingStatsResponse{}, nil
}

// ExportRightsizingData exports rightsizing analysis data (mock)
func (m *MockClient) ExportRightsizingData(ctx context.Context, format string) ([]types.InstanceMetrics, error) {
	return []types.InstanceMetrics{}, nil
}

// GetRightsizingSummary returns a summary of rightsizing analysis (mock)
func (m *MockClient) GetRightsizingSummary(ctx context.Context) (*types.RightsizingSummaryResponse, error) {
	return &types.RightsizingSummaryResponse{}, nil
}

// GetInstanceMetrics returns metrics for an instance (mock)
func (m *MockClient) GetInstanceMetrics(ctx context.Context, instanceID string, days int) ([]types.InstanceMetrics, error) {
	return []types.InstanceMetrics{}, nil
}

// CheckVersionCompatibility checks version compatibility (mock)
func (m *MockClient) CheckVersionCompatibility(ctx context.Context, version string) error {
	return nil
}

// CleanupAMIs cleans up old AMIs (mock)
func (m *MockClient) CleanupAMIs(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"cleaned": 0}, nil
}

// DeleteAMI deletes an AMI (mock)
func (m *MockClient) DeleteAMI(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true}, nil
}

// ListAMISnapshots lists AMI snapshots (mock)
func (m *MockClient) ListAMISnapshots(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"snapshots": []string{}}, nil
}

// CreateAMISnapshot creates an AMI snapshot (mock)
func (m *MockClient) CreateAMISnapshot(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"snapshot_id": "mock-snapshot"}, nil
}

// RestoreAMIFromSnapshot restores AMI from snapshot (mock)
func (m *MockClient) RestoreAMIFromSnapshot(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"ami_id": "mock-restored-ami"}, nil
}

// DeleteAMISnapshot deletes an AMI snapshot (mock)
func (m *MockClient) DeleteAMISnapshot(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true}, nil
}

// CreateInstanceSnapshot creates an instance snapshot (mock)
func (m *MockClient) CreateInstanceSnapshot(ctx context.Context, req types.InstanceSnapshotRequest) (*types.InstanceSnapshotResult, error) {
	return &types.InstanceSnapshotResult{}, nil
}

// ListInstanceSnapshots lists instance snapshots (mock)
func (m *MockClient) ListInstanceSnapshots(ctx context.Context) (*types.InstanceSnapshotListResponse, error) {
	return &types.InstanceSnapshotListResponse{}, nil
}

// GetInstanceSnapshot gets an instance snapshot (mock)
func (m *MockClient) GetInstanceSnapshot(ctx context.Context, snapshotID string) (*types.InstanceSnapshotInfo, error) {
	return &types.InstanceSnapshotInfo{}, nil
}

// DeleteInstanceSnapshot deletes an instance snapshot (mock)
func (m *MockClient) DeleteInstanceSnapshot(ctx context.Context, snapshotID string) (*types.InstanceSnapshotDeleteResult, error) {
	return &types.InstanceSnapshotDeleteResult{}, nil
}

// RestoreInstanceFromSnapshot restores an instance from snapshot (mock)
func (m *MockClient) RestoreInstanceFromSnapshot(ctx context.Context, snapshotID string, req types.InstanceRestoreRequest) (*types.InstanceRestoreResult, error) {
	return &types.InstanceRestoreResult{}, nil
}

// CreateBackup creates a backup (mock)
func (m *MockClient) CreateBackup(ctx context.Context, req types.BackupCreateRequest) (*types.BackupCreateResult, error) {
	return &types.BackupCreateResult{}, nil
}

// ListBackups lists backups (mock)
func (m *MockClient) ListBackups(ctx context.Context) (*types.BackupListResponse, error) {
	return &types.BackupListResponse{}, nil
}

// GetBackup gets a backup (mock)
func (m *MockClient) GetBackup(ctx context.Context, backupID string) (*types.BackupInfo, error) {
	return &types.BackupInfo{}, nil
}

// DeleteBackup deletes a backup (mock)
func (m *MockClient) DeleteBackup(ctx context.Context, backupID string) (*types.BackupDeleteResult, error) {
	return &types.BackupDeleteResult{}, nil
}

// GetBackupContents gets backup contents (mock)
func (m *MockClient) GetBackupContents(ctx context.Context, req types.BackupContentsRequest) (*types.BackupContentsResponse, error) {
	return &types.BackupContentsResponse{}, nil
}

// VerifyBackup verifies a backup (mock)
func (m *MockClient) VerifyBackup(ctx context.Context, req types.BackupVerifyRequest) (*types.BackupVerifyResult, error) {
	return &types.BackupVerifyResult{}, nil
}

// RestoreBackup restores a backup (mock)
func (m *MockClient) RestoreBackup(ctx context.Context, req types.RestoreRequest) (*types.RestoreResult, error) {
	return &types.RestoreResult{}, nil
}

// GetRestoreStatus gets restore status (mock)
func (m *MockClient) GetRestoreStatus(ctx context.Context, restoreID string) (*types.RestoreResult, error) {
	return &types.RestoreResult{}, nil
}

// ListRestoreOperations lists restore operations (mock)
func (m *MockClient) ListRestoreOperations(ctx context.Context) ([]types.RestoreResult, error) {
	return []types.RestoreResult{}, nil
}

// DisableProjectBudget disables project budget (mock)
func (m *MockClient) DisableProjectBudget(ctx context.Context, projectID string) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true}, nil
}

// SetProjectBudget sets project budget (mock)
func (m *MockClient) SetProjectBudget(ctx context.Context, projectID string, req client.SetProjectBudgetRequest) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true}, nil
}

// UpdateProjectBudget updates project budget (mock)
func (m *MockClient) UpdateProjectBudget(ctx context.Context, projectID string, req client.UpdateProjectBudgetRequest) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true}, nil
}

// ExecInstance executes a command on an instance (mock)
func (m *MockClient) ExecInstance(ctx context.Context, instanceName string, execRequest types.ExecRequest) (*types.ExecResult, error) {
	return &types.ExecResult{}, nil
}

// ResizeInstance resizes an instance (mock)
func (m *MockClient) ResizeInstance(ctx context.Context, req types.ResizeRequest) (*types.ResizeResponse, error) {
	return &types.ResizeResponse{}, nil
}

// GetInstanceLogs gets instance logs (mock)
func (m *MockClient) GetInstanceLogs(ctx context.Context, name string, req types.LogRequest) (*types.LogResponse, error) {
	return &types.LogResponse{}, nil
}

// GetInstanceLogTypes gets instance log types (mock)
func (m *MockClient) GetInstanceLogTypes(ctx context.Context, name string) (*types.LogTypesResponse, error) {
	return &types.LogTypesResponse{}, nil
}

// GetLogsSummary gets logs summary (mock)
func (m *MockClient) GetLogsSummary(ctx context.Context) (*types.LogSummaryResponse, error) {
	return &types.LogSummaryResponse{}, nil
}
