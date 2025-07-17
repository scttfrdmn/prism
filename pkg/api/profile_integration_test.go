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

// IntegrationTestServer creates a test server for integration testing
func IntegrationTestServer() (*httptest.Server, *testServerState) {
	state := &testServerState{
		instances:   make(map[string]types.Instance),
		volumes:     make(map[string]types.EFSVolume),
		ebsVolumes:  make(map[string]types.EBSVolume),
		templates:   make(map[string]types.Template),
		profileData: make(map[string]map[string]interface{}),
	}
	
	// Initialize with some test data
	state.templates["python-research"] = types.Template{
		Name:        "python-research",
		Description: "Python data science environment",
		EstimatedCostPerHour: map[string]float64{
			"x86_64": 0.10,
			"arm64":  0.08,
		},
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract profile from request
		profileID := r.Header.Get("X-Profile-ID")
		if profileID == "" {
			// Use default profile if not specified
			profileID = "default"
		}
		
		// Initialize profile data if needed
		if _, exists := state.profileData[profileID]; !exists {
			state.profileData[profileID] = make(map[string]interface{})
			state.profileData[profileID]["instances"] = make(map[string]types.Instance)
			state.profileData[profileID]["volumes"] = make(map[string]types.EFSVolume)
			state.profileData[profileID]["ebsVolumes"] = make(map[string]types.EBSVolume)
		}
		
		// Extract AWS settings
		awsProfile := r.Header.Get("X-AWS-Profile")
		awsRegion := r.Header.Get("X-AWS-Region")
		
		// Extract invitation settings
		invitationToken := r.Header.Get("X-Invitation-Token")
		ownerAccount := r.Header.Get("X-Owner-Account")
		s3ConfigPath := r.Header.Get("X-S3-Config-Path")
		
		// Record the request
		state.lastProfileID = profileID
		state.lastAWSProfile = awsProfile
		state.lastAWSRegion = awsRegion
		state.lastInvitationToken = invitationToken
		state.lastOwnerAccount = ownerAccount
		state.lastS3ConfigPath = s3ConfigPath
		
		// Handle ping request
		if r.URL.Path == "/api/v1/ping" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		
		// Handle templates request
		if r.URL.Path == "/api/v1/templates" {
			w.WriteHeader(http.StatusOK)
			renderJSON(w, state.templates)
			return
		}
		
		// Handle instance listing
		if r.URL.Path == "/api/v1/instances" && r.Method == http.MethodGet {
			// Return instances for this profile
			profileInstances, ok := state.profileData[profileID]["instances"].(map[string]types.Instance)
			if !ok {
				profileInstances = make(map[string]types.Instance)
			}
			
			response := types.ListResponse{
				Instances: make([]types.Instance, 0),
			}
			
			for _, instance := range profileInstances {
				response.Instances = append(response.Instances, instance)
			}
			
			w.WriteHeader(http.StatusOK)
			renderJSON(w, response)
			return
		}
		
		// Handle instance creation
		if r.URL.Path == "/api/v1/instances" && r.Method == http.MethodPost {
			var req types.LaunchRequest
			if err := decodeJSON(r, &req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			
			// Create instance
			instance := types.Instance{
				Name:     req.Name,
				Template: req.Template,
				State:    "running",
				PublicIP: "203.0.113.10",
			}
			
			// Store instance for this profile
			profileInstances, ok := state.profileData[profileID]["instances"].(map[string]types.Instance)
			if !ok {
				profileInstances = make(map[string]types.Instance)
			}
			profileInstances[req.Name] = instance
			state.profileData[profileID]["instances"] = profileInstances
			
			// Return response
			response := types.LaunchResponse{
				Message:       "Instance launched successfully",
				EstimatedCost: "$0.10/hour",
				ConnectionInfo: "ssh ubuntu@203.0.113.10",
			}
			
			w.WriteHeader(http.StatusOK)
			renderJSON(w, response)
			return
		}
		
		// Handle 404 for unknown endpoints
		w.WriteHeader(http.StatusNotFound)
	}))
	
	return server, state
}

// testServerState tracks the state of the test server
type testServerState struct {
	instances   map[string]types.Instance
	volumes     map[string]types.EFSVolume
	ebsVolumes  map[string]types.EBSVolume
	templates   map[string]types.Template
	profileData map[string]map[string]interface{}
	
	// Last request information
	lastProfileID       string
	lastAWSProfile      string
	lastAWSRegion       string
	lastInvitationToken string
	lastOwnerAccount    string
	lastS3ConfigPath    string
}

