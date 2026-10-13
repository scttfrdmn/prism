// Package course — audit log implementation for academic integrity tracking.
package course

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditAction categorises a course audit event.
type AuditAction string

const (
	AuditActionEnroll           AuditAction = "member.enroll"
	AuditActionUnenroll         AuditAction = "member.unenroll"
	AuditActionBudgetSpend      AuditAction = "budget.spend"
	AuditActionBudgetDistribute AuditAction = "budget.distribute"
	AuditActionTADebug          AuditAction = "ta.debug"
	AuditActionTAReset          AuditAction = "ta.reset"
	AuditActionInstanceLaunch   AuditAction = "instance.launch"
	AuditActionInstanceStop     AuditAction = "instance.stop"
	AuditActionCourseClose      AuditAction = "course.close"
	AuditActionCourseArchive    AuditAction = "course.archive"
	AuditActionTAAccessGrant    AuditAction = "ta.access.grant"
	AuditActionTAAccessRevoke   AuditAction = "ta.access.revoke"
	AuditActionTASSHConnect     AuditAction = "ta.ssh.connect"
	AuditActionTASSHDisconnect  AuditAction = "ta.ssh.disconnect"
	AuditActionMaterialsCreate  AuditAction = "materials.create"
	AuditActionMaterialsMount   AuditAction = "materials.mount"
	AuditActionMaterialsUpload  AuditAction = "materials.upload"
)

// AuditEntry is one line in the course audit JSONL log.
type AuditEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	CourseID  string                 `json:"course_id"`
	Actor     string                 `json:"actor"` // who performed the action
	Action    AuditAction            `json:"action"`
	Target    string                 `json:"target,omitempty"` // student ID, instance name, etc.
	Detail    map[string]interface{} `json:"detail,omitempty"`
}

// AuditLog manages the append-only JSONL audit log for a single course.
// The log lives at <stateDir>/<courseID>.jsonl.
type AuditLog struct {
	path string
	mu   sync.Mutex
}

// NewAuditLog returns an AuditLog for the given course. stateDir is the
// directory that holds all course audit files (e.g. ~/.prism/course-audits/).
func NewAuditLog(courseID, stateDir string) *AuditLog {
	return &AuditLog{
		path: filepath.Join(stateDir, fmt.Sprintf("%s.jsonl", courseID)),
	}
}

// Append writes one entry to the log. It opens the file, writes one JSON line,
// then closes it immediately — no buffering — so entries are never lost on crash.
// It is safe for concurrent use.
func (l *AuditLog) Append(entry AuditEntry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("audit marshal: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("audit open: %w", err)
	}
	if _, err := fmt.Fprintf(f, "%s\n", data); err != nil {
		_ = f.Close() // the write error is the one worth reporting
		return fmt.Errorf("audit write: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("audit close: %w", err)
	}
	return nil
}

// Query returns audit entries matching the optional filters.
// studentID == "" matches all students.
// since.IsZero() returns all entries.
// limit <= 0 means no limit.
// It reads the entire log file into memory; files are bounded by course lifetime.
func (l *AuditLog) Query(studentID string, since time.Time, limit int) ([]AuditEntry, error) {
	f, err := os.Open(l.path)
	if os.IsNotExist(err) {
		return nil, nil // no log yet is fine
	}
	if err != nil {
		return nil, fmt.Errorf("audit open: %w", err)
	}
	defer func() { _ = f.Close() }() // read-only; close error carries no data-loss risk

	var results []AuditEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry AuditEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // skip malformed lines
		}
		if studentID != "" && entry.Target != studentID && entry.Actor != studentID {
			continue
		}
		if !since.IsZero() && entry.Timestamp.Before(since) {
			continue
		}
		results = append(results, entry)
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return results, fmt.Errorf("audit scan: %w", err)
	}
	return results, nil
}
