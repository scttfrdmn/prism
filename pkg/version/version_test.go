package version

import (
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	version := GetVersion()
	if version == "" {
		t.Error("GetVersion() returned empty string")
	}

	// Should match semantic versioning pattern
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		t.Errorf("Expected version to have at least major.minor format, got: %s", version)
	}
}

func TestSemVer(t *testing.T) {
	tests := []struct {
		name          string
		version       string
		expectedMajor int
		expectedMinor int
		expectedPatch int
	}{
		{
			name:          "valid semantic version",
			version:       "1.2.3",
			expectedMajor: 1,
			expectedMinor: 2,
			expectedPatch: 3,
		},
		{
			name:          "zero version",
			version:       "0.0.0",
			expectedMajor: 0,
			expectedMinor: 0,
			expectedPatch: 0,
		},
		{
			name:          "single digit version",
			version:       "5.10.15",
			expectedMajor: 5,
			expectedMinor: 10,
			expectedPatch: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Temporarily set Version for testing
			originalVersion := Version
			Version = tt.version
			defer func() { Version = originalVersion }()

			major, minor, patch := SemVer()

			if major != tt.expectedMajor {
				t.Errorf("Expected major version %d, got %d", tt.expectedMajor, major)
			}
			if minor != tt.expectedMinor {
				t.Errorf("Expected minor version %d, got %d", tt.expectedMinor, minor)
			}
			if patch != tt.expectedPatch {
				t.Errorf("Expected patch version %d, got %d", tt.expectedPatch, patch)
			}
		})
	}
}

func TestSemVerInvalidFormat(t *testing.T) {
	tests := []struct {
		name           string
		invalidVersion string
	}{
		{"empty version", ""},
		{"single number", "1"},
		{"two numbers", "1.2"},
		{"non-numeric", "v1.2.3"},
		{"malformed", "1.2.x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalVersion := Version
			Version = tt.invalidVersion
			defer func() { Version = originalVersion }()

			major, minor, patch := SemVer()

			// Should return zeros for invalid format
			if major != 0 || minor != 0 || patch != 0 {
				t.Errorf("Expected (0,0,0) for invalid version %s, got (%d,%d,%d)",
					tt.invalidVersion, major, minor, patch)
			}
		})
	}
}

func TestSemVerWithSuffix(t *testing.T) {
	// Test that versions with suffixes still parse the numeric part correctly
	originalVersion := Version
	Version = "1.2.3-beta"
	defer func() { Version = originalVersion }()

	major, minor, patch := SemVer()

	// fmt.Sscanf successfully parses the numeric part even with suffix
	if major != 1 || minor != 2 || patch != 3 {
		t.Errorf("Expected (1,2,3) for version with suffix, got (%d,%d,%d)", major, minor, patch)
	}
}

func TestGetVersionInfo(t *testing.T) {
	info := GetVersionInfo()

	if !strings.Contains(info, "Prism") {
		t.Error("Version info should contain 'Prism'")
	}

	if !strings.Contains(info, "v") {
		t.Error("Version info should contain version prefix 'v'")
	}
}

func TestGetCLIVersionInfo(t *testing.T) {
	info := GetCLIVersionInfo()

	if !strings.Contains(info, "Prism CLI") {
		t.Error("CLI version info should contain 'Prism CLI'")
	}

	if !strings.Contains(info, "v") {
		t.Error("CLI version info should contain version prefix 'v'")
	}
}

func TestGetDaemonVersionInfo(t *testing.T) {
	info := GetDaemonVersionInfo()

	if !strings.Contains(info, "Prism Daemon") {
		t.Error("Daemon version info should contain 'Prism Daemon'")
	}

	if !strings.Contains(info, "v") {
		t.Error("Daemon version info should contain version prefix 'v'")
	}
}

