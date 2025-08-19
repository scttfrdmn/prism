// Package cli tests for constants and helper functions
package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConstants tests that all constants have expected values
func TestConstants(t *testing.T) {
	// Test network constants
	assert.Equal(t, "http://localhost:8947", DefaultDaemonURL)
	assert.Equal(t, "8947", DefaultDaemonPort)
	assert.Equal(t, "CWSD_URL", DaemonURLEnvVar)

	// Test configuration constants
	assert.Equal(t, "gp3", DefaultVolumeType)
	assert.Equal(t, 5, DefaultInstanceRetentionMinutes)
	assert.Equal(t, "/mnt/", DefaultMountPointPrefix)
	assert.Equal(t, ".cloudworkstation", DefaultConfigDir)
	assert.Equal(t, "daemon_config.json", DefaultConfigFile)

	// Test file and path constants
	assert.Equal(t, "/var/log/cloudworkstation-analytics.json", AnalyticsLogFile)
	assert.Equal(t, "/var/log/cloudworkstation-rightsizing.json", RightsizingLogFile)
	assert.Equal(t, "./templates", DefaultTemplateDir)
	assert.Equal(t, ".yml", TemplateFileExtensionYML)
	assert.Equal(t, ".yaml", TemplateFileExtensionYAML)

	// Test numeric constants
	assert.Equal(t, 0644, DefaultFilePermissions)
	assert.Equal(t, 0755, DefaultDirPermissions)
	assert.Equal(t, 30, DaysToMonthEstimate)
	assert.Equal(t, 365, DaysToYearEstimate)
	assert.Equal(t, 1000, DefaultAnalyticsSampleCount)
	assert.Equal(t, 33, DefaultAnalyticsSampleHours)

	// Test tabwriter constants
	assert.Equal(t, 0, TabWriterMinWidth)
	assert.Equal(t, 8, TabWriterTabWidth)
	assert.Equal(t, 2, TabWriterPadding)
	assert.Equal(t, ' ', TabWriterPadChar)
	assert.Equal(t, 0, TabWriterFlags)

	// Test package manager constants
	assert.Equal(t, "AMI", PackageManagerAMI)
	assert.Equal(t, "APT", PackageManagerAPT)
	assert.Equal(t, "DNF", PackageManagerDNF)
	assert.Equal(t, "conda", PackageManagerConda)

	// Test cost constants
	assert.Equal(t, 0.25, DefaultSavingsEstimate)
	assert.Equal(t, 20, TypicalRightsizingSavingsMin)
	assert.Equal(t, 40, TypicalRightsizingSavingsMax)
	assert.Equal(t, 30, OverProvisioningWastePercent)
}

// TestUsageMessages tests that all usage messages contain expected content
func TestUsageMessages(t *testing.T) {
	// Test daemon message
	assert.Contains(t, DaemonNotRunningMessage, "daemon not running")
	assert.Contains(t, DaemonNotRunningMessage, "cws daemon start")

	// Test no instances messages
	assert.Contains(t, NoInstancesFoundMessage, "No workstations found")
	assert.Contains(t, NoInstancesFoundMessage, "cws launch")

	assert.Contains(t, NoInstancesFoundProjectMessage, "No workstations found in project")
	assert.Contains(t, NoInstancesFoundProjectMessage, "%s")

	// Test volume messages
	assert.Contains(t, NoEFSVolumesFoundMessage, "No EFS volumes found")
	assert.Contains(t, NoEFSVolumesFoundMessage, "cws volume create")

	assert.Contains(t, NoEBSVolumesFoundMessage, "No EBS volumes found")
	assert.Contains(t, NoEBSVolumesFoundMessage, "cws storage create")
}

// TestProgressMessages tests launch progress and state messages
func TestProgressMessages(t *testing.T) {
	messages := []string{
		LaunchProgressAMIMessage,
		LaunchProgressPackageMessage,
		LaunchProgressPackageTiming,
		SetupTimeoutMessage,
		SetupTimeoutHelpMessage,
		SetupTimeoutConnectMessage,
		AMITimeoutMessage,
		StateMessageInitializing,
		StateMessageRunningReady,
		StateMessageDryRunSuccess,
		StateMessageUnexpectedStop,
		StateMessageTerminated,
	}

	for _, message := range messages {
		assert.NotEmpty(t, message, "Progress message should not be empty")
		// Most messages should contain some form of emoji or indicator
		hasIndicator := strings.ContainsAny(message, "📦⏳🔄✅❌⚠️🔧📥⚙️💡")
		if !hasIndicator {
			t.Logf("Message without indicator: %s", message)
		}
	}
}

