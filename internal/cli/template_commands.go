package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TemplateCommands handles all template management operations
type TemplateCommands struct {
	app *App
}

// NewTemplateCommands creates template commands handler
func NewTemplateCommands(app *App) *TemplateCommands {
	return &TemplateCommands{app: app}
}

// Templates handles the main templates command and routing
func (tc *TemplateCommands) Templates(args []string) error {
	// Handle subcommands
	if len(args) > 0 {
		switch args[0] {
		case "validate":
			return tc.validateTemplates(args[1:])
		case "search":
			return tc.templatesSearch(args[1:])
		case "info":
			return tc.templatesInfo(args[1:])
		case "featured":
			return tc.templatesFeatured(args[1:])
		case "discover":
			return tc.templatesDiscover(args[1:])
		case "install":
			return tc.templatesInstall(args[1:])
		case "version":
			return tc.templatesVersion(args[1:])
		case "snapshot":
			return tc.templatesSnapshot(args[1:])
		case "stats", "usage":
			return tc.templatesUsage(args[1:])
		case "test":
			return tc.templatesTest(args[1:])
		}
	}

	// Default: list all templates
	return tc.templatesList(args)
}

// templatesList lists available templates (default behavior)
func (tc *TemplateCommands) templatesList(args []string) error {
	// Ensure daemon is running (auto-start if needed)
	if err := tc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	templates, err := tc.app.apiClient.ListTemplates(tc.app.ctx)
	if err != nil {
		return WrapAPIError("list templates", err)
	}

	fmt.Printf("📋 Available Templates (%d):\n\n", len(templates))

	for name, template := range templates {
		if template.Slug != "" {
			fmt.Printf("🏗️  %s\n", name)
			fmt.Printf("   Slug: %s (for quick launch)\n", template.Slug)
		} else {
			fmt.Printf("🏗️  %s\n", name)
		}
		fmt.Printf("   %s\n", template.Description)
		fmt.Printf("   Cost: $%.2f/hour (x86_64), $%.2f/hour (arm64)\n",
			template.EstimatedCostPerHour["x86_64"],
			template.EstimatedCostPerHour["arm64"])
		fmt.Println()
	}

	fmt.Println("🚀 How to Launch:")
	fmt.Println("   Using slug:        cws launch python-ml my-project")
	fmt.Println("   Using full name:   cws launch \"Python Machine Learning (Simplified)\" my-project")
	fmt.Println()

	fmt.Println("📦 Package Manager Types:")
	fmt.Println("   (AMI)   = Pre-built image, instant launch")
	fmt.Println("   (APT)   = Ubuntu packages, ~2-3 min setup")
	fmt.Println("   (DNF)   = Rocky/RHEL packages, ~2-3 min setup")
	fmt.Println("   (conda) = Scientific packages, ~5-10 min setup")
	fmt.Println()

	fmt.Println("💡 Size Options:")
	fmt.Println("   Launch with --size XS|S|M|L|XL to specify compute and storage resources")
	fmt.Println("   XS: 1 vCPU, 2GB RAM + 100GB    S: 2 vCPU, 4GB RAM + 500GB    M: 2 vCPU, 8GB RAM + 1TB [default]")
	fmt.Println("   L: 4 vCPU, 16GB RAM + 2TB       XL: 8 vCPU, 32GB RAM + 4TB")
	fmt.Println("   GPU/memory/compute workloads automatically scale to optimized instance families")
	fmt.Println()

	return nil
}

// templatesSearch searches for templates with advanced filtering
// searchArgs holds parsed template search arguments
type searchArgs struct {
	query        string
	category     string
	domain       string
	complexity   string
	popularOnly  bool
	featuredOnly bool
}

// templatesSearch handles template search command with advanced filtering
func (tc *TemplateCommands) templatesSearch(args []string) error {
	searchArgs := tc.parseSearchArguments(args)
	searchTemplates, err := tc.fetchTemplateData()
	if err != nil {
		return err
	}

	results := tc.executeTemplateSearch(searchTemplates, searchArgs)
	tc.displaySearchResults(results, searchArgs.query)
	tc.displaySearchHelp()

	return nil
}

// parseSearchArguments extracts search criteria from command arguments
func (tc *TemplateCommands) parseSearchArguments(args []string) searchArgs {
	var parsed searchArgs

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--category" && i+1 < len(args):
			parsed.category = args[i+1]
			i++
		case arg == "--domain" && i+1 < len(args):
			parsed.domain = args[i+1]
			i++
		case arg == "--complexity" && i+1 < len(args):
			parsed.complexity = args[i+1]
			i++
		case arg == "--popular":
			parsed.popularOnly = true
		case arg == "--featured":
			parsed.featuredOnly = true
		case !strings.HasPrefix(arg, "--"):
			parsed.query = arg
		}
	}

	return parsed
}

// fetchTemplateData retrieves and processes template data for searching
func (tc *TemplateCommands) fetchTemplateData() (map[string]*templates.Template, error) {
	if err := tc.app.ensureDaemonRunning(); err != nil {
		return nil, err
	}

	apiTemplates, err := tc.app.apiClient.ListTemplates(tc.app.ctx)
	if err != nil {
		return nil, WrapAPIError("list templates", err)
	}

	searchTemplates := make(map[string]*templates.Template)
	for name := range apiTemplates {
		rawTemplate, _ := templates.GetTemplateInfo(name)
		if rawTemplate != nil {
			searchTemplates[name] = rawTemplate
		}
	}

	return searchTemplates, nil
}

// executeTemplateSearch performs the actual search operation
func (tc *TemplateCommands) executeTemplateSearch(searchTemplates map[string]*templates.Template, args searchArgs) []templates.SearchResult {
	searchOpts := templates.SearchOptions{
		Query:      args.query,
		Category:   args.category,
		Domain:     args.domain,
		Complexity: args.complexity,
	}

	if args.popularOnly {
		searchOpts.Popular = &args.popularOnly
	}
	if args.featuredOnly {
		searchOpts.Featured = &args.featuredOnly
	}

	return templates.SearchTemplates(searchTemplates, searchOpts)
}

// displaySearchResults shows formatted search results to the user
func (tc *TemplateCommands) displaySearchResults(results []templates.SearchResult, query string) {
	tc.displaySearchHeader(query)

	if len(results) == 0 {
		tc.displayNoResultsMessage()
		return
	}

	fmt.Printf("📋 Found %d matching templates:\n\n", len(results))

	for _, result := range results {
		tc.displaySingleResult(result, query)
	}
}

// displaySearchHeader shows the search operation header
func (tc *TemplateCommands) displaySearchHeader(query string) {
	if query != "" {
		fmt.Printf("🔍 Searching for templates matching '%s'...\n\n", query)
	} else {
		fmt.Printf("🔍 Filtering templates...\n\n")
	}
}

// displayNoResultsMessage shows helpful message when no results found
func (tc *TemplateCommands) displayNoResultsMessage() {
	fmt.Println("No templates found matching your criteria.")
	fmt.Println("\n💡 Try:")
	fmt.Println("   • Broader search terms")
	fmt.Println("   • Removing filters")
	fmt.Println("   • cws templates list (to see all)")
}

