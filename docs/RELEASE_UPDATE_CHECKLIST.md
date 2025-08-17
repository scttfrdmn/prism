# Release Update Checklist

## Issues Identified and Solutions

### 🐛 **Issue 1: Help System Missing Commands**
**Problem**: The `cws help` command is missing `profiles` and `security` commands, even though they exist in v0.4.2.

**Root Cause**: Two help systems exist:
- **Legacy Help** (`cmd/cws/main.go:printUsage()`) - triggered by `--help` flag  
- **Cobra Help** (`internal/cli`) - triggered by `help` command

**Solution Applied**:
1. ✅ **Updated Legacy Help**: Added missing profiles and security commands to `printUsage()` function
2. ✅ **Added Examples**: Included usage examples for new commands
3. 📋 **TODO**: Remove legacy help system completely in next release (use only Cobra)

### 🐛 **Issue 2: Version Mismatch**
**Problem**: Homebrew installation shows "stable 0.4.1" but source shows v0.4.2.

**Root Cause**: Homebrew tap repository still points to v0.4.1 release artifacts.

**Solution Applied**:
1. ✅ **Updated Formula**: Changed from pre-built binaries to source builds
2. ✅ **Enhanced Description**: "Enterprise research management platform"
3. ✅ **Added Revision**: Marks as development/beta release
4. ✅ **Improved Caveats**: Better installation guidance with Cobra CLI

### 🐛 **Issue 3: Homebrew "Stable" Designation**
**Problem**: User wanted different designation than "stable" for development releases.

**Solutions Implemented**:
1. ✅ **Added Revision**: Formula now shows as development/beta release
2. ✅ **Version Suffix**: Uses "0.4.2-dev" to indicate development build
3. ✅ **Clear Messaging**: Caveats explain this includes "latest enterprise features"

## Updated Formula Changes

### **New Formula Structure**:
```ruby
class Cloudworkstation < Formula
  desc "Enterprise research management platform - Launch cloud research environments in seconds"
  # Development/Beta release - includes latest enterprise features
  revision 1
  
  # Use HEAD version for latest features (development builds)
  url "https://github.com/scttfrdmn/cloudworkstation.git", 
      using: :git, revision: "main"
  version "0.4.2-dev"
  
  def install
    # Build from source for latest features and full functionality
    system "make", "build"
    bin.install "bin/cws"
    bin.install "bin/cwsd"
    if OS.mac?
      bin.install "bin/cws-gui"  # GUI available on macOS
    end
  end
end
```

### **Benefits**:
- **Always Latest**: Builds from main branch with latest features
- **Full Functionality**: Includes GUI on macOS, all latest commands
- **Clear Expectations**: Users know they're getting development builds
- **Source Builds**: No dependency on release artifacts

## Remaining Actions Required

### **🔴 Critical: Update Homebrew Tap Repository**
The local formula updates won't take effect until pushed to the tap repository.

**Required Steps**:
1. Push updated `Formula/cloudworkstation.rb` to https://github.com/scttfrdmn/homebrew-cloudworkstation
2. Test installation: `brew uninstall cloudworkstation && brew install scttfrdmn/cloudworkstation/cloudworkstation`
3. Verify: `cws help | grep profiles` should show profiles commands

### **📋 Future Release Tasks**

#### **Next Release (v0.4.3)**:
1. **Remove Legacy Help**: Delete `printUsage()` function, use only Cobra CLI
2. **Homebrew Service Integration**: Add proper `brew services` support
3. **Auto-start Daemon**: Daemon starts automatically after installation
4. **Production Formula**: Consider stable release formula alongside development

#### **Release Process Improvements**:
1. **Automated Formula Updates**: CI/CD to update formula on releases
2. **Version Consistency**: Ensure all help systems show same version info
3. **Release Testing**: Automated testing of Homebrew installation
4. **Documentation**: User guide for different installation methods

## Testing Checklist

### **After Tap Update**:
- [ ] `brew info cloudworkstation` shows v0.4.2-dev
- [ ] `cws --version` shows v0.4.2
- [ ] `cws help` shows complete command list including:
  - [ ] `profiles` commands
  - [ ] `security` commands  
  - [ ] `idle` commands with updated descriptions
- [ ] `cws-gui` available on macOS installations
- [ ] All Cobra commands work: `cws profiles list`, `cws security health`

### **Cross-Platform Testing**:
- [ ] macOS Intel: All features including GUI
- [ ] macOS ARM: All features including GUI  
- [ ] Linux: CLI, TUI, daemon (no GUI expected)

## Impact Summary

### **User Experience Improvements**:
- **Complete Help**: All commands now visible in help system
- **Latest Features**: Access to v0.4.2 enterprise features via Homebrew
- **GUI Included**: macOS users get desktop application
- **Modern CLI**: Cobra-based command system with better UX

### **Development Benefits**:
- **No Release Lag**: Users get latest features immediately
- **Easier Testing**: Community can test features before stable release
- **Better Feedback**: Faster iteration cycle with user input
- **Future-Proof**: Source builds eliminate binary distribution issues

---

**Status**: Formula updated locally, **pending tap repository push** for activation.