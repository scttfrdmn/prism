package security

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewAWSComplianceValidator tests compliance validator creation
func TestNewAWSComplianceValidator(t *testing.T) {
	// Test with valid AWS profile and region
	validator, err := NewAWSComplianceValidator("default", "us-east-1")
	
	if err != nil {
		// May fail if AWS credentials not configured, which is expected in test environment
		assert.Contains(t, err.Error(), "failed to load AWS config")
	} else {
		assert.NotNil(t, validator)
		assert.Equal(t, "default", validator.awsProfile)
		assert.Equal(t, "us-east-1", validator.region)
	}
}

// TestNewAWSComplianceValidatorInvalidProfile tests validator creation with invalid profile
func TestNewAWSComplianceValidatorInvalidProfile(t *testing.T) {
	validator, err := NewAWSComplianceValidator("non-existent-profile", "us-east-1")
	
	// Should fail with AWS config error
	assert.Error(t, err)
	assert.Nil(t, validator)
	assert.Contains(t, err.Error(), "failed to load AWS config")
}

// TestComplianceFrameworkConstants tests compliance framework definitions
func TestComplianceFrameworkConstants(t *testing.T) {
	frameworks := []ComplianceFramework{
		ComplianceNIST800171,
		ComplianceNIST80053,
		ComplianceSOC2,
		ComplianceHIPAA,
		ComplianceGDPR,
		ComplianceFedRAMP,
		ComplianceISO27001,
		CompliancePCIDSS,
		ComplianceITAR,
		ComplianceEAR,
		ComplianceCSA,
		ComplianceFISMA,
		ComplianceDFARS,
		ComplianceCMMC,
		ComplianceCMMCL1,
		ComplianceCMMCL2,
		ComplianceCMMCL3,
		ComplianceENISA,
		ComplianceC5,
		ComplianceFERPA,
	}
	
	// Verify framework constants are properly defined
	for _, framework := range frameworks {
		assert.NotEmpty(t, string(framework))
	}
	
	// Test specific framework values
	assert.Equal(t, "NIST-800-171", string(ComplianceNIST800171))
	assert.Equal(t, "SOC-2", string(ComplianceSOC2))
	assert.Equal(t, "HIPAA", string(ComplianceHIPAA))
	assert.Equal(t, "FedRAMP", string(ComplianceFedRAMP))
	assert.Equal(t, "CMMC-L2", string(ComplianceCMMCL2))
}

// TestAWSComplianceStatus struct validation
func TestAWSComplianceStatus(t *testing.T) {
	now := time.Now()
	
	status := AWSComplianceStatus{
		Framework:        ComplianceSOC2,
		AWSCompliant:     true,
		ArtifactReportID: "report-123",
		LastUpdated:      now,
		ComplianceScope:  []string{"compute", "storage", "networking"},
		RequiredSCPs:     []string{"DenyRootUserAccess", "RequireMFAForConsoleAccess"},
		ImplementedSCPs:  []string{"DenyRootUserAccess"},
		GapAnalysis:      []ComplianceGap{},
		AWSServices:      []AWSServiceCompliance{},
		RecommendedActions: []ComplianceRecommendation{},
	}
	
	assert.Equal(t, ComplianceSOC2, status.Framework)
	assert.True(t, status.AWSCompliant)
	assert.Equal(t, "report-123", status.ArtifactReportID)
	assert.Equal(t, now, status.LastUpdated)
	assert.Len(t, status.ComplianceScope, 3)
	assert.Len(t, status.RequiredSCPs, 2)
	assert.Len(t, status.ImplementedSCPs, 1)
}

// TestComplianceGap struct validation
func TestComplianceGap(t *testing.T) {
	gap := ComplianceGap{
		Control:             "AC-2 Account Management",
		AWSImplementation:   "IAM provides account lifecycle management",
		CloudWorkstationGap: "Manual account provisioning in templates",
		Severity:            "HIGH",
		Remediation:         "Integrate with AWS SSO",
	}
	
	assert.Equal(t, "AC-2 Account Management", gap.Control)
	assert.Equal(t, "HIGH", gap.Severity)
	assert.NotEmpty(t, gap.Remediation)
}

