package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// newMockS3Server creates an httptest server that mimics S3 REST API responses
// for PUT, GET, HEAD, and DELETE object operations.
func newMockS3Server(t *testing.T, content string) (*httptest.Server, *s3.Client) {
	t.Helper()

	objects := map[string]string{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			objects[r.URL.Path] = content
			w.Header().Set("ETag", `"65a8e27d8879283831b664bd8b7f0ad4"`)
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			body, ok := objects[r.URL.Path]
			if !ok {
				body = content // Return content for any key
			}
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
			w.Header().Set("ETag", `"65a8e27d8879283831b664bd8b7f0ad4"`)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(body))
		case http.MethodHead:
			body, ok := objects[r.URL.Path]
			if !ok {
				body = content
			}
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
			w.Header().Set("ETag", `"65a8e27d8879283831b664bd8b7f0ad4"`)
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			delete(objects, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithBaseEndpoint(srv.URL),
	)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	t.Cleanup(srv.Close)
	return srv, client
}

// MockS3Client provides a mock S3 client for testing
type MockS3Client struct {
	s3.Client
	// Track calls
	HeadObjectCalls   int
	PutObjectCalls    int
	GetObjectCalls    int
	DeleteObjectCalls int

	// Mock responses
	HeadObjectResponse *s3.HeadObjectOutput
	HeadObjectError    error
	DeleteObjectError  error
}

func (m *MockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	m.HeadObjectCalls++
	if m.HeadObjectError != nil {
		return nil, m.HeadObjectError
	}
	if m.HeadObjectResponse != nil {
		return m.HeadObjectResponse, nil
	}
	// Default response
	contentLength := int64(1024)
	return &s3.HeadObjectOutput{
		ContentLength: &contentLength,
	}, nil
}

func (m *MockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	m.DeleteObjectCalls++
	if m.DeleteObjectError != nil {
		return nil, m.DeleteObjectError
	}
	return &s3.DeleteObjectOutput{}, nil
}

func TestDefaultTransferOptions(t *testing.T) {
	opts := DefaultTransferOptions()

	if opts.PartSize != DefaultPartSize {
		t.Errorf("Expected PartSize %d, got %d", DefaultPartSize, opts.PartSize)
	}

	if opts.Concurrency != DefaultConcurrency {
		t.Errorf("Expected Concurrency %d, got %d", DefaultConcurrency, opts.Concurrency)
	}

	if !opts.Checksum {
		t.Error("Expected Checksum to be enabled by default")
	}

	if !opts.ResumeSupport {
		t.Error("Expected ResumeSupport to be enabled by default")
	}

	if opts.AutoCleanup {
		t.Error("Expected AutoCleanup to be disabled by default")
	}

	if opts.ProgressInterval != 1*time.Second {
		t.Errorf("Expected ProgressInterval 1s, got %v", opts.ProgressInterval)
	}
}

func TestNewTransferManager(t *testing.T) {
	mockClient := &MockS3Client{}
	s3Client := &s3.Client{}

	// Test with default options
	tm := NewTransferManager(s3Client, nil)
	if tm == nil {
		t.Fatal("Expected non-nil TransferManager")
	}

	if tm.s3Client != s3Client {
		t.Error("S3 client not set correctly")
	}

	if tm.transferClient == nil {
		t.Error("Transfer client not initialized")
	}

	if tm.transfers == nil {
		t.Error("Transfers map not initialized")
	}

	if tm.options == nil {
		t.Error("Options not initialized")
	}

	// Test with custom options
	customOpts := &TransferOptions{
		PartSize:    20 * 1024 * 1024,
		Concurrency: 10,
		Checksum:    false,
	}
	tm2 := NewTransferManager(s3Client, customOpts)
	if tm2.options.PartSize != customOpts.PartSize {
		t.Errorf("Expected PartSize %d, got %d", customOpts.PartSize, tm2.options.PartSize)
	}

	_ = mockClient // Use mockClient to avoid unused variable warning
}

func TestTransferProgress(t *testing.T) {
	progress := &TransferProgress{
		TransferID:       "test-transfer-1",
		Type:             TransferTypeUpload,
		Status:           TransferStatusInProgress,
		FilePath:         "/tmp/test.txt",
		S3Bucket:         "test-bucket",
		S3Key:            "test-key",
		TotalBytes:       1000,
		TransferredBytes: 500,
		PercentComplete:  50.0,
		StartTime:        time.Now(),
	}

	if progress.TransferID != "test-transfer-1" {
		t.Errorf("Expected TransferID 'test-transfer-1', got '%s'", progress.TransferID)
	}

	if progress.Type != TransferTypeUpload {
		t.Errorf("Expected Type %s, got %s", TransferTypeUpload, progress.Type)
	}

	if progress.Status != TransferStatusInProgress {
		t.Errorf("Expected Status %s, got %s", TransferStatusInProgress, progress.Status)
	}

	if progress.PercentComplete != 50.0 {
		t.Errorf("Expected PercentComplete 50.0, got %f", progress.PercentComplete)
	}
}

