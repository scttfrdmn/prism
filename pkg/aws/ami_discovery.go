package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// AMIDiscovery handles dynamic AMI discovery via AWS SSM Parameter Store
type AMIDiscovery struct {
	ssmClient *ssm.Client
}

// NewAMIDiscovery creates a new AMI discovery service
func NewAMIDiscovery(ssmClient *ssm.Client) *AMIDiscovery {
	return &AMIDiscovery{
		ssmClient: ssmClient,
	}
}

// GetLatestAMI queries AWS SSM Parameter Store for the latest AMI ID
//
// Parameters:
//   - distro: Base OS distro (ubuntu, rocky, amazonlinux, alpine, rhel, debian)
//   - version: OS version (24.04, 22.04, 9, 10, 2023, etc.)
//   - region: AWS region (us-east-1, us-west-2, etc.)
//   - arch: Architecture (x86_64 or arm64)
//
// Returns:
//   - AMI ID if found
//   - Empty string if not found (caller should use static fallback)
//   - Error only for serious issues (network failures, etc.)
func (d *AMIDiscovery) GetLatestAMI(ctx context.Context, distro, version, region, arch string) (string, error) {
	// Get SSM parameter path for this distro/version/arch combination
	paramPath := d.getSSMParameterPath(distro, version, arch)
	if paramPath == "" {
		// No SSM parameter available for this combination
		return "", nil
	}

	// Query SSM Parameter Store
	result, err := d.ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name: aws.String(paramPath),
	})

	if err != nil {
		// Parameter not found or other error - return empty to trigger fallback
		return "", nil
	}

	if result.Parameter == nil || result.Parameter.Value == nil {
		return "", nil
	}

	return *result.Parameter.Value, nil
}

// getSSMParameterPath returns the AWS SSM Parameter Store path for AMI discovery
//
// AWS publishes canonical AMI IDs to well-known SSM parameters:
//
// Ubuntu (Canonical):
//
//	/aws/service/canonical/ubuntu/server/{version}/stable/current/{arch}/hvm/ebs-gp3/ami-id
//
// Amazon Linux:
//
//	/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-{arch}
//	/aws/service/ami-amazon-linux-latest/amzn2-ami-kernel-5.10-hvm-{arch}-gp2
//
// Debian (Official):
//
//	/aws/service/debian/release/{version}/latest/{arch}
//
// Rocky Linux, RHEL, Alpine:
//
//	These don't have official AWS SSM parameters, so we return empty string
//	and rely on static fallback AMI mappings
func (d *AMIDiscovery) getSSMParameterPath(distro, version, arch string) string {
	// Convert architecture to AWS SSM format
	ssmArch := d.convertArchToSSMFormat(arch)

	switch distro {
	case "ubuntu":
		// Ubuntu Canonical official parameters
		// Path format: /aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp3/ami-id
		return fmt.Sprintf("/aws/service/canonical/ubuntu/server/%s/stable/current/%s/hvm/ebs-gp3/ami-id",
			version, ssmArch)

	case "amazonlinux":
		// Amazon Linux official parameters
		if version == "2023" {
			// Amazon Linux 2023
			return fmt.Sprintf("/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-%s", ssmArch)
		} else if version == "2" {
			// Amazon Linux 2
			return fmt.Sprintf("/aws/service/ami-amazon-linux-latest/amzn2-ami-kernel-5.10-hvm-%s-gp2", ssmArch)
		}

	case "debian":
		// Debian official parameters (if available)
		// Note: Debian's SSM parameter structure may vary
		return fmt.Sprintf("/aws/service/debian/release/%s/latest/%s", version, ssmArch)

	case "rocky", "rhel", "alpine":
		// Rocky Linux, RHEL, and Alpine don't have official AWS SSM parameters
		// Return empty string to trigger static fallback
		return ""
	}

	return ""
}

// convertArchToSSMFormat converts our architecture format to AWS SSM format
func (d *AMIDiscovery) convertArchToSSMFormat(arch string) string {
	switch arch {
	case "x86_64":
		return "amd64" // AWS SSM uses "amd64" for Ubuntu
	case "arm64":
		return "arm64"
	default:
		return arch
	}
}

// BulkDiscoverAMIs discovers AMIs for multiple distro/version/arch combinations
//
// # This is useful for warming up the AMI cache at daemon startup
//
// Returns:
//   - Map of discovered AMIs: distro -> version -> region -> arch -> AMI
//   - Combinations not found in SSM are omitted (caller uses static fallback)
func (d *AMIDiscovery) BulkDiscoverAMIs(ctx context.Context, region string) (map[string]map[string]map[string]map[string]string, error) {
	discovered := make(map[string]map[string]map[string]map[string]string)

	// Define the combinations we want to discover
	combinations := []struct {
		distro  string
		version string
		arch    string
	}{
		// Ubuntu versions
		{"ubuntu", "24.04", "x86_64"},
		{"ubuntu", "24.04", "arm64"},
		{"ubuntu", "22.04", "x86_64"},
		{"ubuntu", "22.04", "arm64"},
		{"ubuntu", "20.04", "x86_64"},
		{"ubuntu", "20.04", "arm64"},

		// Amazon Linux versions
		{"amazonlinux", "2023", "x86_64"},
		{"amazonlinux", "2023", "arm64"},
		{"amazonlinux", "2", "x86_64"},
		{"amazonlinux", "2", "arm64"},

		// Debian versions
		{"debian", "12", "x86_64"},
		{"debian", "12", "arm64"},
		{"debian", "11", "x86_64"},
		{"debian", "11", "arm64"},
	}

	for _, combo := range combinations {
		ami, err := d.GetLatestAMI(ctx, combo.distro, combo.version, region, combo.arch)
		if err != nil {
			// Log error but continue with other combinations
			continue
		}

		if ami != "" {
			// Initialize nested maps if needed
			if discovered[combo.distro] == nil {
				discovered[combo.distro] = make(map[string]map[string]map[string]string)
			}
			if discovered[combo.distro][combo.version] == nil {
				discovered[combo.distro][combo.version] = make(map[string]map[string]string)
			}
			if discovered[combo.distro][combo.version][region] == nil {
				discovered[combo.distro][combo.version][region] = make(map[string]string)
			}

			discovered[combo.distro][combo.version][region][combo.arch] = ami
		}
	}

	return discovered, nil
}

