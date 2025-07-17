package profile_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

// BenchmarkBatchInvitationCreation benchmarks the creation of batch invitations
func BenchmarkBatchInvitationCreation(b *testing.B) {
	// Skip in short mode
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-batch-invitation-benchmark")
	if err != nil {
		b.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config directory
	configDir := filepath.Join(tempDir, ".cloudworkstation")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		b.Fatalf("Failed to create config directory: %v", err)
	}

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create a profile manager for testing
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		b.Fatalf("Failed to create profile manager: %v", err)
	}

	// Create a secure invitation manager
	secureManager, err := profile.NewSecureInvitationManager(profileManager)
	if err != nil {
		b.Fatalf("Failed to create secure invitation manager: %v", err)
	}

	// Create a batch invitation manager
	batchManager := profile.NewBatchInvitationManager(secureManager)

	// Test with various batch sizes
	batchSizes := []int{1, 10, 50, 100}
	concurrencies := []int{1, 2, 5, 10, 20}

	for _, batchSize := range batchSizes {
		// Generate test invitations
		invitations := generateTestInvitations(batchSize)

		for _, concurrency := range concurrencies {
			b.Run(fmt.Sprintf("Size_%d_Concurrency_%d", batchSize, concurrency), func(b *testing.B) {
				// Reset timer for accurate measurement
				b.ResetTimer()

				// Run the benchmark
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					// Deep copy invitations to avoid side effects
					testInvitations := copyInvitations(invitations)
					b.StartTimer()

					// Create batch invitations
					result := batchManager.CreateBatchInvitations(testInvitations, "", "", concurrency)
					
					// Verify result (without affecting timing)
					b.StopTimer()
					if result.TotalProcessed != batchSize {
						b.Fatalf("Expected %d processed invitations, got %d", batchSize, result.TotalProcessed)
					}
					b.StartTimer()
				}
			})
		}
	}
}

// BenchmarkBatchInvitationImport benchmarks the CSV import functionality
func BenchmarkBatchInvitationImport(b *testing.B) {
	// Skip in short mode
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-batch-import-benchmark")
	if err != nil {
		b.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a profile manager for testing
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		b.Fatalf("Failed to create profile manager: %v", err)
	}

	// Create a secure invitation manager
	secureManager, err := profile.NewSecureInvitationManager(profileManager)
	if err != nil {
		b.Fatalf("Failed to create secure invitation manager: %v", err)
	}

	// Create a batch invitation manager
	batchManager := profile.NewBatchInvitationManager(secureManager)

	// Test with various CSV sizes
	csvSizes := []int{10, 100, 500, 1000}

	for _, csvSize := range csvSizes {
		// Generate test CSV
		csvData := generateTestCSV(csvSize)
		csvFile := filepath.Join(tempDir, fmt.Sprintf("test_%d.csv", csvSize))
		if err := os.WriteFile(csvFile, []byte(csvData), 0644); err != nil {
			b.Fatalf("Failed to write test CSV file: %v", err)
		}

		b.Run(fmt.Sprintf("Import_Size_%d", csvSize), func(b *testing.B) {
			// Reset timer for accurate measurement
			b.ResetTimer()

			// Run the benchmark
			for i := 0; i < b.N; i++ {
				// Import invitations from CSV
				invitations, err := batchManager.ImportBatchInvitationsFromCSVFile(csvFile, true)
				if err != nil {
					b.Fatalf("Failed to import invitations: %v", err)
				}

				// Verify result (without affecting timing)
				if len(invitations) != csvSize {
					b.Fatalf("Expected %d invitations, got %d", csvSize, len(invitations))
				}
			}
		})

		b.Run(fmt.Sprintf("Export_Size_%d", csvSize), func(b *testing.B) {
			// First import invitations
			invitations, err := batchManager.ImportBatchInvitationsFromCSVFile(csvFile, true)
			if err != nil {
				b.Fatalf("Failed to import invitations: %v", err)
			}

			// Create a result for export
			result := &profile.BatchInvitationResult{
				Successful:      invitations,
				Failed:          []*profile.BatchInvitation{},
				TotalProcessed:  len(invitations),
				TotalSuccessful: len(invitations),
				TotalFailed:     0,
			}

			// Prepare output file
			outputFile := filepath.Join(tempDir, fmt.Sprintf("output_%d.csv", csvSize))

			// Reset timer for accurate measurement
			b.ResetTimer()

			// Run the benchmark
			for i := 0; i < b.N; i++ {
				// Export invitations to CSV
				err := batchManager.ExportBatchInvitationsToCSVFile(outputFile, result, true)
				if err != nil {
					b.Fatalf("Failed to export invitations: %v", err)
				}
			}
		})
	}
}