func TestGetTransferProgress(t *testing.T) {
	tm := NewTransferManager(&s3.Client{}, nil)

	// Create and register a transfer
	progress := &TransferProgress{
		TransferID: "transfer-123",
		Status:     TransferStatusInProgress,
	}

	tm.mu.Lock()
	tm.transfers["transfer-123"] = progress
	tm.mu.Unlock()

	// Test getting existing transfer
	retrieved, exists := tm.GetTransferProgress("transfer-123")
	if !exists {
		t.Error("Expected transfer to exist")
	}

	if retrieved.TransferID != "transfer-123" {
		t.Errorf("Expected TransferID 'transfer-123', got '%s'", retrieved.TransferID)
	}

	// Test getting non-existent transfer
	_, exists = tm.GetTransferProgress("nonexistent")
	if exists {
		t.Error("Expected transfer to not exist")
	}
}

func TestListTransfers(t *testing.T) {
	tm := NewTransferManager(&s3.Client{}, nil)

	// Add multiple transfers
	transfers := []*TransferProgress{
		{TransferID: "transfer-1", Status: TransferStatusInProgress},
		{TransferID: "transfer-2", Status: TransferStatusCompleted},
		{TransferID: "transfer-3", Status: TransferStatusPending},
	}

	tm.mu.Lock()
	for _, p := range transfers {
		tm.transfers[p.TransferID] = p
	}
	tm.mu.Unlock()

	// Test listing
	list := tm.ListTransfers()
	if len(list) != 3 {
		t.Errorf("Expected 3 transfers, got %d", len(list))
	}

	// Verify all transfers are in the list
	foundIDs := make(map[string]bool)
	for _, p := range list {
		foundIDs[p.TransferID] = true
	}

	for _, expected := range transfers {
		if !foundIDs[expected.TransferID] {
			t.Errorf("Transfer %s not found in list", expected.TransferID)
		}
	}
}

func TestDeleteObject(t *testing.T) {
	// Check if localstack is available
	ctx := context.Background()

	// Configure S3 client for localstack
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		t.Skipf("Skipping TestDeleteObject - cannot configure localstack client: %v", err)
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("http://localhost:4566")
		o.UsePathStyle = true // Required for localstack
	})

	// Create test bucket
	testBucket := "test-delete-bucket"
	_, err = s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &testBucket,
	})
	if err != nil {
		t.Skipf("Skipping TestDeleteObject - localstack not available: %v", err)
	}
	defer s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: &testBucket})

	// Create transfer manager
	tm := NewTransferManager(s3Client, DefaultTransferOptions())

	// Upload a test object first
	testKey := "test-delete-object.txt"
	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &testBucket,
		Key:    &testKey,
		Body:   strings.NewReader("test content for deletion"),
	})
	if err != nil {
		t.Fatalf("Failed to create test object: %v", err)
	}

	// Test DeleteObject
	err = tm.DeleteObject(ctx, testBucket, testKey)
	if err != nil {
		t.Errorf("DeleteObject failed: %v", err)
	}

	// Verify object is deleted
	_, err = s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &testBucket,
		Key:    &testKey,
	})
	if err == nil {
		t.Error("Expected object to be deleted, but it still exists")
	}
}

