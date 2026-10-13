// Package cli provides shared token CLI implementations for Prism v0.5.13+
package cli

import (
	"fmt"
	"os"
	"os/user"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// projectInvitationsSharedCreate creates a shared invitation token
func (a *App) projectInvitationsSharedCreate(args []string, name, role, message string, redemptionLimit int, expiresIn, expiresOn string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism project invitations shared create <project>")
	}

	projectName := args[0]

	// Validate inputs
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	if redemptionLimit <= 0 {
		return fmt.Errorf("--redemption-limit must be positive")
	}

	// Get project ID by name
	projectResponse, err := a.apiClient.ListProjects(a.ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	var projectID string
	for _, proj := range projectResponse.Projects {
		if proj.Name == projectName {
			projectID = proj.ID
			break
		}
	}

	if projectID == "" {
		return fmt.Errorf("project not found: %s", projectName)
	}

	// Use OS username for attribution until a full auth system is in place (#484).
	createdBy := "cli-user"
	if u, err := user.Current(); err == nil && u.Username != "" {
		createdBy = u.Username
	}

	// Build request
	req := &types.CreateSharedTokenRequest{
		ProjectID:       projectID,
		Name:            name,
		Role:            types.ProjectRole(role),
		Message:         message,
		RedemptionLimit: redemptionLimit,
		ExpiresIn:       expiresIn,
		CreatedBy:       createdBy,
	}

	// Parse expires-on if provided
	if expiresOn != "" {
		expiresAt, err := time.Parse("2006-01-02", expiresOn)
		if err != nil {
			return fmt.Errorf("invalid expires-on date (use YYYY-MM-DD): %w", err)
		}
		req.ExpiresAt = &expiresAt
		req.ExpiresIn = ""
	}

	fmt.Printf("🎫 Creating shared token for project '%s'...\n", projectName)
	fmt.Printf("   Name: %s\n", name)
	fmt.Printf("   Redemption limit: %d\n", redemptionLimit)
	fmt.Printf("   Role: %s\n", role)
	if message != "" {
		fmt.Printf("   Message: %s\n", message)
	}
	if req.ExpiresAt != nil {
		fmt.Printf("   Expires on: %s\n", req.ExpiresAt.Format("2006-01-02"))
	} else {
		fmt.Printf("   Expires in: %s\n", expiresIn)
	}
	fmt.Println()

	// Create shared token
	token, err := a.apiClient.CreateSharedToken(a.ctx, projectID, req)
	if err != nil {
		return fmt.Errorf("failed to create shared token: %w", err)
	}

	// Display result
	fmt.Println("🎫 Shared Invitation Token Generated")
	fmt.Println()
	fmt.Printf("   Token: %s\n", token.Token)
	fmt.Printf("   Project: %s\n", projectName)
	fmt.Printf("   Role: %s\n", token.Role)
	fmt.Printf("   Redemptions: %d / %d\n", token.RedemptionCount, token.RedemptionLimit)
	if token.ExpiresAt != nil {
		fmt.Printf("   Expires: %s\n", token.ExpiresAt.Format("Jan 2, 2006"))
	}
	fmt.Println()
	fmt.Println("   Share this token with all participants:")
	fmt.Printf("   📧 Email: Include %s in registration emails\n", token.Token)
	fmt.Printf("   🔗 URL: https://prism.dev/join/%s\n", token.Token)
	fmt.Printf("   💡 This single token works for all %d participants (first-come-first-served)\n", token.RedemptionLimit)
	fmt.Println()
	fmt.Println("   Generate QR code:")
	fmt.Printf("   prism project invitations shared qr %s\n", token.Token)

	return nil
}

// projectInvitationsSharedShow displays shared token information
func (a *App) projectInvitationsSharedShow(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism project invitations shared show <token>")
	}

	tokenCode := args[0]

	// Get token
	token, err := a.apiClient.GetSharedToken(a.ctx, tokenCode)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Display token info
	fmt.Println("🎫 Shared Token Information")
	fmt.Println()
	fmt.Printf("   Token: %s\n", token.Token)
	fmt.Printf("   Name: %s\n", token.Name)
	fmt.Printf("   Project ID: %s\n", token.ProjectID)
	fmt.Printf("   Role: %s\n", token.Role)
	fmt.Printf("   Status: %s\n", token.Status)
	fmt.Println()
	fmt.Println("   Redemptions:")
	fmt.Printf("   • Used: %d / %d (%d%%)\n", token.RedemptionCount, token.RedemptionLimit,
		(token.RedemptionCount*100)/token.RedemptionLimit)
	fmt.Printf("   • Remaining: %d\n", token.RemainingRedemptions())
	fmt.Println()
	if token.ExpiresAt != nil {
		fmt.Printf("   Expires: %s\n", token.ExpiresAt.Format("Jan 2, 2006 15:04 MST"))
		if token.IsExpired() {
			fmt.Println("   ⚠️  Token has expired")
		}
	} else {
		fmt.Println("   Expires: Never")
	}
	fmt.Printf("   Created: %s\n", token.CreatedAt.Format("Jan 2, 2006 15:04 MST"))
	fmt.Println()

	// Show recent redemptions
	if len(token.Redemptions) > 0 {
		fmt.Println("   Recent Redemptions:")
		count := len(token.Redemptions)
		start := 0
		if count > 10 {
			start = count - 10
		}
		for i := start; i < count; i++ {
			redemption := token.Redemptions[i]
			fmt.Printf("   %d. %s - %s\n", redemption.Position, redemption.RedeemedBy,
				redemption.RedeemedAt.Format("Jan 2, 2006 15:04"))
		}
		if count > 10 {
			fmt.Printf("   ... and %d more\n", count-10)
		}
	}

	return nil
}

