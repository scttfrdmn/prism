package tests

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// MockCloudWorkstationAPI implements the CloudWorkstationAPI interface for testing
type MockCloudWorkstationAPI struct {
	mock.Mock
}

func (m *MockCloudWorkstationAPI) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCloudWorkstationAPI) ListInstances(ctx context.Context) (*types.ListInstancesResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*types.ListInstancesResponse), args.Error(1)
}

func (m *MockCloudWorkstationAPI) GetInstance(ctx context.Context, name string) (*types.Instance, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*types.Instance), args.Error(1)
}

func (m *MockCloudWorkstationAPI) LaunchInstance(ctx context.Context, req types.LaunchRequest) (*types.LaunchResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*types.LaunchResponse), args.Error(1)
}

func (m *MockCloudWorkstationAPI) StartInstance(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockCloudWorkstationAPI) StopInstance(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockCloudWorkstationAPI) DeleteInstance(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockCloudWorkstationAPI) ConnectInstance(ctx context.Context, name string) (*types.ConnectResponse, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*types.ConnectResponse), args.Error(1)
}

// MockProfileManager implements the profile.Manager interface for testing
type MockProfileManager struct {
	mock.Mock
}

func (m *MockProfileManager) GetCurrentProfile() (*profile.Profile, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profile.Profile), args.Error(1)
}

func (m *MockProfileManager) SetCurrentProfile(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockProfileManager) ListProfiles() ([]profile.Profile, error) {
	args := m.Called()
	return args.Get(0).([]profile.Profile), args.Error(1)
}

func (m *MockProfileManager) AddProfile(p profile.Profile) error {
	args := m.Called(p)
	return args.Error(0)
}

func (m *MockProfileManager) GetProfile(name string) (*profile.Profile, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profile.Profile), args.Error(1)
}

func (m *MockProfileManager) RemoveProfile(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// MockStateManager implements the profile.StateManager interface for testing
type MockStateManager struct {
	mock.Mock
}

func (m *MockStateManager) GetState() (*profile.State, error) {
	args := m.Called()
	return args.Get(0).(*profile.State), args.Error(1)
}

func (m *MockStateManager) SaveState(state *profile.State) error {
	args := m.Called(state)
	return args.Error(0)
}

// Helper function to create mock instances for testing
func CreateMockInstances() []types.Instance {
	return []types.Instance{
		{
			ID:                "i-12345",
			Name:              "test-instance-1",
			Template:          "r-research",
			InstanceType:      "t3.medium",
			State:             "running",
			PublicIP:          "1.2.3.4",
			LaunchTime:        time.Now().Add(-24 * time.Hour),
			EstimatedDailyCost: 2.40,
		},
		{
			ID:                "i-67890",
			Name:              "test-instance-2",
			Template:          "python-research",
			InstanceType:      "g4dn.xlarge",
			State:             "stopped",
			PublicIP:          "",
			LaunchTime:        time.Now().Add(-48 * time.Hour),
			EstimatedDailyCost: 0.00, // Stopped, so no cost
		},
	}
}

// Helper function to create mock profiles for testing
func CreateMockProfiles() []profile.Profile {
	return []profile.Profile{
		{
			Type:       "personal",
			Name:       "default",
			AWSProfile: "default",
			Region:     "us-west-2",
			CreatedAt:  time.Now().Add(-30 * 24 * time.Hour),
		},
		{
			Type:            "invitation",
			Name:            "research-project",
			AWSProfile:      "research-project",
			InvitationToken: "inv-12345",
			OwnerAccount:    "123456789012",
			S3ConfigPath:    "s3://cloudworkstation-invitations/12345",
			CreatedAt:       time.Now().Add(-7 * 24 * time.Hour),
			DeviceBound:     true,
			BindingRef:      "binding-12345",
		},
	}
}

// MockProfileAwareClient implements a mock for the profile-aware client
type MockProfileAwareClient struct {
	mock.Mock
	mockAPI *MockCloudWorkstationAPI
}

func NewMockProfileAwareClient() *MockProfileAwareClient {
	return &MockProfileAwareClient{
		mockAPI: new(MockCloudWorkstationAPI),
	}
}

func (m *MockProfileAwareClient) Client() api.CloudWorkstationAPI {
	return m.mockAPI
}

func (m *MockProfileAwareClient) SwitchProfile(profileName string) error {
	args := m.Called(profileName)
	return args.Error(0)
}

func (m *MockProfileAwareClient) WithProfile(profileName string) (api.CloudWorkstationAPI, error) {
	args := m.Called(profileName)
	return args.Get(0).(api.CloudWorkstationAPI), args.Error(1)
}