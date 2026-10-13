package security

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

// AccessManager provides high-level access management functionality
type AccessManager struct {
	updater *SecurityGroupUpdater
}

// NewAccessManager creates a new access manager
func NewAccessManager(ec2Client EC2ClientInterface, securityGroupID string) *AccessManager {
	return &AccessManager{
		updater: NewSecurityGroupUpdater(ec2Client, securityGroupID),
	}
}

// RefreshAccess updates security group rules for current IP
func (am *AccessManager) RefreshAccess() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("🔄 Refreshing web interface access...")

	config := DetermineAccessStrategy()

	switch config.Strategy {
	case AccessDirect:
		log.Printf("📍 Using direct IP access: %s", config.UserIP)
	case AccessSubnet:
		log.Printf("📍 Using subnet access: %s", config.SubnetCIDR)
	case AccessTunneled:
		log.Printf("🔒 Using SSH tunneling (IP detection failed)")
	}

	err := am.updater.UpdateAccessRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to update access rules: %w", err)
	}

	log.Println("✅ Access rules updated successfully")
	return nil
}

// GetAccessInfo returns current access configuration and instructions
func GetAccessInfo() *AccessInfo {
	config := DetermineAccessStrategy()

	info := &AccessInfo{
		Strategy:   config.Strategy,
		UserIP:     config.UserIP,
		SubnetCIDR: config.SubnetCIDR,
		BindIP:     config.BindIP,
		Message:    config.Message,
	}

	switch config.Strategy {
	case AccessDirect:
		info.Instructions = []string{
			fmt.Sprintf("✅ Direct access available from your IP: %s", config.UserIP),
			"📱 Web interfaces accessible at: http://<instance-ip>:8888",
			"🔄 Run 'prism access refresh' if your IP changes",
		}
	case AccessSubnet:
		info.Instructions = []string{
			fmt.Sprintf("✅ Subnet access configured: %s", config.SubnetCIDR),
			"📱 Works across DHCP changes within your network",
			"🌐 Web interfaces accessible at: http://<instance-ip>:8888",
			"🔄 Run 'prism access refresh' if you change networks",
		}
	case AccessTunneled:
		info.Instructions = []string{
			"🔒 SSH tunneling required (IP detection failed)",
			"🚇 Access Jupyter: ssh -L 8888:localhost:8888 user@<instance-ip>",
			"🚇 Access RStudio: ssh -L 8787:localhost:8787 user@<instance-ip>",
			"💻 Then open http://localhost:8888 in your browser",
		}
	}

	return info
}

// AccessInfo contains information about current access configuration
type AccessInfo struct {
	Strategy     AccessStrategy
	UserIP       string
	SubnetCIDR   string
	BindIP       string
	Message      string
	Instructions []string
}

// WatchIPChanges monitors for IP changes and updates access rules
func (am *AccessManager) WatchIPChanges(interval time.Duration, stopChan <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastIP string

	for {
		select {
		case <-ticker.C:
			currentIP, err := DetectUserExternalIP()
			if err != nil {
				log.Printf("IP check failed: %v", err)
				continue
			}

			if lastIP != "" && lastIP != currentIP {
				log.Printf("🔄 IP change detected: %s -> %s", lastIP, currentIP)
				if err := am.RefreshAccess(); err != nil {
					log.Printf("❌ Failed to update access for IP change: %v", err)
				} else {
					log.Printf("✅ Access updated for new IP: %s", currentIP)
				}
			}

			lastIP = currentIP

		case <-stopChan:
			log.Println("🛑 Stopping IP change monitoring")
			return
		}
	}
}

// ValidateWebAccess tests if web interfaces are accessible
func ValidateWebAccess(instanceIP string, ports []int) *ValidationResult {
	config := DetermineAccessStrategy()

	result := &ValidationResult{
		Strategy:      config.Strategy,
		AccessibleIPs: make(map[string]bool),
		PortsChecked:  ports,
		Timestamp:     time.Now(),
	}

	// Test actual HTTP connections to verify access
	accessible := false
	for _, port := range ports {
		if testHTTPConnection(instanceIP, port, 5*time.Second) {
			accessible = true
			result.AccessibleIPs[fmt.Sprintf("%s:%d", instanceIP, port)] = true
		}
	}

	result.DirectAccessAvailable = accessible

	switch config.Strategy {
	case AccessDirect, AccessSubnet:
		if accessible {
			result.Message = "Direct web access verified"
		} else {
			result.Message = "Direct access expected but connection failed - check security groups"
		}
	case AccessTunneled:
		if accessible {
			result.Message = "Warning: Direct access available despite tunneled strategy"
		} else {
			result.Message = "SSH tunneling required for web access (as expected)"
		}
	}

	return result
}

// testHTTPConnection tests HTTP connectivity to a host:port
func testHTTPConnection(host string, port int, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return false
	}
	defer func() { _ = conn.Close() }() // probe connection; close error is not actionable

	// Try HTTP GET request
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return false
	}
	if _, err := fmt.Fprintf(conn, "GET / HTTP/1.0\r\nHost: %s\r\n\r\n", host); err != nil {
		return false
	}

	// Read response (just check if we get something back)
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	// Check for HTTP response
	response := string(buffer[:n])
	return strings.HasPrefix(response, "HTTP/")
}

// ValidationResult contains results of web access validation
type ValidationResult struct {
	Strategy              AccessStrategy
	DirectAccessAvailable bool
	AccessibleIPs         map[string]bool
	PortsChecked          []int
	Message               string
	Timestamp             time.Time
}
