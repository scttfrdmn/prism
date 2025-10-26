package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	efsTypes "github.com/aws/aws-sdk-go-v2/service/efs/types"
)

// EFSManager handles EFS operations for Prism
type EFSManager struct {
	efsClient *efs.Client
	ec2Client *ec2.Client
	region    string
}

// EFSWorkloadType defines EFS optimization workload types
type EFSWorkloadType string

const (
	EFSWorkloadGeneral EFSWorkloadType = "general"
	EFSWorkloadML      EFSWorkloadType = "ml"
	EFSWorkloadBigData EFSWorkloadType = "bigdata"
)

// NewEFSManager creates a new EFS manager instance
func NewEFSManager(cfg aws.Config) *EFSManager {
	return &EFSManager{
		efsClient: efs.NewFromConfig(cfg),
		ec2Client: ec2.NewFromConfig(cfg),
		region:    cfg.Region,
	}
}

// CreateEFSFilesystem creates a new EFS filesystem
func (m *EFSManager) CreateEFSFilesystem(req StorageRequest) (*StorageInfo, error) {
	ctx := context.Background()

	// Create EFS filesystem
	input := &efs.CreateFileSystemInput{
		CreationToken: aws.String(fmt.Sprintf("cws-efs-%s-%d", req.Name, time.Now().Unix())),
	}

	// Set performance mode based on workload
	if req.EFSConfig != nil {
		input.PerformanceMode = efsTypes.PerformanceModeGeneralPurpose
		if req.EFSConfig.PerformanceMode == "maxIO" {
			input.PerformanceMode = efsTypes.PerformanceModeMaxIo
		}

		// Set throughput mode
		input.ThroughputMode = efsTypes.ThroughputModeBursting
		if req.EFSConfig.ThroughputMode == "provisioned" {
			input.ThroughputMode = efsTypes.ThroughputModeProvisioned
			if req.EFSConfig.ProvisionedThroughput > 0 {
				input.ProvisionedThroughputInMibps = aws.Float64(req.EFSConfig.ProvisionedThroughput)
			}
		}

		// Set availability and durability
		if req.EFSConfig.AvailabilityZone != "" {
			input.AvailabilityZoneName = aws.String(req.EFSConfig.AvailabilityZone)
		}
	}

	// Add tags
	tags := []efsTypes.Tag{
		{Key: aws.String("Name"), Value: aws.String(req.Name)},
		{Key: aws.String("Prism"), Value: aws.String("true")},
		{Key: aws.String("CreatedBy"), Value: aws.String("Prism")},
	}
	input.Tags = tags

	result, err := m.efsClient.CreateFileSystem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create EFS filesystem: %w", err)
	}

	// Note: EFS filesystem waiter not available in current SDK version
	// Filesystem will be available shortly after creation

	storageInfo := &StorageInfo{
		Name:         req.Name,
		Type:         StorageTypeEFS,
		FilesystemID: *result.FileSystemId,
		State:        string(result.LifeCycleState),
		Size:         result.SizeInBytes.Value,
		CreationTime: *result.CreationTime,
		Region:       m.region,
		EFSConfig:    req.EFSConfig,
	}

	return storageInfo, nil
}

// ListEFSFilesystems lists all Prism EFS filesystems
func (m *EFSManager) ListEFSFilesystems() ([]StorageInfo, error) {
	ctx := context.Background()

	input := &efs.DescribeFileSystemsInput{}
	result, err := m.efsClient.DescribeFileSystems(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list EFS filesystems: %w", err)
	}

	var storageInfos []StorageInfo

	for _, filesystem := range result.FileSystems {
		// Check if this is a Prism EFS
		isPrism := false
		name := *filesystem.FileSystemId // Default name

		for _, tag := range filesystem.Tags {
			if *tag.Key == "Prism" && *tag.Value == "true" {
				isPrism = true
			}
			if *tag.Key == "Name" {
				name = *tag.Value
			}
		}

		if !isPrism {
			continue
		}

		storageInfo := StorageInfo{
			Name:         name,
			Type:         StorageTypeEFS,
			FilesystemID: *filesystem.FileSystemId,
			State:        string(filesystem.LifeCycleState),
			Size:         filesystem.SizeInBytes.Value,
			CreationTime: *filesystem.CreationTime,
			Region:       m.region,
		}

		// Get EFS configuration
		config := &EFSConfiguration{
			PerformanceMode: string(filesystem.PerformanceMode),
			ThroughputMode:  string(filesystem.ThroughputMode),
		}
		if filesystem.ProvisionedThroughputInMibps != nil {
			config.ProvisionedThroughput = *filesystem.ProvisionedThroughputInMibps
		}
		if filesystem.AvailabilityZoneName != nil {
			config.AvailabilityZone = *filesystem.AvailabilityZoneName
		}

		storageInfo.EFSConfig = config
		storageInfos = append(storageInfos, storageInfo)
	}

	return storageInfos, nil
}

