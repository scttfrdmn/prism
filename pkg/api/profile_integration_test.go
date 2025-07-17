package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type mockStateManager struct {
	states map[string]*types.State
}

func newMockStateManager() *mockStateManager {
	return &mockStateManager{
		states: make(map[string]*types.State),
	}
}

func (m *mockStateManager) GetState(profileID string) (*types.State, error) {
	state, exists := m.states[profileID]
	if !exists {
		return &types.State{
			Instances:  make(map[string]types.Instance),
			Volumes:    make(map[string]types.EFSVolume),
			EBSVolumes: make(map[string]types.EBSVolume),
			Config:     types.Config{},
		}, nil
	}
	return state, nil
}

func (m *mockStateManager) SaveState(profileID string, state *types.State) error {
	m.states[profileID] = state
	return nil
}

func (m *mockStateManager) DeleteState(profileID string) error {
	delete(m.states, profileID)
	return nil
}

type mockProfileManager struct {
	profiles       map[string]profile.Profile
	currentProfile string
}

func newMockProfileManager() *mockProfileManager {
	profiles := map[string]profile.Profile{
		"personal": {
			Type:       profile.ProfileTypePersonal,
			Name:       "Personal Profile",
			AWSProfile: "default",
			Region:     "us-west-2",
		},
		"work": {
			Type:       profile.ProfileTypePersonal,
			Name:       "Work Profile",
			AWSProfile: "work",
			Region:     "us-east-1",
		},
		"invitation": {
			Type:            profile.ProfileTypeInvitation,
			Name:            "Invitation Profile",
			Region:          "eu-west-1",
			InvitationToken: "test-token",
			OwnerAccount:    "test-account",
			S3ConfigPath:    "s3://test/path",
		},
	}
	
	return &mockProfileManager{
		profiles:       profiles,
		currentProfile: "personal",
	}
}

func (m *mockProfileManager) GetCurrentProfile() (*profile.Profile, error) {
	prof, exists := m.profiles[m.currentProfile]
	if !exists {
		return nil, profile.ErrProfileNotFound
	}
	return &prof, nil
}

func (m *mockProfileManager) GetProfile(id string) (*profile.Profile, error) {
	prof, exists := m.profiles[id]
	if !exists {
		return nil, profile.ErrProfileNotFound
	}
	return &prof, nil
}

func (m *mockProfileManager) ListProfiles() ([]profile.Profile, error) {
	result := make([]profile.Profile, 0, len(m.profiles))
	for _, prof := range m.profiles {
		result = append(result, prof)
	}
	return result, nil
}

func (m *mockProfileManager) SwitchProfile(id string) error {
	if _, exists := m.profiles[id]; !exists {
		return profile.ErrProfileNotFound
	}
	m.currentProfile = id
	return nil
}

func (m *mockProfileManager) AddProfile(prof profile.Profile) error {
	m.profiles[prof.AWSProfile] = prof
	return nil
}

func (m *mockProfileManager) UpdateProfile(id string, updates profile.Profile) error {
	if _, exists := m.profiles[id]; !exists {
		return profile.ErrProfileNotFound
	}
	m.profiles[id] = updates
	return nil
}

func (m *mockProfileManager) RemoveProfile(id string) error {
	if _, exists := m.profiles[id]; !exists {
		return profile.ErrProfileNotFound
	}
	delete(m.profiles, id)
	return nil
}

func (m *mockProfileManager) StoreProfileCredentials(profileID string, creds *profile.Credentials) error {
	return nil
}

type mockProfileAwareStateManager struct {
	stateManager   *mockStateManager
	profileManager *mockProfileManager
}

func newMockProfileAwareStateManager(stateManager *mockStateManager, profileManager *mockProfileManager) *mockProfileAwareStateManager {
	return &mockProfileAwareStateManager{
		stateManager:   stateManager,
		profileManager: profileManager,
	}
}

func (m *mockProfileAwareStateManager) GetCurrentState() (*types.State, error) {
	prof, err := m.profileManager.GetCurrentProfile()
	if err != nil {
		return nil, err
	}
	return m.stateManager.GetState(prof.AWSProfile)
}

func (m *mockProfileAwareStateManager) GetProfileState(profileID string) (*types.State, error) {
	prof, err := m.profileManager.GetProfile(profileID)
	if err != nil {
		return nil, err
	}
	return m.stateManager.GetState(prof.AWSProfile)
}

func (m *mockProfileAwareStateManager) SaveCurrentState(state *types.State) error {
	prof, err := m.profileManager.GetCurrentProfile()
	if err != nil {
		return err
	}
	return m.stateManager.SaveState(prof.AWSProfile, state)
}