// TestAWSServiceCompliance struct validation
func TestAWSServiceCompliance(t *testing.T) {
	serviceCompliance := AWSServiceCompliance{
		ServiceName:      "Amazon EC2",
		ComplianceStatus: "CERTIFIED",
		CertifiedRegions: []string{"us-east-1", "us-west-2"},
		RequiredFeatures: []string{"Instance Metadata Service v2", "EBS Encryption"},
		ConfigurationNeeded: map[string]interface{}{
			"encryption": "enabled",
			"metadata":   "v2",
		},
	}
	
	assert.Equal(t, "Amazon EC2", serviceCompliance.ServiceName)
	assert.Equal(t, "CERTIFIED", serviceCompliance.ComplianceStatus)
	assert.Len(t, serviceCompliance.CertifiedRegions, 2)
	assert.Len(t, serviceCompliance.RequiredFeatures, 2)
	assert.Contains(t, serviceCompliance.ConfigurationNeeded, "encryption")
}

// TestComplianceRecommendation struct validation
func TestComplianceRecommendation(t *testing.T) {
	recommendation := ComplianceRecommendation{
		Priority:       "HIGH",
		Action:         "Enable AWS Config for comprehensive resource monitoring",
		AWSService:     "AWS Config",
		SCPRequired:    "RequireConfigRecording",
		Impact:         "Provides continuous compliance monitoring",
		Implementation: "cws aws config enable --compliance-rules SOC2",
	}
	
	assert.Equal(t, "HIGH", recommendation.Priority)
	assert.Equal(t, "AWS Config", recommendation.AWSService)
	assert.NotEmpty(t, recommendation.Action)
	assert.NotEmpty(t, recommendation.Implementation)
}

// TestGetServiceComplianceStatus tests service compliance status retrieval
func TestGetServiceComplianceStatus(t *testing.T) {
	validator := &AWSComplianceValidator{
		region: "us-east-1",
	}
	
	// Test known service with SOC2
	ec2Compliance := validator.getServiceComplianceStatus("EC2", ComplianceSOC2)
	
	assert.Equal(t, "Amazon EC2", ec2Compliance.ServiceName)
	assert.Equal(t, "CERTIFIED", ec2Compliance.ComplianceStatus)
	assert.Contains(t, ec2Compliance.CertifiedRegions, "us-east-1")
	assert.Contains(t, ec2Compliance.RequiredFeatures, "Instance Metadata Service v2")
	
	// Test known service with HIPAA
	ec2HIPAACompliance := validator.getServiceComplianceStatus("EC2", ComplianceHIPAA)
	
	assert.Equal(t, "Amazon EC2", ec2HIPAACompliance.ServiceName)
	assert.Equal(t, "ELIGIBLE", ec2HIPAACompliance.ComplianceStatus)
	assert.Contains(t, ec2HIPAACompliance.RequiredFeatures, "Dedicated Tenancy")
	
	// Test unknown service
	unknownCompliance := validator.getServiceComplianceStatus("UnknownService", ComplianceSOC2)
	
	assert.Equal(t, "UnknownService", unknownCompliance.ServiceName)
	assert.Equal(t, "REVIEW_REQUIRED", unknownCompliance.ComplianceStatus)
	assert.Contains(t, unknownCompliance.CertifiedRegions, "us-east-1")
}

// TestAnalyzeSOC2Gaps tests SOC2 gap analysis
func TestAnalyzeSOC2Gaps(t *testing.T) {
	validator := &AWSComplianceValidator{}
	status := &AWSComplianceStatus{}
	
	validator.analyzeSOC2Gaps(status)
	
	assert.NotEmpty(t, status.GapAnalysis)
	
	// Check for expected SOC2 controls
	foundLogicalAccess := false
	foundDataTransmission := false
	
	for _, gap := range status.GapAnalysis {
		if gap.Control == "CC6.1 - Logical Access Controls" {
			foundLogicalAccess = true
		}
		if gap.Control == "CC6.7 - Data Transmission Controls" {
			foundDataTransmission = true
		}
	}
	
	assert.True(t, foundLogicalAccess)
	assert.True(t, foundDataTransmission)
}

