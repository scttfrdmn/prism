package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	apiclient "github.com/scttfrdmn/prism/pkg/api/client"
)

// courseClient returns the underlying *HTTPClient, or an error if unavailable.
func (a *App) courseClient() (*apiclient.HTTPClient, error) {
	hc, ok := a.apiClient.(*apiclient.HTTPClient)
	if !ok {
		return nil, fmt.Errorf("course commands require daemon connection")
	}
	return hc, nil
}

// Course handles the 'prism course' command dispatcher.
func (a *App) Course(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism course <action> [args]")
	}
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: prism admin daemon start")
	}
	hc, err := a.courseClient()
	if err != nil {
		return err
	}
	action := args[0]
	rest := args[1:]
	switch action {
	case "list":
		return a.courseList(hc, rest)
	case "create":
		return a.courseCreate(hc, rest)
	case "show":
		return a.courseShow(hc, rest)
	case "close":
		return a.courseClose(hc, rest)
	case "delete":
		return a.courseDelete(hc, rest)
	case "templates":
		return a.courseTemplates(hc, rest)
	case "students":
		return a.courseStudents(hc, rest)
	case "budget":
		return a.courseBudget(hc, rest)
	case "archive":
		return a.courseArchive(hc, rest)
	case "report":
		return a.courseReport(hc, rest)
	case "audit":
		return a.courseAudit(hc, rest)
	case "ta-access":
		return a.courseTAAccess(hc, rest)
	case "materials":
		return a.courseMaterials(hc, rest)
	case "reset-workspace":
		return a.courseResetWorkspace(hc, rest)
	default:
		return fmt.Errorf("unknown course action: %s", action)
	}
}

// TA handles the 'prism ta' command dispatcher.
func (a *App) TA(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism ta <action> [args]")
	}
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: prism admin daemon start")
	}
	hc, err := a.courseClient()
	if err != nil {
		return err
	}
	action := args[0]
	rest := args[1:]
	switch action {
	case "debug":
		return a.taDebug(hc, rest)
	case "reset":
		return a.taReset(hc, rest)
	case "overview":
		return a.taOverview(hc, rest)
	default:
		return fmt.Errorf("unknown ta action: %s", action)
	}
}

// --- course list ---

func (a *App) courseList(hc *apiclient.HTTPClient, args []string) error {
	params := ""
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--semester" && i+1 < len(args):
			params += "semester=" + args[i+1] + "&"
			i++
		case args[i] == "--owner" && i+1 < len(args):
			params += "owner=" + args[i+1] + "&"
			i++
		case args[i] == "--status" && i+1 < len(args):
			params += "status=" + args[i+1] + "&"
			i++
		}
	}

	result, err := hc.ListCourses(a.ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list courses: %w", err)
	}

	courses, _ := result["courses"].([]interface{})
	if len(courses) == 0 {
		fmt.Println("No courses found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCODE\tTITLE\tSEMESTER\tSTATUS\tMEMBERS")
	fmt.Fprintln(w, "--\t----\t-----\t--------\t------\t-------")
	for _, raw := range courses {
		c, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		members := 0.0
		if m, ok := c["members"].([]interface{}); ok {
			members = float64(len(m))
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%.0f\n",
			strTrunc(c["id"], 8),
			strVal(c["code"]),
			strTrunc(c["title"], 30),
			strVal(c["semester"]),
			strVal(c["status"]),
			members,
		)
	}
	return w.Flush()
}

// --- course create ---

