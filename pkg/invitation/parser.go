// Package invitation provides file parsing utilities for bulk invitations
//
// This file implements CSV and plain text parsing for bulk invitation operations,
// supporting both simple email lists and structured CSV formats with roles and messages.
package invitation

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/scttfrdmn/prism/pkg/types"
)

// ParseInvitationFile parses a file containing invitation data
// Automatically detects format (CSV or plain text) based on file extension
func ParseInvitationFile(filePath string) ([]types.BulkInvitationEntry, error) {
	file, err := os.Open(filePath) //nolint:gosec // G304: path supplied by the local CLI user (--file flag)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }() // read-only; close error carries no data-loss risk

	// Detect format based on file extension
	if strings.HasSuffix(strings.ToLower(filePath), ".csv") {
		return ParseCSV(file)
	}

	// Default to plain text format
	return ParsePlainText(file)
}

// parseCSVOptionalFields fills in optional role and message from CSV record columns 2 and 3.
func parseCSVOptionalFields(record []string, entry *types.BulkInvitationEntry) {
	if len(record) > 1 {
		if role := strings.TrimSpace(record[1]); role != "" {
			entry.Role = types.ProjectRole(role)
		}
	}
	if len(record) > 2 {
		if message := strings.TrimSpace(record[2]); message != "" {
			entry.Message = message
		}
	}
}

// ParseCSV parses a CSV file with format: email,role,message
// First row is treated as header if it contains "email"
// Role and message columns are optional
func ParseCSV(reader io.Reader) ([]types.BulkInvitationEntry, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true
	csvReader.Comment = '#'

	var entries []types.BulkInvitationEntry
	lineNum := 0

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("CSV parse error at line %d: %w", lineNum+1, err)
		}

		lineNum++

		// Skip empty lines
		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue
		}

		// Check if first row is a header and skip it
		if lineNum == 1 && strings.ToLower(strings.TrimSpace(record[0])) == "email" {
			continue
		}

		// Parse email (required)
		email := strings.TrimSpace(record[0])
		if email == "" {
			return nil, fmt.Errorf("missing email at line %d", lineNum)
		}

		// Validate email format
		if !isValidEmail(email) {
			return nil, fmt.Errorf("invalid email format at line %d: %s", lineNum, email)
		}

		entry := types.BulkInvitationEntry{Email: email}
		parseCSVOptionalFields(record, &entry)
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no valid entries found in CSV file")
	}

	return entries, nil
}

// ParsePlainText parses a plain text file with one email per line
// Lines starting with # are treated as comments
// Empty lines are ignored
func ParsePlainText(reader io.Reader) ([]types.BulkInvitationEntry, error) {
	scanner := bufio.NewScanner(reader)
	var entries []types.BulkInvitationEntry
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Validate email format
		if !isValidEmail(line) {
			return nil, fmt.Errorf("invalid email format at line %d: %s", lineNum, line)
		}

		entries = append(entries, types.BulkInvitationEntry{
			Email: line,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no valid email addresses found in file")
	}

	return entries, nil
}

// ParseInlineEmails parses a comma-separated or whitespace-separated list of emails
// Useful for CLI inline email lists: "alice@example.com, bob@example.com"
func ParseInlineEmails(emailList string) ([]types.BulkInvitationEntry, error) {
	// Split by comma or whitespace
	emails := strings.FieldsFunc(emailList, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})

	var entries []types.BulkInvitationEntry
	for i, email := range emails {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}

		// Validate email format
		if !isValidEmail(email) {
			return nil, fmt.Errorf("invalid email format at position %d: %s", i+1, email)
		}

		entries = append(entries, types.BulkInvitationEntry{
			Email: email,
		})
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no valid email addresses found")
	}

	return entries, nil
}

// isValidEmail performs basic email validation
// Format: local@domain
func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}

	// Must contain exactly one @
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	// Local part must not be empty
	local := strings.TrimSpace(parts[0])
	if local == "" {
		return false
	}

	// Domain part must not be empty and must contain at least one dot
	domain := strings.TrimSpace(parts[1])
	if domain == "" || !strings.Contains(domain, ".") {
		return false
	}

	// Basic domain validation: must end with a valid TLD-like pattern
	if strings.HasSuffix(domain, ".") {
		return false
	}

	return true
}

// ValidateRoles checks if all roles in entries are valid
func ValidateRoles(entries []types.BulkInvitationEntry) error {
	validRoles := map[types.ProjectRole]bool{
		types.ProjectRoleOwner:  true,
		types.ProjectRoleAdmin:  true,
		types.ProjectRoleMember: true,
		types.ProjectRoleViewer: true,
	}

	for i, entry := range entries {
		if entry.Role != "" && !validRoles[entry.Role] {
			return fmt.Errorf("invalid role at entry %d: %s (valid: owner, admin, member, viewer)", i+1, entry.Role)
		}
	}

	return nil
}

// FormatSummary formats bulk invitation results as a human-readable summary
func FormatSummary(response *types.BulkInvitationResponse) string {
	var sb strings.Builder

	sb.WriteString("📬 Bulk Invitation Summary\n\n")
	sb.WriteString(fmt.Sprintf("Total:   %d invitations\n", response.Summary.Total))
	sb.WriteString(fmt.Sprintf("✅ Sent:    %d (%d%%)\n", response.Summary.Sent, percentage(response.Summary.Sent, response.Summary.Total)))
	sb.WriteString(fmt.Sprintf("⏭️  Skipped: %d (%d%%)\n", response.Summary.Skipped, percentage(response.Summary.Skipped, response.Summary.Total)))
	sb.WriteString(fmt.Sprintf("❌ Failed:  %d (%d%%)\n", response.Summary.Failed, percentage(response.Summary.Failed, response.Summary.Total)))

	// Show details for skipped and failed
	if response.Summary.Skipped > 0 {
		sb.WriteString("\nSkipped:\n")
		for _, result := range response.Results {
			if result.Status == "skipped" {
				sb.WriteString(fmt.Sprintf("  • %s: %s\n", result.Email, result.Reason))
			}
		}
	}

	if response.Summary.Failed > 0 {
		sb.WriteString("\nFailed:\n")
		for _, result := range response.Results {
			if result.Status == "failed" {
				sb.WriteString(fmt.Sprintf("  • %s: %s\n", result.Email, result.Error))
			}
		}
	}

	return sb.String()
}

// percentage calculates percentage with integer division
func percentage(part, total int) int {
	if total == 0 {
		return 0
	}
	return (part * 100) / total
}
