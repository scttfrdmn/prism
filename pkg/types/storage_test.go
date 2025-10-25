package types

import (
	"testing"
	"time"
)

// TestEBSVolumeToStorageVolume tests conversion from EBS to unified storage
func TestEBSVolumeToStorageVolume(t *testing.T) {
	now := time.Now()
	ebs := &EBSVolume{
		Name:            "test-ebs",
		VolumeID:        "vol-123456",
		Region:          "us-west-2",
		CreationTime:    now,
		State:           "available",
		VolumeType:      "gp3",
		SizeGB:          100,
		IOPS:            3000,
		Throughput:      125,
		EstimatedCostGB: 0.08,
		AttachedTo:      "test-instance",
	}

	storage := EBSVolumeToStorageVolume(ebs)

	// Verify basic fields
	if storage.Name != "test-ebs" {
		t.Errorf("Expected name 'test-ebs', got '%s'", storage.Name)
	}
	if storage.Type != StorageTypeWorkspace {
		t.Errorf("Expected type 'workspace', got '%s'", storage.Type)
	}
	if storage.AWSService != AWSServiceEBS {
		t.Errorf("Expected service 'ebs', got '%s'", storage.AWSService)
	}
	if storage.Region != "us-west-2" {
		t.Errorf("Expected region 'us-west-2', got '%s'", storage.Region)
	}
	if storage.State != "available" {
		t.Errorf("Expected state 'available', got '%s'", storage.State)
	}
	if storage.CreationTime != now {
		t.Errorf("Creation time mismatch")
	}

	// Verify EBS-specific fields
	if storage.VolumeID != "vol-123456" {
		t.Errorf("Expected volume ID 'vol-123456', got '%s'", storage.VolumeID)
	}
	if storage.VolumeType != "gp3" {
		t.Errorf("Expected volume type 'gp3', got '%s'", storage.VolumeType)
	}
	if storage.AttachedTo != "test-instance" {
		t.Errorf("Expected attached to 'test-instance', got '%s'", storage.AttachedTo)
	}

	// Verify pointer fields
	if storage.SizeGB == nil || *storage.SizeGB != 100 {
		t.Errorf("Expected size 100 GB, got %v", storage.SizeGB)
	}
	if storage.IOPS == nil || *storage.IOPS != 3000 {
		t.Errorf("Expected IOPS 3000, got %v", storage.IOPS)
	}
	if storage.Throughput == nil || *storage.Throughput != 125 {
		t.Errorf("Expected throughput 125, got %v", storage.Throughput)
	}

	// Verify cost
	if storage.EstimatedCostGB != 0.08 {
		t.Errorf("Expected cost 0.08, got %f", storage.EstimatedCostGB)
	}
}

// TestEFSVolumeToStorageVolume tests conversion from EFS to unified storage
func TestEFSVolumeToStorageVolume(t *testing.T) {
	now := time.Now()
	efs := &EFSVolume{
		Name:            "test-efs",
		FileSystemId:    "fs-123456",
		Region:          "us-west-2",
		CreationTime:    now,
		MountTargets:    []string{"mt-1", "mt-2"},
		State:           "available",
		PerformanceMode: "generalPurpose",
		ThroughputMode:  "bursting",
		EstimatedCostGB: 0.30,
		SizeBytes:       1073741824, // 1 GB
	}

	storage := EFSVolumeToStorageVolume(efs)

	// Verify basic fields
	if storage.Name != "test-efs" {
		t.Errorf("Expected name 'test-efs', got '%s'", storage.Name)
	}
	if storage.Type != StorageTypeShared {
		t.Errorf("Expected type 'shared', got '%s'", storage.Type)
	}
	if storage.AWSService != AWSServiceEFS {
		t.Errorf("Expected service 'efs', got '%s'", storage.AWSService)
	}
	if storage.Region != "us-west-2" {
		t.Errorf("Expected region 'us-west-2', got '%s'", storage.Region)
	}
	if storage.State != "available" {
		t.Errorf("Expected state 'available', got '%s'", storage.State)
	}

	// Verify EFS-specific fields
	if storage.FileSystemID != "fs-123456" {
		t.Errorf("Expected filesystem ID 'fs-123456', got '%s'", storage.FileSystemID)
	}
	if storage.PerformanceMode != "generalPurpose" {
		t.Errorf("Expected performance mode 'generalPurpose', got '%s'", storage.PerformanceMode)
	}
	if storage.ThroughputMode != "bursting" {
		t.Errorf("Expected throughput mode 'bursting', got '%s'", storage.ThroughputMode)
	}
	if len(storage.MountTargets) != 2 {
		t.Errorf("Expected 2 mount targets, got %d", len(storage.MountTargets))
	}

	// Verify size
	if storage.SizeBytes == nil || *storage.SizeBytes != 1073741824 {
		t.Errorf("Expected size 1073741824 bytes, got %v", storage.SizeBytes)
	}

	// Verify cost
	if storage.EstimatedCostGB != 0.30 {
		t.Errorf("Expected cost 0.30, got %f", storage.EstimatedCostGB)
	}
}

