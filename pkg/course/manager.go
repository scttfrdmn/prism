// Package course provides course and education management functionality for Prism.
//
// This package implements course-based resource organization, template policy
// enforcement, and per-student budget tracking that enable instructors to
// manage classroom computing environments at scale.
package course

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/scttfrdmn/prism/pkg/types"
)

// Manager handles course lifecycle, member enrollment, template policies, and budgets
type Manager struct {
	coursesPath string
	auditDir    string // directory for per-course JSONL audit logs
	mutex       sync.RWMutex
	courses     map[string]*types.Course
}

// NewManager creates a new course manager
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".prism")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	auditDir := filepath.Join(stateDir, "course-audits")
	if err := os.MkdirAll(auditDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit directory: %w", err)
	}

	manager := &Manager{
		coursesPath: filepath.Join(stateDir, "courses.json"),
		auditDir:    auditDir,
		courses:     make(map[string]*types.Course),
	}

	if err := manager.loadCourses(); err != nil {
		return nil, fmt.Errorf("failed to load courses: %w", err)
	}

	return manager, nil
}

// --- Lifecycle ---

// CreateCourse creates a new course
func (m *Manager) CreateCourse(ctx context.Context, req *CreateCourseRequest) (*types.Course, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid course request: %w", err)
	}

	// Check for duplicate (same code + semester)
	for _, c := range m.courses {
		if strings.EqualFold(c.Code, req.Code) && strings.EqualFold(c.Semester, req.Semester) {
			return nil, ErrDuplicateCourse
		}
	}

	graceDays := req.GracePeriodDays
	if graceDays <= 0 {
		graceDays = 7
	}

	// Determine initial status based on current date
	now := time.Now()
	status := types.CourseStatusPending
	if now.After(req.SemesterStart) && now.Before(req.SemesterEnd) {
		status = types.CourseStatusActive
	}

	course := &types.Course{
		ID:                    uuid.New().String(),
		Code:                  req.Code,
		Title:                 req.Title,
		Department:            req.Department,
		Semester:              req.Semester,
		SemesterStart:         req.SemesterStart,
		SemesterEnd:           req.SemesterEnd,
		GracePeriodDays:       graceDays,
		Owner:                 req.Owner,
		Members:               []types.ClassMember{},
		ApprovedTemplates:     req.ApprovedTemplates,
		PerStudentBudget:      req.PerStudentBudget,
		TotalBudget:           req.TotalBudget,
		Tags:                  req.Tags,
		DefaultTemplate:       req.DefaultTemplate,
		AutoProvisionOnEnroll: req.AutoProvisionOnEnroll,
		CreatedAt:             now,
		UpdatedAt:             now,
		Status:                status,
	}

	// Enroll the owner as instructor
	if req.Owner != "" {
		expiresAt := req.SemesterEnd.AddDate(0, 0, graceDays)
		course.Members = append(course.Members, types.ClassMember{
			UserID:    req.Owner,
			Role:      types.ClassRoleInstructor,
			AddedAt:   now,
			AddedBy:   req.Owner,
			ExpiresAt: &expiresAt,
		})
	}

	m.courses[course.ID] = course
	if err := m.saveCourses(); err != nil {
		delete(m.courses, course.ID)
		return nil, fmt.Errorf("failed to save course: %w", err)
	}

	copy := *course
	return &copy, nil
}

// GetCourse retrieves a course by ID
func (m *Manager) GetCourse(ctx context.Context, courseID string) (*types.Course, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	c, ok := m.courses[courseID]
	if !ok {
		return nil, ErrCourseNotFound
	}
	copy := *c
	return &copy, nil
}

// GetCourseByCode retrieves a course by its code and semester
func (m *Manager) GetCourseByCode(ctx context.Context, code, semester string) (*types.Course, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, c := range m.courses {
		if strings.EqualFold(c.Code, code) && strings.EqualFold(c.Semester, semester) {
			copy := *c
			return &copy, nil
		}
	}
	return nil, ErrCourseNotFound
}

// ListCourses retrieves all courses matching the optional filter
func (m *Manager) ListCourses(ctx context.Context, filter *CourseFilter) ([]*types.Course, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var results []*types.Course
	for _, c := range m.courses {
		if filter != nil && !filter.Matches(c) {
			continue
		}
		copy := *c
		results = append(results, &copy)
	}
	return results, nil
}

// UpdateCourse applies a partial update to an existing course
func (m *Manager) UpdateCourse(ctx context.Context, courseID string, req *UpdateCourseRequest) (*types.Course, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c, ok := m.courses[courseID]
	if !ok {
		return nil, ErrCourseNotFound
	}

	if req.Title != nil {
		c.Title = *req.Title
	}
	if req.Department != nil {
		c.Department = *req.Department
	}
	if req.SemesterEnd != nil {
		c.SemesterEnd = *req.SemesterEnd
	}
	if req.GracePeriodDays != nil {
		c.GracePeriodDays = *req.GracePeriodDays
	}
	if req.ApprovedTemplates != nil {
		c.ApprovedTemplates = req.ApprovedTemplates
	}
	if req.PerStudentBudget != nil {
		c.PerStudentBudget = *req.PerStudentBudget
	}
	if req.TotalBudget != nil {
		c.TotalBudget = *req.TotalBudget
	}
	if req.Tags != nil {
		c.Tags = req.Tags
	}
	if req.Status != nil {
		c.Status = *req.Status
	}
	if req.DefaultTemplate != nil {
		c.DefaultTemplate = *req.DefaultTemplate
	}
	if req.AutoProvisionOnEnroll != nil {
		c.AutoProvisionOnEnroll = *req.AutoProvisionOnEnroll
	}
	c.UpdatedAt = time.Now()

	if err := m.saveCourses(); err != nil {
		return nil, fmt.Errorf("failed to save course update: %w", err)
	}

	copy := *c
	return &copy, nil
}

