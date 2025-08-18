using System;
using System.IO;
using System.Diagnostics;
using Microsoft.Deployment.WindowsInstaller;

namespace SetupCustomActions
{
    /// <summary>
    /// Handles first-run setup and configuration for CloudWorkstation
    /// </summary>
    public class FirstRunSetup
    {
        private readonly Session _session;
        
        public FirstRunSetup(Session session)
        {
            _session = session;
        }

        /// <summary>
        /// Launches the first-run setup wizard asynchronously
        /// </summary>
        public bool LaunchAsync(string installDir)
        {
            try
            {
                _session.Log("Preparing first-run setup...");

                // Create first-run setup script
                string setupScript = CreateFirstRunScript(installDir);
                
                if (string.IsNullOrEmpty(setupScript))
                {
                    _session.Log("Failed to create first-run setup script");
                    return false;
                }

                // Launch setup script asynchronously
                var startInfo = new ProcessStartInfo
                {
                    FileName = "powershell.exe",
                    Arguments = $"-ExecutionPolicy Bypass -File \"{setupScript}\"",
                    UseShellExecute = true,
                    CreateNoWindow = false,
                    WindowStyle = ProcessWindowStyle.Normal
                };

                var process = Process.Start(startInfo);
                
                if (process != null)
                {
                    _session.Log("First-run setup launched successfully");
                    return true;
                }
                else
                {
                    _session.Log("Failed to launch first-run setup");
                    return false;
                }
            }
            catch (Exception ex)
            {
                _session.Log($"Error launching first-run setup: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Creates the first-run setup PowerShell script
        /// </summary>
        private string CreateFirstRunScript(string installDir)
        {
            try
            {
                string tempPath = Path.GetTempPath();
                string scriptPath = Path.Combine(tempPath, "CloudWorkstation-FirstRun.ps1");

                string scriptContent = GenerateFirstRunScriptContent(installDir);
                
                File.WriteAllText(scriptPath, scriptContent);
                
                _session.Log($"First-run script created: {scriptPath}");
                return scriptPath;
            }
            catch (Exception ex)
            {
                _session.Log($"Error creating first-run script: {ex.Message}");
                return null;
            }
        }

        /// <summary>
        /// Generates the content for the first-run setup script
        /// </summary>
        private string GenerateFirstRunScriptContent(string installDir)
        {
            string binPath = Path.Combine(installDir, "bin");
            string cwsPath = Path.Combine(binPath, "cws.exe");

            return $@"
# CloudWorkstation First-Run Setup
# This script runs after CloudWorkstation installation to complete setup

param([switch]$Silent)

# Configuration
$InstallPath = ""{installDir}""
$BinPath = ""{binPath}""
$CWSPath = ""{cwsPath}""

# Color output functions
function Write-ColorOutput {{
    param([string]$Message, [string]$Color = ""White"")
    $colors = @{{
        ""Red"" = [ConsoleColor]::Red
        ""Green"" = [ConsoleColor]::Green
        ""Yellow"" = [ConsoleColor]::Yellow
        ""Blue"" = [ConsoleColor]::Blue
        ""Cyan"" = [ConsoleColor]::Cyan
        ""White"" = [ConsoleColor]::White
    }}
    Write-Host $Message -ForegroundColor $colors[$Color]
}}

function Write-Welcome {{
    Clear-Host
    Write-ColorOutput ""======================================"" ""Cyan""
    Write-ColorOutput ""  CloudWorkstation First-Run Setup  "" ""Cyan""
    Write-ColorOutput ""======================================"" ""Cyan""
    Write-Host
    Write-ColorOutput ""Welcome to CloudWorkstation!"" ""Green""
    Write-ColorOutput ""Enterprise research management platform for launching cloud environments in seconds."" ""White""
    Write-Host
}}

function Test-Installation {{
    Write-ColorOutput ""Verifying installation..."" ""Blue""
    
    # Check if binaries exist
    if (-not (Test-Path $CWSPath)) {{
        Write-ColorOutput ""Error: CLI binary not found at $CWSPath"" ""Red""
        return $false
    }}
    
    # Check service status
    $service = Get-Service -Name ""CloudWorkstationDaemon"" -ErrorAction SilentlyContinue
    if ($service) {{
        Write-ColorOutput ""✓ CloudWorkstation service installed"" ""Green""
        Write-ColorOutput ""  Status: $($service.Status)"" ""White""
        
        if ($service.Status -ne ""Running"") {{
            Write-ColorOutput ""Starting CloudWorkstation service..."" ""Blue""
            try {{
                Start-Service -Name ""CloudWorkstationDaemon""
                Write-ColorOutput ""✓ Service started successfully"" ""Green""
            }} catch {{
                Write-ColorOutput ""⚠ Failed to start service: $_"" ""Yellow""
            }}
        }}
    }} else {{
        Write-ColorOutput ""⚠ CloudWorkstation service not found"" ""Yellow""
    }}
    
    Write-ColorOutput ""✓ Installation verification completed"" ""Green""
    return $true
}}

function Show-QuickStart {{
    Write-Host
    Write-ColorOutput ""Quick Start Guide:"" ""Cyan""
    Write-Host
    Write-ColorOutput ""1. Command Line Interface (CLI):"" ""Blue""
    Write-ColorOutput ""   cws --help                    # Show help"" ""White""
    Write-ColorOutput ""   cws templates                 # List research templates"" ""White""
    Write-ColorOutput ""   cws launch python-ml my-proj # Launch ML environment"" ""White""
    Write-Host
    Write-ColorOutput ""2. Terminal User Interface (TUI):"" ""Blue""
    Write-ColorOutput ""   cws tui                       # Interactive terminal interface"" ""White""
    Write-Host
    Write-ColorOutput ""3. Graphical User Interface (GUI):"" ""Blue""
    Write-ColorOutput ""   cws-gui                       # Desktop application"" ""White""
    Write-Host
    Write-ColorOutput ""4. Service Management:"" ""Blue""
    Write-ColorOutput ""   Get-Service CloudWorkstationDaemon  # Check service status"" ""White""
    Write-ColorOutput ""   Restart-Service CloudWorkstationDaemon  # Restart service"" ""White""
    Write-Host
    Write-ColorOutput ""Documentation:"" ""Cyan""
    Write-ColorOutput ""  • Getting Started: $InstallPath\docs\GETTING_STARTED.md"" ""White""
    Write-ColorOutput ""  • User Guide: $InstallPath\docs\GUI_USER_GUIDE.md"" ""White""
    Write-ColorOutput ""  • Online: https://github.com/scttfrdmn/cloudworkstation"" ""White""
}}

function Test-CLIConnectivity {{
    Write-ColorOutput ""Testing CLI connectivity..."" ""Blue""
    
    try {{
        $output = & $CWSPath --version 2>&1
        if ($LASTEXITCODE -eq 0) {{
            Write-ColorOutput ""✓ CLI is working: $output"" ""Green""
            return $true
        }} else {{
            Write-ColorOutput ""⚠ CLI test failed with exit code $LASTEXITCODE"" ""Yellow""
            return $false
        }}
    }} catch {{
        Write-ColorOutput ""⚠ CLI test failed: $_"" ""Yellow""
        return $false
    }}
}}

function Show-ConfigurationOptions {{
    Write-Host
    Write-ColorOutput ""Configuration Options:"" ""Cyan""
    Write-Host
    Write-ColorOutput ""AWS Configuration:"" ""Blue""
    Write-ColorOutput ""  CloudWorkstation requires AWS credentials to manage cloud resources."" ""White""
    Write-ColorOutput ""  Configure using one of these methods:"" ""White""
    Write-Host
    Write-ColorOutput ""  1. AWS CLI: aws configure"" ""White""
    Write-ColorOutput ""  2. Environment variables: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY"" ""White""
    Write-ColorOutput ""  3. IAM roles (for EC2 instances)"" ""White""
    Write-ColorOutput ""  4. AWS credential file: ~/.aws/credentials"" ""White""
    Write-Host
    Write-ColorOutput ""Profile Management:"" ""Blue""
    Write-ColorOutput ""  cws profile create my-profile --aws-profile default --region us-west-2"" ""White""
    Write-ColorOutput ""  cws profile switch my-profile"" ""White""
    Write-Host
}}

function Invoke-FirstRunWizard {{
    if (-not $Silent) {{
        Write-Welcome
        
        # Installation verification
        if (-not (Test-Installation)) {{
            Write-ColorOutput ""Installation verification failed. Please reinstall CloudWorkstation."" ""Red""
            Read-Host ""Press Enter to exit""
            return
        }}
        
        # CLI connectivity test
        Test-CLIConnectivity | Out-Null
        
        # Show quick start guide
        Show-QuickStart
        
        # Show configuration options
        Show-ConfigurationOptions
        
        Write-Host
        Write-ColorOutput ""Setup Complete!"" ""Green""
        Write-ColorOutput ""CloudWorkstation is ready to use. Start with 'cws --help' or 'cws tui'."" ""White""
        Write-Host
        
        # Offer to launch GUI
        $response = Read-Host ""Would you like to launch the CloudWorkstation GUI now? (y/N)""
        if ($response -eq ""y"" -or $response -eq ""Y"") {{
            Write-ColorOutput ""Launching CloudWorkstation GUI..."" ""Blue""
            try {{
                $guiPath = Join-Path $BinPath ""cws-gui.exe""
                if (Test-Path $guiPath) {{
                    Start-Process $guiPath
                    Write-ColorOutput ""✓ GUI launched successfully"" ""Green""
                }} else {{
                    Write-ColorOutput ""⚠ GUI not available in this installation"" ""Yellow""
                    Write-ColorOutput ""  Use 'cws tui' for terminal interface"" ""White""
                }}
            }} catch {{
                Write-ColorOutput ""⚠ Failed to launch GUI: $_"" ""Yellow""
            }}
        }}
        
        Write-Host
        Write-ColorOutput ""First-run setup completed successfully!"" ""Green""
        Read-Host ""Press Enter to exit""
    }} else {{
        # Silent mode - just verify installation
        Test-Installation | Out-Null
        Test-CLIConnectivity | Out-Null
    }}
}}

# Main execution
try {{
    Invoke-FirstRunWizard
    
    # Clean up this script
    Remove-Item $PSCommandPath -Force -ErrorAction SilentlyContinue
}} catch {{
    Write-ColorOutput ""Error in first-run setup: $_"" ""Red""
    if (-not $Silent) {{
        Read-Host ""Press Enter to exit""
    }}
}}
";
        }

        /// <summary>
        /// Creates a simple batch file alternative if PowerShell is not available
        /// </summary>
        private string CreateFirstRunBatch(string installDir)
        {
            try
            {
                string tempPath = Path.GetTempPath();
                string batchPath = Path.Combine(tempPath, "CloudWorkstation-FirstRun.bat");
                
                string batchContent = $@"
@echo off
echo ======================================
echo   CloudWorkstation First-Run Setup
echo ======================================
echo.
echo Welcome to CloudWorkstation!
echo.

REM Test CLI
echo Testing CloudWorkstation CLI...
""{Path.Combine(installDir, "bin", "cws.exe")}"" --version
if !errorlevel! equ 0 (
    echo CLI is working correctly
) else (
    echo Warning: CLI test failed
)

echo.
echo Quick Start:
echo   cws --help                    # Show help  
echo   cws templates                 # List research templates
echo   cws launch python-ml my-proj # Launch ML environment
echo   cws tui                       # Interactive terminal interface
echo.
echo Setup complete! CloudWorkstation is ready to use.
echo.
pause

REM Clean up this batch file
del ""%~f0"" 2>nul
";

                File.WriteAllText(batchPath, batchContent);
                _session.Log($"First-run batch file created: {batchPath}");
                return batchPath;
            }
            catch (Exception ex)
            {
                _session.Log($"Error creating first-run batch file: {ex.Message}");
                return null;
            }
        }
    }
}