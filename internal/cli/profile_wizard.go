// Package cli provides interactive profile management wizards for Prism.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/scttfrdmn/prism/pkg/profile"
)

// ProfileWizard provides interactive profile creation and management
type ProfileWizard struct {
	config         *Config
	profileManager *profile.ManagerEnhanced
	reader         *bufio.Reader
}

// NewProfileWizard creates a new profile wizard
func NewProfileWizard(config *Config) (*ProfileWizard, error) {
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		return nil, err
	}

	return &ProfileWizard{
		config:         config,
		profileManager: profileManager,
		reader:         bufio.NewReader(os.Stdin),
	}, nil
}

// RunInteractiveSetup runs the interactive profile setup wizard
func (pw *ProfileWizard) RunInteractiveSetup() error {
	pw.showWelcome()

	// Check current profiles
	profiles, err := pw.profileManager.ListProfilesWithIDs()
	if err != nil {
		return fmt.Errorf("failed to get current profiles: %w", err)
	}

	if len(profiles) > 0 {
		fmt.Printf("📋 You have %d existing profiles:\n", len(profiles))
		for _, p := range profiles {
			fmt.Printf("   %s (%s)\n", p.Profile.Name, p.ID)
		}
		fmt.Println()

		if !pw.promptYesNo("Would you like to add another profile?", true) {
			fmt.Println("👋 Profile setup cancelled.")
			return nil
		}
		fmt.Println()
	}

	// Start profile creation wizard
	return pw.runProfileCreationWizard()
}

// showWelcome displays the wizard welcome message
func (pw *ProfileWizard) showWelcome() {
	fmt.Printf("🧙‍♂️ %s\n", color.CyanString("Prism Profile Setup Wizard"))
	fmt.Println()
	fmt.Println("This wizard will help you set up a new profile for working with AWS accounts.")
	fmt.Println("Profiles let you easily switch between different AWS accounts or regions.")
	fmt.Println()
}

// runProfileCreationWizard runs the main profile creation workflow
func (pw *ProfileWizard) runProfileCreationWizard() error {
	// Step 1: Choose profile type
	profileType := pw.chooseProfileType()
	if profileType == "" {
		return nil // User cancelled
	}

	switch profileType {
	case "personal":
		return pw.createPersonalProfile()
	case "invitation":
		return pw.createInvitationProfile()
	default:
		return fmt.Errorf("unknown profile type: %s", profileType)
	}
}

// chooseProfileType prompts user to choose between personal and invitation profiles
func (pw *ProfileWizard) chooseProfileType() string {
	fmt.Printf("🎯 %s\n", color.YellowString("Step 1: Choose Profile Type"))
	fmt.Println()
	fmt.Println("There are two types of profiles you can create:")
	fmt.Println("  1. Personal Profile  - Connect to your own AWS account")
	fmt.Println("  2. Invitation Profile - Access shared resources via invitation")
	fmt.Println()

	for {
		choice := pw.promptString("Choose profile type (1 for Personal, 2 for Invitation, or 'q' to quit)", "1")

		switch strings.ToLower(choice) {
		case "1", "personal", "p":
			return "personal"
		case "2", "invitation", "i":
			return "invitation"
		case "q", "quit", "cancel":
			fmt.Println("👋 Profile setup cancelled.")
			return ""
		default:
			fmt.Printf("❌ Invalid choice '%s'. Please enter 1, 2, or 'q'.\n\n", choice)
		}
	}
}