func (a *App) courseCreate(hc *apiclient.HTTPClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism course create <code> [--title <t>] [--semester <s>] [--owner <id>] [--start <date>] [--end <date>]")
	}

	req := map[string]interface{}{
		"code": args[0],
	}

	for i := 1; i < len(args); i++ {
		switch {
		case args[i] == "--title" && i+1 < len(args):
			req["title"] = args[i+1]
			i++
		case args[i] == "--semester" && i+1 < len(args):
			req["semester"] = args[i+1]
			i++
		case args[i] == "--owner" && i+1 < len(args):
			req["owner"] = args[i+1]
			i++
		case args[i] == "--start" && i+1 < len(args):
			req["semester_start"] = args[i+1] + "T00:00:00Z"
			i++
		case args[i] == "--end" && i+1 < len(args):
			req["semester_end"] = args[i+1] + "T00:00:00Z"
			i++
		case args[i] == "--budget" && i+1 < len(args):
			if v, err := strconv.ParseFloat(args[i+1], 64); err == nil {
				req["per_student_budget"] = v
			}
			i++
		case args[i] == "--department" && i+1 < len(args):
			req["department"] = args[i+1]
			i++
		}
	}

	if req["title"] == nil {
		req["title"] = req["code"]
	}

	result, err := hc.CreateCourse(a.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create course: %w", err)
	}

	fmt.Printf("Course created:\n")
	fmt.Printf("  ID:       %s\n", strVal(result["id"]))
	fmt.Printf("  Code:     %s\n", strVal(result["code"]))
	fmt.Printf("  Title:    %s\n", strVal(result["title"]))
	fmt.Printf("  Semester: %s\n", strVal(result["semester"]))
	fmt.Printf("  Status:   %s\n", strVal(result["status"]))
	return nil
}

// --- course show ---

func (a *App) courseShow(hc *apiclient.HTTPClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism course show <id>")
	}
	result, err := hc.GetCourse(a.ctx, args[0])
	if err != nil {
		return fmt.Errorf("failed to get course: %w", err)
	}
	printCourseDetail(result)
	return nil
}

func printCourseDetail(c map[string]interface{}) {
	fmt.Printf("Course: %s — %s\n", strVal(c["code"]), strVal(c["title"]))
	fmt.Printf("  ID:         %s\n", strVal(c["id"]))
	fmt.Printf("  Department: %s\n", strVal(c["department"]))
	fmt.Printf("  Semester:   %s\n", strVal(c["semester"]))
	fmt.Printf("  Status:     %s\n", strVal(c["status"]))
	fmt.Printf("  Owner:      %s\n", strVal(c["owner"]))
	if templates, ok := c["approved_templates"].([]interface{}); ok && len(templates) > 0 {
		fmt.Printf("  Templates:  ")
		for i, t := range templates {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(t)
		}
		fmt.Println()
	} else {
		fmt.Printf("  Templates:  (all allowed)\n")
	}
	if members, ok := c["members"].([]interface{}); ok {
		fmt.Printf("  Members:    %d\n", len(members))
	}
}

// --- course close ---

func (a *App) courseClose(hc *apiclient.HTTPClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism course close <id>")
	}
	if err := hc.CloseCourse(a.ctx, args[0]); err != nil {
		return fmt.Errorf("failed to close course: %w", err)
	}
	fmt.Printf("Course %s closed.\n", args[0])
	return nil
}

// --- course delete ---

func (a *App) courseDelete(hc *apiclient.HTTPClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism course delete <id>")
	}
	if err := hc.DeleteCourse(a.ctx, args[0]); err != nil {
		return fmt.Errorf("failed to delete course: %w", err)
	}
	fmt.Printf("Course %s deleted.\n", args[0])
	return nil
}

// --- course templates ---

func (a *App) courseTemplates(hc *apiclient.HTTPClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: prism course templates <list|add|remove> <course-id> [template-slug]")
	}
	action := args[0]
	courseID := args[1]
	switch action {
	case "list":
		result, err := hc.ListCourseTemplates(a.ctx, courseID)
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}
		templates, _ := result["approved_templates"].([]interface{})
		if len(templates) == 0 {
			fmt.Println("No whitelist set — all templates allowed.")
			return nil
		}
		fmt.Printf("Approved templates for course %s:\n", courseID)
		for _, t := range templates {
			fmt.Printf("  • %s\n", t)
		}
	case "add":
		if len(args) < 3 {
			return fmt.Errorf("usage: prism course templates add <course-id> <template-slug>")
		}
		if err := hc.AddCourseTemplate(a.ctx, courseID, args[2]); err != nil {
			return fmt.Errorf("failed to add template: %w", err)
		}
		fmt.Printf("Template %q added to course %s.\n", args[2], courseID)
	case "remove":
		if len(args) < 3 {
			return fmt.Errorf("usage: prism course templates remove <course-id> <template-slug>")
		}
		if err := hc.RemoveCourseTemplate(a.ctx, courseID, args[2]); err != nil {
			return fmt.Errorf("failed to remove template: %w", err)
		}
		fmt.Printf("Template %q removed from course %s.\n", args[2], courseID)
	default:
		return fmt.Errorf("unknown templates action: %s", action)
	}
	return nil
}

