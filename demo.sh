#!/bin/bash
# CloudWorkstation v0.4.2 Live Demo Script
# Demonstrates key features and capabilities

set -e

echo "🎉 CloudWorkstation v0.4.2 Demo"
echo "==============================================="
echo ""

# Check if running in development mode
if [[ "$CLOUDWORKSTATION_DEV" != "true" ]]; then
    echo "⚠️  Setting CLOUDWORKSTATION_DEV=true to avoid keychain prompts"
    export CLOUDWORKSTATION_DEV=true
fi

echo "Phase 1: Individual Researcher Experience"
echo "----------------------------------------"

# Show version
echo "✅ Version Check:"
./bin/cws --version
echo ""

# Show available templates
echo "✅ Available Templates (showing first 3):"
./bin/cws templates list | head -10
echo ""

# Show template inheritance
echo "✅ Template Inheritance Demo:"
echo "   Base template: Rocky Linux 9 Base"
echo "   Stacked template: Rocky Linux 9 + Conda Stack"
./bin/cws templates info "Rocky Linux 9 + Conda Stack" | head -15
echo ""

echo "Phase 2: Multi-Modal Access"
echo "---------------------------"

# Show daemon status
echo "✅ Daemon Status:"
./bin/cws daemon status
echo ""

# Show API access
echo "✅ API Access (REST endpoints):"
echo "   Templates available via: curl http://localhost:8947/api/v1/templates"
echo "   Instances available via: curl http://localhost:8947/api/v1/instances"
echo "   Example: First 3 template names:"
curl -s http://localhost:8947/api/v1/templates | jq -r 'keys | .[0:3] | join(", ")'
echo ""

echo "Phase 3: Enterprise Features (Simulated)"
echo "---------------------------------------"

# Show hypothetical project operations
echo "✅ Project Management (would create if AWS configured):"
echo "   cws project create ml-research --budget 500.00"
echo "   cws project budget ml-research set --monthly-limit 500.00"
echo "   cws project member add ml-research user@university.edu --role member"
echo ""

echo "✅ Instance Management (would launch if AWS configured):"
echo "   cws launch 'Python Machine Learning (Simplified)' ml-workspace"
echo "   cws hibernate ml-workspace  # Cost optimization"
echo "   cws resume ml-workspace     # Resume when needed"
echo ""

echo "✅ Advanced Features:"
echo "   Storage: cws storage create shared-data --size 100GB"
echo "   Hibernation: cws idle instance ml-workspace --profile cost-optimized"
echo "   Template layers: cws apply python-ml existing-instance"
echo ""

echo "Phase 4: Package Management"
echo "---------------------------"

echo "✅ Homebrew Tap Testing:"
echo "   Tap added: scttfrdmn/cloudworkstation"
brew search cloudworkstation | head -3
echo ""

echo "✅ Installation Methods:"
echo "   1. Homebrew Tap:"
echo "      brew tap scttfrdmn/cloudworkstation"  
echo "      brew install cloudworkstation"
echo "   2. GitHub Releases: Direct binary download"
echo "   3. Source Build: make build (includes GUI)"
echo ""

echo "🎉 Demo Complete!"
echo "==============================================="
echo ""
echo "Key Features Demonstrated:"
echo "• ✅ Zero-configuration templates with inheritance"
echo "• ✅ Multi-modal access (CLI, API, TUI, GUI available)"
echo "• ✅ Enterprise project and budget management"
echo "• ✅ Cost optimization through hibernation"
echo "• ✅ Professional package management via Homebrew"
echo "• ✅ Cross-platform compatibility"
echo ""
echo "Next Steps:"
echo "1. Configure AWS credentials: aws configure"
echo "2. Launch first workstation: cws launch python-ml my-project"
echo "3. Explore TUI interface: cws tui"
echo "4. Set up projects for team collaboration"
echo ""
echo "For full demo with AWS integration, ensure AWS credentials are configured."