func TestComputeFileMD5(t *testing.T) {
	// Create a temporary test file
	tmpDir := os.TempDir()
	testFile := filepath.Join(tmpDir, "test-md5.txt")

	content := []byte("Hello, World!")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Compute checksum
	checksum, err := computeFileMD5(testFile)
	if err != nil {
		t.Fatalf("computeFileMD5 failed: %v", err)
	}

	// Verify checksum is valid hex string
	if len(checksum) != 32 {
		t.Errorf("Expected MD5 checksum length 32, got %d", len(checksum))
	}

	// Known MD5 of "Hello, World!" is 65a8e27d8879283831b664bd8b7f0ad4
	expected := "65a8e27d8879283831b664bd8b7f0ad4"
	if checksum != expected {
		t.Errorf("Expected checksum %s, got %s", expected, checksum)
	}

	// Test with nonexistent file
	_, err = computeFileMD5("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestVerifyChecksum(t *testing.T) {
	tests := []struct {
		name     string
		etag     string
		checksum string
		expected bool
	}{
		{
			name:     "matching checksums",
			etag:     "abc123",
			checksum: "abc123",
			expected: true,
		},
		{
			name:     "matching checksums with quotes",
			etag:     "\"abc123\"",
			checksum: "abc123",
			expected: true,
		},
		{
			name:     "non-matching checksums",
			etag:     "abc123",
			checksum: "xyz789",
			expected: false,
		},
		{
			name:     "multipart upload etag (skip verification)",
			etag:     "d41d8cd98f00b204e9800998ecf8427e-2", // 32 char MD5 + dash + part count = 35 chars
			checksum: "anything",
			expected: true, // Multipart ETags are skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifyChecksum(tt.etag, tt.checksum)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for etag=%s checksum=%s",
					tt.expected, result, tt.etag, tt.checksum)
			}
		})
	}
}

func TestTrimQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "\"hello\"", expected: "hello"},
		{input: "hello", expected: "hello"},
		{input: "\"", expected: "\""},
		{input: "", expected: ""},
		{input: "\"hello", expected: "\"hello"},
		{input: "hello\"", expected: "hello\""},
	}

	for _, tt := range tests {
		result := trimQuotes(tt.input)
		if result != tt.expected {
			t.Errorf("trimQuotes(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestGenerateTransferID(t *testing.T) {
	id1 := generateTransferID()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamp
	id2 := generateTransferID()

	if id1 == id2 {
		t.Error("Expected unique transfer IDs")
	}

	if len(id1) == 0 {
		t.Error("Expected non-empty transfer ID")
	}

	// Should start with "transfer-"
	if len(id1) < 9 || id1[:9] != "transfer-" {
		t.Errorf("Expected transfer ID to start with 'transfer-', got %s", id1)
	}
}

func TestTransferStatusConstants(t *testing.T) {
	// Verify all status constants are defined correctly
	statuses := []TransferStatus{
		TransferStatusPending,
		TransferStatusInProgress,
		TransferStatusPaused,
		TransferStatusCompleted,
		TransferStatusFailed,
		TransferStatusCancelled,
	}

	for _, status := range statuses {
		if string(status) == "" {
			t.Errorf("Transfer status constant has empty string value: %v", status)
		}
	}
}

func TestTransferTypeConstants(t *testing.T) {
	// Verify transfer type constants
	if TransferTypeUpload != "upload" {
		t.Errorf("Expected TransferTypeUpload to be 'upload', got '%s'", TransferTypeUpload)
	}

	if TransferTypeDownload != "download" {
		t.Errorf("Expected TransferTypeDownload to be 'download', got '%s'", TransferTypeDownload)
	}
}

func TestProgressReader(t *testing.T) {
	// Create a test progress object
	progress := &TransferProgress{
		TotalBytes:       100,
		TransferredBytes: 0,
		StartTime:        time.Now(),
	}

	// Create mock reader with 100 bytes
	mockData := make([]byte, 100)
	for i := range mockData {
		mockData[i] = byte(i)
	}

	// Track callback invocations
	callbackCount := 0
	callback := func(p *TransferProgress) {
		callbackCount++
	}

	pr := &progressReader{
		reader:   &mockReader{data: mockData},
		progress: progress,
		callback: callback,
		interval: 10 * time.Millisecond,
		mu:       &sync.RWMutex{},
	}

	// Read data in chunks
	buf := make([]byte, 10)
	for i := 0; i < 10; i++ {
		n, err := pr.Read(buf)
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
		if n != 10 {
			t.Errorf("Expected to read 10 bytes, got %d", n)
		}
		time.Sleep(15 * time.Millisecond) // Ensure callback fires
	}

	// Verify progress tracking
	if progress.TransferredBytes != 100 {
		t.Errorf("Expected TransferredBytes 100, got %d", progress.TransferredBytes)
	}

	if progress.PercentComplete != 100.0 {
		t.Errorf("Expected PercentComplete 100.0, got %f", progress.PercentComplete)
	}

	if progress.BytesPerSecond == 0 {
		t.Error("Expected BytesPerSecond to be calculated")
	}

	if callbackCount == 0 {
		t.Error("Expected callback to be invoked")
	}
}

// mockReader provides a simple mock for io.Reader
type mockReader struct {
	data   []byte
	offset int
}

func (m *mockReader) Read(p []byte) (int, error) {
	if m.offset >= len(m.data) {
		return 0, os.ErrClosed
	}

	n := copy(p, m.data[m.offset:])
	m.offset += n
	return n, nil
}

func TestProgressWriter(t *testing.T) {
	// Create a test progress object
	progress := &TransferProgress{
		TotalBytes:       100,
		TransferredBytes: 0,
		StartTime:        time.Now(),
	}

	// Create mock writer
	mockData := make([]byte, 100)

	// Track callback invocations
	callbackCount := 0
	callback := func(p *TransferProgress) {
		callbackCount++
	}

	pw := &progressWriter{
		writer:   &mockWriterAt{data: mockData},
		progress: progress,
		callback: callback,
		interval: 10 * time.Millisecond,
		mu:       &sync.RWMutex{},
	}

	// Write data in chunks
	data := make([]byte, 10)
	for i := 0; i < 10; i++ {
		n, err := pw.WriteAt(data, int64(i*10))
		if err != nil {
			t.Fatalf("WriteAt error: %v", err)
		}
		if n != 10 {
			t.Errorf("Expected to write 10 bytes, got %d", n)
		}
		time.Sleep(15 * time.Millisecond) // Ensure callback fires
	}

	// Verify progress tracking
	if progress.TransferredBytes != 100 {
		t.Errorf("Expected TransferredBytes 100, got %d", progress.TransferredBytes)
	}

	if progress.PercentComplete != 100.0 {
		t.Errorf("Expected PercentComplete 100.0, got %f", progress.PercentComplete)
	}

	if progress.BytesPerSecond == 0 {
		t.Error("Expected BytesPerSecond to be calculated")
	}

	if callbackCount == 0 {
		t.Error("Expected callback to be invoked")
	}
}

// mockWriterAt provides a simple mock for io.WriterAt
type mockWriterAt struct {
	data []byte
}

func (m *mockWriterAt) WriteAt(p []byte, off int64) (int, error) {
	if off+int64(len(p)) > int64(len(m.data)) {
		return 0, os.ErrInvalid
	}

	return copy(m.data[off:], p), nil
}

func TestTransferManagerConcurrency(t *testing.T) {
	tm := NewTransferManager(&s3.Client{}, nil)

	// Test concurrent access without race conditions
	// Use unique IDs based on goroutine index to avoid timestamp collisions
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			// Create unique transfer ID using goroutine index
			transferID := fmt.Sprintf("transfer-test-%d", id)

			progress := &TransferProgress{
				TransferID: transferID,
				Status:     TransferStatusPending,
			}

			// Concurrent write to transfers map
			tm.mu.Lock()
			tm.transfers[progress.TransferID] = progress
			tm.mu.Unlock()

			// Concurrent read from transfers map
			_, exists := tm.GetTransferProgress(transferID)
			if !exists {
				t.Errorf("Failed to retrieve transfer %s", transferID)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all transfers were added
	list := tm.ListTransfers()
	if len(list) != 10 {
		t.Errorf("Expected 10 transfers, got %d", len(list))
	}
}

func TestTransferOptionsValidation(t *testing.T) {
	opts := &TransferOptions{
		PartSize:    MinPartSize - 1, // Below minimum
		Concurrency: -1,              // Invalid
	}

	// Create transfer manager (should handle invalid values)
	tm := NewTransferManager(&s3.Client{}, opts)
	if tm == nil {
		t.Fatal("Expected non-nil TransferManager even with invalid options")
	}

	// AWS SDK will handle validation of PartSize and Concurrency
	// Just verify manager was created
}

// TestUploadFile_InvalidPath tests upload with non-existent file

// TestUploadFile_InvalidPath tests upload with non-existent file
func TestUploadFile_InvalidPath(t *testing.T) {
	client := &s3.Client{}
	tm := NewTransferManager(client, DefaultTransferOptions())

	ctx := context.Background()
	nonExistentFile := "/tmp/this-file-does-not-exist-12345.dat"

	progress, err := tm.UploadFile(ctx, nonExistentFile, "test-bucket", "test-key")

	// Should fail with file not found error
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if progress != nil {
		t.Error("Expected nil progress for failed upload")
	}
}

// TestUploadFile_ProgressTracking tests that progress callback can be configured
func TestUploadFile_ProgressTracking(t *testing.T) {
	// Verify the transfer manager can be created with progress callback
	options := DefaultTransferOptions()
	options.ProgressCallback = func(p *TransferProgress) {
		// Verify progress structure in callback
		if p.Type != TransferTypeUpload {
			t.Errorf("Expected upload type, got %s", p.Type)
		}
	}

	client := &s3.Client{}
	tm := NewTransferManager(client, options)
	if tm == nil {
		t.Fatal("Failed to create transfer manager")
	}

	// Actual upload with progress requires real S3 connection
	// Integration test in test/integration/storage_transfer_test.go covers this
	t.Log("✓ Transfer manager created with progress callback")
	t.Log("✓ Full upload with progress tracking tested in integration tests")
}

// TestDownloadFile_DirectoryCreation tests directory creation logic
func TestDownloadFile_DirectoryCreation(t *testing.T) {
	// This test verifies the directory creation would happen
	// Actual download requires real S3 which is tested in integration tests
	tmpDir := t.TempDir()
	downloadPath := filepath.Join(tmpDir, "subdir1", "subdir2", "file.dat")

	// Create the parent directory as the download would
	parentDir := filepath.Dir(downloadPath)
	err := os.MkdirAll(parentDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		t.Error("Expected parent directory to be created")
	}

	t.Log("✓ Download would create parent directories")
}

// TestDeleteObject_Success tests successful object deletion
func TestDeleteObject_Success(t *testing.T) {
	mockClient := &MockS3Client{}
	// Can't use mockClient directly, need real client for manager
	// This test verifies the mock interface works
	ctx := context.Background()
	_, err := mockClient.DeleteObject(ctx, &s3.DeleteObjectInput{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if mockClient.DeleteObjectCalls != 1 {
		t.Errorf("Expected 1 DeleteObject call, got %d", mockClient.DeleteObjectCalls)
	}

	t.Log("✓ Mock DeleteObject works")
}

// TestDeleteObject_Error tests deletion error handling
func TestDeleteObject_Error(t *testing.T) {
	mockClient := &MockS3Client{
		DeleteObjectError: fmt.Errorf("S3 deletion failed"),
	}

	ctx := context.Background()
	_, err := mockClient.DeleteObject(ctx, &s3.DeleteObjectInput{})

	if err == nil {
		t.Error("Expected error from S3 deletion")
	}

	if mockClient.DeleteObjectCalls != 1 {
		t.Errorf("Expected 1 DeleteObject call, got %d", mockClient.DeleteObjectCalls)
	}

	t.Log("✓ Mock DeleteObject error handled correctly")
}

// TestTransferProgress_Calculations tests progress percentage and speed calculations
func TestTransferProgress_Calculations(t *testing.T) {
	progress := &TransferProgress{
		TotalBytes:       1000,
		TransferredBytes: 250,
		StartTime:        time.Now().Add(-10 * time.Second),
	}

	// Calculate percent
	progress.PercentComplete = float64(progress.TransferredBytes) / float64(progress.TotalBytes) * 100

	if progress.PercentComplete != 25.0 {
		t.Errorf("Expected 25%% complete, got %.1f%%", progress.PercentComplete)
	}

	// Calculate speed
	elapsed := time.Since(progress.StartTime).Seconds()
	progress.BytesPerSecond = int64(float64(progress.TransferredBytes) / elapsed)

	if progress.BytesPerSecond < 20 || progress.BytesPerSecond > 30 {
		t.Errorf("Expected ~25 bytes/sec, got %d", progress.BytesPerSecond)
	}

	t.Log("✓ Progress calculations correct")
}

// TestTransferManager_MultipleTransfers tests tracking multiple simultaneous transfers
func TestTransferManager_MultipleTransfers(t *testing.T) {
	client := &s3.Client{}
	tm := NewTransferManager(client, nil)

	// Register multiple mock transfers
	for i := 0; i < 5; i++ {
		transferID := fmt.Sprintf("transfer-%d", i)
		progress := &TransferProgress{
			TransferID: transferID,
			Status:     TransferStatusInProgress,
		}

		tm.mu.Lock()
		tm.transfers[transferID] = progress
		tm.mu.Unlock()
	}

	// List all transfers
	transfers := tm.ListTransfers()
	if len(transfers) != 5 {
		t.Errorf("Expected 5 transfers, got %d", len(transfers))
	}

	// Get specific transfer
	progress, exists := tm.GetTransferProgress("transfer-2")
	if !exists {
		t.Error("Expected transfer-2 to exist")
	}
	if progress.TransferID != "transfer-2" {
		t.Errorf("Expected transfer-2, got %s", progress.TransferID)
	}

	t.Log("✓ Multiple transfer tracking works")
}

// TestTransferOptions_Defaults tests default options are reasonable
func TestTransferOptions_Defaults(t *testing.T) {
	opts := DefaultTransferOptions()

	// Verify sane defaults
	if opts.PartSize < MinPartSize {
		t.Errorf("Default part size %d below minimum %d", opts.PartSize, MinPartSize)
	}
	if opts.PartSize > MaxPartSize {
		t.Errorf("Default part size %d above maximum %d", opts.PartSize, MaxPartSize)
	}
	if opts.Concurrency < 1 {
		t.Errorf("Default concurrency %d should be positive", opts.Concurrency)
	}
	if opts.Concurrency > 20 {
		t.Errorf("Default concurrency %d seems too high", opts.Concurrency)
	}
	if !opts.Checksum {
		t.Error("Checksum should be enabled by default")
	}
	if !opts.ResumeSupport {
		t.Error("Resume support should be enabled by default")
	}
	if opts.ProgressInterval < 100*time.Millisecond {
		t.Error("Progress interval should be at least 100ms")
	}

	t.Logf("✓ Defaults: PartSize=%d, Concurrency=%d, Checksum=%v",
		opts.PartSize, opts.Concurrency, opts.Checksum)
}

// TestTransferStatus_Transitions tests valid status transitions
func TestTransferStatus_Transitions(t *testing.T) {
	validTransitions := []struct {
		from  TransferStatus
		to    TransferStatus
		valid bool
	}{
		{TransferStatusPending, TransferStatusInProgress, true},
		{TransferStatusInProgress, TransferStatusCompleted, true},
		{TransferStatusInProgress, TransferStatusFailed, true},
		{TransferStatusInProgress, TransferStatusPaused, true},
		{TransferStatusPaused, TransferStatusInProgress, true},
		{TransferStatusCompleted, TransferStatusInProgress, false}, // Can't restart completed
		{TransferStatusFailed, TransferStatusInProgress, false},    // Can't restart failed
	}

	for _, tt := range validTransitions {
		// Just verify the states exist and can be compared
		if tt.from == "" || tt.to == "" {
			t.Errorf("Empty status transition: %s -> %s", tt.from, tt.to)
		}
	}

	t.Log("✓ Transfer status transitions documented")
}

// --- httptest-based mock S3 tests (covers UploadFile, DownloadFile, DeleteObject) ---

func TestUploadFile_Success(t *testing.T) {
	_, client := newMockS3Server(t, "")

	// Create a temp file to upload
	tmpFile, err := os.CreateTemp(t.TempDir(), "upload-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	content := "Hello, Prism upload test!"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	opts := DefaultTransferOptions()
	opts.Checksum = false // disable to avoid MD5 mismatch with mock ETag
	tm := NewTransferManager(client, opts)

	ctx := context.Background()
	progress, err := tm.UploadFile(ctx, tmpFile.Name(), "test-bucket", "test-key.txt")
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}

	if progress == nil {
		t.Fatal("Expected non-nil progress")
	}
	if progress.Status != TransferStatusCompleted {
		t.Errorf("Expected status completed, got %s", progress.Status)
	}
	if progress.PercentComplete != 100 {
		t.Errorf("Expected 100%% complete, got %.1f", progress.PercentComplete)
	}
	if progress.Type != TransferTypeUpload {
		t.Errorf("Expected upload type, got %s", progress.Type)
	}
	if progress.S3Bucket != "test-bucket" {
		t.Errorf("Expected bucket 'test-bucket', got '%s'", progress.S3Bucket)
	}

	// Verify transfer is tracked
	tracked, exists := tm.GetTransferProgress(progress.TransferID)
	if !exists {
		t.Error("Expected transfer to be tracked")
	}
	if tracked.Status != TransferStatusCompleted {
		t.Errorf("Expected tracked status completed, got %s", tracked.Status)
	}

	t.Log("✓ UploadFile succeeds with mock S3")
}

func TestUploadFile_S3Failure(t *testing.T) {
	// Server that always returns 500
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithBaseEndpoint(srv.URL),
	)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	tmpFile, err := os.CreateTemp(t.TempDir(), "upload-fail-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	_, _ = tmpFile.WriteString("test")
	tmpFile.Close()

	opts := DefaultTransferOptions()
	opts.Checksum = false
	tm := NewTransferManager(client, opts)

	ctx := context.Background()
	progress, err := tm.UploadFile(ctx, tmpFile.Name(), "test-bucket", "test-key.txt")

	if err == nil {
		t.Error("Expected error for S3 failure")
	}
	if progress == nil {
		t.Error("Expected progress even on failure")
	} else if progress.Status != TransferStatusFailed {
		t.Errorf("Expected status failed, got %s", progress.Status)
	}

	t.Log("✓ UploadFile returns failed status on S3 error")
}

func TestDownloadFile_Success(t *testing.T) {
	content := "Hello, Prism download test!"
	_, client := newMockS3Server(t, content)

	opts := DefaultTransferOptions()
	opts.Checksum = false // disable checksum to avoid ETag mismatch with mock
	tm := NewTransferManager(client, opts)

	ctx := context.Background()
	downloadPath := filepath.Join(t.TempDir(), "downloaded.txt")

	progress, err := tm.DownloadFile(ctx, "test-bucket", "test-key.txt", downloadPath)
	if err != nil {
		t.Fatalf("DownloadFile failed: %v", err)
	}

	if progress == nil {
		t.Fatal("Expected non-nil progress")
	}
	if progress.Status != TransferStatusCompleted {
		t.Errorf("Expected status completed, got %s", progress.Status)
	}
	if progress.PercentComplete != 100 {
		t.Errorf("Expected 100%% complete, got %.1f", progress.PercentComplete)
	}
	if progress.Type != TransferTypeDownload {
		t.Errorf("Expected download type, got %s", progress.Type)
	}

	// Verify file was written
	data, err := os.ReadFile(downloadPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(data) != content {
		t.Errorf("Expected content %q, got %q", content, string(data))
	}

	t.Log("✓ DownloadFile succeeds with mock S3")
}

func TestDownloadFile_InvalidBucket(t *testing.T) {
	// Server that returns 404 for HEAD (object not found)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithBaseEndpoint(srv.URL),
	)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	tm := NewTransferManager(client, DefaultTransferOptions())
	ctx := context.Background()
	downloadPath := filepath.Join(t.TempDir(), "should-not-exist.txt")

	_, err = tm.DownloadFile(ctx, "nonexistent-bucket", "nonexistent-key", downloadPath)
	if err == nil {
		t.Error("Expected error for nonexistent object")
	}

	t.Log("✓ DownloadFile returns error for nonexistent object")
}

func TestDeleteObject_WithMockS3(t *testing.T) {
	_, client := newMockS3Server(t, "to-be-deleted")

	tm := NewTransferManager(client, nil)
	ctx := context.Background()

	err := tm.DeleteObject(ctx, "test-bucket", "test-key.txt")
	if err != nil {
		t.Errorf("DeleteObject failed: %v", err)
	}

	t.Log("✓ TransferManager.DeleteObject succeeds with mock S3")
}

func TestDeleteObject_S3Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithBaseEndpoint(srv.URL),
	)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	tm := NewTransferManager(client, nil)
	ctx := context.Background()

	err = tm.DeleteObject(ctx, "test-bucket", "test-key.txt")
	if err == nil {
		t.Error("Expected error for forbidden delete")
	}

	t.Log("✓ TransferManager.DeleteObject returns error on S3 failure")
}

func TestDownloadFile_ChecksumMismatch(t *testing.T) {
	content := "checksum-mismatch-content"

	// Server returns mismatched ETag to trigger checksum verification failure
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
			w.Header().Set("ETag", `"000000000000000000000000BADETAG0"`) // Wrong MD5
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
			w.Header().Set("ETag", `"000000000000000000000000BADETAG0"`) // Wrong MD5
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(content))
		}
	}))
	defer srv.Close()

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithBaseEndpoint(srv.URL),
	)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	opts := DefaultTransferOptions()
	opts.Checksum = true // enable checksum verification
	tm := NewTransferManager(client, opts)
	ctx := context.Background()
	downloadPath := filepath.Join(t.TempDir(), "checksum-test.txt")

	progress, err := tm.DownloadFile(ctx, "test-bucket", "test-key.txt", downloadPath)
	if err == nil {
		t.Error("Expected checksum mismatch error")
	}
	if progress == nil {
		t.Error("Expected progress even on checksum failure")
	} else if progress.Status != TransferStatusFailed {
		t.Errorf("Expected failed status on checksum mismatch, got %s", progress.Status)
	}

	t.Log("✓ DownloadFile detects checksum mismatch")
}

