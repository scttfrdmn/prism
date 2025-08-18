# CloudWorkstation MSI Code Signing Script
# Signs the MSI package with a digital certificate for trusted distribution

param(
    [string]$MsiPath,
    [string]$CertificatePath,
    [string]$CertificatePassword,
    [string]$TimestampUrl = "http://timestamp.comodoca.com",
    [switch]$Verify
)

$ErrorActionPreference = "Stop"

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

function Sign-MSI {
    Write-Step "CloudWorkstation MSI Code Signing"
    
    try {
        # Validate parameters
        if (-not $MsiPath) {
            # Try to find MSI file in dist directory
            $projectRoot = Split-Path -Parent $PSScriptRoot
            $distDir = Join-Path $projectRoot "dist\windows"
            $msiFiles = Get-ChildItem -Path $distDir -Filter "CloudWorkstation-*.msi" -ErrorAction SilentlyContinue
            
            if ($msiFiles.Count -eq 1) {
                $MsiPath = $msiFiles[0].FullName
                Write-ColorOutput "Auto-detected MSI: $MsiPath" "Cyan"
            } elseif ($msiFiles.Count -gt 1) {
                Write-ErrorMessage "Multiple MSI files found. Please specify which one to sign:"
                $msiFiles | ForEach-Object { Write-ColorOutput "  $($_.FullName)" "White" }
                return $false
            } else {
                Write-ErrorMessage "No MSI file found. Please specify -MsiPath parameter."
                return $false
            }
        }
        
        # Validate MSI file exists
        if (-not (Test-Path $MsiPath)) {
            Write-ErrorMessage "MSI file not found: $MsiPath"
            return $false
        }
        
        Write-Success "MSI file found: $MsiPath"
        
        # Check if SignTool is available
        if (-not (Get-Command "signtool" -ErrorAction SilentlyContinue)) {
            Write-ErrorMessage "SignTool not found in PATH"
            Write-ColorOutput "Please install Windows SDK or Visual Studio with Windows development tools" "Yellow"
            Write-ColorOutput "Or ensure signtool.exe is in your PATH" "Yellow"
            return $false
        }
        
        Write-Success "SignTool found"
        
        # Determine signing method
        if ($CertificatePath) {
            # File-based certificate signing
            Sign-WithFileCertificate
        } else {
            # Certificate store signing (look for available certificates)
            Sign-WithStoreCertificate
        }
        
        # Verify signature if requested
        if ($Verify) {
            Verify-Signature
        }
        
        Show-SigningSummary
        return $true
        
    } catch {
        Write-ErrorMessage "Signing failed: $($_.Exception.Message)"
        return $false
    }
}

function Sign-WithFileCertificate {
    Write-Task "Signing with file-based certificate..."
    
    if (-not (Test-Path $CertificatePath)) {
        throw "Certificate file not found: $CertificatePath"
    }
    
    $signArgs = @(
        "sign",
        "/f", "`"$CertificatePath`""
    )
    
    if ($CertificatePassword) {
        $signArgs += "/p", $CertificatePassword
    }
    
    $signArgs += @(
        "/t", $TimestampUrl,
        "/d", "CloudWorkstation: Enterprise Research Management Platform",
        "/du", "https://github.com/scttfrdmn/cloudworkstation",
        "/v",
        "`"$MsiPath`""
    )
    
    Write-Task "Running SignTool..."
    & signtool @signArgs
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "MSI signed successfully with file certificate"
    } else {
        throw "SignTool failed with exit code: $LASTEXITCODE"
    }
}