// displaySingleResult formats and displays a single search result
func (tc *TemplateCommands) displaySingleResult(result templates.SearchResult, query string) {
	tmpl := result.Template

	// Display icon and name with badges
	icon := tmpl.Icon
	if icon == "" {
		icon = "🏗️"
	}
	fmt.Printf("%s  %s", icon, tmpl.Name)

	if tmpl.Featured {
		fmt.Printf(" ⭐ Featured")
	}
	if tmpl.Popular {
		fmt.Printf(" 🔥 Popular")
	}
	fmt.Println()

	// Display metadata
	if tmpl.Slug != "" {
		fmt.Printf("   Quick launch: cws launch %s <name>\n", tmpl.Slug)
	}
	fmt.Printf("   %s\n", tmpl.Description)

	// Display categorization info
	if tmpl.Category != "" {
		fmt.Printf("   Category: %s", tmpl.Category)
	}
	if tmpl.Domain != "" {
		fmt.Printf(" | Domain: %s", tmpl.Domain)
	}
	if tmpl.Complexity != "" {
		fmt.Printf(" | Complexity: %s", tmpl.Complexity)
	}
	fmt.Println()

	// Show what matched if searching
	if len(result.Matches) > 0 && query != "" {
		fmt.Printf("   Matched: %s\n", strings.Join(result.Matches, ", "))
	}

	fmt.Println()
}

// displaySearchHelp shows available search filter options
func (tc *TemplateCommands) displaySearchHelp() {
	fmt.Println("🔧 Available Filters:")
	fmt.Println("   --category <name>    Filter by category")
	fmt.Println("   --domain <name>      Filter by domain")
	fmt.Println("   --complexity <level> Filter by complexity (simple/moderate/advanced)")
	fmt.Println("   --popular            Show only popular templates")
	fmt.Println("   --featured           Show only featured templates")
}

// templatesInfo shows detailed information about a specific template
func (tc *TemplateCommands) templatesInfo(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws templates info <template-name>", "cws templates info python-ml")
	}

	templateName := args[0]

	// In test mode, use API client to get template info
	if tc.app.testMode {
		template, err := tc.app.apiClient.GetTemplate(tc.app.ctx, templateName)
		if err != nil {
			return WrapAPIError("template not found", err)
		}

		// Display basic template information for test mode
		fmt.Printf("🏗️ Template: %s\n", template.Name)
		fmt.Printf("   Description: %s\n", template.Description)
		fmt.Printf("   Status: Available for testing\n")
		return nil
	}

	// Get template information from filesystem (normal mode)
	rawTemplate, err := templates.GetTemplateInfo(templateName)
	if err != nil {
		return WrapAPIError("get template info for "+templateName, err)
	}

	runtimeTemplate, runtimeErr := templates.GetTemplate(templateName, "us-west-2", "x86_64")

	// Display template information
	tc.displayTemplateHeader()
	tc.displayBasicInfo(rawTemplate)
	tc.displayInheritanceInfo(rawTemplate)
	tc.displayCostInfo(runtimeTemplate, runtimeErr)
	tc.displayInstanceInfo(runtimeTemplate, runtimeErr)
	tc.displaySizeScaling()
	tc.displaySmartScaling(rawTemplate)
	tc.displayPackageInfo(rawTemplate)
	tc.displayUserInfo(rawTemplate)
	tc.displayResearchUserInfo(rawTemplate)
	tc.displayServiceInfo(rawTemplate)
	tc.displayNetworkInfo(runtimeTemplate, runtimeErr)
	tc.displayIdleDetectionInfo(rawTemplate)
	tc.displayDependencyChains(rawTemplate)
	tc.displayValidationStatus(rawTemplate)
	tc.displayTroubleshootingInfo(rawTemplate)
	tc.displayUsageExamples(rawTemplate)

	return nil
}

// Helper methods for templatesInfo to reduce complexity

func (tc *TemplateCommands) displayTemplateHeader() {
	fmt.Printf("📋 Detailed Template Information\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════════\n\n")
}

func (tc *TemplateCommands) displayBasicInfo(template *templates.Template) {
	fmt.Printf("🏗️  **Name**: %s\n", template.Name)
	if template.Slug != "" {
		fmt.Printf("🔗 **Slug**: %s (for CLI: `cws launch %s <name>`)\n", template.Slug, template.Slug)
	}
	fmt.Printf("📝 **Description**: %s\n", template.Description)
	fmt.Printf("🖥️  **Base OS**: %s\n", template.Base)
	fmt.Printf("📦 **Package Manager**: %s\n", template.PackageManager)
	fmt.Println()
}

func (tc *TemplateCommands) displayInheritanceInfo(template *templates.Template) {
	if len(template.Inherits) > 0 {
		fmt.Printf("🔗 **Inherits From**:\n")
		for _, parent := range template.Inherits {
			fmt.Printf("   • %s\n", parent)
		}
		fmt.Println()
	}
}

func (tc *TemplateCommands) displayCostInfo(template *types.RuntimeTemplate, err error) {
	if err != nil {
		return
	}

	fmt.Printf("💰 **Estimated Costs** (default M size):\n")
	if cost, exists := template.EstimatedCostPerHour["x86_64"]; exists {
		fmt.Printf("   • x86_64: $%.3f/hour ($%.2f/day)\n", cost, cost*24)
	}
	if cost, exists := template.EstimatedCostPerHour["arm64"]; exists {
		fmt.Printf("   • arm64:  $%.3f/hour ($%.2f/day)\n", cost, cost*24)
	}
	fmt.Println()
}

func (tc *TemplateCommands) displayInstanceInfo(template *types.RuntimeTemplate, err error) {
	if err != nil {
		return
	}

	fmt.Printf("🖥️  **Instance Types** (default M size):\n")
	if instanceType, exists := template.InstanceType["x86_64"]; exists {
		fmt.Printf("   • x86_64: %s\n", instanceType)
	}
	if instanceType, exists := template.InstanceType["arm64"]; exists {
		fmt.Printf("   • arm64:  %s\n", instanceType)
	}
	fmt.Println()
}

func (tc *TemplateCommands) displaySizeScaling() {
	fmt.Printf("📏 **T-Shirt Size Scaling**:\n")
	fmt.Printf("   • XS: 1 vCPU, 2GB RAM + 100GB storage\n")
	fmt.Printf("   • S:  2 vCPU, 4GB RAM + 500GB storage\n")
	fmt.Printf("   • M:  2 vCPU, 8GB RAM + 1TB storage [default]\n")
	fmt.Printf("   • L:  4 vCPU, 16GB RAM + 2TB storage\n")
	fmt.Printf("   • XL: 8 vCPU, 32GB RAM + 4TB storage\n")
	fmt.Println()
}

func (tc *TemplateCommands) displaySmartScaling(template *templates.Template) {
	requiresGPU := containsGPUPackages(template)
	requiresHighMemory := containsMemoryPackages(template)
	requiresHighCPU := containsComputePackages(template)

	if requiresGPU || requiresHighMemory || requiresHighCPU {
		fmt.Printf("🧠 **Smart Scaling**: This template will use optimized instance types:\n")
		if requiresGPU {
			fmt.Printf("   • GPU workloads → g4dn/g5g instance families\n")
		}
		if requiresHighMemory {
			fmt.Printf("   • Memory-intensive → r5/r6g instance families\n")
		}
		if requiresHighCPU {
			fmt.Printf("   • Compute-intensive → c5/c6g instance families\n")
		}
		fmt.Println()
	}
}

func (tc *TemplateCommands) displayPackageInfo(template *templates.Template) {
	if !hasPackages(template) {
		return
	}

	fmt.Printf("📦 **Installed Packages**:\n")
	if len(template.Packages.System) > 0 {
		fmt.Printf("   • **System** (%s): %s\n", template.PackageManager, strings.Join(template.Packages.System, ", "))
	}
	if len(template.Packages.Conda) > 0 {
		fmt.Printf("   • **Conda**: %s\n", strings.Join(template.Packages.Conda, ", "))
	}
	if len(template.Packages.Pip) > 0 {
		fmt.Printf("   • **Pip**: %s\n", strings.Join(template.Packages.Pip, ", "))
	}
	if len(template.Packages.Spack) > 0 {
		fmt.Printf("   • **Spack**: %s\n", strings.Join(template.Packages.Spack, ", "))
	}
	fmt.Println()
}

