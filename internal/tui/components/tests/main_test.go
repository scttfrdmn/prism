package tests

import (
	"os"
	"testing"
)

// TestMain is the main entry point for testing
func TestMain(m *testing.M) {
	// Setup code before tests
	setupTestEnvironment()

	// Run tests
	exitCode := m.Run()

	// Cleanup code after tests
	cleanupTestEnvironment()

	// Exit with the same code as the tests
	os.Exit(exitCode)
}

// setupTestEnvironment sets up the test environment
func setupTestEnvironment() {
	// Set any environment variables needed for tests
	os.Setenv("TERM", "xterm-256color") // Ensure terminal supports colors
}

// cleanupTestEnvironment cleans up the test environment
func cleanupTestEnvironment() {
	// Clean up any resources
}