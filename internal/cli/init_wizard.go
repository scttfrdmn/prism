// Package cli provides interactive onboarding wizard for CloudWorkstation.
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
)

// InitWizard provides interactive first-time setup for CloudWorkstation
type InitWizard struct {
	config *Config
	reader *bufio.Reader
}

// ResearchArea represents a research domain
type ResearchArea struct {
	Name                 string
	Description          string
	RecommendedPolicy    string   // Idle detection policy
	RecommendedTemplates []string // Template names to suggest
}

// Available research areas
var researchAreas = []ResearchArea{
	{
		Name:              "Machine Learning / Data Science",
		Description:       "Deep learning, neural networks, data analysis, ML workflows",
		RecommendedPolicy: "gpu", // 15min idle → stop for expensive GPU instances
		RecommendedTemplates: []string{
			"Python Machine Learning (Simplified)",
			"R Research Environment (Simplified)",
		},
	},
	{
		Name:              "Bioinformatics / Genomics",
		Description:       "Genomic analysis, sequencing, protein modeling, computational biology",
		RecommendedPolicy: "batch", // 60min idle → hibernate for long-running jobs
		RecommendedTemplates: []string{
			"Python Machine Learning (Simplified)",
			"Basic Ubuntu (APT)",
		},
	},
	{
		Name:              "Social Science / Statistics",
		Description:       "Statistical analysis, survey data, regression modeling, data visualization",
		RecommendedPolicy: "balanced", // 30min idle → hibernate
		RecommendedTemplates: []string{
			"R Research Environment (Simplified)",
			"Python Machine Learning (Simplified)",
		},
	},
	{
		Name:              "Other",
		Description:       "General research computing, custom workflows",
		RecommendedPolicy: "balanced", // 30min idle → hibernate
		RecommendedTemplates: []string{
			"Basic Ubuntu (APT)",
			"Python Machine Learning (Simplified)",
		},
	},
}

// NewInitWizard creates a new initialization wizard
func NewInitWizard(config *Config) *InitWizard {
	return &InitWizard{
		config: config,
		reader: bufio.NewReader(os.Stdin),
	}
}

// Run executes the complete initialization wizard
func (iw *InitWizard) Run() error {
	iw.showWelcome()

	// Step 1: AWS Configuration
	if err := iw.stepAWSConfiguration(); err != nil {
		return err
	}

	// Step 2: Research Area
	area := iw.stepResearchArea()
	if area == nil {
		fmt.Println("👋 Setup cancelled.")
		return nil
	}

	// Step 3: Budget (Optional)
	budget := iw.stepBudget()

	// Step 4: Hibernation Policy
	policy := iw.stepHibernation(area.RecommendedPolicy)

	// Step 5: Template Recommendations
	iw.stepTemplateRecommendations(area)

	// Save initialization status
	if err := iw.saveInitialization(area, budget, policy); err != nil {
		fmt.Printf("⚠️  Failed to save initialization: %v\n", err)
	}

	// Show completion message
	iw.showCompletion()

	return nil
}

// showWelcome displays the welcome message
func (iw *InitWizard) showWelcome() {
	fmt.Printf("\n🎉 %s\n", color.CyanString("Welcome to CloudWorkstation!"))
	fmt.Println()
	fmt.Println("This wizard will help you set up your research computing environment.")
	fmt.Println("The setup takes about 2 minutes.")
	fmt.Println()
	fmt.Println(color.YellowString("Press Ctrl+C at any time to exit."))
	fmt.Println()
}

// stepAWSConfiguration handles AWS credential detection and validation
func (iw *InitWizard) stepAWSConfiguration() error {
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("🔧 %s\n", color.GreenString("Step 1: AWS Configuration"))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Println()

	// Check for AWS credentials
	homeDir, _ := os.UserHomeDir()
	credentialsPath := fmt.Sprintf("%s/.aws/credentials", homeDir)

	if _, err := os.Stat(credentialsPath); err == nil {
		fmt.Printf("✅ AWS credentials detected: %s\n", credentialsPath)

		// Detect AWS profiles
		profiles := iw.detectAWSProfiles()
		if len(profiles) > 0 {
			fmt.Printf("✅ Found %d AWS profile(s): %s\n", len(profiles), strings.Join(profiles, ", "))
		}

		// Detect default region
		defaultRegion := iw.detectDefaultRegion()
		if defaultRegion != "" {
			fmt.Printf("✅ Default region: %s\n", defaultRegion)
			iw.config.AWS.Region = defaultRegion
		}

		// Validate credentials
		fmt.Println()
		fmt.Print("🔍 Validating AWS credentials... ")
		if iw.validateAWSCredentials() {
			fmt.Println(color.GreenString("✓"))
		} else {
			fmt.Println(color.YellowString("⚠"))
			fmt.Println()
			fmt.Println(color.YellowString("⚠️  AWS credential validation failed"))
			fmt.Println("   You can continue, but you may need to configure AWS later.")
			fmt.Println()
			fmt.Println("💡 To fix this:")
			fmt.Println("   1. Install AWS CLI: https://aws.amazon.com/cli/")
			fmt.Println("   2. Configure credentials: aws configure")
			fmt.Println("   3. Verify access: aws sts get-caller-identity")
			fmt.Println()

			if !iw.promptYesNo("Continue anyway?", true) {
				return fmt.Errorf("setup cancelled")
			}
		}
	} else {
		fmt.Println(color.YellowString("⚠️  AWS credentials not found"))
		fmt.Println()
		fmt.Println("CloudWorkstation requires AWS credentials to launch workspaces.")
		fmt.Println()
		fmt.Println("💡 Set up AWS credentials:")
		fmt.Println("   1. Install AWS CLI: https://aws.amazon.com/cli/")
		fmt.Println("   2. Run: aws configure")
		fmt.Println("   3. Enter your AWS access key ID and secret access key")
		fmt.Println()

		if !iw.promptYesNo("Continue without AWS credentials? (you'll need to set them up later)", false) {
			return fmt.Errorf("setup cancelled - AWS credentials required")
		}
	}

	fmt.Println()
	return nil
}

