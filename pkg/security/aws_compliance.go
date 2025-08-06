// Package security provides AWS Artifact compliance validation and SCP enforcement
package security

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/artifact"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// AWSComplianceValidator validates CloudWorkstation against AWS Artifact compliance reports and SCPs
type AWSComplianceValidator struct {
	artifactClient      *artifact.Client
	organizationsClient *organizations.Client
	stsClient           *sts.Client
	awsProfile          string
	region              string
}

// ComplianceFramework represents supported compliance frameworks
type ComplianceFramework string

const (
	// Primary research compliance frameworks
	ComplianceNIST800171    ComplianceFramework = "NIST-800-171"
	ComplianceSOC2          ComplianceFramework = "SOC-2"
	ComplianceHIPAA         ComplianceFramework = "HIPAA"
	ComplianceGDPR          ComplianceFramework = "GDPR"
	ComplianceFedRAMP       ComplianceFramework = "FedRAMP"
	ComplianceISO27001      ComplianceFramework = "ISO-27001"
	CompliancePCIDSS        ComplianceFramework = "PCI-DSS"
	
	// Additional frameworks
	ComplianceCSA           ComplianceFramework = "CSA-STAR"
	ComplianceFISMA         ComplianceFramework = "FISMA"
	ComplianceDFARS         ComplianceFramework = "DFARS"
	ComplianceCMMC          ComplianceFramework = "CMMC"
	ComplianceENISA         ComplianceFramework = "ENISA"
	ComplianceC5            ComplianceFramework = "C5"
)

// AWSComplianceStatus represents the compliance status against AWS Artifact reports
type AWSComplianceStatus struct {
	Framework           ComplianceFramework        `json:"framework"`
	AWSCompliant        bool                       `json:"aws_compliant"`
	ArtifactReportID    string                     `json:"artifact_report_id,omitempty"`
	LastUpdated         time.Time                  `json:"last_updated"`
	ComplianceScope     []string                   `json:"compliance_scope"`
	RequiredSCPs        []string                   `json:"required_scps"`
	ImplementedSCPs     []string                   `json:"implemented_scps"`
	GapAnalysis         []ComplianceGap            `json:"gap_analysis"`
	AWSServices         []AWSServiceCompliance     `json:"aws_services"`
	RecommendedActions  []ComplianceRecommendation `json:"recommended_actions"`
}

// ComplianceGap represents gaps between CloudWorkstation and AWS compliance posture
type ComplianceGap struct {
	Control             string `json:"control"`
	AWSImplementation   string `json:"aws_implementation"`
	CloudWorkstationGap string `json:"cloudworkstation_gap"`
	Severity            string `json:"severity"`
	Remediation         string `json:"remediation"`
}

// AWSServiceCompliance represents compliance status of AWS services used by CloudWorkstation
type AWSServiceCompliance struct {
	ServiceName         string                 `json:"service_name"`
	ComplianceStatus    string                 `json:"compliance_status"`
	CertifiedRegions    []string               `json:"certified_regions"`
	RequiredFeatures    []string               `json:"required_features"`
	ConfigurationNeeded map[string]interface{} `json:"configuration_needed"`
}

// ComplianceRecommendation provides specific actions to improve compliance alignment
type ComplianceRecommendation struct {
	Priority        string `json:"priority"`
	Action          string `json:"action"`
	AWSService      string `json:"aws_service,omitempty"`
	SCPRequired     string `json:"scp_required,omitempty"`
	Impact          string `json:"impact"`
	Implementation  string `json:"implementation"`
}

// NewAWSComplianceValidator creates a new AWS compliance validator
func NewAWSComplianceValidator(awsProfile, region string) (*AWSComplianceValidator, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile(awsProfile),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &AWSComplianceValidator{
		artifactClient:      artifact.NewFromConfig(cfg),
		organizationsClient: organizations.NewFromConfig(cfg),
		stsClient:           sts.NewFromConfig(cfg),
		awsProfile:          awsProfile,
		region:              region,
	}, nil
}

