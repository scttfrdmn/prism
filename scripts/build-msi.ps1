# Prism Windows MSI Build Script (PowerShell)
# Builds a professional Windows installer using WiX Toolset

param(
    [string]$Version = "",
    [switch]$SkipBuild,
    [switch]$SkipCustomActions,
    [switch]$Verbose,
    [switch]$Clean
)

# Configuration
$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$BuildDir = Join-Path $ProjectRoot "build\windows"
$DistDir = Join-Path $ProjectRoot "dist\windows" 
$WixDir = Join-Path $ProjectRoot "packaging\windows"
$ReleaseDir = Join-Path $BuildDir "release"
if ([string]::IsNullOrEmpty($Version)) {
    # Auto-detect version from pkg/version/version.go
    $versionFile = Join-Path $ProjectRoot "pkg\version\version.go"
    if (Test-Path $versionFile) {
        $match = Select-String -Path $versionFile -Pattern 'Version\s*=\s*"([^"]+)"' | Select-Object -First 1
        if ($match) {
            $Version = $match.Matches[0].Groups[1].Value
        }
    }
    if ([string]::IsNullOrEmpty($Version)) {
        $Version = "0.0.0-dev"
    }
}
$MsiName = "Prism-v$Version-x64.msi"
$LogFile = Join-Path $BuildDir "build-msi.log"

# Color output functions
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    
    $colorMap = @{
        "Red" = [ConsoleColor]::Red
        "Green" = [ConsoleColor]::Green
        "Yellow" = [ConsoleColor]::Yellow
        "Blue" = [ConsoleColor]::Blue
        "Cyan" = [ConsoleColor]::Cyan
        "Magenta" = [ConsoleColor]::Magenta
        "White" = [ConsoleColor]::White
    }
    
    Write-Host $Message -ForegroundColor $colorMap[$Color]
}

function Write-Step {
    param([string]$Message)
    Write-ColorOutput "======================================" "Cyan"
    Write-ColorOutput "  $Message" "Cyan"  
    Write-ColorOutput "======================================" "Cyan"
    Write-Host
}

function Write-Task {
    param([string]$Message)
    Write-ColorOutput $Message "Blue"
}

function Write-Success {
    param([string]$Message)
    Write-ColorOutput "✓ $Message" "Green"
}

function Write-Warning {
    param([string]$Message)
    Write-ColorOutput "⚠ $Message" "Yellow"
}

function Write-ErrorMessage {
    param([string]$Message)
    Write-ColorOutput "✗ $Message" "Red"
}

# Main build function
function Build-MSI {
    Write-Step "Prism Windows MSI Builder"
    
    try {
        # Step 0: Environment validation
        Test-BuildEnvironment
        
        if ($Clean) {
            Clean-BuildArtifacts
        }
        
        # Step 1: Setup build directories
        Initialize-BuildDirectories
        
        # Step 2: Build Go binaries
        if (-not $SkipBuild) {
            Build-GoBinaries
        }
        
        # Step 3: Prepare supporting files
        Prepare-SupportingFiles
        
        # Step 4: Build custom actions DLL (always run — creates placeholder when MSBuild absent)
        Build-CustomActions
        
        # Step 5: Compile WiX source
        Compile-WixSource
        
        # Step 6: Link MSI package
        Link-MsiPackage
        
        # Step 7: Finalize distribution
        Finalize-Distribution
        
        # Step 8: Validation and summary
        Validate-Build
        Show-BuildSummary
        
        Write-Success "Build completed successfully!"
        return $true
        
    } catch {
        Write-ErrorMessage "Build failed: $($_.Exception.Message)"
        if ($Verbose) {
            Write-ErrorMessage "Stack trace: $($_.ScriptStackTrace)"
        }
        return $false
    }
}

