package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/scttfrdmn/prism/pkg/storage"
)

// TransferRequest represents a request to start a file transfer
type TransferRequest struct {
	// Type is the transfer operation type ("upload" or "download")
	Type string `json:"type"`

	// LocalPath is the local file path
	LocalPath string `json:"local_path"`

	// S3Bucket is the S3 bucket name
	S3Bucket string `json:"s3_bucket"`

	// S3Key is the S3 object key
	S3Key string `json:"s3_key"`

	// Options for transfer configuration (optional)
	Options *storage.TransferOptions `json:"options,omitempty"`
}

// TransferResponse represents the response from a transfer request
type TransferResponse struct {
	// TransferID is the unique identifier for the transfer
	TransferID string `json:"transfer_id"`

	// Status is the current transfer status
	Status storage.TransferStatus `json:"status"`

	// Progress contains detailed transfer progress
	Progress *storage.TransferProgress `json:"progress,omitempty"`

	// Error contains error message if transfer failed to start
	Error string `json:"error,omitempty"`
}

// handleStorageTransfer handles storage transfer collection operations
func (s *Server) handleStorageTransfer(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListTransfers(w, r)
	case http.MethodPost:
		s.handleStartTransfer(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleListTransfers lists all active transfers
func (s *Server) handleListTransfers(w http.ResponseWriter, r *http.Request) {
	// Get transfer manager
	transferMgr, err := s.getTransferManager(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get transfer manager: %v", err))
		return
	}

	// Get all transfers
	transfers := transferMgr.ListTransfers()

	_ = json.NewEncoder(w).Encode(transfers)
}

// handleStartTransfer starts a new file transfer (upload or download)
func (s *Server) handleStartTransfer(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.Type != string(storage.TransferTypeUpload) && req.Type != string(storage.TransferTypeDownload) {
		s.writeError(w, http.StatusBadRequest, "Invalid transfer type (must be 'upload' or 'download')")
		return
	}

	if req.LocalPath == "" {
		s.writeError(w, http.StatusBadRequest, "local_path is required")
		return
	}

	if req.S3Bucket == "" {
		s.writeError(w, http.StatusBadRequest, "s3_bucket is required")
		return
	}

	if req.S3Key == "" {
		s.writeError(w, http.StatusBadRequest, "s3_key is required")
		return
	}

	// Get transfer manager
	transferMgr, err := s.getTransferManager(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get transfer manager: %v", err))
		return
	}

	// Start transfer based on type
	ctx, cancel := context.WithTimeout(r.Context(), storage.TransferTimeout)
	defer cancel()

	var progress *storage.TransferProgress

	if req.Type == string(storage.TransferTypeUpload) {
		progress, err = transferMgr.UploadFile(ctx, req.LocalPath, req.S3Bucket, req.S3Key)
	} else {
		progress, err = transferMgr.DownloadFile(ctx, req.S3Bucket, req.S3Key, req.LocalPath)
	}

	if err != nil {
		resp := TransferResponse{
			Status: storage.TransferStatusFailed,
			Error:  err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Return transfer response
	resp := TransferResponse{
		TransferID: progress.TransferID,
		Status:     progress.Status,
		Progress:   progress,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// handleStorageTransferOperations handles operations on specific transfers
func (s *Server) handleStorageTransferOperations(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/v1/storage/transfer/"):]
	parts := splitPath(path)
	if len(parts) == 0 {
		s.writeError(w, http.StatusBadRequest, "Missing transfer ID")
		return
	}

	transferID := parts[0]

	switch r.Method {
	case http.MethodGet:
		s.handleGetTransferStatus(w, r, transferID)
	case http.MethodDelete:
		s.handleCancelTransfer(w, r, transferID)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGetTransferStatus retrieves the status of a specific transfer
func (s *Server) handleGetTransferStatus(w http.ResponseWriter, r *http.Request, transferID string) {
	// Get transfer manager
	transferMgr, err := s.getTransferManager(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get transfer manager: %v", err))
		return
	}

	// Get transfer progress
	progress, exists := transferMgr.GetTransferProgress(transferID)
	if !exists {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Transfer %s not found", transferID))
		return
	}

	_ = json.NewEncoder(w).Encode(progress)
}

// handleCancelTransfer cancels a transfer in progress
func (s *Server) handleCancelTransfer(w http.ResponseWriter, r *http.Request, transferID string) {
	// Get transfer manager
	transferMgr, err := s.getTransferManager(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get transfer manager: %v", err))
		return
	}

	// Get transfer progress
	progress, exists := transferMgr.GetTransferProgress(transferID)
	if !exists {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Transfer %s not found", transferID))
		return
	}

	// Update status to cancelled
	progress.Status = storage.TransferStatusCancelled
	progress.LastUpdate = time.Now()

	// TODO: Implement actual transfer cancellation
	// This requires modifying the TransferManager to support cancellation
	// For now, just update the status

	_ = json.NewEncoder(w).Encode(progress)
}

// getTransferManager gets or creates a transfer manager for the current AWS session
func (s *Server) getTransferManager(r *http.Request) (*storage.TransferManager, error) {
	// Create AWS manager from request
	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS manager: %w", err)
	}

	// Create S3 client from AWS manager's config
	s3Client, err := awsManager.CreateS3Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Create transfer manager with default options
	options := storage.DefaultTransferOptions()

	return storage.NewTransferManager(s3Client, options), nil
}