// ValidateCompliance validates CloudWorkstation against specified compliance framework
func (v *AWSComplianceValidator) ValidateCompliance(ctx context.Context, framework ComplianceFramework) (*AWSComplianceStatus, error) {
	status := &AWSComplianceStatus{
		Framework:   framework,
		LastUpdated: time.Now(),
	}

	// Get AWS Artifact compliance report
	if err := v.getArtifactReport(ctx, framework, status); err != nil {
		return nil, fmt.Errorf("failed to get artifact report: %w", err)
	}

	// Validate AWS services compliance
	if err := v.validateAWSServices(ctx, framework, status); err != nil {
		return nil, fmt.Errorf("failed to validate AWS services: %w", err)
	}

	// Check required SCPs
	if err := v.validateSCPs(ctx, framework, status); err != nil {
		return nil, fmt.Errorf("failed to validate SCPs: %w", err)
	}

	// Perform gap analysis
	if err := v.performGapAnalysis(framework, status); err != nil {
		return nil, fmt.Errorf("failed to perform gap analysis: %w", err)
	}

	// Generate recommendations
	v.generateRecommendations(framework, status)

	return status, nil
}

// getArtifactReport retrieves relevant AWS Artifact compliance reports
func (v *AWSComplianceValidator) getArtifactReport(ctx context.Context, framework ComplianceFramework, status *AWSComplianceStatus) error {
	// Map framework to AWS Artifact report types
	artifactMapping := map[ComplianceFramework][]string{
		ComplianceSOC2:       {"SOC", "SOC 2 Type II"},
		ComplianceHIPAA:      {"HIPAA", "HIPAA BAA"},
		ComplianceFedRAMP:    {"FedRAMP", "FedRAMP Moderate", "FedRAMP High"},
		ComplianceISO27001:   {"ISO 27001", "ISO 27017", "ISO 27018"},
		CompliancePCIDSS:     {"PCI DSS", "PCI"},
		ComplianceNIST800171: {"NIST", "NIST 800-171"},
		ComplianceCSA:        {"CSA STAR", "Cloud Security Alliance"},
		ComplianceFISMA:      {"FISMA", "FedRAMP"},
		ComplianceDFARS:      {"DFARS", "NIST 800-171"},
		ComplianceCMMC:       {"CMMC", "NIST 800-171"},
	}

	searchTerms, exists := artifactMapping[framework]
	if !exists {
		return fmt.Errorf("unsupported compliance framework: %s", framework)
	}

	// Search for relevant reports in AWS Artifact
	for _, term := range searchTerms {
		reports, err := v.artifactClient.ListReports(ctx, &artifact.ListReportsInput{})
		if err != nil {
			continue // Try next term if this fails
		}

		for _, report := range reports.ReportSummaries {
			if report.Name != nil && strings.Contains(strings.ToLower(*report.Name), strings.ToLower(term)) {
				status.ArtifactReportID = *report.Id
				status.AWSCompliant = true
				if report.UploadState != nil {
					status.LastUpdated = *report.UploadState
				}
				break
			}
		}

		if status.ArtifactReportID != "" {
			break
		}
	}

	return nil
}

// validateAWSServices validates compliance of AWS services used by CloudWorkstation
func (v *AWSComplianceValidator) validateAWSServices(ctx context.Context, framework ComplianceFramework, status *AWSComplianceStatus) error {
	// CloudWorkstation core AWS services
	coreServices := []string{"EC2", "VPC", "IAM", "CloudTrail", "EFS", "EBS", "Systems Manager"}
	
	for _, serviceName := range coreServices {
		serviceCompliance := v.getServiceComplianceStatus(serviceName, framework)
		status.AWSServices = append(status.AWSServices, serviceCompliance)
	}

	return nil
}

