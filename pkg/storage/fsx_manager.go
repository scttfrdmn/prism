package storage

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/fsx"
	fsxTypes "github.com/aws/aws-sdk-go-v2/service/fsx/types"
)

// FSxManager handles Amazon FSx operations for CloudWorkstation
type FSxManager struct {
	fsxClient *fsx.Client
	region    string
}

// NewFSxManager creates a new FSx manager instance
func NewFSxManager(cfg aws.Config) *FSxManager {
	return &FSxManager{
		fsxClient: fsx.NewFromConfig(cfg),
		region:    cfg.Region,
	}
}

// CreateFSxFilesystem creates a new FSx filesystem
func (m *FSxManager) CreateFSxFilesystem(req StorageRequest) (*StorageInfo, error) {
	ctx := context.Background()

	if req.FSxConfig == nil {
		return nil, fmt.Errorf("FSx configuration is required")
	}

	// Create appropriate FSx filesystem based on type
	switch req.FSxConfig.FilesystemType {
	case FSxTypeLustre:
		return m.createLustreFilesystem(ctx, req)
	case FSxTypeOpenZFS, FSxTypeZFS:
		return m.createOpenZFSFilesystem(ctx, req)
	case FSxTypeWindows:
		return m.createWindowsFilesystem(ctx, req)
	case FSxTypeNetApp:
		return m.createNetAppFilesystem(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported FSx filesystem type: %s", req.FSxConfig.FilesystemType)
	}
}

// ListFSxFilesystems lists all CloudWorkstation FSx filesystems
func (m *FSxManager) ListFSxFilesystems() ([]StorageInfo, error) {
	ctx := context.Background()

	// List all FSx filesystems
	result, err := m.fsxClient.DescribeFileSystems(ctx, &fsx.DescribeFileSystemsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list FSx filesystems: %w", err)
	}

	var storageInfos []StorageInfo
	for _, fs := range result.FileSystems {
		// Check if this is a CloudWorkstation filesystem by checking tags
		isCloudWorkstation := false
		for _, tag := range fs.Tags {
			if tag.Key != nil && *tag.Key == "ManagedBy" {
				if tag.Value != nil && *tag.Value == "CloudWorkstation" {
					isCloudWorkstation = true
					break
				}
			}
		}

		// Only include CloudWorkstation-managed filesystems
		if !isCloudWorkstation {
			continue
		}

		// All FSx filesystems use the generic FSx type
		fsType := StorageTypeFSx

		var name string
		if fs.FileSystemId != nil {
			name = *fs.FileSystemId
		}

		// Check for Name tag
		for _, tag := range fs.Tags {
			if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
				name = *tag.Value
				break
			}
		}

		storageInfo := StorageInfo{
			Name:      name,
			Type:      fsType,
			Id:        aws.ToString(fs.FileSystemId),
			State:     string(fs.Lifecycle),
			Size:      int64(aws.ToInt32(fs.StorageCapacity)),
			CreatedAt: aws.ToTime(fs.CreationTime),
			DNSName:   aws.ToString(fs.DNSName),
		}

		storageInfos = append(storageInfos, storageInfo)
	}

	return storageInfos, nil
}

// DeleteFSxFilesystem deletes an FSx filesystem
func (m *FSxManager) DeleteFSxFilesystem(name string, skipFinalBackup bool) error {
	ctx := context.Background()

	// First, find the filesystem ID by name or ID
	filesystems, err := m.ListFSxFilesystems()
	if err != nil {
		return fmt.Errorf("failed to list filesystems: %w", err)
	}

	var filesystemId string
	for _, fs := range filesystems {
		if fs.Name == name || fs.Id == name {
			filesystemId = fs.Id
			break
		}
	}

	if filesystemId == "" {
		return fmt.Errorf("filesystem not found: %s", name)
	}

	// Delete the filesystem
	deleteInput := &fsx.DeleteFileSystemInput{
		FileSystemId: aws.String(filesystemId),
	}

	// Configure backup behavior
	if skipFinalBackup {
		deleteInput.WindowsConfiguration = &fsxTypes.DeleteFileSystemWindowsConfiguration{
			SkipFinalBackup: aws.Bool(true),
		}
		deleteInput.LustreConfiguration = &fsxTypes.DeleteFileSystemLustreConfiguration{
			SkipFinalBackup: aws.Bool(true),
		}
		deleteInput.OpenZFSConfiguration = &fsxTypes.DeleteFileSystemOpenZFSConfiguration{
			SkipFinalBackup: aws.Bool(true),
		}
	}

	_, err = m.fsxClient.DeleteFileSystem(ctx, deleteInput)
	if err != nil {
		return fmt.Errorf("failed to delete FSx filesystem: %w", err)
	}

	return nil
}

