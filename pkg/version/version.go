// Package version provides centralized version information for CloudWorkstation.
//
// This package implements the Semantic Versioning 2.0.0 specification
// (https://semver.org/) for the CloudWorkstation project. It provides
// a single source of truth for version information used by all
// components and binaries in the project.
package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// These variables are populated by the build system.
var (
	// Version is the current version of CloudWorkstation.
	// Should be in the format MAJOR.MINOR.PATCH.
	Version = "0.5.1"

	// GitCommit is the git commit hash of the build.
	GitCommit = ""

	// BuildDate is the build date of the build.
	BuildDate = ""

	// GoVersion is the go version used to compile the build.
	GoVersion = ""
)

// init attempts to populate version information from build info if available.
func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	// Use GoVersion from build info
	GoVersion = info.GoVersion

	// Try to find version info from build settings
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if GitCommit == "" {
				GitCommit = setting.Value
			}
		case "vcs.time":
			if BuildDate == "" {
				BuildDate = setting.Value
			}
		}
	}
}

// GetVersionInfo returns a formatted string with version information.
func GetVersionInfo() string {
	return buildVersionString("CloudWorkstation")
}

// GetCLIVersionInfo returns version info specifically for the CLI component.
func GetCLIVersionInfo() string {
	return buildVersionString("CloudWorkstation CLI")
}

// GetDaemonVersionInfo returns version info specifically for the daemon component.
func GetDaemonVersionInfo() string {
	return buildVersionString("CloudWorkstation Daemon")
}

// buildVersionString constructs optimized version string using strings.Builder
func buildVersionString(component string) string {
	var builder strings.Builder
	// Pre-allocate capacity for typical version string length
	builder.Grow(len(component) + len(Version) + 50)

	builder.WriteString(component)
	builder.WriteString(" v")
	builder.WriteString(Version)

	if GitCommit != "" {
		builder.WriteString(" (commit: ")
		commitLen := len(GitCommit)
		if commitLen > 8 {
			commitLen = 8
		}
		builder.WriteString(GitCommit[:commitLen])

		if BuildDate != "" {
			builder.WriteString(", built: ")
			builder.WriteString(BuildDate)
		}
		builder.WriteByte(')')
	}

	if GoVersion != "" {
		builder.WriteString(" [")
		builder.WriteString(GoVersion)
		builder.WriteByte(']')
	}

	return builder.String()
}

// GetVersion returns just the version number.
func GetVersion() string {
	return Version
}

// SemVer returns major, minor, and patch versions as integers.
func SemVer() (major, minor, patch int) {
	_, err := fmt.Sscanf(Version, "%d.%d.%d", &major, &minor, &patch)
	if err != nil {
		// If there's an error parsing, return zeros
		return 0, 0, 0
	}
	return major, minor, patch
}