// getServiceComplianceStatus returns compliance status for specific AWS service
func (v *AWSComplianceValidator) getServiceComplianceStatus(serviceName string, framework ComplianceFramework) AWSServiceCompliance {
	// AWS service compliance matrix (based on AWS compliance documentation)
	complianceMatrix := map[string]map[ComplianceFramework]AWSServiceCompliance{
		"EC2": {
			ComplianceSOC2: {
				ServiceName:      "Amazon EC2",
				ComplianceStatus: "CERTIFIED",
				CertifiedRegions: []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"},
				RequiredFeatures: []string{"Instance Metadata Service v2", "EBS Encryption"},
			},
			ComplianceHIPAA: {
				ServiceName:      "Amazon EC2",
				ComplianceStatus: "ELIGIBLE",
				CertifiedRegions: []string{"us-east-1", "us-west-2", "eu-west-1"},
				RequiredFeatures: []string{"Dedicated Tenancy", "EBS Encryption", "Enhanced Networking"},
			},
			ComplianceFedRAMP: {
				ServiceName:      "Amazon EC2",
				ComplianceStatus: "AUTHORIZED",
				CertifiedRegions: []string{"us-east-1", "us-west-2", "us-gov-east-1", "us-gov-west-1"},
				RequiredFeatures: []string{"FIPS 140-2 Level 1", "EBS Encryption", "CloudTrail Integration"},
			},
		},
		"VPC": {
			ComplianceSOC2: {
				ServiceName:      "Amazon VPC",
				ComplianceStatus: "CERTIFIED",
				CertifiedRegions: []string{"All Commercial Regions"},
				RequiredFeatures: []string{"VPC Flow Logs", "Network ACLs"},
			},
			ComplianceHIPAA: {
				ServiceName:      "Amazon VPC",
				ComplianceStatus: "ELIGIBLE",
				CertifiedRegions: []string{"All Commercial Regions"},
				RequiredFeatures: []string{"Private Subnets", "NAT Gateway", "VPC Endpoints"},
			},
		},
		"IAM": {
			ComplianceSOC2: {
				ServiceName:      "AWS IAM",
				ComplianceStatus: "CERTIFIED",
				CertifiedRegions: []string{"Global Service"},
				RequiredFeatures: []string{"MFA", "Role-Based Access", "Access Keys Rotation"},
			},
			ComplianceHIPAA: {
				ServiceName:      "AWS IAM",
				ComplianceStatus: "ELIGIBLE",
				CertifiedRegions: []string{"Global Service"},
				RequiredFeatures: []string{"Strong Password Policy", "MFA", "Access Logging"},
			},
		},
		"CloudTrail": {
			ComplianceSOC2: {
				ServiceName:      "AWS CloudTrail",
				ComplianceStatus: "CERTIFIED",
				CertifiedRegions: []string{"All Commercial Regions"},
				RequiredFeatures: []string{"Log File Integrity", "S3 Bucket Encryption"},
			},
			ComplianceHIPAA: {
				ServiceName:      "AWS CloudTrail",
				ComplianceStatus: "ELIGIBLE",
				CertifiedRegions: []string{"All Commercial Regions"},
				RequiredFeatures: []string{"Log File Encryption", "Access Logging", "Retention Policy"},
			},
		},
	}

	if serviceMap, exists := complianceMatrix[serviceName]; exists {
		if compliance, exists := serviceMap[framework]; exists {
			return compliance
		}
	}

	// Default compliance status for unlisted services
	return AWSServiceCompliance{
		ServiceName:      serviceName,
		ComplianceStatus: "REVIEW_REQUIRED",
		CertifiedRegions: []string{v.region},
		RequiredFeatures: []string{"Standard AWS Security Features"},
	}
}

// validateSCPs validates Service Control Policies for compliance requirements
func (v *AWSComplianceValidator) validateSCPs(ctx context.Context, framework ComplianceFramework, status *AWSComplianceStatus) error {
	// Define required SCPs for each compliance framework
	requiredSCPs := map[ComplianceFramework][]string{
		ComplianceSOC2: {
			"DenyRootUserAccess",
			"RequireMFAForConsoleAccess",
			"EnforceSSLOnlyRequests",
			"RestrictRegionAccess",
		},
		ComplianceHIPAA: {
			"DenyUnencryptedStorage",
			"RequireMFAForConsoleAccess",
			"RestrictRegionAccess",
			"DenyPublicS3Buckets",
			"EnforceVPCEndpoints",
		},
		ComplianceFedRAMP: {
			"RequireMFAForAllAccess",
			"DenyNonGovCloudRegions",
			"EnforceFIPS140-2",
			"RequireCloudTrailEncryption",
			"DenyPublicAMISharing",
		},
		ComplianceNIST800171: {
			"RequireMFAForConsoleAccess",
			"EnforceEncryptionAtRest",
			"RestrictRegionAccess",
			"RequireDetailedLogging",
			"DenyPublicResources",
		},
	}

	scpList, exists := requiredSCPs[framework]
	if !exists {
		return nil // No specific SCPs required for this framework
	}

	status.RequiredSCPs = scpList

	// Check if organization has SCPs enabled
	caller, err := v.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to get caller identity: %w", err)
	}

	// Try to list organization policies (this requires appropriate permissions)
	policies, err := v.organizationsClient.ListPolicies(ctx, &organizations.ListPoliciesInput{
		Filter: "SERVICE_CONTROL_POLICY",
	})
	
	if err != nil {
		// Organization features may not be available
		status.ImplementedSCPs = []string{"ORGANIZATION_CHECK_REQUIRED"}
		return nil
	}

	// Check which required SCPs are implemented
	for _, policy := range policies.Policies {
		if policy.Name != nil {
			for _, requiredSCP := range scpList {
				if strings.Contains(*policy.Name, requiredSCP) {
					status.ImplementedSCPs = append(status.ImplementedSCPs, *policy.Name)
				}
			}
		}
	}

	// Identify missing SCPs
	for _, requiredSCP := range scpList {
		found := false
		for _, implemented := range status.ImplementedSCPs {
			if strings.Contains(implemented, requiredSCP) {
				found = true
				break
			}
		}
		if !found {
			status.GapAnalysis = append(status.GapAnalysis, ComplianceGap{
				Control:             fmt.Sprintf("SCP-%s", requiredSCP),
				AWSImplementation:   "Service Control Policy available",
				CloudWorkstationGap: "Required SCP not implemented",
				Severity:            "HIGH",
				Remediation:         fmt.Sprintf("Implement %s Service Control Policy", requiredSCP),
			})
		}
	}

	return nil
}

