package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// EBSManager handles EBS operations for Prism
type EBSManager struct {
	ec2Client *ec2.Client
	region    string
}

// EBSWorkloadType defines EBS optimization workload types
type EBSWorkloadType string

const (
	EBSWorkloadGeneral EBSWorkloadType = "general"
	EBSWorkloadML      EBSWorkloadType = "ml"
	EBSWorkloadBigData EBSWorkloadType = "bigdata"
)

// NewEBSManager creates a new EBS manager instance
func NewEBSManager(cfg aws.Config) *EBSManager {
	return &EBSManager{
		ec2Client: ec2.NewFromConfig(cfg),
		region:    cfg.Region,
	}
}

// CreateEBSVolume creates a new EBS volume
func (m *EBSManager) CreateEBSVolume(req StorageRequest) (*StorageInfo, error) {
	ctx := context.Background()

	// Parse and validate configuration
	volumeType, iops, throughput := m.parseVolumeConfiguration(req)

	// Build volume creation input
	input, err := m.buildVolumeInput(ctx, req, volumeType, iops, throughput)
	if err != nil {
		return nil, err
	}

	// Create the volume
	volumeID, err := m.createVolume(ctx, input)
	if err != nil {
		return nil, err
	}

	// Wait for volume availability
	if err := m.waitForVolumeAvailable(ctx, volumeID); err != nil {
		return nil, err
	}

	// Build and return storage info
	return m.buildStorageInfo(ctx, volumeID, req), nil
}

// parseVolumeConfiguration extracts volume type, IOPS, and throughput from request
func (m *EBSManager) parseVolumeConfiguration(req StorageRequest) (types.VolumeType, int32, int32) {
	volumeType := types.VolumeTypeGp3
	iops := int32(3000)      // GP3 default
	throughput := int32(125) // GP3 default (MB/s)

	if req.EBSConfig == nil {
		return volumeType, iops, throughput
	}

	// Parse volume type
	switch req.EBSConfig.VolumeType {
	case "gp2":
		volumeType = types.VolumeTypeGp2
	case "gp3":
		volumeType = types.VolumeTypeGp3
	case "io1":
		volumeType = types.VolumeTypeIo1
	case "io2":
		volumeType = types.VolumeTypeIo2
	case "sc1":
		volumeType = types.VolumeTypeSc1
	case "st1":
		volumeType = types.VolumeTypeSt1
	}

	// Override IOPS and throughput if specified
	if req.EBSConfig.IOPS > 0 {
		iops = req.EBSConfig.IOPS
	}
	if req.EBSConfig.Throughput > 0 {
		throughput = req.EBSConfig.Throughput
	}

	return volumeType, iops, throughput
}

// buildVolumeInput constructs the EC2 CreateVolumeInput
func (m *EBSManager) buildVolumeInput(ctx context.Context, req StorageRequest, volumeType types.VolumeType, iops, throughput int32) (*ec2.CreateVolumeInput, error) {
	input := &ec2.CreateVolumeInput{
		Size:       aws.Int32(int32(req.Size)),
		VolumeType: volumeType,
		Encrypted:  aws.Bool(true), // Always encrypt by default
	}

	// Set IOPS for provisioned IOPS volume types
	if m.supportsIOPS(volumeType) {
		input.Iops = aws.Int32(iops)
	}

	// Set throughput for supported volume types
	if m.supportsThroughput(volumeType) {
		input.Throughput = aws.Int32(throughput)
	}

	// Select availability zone
	az, err := m.selectAvailabilityZone(ctx, req)
	if err != nil {
		return nil, err
	}
	input.AvailabilityZone = aws.String(az)

	// Add tags
	input.TagSpecifications = m.buildTagSpecifications(req.Name)

	return input, nil
}

// supportsIOPS checks if volume type supports IOPS configuration
func (m *EBSManager) supportsIOPS(volumeType types.VolumeType) bool {
	return volumeType == types.VolumeTypeIo1 ||
		volumeType == types.VolumeTypeIo2 ||
		volumeType == types.VolumeTypeGp3
}

// supportsThroughput checks if volume type supports throughput configuration
func (m *EBSManager) supportsThroughput(volumeType types.VolumeType) bool {
	return volumeType == types.VolumeTypeGp3 ||
		volumeType == types.VolumeTypeSt1 ||
		volumeType == types.VolumeTypeSc1
}

