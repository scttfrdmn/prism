// Package profile provides functionality for managing CloudWorkstation profiles
package profile

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

// BatchInvitation represents a single invitation in a batch
type BatchInvitation struct {
	// Required fields
	Name string
	Type InvitationType

	// Optional fields with defaults
	ValidDays    int  // Default: 30
	CanInvite    bool // Default: false (true for admin)
	Transferable bool // Default: false
	DeviceBound  bool // Default: true
	MaxDevices   int  // Default: 1

	// Internal fields for processing
	S3ConfigPath string
	ParentToken  string

	// Result fields
	Token       string
	EncodedData string
	Error       error
}

// BatchInvitationResult holds the results of a batch invitation creation
type BatchInvitationResult struct {
	Successful      []*BatchInvitation
	Failed          []*BatchInvitation
	TotalProcessed  int
	TotalSuccessful int
	TotalFailed     int
}

// BatchInvitationManager provides functionality for batch invitation operations
type BatchInvitationManager struct {
	secureManager *SecureInvitationManager

	// Default settings from configuration
	defaultConcurrency  int
	defaultValidDays    int
	defaultDeviceBound  bool
	defaultMaxDevices   int
	defaultCanInvite    bool
	defaultTransferable bool
}

// NewBatchInvitationManager creates a new batch invitation manager
func NewBatchInvitationManager(secureManager *SecureInvitationManager) *BatchInvitationManager {
	return &BatchInvitationManager{
		secureManager:       secureManager,
		defaultConcurrency:  5,
		defaultValidDays:    30,
		defaultDeviceBound:  true,
		defaultMaxDevices:   1,
		defaultCanInvite:    false,
		defaultTransferable: false,
	}
}

// NewBatchInvitationManagerWithConfig creates a new batch invitation manager with configuration
func NewBatchInvitationManagerWithConfig(secureManager *SecureInvitationManager, config *BatchInvitationConfig) *BatchInvitationManager {
	manager := NewBatchInvitationManager(secureManager)

	// Apply configuration
	if config != nil {
		manager.defaultConcurrency = config.DefaultConcurrency
		manager.defaultValidDays = config.DefaultValidDays
		manager.defaultDeviceBound = config.DefaultDeviceBound
		manager.defaultMaxDevices = config.DefaultMaxDevices
		manager.defaultCanInvite = config.DefaultCanInvite
		manager.defaultTransferable = config.DefaultTransferable
	}

	return manager
}

