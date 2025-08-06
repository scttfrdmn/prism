// AWS Compliance API handlers for CloudWorkstation daemon
package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/scttfrdmn/cloudworkstation/pkg/security"
)

// handleAWSComplianceValidate handles POST requests to /api/v1/security/compliance/validate/{framework}
func (s *Server) handleAWSComplianceValidate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	frameworkStr := vars["framework"]

	// Map CLI framework names to internal types
	frameworkMapping := map[string]security.ComplianceFramework{
		"nist-800-171": security.ComplianceNIST800171,
		"soc-2":        security.ComplianceSOC2,
		"hipaa":        security.ComplianceHIPAA,
		"fedramp":      security.ComplianceFedRAMP,
		"iso-27001":    security.ComplianceISO27001,
		"pci-dss":      security.CompliancePCIDSS,
		"gdpr":         security.ComplianceGDPR,
		"cmmc":         security.ComplianceCMMC,
		"fisma":        security.ComplianceFISMA,
		"dfars":        security.ComplianceDFARS,
	}

	framework, exists := frameworkMapping[strings.ToLower(frameworkStr)]
	if !exists {
		http.Error(w, fmt.Sprintf("Unsupported compliance framework: %s", frameworkStr), http.StatusBadRequest)
		return
	}

	// Create AWS compliance validator
	validator, err := security.NewAWSComplianceValidator("aws", s.getAWSRegion())
	if err != nil {
		s.securityManager.LogSecurityEvent("aws_compliance_validator_error", false, 
			fmt.Sprintf("Failed to create validator: %v", err), map[string]interface{}{
				"framework": framework,
				"error":     err.Error(),
			})
		http.Error(w, fmt.Sprintf("Failed to create AWS compliance validator: %v", err), http.StatusInternalServerError)
		return
	}

	// Perform compliance validation
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	complianceStatus, err := validator.ValidateCompliance(ctx, framework)
	if err != nil {
		s.securityManager.LogSecurityEvent("aws_compliance_validation_failed", false,
			fmt.Sprintf("Compliance validation failed: %v", err), map[string]interface{}{
				"framework": framework,
				"error":     err.Error(),
			})
		http.Error(w, fmt.Sprintf("Failed to validate compliance: %v", err), http.StatusInternalServerError)
		return
	}

	// Log successful validation
	s.securityManager.LogSecurityEvent("aws_compliance_validated", true,
		fmt.Sprintf("Successfully validated %s compliance", framework), map[string]interface{}{
			"framework":      framework,
			"aws_compliant":  complianceStatus.AWSCompliant,
			"gap_count":      len(complianceStatus.GapAnalysis),
			"recommendations": len(complianceStatus.RecommendedActions),
		})

	// Return validation results
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(complianceStatus); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleAWSComplianceReport handles GET requests to /api/v1/security/compliance/report/{framework}
func (s *Server) handleAWSComplianceReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	frameworkStr := vars["framework"]

	// Map CLI framework names to internal types
	frameworkMapping := map[string]security.ComplianceFramework{
		"nist-800-171": security.ComplianceNIST800171,
		"soc-2":        security.ComplianceSOC2,
		"hipaa":        security.ComplianceHIPAA,
		"fedramp":      security.ComplianceFedRAMP,
		"iso-27001":    security.ComplianceISO27001,
		"pci-dss":      security.CompliancePCIDSS,
		"gdpr":         security.ComplianceGDPR,
		"cmmc":         security.ComplianceCMMC,
		"fisma":        security.ComplianceFISMA,
		"dfars":        security.ComplianceDFARS,
	}

	framework, exists := frameworkMapping[strings.ToLower(frameworkStr)]
	if !exists {
		http.Error(w, fmt.Sprintf("Unsupported compliance framework: %s", frameworkStr), http.StatusBadRequest)
		return
	}

	// Create AWS compliance validator
	validator, err := security.NewAWSComplianceValidator("aws", s.getAWSRegion())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create AWS compliance validator: %v", err), http.StatusInternalServerError)
		return
	}

	// Get compliance status
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	complianceStatus, err := validator.ValidateCompliance(ctx, framework)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to validate compliance: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate comprehensive compliance report
	report := s.generateComplianceReport(framework, complianceStatus)

	// Log report generation
	s.securityManager.LogSecurityEvent("aws_compliance_report_generated", true,
		fmt.Sprintf("Generated %s compliance report", framework), map[string]interface{}{
			"framework":        framework,
			"aws_compliant":   complianceStatus.AWSCompliant,
			"compliance_score": report["compliance_score"],
		})

	// Return report
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(report); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleAWSComplianceSCP handles GET requests to /api/v1/security/compliance/scp/{framework}
func (s *Server) handleAWSComplianceSCP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	frameworkStr := vars["framework"]

	// Map CLI framework names to internal types
	frameworkMapping := map[string]security.ComplianceFramework{
		"nist-800-171": security.ComplianceNIST800171,
		"soc-2":        security.ComplianceSOC2,
		"hipaa":        security.ComplianceHIPAA,
		"fedramp":      security.ComplianceFedRAMP,
		"iso-27001":    security.ComplianceISO27001,
		"pci-dss":      security.CompliancePCIDSS,
		"gdpr":         security.ComplianceGDPR,
		"cmmc":         security.ComplianceCMMC,
		"fisma":        security.ComplianceFISMA,
		"dfars":        security.ComplianceDFARS,
	}

	framework, exists := frameworkMapping[strings.ToLower(frameworkStr)]
	if !exists {
		http.Error(w, fmt.Sprintf("Unsupported compliance framework: %s", frameworkStr), http.StatusBadRequest)
		return
	}

	// Create AWS compliance validator
	validator, err := security.NewAWSComplianceValidator("aws", s.getAWSRegion())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create AWS compliance validator: %v", err), http.StatusInternalServerError)
		return
	}

	// Get compliance status with SCP focus
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	complianceStatus, err := validator.ValidateCompliance(ctx, framework)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to validate compliance: %v", err), http.StatusInternalServerError)
		return
	}

	// Focus on SCP-related data
	scpStatus := map[string]interface{}{
		"framework":        framework,
		"required_scps":    complianceStatus.RequiredSCPs,
		"implemented_scps": complianceStatus.ImplementedSCPs,
		"gaps":             s.filterSCPGaps(complianceStatus.GapAnalysis),
		"last_updated":     complianceStatus.LastUpdated,
	}

	// Log SCP validation
	s.securityManager.LogSecurityEvent("aws_scp_validation", true,
		fmt.Sprintf("Validated SCPs for %s compliance", framework), map[string]interface{}{
			"framework":       framework,
			"required_scps":   len(complianceStatus.RequiredSCPs),
			"implemented_scps": len(complianceStatus.ImplementedSCPs),
			"gaps":           len(s.filterSCPGaps(complianceStatus.GapAnalysis)),
		})

	// Return SCP status
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(scpStatus); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// generateComplianceReport creates a comprehensive compliance report
func (s *Server) generateComplianceReport(framework security.ComplianceFramework, status *security.AWSComplianceStatus) map[string]interface{} {
	// Calculate compliance score
	complianceScore := s.calculateComplianceScore(status)

	report := map[string]interface{}{
		"framework":        framework,
		"aws_compliant":    status.AWSCompliant,
		"compliance_score": complianceScore,
		"last_updated":     status.LastUpdated,
		"artifact_report_id": status.ArtifactReportID,
		"sections": []map[string]interface{}{
			{
				"title": "Executive Summary",
				"content": s.generateExecutiveSummary(framework, status, complianceScore),
			},
			{
				"title": "AWS Service Compliance",
				"content": s.generateServiceComplianceSection(status.AWSServices),
			},
			{
				"title": "Gap Analysis",
				"content": s.generateGapAnalysisSection(status.GapAnalysis),
			},
			{
				"title": "Service Control Policies",
				"content": s.generateSCPSection(status.RequiredSCPs, status.ImplementedSCPs),
			},
			{
				"title": "Recommended Actions",
				"content": s.generateRecommendationsSection(status.RecommendedActions),
			},
			{
				"title": "Compliance Roadmap",
				"content": s.generateComplianceRoadmap(framework, status),
			},
		},
	}

	return report
}