// selectAvailabilityZone determines the availability zone for the volume
func (m *EBSManager) selectAvailabilityZone(ctx context.Context, req StorageRequest) (string, error) {
	if req.EBSConfig != nil && req.EBSConfig.AvailabilityZone != "" {
		return req.EBSConfig.AvailabilityZone, nil
	}

	// Use the first AZ in the region by default
	azs, err := m.getAvailabilityZones(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get availability zones: %w", err)
	}

	if len(azs) == 0 {
		return "", fmt.Errorf("no availability zones found in region")
	}

	return azs[0], nil
}

// buildTagSpecifications creates tag specifications for the volume
func (m *EBSManager) buildTagSpecifications(name string) []types.TagSpecification {
	return []types.TagSpecification{
		{
			ResourceType: types.ResourceTypeVolume,
			Tags: []types.Tag{
				{Key: aws.String("Name"), Value: aws.String(name)},
				{Key: aws.String("Prism"), Value: aws.String("true")},
				{Key: aws.String("CreatedBy"), Value: aws.String("Prism")},
			},
		},
	}
}

// createVolume executes the volume creation API call
func (m *EBSManager) createVolume(ctx context.Context, input *ec2.CreateVolumeInput) (string, error) {
	result, err := m.ec2Client.CreateVolume(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to create EBS volume: %w", err)
	}
	return *result.VolumeId, nil
}

// waitForVolumeAvailable waits for volume to reach available state
func (m *EBSManager) waitForVolumeAvailable(ctx context.Context, volumeID string) error {
	waiter := ec2.NewVolumeAvailableWaiter(m.ec2Client)
	err := waiter.Wait(ctx, &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeID},
	}, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("timeout waiting for EBS volume to become available: %w", err)
	}
	return nil
}

// buildStorageInfo constructs the StorageInfo response
func (m *EBSManager) buildStorageInfo(ctx context.Context, volumeID string, req StorageRequest) *StorageInfo {
	// Describe volume to get current state
	result, err := m.ec2Client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeID},
	})

	if err != nil || len(result.Volumes) == 0 {
		// Fallback to basic info if describe fails
		return &StorageInfo{
			Name:      req.Name,
			Type:      StorageTypeEBS,
			Id:        volumeID,
			State:     "available",
			Size:      req.Size,
			CreatedAt: time.Now(),
			EBSConfig: req.EBSConfig,
		}
	}

	volume := result.Volumes[0]
	return &StorageInfo{
		Name:      req.Name,
		Type:      StorageTypeEBS,
		Id:        *volume.VolumeId,
		State:     string(volume.State),
		Size:      int64(*volume.Size),
		CreatedAt: *volume.CreateTime,
		EBSConfig: req.EBSConfig,
	}
}

// ListEBSVolumes lists all Prism EBS volumes
func (m *EBSManager) ListEBSVolumes() ([]StorageInfo, error) {
	ctx := context.Background()

	// Filter for Prism volumes
	input := &ec2.DescribeVolumesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Prism"),
				Values: []string{"true"},
			},
		},
	}

	result, err := m.ec2Client.DescribeVolumes(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list EBS volumes: %w", err)
	}

	var storageInfos []StorageInfo

	for _, volume := range result.Volumes {
		name := *volume.VolumeId // Default name

		// Extract name from tags
		for _, tag := range volume.Tags {
			if *tag.Key == "Name" {
				name = *tag.Value
				break
			}
		}

		storageInfo := StorageInfo{
			Name:      name,
			Type:      StorageTypeEBS,
			Id:        *volume.VolumeId,
			State:     string(volume.State),
			Size:      int64(*volume.Size),
			CreatedAt: *volume.CreateTime,
		}

		// Get EBS configuration
		config := &EBSConfiguration{
			VolumeType:       string(volume.VolumeType),
			AvailabilityZone: *volume.AvailabilityZone,
			Encrypted:        *volume.Encrypted,
		}

		if volume.Iops != nil {
			config.IOPS = *volume.Iops
		}
		if volume.Throughput != nil {
			config.Throughput = *volume.Throughput
		}
		if volume.SnapshotId != nil {
			config.SnapshotID = *volume.SnapshotId
		}

		storageInfo.EBSConfig = config
		storageInfos = append(storageInfos, storageInfo)
	}

	return storageInfos, nil
}

