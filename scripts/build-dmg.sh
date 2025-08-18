#!/bin/bash

# CloudWorkstation DMG Builder
# Creates professional macOS DMG installer package
# Usage: ./scripts/build-dmg.sh [--universal] [--dev]

set -euo pipefail

# Configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
readonly VERSION="0.4.2"
readonly BUILD_DIR="$PROJECT_ROOT/dist/dmg"
readonly VOLUME_NAME="CloudWorkstation-v$VERSION"
readonly DMG_NAME="CloudWorkstation-v$VERSION.dmg"
readonly APP_NAME="CloudWorkstation.app"

# Build flags
UNIVERSAL_BUILD=false
DEV_BUILD=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --universal)
            UNIVERSAL_BUILD=true
            shift
            ;;
        --dev)
            DEV_BUILD=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--universal] [--dev]"
            exit 1
            ;;
    esac
done

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if running on macOS
    if [[ "$(uname)" != "Darwin" ]]; then
        log_error "DMG creation requires macOS"
        exit 1
    fi
    
    # Check for required tools
    local required_tools=("hdiutil" "SetFile" "iconutil")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool '$tool' not found"
            exit 1
        fi
    done
    
    # Check for Xcode tools
    if ! xcode-select -p &> /dev/null; then
        log_error "Xcode command line tools not installed. Run: xcode-select --install"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Build binaries
build_binaries() {
    log_info "Building CloudWorkstation binaries..."
    
    cd "$PROJECT_ROOT"
    
    if [[ "$UNIVERSAL_BUILD" == true ]]; then
        log_info "Building universal binaries (Intel + Apple Silicon)..."
        
        # Create universal binaries
        local temp_dir="$BUILD_DIR/temp"
        mkdir -p "$temp_dir"
        
        # Build for Intel
        log_info "Building for Intel (amd64)..."
        GOOS=darwin GOARCH=amd64 make build-cli build-daemon
        mv bin/cws "$temp_dir/cws-amd64"
        mv bin/cwsd "$temp_dir/cwsd-amd64"
        
        # Build GUI for Intel (if not dev build)
        if [[ "$DEV_BUILD" == false ]]; then
            GOOS=darwin GOARCH=amd64 make build-gui
            mv bin/cws-gui "$temp_dir/cws-gui-amd64"
        fi
        
        # Build for Apple Silicon
        log_info "Building for Apple Silicon (arm64)..."
        GOOS=darwin GOARCH=arm64 make build-cli build-daemon
        mv bin/cws "$temp_dir/cws-arm64"
        mv bin/cwsd "$temp_dir/cwsd-arm64"
        
        # Build GUI for Apple Silicon (if not dev build)
        if [[ "$DEV_BUILD" == false ]]; then
            GOOS=darwin GOARCH=arm64 make build-gui
            mv bin/cws-gui "$temp_dir/cws-gui-arm64"
        fi
        
        # Create universal binaries
        mkdir -p bin
        lipo -create "$temp_dir/cws-amd64" "$temp_dir/cws-arm64" -output "bin/cws"
        lipo -create "$temp_dir/cwsd-amd64" "$temp_dir/cwsd-arm64" -output "bin/cwsd"
        
        if [[ "$DEV_BUILD" == false ]]; then
            lipo -create "$temp_dir/cws-gui-amd64" "$temp_dir/cws-gui-arm64" -output "bin/cws-gui"
        fi
        
        # Clean up temp directory
        rm -rf "$temp_dir"
        
        log_success "Universal binaries created"
    else
        # Build for current architecture
        log_info "Building for current architecture..."
        if [[ "$DEV_BUILD" == true ]]; then
            make build-cli build-daemon
        else
            make build
        fi
    fi
    
    # Verify binaries
    if [[ ! -f "bin/cws" ]] || [[ ! -f "bin/cwsd" ]]; then
        log_error "Failed to build required binaries"
        exit 1
    fi
    
    if [[ "$DEV_BUILD" == false ]] && [[ ! -f "bin/cws-gui" ]]; then
        log_error "Failed to build GUI binary"
        exit 1
    fi
    
    log_success "Binaries built successfully"
}

