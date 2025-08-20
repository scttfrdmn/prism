package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile/export"
	"github.com/spf13/cobra"
)

// AddExportCommands adds the export and import commands using Strategy Pattern (SOLID: Single Responsibility)
func AddExportCommands(profilesCmd *cobra.Command, config *Config) {
	// Create command factory
	commandFactory := NewExportCommandFactory(config)

	// Add export and import commands
	profilesCmd.AddCommand(commandFactory.CreateExportCommand())
	profilesCmd.AddCommand(commandFactory.CreateImportCommand())
}

// Export Command Factory Pattern Implementation (SOLID: Single Responsibility + Open/Closed)

// ExportCommandFactory creates export/import commands using Factory Pattern (SOLID: Single Responsibility)
type ExportCommandFactory struct {
	config *Config
}

func NewExportCommandFactory(config *Config) *ExportCommandFactory {
	return &ExportCommandFactory{config: config}
}

func (f *ExportCommandFactory) CreateExportCommand() *cobra.Command {
	exportCmd := &cobra.Command{
		Use:   "export [output-file]",
		Short: "Export profiles to file",
		Long: `Export CloudWorkstation profiles to a file for backup or sharing.
		
By default, credentials are not included in exports for security reasons.
Use the --include-credentials flag to include credentials (use with caution).

Examples:
  cws profiles export my-profiles.zip                # Export all profiles
  cws profiles export my-profiles.json --format json # Export in JSON format
  cws profiles export --profiles work,personal       # Export specific profiles`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			handler := NewExportHandler(f.config)
			if err := handler.HandleExport(cmd, args); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "export profiles"))
				os.Exit(1)
			}
		},
	}

	f.addExportFlags(exportCmd)
	return exportCmd
}

func (f *ExportCommandFactory) CreateImportCommand() *cobra.Command {
	importCmd := &cobra.Command{
		Use:   "import [input-file]",
		Short: "Import profiles from file",
		Long: `Import CloudWorkstation profiles from a previously exported file.
		
By default, imported profiles will be renamed if they conflict with existing ones.
Use --mode to control how conflicts are handled (skip, overwrite, rename).

Examples:
  cws profiles import my-profiles.zip              # Import all profiles
  cws profiles import my-profiles.zip --mode skip  # Skip existing profiles
  cws profiles import --profiles work,personal     # Import specific profiles`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			handler := NewImportHandler(f.config)
			if err := handler.HandleImport(cmd, args); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "import profiles"))
				os.Exit(1)
			}
		},
	}

	f.addImportFlags(importCmd)
	return importCmd
}

func (f *ExportCommandFactory) addExportFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("include-credentials", false, "Include credentials in the export (use with caution)")
	cmd.Flags().Bool("include-invitations", true, "Include invitation profiles in the export")
	cmd.Flags().String("profiles", "", "Comma-separated list of profile names to export")
	cmd.Flags().String("format", "zip", "Export format (zip, json)")
	cmd.Flags().String("password", "", "Password to encrypt the export (only for zip format)")
}

func (f *ExportCommandFactory) addImportFlags(cmd *cobra.Command) {
	cmd.Flags().String("mode", "rename", "How to handle existing profiles (skip, overwrite, rename)")
	cmd.Flags().String("profiles", "", "Comma-separated list of profile names to import")
	cmd.Flags().Bool("import-credentials", false, "Import credentials if available")
	cmd.Flags().String("password", "", "Password for encrypted imports")
}

// ExportHandler handles export operations using Strategy Pattern (SOLID: Single Responsibility)
type ExportHandler struct {
	config            *Config
	flagService       *ExportFlagService
	profileService    *ExportProfileService
	filterService     *ExportFilterService
	validationService *ExportValidationService
	executionService  *ExportExecutionService
}

func NewExportHandler(config *Config) *ExportHandler {
	return &ExportHandler{
		config:            config,
		flagService:       NewExportFlagService(),
		profileService:    NewExportProfileService(),
		filterService:     NewExportFilterService(),
		validationService: NewExportValidationService(),
		executionService:  NewExportExecutionService(),
	}
}

func (h *ExportHandler) HandleExport(cmd *cobra.Command, args []string) error {
	outputPath := args[0]

	// Parse flags
	exportFlags, err := h.flagService.ParseExportFlags(cmd)
	if err != nil {
		return err
	}

	// Create profile manager
	profileManager, err := h.profileService.CreateProfileManager(h.config)
	if err != nil {
		return err
	}

	// Get and filter profiles
	selectedProfiles, err := h.filterService.GetFilteredProfiles(profileManager, exportFlags)
	if err != nil {
		return err
	}

	// Validate export
	if err := h.validationService.ValidateExport(selectedProfiles); err != nil {
		return err
	}

	// Execute export
	return h.executionService.ExecuteExport(profileManager, selectedProfiles, outputPath, exportFlags)
}