// DeleteEBSVolume deletes an EBS volume
func (m *EBSManager) DeleteEBSVolume(name string) error {
	ctx := context.Background()

	// Find the volume by name
	volumes, err := m.ListEBSVolumes()
	if err != nil {
		return fmt.Errorf("failed to find EBS volume: %w", err)
	}

	var volumeId string
	for _, vol := range volumes {
		if vol.Name == name {
			volumeId = vol.Id
			break
		}
	}

	if volumeId == "" {
		return fmt.Errorf("EBS volume not found: %s", name)
	}

	// Detach volume if attached
	err = m.detachVolumeIfAttached(ctx, volumeId)
	if err != nil {
		return fmt.Errorf("failed to detach volume: %w", err)
	}

	// Delete the volume
	_, err = m.ec2Client.DeleteVolume(ctx, &ec2.DeleteVolumeInput{
		VolumeId: aws.String(volumeId),
	})
	if err != nil {
		return fmt.Errorf("failed to delete EBS volume: %w", err)
	}

	return nil
}

// AttachEBSVolume attaches an EBS volume to an EC2 instance
func (m *EBSManager) AttachEBSVolume(volumeName, instanceID string) error {
	ctx := context.Background()

	// Find the volume by name
	volumes, err := m.ListEBSVolumes()
	if err != nil {
		return fmt.Errorf("failed to find EBS volume: %w", err)
	}

	var volumeId string
	for _, vol := range volumes {
		if vol.Name == volumeName {
			volumeId = vol.Id
			break
		}
	}

	if volumeId == "" {
		return fmt.Errorf("EBS volume not found: %s", volumeName)
	}

	// Find available device name
	deviceName, err := m.findAvailableDeviceName(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to find available device name: %w", err)
	}

	// Attach volume
	_, err = m.ec2Client.AttachVolume(ctx, &ec2.AttachVolumeInput{
		VolumeId:   aws.String(volumeId),
		InstanceId: aws.String(instanceID),
		Device:     aws.String(deviceName),
	})
	if err != nil {
		return fmt.Errorf("failed to attach EBS volume: %w", err)
	}

	// Wait for attachment to complete
	waiter := ec2.NewVolumeInUseWaiter(m.ec2Client)
	err = waiter.Wait(ctx, &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeId},
	}, 2*time.Minute)
	if err != nil {
		return fmt.Errorf("timeout waiting for volume attachment: %w", err)
	}

	return nil
}

// DetachEBSVolume detaches an EBS volume from an EC2 instance
func (m *EBSManager) DetachEBSVolume(volumeName, instanceID string) error {
	ctx := context.Background()

	// Find the volume by name
	volumes, err := m.ListEBSVolumes()
	if err != nil {
		return fmt.Errorf("failed to find EBS volume: %w", err)
	}

	var volumeId string
	for _, vol := range volumes {
		if vol.Name == volumeName {
			volumeId = vol.Id
			break
		}
	}

	if volumeId == "" {
		return fmt.Errorf("EBS volume not found: %s", volumeName)
	}

	// Detach volume
	_, err = m.ec2Client.DetachVolume(ctx, &ec2.DetachVolumeInput{
		VolumeId:   aws.String(volumeId),
		InstanceId: aws.String(instanceID),
		Force:      aws.Bool(false), // Graceful detachment
	})
	if err != nil {
		return fmt.Errorf("failed to detach EBS volume: %w", err)
	}

	// Wait for detachment to complete
	waiter := ec2.NewVolumeAvailableWaiter(m.ec2Client)
	err = waiter.Wait(ctx, &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeId},
	}, 2*time.Minute)
	if err != nil {
		return fmt.Errorf("timeout waiting for volume detachment: %w", err)
	}

	return nil
}