// CloseCourse transitions a course to the "closed" state
func (m *Manager) CloseCourse(ctx context.Context, courseID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c, ok := m.courses[courseID]
	if !ok {
		return ErrCourseNotFound
	}

	c.Status = types.CourseStatusClosed
	c.UpdatedAt = time.Now()

	if err := m.saveCourses(); err != nil {
		return err
	}
	caller := CallerFrom(ctx)
	m.appendAudit(courseID, AuditEntry{
		CourseID: courseID,
		Actor:    caller,
		Action:   AuditActionCourseClose,
	})
	return nil
}

// DeleteCourse permanently removes a course record
func (m *Manager) DeleteCourse(ctx context.Context, courseID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.courses[courseID]; !ok {
		return ErrCourseNotFound
	}

	delete(m.courses, courseID)
	return m.saveCourses()
}

// --- Member Management ---

// EnrollMember adds a single member to the course
func (m *Manager) EnrollMember(ctx context.Context, courseID string, req *EnrollRequest) (*types.ClassMember, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid enroll request: %w", err)
	}

	c, ok := m.courses[courseID]
	if !ok {
		return nil, ErrCourseNotFound
	}
	if c.Status == types.CourseStatusArchived {
		return nil, ErrCourseClosed
	}

	role := req.Role
	if role == "" {
		role = types.ClassRoleStudent
	}

	// Determine effective user ID
	userID := req.UserID
	if userID == "" {
		userID = req.Email
	}

	// Check for duplicate (only compare email when both sides are non-empty)
	for _, mb := range c.Members {
		if mb.UserID == userID {
			return nil, ErrAlreadyEnrolled
		}
		if mb.Email != "" && req.Email != "" && mb.Email == req.Email {
			return nil, ErrAlreadyEnrolled
		}
	}

	addedBy := CallerFrom(ctx)
	if addedBy == "" {
		addedBy = c.Owner
	}

	expiresAt := c.SemesterEnd.AddDate(0, 0, c.GracePeriodDays)
	member := types.ClassMember{
		UserID:      userID,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Role:        role,
		AddedAt:     time.Now(),
		AddedBy:     addedBy,
		ExpiresAt:   &expiresAt,
		BudgetLimit: req.BudgetLimit,
	}

	c.Members = append(c.Members, member)
	c.UpdatedAt = time.Now()

	if err := m.saveCourses(); err != nil {
		// roll back in-memory change
		c.Members = c.Members[:len(c.Members)-1]
		return nil, fmt.Errorf("failed to save enrollment: %w", err)
	}

	memberCopy := member
	m.appendAudit(courseID, AuditEntry{
		CourseID: courseID,
		Actor:    addedBy,
		Action:   AuditActionEnroll,
		Target:   userID,
		Detail:   map[string]interface{}{"role": string(role), "email": req.Email},
	})
	return &memberCopy, nil
}

// enrollMemberInternal is the lock-free version used by BulkEnroll (caller holds write lock)
func (m *Manager) enrollMemberInternal(c *types.Course, req *EnrollRequest, defaultRole types.ClassRole, addedBy string) error {
	role := req.Role
	if role == "" {
		role = defaultRole
	}
	if role == "" {
		role = types.ClassRoleStudent
	}

	userID := req.UserID
	if userID == "" {
		userID = req.Email
	}

	for _, mb := range c.Members {
		if mb.UserID == userID || (req.Email != "" && mb.Email == req.Email) {
			return ErrAlreadyEnrolled
		}
	}

	expiresAt := c.SemesterEnd.AddDate(0, 0, c.GracePeriodDays)
	c.Members = append(c.Members, types.ClassMember{
		UserID:      userID,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Role:        role,
		AddedAt:     time.Now(),
		AddedBy:     addedBy,
		ExpiresAt:   &expiresAt,
		BudgetLimit: req.BudgetLimit,
	})
	return nil
}

// BulkEnroll adds multiple members in one operation.
// Partial failures skip the bad row and continue; row errors are returned alongside the count.
func (m *Manager) BulkEnroll(ctx context.Context, courseID string, req *BulkEnrollRequest) (int, []error, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c, ok := m.courses[courseID]
	if !ok {
		return 0, nil, ErrCourseNotFound
	}
	if c.Status == types.CourseStatusArchived {
		return 0, nil, ErrCourseClosed
	}

	callerID := CallerFrom(ctx)
	if callerID == "" {
		callerID = c.Owner
	}

	defaultRole := req.DefaultRole
	if defaultRole == "" {
		defaultRole = types.ClassRoleStudent
	}

	var rowErrors []error
	enrolled := 0

	for i, member := range req.Members {
		if err := member.Validate(); err != nil {
			rowErrors = append(rowErrors, fmt.Errorf("row %d: %w", i, err))
			continue
		}
		if err := m.enrollMemberInternal(c, &member, defaultRole, callerID); err != nil {
			rowErrors = append(rowErrors, fmt.Errorf("row %d (%s): %w", i, member.Email, err))
			continue
		}
		enrolled++
	}

	if enrolled > 0 {
		c.UpdatedAt = time.Now()
		if err := m.saveCourses(); err != nil {
			return 0, rowErrors, fmt.Errorf("failed to save bulk enrollment: %w", err)
		}
	}

	return enrolled, rowErrors, nil
}

