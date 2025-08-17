package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestClientPerformanceOptions(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create a client with default options
	client := NewClient(server.URL)

	// Create a client with custom performance options
	options := ClientPerformanceOptions{
		MaxIdleConnections:    50,
		IdleConnectionTimeout: 30 * time.Second,
		KeepAlive:             15 * time.Second,
		DialTimeout:           5 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 3 * time.Second,
		ExpectRequestTimeout:  1 * time.Second,
	}

	optimizedClient := client.WithPerformanceOptions(options)

	// Test that client works
	err := optimizedClient.Ping()
	if err != nil {
		t.Fatalf("Expected ping to succeed, got error: %v", err)
	}
}

func TestProfileHTTPClientCache(t *testing.T) {
	cache := newHTTPClientCache()

	// Get client for profile 1
	client1 := cache.getClient("profile1", DefaultClientPerformanceOptions())

	// Get client for profile 1 again
	client2 := cache.getClient("profile1", DefaultClientPerformanceOptions())

	// Get client for profile 2
	client3 := cache.getClient("profile2", DefaultClientPerformanceOptions())

	// Test that the cache works
	if client1 != client2 {
		t.Errorf("Expected to get the same client for the same profile")
	}

	if client1 == client3 {
		t.Errorf("Expected to get different clients for different profiles")
	}
}

func BenchmarkClientWithCache(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Get client for a profile (should use cache after first call)
		profileID := "profile1"
		if i%2 == 1 {
			profileID = "profile2"
		}

		client := NewClient(server.URL)
		client.profileID = profileID
		client.httpClient = getHTTPClientForProfile(profileID)

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)

		client.getWithContext(ctx, "/api/v1/ping", nil)

		cancel()
	}
}

func BenchmarkClientWithoutCache(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create a new client for each request (no cache)
		profileID := "profile1"
		if i%2 == 1 {
			profileID = "profile2"
		}

		client := NewClient(server.URL)
		client.profileID = profileID

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)

		client.getWithContext(ctx, "/api/v1/ping", nil)

		cancel()
	}
}

func TestClientConcurrency(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create clients for different profiles
	client1 := NewClient(server.URL)
	client1.profileID = "profile1"
	client1.httpClient = getHTTPClientForProfile("profile1")

	client2 := NewClient(server.URL)
	client2.profileID = "profile2"
	client2.httpClient = getHTTPClientForProfile("profile2")

	// Test concurrent requests
	const numRequests = 10
	var wg sync.WaitGroup
	wg.Add(numRequests * 2) // Two clients

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			ctx := context.Background()
			client1.getWithContext(ctx, "/api/v1/ping", nil)
		}()

		go func() {
			defer wg.Done()
			ctx := context.Background()
			client2.getWithContext(ctx, "/api/v1/ping", nil)
		}()
	}

	wg.Wait()

	// If we got here without errors or deadlocks, the test passed
}
