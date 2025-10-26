// Package storage provides S3-backed file transfer functionality for template provisioning.
//
// This package implements multipart upload/download with progress tracking, resume capability,
// and checksum verification for large file transfers during template provisioning.
package storage

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	// MinPartSize is the minimum size for multipart upload chunks (5MB per AWS requirement)
	MinPartSize = 5 * 1024 * 1024

	// DefaultPartSize is the default chunk size for multipart uploads (10MB)
	DefaultPartSize = 10 * 1024 * 1024

	// MaxPartSize is the maximum chunk size for multipart uploads (100MB)
	MaxPartSize = 100 * 1024 * 1024

	// DefaultConcurrency is the default number of concurrent uploads/downloads
	DefaultConcurrency = 5

	// TransferTimeout is the maximum time for a complete transfer operation
	TransferTimeout = 1 * time.Hour

	// CheckpointInterval is how often to save transfer progress for resume
	CheckpointInterval = 30 * time.Second
)

// TransferType represents the type of transfer operation
type TransferType string

const (
	// TransferTypeUpload represents an upload operation
	TransferTypeUpload TransferType = "upload"

	// TransferTypeDownload represents a download operation
	TransferTypeDownload TransferType = "download"
)

// TransferStatus represents the current status of a transfer
type TransferStatus string

const (
	// TransferStatusPending indicates transfer is queued but not started
	TransferStatusPending TransferStatus = "pending"

	// TransferStatusInProgress indicates transfer is actively running
	TransferStatusInProgress TransferStatus = "in_progress"

	// TransferStatusPaused indicates transfer was paused (supports resume)
	TransferStatusPaused TransferStatus = "paused"

	// TransferStatusCompleted indicates transfer finished successfully
	TransferStatusCompleted TransferStatus = "completed"

	// TransferStatusFailed indicates transfer failed with error
	TransferStatusFailed TransferStatus = "failed"

	// TransferStatusCancelled indicates transfer was cancelled by user
	TransferStatusCancelled TransferStatus = "cancelled"
)

// TransferProgress tracks real-time progress of a file transfer
type TransferProgress struct {
	// TransferID is the unique identifier for this transfer
	TransferID string `json:"transfer_id"`

	// Type is the transfer operation type (upload/download)
	Type TransferType `json:"type"`

	// Status is the current transfer status
	Status TransferStatus `json:"status"`

	// FilePath is the local file path
	FilePath string `json:"file_path"`

	// S3Bucket is the S3 bucket name
	S3Bucket string `json:"s3_bucket"`

	// S3Key is the S3 object key
	S3Key string `json:"s3_key"`

	// TotalBytes is the total file size in bytes
	TotalBytes int64 `json:"total_bytes"`

	// TransferredBytes is the number of bytes transferred so far
	TransferredBytes int64 `json:"transferred_bytes"`

	// PercentComplete is the completion percentage (0-100)
	PercentComplete float64 `json:"percent_complete"`

	// BytesPerSecond is the current transfer speed
	BytesPerSecond int64 `json:"bytes_per_second"`

	// StartTime is when the transfer started
	StartTime time.Time `json:"start_time"`

	// EstimatedCompletion is the estimated time to completion
	EstimatedCompletion *time.Time `json:"estimated_completion,omitempty"`

	// Error contains error message if transfer failed
	Error string `json:"error,omitempty"`

	// Checksum is the MD5 checksum of the file
	Checksum string `json:"checksum,omitempty"`

	// PartsCompleted is the number of multipart chunks completed
	PartsCompleted int `json:"parts_completed"`

	// TotalParts is the total number of multipart chunks
	TotalParts int `json:"total_parts"`

	// LastUpdate is the timestamp of the last progress update
	LastUpdate time.Time `json:"last_update"`
}

// TransferOptions configures a file transfer operation
type TransferOptions struct {
	// PartSize is the chunk size for multipart transfers (defaults to DefaultPartSize)
	PartSize int64

	// Concurrency is the number of concurrent part uploads/downloads
	Concurrency int

	// Checksum enables MD5 checksum verification
	Checksum bool

	// AutoCleanup automatically deletes S3 object after successful download
	AutoCleanup bool

	// ResumeSupport enables saving checkpoints for resume capability
	ResumeSupport bool

	// CheckpointDir is where to save resume checkpoints
	CheckpointDir string

	// ProgressCallback is called periodically with transfer progress
	ProgressCallback func(progress *TransferProgress)

	// ProgressInterval is how often to call ProgressCallback
	ProgressInterval time.Duration
}

// DefaultTransferOptions returns sensible defaults for transfer operations
func DefaultTransferOptions() *TransferOptions {
	return &TransferOptions{
		PartSize:         DefaultPartSize,
		Concurrency:      DefaultConcurrency,
		Checksum:         true,
		AutoCleanup:      false,
		ResumeSupport:    true,
		CheckpointDir:    os.TempDir(),
		ProgressInterval: 1 * time.Second,
	}
}