// createPersonalProfile creates a new personal profile interactively
func (pw *ProfileWizard) createPersonalProfile() error {
	fmt.Printf("\n🔧 %s\n", color.GreenString("Creating Personal Profile"))
	fmt.Println()
	fmt.Println("A personal profile connects to your AWS account using AWS CLI credentials.")
	fmt.Println()

	// Step 1: Profile name
	name := pw.getProfileName()
	if name == "" {
		return nil // User cancelled
	}

	// Step 2: AWS Profile
	awsProfile := pw.getAWSProfile()
	if awsProfile == "" {
		return nil // User cancelled
	}

	// Step 3: AWS Region
	region := pw.getAWSRegion()
	if region == "" {
		return nil // User cancelled
	}

	// Step 4: Validate AWS credentials
	if !pw.validateAWSCredentials(awsProfile, region) {
		fmt.Println("❌ AWS credential validation failed. Please check your AWS configuration.")
		return nil
	}

	// Step 5: Create the profile
	newProfile := profile.Profile{
		Type:       profile.ProfileTypePersonal,
		Name:       name,
		AWSProfile: awsProfile,
		Region:     region,
		LastUsed:   func() *time.Time { t := time.Now(); return &t }(),
	}

	if err := pw.profileManager.AddProfile(newProfile); err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}

	// Success message
	pw.showProfileCreated(newProfile)

	// Ask if they want to switch to the new profile
	if pw.promptYesNo("Would you like to switch to this profile now?", true) {
		profiles, _ := pw.profileManager.ListProfilesWithIDs()
		for _, p := range profiles {
			if p.Profile.Name == name {
				if err := pw.profileManager.SwitchProfile(p.ID); err != nil {
					fmt.Printf("⚠️  Failed to switch to profile: %v\n", err)
				} else {
					fmt.Printf("✅ Switched to profile '%s'\n", name)
				}
				break
			}
		}
	}

	return nil
}

// createInvitationProfile creates a new invitation profile interactively
func (pw *ProfileWizard) createInvitationProfile() error {
	fmt.Printf("\n🔧 %s\n", color.GreenString("Creating Invitation Profile"))
	fmt.Println()
	fmt.Println("An invitation profile gives you access to shared Prism resources.")
	fmt.Println("You'll need an invitation code provided by the resource owner.")
	fmt.Println()

	// Step 1: Profile name
	name := pw.getProfileName()
	if name == "" {
		return nil // User cancelled
	}

	// Step 2: Invitation token
	token := pw.getInvitationToken()
	if token == "" {
		return nil // User cancelled
	}

	// Step 3: Owner account
	owner := pw.getOwnerAccount()
	if owner == "" {
		return nil // User cancelled
	}

	// Step 4: AWS Region (optional)
	region := pw.getAWSRegion()

	// Step 5: Create the profile
	newProfile := profile.Profile{
		Type:            profile.ProfileTypeInvitation,
		Name:            name,
		Region:          region,
		InvitationToken: token,
		OwnerAccount:    owner,
		LastUsed:        func() *time.Time { t := time.Now(); return &t }(),
	}

	if err := pw.profileManager.AddProfile(newProfile); err != nil {
		return fmt.Errorf("failed to create invitation profile: %w", err)
	}

	// Success message
	pw.showProfileCreated(newProfile)

	return nil
}

// getProfileName prompts for and validates profile name
func (pw *ProfileWizard) getProfileName() string {
	fmt.Printf("📝 %s\n", color.BlueString("Profile Name"))
	fmt.Println("Choose a memorable name for this profile (e.g., 'work-aws', 'research-lab').")
	fmt.Println()

	for {
		name := pw.promptString("Profile name", "")
		if name == "" {
			if pw.promptYesNo("Cancel profile creation?", false) {
				return ""
			}
			continue
		}

		// Validate name (no spaces, reasonable length)
		if !pw.isValidProfileName(name) {
			fmt.Println("❌ Profile name should be 2-50 characters, letters/numbers/dashes only.")
			continue
		}

		// Check if name already exists
		if pw.profileNameExists(name) {
			fmt.Printf("❌ A profile named '%s' already exists. Please choose a different name.\n", name)
			continue
		}

		return name
	}
}

// getAWSProfile prompts for AWS profile selection
func (pw *ProfileWizard) getAWSProfile() string {
	fmt.Printf("\n🔑 %s\n", color.BlueString("AWS Profile"))
	fmt.Println("This should match a profile in your ~/.aws/config or ~/.aws/credentials.")

	// Try to detect available AWS profiles from both config files
	awsProfiles := pw.detectAWSProfilesDetailed()
	if len(awsProfiles) > 0 {
		fmt.Println("Detected AWS profiles:")
		for _, p := range awsProfiles {
			label := ""
			switch p.credType {
			case "sso":
				label = "  [SSO / Identity Center]"
			case "role":
				label = "  [assume role]"
			case "process":
				label = "  [credential process]"
			}
			fmt.Printf("  %-30s%s\n", p.name, label)
		}
	}
	fmt.Println()

	for {
		defaultProfile := "default"
		if len(awsProfiles) > 0 {
			defaultProfile = awsProfiles[0].name
		}

		awsProfile := pw.promptString("AWS profile name", defaultProfile)
		if awsProfile == "" {
			if pw.promptYesNo("Cancel profile creation?", false) {
				return ""
			}
			continue
		}

		return awsProfile
	}
}

