package daemon

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock state manager for testing
type MockStateManager struct {
	mock.Mock
}

func (m *MockStateManager) LoadState() (*types.State, error) {
	args := m.Called()
	return args.Get(0).(*types.State), args.Error(1)
}

func (m *MockStateManager) SaveState(state *types.State) error {
	args := m.Called(state)
	return args.Error(0)
}

func (m *MockStateManager) SaveAPIKey(apiKey string) error {
	args := m.Called(apiKey)
	return args.Error(0)
}

func (m *MockStateManager) GetAPIKey() (string, time.Time, error) {
	args := m.Called()
	return args.String(0), args.Get(1).(time.Time), args.Error(2)
}

func (m *MockStateManager) ClearAPIKey() error {
	args := m.Called()
	return args.Error(0)
}

// Test the auth middleware
func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		method         string
		apiKey         string
		stateAPIKey    string
		expectedStatus int
		authenticated  bool
	}{
		{
			name:           "No API key required when none configured",
			path:           "/api/v1/instances",
			method:         http.MethodGet,
			apiKey:         "",
			stateAPIKey:    "",
			expectedStatus: http.StatusOK,
			authenticated:  false,
		},
		{
			name:           "API key required when configured",
			path:           "/api/v1/instances",
			method:         http.MethodGet,
			apiKey:         "",
			stateAPIKey:    "test-key",
			expectedStatus: http.StatusUnauthorized,
			authenticated:  false,
		},
		{
			name:           "Valid API key",
			path:           "/api/v1/instances",
			method:         http.MethodGet,
			apiKey:         "test-key",
			stateAPIKey:    "test-key",
			expectedStatus: http.StatusOK,
			authenticated:  true,
		},
		{
			name:           "Invalid API key",
			path:           "/api/v1/instances",
			method:         http.MethodGet,
			apiKey:         "wrong-key",
			stateAPIKey:    "test-key",
			expectedStatus: http.StatusUnauthorized,
			authenticated:  false,
		},
		{
			name:           "Ping doesn't require auth",
			path:           "/api/v1/ping",
			method:         http.MethodGet,
			apiKey:         "",
			stateAPIKey:    "test-key",
			expectedStatus: http.StatusOK,
			authenticated:  false,
		},
		{
			name:           "Auth endpoint doesn't require auth for POST",
			path:           "/api/v1/auth",
			method:         http.MethodPost,
			apiKey:         "",
			stateAPIKey:    "test-key",
			expectedStatus: http.StatusOK,
			authenticated:  false,
		},
		{
			name:           "Auth endpoint requires auth for GET",
			path:           "/api/v1/auth",
			method:         http.MethodGet,
			apiKey:         "",
			stateAPIKey:    "test-key",
			expectedStatus: http.StatusUnauthorized,
			authenticated:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mock state manager
			mockStateManager := &MockStateManager{}
			mockStateManager.On("LoadState").Return(&types.State{
				Config: types.Config{
					APIKey: tt.stateAPIKey,
				},
			}, nil)

			// Create server with mock state manager
			server := &Server{
				stateManager: mockStateManager,
			}

			// Create test handler that checks authentication status
			var isAuth bool
			handler := func(w http.ResponseWriter, r *http.Request) {
				isAuth = isAuthenticated(r.Context())
				w.WriteHeader(http.StatusOK)
			}

			// Apply auth middleware
			authHandler := server.authMiddleware(handler)

			// Create request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			// Create response recorder
			rec := httptest.NewRecorder()

			// Execute request
			authHandler(rec, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rec.Code)

			// Check authentication status
			if rec.Code == http.StatusOK {
				assert.Equal(t, tt.authenticated, isAuth)
			}
		})
	}
}

// Test the combined middleware stack
func TestCombinedMiddleware(t *testing.T) {
	mockStateManager := &MockStateManager{}
	mockStateManager.On("LoadState").Return(&types.State{
		Config: types.Config{},
	}, nil)

	server := &Server{
		stateManager: mockStateManager,
	}

	// Create test handler
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// Create middleware functions
	middleware1 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("Test-Header-1", "value1")
			next(w, r)
		}
	}

	middleware2 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("Test-Header-2", "value2")
			next(w, r)
		}
	}

	// Apply combined middleware
	combinedHandler := server.combineMiddleware(handler, middleware1, middleware2)

	// Create request and response recorder
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	combinedHandler(rec, req)

	// Check status code
	assert.Equal(t, http.StatusOK, rec.Code)
	
	// Note: Cannot check headers since they're set on the request which isn't accessible after execution
}