// TestAnalyzeHIPAAGaps tests HIPAA gap analysis
func TestAnalyzeHIPAAGaps(t *testing.T) {
	validator := &AWSComplianceValidator{}
	status := &AWSComplianceStatus{}
	
	validator.analyzeHIPAAGaps(status)
	
	assert.NotEmpty(t, status.GapAnalysis)
	
	// Check for HIPAA specific controls
	foundAccessControl := false
	foundTransmissionSecurity := false
	
	for _, gap := range status.GapAnalysis {
		if gap.Control == "§164.312(a)(1) - Access Control" {
			foundAccessControl = true
		}
		if gap.Control == "§164.312(e)(1) - Transmission Security" {
			foundTransmissionSecurity = true
		}
	}
	
	assert.True(t, foundAccessControl)
	assert.True(t, foundTransmissionSecurity)
}

// TestAnalyzeITARGaps tests ITAR gap analysis with commercial region
func TestAnalyzeITARGaps(t *testing.T) {
	validator := &AWSComplianceValidator{
		region: "us-east-1", // Commercial region
	}
	status := &AWSComplianceStatus{}
	
	validator.analyzeITARGaps(status)
	
	assert.NotEmpty(t, status.GapAnalysis)
	
	// Check for ITAR-specific critical gaps
	foundPhysicalSafeguards := false
	foundRegionalCompliance := false
	
	for _, gap := range status.GapAnalysis {
		if gap.Control == "ITAR 120.17 - Physical and Technical Safeguards" {
			foundPhysicalSafeguards = true
			assert.Equal(t, "CRITICAL", gap.Severity)
		}
		if gap.Control == "ITAR Regional Compliance" {
			foundRegionalCompliance = true
			assert.Equal(t, "CRITICAL", gap.Severity)
			assert.Contains(t, gap.CloudWorkstationGap, "us-east-1")
		}
	}
	
	assert.True(t, foundPhysicalSafeguards)
	assert.True(t, foundRegionalCompliance)
}

// TestAnalyzeITARGapsGovCloud tests ITAR gap analysis with GovCloud region
func TestAnalyzeITARGapsGovCloud(t *testing.T) {
	validator := &AWSComplianceValidator{
		region: "us-gov-east-1", // GovCloud region
	}
	status := &AWSComplianceStatus{}
	
	validator.analyzeITARGaps(status)
	
	// Should still have gaps, but no regional compliance gap
	foundRegionalCompliance := false
	
	for _, gap := range status.GapAnalysis {
		if gap.Control == "ITAR Regional Compliance" {
			foundRegionalCompliance = true
		}
	}
	
	assert.False(t, foundRegionalCompliance, "GovCloud region should not trigger regional compliance gap")
}

// TestAnalyzeCMMCGaps tests CMMC gap analysis
func TestAnalyzeCMMCGaps(t *testing.T) {
	validator := &AWSComplianceValidator{}
	status := &AWSComplianceStatus{}
	
	validator.analyzeCMMCGaps(status)
	
	assert.NotEmpty(t, status.GapAnalysis)
	
	// Check for CMMC maturity processes
	foundMaturityProcess := false
	
	for _, gap := range status.GapAnalysis {
		if gap.Control == "CMMC Maturity Processes" {
			foundMaturityProcess = true
			assert.Contains(t, gap.Remediation, "processes")
		}
	}
	
	assert.True(t, foundMaturityProcess)
}

// TestAnalyzeCMMCL1Gaps tests CMMC Level 1 gap analysis
func TestAnalyzeCMMCL1Gaps(t *testing.T) {
	validator := &AWSComplianceValidator{}
	status := &AWSComplianceStatus{}
	
	validator.analyzeCMMCL1Gaps(status)
	
	assert.NotEmpty(t, status.GapAnalysis)
	
	// CMMC Level 1 should show compliance or low-severity gaps
	for _, gap := range status.GapAnalysis {
		assert.Equal(t, "LOW", gap.Severity, "CMMC Level 1 should have only low-severity gaps")
	}
}

