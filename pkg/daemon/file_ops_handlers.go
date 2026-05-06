package daemon

// file_ops_handlers.go — SSM file transfer handlers for Prism daemon.
//
// Routes (added to handleInstanceSubOperation dispatch table):
//   GET  /api/v1/instances/{name}/files        listInstanceFiles   [?path=]
//   POST /api/v1/instances/{name}/files/push   pushFileToInstance
//   POST /api/v1/instances/{name}/files/pull   pullFileFromInstance
//
// Issue #30 / sub-issue 30a

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/scttfrdmn/prism/pkg/aws"
)

// ---- request / response types -----------------------------------------------

type filePushRequest struct {
	LocalPath  string `json:"local_path"`
	RemotePath string `json:"remote_path"`
}

type filePullRequest struct {
	RemotePath string `json:"remote_path"`
	LocalPath  string `json:"local_path"`
}

// ---- handlers ----------------------------------------------------------------

// handleInstanceFiles dispatches /instances/{name}/files and sub-paths.
// parts[1] == "files"; parts[2] (if present) is "push" or "pull".
func (s *Server) handleInstanceFiles(w http.ResponseWriter, r *http.Request, instanceName string) {
	// Determine sub-path from URL
	path := r.URL.Path
	base := "/api/v1/instances/" + instanceName + "/files"
	suffix := path[len(base):]

	switch {
	case suffix == "" || suffix == "/":
		if r.Method == http.MethodGet {
			s.handleListInstanceFiles(w, r, instanceName)
		} else {
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	case suffix == "/push":
		if r.Method == http.MethodPost {
			s.handlePushFile(w, r, instanceName)
		} else {
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	case suffix == "/pull":
		if r.Method == http.MethodPost {
			s.handlePullFile(w, r, instanceName)
		} else {
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	default:
		s.writeError(w, http.StatusNotFound, "Unknown files sub-path")
	}
}

func (s *Server) handleListInstanceFiles(w http.ResponseWriter, r *http.Request, instanceName string) {
	remotePath := r.URL.Query().Get("path")
	if remotePath == "" {
		remotePath = "/home"
	}
	// Normalize and validate path (#598)
	remotePath = filepath.Clean(remotePath) //nolint:semgrep // path traversal is prevented by the IsAbs check below and validateRemotePath (#598)
	if !filepath.IsAbs(remotePath) {
		s.writeError(w, http.StatusBadRequest, "Path must be absolute")
		return
	}

	s.withAWSManager(w, r, func(m *aws.Manager) error {
		entries, err := m.ListRemoteFiles(r.Context(), instanceName, remotePath)
		if err != nil {
			return err
		}
		return json.NewEncoder(w).Encode(entries)
	})
}

func (s *Server) handlePushFile(w http.ResponseWriter, r *http.Request, instanceName string) {
	var req filePushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.LocalPath == "" || req.RemotePath == "" {
		s.writeError(w, http.StatusBadRequest, "local_path and remote_path are required")
		return
	}

	s.withAWSManager(w, r, func(m *aws.Manager) error {
		result, err := m.PushFile(r.Context(), instanceName, req.LocalPath, req.RemotePath)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(result)
	})
}

func (s *Server) handlePullFile(w http.ResponseWriter, r *http.Request, instanceName string) {
	var req filePullRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.RemotePath == "" || req.LocalPath == "" {
		s.writeError(w, http.StatusBadRequest, "remote_path and local_path are required")
		return
	}

	s.withAWSManager(w, r, func(m *aws.Manager) error {
		result, err := m.PullFile(r.Context(), instanceName, req.RemotePath, req.LocalPath)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(result)
	})
}
