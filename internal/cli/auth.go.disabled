package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/spf13/cobra"
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage API authentication",
	Long:  `Manage API authentication for CloudWorkstation`,
}

// generateCmd represents the auth generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new API key",
	Long:  `Generate a new API key for authenticating with the CloudWorkstation API`,
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		
		// Generate a new API key
		resp, err := client.GenerateAPIKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating API key: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("API Key generated successfully at %s\n", resp.CreatedAt.Format(time.RFC3339))
		fmt.Printf("\nAPI Key: %s\n", resp.APIKey)
		fmt.Printf("\nIMPORTANT: Store this key securely. It will not be shown again.\n")
		fmt.Printf("All future API requests will require this key.\n")
	},
}

// statusCmd represents the auth status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Long:  `Check the current authentication status for CloudWorkstation API`,
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		
		// Get authentication status
		status, err := client.GetAuthStatus()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking authentication status: %v\n", err)
			os.Exit(1)
		}
		
		if status.AuthEnabled {
			fmt.Println("Authentication is enabled")
			fmt.Printf("API Key created at: %s\n", status.CreatedAt.Format(time.RFC3339))
			
			if status.Authenticated {
				fmt.Println("Current client is authenticated")
			} else {
				fmt.Println("Current client is NOT authenticated")
				fmt.Println("Use 'cws config set-key' to configure your API key")
			}
		} else {
			fmt.Println("Authentication is not enabled")
			fmt.Println("Use 'cws auth generate' to enable authentication")
		}
	},
}

// revokeCmd represents the auth revoke command
var revokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke the current API key",
	Long:  `Revoke the current API key for CloudWorkstation API`,
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		
		// Ask for confirmation
		fmt.Println("WARNING: Revoking the API key will invalidate all clients using this key.")
		fmt.Print("Are you sure you want to continue? (y/n): ")
		
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" {
			fmt.Println("Operation cancelled")
			return
		}
		
		// Revoke the API key
		err := client.RevokeAPIKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error revoking API key: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Println("API key revoked successfully")
	},
}

// setKeyCmd represents the config set-key command
var setKeyCmd = &cobra.Command{
	Use:   "set-key [key]",
	Short: "Set the API key for authentication",
	Long:  `Configure the API key for authenticating with the CloudWorkstation API`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiKey := args[0]
		
		// Get the client
		client := getClient()
		
		// Set the API key
		client.SetAPIKey(apiKey)
		
		// Test the key by making a request
		_, err := client.GetAuthStatus()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error validating API key: %v\n", err)
			fmt.Println("The key has been saved, but it may not be valid")
		} else {
			fmt.Println("API key configured successfully")
		}
		
		// Save the key to config
		if err := saveAPIKey(apiKey); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving API key to config: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(generateCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(revokeCmd)
	
	configCmd.AddCommand(setKeyCmd)
}

// saveAPIKey saves the API key to the local config file
func saveAPIKey(apiKey string) error {
	// Implementation will depend on how config is stored
	// This is a placeholder
	return nil
}