// TestAnalyzeCMMCL2Gaps tests CMMC Level 2 gap analysis
func TestAnalyzeCMMCL2Gaps(t *testing.T) {
	validator := &AWSComplianceValidator{}
	status := &AWSComplianceStatus{}
	
	validator.analyzeCMMCL2Gaps(status)
	
	assert.NotEmpty(t, status.GapAnalysis)
	
	// Check for specific CMMC L2 controls
	foundInfoFlow := false
	foundAuditGen := false
	foundKeyMgmt := false
	
	for _, gap := range status.GapAnalysis {
		if gap.Control == "AC.L2-3.1.3 - Control Information Flow" {
			foundInfoFlow = true
		}
		if gap.Control == "AU.L2-3.3.1 - Audit Record Generation" {
			foundAuditGen = true
		}
		if gap.Control == "SC.L2-3.13.11 - Cryptographic Key Management" {
			foundKeyMgmt = true
		}
	}
	
	assert.True(t, foundInfoFlow)
	assert.True(t, foundAuditGen)
	assert.True(t, foundKeyMgmt)
}

// TestAnalyzeFERPAGaps tests FERPA gap analysis
func TestAnalyzeFERPAGaps(t *testing.T) {
	validator := &AWSComplianceValidator{}
	status := &AWSComplianceStatus{}
	
	validator.analyzeFERPAGaps(status)
	
	assert.NotEmpty(t, status.GapAnalysis)
	
	// Check for FERPA-specific controls
	foundDisclosure := false
	foundRecordKeeping := false
	foundDataSecurity := false
	
	for _, gap := range status.GapAnalysis {
		if gap.Control == "FERPA §99.31 - Disclosure without Consent" {
			foundDisclosure = true
		}
		if gap.Control == "FERPA §99.32 - Record of Requests and Disclosures" {
			foundRecordKeeping = true
		}
		if gap.Control == "FERPA Data Security" {
			foundDataSecurity = true
		}
	}
	
	assert.True(t, foundDisclosure)
	assert.True(t, foundRecordKeeping)
	assert.True(t, foundDataSecurity)
}

// TestGetSupportedFrameworks tests supported frameworks list
func TestGetSupportedFrameworks(t *testing.T) {
	validator := &AWSComplianceValidator{}
	
	frameworks := validator.GetSupportedFrameworks()
	
	assert.NotEmpty(t, frameworks)
	assert.Contains(t, frameworks, ComplianceNIST800171)
	assert.Contains(t, frameworks, ComplianceSOC2)
	assert.Contains(t, frameworks, ComplianceHIPAA)
	assert.Contains(t, frameworks, ComplianceFedRAMP)
	assert.Contains(t, frameworks, ComplianceCMMC)
	
	// Should contain major compliance frameworks for research institutions
	assert.Len(t, frameworks, 11) // Update if more frameworks added
}

// TestValidateAWSServices tests AWS service validation
func TestValidateAWSServices(t *testing.T) {
	validator := &AWSComplianceValidator{}
	status := &AWSComplianceStatus{}
	
	err := validator.validateAWSServices(context.Background(), ComplianceSOC2, status)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, status.AWSServices)
	
	// Check that core CloudWorkstation services are covered
	serviceNames := make([]string, len(status.AWSServices))
	for i, service := range status.AWSServices {
		serviceNames[i] = service.ServiceName
	}
	
	assert.Contains(t, serviceNames, "Amazon EC2")
	assert.Contains(t, serviceNames, "Amazon VPC")
	assert.Contains(t, serviceNames, "AWS IAM")
	assert.Contains(t, serviceNames, "AWS CloudTrail")
}

