#!/bin/bash
# Documentation Reorganization Script
# Separates user documentation from architecture/development documentation

set -e

echo "📚 CloudWorkstation Documentation Reorganization"
echo "================================================="
echo ""

# Create new directory structure
echo "Creating directory structure..."
mkdir -p docs/user-guides
mkdir -p docs/admin-guides
mkdir -p docs/architecture
mkdir -p docs/development
mkdir -p docs/releases

echo "✓ Directories created"
echo ""

# === USER DOCUMENTATION ===
echo "📖 Moving user documentation..."

# Getting Started & Installation
[ -f "docs/GETTING_STARTED.md" ] && git mv docs/GETTING_STARTED.md docs/user-guides/
[ -f "docs/LINUX_INSTALLATION.md" ] && git mv docs/LINUX_INSTALLATION.md docs/user-guides/
[ -f "docs/MACOS_DMG_INSTALLATION.md" ] && git mv docs/MACOS_DMG_INSTALLATION.md docs/user-guides/
[ -f "docs/ZERO_SETUP_GUIDE.md" ] && git mv docs/ZERO_SETUP_GUIDE.md docs/user-guides/
[ -f "docs/SCHOOL_PILOT_QUICKSTART.md" ] && git mv docs/SCHOOL_PILOT_QUICKSTART.md docs/user-guides/

# User Guides
[ -f "docs/USER_GUIDE_v0.5.x.md" ] && git mv docs/USER_GUIDE_v0.5.x.md docs/user-guides/
[ -f "docs/GUI_USER_GUIDE.md" ] && git mv docs/GUI_USER_GUIDE.md docs/user-guides/
[ -f "docs/TUI_USER_GUIDE.md" ] && git mv docs/TUI_USER_GUIDE.md docs/user-guides/
[ -f "docs/USER_GUIDE_RESEARCH_USERS.md" ] && git mv docs/USER_GUIDE_RESEARCH_USERS.md docs/user-guides/
[ -f "docs/MULTI_PROFILE_GUIDE.md" ] && git mv docs/MULTI_PROFILE_GUIDE.md docs/user-guides/
[ -f "docs/TROUBLESHOOTING.md" ] && git mv docs/TROUBLESHOOTING.md docs/user-guides/
[ -f "docs/GUI_TROUBLESHOOTING.md" ] && git mv docs/GUI_TROUBLESHOOTING.md docs/user-guides/

# Template Documentation
[ -f "docs/TEMPLATE_FORMAT.md" ] && git mv docs/TEMPLATE_FORMAT.md docs/user-guides/
[ -f "docs/TEMPLATE_FORMAT_ADVANCED.md" ] && git mv docs/TEMPLATE_FORMAT_ADVANCED.md docs/user-guides/
[ -f "docs/TEMPLATE_MARKETPLACE_USER_GUIDE.md" ] && git mv docs/TEMPLATE_MARKETPLACE_USER_GUIDE.md docs/user-guides/

# Web Services
[ -f "docs/WEB_SERVICES_INTEGRATION_GUIDE.md" ] && git mv docs/WEB_SERVICES_INTEGRATION_GUIDE.md docs/user-guides/

echo "✓ User guides moved"
echo ""

# === ADMINISTRATOR DOCUMENTATION ===
echo "🔧 Moving administrator documentation..."

[ -f "docs/ADMINISTRATOR_GUIDE.md" ] && git mv docs/ADMINISTRATOR_GUIDE.md docs/admin-guides/
[ -f "docs/ADMINISTRATOR_GUIDE_BATCH.md" ] && git mv docs/ADMINISTRATOR_GUIDE_BATCH.md docs/admin-guides/
[ -f "docs/BATCH_DEVICE_MANAGEMENT.md" ] && git mv docs/BATCH_DEVICE_MANAGEMENT.md docs/admin-guides/
[ -f "docs/BATCH_INVITATION_GUIDE.md" ] && git mv docs/BATCH_INVITATION_GUIDE.md docs/admin-guides/
[ -f "docs/BATCH_INVITATION_INTERFACE_GUIDE.md" ] && git mv docs/BATCH_INVITATION_INTERFACE_GUIDE.md docs/admin-guides/
[ -f "docs/RESEARCH_USER_MANAGEMENT_GUIDE.md" ] && git mv docs/RESEARCH_USER_MANAGEMENT_GUIDE.md docs/admin-guides/

