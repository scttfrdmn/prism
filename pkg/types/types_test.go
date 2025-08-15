package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestInstanceSerialization(t *testing.T) {
	original := Instance{
		ID:                 "i-1234567890abcdef0",
		Name:               "test-instance",
		Template:           "r-research",
		PublicIP:           "54.123.45.67",
		PrivateIP:          "10.0.1.100",
		State:              "running",
		LaunchTime:         time.Now().UTC().Truncate(time.Second),
		EstimatedDailyCost: 2.40,
		AttachedVolumes:    []string{"vol-1", "vol-2"},
		AttachedEBSVolumes: []string{"ebs-1", "ebs-2"},
	}

	// Test JSON serialization
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal instance: %v", err)
	}

	// Test JSON deserialization
	var restored Instance
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Failed to unmarshal instance: %v", err)
	}

	// Verify all fields
	if restored.ID != original.ID {
		t.Errorf("ID mismatch: got %s, want %s", restored.ID, original.ID)
	}
	if restored.Name != original.Name {
		t.Errorf("Name mismatch: got %s, want %s", restored.Name, original.Name)
	}
	if restored.Template != original.Template {
		t.Errorf("Template mismatch: got %s, want %s", restored.Template, original.Template)
	}
	if restored.PublicIP != original.PublicIP {
		t.Errorf("PublicIP mismatch: got %s, want %s", restored.PublicIP, original.PublicIP)
	}
	if restored.PrivateIP != original.PrivateIP {
		t.Errorf("PrivateIP mismatch: got %s, want %s", restored.PrivateIP, original.PrivateIP)
	}
	if restored.State != original.State {
		t.Errorf("State mismatch: got %s, want %s", restored.State, original.State)
	}
	if !restored.LaunchTime.Equal(original.LaunchTime) {
		t.Errorf("LaunchTime mismatch: got %v, want %v", restored.LaunchTime, original.LaunchTime)
	}
	if restored.EstimatedDailyCost != original.EstimatedDailyCost {
		t.Errorf("EstimatedDailyCost mismatch: got %f, want %f", restored.EstimatedDailyCost, original.EstimatedDailyCost)
	}
	if len(restored.AttachedVolumes) != len(original.AttachedVolumes) {
		t.Errorf("AttachedVolumes length mismatch: got %d, want %d", len(restored.AttachedVolumes), len(original.AttachedVolumes))
	}
	if len(restored.AttachedEBSVolumes) != len(original.AttachedEBSVolumes) {
		t.Errorf("AttachedEBSVolumes length mismatch: got %d, want %d", len(restored.AttachedEBSVolumes), len(original.AttachedEBSVolumes))
	}
}

func TestEFSVolumeSerialization(t *testing.T) {
	original := EFSVolume{
		Name:            "test-volume",
		FileSystemId:    "fs-1234567890abcdef0",
		Region:          "us-east-1",
		CreationTime:    time.Now().UTC().Truncate(time.Second),
		MountTargets:    []string{"fsmt-1", "fsmt-2"},
		State:           "available",
		PerformanceMode: "generalPurpose",
		ThroughputMode:  "bursting",
		EstimatedCostGB: 0.30,
		SizeBytes:       1073741824, // 1GB
	}

	// Test JSON serialization/deserialization
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal EFS volume: %v", err)
	}

	var restored EFSVolume
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Failed to unmarshal EFS volume: %v", err)
	}

	if restored.Name != original.Name {
		t.Errorf("Name mismatch: got %s, want %s", restored.Name, original.Name)
	}
	if restored.FileSystemId != original.FileSystemId {
		t.Errorf("FileSystemId mismatch: got %s, want %s", restored.FileSystemId, original.FileSystemId)
	}
	if restored.SizeBytes != original.SizeBytes {
		t.Errorf("SizeBytes mismatch: got %d, want %d", restored.SizeBytes, original.SizeBytes)
	}
}