// TestPerformGapAnalysis tests gap analysis for different frameworks
func TestPerformGapAnalysis(t *testing.T) {
	validator := &AWSComplianceValidator{}
	
	frameworks := []ComplianceFramework{
		ComplianceSOC2,
		ComplianceHIPAA,
		ComplianceNIST80053,
		ComplianceFedRAMP,
		ComplianceNIST800171,
		ComplianceITAR,
		ComplianceEAR,
		ComplianceISO27001,
		CompliancePCIDSS,
		ComplianceCMMC,
		ComplianceCMMCL1,
		ComplianceCMMCL2,
		ComplianceCMMCL3,
		ComplianceFISMA,
		ComplianceDFARS,
		ComplianceFERPA,
	}
	
	for _, framework := range frameworks {
		t.Run(string(framework), func(t *testing.T) {
			status := &AWSComplianceStatus{}
			
			err := validator.performGapAnalysis(framework, status)
			
			assert.NoError(t, err)
			// Gap analysis should produce results for supported frameworks
			if framework != ComplianceGDPR && framework != ComplianceCSA && framework != ComplianceENISA && framework != ComplianceC5 {
				assert.NotEmpty(t, status.GapAnalysis, "Framework %s should have gap analysis", framework)
			}
		})
	}
}

// TestGenerateRecommendations tests recommendation generation
func TestGenerateRecommendations(t *testing.T) {
	validator := &AWSComplianceValidator{}
	status := &AWSComplianceStatus{}
	
	validator.generateRecommendations(ComplianceSOC2, status)
	
	assert.NotEmpty(t, status.RecommendedActions)
	
	// Check for base recommendations
	foundConfigRecommendation := false
	foundLoggingRecommendation := false
	
	for _, rec := range status.RecommendedActions {
		if rec.AWSService == "AWS Config" {
			foundConfigRecommendation = true
		}
		if rec.AWSService == "CloudWatch Logs" {
			foundLoggingRecommendation = true
		}
	}
	
	assert.True(t, foundConfigRecommendation)
	assert.True(t, foundLoggingRecommendation)
}

// TestGenerateRecommendationsHIPAA tests HIPAA-specific recommendations
func TestGenerateRecommendationsHIPAA(t *testing.T) {
	validator := &AWSComplianceValidator{}
	status := &AWSComplianceStatus{}
	
	validator.generateRecommendations(ComplianceHIPAA, status)
	
	assert.NotEmpty(t, status.RecommendedActions)
	
	// Check for HIPAA-specific BAA recommendation
	foundBAARecommendation := false
	
	for _, rec := range status.RecommendedActions {
		if rec.Priority == "CRITICAL" && rec.Action == "Sign AWS Business Associate Agreement (BAA)" {
			foundBAARecommendation = true
		}
	}
	
	assert.True(t, foundBAARecommendation)
}

// TestGenerateRecommendationsFedRAMP tests FedRAMP-specific recommendations
func TestGenerateRecommendationsFedRAMP(t *testing.T) {
	validator := &AWSComplianceValidator{}
	status := &AWSComplianceStatus{}
	
	validator.generateRecommendations(ComplianceFedRAMP, status)
	
	assert.NotEmpty(t, status.RecommendedActions)
	
	// Check for FedRAMP-specific GovCloud recommendation
	foundGovCloudRecommendation := false
	
	for _, rec := range status.RecommendedActions {
		if rec.Priority == "CRITICAL" && rec.AWSService == "AWS GovCloud" {
			foundGovCloudRecommendation = true
		}
	}
	
	assert.True(t, foundGovCloudRecommendation)
}

// TestValidateComplianceWithoutAWS tests compliance validation without AWS access
func TestValidateComplianceWithoutAWS(t *testing.T) {
	// Test validation logic without requiring actual AWS credentials
	validator := &AWSComplianceValidator{
		region: "us-east-1",
	}
	
	// Test that validation methods can handle context and return structured results
	ctx := context.Background()
	
	// Test individual analysis methods
	status := &AWSComplianceStatus{}
	
	// These should not require AWS API calls
	validator.analyzeSOC2Gaps(status)
	assert.NotEmpty(t, status.GapAnalysis)
	
	// Reset for next test
	status.GapAnalysis = nil
	
	validator.analyzeHIPAAGaps(status)
	assert.NotEmpty(t, status.GapAnalysis)
	
	// Test service compliance (no AWS API calls required)
	err := validator.validateAWSServices(ctx, ComplianceSOC2, status)
	assert.NoError(t, err)
	assert.NotEmpty(t, status.AWSServices)
}

