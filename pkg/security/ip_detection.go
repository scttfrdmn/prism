package security

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DetectUserExternalIP detects the user's current external IP address for security group rules
func DetectUserExternalIP() (string, error) {
	// Try multiple services for reliability
	services := []string{
		"https://checkip.amazonaws.com",
		"https://ipv4.icanhazip.com",
		"https://api.ipify.org",
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				continue
			}

			ip := strings.TrimSpace(string(body))
			if ip != "" && len(ip) < 50 { // Basic validation
				return ip, nil
			}
		}
	}

	return "", fmt.Errorf("failed to detect external IP address from all services")
}

// GetWebInterfaceBindIP returns the appropriate bind IP for web interfaces based on security configuration
func GetWebInterfaceBindIP() string {
	config := DetermineAccessStrategy()
	return config.BindIP
}