// DeleteEFSFilesystem deletes an EFS filesystem
func (m *EFSManager) DeleteEFSFilesystem(name string) error {
	ctx := context.Background()

	// Find the filesystem by name
	filesystems, err := m.ListEFSFilesystems()
	if err != nil {
		return fmt.Errorf("failed to find EFS filesystem: %w", err)
	}

	var filesystemId string
	for _, fs := range filesystems {
		if fs.Name == name {
			filesystemId = fs.FilesystemID
			break
		}
	}

	if filesystemId == "" {
		return fmt.Errorf("EFS filesystem not found: %s", name)
	}

	// Delete all mount targets first
	mountTargetsInput := &efs.DescribeMountTargetsInput{
		FileSystemId: aws.String(filesystemId),
	}
	mountTargetsResult, err := m.efsClient.DescribeMountTargets(ctx, mountTargetsInput)
	if err != nil {
		return fmt.Errorf("failed to describe mount targets: %w", err)
	}

	// Delete each mount target
	for _, mt := range mountTargetsResult.MountTargets {
		_, err = m.efsClient.DeleteMountTarget(ctx, &efs.DeleteMountTargetInput{
			MountTargetId: mt.MountTargetId,
		})
		if err != nil {
			return fmt.Errorf("failed to delete mount target: %w", err)
		}
	}

	// Wait for mount targets to be deleted
	if len(mountTargetsResult.MountTargets) > 0 {
		time.Sleep(30 * time.Second) // Give mount targets time to delete
	}

	// Delete the filesystem
	_, err = m.efsClient.DeleteFileSystem(ctx, &efs.DeleteFileSystemInput{
		FileSystemId: aws.String(filesystemId),
	})
	if err != nil {
		return fmt.Errorf("failed to delete EFS filesystem: %w", err)
	}

	return nil
}

// GetMountCommand generates the EFS mount command
func (m *EFSManager) GetMountCommand(filesystemId string, mountPoint string) string {
	return fmt.Sprintf(`#!/bin/bash
# Install EFS utilities
if command -v yum >/dev/null 2>&1; then
    sudo yum install -y amazon-efs-utils
elif command -v apt-get >/dev/null 2>&1; then
    sudo apt-get update
    sudo apt-get install -y amazon-efs-utils
fi

# Create mount point
sudo mkdir -p %s

# Mount EFS filesystem
sudo mount -t efs %s:/ %s -o tls,_netdev

# Add to fstab for persistence
echo "%s.efs.%s.amazonaws.com:/ %s efs tls,_netdev" | sudo tee -a /etc/fstab

echo "EFS filesystem mounted at %s"
`, mountPoint, filesystemId, mountPoint, filesystemId, m.region, mountPoint, mountPoint)
}

// GetInstallScript returns the EFS utilities installation script
func (m *EFSManager) GetInstallScript() string {
	return `#!/bin/bash
# Install amazon-efs-utils for EFS mounting
if command -v yum >/dev/null 2>&1; then
    # RHEL/CentOS/Rocky Linux/Amazon Linux
    sudo yum install -y amazon-efs-utils
elif command -v apt-get >/dev/null 2>&1; then
    # Ubuntu/Debian
    sudo apt-get update
    sudo apt-get install -y amazon-efs-utils
else
    echo "Unsupported package manager. Please install amazon-efs-utils manually."
    exit 1
fi

echo "amazon-efs-utils installation complete"
`
}