// --- course students ---

// courseStudentsList implements "prism course students list".
func (a *App) courseStudentsList(hc *apiclient.HTTPClient, courseID string, rest []string) error {
	role := ""
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--role" && i+1 < len(rest) {
			role = rest[i+1]
			i++
		}
	}
	result, err := hc.ListCourseMembers(a.ctx, courseID, role)
	if err != nil {
		return fmt.Errorf("failed to list members: %w", err)
	}
	members, _ := result["members"].([]interface{})
	if len(members) == 0 {
		fmt.Println("No members enrolled.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "USER ID\tEMAIL\tROLE\tBUDGET SPENT\tBUDGET LIMIT")
	fmt.Fprintln(w, "-------\t-----\t----\t------------\t------------")
	for _, raw := range members {
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		spent := 0.0
		limit := 0.0
		if v, ok := m["budget_spent"].(float64); ok {
			spent = v
		}
		if v, ok := m["budget_limit"].(float64); ok {
			limit = v
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t$%.2f\t$%.2f\n",
			strVal(m["user_id"]),
			strVal(m["email"]),
			strVal(m["role"]),
			spent, limit,
		)
	}
	return w.Flush()
}

// courseStudentsEnroll implements "prism course students enroll".
func (a *App) courseStudentsEnroll(hc *apiclient.HTTPClient, courseID string, rest []string) error {
	req := map[string]interface{}{"role": "student"}
	for i := 0; i < len(rest); i++ {
		switch {
		case rest[i] == "--email" && i+1 < len(rest):
			req["email"] = rest[i+1]
			i++
		case rest[i] == "--user" && i+1 < len(rest):
			req["user_id"] = rest[i+1]
			i++
		case rest[i] == "--name" && i+1 < len(rest):
			req["display_name"] = rest[i+1]
			i++
		case rest[i] == "--role" && i+1 < len(rest):
			req["role"] = rest[i+1]
			i++
		case rest[i] == "--budget" && i+1 < len(rest):
			if v, err := strconv.ParseFloat(rest[i+1], 64); err == nil {
				req["budget_limit"] = v
			}
			i++
		}
	}
	result, err := hc.EnrollCourseMember(a.ctx, courseID, req)
	if err != nil {
		return fmt.Errorf("failed to enroll member: %w", err)
	}
	fmt.Printf("Enrolled %s as %s in course %s.\n",
		strVal(result["user_id"]), strVal(result["role"]), courseID)
	return nil
}

func (a *App) courseStudents(hc *apiclient.HTTPClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: prism course students <list|enroll|unenroll> <course-id> [options]")
	}
	action := args[0]
	courseID := args[1]
	rest := args[2:]
	switch action {
	case "list":
		return a.courseStudentsList(hc, courseID, rest)
	case "enroll":
		return a.courseStudentsEnroll(hc, courseID, rest)
	case "unenroll":
		if len(rest) < 1 {
			return fmt.Errorf("usage: prism course students unenroll <course-id> <user-id>")
		}
		if err := hc.UnenrollCourseMember(a.ctx, courseID, rest[0]); err != nil {
			return fmt.Errorf("failed to unenroll member: %w", err)
		}
		fmt.Printf("Member %s unenrolled from course %s.\n", rest[0], courseID)
	case "import":
		// prism course students import <course-id> <file> [--format canvas|blackboard|prism]
		if len(rest) < 1 {
			return fmt.Errorf("usage: prism course students import <course-id> <file> [--format canvas|blackboard|prism]")
		}
		filePath := rest[0]
		format := ""
		for i := 1; i < len(rest); i++ {
			if rest[i] == "--format" && i+1 < len(rest) {
				format = rest[i+1]
				i++
			}
		}
		csvBytes, err := os.ReadFile(filePath) //nolint:gosec // G304: roster CSV path supplied by the user on the command line
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		result, err := hc.ImportCourseRoster(a.ctx, courseID, csvBytes, format)
		if err != nil {
			return fmt.Errorf("roster import failed: %w", err)
		}
		enrolled := 0.0
		if v, ok := result["enrolled"].(float64); ok {
			enrolled = v
		}
		fmt.Printf("Imported %d students into course %s.\n", int(enrolled), courseID)
	case "provision":
		// prism course students provision <course-id> <user-id> [--template <slug>] [--name <name>]
		if len(rest) < 1 {
			return fmt.Errorf("usage: prism course students provision <course-id> <user-id> [--template <slug>] [--name <name>]")
		}
		studentID := rest[0]
		req := map[string]interface{}{}
		for i := 1; i < len(rest); i++ {
			switch {
			case rest[i] == "--template" && i+1 < len(rest):
				req["template"] = rest[i+1]
				i++
			case rest[i] == "--name" && i+1 < len(rest):
				req["instance_name"] = rest[i+1]
				i++
			}
		}
		result, err := hc.ProvisionStudent(a.ctx, courseID, studentID, req)
		if err != nil {
			return fmt.Errorf("provisioning failed: %w", err)
		}
		fmt.Printf("Provisioning workspace for student %s in course %s (template: %s, name: %s).\n",
			studentID, courseID, strVal(result["template"]), strVal(result["instance_name"]))
	default:
		return fmt.Errorf("unknown students action: %s", action)
	}
	return nil
}