// BenchmarkDeviceOperations benchmarks the device management operations
func BenchmarkDeviceOperations(b *testing.B) {
	// Skip in short mode
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-device-benchmark")
	if err != nil {
		b.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config directory
	configDir := filepath.Join(tempDir, ".cloudworkstation")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		b.Fatalf("Failed to create config directory: %v", err)
	}

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create a profile manager for testing
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		b.Fatalf("Failed to create profile manager: %v", err)
	}

	// Create a secure invitation manager
	secureManager, err := profile.NewSecureInvitationManager(profileManager)
	if err != nil {
		b.Fatalf("Failed to create secure invitation manager: %v", err)
	}

	// Create a batch device manager
	deviceManager := profile.NewBatchDeviceManager(secureManager)

	// Test with various batch sizes
	deviceCounts := []int{1, 10, 50, 100}
	concurrencies := []int{1, 5, 10}

	for _, deviceCount := range deviceCounts {
		// Generate test device operations
		devices := generateTestDeviceOperations(deviceCount)

		for _, concurrency := range concurrencies {
			b.Run(fmt.Sprintf("Revoke_Size_%d_Concurrency_%d", deviceCount, concurrency), func(b *testing.B) {
				// Reset timer for accurate measurement
				b.ResetTimer()

				// Run the benchmark
				for i := 0; i < b.N; i++ {
					// Copy devices to avoid side effects
					testDevices := make([]profile.DeviceOperationResult, len(devices))
					copy(testDevices, devices)

					// Run the device operation
					result := deviceManager.BatchRevokeDevices(testDevices, concurrency)
					
					// Verify result
					if result.TotalProcessed != deviceCount {
						b.Fatalf("Expected %d processed devices, got %d", deviceCount, result.TotalProcessed)
					}
				}
			})
		}
	}
}

// Helper functions

// generateTestInvitations generates a slice of test invitations
func generateTestInvitations(count int) []*profile.BatchInvitation {
	invitations := make([]*profile.BatchInvitation, count)
	for i := 0; i < count; i++ {
		var invType profile.InvitationType
		switch i % 3 {
		case 0:
			invType = profile.InvitationTypeReadOnly
		case 1:
			invType = profile.InvitationTypeReadWrite
		case 2:
			invType = profile.InvitationTypeAdmin
		}

		invitations[i] = &profile.BatchInvitation{
			Name:        fmt.Sprintf("Test User %d", i+1),
			Type:        invType,
			ValidDays:   30 + i%30,
			CanInvite:   i%5 == 0,
			Transferable: i%10 == 0,
			DeviceBound: i%4 != 0,
			MaxDevices:  (i % 3) + 1,
		}
	}
	return invitations
}

// copyInvitations creates a deep copy of invitations
func copyInvitations(original []*profile.BatchInvitation) []*profile.BatchInvitation {
	copied := make([]*profile.BatchInvitation, len(original))
	for i, inv := range original {
		copied[i] = &profile.BatchInvitation{
			Name:        inv.Name,
			Type:        inv.Type,
			ValidDays:   inv.ValidDays,
			CanInvite:   inv.CanInvite,
			Transferable: inv.Transferable,
			DeviceBound: inv.DeviceBound,
			MaxDevices:  inv.MaxDevices,
		}
	}
	return copied
}

// generateTestCSV generates a test CSV file
func generateTestCSV(rows int) string {
	csv := "Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices\n"
	for i := 0; i < rows; i++ {
		var invType string
		switch i % 3 {
		case 0:
			invType = "read_only"
		case 1:
			invType = "read_write"
		case 2:
			invType = "admin"
		}

		canInvite := "no"
		if i%5 == 0 {
			canInvite = "yes"
		}

		transferable := "no"
		if i%10 == 0 {
			transferable = "yes"
		}

		deviceBound := "yes"
		if i%4 == 0 {
			deviceBound = "no"
		}

		maxDevices := (i % 3) + 1

		csv += fmt.Sprintf(
			"Test User %d,%s,%d,%s,%s,%s,%d\n",
			i+1,
			invType,
			30+i%30,
			canInvite,
			transferable,
			deviceBound,
			maxDevices,
		)
	}
	return csv
}

// generateTestDeviceOperations generates test device operations
func generateTestDeviceOperations(count int) []profile.DeviceOperationResult {
	devices := make([]profile.DeviceOperationResult, count)
	for i := 0; i < count; i++ {
		devices[i] = profile.DeviceOperationResult{
			DeviceID:    fmt.Sprintf("d%016x", i),
			Token:       fmt.Sprintf("inv-%016x", i%10),
			Name:        fmt.Sprintf("Test Device %d", i+1),
			Operation:   "revoke",
			ProcessedAt: time.Now(),
		}
	}
	return devices
}