// TestDateFormats tests date format constants
func TestDateFormats(t *testing.T) {
	formats := map[string]string{
		"StandardDateFormat": StandardDateFormat,
		"ShortDateFormat":    ShortDateFormat,
		"CompactDateFormat":  CompactDateFormat,
		"ISO8601DateFormat":  ISO8601DateFormat,
	}

	for name, format := range formats {
		assert.NotEmpty(t, format, "%s should not be empty", name)
		assert.Contains(t, format, "2006", "%s should be a Go time format", name)
	}
}

// TestTSizeSpecifications tests T-shirt size specifications
func TestTSizeSpecifications(t *testing.T) {
	// Test that all expected sizes exist
	expectedSizes := []string{"XS", "S", "M", "L", "XL"}
	for _, size := range expectedSizes {
		spec, exists := TSizeSpecifications[size]
		assert.True(t, exists, "T-shirt size %s should exist", size)

		// Test that specs have reasonable values
		assert.NotEmpty(t, spec.CPU, "CPU specification should not be empty for size %s", size)
		assert.NotEmpty(t, spec.Memory, "Memory specification should not be empty for size %s", size)
		assert.NotEmpty(t, spec.Storage, "Storage specification should not be empty for size %s", size)
		assert.Greater(t, spec.Cost, 0.0, "Cost should be greater than 0 for size %s", size)

		// Test format consistency
		assert.Contains(t, spec.CPU, "vCPU", "CPU spec should contain 'vCPU' for size %s", size)
		assert.Contains(t, spec.Memory, "GB", "Memory spec should contain 'GB' for size %s", size)
		// Storage can be GB or TB depending on size
		assert.True(t, strings.Contains(spec.Storage, "GB") || strings.Contains(spec.Storage, "TB"), "Storage spec should contain 'GB' or 'TB' for size %s", size)
	}

	// Test that costs increase with size
	assert.Less(t, TSizeSpecifications["XS"].Cost, TSizeSpecifications["S"].Cost)
	assert.Less(t, TSizeSpecifications["S"].Cost, TSizeSpecifications["M"].Cost)
	assert.Less(t, TSizeSpecifications["M"].Cost, TSizeSpecifications["L"].Cost)
	assert.Less(t, TSizeSpecifications["L"].Cost, TSizeSpecifications["XL"].Cost)
}

// TestValidTSizes tests T-shirt size validation map
func TestValidTSizes(t *testing.T) {
	expectedSizes := []string{"XS", "S", "M", "L", "XL"}

	for _, size := range expectedSizes {
		assert.True(t, ValidTSizes[size], "Size %s should be marked as valid", size)
	}

	// Test that the map only contains expected sizes
	assert.Len(t, ValidTSizes, len(expectedSizes), "ValidTSizes should only contain expected sizes")

	// Test invalid sizes
	invalidSizes := []string{"XXS", "XXL", "SMALL", "LARGE", "medium", "xs", "invalid"}
	for _, size := range invalidSizes {
		assert.False(t, ValidTSizes[size], "Size %s should not be marked as valid", size)
	}
}

// TestServicePortMappings tests service port mappings
func TestServicePortMappings(t *testing.T) {
	// Test expected port mappings
	expectedMappings := map[int]string{
		22:   "SSH",
		80:   "HTTP",
		443:  "HTTPS",
		3306: "MySQL",
		5432: "PostgreSQL",
		6379: "Redis",
		8787: "RStudio Server",
		8888: "Jupyter Notebook",
	}

	for port, expectedService := range expectedMappings {
		service, exists := ServicePortMappings[port]
		assert.True(t, exists, "Port %d should have a service mapping", port)
		assert.Equal(t, expectedService, service, "Port %d should map to %s", port, expectedService)
	}

	// Test that all mappings are reasonable
	for port, service := range ServicePortMappings {
		assert.Greater(t, port, 0, "Port should be positive")
		assert.Less(t, port, 65536, "Port should be valid")
		assert.NotEmpty(t, service, "Service name should not be empty")
	}
}

