# CloudWorkstation Windows MSI Installer

Professional Windows installer for CloudWorkstation using WiX Toolset, providing enterprise-grade installation experience with Windows service integration.

## 🎯 Features

### **Professional Installation Experience**
- **MSI-based installer** with Windows Installer technology
- **Custom installation wizard** with feature selection
- **System requirement checks** and compatibility validation
- **Progress reporting** with detailed installation steps
- **Error handling** with clear troubleshooting guidance

### **Enterprise Integration**
- **Windows Service** with automatic startup and recovery
- **System PATH integration** for command-line access
- **Start Menu shortcuts** with professional icons
- **Registry integration** for application configuration
- **Administrative installation** with proper security

### **Advanced Features**
- **PowerShell module** for automation and scripting
- **Custom actions DLL** for advanced setup operations
- **Service health monitoring** and connectivity tests
- **First-run setup wizard** for initial configuration
- **Silent installation** support for enterprise deployment

## 🔧 Prerequisites

### **Development Environment**
- **Windows 10/11** (for building)
- **WiX Toolset v3.11+** - Install from [wixtoolset.org](https://wixtoolset.org/)
- **Visual Studio 2019/2022** or Build Tools (for custom actions)
- **Go 1.21+** - For building CloudWorkstation binaries
- **.NET Framework 4.8 SDK** - For custom actions DLL

### **Installation Requirements**
- **Windows 10 version 1903+** or **Windows 11**
- **64-bit architecture** (x64)
- **Administrator privileges** (for installation)
- **100MB+ disk space** (plus space for templates and data)
- **.NET Framework 4.8** (recommended for full functionality)

## 🚀 Quick Start

### **Building the Installer**

1. **Install Prerequisites**:
   ```powershell
   # Install WiX via Chocolatey (recommended)
   choco install wixtoolset
   
   # Or download from: https://wixtoolset.org/releases/
   ```

2. **Build from PowerShell**:
   ```powershell
   # Development build (faster)
   .\scripts\build-msi.ps1 -SkipCustomActions
   
   # Full production build
   .\scripts\build-msi.ps1 -Version 0.4.2
   
   # Build with signing
   .\scripts\build-msi.ps1 -Version 0.4.2
   .\scripts\sign-msi.ps1
   ```

3. **Build from Makefile**:
   ```bash
   # On Windows with make
   make windows-installer
   
   # Service wrapper only
   make windows-service
   
   # Sign MSI
   make windows-sign-msi
   ```

### **Installing CloudWorkstation**

1. **Interactive Installation**:
   ```cmd
   CloudWorkstation-v0.4.2-x64.msi
   ```

2. **Silent Installation**:
   ```cmd
   msiexec /i CloudWorkstation-v0.4.2-x64.msi /quiet
   ```

3. **Installation with Logging**:
   ```cmd
   msiexec /i CloudWorkstation-v0.4.2-x64.msi /l*v install.log
   ```

4. **Custom Installation Directory**:
   ```cmd
   msiexec /i CloudWorkstation-v0.4.2-x64.msi INSTALLFOLDER="C:\Tools\CloudWorkstation"
   ```

## 📁 Installer Architecture

### **Directory Structure**
```
packaging/windows/
├── CloudWorkstation.wxs           # Main WiX installer definition
├── strings_en-us.wxl             # Localization strings
├── SetupCustomActions/           # Custom actions DLL project
│   ├── SetupCustomActions.csproj # C# project file
│   ├── CustomActions.cs          # Main custom actions
│   ├── SystemChecker.cs          # System requirement checks
│   ├── ServiceManager.cs         # Windows service management
│   ├── FirstRunSetup.cs          # Post-installation setup
│   └── packages.config           # NuGet dependencies
└── README.md                     # This documentation

scripts/
├── build-msi.ps1                # PowerShell build script
├── build-msi.bat                # Batch build script
├── sign-msi.ps1                 # MSI signing script
└── CloudWorkstation.psm1        # PowerShell integration module
```

### **Installation Components**

| Component | Description | Install Level |
|-----------|-------------|---------------|
| **Core** | CLI, daemon, service wrapper | Required |
| **Service** | Windows service integration | Default |
| **Templates** | Research environment templates | Default |
| **Documentation** | User guides and help | Default |
| **Start Menu** | Application shortcuts | Default |
| **Desktop** | Desktop shortcut | Optional |
| **PowerShell** | PowerShell module | Default |

### **Installation Paths**

| Path | Purpose |
|------|---------|
| `C:\Program Files\CloudWorkstation\` | Application files |
| `C:\Program Files\CloudWorkstation\bin\` | Executables |
| `C:\Program Files\CloudWorkstation\templates\` | Research templates |
| `C:\Program Files\CloudWorkstation\docs\` | Documentation |
| `C:\ProgramData\CloudWorkstation\` | Configuration |
| `C:\ProgramData\CloudWorkstation\Logs\` | Service logs |

## 🔧 Advanced Configuration

### **Feature Selection**
```cmd
# Install only core components
msiexec /i CloudWorkstation.msi ADDLOCAL=CoreFeature,ServiceFeature

# Skip service installation
msiexec /i CloudWorkstation.msi REMOVE=ServiceFeature

# Custom feature selection
msiexec /i CloudWorkstation.msi ADDLOCAL=CoreFeature,TemplatesFeature,StartMenuFeature
```

### **Silent Installation Options**
```cmd
# Complete silent install
msiexec /i CloudWorkstation.msi /quiet /norestart

# Silent install with progress
msiexec /i CloudWorkstation.msi /passive

# Unattended install (shows progress)
msiexec /i CloudWorkstation.msi /qb
```

### **Enterprise Deployment**
```cmd
# Administrative installation (network share)
msiexec /a CloudWorkstation.msi TARGETDIR="\\server\share\CloudWorkstation"

# Group Policy deployment
# 1. Copy MSI to domain controller
# 2. Create Group Policy Object
# 3. Assign software installation policy
```

## 🛠️ Development Guide

### **Building Custom Actions**

1. **Prerequisites**:
   ```powershell
   # Install .NET Framework 4.8 SDK
   # Install NuGet packages
   nuget restore packaging/windows/SetupCustomActions/packages.config
   ```

2. **Build DLL**:
   ```powershell
   msbuild packaging/windows/SetupCustomActions/SetupCustomActions.csproj /p:Configuration=Release /p:Platform=x64
   ```

3. **Available Custom Actions**:
   - `CheckSystemRequirements` - Validates system compatibility
   - `ConfigureWindowsService` - Sets up service recovery options
   - `VerifyDaemonStartup` - Tests daemon connectivity
   - `LaunchFirstRunWizard` - Post-installation setup
   - `UpdateSystemPath` - Adds binaries to PATH

### **Modifying the Installer**

1. **Edit WiX Source** (`CloudWorkstation.wxs`):
   - Add new components
   - Modify feature definitions  
   - Update registry settings
   - Change installation logic

2. **Customize UI** (`strings_en-us.wxl`):
   - Update dialog text
   - Change feature descriptions
   - Modify installation messages

3. **Extend Custom Actions** (`SetupCustomActions/`):
   - Add new system checks
   - Implement additional setup steps
   - Enhance service configuration

### **Testing the Installer**

1. **Validation Tests**:
   ```powershell
   # Test on clean Windows VM
   # Verify installation paths
   # Check service installation
   # Test CLI functionality
   # Validate uninstallation
   ```

2. **Automated Testing**:
   ```powershell
   # GitHub Actions workflow
   # Builds and tests MSI automatically
   # Validates installation/uninstallation
   ```

## 🔒 Code Signing

### **Certificate Requirements**
- **Code signing certificate** from trusted CA
- **Extended Validation (EV)** certificate recommended
- **Authenticode compatible** certificate format

### **Signing Process**
```powershell
# Sign with certificate file
.\scripts\sign-msi.ps1 -CertificatePath "cert.pfx" -CertificatePassword "password"

# Sign with certificate store
.\scripts\sign-msi.ps1

# Sign and verify
.\scripts\sign-msi.ps1 -Verify
```

### **Verification**
```powershell
# Verify signature
signtool verify /pa /v CloudWorkstation-v0.4.2-x64.msi

# Check certificate details
Get-AuthenticodeSignature CloudWorkstation-v0.4.2-x64.msi
```

## 🚨 Troubleshooting

### **Build Issues**

**WiX Toolset Not Found**:
```powershell
# Install WiX Toolset
choco install wixtoolset

# Add to PATH manually
$env:PATH += ";${env:ProgramFiles(x86)}\WiX Toolset v3.11\bin"
```

**MSBuild Errors**:
```powershell
# Install Visual Studio Build Tools
# Or use Developer Command Prompt
# Ensure .NET Framework 4.8 SDK installed
```

**Custom Actions Build Failed**:
```powershell
# Check NuGet packages
nuget restore packaging/windows/SetupCustomActions/packages.config

# Verify project references
# Check .NET Framework version
```

### **Installation Issues**

**Insufficient Privileges**:
```cmd
# Run as Administrator
# Right-click Command Prompt → "Run as administrator"
# Then run installer
```

**Service Installation Failed**:
```powershell
# Check Windows service dependencies
Get-Service Tcpip, Dhcp

# Verify service executable
Test-Path "C:\Program Files\CloudWorkstation\bin\cwsd-service.exe"

# Manual service installation
sc create CloudWorkstationDaemon binPath="C:\Program Files\CloudWorkstation\bin\cwsd-service.exe"
```

**Path Not Updated**:
```powershell
# Refresh environment variables
$env:PATH = [System.Environment]::GetEnvironmentVariable("PATH", "Machine")

# Or restart Command Prompt/PowerShell
```

### **Runtime Issues**

**Service Won't Start**:
```powershell
# Check service status
Get-Service CloudWorkstationDaemon

# View service logs
Get-EventLog -LogName Application -Source CloudWorkstationDaemon -Newest 10

# Test daemon manually
& "C:\Program Files\CloudWorkstation\bin\cwsd.exe" --version
```

**CLI Not Found**:
```powershell
# Verify installation
Test-Path "C:\Program Files\CloudWorkstation\bin\cws.exe"

# Check PATH
$env:PATH -split ';' | Where-Object { $_ -like "*CloudWorkstation*" }

# Add to PATH manually
$env:PATH += ";C:\Program Files\CloudWorkstation\bin"
```

## 📚 Resources

### **Documentation**
- [WiX Toolset Documentation](https://wixtoolset.org/documentation/)
- [Windows Installer Reference](https://docs.microsoft.com/en-us/windows/win32/msi/)
- [MSBuild Reference](https://docs.microsoft.com/en-us/visualstudio/msbuild/)

### **Tools**
- [WiX Toolset](https://wixtoolset.org/) - MSI creation
- [Orca](https://docs.microsoft.com/en-us/windows/win32/msi/orca-exe) - MSI editor
- [SignTool](https://docs.microsoft.com/en-us/windows/win32/seccrypto/signtool) - Code signing

### **Support**
- [CloudWorkstation Issues](https://github.com/scttfrdmn/cloudworkstation/issues)
- [WiX Community](https://github.com/wixtoolset/wix3)
- [Windows Installer Forums](https://social.msdn.microsoft.com/Forums/windowsdesktop/)

## 🎉 Success Criteria

A successful CloudWorkstation Windows installer should:

✅ **Install cleanly** on Windows 10/11 systems  
✅ **Start Windows service** automatically after installation  
✅ **Add CLI to PATH** for command-line access  
✅ **Create Start Menu shortcuts** for easy access  
✅ **Pass all system requirement checks**  
✅ **Provide clear error messages** on installation issues  
✅ **Uninstall completely** without leaving artifacts  
✅ **Support silent installation** for enterprise deployment  
✅ **Include comprehensive logging** for troubleshooting  
✅ **Work with Windows security policies** and UAC  

---

**CloudWorkstation Windows MSI Installer** - Professional enterprise-ready installation for researchers and institutions.