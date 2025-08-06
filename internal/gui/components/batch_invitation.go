package components

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// BatchInvitationManager provides GUI integration for batch invitation operations
type BatchInvitationManager struct {
	ctx            context.Context
	secureManager  *profile.SecureInvitationManager
	batchManager   *profile.BatchInvitationManager
	operationMutex sync.Mutex
	lastResult     *BatchOperationResult
}

// BatchOperationResult holds the results of a batch operation for the GUI
type BatchOperationResult struct {
	Operation       string    `json:"operation"`
	TotalProcessed  int       `json:"totalProcessed"`
	TotalSuccessful int       `json:"totalSuccessful"`
	TotalFailed     int       `json:"totalFailed"`
	CompletedAt     time.Time `json:"completedAt"`
	OutputFile      string    `json:"outputFile"`
	Error           string    `json:"error"`
}

// InvitationRow represents a single invitation in the GUI table
type InvitationRow struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	ValidDays   int    `json:"validDays"`
	CanInvite   bool   `json:"canInvite"`
	Transferable bool   `json:"transferable"`
	DeviceBound bool   `json:"deviceBound"`
	MaxDevices  int    `json:"maxDevices"`
	Token       string `json:"token"`
	EncodedData string `json:"encodedData"`
	Status      string `json:"status"`
	Error       string `json:"error"`
}

// NewBatchInvitationManager creates a new batch invitation manager for the GUI
func NewBatchInvitationManager(ctx context.Context, secureManager *profile.SecureInvitationManager) *BatchInvitationManager {
	batchManager := profile.NewBatchInvitationManager(secureManager)
	
	return &BatchInvitationManager{
		ctx:           ctx,
		secureManager: secureManager,
		batchManager:  batchManager,
	}
}

// SelectCSVFileForImport opens a file dialog to select a CSV file for import
func (m *BatchInvitationManager) SelectCSVFileForImport() string {
	filePath, err := runtime.OpenFileDialog(m.ctx, runtime.OpenDialogOptions{
		Title: "Select CSV File with Invitations",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "CSV Files (*.csv)",
				Pattern:     "*.csv",
			},
		},
	})
	
	if err != nil {
		return ""
	}
	
	return filePath
}

// SelectOutputFileForExport opens a file dialog to select an output CSV file
func (m *BatchInvitationManager) SelectOutputFileForExport() string {
	filePath, err := runtime.SaveFileDialog(m.ctx, runtime.SaveDialogOptions{
		Title: "Save Results as CSV",
		DefaultFilename: fmt.Sprintf("invitations_%s.csv", 
			time.Now().Format("2006-01-02")),
		Filters: []runtime.FileFilter{
			{
				DisplayName: "CSV Files (*.csv)",
				Pattern:     "*.csv",
			},
		},
	})
	
	if err != nil {
		return ""
	}
	
	return filePath
}

// PreviewCSVFile reads a CSV file and returns a preview of the invitations
func (m *BatchInvitationManager) PreviewCSVFile(filePath string, hasHeader bool) ([]InvitationRow, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}
	
	// Open and read the CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()
	
	// Import invitations from CSV
	invitations, err := m.batchManager.ImportBatchInvitationsFromCSV(file, hasHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to import invitations: %w", err)
	}
	
	// Convert to GUI-friendly format
	rows := make([]InvitationRow, len(invitations))
	for i, inv := range invitations {
		rows[i] = InvitationRow{
			Name:        inv.Name,
			Type:        string(inv.Type),
			ValidDays:   inv.ValidDays,
			CanInvite:   inv.CanInvite,
			Transferable: inv.Transferable,
			DeviceBound: inv.DeviceBound,
			MaxDevices:  inv.MaxDevices,
			Status:      "Pending",
		}
	}
	
	return rows, nil
}

// CreateBatchInvitations creates invitations from a CSV file
func (m *BatchInvitationManager) CreateBatchInvitations(
	filePath string, 
	s3ConfigPath string, 
	parentToken string, 
	hasHeader bool, 
	concurrency int,
	outputFilePath string,
) (*BatchOperationResult, error) {
	// Prevent multiple operations running simultaneously
	m.operationMutex.Lock()
	defer m.operationMutex.Unlock()
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}
	
	// Set default output path if not provided
	if outputFilePath == "" {
		dir := filepath.Dir(filePath)
		base := filepath.Base(filePath)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]
		outputFilePath = filepath.Join(dir, name+"_results.csv")
	}
	
	// Set default concurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	
	// Process invitations
	result, err := m.batchManager.CreateBatchInvitationsFromCSVFile(
		filePath, s3ConfigPath, parentToken, concurrency, hasHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to create batch invitations: %w", err)
	}
	
	// Export results
	err = m.batchManager.ExportBatchInvitationsToCSVFile(outputFilePath, result, true)
	if err != nil {
		return nil, fmt.Errorf("failed to export results: %w", err)
	}
	
	// Create GUI result
	guiResult := &BatchOperationResult{
		Operation:       "create",
		TotalProcessed:  result.TotalProcessed,
		TotalSuccessful: result.TotalSuccessful,
		TotalFailed:     result.TotalFailed,
		CompletedAt:     time.Now(),
		OutputFile:      outputFilePath,
	}
	
	// Save last result
	m.lastResult = guiResult
	
	return guiResult, nil
}

