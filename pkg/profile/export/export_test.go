package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

func TestExportProfiles(t *testing.T) {
	// Setup test environment
	tempDir, err := os.MkdirTemp("", "cws-profile-export-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test profiles
	profiles := []profile.Profile{
		{
			Type:       "personal",
			Name:       "Test Profile 1",
			AWSProfile: "test-profile-1",
			Region:     "us-west-2",
			CreatedAt:  time.Now(),
		},
		{
			Type:            "invitation",
			Name:            "Test Invitation",
			AWSProfile:      "test-invitation",
			InvitationToken: "inv-test123",
			OwnerAccount:    "test-account",
			Region:          "us-east-1",
			CreatedAt:       time.Now(),
		},
	}

	// Create mock profile manager
	mockManager := &MockProfileManager{}

	// Test export to JSON
	jsonOutput := filepath.Join(tempDir, "profiles.json")
	err = ExportProfiles(mockManager, profiles, jsonOutput, ExportOptions{
		Format: "json",
	})

	if err != nil {
		t.Fatalf("ExportProfiles failed: %v", err)
	}

	// Verify JSON output
	jsonData, err := os.ReadFile(jsonOutput)
	if err != nil {
		t.Fatalf("Failed to read JSON output: %v", err)
	}

	var exportData ExportFormat
	if err := json.Unmarshal(jsonData, &exportData); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if len(exportData.Profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(exportData.Profiles))
	}

	// Test export to ZIP
	zipOutput := filepath.Join(tempDir, "profiles.zip")
	err = ExportProfiles(mockManager, profiles, zipOutput, DefaultExportOptions())

	if err != nil {
		t.Fatalf("ExportProfiles to ZIP failed: %v", err)
	}

	// Verify ZIP file exists
	if _, err := os.Stat(zipOutput); os.IsNotExist(err) {
		t.Errorf("ZIP file not created")
	}
}

func TestImportProfiles(t *testing.T) {
	// Setup test environment
	tempDir, err := os.MkdirTemp("", "cws-profile-import-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test export data
	exportData := ExportFormat{
		Version:    "1.0.0",
		ExportedAt: time.Now(),
		Profiles: []profile.Profile{
			{
				Type:       "personal",
				Name:       "Test Profile 1",
				AWSProfile: "test-profile-1",
				Region:     "us-west-2",
				CreatedAt:  time.Now(),
			},
			{
				Type:            "invitation",
				Name:            "Test Invitation",
				AWSProfile:      "test-invitation",
				InvitationToken: "inv-test123",
				OwnerAccount:    "test-account",
				Region:          "us-east-1",
				CreatedAt:       time.Now(),
			},
		},
		Metadata: map[string]string{
			"app_version": "0.4.2",
		},
	}

	// Write test profiles.json
	jsonPath := filepath.Join(tempDir, "profiles.json")
	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal export data: %v", err)
	}

	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write profiles.json: %v", err)
	}

	// Create mock profile manager
	mockManager := &MockProfileManager{}

	// Test import from JSON
	result, err := ImportProfiles(mockManager, jsonPath, DefaultImportOptions())
	if err != nil {
		t.Fatalf("ImportProfiles failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Import not successful: %s", result.Error)
	}

	if result.ProfilesImported != 2 {
		t.Errorf("Expected 2 profiles imported, got %d", result.ProfilesImported)
	}
}

// MockProfileManager implements necessary methods for testing
type MockProfileManager struct {
	profiles map[string]profile.Profile
}

func (m *MockProfileManager) GetProfileCredentials(profileID string) (*profile.Credentials, error) {
	return &profile.Credentials{}, nil
}

func (m *MockProfileManager) StoreProfileCredentials(profileID string, creds *profile.Credentials) error {
	return nil
}

func (m *MockProfileManager) AddProfile(p profile.Profile) error {
	if m.profiles == nil {
		m.profiles = make(map[string]profile.Profile)
	}
	m.profiles[p.AWSProfile] = p
	return nil
}

func (m *MockProfileManager) ProfileExists(profileID string) bool {
	if m.profiles == nil {
		return false
	}
	_, exists := m.profiles[profileID]
	return exists
}