// Device Manager is an administrative tool for managing device bindings
//
// This tool helps administrators track and manage device bindings for
// CloudWorkstation invitation profiles.
//
// Usage:
//   go run device-manager.go [command] [flags]
//
// Commands:
//   list         List all registered devices for an invitation
//   revoke       Revoke a specific device
//   revoke-all   Revoke all devices for an invitation
//   validate     Validate if a specific device is authorized
//
// Examples:
//   go run device-manager.go list --token inv-abc123def456
//   go run device-manager.go revoke --token inv-abc123def456 --device device-xyz789
//   go run device-manager.go revoke-all --token inv-abc123def456
//   go run device-manager.go validate --token inv-abc123def456 --device device-xyz789
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile/security"
)

func main() {
	// Define commands
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	revokeCmd := flag.NewFlagSet("revoke", flag.ExitOnError)
	revokeAllCmd := flag.NewFlagSet("revoke-all", flag.ExitOnError)
	validateCmd := flag.NewFlagSet("validate", flag.ExitOnError)

	// Define flags for commands
	listToken := listCmd.String("token", "", "Invitation token to list devices for")
	listFormat := listCmd.String("format", "text", "Output format (text, json)")

	revokeToken := revokeCmd.String("token", "", "Invitation token for the device to revoke")
	revokeDevice := revokeCmd.String("device", "", "Device ID to revoke")

	revokeAllToken := revokeAllCmd.String("token", "", "Invitation token to revoke all devices for")
	revokeAllForce := revokeAllCmd.Bool("force", false, "Skip confirmation prompt")

	validateToken := validateCmd.String("token", "", "Invitation token to validate device against")
	validateDevice := validateCmd.String("device", "", "Device ID to validate")

	// Display help if no arguments
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Parse command
	switch os.Args[1] {
	case "list":
		listCmd.Parse(os.Args[2:])
		if *listToken == "" {
			fmt.Println("Error: token is required for list command")
			listCmd.PrintDefaults()
			os.Exit(1)
		}
		handleListCommand(*listToken, *listFormat)

	case "revoke":
		revokeCmd.Parse(os.Args[2:])
		if *revokeToken == "" || *revokeDevice == "" {
			fmt.Println("Error: token and device are required for revoke command")
			revokeCmd.PrintDefaults()
			os.Exit(1)
		}
		handleRevokeCommand(*revokeToken, *revokeDevice)

	case "revoke-all":
		revokeAllCmd.Parse(os.Args[2:])
		if *revokeAllToken == "" {
			fmt.Println("Error: token is required for revoke-all command")
			revokeAllCmd.PrintDefaults()
			os.Exit(1)
		}
		handleRevokeAllCommand(*revokeAllToken, *revokeAllForce)

	case "validate":
		validateCmd.Parse(os.Args[2:])
		if *validateToken == "" || *validateDevice == "" {
			fmt.Println("Error: token and device are required for validate command")
			validateCmd.PrintDefaults()
			os.Exit(1)
		}
		handleValidateCommand(*validateToken, *validateDevice)

	case "help", "-h", "--help":
		printUsage()
		os.Exit(0)

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// Print usage information
func printUsage() {
	fmt.Println("Device Manager - Administrative tool for CloudWorkstation device bindings")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  device-manager [command] [flags]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  list         List all registered devices for an invitation")
	fmt.Println("  revoke       Revoke a specific device")
	fmt.Println("  revoke-all   Revoke all devices for an invitation")
	fmt.Println("  validate     Validate if a specific device is authorized")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  device-manager list --token inv-abc123def456")
	fmt.Println("  device-manager revoke --token inv-abc123def456 --device device-xyz789")
	fmt.Println("  device-manager revoke-all --token inv-abc123def456")
	fmt.Println("  device-manager validate --token inv-abc123def456 --device device-xyz789")
}

// Initialize registry client
func initRegistryClient() *security.RegistryClient {
	config := security.S3RegistryConfig{
		BucketName: os.Getenv("CWS_REGISTRY_BUCKET"),
		Region:     os.Getenv("CWS_REGISTRY_REGION"),
		Enabled:    true,
	}

	// If bucket not specified, use default
	if config.BucketName == "" {
		config.BucketName = "cloudworkstation-invitations"
	}

	// If region not specified, use default
	if config.Region == "" {
		config.Region = "us-west-2"
	}

	// Create registry client
	registry, err := security.NewRegistryClient(config)
	if err != nil {
		fmt.Printf("Error initializing registry client: %v\n", err)
		os.Exit(1)
	}

	return registry
}

// Handle list command
func handleListCommand(token string, format string) {
	registry := initRegistryClient()

	// Get devices for invitation
	devices, err := registry.GetInvitationDevices(token)
	if err != nil {
		fmt.Printf("Error retrieving devices: %v\n", err)
		os.Exit(1)
	}

	if len(devices) == 0 {
		fmt.Println("No devices found for this invitation")
		return
	}

	if format == "json" {
		// Output as JSON
		jsonData, err := json.MarshalIndent(devices, "", "  ")
		if err != nil {
			fmt.Printf("Error formatting JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
	} else {
		// Output as table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "DEVICE ID\tHOSTNAME\tUSERNAME\tREGISTERED")

		for _, device := range devices {
			deviceID := getStringValue(device, "device_id")
			hostname := getStringValue(device, "hostname")
			username := getStringValue(device, "username")
			timestamp := getStringValue(device, "timestamp")

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", deviceID, hostname, username, timestamp)
		}
		w.Flush()
	}
}

// Handle revoke command
func handleRevokeCommand(token, deviceID string) {
	registry := initRegistryClient()

	// Prompt for confirmation
	fmt.Printf("Are you sure you want to revoke device '%s' for invitation '%s'? [y/N] ", deviceID, token)
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" {
		fmt.Println("Revocation cancelled")
		return
	}

	// Revoke device
	err := registry.RevokeDevice(token, deviceID)
	if err != nil {
		fmt.Printf("Error revoking device: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully revoked device '%s' for invitation '%s'\n", deviceID, token)
}

// Handle revoke-all command
func handleRevokeAllCommand(token string, force bool) {
	registry := initRegistryClient()

	// Prompt for confirmation if not forced
	if !force {
		fmt.Printf("WARNING: This will revoke ALL devices for invitation '%s'.\nAre you absolutely sure? [y/N] ", token)
		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) != "y" {
			fmt.Println("Revocation cancelled")
			return
		}
	}

	// Revoke all devices
	err := registry.RevokeInvitation(token)
	if err != nil {
		fmt.Printf("Error revoking all devices: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully revoked all devices for invitation '%s'\n", token)
}

// Handle validate command
func handleValidateCommand(token, deviceID string) {
	registry := initRegistryClient()

	// Validate device
	valid, err := registry.ValidateDevice(token, deviceID)
	if err != nil {
		fmt.Printf("Error validating device: %v\n", err)
		os.Exit(1)
	}

	if valid {
		fmt.Printf("Device '%s' is valid for invitation '%s'\n", deviceID, token)
	} else {
		fmt.Printf("Device '%s' is NOT valid for invitation '%s'\n", deviceID, token)
		os.Exit(1)
	}
}

// Helper function to get string value from map
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case string:
			return v
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return "-"
}