// TestStorageVolumeToEBSVolume tests conversion from unified storage to EBS
func TestStorageVolumeToEBSVolume(t *testing.T) {
	now := time.Now()
	sizeGB := int32(100)
	iops := int32(3000)
	throughput := int32(125)

	storage := &StorageVolume{
		Name:            "test-storage",
		Type:            StorageTypeWorkspace,
		AWSService:      AWSServiceEBS,
		Region:          "us-west-2",
		State:           "available",
		CreationTime:    now,
		SizeGB:          &sizeGB,
		VolumeID:        "vol-123456",
		VolumeType:      "gp3",
		IOPS:            &iops,
		Throughput:      &throughput,
		AttachedTo:      "test-instance",
		EstimatedCostGB: 0.08,
	}

	ebs := StorageVolumeToEBSVolume(storage)

	if ebs == nil {
		t.Fatal("Expected non-nil EBSVolume")
	}

	if ebs.Name != "test-storage" {
		t.Errorf("Expected name 'test-storage', got '%s'", ebs.Name)
	}
	if ebs.VolumeID != "vol-123456" {
		t.Errorf("Expected volume ID 'vol-123456', got '%s'", ebs.VolumeID)
	}
	if ebs.SizeGB != 100 {
		t.Errorf("Expected size 100, got %d", ebs.SizeGB)
	}
	if ebs.IOPS != 3000 {
		t.Errorf("Expected IOPS 3000, got %d", ebs.IOPS)
	}
	if ebs.AttachedTo != "test-instance" {
		t.Errorf("Expected attached to 'test-instance', got '%s'", ebs.AttachedTo)
	}
}

// TestStorageVolumeToEFSVolume tests conversion from unified storage to EFS
func TestStorageVolumeToEFSVolume(t *testing.T) {
	now := time.Now()
	sizeBytes := int64(1073741824)

	storage := &StorageVolume{
		Name:            "test-storage",
		Type:            StorageTypeShared,
		AWSService:      AWSServiceEFS,
		Region:          "us-west-2",
		State:           "available",
		CreationTime:    now,
		SizeBytes:       &sizeBytes,
		FileSystemID:    "fs-123456",
		MountTargets:    []string{"mt-1", "mt-2"},
		PerformanceMode: "generalPurpose",
		ThroughputMode:  "bursting",
		EstimatedCostGB: 0.30,
	}

	efs := StorageVolumeToEFSVolume(storage)

	if efs == nil {
		t.Fatal("Expected non-nil EFSVolume")
	}

	if efs.Name != "test-storage" {
		t.Errorf("Expected name 'test-storage', got '%s'", efs.Name)
	}
	if efs.FileSystemId != "fs-123456" {
		t.Errorf("Expected filesystem ID 'fs-123456', got '%s'", efs.FileSystemId)
	}
	if efs.SizeBytes != 1073741824 {
		t.Errorf("Expected size 1073741824, got %d", efs.SizeBytes)
	}
	if len(efs.MountTargets) != 2 {
		t.Errorf("Expected 2 mount targets, got %d", len(efs.MountTargets))
	}
}