# Create application bundle structure
create_app_bundle() {
    log_info "Creating application bundle..."
    
    local app_path="$BUILD_DIR/$APP_NAME"
    local contents_path="$app_path/Contents"
    local macos_path="$contents_path/MacOS"
    local resources_path="$contents_path/Resources"
    local frameworks_path="$contents_path/Frameworks"
    
    # Clean and create directory structure
    rm -rf "$app_path"
    mkdir -p "$macos_path" "$resources_path" "$frameworks_path"
    
    # Copy binaries
    cp "$PROJECT_ROOT/bin/cws" "$macos_path/"
    cp "$PROJECT_ROOT/bin/cwsd" "$macos_path/"
    if [[ "$DEV_BUILD" == false ]]; then
        cp "$PROJECT_ROOT/bin/cws-gui" "$macos_path/"
    fi
    
    # Create main launcher script
    create_launcher_script "$macos_path/CloudWorkstation"
    
    # Make binaries and launcher executable
    chmod +x "$macos_path"/*
    
    # Create Info.plist
    create_info_plist "$contents_path/Info.plist"
    
    # Copy resources
    copy_resources "$resources_path"
    
    log_success "Application bundle created at $app_path"
}

# Create launcher script
create_launcher_script() {
    local launcher_path="$1"
    
    log_info "Creating launcher script..."
    
    cat > "$launcher_path" << 'EOF'
#!/bin/bash

# CloudWorkstation Launcher
# Main entry point for the macOS application bundle

set -euo pipefail

# Get the directory of this script
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly RESOURCES_DIR="$(cd "$SCRIPT_DIR/../Resources" && pwd)"

# Function to show error dialog
show_error() {
    local message="$1"
    osascript << EOD
        display dialog "$message" with title "CloudWorkstation Error" buttons {"OK"} default button "OK" with icon stop
EOD
}

# Function to show info dialog
show_info() {
    local message="$1"
    osascript << EOD
        display dialog "$message" with title "CloudWorkstation" buttons {"OK"} default button "OK" with icon note
EOD
}

# Function to check if daemon is running
is_daemon_running() {
    pgrep -f "cwsd" > /dev/null 2>&1
}

# Function to start daemon
start_daemon() {
    if ! is_daemon_running; then
        echo "Starting CloudWorkstation daemon..."
        "$SCRIPT_DIR/cwsd" > /tmp/cwsd.log 2>&1 &
        sleep 2
        
        if is_daemon_running; then
            echo "Daemon started successfully"
        else
            show_error "Failed to start CloudWorkstation daemon. Check /tmp/cwsd.log for details."
            exit 1
        fi
    fi
}

# Function to install command line tools
install_cli_tools() {
    local install_needed=false
    
    # Check if CLI tools are in PATH
    if ! command -v cws &> /dev/null; then
        install_needed=true
    fi
    
    if [[ "$install_needed" == true ]]; then
        # Ask user if they want to install CLI tools
        local response
        response=$(osascript << 'EOD'
            display dialog "Would you like to install CloudWorkstation command-line tools? This will add 'cws' and 'cwsd' commands to your PATH." with title "CloudWorkstation Setup" buttons {"Skip", "Install"} default button "Install" with icon question
EOD
        )
        
        if [[ "$response" == *"Install"* ]]; then
            # Run installation script
            "$RESOURCES_DIR/scripts/install-cli-tools.sh"
        fi
    fi
}

# Function to show welcome screen
show_welcome() {
    local response
    response=$(osascript << 'EOD'
        display dialog "Welcome to CloudWorkstation!

CloudWorkstation helps academic researchers launch pre-configured cloud workstations in seconds.

What would you like to do?" with title "CloudWorkstation v0.4.2" buttons {"Open GUI", "Command Line Setup", "Quit"} default button "Open GUI" with icon note
EOD
    )
    
    case "$response" in
        *"Open GUI"*)
            launch_gui
            ;;
        *"Command Line Setup"*)
            launch_cli_setup
            ;;
        *"Quit"*)
            exit 0
            ;;
    esac
}

# Function to launch GUI
launch_gui() {
    if [[ -f "$SCRIPT_DIR/cws-gui" ]]; then
        start_daemon
        exec "$SCRIPT_DIR/cws-gui"
    else
        show_error "GUI component not available in this build."
        exit 1
    fi
}

# Function to launch CLI setup
launch_cli_setup() {
    install_cli_tools
    
    # Open Terminal with setup commands
    osascript << EOD
        tell application "Terminal"
            activate
            do script "echo 'CloudWorkstation CLI Setup'; echo ''; echo 'Available commands:'; echo '  cws --help        # Show help'; echo '  cws templates     # List templates'; echo '  cws profiles      # Manage AWS profiles'; echo ''; echo 'Getting started:'; echo '  cws profiles create my-profile  # Create AWS profile'; echo '  cws launch python-ml my-project  # Launch workstation'; echo ''"
        end tell
EOD
}

# Main execution
main() {
    # Change to the application directory
    cd "$SCRIPT_DIR"
    
    # Install CLI tools if needed
    install_cli_tools
    
    # Show welcome screen
    show_welcome
}

# Run main function
main "$@"
EOF
    
    chmod +x "$launcher_path"
    log_success "Launcher script created"
}

# Create Info.plist
create_info_plist() {
    local plist_path="$1"
    
    log_info "Creating Info.plist..."
    
    cat > "$plist_path" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>CloudWorkstation</string>
    
    <key>CFBundleDisplayName</key>
    <string>CloudWorkstation</string>
    
    <key>CFBundleIdentifier</key>
    <string>com.cloudworkstation.app</string>
    
    <key>CFBundleVersion</key>
    <string>$VERSION</string>
    
    <key>CFBundleShortVersionString</key>
    <string>$VERSION</string>
    
    <key>CFBundleExecutable</key>
    <string>CloudWorkstation</string>
    
    <key>CFBundleIconFile</key>
    <string>CloudWorkstation.icns</string>
    
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    
    <key>CFBundleSignature</key>
    <string>CWS4</string>
    
    <key>LSMinimumSystemVersion</key>
    <string>10.15</string>
    
    <key>NSHighResolutionCapable</key>
    <true/>
    
    <key>LSApplicationCategoryType</key>
    <string>public.app-category.developer-tools</string>
    
    <key>NSHumanReadableCopyright</key>
    <string>© 2024 CloudWorkstation. All rights reserved.</string>
    
    <key>LSArchitecturePriority</key>
    <array>
        <string>arm64</string>
        <string>x86_64</string>
    </array>
    
    <key>NSAppleEventsUsageDescription</key>
    <string>CloudWorkstation uses AppleScript to provide a better user experience by opening Terminal windows and displaying dialogs.</string>
    
    <key>NSSystemAdministrationUsageDescription</key>
    <string>CloudWorkstation may require administrator privileges to install command-line tools and configure system services.</string>
</dict>
</plist>
EOF
    
    log_success "Info.plist created"
}

# Copy resources to app bundle
copy_resources() {
    local resources_path="$1"
    
    log_info "Copying resources..."
    
    # Create icon file
    create_app_icon "$resources_path/CloudWorkstation.icns"
    
    # Copy templates
    cp -r "$PROJECT_ROOT/templates" "$resources_path/"
    
    # Create scripts directory
    mkdir -p "$resources_path/scripts"
    
    # Create CLI installation script
    create_cli_install_script "$resources_path/scripts/install-cli-tools.sh"
    
    # Copy service management script
    cp "$PROJECT_ROOT/scripts/service-manager.sh" "$resources_path/scripts/"
    
    # Create README
    create_readme "$resources_path/README.txt"
    
    log_success "Resources copied"
}

# Create application icon
create_app_icon() {
    local icon_path="$1"
    
    log_info "Creating application icon..."
    
    # Create iconset directory
    local iconset_dir="$BUILD_DIR/CloudWorkstation.iconset"
    mkdir -p "$iconset_dir"
    
    # Create icon sizes (we'll generate from a base image or use a simple programmatic approach)
    # For now, we'll create a simple programmatic icon using built-in macOS tools
    
    local base_icon="$PROJECT_ROOT/assets/icon.png"
    if [[ -f "$base_icon" ]]; then
        # Use existing icon if available
        log_info "Using existing icon file"
        
        # Generate all required icon sizes
        local sizes=(16 32 64 128 256 512 1024)
        for size in "${sizes[@]}"; do
            sips -z "$size" "$size" "$base_icon" --out "$iconset_dir/icon_${size}x${size}.png" &> /dev/null
            if [[ "$size" -le 512 ]]; then
                # Create @2x versions for Retina displays
                local retina_size=$((size * 2))
                sips -z "$retina_size" "$retina_size" "$base_icon" --out "$iconset_dir/icon_${size}x${size}@2x.png" &> /dev/null
            fi
        done
    else
        # Generate a simple programmatic icon
        log_warning "No icon file found, generating simple icon"
        
        local temp_icon="/tmp/cloudworkstation_temp_icon.png"
        
        # Create a simple icon using built-in tools
        python3 << 'PYTHON_EOF'
import os
from PIL import Image, ImageDraw, ImageFont
import sys

# Create base icon
size = 1024
img = Image.new('RGB', (size, size), color='#2563eb')  # Blue background
draw = ImageDraw.Draw(img)

# Draw cloud-like shape
cloud_color = '#ffffff'
# Simple cloud representation with circles
center_x, center_y = size // 2, size // 2
radius = size // 8

# Main cloud body
draw.ellipse([center_x - radius*2, center_y - radius//2, center_x + radius*2, center_y + radius//2], fill=cloud_color)
# Cloud bumps
draw.ellipse([center_x - radius*1.5, center_y - radius, center_x - radius//2, center_y], fill=cloud_color)
draw.ellipse([center_x + radius//2, center_y - radius, center_x + radius*1.5, center_y], fill=cloud_color)
draw.ellipse([center_x - radius//2, center_y - radius*1.2, center_x + radius//2, center_y - radius//5], fill=cloud_color)

# Add "CW" text
try:
    font = ImageFont.truetype("/System/Library/Fonts/Helvetica.ttc", size//8)
    text = "CW"
    bbox = draw.textbbox((0, 0), text, font=font)
    text_width = bbox[2] - bbox[0]
    text_height = bbox[3] - bbox[1]
    text_x = (size - text_width) // 2
    text_y = center_y + radius//3
    draw.text((text_x, text_y), text, font=font, fill='#1e40af')
except:
    # Fallback without custom font
    draw.text((center_x - 30, center_y + 20), "CW", fill='#1e40af')

# Save
img.save('/tmp/cloudworkstation_temp_icon.png')
PYTHON_EOF
        
        if [[ ! -f "$temp_icon" ]]; then
            # Fallback: create with sips
            log_warning "Creating minimal fallback icon"
            # Create a simple colored square
            sips -c RGBA 512 512 --fillColor blue --backgroundColor blue /dev/null --out "$temp_icon" &> /dev/null || {
                # Ultimate fallback: copy system app icon
                cp "/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/GenericApplicationIcon.icns" "$icon_path"
                return
            }
        fi
        
        base_icon="$temp_icon"
        
        # Generate icon sizes
        local sizes=(16 32 64 128 256 512 1024)
        for size in "${sizes[@]}"; do
            sips -z "$size" "$size" "$base_icon" --out "$iconset_dir/icon_${size}x${size}.png" &> /dev/null
            if [[ "$size" -le 512 ]]; then
                local retina_size=$((size * 2))
                sips -z "$retina_size" "$retina_size" "$base_icon" --out "$iconset_dir/icon_${size}x${size}@2x.png" &> /dev/null
            fi
        done
        
        # Clean up temp icon
        rm -f "$temp_icon"
    fi
    
    # Create .icns file
    iconutil -c icns "$iconset_dir" -o "$icon_path"
    
    # Clean up iconset directory
    rm -rf "$iconset_dir"
    
    log_success "Application icon created"
}

# Create CLI tools installation script
create_cli_install_script() {
    local script_path="$1"
    
    log_info "Creating CLI installation script..."
    
    cat > "$script_path" << 'EOF'
#!/bin/bash

# CloudWorkstation CLI Tools Installer
# Installs cws and cwsd command-line tools to /usr/local/bin

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly MACOS_DIR="$(cd "$SCRIPT_DIR/../../MacOS" && pwd)"
readonly INSTALL_DIR="/usr/local/bin"

# Function to show error dialog
show_error() {
    local message="$1"
    osascript << EOD
        display dialog "$message" with title "CloudWorkstation CLI Install Error" buttons {"OK"} default button "OK" with icon stop
EOD
}

# Function to show success dialog
show_success() {
    local message="$1"
    osascript << EOD
        display dialog "$message" with title "CloudWorkstation CLI Install" buttons {"OK"} default button "OK" with icon note
EOD
}

# Main installation function
main() {
    echo "Installing CloudWorkstation CLI tools..."
    
    # Check if installation directory exists
    if [[ ! -d "$INSTALL_DIR" ]]; then
        show_error "Installation directory $INSTALL_DIR does not exist. Please create it first or install using Homebrew instead."
        exit 1
    fi
    
    # Check if we can write to installation directory
    if [[ ! -w "$INSTALL_DIR" ]]; then
        echo "Need administrator privileges to install to $INSTALL_DIR"
        # Try with sudo
        if ! sudo -n true 2>/dev/null; then
            # Request password
            local password
            password=$(osascript << 'EOD'
                display dialog "Administrator password required to install CloudWorkstation CLI tools to /usr/local/bin:" with title "CloudWorkstation CLI Install" default answer "" with hidden answer buttons {"Cancel", "OK"} default button "OK"
EOD
            )
            
            if [[ "$password" == *"OK"* ]]; then
                password=$(echo "$password" | sed 's/.*text returned://; s/, button returned:.*//')
                echo "$password" | sudo -S true
            else
                echo "Installation cancelled by user"
                exit 0
            fi
        fi
        
        # Install with sudo
        sudo cp "$MACOS_DIR/cws" "$INSTALL_DIR/"
        sudo cp "$MACOS_DIR/cwsd" "$INSTALL_DIR/"
        sudo chmod +x "$INSTALL_DIR/cws" "$INSTALL_DIR/cwsd"
    else
        # Install without sudo
        cp "$MACOS_DIR/cws" "$INSTALL_DIR/"
        cp "$MACOS_DIR/cwsd" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/cws" "$INSTALL_DIR/cwsd"
    fi
    
    # Verify installation
    if [[ -x "$INSTALL_DIR/cws" ]] && [[ -x "$INSTALL_DIR/cwsd" ]]; then
        show_success "CloudWorkstation CLI tools installed successfully!

Commands available:
• cws --help    (CLI client)
• cwsd          (daemon service)

You can now use CloudWorkstation from any terminal window."
        echo "✅ Installation complete"
    else
        show_error "Installation failed. Please check permissions and try again."
        exit 1
    fi
}