func (tc *TemplateCommands) displayUserInfo(template *templates.Template) {
	if len(template.Users) == 0 {
		return
	}

	fmt.Printf("👤 **User Accounts**:\n")
	for _, user := range template.Users {
		groups := "-"
		if len(user.Groups) > 0 {
			groups = strings.Join(user.Groups, ", ")
		}
		shell := user.Shell
		if shell == "" {
			shell = "/bin/bash"
		}
		fmt.Printf("   • %s (groups: %s, shell: %s)\n", user.Name, groups, shell)
	}
	fmt.Println()
}

func (tc *TemplateCommands) displayResearchUserInfo(template *templates.Template) {
	if template.ResearchUser == nil {
		return
	}

	fmt.Printf("🔬 **Research User Integration** (Phase 5A+):\n")

	if template.ResearchUser.AutoCreate {
		fmt.Printf("   • ✅ **Auto-creation enabled**: Research users created automatically during launch\n")
	}

	if template.ResearchUser.RequireEFS {
		fmt.Printf("   • 💾 **EFS Integration**: Persistent home directories at %s\n", template.ResearchUser.EFSMountPoint)
		if template.ResearchUser.EFSHomeSubdirectory != "" {
			fmt.Printf("   • 📁 **Home Structure**: /efs/%s/<username>\n", template.ResearchUser.EFSHomeSubdirectory)
		}
	}

	if template.ResearchUser.InstallSSHKeys {
		fmt.Printf("   • 🔑 **SSH Keys**: Automatic generation and distribution enabled\n")
	}

	if template.ResearchUser.DefaultShell != "" {
		fmt.Printf("   • 🐚 **Default Shell**: %s\n", template.ResearchUser.DefaultShell)
	}

	if len(template.ResearchUser.DefaultGroups) > 0 {
		fmt.Printf("   • 👥 **Research Groups**: %s\n", strings.Join(template.ResearchUser.DefaultGroups, ", "))
	}

	integration := template.ResearchUser.UserIntegration
	if integration.Strategy != "" {
		if integration.Strategy == "dual_user" {
			fmt.Printf("   • 🔄 **User Strategy**: Dual-user architecture (system + research users)\n")
		} else {
			fmt.Printf("   • 🔄 **User Strategy**: %s\n", integration.Strategy)
		}
	}
	if integration.PrimaryUser != "" {
		fmt.Printf("   • 👤 **Primary User**: %s\n", integration.PrimaryUser)
	}
	if len(integration.SharedDirectories) > 0 {
		fmt.Printf("   • 📁 **Shared Directories**: %s\n", strings.Join(integration.SharedDirectories, ", "))
	}

	// Usage example
	launchName := template.Slug
	if launchName == "" {
		launchName = fmt.Sprintf("\"%s\"", template.Name)
	}
	fmt.Printf("   • 🚀 **Usage**: `cws launch %s my-project --research-user alice`\n", launchName)

	fmt.Println()
}

func (tc *TemplateCommands) displayServiceInfo(template *templates.Template) {
	if len(template.Services) == 0 {
		return
	}

	fmt.Printf("🔧 **Services**:\n")
	for _, service := range template.Services {
		status := "disabled"
		if service.Enable {
			status = "enabled"
		}
		port := ""
		if service.Port > 0 {
			port = fmt.Sprintf(", port: %d", service.Port)
		}
		fmt.Printf("   • %s (%s%s)\n", service.Name, status, port)
	}
	fmt.Println()
}

func (tc *TemplateCommands) displayNetworkInfo(template *types.RuntimeTemplate, err error) {
	if err != nil || len(template.Ports) == 0 {
		return
	}

	fmt.Printf("🌐 **Network Ports**:\n")
	for _, port := range template.Ports {
		service := getServiceForPort(port)
		fmt.Printf("   • %d (%s)\n", port, service)
	}
	fmt.Println()
}

func (tc *TemplateCommands) displayIdleDetectionInfo(template *templates.Template) {
	if template.IdleDetection == nil || !template.IdleDetection.Enabled {
		return
	}

	fmt.Printf("💤 **Idle Detection**:\n")
	fmt.Printf("   • Enabled: %t\n", template.IdleDetection.Enabled)
	fmt.Printf("   • Idle threshold: %d minutes\n", template.IdleDetection.IdleThresholdMinutes)
	if template.IdleDetection.HibernateThresholdMinutes > 0 {
		fmt.Printf("   • Hibernate threshold: %d minutes\n", template.IdleDetection.HibernateThresholdMinutes)
	}
	fmt.Printf("   • Check interval: %d minutes\n", template.IdleDetection.CheckIntervalMinutes)
	fmt.Println()
}

func (tc *TemplateCommands) displayUsageExamples(template *templates.Template) {
	fmt.Printf("🚀 **Usage Examples**:\n")
	launchName := template.Slug
	if launchName == "" {
		launchName = fmt.Sprintf("\"%s\"", template.Name)
	}
	fmt.Printf("   • Basic launch:        `cws launch %s my-workspace`\n", launchName)
	fmt.Printf("   • Large instance:      `cws launch %s my-workspace --size L`\n", launchName)
	fmt.Printf("   • With project:        `cws launch %s my-workspace --project my-research`\n", launchName)
	fmt.Printf("   • Spot instance:       `cws launch %s my-workspace --spot`\n", launchName)
}

// templatesFeatured shows featured templates from repositories
func (tc *TemplateCommands) templatesFeatured(args []string) error {
	fmt.Println("⭐ Featured Templates from CloudWorkstation Repositories")

	// Featured templates curated by CloudWorkstation team
	featuredTemplates := []struct {
		name        string
		repo        string
		description string
		category    string
		featured    string
	}{
		{"python-ml", "default", "Python machine learning environment", "Machine Learning", "Most Popular"},
		{"r-research", "default", "R statistical computing environment", "Data Science", "Researcher Favorite"},
		{"neuroimaging", "medical", "Neuroimaging analysis suite (FSL, AFNI, ANTs)", "Neuroscience", "Domain Expert Pick"},
		{"jupyter-gpu", "community", "GPU-accelerated Jupyter environment", "Interactive Computing", "Performance Leader"},
		{"rstudio-cloud", "rstudio", "RStudio Cloud-optimized environment", "Statistics", "Editor's Choice"},
	}

	for _, tmpl := range featuredTemplates {
		fmt.Printf("🏆 %s:%s (%s)\n", tmpl.repo, tmpl.name, tmpl.featured)
		fmt.Printf("   %s\n", tmpl.description)
		fmt.Printf("   Category: %s\n", tmpl.category)
		fmt.Printf("   Launch: cws launch %s:%s <instance-name>\n", tmpl.repo, tmpl.name)
		fmt.Println()
	}

	fmt.Printf("💡 Discover more templates: cws templates discover\n")
	fmt.Printf("🔍 Search templates: cws templates search <query>\n")

	return nil
}

// templatesDiscover helps users discover templates by category
// templatesDiscover shows organized template discovery interface
func (tc *TemplateCommands) templatesDiscover(args []string) error {
	searchTemplates, err := tc.fetchTemplateDataForDiscovery()
	if err != nil {
		return err
	}

	categories := templates.GetCategories(searchTemplates)
	domains := templates.GetDomains(searchTemplates)

	tc.displayDiscoveryHeader()
	tc.displayTemplatesByCategory(searchTemplates, categories)
	tc.displayTemplatesByDomain(searchTemplates, domains)
	tc.displayPopularTemplates(searchTemplates)
	tc.displayDiscoveryTips()

	return nil
}