// Integration test for profile switching
func TestProfileSwitchingIntegration(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Create integration test server
	server, serverState := IntegrationTestServer()
	defer server.Close()
	
	// Create profile manager and state manager
	profileManager := newMockProfileManager()
	stateManager := newMockStateManager()
	profileStateManager := newMockProfileAwareStateManager(stateManager, profileManager)
	
	// Create profile-aware client
	client, err := NewProfileAwareClient(server.URL, profileManager, profileStateManager)
	require.NoError(t, err)
	
	// Test switching between profiles
	t.Run("SwitchProfiles", func(t *testing.T) {
		// Create context
		ctx := context.Background()
		
		// Initial profile should be personal
		err := client.Ping(ctx)
		require.NoError(t, err)
		assert.Equal(t, "personal", serverState.lastProfileID)
		assert.Equal(t, "default", serverState.lastAWSProfile)
		assert.Equal(t, "us-west-2", serverState.lastAWSRegion)
		
		// Switch to work profile
		err = client.SwitchProfile("work")
		require.NoError(t, err)
		
		// Check that work profile is active
		err = client.Ping(ctx)
		require.NoError(t, err)
		assert.Equal(t, "work", serverState.lastProfileID)
		assert.Equal(t, "work", serverState.lastAWSProfile)
		assert.Equal(t, "us-east-1", serverState.lastAWSRegion)
		
		// Switch to invitation profile
		err = client.SwitchProfile("invitation")
		require.NoError(t, err)
		
		// Check that invitation profile is active
		err = client.Ping(ctx)
		require.NoError(t, err)
		assert.Equal(t, "invitation", serverState.lastProfileID)
		assert.Equal(t, "", serverState.lastAWSProfile) // No AWS profile for invitation
		assert.Equal(t, "eu-west-1", serverState.lastAWSRegion)
		assert.Equal(t, "test-token", serverState.lastInvitationToken)
		assert.Equal(t, "test-account", serverState.lastOwnerAccount)
	})
	
	// Test profile isolation
	t.Run("ProfileIsolation", func(t *testing.T) {
		// Create context
		ctx := context.Background()
		
		// Switch to personal profile
		err := client.SwitchProfile("personal")
		require.NoError(t, err)
		
		// Launch an instance in personal profile
		req := types.LaunchRequest{
			Template: "python-research",
			Name:     "personal-instance",
		}
		_, err = client.LaunchInstance(ctx, req)
		require.NoError(t, err)
		
		// List instances in personal profile
		personalResp, err := client.ListInstances(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, len(personalResp.Instances))
		assert.Equal(t, "personal-instance", personalResp.Instances[0].Name)
		
		// Switch to work profile
		err = client.SwitchProfile("work")
		require.NoError(t, err)
		
		// List instances in work profile
		workResp, err := client.ListInstances(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, len(workResp.Instances))
		
		// Launch an instance in work profile
		req = types.LaunchRequest{
			Template: "python-research",
			Name:     "work-instance",
		}
		_, err = client.LaunchInstance(ctx, req)
		require.NoError(t, err)
		
		// List instances in work profile again
		workResp, err = client.ListInstances(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, len(workResp.Instances))
		assert.Equal(t, "work-instance", workResp.Instances[0].Name)
		
		// Switch back to personal profile
		err = client.SwitchProfile("personal")
		require.NoError(t, err)
		
		// List instances in personal profile again
		personalResp, err = client.ListInstances(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, len(personalResp.Instances))
		assert.Equal(t, "personal-instance", personalResp.Instances[0].Name)
	})
	
	// Test temporary profile client
	t.Run("TemporaryProfileClient", func(t *testing.T) {
		// Create context
		ctx := context.Background()
		
		// Current profile should be personal
		err := client.Ping(ctx)
		require.NoError(t, err)
		assert.Equal(t, "personal", serverState.lastProfileID)
		
		// Get temporary client for work profile
		workClient, err := client.WithProfile("work")
		require.NoError(t, err)
		
		// Use temporary client
		err = workClient.Ping(ctx)
		require.NoError(t, err)
		assert.Equal(t, "work", serverState.lastProfileID)
		
		// Original client should still use personal profile
		err = client.Ping(ctx)
		require.NoError(t, err)
		assert.Equal(t, "personal", serverState.lastProfileID)
	})
}