function Test-BuildEnvironment {
    Write-Task "Checking build environment..."
    
    # Check for WiX Toolset
    if (-not (Get-Command "candle" -ErrorAction SilentlyContinue)) {
        throw "WiX Toolset not found in PATH. Please install WiX Toolset from https://wixtoolset.org/ or via chocolatey: choco install wixtoolset"
    }
    
    if (-not (Get-Command "light" -ErrorAction SilentlyContinue)) {
        throw "WiX Light tool not found in PATH"
    }
    
    Write-Success "WiX Toolset found"
    
    # Check for Go
    if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
        throw "Go not found in PATH. Please install Go from https://golang.org/"
    }
    
    Write-Success "Go toolchain found"
    
    # Check for MSBuild (for custom actions)
    if (-not (Get-Command "msbuild" -ErrorAction SilentlyContinue)) {
        Write-Warning "MSBuild not found - custom actions will be skipped"
        $script:SkipCustomActions = $true
    } else {
        Write-Success "MSBuild found"
    }
    
    Write-Host
}

function Clean-BuildArtifacts {
    Write-Task "Cleaning previous build artifacts..."
    
    if (Test-Path $BuildDir) {
        Remove-Item -Recurse -Force $BuildDir
        Write-Success "Build directory cleaned"
    }
    
    if (Test-Path $DistDir) {
        Remove-Item -Recurse -Force $DistDir  
        Write-Success "Distribution directory cleaned"
    }
    
    Write-Host
}

