package idle

import (
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClock implements the Clock interface for testing
type mockClock struct {
	currentTime time.Time
}

func (m *mockClock) Now() time.Time {
	return m.currentTime
}

func (m *mockClock) Add(d time.Duration) {
	m.currentTime = m.currentTime.Add(d)
}

// TestNewManager tests creating a new idle manager
func TestNewManager(t *testing.T) {
	manager, err := NewManager()
	require.NoError(t, err)
	assert.NotNil(t, manager)
	
	// Check default values
	assert.True(t, manager.IsEnabled())
	
	profile, err := manager.GetDefaultProfile()
	require.NoError(t, err)
	assert.Equal(t, "standard", profile.Name)
}

// TestManagerEnableDisable tests enabling and disabling idle detection
func TestManagerEnableDisable(t *testing.T) {
	manager, err := NewManager()
	require.NoError(t, err)
	
	// Default is enabled
	assert.True(t, manager.IsEnabled())
	
	// Disable
	err = manager.Disable()
	require.NoError(t, err)
	assert.False(t, manager.IsEnabled())
	
	// Enable again
	err = manager.Enable()
	require.NoError(t, err)
	assert.True(t, manager.IsEnabled())
}

// TestManagerProfiles tests profile management
func TestManagerProfiles(t *testing.T) {
	manager, err := NewManager()
	require.NoError(t, err)
	
	// Get initial profiles
	initialProfiles := manager.GetProfiles()
	assert.NotEmpty(t, initialProfiles)
	assert.Contains(t, initialProfiles, "standard")
	
	// Add a new profile
	newProfile := Profile{
		Name:            "test-profile",
		CPUThreshold:    5.0,
		MemoryThreshold: 20.0,
		NetworkThreshold: 30.0,
		DiskThreshold:    50.0,
		GPUThreshold:     2.0,
		IdleMinutes:      15,
		Action:           Hibernate,
		Notification:     true,
	}
	
	err = manager.AddProfile(newProfile)
	require.NoError(t, err)
	
	// Verify profile was added
	profiles := manager.GetProfiles()
	assert.Contains(t, profiles, "test-profile")
	
	// Set default profile
	err = manager.SetDefaultProfile("test-profile")
	require.NoError(t, err)
	
	defaultProfile, err := manager.GetDefaultProfile()
	require.NoError(t, err)
	assert.Equal(t, "test-profile", defaultProfile.Name)
	
	// Remove profile
	err = manager.RemoveProfile("test-profile")
	require.NoError(t, err)
	
	// Verify profile was removed
	profiles = manager.GetProfiles()
	assert.NotContains(t, profiles, "test-profile")
	
	// Default should revert to standard
	defaultProfile, err = manager.GetDefaultProfile()
	require.NoError(t, err)
	assert.Equal(t, "standard", defaultProfile.Name)
}

// TestManagerDomainMappings tests domain mapping functionality
func TestManagerDomainMappings(t *testing.T) {
	manager, err := NewManager()
	require.NoError(t, err)
	
	// Initial mappings
	initialMappings := manager.GetDomainMappings()
	
	// Set a domain mapping
	err = manager.SetDomainMapping("test-domain", "standard")
	require.NoError(t, err)
	
	// Verify mapping was set
	mappings := manager.GetDomainMappings()
	assert.Equal(t, "standard", mappings["test-domain"])
	
	// Remove mapping
	err = manager.RemoveDomainMapping("test-domain")
	require.NoError(t, err)
	
	// Verify mapping was removed
	mappings = manager.GetDomainMappings()
	assert.NotContains(t, mappings, "test-domain")
}

// TestManagerInstanceOverrides tests instance override functionality
func TestManagerInstanceOverrides(t *testing.T) {
	manager, err := NewManager()
	require.NoError(t, err)
	
	// Initial overrides
	initialOverrides := manager.GetInstanceOverrides()
	
	// Create test values
	cpuThreshold := 15.0
	idleMinutes := 45
	action := Stop
	notification := false
	
	// Set an instance override
	override := InstanceOverride{
		Profile:        "standard",
		CPUThreshold:   &cpuThreshold,
		IdleMinutes:    &idleMinutes,
		Action:         &action,
		Notification:   &notification,
	}
	
	err = manager.SetInstanceOverride("test-instance", override)
	require.NoError(t, err)
	
	// Verify override was set
	overrides := manager.GetInstanceOverrides()
	assert.Contains(t, overrides, "test-instance")
	assert.Equal(t, "standard", overrides["test-instance"].Profile)
	assert.Equal(t, cpuThreshold, *overrides["test-instance"].CPUThreshold)
	assert.Equal(t, idleMinutes, *overrides["test-instance"].IdleMinutes)
	
	// Remove override
	err = manager.RemoveInstanceOverride("test-instance")
	require.NoError(t, err)
	
	// Verify override was removed
	overrides = manager.GetInstanceOverrides()
	assert.NotContains(t, overrides, "test-instance")
}

// TestManagerIdleDetection tests idle detection functionality
func TestManagerIdleDetection(t *testing.T) {
	manager, err := NewManager()
	require.NoError(t, err)
	
	// Override clock for testing
	clock := &mockClock{currentTime: time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)}
	manager.clock = clock
	
	// Create test instance
	instanceID := "i-test12345"
	instanceName := "test-instance"
	
	// Initialize metrics - active
	activeMetrics := Metrics{
		CPU:         20.0, // Above threshold
		Memory:      40.0, // Above threshold
		Network:     60.0, // Above threshold
		Disk:        120.0, // Above threshold
		HasActivity: true,
		Timestamp:   clock.Now(),
	}
	
	// Update metrics
	manager.UpdateMetrics(instanceID, instanceName, activeMetrics)
	
	// Get idle state
	state := manager.GetIdleState(instanceID)
	require.NotNil(t, state)
	assert.Equal(t, instanceName, state.InstanceName)
	assert.Equal(t, instanceID, state.InstanceID)
	assert.False(t, state.IsIdle)
	assert.Nil(t, state.IdleSince)
	
	// Now report metrics below thresholds
	idleMetrics := Metrics{
		CPU:         5.0, // Below threshold
		Memory:      20.0, // Below threshold
		Network:     30.0, // Below threshold
		Disk:        50.0, // Below threshold
		HasActivity: false,
		Timestamp:   clock.Now(),
	}
	
	// Update metrics
	manager.UpdateMetrics(instanceID, instanceName, idleMetrics)
	
	// Should not be idle yet (need consistent metrics below threshold)
	state = manager.GetIdleState(instanceID)
	assert.False(t, state.IsIdle)
	
	// Advance time and send more idle metrics
	clock.Add(5 * time.Minute)
	idleMetrics.Timestamp = clock.Now()
	manager.UpdateMetrics(instanceID, instanceName, idleMetrics)
	
	clock.Add(10 * time.Minute)
	idleMetrics.Timestamp = clock.Now()
	manager.UpdateMetrics(instanceID, instanceName, idleMetrics)
	
	// Should be idle now
	state = manager.GetIdleState(instanceID)
	assert.True(t, state.IsIdle)
	assert.NotNil(t, state.IdleSince)
	
	// Send active metrics again
	clock.Add(5 * time.Minute)
	activeMetrics.Timestamp = clock.Now()
	manager.UpdateMetrics(instanceID, instanceName, activeMetrics)
	
	// Should no longer be idle
	state = manager.GetIdleState(instanceID)
	assert.False(t, state.IsIdle)
	assert.Nil(t, state.IdleSince)
}