// getSecurityGroupIds extracts security group IDs from network interface IDs
func getSecurityGroupIds(networkInterfaceIds []string) []string {
	// This is a simplified extraction
	// In real implementation, we would query EC2 for network interface details
	return []string{}
}

// createLustreFilesystem creates a Lustre filesystem
func (m *FSxManager) createLustreFilesystem(ctx context.Context, req StorageRequest) (*StorageInfo, error) {
	// Prepare Lustre configuration
	lustreConfig := &fsxTypes.CreateFileSystemLustreConfiguration{
		DeploymentType: fsxTypes.LustreDeploymentTypeScratch1,
	}

	if req.FSxConfig.PerSecondThroughput > 0 {
		lustreConfig.PerUnitStorageThroughput = aws.Int32(req.FSxConfig.PerSecondThroughput)
	}

	createInput := &fsx.CreateFileSystemInput{
		FileSystemType:      fsxTypes.FileSystemTypeLustre,
		StorageCapacity:     aws.Int32(int32(req.Size)),
		SubnetIds:           req.FSxConfig.SubnetIds,
		LustreConfiguration: lustreConfig,
		Tags: []fsxTypes.Tag{
			{Key: aws.String("Name"), Value: aws.String(req.Name)},
			{Key: aws.String("ManagedBy"), Value: aws.String("CloudWorkstation")},
		},
	}

	if len(req.FSxConfig.SecurityGroupIds) > 0 {
		createInput.SecurityGroupIds = req.FSxConfig.SecurityGroupIds
	}

	result, err := m.fsxClient.CreateFileSystem(ctx, createInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create Lustre filesystem: %w", err)
	}

	return &StorageInfo{
		Name:      req.Name,
		Type:      StorageTypeFSx,
		Id:        aws.ToString(result.FileSystem.FileSystemId),
		State:     string(result.FileSystem.Lifecycle),
		Size:      int64(aws.ToInt32(result.FileSystem.StorageCapacity)),
		CreatedAt: aws.ToTime(result.FileSystem.CreationTime),
		DNSName:   aws.ToString(result.FileSystem.DNSName),
	}, nil
}

// createOpenZFSFilesystem creates an OpenZFS filesystem
func (m *FSxManager) createOpenZFSFilesystem(ctx context.Context, req StorageRequest) (*StorageInfo, error) {
	// Prepare OpenZFS configuration
	zfsConfig := &fsxTypes.CreateFileSystemOpenZFSConfiguration{
		DeploymentType:     fsxTypes.OpenZFSDeploymentTypeSingleAz1,
		ThroughputCapacity: aws.Int32(64), // Default 64 MB/s
		RootVolumeConfiguration: &fsxTypes.OpenZFSCreateRootVolumeConfiguration{
			DataCompressionType: fsxTypes.OpenZFSDataCompressionTypeZstd,
		},
	}

	if req.FSxConfig.ThroughputCapacity > 0 {
		zfsConfig.ThroughputCapacity = aws.Int32(req.FSxConfig.ThroughputCapacity)
	}

	createInput := &fsx.CreateFileSystemInput{
		FileSystemType:       fsxTypes.FileSystemTypeOpenzfs,
		StorageCapacity:      aws.Int32(int32(req.Size)),
		SubnetIds:            req.FSxConfig.SubnetIds,
		OpenZFSConfiguration: zfsConfig,
		Tags: []fsxTypes.Tag{
			{Key: aws.String("Name"), Value: aws.String(req.Name)},
			{Key: aws.String("ManagedBy"), Value: aws.String("CloudWorkstation")},
		},
	}

	if len(req.FSxConfig.SecurityGroupIds) > 0 {
		createInput.SecurityGroupIds = req.FSxConfig.SecurityGroupIds
	}

	result, err := m.fsxClient.CreateFileSystem(ctx, createInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenZFS filesystem: %w", err)
	}

	return &StorageInfo{
		Name:      req.Name,
		Type:      StorageTypeFSx,
		Id:        aws.ToString(result.FileSystem.FileSystemId),
		State:     string(result.FileSystem.Lifecycle),
		Size:      int64(aws.ToInt32(result.FileSystem.StorageCapacity)),
		CreatedAt: aws.ToTime(result.FileSystem.CreationTime),
		DNSName:   aws.ToString(result.FileSystem.DNSName),
	}, nil
}

