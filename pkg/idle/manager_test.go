package idle

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewManager tests creation of a new idle manager.
func TestNewManager(t *testing.T) {
	// Create temporary directory for config and logs
	tempDir, err := ioutil.TempDir("", "idle-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test environment
	setupTestEnvironment(t, tempDir)

	// Create manager with custom paths
	manager, err := createTestManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Check that idle detection is enabled by default
	if !manager.IsEnabled() {
		t.Fatal("Expected idle detection to be enabled by default")
	}

	// Check that default profiles exist
	profiles := manager.GetProfiles()
	for name := range DefaultProfiles {
		if _, ok := profiles[name]; !ok {
			t.Errorf("Expected profile %q to exist", name)
		}
	}
}

// TestEnableDisable tests enabling and disabling idle detection.
func TestEnableDisable(t *testing.T) {
	// Create temporary directory for config and logs
	tempDir, err := ioutil.TempDir("", "idle-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test environment
	setupTestEnvironment(t, tempDir)

	// Create manager with custom paths
	manager, err := createTestManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Disable idle detection
	if err := manager.Disable(); err != nil {
		t.Fatalf("Failed to disable idle detection: %v", err)
	}

	if manager.IsEnabled() {
		t.Fatal("Expected idle detection to be disabled")
	}

	// Enable idle detection
	if err := manager.Enable(); err != nil {
		t.Fatalf("Failed to enable idle detection: %v", err)
	}

	if !manager.IsEnabled() {
		t.Fatal("Expected idle detection to be enabled")
	}
}

// TestProfileManagement tests profile management.
func TestProfileManagement(t *testing.T) {
	// Create temporary directory for config and logs
	tempDir, err := ioutil.TempDir("", "idle-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test environment
	setupTestEnvironment(t, tempDir)

	// Create manager with custom paths
	manager, err := createTestManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Get default profile
	defaultProfile, err := manager.GetDefaultProfile()
	if err != nil {
		t.Fatalf("Failed to get default profile: %v", err)
	}

	if defaultProfile.Name != "standard" {
		t.Errorf("Expected default profile to be 'standard', got %q", defaultProfile.Name)
	}

	// Add a new profile
	newProfile := Profile{
		Name:            "test",
		CPUThreshold:    5.0,
		MemoryThreshold: 20.0,
		NetworkThreshold: 30.0,
		DiskThreshold:    40.0,
		GPUThreshold:     2.0,
		IdleMinutes:      45,
		Action:           Stop,
		Notification:     true,
	}

	if err := manager.AddProfile(newProfile); err != nil {
		t.Fatalf("Failed to add profile: %v", err)
	}

	// Get the profile
	profile, err := manager.GetProfile("test")
	if err != nil {
		t.Fatalf("Failed to get profile: %v", err)
	}

	if profile.Name != "test" {
		t.Errorf("Expected profile name 'test', got %q", profile.Name)
	}

	if profile.CPUThreshold != 5.0 {
		t.Errorf("Expected CPU threshold 5.0, got %f", profile.CPUThreshold)
	}

	if profile.IdleMinutes != 45 {
		t.Errorf("Expected idle minutes 45, got %d", profile.IdleMinutes)
	}

	// Set as default profile
	if err := manager.SetDefaultProfile("test"); err != nil {
		t.Fatalf("Failed to set default profile: %v", err)
	}

	defaultProfile, err = manager.GetDefaultProfile()
	if err != nil {
		t.Fatalf("Failed to get default profile: %v", err)
	}

	if defaultProfile.Name != "test" {
		t.Errorf("Expected default profile to be 'test', got %q", defaultProfile.Name)
	}

	// Remove profile
	if err := manager.RemoveProfile("test"); err != nil {
		t.Fatalf("Failed to remove profile: %v", err)
	}

	// Try to get removed profile
	_, err = manager.GetProfile("test")
	if err == nil {
		t.Fatal("Expected error when getting removed profile, got nil")
	}

	// Try to remove built-in profile
	if err := manager.RemoveProfile("standard"); err == nil {
		t.Fatal("Expected error when removing built-in profile, got nil")
	}
}

// TestDomainMappings tests domain-to-profile mappings.
func TestDomainMappings(t *testing.T) {
	// Create temporary directory for config and logs
	tempDir, err := ioutil.TempDir("", "idle-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test environment
	setupTestEnvironment(t, tempDir)

	// Create manager with custom paths
	manager, err := createTestManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Check default domain mappings
	mappings := manager.GetDomainMappings()
	if len(mappings) == 0 {
		t.Fatal("Expected default domain mappings to exist")
	}

	// Set domain mapping
	if err := manager.SetDomainMapping("test-domain", "standard"); err != nil {
		t.Fatalf("Failed to set domain mapping: %v", err)
	}

	mappings = manager.GetDomainMappings()
	if mappings["test-domain"] != "standard" {
		t.Errorf("Expected domain mapping for 'test-domain' to be 'standard', got %q", mappings["test-domain"])
	}

	// Remove domain mapping
	if err := manager.RemoveDomainMapping("test-domain"); err != nil {
		t.Fatalf("Failed to remove domain mapping: %v", err)
	}

	mappings = manager.GetDomainMappings()
	if _, ok := mappings["test-domain"]; ok {
		t.Fatal("Expected domain mapping to be removed")
	}
}

// TestInstanceOverrides tests instance overrides.
func TestInstanceOverrides(t *testing.T) {
	// Create temporary directory for config and logs
	tempDir, err := ioutil.TempDir("", "idle-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test environment
	setupTestEnvironment(t, tempDir)

	// Create manager with custom paths
	manager, err := createTestManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create instance override
	cpuThreshold := 3.0
	memThreshold := 15.0
	idleMinutes := 10
	action := Hibernate
	notification := false

	override := InstanceOverride{
		Profile:          "batch",
		CPUThreshold:     &cpuThreshold,
		MemoryThreshold:  &memThreshold,
		IdleMinutes:      &idleMinutes,
		Action:           &action,
		Notification:     &notification,
	}

	// Set instance override
	if err := manager.SetInstanceOverride("test-instance", override); err != nil {
		t.Fatalf("Failed to set instance override: %v", err)
	}

	// Get instance override
	overrides := manager.GetInstanceOverrides()
	if _, ok := overrides["test-instance"]; !ok {
		t.Fatal("Expected instance override to exist")
	}

	// Check values
	instanceOverride, ok := manager.GetInstanceOverride("test-instance")
	if !ok {
		t.Fatal("Expected to find instance override")
	}

	if instanceOverride.Profile != "batch" {
		t.Errorf("Expected profile 'batch', got %q", instanceOverride.Profile)
	}

	if *instanceOverride.CPUThreshold != cpuThreshold {
		t.Errorf("Expected CPU threshold %f, got %f", cpuThreshold, *instanceOverride.CPUThreshold)
	}

	if *instanceOverride.IdleMinutes != idleMinutes {
		t.Errorf("Expected idle minutes %d, got %d", idleMinutes, *instanceOverride.IdleMinutes)
	}

	// Remove instance override
	if err := manager.RemoveInstanceOverride("test-instance"); err != nil {
		t.Fatalf("Failed to remove instance override: %v", err)
	}

	// Check that override was removed
	_, ok = manager.GetInstanceOverride("test-instance")
	if ok {
		t.Fatal("Expected instance override to be removed")
	}
}

// TestProcessMetrics tests processing metrics.
func TestProcessMetrics(t *testing.T) {
	// Create temporary directory for config and logs
	tempDir, err := ioutil.TempDir("", "idle-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test environment
	setupTestEnvironment(t, tempDir)

	// Create manager with custom paths
	manager, err := createTestManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create test metrics
	now := time.Now()
	metrics := &UsageMetrics{
		Timestamp:   now,
		CPU:         2.0,  // Below threshold (10.0)
		Memory:      15.0, // Below threshold (30.0)
		Network:     20.0, // Below threshold (50.0)
		Disk:        30.0, // Below threshold (100.0)
		GPU:         nil,
		HasActivity: false,
	}

	// Process metrics (first time)
	state, err := manager.ProcessMetrics("i-12345678", "test-instance", metrics)
	if err != nil {
		t.Fatalf("Failed to process metrics: %v", err)
	}

	// Check state
	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	if state.InstanceID != "i-12345678" {
		t.Errorf("Expected instance ID 'i-12345678', got %q", state.InstanceID)
	}

	if state.InstanceName != "test-instance" {
		t.Errorf("Expected instance name 'test-instance', got %q", state.InstanceName)
	}

	if state.Profile != "standard" {
		t.Errorf("Expected profile 'standard', got %q", state.Profile)
	}

	if !state.IsIdle {
		t.Fatal("Expected instance to be idle")
	}

	if state.IdleSince == nil {
		t.Fatal("Expected non-nil idle since time")
	}

	if state.NextAction == nil {
		t.Fatal("Expected non-nil next action")
	}

	if state.NextAction.Action != Stop {
		t.Errorf("Expected next action to be Stop, got %v", state.NextAction.Action)
	}

	// Check that next action time is correct
	expectedActionTime := now.Add(time.Duration(30) * time.Minute)
	if state.NextAction.Time.Unix() != expectedActionTime.Unix() {
		t.Errorf("Expected next action time %v, got %v", expectedActionTime, state.NextAction.Time)
	}

	// Process metrics with activity
	metrics.HasActivity = true
	metrics.Timestamp = now.Add(5 * time.Minute)

	state, err = manager.ProcessMetrics("i-12345678", "test-instance", metrics)
	if err != nil {
		t.Fatalf("Failed to process metrics: %v", err)
	}

	// Check state - should not be idle now
	if state.IsIdle {
		t.Fatal("Expected instance to not be idle")
	}

	if state.IdleSince != nil {
		t.Fatal("Expected nil idle since time")
	}

	if state.NextAction != nil {
		t.Fatal("Expected nil next action")
	}

	// Process metrics with high CPU
	metrics.HasActivity = false
	metrics.CPU = 20.0 // Above threshold
	metrics.Timestamp = now.Add(10 * time.Minute)

	state, err = manager.ProcessMetrics("i-12345678", "test-instance", metrics)
	if err != nil {
		t.Fatalf("Failed to process metrics: %v", err)
	}

	// Check state - should not be idle due to high CPU
	if state.IsIdle {
		t.Fatal("Expected instance to not be idle")
	}

	// Test with instance override
	cpuThreshold := 30.0 // Higher than metric value
	override := InstanceOverride{
		Profile:     "standard",
		CPUThreshold: &cpuThreshold,
	}

	if err := manager.SetInstanceOverride("test-instance", override); err != nil {
		t.Fatalf("Failed to set instance override: %v", err)
	}

	// Process metrics again - should be idle now with override
	state, err = manager.ProcessMetrics("i-12345678", "test-instance", metrics)
	if err != nil {
		t.Fatalf("Failed to process metrics: %v", err)
	}

	if !state.IsIdle {
		t.Fatal("Expected instance to be idle with override")
	}
}

// setupTestEnvironment sets up a test environment with custom paths.
func setupTestEnvironment(t *testing.T, tempDir string) {
	// Create config directory
	configDir := filepath.Join(tempDir, ConfigDirName)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Create log directory
	logDir := filepath.Join(configDir, IdleLogDirName)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}
}

// createTestManager creates a new manager with custom paths.
func createTestManager(tempDir string) (*Manager, error) {
	configDir := filepath.Join(tempDir, ConfigDirName)
	logDir := filepath.Join(configDir, IdleLogDirName)
	configPath := filepath.Join(configDir, IdleConfigFileName)
	historyPath := filepath.Join(configDir, IdleHistoryFileName)
	logPath := filepath.Join(logDir, IdleActionsLogName)

	manager := &Manager{
		configPath:  configPath,
		historyPath: historyPath,
		logDirPath:  logDir,
		logPath:     logPath,
		config: &Config{
			Enabled:        true,
			DefaultProfile: "standard",
			Profiles:       DefaultProfiles,
			DomainMappings: DefaultDomainMappings,
			InstanceOverrides: make(map[string]InstanceOverride),
		},
		history: &History{
			Entries: []HistoryEntry{},
		},
		states: make(map[string]*IdleState),
	}

	return manager, nil
}