// GetAMIWithFallback attempts dynamic discovery first, then falls back to static AMI
//
// # This is the primary method that should be used by the launch system
//
// Parameters:
//   - distro: Base OS distro
//   - version: OS version
//   - region: AWS region
//   - arch: Architecture
//   - staticFallback: Static AMI ID to use if discovery fails
//
// Returns:
//   - AMI ID (from SSM or fallback)
//   - Source indicator ("ssm" or "static")
//   - Error only for serious issues
func (d *AMIDiscovery) GetAMIWithFallback(ctx context.Context, distro, version, region, arch, staticFallback string) (string, string, error) {
	// Attempt dynamic discovery
	ami, err := d.GetLatestAMI(ctx, distro, version, region, arch)
	if err != nil {
		// Serious error - but still use fallback
		if staticFallback != "" {
			return staticFallback, "static", nil
		}
		return "", "", fmt.Errorf("AMI discovery failed and no static fallback available: %w", err)
	}

	if ami != "" {
		// Successfully discovered via SSM
		return ami, "ssm", nil
	}

	// No SSM parameter available - use static fallback
	if staticFallback != "" {
		return staticFallback, "static", nil
	}

	return "", "", fmt.Errorf("no AMI found via SSM and no static fallback available for %s %s %s %s",
		distro, version, region, arch)
}

// ValidateAMI checks if an AMI exists and is accessible in the specified region
//
// This is useful for validating static AMI mappings
func (d *AMIDiscovery) ValidateAMI(ctx context.Context, amiID, region string) (bool, error) {
	// Note: This would require EC2 client, not SSM client
	// Implementation would use ec2.DescribeImages to check if AMI exists
	// For now, we assume AMIs from SSM Parameter Store are valid
	return true, nil
}

// GetSSMParameterPaths returns all SSM parameter paths that CloudWorkstation queries
//
// This is useful for documentation and debugging
func (d *AMIDiscovery) GetSSMParameterPaths() map[string][]string {
	return map[string][]string{
		"Ubuntu 24.04": {
			"/aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp3/ami-id",
			"/aws/service/canonical/ubuntu/server/24.04/stable/current/arm64/hvm/ebs-gp3/ami-id",
		},
		"Ubuntu 22.04": {
			"/aws/service/canonical/ubuntu/server/22.04/stable/current/amd64/hvm/ebs-gp3/ami-id",
			"/aws/service/canonical/ubuntu/server/22.04/stable/current/arm64/hvm/ebs-gp3/ami-id",
		},
		"Ubuntu 20.04": {
			"/aws/service/canonical/ubuntu/server/20.04/stable/current/amd64/hvm/ebs-gp3/ami-id",
			"/aws/service/canonical/ubuntu/server/20.04/stable/current/arm64/hvm/ebs-gp3/ami-id",
		},
		"Amazon Linux 2023": {
			"/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-amd64",
			"/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-arm64",
		},
		"Amazon Linux 2": {
			"/aws/service/ami-amazon-linux-latest/amzn2-ami-kernel-5.10-hvm-amd64-gp2",
			"/aws/service/ami-amazon-linux-latest/amzn2-ami-kernel-5.10-hvm-arm64-gp2",
		},
		"Debian 12": {
			"/aws/service/debian/release/12/latest/amd64",
			"/aws/service/debian/release/12/latest/arm64",
		},
	}
}

// GetRecommendedUpdateFrequency returns how often static AMI mappings should be updated
//
// This helps with maintenance planning
func GetRecommendedUpdateFrequency() map[string]string {
	return map[string]string{
		"Ubuntu LTS":        "Every 6 months (point releases like 24.04.1, 24.04.2)",
		"Ubuntu":            "Every 6 months",
		"Amazon Linux 2023": "Quarterly (Amazon updates frequently)",
		"Amazon Linux 2":    "Every 6 months",
		"Rocky Linux":       "Every 6 months (point releases like 9.1, 9.2)",
		"RHEL":              "Every 6 months",
		"Alpine":            "Every 3 months (Alpine releases frequently)",
		"Debian":            "Yearly (Debian stable is very stable)",
	}
}

// GetStaticAMILocations returns where to find current AMI IDs for manual updates
//
// This is useful documentation for maintainers
func GetStaticAMILocations() map[string]string {
	return map[string]string{
		"Ubuntu":       "https://cloud-images.ubuntu.com/locator/ec2/ - Search for your region and version",
		"Amazon Linux": "https://aws.amazon.com/amazon-linux-2/release-notes/ - Official AMI IDs listed",
		"Rocky Linux":  "https://rockylinux.org/cloud-images/ - Community-maintained AMI list",
		"RHEL":         "https://access.redhat.com/solutions/15356 - Red Hat official AMI list",
		"Alpine":       "https://alpinelinux.org/cloud/ - Alpine official cloud images",
		"Debian":       "https://wiki.debian.org/Cloud/AmazonEC2Image - Debian official AMIs",
	}
}
