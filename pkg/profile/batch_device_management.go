package profile

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile/security"
)

// BatchDeviceManagementResult holds the results of a batch device management operation
type BatchDeviceManagementResult struct {
	Successful     []DeviceOperationResult
	Failed         []DeviceOperationResult
	TotalProcessed int
	TotalSuccessful int
	TotalFailed    int
}

// DeviceOperationResult holds the result of a single device operation
type DeviceOperationResult struct {
	DeviceID    string
	Token       string
	Name        string
	Operation   string // "revoke", "validate", "info"
	Success     bool
	Error       error
	Details     map[string]interface{}
	ProcessedAt time.Time
}

// BatchDeviceManager provides functionality for batch device operations
type BatchDeviceManager struct {
	secureManager      *SecureInvitationManager
	defaultConcurrency int
}

// NewBatchDeviceManager creates a new batch device manager
func NewBatchDeviceManager(secureManager *SecureInvitationManager) *BatchDeviceManager {
	return &BatchDeviceManager{
		secureManager: secureManager,
		defaultConcurrency: 5,
	}
}

// NewBatchDeviceManagerWithConfig creates a new batch device manager with configuration
func NewBatchDeviceManagerWithConfig(secureManager *SecureInvitationManager, config *BatchInvitationConfig) *BatchDeviceManager {
	manager := NewBatchDeviceManager(secureManager)
	
	// Apply configuration
	if config != nil {
		manager.defaultConcurrency = config.DefaultConcurrency
	}
	
	return manager
}

// BatchRevokeDevices revokes multiple devices across multiple invitations
func (m *BatchDeviceManager) BatchRevokeDevices(
	devices []DeviceOperationResult,
	concurrency int,
) *BatchDeviceManagementResult {
	// Set default concurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	
	// Create worker pool
	var wg sync.WaitGroup
	jobs := make(chan DeviceOperationResult, len(devices))
	results := &BatchDeviceManagementResult{
		Successful: make([]DeviceOperationResult, 0),
		Failed:     make([]DeviceOperationResult, 0),
	}
	
	// Mutex for thread-safe result collection
	var resultMutex sync.Mutex
	
	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for device := range jobs {
				// Set operation type and timestamp
				device.Operation = "revoke"
				device.ProcessedAt = time.Now()
				
				// Revoke the device
				err := m.secureManager.RevokeDevice(device.Token, device.DeviceID)
				
				resultMutex.Lock()
				if err != nil {
					device.Success = false
					device.Error = err
					results.Failed = append(results.Failed, device)
					results.TotalFailed++
				} else {
					device.Success = true
					results.Successful = append(results.Successful, device)
					results.TotalSuccessful++
				}
				resultMutex.Unlock()
			}
		}()
	}
	
	// Queue jobs
	for _, device := range devices {
		jobs <- device
	}
	close(jobs)
	
	// Wait for all workers to finish
	wg.Wait()
	
	// Set total processed
	results.TotalProcessed = len(devices)
	
	return results
}

// BatchValidateDevices validates multiple devices across multiple invitations
func (m *BatchDeviceManager) BatchValidateDevices(
	devices []DeviceOperationResult,
	concurrency int,
) *BatchDeviceManagementResult {
	// Set default concurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	
	// Create worker pool
	var wg sync.WaitGroup
	jobs := make(chan DeviceOperationResult, len(devices))
	results := &BatchDeviceManagementResult{
		Successful: make([]DeviceOperationResult, 0),
		Failed:     make([]DeviceOperationResult, 0),
	}
	
	// Mutex for thread-safe result collection
	var resultMutex sync.Mutex
	
	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for device := range jobs {
				// Set operation type and timestamp
				device.Operation = "validate"
				device.ProcessedAt = time.Now()
				
				// Validate the device
				valid, err := m.secureManager.registry.ValidateDevice(device.Token, device.DeviceID)
				
				resultMutex.Lock()
				if err != nil || !valid {
					device.Success = false
					if err != nil {
						device.Error = err
					} else {
						device.Error = fmt.Errorf("device not valid")
					}
					results.Failed = append(results.Failed, device)
					results.TotalFailed++
				} else {
					device.Success = true
					results.Successful = append(results.Successful, device)
					results.TotalSuccessful++
				}
				resultMutex.Unlock()
			}
		}()
	}
	
	// Queue jobs
	for _, device := range devices {
		jobs <- device
	}
	close(jobs)
	
	// Wait for all workers to finish
	wg.Wait()
	
	// Set total processed
	results.TotalProcessed = len(devices)
	
	return results
}