// createWindowsFilesystem creates a Windows File Server filesystem
func (m *FSxManager) createWindowsFilesystem(ctx context.Context, req StorageRequest) (*StorageInfo, error) {
	// Windows configuration requires Active Directory integration
	if req.FSxConfig.WindowsConfig == nil || req.FSxConfig.WindowsConfig.ActiveDirectoryId == "" {
		return nil, fmt.Errorf("Active Directory ID is required for Windows File Server")
	}

	windowsConfig := &fsxTypes.CreateFileSystemWindowsConfiguration{
		ActiveDirectoryId:  aws.String(req.FSxConfig.WindowsConfig.ActiveDirectoryId),
		ThroughputCapacity: aws.Int32(32), // Default 32 MB/s
		DeploymentType:     fsxTypes.WindowsDeploymentTypeSingleAz1,
	}

	if req.FSxConfig.ThroughputCapacity > 0 {
		windowsConfig.ThroughputCapacity = aws.Int32(req.FSxConfig.ThroughputCapacity)
	}

	createInput := &fsx.CreateFileSystemInput{
		FileSystemType:       fsxTypes.FileSystemTypeWindows,
		StorageCapacity:      aws.Int32(int32(req.Size)),
		SubnetIds:            req.FSxConfig.SubnetIds,
		WindowsConfiguration: windowsConfig,
		Tags: []fsxTypes.Tag{
			{Key: aws.String("Name"), Value: aws.String(req.Name)},
			{Key: aws.String("ManagedBy"), Value: aws.String("CloudWorkstation")},
		},
	}

	if len(req.FSxConfig.SecurityGroupIds) > 0 {
		createInput.SecurityGroupIds = req.FSxConfig.SecurityGroupIds
	}

	result, err := m.fsxClient.CreateFileSystem(ctx, createInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create Windows filesystem: %w", err)
	}

	return &StorageInfo{
		Name:      req.Name,
		Type:      StorageTypeFSx,
		Id:        aws.ToString(result.FileSystem.FileSystemId),
		State:     string(result.FileSystem.Lifecycle),
		Size:      int64(aws.ToInt32(result.FileSystem.StorageCapacity)),
		CreatedAt: aws.ToTime(result.FileSystem.CreationTime),
		DNSName:   aws.ToString(result.FileSystem.DNSName),
	}, nil
}

// createNetAppFilesystem creates a NetApp ONTAP filesystem
func (m *FSxManager) createNetAppFilesystem(ctx context.Context, req StorageRequest) (*StorageInfo, error) {
	// Prepare NetApp ONTAP configuration
	ontapConfig := &fsxTypes.CreateFileSystemOntapConfiguration{
		DeploymentType:     fsxTypes.OntapDeploymentTypeSingleAz1,
		ThroughputCapacity: aws.Int32(128), // Default 128 MB/s
	}

	if req.FSxConfig.ThroughputCapacity > 0 {
		ontapConfig.ThroughputCapacity = aws.Int32(req.FSxConfig.ThroughputCapacity)
	}

	createInput := &fsx.CreateFileSystemInput{
		FileSystemType:     fsxTypes.FileSystemTypeOntap,
		StorageCapacity:    aws.Int32(int32(req.Size)),
		SubnetIds:          req.FSxConfig.SubnetIds,
		OntapConfiguration: ontapConfig,
		Tags: []fsxTypes.Tag{
			{Key: aws.String("Name"), Value: aws.String(req.Name)},
			{Key: aws.String("ManagedBy"), Value: aws.String("CloudWorkstation")},
		},
	}

	if len(req.FSxConfig.SecurityGroupIds) > 0 {
		createInput.SecurityGroupIds = req.FSxConfig.SecurityGroupIds
	}

	result, err := m.fsxClient.CreateFileSystem(ctx, createInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create NetApp ONTAP filesystem: %w", err)
	}

	return &StorageInfo{
		Name:      req.Name,
		Type:      StorageTypeFSx,
		Id:        aws.ToString(result.FileSystem.FileSystemId),
		State:     string(result.FileSystem.Lifecycle),
		Size:      int64(aws.ToInt32(result.FileSystem.StorageCapacity)),
		CreatedAt: aws.ToTime(result.FileSystem.CreationTime),
		DNSName:   aws.ToString(result.FileSystem.DNSName),
	}, nil
}