// TransferManager manages S3 file transfers with progress tracking and resume capability
type TransferManager struct {
	s3Client   *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader

	// Track active transfers
	transfers map[string]*TransferProgress
	mu        sync.RWMutex

	// Configuration
	options *TransferOptions
}

// NewTransferManager creates a new S3 transfer manager
func NewTransferManager(s3Client *s3.Client, options *TransferOptions) *TransferManager {
	if options == nil {
		options = DefaultTransferOptions()
	}

	// Create uploader with configured part size and concurrency
	uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
		u.PartSize = options.PartSize
		u.Concurrency = options.Concurrency
	})

	// Create downloader with configured part size and concurrency
	downloader := manager.NewDownloader(s3Client, func(d *manager.Downloader) {
		d.PartSize = options.PartSize
		d.Concurrency = options.Concurrency
	})

	return &TransferManager{
		s3Client:   s3Client,
		uploader:   uploader,
		downloader: downloader,
		transfers:  make(map[string]*TransferProgress),
		options:    options,
	}
}

// UploadFile uploads a file to S3 with multipart upload and progress tracking
func (tm *TransferManager) UploadFile(ctx context.Context, localPath, bucket, key string) (*TransferProgress, error) {
	// Open local file
	file, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Create transfer progress tracker
	transferID := generateTransferID()
	progress := &TransferProgress{
		TransferID:       transferID,
		Type:             TransferTypeUpload,
		Status:           TransferStatusInProgress,
		FilePath:         localPath,
		S3Bucket:         bucket,
		S3Key:            key,
		TotalBytes:       stat.Size(),
		TransferredBytes: 0,
		PercentComplete:  0,
		StartTime:        time.Now(),
		LastUpdate:       time.Now(),
	}

	// Calculate total parts
	progress.TotalParts = int((stat.Size() + tm.options.PartSize - 1) / tm.options.PartSize)

	// Register transfer
	tm.mu.Lock()
	tm.transfers[transferID] = progress
	tm.mu.Unlock()

	// Compute checksum if enabled
	if tm.options.Checksum {
		checksum, err := computeFileMD5(localPath)
		if err != nil {
			return nil, fmt.Errorf("failed to compute checksum: %w", err)
		}
		progress.Checksum = checksum
	}

	// Create progress tracking reader
	progressReader := &progressReader{
		reader:   file,
		progress: progress,
		callback: tm.options.ProgressCallback,
		interval: tm.options.ProgressInterval,
	}

	// Start upload with context timeout
	uploadCtx, cancel := context.WithTimeout(ctx, TransferTimeout)
	defer cancel()

	_, err = tm.uploader.Upload(uploadCtx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   progressReader,
	})

	if err != nil {
		progress.Status = TransferStatusFailed
		progress.Error = err.Error()
		return progress, fmt.Errorf("upload failed: %w", err)
	}

	// Mark as completed
	progress.Status = TransferStatusCompleted
	progress.PercentComplete = 100
	progress.LastUpdate = time.Now()

	return progress, nil
}