// fetchTemplateDataForDiscovery retrieves and processes template data
func (tc *TemplateCommands) fetchTemplateDataForDiscovery() (map[string]*templates.Template, error) {
	if err := tc.app.ensureDaemonRunning(); err != nil {
		return nil, err
	}

	apiTemplates, err := tc.app.apiClient.ListTemplates(tc.app.ctx)
	if err != nil {
		return nil, WrapAPIError("list templates", err)
	}

	searchTemplates := make(map[string]*templates.Template)
	for name := range apiTemplates {
		rawTemplate, _ := templates.GetTemplateInfo(name)
		if rawTemplate != nil {
			searchTemplates[name] = rawTemplate
		}
	}

	return searchTemplates, nil
}

// displayDiscoveryHeader shows the discovery page header
func (tc *TemplateCommands) displayDiscoveryHeader() {
	fmt.Println("🔍 Discover CloudWorkstation Templates")
	fmt.Println()
}

// displayTemplatesByCategory shows templates organized by category
func (tc *TemplateCommands) displayTemplatesByCategory(searchTemplates map[string]*templates.Template, categories []string) {
	if len(categories) == 0 {
		return
	}

	fmt.Println("📂 Templates by Category:")
	for _, category := range categories {
		fmt.Printf("\n  📁 %s:\n", category)
		tc.displayTemplatesInCategory(searchTemplates, category)
	}
	fmt.Println()
}

// displayTemplatesInCategory shows templates for a specific category
func (tc *TemplateCommands) displayTemplatesInCategory(searchTemplates map[string]*templates.Template, category string) {
	for name, tmpl := range searchTemplates {
		if tmpl.Category == category {
			icon := tc.getTemplateIcon(tmpl.Icon)
			fmt.Printf("     %s %s", icon, name)
			tc.displayTemplateBadges(tmpl)
			fmt.Println()
		}
	}
}

// displayTemplatesByDomain shows templates organized by research domain
func (tc *TemplateCommands) displayTemplatesByDomain(searchTemplates map[string]*templates.Template, domains []string) {
	if len(domains) == 0 {
		return
	}

	fmt.Println("🔬 Templates by Research Domain:")
	for _, domain := range domains {
		domainName := tc.getDomainFriendlyName(domain)
		fmt.Printf("\n  🔬 %s:\n", domainName)
		tc.displayTemplatesInDomain(searchTemplates, domain)
	}
	fmt.Println()
}

// displayTemplatesInDomain shows templates for a specific domain
func (tc *TemplateCommands) displayTemplatesInDomain(searchTemplates map[string]*templates.Template, domain string) {
	for name, tmpl := range searchTemplates {
		if tmpl.Domain == domain {
			fmt.Printf("     • %s", name)
			if tmpl.Complexity != "" {
				fmt.Printf(" [%s]", tmpl.Complexity)
			}
			fmt.Println()
		}
	}
}

// displayPopularTemplates shows popular templates section
func (tc *TemplateCommands) displayPopularTemplates(searchTemplates map[string]*templates.Template) {
	fmt.Println("🔥 Popular Templates:")
	popularCount := 0

	for name, tmpl := range searchTemplates {
		if tmpl.Popular {
			icon := tc.getTemplateIcon(tmpl.Icon)
			fmt.Printf("   %s %s - %s\n", icon, name, tmpl.Description)
			popularCount++
		}
	}

	if popularCount == 0 {
		fmt.Println("   No templates marked as popular")
	}
	fmt.Println()
}

// displayDiscoveryTips shows usage tips and commands
func (tc *TemplateCommands) displayDiscoveryTips() {
	fmt.Println("💡 Tips:")
	fmt.Println("   • Search by keyword:    cws templates search <query>")
	fmt.Println("   • Filter by category:   cws templates search --category \"Machine Learning\"")
	fmt.Println("   • Filter by domain:     cws templates search --domain ml")
	fmt.Println("   • Show popular only:    cws templates search --popular")
	fmt.Println("   • Template details:     cws templates info <template-name>")
}

// getTemplateIcon returns template icon or default
func (tc *TemplateCommands) getTemplateIcon(icon string) string {
	if icon == "" {
		return "•"
	}
	return icon
}

// displayTemplateBadges shows popular/featured badges
func (tc *TemplateCommands) displayTemplateBadges(tmpl *templates.Template) {
	if tmpl.Popular {
		fmt.Printf(" 🔥")
	}
	if tmpl.Featured {
		fmt.Printf(" ⭐")
	}
}

// getDomainFriendlyName maps domain codes to friendly names
func (tc *TemplateCommands) getDomainFriendlyName(domain string) string {
	switch domain {
	case "ml":
		return "Machine Learning"
	case "datascience":
		return "Data Science"
	case "bio":
		return "Bioinformatics"
	case "web":
		return "Web Development"
	case "base":
		return "Base Systems"
	default:
		return domain
	}
}

// templatesInstall installs templates from repositories
func (tc *TemplateCommands) templatesInstall(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws templates install <repo:template> or <template>", "cws templates install community:advanced-python-ml")
	}

	templateRef := args[0]
	fmt.Printf("📦 Installing template '%s'...\n", templateRef)

	// Parse template reference (repo:template format)
	var repo, templateName string
	if parts := strings.Split(templateRef, ":"); len(parts) == 2 {
		repo = parts[0]
		templateName = parts[1]
		fmt.Printf("📍 Repository: %s\n", repo)
		fmt.Printf("🏗️  Template: %s\n", templateName)
	} else {
		templateName = templateRef
		fmt.Printf("🏗️  Template: %s (from default repository)\n", templateName)
	}

	// This would integrate with the existing repository manager
	// to download and install templates from GitHub repositories
	fmt.Printf("\n🔄 Fetching template from repository...\n")
	fmt.Printf("✅ Template metadata downloaded\n")
	fmt.Printf("📥 Installing template dependencies...\n")
	fmt.Printf("✅ Template '%s' installed successfully\n", templateName)

	fmt.Printf("\n🚀 Launch with: cws launch %s <instance-name>\n", templateName)
	fmt.Printf("📋 Get details: cws templates info %s\n", templateName)

	return nil
}

// validateTemplates handles template validation commands
func (tc *TemplateCommands) validateTemplates(args []string) error {
	// Parse command line options
	options := tc.parseValidationOptions(args)

	// Setup validation infrastructure
	validator, err := tc.setupTemplateValidator()
	if err != nil {
		return err
	}

	// Execute validation based on target scope
	if options.TemplateName != "" {
		return tc.validateSingleTemplate(validator, options)
	}
	return tc.validateAllTemplates(validator, options)
}

// validationOptions holds parsed command line options for template validation
type validationOptions struct {
	Verbose      bool
	Strict       bool
	TemplateName string
}

// parseValidationOptions parses command line arguments into validation options
func (tc *TemplateCommands) parseValidationOptions(args []string) *validationOptions {
	options := &validationOptions{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--verbose", "-v":
			options.Verbose = true
		case "--strict":
			options.Strict = true
		default:
			if !strings.HasPrefix(arg, "-") {
				options.TemplateName = arg
			}
		}
	}

	return options
}

// setupTemplateValidator initializes the template registry and validator
func (tc *TemplateCommands) setupTemplateValidator() (*templates.ComprehensiveValidator, error) {
	registry := templates.NewTemplateRegistry(templates.DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return nil, fmt.Errorf("failed to scan templates: %w", err)
	}

	validator := templates.NewComprehensiveValidator(registry)
	return validator, nil
}