# Run installation
main "$@"
EOF
    
    chmod +x "$script_path"
    log_success "CLI installation script created"
}

# Create README file
create_readme() {
    local readme_path="$1"
    
    log_info "Creating README.txt..."
    
    cat > "$readme_path" << EOF
CloudWorkstation v$VERSION - macOS Installation
============================================

Thank you for downloading CloudWorkstation!

INSTALLATION INSTRUCTIONS:
1. Drag CloudWorkstation.app to your Applications folder
2. Launch CloudWorkstation from Applications or Spotlight
3. Follow the setup wizard to configure AWS credentials
4. Start launching cloud workstations!

WHAT'S INCLUDED:
• CloudWorkstation GUI - Visual interface for managing workstations
• Command-line tools (cws, cwsd) - Terminal interface and daemon
• Pre-configured templates for research environments
• Automatic service management

FIRST TIME SETUP:
1. Open CloudWorkstation
2. Choose "Command Line Setup" to install CLI tools (optional)
3. Create an AWS profile: File → AWS Setup or 'cws profiles create'
4. Browse templates and launch your first workstation

GETTING STARTED:
• GUI: Launch CloudWorkstation from Applications
• CLI: Open Terminal and run 'cws --help'
• Templates: 'cws templates' to see available environments
• Launch: 'cws launch python-ml my-project'

SYSTEM REQUIREMENTS:
• macOS 10.15 (Catalina) or later
• AWS account with appropriate permissions
• Internet connection

SUPPORT:
• Documentation: https://github.com/scttfrdmn/cloudworkstation
• Issues: https://github.com/scttfrdmn/cloudworkstation/issues
• User Guide: Open CloudWorkstation → Help → User Guide

UNINSTALLATION:
• Remove CloudWorkstation.app from Applications
• Remove CLI tools: sudo rm /usr/local/bin/cws /usr/local/bin/cwsd
• Remove preferences: ~/Library/Preferences/com.cloudworkstation.*

CloudWorkstation helps academic researchers launch pre-configured
cloud workstations in seconds rather than spending hours setting
up research environments.

Version: $VERSION
Build Date: $(date -u '+%Y-%m-%d %H:%M:%S UTC')

© 2024 CloudWorkstation. All rights reserved.
EOF
    
    log_success "README.txt created"
}