// --- course budget ---

func (a *App) courseBudget(hc *apiclient.HTTPClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: prism course budget <show|distribute> <course-id> [--amount <n>]")
	}
	action := args[0]
	courseID := args[1]
	rest := args[2:]
	switch action {
	case "show":
		result, err := hc.GetCourseBudget(a.ctx, courseID)
		if err != nil {
			return fmt.Errorf("failed to get budget: %w", err)
		}
		fmt.Printf("Budget summary for course %s:\n", courseID)
		fmt.Printf("  Total budget:      $%.2f\n", floatVal(result["total_budget"]))
		fmt.Printf("  Total spent:       $%.2f\n", floatVal(result["total_spent"]))
		fmt.Printf("  Per-student limit: $%.2f\n", floatVal(result["per_student_limit"]))
		if students, ok := result["students"].([]interface{}); ok && len(students) > 0 {
			fmt.Printf("\n  %-20s %-10s %-10s %-10s\n", "User", "Limit", "Spent", "Remaining")
			for _, raw := range students {
				s, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}
				fmt.Printf("  %-20s $%-9.2f $%-9.2f $%-9.2f\n",
					strVal(s["user_id"]),
					floatVal(s["budget_limit"]),
					floatVal(s["budget_spent"]),
					floatVal(s["remaining"]),
				)
			}
		}
	case "distribute":
		amount := 0.0
		for i := 0; i < len(rest); i++ {
			if rest[i] == "--amount" && i+1 < len(rest) {
				if v, err := strconv.ParseFloat(rest[i+1], 64); err == nil {
					amount = v
				}
				i++
			}
		}
		if amount <= 0 {
			return fmt.Errorf("--amount <n> is required and must be > 0")
		}
		result, err := hc.DistributeCourseBudget(a.ctx, courseID, amount)
		if err != nil {
			return fmt.Errorf("failed to distribute budget: %w", err)
		}
		count := floatVal(result["students_updated"])
		fmt.Printf("Budget set to $%.2f/student for %d student(s) in course %s.\n",
			amount, int(count), courseID)
	default:
		return fmt.Errorf("unknown budget action: %s", action)
	}
	return nil
}

