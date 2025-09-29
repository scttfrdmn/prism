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
)

// These variables are populated by the build system.
var (
	// Version is the current version of CloudWorkstation.
	// Should be in the format MAJOR.MINOR.PATCH.
	Version = "0.4.6"

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
	result := fmt.Sprintf("CloudWorkstation v%s", Version)
	if GitCommit != "" {
		commitLen := len(GitCommit)
		if commitLen > 8 {
			commitLen = 8
		}
		result += fmt.Sprintf(" (commit: %s", GitCommit[:commitLen])
		if BuildDate != "" {
			result += fmt.Sprintf(", built: %s", BuildDate)
		}
		result += ")"
	}
	if GoVersion != "" {
		result += fmt.Sprintf(" [%s]", GoVersion)
	}
	return result
}

// GetCLIVersionInfo returns version info specifically for the CLI component.
func GetCLIVersionInfo() string {
	result := fmt.Sprintf("CloudWorkstation CLI v%s", Version)
	if GitCommit != "" {
		commitLen := len(GitCommit)
		if commitLen > 8 {
			commitLen = 8
		}
		result += fmt.Sprintf(" (commit: %s", GitCommit[:commitLen])
		if BuildDate != "" {
			result += fmt.Sprintf(", built: %s", BuildDate)
		}
		result += ")"
	}
	if GoVersion != "" {
		result += fmt.Sprintf(" [%s]", GoVersion)
	}
	return result
}

// GetDaemonVersionInfo returns version info specifically for the daemon component.
func GetDaemonVersionInfo() string {
	result := fmt.Sprintf("CloudWorkstation Daemon v%s", Version)
	if GitCommit != "" {
		commitLen := len(GitCommit)
		if commitLen > 8 {
			commitLen = 8
		}
		result += fmt.Sprintf(" (commit: %s", GitCommit[:commitLen])
		if BuildDate != "" {
			result += fmt.Sprintf(", built: %s", BuildDate)
		}
		result += ")"
	}
	if GoVersion != "" {
		result += fmt.Sprintf(" [%s]", GoVersion)
	}
	return result
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