// projectInvitationsSharedList lists shared tokens for a project
func (a *App) projectInvitationsSharedList(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism project invitations shared list <project>")
	}

	projectName := args[0]

	// Get project ID by name
	projectResponse, err := a.apiClient.ListProjects(a.ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	var projectID string
	for _, proj := range projectResponse.Projects {
		if proj.Name == projectName {
			projectID = proj.ID
			break
		}
	}

	if projectID == "" {
		return fmt.Errorf("project not found: %s", projectName)
	}

	// List tokens
	tokens, err := a.apiClient.ListSharedTokens(a.ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to list shared tokens: %w", err)
	}

	if len(tokens) == 0 {
		fmt.Printf("No shared tokens found for project '%s'\n", projectName)
		fmt.Println()
		fmt.Println("Create one with:")
		fmt.Printf("  prism project invitations shared create %s --name \"Token Name\" --redemption-limit 60\n", projectName)
		return nil
	}

	// Display tokens
	fmt.Printf("📋 Shared Tokens for Project '%s'\n\n", projectName)

	for _, token := range tokens {
		statusIcon := "✅"
		switch token.Status {
		case types.SharedTokenExpired:
			statusIcon = "⏰"
		case types.SharedTokenRevoked:
			statusIcon = "❌"
		case types.SharedTokenExhausted:
			statusIcon = "🔒"
		}

		fmt.Printf("%s %s\n", statusIcon, token.Name)
		fmt.Printf("   Token: %s\n", token.Token)
		fmt.Printf("   Status: %s\n", token.Status)
		fmt.Printf("   Redemptions: %d / %d (%d%%)\n", token.RedemptionCount, token.RedemptionLimit,
			(token.RedemptionCount*100)/max(1, token.RedemptionLimit))
		if token.ExpiresAt != nil {
			fmt.Printf("   Expires: %s\n", token.ExpiresAt.Format("Jan 2, 2006"))
		}
		fmt.Println()
	}

	return nil
}

// projectInvitationsSharedExtend extends token expiration
func (a *App) projectInvitationsSharedExtend(args []string, addDays int) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism project invitations shared extend <token> --add-days <N>")
	}

	tokenCode := args[0]

	if addDays <= 0 {
		return fmt.Errorf("--add-days must be positive")
	}

	fmt.Printf("🔧 Extending token expiration by %d day(s)...\n", addDays)

	// Extend token
	err := a.apiClient.ExtendSharedToken(a.ctx, tokenCode, addDays)
	if err != nil {
		return fmt.Errorf("failed to extend token: %w", err)
	}

	// Get updated token
	token, err := a.apiClient.GetSharedToken(a.ctx, tokenCode)
	if err != nil {
		return fmt.Errorf("failed to get updated token: %w", err)
	}

	fmt.Println("✅ Token expiration extended")
	fmt.Println()
	fmt.Printf("   Token: %s\n", token.Token)
	if token.ExpiresAt != nil {
		fmt.Printf("   New expiration: %s\n", token.ExpiresAt.Format("Jan 2, 2006 15:04 MST"))
	}
	fmt.Println()
	fmt.Printf("💡 All %d redeemed participants automatically get the extension\n", token.RedemptionCount)

	return nil
}

// projectInvitationsSharedRevoke revokes a shared token
func (a *App) projectInvitationsSharedRevoke(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism project invitations shared revoke <token>")
	}

	tokenCode := args[0]

	fmt.Printf("⚠️  Revoking token %s...\n", tokenCode)
	fmt.Println()
	fmt.Println("   This will:")
	fmt.Println("   • Prevent any new redemptions")
	fmt.Println("   • Existing participants retain access")
	fmt.Println()

	// Revoke token
	err := a.apiClient.RevokeSharedToken(a.ctx, tokenCode)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	fmt.Println("✅ Token revoked successfully")
	fmt.Println()
	fmt.Println("   No new redemptions will be accepted")

	return nil
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// projectInvitationsSharedQR generates a QR code for a shared token
func (a *App) projectInvitationsSharedQR(args []string, output string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism project invitations shared qr <token> [--output <file.png>]")
	}

	tokenCode := args[0]

	fmt.Printf("🔲 Generating QR code for token %s...\n", tokenCode)

	qrBytes, err := a.apiClient.GetSharedTokenQR(a.ctx, tokenCode)
	if err != nil {
		// Fall back to informational display if API call fails
		redemptionURL := fmt.Sprintf("https://prism.dev/join/%s", tokenCode)
		fmt.Println()
		fmt.Println("⚠️  Could not fetch QR code from daemon:", err)
		fmt.Println()
		fmt.Printf("   Redemption URL: %s\n", redemptionURL)
		fmt.Printf("   curl http://localhost:8947/api/v1/invitations/shared/%s/qr?format=png > qr.png\n", tokenCode)
		return nil
	}

	if output != "" {
		if writeErr := os.WriteFile(output, qrBytes, 0600); writeErr != nil {
			return fmt.Errorf("failed to write QR code to %s: %w", output, writeErr)
		}
		fmt.Println()
		fmt.Printf("✅ QR code saved to: %s\n", output)
		fmt.Println()
		fmt.Println("   Print this QR code and display it at:")
		fmt.Println("   • Workshop registration desk")
		fmt.Println("   • Conference booth")
		fmt.Println("   • Classroom door")
		fmt.Println("   • Workshop materials")
		fmt.Println()
		fmt.Println("   Participants can scan with their phone camera to redeem")
	} else {
		fmt.Println()
		fmt.Printf("✅ QR code generated (%d bytes). Use --output to save to a file:\n", len(qrBytes))
		fmt.Printf("   prism project invitations shared qr %s --output workshop-qr.png\n", tokenCode)
	}

	return nil
}
