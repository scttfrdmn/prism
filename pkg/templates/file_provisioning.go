package templates

import (
	"fmt"
	"strings"
)

// GenerateFileProvisioningScript generates a bash script to download and provision files from S3
func GenerateFileProvisioningScript(files []FileConfig, instanceRegion string) string {
	if len(files) == 0 {
		return ""
	}

	var script strings.Builder

	script.WriteString("\n# File Provisioning from S3 (v0.5.7)\n")
	script.WriteString("echo 'Starting file provisioning from S3...'\n\n")

	// Install AWS CLI if not present
	script.WriteString("# Ensure AWS CLI is available\n")
	script.WriteString("if ! command -v aws &> /dev/null; then\n")
	script.WriteString("  echo 'Installing AWS CLI...'\n")
	script.WriteString("  curl 'https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip' -o '/tmp/awscliv2.zip'\n")
	script.WriteString("  unzip -q /tmp/awscliv2.zip -d /tmp\n")
	script.WriteString("  /tmp/aws/install\n")
	script.WriteString("  rm -rf /tmp/awscliv2.zip /tmp/aws\n")
	script.WriteString("fi\n\n")

	// Set region
	script.WriteString(fmt.Sprintf("export AWS_DEFAULT_REGION='%s'\n\n", instanceRegion))

	// Process each file
	for i, file := range files {
		script.WriteString(fmt.Sprintf("# File %d: %s\n", i+1, file.Description))

		// Check conditional
		if file.OnlyIf != "" {
			script.WriteString(fmt.Sprintf("# Conditional: %s\n", file.OnlyIf))
			script.WriteString(generateConditionalCheck(file.OnlyIf))
			script.WriteString("if [ $CONDITION_MET = true ]; then\n")
		}

		// Create destination directory
		script.WriteString(fmt.Sprintf("  mkdir -p $(dirname '%s')\n", file.DestinationPath))

		// Download file with progress
		s3URI := fmt.Sprintf("s3://%s/%s", file.S3Bucket, file.S3Key)
		script.WriteString(fmt.Sprintf("  echo 'Downloading %s...'\n", file.Description))

		// Use AWS CLI to download with checksum verification
		if file.Checksum {
			script.WriteString(fmt.Sprintf("  aws s3 cp '%s' '%s' --only-show-errors", s3URI, file.DestinationPath))
		} else {
			script.WriteString(fmt.Sprintf("  aws s3 cp '%s' '%s' --no-verify-ssl --only-show-errors", s3URI, file.DestinationPath))
		}

		// Handle download result
		script.WriteString("\n  DOWNLOAD_RESULT=$?\n")
		script.WriteString("  if [ $DOWNLOAD_RESULT -eq 0 ]; then\n")
		script.WriteString(fmt.Sprintf("    echo '✓ Downloaded: %s'\n", file.Description))

		// Set ownership if specified
		if file.Owner != "" || file.Group != "" {
			owner := file.Owner
			if owner == "" {
				owner = "ubuntu" // Default owner
			}
			group := file.Group
			if group == "" {
				group = owner // Default group to owner
			}
			script.WriteString(fmt.Sprintf("    chown %s:%s '%s'\n", owner, group, file.DestinationPath))
		}

		// Set permissions if specified
		if file.Permissions != "" {
			script.WriteString(fmt.Sprintf("    chmod %s '%s'\n", file.Permissions, file.DestinationPath))
		}

		// Auto-cleanup from S3 if requested
		if file.AutoCleanup {
			script.WriteString(fmt.Sprintf("    aws s3 rm '%s' --only-show-errors\n", s3URI))
			script.WriteString("    echo '  (Removed from S3)'\n")
		}

		script.WriteString("  else\n")
		script.WriteString(fmt.Sprintf("    echo '✗ Failed to download: %s'\n", file.Description))

		// Handle required vs optional files
		if file.Required {
			script.WriteString("    echo 'ERROR: Required file download failed. Instance provisioning cannot continue.'\n")
			script.WriteString("    exit 1\n")
		} else {
			script.WriteString("    echo 'WARNING: Optional file download failed. Continuing...'\n")
		}

		script.WriteString("  fi\n")

		// Close conditional if present
		if file.OnlyIf != "" {
			script.WriteString("else\n")
			script.WriteString(fmt.Sprintf("  echo 'Skipped: %s (condition not met)'\n", file.Description))
			script.WriteString("fi\n")
		}

		script.WriteString("\n")
	}

	script.WriteString("echo 'File provisioning complete!'\n\n")

	return script.String()
}

// generateConditionalCheck generates bash code to evaluate conditional expressions
func generateConditionalCheck(condition string) string {
	var check strings.Builder

	check.WriteString("  CONDITION_MET=false\n")

	// Parse simple conditions like "arch == 'x86_64'"
	if strings.Contains(condition, "arch") {
		if strings.Contains(condition, "x86_64") {
			check.WriteString("  if [ $(uname -m) = 'x86_64' ]; then\n")
			check.WriteString("    CONDITION_MET=true\n")
			check.WriteString("  fi\n")
		} else if strings.Contains(condition, "arm64") || strings.Contains(condition, "aarch64") {
			check.WriteString("  if [ $(uname -m) = 'aarch64' ] || [ $(uname -m) = 'arm64' ]; then\n")
			check.WriteString("    CONDITION_MET=true\n")
			check.WriteString("  fi\n")
		}
	}

	// Add more condition types as needed
	// For now, support basic architecture checks

	return check.String()
}

// ValidateFileConfig validates a file configuration
func ValidateFileConfig(file FileConfig) error {
	if file.S3Bucket == "" {
		return fmt.Errorf("s3_bucket is required")
	}

	if file.S3Key == "" {
		return fmt.Errorf("s3_key is required")
	}

	if file.DestinationPath == "" {
		return fmt.Errorf("destination_path is required")
	}

	// Validate permissions format if specified
	if file.Permissions != "" {
		if len(file.Permissions) != 4 || file.Permissions[0] != '0' {
			return fmt.Errorf("permissions must be in octal format (e.g., '0644', '0755')")
		}
	}

	return nil
}

// ValidateTemplateFiles validates all file configurations in a template
func ValidateTemplateFiles(template *Template) error {
	for i, file := range template.Files {
		if err := ValidateFileConfig(file); err != nil {
			return fmt.Errorf("file %d (%s): %w", i+1, file.Description, err)
		}
	}
	return nil
}

// EstimateFileProvisioningTime estimates the time required to provision files
func EstimateFileProvisioningTime(files []FileConfig) int {
	if len(files) == 0 {
		return 0
	}

	// Rough estimates:
	// - 1 minute for AWS CLI installation
	// - Files are estimated based on typical sizes and transfer speeds
	// - Small files (<100MB): 1 minute
	// - Medium files (100MB-1GB): 2-5 minutes
	// - Large files (1GB-10GB): 5-15 minutes
	// - Very large files (>10GB): 15+ minutes

	// For now, provide conservative estimates
	baseTime := 1 // AWS CLI installation

	// Add 5 minutes per file as a conservative estimate
	// This will be refined once we have actual transfer data
	estimatedTime := baseTime + (len(files) * 5)

	return estimatedTime
}