func TestEBSVolumeSerialization(t *testing.T) {
	original := EBSVolume{
		Name:            "test-storage",
		VolumeID:        "vol-1234567890abcdef0",
		Region:          "us-east-1",
		CreationTime:    time.Now().UTC().Truncate(time.Second),
		State:           "available",
		VolumeType:      "gp3",
		SizeGB:          100,
		IOPS:            3000,
		Throughput:      125,
		EstimatedCostGB: 0.08,
		AttachedTo:      "test-instance",
	}

	// Test JSON serialization/deserialization
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal EBS volume: %v", err)
	}

	var restored EBSVolume
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Failed to unmarshal EBS volume: %v", err)
	}

	if restored.VolumeType != original.VolumeType {
		t.Errorf("VolumeType mismatch: got %s, want %s", restored.VolumeType, original.VolumeType)
	}
	if restored.SizeGB != original.SizeGB {
		t.Errorf("SizeGB mismatch: got %d, want %d", restored.SizeGB, original.SizeGB)
	}
	if restored.IOPS != original.IOPS {
		t.Errorf("IOPS mismatch: got %d, want %d", restored.IOPS, original.IOPS)
	}
}

func TestAPIErrorImplementsError(t *testing.T) {
	apiErr := APIError{
		Code:    "404",
		Message: "Instance not found",
		Details: "The specified instance does not exist",
	}

	// Test that APIError implements error interface
	var err error = apiErr
	expected := "404: Instance not found (The specified instance does not exist)"
	if err.Error() != expected {
		t.Errorf("Error() mismatch: got %s, want %s", err.Error(), expected)
	}
}

func TestLaunchRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request LaunchRequest
		wantErr bool
	}{
		{
			name: "valid minimal request",
			request: LaunchRequest{
				Template: "r-research",
				Name:     "my-instance",
			},
			wantErr: false,
		},
		{
			name: "valid full request",
			request: LaunchRequest{
				Template:   "python-research",
				Name:       "ml-workstation",
				Size:       "L",
				Volumes:    []string{"shared-data"},
				EBSVolumes: []string{"fast-storage"},
				Region:     "us-west-2",
				Spot:       true,
				DryRun:     true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON serialization
			data, err := json.Marshal(tt.request)
			if err != nil && !tt.wantErr {
				t.Errorf("Failed to marshal request: %v", err)
			}

			// Test JSON deserialization
			var restored LaunchRequest
			err = json.Unmarshal(data, &restored)
			if err != nil && !tt.wantErr {
				t.Errorf("Failed to unmarshal request: %v", err)
			}

			if !tt.wantErr {
				if restored.Template != tt.request.Template {
					t.Errorf("Template mismatch: got %s, want %s", restored.Template, tt.request.Template)
				}
				if restored.Name != tt.request.Name {
					t.Errorf("Name mismatch: got %s, want %s", restored.Name, tt.request.Name)
				}
			}
		})
	}
}

func TestStateSerialization(t *testing.T) {
	state := State{
		Instances: map[string]Instance{
			"test-1": {
				ID:       "i-123",
				Name:     "test-1",
				Template: "r-research",
				State:    "running",
			},
		},
		Volumes: map[string]EFSVolume{
			"vol-1": {
				Name:         "vol-1",
				FileSystemId: "fs-123",
				State:        "available",
			},
		},
		EBSVolumes: map[string]EBSVolume{
			"ebs-1": {
				Name:     "ebs-1",
				VolumeID: "vol-123",
				State:    "available",
			},
		},
		Config: Config{
			DefaultRegion: "us-east-1",
		},
	}

	// Test JSON serialization/deserialization
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Failed to marshal state: %v", err)
	}

	var restored State
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Failed to unmarshal state: %v", err)
	}

	if len(restored.Instances) != 1 {
		t.Errorf("Instances count mismatch: got %d, want 1", len(restored.Instances))
	}
	if len(restored.Volumes) != 1 {
		t.Errorf("Volumes count mismatch: got %d, want 1", len(restored.Volumes))
	}
	if len(restored.EBSVolumes) != 1 {
		t.Errorf("EBSVolumes count mismatch: got %d, want 1", len(restored.EBSVolumes))
	}
	if restored.Config.DefaultRegion != "us-east-1" {
		t.Errorf("Config DefaultRegion mismatch: got %s, want us-east-1", restored.Config.DefaultRegion)
	}
}