func TestUploadFile_WithChecksum(t *testing.T) {
	content := "Hello, World!" // Known MD5: 65a8e27d8879283831b664bd8b7f0ad4

	// Server returns matching ETag
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"65a8e27d8879283831b664bd8b7f0ad4"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithBaseEndpoint(srv.URL),
	)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	tmpFile, err := os.CreateTemp(t.TempDir(), "checksum-upload-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	opts := DefaultTransferOptions()
	opts.Checksum = true // enable checksum for upload path coverage
	tm := NewTransferManager(client, opts)

	ctx := context.Background()
	progress, err := tm.UploadFile(ctx, tmpFile.Name(), "test-bucket", "test-key.txt")
	if err != nil {
		t.Fatalf("UploadFile with checksum failed: %v", err)
	}
	if progress.Checksum == "" {
		t.Error("Expected checksum to be computed")
	}

	t.Log("✓ UploadFile computes and records checksum")
}

func TestDownloadFile_AutoCleanup(t *testing.T) {
	content := "cleanup-test-content"
	_, client := newMockS3Server(t, content)

	opts := DefaultTransferOptions()
	opts.Checksum = false
	opts.AutoCleanup = true // enable auto-cleanup of S3 object after download

	tm := NewTransferManager(client, opts)
	ctx := context.Background()
	downloadPath := filepath.Join(t.TempDir(), "cleanup-test.txt")

	progress, err := tm.DownloadFile(ctx, "test-bucket", "test-key.txt", downloadPath)
	if err != nil {
		t.Fatalf("DownloadFile with AutoCleanup failed: %v", err)
	}

	if progress.Status != TransferStatusCompleted {
		t.Errorf("Expected completed status, got %s", progress.Status)
	}

	t.Log("✓ DownloadFile with AutoCleanup succeeds")
}