function Initialize-BuildDirectories {
    Write-Task "Creating build directories..."
    
    $directories = @(
        $BuildDir,
        $DistDir,
        (Join-Path $BuildDir "obj"),
        (Join-Path $ReleaseDir "windows-amd64"),
        (Join-Path $ReleaseDir "templates"),
        (Join-Path $ReleaseDir "docs"),
        (Join-Path $ReleaseDir "scripts"),
        (Join-Path $ReleaseDir "assets")
    )
    
    foreach ($dir in $directories) {
        if (-not (Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }
    }
    
    Write-Success "Build directories created"
    Write-Host
}

function Build-GoBinaries {
    Write-Step "Building Go binaries"
    
    $originalLocation = Get-Location
    Set-Location $ProjectRoot
    
    try {
        # Set Go build environment for Windows
        $env:GOOS = "windows"
        $env:GOARCH = "amd64" 
        $env:CGO_ENABLED = "0"
        
        $buildDate = Get-Date -Format "yyyy-MM-dd_HH:mm:ss"
        $ldflags = "-X github.com/scttfrdmn/prism/pkg/version.Version=$Version -X github.com/scttfrdmn/prism/pkg/version.BuildDate=$buildDate -X github.com/scttfrdmn/prism/pkg/version.GitCommit=msi-build"
        
        # Build CLI binary
        Write-Task "Building CLI binary (prism.exe)..."
        $cliPath = Join-Path $ReleaseDir "windows-amd64\prism.exe"
        & go build -ldflags $ldflags -o $cliPath ./cmd/prism
        if ($LASTEXITCODE -ne 0) { throw "Failed to build CLI binary" }
        Write-Success "CLI binary built successfully"
        
        # Build daemon binary
        Write-Task "Building daemon binary (prismd.exe)..."
        $daemonPath = Join-Path $ReleaseDir "windows-amd64\prismd.exe"
        & go build -ldflags $ldflags -o $daemonPath ./cmd/prismd
        if ($LASTEXITCODE -ne 0) { throw "Failed to build daemon binary" }
        Write-Success "Daemon binary built successfully"
        
        # Build service wrapper
        Write-Task "Building service wrapper (prismd-service.exe)..."
        $servicePath = Join-Path $ReleaseDir "windows-amd64\prismd-service.exe"
        & go build -ldflags $ldflags -o $servicePath ./cmd/prismd-service
        if ($LASTEXITCODE -ne 0) { throw "Failed to build service wrapper" }
        Write-Success "Service wrapper built successfully"
        
        # Build GUI binary. It's required: an MSI without it must fail rather than ship a
        # placeholder (it used to copy cmd.exe in as prism-gui.exe, #754).
        # cmd/prism-gui is its own Go module and embeds frontend/dist, so build the
        # frontend first and run go in that module's directory.
        Write-Task "Building GUI frontend..."
        $frontendDir = Join-Path $ProjectRoot "cmd\prism-gui\frontend"
        & npm --prefix $frontendDir ci
        if ($LASTEXITCODE -ne 0) { throw "Failed to install GUI frontend dependencies" }
        & npm --prefix $frontendDir run build
        if ($LASTEXITCODE -ne 0) { throw "Failed to build GUI frontend" }

        Write-Task "Building GUI binary (prism-gui.exe)..."
        $guiPath = Join-Path $ReleaseDir "windows-amd64\prism-gui.exe"
        # Wails v3 on Windows needs no cgo; -H windowsgui stops a console window opening with it
        & go -C (Join-Path $ProjectRoot "cmd\prism-gui") build -ldflags "$ldflags -H windowsgui" -o $guiPath .
        if ($LASTEXITCODE -ne 0) { throw "Failed to build GUI binary" }
        Write-Success "GUI binary built successfully"
        
    } finally {
        Set-Location $originalLocation
    }
    
    Write-Host
}

function Prepare-SupportingFiles {
    Write-Step "Preparing supporting files"
    
    # Copy templates
    Write-Task "Copying templates..."
    $templatesSource = Join-Path $ProjectRoot "templates"
    $templatesTarget = Join-Path $ReleaseDir "templates"
    
    if (Test-Path $templatesSource) {
        Copy-Item (Join-Path $templatesSource "*.yml") $templatesTarget -ErrorAction SilentlyContinue
        Copy-Item (Join-Path $templatesSource "*.json") $templatesTarget -ErrorAction SilentlyContinue
        Write-Success "Templates copied"
    } else {
        Write-Warning "Templates directory not found"
    }
    
    # Copy documentation  
    Write-Task "Copying documentation..."
    $docsSource = Join-Path $ProjectRoot "docs"
    $docsTarget = Join-Path $ReleaseDir "docs"
    
    if (Test-Path $docsSource) {
        Copy-Item (Join-Path $docsSource "*.md") $docsTarget -ErrorAction SilentlyContinue
        Write-Success "Documentation copied"
    } else {
        Write-Warning "Documentation directory not found"
    }
    
    # Copy license
    $licenseSource = Join-Path $ProjectRoot "LICENSE"
    if (Test-Path $licenseSource) {
        Copy-Item $licenseSource $ReleaseDir
        Write-Success "License file copied"
    }
    
    # Create PowerShell module
    Write-Task "Creating PowerShell module..."
    $psModuleSource = Join-Path $ProjectRoot "scripts\Prism.psm1"
    $psModuleTarget = Join-Path $ReleaseDir "scripts\Prism.psm1"
    
    if (Test-Path $psModuleSource) {
        Copy-Item $psModuleSource $psModuleTarget
    } else {
        Write-Warning "PowerShell module not found, creating basic one"
        @"
# Prism PowerShell Module
function Get-Prism { prism --help }
Export-ModuleMember -Function Get-Prism
"@ | Out-File $psModuleTarget -Encoding UTF8
    }
    Write-Success "PowerShell module prepared"
    
    # Prepare application icon
    Write-Task "Preparing application icon..."
    $iconSource = Join-Path $ProjectRoot "assets\prism.ico"
    $iconTarget = Join-Path $ReleaseDir "assets\prism.ico"
    
    if (Test-Path $iconSource) {
        Copy-Item $iconSource $iconTarget
        Write-Success "Application icon copied"
    } else {
        Write-Warning "Application icon not found, generating minimal placeholder"
        # Create a minimal valid 16x16 ICO using System.Drawing
        try {
            Add-Type -AssemblyName System.Drawing
            $bmp = New-Object System.Drawing.Bitmap(32, 32)
            $g = [System.Drawing.Graphics]::FromImage($bmp)
            $g.Clear([System.Drawing.Color]::FromArgb(37, 99, 235))
            $g.Dispose()
            $iconHandle = $bmp.GetHicon()
            $icon = [System.Drawing.Icon]::FromHandle($iconHandle)
            $iconStream = [System.IO.File]::Create($iconTarget)
            $icon.Save($iconStream)
            $iconStream.Close()
            $icon.Dispose()
            $bmp.Dispose()
            Write-Success "Placeholder icon generated"
        } catch {
            Write-Warning "Could not generate icon: $_"
            # Write a known-good minimal 1x1 ICO file (16-byte header + 40-byte BITMAPINFOHEADER + 4 bytes pixel data)
            $icoBytes = [byte[]](0,0,1,0,1,0,1,1,0,0,1,0,32,0,40,0,0,0,22,0,0,0,40,0,0,0,1,0,0,0,2,0,0,0,1,0,32,0,0,0,0,0,8,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,37,99,235,255,0,0,0,0)
            [System.IO.File]::WriteAllBytes($iconTarget, $icoBytes)
        }
    }
    
    Write-Host
}

function Build-CustomActions {
    Write-Step "Building Custom Actions DLL"

    $customActionsProject = Join-Path $WixDir "SetupCustomActions\SetupCustomActions.csproj"
    $customActionsDll = Join-Path $ReleaseDir "SetupCustomActions.dll"

    if (-not (Get-Command "msbuild" -ErrorAction SilentlyContinue)) {
        Write-Warning "MSBuild not found - creating placeholder DLL for WiX compilation"
        New-Item -ItemType File -Path $customActionsDll -Force | Out-Null
        $script:SkipCustomActions = $true
    } elseif (Test-Path $customActionsProject) {
        Write-Task "Building SetupCustomActions.dll..."

        & msbuild $customActionsProject /p:Configuration=Release /p:Platform=x64 /p:OutputPath=$ReleaseDir /nologo /verbosity:minimal
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Custom Actions DLL built successfully"
        } else {
            Write-Warning "Custom Actions DLL build failed, continuing without custom actions"
            New-Item -ItemType File -Path $customActionsDll -Force | Out-Null
            $script:SkipCustomActions = $true
        }
    } else {
        Write-Warning "Custom Actions project not found, creating placeholder"
        New-Item -ItemType File -Path $customActionsDll -Force | Out-Null
        $script:SkipCustomActions = $true
    }
    
    Write-Host
}