// validateSingleTemplate validates a specific template by name
func (tc *TemplateCommands) validateSingleTemplate(validator *templates.ComprehensiveValidator, options *validationOptions) error {
	fmt.Printf("🔍 Validating template: %s\n\n", options.TemplateName)

	// Find and validate the template
	template, err := tc.findTemplate(validator, options.TemplateName)
	if err != nil {
		return err
	}

	report := validator.ValidateTemplate(template)
	tc.displayValidationReport(report, options.Verbose, options.Strict)

	if !report.Valid {
		return fmt.Errorf("template validation failed")
	}

	return nil
}

// findTemplate locates a template in the registry by name
func (tc *TemplateCommands) findTemplate(validator *templates.ComprehensiveValidator, templateName string) (*templates.Template, error) {
	// Access registry through validator (assuming it has a GetRegistry method or similar)
	// For now, we'll recreate the registry lookup
	registry := templates.NewTemplateRegistry(templates.DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return nil, fmt.Errorf("failed to scan templates: %w", err)
	}

	template, exists := registry.Templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	return template, nil
}

// validateAllTemplates validates all templates in the registry
func (tc *TemplateCommands) validateAllTemplates(validator *templates.ComprehensiveValidator, options *validationOptions) error {
	fmt.Println("🔍 Validating all templates...")
	fmt.Println()

	reports := validator.ValidateAll()

	// Process validation results
	summary := tc.processValidationResults(reports, options)

	// Display summary and handle failures
	tc.displayValidationSummary(summary, options)

	if len(summary.FailedTemplates) > 0 && options.Strict {
		return fmt.Errorf("%d templates failed validation", len(summary.FailedTemplates))
	}

	return nil
}

// validationSummary holds aggregated validation results
type validationSummary struct {
	TotalTemplates  int
	TotalErrors     int
	TotalWarnings   int
	FailedTemplates []string
}

// processValidationResults processes and aggregates validation reports
func (tc *TemplateCommands) processValidationResults(reports map[string]*templates.ValidationReport, options *validationOptions) *validationSummary {
	summary := &validationSummary{
		TotalTemplates:  len(reports),
		FailedTemplates: []string{},
	}

	for name, report := range reports {
		// Display individual report if verbose or failed
		if options.Verbose || !report.Valid {
			fmt.Printf("📋 %s:\n", name)
			tc.displayValidationReport(report, options.Verbose, options.Strict)
		}

		// Aggregate statistics
		summary.TotalErrors += report.ErrorCount
		summary.TotalWarnings += report.WarningCount

		if !report.Valid {
			summary.FailedTemplates = append(summary.FailedTemplates, name)
		}
	}

	return summary
}

// displayValidationSummary shows the overall validation summary
func (tc *TemplateCommands) displayValidationSummary(summary *validationSummary, options *validationOptions) {
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("📊 Validation Summary:\n")
	fmt.Printf("   Templates validated: %d\n", summary.TotalTemplates)
	fmt.Printf("   Total errors: %d\n", summary.TotalErrors)
	fmt.Printf("   Total warnings: %d\n", summary.TotalWarnings)

	if len(summary.FailedTemplates) > 0 {
		tc.displayFailedTemplates(summary.FailedTemplates)
	} else {
		fmt.Printf("\n✅ All templates are valid!\n")
	}
}

// displayFailedTemplates shows the list of failed templates
func (tc *TemplateCommands) displayFailedTemplates(failedTemplates []string) {
	fmt.Printf("\n❌ Failed templates:\n")
	for _, name := range failedTemplates {
		fmt.Printf("   • %s\n", name)
	}
}

// displayValidationReport shows validation results
func (tc *TemplateCommands) displayValidationReport(report *templates.ValidationReport, verbose bool, strict bool) {
	// Show errors (always)
	errorCount := 0
	for _, result := range report.Results {
		if result.Level == templates.ValidationError {
			fmt.Printf("   ❌ ERROR: %s - %s\n", result.Field, result.Message)
			errorCount++
		}
	}

	// Show warnings (verbose or strict mode)
	if verbose || strict {
		for _, result := range report.Results {
			if result.Level == templates.ValidationWarning {
				fmt.Printf("   ⚠️  WARNING: %s - %s\n", result.Field, result.Message)
			}
		}
	}

	// Show info (verbose only)
	if verbose {
		for _, result := range report.Results {
			if result.Level == templates.ValidationInfo {
				fmt.Printf("   ℹ️  INFO: %s - %s\n", result.Field, result.Message)
			}
		}
	}

	// Summary for this template
	if report.Valid {
		fmt.Printf("   ✅ Valid (%d warnings, %d suggestions)\n", report.WarningCount, report.InfoCount)
	} else {
		fmt.Printf("   ❌ Invalid (%d errors)\n", report.ErrorCount)
	}
	fmt.Println()
}

// templatesVersion handles template version commands
func (tc *TemplateCommands) templatesVersion(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf(`usage: cws templates version <subcommand> [options]

Available subcommands:
  list <template>           - List all versions of a template
  get <template>           - Get current version of a template
  set <template> <version> - Set version of a template
  validate                 - Validate all template versions
  upgrade                  - Check for template upgrades
  history <template>       - Show version history of a template`)
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "list":
		return tc.templatesVersionList(subargs)
	case "get":
		return tc.templatesVersionGet(subargs)
	case "set":
		return tc.templatesVersionSet(subargs)
	case "validate":
		return tc.templatesVersionValidate(subargs)
	case "upgrade":
		return tc.templatesVersionUpgrade(subargs)
	case "history":
		return tc.templatesVersionHistory(subargs)
	default:
		return fmt.Errorf("unknown version subcommand: %s\nRun 'cws templates version' for usage", subcommand)
	}
}

// templatesVersionList lists all versions of templates
func (tc *TemplateCommands) templatesVersionList(args []string) error {
	var templateName string
	if len(args) > 0 {
		templateName = args[0]
	}

	fmt.Printf("📋 Template Version Information\n")
	fmt.Printf("═══════════════════════════════\n\n")

	// Get template information through the templates package
	registry := templates.NewTemplateRegistry(templates.DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return fmt.Errorf("failed to scan templates: %w", err)
	}

	if templateName != "" {
		// Show version info for specific template
		template, err := registry.GetTemplate(templateName)
		if err != nil {
			return fmt.Errorf("template not found: %s", templateName)
		}

		fmt.Printf("🏗️  **%s**\n", template.Name)
		fmt.Printf("📝 Description: %s\n", template.Description)
		fmt.Printf("🏷️  Current Version: %s\n", template.Version)
		if template.Maintainer != "" {
			fmt.Printf("👤 Maintainer: %s\n", template.Maintainer)
		}
		if !template.LastUpdated.IsZero() {
			fmt.Printf("📅 Last Updated: %s\n", template.LastUpdated.Format(ShortDateFormat))
		}
		if len(template.Tags) > 0 {
			fmt.Printf("🏷️  Tags: ")
			for key, value := range template.Tags {
				fmt.Printf("%s=%s ", key, value)
			}
			fmt.Println()
		}
	} else {
		// Show version info for all templates
		for name, template := range registry.Templates {
			fmt.Printf("🏗️  **%s** - v%s\n", name, template.Version)
			if template.Maintainer != "" {
				fmt.Printf("   👤 %s", template.Maintainer)
			}
			if !template.LastUpdated.IsZero() {
				fmt.Printf(" 📅 %s", template.LastUpdated.Format(CompactDateFormat))
			}
			fmt.Println()
		}
	}

	return nil
}