// BatchGetDeviceInfo retrieves information for multiple devices
func (m *BatchDeviceManager) BatchGetDeviceInfo(
	invitations []*InvitationToken,
	concurrency int,
) *BatchDeviceManagementResult {
	// Set default concurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	
	// Create worker pool
	var wg sync.WaitGroup
	jobs := make(chan *InvitationToken, len(invitations))
	results := &BatchDeviceManagementResult{
		Successful: make([]DeviceOperationResult, 0),
		Failed:     make([]DeviceOperationResult, 0),
	}
	
	// Mutex for thread-safe result collection
	var resultMutex sync.Mutex
	
	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for inv := range jobs {
				// Get the devices for this invitation
				devices, err := m.secureManager.GetInvitationDevices(inv.Token)
				
				resultMutex.Lock()
				if err != nil {
					// Failed to get devices
					result := DeviceOperationResult{
						Token:       inv.Token,
						Name:        inv.Name,
						Operation:   "info",
						Success:     false,
						Error:       err,
						ProcessedAt: time.Now(),
					}
					results.Failed = append(results.Failed, result)
					results.TotalFailed++
				} else {
					// Process each device
					for _, device := range devices {
						deviceID, ok := device["device_id"].(string)
						if !ok {
							continue
						}
						
						result := DeviceOperationResult{
							DeviceID:    deviceID,
							Token:       inv.Token,
							Name:        inv.Name,
							Operation:   "info",
							Success:     true,
							Details:     device,
							ProcessedAt: time.Now(),
						}
						results.Successful = append(results.Successful, result)
						results.TotalSuccessful++
					}
				}
				resultMutex.Unlock()
			}
		}()
	}
	
	// Queue jobs
	for _, inv := range invitations {
		jobs <- inv
	}
	close(jobs)
	
	// Wait for all workers to finish
	wg.Wait()
	
	// Set total processed
	results.TotalProcessed = len(invitations)
	
	return results
}

// ExportDeviceInfoToCSV exports device information to a CSV file
func (m *BatchDeviceManager) ExportDeviceInfoToCSV(
	writer io.Writer,
	results *BatchDeviceManagementResult,
) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()
	
	// Write header
	header := []string{
		"Device ID", "Token", "Invitation Name", "Operation",
		"Status", "Registered At", "Last Seen", "Details", "Error",
	}
	
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}
	
	// Write successful operations
	for _, result := range results.Successful {
		// Extract additional details
		registeredAt := ""
		lastSeen := ""
		details := ""
		
		if result.Details != nil {
			if reg, ok := result.Details["registered_at"]; ok {
				registeredAt = fmt.Sprintf("%v", reg)
			}
			if seen, ok := result.Details["last_seen"]; ok {
				lastSeen = fmt.Sprintf("%v", seen)
			}
			
			// Format other details as a string
			var detailItems []string
			for k, v := range result.Details {
				if k != "device_id" && k != "registered_at" && k != "last_seen" {
					detailItems = append(detailItems, fmt.Sprintf("%s: %v", k, v))
				}
			}
			details = strings.Join(detailItems, "; ")
		}
		
		record := []string{
			result.DeviceID,
			result.Token,
			result.Name,
			result.Operation,
			"Success",
			registeredAt,
			lastSeen,
			details,
			"",
		}
		
		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}
	
	// Write failed operations
	for _, result := range results.Failed {
		errMsg := ""
		if result.Error != nil {
			errMsg = result.Error.Error()
		}
		
		record := []string{
			result.DeviceID,
			result.Token,
			result.Name,
			result.Operation,
			"Failed",
			"",
			"",
			"",
			errMsg,
		}
		
		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}
	
	return nil
}