func (m *mockProfileAwareStateManager) SaveProfileState(profileID string, state *types.State) error {
	prof, err := m.profileManager.GetProfile(profileID)
	if err != nil {
		return err
	}
	return m.stateManager.SaveState(prof.AWSProfile, state)
}

// Tests for ProfileAwareClient
func TestProfileAwareClient(t *testing.T) {
	// Create a test server that checks for profile-specific headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/test" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()
	
	// Create mock components
	stateManager := newMockStateManager()
	profileManager := newMockProfileManager()
	profileStateManager := newMockProfileAwareStateManager(stateManager, profileManager)
	
	// Create profile-aware client
	client, err := NewProfileAwareClient(server.URL, profileManager, profileStateManager)
	require.NoError(t, err)
	
	// Test initial state
	assert.Equal(t, "personal", client.CurrentProfile())
	
	t.Run("GetCurrentProfile", func(t *testing.T) {
		profile, err := profileManager.GetCurrentProfile()
		require.NoError(t, err)
		assert.Equal(t, "default", profile.AWSProfile)
	})
	
	t.Run("ListProfiles", func(t *testing.T) {
		profiles, err := client.ListProfiles()
		require.NoError(t, err)
		assert.Equal(t, 3, len(profiles))
	})
	
	t.Run("SwitchProfile", func(t *testing.T) {
		// Switch to work profile
		err := client.SwitchProfile("work")
		require.NoError(t, err)
		
		// Check that profile was switched
		assert.Equal(t, "work", client.CurrentProfile())
		
		// Get current profile
		profile, err := profileManager.GetCurrentProfile()
		require.NoError(t, err)
		assert.Equal(t, "work", profile.AWSProfile)
		
		// Try to switch to non-existent profile
		err = client.SwitchProfile("non-existent")
		assert.Error(t, err)
		
		// Profile should not have changed
		assert.Equal(t, "work", client.CurrentProfile())
	})
	
	t.Run("WithProfile", func(t *testing.T) {
		// Get client for invitation profile
		invitationClient, err := client.WithProfile("invitation")
		require.NoError(t, err)
		
		// Current client should not change
		assert.Equal(t, "work", client.CurrentProfile())
		
		// Test that invitation client works
		ctx := context.Background()
		err = invitationClient.Ping(ctx)
		require.NoError(t, err)
		
		// Try non-existent profile
		_, err = client.WithProfile("non-existent")
		assert.Error(t, err)
	})
	
	t.Run("WithProfileContext", func(t *testing.T) {
		// Create context with current profile
		ctx := context.Background()
		ctxWithProfile, err := client.WithProfileContext(ctx)
		require.NoError(t, err)
		
		// Check that profile is in context
		profile, ok := profile.GetProfileFromContext(ctxWithProfile)
		assert.True(t, ok)
		assert.Equal(t, "work", profile.AWSProfile)
	})
	
	t.Run("ProfileOperations", func(t *testing.T) {
		// Get profile
		prof, err := client.GetProfile("personal")
		require.NoError(t, err)
		assert.Equal(t, "Personal Profile", prof.Name)
		
		// Add profile
		newProf := profile.Profile{
			Type:       profile.ProfileTypePersonal,
			Name:       "New Profile",
			AWSProfile: "new",
			Region:     "ap-south-1",
		}
		err = client.AddProfile(newProf)
		require.NoError(t, err)
		
		// Check that profile was added
		addedProf, err := client.GetProfile("new")
		require.NoError(t, err)
		assert.Equal(t, "New Profile", addedProf.Name)
		
		// Update profile
		updates := profile.Profile{
			Type:       profile.ProfileTypePersonal,
			Name:       "Updated Profile",
			AWSProfile: "new",
			Region:     "ap-south-1",
		}
		err = client.UpdateProfile("new", updates)
		require.NoError(t, err)
		
		// Check that profile was updated
		updatedProf, err := client.GetProfile("new")
		require.NoError(t, err)
		assert.Equal(t, "Updated Profile", updatedProf.Name)
		
		// Remove profile
		err = client.RemoveProfile("new")
		require.NoError(t, err)
		
		// Check that profile was removed
		_, err = client.GetProfile("new")
		assert.Error(t, err)
	})
}

func TestProfileAwareClient_Client(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/status" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}
	}))
	defer server.Close()
	
	// Create mock components
	stateManager := newMockStateManager()
	profileManager := newMockProfileManager()
	profileStateManager := newMockProfileAwareStateManager(stateManager, profileManager)
	
	// Create profile-aware client
	client, err := NewProfileAwareClient(server.URL, profileManager, profileStateManager)
	require.NoError(t, err)
	
	// Get CloudWorkstationAPI client
	apiClient := client.Client()
	require.NotNil(t, apiClient)
	
	// Test API method
	ctx := context.Background()
	err = apiClient.Ping(ctx)
	require.NoError(t, err)
}