// stepResearchArea handles research area selection
func (iw *InitWizard) stepResearchArea() *ResearchArea {
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("🔬 %s\n", color.GreenString("Step 2: Research Area"))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Println()
	fmt.Println("Select your primary research area to get personalized recommendations:")
	fmt.Println()

	for i, area := range researchAreas {
		fmt.Printf("  %d. %s\n", i+1, color.CyanString(area.Name))
		fmt.Printf("     %s\n", color.New(color.FgHiBlack).Sprint(area.Description))
		fmt.Println()
	}

	for {
		choice := iw.promptString("Select research area (1-4) or 'q' to quit", "1")

		if strings.ToLower(choice) == "q" || strings.ToLower(choice) == "quit" {
			return nil
		}

		// Parse choice
		var selected int
		if _, err := fmt.Sscanf(choice, "%d", &selected); err == nil {
			if selected >= 1 && selected <= len(researchAreas) {
				area := &researchAreas[selected-1]
				fmt.Printf("\n✅ Selected: %s\n\n", color.GreenString(area.Name))
				return area
			}
		}

		fmt.Printf("❌ Invalid choice. Please enter 1-%d or 'q'.\n\n", len(researchAreas))
	}
}

// stepBudget handles optional budget setup
func (iw *InitWizard) stepBudget() float64 {
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("💰 %s\n", color.GreenString("Step 3: Budget (Optional)"))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Println()
	fmt.Println("Set a monthly budget to receive alerts when approaching the limit.")
	fmt.Println("CloudWorkstation will notify you but won't automatically stop workspaces.")
	fmt.Println()

	if !iw.promptYesNo("Would you like to set a monthly budget?", false) {
		fmt.Println("⏭️  Skipping budget setup (you can configure this later with 'cws budget')")
		fmt.Println()
		return 0
	}

	for {
		budgetStr := iw.promptString("Monthly budget in USD (e.g., 100)", "")
		if budgetStr == "" {
			return 0
		}

		var budget float64
		if _, err := fmt.Sscanf(budgetStr, "%f", &budget); err == nil {
			if budget > 0 {
				fmt.Printf("✅ Budget set to $%.2f/month\n", budget)
				fmt.Printf("💡 You'll receive alerts at 50%%, 80%%, and 100%% of your budget\n\n")
				return budget
			}
		}

		fmt.Println("❌ Please enter a valid budget amount (e.g., 100)")
	}
}

// stepHibernation handles hibernation policy recommendation
func (iw *InitWizard) stepHibernation(recommendedPolicy string) string {
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("💤 %s\n", color.GreenString("Step 4: Hibernation (Recommended)"))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Println()
	fmt.Println("Hibernation automatically pauses idle workspaces to save costs while")
	fmt.Println("preserving your work. When you reconnect, everything is exactly as you left it.")
	fmt.Println()

	// Show recommended policy details
	policyDescriptions := map[string]string{
		"gpu":      "15 minutes idle → stop (optimized for expensive GPU instances)",
		"batch":    "60 minutes idle → hibernate (for long-running batch jobs)",
		"balanced": "30 minutes idle → hibernate (balanced for most research workflows)",
	}

	description, exists := policyDescriptions[recommendedPolicy]
	if exists {
		fmt.Printf("📋 Recommended for your research area: %s\n", color.CyanString(recommendedPolicy))
		fmt.Printf("   %s\n", description)
		fmt.Println()
	}

	fmt.Println("Options:")
	fmt.Println("  1. Use recommended policy (recommended)")
	fmt.Println("  2. Customize idle detection settings")
	fmt.Println("  3. Skip (manage manually)")
	fmt.Println()

	for {
		choice := iw.promptString("Select hibernation option (1-3)", "1")

		switch choice {
		case "1", "recommended", "yes", "y":
			fmt.Printf("✅ Using '%s' hibernation policy\n", recommendedPolicy)
			fmt.Printf("💡 You can adjust this later with 'cws idle profile'\n\n")
			return recommendedPolicy
		case "2", "customize", "custom":
			fmt.Println("⚙️  Advanced hibernation configuration:")
			fmt.Println("   Use 'cws idle profile create' to create custom policies")
			fmt.Println("⏭️  Skipping for now (using default)\n")
			return "balanced"
		case "3", "skip", "no", "n":
			fmt.Println("⏭️  Skipping hibernation setup (you can configure this later)\n")
			return ""
		default:
			fmt.Println("❌ Please enter 1, 2, or 3")
		}
	}
}

