# Prism v0.4.1 Release Notes

## 🎉 Major Stability & Polish Release

This release represents a major stability milestone for Prism, resolving critical GUI issues, fixing daemon communication bugs, and providing a completely reorganized documentation structure. All interfaces (CLI, TUI, GUI) now work seamlessly with improved error handling and user experience.

## 🐛 Critical Bug Fixes

### GUI Content Display Issues ✅
- **Fixed blank white areas** in Dashboard, Instances, Templates, and Storage sections
- **Root cause**: Removed incompatible scroll wrappers that prevented content rendering
- **Impact**: GUI is now fully functional with all sections displaying content properly

### Daemon Version Reporting ✅  
- **Fixed hardcoded version bug** where daemon reported "0.1.0" instead of actual version
- **Added proper version verification** after daemon startup for both CLI and GUI
- **Enhanced error messages** when version mismatches occur
- **Impact**: No more user confusion about daemon/client version mismatches

### CLI Stability Improvements ✅
- **Fixed version command crash** when GitCommit string is shorter than 8 characters  
- **Added robust bounds checking** for version string parsing
- **Improved CLI daemon startup verification** with timeout and retry logic
- **Impact**: Homebrew installations and development builds no longer crash

### Storage API Consistency ✅
- **Fixed JSON unmarshaling errors** in EFS and EBS volume endpoints
- **Root cause**: API returned maps (`{}`) but clients expected arrays (`[]`)
- **Solution**: Convert state maps to arrays at API layer for consistency
- **Impact**: Storage & Volumes section in GUI now works without errors

## 🔧 User Experience Improvements

### System Tray Integration
- **Enhanced window management** when showing GUI from system tray
- **Automatic data refresh** when window is displayed from tray
- **Better connection status detection** with proper timeouts
- **Improved visual feedback** for daemon connection status

### Navigation & Interface
- **Fixed sidebar highlighting** without rebuilding entire sidebar
- **Eliminated GUI threading warnings** and improved stability  
- **More helpful error messages** throughout the application
- **Better daemon connection status** with visual indicators

## 📚 Documentation Organization

### Major Cleanup Completed
- **Before**: 50+ scattered markdown files across root and docs directories
- **After**: Clean, professional structure with logical organization
  - **Root**: 14 essential project files (README, CHANGELOG, core docs)
  - **docs/**: 41 current documentation files organized by category
  - **docs/archive/**: 42 historical files properly archived

### Improved Navigation
- **Comprehensive documentation index** at `docs/index.md`  
- **Category-based organization**: User guides, developer docs, admin guides, templates
- **Clear separation** between current and historical documentation
- **Professional structure** following documentation best practices

## 🔧 Technical Improvements

### API Consistency
- **Storage and volume endpoints** now return arrays instead of maps
- **Consistent JSON responses** across all API endpoints
- **Better error handling** with proper HTTP status codes

### Version System
- **Robust version verification** across CLI and GUI interfaces
- **Proper version reporting** in all components
- **Enhanced build system** with clean compilation across all platforms

### Build & Distribution
- **Updated Homebrew formula** for proper public distribution
- **Complete end-to-end validation** of Homebrew installation process
- **Clean build process** with zero compilation errors

## 🏗️ Installation & Distribution

### Homebrew Formula
This release includes a production-ready Homebrew formula:

```bash
# For public release (future):
brew tap scttfrdmn/prism  
brew install prism

# Current development:
brew install path/to/prism.rb
```

### Multi-Interface Support
- **CLI**: `prism --help` (command-line interface)
- **TUI**: `prism tui` (terminal user interface) 
- **GUI**: `prism-gui` (graphical user interface)

All interfaces now work seamlessly with the same daemon backend.

## 🧪 Validation & Testing

### End-to-End Testing
- **Complete Homebrew installation** tested and validated
- **All GUI sections** verified to display content properly
- **Daemon startup and version verification** tested across interfaces
- **Storage API endpoints** validated with proper array responses

### Cross-Platform Support  
- **macOS**: Intel and Apple Silicon support
- **Linux**: AMD64 and ARM64 support  
- **Build system**: Clean compilation on all platforms

## 🚀 What's Next

With v0.4.1's stability improvements, the foundation is set for exciting v0.5.0 features:
- **Ubuntu Desktop + NICE DCV support** for windowed desktop environments
- **Template desktop switch** to toggle between headless and desktop modes  
- **Auto-browser integration** for seamless desktop connection UX

## 📦 Download & Installation

### GitHub Release
- **Source code**: `prism-0.4.1.tar.gz`
- **SHA256**: `3a747a4e0fd8fd85ee621699b443d288d4e254180acafa5dbaa5674e9e5ee922`

### Requirements
- **Go 1.19+** (for building from source)
- **AWS credentials** configured
- **macOS 10.14+** or **Linux** (GUI requires X11/Wayland)

### Quick Start
```bash
# Install from source
tar -xzf prism-0.4.1.tar.gz
cd prism-0.4.1
make build

# View available templates
./bin/cws templates

# Launch your first research environment
./bin/cws launch "Python Machine Learning" my-research
```

## 🙏 Acknowledgments

This release represents significant improvements in stability, user experience, and code quality. Special thanks to the comprehensive testing and validation that identified and resolved these critical issues.

---

**Full Changelog**: https://github.com/scttfrdmn/prism/compare/v0.4.0...v0.4.1