// TestInstanceTypeSizeMapping tests instance type to size mappings
func TestInstanceTypeSizeMapping(t *testing.T) {
	// Test that all mappings point to valid T-shirt sizes
	for instanceType, size := range InstanceTypeSizeMapping {
		assert.True(t, ValidTSizes[size], "Instance type %s maps to invalid size %s", instanceType, size)
		assert.NotEmpty(t, instanceType, "Instance type should not be empty")
	}

	// Test expected instance types
	expectedTypes := []string{"t3.small", "t3.medium", "t3.large", "t4g.small", "t4g.medium", "t4g.large"}
	for _, instanceType := range expectedTypes {
		size, exists := InstanceTypeSizeMapping[instanceType]
		assert.True(t, exists, "Instance type %s should have a size mapping", instanceType)
		assert.True(t, ValidTSizes[size], "Instance type %s should map to valid size", instanceType)
	}
}

// TestSizeInstanceTypeMapping tests size to instance type mappings
func TestSizeInstanceTypeMapping(t *testing.T) {
	// Test that all T-shirt sizes have mappings
	for size := range ValidTSizes {
		instanceType, exists := SizeInstanceTypeMapping[size]
		assert.True(t, exists, "Size %s should have an instance type mapping", size)
		assert.NotEmpty(t, instanceType, "Instance type should not be empty for size %s", size)
	}

	// Test consistency between mappings
	for size, instanceType := range SizeInstanceTypeMapping {
		if mappedSize, exists := InstanceTypeSizeMapping[instanceType]; exists {
			assert.Equal(t, size, mappedSize, "Mapping inconsistency: size %s maps to %s which maps back to %s", size, instanceType, mappedSize)
		}
	}
}

// TestPackageIndicators tests package detection arrays
func TestPackageIndicators(t *testing.T) {
	// Test GPU indicators
	assert.NotEmpty(t, GPUPackageIndicators, "GPU package indicators should not be empty")
	expectedGPU := []string{"cuda", "nvidia", "pytorch", "tensorflow-gpu"}
	for _, indicator := range expectedGPU {
		assert.Contains(t, GPUPackageIndicators, indicator, "Should contain GPU indicator: %s", indicator)
	}

	// Test memory indicators
	assert.NotEmpty(t, MemoryPackageIndicators, "Memory package indicators should not be empty")
	expectedMemory := []string{"r-base", "spark", "hadoop"}
	for _, indicator := range expectedMemory {
		assert.Contains(t, MemoryPackageIndicators, indicator, "Should contain memory indicator: %s", indicator)
	}

	// Test compute indicators
	assert.NotEmpty(t, ComputePackageIndicators, "Compute package indicators should not be empty")
	expectedCompute := []string{"openmpi", "fftw", "blas", "mkl"}
	for _, indicator := range expectedCompute {
		assert.Contains(t, ComputePackageIndicators, indicator, "Should contain compute indicator: %s", indicator)
	}
}

