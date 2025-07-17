package pages

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/scttfrdmn/cloudworkstation/internal/gui/components"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// BatchInvitationPage represents the batch invitation management page
type BatchInvitationPage struct {
	ctx            context.Context
	batchManager   *components.BatchInvitationManager
	secureManager  *profile.SecureInvitationManager
	assets         embed.FS
}

// NewBatchInvitationPage creates a new batch invitation page
func NewBatchInvitationPage(ctx context.Context, secureManager *profile.SecureInvitationManager, assets embed.FS) *BatchInvitationPage {
	return &BatchInvitationPage{
		ctx:           ctx,
		secureManager: secureManager,
		assets:        assets,
		batchManager:  components.NewBatchInvitationManager(ctx, secureManager),
	}
}

// SelectImportFile opens a file dialog to select a CSV file for import
func (p *BatchInvitationPage) SelectImportFile() string {
	return p.batchManager.SelectCSVFileForImport()
}

// SelectExportFile opens a file dialog to select an output CSV file
func (p *BatchInvitationPage) SelectExportFile() string {
	return p.batchManager.SelectOutputFileForExport()
}

// PreviewCSVFile reads a CSV file and returns a preview of the invitations
func (p *BatchInvitationPage) PreviewCSVFile(filePath string, hasHeader bool) string {
	rows, err := p.batchManager.PreviewCSVFile(filePath, hasHeader)
	if err != nil {
		return p.errorResponse(err.Error())
	}
	
	return p.jsonResponse(rows)
}

// CreateInvitations creates invitations from a CSV file
func (p *BatchInvitationPage) CreateInvitations(filePath, s3ConfigPath, parentToken string, hasHeader bool, concurrency int, outputFile string) string {
	result, err := p.batchManager.CreateBatchInvitations(filePath, s3ConfigPath, parentToken, hasHeader, concurrency, outputFile)
	if err != nil {
		return p.errorResponse(err.Error())
	}
	
	return p.jsonResponse(result)
}

// ExportAllInvitations exports all current invitations to a CSV file
func (p *BatchInvitationPage) ExportAllInvitations(outputFile string) string {
	result, err := p.batchManager.ExportAllInvitations(outputFile)
	if err != nil {
		return p.errorResponse(err.Error())
	}
	
	return p.jsonResponse(result)
}

// AcceptInvitations accepts multiple invitations from a CSV file
func (p *BatchInvitationPage) AcceptInvitations(filePath, namePrefix string, hasHeader bool) string {
	result, err := p.batchManager.AcceptBatchInvitations(filePath, namePrefix, hasHeader)
	if err != nil {
		return p.errorResponse(err.Error())
	}
	
	return p.jsonResponse(result)
}

// GenerateCSVTemplate creates an empty CSV template for invitations
func (p *BatchInvitationPage) GenerateCSVTemplate() string {
	// Create temp file path
	tempDir, err := runtime.TempDir()
	if err != nil {
		return p.errorResponse(err.Error())
	}
	
	fileName := "invitation_template.csv"
	filePath := filepath.Join(tempDir, fileName)
	
	err = p.batchManager.GenerateEmptyCSVTemplate(filePath)
	if err != nil {
		return p.errorResponse(err.Error())
	}
	
	// Open the file with the default application
	err = p.batchManager.ShowCSVFile(filePath)
	if err != nil {
		return p.errorResponse(err.Error())
	}
	
	return p.successResponse(fmt.Sprintf("Template created and opened: %s", filePath))
}

// OpenCSVFile opens a CSV file in the default application
func (p *BatchInvitationPage) OpenCSVFile(filePath string) string {
	err := p.batchManager.ShowCSVFile(filePath)
	if err != nil {
		return p.errorResponse(err.Error())
	}
	
	return p.successResponse("File opened successfully")
}

// OpenCSVFolder opens the folder containing a CSV file
func (p *BatchInvitationPage) OpenCSVFolder(filePath string) string {
	err := p.batchManager.OpenCSVFolder(filePath)
	if err != nil {
		return p.errorResponse(err.Error())
	}
	
	return p.successResponse("Folder opened successfully")
}

// GetLastOperationResult returns the result of the last batch operation
func (p *BatchInvitationPage) GetLastOperationResult() string {
	result := p.batchManager.GetLastOperationResult()
	if result == nil {
		return p.errorResponse("No operation has been performed yet")
	}
	
	return p.jsonResponse(result)
}

// GetAllInvitations returns all current invitations
func (p *BatchInvitationPage) GetAllInvitations() string {
	invitations := p.secureManager.ListInvitations()
	
	// Convert to simplified format
	type InvitationInfo struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		Token      string `json:"token"`
		ExpiresIn  string `json:"expiresIn"`
		DeviceBound bool   `json:"deviceBound"`
		MaxDevices int    `json:"maxDevices"`
	}
	
	result := make([]InvitationInfo, len(invitations))
	for i, inv := range invitations {
		expiresIn := inv.GetExpirationDuration()
		expiresText := "Expired"
		
		if expiresIn.Hours() > 0 {
			days := int(expiresIn.Hours() / 24)
			if days > 0 {
				expiresText = fmt.Sprintf("%d days", days)
			} else {
				expiresText = fmt.Sprintf("%d hours", int(expiresIn.Hours()))
			}
		}
		
		result[i] = InvitationInfo{
			Name:       inv.Name,
			Type:       string(inv.Type),
			Token:      inv.Token,
			ExpiresIn:  expiresText,
			DeviceBound: inv.DeviceBound,
			MaxDevices: inv.MaxDevices,
		}
	}
	
	return p.jsonResponse(result)
}

// Helper functions for JSON responses
func (p *BatchInvitationPage) jsonResponse(data interface{}) string {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return p.errorResponse(err.Error())
	}
	return string(jsonBytes)
}

func (p *BatchInvitationPage) errorResponse(message string) string {
	response := map[string]interface{}{
		"success": false,
		"error":   message,
	}
	jsonBytes, _ := json.Marshal(response)
	return string(jsonBytes)
}

func (p *BatchInvitationPage) successResponse(message string) string {
	response := map[string]interface{}{
		"success": true,
		"message": message,
	}
	jsonBytes, _ := json.Marshal(response)
	return string(jsonBytes)
}