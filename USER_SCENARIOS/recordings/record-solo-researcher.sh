#!/bin/bash
# Helper script for recording Solo Researcher CLI workflows
# Run this in your terminal: bash docs/USER_SCENARIOS/recordings/record-solo-researcher.sh

set -e

# Add bin directory to PATH for clean command usage
export PATH="$PWD/bin:$PATH"

RECORDINGS_DIR="docs/USER_SCENARIOS/recordings/01-solo-researcher"
PRISM_BIN="prism"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Prism Solo Researcher Workflow Recording Helper${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# Check asciinema is installed
if ! command -v asciinema &> /dev/null; then
    echo -e "${YELLOW}❌ asciinema not found. Install with: brew install asciinema${NC}"
    exit 1
fi

# Check Prism binary exists
if [ ! -f "bin/prism" ]; then
    echo -e "${YELLOW}❌ Prism binary not found at bin/prism${NC}"
    echo -e "${YELLOW}   Build it with: make build${NC}"
    exit 1
fi

echo -e "${GREEN}✅ asciinema installed (version $(asciinema --version))${NC}"
echo -e "${GREEN}✅ Prism CLI available (version $(prism --version | head -1))${NC}"
echo

# Menu
echo "Select workflow to record:"
echo
echo "  1) prism init wizard         (~45 seconds)"
echo "  2) Daily operations          (~60 seconds)"
echo "  3) Cost tracking             (~30 seconds)"
echo "  4) Record all 3 workflows    (sequential)"
echo "  5) Exit"
echo

read -p "Enter choice [1-5]: " choice

case $choice in
    1)
        echo
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${BLUE}  Recording: prism init wizard${NC}"
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo
        echo "When recording starts:"
        echo "  • Type: $PRISM_BIN init"
        echo "  • Select template (e.g., Bioinformatics Suite)"
        echo "  • Name: sarahs-rnaseq"
        echo "  • Size: M (Medium)"
        echo "  • Confirm: y"
        echo "  • Wait for launch to complete"
        echo "  • Press Ctrl+D to stop recording"
        echo
        read -p "Press Enter to start recording..."
        asciinema rec "$RECORDINGS_DIR/cli-init-wizard.cast"
        echo -e "${GREEN}✅ Recording saved to $RECORDINGS_DIR/cli-init-wizard.cast${NC}"
        ;;

    2)
        echo
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${BLUE}  Recording: Daily operations${NC}"
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo
        echo "When recording starts, run these commands:"
        echo "  • $PRISM_BIN workspace list"
        echo "  • $PRISM_BIN workspace connect <name>"
        echo "  • $PRISM_BIN workspace stop <name>"
        echo "  • $PRISM_BIN workspace list (show stopped)"
        echo "  • $PRISM_BIN workspace start <name>"
        echo "  • Press Ctrl+D to stop recording"
        echo
        read -p "Press Enter to start recording..."
        asciinema rec "$RECORDINGS_DIR/cli-daily-operations.cast"
        echo -e "${GREEN}✅ Recording saved to $RECORDINGS_DIR/cli-daily-operations.cast${NC}"
        ;;

    3)
        echo
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${BLUE}  Recording: Cost tracking${NC}"
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo
        echo "When recording starts, run these commands:"
        echo "  • $PRISM_BIN project costs"
        echo "  • $PRISM_BIN workspace list --verbose"
        echo "  • $PRISM_BIN storage list"
        echo "  • Press Ctrl+D to stop recording"
        echo
        read -p "Press Enter to start recording..."
        asciinema rec "$RECORDINGS_DIR/cli-cost-tracking.cast"
        echo -e "${GREEN}✅ Recording saved to $RECORDINGS_DIR/cli-cost-tracking.cast${NC}"
        ;;

    4)
        echo
        echo -e "${YELLOW}📹 Recording all 3 workflows sequentially...${NC}"
        echo

        # Workflow 1
        echo -e "${BLUE}━━━ Workflow 1/3: prism init wizard ━━━${NC}"
        read -p "Press Enter to start..."
        asciinema rec "$RECORDINGS_DIR/cli-init-wizard.cast"
        echo -e "${GREEN}✅ Workflow 1/3 complete${NC}"
        echo

        # Workflow 2
        echo -e "${BLUE}━━━ Workflow 2/3: Daily operations ━━━${NC}"
        read -p "Press Enter to start..."
        asciinema rec "$RECORDINGS_DIR/cli-daily-operations.cast"
        echo -e "${GREEN}✅ Workflow 2/3 complete${NC}"
        echo

        # Workflow 3
        echo -e "${BLUE}━━━ Workflow 3/3: Cost tracking ━━━${NC}"
        read -p "Press Enter to start..."
        asciinema rec "$RECORDINGS_DIR/cli-cost-tracking.cast"
        echo -e "${GREEN}✅ Workflow 3/3 complete${NC}"
        echo

        echo -e "${GREEN}🎉 All 3 workflows recorded!${NC}"
        ;;

    5)
        echo "Exiting..."
        exit 0
        ;;

    *)
        echo -e "${YELLOW}Invalid choice. Exiting...${NC}"
        exit 1
        ;;
esac

# Review prompt
echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo "To review your recording:"
echo "  asciinema play $RECORDINGS_DIR/<filename>.cast"
echo
echo "To re-record (overwrites):"
echo "  bash docs/USER_SCENARIOS/recordings/record-solo-researcher.sh"
echo
echo "To commit recordings:"
echo "  git add docs/USER_SCENARIOS/recordings/"
echo "  git commit -m \"🎬 Add Solo Researcher CLI workflow recordings\""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
