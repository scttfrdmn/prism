package fixtures

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// WaitForRStudioServer waits for RStudio Server to be accessible via HTTP
func WaitForRStudioServer(t *testing.T, host string, port int, timeout time.Duration) error {
	t.Helper()

	deadline := time.Now().Add(timeout)
	rstudioURL := fmt.Sprintf("http://%s:%d", host, port)

	t.Logf("Waiting for RStudio Server at %s (timeout: %v)", rstudioURL, timeout)

	for time.Now().Before(deadline) {
		resp, err := http.Get(rstudioURL) //nolint:gosec // G107: URL is the test instance endpoint under test
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close() // probe only; body unused, close error irrelevant
			t.Logf("✓ RStudio Server is accessible at %s", rstudioURL)
			return nil
		}

		if err != nil {
			t.Logf("RStudio Server not ready: %v (retrying...)", err)
		} else {
			_ = resp.Body.Close() // probe only; body unused, close error irrelevant
			t.Logf("RStudio Server returned %d (waiting for 200)", resp.StatusCode)
		}

		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timeout waiting for RStudio Server at %s after %v", rstudioURL, timeout)
}

// CreateRStudioUser creates a user configured for RStudio Server access
func CreateRStudioUser(t *testing.T, host, existingUser, newUsername, password string) error {
	t.Helper()

	t.Logf("Creating RStudio user: %s", newUsername)

	// Commands to create user with RStudio access
	commands := []string{
		// Create user without interactive prompts
		fmt.Sprintf("sudo adduser --disabled-password --gecos '' %s", newUsername),
		// Set password
		fmt.Sprintf("echo '%s:%s' | sudo chpasswd", newUsername, password),
		// Add to sudo group (RStudio Server auth-required-user-group)
		fmt.Sprintf("sudo usermod -aG sudo %s", newUsername),
	}

	for _, cmd := range commands {
		_, err := SSHCommand(t, host, existingUser, cmd)
		if err != nil {
			return fmt.Errorf("failed to execute command '%s': %w", cmd, err)
		}
	}

	t.Logf("✓ User %s created and configured for RStudio access", newUsername)
	return nil
}

// CreateSharedDirectory creates a directory accessible to multiple users
func CreateSharedDirectory(t *testing.T, host, user, dirPath, groupName string) error {
	t.Helper()

	t.Logf("Creating shared directory: %s (group: %s)", dirPath, groupName)

	commands := []string{
		// Create directory
		fmt.Sprintf("sudo mkdir -p %s", dirPath),
		// Change group ownership
		fmt.Sprintf("sudo chgrp %s %s", groupName, dirPath),
		// Set permissions: rwxrwsr-x (2775)
		// The 's' (setgid) ensures new files inherit the group
		fmt.Sprintf("sudo chmod 2775 %s", dirPath),
	}

	for _, cmd := range commands {
		_, err := SSHCommand(t, host, user, cmd)
		if err != nil {
			return fmt.Errorf("failed to execute command '%s': %w", cmd, err)
		}
	}

	t.Logf("✓ Shared directory created: %s", dirPath)
	return nil
}