// getAWSRegion prompts for AWS region selection
func (pw *ProfileWizard) getAWSRegion() string {
	fmt.Printf("\n🌍 %s\n", color.BlueString("AWS Region"))
	fmt.Println("Choose your preferred AWS region (or leave empty for AWS default).")
	fmt.Println("Popular regions: us-east-1, us-west-2, eu-west-1, ap-southeast-1")
	fmt.Println()

	region := pw.promptString("AWS region (optional)", "")

	// Validate region format if provided
	if region != "" && !pw.isValidRegion(region) {
		fmt.Println("⚠️  Region format looks unusual, but proceeding anyway.")
	}

	return region
}

// getInvitationToken prompts for invitation token
func (pw *ProfileWizard) getInvitationToken() string {
	fmt.Printf("\n🎫 %s\n", color.BlueString("Invitation Token"))
	fmt.Println("Enter the invitation code provided by the resource owner.")
	fmt.Println()

	for {
		token := pw.promptString("Invitation token", "")
		if token == "" {
			if pw.promptYesNo("Cancel profile creation?", false) {
				return ""
			}
			continue
		}

		if len(token) < 10 {
			fmt.Println("❌ Invitation token seems too short. Please check and try again.")
			continue
		}

		return token
	}
}

// getOwnerAccount prompts for owner account
func (pw *ProfileWizard) getOwnerAccount() string {
	fmt.Printf("\n👤 %s\n", color.BlueString("Owner Account"))
	fmt.Println("Enter the AWS account ID or email of the resource owner.")
	fmt.Println()

	for {
		owner := pw.promptString("Owner account", "")
		if owner == "" {
			if pw.promptYesNo("Cancel profile creation?", false) {
				return ""
			}
			continue
		}

		return owner
	}
}

// validateAWSCredentials validates AWS credentials
func (pw *ProfileWizard) validateAWSCredentials(awsProfile, region string) bool {
	fmt.Printf("\n🔍 %s\n", color.MagentaString("Validating AWS Credentials"))
	fmt.Println("Testing your AWS credentials...")

	// Simple validation using AWS CLI if available
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--profile", awsProfile) //nolint:gosec // G204: aws CLI, args are argv not shell
	if region != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("AWS_DEFAULT_REGION=%s", region))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ AWS validation failed: %v\n", err)
		fmt.Printf("Output: %s\n", string(output))

		fmt.Println("\n💡 To fix this:")
		fmt.Println("   1. Install AWS CLI: https://aws.amazon.com/cli/")
		fmt.Printf("   2. Configure credentials: aws configure --profile %s\n", awsProfile)
		fmt.Println("   3. Verify access: aws sts get-caller-identity")

		return pw.promptYesNo("Continue anyway? (profile will be created but may not work)", false)
	}

	fmt.Println("✅ AWS credentials validated successfully!")
	return true
}

// showProfileCreated displays success message
func (pw *ProfileWizard) showProfileCreated(newProfile profile.Profile) {
	fmt.Printf("\n🎉 %s\n", color.GreenString("Profile Created Successfully!"))
	fmt.Println()
	fmt.Printf("✅ Profile '%s' has been created\n", newProfile.Name)
	fmt.Printf("📋 Type: %s\n", pw.getProfileTypeDisplay(newProfile.Type))

	if newProfile.Type == profile.ProfileTypePersonal {
		fmt.Printf("🔑 AWS Profile: %s\n", newProfile.AWSProfile)
		if newProfile.Region != "" {
			fmt.Printf("🌍 Region: %s\n", newProfile.Region)
		}
	}

	fmt.Println()
	fmt.Println("💡 What's next?")
	fmt.Println("   • List all profiles: prism profiles list")
	fmt.Println("   • Switch profiles: prism profiles switch <profile-id>")
	fmt.Println("   • Launch an instance: prism workspace launch python-ml my-project")
	fmt.Println()
}