// stepTemplateRecommendations shows recommended templates
func (iw *InitWizard) stepTemplateRecommendations(area *ResearchArea) {
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📦 %s\n", color.GreenString("Step 5: Recommended Templates"))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Println()
	fmt.Printf("Based on your research area (%s),\n", area.Name)
	fmt.Println("here are the recommended templates to get started:")
	fmt.Println()

	for i, template := range area.RecommendedTemplates {
		fmt.Printf("  %d. %s\n", i+1, color.CyanString(template))
	}

	fmt.Println()
	fmt.Println("💡 Templates are pre-configured environments with all necessary tools installed.")
	fmt.Println("   You can explore all templates with: cws templates list")
	fmt.Println()
}

// saveInitialization saves initialization status to config
func (iw *InitWizard) saveInitialization(area *ResearchArea, budget float64, policy string) error {
	// Create initialization status file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	initFile := fmt.Sprintf("%s/.cloudworkstation/.initialized", homeDir)

	// Create status content
	status := fmt.Sprintf("initialized: true\nresearch_area: %s\ndate: %s\n",
		area.Name,
		fmt.Sprintf("%d-%02d-%02d", time.Now().Year(), time.Now().Month(), time.Now().Day()))

	if budget > 0 {
		status += fmt.Sprintf("budget: %.2f\n", budget)
	}

	if policy != "" {
		status += fmt.Sprintf("hibernation_policy: %s\n", policy)
	}

	// Write status file
	return os.WriteFile(initFile, []byte(status), 0644)
}

// showCompletion displays the completion message
func (iw *InitWizard) showCompletion() {
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✅ %s\n", color.GreenString("Setup Complete!"))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Println()
	fmt.Println("You're all set! Here's how to launch your first workspace:")
	fmt.Println()
	fmt.Printf("  %s\n", color.CyanString("cws launch \"Python Machine Learning (Simplified)\" my-first-project"))
	fmt.Println()
	fmt.Println("Other helpful commands:")
	fmt.Printf("  • %s - List all available templates\n", color.New(color.FgHiBlack).Sprint("cws templates list"))
	fmt.Printf("  • %s - List your workspaces\n", color.New(color.FgHiBlack).Sprint("cws list"))
	fmt.Printf("  • %s - Connect to a workspace\n", color.New(color.FgHiBlack).Sprint("cws connect <name>"))
	fmt.Printf("  • %s - Launch interactive interface\n", color.New(color.FgHiBlack).Sprint("cws tui"))
	fmt.Println()
	fmt.Println("📚 Documentation: https://docs.cloudworkstation.io")
	fmt.Println()
}

// Helper methods

func (iw *InitWizard) promptString(prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	input, _ := iw.reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}

func (iw *InitWizard) promptYesNo(prompt string, defaultValue bool) bool {
	defaultChar := "y/N"
	if defaultValue {
		defaultChar = "Y/n"
	}

	fmt.Printf("%s [%s]: ", prompt, defaultChar)
	input, _ := iw.reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		return defaultValue
	}

	return input == "y" || input == "yes"
}

func (iw *InitWizard) detectAWSProfiles() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return []string{}
	}

	credentialsPath := fmt.Sprintf("%s/.aws/credentials", homeDir)
	file, err := os.Open(credentialsPath)
	if err != nil {
		return []string{}
	}
	defer file.Close()

	var profiles []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			profile := strings.Trim(line, "[]")
			if profile != "" {
				profiles = append(profiles, profile)
			}
		}
	}

	return profiles
}

func (iw *InitWizard) detectDefaultRegion() string {
	// Try to detect from AWS config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	configPath := fmt.Sprintf("%s/.aws/config", homeDir)
	file, err := os.Open(configPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	regionRegex := regexp.MustCompile(`^region\s*=\s*(.+)$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if matches := regionRegex.FindStringSubmatch(line); len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	// Fall back to environment variables
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region
	}

	return ""
}

func (iw *InitWizard) validateAWSCredentials() bool {
	cmd := exec.Command("aws", "sts", "get-caller-identity")

	// Set AWS region if detected
	if iw.config.AWS.Region != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("AWS_DEFAULT_REGION=%s", iw.config.AWS.Region))
	}

	err := cmd.Run()
	return err == nil
}

// IsInitialized checks if the user has completed initialization
func IsInitialized() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	initFile := fmt.Sprintf("%s/.cloudworkstation/.initialized", homeDir)
	_, err = os.Stat(initFile)
	return err == nil
}