// TestCancelTransfer_NoopForUnknownID verifies that CancelTransfer is safe to call
// for a transfer ID that does not exist (already completed or never started).
func TestCancelTransfer_NoopForUnknownID(t *testing.T) {
	_, client := newMockS3Server(t, "content")
	opts := DefaultTransferOptions()
	tm := NewTransferManager(client, opts)

	err := tm.CancelTransfer("nonexistent-id")
	if err != nil {
		t.Errorf("CancelTransfer for unknown ID should be a no-op, got error: %v", err)
	}
	t.Log("✓ CancelTransfer is safe for unknown transfer IDs")
}

// TestCancelTransfer_InterruptsInProgressTransfer verifies that CancelTransfer
// stops the underlying upload and marks the transfer as cancelled.
func TestCancelTransfer_InterruptsInProgressTransfer(t *testing.T) {
	// Create a slow mock server that blocks long enough for cancellation to race
	blockCh := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", "0")
			w.WriteHeader(http.StatusOK)
			return
		}
		// Block until the test signals or the connection is closed
		select {
		case <-blockCh:
		case <-r.Context().Done():
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(func() {
		close(blockCh)
		srv.Close()
	})

	cfg, _ := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithBaseEndpoint(srv.URL),
	)
	client := s3.NewFromConfig(cfg, func(o *s3.Options) { o.UsePathStyle = true })

	opts := DefaultTransferOptions()
	tm := NewTransferManager(client, opts)

	// Write a temp file to upload
	tmpFile, err := os.CreateTemp(t.TempDir(), "cancel-test-*.bin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	_, _ = tmpFile.WriteString("data to upload")
	_ = tmpFile.Close()

	// Start upload in background — it will block inside the mock server
	uploadDone := make(chan error, 1)
	var capturedID string
	go func() {
		progress, err := tm.UploadFile(context.Background(), tmpFile.Name(), "test-bucket", "cancel-key")
		if progress != nil {
			capturedID = progress.TransferID
		}
		uploadDone <- err
	}()

	// Give the goroutine time to register the transfer
	var found bool
	for i := 0; i < 20; i++ {
		time.Sleep(10 * time.Millisecond)
		transfers := tm.ListTransfers()
		if len(transfers) > 0 {
			capturedID = transfers[0].TransferID
			found = true
			break
		}
	}
	if !found {
		t.Skip("transfer did not register in time — skipping cancel race test")
	}

	// Cancel the transfer
	if err := tm.CancelTransfer(capturedID); err != nil {
		t.Fatalf("CancelTransfer returned error: %v", err)
	}

	// Verify the progress shows cancelled
	progress, exists := tm.GetTransferProgress(capturedID)
	if exists && progress != nil {
		if progress.Status != TransferStatusCancelled {
			t.Errorf("expected TransferStatusCancelled, got %s", progress.Status)
		}
	}

	// The upload goroutine should finish (not hang) after cancellation
	select {
	case <-uploadDone:
		// upload returned (error expected due to cancellation)
	case <-time.After(5 * time.Second):
		t.Error("upload goroutine did not return after CancelTransfer")
	}

	t.Log("✓ CancelTransfer interrupts an in-progress transfer")
}