// CreateBatchInvitations creates multiple invitations in a batch
func (m *BatchInvitationManager) CreateBatchInvitations(
	invitations []*BatchInvitation,
	s3ConfigPath string,
	parentToken string,
	concurrency int,
) *BatchInvitationResult {
	// Set default concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	// Apply default settings to invitations
	for _, inv := range invitations {
		// Set defaults for empty values
		if inv.ValidDays <= 0 {
			inv.ValidDays = 30
		}

		// Set default CanInvite based on type
		if inv.Type == InvitationTypeAdmin && !inv.CanInvite {
			inv.CanInvite = true
		}

		// Default to device-bound
		if !inv.DeviceBound {
			inv.DeviceBound = true
		}

		// Default max devices
		if inv.MaxDevices <= 0 {
			inv.MaxDevices = 1
		}

		// Set common parameters
		inv.S3ConfigPath = s3ConfigPath
		inv.ParentToken = parentToken
	}

	// Create worker pool
	var wg sync.WaitGroup
	jobs := make(chan *BatchInvitation, len(invitations))
	results := &BatchInvitationResult{
		Successful: make([]*BatchInvitation, 0),
		Failed:     make([]*BatchInvitation, 0),
	}

	// Mutex for thread-safe result collection
	var resultMutex sync.Mutex

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Process jobs
			for inv := range jobs {
				// Create invitation
				invitation, err := m.secureManager.CreateSecureInvitation(
					inv.Name,
					inv.Type,
					inv.ValidDays,
					inv.S3ConfigPath,
					inv.CanInvite,
					inv.Transferable,
					inv.DeviceBound,
					inv.MaxDevices,
					inv.ParentToken,
				)

				resultMutex.Lock()
				if err != nil {
					// Failed invitation
					inv.Error = err
					results.Failed = append(results.Failed, inv)
					results.TotalFailed++
				} else {
					// Successful invitation
					inv.Token = invitation.Token

					// Encode for sharing
					encoded, encodeErr := invitation.EncodeToString()
					if encodeErr != nil {
						inv.Error = fmt.Errorf("created invitation but failed to encode: %w", encodeErr)
						results.Failed = append(results.Failed, inv)
						results.TotalFailed++
					} else {
						inv.EncodedData = encoded
						results.Successful = append(results.Successful, inv)
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

// ImportBatchInvitationsFromCSV imports batch invitations from a CSV file
func (m *BatchInvitationManager) ImportBatchInvitationsFromCSV(
	reader io.Reader,
	hasHeader bool,
) ([]*BatchInvitation, error) {
	csvReader := csv.NewReader(reader)

	// Read header if present
	if hasHeader {
		_, err := csvReader.Read()
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV header: %w", err)
		}
	}

	// Read rows
	var invitations []*BatchInvitation

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV: %w", err)
		}

		// Validate row length
		if len(record) < 2 {
			return nil, fmt.Errorf("invalid CSV format: row must have at least name and type columns")
		}

		// Parse required fields
		name := strings.TrimSpace(record[0])
		typeStr := strings.TrimSpace(record[1])

		if name == "" {
			return nil, fmt.Errorf("name cannot be empty")
		}

		// Parse invitation type
		var invType InvitationType
		switch strings.ToLower(typeStr) {
		case "read_only", "readonly", "ro":
			invType = InvitationTypeReadOnly
		case "read_write", "readwrite", "rw":
			invType = InvitationTypeReadWrite
		case "admin", "administrator":
			invType = InvitationTypeAdmin
		default:
			return nil, fmt.Errorf("invalid invitation type: %s", typeStr)
		}

		// Create invitation with required fields
		invitation := &BatchInvitation{
			Name:        name,
			Type:        invType,
			ValidDays:   30,   // default
			DeviceBound: true, // default
			MaxDevices:  1,    // default
		}

		// Parse optional fields if present
		if len(record) >= 3 && record[2] != "" {
			if validDays, err := strconv.Atoi(strings.TrimSpace(record[2])); err == nil && validDays > 0 {
				invitation.ValidDays = validDays
			}
		}

		if len(record) >= 4 && record[3] != "" {
			canInvite := parseBoolString(strings.TrimSpace(record[3]))
			invitation.CanInvite = canInvite
		} else {
			// Default canInvite to true for admin type
			invitation.CanInvite = invType == InvitationTypeAdmin
		}

		if len(record) >= 5 && record[4] != "" {
			transferable := parseBoolString(strings.TrimSpace(record[4]))
			invitation.Transferable = transferable
		}

		if len(record) >= 6 && record[5] != "" {
			deviceBound := parseBoolString(strings.TrimSpace(record[5]))
			invitation.DeviceBound = deviceBound
		}

		if len(record) >= 7 && record[6] != "" {
			if maxDevices, err := strconv.Atoi(strings.TrimSpace(record[6])); err == nil && maxDevices > 0 {
				invitation.MaxDevices = maxDevices
			}
		}

		invitations = append(invitations, invitation)
	}

	return invitations, nil
}

// ExportBatchInvitationsToCSV exports batch invitation results to a CSV file
func (m *BatchInvitationManager) ExportBatchInvitationsToCSV(
	writer io.Writer,
	results *BatchInvitationResult,
	includeEncodedData bool,
) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write header
	var header []string
	if includeEncodedData {
		header = []string{
			"Name", "Type", "Token", "Valid Days", "Can Invite", "Transferable",
			"Device Bound", "Max Devices", "Status", "Encoded Data", "Error",
		}
	} else {
		header = []string{
			"Name", "Type", "Token", "Valid Days", "Can Invite", "Transferable",
			"Device Bound", "Max Devices", "Status", "Error",
		}
	}

	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write successful invitations
	for _, inv := range results.Successful {
		record := []string{
			inv.Name,
			string(inv.Type),
			inv.Token,
			strconv.Itoa(inv.ValidDays),
			boolToString(inv.CanInvite),
			boolToString(inv.Transferable),
			boolToString(inv.DeviceBound),
			strconv.Itoa(inv.MaxDevices),
			"Success",
			"", // No error
		}

		// Add encoded data if requested
		if includeEncodedData {
			record = append(record, inv.EncodedData)
		}

		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	// Write failed invitations
	for _, inv := range results.Failed {
		errMsg := ""
		if inv.Error != nil {
			errMsg = inv.Error.Error()
		}

		record := []string{
			inv.Name,
			string(inv.Type),
			inv.Token, // Will be empty for failures
			strconv.Itoa(inv.ValidDays),
			boolToString(inv.CanInvite),
			boolToString(inv.Transferable),
			boolToString(inv.DeviceBound),
			strconv.Itoa(inv.MaxDevices),
			"Failed",
			errMsg,
		}

		// Add encoded data if requested
		if includeEncodedData {
			record = append(record, "") // Empty for failures
		}

		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}

// CreateBatchInvitationsFromCSVFile creates batch invitations from a CSV file
func (m *BatchInvitationManager) CreateBatchInvitationsFromCSVFile(
	csvPath string,
	s3ConfigPath string,
	parentToken string,
	concurrency int,
	hasHeader bool,
) (*BatchInvitationResult, error) {
	// Open CSV file
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Import invitations from CSV
	invitations, err := m.ImportBatchInvitationsFromCSV(file, hasHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to import invitations from CSV: %w", err)
	}

	// Create batch invitations
	results := m.CreateBatchInvitations(invitations, s3ConfigPath, parentToken, concurrency)
	return results, nil
}

// ExportBatchInvitationsToCSVFile exports batch invitation results to a CSV file
func (m *BatchInvitationManager) ExportBatchInvitationsToCSVFile(
	csvPath string,
	results *BatchInvitationResult,
	includeEncodedData bool,
) error {
	// Create CSV file
	file, err := os.Create(csvPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Export invitations to CSV
	return m.ExportBatchInvitationsToCSV(file, results, includeEncodedData)
}

// Helper functions

// parseBoolString parses a string into a bool
func parseBoolString(s string) bool {
	s = strings.ToLower(s)
	return s == "yes" || s == "true" || s == "t" || s == "1" || s == "y"
}

// boolToString converts a bool to a string
func boolToString(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
