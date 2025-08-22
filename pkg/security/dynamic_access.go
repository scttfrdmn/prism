package security

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// AccessStrategy defines how web interfaces should be accessed
type AccessStrategy int

const (
	AccessDirect   AccessStrategy = iota // Direct access from user IP
	AccessSubnet                         // Access from user's subnet (/24)
	AccessTunneled                       // SSH tunneling required
)

// AccessConfig contains the determined access configuration
type AccessConfig struct {
	Strategy   AccessStrategy
	UserIP     string
	SubnetCIDR string
	BindIP     string
	Message    string
}

// DetermineAccessStrategy determines the best access strategy for web interfaces
func DetermineAccessStrategy() *AccessConfig {
	config := &AccessConfig{
		Strategy: AccessTunneled,
		BindIP:   "127.0.0.1",
		Message:  "IP detection failed - SSH tunneling required for security",
	}

	userIP, err := DetectUserExternalIP()
	if err != nil {
		log.Printf("Warning: IP detection failed (%v), using SSH tunneling", err)
		return config
	}

	config.UserIP = userIP

	// Check if IP looks like a static/business connection
	if isLikelyStaticIP(userIP) {
		config.Strategy = AccessDirect
		config.BindIP = "0.0.0.0"
		config.Message = fmt.Sprintf("Direct access from static IP %s", userIP)
		return config
	}

	// For dynamic IPs, use subnet-based access
	subnet, err := getSubnetCIDR(userIP)
	if err != nil {
		log.Printf("Warning: Subnet detection failed (%v), falling back to SSH tunneling", err)
		return config
	}

	config.Strategy = AccessSubnet
	config.SubnetCIDR = subnet
	config.BindIP = "0.0.0.0"
	config.Message = fmt.Sprintf("Subnet access from %s (handles DHCP changes)", subnet)

	return config
}

// getSubnetCIDR returns the /24 subnet for the given IP
func getSubnetCIDR(ipStr string) (string, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// Convert to /24 subnet
	ip4 := ip.To4()
	if ip4 == nil {
		return "", fmt.Errorf("not an IPv4 address: %s", ipStr)
	}

	// Create /24 subnet (e.g., 192.168.1.100 -> 192.168.1.0/24)
	subnet := fmt.Sprintf("%d.%d.%d.0/24", ip4[0], ip4[1], ip4[2])
	return subnet, nil
}

// isLikelyStaticIP heuristically determines if an IP is likely static
func isLikelyStaticIP(ipStr string) bool {
	// Business/cloud provider patterns that suggest static IPs
	staticProviders := []string{
		// AWS EC2 instance patterns
		"ec2-",
		// Google Cloud patterns
		"googleusercontent.com",
		// Azure patterns
		"cloudapp.net",
		// Business ISP patterns (common static ranges)
		// This is heuristic - could be improved with GeoIP data
	}

	for _, provider := range staticProviders {
		if strings.Contains(ipStr, provider) {
			return true
		}
	}

	// Could add more sophisticated detection:
	// - Check for reverse DNS patterns
	// - Use GeoIP databases to identify business vs residential
	// - Historical IP stability tracking

	return false
}

// SecurityGroupUpdater handles dynamic security group updates
type SecurityGroupUpdater struct {
	ec2Client   EC2ClientInterface
	groupID     string
	lastIP      string
	lastSubnet  string
	updateCount int
}

// NewSecurityGroupUpdater creates a new security group updater
func NewSecurityGroupUpdater(ec2Client EC2ClientInterface, groupID string) *SecurityGroupUpdater {
	return &SecurityGroupUpdater{
		ec2Client: ec2Client,
		groupID:   groupID,
	}
}

// UpdateAccessRules updates security group rules when IP changes
func (sgu *SecurityGroupUpdater) UpdateAccessRules(ctx context.Context) error {
	config := DetermineAccessStrategy()

	switch config.Strategy {
	case AccessDirect:
		return sgu.updateDirectAccess(ctx, config.UserIP)
	case AccessSubnet:
		return sgu.updateSubnetAccess(ctx, config.SubnetCIDR)
	default:
		return sgu.removeWebAccess(ctx)
	}
}