# Create DMG visual assets
create_dmg_assets() {
    log_info "Creating DMG visual assets..."
    
    local assets_dir="$BUILD_DIR/.background"
    mkdir -p "$assets_dir"
    
    # Create DMG background image
    create_dmg_background "$assets_dir/dmg-background.png"
    
    log_success "DMG visual assets created"
}

# Create DMG background image
create_dmg_background() {
    local bg_path="$1"
    
    log_info "Creating DMG background image..."
    
    # Create background using Python/PIL or sips
    python3 << PYTHON_EOF
import os
from PIL import Image, ImageDraw, ImageFont

# Create background image (600x400 for DMG window)
width, height = 600, 400
img = Image.new('RGB', (width, height), color='#f8fafc')  # Light background
draw = ImageDraw.Draw(img)

# Add subtle gradient effect
for y in range(height):
    alpha = int(255 * (1 - y / height * 0.1))  # Very subtle gradient
    color = (248, 250, 252, alpha)
    # Simple gradient effect
    if y % 2 == 0:  # Every other line for subtle texture
        draw.line([(0, y), (width, y)], fill='#f1f5f9')

# Add CloudWorkstation branding
title = "CloudWorkstation"
subtitle = "Academic Research Cloud Platform"

# Title area (center-left, where app icon will be)
title_x, title_y = 50, 50
try:
    # Try to use system font
    title_font = ImageFont.truetype("/System/Library/Fonts/Helvetica.ttc", 32)
    subtitle_font = ImageFont.truetype("/System/Library/Fonts/Helvetica.ttc", 18)
    
    draw.text((title_x, title_y), title, font=title_font, fill='#1e293b')
    draw.text((title_x, title_y + 45), subtitle, font=subtitle_font, fill='#64748b')
except:
    # Fallback without custom fonts
    draw.text((title_x, title_y), title, fill='#1e293b')
    draw.text((title_x, title_y + 40), subtitle, fill='#64748b')

# Add installation instruction
instruction = "Drag CloudWorkstation to Applications to install"
try:
    inst_font = ImageFont.truetype("/System/Library/Fonts/Helvetica.ttc", 14)
    draw.text((50, height - 60), instruction, font=inst_font, fill='#475569')
except:
    draw.text((50, height - 60), instruction, fill='#475569')

# Add arrow pointing from app location to Applications folder
# App icon will be at approximately (150, 200)
# Applications folder will be at approximately (450, 200)
arrow_color = '#3b82f6'
# Simple arrow
arrow_points = [
    (280, 195),  # Start
    (380, 195),  # Middle
    (375, 185),  # Arrow head top
    (380, 195),  # Arrow head middle
    (375, 205)   # Arrow head bottom
]
for i in range(len(arrow_points) - 1):
    draw.line([arrow_points[i], arrow_points[i+1]], fill=arrow_color, width=3)

# Save the image
img.save('$bg_path')
PYTHON_EOF
    
    if [[ ! -f "$bg_path" ]]; then
        log_warning "Failed to create custom background, using fallback"
        # Create simple background with sips
        sips -c RGBA 600 400 --fillColor lightGray --backgroundColor white /dev/null --out "$bg_path" &> /dev/null || {
            # Ultimate fallback: create empty file
            touch "$bg_path"
        }
    fi
    
    log_success "DMG background image created"
}