func TestBuildVersionStringWithCommitAndBuildDate(t *testing.T) {
	// Save original values
	originalGitCommit := GitCommit
	originalBuildDate := BuildDate
	originalGoVersion := GoVersion

	// Set test values
	GitCommit = "abcdef1234567890"
	BuildDate = "2024-01-15T10:30:00Z"
	GoVersion = "go1.21.0"

	defer func() {
		GitCommit = originalGitCommit
		BuildDate = originalBuildDate
		GoVersion = originalGoVersion
	}()

	info := buildVersionString("TestComponent")

	// Should contain component name
	if !strings.Contains(info, "TestComponent") {
		t.Error("Version string should contain component name")
	}

	// Should contain abbreviated commit (8 chars)
	if !strings.Contains(info, "abcdef12") {
		t.Error("Version string should contain abbreviated git commit")
	}

	// Should not contain full commit
	if strings.Contains(info, "abcdef1234567890") {
		t.Error("Version string should not contain full git commit")
	}

	// Should contain build date
	if !strings.Contains(info, BuildDate) {
		t.Error("Version string should contain build date")
	}

	// Should contain Go version
	if !strings.Contains(info, GoVersion) {
		t.Error("Version string should contain Go version")
	}

	// Should have proper formatting
	if !strings.Contains(info, "(commit:") {
		t.Error("Version string should contain commit section")
	}

	if !strings.Contains(info, ", built:") {
		t.Error("Version string should contain build date section")
	}

	if !strings.Contains(info, "[go1.21.0]") {
		t.Error("Version string should contain Go version in brackets")
	}
}

func TestBuildVersionStringWithoutOptionalFields(t *testing.T) {
	// Save original values
	originalGitCommit := GitCommit
	originalBuildDate := BuildDate
	originalGoVersion := GoVersion

	// Clear optional fields
	GitCommit = ""
	BuildDate = ""
	GoVersion = ""

	defer func() {
		GitCommit = originalGitCommit
		BuildDate = originalBuildDate
		GoVersion = originalGoVersion
	}()

	info := buildVersionString("TestComponent")

	// Should contain component name and version
	if !strings.Contains(info, "TestComponent v") {
		t.Error("Version string should contain component name and version")
	}

	// Should not contain optional fields
	if strings.Contains(info, "commit:") {
		t.Error("Version string should not contain commit info when not available")
	}

	if strings.Contains(info, "built:") {
		t.Error("Version string should not contain build date when not available")
	}

	if strings.Contains(info, "[") || strings.Contains(info, "]") {
		t.Error("Version string should not contain Go version brackets when not available")
	}
}

func TestBuildVersionStringWithShortCommit(t *testing.T) {
	originalGitCommit := GitCommit
	GitCommit = "abc123" // Short commit (6 chars)

	defer func() {
		GitCommit = originalGitCommit
	}()

	info := buildVersionString("TestComponent")

	if !strings.Contains(info, "abc123") {
		t.Error("Version string should contain full short commit")
	}
}

func TestBuildVersionStringCommitWithoutBuildDate(t *testing.T) {
	originalGitCommit := GitCommit
	originalBuildDate := BuildDate

	GitCommit = "abcdef1234567890"
	BuildDate = "" // No build date

	defer func() {
		GitCommit = originalGitCommit
		BuildDate = originalBuildDate
	}()

	info := buildVersionString("TestComponent")

	if !strings.Contains(info, "(commit: abcdef12)") {
		t.Error("Version string should contain commit without build date")
	}

	if strings.Contains(info, ", built:") {
		t.Error("Version string should not contain build date section when empty")
	}
}

func TestVersionVariables(t *testing.T) {
	// Test that version variables are properly initialized
	if Version == "" {
		t.Error("Version should not be empty")
	}

	// GitCommit, BuildDate, and GoVersion can be empty (set by build system)
	// but they should be defined as string variables
	gitCommitType := GitCommit
	buildDateType := BuildDate
	goVersionType := GoVersion

	// This test ensures the variables exist and are of correct type
	_ = gitCommitType
	_ = buildDateType
	_ = goVersionType
}

func BenchmarkGetVersionInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetVersionInfo()
	}
}

func BenchmarkSemVer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = SemVer()
	}
}

func BenchmarkBuildVersionString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = buildVersionString("BenchmarkComponent")
	}
}