// ImportHandler handles import operations using Strategy Pattern (SOLID: Single Responsibility)
type ImportHandler struct {
	config            *Config
	flagService       *ImportFlagService
	profileService    *ImportProfileService
	validationService *ImportValidationService
	executionService  *ImportExecutionService
	resultService     *ImportResultService
}

func NewImportHandler(config *Config) *ImportHandler {
	return &ImportHandler{
		config:            config,
		flagService:       NewImportFlagService(),
		profileService:    NewImportProfileService(),
		validationService: NewImportValidationService(),
		executionService:  NewImportExecutionService(),
		resultService:     NewImportResultService(),
	}
}

func (h *ImportHandler) HandleImport(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	// Parse flags
	importFlags, err := h.flagService.ParseImportFlags(cmd)
	if err != nil {
		return err
	}

	// Create profile manager
	profileManager, err := h.profileService.CreateProfileManager(h.config)
	if err != nil {
		return err
	}

	// Validate import
	if err := h.validationService.ValidateImport(importFlags); err != nil {
		return err
	}

	// Execute import
	result, err := h.executionService.ExecuteImport(profileManager, inputPath, importFlags)
	if err != nil {
		return err
	}

	// Display results
	return h.resultService.DisplayResults(result)
}

// Export Strategy Services (SOLID: Single Responsibility + Open/Closed)

type ExportFlags struct {
	IncludeCredentials bool
	IncludeInvitations bool
	ProfilesFlag       string
	FormatFlag         string
	PasswordFlag       string
}

type ExportFlagService struct{}

func NewExportFlagService() *ExportFlagService {
	return &ExportFlagService{}
}

func (s *ExportFlagService) ParseExportFlags(cmd *cobra.Command) (*ExportFlags, error) {
	includeCredentials, _ := cmd.Flags().GetBool("include-credentials")
	includeInvitations, _ := cmd.Flags().GetBool("include-invitations")
	profilesFlag, _ := cmd.Flags().GetString("profiles")
	formatFlag, _ := cmd.Flags().GetString("format")
	passwordFlag, _ := cmd.Flags().GetString("password")

	return &ExportFlags{
		IncludeCredentials: includeCredentials,
		IncludeInvitations: includeInvitations,
		ProfilesFlag:       profilesFlag,
		FormatFlag:         formatFlag,
		PasswordFlag:       passwordFlag,
	}, nil
}

type ExportProfileService struct{}

func NewExportProfileService() *ExportProfileService {
	return &ExportProfileService{}
}

func (s *ExportProfileService) CreateProfileManager(config *Config) (*profile.ManagerEnhanced, error) {
	return createProfileManager(config)
}

type ExportFilterService struct{}

func NewExportFilterService() *ExportFilterService {
	return &ExportFilterService{}
}

func (s *ExportFilterService) GetFilteredProfiles(profileManager *profile.ManagerEnhanced, flags *ExportFlags) ([]profile.Profile, error) {
	// Get all profiles
	allProfiles, err := profileManager.ListProfiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}

	// Filter by profile names if specified
	selectedProfiles := s.filterByNames(allProfiles, flags.ProfilesFlag)

	// Filter invitations if not included
	if !flags.IncludeInvitations {
		selectedProfiles = s.filterOutInvitations(selectedProfiles)
	}

	return selectedProfiles, nil
}

func (s *ExportFilterService) filterByNames(allProfiles []profile.Profile, profilesFlag string) []profile.Profile {
	if profilesFlag == "" {
		return allProfiles
	}

	profileNames := strings.Split(profilesFlag, ",")
	var selectedProfiles []profile.Profile

	for _, prof := range allProfiles {
		for _, name := range profileNames {
			if prof.Name == name || prof.AWSProfile == name {
				selectedProfiles = append(selectedProfiles, prof)
				break
			}
		}
	}

	return selectedProfiles
}

func (s *ExportFilterService) filterOutInvitations(profiles []profile.Profile) []profile.Profile {
	filteredProfiles := make([]profile.Profile, 0, len(profiles))
	for _, prof := range profiles {
		if prof.Type != "invitation" {
			filteredProfiles = append(filteredProfiles, prof)
		}
	}
	return filteredProfiles
}

type ExportValidationService struct{}

func NewExportValidationService() *ExportValidationService {
	return &ExportValidationService{}
}

func (s *ExportValidationService) ValidateExport(selectedProfiles []profile.Profile) error {
	if len(selectedProfiles) == 0 {
		fmt.Println("No profiles found to export.")
		return nil
	}
	return nil
}