// Helper methods

func (pw *ProfileWizard) promptString(prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	input, _ := pw.reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}

func (pw *ProfileWizard) promptYesNo(prompt string, defaultValue bool) bool {
	defaultChar := "y/N"
	if defaultValue {
		defaultChar = "Y/n"
	}

	fmt.Printf("%s [%s]: ", prompt, defaultChar)
	input, _ := pw.reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		return defaultValue
	}

	return input == "y" || input == "yes"
}

func (pw *ProfileWizard) isValidProfileName(name string) bool {
	if len(name) < 2 || len(name) > 50 {
		return false
	}
	// Allow letters, numbers, dashes, underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return matched
}

func (pw *ProfileWizard) isValidRegion(region string) bool {
	// Basic region format validation (region-direction-number)
	matched, _ := regexp.MatchString(`^[a-z]+-[a-z]+-\d+[a-z]?$`, region)
	return matched
}

func (pw *ProfileWizard) profileNameExists(name string) bool {
	profiles, err := pw.profileManager.ListProfilesWithIDs()
	if err != nil {
		return false
	}

	for _, p := range profiles {
		if p.Profile.Name == name {
			return true
		}
	}
	return false
}

// awsProfileInfo holds a detected AWS profile name and its credential type.
type awsProfileInfo struct {
	name     string
	credType string // "sso", "role", "process", or "" for static/unknown
}

// detectAWSProfilesDetailed reads both ~/.aws/config and ~/.aws/credentials
// and returns all profiles with their credential type annotated.
// ~/.aws/config is read first so SSO/role profiles appear at the top.
func (pw *ProfileWizard) detectAWSProfilesDetailed() []awsProfileInfo {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	var profiles []awsProfileInfo

	// --- ~/.aws/config ---
	// Profiles are [default] or [profile <name>].
	// We read each section's keys to determine the credential type.
	if f, err := os.Open(homeDir + "/.aws/config"); err == nil { //nolint:gosec // G304: fixed path under the user's home directory
		defer func() { _ = f.Close() }() // read-only; close error is not actionable
		var current string
		var info awsProfileInfo
		flush := func() {
			if current != "" && !seen[current] {
				seen[current] = true
				profiles = append(profiles, info)
			}
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				flush()
				section := strings.Trim(line, "[]")
				switch {
				case section == "default":
					current = "default"
				case strings.HasPrefix(section, "profile "):
					current = strings.TrimPrefix(section, "profile ")
				default:
					current = "" // e.g. [sso-session ...] — not a profile section
				}
				info = awsProfileInfo{name: current}
				continue
			}
			if current == "" {
				continue
			}
			key := strings.SplitN(line, "=", 2)[0]
			key = strings.TrimSpace(key)
			switch key {
			case "sso_start_url", "sso_session":
				info.credType = "sso"
			case "role_arn":
				if info.credType == "" {
					info.credType = "role"
				}
			case "credential_process":
				if info.credType == "" {
					info.credType = "process"
				}
			}
		}
		flush()
	}

	// --- ~/.aws/credentials ---
	// Profiles are [<name>] with static access keys.
	if f, err := os.Open(homeDir + "/.aws/credentials"); err == nil { //nolint:gosec // G304: fixed path under the user's home directory
		defer func() { _ = f.Close() }() // read-only; close error is not actionable
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				name := strings.Trim(line, "[]")
				if name != "" && !seen[name] {
					seen[name] = true
					profiles = append(profiles, awsProfileInfo{name: name})
				}
			}
		}
	}

	return profiles
}

func (pw *ProfileWizard) getProfileTypeDisplay(profileType profile.ProfileType) string {
	switch profileType {
	case profile.ProfileTypePersonal:
		return "Personal"
	case profile.ProfileTypeInvitation:
		return "Invitation"
	default:
		return "Unknown"
	}
}