// TestErrorHelperFunctions tests error helper functions
func TestErrorHelperFunctions(t *testing.T) {
	// Test WrapAPIError
	originalErr := errors.New("connection failed")
	wrappedErr := WrapAPIError("connect to service", originalErr)
	assert.Error(t, wrappedErr)
	assert.Contains(t, wrappedErr.Error(), "failed to connect to service")
	assert.Contains(t, wrappedErr.Error(), "connection failed")

	// Test WrapDaemonError
	daemonErr := WrapDaemonError(originalErr)
	assert.Error(t, daemonErr)
	assert.Equal(t, DaemonNotRunningMessage, daemonErr.Error())

	// Test NewUsageError
	usageErr := NewUsageError("command <arg>", "command example")
	assert.Error(t, usageErr)
	assert.Contains(t, usageErr.Error(), "usage: command <arg>")
	assert.Contains(t, usageErr.Error(), "Example: command example")

	usageErrNoExample := NewUsageError("command <arg>", "")
	assert.Error(t, usageErrNoExample)
	assert.Contains(t, usageErrNoExample.Error(), "usage: command <arg>")
	assert.NotContains(t, usageErrNoExample.Error(), "Example:")

	// Test NewValidationError
	validationErr := NewValidationError("flag", "invalid", "valid1, valid2")
	assert.Error(t, validationErr)
	assert.Contains(t, validationErr.Error(), "invalid flag 'invalid'")
	assert.Contains(t, validationErr.Error(), "expected valid1, valid2")

	validationErrNoExpected := NewValidationError("field", "bad", "")
	assert.Error(t, validationErrNoExpected)
	assert.Contains(t, validationErrNoExpected.Error(), "invalid field 'bad'")
	assert.NotContains(t, validationErrNoExpected.Error(), "expected")

	// Test NewNotFoundError
	notFoundErr := NewNotFoundError("resource", "name", "try something else")
	assert.Error(t, notFoundErr)
	assert.Contains(t, notFoundErr.Error(), "resource 'name' not found")
	assert.Contains(t, notFoundErr.Error(), "try something else")

	notFoundErrNoSuggestion := NewNotFoundError("item", "missing", "")
	assert.Error(t, notFoundErrNoSuggestion)
	assert.Contains(t, notFoundErrNoSuggestion.Error(), "item 'missing' not found")

	// Test NewStateError
	stateErr := NewStateError("instance", "test", "stopped", "running")
	assert.Error(t, stateErr)
	assert.Contains(t, stateErr.Error(), "instance 'test' is in state 'stopped'")
	assert.Contains(t, stateErr.Error(), "expected 'running'")

	stateErrNoExpected := NewStateError("service", "web", "down", "")
	assert.Error(t, stateErrNoExpected)
	assert.Contains(t, stateErrNoExpected.Error(), "service 'web' is in invalid state 'down'")
}

// TestFormatHelperFunctions tests message formatting functions
func TestFormatHelperFunctions(t *testing.T) {
	// Test FormatSuccessMessage
	successMsg := FormatSuccessMessage("Created", "instance", "(i-123)")
	assert.Equal(t, "✅ Created instance (i-123)", successMsg)

	successMsgNoDetails := FormatSuccessMessage("Started", "service", "")
	assert.Equal(t, "✅ Started service", successMsgNoDetails)

	// Test FormatProgressMessage
	progressMsg := FormatProgressMessage("Downloading", "packages")
	assert.Equal(t, "🔄 Downloading packages...", progressMsg)

	// Test FormatWarningMessage
	warningMsg := FormatWarningMessage("Connection", "timeout occurred")
	assert.Equal(t, "⚠️  Connection: timeout occurred", warningMsg)

	// Test FormatErrorMessage
	errorMsg := FormatErrorMessage("Validation", "field is required")
	assert.Equal(t, "❌ Validation: field is required", errorMsg)

	// Test FormatInfoMessage
	infoMsg := FormatInfoMessage("Use --help for more options")
	assert.Equal(t, "💡 Use --help for more options", infoMsg)
}

// TestTimeConstants tests time-related constants
func TestTimeConstants(t *testing.T) {
	// Test that time constants are reasonable
	assert.Greater(t, DaemonStartupTimeout.Seconds(), 0.0)
	assert.Less(t, DaemonStartupTimeout.Seconds(), 60.0)

	assert.Greater(t, DaemonStartupMaxAttempts, 0)
	assert.Less(t, DaemonStartupMaxAttempts, 100)

	assert.Greater(t, DaemonStartupRetryInterval.Milliseconds(), int64(0))
	assert.Less(t, DaemonStartupRetryInterval.Milliseconds(), int64(10000))

	assert.Greater(t, AMILaunchMonitorTimeout, 0)
	assert.Greater(t, PackageLaunchMonitorTimeout, AMILaunchMonitorTimeout)

	assert.Greater(t, LaunchProgressInterval.Seconds(), 0.0)
	assert.Less(t, LaunchProgressInterval.Seconds(), 60.0)
}

