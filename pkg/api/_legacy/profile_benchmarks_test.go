package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

func BenchmarkProfileSwitching(b *testing.B) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	// Create mock profile manager with multiple profiles
	profileManager := newMockProfileManager()
	
	// Create mock state manager
	stateManager := newMockStateManager()
	profileStateManager := newMockProfileAwareStateManager(stateManager, profileManager)
	
	// Create client
	client, err := NewProfileAwareClient(server.URL, profileManager, profileStateManager)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	
	// Run benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Switch between personal and work profiles in a cycle
		profileID := "personal"
		if i%2 == 1 {
			profileID = "work"
		}
		
		if err := client.SwitchProfile(profileID); err != nil {
			b.Fatalf("Failed to switch profile: %v", err)
		}
	}
}

func BenchmarkWithProfile(b *testing.B) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	// Create mock profile manager with multiple profiles
	profileManager := newMockProfileManager()
	
	// Create mock state manager
	stateManager := newMockStateManager()
	profileStateManager := newMockProfileAwareStateManager(stateManager, profileManager)
	
	// Create client
	client, err := NewProfileAwareClient(server.URL, profileManager, profileStateManager)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	
	// Run benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Get client for different profiles in a cycle
		profileID := "personal"
		if i%2 == 1 {
			profileID = "work"
		}
		
		_, err := client.WithProfile(profileID)
		if err != nil {
			b.Fatalf("Failed to get client with profile: %v", err)
		}
	}
}

func BenchmarkWithProfileContext(b *testing.B) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	// Create mock profile manager with multiple profiles
	profileManager := newMockProfileManager()
	
	// Create mock state manager
	stateManager := newMockStateManager()
	profileStateManager := newMockProfileAwareStateManager(stateManager, profileManager)
	
	// Create client
	client, err := NewProfileAwareClient(server.URL, profileManager, profileStateManager)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	
	// Create base context
	ctx := context.Background()
	
	// Run benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create context with profile
		_, err := client.WithProfileContext(ctx)
		if err != nil {
			b.Fatalf("Failed to create context with profile: %v", err)
		}
	}
}