// UpdateMember changes a member's role or budget limit
func (m *Manager) UpdateMember(ctx context.Context, courseID, userID string, role types.ClassRole, budgetLimit float64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c, ok := m.courses[courseID]
	if !ok {
		return ErrCourseNotFound
	}

	for i, mb := range c.Members {
		if mb.UserID == userID || mb.Email == userID {
			if role != "" {
				c.Members[i].Role = role
			}
			if budgetLimit >= 0 {
				c.Members[i].BudgetLimit = budgetLimit
			}
			c.UpdatedAt = time.Now()
			return m.saveCourses()
		}
	}
	return ErrMemberNotFound
}

// UnenrollMember removes a member from the course
func (m *Manager) UnenrollMember(ctx context.Context, courseID, userID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c, ok := m.courses[courseID]
	if !ok {
		return ErrCourseNotFound
	}

	for i, mb := range c.Members {
		if mb.UserID == userID || mb.Email == userID {
			c.Members = append(c.Members[:i], c.Members[i+1:]...)
			c.UpdatedAt = time.Now()
			if err := m.saveCourses(); err != nil {
				return err
			}
			caller := CallerFrom(ctx)
			m.appendAudit(courseID, AuditEntry{
				CourseID: courseID,
				Actor:    caller,
				Action:   AuditActionUnenroll,
				Target:   mb.UserID,
			})
			return nil
		}
	}
	return ErrMemberNotFound
}

// ListMembers returns enrolled members, optionally filtered by role
func (m *Manager) ListMembers(ctx context.Context, courseID string, role *types.ClassRole) ([]types.ClassMember, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	c, ok := m.courses[courseID]
	if !ok {
		return nil, ErrCourseNotFound
	}

	if role == nil {
		out := make([]types.ClassMember, len(c.Members))
		copy(out, c.Members)
		return out, nil
	}

	var result []types.ClassMember
	for _, mb := range c.Members {
		if mb.Role == *role {
			result = append(result, mb)
		}
	}
	return result, nil
}

// GetMember returns a single member by user ID or email
func (m *Manager) GetMember(ctx context.Context, courseID, userID string) (*types.ClassMember, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	c, ok := m.courses[courseID]
	if !ok {
		return nil, ErrCourseNotFound
	}

	for _, mb := range c.Members {
		if mb.UserID == userID || mb.Email == userID {
			copy := mb
			return &copy, nil
		}
	}
	return nil, ErrMemberNotFound
}

// --- Template Whitelist (#46) ---

// AddApprovedTemplate adds a template slug to the course whitelist
func (m *Manager) AddApprovedTemplate(ctx context.Context, courseID, templateSlug string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c, ok := m.courses[courseID]
	if !ok {
		return ErrCourseNotFound
	}

	for _, t := range c.ApprovedTemplates {
		if t == templateSlug {
			return nil // already present
		}
	}

	c.ApprovedTemplates = append(c.ApprovedTemplates, templateSlug)
	c.UpdatedAt = time.Now()
	return m.saveCourses()
}

// RemoveApprovedTemplate removes a template slug from the course whitelist
func (m *Manager) RemoveApprovedTemplate(ctx context.Context, courseID, templateSlug string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c, ok := m.courses[courseID]
	if !ok {
		return ErrCourseNotFound
	}

	for i, t := range c.ApprovedTemplates {
		if t == templateSlug {
			c.ApprovedTemplates = append(c.ApprovedTemplates[:i], c.ApprovedTemplates[i+1:]...)
			c.UpdatedAt = time.Now()
			return m.saveCourses()
		}
	}
	return nil // not present; no-op
}

// IsTemplateApproved returns true if the template is allowed for this course.
// Returns true when the course has no whitelist (empty ApprovedTemplates = unrestricted).
func (m *Manager) IsTemplateApproved(courseID, templateSlug string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	c, ok := m.courses[courseID]
	if !ok {
		return true // unknown course: don't block
	}

	if len(c.ApprovedTemplates) == 0 {
		return true // no restriction
	}

	for _, t := range c.ApprovedTemplates {
		if t == templateSlug {
			return true
		}
	}
	return false
}

// --- Budget (#47) ---

// DistributeBudget sets per-student budget allocations for all currently enrolled students
func (m *Manager) DistributeBudget(ctx context.Context, courseID string, amountPerStudent float64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c, ok := m.courses[courseID]
	if !ok {
		return ErrCourseNotFound
	}

	c.PerStudentBudget = amountPerStudent
	for i, mb := range c.Members {
		if mb.Role == types.ClassRoleStudent && mb.BudgetLimit == 0 {
			c.Members[i].BudgetLimit = amountPerStudent
		}
	}
	c.UpdatedAt = time.Now()
	if err := m.saveCourses(); err != nil {
		return err
	}
	caller := CallerFrom(ctx)
	m.appendAudit(courseID, AuditEntry{
		CourseID: courseID,
		Actor:    caller,
		Action:   AuditActionBudgetDistribute,
		Detail:   map[string]interface{}{"amount_per_student": amountPerStudent},
	})
	return nil
}