type ExportExecutionService struct{}

func NewExportExecutionService() *ExportExecutionService {
	return &ExportExecutionService{}
}

func (s *ExportExecutionService) ExecuteExport(profileManager *profile.ManagerEnhanced, selectedProfiles []profile.Profile, outputPath string, flags *ExportFlags) error {
	// Set export options
	options := export.ExportOptions{
		IncludeCredentials: flags.IncludeCredentials,
		IncludeInvitations: flags.IncludeInvitations,
		Password:           flags.PasswordFlag,
		Format:             flags.FormatFlag,
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Export profiles
	if err := export.ExportProfiles(profileManager, selectedProfiles, outputPath, options); err != nil {
		return fmt.Errorf("failed to export profiles: %w", err)
	}

	fmt.Printf("Successfully exported %d profiles to %s\n", len(selectedProfiles), outputPath)
	return nil
}

// Import Strategy Services (SOLID: Single Responsibility + Open/Closed)

type ImportFlags struct {
	ModeFlag              string
	ProfilesFlag          string
	ImportCredentialsFlag bool
	PasswordFlag          string
	ImportMode            export.ImportMode
	ProfileFilter         []string
}

type ImportFlagService struct{}

func NewImportFlagService() *ImportFlagService {
	return &ImportFlagService{}
}

func (s *ImportFlagService) ParseImportFlags(cmd *cobra.Command) (*ImportFlags, error) {
	modeFlag, _ := cmd.Flags().GetString("mode")
	profilesFlag, _ := cmd.Flags().GetString("profiles")
	importCredentialsFlag, _ := cmd.Flags().GetBool("import-credentials")
	passwordFlag, _ := cmd.Flags().GetString("password")

	// Parse import mode
	importMode, err := s.parseImportMode(modeFlag)
	if err != nil {
		return nil, err
	}

	// Parse profile filter
	var profileFilter []string
	if profilesFlag != "" {
		profileFilter = strings.Split(profilesFlag, ",")
	}

	return &ImportFlags{
		ModeFlag:              modeFlag,
		ProfilesFlag:          profilesFlag,
		ImportCredentialsFlag: importCredentialsFlag,
		PasswordFlag:          passwordFlag,
		ImportMode:            importMode,
		ProfileFilter:         profileFilter,
	}, nil
}

func (s *ImportFlagService) parseImportMode(modeFlag string) (export.ImportMode, error) {
	switch modeFlag {
	case "skip":
		return export.ImportModeSkip, nil
	case "overwrite":
		return export.ImportModeOverwrite, nil
	case "rename":
		return export.ImportModeRename, nil
	default:
		return "", fmt.Errorf("invalid import mode '%s'. Must be one of: skip, overwrite, rename", modeFlag)
	}
}

type ImportProfileService struct{}

func NewImportProfileService() *ImportProfileService {
	return &ImportProfileService{}
}

func (s *ImportProfileService) CreateProfileManager(config *Config) (*profile.ManagerEnhanced, error) {
	return createProfileManager(config)
}

type ImportValidationService struct{}

func NewImportValidationService() *ImportValidationService {
	return &ImportValidationService{}
}

func (s *ImportValidationService) ValidateImport(flags *ImportFlags) error {
	// Additional validation logic can be added here
	return nil
}

type ImportExecutionService struct{}

func NewImportExecutionService() *ImportExecutionService {
	return &ImportExecutionService{}
}

func (s *ImportExecutionService) ExecuteImport(profileManager *profile.ManagerEnhanced, inputPath string, flags *ImportFlags) (*export.ImportResult, error) {
	// Set import options
	options := export.ImportOptions{
		ImportMode:        flags.ImportMode,
		ProfileFilter:     flags.ProfileFilter,
		ImportCredentials: flags.ImportCredentialsFlag,
		Password:          flags.PasswordFlag,
	}

	// Import profiles
	return export.ImportProfiles(profileManager, inputPath, options)
}

type ImportResultService struct{}

func NewImportResultService() *ImportResultService {
	return &ImportResultService{}
}

func (s *ImportResultService) DisplayResults(result *export.ImportResult) error {
	// Check import success
	if !result.Success {
		return fmt.Errorf("import failed: %s", result.Error)
	}

	fmt.Printf("Successfully imported %d profiles\n", result.ProfilesImported)

	// Show failed profiles if any
	if len(result.FailedProfiles) > 0 {
		fmt.Println("\nThe following profiles could not be imported:")
		for name, reason := range result.FailedProfiles {
			fmt.Printf("  %s: %s\n", name, reason)
		}
	}

	return nil
}