// TestComplianceFrameworkCoverage tests that all defined frameworks have analysis
func TestComplianceFrameworkCoverage(t *testing.T) {
	validator := &AWSComplianceValidator{}
	
	// Test that performGapAnalysis handles all defined frameworks
	frameworks := []ComplianceFramework{
		ComplianceSOC2,
		ComplianceHIPAA,
		ComplianceNIST80053,
		ComplianceFedRAMP,
		ComplianceNIST800171,
		ComplianceITAR,
		ComplianceEAR,
		ComplianceISO27001,
		CompliancePCIDSS,
		ComplianceCMMC,
		ComplianceCMMCL1,
		ComplianceCMMCL2,
		ComplianceCMMCL3,
		ComplianceFISMA,
		ComplianceDFARS,
		ComplianceFERPA,
	}
	
	for _, framework := range frameworks {
		t.Run(string(framework), func(t *testing.T) {
			status := &AWSComplianceStatus{}
			
			// Should not panic for any framework
			err := validator.performGapAnalysis(framework, status)
			assert.NoError(t, err)
		})
	}
}

// TestComplianceValidationErrorHandling tests error handling in validation
func TestComplianceValidationErrorHandling(t *testing.T) {
	validator := &AWSComplianceValidator{
		region: "us-east-1",
	}
	
	// Test with unsupported framework
	status := &AWSComplianceStatus{}
	
	// Test getArtifactReport with unsupported framework
	err := validator.getArtifactReport(context.Background(), ComplianceFramework("UNSUPPORTED"), status)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported compliance framework")
}

// TestCMMCLevelProgression tests CMMC level progression analysis
func TestCMMCLevelProgression(t *testing.T) {
	validator := &AWSComplianceValidator{
		region: "us-east-1",
	}
	
	// Test CMMC Level progression - each level should build on previous
	levels := []ComplianceFramework{
		ComplianceCMMCL1,
		ComplianceCMMCL2,
		ComplianceCMMCL3,
	}
	
	for _, level := range levels {
		t.Run(string(level), func(t *testing.T) {
			status := &AWSComplianceStatus{}
			
			err := validator.performGapAnalysis(level, status)
			assert.NoError(t, err)
			
			// Higher levels should have more stringent requirements
			if level == ComplianceCMMCL3 {
				foundHighSeverity := false
				for _, gap := range status.GapAnalysis {
					if gap.Severity == "HIGH" || gap.Severity == "CRITICAL" {
						foundHighSeverity = true
						break
					}
				}
				assert.True(t, foundHighSeverity, "CMMC Level 3 should have high-severity requirements")
			}
		})
	}
}

// TestRegionalComplianceValidation tests region-specific compliance validation
func TestRegionalComplianceValidation(t *testing.T) {
	regions := []struct {
		region   string
		isGovCloud bool
	}{
		{"us-east-1", false},
		{"us-west-2", false},
		{"eu-west-1", false},
		{"us-gov-east-1", true},
		{"us-gov-west-1", true},
	}
	
	for _, r := range regions {
		t.Run(r.region, func(t *testing.T) {
			validator := &AWSComplianceValidator{
				region: r.region,
			}
			
			status := &AWSComplianceStatus{}
			
			// Test ITAR gap analysis - should identify region compliance
			validator.analyzeITARGaps(status)
			
			foundRegionalGap := false
			for _, gap := range status.GapAnalysis {
				if gap.Control == "ITAR Regional Compliance" {
					foundRegionalGap = true
					break
				}
			}
			
			// Commercial regions should have regional gap, GovCloud should not
			assert.Equal(t, !r.isGovCloud, foundRegionalGap, 
				"Region %s GovCloud=%v should have regional gap=%v", r.region, r.isGovCloud, !r.isGovCloud)
		})
	}
}