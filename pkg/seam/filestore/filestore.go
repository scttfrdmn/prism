// Package filestore is the file-backed seam.Store: the desktop-standalone implementation.
//
// It keeps Prism's domain truths but drops the laptop-locality artifacts (design §5.3): no
// PID-file singleton, no single-~/.prism-tree assumption. Records are partitioned by Scope, so one
// process can hold many tenants. Writes are atomic per record (temp file + rename) — the one
// local-FS truth worth keeping.
//
// Layout: <root>/<scope-key>/<id>.json, where scope-key is Scope.Key() with '/' replaced by a
// filesystem-safe separator. Identical to prp's filestore so a desktop tree and the cloud store
// hold the same record shapes.
package filestore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/scttfrdmn/prism/pkg/seam"
)

// Store is a file-backed seam.Store[T]. Construct with New.
type Store[T any] struct {
	root string
	mu   sync.RWMutex // guards against torn concurrent access within this process only
}

// New returns a file-backed store rooted at dir. dir is created on first write.
func New[T any](dir string) *Store[T] {
	return &Store[T]{root: dir}
}

// scopeDir maps a Scope to its directory. Scope.Key() uses '/' as its field separator; we swap
// that for "__" so the multi-field key becomes one safe path segment we can list.
func (s *Store[T]) scopeDir(scope seam.Scope) string {
	seg := strings.ReplaceAll(scope.Key(), "/", "__")
	return filepath.Join(s.root, seg)
}

func (s *Store[T]) path(scope seam.Scope, id string) string {
	return filepath.Join(s.scopeDir(scope), id+".json")
}

// Get returns the record at (scope, id), or seam.ErrNotFound if none exists.
func (s *Store[T]) Get(_ context.Context, scope seam.Scope, id string) (T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var zero T
	data, err := os.ReadFile(s.path(scope, id))
	if err != nil {
		if os.IsNotExist(err) {
			return zero, seam.ErrNotFound
		}
		return zero, fmt.Errorf("filestore get: %w", err)
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return zero, fmt.Errorf("filestore get unmarshal: %w", err)
	}
	return v, nil
}

// List returns all records in scope (empty slice when none), or an error on a read failure.
func (s *Store[T]) List(_ context.Context, scope seam.Scope) ([]T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dir := s.scopeDir(scope)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []T{}, nil // empty scope is empty, not an error
		}
		return nil, fmt.Errorf("filestore list: %w", err)
	}
	out := make([]T, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		// #nosec G304 -- path is composed from our own root + a directory entry we just listed,
		// not from external input; this is the store reading its own files.
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("filestore list read %s: %w", e.Name(), err)
		}
		var v T
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("filestore list unmarshal %s: %w", e.Name(), err)
		}
		out = append(out, v)
	}
	return out, nil
}

// Put writes v at (scope, id) atomically (temp file + rename), creating the scope dir as needed.
func (s *Store[T]) Put(_ context.Context, scope seam.Scope, id string, v T) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := s.scopeDir(scope)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("filestore put mkdir: %w", err)
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("filestore put marshal: %w", err)
	}
	// Atomic write: temp file in the same dir, then rename over the target.
	final := s.path(scope, id)
	tmp, err := os.CreateTemp(dir, "."+id+".*.tmp")
	if err != nil {
		return fmt.Errorf("filestore put tempfile: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("filestore put write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("filestore put close: %w", err)
	}
	if err := os.Rename(tmpName, final); err != nil {
		cleanup()
		return fmt.Errorf("filestore put rename: %w", err)
	}
	return nil
}

// Delete removes the record at (scope, id), or returns seam.ErrNotFound if none exists.
func (s *Store[T]) Delete(_ context.Context, scope seam.Scope, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(s.path(scope, id))
	if err != nil {
		if os.IsNotExist(err) {
			return seam.ErrNotFound
		}
		return fmt.Errorf("filestore delete: %w", err)
	}
	return nil
}

// compile-time check
var _ seam.Store[int] = (*Store[int])(nil)
