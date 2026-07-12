package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ───────────────────────────────────────────────────────────────

func noAuthFunc(r *http.Request) (bool, string) { return true, "test-user" }

func sampleInstances() []*types.Instance {
	return []*types.Instance{
		{
			ID:           "i-test-1",
			Name:         "test-inst",
			InstanceType: "t3.micro",
			State:        "running",
			PublicIP:     "1.2.3.4",
			Template:     "python-ml",
			LaunchTime:   time.Now(),
		},
	}
}

func sampleTemplates() map[string]types.RuntimeTemplate {
	return map[string]types.RuntimeTemplate{
		"python-ml": {Name: "python-ml"},
	}
}

func newDashboard() *DashboardServer {
	pm := NewProxyManager(noAuthFunc)
	return NewDashboardServer(
		func() ([]*types.Instance, error) { return sampleInstances(), nil },
		func() (map[string]types.RuntimeTemplate, error) { return sampleTemplates(), nil },
		pm,
	)
}

// ── DashboardServer ───────────────────────────────────────────────────────

func TestSameOriginCheck(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		origin string
		want   bool
	}{
		{"no origin (CLI/native client)", "example.com", "", true},
		{"same origin", "app.example.com", "http://app.example.com", true},
		{"same origin with port", "app.example.com:8947", "http://app.example.com:8947", true},
		{"loopback origin", "app.example.com", "http://localhost:3000", true},
		{"127.0.0.1 origin", "app.example.com", "http://127.0.0.1:5173", true},
		{"cross-site origin rejected", "app.example.com", "http://evil.example.net", false},
		{"malformed origin rejected", "app.example.com", "://not a url", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "http://"+tt.host+"/terminal", nil)
			r.Host = tt.host
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			assert.Equal(t, tt.want, sameOriginCheck(r))
		})
	}
}

func TestDashboardServer_Dashboard(t *testing.T) {
	srv := httptest.NewServer(newDashboard())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Prism Dashboard")
}

func TestDashboardServer_Instances(t *testing.T) {
	srv := httptest.NewServer(newDashboard())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/instances")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var instances []*types.Instance
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&instances))
	assert.Len(t, instances, 1)
}

func TestDashboardServer_Templates(t *testing.T) {
	srv := httptest.NewServer(newDashboard())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/templates")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var templates map[string]types.RuntimeTemplate
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&templates))
	assert.Contains(t, templates, "python-ml")
}

func TestDashboardServer_ProxyStats(t *testing.T) {
	srv := httptest.NewServer(newDashboard())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/proxy/stats")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var stats map[string]ProxyStats
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&stats))
	// No proxies registered → empty map
	assert.Empty(t, stats)
}

func TestDashboardServer_StaticCSS(t *testing.T) {
	srv := httptest.NewServer(newDashboard())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/static/style.css")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/css")
}

func TestDashboardServer_StaticJS(t *testing.T) {
	srv := httptest.NewServer(newDashboard())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/static/script.js")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/javascript")
}

func TestDashboardServer_NotFound(t *testing.T) {
	srv := httptest.NewServer(newDashboard())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/unknown-path")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ── ProxyManager ──────────────────────────────────────────────────────────

// testBackend creates a simple httptest.Server acting as a mock upstream.
// Returns server and parsed host+port.
func testBackend(t *testing.T) (*httptest.Server, string, int) {
	t.Helper()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello from backend"))
	}))
	t.Cleanup(backend.Close)

	u, err := url.Parse(backend.URL)
	require.NoError(t, err)
	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	return backend, host, port
}

func TestProxyManager_RegisterAndServe(t *testing.T) {
	_, host, port := testBackend(t)

	pm := NewProxyManager(noAuthFunc)
	inst := &types.Instance{
		ID:              "i-proxy-1",
		Name:            "web-inst",
		PublicIP:        host,
		WebPort:         port,
		HasWebInterface: true,
	}
	require.NoError(t, pm.RegisterInstance(inst))

	// Make a request through the proxy
	req := httptest.NewRequest(http.MethodGet, "/proxy/web-inst/", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	pm.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "hello from backend")
}

func TestProxyManager_GetProxyStats(t *testing.T) {
	_, host, port := testBackend(t)

	pm := NewProxyManager(noAuthFunc)
	inst := &types.Instance{
		ID:              "i-stats-1",
		Name:            "stats-inst",
		PublicIP:        host,
		WebPort:         port,
		HasWebInterface: true,
	}
	require.NoError(t, pm.RegisterInstance(inst))

	// Make a request to increment the counter
	req := httptest.NewRequest(http.MethodGet, "/proxy/stats-inst/", nil)
	req.RemoteAddr = "127.0.0.1:0"
	rec := httptest.NewRecorder()
	pm.ServeHTTP(rec, req)

	stats := pm.GetProxyStats()
	require.Contains(t, stats, "i-stats-1")
	assert.Equal(t, int64(1), stats["i-stats-1"].AccessCount)
}

func TestProxyManager_UnregisterInstance(t *testing.T) {
	_, host, port := testBackend(t)

	pm := NewProxyManager(noAuthFunc)
	inst := &types.Instance{
		ID:              "i-unreg-1",
		Name:            "unreg-inst",
		PublicIP:        host,
		WebPort:         port,
		HasWebInterface: true,
	}
	require.NoError(t, pm.RegisterInstance(inst))

	// Unregister
	require.NoError(t, pm.UnregisterInstance("i-unreg-1"))

	// Subsequent request should get 404
	req := httptest.NewRequest(http.MethodGet, "/proxy/unreg-inst/", nil)
	req.RemoteAddr = "127.0.0.1:0"
	rec := httptest.NewRecorder()
	pm.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestProxyManager_Auth_Reject(t *testing.T) {
	denyAll := func(r *http.Request) (bool, string) { return false, "" }
	pm := NewProxyManager(denyAll)

	req := httptest.NewRequest(http.MethodGet, "/proxy/any/", nil)
	rec := httptest.NewRecorder()
	pm.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, strings.ToLower(rec.Body.String()), "unauthorized")
}

func TestProxyManager_RegisterInstance_NoWebInterface(t *testing.T) {
	pm := NewProxyManager(noAuthFunc)
	inst := &types.Instance{
		ID:              "i-noweb",
		Name:            "no-web",
		HasWebInterface: false,
	}
	err := pm.RegisterInstance(inst)
	assert.Error(t, err)
}

func TestProxyManager_UnregisterInstance_NotFound(t *testing.T) {
	pm := NewProxyManager(noAuthFunc)
	err := pm.UnregisterInstance("i-does-not-exist")
	assert.Error(t, err)
}

func TestProxyManager_NoRoute(t *testing.T) {
	pm := NewProxyManager(noAuthFunc)

	req := httptest.NewRequest(http.MethodGet, "/proxy/ghost/", nil)
	rec := httptest.NewRecorder()
	pm.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