// --- ta debug ---

func (a *App) taDebug(hc *apiclient.HTTPClient, args []string) error {
	courseID := ""
	studentID := ""
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--course" && i+1 < len(args):
			courseID = args[i+1]
			i++
		case args[i] == "--student" && i+1 < len(args):
			studentID = args[i+1]
			i++
		default:
			if courseID == "" {
				courseID = args[i]
			} else if studentID == "" {
				studentID = args[i]
			}
		}
	}
	if courseID == "" || studentID == "" {
		return fmt.Errorf("usage: prism ta debug <course-id> <student-id>")
	}
	result, err := hc.TADebugStudent(a.ctx, courseID, studentID)
	if err != nil {
		return fmt.Errorf("failed to get debug info: %w", err)
	}
	fmt.Printf("Debug info for student %s in course %s:\n", strVal(result["student_id"]), strVal(result["course_id"]))
	fmt.Printf("  Budget spent:  $%.2f\n", floatVal(result["budget_spent"]))
	fmt.Printf("  Budget limit:  $%.2f\n", floatVal(result["budget_limit"]))
	if instances, ok := result["instances"].([]interface{}); ok {
		fmt.Printf("  Instances:     %d\n", len(instances))
		for _, raw := range instances {
			inst, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			fmt.Printf("    • %s (%s) — %s\n",
				strVal(inst["name"]), strVal(inst["instance_type"]), strVal(inst["state"]))
		}
	}
	if events, ok := result["recent_events"].([]interface{}); ok && len(events) > 0 {
		fmt.Printf("  Recent events:\n")
		for _, e := range events {
			fmt.Printf("    • %s\n", e)
		}
	}
	return nil
}

// --- ta reset ---

func (a *App) taReset(hc *apiclient.HTTPClient, args []string) error {
	courseID := ""
	studentID := ""
	reason := ""
	retention := 7
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--course" && i+1 < len(args):
			courseID = args[i+1]
			i++
		case args[i] == "--student" && i+1 < len(args):
			studentID = args[i+1]
			i++
		case args[i] == "--reason" && i+1 < len(args):
			reason = args[i+1]
			i++
		case args[i] == "--retention" && i+1 < len(args):
			if v, err := strconv.Atoi(args[i+1]); err == nil {
				retention = v
			}
			i++
		default:
			if courseID == "" {
				courseID = args[i]
			} else if studentID == "" {
				studentID = args[i]
			}
		}
	}
	if courseID == "" || studentID == "" {
		return fmt.Errorf("usage: prism ta reset <course-id> <student-id> --reason <reason>")
	}
	if reason == "" {
		return fmt.Errorf("--reason is required for audit trail")
	}
	req := map[string]interface{}{
		"student_id":            studentID,
		"reason":                reason,
		"backup_retention_days": retention,
	}
	result, err := hc.TAResetStudent(a.ctx, courseID, req)
	if err != nil {
		return fmt.Errorf("failed to reset student instance: %w", err)
	}
	fmt.Printf("Reset initiated for student %s: %s\n", studentID, strVal(result["message"]))
	return nil
}

// --- course archive ---

func (a *App) courseArchive(hc *apiclient.HTTPClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism course archive <course-id>")
	}
	courseID := args[0]
	result, err := hc.ArchiveCourse(a.ctx, courseID)
	if err != nil {
		return fmt.Errorf("archive failed: %w", err)
	}
	stopped, _ := result["instances_stopped"].([]interface{})
	errs, _ := result["errors"].([]interface{})
	fmt.Printf("Course %s archived. Instances stopped: %d", courseID, len(stopped))
	if len(errs) > 0 {
		fmt.Printf(", errors: %d", len(errs))
	}
	fmt.Println()
	return nil
}

// --- course report ---