// RecordSpend increments the spend counter for a member
func (m *Manager) RecordSpend(ctx context.Context, courseID, userID string, amount float64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c, ok := m.courses[courseID]
	if !ok {
		return ErrCourseNotFound
	}

	for i, mb := range c.Members {
		if mb.UserID == userID || mb.Email == userID {
			c.Members[i].BudgetSpent += amount
			c.UpdatedAt = time.Now()
			if err := m.saveCourses(); err != nil {
				return err
			}
			m.appendAudit(courseID, AuditEntry{
				CourseID: courseID,
				Actor:    userID,
				Action:   AuditActionBudgetSpend,
				Target:   userID,
				Detail:   map[string]interface{}{"amount": amount, "total_spent": c.Members[i].BudgetSpent},
			})
			return nil
		}
	}
	return ErrMemberNotFound
}

// GetBudgetSummary returns budget usage across the entire course
func (m *Manager) GetBudgetSummary(ctx context.Context, courseID string) (*CourseBudgetSummary, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	c, ok := m.courses[courseID]
	if !ok {
		return nil, ErrCourseNotFound
	}

	summary := &CourseBudgetSummary{
		TotalBudget:       c.TotalBudget,
		PerStudentDefault: c.PerStudentBudget,
	}

	for _, mb := range c.Members {
		if mb.Role != types.ClassRoleStudent {
			continue
		}
		limit := mb.BudgetLimit
		if limit == 0 {
			limit = c.PerStudentBudget
		}
		summary.TotalSpent += mb.BudgetSpent
		summary.Students = append(summary.Students, StudentBudgetInfo{
			UserID:      mb.UserID,
			DisplayName: mb.DisplayName,
			Email:       mb.Email,
			BudgetLimit: limit,
			BudgetSpent: mb.BudgetSpent,
			Remaining:   limit - mb.BudgetSpent,
		})
	}

	return summary, nil
}

// --- Roster CSV Import (#47) ---

// ImportRosterCSV parses a CSV reader and bulk-enrolls students.
// CSV columns: email, display_name, budget (optional), role (optional)
// Header row is required. Malformed rows are skipped; errors returned per row.
func (m *Manager) ImportRosterCSV(ctx context.Context, courseID string, rows []RosterRow, sendInvites bool) (int, []error, error) {
	reqs := make([]EnrollRequest, 0, len(rows))
	for _, r := range rows {
		role := types.ClassRole(r.Role)
		if role == "" {
			role = types.ClassRoleStudent
		}
		reqs = append(reqs, EnrollRequest{
			Email:       r.Email,
			DisplayName: r.DisplayName,
			Role:        role,
			BudgetLimit: r.Budget,
		})
	}

	return m.BulkEnroll(ctx, courseID, &BulkEnrollRequest{
		Members:     reqs,
		DefaultRole: types.ClassRoleStudent,
		SendInvites: sendInvites,
	})
}

// ParseRosterCSV parses CSV bytes into RosterRow slice.
// Expected columns (with header): email, display_name, budget, role
func ParseRosterCSV(data []byte) ([]RosterRow, error) {
	r := csv.NewReader(strings.NewReader(string(data)))
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must have a header row and at least one data row")
	}

	// Build column index from header
	header := make(map[string]int)
	for i, col := range records[0] {
		header[strings.ToLower(strings.TrimSpace(col))] = i
	}
	emailIdx, ok := header["email"]
	if !ok {
		return nil, fmt.Errorf("CSV must have an 'email' column")
	}

	var rows []RosterRow
	for _, rec := range records[1:] {
		if len(rec) == 0 {
			continue
		}
		email := ""
		if emailIdx < len(rec) {
			email = strings.TrimSpace(rec[emailIdx])
		}
		if email == "" {
			continue
		}

		row := RosterRow{Email: email}

		if idx, ok := header["display_name"]; ok && idx < len(rec) {
			row.DisplayName = strings.TrimSpace(rec[idx])
		}
		if idx, ok := header["name"]; ok && row.DisplayName == "" && idx < len(rec) {
			row.DisplayName = strings.TrimSpace(rec[idx])
		}
		if idx, ok := header["budget"]; ok && idx < len(rec) {
			if v, err := strconv.ParseFloat(strings.TrimSpace(rec[idx]), 64); err == nil {
				row.Budget = v
			}
		}
		if idx, ok := header["role"]; ok && idx < len(rec) {
			row.Role = strings.TrimSpace(rec[idx])
		}

		rows = append(rows, row)
	}

	return rows, nil
}

// --- TA Operations (#48, #49) ---

// GetStudentDebugInfo returns read-only debug information for a student in a course.
// The caller (taID) must be a TA or instructor in the course.
func (m *Manager) GetStudentDebugInfo(ctx context.Context, courseID, taID, studentID string) (*TADebugInfo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	c, ok := m.courses[courseID]
	if !ok {
		return nil, ErrCourseNotFound
	}

	if !m.isTAOrInstructorLocked(c, taID) {
		return nil, ErrNotAuthorized
	}

	var studentMember *types.ClassMember
	for i, mb := range c.Members {
		if mb.UserID == studentID || mb.Email == studentID {
			studentMember = &c.Members[i]
			break
		}
	}
	if studentMember == nil {
		return nil, ErrMemberNotFound
	}

	limit := studentMember.BudgetLimit
	if limit == 0 {
		limit = c.PerStudentBudget
	}

	info := &TADebugInfo{
		CourseID:     courseID,
		StudentID:    studentMember.UserID,
		StudentEmail: studentMember.Email,
		Instances:    nil, // populated by the handler using the AWS manager
		BudgetSpent:  studentMember.BudgetSpent,
		BudgetLimit:  limit,
	}
	m.appendAudit(courseID, AuditEntry{
		CourseID: courseID,
		Actor:    taID,
		Action:   AuditActionTADebug,
		Target:   studentMember.UserID,
	})
	return info, nil
}

