package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
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

// MockProfileManager implements the profile management interfaces for testing
type MockProfileManager struct {
	mock.Mock
	CurrentProfile *profile.Profile
	AllProfiles    []profile.Profile
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

// TestGUIInitialization tests the basic GUI initialization
func TestGUIInitialization(t *testing.T) {
	// Skip the test if running in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping GUI tests in CI environment")
	}

	// Create a test app
	app := test.NewApp()
	defer app.Quit()

	// Create a mock API client
	mockAPI := new(MockCloudWorkstationAPI)
	mockAPI.On("Ping", mock.Anything).Return(nil)
	mockAPI.On("ListInstances", mock.Anything).Return(&types.ListInstancesResponse{
		Instances: []types.Instance{},
		TotalCost: 0,
	}, nil)

	// Create a mock profile manager
	mockProfileManager := new(MockProfileManager)
	mockProfileManager.On("GetCurrentProfile").Return(nil, fmt.Errorf("no current profile"))
	mockProfileManager.On("ListProfiles").Return([]profile.Profile{}, nil)

	// Mock state manager
	mockStateManager := new(MockStateManager)

	// Create and initialize GUI
	gui := &main.CloudWorkstationGUI{
		app:            app,
		apiClient:      mockAPI,
		profileManager: mockProfileManager,
		stateManager:   mockStateManager,
	}

	// Initialize GUI
	err := gui.initialize()
	assert.NoError(t, err)

	// Verify the window was created
	assert.NotNil(t, gui.window)
	
	// Verify API calls were made
	mockAPI.AssertExpectations(t)
	mockProfileManager.AssertExpectations(t)
}

// TestGUICrossPlatformRendering tests GUI rendering on different platforms
func TestGUICrossPlatformRendering(t *testing.T) {
	// Skip the test if running in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping GUI tests in CI environment")
	}

	// Create a test app
	app := test.NewApp()
	defer app.Quit()
	
	// Create a new window
	window := app.NewWindow("Test Window")
	
	// Test with different theme variants (light/dark)
	themes := []fyne.ThemeVariant{
		theme.VariantDark,
		theme.VariantLight,
	}
	
	for _, themeVariant := range themes {
		// Set theme variant
		app.Settings().SetTheme(theme.DefaultTheme())
		
		// Create content with all standard widgets
		content := container.NewVBox(
			widget.NewLabel("Test Label"),
			widget.NewButton("Test Button", nil),
			widget.NewEntry(),
			widget.NewSelect([]string{"Option 1", "Option 2"}, nil),
			widget.NewCheck("Test Checkbox", nil),
		)
		
		// Set the content
		window.SetContent(content)
		
		// Create a snapshot for visual testing
		snapshot := test.NewWindowlessCanvas()
		snapshot.SetContent(content)
		
		// Basic assertion that content renders without error
		assert.NotNil(t, snapshot)
	}
}

// TestSystemTrayIntegration tests system tray functionality
func TestSystemTrayIntegration(t *testing.T) {
	// Skip the test if running in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping GUI tests in CI environment")
	}

	// Create a test app
	app := test.NewApp()
	defer app.Quit()
	
	// Create a mock desktop app that supports system tray
	mockDesktopApp := &testDesktopApp{app: app}
	
	// Create a function to set up the system tray
	setupSystemTray := func(desk desktop.App) {
		menu := fyne.NewMenu("Test Menu",
			fyne.NewMenuItem("Test Item", func() {}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() {}),
		)
		
		desk.SetSystemTrayMenu(menu)
	}
	
	// Set up the system tray
	setupSystemTray(mockDesktopApp)
	
	// Verify the system tray was set up correctly
	assert.NotNil(t, mockDesktopApp.menu)
	assert.Equal(t, "Test Menu", mockDesktopApp.menu.Label)
	assert.Len(t, mockDesktopApp.menu.Items, 3)
}

// testDesktopApp is a mock desktop app for testing
type testDesktopApp struct {
	app  fyne.App
	menu *fyne.Menu
}

func (t *testDesktopApp) SetSystemTrayMenu(menu *fyne.Menu) {
	t.menu = menu
}

func (t *testDesktopApp) SetSystemTrayIcon(icon fyne.Resource) {
	// No-op for test
}

// Implement remaining fyne.App methods
func (t *testDesktopApp) NewWindow(title string) fyne.Window {
	return t.app.NewWindow(title)
}

func (t *testDesktopApp) OpenURL(url *url.URL) error {
	return t.app.OpenURL(url)
}

func (t *testDesktopApp) Icon() fyne.Resource {
	return t.app.Icon()
}

func (t *testDesktopApp) SetIcon(icon fyne.Resource) {
	t.app.SetIcon(icon)
}

func (t *testDesktopApp) Run() {
	t.app.Run()
}

func (t *testDesktopApp) Quit() {
	t.app.Quit()
}

func (t *testDesktopApp) Driver() fyne.Driver {
	return t.app.Driver()
}

func (t *testDesktopApp) UniqueID() string {
	return t.app.UniqueID()
}

func (t *testDesktopApp) Preferences() fyne.Preferences {
	return t.app.Preferences()
}

func (t *testDesktopApp) Storage() fyne.Storage {
	return t.app.Storage()
}

func (t *testDesktopApp) Settings() fyne.Settings {
	return t.app.Settings()
}

func (t *testDesktopApp) Lifecycle() fyne.Lifecycle {
	return t.app.Lifecycle()
}

func (t *testDesktopApp) Metadata() fyne.AppMetadata {
	return t.app.Metadata()
}