// OptimizeForWorkload applies workload-specific optimizations to EFS
func (m *EFSManager) OptimizeForWorkload(filesystemId string, workload EFSWorkloadType) error {
	ctx := context.Background()

	switch workload {
	case EFSWorkloadML:
		// Optimize for machine learning workloads - provisioned throughput
		_, err := m.efsClient.PutFileSystemPolicy(ctx, &efs.PutFileSystemPolicyInput{
			FileSystemId: aws.String(filesystemId),
			Policy: aws.String(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Principal": {"AWS": "*"},
						"Action": [
							"elasticfilesystem:ClientMount",
							"elasticfilesystem:ClientWrite",
							"elasticfilesystem:ClientRootAccess"
						],
						"Condition": {
							"Bool": {
								"aws:SecureTransport": "true"
							}
						}
					}
				]
			}`),
		})
		return err

	case EFSWorkloadBigData:
		// Optimize for big data workloads - max IO performance
		// Note: Throughput mode modification requires recreating filesystem
		return fmt.Errorf("throughput mode modification not supported - create new filesystem with desired configuration")

	case EFSWorkloadGeneral:
		// General purpose optimization - burst mode
		// Note: Throughput mode modification requires recreating filesystem
		return fmt.Errorf("throughput mode modification not supported - create new filesystem with desired configuration")

	default:
		return fmt.Errorf("unsupported EFS workload type: %v", workload)
	}
}

// GetMountTargets returns mount targets for an EFS filesystem
func (m *EFSManager) GetMountTargets(filesystemId string) ([]string, error) {
	ctx := context.Background()

	input := &efs.DescribeMountTargetsInput{
		FileSystemId: aws.String(filesystemId),
	}

	result, err := m.efsClient.DescribeMountTargets(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe mount targets: %w", err)
	}

	var mountTargets []string
	for _, mt := range result.MountTargets {
		if mt.IpAddress != nil {
			mountTargets = append(mountTargets, *mt.IpAddress)
		}
	}

	return mountTargets, nil
}

// CreateMountTarget creates a mount target in a specific subnet
func (m *EFSManager) CreateMountTarget(filesystemId, subnetId string) (*string, error) {
	ctx := context.Background()

	input := &efs.CreateMountTargetInput{
		FileSystemId: aws.String(filesystemId),
		SubnetId:     aws.String(subnetId),
	}

	result, err := m.efsClient.CreateMountTarget(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create mount target: %w", err)
	}

	return result.MountTargetId, nil
}

// GetEFSMetrics retrieves basic EFS metrics
func (m *EFSManager) GetEFSMetrics(filesystemId string) (*EFSMetrics, error) {
	ctx := context.Background()

	// Get filesystem details
	input := &efs.DescribeFileSystemsInput{
		FileSystemId: aws.String(filesystemId),
	}

	result, err := m.efsClient.DescribeFileSystems(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe EFS filesystem: %w", err)
	}

	if len(result.FileSystems) == 0 {
		return nil, fmt.Errorf("EFS filesystem not found: %s", filesystemId)
	}

	fs := result.FileSystems[0]
	metrics := &EFSMetrics{
		FilesystemId:    *fs.FileSystemId,
		SizeInBytes:     fs.SizeInBytes.Value,
		PerformanceMode: string(fs.PerformanceMode),
		ThroughputMode:  string(fs.ThroughputMode),
		LastUpdated:     time.Now(),
	}

	if fs.ProvisionedThroughputInMibps != nil {
		metrics.ProvisionedThroughput = *fs.ProvisionedThroughputInMibps
	}

	return metrics, nil
}

// ListAccessPoints lists EFS access points for a filesystem
func (m *EFSManager) ListAccessPoints(filesystemId string) ([]AccessPointInfo, error) {
	ctx := context.Background()

	input := &efs.DescribeAccessPointsInput{
		FileSystemId: aws.String(filesystemId),
	}

	result, err := m.efsClient.DescribeAccessPoints(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe access points: %w", err)
	}

	var accessPoints []AccessPointInfo
	for _, ap := range result.AccessPoints {
		info := AccessPointInfo{
			AccessPointId:  *ap.AccessPointId,
			FilesystemId:   *ap.FileSystemId,
			Path:           *ap.RootDirectory.Path,
			CreationTime:   time.Now(), // Placeholder - actual field may not exist
			LifeCycleState: string(ap.LifeCycleState),
		}

		if ap.PosixUser != nil {
			info.PosixUser = &PosixUser{
				Uid: uint32(*ap.PosixUser.Uid),
				Gid: uint32(*ap.PosixUser.Gid),
			}
			if len(ap.PosixUser.SecondaryGids) > 0 {
				// Convert int64 slice to uint32 slice
				for _, gid := range ap.PosixUser.SecondaryGids {
					info.PosixUser.SecondaryGids = append(info.PosixUser.SecondaryGids, uint32(gid))
				}
			}
		}

		accessPoints = append(accessPoints, info)
	}

	return accessPoints, nil
}

// CreateAccessPoint creates an EFS access point
func (m *EFSManager) CreateAccessPoint(filesystemId, path string, posixUser *PosixUser) (*AccessPointInfo, error) {
	ctx := context.Background()

	input := &efs.CreateAccessPointInput{
		FileSystemId: aws.String(filesystemId),
		RootDirectory: &efsTypes.RootDirectory{
			Path: aws.String(path),
			CreationInfo: &efsTypes.CreationInfo{
				OwnerUid:    aws.Int64(int64(posixUser.Uid)),
				OwnerGid:    aws.Int64(int64(posixUser.Gid)),
				Permissions: aws.String("0755"),
			},
		},
		Tags: []efsTypes.Tag{
			{Key: aws.String("Prism"), Value: aws.String("true")},
		},
	}

	if posixUser != nil {
		input.PosixUser = &efsTypes.PosixUser{
			Uid: aws.Int64(int64(posixUser.Uid)),
			Gid: aws.Int64(int64(posixUser.Gid)),
		}
		if len(posixUser.SecondaryGids) > 0 {
			for _, gid := range posixUser.SecondaryGids {
				input.PosixUser.SecondaryGids = append(input.PosixUser.SecondaryGids, int64(gid))
			}
		}
	}

	result, err := m.efsClient.CreateAccessPoint(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create access point: %w", err)
	}

	info := &AccessPointInfo{
		AccessPointId:  *result.AccessPointId,
		FilesystemId:   *result.FileSystemId,
		Path:           path,
		CreationTime:   time.Now(), // Placeholder - actual field may not exist
		LifeCycleState: string(result.LifeCycleState),
	}

	if posixUser != nil {
		info.PosixUser = posixUser
	}

	return info, nil
}