// GetMountCommand generates the EBS volume mount command
func (m *EBSManager) GetMountCommand(volumeId string, mountPoint string, config *EBSConfiguration) string {
	devicePath := "/dev/xvdf" // Default device path
	filesystem := "ext4"      // Default filesystem

	if config != nil && config.Filesystem != "" {
		filesystem = config.Filesystem
	}

	return fmt.Sprintf(`#!/bin/bash
# Wait for device to be available
DEVICE_PATH="%s"
while [ ! -e $DEVICE_PATH ]; do
    echo "Waiting for device $DEVICE_PATH to be available..."
    sleep 2
done

# Check if filesystem exists, create if not
if ! blkid $DEVICE_PATH; then
    echo "Creating %s filesystem on $DEVICE_PATH..."
    sudo mkfs.%s $DEVICE_PATH
fi

# Create mount point
sudo mkdir -p %s

# Mount the volume
sudo mount $DEVICE_PATH %s

# Add to fstab for persistence (using UUID for reliability)
UUID=$(sudo blkid -s UUID -o value $DEVICE_PATH)
if [ ! -z "$UUID" ]; then
    echo "UUID=$UUID %s %s defaults,nofail 0 2" | sudo tee -a /etc/fstab
fi

echo "EBS volume mounted at %s"
`, devicePath, filesystem, filesystem, mountPoint, mountPoint, mountPoint, filesystem, mountPoint)
}

// GetInstallScript returns the EBS utilities installation script (minimal for EBS)
func (m *EBSManager) GetInstallScript() string {
	return `#!/bin/bash
# EBS volumes don't require special utilities - they work with standard Linux tools
# Ensure essential filesystem tools are available

if command -v yum >/dev/null 2>&1; then
    # RHEL/CentOS/Rocky Linux/Amazon Linux
    sudo yum install -y util-linux e2fsprogs xfsprogs
elif command -v apt-get >/dev/null 2>&1; then
    # Ubuntu/Debian
    sudo apt-get update
    sudo apt-get install -y util-linux e2fsprogs xfsprogs
else
    echo "Filesystem utilities should already be available on most Linux distributions"
fi

echo "EBS volume support ready"
`
}

// OptimizeForWorkload applies workload-specific optimizations to EBS
func (m *EBSManager) OptimizeForWorkload(volumeId string, workload EBSWorkloadType) error {
	ctx := context.Background()

	switch workload {
	case EBSWorkloadML:
		// Optimize for machine learning workloads - high IOPS
		return m.modifyVolume(ctx, volumeId, "gp3", 10000, 500)

	case EBSWorkloadBigData:
		// Optimize for big data workloads - high throughput
		return m.modifyVolume(ctx, volumeId, "gp3", 16000, 1000)

	case EBSWorkloadGeneral:
		// General purpose optimization
		return m.modifyVolume(ctx, volumeId, "gp3", 3000, 125)

	default:
		return fmt.Errorf("unsupported EBS workload type: %v", workload)
	}
}

// modifyVolume modifies EBS volume configuration
func (m *EBSManager) modifyVolume(ctx context.Context, volumeId, volumeType string, iops, throughput int32) error {
	input := &ec2.ModifyVolumeInput{
		VolumeId:   aws.String(volumeId),
		VolumeType: types.VolumeType(volumeType),
	}

	// Set IOPS for relevant volume types
	if volumeType == "gp3" || volumeType == "io1" || volumeType == "io2" {
		input.Iops = aws.Int32(iops)
	}

	// Set throughput for GP3
	if volumeType == "gp3" {
		input.Throughput = aws.Int32(throughput)
	}

	_, err := m.ec2Client.ModifyVolume(ctx, input)
	return err
}

// CreateSnapshot creates a snapshot of an EBS volume
func (m *EBSManager) CreateSnapshot(volumeName, description string) (*SnapshotInfo, error) {
	ctx := context.Background()

	// Find the volume by name
	volumes, err := m.ListEBSVolumes()
	if err != nil {
		return nil, fmt.Errorf("failed to find EBS volume: %w", err)
	}

	var volumeId string
	for _, vol := range volumes {
		if vol.Name == volumeName {
			volumeId = vol.Id
			break
		}
	}

	if volumeId == "" {
		return nil, fmt.Errorf("EBS volume not found: %s", volumeName)
	}

	// Create snapshot
	input := &ec2.CreateSnapshotInput{
		VolumeId:    aws.String(volumeId),
		Description: aws.String(description),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSnapshot,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("%s-snapshot", volumeName))},
					{Key: aws.String("Prism"), Value: aws.String("true")},
					{Key: aws.String("SourceVolume"), Value: aws.String(volumeName)},
				},
			},
		},
	}

	result, err := m.ec2Client.CreateSnapshot(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	info := &SnapshotInfo{
		SnapshotId:  *result.SnapshotId,
		VolumeId:    *result.VolumeId,
		State:       string(result.State),
		Progress:    *result.Progress,
		StartTime:   *result.StartTime,
		Description: description,
		VolumeSize:  *result.VolumeSize,
	}

	return info, nil
}