# Create the DMG
create_dmg() {
    log_info "Creating DMG package..."
    
    local temp_dmg="$BUILD_DIR/temp.dmg"
    local final_dmg="$BUILD_DIR/$DMG_NAME"
    
    # Calculate size needed (app bundle + some overhead)
    local app_size
    app_size=$(du -sm "$BUILD_DIR/$APP_NAME" | cut -f1)
    local dmg_size=$((app_size + 50))  # Add 50MB overhead
    
    # Create temporary DMG
    log_info "Creating temporary DMG ($dmg_size MB)..."
    hdiutil create -srcfolder "$BUILD_DIR/$APP_NAME" -volname "$VOLUME_NAME" -fs HFS+ \
        -fsargs "-c c=64,a=16,e=16" -format UDRW -size "${dmg_size}m" "$temp_dmg"
    
    # Mount the DMG
    log_info "Mounting DMG for customization..."
    local mount_output
    mount_output=$(hdiutil attach -readwrite -noverify -noautoopen "$temp_dmg" | grep '/dev/disk')
    local device
    device=$(echo "$mount_output" | awk '{print $1}')
    local mount_point
    mount_point=$(echo "$mount_output" | sed 's/.*\(\/Volumes\/.*\)/\1/')
    
    if [[ -z "$mount_point" ]]; then
        log_error "Failed to mount DMG"
        exit 1
    fi
    
    log_info "DMG mounted at: $mount_point"
    
    # Create Applications symlink
    ln -s /Applications "$mount_point/Applications"
    
    # Copy visual assets
    create_dmg_assets
    cp -r "$BUILD_DIR/.background" "$mount_point/"
    
    # Copy README
    cp "$BUILD_DIR/$APP_NAME/Contents/Resources/README.txt" "$mount_point/"
    
    # Set custom icon for DMG volume (if we have one)
    if [[ -f "$BUILD_DIR/$APP_NAME/Contents/Resources/CloudWorkstation.icns" ]]; then
        cp "$BUILD_DIR/$APP_NAME/Contents/Resources/CloudWorkstation.icns" "$mount_point/.VolumeIcon.icns"
        SetFile -c icnC "$mount_point/.VolumeIcon.icns"
        SetFile -a C "$mount_point"
    fi
    
    # Set up DMG layout with AppleScript
    setup_dmg_layout "$mount_point"
    
    # Unmount the DMG
    log_info "Finalizing DMG..."
    hdiutil detach "$device"
    
    # Convert to compressed read-only DMG
    log_info "Compressing DMG..."
    hdiutil convert "$temp_dmg" -format UDZO -imagekey zlib-level=9 -o "$final_dmg"
    
    # Clean up temporary DMG
    rm -f "$temp_dmg"
    
    # Verify final DMG
    if [[ -f "$final_dmg" ]]; then
        local final_size
        final_size=$(du -h "$final_dmg" | cut -f1)
        log_success "DMG created successfully: $final_dmg ($final_size)"
        
        # Test mount the final DMG
        log_info "Verifying DMG integrity..."
        if hdiutil verify "$final_dmg" &> /dev/null; then
            log_success "DMG integrity verified"
        else
            log_warning "DMG integrity check failed, but file was created"
        fi
    else
        log_error "Failed to create final DMG"
        exit 1
    fi
}