// performGapAnalysis analyzes gaps between CloudWorkstation and AWS compliance posture
func (v *AWSComplianceValidator) performGapAnalysis(framework ComplianceFramework, status *AWSComplianceStatus) error {
	// Framework-specific gap analysis
	switch framework {
	case ComplianceSOC2:
		v.analyzeSOC2Gaps(status)
	case ComplianceHIPAA:
		v.analyzeHIPAAGaps(status)
	case ComplianceFedRAMP:
		v.analyzeFedRAMPGaps(status)
	case ComplianceNIST800171:
		v.analyzeNIST800171Gaps(status)
	case ComplianceISO27001:
		v.analyzeISO27001Gaps(status)
	case CompliancePCIDSS:
		v.analyzePCIDSSGaps(status)
	}

	return nil
}

// analyzeSOC2Gaps performs SOC 2 specific gap analysis
func (v *AWSComplianceValidator) analyzeSOC2Gaps(status *AWSComplianceStatus) {
	gaps := []ComplianceGap{
		{
			Control:             "CC6.1 - Logical Access Controls",
			AWSImplementation:   "IAM with MFA and role-based access",
			CloudWorkstationGap: "Device binding authentication may need AWS integration",
			Severity:            "MEDIUM",
			Remediation:         "Integrate device binding with AWS IAM roles",
		},
		{
			Control:             "CC6.7 - Data Transmission Controls",
			AWSImplementation:   "TLS 1.2+ for all AWS services",
			CloudWorkstationGap: "Registry communication uses custom encryption",
			Severity:            "LOW",
			Remediation:         "Document custom encryption alignment with AWS standards",
		},
	}

	status.GapAnalysis = append(status.GapAnalysis, gaps...)
}

// analyzeHIPAAGaps performs HIPAA specific gap analysis
func (v *AWSComplianceValidator) analyzeHIPAAGaps(status *AWSComplianceStatus) {
	gaps := []ComplianceGap{
		{
			Control:             "§164.312(a)(1) - Access Control",
			AWSImplementation:   "IAM with unique user identification",
			CloudWorkstationGap: "Need to ensure PHI access controls align with AWS BAA",
			Severity:            "HIGH",
			Remediation:         "Implement HIPAA-compliant access controls with AWS services",
		},
		{
			Control:             "§164.312(e)(1) - Transmission Security",
			AWSImplementation:   "End-to-end encryption for all data transmission",
			CloudWorkstationGap: "Custom invitation system encryption needs BAA coverage",
			Severity:            "CRITICAL",
			Remediation:         "Ensure all CloudWorkstation encryption aligns with AWS BAA",
		},
	}

	status.GapAnalysis = append(status.GapAnalysis, gaps...)
}

// analyzeFedRAMPGaps performs FedRAMP specific gap analysis
func (v *AWSComplianceValidator) analyzeFedRAMPGaps(status *AWSComplianceStatus) {
	gaps := []ComplianceGap{
		{
			Control:             "AC-2 - Account Management",
			AWSImplementation:   "Automated account lifecycle management",
			CloudWorkstationGap: "Manual account provisioning in templates",
			Severity:            "HIGH",
			Remediation:         "Integrate with AWS SSO or automated account management",
		},
		{
			Control:             "AU-2 - Event Logging",
			AWSImplementation:   "CloudTrail comprehensive event logging",
			CloudWorkstationGap: "Local audit logging may need CloudTrail integration",
			Severity:            "MEDIUM",
			Remediation:         "Forward audit logs to CloudTrail for centralized logging",
		},
	}

	status.GapAnalysis = append(status.GapAnalysis, gaps...)
}

