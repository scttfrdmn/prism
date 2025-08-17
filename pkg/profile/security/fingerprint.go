// Package security provides device fingerprinting for secure device binding
package security

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/user"
	"runtime"
	"sort"
	"strings"
	"time"
)

// DeviceFingerprint represents a comprehensive device identifier
type DeviceFingerprint struct {
	// System identifiers
	Hostname     string `json:"hostname"`
	SystemUUID   string `json:"system_uuid,omitempty"`
	MachineID    string `json:"machine_id,omitempty"`
	OSVersion    string `json:"os_version"`
	Architecture string `json:"architecture"`

	// Network identifiers
	MACAddresses []string `json:"mac_addresses"`
	PrimaryMAC   string   `json:"primary_mac"`

	// User context
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	HomeDir  string `json:"home_dir"`

	// Temporal context
	Created     time.Time `json:"created"`
	InstallTime time.Time `json:"install_time"`

	// Computed fingerprint hash
	Hash string `json:"hash"`
}

// GenerateDeviceFingerprint creates a comprehensive device fingerprint
func GenerateDeviceFingerprint() (*DeviceFingerprint, error) {
	fp := &DeviceFingerprint{
		Created:      time.Now(),
		OSVersion:    runtime.GOOS + "-" + runtime.GOARCH,
		Architecture: runtime.GOARCH,
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		fp.Hostname = hostname
	}

	// Get user information
	if currentUser, err := user.Current(); err == nil {
		fp.UserID = currentUser.Uid
		fp.Username = currentUser.Username
		fp.HomeDir = currentUser.HomeDir
	}

	// Get network interfaces
	if macs, err := getMACAddresses(); err == nil {
		fp.MACAddresses = macs
		if len(macs) > 0 {
			fp.PrimaryMAC = macs[0] // First MAC as primary
		}
	}

	// Get system-specific identifiers
	fp.SystemUUID = getSystemUUID()
	fp.MachineID = getMachineID()
	fp.InstallTime = getCloudWorkstationInstallTime()

	// Generate hash
	if err := fp.generateHash(); err != nil {
		return nil, fmt.Errorf("failed to generate fingerprint hash: %w", err)
	}

	return fp, nil
}

// generateHash creates a SHA-256 hash of the fingerprint components
func (fp *DeviceFingerprint) generateHash() error {
	// Create canonical representation for hashing
	hashInput := struct {
		Hostname     string    `json:"hostname"`
		SystemUUID   string    `json:"system_uuid"`
		MachineID    string    `json:"machine_id"`
		OSVersion    string    `json:"os_version"`
		MACAddresses []string  `json:"mac_addresses"`
		UserID       string    `json:"user_id"`
		Username     string    `json:"username"`
		InstallTime  time.Time `json:"install_time"`
	}{
		Hostname:     fp.Hostname,
		SystemUUID:   fp.SystemUUID,
		MachineID:    fp.MachineID,
		OSVersion:    fp.OSVersion,
		MACAddresses: fp.MACAddresses,
		UserID:       fp.UserID,
		Username:     fp.Username,
		InstallTime:  fp.InstallTime,
	}

	// Sort MAC addresses for consistent hashing
	sort.Strings(hashInput.MACAddresses)

	// Marshal to JSON for canonical representation
	data, err := json.Marshal(hashInput)
	if err != nil {
		return fmt.Errorf("failed to marshal fingerprint data: %w", err)
	}

	// Generate SHA-256 hash
	hash := sha256.Sum256(data)
	fp.Hash = hex.EncodeToString(hash[:])

	return nil
}

// Matches compares two device fingerprints for equality
func (fp *DeviceFingerprint) Matches(other *DeviceFingerprint) bool {
	if fp == nil || other == nil {
		return false
	}

	// Primary comparison: hash equality
	if fp.Hash != "" && other.Hash != "" {
		return fp.Hash == other.Hash
	}

	// Fallback comparison: individual components
	return fp.matchesComponents(other)
}