// ResetStudentInstance records a TA reset request in the course audit trail.
// The actual AWS snapshot + re-provision is handled by the daemon handler (requires awsManager).
func (m *Manager) ResetStudentInstance(ctx context.Context, courseID, taID string, req *TAResetRequest) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	c, ok := m.courses[courseID]
	if !ok {
		return ErrCourseNotFound
	}

	if !m.isTAOrInstructorLocked(c, taID) {
		return ErrNotAuthorized
	}

	// Verify student is enrolled
	for _, mb := range c.Members {
		if mb.UserID == req.StudentID || mb.Email == req.StudentID {
			m.appendAudit(courseID, AuditEntry{
				CourseID: courseID,
				Actor:    taID,
				Action:   AuditActionTAReset,
				Target:   mb.UserID,
				Detail:   map[string]interface{}{"reason": req.Reason},
			})
			return nil // authorized; actual reset done by daemon handler
		}
	}
	return ErrMemberNotFound
}

// --- TA Access Management (#48, #160) ---

// GrantTAAccess enrolls taEmail as a TA in the course and records a dedicated audit event.
// Idempotent: if the user is already enrolled as a TA, returns nil without error.
func (m *Manager) GrantTAAccess(ctx context.Context, courseID, taEmail, displayName string) (*types.ClassMember, error) {
	req := &EnrollRequest{
		Email:       taEmail,
		DisplayName: displayName,
		Role:        types.ClassRoleTA,
	}
	member, err := m.EnrollMember(ctx, courseID, req)
	if err == ErrAlreadyEnrolled {
		// Already a TA: look up and return existing member
		m.mutex.RLock()
		c, ok := m.courses[courseID]
		m.mutex.RUnlock()
		if !ok {
			return nil, ErrCourseNotFound
		}
		for i, mb := range c.Members {
			if mb.Email == taEmail || mb.UserID == taEmail {
				if mb.Role != types.ClassRoleTA {
					break // enrolled but not as TA; fall through to return nil
				}
				return &c.Members[i], nil
			}
		}
		return nil, ErrAlreadyEnrolled
	}
	if err != nil {
		return nil, err
	}
	caller := CallerFrom(ctx)
	m.appendAudit(courseID, AuditEntry{
		CourseID: courseID,
		Actor:    caller,
		Action:   AuditActionTAAccessGrant,
		Target:   taEmail,
		Detail:   map[string]interface{}{"display_name": displayName},
	})
	return member, nil
}

// RevokeTAAccess removes a TA from the course and records the revocation in the audit log.
// Returns ErrMemberNotFound if the user is not enrolled.
func (m *Manager) RevokeTAAccess(ctx context.Context, courseID, taEmail string) error {
	// Find the TA's user ID first
	m.mutex.RLock()
	c, ok := m.courses[courseID]
	m.mutex.RUnlock()
	if !ok {
		return ErrCourseNotFound
	}

	var taUserID string
	for _, mb := range c.Members {
		if (mb.Email == taEmail || mb.UserID == taEmail) && mb.Role == types.ClassRoleTA {
			taUserID = mb.UserID
			if taUserID == "" {
				taUserID = mb.Email
			}
			break
		}
	}
	if taUserID == "" {
		return ErrMemberNotFound
	}

	if err := m.UnenrollMember(ctx, courseID, taUserID); err != nil {
		return err
	}
	caller := CallerFrom(ctx)
	m.appendAudit(courseID, AuditEntry{
		CourseID: courseID,
		Actor:    caller,
		Action:   AuditActionTAAccessRevoke,
		Target:   taEmail,
	})
	return nil
}

// ListTAAccess returns all members enrolled with role=ta in the course.
func (m *Manager) ListTAAccess(ctx context.Context, courseID string) ([]types.ClassMember, error) {
	taRole := types.ClassRoleTA
	return m.ListMembers(ctx, courseID, &taRole)
}

// LogTASSHConnect records a TA SSH access initiation in the course audit log.
func (m *Manager) LogTASSHConnect(ctx context.Context, courseID string, entry TAAccessEntry) error {
	m.mutex.RLock()
	c, ok := m.courses[courseID]
	m.mutex.RUnlock()
	if !ok {
		return ErrCourseNotFound
	}

	if !m.isTAOrInstructorLocked(c, entry.TAID) {
		return ErrNotAuthorized
	}

	m.appendAudit(courseID, AuditEntry{
		CourseID: courseID,
		Actor:    entry.TAID,
		Action:   AuditActionTASSHConnect,
		Target:   entry.StudentID,
		Detail: map[string]interface{}{
			"reason":      entry.Reason,
			"instance_id": entry.InstanceID,
			"public_ip":   entry.PublicIP,
		},
	})
	return nil
}

// CheckTemplateAllowed returns ErrNotAuthorized if the template is not in this course's
// whitelist. Returns nil when the course has no whitelist or the course is unknown.
func (m *Manager) CheckTemplateAllowed(courseID, templateSlug string) error {
	if !m.IsTemplateApproved(courseID, templateSlug) {
		return fmt.Errorf("%w: template %q is not approved for course %s", ErrNotAuthorized, templateSlug, courseID)
	}
	return nil
}