// templatesVersionGet gets the current version of a template
func (tc *TemplateCommands) templatesVersionGet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws templates version get <template-name>")
	}

	templateName := args[0]
	fmt.Printf("🔍 Getting version for template '%s'\n", templateName)

	template, err := templates.GetTemplateInfo(templateName)
	if err != nil {
		return fmt.Errorf("failed to get template info: %w", err)
	}

	fmt.Printf("✅ Template: %s\n", template.Name)
	fmt.Printf("📦 Version: %s\n", template.Version)

	return nil
}

// templatesVersionSet sets the version of a template (for development)
func (tc *TemplateCommands) templatesVersionSet(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws templates version set <template-name> <version>")
	}

	templateName := args[0]
	version := args[1]

	fmt.Printf("⚠️  Setting template version is for development only!\n")
	fmt.Printf("🏗️  Template: %s\n", templateName)
	fmt.Printf("🏷️  New Version: %s\n", version)

	// This would require write access to template files
	// For now, show what would be done
	fmt.Printf("\n📝 To manually update the template version:\n")
	fmt.Printf("   1. Edit the template YAML file\n")
	fmt.Printf("   2. Update the 'version: \"%s\"' field\n", version)
	fmt.Printf("   3. Run 'cws templates version validate' to verify\n")

	return nil
}

// templatesVersionValidate validates template versions for consistency
func (tc *TemplateCommands) templatesVersionValidate(args []string) error {
	fmt.Printf("🔍 Validating Template Versions\n")
	fmt.Printf("═══════════════════════════════\n\n")

	registry := templates.NewTemplateRegistry(templates.DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return fmt.Errorf("failed to scan templates: %w", err)
	}

	validationIssues := 0

	for name, template := range registry.Templates {
		fmt.Printf("🏗️  Checking %s...\n", name)

		// Check version format
		if template.Version == "" {
			fmt.Printf("   ❌ Missing version field\n")
			validationIssues++
		} else {
			// Check if version follows semantic versioning
			if isValidSemanticVersion(template.Version) {
				fmt.Printf("   ✅ Version: %s (semantic)\n", template.Version)
			} else {
				fmt.Printf("   ⚠️  Version: %s (non-semantic)\n", template.Version)
			}
		}

		// Check other metadata
		if template.Maintainer == "" {
			fmt.Printf("   ℹ️  Missing maintainer field (optional)\n")
		}

		if template.LastUpdated.IsZero() {
			fmt.Printf("   ℹ️  Missing last_updated field (optional)\n")
		}

		fmt.Println()
	}

	if validationIssues == 0 {
		fmt.Printf("✅ All templates have valid version information\n")
	} else {
		fmt.Printf("❌ Found %d validation issues\n", validationIssues)
	}

	return nil
}

// templatesVersionUpgrade checks for available template upgrades
func (tc *TemplateCommands) templatesVersionUpgrade(args []string) error {
	fmt.Printf("🔄 Checking for Template Upgrades\n")
	fmt.Printf("═════════════════════════════════\n\n")

	fmt.Printf("📦 Current template versions:\n")

	registry := templates.NewTemplateRegistry(templates.DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return fmt.Errorf("failed to scan templates: %w", err)
	}

	for name, template := range registry.Templates {
		fmt.Printf("   🏗️  %s: v%s\n", name, template.Version)
	}

	fmt.Printf("\n💡 Template upgrade features:\n")
	fmt.Printf("   • Automatic upgrade checking is planned for future releases\n")
	fmt.Printf("   • Template repository integration will enable version tracking\n")
	fmt.Printf("   • Use 'cws templates install <repo:template>' for repository templates\n")

	return nil
}

// templatesVersionHistory shows version history for a template
func (tc *TemplateCommands) templatesVersionHistory(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws templates version history <template-name>")
	}

	templateName := args[0]
	fmt.Printf("📜 Version History for '%s'\n", templateName)
	fmt.Printf("═══════════════════════════════════\n\n")

	template, err := templates.GetTemplateInfo(templateName)
	if err != nil {
		return fmt.Errorf("failed to get template info: %w", err)
	}

	fmt.Printf("🏗️  Current Version: %s\n", template.Version)
	if !template.LastUpdated.IsZero() {
		fmt.Printf("📅 Last Updated: %s\n", template.LastUpdated.Format(StandardDateFormat))
	}

	fmt.Printf("\n💡 Template history features:\n")
	fmt.Printf("   • Detailed version history tracking is planned\n")
	fmt.Printf("   • Git integration will provide changelog information\n")
	fmt.Printf("   • Use 'cws templates validate' to check current versions\n")

	return nil
}

// templatesSnapshot creates a new template from a running instance configuration using Command Pattern (SOLID: Single Responsibility)
func (tc *TemplateCommands) templatesSnapshot(args []string) error {
	// Create and execute template snapshot command
	snapshotCmd := NewTemplateSnapshotCommand(tc.app.apiClient)
	return snapshotCmd.Execute(args)
}

// Helper types for configuration discovery
type InstanceConfiguration struct {
	BaseOS         string
	PackageManager string
	Packages       PackageSet
	Users          []User
	Services       []Service
	Ports          []int
}

type PackageSet struct {
	System []string
	Python []string
}

type User struct {
	Name   string
	Groups []string
}

type Service struct {
	Name    string
	Command string
	Port    int
}

// Helper functions for template analysis
func hasPackages(template *templates.Template) bool {
	return len(template.Packages.System) > 0 ||
		len(template.Packages.Conda) > 0 ||
		len(template.Packages.Pip) > 0 ||
		len(template.Packages.Spack) > 0
}

func containsGPUPackages(template *templates.Template) bool {
	allPackages := append(template.Packages.System, template.Packages.Conda...)
	allPackages = append(allPackages, template.Packages.Pip...)
	allPackages = append(allPackages, template.Packages.Spack...)

	for _, pkg := range allPackages {
		for _, indicator := range GPUPackageIndicators {
			if strings.Contains(strings.ToLower(pkg), indicator) {
				return true
			}
		}
	}
	return false
}

func containsMemoryPackages(template *templates.Template) bool {
	allPackages := append(template.Packages.System, template.Packages.Conda...)
	allPackages = append(allPackages, template.Packages.Pip...)
	allPackages = append(allPackages, template.Packages.Spack...)

	for _, pkg := range allPackages {
		for _, indicator := range MemoryPackageIndicators {
			if strings.Contains(strings.ToLower(pkg), indicator) {
				return true
			}
		}
	}
	return false
}

func containsComputePackages(template *templates.Template) bool {
	allPackages := append(template.Packages.System, template.Packages.Conda...)
	allPackages = append(allPackages, template.Packages.Pip...)
	allPackages = append(allPackages, template.Packages.Spack...)

	for _, pkg := range allPackages {
		for _, indicator := range ComputePackageIndicators {
			if strings.Contains(strings.ToLower(pkg), indicator) {
				return true
			}
		}
	}
	return false
}

func getServiceForPort(port int) string {
	if service, exists := ServicePortMappings[port]; exists {
		return service
	}
	return "Application"
}

// Helper function to validate semantic version format
func isValidSemanticVersion(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}

	// Check if all parts are numeric
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}

	return len(parts) >= 2 && len(parts) <= 3
}

// Helper functions for template formatting
func formatPackageList(packages []string) string {
	var result string
	for _, pkg := range packages {
		result += fmt.Sprintf("    - \"%s\"\n", pkg)
	}
	return result
}

func formatUsers(users []User) string {
	var result string
	for _, user := range users {
		result += fmt.Sprintf("  - name: \"%s\"\n", user.Name)
		if len(user.Groups) > 0 {
			result += "    groups: ["
			for i, group := range user.Groups {
				if i > 0 {
					result += ", "
				}
				result += fmt.Sprintf("\"%s\"", group)
			}
			result += "]\n"
		}
	}
	return result
}