# Security & Compliance
[ -f "docs/SECURITY_HARDENING_GUIDE.md" ] && git mv docs/SECURITY_HARDENING_GUIDE.md docs/admin-guides/
[ -f "docs/NIST_800_171_COMPLIANCE.md" ] && git mv docs/NIST_800_171_COMPLIANCE.md docs/admin-guides/
[ -f "docs/SECURE_INVITATION_ARCHITECTURE.md" ] && git mv docs/SECURE_INVITATION_ARCHITECTURE.md docs/admin-guides/
[ -f "docs/SECURE_PROFILE_IMPLEMENTATION.md" ] && git mv docs/SECURE_PROFILE_IMPLEMENTATION.md docs/admin-guides/

# Policy Management
[ -f "docs/BASIC_POLICY_EXAMPLES.md" ] && git mv docs/BASIC_POLICY_EXAMPLES.md docs/admin-guides/
[ -f "docs/AMI_POLICY_ENFORCEMENT.md" ] && git mv docs/AMI_POLICY_ENFORCEMENT.md docs/admin-guides/
[ -f "docs/TEMPLATE_POLICY_FRAMEWORK.md" ] && git mv docs/TEMPLATE_POLICY_FRAMEWORK.md docs/admin-guides/

# AWS Setup
[ -f "docs/AWS_IAM_PERMISSIONS.md" ] && git mv docs/AWS_IAM_PERMISSIONS.md docs/admin-guides/
[ -f "docs/PROFILE_EXPORT_IMPORT.md" ] && git mv docs/PROFILE_EXPORT_IMPORT.md docs/admin-guides/

echo "✓ Admin guides moved"
echo ""

# === ARCHITECTURE DOCUMENTATION ===
echo "🏗️ Moving architecture documentation..."

[ -f "docs/GUI_ARCHITECTURE.md" ] && git mv docs/GUI_ARCHITECTURE.md docs/architecture/
[ -f "docs/DAEMON_API_REFERENCE.md" ] && git mv docs/DAEMON_API_REFERENCE.md docs/architecture/
[ -f "docs/DUAL_USER_ARCHITECTURE.md" ] && git mv docs/DUAL_USER_ARCHITECTURE.md docs/architecture/
[ -f "docs/RESEARCH_USER_ARCHITECTURE.md" ] && git mv docs/RESEARCH_USER_ARCHITECTURE.md docs/architecture/
[ -f "docs/PHASE_5A_RESEARCH_USER_ARCHITECTURE.md" ] && git mv docs/PHASE_5A_RESEARCH_USER_ARCHITECTURE.md docs/architecture/
[ -f "docs/PHASE_5C_ADVANCED_STORAGE_ARCHITECTURE.md" ] && git mv docs/PHASE_5C_ADVANCED_STORAGE_ARCHITECTURE.md docs/architecture/
[ -f "docs/TEMPLATE_MARKETPLACE_ARCHITECTURE.md" ] && git mv docs/TEMPLATE_MARKETPLACE_ARCHITECTURE.md docs/architecture/
[ -f "docs/PLUGIN_ARCHITECTURE.md" ] && git mv docs/PLUGIN_ARCHITECTURE.md docs/architecture/
[ -f "docs/GUI_SKINNING_ARCHITECTURE.md" ] && git mv docs/GUI_SKINNING_ARCHITECTURE.md docs/architecture/
[ -f "docs/AUTO_AMI_SYSTEM.md" ] && git mv docs/AUTO_AMI_SYSTEM.md docs/architecture/
[ -f "docs/IDLE_DETECTION.md" ] && git mv docs/IDLE_DETECTION.md docs/architecture/
[ -f "docs/AUTONOMOUS_IDLE_DETECTION.md" ] && git mv docs/AUTONOMOUS_IDLE_DETECTION.md docs/architecture/
[ -f "docs/API_AUTHENTICATION.md" ] && git mv docs/API_AUTHENTICATION.md docs/architecture/
[ -f "docs/WEB_SERVICE_IMPLEMENTATION.md" ] && git mv docs/WEB_SERVICE_IMPLEMENTATION.md docs/architecture/