// --- Shared Course Materials (#167) ---

// SetCourseMaterials persists the shared EFS metadata on the course record.
// Called by the daemon handler after the AWS EFS filesystem has been created.
func (m *Manager) SetCourseMaterials(ctx context.Context, courseID, efsID, mountPath string, sizeGB int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c, ok := m.courses[courseID]
	if !ok {
		return ErrCourseNotFound
	}

	c.SharedMaterialsEFSID = efsID
	c.SharedMaterialsMountPath = mountPath
	c.SharedMaterialsSizeGB = sizeGB
	c.UpdatedAt = time.Now()
	if err := m.saveCourses(); err != nil {
		return err
	}

	caller := CallerFrom(ctx)
	m.appendAudit(courseID, AuditEntry{
		CourseID: courseID,
		Actor:    caller,
		Action:   AuditActionMaterialsCreate,
		Detail: map[string]interface{}{
			"efs_id":     efsID,
			"mount_path": mountPath,
			"size_gb":    sizeGB,
		},
	})
	return nil
}

// GetCourseMaterials returns the SharedMaterialsVolume metadata for a course.
// Returns (nil, nil) when no materials volume has been created yet.
func (m *Manager) GetCourseMaterials(ctx context.Context, courseID string) (*SharedMaterialsVolume, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	c, ok := m.courses[courseID]
	if !ok {
		return nil, ErrCourseNotFound
	}

	if c.SharedMaterialsEFSID == "" {
		return nil, nil
	}

	return &SharedMaterialsVolume{
		CourseID:  courseID,
		EFSID:     c.SharedMaterialsEFSID,
		SizeGB:    c.SharedMaterialsSizeGB,
		MountPath: c.SharedMaterialsMountPath,
		State:     "available",
		CreatedAt: c.UpdatedAt,
	}, nil
}

// --- Maintenance ---

// CheckAndCloseExpiredCourses transitions courses past their end date + grace period to "closed"
func (m *Manager) CheckAndCloseExpiredCourses(ctx context.Context) ([]string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	var closed []string

	for id, c := range m.courses {
		if c.Status != types.CourseStatusActive {
			continue
		}
		deadline := c.SemesterEnd.AddDate(0, 0, c.GracePeriodDays)
		if now.After(deadline) {
			c.Status = types.CourseStatusClosed
			c.UpdatedAt = now
			closed = append(closed, id)
		}
	}

	if len(closed) > 0 {
		if err := m.saveCourses(); err != nil {
			return nil, fmt.Errorf("failed to save course closures: %w", err)
		}
	}

	return closed, nil
}

// --- v0.16.0: Budget enforcement (#163) ---

// CheckStudentBudget returns a *BudgetExceededError if the student's accumulated spend
// plus estimatedCost would exceed their effective budget limit.
// Returns nil when the course has no budget configured, or when the member is not found
// (unknown users are not blocked — they will fail enrollment checks elsewhere).
func (m *Manager) CheckStudentBudget(courseID, userID string, estimatedCost float64) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	c, ok := m.courses[courseID]
	if !ok {
		return nil // unknown course: don't block
	}

	for _, mb := range c.Members {
		if mb.UserID != userID && mb.Email != userID {
			continue
		}
		// Resolve effective limit: member override → course default → 0 (unlimited)
		limit := mb.BudgetLimit
		if limit == 0 {
			limit = c.PerStudentBudget
		}
		if limit == 0 {
			return nil // unlimited
		}
		if mb.BudgetSpent+estimatedCost > limit {
			return &BudgetExceededError{
				UserID:    mb.UserID,
				Spent:     mb.BudgetSpent,
				Limit:     limit,
				Requested: estimatedCost,
			}
		}
		return nil
	}
	return nil // member not enrolled: don't block
}

// --- v0.16.0: Audit log (#165) ---

// appendAudit writes a single audit entry for courseID.
// Errors are logged to stderr but never propagated — audit write failure must not block operations.
func (m *Manager) appendAudit(courseID string, entry AuditEntry) {
	log := NewAuditLog(courseID, m.auditDir)
	if err := log.Append(entry); err != nil {
		fmt.Fprintf(os.Stderr, "audit write warning [%s]: %v\n", courseID, err)
	}
}

// AppendCourseAudit is the daemon-layer entry point for writing audit events that occur
// outside of manager methods (e.g. instance launch/stop events).
func (m *Manager) AppendCourseAudit(courseID string, entry AuditEntry) error {
	log := NewAuditLog(courseID, m.auditDir)
	return log.Append(entry)
}

// QueryAuditLog returns audit entries for a course, filtered by optional studentID,
// since timestamp, and entry limit. Returns nil (no error) when no log file exists yet.
func (m *Manager) QueryAuditLog(courseID, studentID string, since time.Time, limit int) ([]AuditEntry, error) {
	log := NewAuditLog(courseID, m.auditDir)
	return log.Query(studentID, since, limit)
}

// --- v0.16.0: LMS CSV parsers (#166) ---