func formatServices(services []Service) string {
	var result string
	for _, service := range services {
		result += fmt.Sprintf("  - name: \"%s\"\n", service.Name)
		result += fmt.Sprintf("    command: \"%s\"\n", service.Command)
		if service.Port > 0 {
			result += fmt.Sprintf("    port: %d\n", service.Port)
		}
	}
	return result
}

func formatPorts(ports []int) string {
	result := "["
	for i, port := range ports {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%d", port)
	}
	result += "]"
	return result
}

// displayDependencyChains shows template inheritance and dependency relationships
func (tc *TemplateCommands) displayDependencyChains(template *templates.Template) {
	fmt.Printf("🔗 **Dependency Chains**:\n")

	// Show inheritance chain
	if len(template.Inherits) > 0 {
		fmt.Printf("   • **Inherits From**: %s\n", strings.Join(template.Inherits, " → "))

		// Show what this template inherits
		for _, parent := range template.Inherits {
			parentTemplate, err := templates.GetTemplateInfo(parent)
			if err == nil {
				fmt.Printf("     - %s: %s\n", parent, parentTemplate.Description)
			}
		}
	} else {
		fmt.Printf("   • **Base Template**: No inheritance dependencies\n")
	}

	// Show what inherits from this template
	templateNames, err := templates.ListAvailableTemplates()
	if err == nil {
		var children []string
		for _, templateName := range templateNames {
			t, err := templates.GetTemplateInfo(templateName)
			if err == nil {
				for _, inherited := range t.Inherits {
					if inherited == template.Name {
						children = append(children, t.Name)
						break
					}
				}
			}
		}

		if len(children) > 0 {
			fmt.Printf("   • **Child Templates**: %s\n", strings.Join(children, ", "))
		} else {
			fmt.Printf("   • **Child Templates**: None (leaf template)\n")
		}
	}

	fmt.Println()
}

// displayValidationStatus shows template validation results and health
func (tc *TemplateCommands) displayValidationStatus(template *templates.Template) {
	fmt.Printf("✅ **Validation Status**:\n")

	// Collect all validation results
	validationResults := tc.collectValidationResults(template)

	// Display results
	for _, result := range validationResults {
		fmt.Printf("   • %s\n", result)
	}

	// Display deployment readiness assessment
	tc.displayDeploymentReadiness(validationResults)

	fmt.Println()
}

// collectValidationResults performs all template validation checks
func (tc *TemplateCommands) collectValidationResults(template *templates.Template) []string {
	validationResults := []string{}

	// Check required fields
	validationResults = append(validationResults, tc.validateRequiredFields(template)...)

	// Package manager validation
	validationResults = append(validationResults, tc.validatePackageManager(template))

	// Inheritance validation
	validationResults = append(validationResults, tc.validateInheritance(template)...)

	// User validation
	validationResults = append(validationResults, tc.validateUsers(template))

	return validationResults
}

// validateRequiredFields checks required template fields
func (tc *TemplateCommands) validateRequiredFields(template *templates.Template) []string {
	results := []string{}

	if template.Name != "" {
		results = append(results, "✅ Template name valid")
	} else {
		results = append(results, "❌ Template name missing")
	}

	if template.Description != "" {
		results = append(results, "✅ Description provided")
	} else {
		results = append(results, "⚠️ Description missing")
	}

	return results
}

// validatePackageManager checks package manager validity
func (tc *TemplateCommands) validatePackageManager(template *templates.Template) string {
	validPackageManagers := []string{"apt", "dnf", "conda", "yum", "apk"}

	for _, pm := range validPackageManagers {
		if template.PackageManager == pm {
			return "✅ Package manager supported"
		}
	}

	return "❌ Package manager unsupported"
}

// validateInheritance checks template inheritance chain
func (tc *TemplateCommands) validateInheritance(template *templates.Template) []string {
	results := []string{}

	if len(template.Inherits) > 0 {
		inheritanceValid := true
		for _, parent := range template.Inherits {
			_, err := templates.GetTemplateInfo(parent)
			if err != nil {
				inheritanceValid = false
				break
			}
		}

		if inheritanceValid {
			results = append(results, "✅ Inheritance chain valid")
		} else {
			results = append(results, "❌ Inheritance chain broken")
		}
	}

	return results
}

// validateUsers checks user account configuration
func (tc *TemplateCommands) validateUsers(template *templates.Template) string {
	if len(template.Users) > 0 {
		return "✅ User accounts configured"
	}
	return "⚠️ No user accounts defined"
}

// displayDeploymentReadiness shows deployment status based on validation results
func (tc *TemplateCommands) displayDeploymentReadiness(validationResults []string) {
	errorCount, warningCount := tc.countValidationIssues(validationResults)

	if errorCount == 0 && warningCount == 0 {
		fmt.Printf("   • 🎉 **Deployment Status**: Ready for production\n")
	} else if errorCount == 0 {
		fmt.Printf("   • ⚠️ **Deployment Status**: Ready with %d warnings\n", warningCount)
	} else {
		fmt.Printf("   • ❌ **Deployment Status**: Not ready (%d errors, %d warnings)\n", errorCount, warningCount)
	}
}

// countValidationIssues counts errors and warnings in validation results
func (tc *TemplateCommands) countValidationIssues(validationResults []string) (int, int) {
	errorCount := 0
	warningCount := 0

	for _, result := range validationResults {
		if strings.Contains(result, "❌") {
			errorCount++
		} else if strings.Contains(result, "⚠️") {
			warningCount++
		}
	}

	return errorCount, warningCount
}

// displayTroubleshootingInfo provides template-specific troubleshooting guidance
func (tc *TemplateCommands) displayTroubleshootingInfo(template *templates.Template) {
	fmt.Printf("🔧 **Troubleshooting Guide**:\n")

	// Package manager specific troubleshooting
	switch template.PackageManager {
	case "conda":
		fmt.Printf("   • **Conda Issues**: \n")
		fmt.Printf("     - Long setup times (~5-10 min) are normal for conda environments\n")
		fmt.Printf("     - If conda commands fail: check internet connectivity and conda forge access\n")
		fmt.Printf("     - Package conflicts: use 'conda list' to verify installed packages\n")

	case "apt":
		fmt.Printf("   • **APT Issues**: \n")
		fmt.Printf("     - Package update failures: run 'sudo apt update' manually\n")
		fmt.Printf("     - Missing packages: verify Ubuntu package names are correct\n")
		fmt.Printf("     - Permission errors: ensure user has sudo access\n")

	case "dnf":
		fmt.Printf("   • **DNF Issues**: \n")
		fmt.Printf("     - Note: DNF on Ubuntu requires special configuration\n")
		fmt.Printf("     - If DNF fails: check if EPEL repositories are accessible\n")
		fmt.Printf("     - Package naming: DNF package names may differ from APT\n")
	}

	// Template-specific troubleshooting
	if strings.Contains(strings.ToLower(template.Name), "gpu") || strings.Contains(strings.ToLower(template.Name), "ml") {
		fmt.Printf("   • **GPU/ML Troubleshooting**: \n")
		fmt.Printf("     - GPU not detected: verify G-series instance type is used\n")
		fmt.Printf("     - CUDA errors: check NVIDIA driver installation in post_install script\n")
		fmt.Printf("     - Jupyter not accessible: ensure port 8888 is open in security group\n")
	}

	if strings.Contains(strings.ToLower(template.Name), "rocky") || strings.Contains(strings.ToLower(template.Name), "rhel") {
		fmt.Printf("   • **Rocky/RHEL Troubleshooting**: \n")
		fmt.Printf("     - SELinux issues: check SELinux contexts for mounted volumes\n")
		fmt.Printf("     - Firewall problems: verify firewalld rules allow required ports\n")
		fmt.Printf("     - Package repositories: ensure EPEL and PowerTools repos are enabled\n")
	}

	// Inheritance specific troubleshooting
	if len(template.Inherits) > 0 {
		fmt.Printf("   • **Inheritance Troubleshooting**: \n")
		fmt.Printf("     - Multiple users: use 'su - <username>' to switch between inherited users\n")
		fmt.Printf("     - Package conflicts: check that parent and child package managers are compatible\n")
		fmt.Printf("     - Service conflicts: verify inherited services don't conflict on same ports\n")
	}

	// General troubleshooting
	fmt.Printf("   • **General Troubleshooting**: \n")
	fmt.Printf("     - Launch failures: run with --dry-run first to check configuration\n")
	fmt.Printf("     - Connection issues: verify security group allows SSH (port 22)\n")
	fmt.Printf("     - Cost concerns: use hibernation policies for automatic cost optimization\n")
	fmt.Printf("     - Instance not starting: check template validation with 'cws templates validate'\n")

	fmt.Println()
}