// analyzeNIST800171Gaps performs NIST 800-171 specific gap analysis
func (v *AWSComplianceValidator) analyzeNIST800171Gaps(status *AWSComplianceStatus) {
	gaps := []ComplianceGap{
		{
			Control:             "3.1.1 - Authorized Access Control",
			AWSImplementation:   "IAM policies and roles with fine-grained permissions",
			CloudWorkstationGap: "Template-based access may need AWS IAM integration",
			Severity:            "HIGH",
			Remediation:         "Map template users to AWS IAM roles for CUI access",
		},
		{
			Control:             "3.3.1 - Audit Record Creation",
			AWSImplementation:   "CloudTrail and AWS Config for comprehensive auditing",
			CloudWorkstationGap: "Local audit logs need integration with AWS audit services",
			Severity:            "HIGH",
			Remediation:         "Forward security audit logs to CloudWatch and CloudTrail",
		},
	}

	status.GapAnalysis = append(status.GapAnalysis, gaps...)
}

// analyzeISO27001Gaps performs ISO 27001 specific gap analysis
func (v *AWSComplianceValidator) analyzeISO27001Gaps(status *AWSComplianceStatus) {
	gaps := []ComplianceGap{
		{
			Control:             "A.9.1.2 - Access to Networks and Network Services",
			AWSImplementation:   "VPC with network segmentation and access controls",
			CloudWorkstationGap: "Instance networking may need VPC endpoint integration",
			Severity:            "MEDIUM",
			Remediation:         "Configure VPC endpoints for AWS service access",
		},
	}

	status.GapAnalysis = append(status.GapAnalysis, gaps...)
}

// analyzePCIDSSGaps performs PCI DSS specific gap analysis  
func (v *AWSComplianceValidator) analyzePCIDSSGaps(status *AWSComplianceStatus) {
	gaps := []ComplianceGap{
		{
			Control:             "Requirement 3 - Protect Stored Cardholder Data",
			AWSImplementation:   "EBS and S3 encryption with AWS KMS",
			CloudWorkstationGap: "Custom encryption keys may need AWS KMS integration",
			Severity:            "CRITICAL",
			Remediation:         "Integrate invitation system encryption with AWS KMS",
		},
	}

	status.GapAnalysis = append(status.GapAnalysis, gaps...)
}

// generateRecommendations creates specific recommendations for improving compliance
func (v *AWSComplianceValidator) generateRecommendations(framework ComplianceFramework, status *AWSComplianceStatus) {
	baseRecommendations := []ComplianceRecommendation{
		{
			Priority:       "HIGH",
			Action:         "Enable AWS Config for comprehensive resource monitoring",
			AWSService:     "AWS Config",
			Impact:         "Provides continuous compliance monitoring and assessment",
			Implementation: "cws aws config enable --compliance-rules " + string(framework),
		},
		{
			Priority:       "HIGH", 
			Action:         "Integrate CloudWorkstation audit logs with CloudWatch",
			AWSService:     "CloudWatch Logs",
			Impact:         "Centralized logging and compliance reporting",
			Implementation: "Configure log forwarding to CloudWatch Logs",
		},
		{
			Priority:       "MEDIUM",
			Action:         "Implement VPC endpoints for AWS service access",
			AWSService:     "VPC Endpoints",
			Impact:         "Enhanced network security and compliance",
			Implementation: "Configure VPC endpoints for EC2, S3, and other services",
		},
	}

	// Add framework-specific recommendations
	switch framework {
	case ComplianceHIPAA:
		baseRecommendations = append(baseRecommendations, ComplianceRecommendation{
			Priority:       "CRITICAL",
			Action:         "Sign AWS Business Associate Agreement (BAA)",
			Impact:         "Required for HIPAA compliance with AWS services",
			Implementation: "Contact AWS to execute BAA for your account",
		})
	case ComplianceFedRAMP:
		baseRecommendations = append(baseRecommendations, ComplianceRecommendation{
			Priority:       "CRITICAL",
			Action:         "Migrate to AWS GovCloud regions for FedRAMP High",
			AWSService:     "AWS GovCloud",
			Impact:         "Required for FedRAMP High compliance",
			Implementation: "Plan migration to us-gov-east-1 or us-gov-west-1",
		})
	}

	status.RecommendedActions = baseRecommendations
}

// GetSupportedFrameworks returns list of supported compliance frameworks
func (v *AWSComplianceValidator) GetSupportedFrameworks() []ComplianceFramework {
	return []ComplianceFramework{
		ComplianceNIST800171,
		ComplianceSOC2,
		ComplianceHIPAA,
		ComplianceGDPR,
		ComplianceFedRAMP,
		ComplianceISO27001,
		CompliancePCIDSS,
		ComplianceCSA,
		ComplianceFISMA,
		ComplianceDFARS,
		ComplianceCMMC,
	}
}