// ExportDeviceInfoToCSVFile exports device information to a CSV file
func (m *BatchDeviceManager) ExportDeviceInfoToCSVFile(
	csvPath string,
	results *BatchDeviceManagementResult,
) error {
	file, err := os.Create(csvPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()
	
	return m.ExportDeviceInfoToCSV(file, results)
}

// ImportDevicesFromCSV imports device operations from a CSV file
func (m *BatchDeviceManager) ImportDevicesFromCSV(
	reader io.Reader,
	hasHeader bool,
) ([]DeviceOperationResult, error) {
	csvReader := csv.NewReader(reader)
	
	// Read header if present
	if hasHeader {
		_, err := csvReader.Read()
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV header: %w", err)
		}
	}
	
	// Read rows
	var devices []DeviceOperationResult
	
	for rowNum := 1; ; rowNum++ {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV: %w", err)
		}
		
		// Validate row length
		if len(record) < 3 {
			return nil, fmt.Errorf("invalid CSV format at row %d: row must have at least device_id, token, and operation columns", rowNum)
		}
		
		// Parse required fields
		deviceID := strings.TrimSpace(record[0])
		token := strings.TrimSpace(record[1])
		name := ""
		if len(record) > 2 {
			name = strings.TrimSpace(record[2])
		}
		
		if deviceID == "" {
			return nil, fmt.Errorf("device_id cannot be empty at row %d", rowNum)
		}
		
		if token == "" {
			return nil, fmt.Errorf("token cannot be empty at row %d", rowNum)
		}
		
		// Create device operation
		device := DeviceOperationResult{
			DeviceID:  deviceID,
			Token:     token,
			Name:      name,
			Operation: "revoke", // Default operation
		}
		
		// Add optional operation if present
		if len(record) > 3 && record[3] != "" {
			op := strings.ToLower(strings.TrimSpace(record[3]))
			if op == "revoke" || op == "validate" || op == "info" {
				device.Operation = op
			}
		}
		
		devices = append(devices, device)
	}
	
	return devices, nil
}

// ImportDevicesFromCSVFile imports device operations from a CSV file
func (m *BatchDeviceManager) ImportDevicesFromCSVFile(
	csvPath string,
	hasHeader bool,
) ([]DeviceOperationResult, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()
	
	return m.ImportDevicesFromCSV(file, hasHeader)
}

// ExecuteBatchDeviceOperation processes device operations from a CSV file
func (m *BatchDeviceManager) ExecuteBatchDeviceOperation(
	csvPath string,
	operation string,
	concurrency int,
	hasHeader bool,
) (*BatchDeviceManagementResult, error) {
	// Import devices from CSV
	devices, err := m.ImportDevicesFromCSVFile(csvPath, hasHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to import devices from CSV: %w", err)
	}
	
	// Override operation if specified
	if operation != "" {
		for i := range devices {
			devices[i].Operation = operation
		}
	}
	
	// Group devices by operation
	revokeDevices := make([]DeviceOperationResult, 0)
	validateDevices := make([]DeviceOperationResult, 0)
	infoInvitations := make(map[string]*InvitationToken)
	
	for _, device := range devices {
		switch device.Operation {
		case "revoke":
			revokeDevices = append(revokeDevices, device)
		case "validate":
			validateDevices = append(validateDevices, device)
		case "info":
			// For info operations, we need to collect unique invitations
			if _, exists := infoInvitations[device.Token]; !exists {
				// Get invitation details
				inv, err := m.secureManager.GetInvitation(device.Token)
				if err == nil {
					infoInvitations[device.Token] = inv
				}
			}
		}
	}
	
	// Process each operation type
	var results BatchDeviceManagementResult
	
	// Process revoke operations
	if len(revokeDevices) > 0 {
		revokeResults := m.BatchRevokeDevices(revokeDevices, concurrency)
		results.Successful = append(results.Successful, revokeResults.Successful...)
		results.Failed = append(results.Failed, revokeResults.Failed...)
		results.TotalSuccessful += revokeResults.TotalSuccessful
		results.TotalFailed += revokeResults.TotalFailed
	}
	
	// Process validate operations
	if len(validateDevices) > 0 {
		validateResults := m.BatchValidateDevices(validateDevices, concurrency)
		results.Successful = append(results.Successful, validateResults.Successful...)
		results.Failed = append(results.Failed, validateResults.Failed...)
		results.TotalSuccessful += validateResults.TotalSuccessful
		results.TotalFailed += validateResults.TotalFailed
	}
	
	// Process info operations
	if len(infoInvitations) > 0 {
		invitationsList := make([]*InvitationToken, 0, len(infoInvitations))
		for _, inv := range infoInvitations {
			invitationsList = append(invitationsList, inv)
		}
		
		infoResults := m.BatchGetDeviceInfo(invitationsList, concurrency)
		results.Successful = append(results.Successful, infoResults.Successful...)
		results.Failed = append(results.Failed, infoResults.Failed...)
		results.TotalSuccessful += infoResults.TotalSuccessful
		results.TotalFailed += infoResults.TotalFailed
	}
	
	// Set total processed
	results.TotalProcessed = len(devices)
	
	return &results, nil
}