// GetMountCommand generates the FSx mount command
func (m *FSxManager) GetMountCommand(filesystemType FSxType, filesystemId string, mountPoint string) string {
	switch filesystemType {
	case FSxTypeLustre:
		return fmt.Sprintf(`#!/bin/bash
# Install Lustre client (Amazon Linux 2 / RHEL / CentOS)
if command -v yum >/dev/null 2>&1; then
    sudo yum install -y lustre-client
elif command -v apt-get >/dev/null 2>&1; then
    # Ubuntu / Debian
    sudo apt-get update
    sudo apt-get install -y lustre-client-modules-$(uname -r) lustre-client
fi

# Create mount point
sudo mkdir -p %s

# Mount FSx Lustre filesystem
sudo mount -t lustre %s.fsx.%s.amazonaws.com@tcp:/%s %s

# Verify mount
df -h %s

echo "FSx Lustre filesystem mounted at %s"
`, mountPoint, filesystemId, m.region, filesystemId, mountPoint, mountPoint, mountPoint)

	case FSxTypeOpenZFS, FSxTypeZFS:
		return fmt.Sprintf(`#!/bin/bash
# Install NFS client
if command -v yum >/dev/null 2>&1; then
    sudo yum install -y nfs-utils
elif command -v apt-get >/dev/null 2>&1; then
    sudo apt-get update
    sudo apt-get install -y nfs-common
fi

# Create mount point
sudo mkdir -p %s

# Mount FSx OpenZFS filesystem
sudo mount -t nfs -o nfsvers=3 %s.fsx.%s.amazonaws.com:/fsx %s

# Verify mount
df -h %s

echo "FSx OpenZFS filesystem mounted at %s"
`, mountPoint, filesystemId, m.region, mountPoint, mountPoint, mountPoint)

	case FSxTypeWindows:
		return fmt.Sprintf(`# Windows File Server mount
# Run this in PowerShell as Administrator

# Create mount point (drive letter)
$DriveLetter = "Z:"
$RemotePath = "\\%s.%s.fsx.amazonaws.com\share"

# Mount the filesystem
net use $DriveLetter $RemotePath /persistent:yes

# Verify mount
Get-PSDrive -Name ($DriveLetter -replace ':','')

Write-Host "FSx Windows File Server mounted at $DriveLetter"
`, filesystemId, m.region)

	case FSxTypeNetApp:
		return fmt.Sprintf(`#!/bin/bash
# Install NFS client for NetApp ONTAP
if command -v yum >/dev/null 2>&1; then
    sudo yum install -y nfs-utils
elif command -v apt-get >/dev/null 2>&1; then
    sudo apt-get update
    sudo apt-get install -y nfs-common
fi

# Create mount point
sudo mkdir -p %s

# Mount FSx NetApp ONTAP filesystem (NFS v4.1)
sudo mount -t nfs -o nfsvers=4.1 %s.fsx.%s.amazonaws.com:/vol1 %s

# Verify mount
df -h %s

echo "FSx NetApp ONTAP filesystem mounted at %s"
`, mountPoint, filesystemId, m.region, mountPoint, mountPoint, mountPoint)

	default:
		return "# Unknown FSx filesystem type"
	}
}

// GetInstallScript returns installation script for FSx clients
func (m *FSxManager) GetInstallScript() string {
	return `#!/bin/bash
# Install FSx client utilities for different filesystem types

echo "Installing FSx filesystem clients..."

# Detect OS
if command -v yum >/dev/null 2>&1; then
    OS_TYPE="rhel"
elif command -v apt-get >/dev/null 2>&1; then
    OS_TYPE="debian"
else
    echo "Unsupported OS type"
    exit 1
fi

# Install NFS client (for OpenZFS and NetApp ONTAP)
echo "Installing NFS client..."
if [ "$OS_TYPE" = "rhel" ]; then
    sudo yum install -y nfs-utils
else
    sudo apt-get update
    sudo apt-get install -y nfs-common
fi

# Install Lustre client (for FSx Lustre)
echo "Installing Lustre client..."
if [ "$OS_TYPE" = "rhel" ]; then
    sudo amazon-linux-extras install -y lustre || sudo yum install -y lustre-client
else
    # Ubuntu / Debian
    sudo apt-get install -y lustre-client-modules-$(uname -r) lustre-client || echo "Lustre client not available for this kernel"
fi

echo "FSx client utilities installation complete"
`
}