function Compile-WixSource {
    Write-Step "Compiling WiX source"
    
    $originalLocation = Get-Location
    Set-Location $WixDir
    
    try {
        Write-Task "Running WiX Candle compiler..."
        
        $wixVariables = @(
            "-dSourceDir=$ReleaseDir",
            "-dVersion=$Version"
        )
        
        $objFile = Join-Path $BuildDir "obj\Prism.wixobj"

        $candleOutput = & candle -arch x64 $wixVariables -out $objFile Prism.wxs -ext WixUtilExtension 2>&1
        $candleOutput | Out-File $LogFile -Encoding utf8
        if ($LASTEXITCODE -ne 0) {
            throw "WiX Candle compilation failed:`n$($candleOutput -join "`n")"
        }
        
        Write-Success "WiX source compiled successfully"
        
    } finally {
        Set-Location $originalLocation
    }
    
    Write-Host
}

function Link-MsiPackage {
    Write-Step "Linking MSI package"
    
    Write-Task "Running WiX Light linker..."
    
    $objFile = Join-Path $BuildDir "obj\Prism.wixobj"
    $msiFile = Join-Path $BuildDir $MsiName
    $stringsFile = Join-Path $WixDir "strings_en-us.wxl"
    
    # Create basic strings file if it doesn't exist
    if (-not (Test-Path $stringsFile)) {
        @'
<?xml version="1.0" encoding="utf-8"?>
<WixLocalization Culture="en-us" Language="1033" xmlns="http://schemas.microsoft.com/wix/2006/localization">
  <String Id="LANG">1033</String>
</WixLocalization>
'@ | Out-File $stringsFile -Encoding UTF8
    }
    
    $lightOutput = & light -out $msiFile $objFile -ext WixUIExtension -ext WixUtilExtension -cultures:en-US 2>&1
    $lightOutput | Out-File $LogFile -Encoding utf8 -Append
    if ($LASTEXITCODE -ne 0) {
        throw "WiX Light linking failed:`n$($lightOutput -join "`n")"
    }
    
    Write-Success "MSI package linked successfully"
    Write-Host
}

function Finalize-Distribution {
    Write-Step "Finalizing distribution"
    
    $sourceMsi = Join-Path $BuildDir $MsiName
    $targetMsi = Join-Path $DistDir $MsiName
    
    # Move MSI to distribution directory
    Move-Item $sourceMsi $targetMsi
    Write-Success "MSI moved to distribution directory"
    
    # Generate checksums
    Write-Task "Generating checksums..."
    $hash = Get-FileHash $targetMsi -Algorithm SHA256
    $hashFile = Join-Path $DistDir "$MsiName.sha256"
    $hash.Hash | Out-File $hashFile -Encoding ASCII
    Write-Success "SHA256 checksum generated"
    
    Write-Host
}

function Validate-Build {
    Write-Step "Build validation"
    
    $msiPath = Join-Path $DistDir $MsiName
    
    Write-Task "Validating MSI package..."
    if (Test-Path $msiPath) {
        $msiSize = (Get-Item $msiPath).Length
        Write-Success "MSI package created successfully"
        Write-ColorOutput "  File: $msiPath" "White"
        Write-ColorOutput "  Size: $msiSize bytes" "White"
        
        # Display SHA256 hash
        $hashFile = Join-Path $DistDir "$MsiName.sha256"
        if (Test-Path $hashFile) {
            $hash = Get-Content $hashFile
            Write-ColorOutput "  SHA256: $hash" "White"
        }
    } else {
        throw "MSI package not found at $msiPath"
    }
    
    # Cleanup temporary files
    Write-Task "Cleaning up temporary files..."
    $objDir = Join-Path $BuildDir "obj"
    if (Test-Path $objDir) {
        Remove-Item $objDir -Recurse -Force
    }
    Write-Success "Temporary files cleaned"
    
    Write-Host
}

function Show-BuildSummary {
    Write-ColorOutput "======================================" "Green"
    Write-ColorOutput "  BUILD COMPLETED SUCCESSFULLY!" "Green"
    Write-ColorOutput "======================================" "Green"
    Write-Host
    
    Write-ColorOutput "Prism Windows Installer:" "Cyan"
    Write-ColorOutput "  Location: $(Join-Path $DistDir $MsiName)" "White"
    Write-ColorOutput "  Version:  $Version" "White"
    Write-ColorOutput "  Platform: Windows x64" "White"
    Write-Host
    
    Write-ColorOutput "Installation Commands:" "Cyan"
    Write-ColorOutput "  Silent install:   msiexec /i `"$MsiName`" /quiet" "White"
    Write-ColorOutput "  With logging:     msiexec /i `"$MsiName`" /l*v install.log" "White"  
    Write-ColorOutput "  Uninstall:        msiexec /x `"$MsiName`" /quiet" "White"
    Write-Host
    
    Write-ColorOutput "Next Steps:" "Cyan"
    Write-ColorOutput "  1. Test the installer on a clean Windows system" "White"
    Write-ColorOutput "  2. Verify service installation and startup" "White"
    Write-ColorOutput "  3. Test CLI, daemon, and GUI functionality" "White"
    Write-ColorOutput "  4. Optional: Code sign the MSI for distribution" "White"
    Write-Host
}

# Main execution
if ($MyInvocation.InvocationName -ne '.') {
    $success = Build-MSI
    exit ([int](-not $success))
}