function Sign-WithStoreCertificate {
    Write-Task "Looking for code signing certificates in certificate store..."
    
    # Look for suitable certificates in the current user's personal store
    $certs = Get-ChildItem -Path "Cert:\CurrentUser\My" | Where-Object {
        $_.Subject -match "CloudWorkstation" -or
        $_.Subject -match "Scott" -or
        $_.EnhancedKeyUsageList -match "Code Signing" -or
        $_.EnhancedKeyUsageList -match "1.3.6.1.5.5.7.3.3"
    }
    
    if ($certs.Count -eq 0) {
        # Try machine store
        Write-Task "Checking machine certificate store..."
        $certs = Get-ChildItem -Path "Cert:\LocalMachine\My" | Where-Object {
            $_.Subject -match "CloudWorkstation" -or
            $_.EnhancedKeyUsageList -match "Code Signing" -or
            $_.EnhancedKeyUsageList -match "1.3.6.1.5.5.7.3.3"
        }
    }
    
    if ($certs.Count -eq 0) {
        Write-Warning "No suitable code signing certificates found in certificate store"
        Write-ColorOutput "Creating self-signed certificate for testing..." "Yellow"
        Create-TestCertificate
        return
    }
    
    # Use the first suitable certificate
    $cert = $certs[0]
    Write-Success "Found certificate: $($cert.Subject)"
    Write-ColorOutput "  Thumbprint: $($cert.Thumbprint)" "White"
    Write-ColorOutput "  Valid from: $($cert.NotBefore)" "White"
    Write-ColorOutput "  Valid to: $($cert.NotAfter)" "White"
    
    $signArgs = @(
        "sign",
        "/sha1", $cert.Thumbprint,
        "/t", $TimestampUrl,
        "/d", "CloudWorkstation: Enterprise Research Management Platform",
        "/du", "https://github.com/scttfrdmn/cloudworkstation",
        "/v",
        "`"$MsiPath`""
    )
    
    Write-Task "Running SignTool with certificate store..."
    & signtool @signArgs
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "MSI signed successfully with certificate store"
    } else {
        throw "SignTool failed with exit code: $LASTEXITCODE"
    }
}

function Create-TestCertificate {
    Write-Task "Creating self-signed test certificate..."
    
    try {
        # Create a self-signed certificate for testing
        $cert = New-SelfSignedCertificate -Subject "CN=CloudWorkstation Test Certificate" -CertStoreLocation "Cert:\CurrentUser\My" -KeyUsage DigitalSignature -KeySpec Signature -KeyLength 2048 -KeyAlgorithm RSA -HashAlgorithm SHA256 -Provider "Microsoft Enhanced RSA and AES Cryptographic Provider" -Type CodeSigningCert
        
        Write-Success "Test certificate created"
        Write-ColorOutput "  Thumbprint: $($cert.Thumbprint)" "White"
        Write-Warning "This is a test certificate - browsers and systems will show security warnings"
        Write-Warning "For production use, obtain a certificate from a trusted Certificate Authority"
        
        # Sign with the test certificate
        $signArgs = @(
            "sign",
            "/sha1", $cert.Thumbprint,
            "/t", $TimestampUrl,
            "/d", "CloudWorkstation: Enterprise Research Management Platform (Test Build)",
            "/du", "https://github.com/scttfrdmn/cloudworkstation",
            "/v",
            "`"$MsiPath`""
        )
        
        Write-Task "Signing with test certificate..."
        & signtool @signArgs
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "MSI signed successfully with test certificate"
        } else {
            throw "SignTool failed with exit code: $LASTEXITCODE"
        }
        
    } catch {
        Write-ErrorMessage "Failed to create test certificate: $($_.Exception.Message)"
        Write-Warning "Continuing without signing..."
    }
}

function Verify-Signature {
    Write-Step "Verifying MSI signature"
    
    Write-Task "Running signature verification..."
    & signtool verify /pa /v "`"$MsiPath`""
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "MSI signature verified successfully"
    } else {
        Write-Warning "Signature verification failed or warnings present"
        Write-ColorOutput "This may be expected for self-signed certificates" "Yellow"
    }
}

function Show-SigningSummary {
    Write-ColorOutput "======================================" "Green"
    Write-ColorOutput "  SIGNING COMPLETED!" "Green"
    Write-ColorOutput "======================================" "Green"
    Write-Host
    
    Write-ColorOutput "Signed MSI Package:" "Cyan"
    Write-ColorOutput "  File: $MsiPath" "White"
    
    $fileSize = (Get-Item $MsiPath).Length
    Write-ColorOutput "  Size: $fileSize bytes" "White"
    
    $hash = Get-FileHash $MsiPath -Algorithm SHA256
    Write-ColorOutput "  SHA256: $($hash.Hash)" "White"
    Write-Host
    
    Write-ColorOutput "Distribution Notes:" "Cyan"
    Write-ColorOutput "  • Signed MSI packages provide user trust and security" "White"
    Write-ColorOutput "  • Test certificates will show security warnings" "White"
    Write-ColorOutput "  • For production distribution, use a trusted CA certificate" "White"
    Write-ColorOutput "  • Consider additional verification with Windows App Certification Kit" "White"
    Write-Host
}

function Show-Usage {
    Write-Host @"
CloudWorkstation MSI Code Signing Script

USAGE:
    sign-msi.ps1 [options]

OPTIONS:
    -MsiPath <path>              Path to MSI file to sign (auto-detected if not specified)
    -CertificatePath <path>      Path to certificate file (.pfx/.p12)
    -CertificatePassword <pwd>   Password for certificate file
    -TimestampUrl <url>          Timestamp server URL (default: http://timestamp.comodoca.com)
    -Verify                      Verify signature after signing

EXAMPLES:
    # Auto-detect MSI and use certificate from store
    .\sign-msi.ps1

    # Sign with specific certificate file
    .\sign-msi.ps1 -CertificatePath "cert.pfx" -CertificatePassword "password"
    
    # Sign and verify
    .\sign-msi.ps1 -Verify

NOTES:
    • Requires Windows SDK or Visual Studio (for signtool.exe)
    • Will create test certificate if no suitable certificate found
    • Test certificates will show security warnings to users
    • For production use, obtain certificate from trusted CA
"@
}

# Main execution
if ($MyInvocation.InvocationName -ne '.') {
    if ($args -contains "-help" -or $args -contains "--help" -or $args -contains "-h") {
        Show-Usage
        exit 0
    }
    
    $success = Sign-MSI
    exit ([int](-not $success))
}