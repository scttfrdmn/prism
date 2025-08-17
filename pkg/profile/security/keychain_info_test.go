// Package security provides tests for keychain information and diagnostics
package security

import (
	"runtime"
	"testing"
)

// TestGetKeychainInfo validates keychain information retrieval
func TestGetKeychainInfo(t *testing.T) {
	info, err := GetKeychainInfo()
	if err != nil {
		t.Fatalf("Failed to get keychain info: %v", err)
	}

	// Validate basic fields
	if info.Platform != runtime.GOOS {
		t.Errorf("Expected platform %s, got %s", runtime.GOOS, info.Platform)
	}

	if !info.Available {
		t.Error("Keychain should be available")
	}

	if info.Provider == "" {
		t.Error("Provider should not be empty")
	}

	if info.SecurityLevel == "" {
		t.Error("Security level should not be empty")
	}

	t.Logf("Keychain Info:")
	t.Logf("  Provider: %s", info.Provider)
	t.Logf("  Platform: %s", info.Platform)
	t.Logf("  Native: %v", info.Native)
	t.Logf("  Security Level: %s", info.SecurityLevel)

	if info.FallbackReason != "" {
		t.Logf("  Fallback Reason: %s", info.FallbackReason)
	}

	// On macOS, we should have native keychain
	if runtime.GOOS == "darwin" && !info.Native {
		t.Logf("Warning: Expected native macOS keychain, but got fallback: %s", info.FallbackReason)
	}
}

// TestValidateKeychainProvider validates keychain provider functionality
func TestValidateKeychainProvider(t *testing.T) {
	err := ValidateKeychainProvider()
	if err != nil {
		t.Fatalf("Keychain provider validation failed: %v", err)
	}

	t.Log("✅ Keychain provider validation successful")
}

// TestDiagnoseKeychainIssues validates keychain diagnostics
func TestDiagnoseKeychainIssues(t *testing.T) {
	diagnostics := DiagnoseKeychainIssues()

	if diagnostics.Platform != runtime.GOOS {
		t.Errorf("Expected platform %s, got %s", runtime.GOOS, diagnostics.Platform)
	}

	t.Logf("Keychain Diagnostics:")
	t.Logf("  Platform: %s", diagnostics.Platform)

	if diagnostics.Info != nil {
		t.Logf("  Provider: %s", diagnostics.Info.Provider)
		t.Logf("  Native: %v", diagnostics.Info.Native)
	}

	if len(diagnostics.Issues) > 0 {
		t.Logf("  Issues:")
		for _, issue := range diagnostics.Issues {
			t.Logf("    - %s", issue)
		}
	}

	if len(diagnostics.Warnings) > 0 {
		t.Logf("  Warnings:")
		for _, warning := range diagnostics.Warnings {
			t.Logf("    - %s", warning)
		}
	}

	if len(diagnostics.Recommendations) > 0 {
		t.Logf("  Recommendations:")
		for _, rec := range diagnostics.Recommendations {
			t.Logf("    - %s", rec)
		}
	}
}

// TestKeychainInfoDetails validates detailed keychain information
func TestKeychainInfoDetails(t *testing.T) {
	info, err := GetKeychainInfo()
	if err != nil {
		t.Fatalf("Failed to get keychain info: %v", err)
	}

	// Details should always be provided
	if info.Details == nil {
		t.Error("Keychain details should not be nil")
	}

	// Validate platform-specific details
	switch runtime.GOOS {
	case "darwin":
		if info.Native {
			// Should have macOS-specific details
			if _, ok := info.Details["framework"]; !ok {
				t.Error("macOS keychain should include framework information")
			}
		}
	case "windows":
		if info.Native {
			// Should have Windows-specific details
			if _, ok := info.Details["api"]; !ok {
				t.Error("Windows keychain should include API information")
			}
		}
	case "linux":
		if info.Native {
			// Should have Linux-specific details
			if _, ok := info.Details["protocol"]; !ok {
				t.Error("Linux keychain should include protocol information")
			}
		}
	}

	// Fallback should have encryption details
	if !info.Native {
		if _, ok := info.Details["encryption"]; !ok {
			t.Error("Fallback storage should include encryption information")
		}
	}

	t.Log("✅ Keychain details validation successful")
}