// TestStorageVolumeHelperMethods tests the helper methods
func TestStorageVolumeHelperMethods(t *testing.T) {
	// Test IsWorkspace
	ebsStorage := &StorageVolume{
		Type:       StorageTypeWorkspace,
		AWSService: AWSServiceEBS,
	}
	if !ebsStorage.IsWorkspace() {
		t.Error("Expected EBS storage to be workspace storage")
	}
	if ebsStorage.IsShared() {
		t.Error("Expected EBS storage not to be shared")
	}
	if ebsStorage.IsCloud() {
		t.Error("Expected EBS storage not to be cloud")
	}

	// Test IsShared
	efsStorage := &StorageVolume{
		Type:       StorageTypeShared,
		AWSService: AWSServiceEFS,
	}
	if efsStorage.IsWorkspace() {
		t.Error("Expected EFS storage not to be workspace storage")
	}
	if !efsStorage.IsShared() {
		t.Error("Expected EFS storage to be shared")
	}
	if efsStorage.IsCloud() {
		t.Error("Expected EFS storage not to be cloud")
	}

	// Test IsCloud
	s3Storage := &StorageVolume{
		Type:       StorageTypeCloud,
		AWSService: AWSServiceS3,
	}
	if s3Storage.IsWorkspace() {
		t.Error("Expected S3 storage not to be workspace storage")
	}
	if s3Storage.IsShared() {
		t.Error("Expected S3 storage not to be shared")
	}
	if !s3Storage.IsCloud() {
		t.Error("Expected S3 storage to be cloud")
	}
}

// TestGetDisplayType tests the user-friendly type display
func TestGetDisplayType(t *testing.T) {
	tests := []struct {
		storageType StorageType
		expected    string
	}{
		{StorageTypeWorkspace, "Workspace Storage"},
		{StorageTypeShared, "Shared Storage"},
		{StorageTypeCloud, "Cloud Storage"},
		{StorageType("unknown"), "unknown"},
	}

	for _, tt := range tests {
		storage := &StorageVolume{Type: tt.storageType}
		result := storage.GetDisplayType()
		if result != tt.expected {
			t.Errorf("For type %s, expected '%s', got '%s'", tt.storageType, tt.expected, result)
		}
	}
}

// TestGetTechnicalType tests the AWS service type display
func TestGetTechnicalType(t *testing.T) {
	tests := []struct {
		service    AWSService
		volumeType string
		expected   string
	}{
		{AWSServiceEBS, "gp3", "EBS gp3"},
		{AWSServiceEBS, "", "EBS"},
		{AWSServiceEFS, "", "EFS"},
		{AWSServiceS3, "", "S3"},
		{AWSService("unknown"), "", "unknown"},
	}

	for _, tt := range tests {
		storage := &StorageVolume{
			AWSService: tt.service,
			VolumeType: tt.volumeType,
		}
		result := storage.GetTechnicalType()
		if result != tt.expected {
			t.Errorf("For service %s and volume type '%s', expected '%s', got '%s'",
				tt.service, tt.volumeType, tt.expected, result)
		}
	}
}

// TestNilConversions tests that conversion functions handle nil gracefully
func TestNilConversions(t *testing.T) {
	if result := EBSVolumeToStorageVolume(nil); result != nil {
		t.Error("Expected nil result from EBSVolumeToStorageVolume(nil)")
	}

	if result := EFSVolumeToStorageVolume(nil); result != nil {
		t.Error("Expected nil result from EFSVolumeToStorageVolume(nil)")
	}

	if result := StorageVolumeToEBSVolume(nil); result != nil {
		t.Error("Expected nil result from StorageVolumeToEBSVolume(nil)")
	}

	if result := StorageVolumeToEFSVolume(nil); result != nil {
		t.Error("Expected nil result from StorageVolumeToEFSVolume(nil)")
	}
}

// TestWrongTypeConversions tests conversions with incorrect types
func TestWrongTypeConversions(t *testing.T) {
	// Try to convert EFS StorageVolume to EBS
	efsStorage := &StorageVolume{
		Type:       StorageTypeShared,
		AWSService: AWSServiceEFS,
	}
	if result := StorageVolumeToEBSVolume(efsStorage); result != nil {
		t.Error("Expected nil when converting EFS StorageVolume to EBS")
	}

	// Try to convert EBS StorageVolume to EFS
	ebsStorage := &StorageVolume{
		Type:       StorageTypeWorkspace,
		AWSService: AWSServiceEBS,
	}
	if result := StorageVolumeToEFSVolume(ebsStorage); result != nil {
		t.Error("Expected nil when converting EBS StorageVolume to EFS")
	}
}