// TestAnalyticsConstants tests analytics-related constants
func TestAnalyticsConstants(t *testing.T) {
	// Test analytics collection interval
	assert.NotEmpty(t, AnalyticsCollectionInterval)
	assert.Contains(t, AnalyticsCollectionInterval, "minutes")

	// Test that sample count and hours are consistent
	assert.Greater(t, DefaultAnalyticsSampleCount, 0)
	assert.Greater(t, DefaultAnalyticsSampleHours, 0)

	// Rough consistency check (2 minutes interval * 1000 samples ≈ 33 hours)
	approximateHours := (DefaultAnalyticsSampleCount * 2) / 60 // 2 minutes per sample
	assert.InDelta(t, float64(DefaultAnalyticsSampleHours), float64(approximateHours), 5.0)
}

// TestByteConversion tests byte conversion constants
func TestByteConversion(t *testing.T) {
	assert.Equal(t, 1024*1024*1024, BytesToGB)

	// Test that 1GB converts correctly
	bytes := int64(BytesToGB)
	gb := float64(bytes) / float64(BytesToGB)
	assert.Equal(t, 1.0, gb)

	// Test that 2.5GB converts correctly
	bytes = int64(2.5 * float64(BytesToGB))
	gb = float64(bytes) / float64(BytesToGB)
	assert.InDelta(t, 2.5, gb, 0.1)
}

// TestStringConstants tests that string constants don't have unexpected content
func TestStringConstants(t *testing.T) {
	stringConstants := map[string]string{
		"DefaultDaemonURL":            DefaultDaemonURL,
		"DaemonURLEnvVar":             DaemonURLEnvVar,
		"DefaultConfigDir":            DefaultConfigDir,
		"DefaultConfigFile":           DefaultConfigFile,
		"DefaultTemplateDir":          DefaultTemplateDir,
		"TemplateFileExtensionYML":    TemplateFileExtensionYML,
		"TemplateFileExtensionYAML":   TemplateFileExtensionYAML,
		"AnalyticsLogFile":            AnalyticsLogFile,
		"RightsizingLogFile":          RightsizingLogFile,
		"NoInstancesFoundMessage":     NoInstancesFoundMessage,
		"NoEFSVolumesFoundMessage":    NoEFSVolumesFoundMessage,
		"NoEBSVolumesFoundMessage":    NoEBSVolumesFoundMessage,
		"StandardDateFormat":          StandardDateFormat,
		"ShortDateFormat":             ShortDateFormat,
		"CompactDateFormat":           CompactDateFormat,
		"ISO8601DateFormat":           ISO8601DateFormat,
		"AnalyticsCollectionInterval": AnalyticsCollectionInterval,
	}

	for name, value := range stringConstants {
		assert.NotEmpty(t, value, "String constant %s should not be empty", name)
		assert.NotContains(t, value, "\x00", "String constant %s should not contain null bytes", name)

		// Test that file paths are reasonable
		if strings.Contains(name, "File") || strings.Contains(name, "Dir") {
			if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "./") {
				// Absolute or relative path is ok
			} else if strings.Contains(value, "/") {
				// Relative path with directories
			} else {
				// Just a filename
				assert.NotContains(t, value, " ", "File constant %s should not contain spaces", name)
			}
		}

		// Test URL format - but skip environment variable names
		if strings.Contains(name, "URL") && !strings.Contains(name, "EnvVar") {
			assert.True(t, strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://"),
				"URL constant %s should start with http:// or https://", name)
		}
	}
}

// BenchmarkErrorFunctions benchmarks error helper functions
func BenchmarkErrorFunctions(b *testing.B) {
	originalErr := errors.New("test error")

	b.Run("WrapAPIError", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = WrapAPIError("test operation", originalErr)
		}
	})

	b.Run("WrapDaemonError", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = WrapDaemonError(originalErr)
		}
	})

	b.Run("NewUsageError", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewUsageError("command <arg>", "example")
		}
	})

	b.Run("NewValidationError", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewValidationError("field", "value", "expected")
		}
	})
}

// BenchmarkFormatFunctions benchmarks message formatting functions
func BenchmarkFormatFunctions(b *testing.B) {
	b.Run("FormatSuccessMessage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = FormatSuccessMessage("Action", "resource", "details")
		}
	})

	b.Run("FormatProgressMessage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = FormatProgressMessage("Processing", "items")
		}
	})

	b.Run("FormatWarningMessage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = FormatWarningMessage("Context", "message")
		}
	})

	b.Run("FormatInfoMessage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = FormatInfoMessage("helpful tip")
		}
	})
}