# Design Documentation
[ -f "docs/GUI_DESIGN_SYSTEM.md" ] && git mv docs/GUI_DESIGN_SYSTEM.md docs/architecture/
[ -f "docs/GUI_UX_DESIGN_REVIEW.md" ] && git mv docs/GUI_UX_DESIGN_REVIEW.md docs/architecture/
[ -f "docs/UI_ALIGNMENT_PRINCIPLES.md" ] && git mv docs/UI_ALIGNMENT_PRINCIPLES.md docs/architecture/
[ -f "docs/UX_EVALUATION_AND_RECOMMENDATIONS.md" ] && git mv docs/UX_EVALUATION_AND_RECOMMENDATIONS.md docs/architecture/

# Planning (Future Work)
[ -f "docs/NICE_DCV_INTEGRATION_PLAN.md" ] && git mv docs/NICE_DCV_INTEGRATION_PLAN.md docs/architecture/
[ -f "docs/SAGEMAKER_INTEGRATION_DESIGN.md" ] && git mv docs/SAGEMAKER_INTEGRATION_DESIGN.md docs/architecture/
[ -f "docs/TABBED_EMBEDDED_CONNECTIONS_PLAN.md" ] && git mv docs/TABBED_EMBEDDED_CONNECTIONS_PLAN.md docs/architecture/

echo "✓ Architecture docs moved"
echo ""

# === DEVELOPMENT DOCUMENTATION ===
echo "💻 Moving development documentation..."

# Development Guides
[ -f "docs/guides/DEVELOPMENT_SETUP.md" ] && git mv docs/guides/DEVELOPMENT_SETUP.md docs/development/
[ -f "docs/guides/TESTING.md" ] && git mv docs/guides/TESTING.md docs/development/
[ -f "docs/guides/WEB_SERVICE_TESTING.md" ] && git mv docs/guides/WEB_SERVICE_TESTING.md docs/development/
[ -f "docs/CODE_QUALITY_BEST_PRACTICES.md" ] && git mv docs/CODE_QUALITY_BEST_PRACTICES.md docs/development/
[ -f "docs/GO_QUALITY_BASELINE_ASSESSMENT.md" ] && git mv docs/GO_QUALITY_BASELINE_ASSESSMENT.md docs/development/

# Testing
[ -f "docs/TESTING_PLAN.md" ] && git mv docs/TESTING_PLAN.md docs/development/
[ -f "docs/TESTING_ROADMAP.md" ] && git mv docs/TESTING_ROADMAP.md docs/development/
[ -f "docs/TESTING_STRATEGY_80_85.md" ] && git mv docs/TESTING_STRATEGY_80_85.md docs/development/
[ -f "docs/TESTING_AND_LINTING.md" ] && git mv docs/TESTING_AND_LINTING.md docs/development/
[ -f "docs/GUI_TESTING_GUIDE.md" ] && git mv docs/GUI_TESTING_GUIDE.md docs/development/
[ -f "docs/SINGLETON_AND_AUTOSTART_TESTING.md" ] && git mv docs/SINGLETON_AND_AUTOSTART_TESTING.md docs/development/

# Implementation Details
[ -f "docs/TEMPLATE_SYSTEM_IMPLEMENTATION.md" ] && git mv docs/TEMPLATE_SYSTEM_IMPLEMENTATION.md docs/development/
[ -f "docs/TEMPLATE_INHERITANCE.md" ] && git mv docs/TEMPLATE_INHERITANCE.md docs/development/
[ -f "docs/VERSION_SYSTEM_IMPLEMENTATION.md" ] && git mv docs/VERSION_SYSTEM_IMPLEMENTATION.md docs/development/
[ -f "docs/DAEMON_AUTO_START_FEATURE.md" ] && git mv docs/DAEMON_AUTO_START_FEATURE.md docs/development/

