#!/bin/bash
# CloudWorkstation v0.4.2-1 Live Demo Script
# Complete workflow from installation to workstation connection
#
# Usage:
#   For Homebrew installation: ./demo.sh
#   For source build: ./demo.sh (will work with both 'cws' in PATH and './bin/cws')

set -e

# Check if we're in a source build environment
if [[ -x "./bin/cws" && -x "./bin/cwsd" && ! $(which cws 2>/dev/null) ]]; then
    echo "🔧 Source build detected - using ./bin/ binaries"
    CWS_CMD="./bin/cws"
    CWSD_CMD="./bin/cwsd"
else
    echo "📦 Using system installation (Homebrew/PATH)"
    CWS_CMD="cws"
    CWSD_CMD="cwsd"
fi

echo "🎉 CloudWorkstation v0.4.2-1 Complete Workflow Demo"
echo "================================================="
echo ""

# Check if running in development mode
if [[ "$CLOUDWORKSTATION_DEV" != "true" ]]; then
    echo "⚠️  Setting CLOUDWORKSTATION_DEV=true to avoid keychain prompts"
    export CLOUDWORKSTATION_DEV=true
fi

echo "Phase 1: Installation & Setup"
echo "-----------------------------"

# Show installation options
echo "✅ Installation Options Demonstrated:"
echo "   1. Homebrew Tap: brew tap scttfrdmn/cloudworkstation && brew install cloudworkstation"
echo "   2. GitHub Releases: Direct binary download"
echo "   3. Source Build: make build (includes GUI)"
echo ""

# Show version
echo "✅ Version Verification:"
$CWS_CMD --version
$CWSD_CMD --version
echo ""

# AWS Configuration
echo "✅ AWS Configuration (CloudWorkstation Profiles - RECOMMENDED):"
echo "   1. aws configure --profile aws"
echo "   2. cws profiles add personal my-research --aws-profile aws --region us-west-2"
echo "   3. cws profiles switch aws"
echo "   4. cws profiles current  # Verify active profile"
echo ""

echo "Phase 2: First Workstation Launch"
echo "---------------------------------"

# Start daemon
echo "✅ Starting Daemon:"
$CWS_CMD daemon start
echo ""

# Show available templates
echo "✅ Available Templates (showing top 5):"
$CWS_CMD templates list | head -12
echo ""

# Show template details with cost info
echo "✅ Template Details with Cost Estimation:"
$CWS_CMD templates info "Python Machine Learning (Simplified)" | head -10
echo ""

echo "✅ Workstation Launch Workflow (simulated - requires AWS):"
echo "   1. cws launch 'Python Machine Learning (Simplified)' ml-research"
echo "   2. cws list                    # Show running instances"
echo "   3. cws info ml-research        # Get connection details"
echo "   4. cws connect ml-research     # KEY STEP: SSH to workstation"
echo "   5. [Inside workstation] whoami, conda list, jupyter --version"
echo "   6. exit                        # Return to local machine"
echo ""

echo "Phase 3: Template Inheritance System"
echo "-----------------------------------"

# Show template inheritance
echo "✅ Template Stacking Architecture:"
echo "   Base: Rocky Linux 9 Base (system + rocky user)"
echo "   Stack: Rocky Linux 9 + Conda Stack (inherits base + adds conda + datascientist user)"
$CWS_CMD templates info "Rocky Linux 9 + Conda Stack" | head -12
echo ""

echo "Phase 4: Multi-Modal Access"
echo "---------------------------"

# Show daemon status and API access
echo "✅ Daemon API Access:"
$CWS_CMD daemon status
echo ""

echo "✅ REST API Endpoints:"
echo "   Templates: curl http://localhost:8947/api/v1/templates"
echo "   Instances: curl http://localhost:8947/api/v1/instances"
echo "   Projects: curl http://localhost:8947/api/v1/projects"
echo "   Example - First 3 template names:"
curl -s http://localhost:8947/api/v1/templates | jq -r 'keys | .[0:3] | join(", ")'
echo ""

echo "✅ TUI Interface Available:"
echo "   cws tui  # Navigate: 1=Dashboard, 2=Instances, 3=Templates, 4=Storage"
echo ""

echo "✅ Profile Management:"
echo "   cws profiles list     # Show all profiles"
echo "   cws profiles current  # Show active profile"
echo "   cws profiles switch <profile>  # Switch profiles"
echo ""

echo "Phase 5: Cost Optimization"
echo "--------------------------"

echo "✅ Hibernation Workflow (simulated - requires AWS):"
echo "   1. cws hibernation-status ml-research      # Check hibernation support"
echo "   2. cws hibernate ml-research               # Save costs, preserve state"
echo "   3. cws list                                # Shows hibernated state"
echo "   4. cws resume ml-research                  # Resume when needed"
echo "   5. cws connect ml-research                 # Environment preserved exactly"
echo ""

echo "✅ Automated Hibernation Policies:"
echo "   cws idle profile list                      # Show available policies"
echo "   cws idle instance ml-research --profile cost-optimized"
echo "   cws idle history                          # Audit trail of actions"
echo ""

echo "Phase 6: Enterprise Features (Simulated)"
echo "---------------------------------------"

echo "✅ Project Management:"
echo "   cws project create ml-research --budget 500.00"
echo "   cws project member add ml-research researcher@university.edu --role member"
echo "   cws project cost ml-research --breakdown"
echo ""

echo "✅ Storage & Advanced Features:"
echo "   cws storage create shared-data --size 100GB --type efs"
echo "   cws storage attach shared-data ml-research /mnt/shared"
echo "   cws connect ml-research → df -h | grep /mnt/shared"
echo ""

echo "Phase 7: Package Management"
echo "---------------------------"

echo "✅ Homebrew Tap Integration:"
brew search cloudworkstation | head -3
echo ""

echo "🎉 Complete Workflow Demo Finished!"
echo "==============================================="
echo ""
echo "✅ Workflow Demonstrated (Installation → Connection):"
echo "1. 📦 Installation: Professional Homebrew tap integration"
echo "2. 🚀 Launch: Zero-config template selection"
echo "3. 🔗 Connect: Direct SSH to pre-configured environment (KEY STEP)"
echo "4. 🧬 Inheritance: Template stacking (Base → Conda Stack)"
echo "5. 💰 Optimization: Hibernation with state preservation"
echo "6. 🏢 Enterprise: Project budgets and collaboration"
echo "7. 📱 Multi-Modal: CLI, TUI, API access"
echo "8. 💾 Storage: Shared storage attachment and verification"
echo ""
echo "🎯 Key Value Propositions:"
echo "• Setup Time: From hours → seconds for research environments"
echo "• Cost Savings: Hibernation preserves work state while reducing costs"
echo "• Collaboration: Project-based organization with budget management"
echo "• Integration: REST API and multi-modal access for any workflow"
echo ""
echo "🚀 Next Steps (complete setup):"
echo "1. aws configure --profile aws                                  # Configure AWS CLI"
echo "2. cws profiles add personal research --aws-profile aws --region us-west-2"
echo "3. cws profiles switch aws                                      # Activate profile"
echo "4. cws launch 'Python Machine Learning' my-project             # Launch workstation"
echo "5. cws connect my-project                                       # SSH to workstation"
echo "6. [Inside workstation] jupyter lab --ip=0.0.0.0               # Start research tools"
echo "7. cws hibernate my-project                                     # Save costs when done"
echo ""
echo "📚 Documentation:"
echo "• Installation Guide: INSTALL.md"
echo "• Complete Demo: DEMO_SEQUENCE.md (15-minute guided tour)"
echo "• Test Results: DEMO_RESULTS.md"