// ListSnapshots lists EBS snapshots created by Prism
func (m *EBSManager) ListSnapshots() ([]SnapshotInfo, error) {
	ctx := context.Background()

	input := &ec2.DescribeSnapshotsInput{
		OwnerIds: []string{"self"},
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Prism"),
				Values: []string{"true"},
			},
		},
	}

	result, err := m.ec2Client.DescribeSnapshots(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	var snapshots []SnapshotInfo
	for _, snapshot := range result.Snapshots {
		info := SnapshotInfo{
			SnapshotId:  *snapshot.SnapshotId,
			VolumeId:    *snapshot.VolumeId,
			State:       string(snapshot.State),
			Progress:    *snapshot.Progress,
			StartTime:   *snapshot.StartTime,
			VolumeSize:  *snapshot.VolumeSize,
			Description: *snapshot.Description,
		}

		snapshots = append(snapshots, info)
	}

	return snapshots, nil
}

// Helper methods

func (m *EBSManager) getAvailabilityZones(ctx context.Context) ([]string, error) {
	result, err := m.ec2Client.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"available"},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	var zones []string
	for _, zone := range result.AvailabilityZones {
		zones = append(zones, *zone.ZoneName)
	}
	return zones, nil
}

func (m *EBSManager) findAvailableDeviceName(ctx context.Context, instanceID string) (string, error) {
	// Get instance details to see what devices are already attached
	result, err := m.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return "", err
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return "", fmt.Errorf("instance not found: %s", instanceID)
	}

	instance := result.Reservations[0].Instances[0]
	usedDevices := make(map[string]bool)

	// Track used device names
	for _, mapping := range instance.BlockDeviceMappings {
		usedDevices[*mapping.DeviceName] = true
	}

	// Try standard device names
	deviceNames := []string{
		"/dev/sdf", "/dev/sdg", "/dev/sdh", "/dev/sdi", "/dev/sdj",
		"/dev/sdk", "/dev/sdl", "/dev/sdm", "/dev/sdn", "/dev/sdo", "/dev/sdp",
	}

	for _, deviceName := range deviceNames {
		if !usedDevices[deviceName] {
			return deviceName, nil
		}
	}

	return "", fmt.Errorf("no available device names for instance %s", instanceID)
}

func (m *EBSManager) detachVolumeIfAttached(ctx context.Context, volumeId string) error {
	// Check if volume is attached
	result, err := m.ec2Client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeId},
	})
	if err != nil {
		return err
	}

	if len(result.Volumes) == 0 {
		return fmt.Errorf("volume not found: %s", volumeId)
	}

	volume := result.Volumes[0]
	if volume.State == types.VolumeStateInUse && len(volume.Attachments) > 0 {
		attachment := volume.Attachments[0]
		_, err = m.ec2Client.DetachVolume(ctx, &ec2.DetachVolumeInput{
			VolumeId:   aws.String(volumeId),
			InstanceId: attachment.InstanceId,
		})
		if err != nil {
			return err
		}

		// Wait for detachment
		waiter := ec2.NewVolumeAvailableWaiter(m.ec2Client)
		err = waiter.Wait(ctx, &ec2.DescribeVolumesInput{
			VolumeIds: []string{volumeId},
		}, 2*time.Minute)
		return err
	}

	return nil
}

// GetEBSMetrics retrieves basic EBS metrics
func (m *EBSManager) GetEBSMetrics(volumeId string) (*EBSMetrics, error) {
	ctx := context.Background()

	result, err := m.ec2Client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeId},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe EBS volume: %w", err)
	}

	if len(result.Volumes) == 0 {
		return nil, fmt.Errorf("EBS volume not found: %s", volumeId)
	}

	volume := result.Volumes[0]
	metrics := &EBSMetrics{
		VolumeId:    *volume.VolumeId,
		VolumeType:  string(volume.VolumeType),
		Size:        int64(*volume.Size),
		State:       string(volume.State),
		Encrypted:   *volume.Encrypted,
		LastUpdated: time.Now(),
	}

	if volume.Iops != nil {
		metrics.IOPS = *volume.Iops
	}
	if volume.Throughput != nil {
		metrics.Throughput = *volume.Throughput
	}

	return metrics, nil
}