// ParseCanvasCSV parses a Canvas LMS student roster CSV export into RosterRows.
// Expected header columns (case-insensitive): Student (display name), SIS Login ID (user ID),
// Email Address (email). Rows where the Student column equals "Points Possible" are skipped.
func ParseCanvasCSV(data []byte) ([]RosterRow, error) {
	r := csvReaderFromBytes(data)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("canvas csv parse: %w", err)
	}
	if len(records) < 2 {
		return nil, nil
	}

	// Build column index map from header row
	colIdx := make(map[string]int)
	for i, h := range records[0] {
		colIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	nameCol := colIdx["student"]
	emailCol, hasEmail := colIdx["email address"]
	userCol, hasUser := colIdx["sis login id"]

	var rows []RosterRow
	for _, rec := range records[1:] {
		name := safeCol(rec, nameCol)
		if strings.EqualFold(strings.TrimSpace(name), "points possible") {
			continue // Canvas summary row sentinel
		}
		email := ""
		if hasEmail {
			email = safeCol(rec, emailCol)
		}
		userID := ""
		if hasUser {
			userID = safeCol(rec, userCol)
		}
		if email == "" && userID == "" {
			continue // skip rows with no identity
		}
		rows = append(rows, RosterRow{
			Email:       email,
			DisplayName: name,
			Role:        userID, // temporarily store SIS ID; handler maps to UserID
		})
		// Note: RosterRow.Role is overloaded here to carry the Canvas UserID string.
		// The import handler will normalise this before calling EnrollMember.
	}
	return rows, nil
}

// ParseBlackboardCSV parses a Blackboard Learn user list CSV export into RosterRows.
// Expected header columns: Last Name, First Name, Username (user ID), Email.
func ParseBlackboardCSV(data []byte) ([]RosterRow, error) {
	r := csvReaderFromBytes(data)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("blackboard csv parse: %w", err)
	}
	if len(records) < 2 {
		return nil, nil
	}

	colIdx := make(map[string]int)
	for i, h := range records[0] {
		colIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	lastCol, hasLast := colIdx["last name"]
	firstCol, hasFirst := colIdx["first name"]
	userCol, hasUser := colIdx["username"]
	emailCol, hasEmail := colIdx["email"]

	var rows []RosterRow
	for _, rec := range records[1:] {
		email := ""
		if hasEmail {
			email = safeCol(rec, emailCol)
		}
		userID := ""
		if hasUser {
			userID = safeCol(rec, userCol)
		}
		if email == "" && userID == "" {
			continue
		}
		last := ""
		if hasLast {
			last = safeCol(rec, lastCol)
		}
		first := ""
		if hasFirst {
			first = safeCol(rec, firstCol)
		}
		displayName := strings.TrimSpace(last + ", " + first)
		if displayName == ", " || displayName == "," {
			displayName = ""
		}
		rows = append(rows, RosterRow{
			Email:       email,
			DisplayName: displayName,
			Role:        userID, // SIS/username stored in Role field; normalised by handler
		})
	}
	return rows, nil
}

// csvReaderFromBytes returns a csv.Reader configured for RFC 4180 CSV.
func csvReaderFromBytes(data []byte) *csv.Reader {
	r := csv.NewReader(strings.NewReader(string(data)))
	r.TrimLeadingSpace = true
	r.LazyQuotes = true
	return r
}

// safeCol returns rec[idx] if idx is within bounds, otherwise "".
func safeCol(rec []string, idx int) string {
	if idx < 0 || idx >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[idx])
}

// --- v0.16.0: Workspace provisioning (#172) ---

// GetProvisioningContext returns the template and budget context needed to
// launch a workspace for a student on behalf of an instructor/TA.
// overrideTemplate takes precedence over the course's DefaultTemplate.
// Returns ErrCourseNotFound, ErrMemberNotFound, or a descriptive error if
// no template is available.
func (m *Manager) GetProvisioningContext(ctx context.Context, courseID, studentID, overrideTemplate string) (*ProvisioningContext, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	c, ok := m.courses[courseID]
	if !ok {
		return nil, ErrCourseNotFound
	}

	template := overrideTemplate
	if template == "" {
		template = c.DefaultTemplate
	}
	if template == "" {
		return nil, fmt.Errorf("no template specified and course has no DefaultTemplate set")
	}

	for _, mb := range c.Members {
		if mb.UserID != studentID && mb.Email != studentID {
			continue
		}
		limit := mb.BudgetLimit
		if limit == 0 {
			limit = c.PerStudentBudget
		}
		return &ProvisioningContext{
			CourseID:     courseID,
			StudentID:    mb.UserID,
			StudentEmail: mb.Email,
			Template:     template,
			BudgetLimit:  limit,
			BudgetSpent:  mb.BudgetSpent,
		}, nil
	}
	return nil, ErrMemberNotFound
}

// --- v0.16.0: Course overview / TA dashboard (#168) ---

