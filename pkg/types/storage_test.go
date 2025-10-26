package types

import (
	"testing"
)

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

// TestStorageVolumeTypeClassification tests type classification
func TestStorageVolumeTypeClassification(t *testing.T) {
	tests := []struct {
		name        string
		volume      StorageVolume
		isWorkspace bool
		isShared    bool
		isCloud     bool
	}{
		{
			name: "EBS Workspace Storage",
			volume: StorageVolume{
				Type:       StorageTypeWorkspace,
				AWSService: AWSServiceEBS,
			},
			isWorkspace: true,
			isShared:    false,
			isCloud:     false,
		},
		{
			name: "EFS Shared Storage",
			volume: StorageVolume{
				Type:       StorageTypeShared,
				AWSService: AWSServiceEFS,
			},
			isWorkspace: false,
			isShared:    true,
			isCloud:     false,
		},
		{
			name: "S3 Cloud Storage",
			volume: StorageVolume{
				Type:       StorageTypeCloud,
				AWSService: AWSServiceS3,
			},
			isWorkspace: false,
			isShared:    false,
			isCloud:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.volume.IsWorkspace() != tt.isWorkspace {
				t.Errorf("%s: IsWorkspace() = %v, want %v", tt.name, tt.volume.IsWorkspace(), tt.isWorkspace)
			}
			if tt.volume.IsShared() != tt.isShared {
				t.Errorf("%s: IsShared() = %v, want %v", tt.name, tt.volume.IsShared(), tt.isShared)
			}
			if tt.volume.IsCloud() != tt.isCloud {
				t.Errorf("%s: IsCloud() = %v, want %v", tt.name, tt.volume.IsCloud(), tt.isCloud)
			}
		})
	}
}