// ExportAllInvitations exports all current invitations to a CSV file
func (m *BatchInvitationManager) ExportAllInvitations(outputFilePath string) (*BatchOperationResult, error) {
	// Prevent multiple operations running simultaneously
	m.operationMutex.Lock()
	defer m.operationMutex.Unlock()
	
	// Get all invitations from the secure manager
	invitations := m.secureManager.ListInvitations()
	if len(invitations) == 0 {
		return nil, fmt.Errorf("no invitations found to export")
	}
	
	// Convert to batch format
	batchInvitations := make([]*profile.BatchInvitation, len(invitations))
	for i, inv := range invitations {
		// Create encoded form for sharing
		encodedData, err := inv.EncodeToString()
		if err != nil {
			return nil, fmt.Errorf("failed to encode invitation: %w", err)
		}
		
		batchInvitations[i] = &profile.BatchInvitation{
			Name:        inv.Name,
			Type:        inv.Type,
			ValidDays:   int(inv.GetExpirationDuration().Hours() / 24),
			CanInvite:   inv.CanInvite,
			Transferable: inv.Transferable,
			DeviceBound: inv.DeviceBound,
			MaxDevices:  inv.MaxDevices,
			Token:       inv.Token,
			EncodedData: encodedData,
		}
	}
	
	// Create batch result
	results := &profile.BatchInvitationResult{
		Successful:     batchInvitations,
		Failed:         []*profile.BatchInvitation{},
		TotalProcessed: len(batchInvitations),
		TotalSuccessful: len(batchInvitations),
		TotalFailed:    0,
	}
	
	// Export to file
	err := m.batchManager.ExportBatchInvitationsToCSVFile(
		outputFilePath, results, true)
	if err != nil {
		return nil, fmt.Errorf("failed to export invitations: %w", err)
	}
	
	// Create GUI result
	guiResult := &BatchOperationResult{
		Operation:       "export",
		TotalProcessed:  len(invitations),
		TotalSuccessful: len(invitations),
		TotalFailed:     0,
		CompletedAt:     time.Now(),
		OutputFile:      outputFilePath,
	}
	
	// Save last result
	m.lastResult = guiResult
	
	return guiResult, nil
}

// AcceptBatchInvitations accepts multiple invitations from a CSV file
func (m *BatchInvitationManager) AcceptBatchInvitations(
	filePath string, 
	namePrefix string,
	hasHeader bool,
) (*BatchOperationResult, error) {
	// Prevent multiple operations running simultaneously
	m.operationMutex.Lock()
	defer m.operationMutex.Unlock()
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}
	
	// Open the CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()
	
	// Import invitations from CSV - since this is just for accepting,
	// we can assume the data includes encoded invitations in the 10th column
	invitations, err := m.batchManager.ImportBatchInvitationsFromCSV(file, hasHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to import invitations: %w", err)
	}
	
	// Process each invitation
	successful := 0
	failed := 0
	errors := make([]string, 0)
	
	for _, inv := range invitations {
		if inv.EncodedData == "" {
			failed++
			errors = append(errors, fmt.Sprintf("Missing encoded data for %s", inv.Name))
			continue
		}
		
		// Generate profile name from invitation name
		profileName := inv.Name
		if namePrefix != "" {
			profileName = fmt.Sprintf("%s-%s", namePrefix, inv.Name)
		}
		
		// Accept the invitation
		err := m.secureManager.SecureAddToProfile(inv.EncodedData, profileName)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("Failed to accept invitation for %s: %v", inv.Name, err))
		} else {
			successful++
		}
	}
	
	// Create GUI result
	guiResult := &BatchOperationResult{
		Operation:       "accept",
		TotalProcessed:  len(invitations),
		TotalSuccessful: successful,
		TotalFailed:     failed,
		CompletedAt:     time.Now(),
		Error:           "",
	}
	
	// Include error summary if there were failures
	if failed > 0 {
		if len(errors) > 3 {
			guiResult.Error = fmt.Sprintf("%s (and %d more errors)", 
				errors[0], len(errors)-1)
		} else {
			guiResult.Error = errors[0]
		}
	}
	
	// Save last result
	m.lastResult = guiResult
	
	return guiResult, nil
}

// GetLastOperationResult returns the result of the last batch operation
func (m *BatchInvitationManager) GetLastOperationResult() *BatchOperationResult {
	return m.lastResult
}

// GenerateEmptyCSVTemplate creates an empty CSV template for invitations
func (m *BatchInvitationManager) GenerateEmptyCSVTemplate(filePath string) error {
	// Create the template content
	template := `Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
User 1,read_only,30,no,no,yes,1
User 2,read_write,60,no,no,yes,2
Admin User,admin,90,yes,no,yes,3
`

	// Write to file
	err := os.WriteFile(filePath, []byte(template), 0644)
	if err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}
	
	return nil
}

// ShowCSVFile opens the CSV file in the default application
func (m *BatchInvitationManager) ShowCSVFile(filePath string) error {
	// TODO: Implement file opening functionality
	// Could use exec.Command("open", filePath) on macOS or similar per-platform
	return fmt.Errorf("file opening not implemented yet")
}

// OpenCSVFolder opens the folder containing the CSV file
func (m *BatchInvitationManager) OpenCSVFolder(filePath string) error {
	dir := filepath.Dir(filePath)
	// TODO: Implement folder opening functionality
	// Could use exec.Command("open", dir) on macOS or similar per-platform
	_ = dir
	return fmt.Errorf("folder opening not implemented yet")
}