# Build & Distribution
[ -f "docs/DISTRIBUTION.md" ] && git mv docs/DISTRIBUTION.md docs/development/
[ -f "docs/DMG_BUILD_GUIDE.md" ] && git mv docs/DMG_BUILD_GUIDE.md docs/development/
[ -f "docs/HOMEBREW_TAP.md" ] && git mv docs/HOMEBREW_TAP.md docs/development/
[ -f "docs/HOMEBREW_RELEASE_PROCESS.md" ] && git mv docs/HOMEBREW_RELEASE_PROCESS.md docs/development/
[ -f "docs/CHOCOLATEY_PACKAGE.md" ] && git mv docs/CHOCOLATEY_PACKAGE.md docs/development/
[ -f "docs/CONDA_PACKAGE.md" ] && git mv docs/CONDA_PACKAGE.md docs/development/
[ -f "docs/PACKAGING_IMPROVEMENTS.md" ] && git mv docs/PACKAGING_IMPROVEMENTS.md docs/development/

# Release Management
[ -f "docs/RELEASE_PROCESS.md" ] && git mv docs/RELEASE_PROCESS.md docs/development/
[ -f "docs/RELEASE_UPDATE_CHECKLIST.md" ] && git mv docs/RELEASE_UPDATE_CHECKLIST.md docs/development/
[ -f "docs/PRODUCTION_READINESS_CHECKLIST.md" ] && git mv docs/PRODUCTION_READINESS_CHECKLIST.md docs/development/
[ -f "docs/REAL_TESTER_AWS_VALIDATION_PLAN.md" ] && git mv docs/REAL_TESTER_AWS_VALIDATION_PLAN.md docs/development/

# Repositories & Infrastructure
[ -f "docs/REPOSITORIES.md" ] && git mv docs/REPOSITORIES.md docs/development/

echo "✓ Development docs moved"
echo ""

# === RELEASES ===
echo "📋 Moving release documentation..."

[ -f "docs/RELEASE_NOTES.md" ] && git mv docs/RELEASE_NOTES.md docs/releases/
[ -f "docs/RELEASE_NOTES_v0.5.1.md" ] && git mv docs/RELEASE_NOTES_v0.5.1.md docs/releases/
[ -f "docs/RELEASE_NOTES_v0.5.2.md" ] && git mv docs/RELEASE_NOTES_v0.5.2.md docs/releases/

echo "✓ Release notes moved"
echo ""

# === SESSION/PROJECT STATUS (to archive) ===
echo "📦 Moving session summaries to archive..."

[ -f "docs/SESSION_10_11_GUI_IMPLEMENTATION.md" ] && git mv docs/SESSION_10_11_GUI_IMPLEMENTATION.md docs/archive/sessions/
[ -f "docs/SESSION_12_CONTINUATION_SUMMARY.md" ] && git mv docs/SESSION_12_CONTINUATION_SUMMARY.md docs/archive/sessions/
[ -f "docs/SESSION_12_FINAL_COMPLETE.md" ] && git mv docs/SESSION_12_FINAL_COMPLETE.md docs/archive/sessions/
[ -f "docs/SESSION_12_FINAL_SUMMARY.md" ] && git mv docs/SESSION_12_FINAL_SUMMARY.md docs/archive/sessions/
[ -f "docs/SESSION_12_SUMMARY.md" ] && git mv docs/SESSION_12_SUMMARY.md docs/archive/sessions/
[ -f "docs/SESSION_COMPLETION_OCT17_2025.md" ] && git mv docs/SESSION_COMPLETION_OCT17_2025.md docs/archive/sessions/
[ -f "docs/SESSION_SUMMARY_OCT15_2025.md" ] && git mv docs/SESSION_SUMMARY_OCT15_2025.md docs/archive/sessions/
[ -f "docs/SPRINT_0-2_COMPLETION_SUMMARY.md" ] && git mv docs/SPRINT_0-2_COMPLETION_SUMMARY.md docs/archive/sessions/
[ -f "docs/PHASE_5A_COMPLETION_SUMMARY.md" ] && git mv docs/PHASE_5A_COMPLETION_SUMMARY.md docs/archive/sessions/
[ -f "docs/PROJECT_STATUS_COMPREHENSIVE_v0.5.0.md" ] && git mv docs/PROJECT_STATUS_COMPREHENSIVE_v0.5.0.md docs/archive/sessions/