// GetCourseOverview returns a TA-facing summary of all students.
// instancesByStudent maps studentUserID → list of their instances (populated by the daemon
// handler from state, keyed by instance.ProjectID == courseID).
func (m *Manager) GetCourseOverview(ctx context.Context, courseID string, instancesByStudent map[string][]types.Instance) (*CourseOverview, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	c, ok := m.courses[courseID]
	if !ok {
		return nil, ErrCourseNotFound
	}

	overview := &CourseOverview{
		CourseID:    courseID,
		CourseCode:  c.Code,
		GeneratedAt: time.Now().UTC(),
	}

	for _, mb := range c.Members {
		if mb.Role != types.ClassRoleStudent {
			continue
		}
		overview.TotalStudents++

		instances := instancesByStudent[mb.UserID]
		activeCount := 0
		for _, inst := range instances {
			if inst.State == "running" {
				activeCount++
				overview.ActiveInstances++
			}
		}

		limit := mb.BudgetLimit
		if limit == 0 {
			limit = c.PerStudentBudget
		}

		status := "ok"
		if limit > 0 {
			pct := mb.BudgetSpent / limit
			switch {
			case pct > 1.0:
				status = "exceeded"
			case pct > 0.80:
				status = "warning"
			}
		}

		overview.TotalBudgetSpent += mb.BudgetSpent
		overview.Students = append(overview.Students, StudentStatus{
			UserID:       mb.UserID,
			Email:        mb.Email,
			DisplayName:  mb.DisplayName,
			Instances:    instances,
			BudgetSpent:  mb.BudgetSpent,
			BudgetLimit:  limit,
			BudgetStatus: status,
		})
	}

	return overview, nil
}

// --- v0.16.0: Usage reports (#173) ---

// GetUsageReport aggregates semester-end usage for all students.
// instanceHoursByStudent and instanceCountByStudent are keyed by student UserID and
// are populated by the daemon handler from instance state history.
func (m *Manager) GetUsageReport(ctx context.Context, courseID string,
	instanceHoursByStudent map[string]float64,
	instanceCountByStudent map[string]int,
) (*UsageReport, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	c, ok := m.courses[courseID]
	if !ok {
		return nil, ErrCourseNotFound
	}

	report := &UsageReport{
		CourseID:    courseID,
		CourseCode:  c.Code,
		Semester:    c.Semester,
		TotalBudget: c.TotalBudget,
		GeneratedAt: time.Now().UTC(),
	}

	for _, mb := range c.Members {
		if mb.Role != types.ClassRoleStudent {
			continue
		}
		limit := mb.BudgetLimit
		if limit == 0 {
			limit = c.PerStudentBudget
		}
		report.TotalSpent += mb.BudgetSpent
		report.Students = append(report.Students, StudentUsageRecord{
			UserID:        mb.UserID,
			Email:         mb.Email,
			DisplayName:   mb.DisplayName,
			TotalSpent:    mb.BudgetSpent,
			BudgetLimit:   limit,
			InstanceHours: instanceHoursByStudent[mb.UserID],
			InstanceCount: instanceCountByStudent[mb.UserID],
		})
	}

	return report, nil
}

// --- v0.16.0: Archive eligible courses (#162) ---

// ListArchiveEligibleCourses returns courses in "closed" state whose
// SemesterEnd + GracePeriodDays has passed and that have not yet been archived.
func (m *Manager) ListArchiveEligibleCourses(ctx context.Context) ([]*types.Course, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	now := time.Now()
	var eligible []*types.Course
	for _, c := range m.courses {
		if c.Status != types.CourseStatusClosed {
			continue
		}
		archiveAfter := c.SemesterEnd.AddDate(0, 0, c.GracePeriodDays)
		if now.After(archiveAfter) {
			// Return a shallow copy to avoid holding a reference into the map
			cp := *c
			eligible = append(eligible, &cp)
		}
	}
	return eligible, nil
}

// MarkCourseArchived transitions a course from "closed" to "archived" and saves.
func (m *Manager) MarkCourseArchived(ctx context.Context, courseID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c, ok := m.courses[courseID]
	if !ok {
		return ErrCourseNotFound
	}
	if c.Status != types.CourseStatusClosed {
		return fmt.Errorf("course %s is not in closed state (current: %s)", courseID, c.Status)
	}

	c.Status = types.CourseStatusArchived
	c.UpdatedAt = time.Now().UTC()

	if err := m.saveCourses(); err != nil {
		return fmt.Errorf("failed to save archived course: %w", err)
	}

	m.appendAudit(courseID, AuditEntry{
		CourseID: courseID,
		Actor:    "system",
		Action:   AuditActionCourseArchive,
		Detail:   map[string]interface{}{"auto": true},
	})

	return nil
}

// --- Internal Helpers ---

// isTAOrInstructorLocked checks if userID is a TA or instructor. Caller must hold at least RLock.
func (m *Manager) isTAOrInstructorLocked(c *types.Course, userID string) bool {
	if c.Owner == userID {
		return true
	}
	for _, mb := range c.Members {
		if (mb.UserID == userID || mb.Email == userID) &&
			(mb.Role == types.ClassRoleTA || mb.Role == types.ClassRoleInstructor) {
			return true
		}
	}
	return false
}

// loadCourses loads the course map from disk
func (m *Manager) loadCourses() error {
	if _, err := os.Stat(m.coursesPath); os.IsNotExist(err) {
		return nil // no file yet; start empty
	}

	data, err := os.ReadFile(m.coursesPath)
	if err != nil {
		return fmt.Errorf("failed to read courses file: %w", err)
	}

	var courses map[string]*types.Course
	if err := json.Unmarshal(data, &courses); err != nil {
		return fmt.Errorf("failed to parse courses file: %w", err)
	}

	m.courses = courses
	return nil
}

// saveCourses persists the course map to disk atomically
func (m *Manager) saveCourses() error {
	data, err := json.MarshalIndent(m.courses, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal courses: %w", err)
	}

	tempPath := m.coursesPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary courses file: %w", err)
	}

	if err := os.Rename(tempPath, m.coursesPath); err != nil {
		return fmt.Errorf("failed to rename courses file: %w", err)
	}

	return nil
}