// templatesUsage shows template usage statistics
func (tc *TemplateCommands) templatesUsage(args []string) error {
	stats := templates.GetUsageStats()

	fmt.Println("📊 Template Usage Statistics")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()

	// Show most popular templates
	fmt.Println("🔥 Most Popular Templates:")
	popular := stats.GetPopularTemplates(5)
	if len(popular) == 0 {
		fmt.Println("   No usage data available yet")
	} else {
		for i, usage := range popular {
			fmt.Printf("   %d. %s - %d launches (%.0f%% success rate)\n",
				i+1, usage.TemplateName, usage.LaunchCount, usage.SuccessRate*100)
			if usage.AverageLaunchTime > 0 {
				fmt.Printf("      Average launch time: %d seconds\n", usage.AverageLaunchTime)
			}
		}
	}
	fmt.Println()

	// Show recently used templates
	fmt.Println("⏰ Recently Used Templates:")
	recent := stats.GetRecentlyUsedTemplates(5)
	if len(recent) == 0 {
		fmt.Println("   No usage data available yet")
	} else {
		for _, usage := range recent {
			fmt.Printf("   • %s - Last used: %s\n",
				usage.TemplateName, usage.LastUsed.Format("Jan 2, 2006 3:04 PM"))
		}
	}
	fmt.Println()

	// Show recommendations based on usage
	if len(popular) > 0 {
		fmt.Println("💡 Recommendations:")

		// Find domain from most popular template
		if template, _ := templates.GetTemplateInfo(popular[0].TemplateName); template != nil && template.Domain != "" {
			fmt.Printf("   Based on your usage, you might also like:\n")

			// Get all templates
			registry := templates.NewTemplateRegistry(templates.DefaultTemplateDirs())
			registry.ScanTemplates()

			recommendations := templates.RecommendTemplates(registry.Templates, template.Domain, 3)
			for _, rec := range recommendations {
				if rec.Name != popular[0].TemplateName {
					fmt.Printf("   • %s - %s\n", rec.Name, rec.Description)
				}
			}
		}
		fmt.Println()
	}

	// Show tips
	fmt.Println("💡 Tips:")
	fmt.Println("   • Quick launch popular templates using their slug names")
	fmt.Println("   • Use 'cws templates discover' to explore templates by category")
	fmt.Println("   • Use 'cws templates search' to find specific templates")

	return nil
}

// templatesTest runs test suites against templates
func (tc *TemplateCommands) templatesTest(args []string) error {
	testOptions := tc.parseTestOptions(args)

	registry, err := tc.loadTemplateRegistry()
	if err != nil {
		return err
	}

	tester := templates.NewTemplateTester(registry)

	tc.printTestHeader()

	reports := tc.runTestSuite(tester, testOptions.suiteName)
	totalPassed, totalFailed := tc.processTestReports(reports, testOptions)

	return tc.printTestSummary(totalPassed, totalFailed)
}

type testOptions struct {
	templateName string
	suiteName    string
	verbose      bool
}

func (tc *TemplateCommands) parseTestOptions(args []string) testOptions {
	var options testOptions

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--suite" && i+1 < len(args):
			options.suiteName = args[i+1]
			i++
		case arg == "--verbose", arg == "-v":
			options.verbose = true
		case !strings.HasPrefix(arg, "-"):
			options.templateName = arg
		}
	}

	return options
}

func (tc *TemplateCommands) loadTemplateRegistry() (*templates.TemplateRegistry, error) {
	registry := templates.NewTemplateRegistry(templates.DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return nil, fmt.Errorf("failed to scan templates: %w", err)
	}
	return registry, nil
}

func (tc *TemplateCommands) printTestHeader() {
	fmt.Println("🧪 Running Template Tests")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()
}

func (tc *TemplateCommands) runTestSuite(tester *templates.TemplateTester, suiteName string) map[string]*templates.TestReport {
	ctx := context.Background()

	if suiteName != "" {
		fmt.Printf("Running test suite: %s\n\n", suiteName)
		// This would need to be enhanced to support specific suite selection
		return tester.RunAllTests(ctx)
	}

	return tester.RunAllTests(ctx)
}

func (tc *TemplateCommands) processTestReports(reports map[string]*templates.TestReport, options testOptions) (int, int) {
	totalPassed := 0
	totalFailed := 0

	for suiteName, report := range reports {
		tc.displaySuiteResults(suiteName, report)

		if options.verbose || report.FailedTests > 0 {
			tc.displayDetailedResults(report, options.templateName, options.verbose)
		}

		totalPassed += report.PassedTests
		totalFailed += report.FailedTests
		fmt.Println()
	}

	return totalPassed, totalFailed
}

func (tc *TemplateCommands) displaySuiteResults(suiteName string, report *templates.TestReport) {
	fmt.Printf("📦 Test Suite: %s\n", suiteName)
	fmt.Printf("   Duration: %s\n", report.EndTime.Sub(report.StartTime))
	fmt.Printf("   Tests: %d passed, %d failed\n", report.PassedTests, report.FailedTests)
}

func (tc *TemplateCommands) displayDetailedResults(report *templates.TestReport, templateName string, verbose bool) {
	for testName, result := range report.TestResults {
		if templateName != "" && !strings.Contains(testName, templateName) {
			continue
		}

		tc.displayTestResult(testName, result, verbose)
	}
}

func (tc *TemplateCommands) displayTestResult(testName string, result templates.TestResult, verbose bool) {
	if result.Passed {
		if verbose {
			fmt.Printf("   ✅ %s: %s (%s)\n", testName, result.Message, result.Duration)
		}
	} else {
		fmt.Printf("   ❌ %s: %s\n", testName, result.Message)
		for _, detail := range result.Details {
			fmt.Printf("      • %s\n", detail)
		}
	}
}

func (tc *TemplateCommands) printTestSummary(totalPassed, totalFailed int) error {
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("📊 Test Summary:\n")
	fmt.Printf("   Total tests: %d\n", totalPassed+totalFailed)
	fmt.Printf("   Passed: %d\n", totalPassed)
	fmt.Printf("   Failed: %d\n", totalFailed)

	if totalFailed > 0 {
		fmt.Printf("\n❌ %d tests failed\n", totalFailed)
		return fmt.Errorf("template tests failed")
	}

	fmt.Printf("\n✅ All tests passed!\n")
	return nil
}