echo "✓ Session summaries archived"
echo ""

# === PLANNING/ROADMAP (to archive or move) ===
echo "🗺️ Moving planning documentation..."

mkdir -p docs/archive/roadmap
[ -f "docs/PHASE_5_DEVELOPMENT_PLAN.md" ] && git mv docs/PHASE_5_DEVELOPMENT_PLAN.md docs/archive/roadmap/
[ -f "docs/PHASE_5A_MULTI_USER_FOUNDATION_PLAN.md" ] && git mv docs/PHASE_5A_MULTI_USER_FOUNDATION_PLAN.md docs/archive/roadmap/
[ -f "docs/PHASE_5A_POLICY_FRAMEWORK.md" ] && git mv docs/PHASE_5A_POLICY_FRAMEWORK.md docs/archive/roadmap/
[ -f "docs/PHASE_5C_ADVANCED_STORAGE_USER_GUIDE.md" ] && git mv docs/PHASE_5C_ADVANCED_STORAGE_USER_GUIDE.md docs/archive/roadmap/
[ -f "docs/TEMPLATE_MARKETPLACE_PLANNING.md" ] && git mv docs/TEMPLATE_MARKETPLACE_PLANNING.md docs/archive/roadmap/
[ -f "docs/UNIVERSAL_AMI_SYSTEM_PLANNING.md" ] && git mv docs/UNIVERSAL_AMI_SYSTEM_PLANNING.md docs/archive/roadmap/
[ -f "docs/UNIVERSAL_AMI_SYSTEM_ENHANCEMENTS.md" ] && git mv docs/UNIVERSAL_AMI_SYSTEM_ENHANCEMENTS.md docs/archive/roadmap/
[ -f "docs/REMAINING_WORK_ITEMS_2025-08-15.md" ] && git mv docs/REMAINING_WORK_ITEMS_2025-08-15.md docs/archive/roadmap/
[ -f "docs/TECHNICAL_DEBT_BACKLOG.md" ] && git mv docs/TECHNICAL_DEBT_BACKLOG.md docs/archive/roadmap/
[ -f "docs/ROADMAP_OCTOBER_2025_UPDATE.md" ] && git mv docs/ROADMAP_OCTOBER_2025_UPDATE.md docs/archive/roadmap/
[ -f "docs/UPDATED_ROADMAP_DECEMBER_2024.md" ] && git mv docs/UPDATED_ROADMAP_DECEMBER_2024.md docs/archive/roadmap/
[ -f "docs/GUI_FUTURE_ENHANCEMENTS.md" ] && git mv docs/GUI_FUTURE_ENHANCEMENTS.md docs/archive/roadmap/
[ -f "docs/TUI_MOCK_FIX_REMAINING.md" ] && git mv docs/TUI_MOCK_FIX_REMAINING.md docs/archive/roadmap/
[ -f "docs/SYSTEM_IMPLEMENTATION_v0.5.x.md" ] && git mv docs/SYSTEM_IMPLEMENTATION_v0.5.x.md docs/archive/roadmap/

echo "✓ Planning docs archived"
echo ""

echo "================================================="
echo "✅ Documentation reorganization complete!"
echo ""
echo "New structure:"
echo "  docs/user-guides/        - End-user documentation"
echo "  docs/admin-guides/       - Administrator documentation"
echo "  docs/architecture/       - System architecture & design"
echo "  docs/development/        - Developer documentation"
echo "  docs/releases/           - Release notes"
echo "  docs/archive/            - Historical documentation"
echo ""