func (a *App) courseReport(hc *apiclient.HTTPClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism course report <course-id> [--format json|csv] [--output <file>]")
	}
	courseID := args[0]
	format := ""
	outputFile := ""
	for i := 1; i < len(args); i++ {
		switch {
		case args[i] == "--format" && i+1 < len(args):
			format = args[i+1]
			i++
		case args[i] == "--output" && i+1 < len(args):
			outputFile = args[i+1]
			i++
		}
	}

	result, err := hc.GetCourseReport(a.ctx, courseID, format)
	if err != nil {
		return fmt.Errorf("report failed: %w", err)
	}

	if outputFile != "" {
		data, _ := json.MarshalIndent(result, "", "  ")
		if err := os.WriteFile(outputFile, data, 0600); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		fmt.Printf("Report written to %s.\n", outputFile)
		return nil
	}

	// Print table
	students, _ := result["students"].([]interface{})
	if len(students) == 0 {
		fmt.Println("No student usage data.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "USER\tEMAIL\tSPENT\tBUDGET\tHOURS\tINSTANCES")
	fmt.Fprintln(w, "----\t-----\t-----\t------\t-----\t---------")
	for _, raw := range students {
		s, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t$%.2f\t$%.2f\t%.1f\t%d\n",
			strTrunc(s["user_id"], 16),
			strTrunc(s["email"], 24),
			floatVal(s["total_spent"]),
			floatVal(s["budget_limit"]),
			floatVal(s["instance_hours"]),
			int(floatVal(s["instance_count"])),
		)
	}
	return w.Flush()
}

// --- course audit ---

func (a *App) courseAudit(hc *apiclient.HTTPClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism course audit <course-id> [--student <id>] [--since 2026-01-01] [--limit 100]")
	}
	courseID := args[0]
	studentID := ""
	since := ""
	limit := 0
	for i := 1; i < len(args); i++ {
		switch {
		case args[i] == "--student" && i+1 < len(args):
			studentID = args[i+1]
			i++
		case args[i] == "--since" && i+1 < len(args):
			since = args[i+1]
			// normalize date-only to RFC3339
			if len(since) == 10 {
				since = since + "T00:00:00Z"
			}
			i++
		case args[i] == "--limit" && i+1 < len(args):
			if n, err := strconv.Atoi(args[i+1]); err == nil {
				limit = n
			}
			i++
		}
	}

	result, err := hc.GetCourseAuditLog(a.ctx, courseID, studentID, since, limit)
	if err != nil {
		return fmt.Errorf("audit query failed: %w", err)
	}

	entries, _ := result["entries"].([]interface{})
	if len(entries) == 0 {
		fmt.Println("No audit entries found.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIMESTAMP\tACTOR\tACTION\tTARGET")
	fmt.Fprintln(w, "---------\t-----\t------\t------")
	for _, raw := range entries {
		e, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		ts := strVal(e["timestamp"])
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			ts = t.Local().Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			ts,
			strTrunc(e["actor"], 16),
			strVal(e["action"]),
			strVal(e["target"]),
		)
	}
	return w.Flush()
}

// --- ta overview ---