# Set up DMG window layout
setup_dmg_layout() {
    local mount_point="$1"
    
    log_info "Configuring DMG layout..."
    
    # Create .DS_Store file to control window layout
    osascript << EOF
tell application "Finder"
    tell disk "$VOLUME_NAME"
        open
        set current view of container window to icon view
        set toolbar visible of container window to false
        set statusbar visible of container window to false
        set the bounds of container window to {100, 100, 700, 500}
        set theViewOptions to the icon view options of container window
        set arrangement of theViewOptions to not arranged
        set icon size of theViewOptions to 128
        set background picture of theViewOptions to file ".background:dmg-background.png"
        
        -- Position icons
        set position of item "$APP_NAME" of container window to {150, 200}
        set position of item "Applications" of container window to {450, 200}
        set position of item "README.txt" of container window to {300, 350}
        
        -- Update and close
        update without registering applications
        delay 2
        close
    end tell
end tell
EOF
    
    # Hide background folder
    SetFile -a V "$mount_point/.background"
    
    log_success "DMG layout configured"
}

# Main execution function
main() {
    log_info "Starting CloudWorkstation DMG build process..."
    log_info "Version: $VERSION"
    log_info "Build directory: $BUILD_DIR"
    log_info "Universal build: $UNIVERSAL_BUILD"
    log_info "Development build: $DEV_BUILD"
    
    # Clean and create build directory
    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR"
    
    # Execute build steps
    check_prerequisites
    build_binaries
    create_app_bundle
    create_dmg
    
    log_success "DMG build process completed successfully!"
    log_info "DMG location: $BUILD_DIR/$DMG_NAME"
    
    # Show final instructions
    echo ""
    echo "Next steps:"
    echo "1. Test the DMG: open '$BUILD_DIR/$DMG_NAME'"
    echo "2. Sign the DMG: ./scripts/sign-dmg.sh '$BUILD_DIR/$DMG_NAME'"
    echo "3. Notarize the DMG: ./scripts/notarize-dmg.sh '$BUILD_DIR/$DMG_NAME'"
    echo "4. Distribute via GitHub releases"
    echo ""
}

# Run main function
main "$@"