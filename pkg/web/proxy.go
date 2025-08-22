// Package web provides web interface and proxy capabilities for CloudWorkstation
package web

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// ProxyManager manages web proxies for instance services
type ProxyManager struct {
	mu       sync.RWMutex
	proxies  map[string]*InstanceProxy
	routes   map[string]string // path -> instanceID mapping
	authFunc AuthFunc
}

// AuthFunc is a function that validates authentication for a request
type AuthFunc func(r *http.Request) (bool, string) // returns (authorized, username)

// InstanceProxy represents a proxy to an instance's web service
type InstanceProxy struct {
	InstanceID   string
	InstanceName string
	TargetURL    *url.URL
	Proxy        *httputil.ReverseProxy
	Created      time.Time
	LastAccessed time.Time
	AccessCount  int64
}

// NewProxyManager creates a new proxy manager
func NewProxyManager(authFunc AuthFunc) *ProxyManager {
	return &ProxyManager{
		proxies:  make(map[string]*InstanceProxy),
		routes:   make(map[string]string),
		authFunc: authFunc,
	}
}

// RegisterInstance registers a new instance for proxying
func (pm *ProxyManager) RegisterInstance(instance *types.Instance) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !instance.HasWebInterface {
		return fmt.Errorf("instance %s does not have a web interface", instance.Name)
	}

	// Create target URL
	targetURL, err := url.Parse(fmt.Sprintf("http://%s:%d", instance.PublicIP, instance.WebPort))
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	// Create reverse proxy with custom director
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.Host = targetURL.Host

			// Strip the proxy path prefix
			proxyPath := fmt.Sprintf("/proxy/%s", instance.Name)
			if strings.HasPrefix(req.URL.Path, proxyPath) {
				req.URL.Path = strings.TrimPrefix(req.URL.Path, proxyPath)
				if !strings.HasPrefix(req.URL.Path, "/") {
					req.URL.Path = "/" + req.URL.Path
				}
			}

			// Add forwarding headers
			if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
				req.Header.Set("X-Forwarded-For", clientIP)
			}
			req.Header.Set("X-Forwarded-Proto", "http")
			req.Header.Set("X-CloudWorkstation-Instance", instance.Name)
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Allow self-signed certificates
			},
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		ModifyResponse: func(resp *http.Response) error {
			// Rewrite location headers for redirects
			if location := resp.Header.Get("Location"); location != "" {
				if u, err := url.Parse(location); err == nil {
					if u.Host == targetURL.Host {
						u.Scheme = "http"
						u.Host = resp.Request.Host
						u.Path = fmt.Sprintf("/proxy/%s%s", instance.Name, u.Path)
						resp.Header.Set("Location", u.String())
					}
				}
			}

			// Add security headers
			resp.Header.Set("X-Frame-Options", "SAMEORIGIN")
			resp.Header.Set("X-Content-Type-Options", "nosniff")
			resp.Header.Set("X-CloudWorkstation-Proxied", "true")

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, fmt.Sprintf("Proxy error: %v", err), http.StatusBadGateway)
		},
	}

	// Create instance proxy
	instanceProxy := &InstanceProxy{
		InstanceID:   instance.ID,
		InstanceName: instance.Name,
		TargetURL:    targetURL,
		Proxy:        proxy,
		Created:      time.Now(),
		LastAccessed: time.Now(),
	}

	// Register proxy and route
	pm.proxies[instance.ID] = instanceProxy
	pm.routes[fmt.Sprintf("/proxy/%s", instance.Name)] = instance.ID

	return nil
}

// UnregisterInstance removes an instance from proxying
func (pm *ProxyManager) UnregisterInstance(instanceID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	proxy, exists := pm.proxies[instanceID]
	if !exists {
		return fmt.Errorf("instance not registered: %s", instanceID)
	}

	// Remove route
	delete(pm.routes, fmt.Sprintf("/proxy/%s", proxy.InstanceName))
	// Remove proxy
	delete(pm.proxies, instanceID)

	return nil
}

// ServeHTTP implements http.Handler for the proxy manager
func (pm *ProxyManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	if pm.authFunc != nil {
		authorized, username := pm.authFunc(r)
		if !authorized {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		r.Header.Set("X-CloudWorkstation-User", username)
	}

	// Find matching route
	pm.mu.RLock()
	instanceID := ""
	for path, id := range pm.routes {
		if strings.HasPrefix(r.URL.Path, path) {
			instanceID = id
			break
		}
	}
	
	if instanceID == "" {
		pm.mu.RUnlock()
		http.Error(w, "No proxy route found", http.StatusNotFound)
		return
	}

	proxy, exists := pm.proxies[instanceID]
	if !exists {
		pm.mu.RUnlock()
		http.Error(w, "Instance proxy not found", http.StatusNotFound)
		return
	}
	pm.mu.RUnlock()

	// Update access stats
	pm.mu.Lock()
	proxy.LastAccessed = time.Now()
	proxy.AccessCount++
	pm.mu.Unlock()

	// Proxy the request
	proxy.Proxy.ServeHTTP(w, r)
}

// GetProxyStats returns statistics for all proxies
func (pm *ProxyManager) GetProxyStats() map[string]ProxyStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := make(map[string]ProxyStats)
	for id, proxy := range pm.proxies {
		stats[id] = ProxyStats{
			InstanceID:   proxy.InstanceID,
			InstanceName: proxy.InstanceName,
			TargetURL:    proxy.TargetURL.String(),
			Created:      proxy.Created,
			LastAccessed: proxy.LastAccessed,
			AccessCount:  proxy.AccessCount,
		}
	}
	return stats
}

// ProxyStats contains statistics for a proxy
type ProxyStats struct {
	InstanceID   string    `json:"instance_id"`
	InstanceName string    `json:"instance_name"`
	TargetURL    string    `json:"target_url"`
	Created      time.Time `json:"created"`
	LastAccessed time.Time `json:"last_accessed"`
	AccessCount  int64     `json:"access_count"`
}

// WebSocketProxy handles WebSocket proxying for instances
type WebSocketProxy struct {
	targetURL *url.URL
	upgrader  *WebSocketUpgrader
}

// WebSocketUpgrader handles WebSocket upgrade
type WebSocketUpgrader struct {
	CheckOrigin func(r *http.Request) bool
}

// NewWebSocketProxy creates a new WebSocket proxy
func NewWebSocketProxy(targetURL string) (*WebSocketProxy, error) {
	u, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	// Convert http to ws scheme
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}

	return &WebSocketProxy{
		targetURL: u,
		upgrader: &WebSocketUpgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
	}, nil
}

// ServeHTTP handles WebSocket proxy requests
func (wp *WebSocketProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// This is a simplified WebSocket proxy
	// In production, you'd use a proper WebSocket library like gorilla/websocket
	
	// Connect to backend
	targetConn, err := net.Dial("tcp", wp.targetURL.Host)
	if err != nil {
		http.Error(w, "Cannot connect to backend", http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	// Hijack the connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Proxy the WebSocket handshake and data
	go io.Copy(targetConn, clientConn)
	io.Copy(clientConn, targetConn)
}