// OptimizeForWorkload optimizes FSx filesystem for specific workloads
func (m *FSxManager) OptimizeForWorkload(filesystemId string, workload string) error {
	ctx := context.Background()

	// Get filesystem details first
	describeResult, err := m.fsxClient.DescribeFileSystems(ctx, &fsx.DescribeFileSystemsInput{
		FileSystemIds: []string{filesystemId},
	})
	if err != nil {
		return fmt.Errorf("failed to describe filesystem: %w", err)
	}

	if len(describeResult.FileSystems) == 0 {
		return fmt.Errorf("filesystem not found: %s", filesystemId)
	}

	fs := describeResult.FileSystems[0]

	// Apply workload-specific optimizations based on filesystem type
	switch fs.FileSystemType {
	case fsxTypes.FileSystemTypeLustre:
		return m.optimizeLustreForWorkload(ctx, filesystemId, workload)
	case fsxTypes.FileSystemTypeOpenzfs:
		return m.optimizeOpenZFSForWorkload(ctx, filesystemId, workload)
	case fsxTypes.FileSystemTypeOntap:
		return m.optimizeNetAppForWorkload(ctx, filesystemId, workload)
	default:
		return fmt.Errorf("workload optimization not supported for filesystem type: %s", fs.FileSystemType)
	}
}

// optimizeLustreForWorkload optimizes Lustre for specific workloads
func (m *FSxManager) optimizeLustreForWorkload(ctx context.Context, filesystemId string, workload string) error {
	// Lustre optimization via UpdateFileSystem API
	updateInput := &fsx.UpdateFileSystemInput{
		FileSystemId: aws.String(filesystemId),
		LustreConfiguration: &fsxTypes.UpdateFileSystemLustreConfiguration{
			// Workload-specific settings would be applied here
			AutoImportPolicy: fsxTypes.AutoImportPolicyTypeNewChanged, // Enable auto-import for data workloads
		},
	}

	_, err := m.fsxClient.UpdateFileSystem(ctx, updateInput)
	if err != nil {
		return fmt.Errorf("failed to optimize Lustre filesystem: %w", err)
	}

	return nil
}

// optimizeOpenZFSForWorkload optimizes OpenZFS for specific workloads
func (m *FSxManager) optimizeOpenZFSForWorkload(ctx context.Context, filesystemId string, workload string) error {
	// OpenZFS optimization via UpdateFileSystem API
	updateInput := &fsx.UpdateFileSystemInput{
		FileSystemId: aws.String(filesystemId),
		OpenZFSConfiguration: &fsxTypes.UpdateFileSystemOpenZFSConfiguration{
			// Workload-specific settings
			ThroughputCapacity: aws.Int32(256), // Increase for high-performance workloads
		},
	}

	_, err := m.fsxClient.UpdateFileSystem(ctx, updateInput)
	if err != nil {
		return fmt.Errorf("failed to optimize OpenZFS filesystem: %w", err)
	}

	return nil
}

// optimizeNetAppForWorkload optimizes NetApp ONTAP for specific workloads
func (m *FSxManager) optimizeNetAppForWorkload(ctx context.Context, filesystemId string, workload string) error {
	// NetApp ONTAP optimization via UpdateFileSystem API
	updateInput := &fsx.UpdateFileSystemInput{
		FileSystemId: aws.String(filesystemId),
		OntapConfiguration: &fsxTypes.UpdateFileSystemOntapConfiguration{
			// Workload-specific settings
			ThroughputCapacity: aws.Int32(512), // Increase for high-performance workloads
		},
	}

	_, err := m.fsxClient.UpdateFileSystem(ctx, updateInput)
	if err != nil {
		return fmt.Errorf("failed to optimize NetApp ONTAP filesystem: %w", err)
	}

	return nil
}