func (a *App) taOverview(hc *apiclient.HTTPClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism ta overview <course-id>")
	}
	courseID := args[0]
	result, err := hc.GetCourseOverview(a.ctx, courseID)
	if err != nil {
		return fmt.Errorf("overview failed: %w", err)
	}

	fmt.Printf("Course: %s  Students: %d  Active Instances: %d  Total Spent: $%.2f\n\n",
		strVal(result["course_code"]),
		int(floatVal(result["total_students"])),
		int(floatVal(result["active_instances"])),
		floatVal(result["total_budget_spent"]),
	)

	students, _ := result["students"].([]interface{})
	if len(students) == 0 {
		fmt.Println("No students enrolled.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "USER\tEMAIL\tSTATUS\tSPENT\tLIMIT\tINSTANCES")
	fmt.Fprintln(w, "----\t-----\t------\t-----\t-----\t---------")
	for _, raw := range students {
		s, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		instCount := 0
		if insts, ok := s["instances"].([]interface{}); ok {
			instCount = len(insts)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t$%.2f\t$%.2f\t%d\n",
			strTrunc(s["user_id"], 16),
			strTrunc(s["email"], 24),
			strVal(s["budget_status"]),
			floatVal(s["budget_spent"]),
			floatVal(s["budget_limit"]),
			instCount,
		)
	}
	return w.Flush()
}

// --- v0.19.0: TA Access, Materials, Reset Workspace ---

// courseTAAccess handles 'course ta-access <sub> ...'
func (a *App) courseTAAccess(hc *apiclient.HTTPClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: course ta-access <list|grant|revoke|connect> <course-id> [flags]")
	}
	sub := args[0]
	rest := args[1:]

	flags := parseSimpleFlags(rest)
	courseID := ""
	if len(rest) > 0 && !strings.HasPrefix(rest[0], "--") {
		courseID = rest[0]
	}

	switch sub {
	case "list":
		if courseID == "" {
			return fmt.Errorf("course-id required")
		}
		data, err := hc.MakeRequest("GET", fmt.Sprintf("/api/v1/courses/%s/ta-access", courseID), nil)
		if err != nil {
			return err
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(data, &resp); err != nil {
			return err
		}
		members, _ := resp["ta_members"].([]interface{})
		if len(members) == 0 {
			fmt.Println("No TAs configured for this course.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "EMAIL\tDISPLAY NAME\tADDED")
		for _, m := range members {
			mb, _ := m.(map[string]interface{})
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				strVal(mb["email"]),
				strVal(mb["display_name"]),
				strVal(mb["added_at"]),
			)
		}
		return w.Flush()

	case "grant":
		if courseID == "" {
			return fmt.Errorf("course-id required")
		}
		email := flags["--email"]
		if email == "" {
			return fmt.Errorf("--email is required")
		}
		body := map[string]string{"email": email, "display_name": flags["--name"]}
		if _, err := hc.MakeRequest("POST", fmt.Sprintf("/api/v1/courses/%s/ta-access", courseID), body); err != nil {
			return err
		}
		fmt.Printf("TA access granted to %s\n", email)
		return nil

	case "revoke":
		if courseID == "" {
			return fmt.Errorf("course-id required")
		}
		email := flags["--email"]
		if email == "" {
			return fmt.Errorf("--email is required")
		}
		if _, err := hc.MakeRequest("DELETE", fmt.Sprintf("/api/v1/courses/%s/ta-access/%s", courseID, email), nil); err != nil {
			return err
		}
		fmt.Printf("TA access revoked for %s\n", email)
		return nil

	case "connect":
		if courseID == "" {
			return fmt.Errorf("course-id required")
		}
		student := flags["--student"]
		reason := flags["--reason"]
		if student == "" || reason == "" {
			return fmt.Errorf("--student and --reason are required")
		}
		body := map[string]string{"student_id": student, "reason": reason}
		data, err := hc.MakeRequest("POST", fmt.Sprintf("/api/v1/courses/%s/ta-access/connect", courseID), body)
		if err != nil {
			return err
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(data, &resp); err != nil {
			return err
		}
		fmt.Printf("SSH command: %s\n", strVal(resp["ssh_command"]))
		if note := strVal(resp["note"]); note != "" {
			fmt.Printf("Note: %s\n", note)
		}
		return nil

	default:
		return fmt.Errorf("unknown ta-access sub-command: %s", sub)
	}
}

// courseMaterials handles 'course materials <sub> ...'
func (a *App) courseMaterials(hc *apiclient.HTTPClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: course materials <create|list|mount> <course-id> [flags]")
	}
	sub := args[0]
	rest := args[1:]
	flags := parseSimpleFlags(rest)
	courseID := ""
	if len(rest) > 0 && !strings.HasPrefix(rest[0], "--") {
		courseID = rest[0]
	}

	switch sub {
	case "list":
		if courseID == "" {
			return fmt.Errorf("course-id required")
		}
		data, err := hc.MakeRequest("GET", fmt.Sprintf("/api/v1/courses/%s/materials", courseID), nil)
		if err != nil {
			return err
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(data, &resp); err != nil {
			return err
		}
		if resp["materials"] == nil {
			fmt.Println("No shared materials volume configured for this course.")
			return nil
		}
		vol, _ := resp["materials"].(map[string]interface{})
		fmt.Printf("EFS ID:      %s\n", strVal(vol["efs_id"]))
		fmt.Printf("Size:        %v GB\n", vol["size_gb"])
		fmt.Printf("Mount Path:  %s\n", strVal(vol["mount_path"]))
		fmt.Printf("State:       %s\n", strVal(vol["state"]))
		return nil

	case "create":
		if courseID == "" {
			return fmt.Errorf("course-id required")
		}
		sizeStr := flags["--size"]
		if sizeStr == "" {
			sizeStr = "50"
		}
		size, err := strconv.Atoi(sizeStr)
		if err != nil || size <= 0 {
			return fmt.Errorf("invalid --size %q: must be a positive integer (GB)", sizeStr)
		}
		mountPath := flags["--mount"]
		if mountPath == "" {
			mountPath = "/mnt/course-materials"
		}
		body := map[string]interface{}{"size_gb": size, "mount_path": mountPath}
		data, err := hc.MakeRequest("POST", fmt.Sprintf("/api/v1/courses/%s/materials", courseID), body)
		if err != nil {
			return err
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(data, &resp); err != nil {
			return err
		}
		if vol, ok := resp["materials"].(map[string]interface{}); ok {
			fmt.Printf("Created EFS: %s (mount: %s, size: %v GB)\n",
				strVal(vol["efs_id"]), strVal(vol["mount_path"]), vol["size_gb"])
		}
		return nil

	case "mount":
		if courseID == "" {
			return fmt.Errorf("course-id required")
		}
		data, err := hc.MakeRequest("POST", fmt.Sprintf("/api/v1/courses/%s/materials/mount", courseID), nil)
		if err != nil {
			return err
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(data, &resp); err != nil {
			return err
		}
		fmt.Printf("%s\n", strVal(resp["status"]))
		if note := strVal(resp["note"]); note != "" {
			fmt.Printf("Note: %s\n", note)
		}
		return nil

	default:
		return fmt.Errorf("unknown materials sub-command: %s", sub)
	}
}

// courseResetWorkspace handles 'course reset-workspace <course-id> --student ... --reason ...'
func (a *App) courseResetWorkspace(hc *apiclient.HTTPClient, args []string) error {
	flags := parseSimpleFlags(args)
	courseID := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		courseID = args[0]
	}
	student := flags["--student"]
	reason := flags["--reason"]
	if courseID == "" || student == "" || reason == "" {
		return fmt.Errorf("usage: course reset-workspace <course-id> --student <email> --reason <text> [--backup]")
	}
	_, backup := flags["--backup"]
	body := map[string]interface{}{
		"student_id":            student,
		"reason":                reason,
		"backup_retention_days": 7,
	}

	data, err := hc.MakeRequest("POST", fmt.Sprintf("/api/v1/courses/%s/ta/reset/%s", courseID, student), body)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	fmt.Printf("Reset scheduled for student %s\n", student)
	if backup {
		if url := strVal(resp["backup_download_url"]); url != "" {
			fmt.Printf("Backup URL: %s\n", url)
		}
	}
	return nil
}

// parseSimpleFlags parses "--key value" pairs from an args slice into a map.
// Presence-only flags (no value) are stored with an empty string value.
func parseSimpleFlags(args []string) map[string]string {
	m := make(map[string]string)
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			key := args[i]
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				m[key] = args[i+1]
				i++
			} else {
				m[key] = ""
			}
		}
	}
	return m
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func strVal(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func strTrunc(v interface{}, n int) string {
	s := strVal(v)
	if len(s) > n {
		return s[:n]
	}
	return s
}

func floatVal(v interface{}) float64 {
	if v == nil {
		return 0
	}
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}