// calculateComplianceScore calculates overall compliance score
func (s *Server) calculateComplianceScore(status *security.AWSComplianceStatus) int {
	score := 100

	// Deduct for gaps
	for _, gap := range status.GapAnalysis {
		switch gap.Severity {
		case "CRITICAL":
			score -= 25
		case "HIGH":
			score -= 15
		case "MEDIUM":
			score -= 10
		case "LOW":
			score -= 5
		}
	}

	// AWS compliance bonus
	if status.AWSCompliant {
		score += 10
	}

	// Service compliance bonus
	for _, service := range status.AWSServices {
		if service.ComplianceStatus == "CERTIFIED" || service.ComplianceStatus == "AUTHORIZED" {
			score += 2
		}
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// generateExecutiveSummary creates executive summary section
func (s *Server) generateExecutiveSummary(framework security.ComplianceFramework, status *security.AWSComplianceStatus, score int) string {
	summary := fmt.Sprintf(`CloudWorkstation %s Compliance Assessment

Overall Compliance Score: %d/100
AWS Service Alignment: %t
Identified Gaps: %d
Recommended Actions: %d

`, framework, score, status.AWSCompliant, len(status.GapAnalysis), len(status.RecommendedActions))

	if status.AWSCompliant {
		summary += "✅ AWS infrastructure is compliant with " + string(framework) + " requirements.\n"
	} else {
		summary += "⚠️ Additional configuration required to achieve full " + string(framework) + " compliance.\n"
	}

	if len(status.GapAnalysis) == 0 {
		summary += "✅ No significant compliance gaps identified.\n"
	} else {
		summary += fmt.Sprintf("📋 %d compliance gaps require attention.\n", len(status.GapAnalysis))
	}

	return summary
}

// generateServiceComplianceSection creates AWS service compliance section
func (s *Server) generateServiceComplianceSection(services []security.AWSServiceCompliance) string {
	if len(services) == 0 {
		return "No AWS service compliance data available."
	}

	content := "AWS Service Compliance Status:\n\n"
	for _, service := range services {
		statusIcon := "✅"
		if service.ComplianceStatus != "CERTIFIED" && service.ComplianceStatus != "AUTHORIZED" {
			statusIcon = "⚠️"
		}

		content += fmt.Sprintf("%s %s: %s\n", statusIcon, service.ServiceName, service.ComplianceStatus)
		
		if len(service.CertifiedRegions) > 0 {
			if len(service.CertifiedRegions) <= 3 {
				content += fmt.Sprintf("   Regions: %s\n", strings.Join(service.CertifiedRegions, ", "))
			} else {
				content += fmt.Sprintf("   Regions: %s... (%d total)\n", 
					strings.Join(service.CertifiedRegions[:3], ", "), len(service.CertifiedRegions))
			}
		}

		if len(service.RequiredFeatures) > 0 {
			content += fmt.Sprintf("   Required Features: %s\n", strings.Join(service.RequiredFeatures, ", "))
		}
		content += "\n"
	}

	return content
}

// generateGapAnalysisSection creates gap analysis section
func (s *Server) generateGapAnalysisSection(gaps []security.ComplianceGap) string {
	if len(gaps) == 0 {
		return "✅ No compliance gaps identified."
	}

	content := "Identified Compliance Gaps:\n\n"
	
	criticalGaps := 0
	highGaps := 0
	
	for _, gap := range gaps {
		severityIcon := "💡"
		switch gap.Severity {
		case "CRITICAL":
			severityIcon = "🚨"
			criticalGaps++
		case "HIGH":
			severityIcon = "⚠️"
			highGaps++
		case "MEDIUM":
			severityIcon = "⚡"
		}

		content += fmt.Sprintf("%s %s - %s\n", severityIcon, gap.Control, gap.Severity)
		content += fmt.Sprintf("   Issue: %s\n", gap.CloudWorkstationGap)
		content += fmt.Sprintf("   Remediation: %s\n\n", gap.Remediation)
	}

	if criticalGaps > 0 {
		content = fmt.Sprintf("🚨 %d CRITICAL gaps require immediate attention.\n\n", criticalGaps) + content
	}
	if highGaps > 0 {
		content = fmt.Sprintf("⚠️ %d HIGH priority gaps should be addressed soon.\n\n", highGaps) + content
	}

	return content
}

// generateSCPSection creates Service Control Policy section
func (s *Server) generateSCPSection(required []string, implemented []string) string {
	content := "Service Control Policy Status:\n\n"

	if len(required) == 0 {
		return content + "No specific SCPs required for this framework."
	}

	implementedMap := make(map[string]bool)
	for _, scp := range implemented {
		implementedMap[scp] = true
	}

	missing := 0
	for _, scp := range required {
		if implementedMap[scp] {
			content += fmt.Sprintf("✅ %s: Implemented\n", scp)
		} else {
			content += fmt.Sprintf("❌ %s: Missing\n", scp)
			missing++
		}
	}

	if missing > 0 {
		content += fmt.Sprintf("\n⚠️ %d required SCPs are not implemented.\n", missing)
	}

	return content
}

// generateRecommendationsSection creates recommendations section
func (s *Server) generateRecommendationsSection(recommendations []security.ComplianceRecommendation) string {
	if len(recommendations) == 0 {
		return "No specific recommendations at this time."
	}

	content := "Recommended Actions:\n\n"

	for _, rec := range recommendations {
		priorityIcon := "💡"
		switch rec.Priority {
		case "CRITICAL":
			priorityIcon = "🚨"
		case "HIGH":
			priorityIcon = "⚠️"
		case "MEDIUM":
			priorityIcon = "⚡"
		}

		content += fmt.Sprintf("%s %s Priority: %s\n", priorityIcon, rec.Priority, rec.Action)
		if rec.AWSService != "" {
			content += fmt.Sprintf("   AWS Service: %s\n", rec.AWSService)
		}
		content += fmt.Sprintf("   Benefit: %s\n", rec.Impact)
		if rec.Implementation != "" {
			content += fmt.Sprintf("   Implementation: %s\n", rec.Implementation)
		}
		content += "\n"
	}

	return content
}

// generateComplianceRoadmap creates compliance roadmap section
func (s *Server) generateComplianceRoadmap(framework security.ComplianceFramework, status *security.AWSComplianceStatus) string {
	content := fmt.Sprintf("%s Compliance Roadmap:\n\n", framework)

	// Immediate actions (next 30 days)
	content += "📅 Immediate Actions (Next 30 Days):\n"
	for _, gap := range status.GapAnalysis {
		if gap.Severity == "CRITICAL" {
			content += fmt.Sprintf("   • %s\n", gap.Remediation)
		}
	}

	// Short-term actions (30-90 days)
	content += "\n📅 Short-term Actions (30-90 Days):\n"
	for _, gap := range status.GapAnalysis {
		if gap.Severity == "HIGH" {
			content += fmt.Sprintf("   • %s\n", gap.Remediation)
		}
	}

	// Long-term actions (3-6 months)
	content += "\n📅 Long-term Actions (3-6 Months):\n"
	for _, rec := range status.RecommendedActions {
		if rec.Priority == "MEDIUM" || rec.Priority == "LOW" {
			content += fmt.Sprintf("   • %s\n", rec.Action)
		}
	}

	content += "\n📋 Ongoing Activities:\n"
	content += "   • Regular compliance monitoring and assessment\n"
	content += "   • Security posture reviews and updates\n"
	content += "   • Staff training on compliance requirements\n"
	content += "   • Documentation updates and maintenance\n"

	return content
}

// filterSCPGaps filters gaps related to Service Control Policies
func (s *Server) filterSCPGaps(gaps []security.ComplianceGap) []security.ComplianceGap {
	var scpGaps []security.ComplianceGap
	
	for _, gap := range gaps {
		if strings.Contains(strings.ToUpper(gap.Control), "SCP") || 
		   strings.Contains(strings.ToUpper(gap.Remediation), "SERVICE CONTROL POLICY") {
			scpGaps = append(scpGaps, gap)
		}
	}
	
	return scpGaps
}

// getAWSRegion returns the configured AWS region
func (s *Server) getAWSRegion() string {
	// Default to us-west-2 if not configured
	return "us-west-2"
}