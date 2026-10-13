package daemon

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/scttfrdmn/prism/pkg/course"
	"github.com/scttfrdmn/prism/pkg/types"
)

// handleCourseOperations routes /api/v1/courses (no trailing path segment)
func (s *Server) handleCourseOperations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListCourses(w, r)
	case http.MethodPost:
		s.handleCreateCourse(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleCourseByID dispatches /api/v1/courses/{id}[/{sub-resource}...]
func (s *Server) handleCourseByID(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/v1/courses/"):]
	parts := splitPath(path)
	if len(parts) == 0 {
		s.writeError(w, http.StatusBadRequest, "Missing course ID")
		return
	}
	courseID := parts[0]
	if len(parts) == 1 {
		s.handleCourseDirectOp(w, r, courseID)
		return
	}

	switch parts[1] {
	case "close":
		if r.Method != http.MethodPost {
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		s.handleCloseCourse(w, r, courseID)
	case "members":
		s.handleCourseMembers(w, r, courseID, parts)
	case "templates":
		s.handleCourseTemplates(w, r, courseID, parts)
	case "budget":
		s.handleCourseBudget(w, r, courseID, parts)
	case "ta":
		s.handleCourseTA(w, r, courseID, parts)
	case "overview":
		if r.Method != http.MethodGet {
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		s.handleCourseOverview(w, r, courseID)
	case "report":
		if r.Method != http.MethodGet {
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		s.handleCourseReport(w, r, courseID)
	case "audit":
		if r.Method != http.MethodGet {
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		s.handleCourseAudit(w, r, courseID)
	case "archive":
		if r.Method != http.MethodPost {
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		s.handleArchiveCourse(w, r, courseID)
	case "ta-access":
		s.handleCourseTAAccess(w, r, courseID, parts)
	case "materials":
		s.handleCourseMaterials(w, r, courseID, parts)
	default:
		http.NotFound(w, r)
	}
}

// handleCourseDirectOp handles GET/PUT/DELETE on /api/v1/courses/{id}
func (s *Server) handleCourseDirectOp(w http.ResponseWriter, r *http.Request, courseID string) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetCourse(w, r, courseID)
	case http.MethodPut:
		s.handleUpdateCourse(w, r, courseID)
	case http.MethodDelete:
		s.handleDeleteCourse(w, r, courseID)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// --- CRUD ---

func (s *Server) handleListCourses(w http.ResponseWriter, r *http.Request) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	q := r.URL.Query()
	filter := &course.CourseFilter{
		Owner:      q.Get("owner"),
		Semester:   q.Get("semester"),
		Department: q.Get("department"),
	}
	if statusStr := q.Get("status"); statusStr != "" {
		st := types.CourseStatus(statusStr)
		filter.Status = &st
	}

	courses, err := s.courseManager.ListCourses(r.Context(), filter)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if courses == nil {
		courses = []*types.Course{}
	}
	s.writeJSON(w, http.StatusOK, map[string]interface{}{"courses": courses})
}

func (s *Server) handleCreateCourse(w http.ResponseWriter, r *http.Request) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	var req course.CreateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	c, err := s.courseManager.CreateCourse(r.Context(), &req)
	if err != nil {
		if err == course.ErrDuplicateCourse {
			s.writeError(w, http.StatusConflict, err.Error())
			return
		}
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleGetCourse(w http.ResponseWriter, r *http.Request, courseID string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	c, err := s.courseManager.GetCourse(r.Context(), courseID)
	if err != nil {
		if err == course.ErrCourseNotFound {
			s.writeError(w, http.StatusNotFound, "Course not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleUpdateCourse(w http.ResponseWriter, r *http.Request, courseID string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	var req course.UpdateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	c, err := s.courseManager.UpdateCourse(r.Context(), courseID, &req)
	if err != nil {
		if err == course.ErrCourseNotFound {
			s.writeError(w, http.StatusNotFound, "Course not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleDeleteCourse(w http.ResponseWriter, r *http.Request, courseID string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	if err := s.courseManager.DeleteCourse(r.Context(), courseID); err != nil {
		if err == course.ErrCourseNotFound {
			s.writeError(w, http.StatusNotFound, "Course not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCloseCourse(w http.ResponseWriter, r *http.Request, courseID string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	if err := s.courseManager.CloseCourse(r.Context(), courseID); err != nil {
		if err == course.ErrCourseNotFound {
			s.writeError(w, http.StatusNotFound, "Course not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "closed"})
}

// --- Member Management ---

func (s *Server) handleCourseMembers(w http.ResponseWriter, r *http.Request, courseID string, parts []string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	// /courses/{id}/members/bulk  or  /courses/{id}/members/import
	if len(parts) >= 3 {
		switch parts[2] {
		case "bulk":
			if r.Method != http.MethodPost {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			s.handleCourseBulkEnroll(w, r, courseID)
			return
		case "import":
			if r.Method != http.MethodPost {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			s.handleCourseRosterImport(w, r, courseID)
			return
		default:
			// /courses/{id}/members/{userID}
			if r.Method == http.MethodDelete {
				s.handleUnenrollMember(w, r, courseID, parts[2])
				return
			}
			// /courses/{id}/members/{userID}/provision
			if len(parts) >= 4 && parts[3] == "provision" && r.Method == http.MethodPost {
				s.handleProvisionStudent(w, r, courseID, parts[2])
				return
			}
			http.NotFound(w, r)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		s.handleListCourseMembers(w, r, courseID)
	case http.MethodPost:
		s.handleEnrollMember(w, r, courseID)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleListCourseMembers(w http.ResponseWriter, r *http.Request, courseID string) {
	var rolePtr *types.ClassRole
	if roleStr := r.URL.Query().Get("role"); roleStr != "" {
		role := types.ClassRole(roleStr)
		rolePtr = &role
	}

	members, err := s.courseManager.ListMembers(r.Context(), courseID, rolePtr)
	if err != nil {
		if err == course.ErrCourseNotFound {
			s.writeError(w, http.StatusNotFound, "Course not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if members == nil {
		members = []types.ClassMember{}
	}
	s.writeJSON(w, http.StatusOK, map[string]interface{}{"members": members})
}

func (s *Server) handleEnrollMember(w http.ResponseWriter, r *http.Request, courseID string) {
	var req course.EnrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	mb, err := s.courseManager.EnrollMember(r.Context(), courseID, &req)
	if err != nil {
		switch err {
		case course.ErrCourseNotFound:
			s.writeError(w, http.StatusNotFound, "Course not found")
		case course.ErrAlreadyEnrolled:
			s.writeError(w, http.StatusConflict, err.Error())
		case course.ErrCourseClosed:
			s.writeError(w, http.StatusForbidden, err.Error())
		default:
			s.writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	s.writeJSON(w, http.StatusCreated, mb)
}

func (s *Server) handleCourseBulkEnroll(w http.ResponseWriter, r *http.Request, courseID string) {
	var req course.BulkEnrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	enrolled, rowErrors, err := s.courseManager.BulkEnroll(r.Context(), courseID, &req)
	if err != nil {
		if err == course.ErrCourseNotFound {
			s.writeError(w, http.StatusNotFound, "Course not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	errMsgs := make([]string, 0, len(rowErrors))
	for _, e := range rowErrors {
		errMsgs = append(errMsgs, e.Error())
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"enrolled": enrolled,
		"errors":   errMsgs,
	})
}

func (s *Server) handleCourseRosterImport(w http.ResponseWriter, r *http.Request, courseID string) {
	// Accept either JSON rows or raw CSV body
	var rows []course.RosterRow

	format := course.RosterFormat(r.URL.Query().Get("format"))
	if format == "" {
		format = course.RosterFormatPrism
	}

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/csv") || strings.Contains(contentType, "text/plain") {
		// Read raw CSV bytes
		var buf []byte
		buf = make([]byte, 0, 1024*1024)
		tmp := make([]byte, 4096)
		for {
			n, err := r.Body.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
			}
			if err != nil {
				break
			}
		}
		var err error
		switch format {
		case course.RosterFormatCanvas:
			rows, err = course.ParseCanvasCSV(buf)
		case course.RosterFormatBlackboard:
			rows, err = course.ParseBlackboardCSV(buf)
		default:
			rows, err = course.ParseRosterCSV(buf)
		}
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "CSV parse error: "+err.Error())
			return
		}
	} else {
		// Expect JSON array of RosterRow
		if err := json.NewDecoder(r.Body).Decode(&rows); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}
	}

	sendInvites := r.URL.Query().Get("send_invites") == "true"
	enrolled, rowErrors, err := s.courseManager.ImportRosterCSV(r.Context(), courseID, rows, sendInvites)
	if err != nil {
		if err == course.ErrCourseNotFound {
			s.writeError(w, http.StatusNotFound, "Course not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	errMsgs := make([]string, 0, len(rowErrors))
	for _, e := range rowErrors {
		errMsgs = append(errMsgs, e.Error())
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"enrolled": enrolled,
		"errors":   errMsgs,
	})
}

func (s *Server) handleUnenrollMember(w http.ResponseWriter, r *http.Request, courseID, userID string) {
	if err := s.courseManager.UnenrollMember(r.Context(), courseID, userID); err != nil {
		switch err {
		case course.ErrCourseNotFound:
			s.writeError(w, http.StatusNotFound, "Course not found")
		case course.ErrMemberNotFound:
			s.writeError(w, http.StatusNotFound, "Member not found")
		default:
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Template Whitelist (#46) ---

func (s *Server) handleCourseTemplates(w http.ResponseWriter, r *http.Request, courseID string, parts []string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	// /courses/{id}/templates/{slug}
	if len(parts) >= 3 {
		slug := parts[2]
		if r.Method == http.MethodDelete {
			if err := s.courseManager.RemoveApprovedTemplate(r.Context(), courseID, slug); err != nil {
				if err == course.ErrCourseNotFound {
					s.writeError(w, http.StatusNotFound, "Course not found")
					return
				}
				s.writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		c, err := s.courseManager.GetCourse(r.Context(), courseID)
		if err != nil {
			if err == course.ErrCourseNotFound {
				s.writeError(w, http.StatusNotFound, "Course not found")
				return
			}
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		templates := c.ApprovedTemplates
		if templates == nil {
			templates = []string{}
		}
		s.writeJSON(w, http.StatusOK, map[string]interface{}{"approved_templates": templates})

	case http.MethodPost:
		var body struct {
			Template string `json:"template"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Template == "" {
			s.writeError(w, http.StatusBadRequest, "Request body must include {\"template\": \"slug\"}")
			return
		}
		if err := s.courseManager.AddApprovedTemplate(r.Context(), courseID, body.Template); err != nil {
			if err == course.ErrCourseNotFound {
				s.writeError(w, http.StatusNotFound, "Course not found")
				return
			}
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.writeJSON(w, http.StatusCreated, map[string]string{"template": body.Template})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// --- Budget (#47) ---

func (s *Server) handleCourseBudget(w http.ResponseWriter, r *http.Request, courseID string, parts []string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	// /courses/{id}/budget/distribute
	if len(parts) >= 3 && parts[2] == "distribute" {
		if r.Method != http.MethodPost {
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		var body struct {
			AmountPerStudent float64 `json:"amount_per_student"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}
		if err := s.courseManager.DistributeBudget(r.Context(), courseID, body.AmountPerStudent); err != nil {
			if err == course.ErrCourseNotFound {
				s.writeError(w, http.StatusNotFound, "Course not found")
				return
			}
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]string{"status": "distributed"})
		return
	}

	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	summary, err := s.courseManager.GetBudgetSummary(r.Context(), courseID)
	if err != nil {
		if err == course.ErrCourseNotFound {
			s.writeError(w, http.StatusNotFound, "Course not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, summary)
}

// --- TA Operations (#48, #49) ---

func (s *Server) handleCourseTA(w http.ResponseWriter, r *http.Request, courseID string, parts []string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	// Expect /courses/{id}/ta/{action}/{studentID}
	if len(parts) < 4 {
		s.writeError(w, http.StatusBadRequest, "Expected /ta/{debug|reset}/{studentID}")
		return
	}

	action := parts[2]
	studentID := parts[3]

	// The caller's identity comes from the request context (set by auth middleware)
	taID, _ := r.Context().Value(userIDKey).(string)

	switch action {
	case "debug":
		if r.Method != http.MethodGet {
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		s.handleTADebug(w, r, courseID, taID, studentID)

	case "reset":
		if r.Method != http.MethodPost {
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		s.handleTAReset(w, r, courseID, taID, studentID)

	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleTADebug(w http.ResponseWriter, r *http.Request, courseID, taID, studentID string) {
	// Build a context carrying the caller ID for authorization
	ctx := course.WithCaller(r.Context(), taID)

	info, err := s.courseManager.GetStudentDebugInfo(ctx, courseID, taID, studentID)
	if err != nil {
		switch err {
		case course.ErrCourseNotFound:
			s.writeError(w, http.StatusNotFound, "Course not found")
		case course.ErrMemberNotFound:
			s.writeError(w, http.StatusNotFound, "Student not found in course")
		case course.ErrNotAuthorized:
			s.writeError(w, http.StatusForbidden, "Not authorized: must be a TA or instructor")
		default:
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Populate live instance data from state (no AWS call needed for basic debug info)
	if s.stateManager != nil {
		if state, loadErr := s.stateManager.LoadState(); loadErr == nil {
			var instances []types.Instance
			for _, inst := range state.Instances {
				if inst.ProjectID == courseID || inst.Name == studentID {
					instances = append(instances, inst)
				}
			}
			info.Instances = instances
		}
	}

	s.writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleTAReset(w http.ResponseWriter, r *http.Request, courseID, taID, studentID string) {
	var req course.TAResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	req.StudentID = studentID

	if req.Reason == "" {
		s.writeError(w, http.StatusBadRequest, "reason is required for TA reset")
		return
	}
	if req.BackupRetentionDays <= 0 {
		req.BackupRetentionDays = 7
	}

	ctx := course.WithCaller(r.Context(), taID)

	if err := s.courseManager.ResetStudentInstance(ctx, courseID, taID, &req); err != nil {
		switch err {
		case course.ErrCourseNotFound:
			s.writeError(w, http.StatusNotFound, "Course not found")
		case course.ErrMemberNotFound:
			s.writeError(w, http.StatusNotFound, "Student not found in course")
		case course.ErrNotAuthorized:
			s.writeError(w, http.StatusForbidden, "Not authorized: must be a TA or instructor")
		default:
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Return 202 Accepted — the actual snapshot + re-provision is async
	s.writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":                "reset_scheduled",
		"student_id":            studentID,
		"backup_retention_days": req.BackupRetentionDays,
		"reason":                req.Reason,
	})
}

// --- v0.16.0 handlers ---

// handleCourseOverview serves GET /api/v1/courses/{id}/overview (#168)
func (s *Server) handleCourseOverview(w http.ResponseWriter, r *http.Request, courseID string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	// Build instancesByStudent from local state
	instancesByStudent := make(map[string][]types.Instance)
	if s.stateManager != nil {
		if state, err := s.stateManager.LoadState(); err == nil {
			for _, inst := range state.Instances {
				if inst.ProjectID == courseID {
					instancesByStudent[inst.Name] = append(instancesByStudent[inst.Name], inst)
				}
			}
		}
	}

	overview, err := s.courseManager.GetCourseOverview(r.Context(), courseID, instancesByStudent)
	if err != nil {
		if err == course.ErrCourseNotFound {
			s.writeError(w, http.StatusNotFound, "Course not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusOK, overview)
}

// handleCourseReport serves GET /api/v1/courses/{id}/report[?format=json|csv] (#173)
func (s *Server) handleCourseReport(w http.ResponseWriter, r *http.Request, courseID string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	// Build per-student instance hour/count stats from local state
	hoursByStudent := make(map[string]float64)
	countByStudent := make(map[string]int)
	if s.stateManager != nil {
		if state, err := s.stateManager.LoadState(); err == nil {
			for _, inst := range state.Instances {
				if inst.ProjectID == courseID {
					// Estimate hours from launch time to now
					if !inst.LaunchTime.IsZero() {
						hours := time.Since(inst.LaunchTime).Hours()
						hoursByStudent[inst.Name] += hours
					}
					countByStudent[inst.Name]++
				}
			}
		}
	}

	report, err := s.courseManager.GetUsageReport(r.Context(), courseID, hoursByStudent, countByStudent)
	if err != nil {
		if err == course.ErrCourseNotFound {
			s.writeError(w, http.StatusNotFound, "Course not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if r.URL.Query().Get("format") == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"course-%s-report.csv\"", report.CourseCode))
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"user_id", "email", "display_name", "total_spent", "budget_limit", "instance_hours", "instance_count"})
		for _, st := range report.Students {
			_ = cw.Write([]string{
				st.UserID, st.Email, st.DisplayName,
				strconv.FormatFloat(st.TotalSpent, 'f', 2, 64),
				strconv.FormatFloat(st.BudgetLimit, 'f', 2, 64),
				strconv.FormatFloat(st.InstanceHours, 'f', 2, 64),
				strconv.Itoa(st.InstanceCount),
			})
		}
		cw.Flush()
		return
	}

	s.writeJSON(w, http.StatusOK, report)
}

// handleCourseAudit serves GET /api/v1/courses/{id}/audit (#165)
func (s *Server) handleCourseAudit(w http.ResponseWriter, r *http.Request, courseID string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	q := r.URL.Query()
	studentID := q.Get("student_id")

	var since time.Time
	if sinceStr := q.Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}

	limit := 100
	if limitStr := q.Get("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	entries, err := s.courseManager.QueryAuditLog(courseID, studentID, since, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entries == nil {
		entries = []course.AuditEntry{}
	}
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
	})
}

// handleArchiveCourse serves POST /api/v1/courses/{id}/archive (#162)
func (s *Server) handleArchiveCourse(w http.ResponseWriter, r *http.Request, courseID string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	result, err := s.archiveCourse(r.Context(), courseID)
	if err != nil {
		if err == course.ErrCourseNotFound {
			s.writeError(w, http.StatusNotFound, "Course not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}

// archiveCourse records which instances belong to the course and marks it archived.
// Actual AWS stop/snapshot is performed asynchronously by the daemon's background sweep;
// this call returns immediately with the list of affected instances.
func (s *Server) archiveCourse(ctx context.Context, courseID string) (*course.ArchiveResult, error) {
	result := &course.ArchiveResult{CourseID: courseID}

	// Scan local state for instances belonging to this course
	if s.stateManager != nil {
		if state, err := s.stateManager.LoadState(); err == nil {
			for name, inst := range state.Instances {
				if inst.ProjectID == courseID {
					result.InstancesStopped = append(result.InstancesStopped, name)
					_ = inst // referenced to satisfy compiler
				}
			}
		}
	}

	if err := s.courseManager.MarkCourseArchived(ctx, courseID); err != nil {
		return nil, err
	}
	return result, nil
}

// handleProvisionStudent serves POST /api/v1/courses/{id}/members/{userID}/provision (#172)
func (s *Server) handleProvisionStudent(w http.ResponseWriter, r *http.Request, courseID, studentID string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	var body struct {
		Template     string `json:"template"`
		InstanceName string `json:"instance_name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	pctx, err := s.courseManager.GetProvisioningContext(r.Context(), courseID, studentID, body.Template)
	if err != nil {
		switch err {
		case course.ErrCourseNotFound:
			s.writeError(w, http.StatusNotFound, "Course not found")
		case course.ErrMemberNotFound:
			s.writeError(w, http.StatusNotFound, "Student not found in course")
		default:
			s.writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	name := body.InstanceName
	if name == "" {
		name = fmt.Sprintf("%s-ws", strings.ReplaceAll(pctx.StudentID, "@", "-"))
	}

	// Synthesize a LaunchRequest and delegate to the standard launch handler logic
	launchReq := types.LaunchRequest{
		Name:     name,
		Template: pctx.Template,
		CourseID: pctx.CourseID,
	}

	w.Header().Set("Content-Type", "application/json")
	s.writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":        "provisioning",
		"student_id":    pctx.StudentID,
		"course_id":     pctx.CourseID,
		"template":      launchReq.Template,
		"instance_name": launchReq.Name,
	})
}

// --- TA Access (#48, #160) ---

// handleCourseTAAccess handles /api/v1/courses/{id}/ta-access[/connect|/{email}]
//
//	GET    /ta-access              → list TAs
//	POST   /ta-access              → grant TA access (body: {email, display_name})
//	DELETE /ta-access/{email}      → revoke TA access
//	POST   /ta-access/connect      → get SSH command for TA→student access
func (s *Server) handleCourseTAAccess(w http.ResponseWriter, r *http.Request, courseID string, parts []string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	// parts[2] is the sub-resource (absent for list/grant)
	sub := ""
	if len(parts) > 2 {
		sub = parts[2]
	}

	switch {
	case sub == "connect" && r.Method == http.MethodPost:
		s.handleTASSHConnect(w, r, courseID)
	case sub == "" && r.Method == http.MethodGet:
		// List TAs
		members, err := s.courseManager.ListTAAccess(r.Context(), courseID)
		if err != nil {
			s.writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if members == nil {
			members = []types.ClassMember{}
		}
		s.writeJSON(w, http.StatusOK, map[string]interface{}{"ta_members": members})
	case sub == "" && r.Method == http.MethodPost:
		// Grant TA access
		var body course.TAAccessGrantRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
			s.writeError(w, http.StatusBadRequest, "email is required")
			return
		}
		member, err := s.courseManager.GrantTAAccess(r.Context(), courseID, body.Email, body.DisplayName)
		if err != nil {
			switch err {
			case course.ErrCourseNotFound:
				s.writeError(w, http.StatusNotFound, "Course not found")
			case course.ErrNotAuthorized:
				s.writeError(w, http.StatusForbidden, err.Error())
			default:
				s.writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}
		s.writeJSON(w, http.StatusCreated, member)
	case sub != "" && sub != "connect" && r.Method == http.MethodDelete:
		// Revoke TA access: sub is the email (URL-encoded)
		taEmail := sub
		if err := s.courseManager.RevokeTAAccess(r.Context(), courseID, taEmail); err != nil {
			switch err {
			case course.ErrCourseNotFound:
				s.writeError(w, http.StatusNotFound, "Course not found")
			case course.ErrMemberNotFound:
				s.writeError(w, http.StatusNotFound, "TA member not found")
			default:
				s.writeError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleTASSHConnect handles POST /api/v1/courses/{id}/ta-access/connect
// It verifies the TA is authorized, finds the student's running instance,
// builds a ready-to-paste SSH command, and records the access in the audit log.
func (s *Server) handleTASSHConnect(w http.ResponseWriter, r *http.Request, courseID string) {
	var req course.TASSHConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.StudentID == "" || req.Reason == "" {
		s.writeError(w, http.StatusBadRequest, "student_id and reason are required")
		return
	}

	// TA identity is client-asserted: the daemon authenticates a single API key and has no per-user
	// identity in the request context (the "user_id" value previously read here was never set).
	taID := r.Header.Get("X-User-ID")
	if taID == "" {
		taID = "ta-user" // fallback for test mode
	}

	// Look up the student's debug info to find instance IP
	info, err := s.courseManager.GetStudentDebugInfo(r.Context(), courseID, taID, req.StudentID)
	if err != nil {
		switch err {
		case course.ErrCourseNotFound:
			s.writeError(w, http.StatusNotFound, "Course not found")
		case course.ErrMemberNotFound:
			s.writeError(w, http.StatusNotFound, "Student not found in course")
		case course.ErrNotAuthorized:
			s.writeError(w, http.StatusForbidden, "Not authorized: must be TA or instructor")
		default:
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Find a running instance with a public IP
	var publicIP, instanceID string
	for _, inst := range info.Instances {
		if inst.State == "running" && inst.PublicIP != "" {
			publicIP = inst.PublicIP
			instanceID = inst.ID
			break
		}
	}

	if publicIP == "" {
		s.writeError(w, http.StatusNotFound, "No running instance found for student")
		return
	}

	// Record TA access in audit log
	_ = s.courseManager.LogTASSHConnect(r.Context(), courseID, course.TAAccessEntry{
		TAID:        taID,
		StudentID:   req.StudentID,
		Reason:      req.Reason,
		ConnectedAt: time.Now(),
		InstanceID:  instanceID,
		PublicIP:    publicIP,
	})

	// Return an SSH command the TA can paste directly into their terminal
	sshCmd := fmt.Sprintf("ssh -i ~/.ssh/prism_key ec2-user@%s", publicIP)
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"ssh_command": sshCmd,
		"public_ip":   publicIP,
		"instance_id": instanceID,
		"student_id":  req.StudentID,
		"note":        "Connection is logged. Disconnect with 'exit' when done.",
	})
}

// --- Shared Course Materials (#167) ---

// handleCourseMaterials handles /api/v1/courses/{id}/materials[/mount]
//
//	GET  /materials       → get materials volume metadata
//	POST /materials       → create materials EFS volume (body: {size_gb, mount_path})
//	POST /materials/mount → mount materials on all student instances
func (s *Server) handleCourseMaterials(w http.ResponseWriter, r *http.Request, courseID string, parts []string) {
	if s.courseManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Course manager unavailable")
		return
	}

	sub := ""
	if len(parts) > 2 {
		sub = parts[2]
	}

	switch {
	case sub == "mount" && r.Method == http.MethodPost:
		s.handleMountCourseMaterials(w, r, courseID)
	case sub == "" && r.Method == http.MethodGet:
		// Return materials metadata
		vol, err := s.courseManager.GetCourseMaterials(r.Context(), courseID)
		if err != nil {
			switch err {
			case course.ErrCourseNotFound:
				s.writeError(w, http.StatusNotFound, "Course not found")
			default:
				s.writeError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		if vol == nil {
			s.writeJSON(w, http.StatusOK, map[string]interface{}{"materials": nil})
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]interface{}{"materials": vol})
	case sub == "" && r.Method == http.MethodPost:
		var req course.CourseMaterialsCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if req.SizeGB <= 0 {
			req.SizeGB = 50
		}
		if req.MountPath == "" {
			req.MountPath = "/mnt/course-materials"
		}

		// Create a real EFS volume for course materials storage.
		// In test mode, skip the AWS call and use a deterministic fake EFS ID.
		var efsID string
		if s.testMode {
			efsID = fmt.Sprintf("fs-test-%s", courseID[:8])
		} else {
			if s.awsManager == nil {
				s.writeError(w, http.StatusServiceUnavailable, "AWS manager not available; cannot create EFS volume")
				return
			}
			efsVol, err := s.awsManager.CreateVolume(types.VolumeCreateRequest{
				Name:            fmt.Sprintf("course-materials-%s", courseID),
				PerformanceMode: "generalPurpose",
				ThroughputMode:  "bursting",
			})
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create EFS volume: %v", err))
				return
			}
			efsID = efsVol.FileSystemID
		}

		if err := s.courseManager.SetCourseMaterials(r.Context(), courseID, efsID, req.MountPath, req.SizeGB); err != nil {
			switch err {
			case course.ErrCourseNotFound:
				s.writeError(w, http.StatusNotFound, "Course not found")
			default:
				s.writeError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		vol, _ := s.courseManager.GetCourseMaterials(r.Context(), courseID)
		s.writeJSON(w, http.StatusCreated, map[string]interface{}{"materials": vol})
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleMountCourseMaterials handles POST /api/v1/courses/{id}/materials/mount
// It records the mount action in the audit log and returns a summary.
// The actual EFS mount-on-instance is out-of-scope for this release
// (would require SSM RunCommand on each running student instance).
func (s *Server) handleMountCourseMaterials(w http.ResponseWriter, r *http.Request, courseID string) {
	vol, err := s.courseManager.GetCourseMaterials(r.Context(), courseID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if vol == nil {
		s.writeError(w, http.StatusConflict, "No materials volume exists for this course; create one first")
		return
	}

	// Record the mount action
	ctx := course.WithCaller(r.Context(), "system")
	_ = s.courseManager.AppendCourseAudit(courseID, course.AuditEntry{
		CourseID: courseID,
		Actor:    "system",
		Action:   course.AuditActionMaterialsMount,
		Detail: map[string]interface{}{
			"efs_id":     vol.EFSID,
			"mount_path": vol.MountPath,
		},
	})
	_ = ctx

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "mount_scheduled",
		"efs_id":     vol.EFSID,
		"mount_path": vol.MountPath,
		"note":       "EFS will be mounted on student instances at next launch. Running instances require manual mount or restart.",
	})
}