// matchesComponents performs component-wise comparison
func (fp *DeviceFingerprint) matchesComponents(other *DeviceFingerprint) bool {
	// Critical components that must match
	if fp.Hostname != other.Hostname {
		return false
	}

	if fp.UserID != other.UserID || fp.Username != other.Username {
		return false
	}

	// System identifiers (if available)
	if fp.SystemUUID != "" && other.SystemUUID != "" && fp.SystemUUID != other.SystemUUID {
		return false
	}

	if fp.MachineID != "" && other.MachineID != "" && fp.MachineID != other.MachineID {
		return false
	}

	// MAC address comparison (at least one must match)
	if !fp.HasMatchingMAC(other) {
		return false
	}

	// Installation time must be close (within 1 hour to account for clock drift)
	if !fp.InstallTime.IsZero() && !other.InstallTime.IsZero() {
		diff := fp.InstallTime.Sub(other.InstallTime)
		if diff < 0 {
			diff = -diff
		}
		if diff > time.Hour {
			return false
		}
	}

	return true
}

// HasMatchingMAC checks if fingerprints have at least one matching MAC address
func (fp *DeviceFingerprint) HasMatchingMAC(other *DeviceFingerprint) bool {
	for _, mac1 := range fp.MACAddresses {
		for _, mac2 := range other.MACAddresses {
			if strings.EqualFold(mac1, mac2) {
				return true
			}
		}
	}
	return false
}

// GetRiskLevel assesses the risk level of fingerprint differences
func (fp *DeviceFingerprint) GetRiskLevel(other *DeviceFingerprint) RiskLevel {
	if fp.Matches(other) {
		return RiskLevelLow
	}

	// High risk: Different user or hostname
	if fp.Username != other.Username || fp.Hostname != other.Hostname {
		return RiskLevelHigh
	}

	// Medium risk: Different system identifiers
	if (fp.SystemUUID != "" && other.SystemUUID != "" && fp.SystemUUID != other.SystemUUID) ||
		(fp.MachineID != "" && other.MachineID != "" && fp.MachineID != other.MachineID) {
		return RiskLevelMedium
	}

	// Medium risk: No matching MAC addresses
	if !fp.HasMatchingMAC(other) {
		return RiskLevelMedium
	}

	return RiskLevelLow
}

// String returns a human-readable representation of the fingerprint
func (fp *DeviceFingerprint) String() string {
	return fmt.Sprintf("DeviceFingerprint{Host:%s, User:%s, MACs:%v, Hash:%s}",
		fp.Hostname, fp.Username, fp.MACAddresses, fp.Hash[:8])
}

// RiskLevel represents the security risk level
type RiskLevel int

const (
	RiskLevelLow RiskLevel = iota
	RiskLevelMedium
	RiskLevelHigh
	RiskLevelCritical
)

func (r RiskLevel) String() string {
	switch r {
	case RiskLevelLow:
		return "low"
	case RiskLevelMedium:
		return "medium"
	case RiskLevelHigh:
		return "high"
	case RiskLevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Helper functions for platform-specific system identification

func getMACAddresses() ([]string, error) {
	var addresses []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, iface := range interfaces {
		// Skip loopback interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Skip interfaces without hardware address
		if len(iface.HardwareAddr) == 0 {
			continue
		}

		addresses = append(addresses, iface.HardwareAddr.String())
	}

	// Sort for consistent ordering
	sort.Strings(addresses)

	return addresses, nil
}

func getSystemUUID() string {
	switch runtime.GOOS {
	case "darwin":
		return getMacOSSystemUUID()
	case "windows":
		return getWindowsMachineGUID()
	case "linux":
		return getLinuxMachineID()
	default:
		return ""
	}
}

func getMachineID() string {
	switch runtime.GOOS {
	case "linux":
		return getLinuxMachineID()
	case "darwin":
		// On macOS, use system UUID as machine ID
		return getMacOSSystemUUID()
	case "windows":
		return getWindowsMachineGUID()
	default:
		return ""
	}
}

// getInstallationTime is implemented in crypto.go as getCloudWorkstationInstallTime

// Platform-specific implementations use functions from crypto.go

// DeviceFingerprintError represents fingerprinting-related errors
type DeviceFingerprintError struct {
	Operation string
	Err       error
}

func (e *DeviceFingerprintError) Error() string {
	return fmt.Sprintf("device fingerprinting %s failed: %v", e.Operation, e.Err)
}

func (e *DeviceFingerprintError) Unwrap() error {
	return e.Err
}

// Common fingerprinting errors
var (
	ErrFingerprintMismatch   = &DeviceFingerprintError{Operation: "validation", Err: fmt.Errorf("device fingerprint mismatch")}
	ErrFingerprintGeneration = &DeviceFingerprintError{Operation: "generation", Err: fmt.Errorf("fingerprint generation failed")}
)