// updateDirectAccess updates rules for direct IP access
func (sgu *SecurityGroupUpdater) updateDirectAccess(ctx context.Context, userIP string) error {
	if sgu.lastIP == userIP {
		return nil // No change needed
	}

	// Remove old IP rules if they exist
	if sgu.lastIP != "" {
		if err := sgu.removeIPRules(ctx, sgu.lastIP+"/32"); err != nil {
			log.Printf("Warning: Failed to remove old IP rules: %v", err)
		}
	}

	// Add new IP rules
	if err := sgu.addWebAccessRules(ctx, userIP+"/32"); err != nil {
		return fmt.Errorf("failed to add direct access rules: %w", err)
	}

	sgu.lastIP = userIP
	sgu.updateCount++

	log.Printf("✅ Updated direct access for IP %s (update #%d)", userIP, sgu.updateCount)
	return nil
}

// updateSubnetAccess updates rules for subnet-based access
func (sgu *SecurityGroupUpdater) updateSubnetAccess(ctx context.Context, subnetCIDR string) error {
	if sgu.lastSubnet == subnetCIDR {
		return nil // No change needed
	}

	// Remove old subnet rules if they exist
	if sgu.lastSubnet != "" {
		if err := sgu.removeIPRules(ctx, sgu.lastSubnet); err != nil {
			log.Printf("Warning: Failed to remove old subnet rules: %v", err)
		}
	}

	// Add new subnet rules
	if err := sgu.addWebAccessRules(ctx, subnetCIDR); err != nil {
		return fmt.Errorf("failed to add subnet access rules: %w", err)
	}

	sgu.lastSubnet = subnetCIDR
	sgu.updateCount++

	log.Printf("✅ Updated subnet access for %s (update #%d)", subnetCIDR, sgu.updateCount)
	return nil
}

// addWebAccessRules adds web interface access rules to security group
func (sgu *SecurityGroupUpdater) addWebAccessRules(ctx context.Context, cidr string) error {
	webPorts := []int32{80, 443, 8888, 8787}

	for _, port := range webPorts {
		_, err := sgu.ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: aws.String(sgu.groupID),
			IpPermissions: []ec2types.IpPermission{
				{
					IpProtocol: aws.String("tcp"),
					FromPort:   aws.Int32(port),
					ToPort:     aws.Int32(port),
					IpRanges: []ec2types.IpRange{
						{
							CidrIp:      aws.String(cidr),
							Description: aws.String(fmt.Sprintf("Dynamic web access port %d", port)),
						},
					},
				},
			},
		})
		if err != nil {
			// Check if rule already exists
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("failed to add rule for port %d: %w", port, err)
			}
		}
	}

	return nil
}

// removeIPRules removes access rules for the specified CIDR
func (sgu *SecurityGroupUpdater) removeIPRules(ctx context.Context, cidr string) error {
	webPorts := []int32{80, 443, 8888, 8787}

	for _, port := range webPorts {
		_, err := sgu.ec2Client.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
			GroupId: aws.String(sgu.groupID),
			IpPermissions: []ec2types.IpPermission{
				{
					IpProtocol: aws.String("tcp"),
					FromPort:   aws.Int32(port),
					ToPort:     aws.Int32(port),
					IpRanges: []ec2types.IpRange{
						{
							CidrIp: aws.String(cidr),
						},
					},
				},
			},
		})
		if err != nil && !strings.Contains(err.Error(), "does not exist") {
			log.Printf("Warning: Failed to remove rule for port %d: %v", port, err)
		}
	}

	return nil
}

// removeWebAccess removes all web access rules (fallback to SSH tunneling)
func (sgu *SecurityGroupUpdater) removeWebAccess(ctx context.Context) error {
	if sgu.lastIP != "" {
		_ = sgu.removeIPRules(ctx, sgu.lastIP+"/32")
		sgu.lastIP = ""
	}

	if sgu.lastSubnet != "" {
		_ = sgu.removeIPRules(ctx, sgu.lastSubnet)
		sgu.lastSubnet = ""
	}

	return nil
}

// EC2ClientInterface defines the EC2 operations needed for security group updates
type EC2ClientInterface interface {
	AuthorizeSecurityGroupIngress(ctx context.Context, params *ec2.AuthorizeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.AuthorizeSecurityGroupIngressOutput, error)
	RevokeSecurityGroupIngress(ctx context.Context, params *ec2.RevokeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupIngressOutput, error)
}