// VerifyRStudioLogin tests authentication to RStudio Server
// Returns true if login succeeds, false otherwise
func VerifyRStudioLogin(t *testing.T, host string, port int, username, password string) bool {
	t.Helper()

	baseURL := fmt.Sprintf("http://%s:%d", host, port)
	t.Logf("Testing RStudio Server login for user: %s at %s", username, baseURL)

	// First, verify login page is accessible
	loginPageURL := baseURL + "/auth-sign-in"
	resp, err := http.Get(loginPageURL) //nolint:gosec // G107: URL is the test instance endpoint under test
	if err != nil {
		t.Logf("Failed to access login page: %v", err)
		return false
	}
	_ = resp.Body.Close() // probe only; body unused, close error irrelevant

	if resp.StatusCode != http.StatusOK {
		t.Logf("Login page returned unexpected status: %d", resp.StatusCode)
		return false
	}

	// Attempt authentication
	// RStudio Server uses form-based auth with POST to /auth-do-sign-in
	loginData := url.Values{
		"username": {username},
		"password": {password},
	}

	client := &http.Client{
		// Don't follow redirects - we want to check if login succeeded
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	loginURL := baseURL + "/auth-do-sign-in"
	resp, err = client.PostForm(loginURL, loginData)
	if err != nil {
		t.Logf("Failed to submit login form: %v", err)
		return false
	}
	defer resp.Body.Close()

	// Successful login typically results in a redirect (302 or 303)
	// or a 200 OK with session cookie
	if resp.StatusCode == http.StatusFound ||
		resp.StatusCode == http.StatusSeeOther ||
		resp.StatusCode == http.StatusOK {
		t.Logf("✓ RStudio Server login successful for user: %s", username)
		return true
	}

	t.Logf("RStudio Server login failed with status: %d", resp.StatusCode)
	return false
}

// VerifyFileAccess tests read/write access to shared files via SSH
func VerifyFileAccess(t *testing.T, host, username, filepath string, expectRead, expectWrite bool) error {
	t.Helper()

	t.Logf("Verifying file access for user %s: %s (read=%v, write=%v)",
		username, filepath, expectRead, expectWrite)

	// Test read access
	if expectRead {
		cmd := fmt.Sprintf("sudo -u %s test -r %s && echo 'readable' || echo 'not-readable'", username, filepath)
		output, err := SSHCommand(t, host, "researcher", cmd)
		if err != nil {
			return fmt.Errorf("failed to test read access: %w", err)
		}

		if !strings.Contains(output, "readable") {
			return fmt.Errorf("user %s cannot read %s (expected readable)", username, filepath)
		}
		t.Logf("  ✓ User %s can read %s", username, filepath)
	}

	// Test write access to parent directory
	if expectWrite {
		// Extract directory from filepath
		dirPath := filepath
		if strings.Contains(filepath, "/") {
			parts := strings.Split(filepath, "/")
			dirPath = strings.Join(parts[:len(parts)-1], "/")
			if dirPath == "" {
				dirPath = "/"
			}
		}

		cmd := fmt.Sprintf("sudo -u %s test -w %s && echo 'writable' || echo 'not-writable'", username, dirPath)
		output, err := SSHCommand(t, host, "researcher", cmd)
		if err != nil {
			return fmt.Errorf("failed to test write access: %w", err)
		}

		if !strings.Contains(output, "writable") {
			return fmt.Errorf("user %s cannot write to %s (expected writable)", username, dirPath)
		}
		t.Logf("  ✓ User %s can write to %s", username, dirPath)
	}

	return nil
}

// CreateSharedRProject creates an R project in a shared directory
func CreateSharedRProject(t *testing.T, host, user, projectPath, projectName string) error {
	t.Helper()

	fullPath := fmt.Sprintf("%s/%s", projectPath, projectName)
	t.Logf("Creating shared R project: %s", fullPath)

	commands := []string{
		// Create project directory
		fmt.Sprintf("sudo -u %s mkdir -p %s", user, fullPath),
		// Create a simple R script
		fmt.Sprintf("echo 'data <- read.csv(\"data.csv\")' | sudo -u %s tee %s/analysis.R > /dev/null", user, fullPath),
		// Create .Rproj file (makes it an RStudio project)
		fmt.Sprintf("echo 'Version: 1.0\n\nRestoreWorkspace: Default\nSaveWorkspace: Default\nAlwaysSaveHistory: Default' | sudo -u %s tee %s/%s.Rproj > /dev/null", user, fullPath, projectName),
	}

	for _, cmd := range commands {
		_, err := SSHCommand(t, host, "researcher", cmd)
		if err != nil {
			return fmt.Errorf("failed to execute command: %w", err)
		}
	}

	t.Logf("✓ R project created: %s", fullPath)
	return nil
}

// VerifyRSessionWorks tests if R is functional via RStudio Server
// This is a basic smoke test using SSH to verify R is installed and working
func VerifyRSessionWorks(t *testing.T, host, user string) error {
	t.Helper()

	t.Logf("Verifying R session works for user: %s", user)

	// Execute a simple R command via SSH
	cmd := "R --version"
	output, err := SSHCommand(t, host, user, cmd)
	if err != nil {
		return fmt.Errorf("running R failed: %w", err)
	}

	if !strings.Contains(output, "R version") {
		return fmt.Errorf("unexpected R version output: %s", output)
	}

	// Test basic R functionality
	cmd = "Rscript -e 'cat(\"R works\")'"
	output, err = SSHCommand(t, host, user, cmd)
	if err != nil {
		return fmt.Errorf("running Rscript failed: %w", err)
	}

	if !strings.Contains(output, "R works") {
		return fmt.Errorf("the R script did not execute correctly: %s", output)
	}

	t.Logf("✓ R session verified working")
	return nil
}