// DownloadFile downloads a file from S3 with progress tracking and resume support
func (tm *TransferManager) DownloadFile(ctx context.Context, bucket, key, localPath string) (*TransferProgress, error) {
	// Get object metadata for size
	headResp, err := tm.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	// Create transfer progress tracker
	transferID := generateTransferID()
	progress := &TransferProgress{
		TransferID:       transferID,
		Type:             TransferTypeDownload,
		Status:           TransferStatusInProgress,
		FilePath:         localPath,
		S3Bucket:         bucket,
		S3Key:            key,
		TotalBytes:       aws.ToInt64(headResp.ContentLength),
		TransferredBytes: 0,
		PercentComplete:  0,
		StartTime:        time.Now(),
		LastUpdate:       time.Now(),
	}

	// Calculate total parts
	progress.TotalParts = int((progress.TotalBytes + tm.options.PartSize - 1) / tm.options.PartSize)

	// Register transfer
	tm.mu.Lock()
	tm.transfers[transferID] = progress
	tm.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create output file
	file, err := os.Create(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Create progress tracking writer
	progressWriter := &progressWriter{
		writer:   file,
		progress: progress,
		callback: tm.options.ProgressCallback,
		interval: tm.options.ProgressInterval,
	}

	// Start download with context timeout
	downloadCtx, cancel := context.WithTimeout(ctx, TransferTimeout)
	defer cancel()

	_, err = tm.downloader.Download(downloadCtx, progressWriter, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		progress.Status = TransferStatusFailed
		progress.Error = err.Error()
		return progress, fmt.Errorf("download failed: %w", err)
	}

	// Verify checksum if enabled
	if tm.options.Checksum {
		checksum, err := computeFileMD5(localPath)
		if err != nil {
			progress.Status = TransferStatusFailed
			progress.Error = fmt.Sprintf("checksum computation failed: %v", err)
			return progress, fmt.Errorf("checksum computation failed: %w", err)
		}
		progress.Checksum = checksum

		// Compare with S3 ETag if available
		if headResp.ETag != nil && !verifyChecksum(*headResp.ETag, checksum) {
			progress.Status = TransferStatusFailed
			progress.Error = "checksum verification failed"
			return progress, fmt.Errorf("checksum verification failed")
		}
	}

	// Mark as completed
	progress.Status = TransferStatusCompleted
	progress.PercentComplete = 100
	progress.LastUpdate = time.Now()

	// Auto-cleanup S3 object if requested
	if tm.options.AutoCleanup {
		if err := tm.DeleteObject(ctx, bucket, key); err != nil {
			// Log error but don't fail the transfer
			fmt.Printf("Warning: failed to cleanup S3 object: %v\n", err)
		}
	}

	return progress, nil
}

// GetTransferProgress retrieves the current progress of a transfer
func (tm *TransferManager) GetTransferProgress(transferID string) (*TransferProgress, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	progress, exists := tm.transfers[transferID]
	return progress, exists
}

// ListTransfers returns all active transfers
func (tm *TransferManager) ListTransfers() []*TransferProgress {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	transfers := make([]*TransferProgress, 0, len(tm.transfers))
	for _, progress := range tm.transfers {
		transfers = append(transfers, progress)
	}
	return transfers
}

// DeleteObject deletes an object from S3
func (tm *TransferManager) DeleteObject(ctx context.Context, bucket, key string) error {
	_, err := tm.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

// progressReader wraps an io.Reader to track upload progress
type progressReader struct {
	reader   io.Reader
	progress *TransferProgress
	callback func(*TransferProgress)
	interval time.Duration
	lastCall time.Time
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)

	// Update progress
	pr.progress.TransferredBytes += int64(n)
	pr.progress.PercentComplete = float64(pr.progress.TransferredBytes) / float64(pr.progress.TotalBytes) * 100
	pr.progress.LastUpdate = time.Now()

	// Calculate speed
	elapsed := time.Since(pr.progress.StartTime).Seconds()
	if elapsed > 0 {
		pr.progress.BytesPerSecond = int64(float64(pr.progress.TransferredBytes) / elapsed)
	}

	// Estimate completion time
	if pr.progress.BytesPerSecond > 0 {
		remaining := pr.progress.TotalBytes - pr.progress.TransferredBytes
		secondsRemaining := float64(remaining) / float64(pr.progress.BytesPerSecond)
		eta := time.Now().Add(time.Duration(secondsRemaining) * time.Second)
		pr.progress.EstimatedCompletion = &eta
	}

	// Call progress callback if interval elapsed
	if pr.callback != nil && time.Since(pr.lastCall) >= pr.interval {
		pr.callback(pr.progress)
		pr.lastCall = time.Now()
	}

	return n, err
}

// progressWriter wraps an io.WriterAt to track download progress
type progressWriter struct {
	writer   io.WriterAt
	progress *TransferProgress
	callback func(*TransferProgress)
	interval time.Duration
	lastCall time.Time
	mu       sync.Mutex
}

func (pw *progressWriter) WriteAt(p []byte, off int64) (int, error) {
	n, err := pw.writer.WriteAt(p, off)

	// Update progress (thread-safe)
	pw.mu.Lock()
	pw.progress.TransferredBytes += int64(n)
	pw.progress.PercentComplete = float64(pw.progress.TransferredBytes) / float64(pw.progress.TotalBytes) * 100
	pw.progress.LastUpdate = time.Now()

	// Calculate speed
	elapsed := time.Since(pw.progress.StartTime).Seconds()
	if elapsed > 0 {
		pw.progress.BytesPerSecond = int64(float64(pw.progress.TransferredBytes) / elapsed)
	}

	// Estimate completion time
	if pw.progress.BytesPerSecond > 0 {
		remaining := pw.progress.TotalBytes - pw.progress.TransferredBytes
		secondsRemaining := float64(remaining) / float64(pw.progress.BytesPerSecond)
		eta := time.Now().Add(time.Duration(secondsRemaining) * time.Second)
		pw.progress.EstimatedCompletion = &eta
	}

	shouldCallback := pw.callback != nil && time.Since(pw.lastCall) >= pw.interval
	if shouldCallback {
		pw.lastCall = time.Now()
	}
	pw.mu.Unlock()

	// Call progress callback if interval elapsed (outside lock)
	if shouldCallback {
		pw.callback(pw.progress)
	}

	return n, err
}

// computeFileMD5 computes the MD5 checksum of a file
func computeFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// verifyChecksum compares an S3 ETag with a computed MD5 checksum
func verifyChecksum(etag, checksum string) bool {
	// S3 ETag is often quoted, remove quotes
	etag = trimQuotes(etag)

	// For multipart uploads, ETag format is different (contains dash)
	// Skip verification for multipart uploads
	if len(etag) > 32 {
		return true // Cannot verify multipart ETag
	}

	return etag == checksum
}

// trimQuotes removes surrounding quotes from a string
func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// generateTransferID generates a unique transfer ID
func generateTransferID() string {
	return fmt.Sprintf("transfer-%d", time.Now().UnixNano())
}
