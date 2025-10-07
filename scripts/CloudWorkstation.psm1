# CloudWorkstation PowerShell Module
# Provides PowerShell integration and cmdlets for CloudWorkstation

# Module information
$ModuleVersion = "0.5.1"
$ModuleDescription = "CloudWorkstation PowerShell integration module for enterprise research management"

# Import required assemblies
Add-Type -AssemblyName System.Net.Http
Add-Type -AssemblyName System.Web

#region Configuration and Paths

# Get CloudWorkstation installation paths
function Get-CloudWorkstationPaths {
    [CmdletBinding()]
    param()
    
    $paths = @{}
    
    # Try to get installation path from registry
    try {
        $regKey = Get-ItemProperty -Path "HKLM:\SOFTWARE\CloudWorkstation" -ErrorAction SilentlyContinue
        if ($regKey) {
            $paths.InstallPath = $regKey.InstallPath
            $paths.BinPath = $regKey.BinPath
            $paths.ConfigPath = $regKey.ConfigPath
        }
    } catch {
        Write-Verbose "Could not read CloudWorkstation registry keys"
    }
    
    # Fallback to default paths
    if (-not $paths.InstallPath) {
        $paths.InstallPath = "${env:ProgramFiles}\CloudWorkstation"
        $paths.BinPath = "${env:ProgramFiles}\CloudWorkstation\bin"
        $paths.ConfigPath = "${env:ProgramData}\CloudWorkstation"
    }
    
    # Add executable paths
    $paths.CWS = Join-Path $paths.BinPath "cws.exe"
    $paths.CWSD = Join-Path $paths.BinPath "cwsd.exe"
    $paths.CWSGUI = Join-Path $paths.BinPath "cws-gui.exe"
    $paths.CWSService = Join-Path $paths.BinPath "cwsd-service.exe"
    
    return $paths
}

# Global paths variable
$CloudWorkstationPaths = Get-CloudWorkstationPaths

#endregion

#region Core Functions

<#
.SYNOPSIS
Gets CloudWorkstation version information.

.DESCRIPTION
Retrieves version information from the CloudWorkstation CLI binary.

.EXAMPLE
Get-CloudWorkstationVersion
Returns version information for CloudWorkstation.
#>
function Get-CloudWorkstationVersion {
    [CmdletBinding()]
    param()
    
    try {
        if (Test-Path $CloudWorkstationPaths.CWS) {
            $output = & $CloudWorkstationPaths.CWS --version 2>&1
            if ($LASTEXITCODE -eq 0) {
                return $output
            } else {
                throw "CloudWorkstation CLI returned error code $LASTEXITCODE"
            }
        } else {
            throw "CloudWorkstation CLI not found at $($CloudWorkstationPaths.CWS)"
        }
    } catch {
        Write-Error "Failed to get CloudWorkstation version: $_"
    }
}

<#
.SYNOPSIS
Gets the status of CloudWorkstation daemon service.

.DESCRIPTION
Retrieves the current status of the CloudWorkstation Windows service.

.EXAMPLE
Get-CloudWorkstationServiceStatus
Returns the status of CloudWorkstation daemon service.
#>
function Get-CloudWorkstationServiceStatus {
    [CmdletBinding()]
    param()
    
    try {
        $service = Get-Service -Name "CloudWorkstationDaemon" -ErrorAction Stop
        
        $status = [PSCustomObject]@{
            Name = $service.Name
            DisplayName = $service.DisplayName  
            Status = $service.Status
            StartType = $service.StartType
            CanPauseAndContinue = $service.CanPauseAndContinue
            CanShutdown = $service.CanShutdown
            CanStop = $service.CanStop
            ServiceType = $service.ServiceType
        }
        
        # Add process information if service is running
        if ($service.Status -eq "Running") {
            try {
                $wmiService = Get-WmiObject -Class Win32_Service -Filter "Name='CloudWorkstationDaemon'"
                if ($wmiService.ProcessId -ne 0) {
                    $status | Add-Member -NotePropertyName ProcessId -NotePropertyValue $wmiService.ProcessId
                    
                    $process = Get-Process -Id $wmiService.ProcessId -ErrorAction SilentlyContinue
                    if ($process) {
                        $status | Add-Member -NotePropertyName CPU -NotePropertyValue $process.CPU
                        $status | Add-Member -NotePropertyName WorkingSet -NotePropertyValue $process.WorkingSet64
                        $status | Add-Member -NotePropertyName StartTime -NotePropertyValue $process.StartTime
                    }
                }
            } catch {
                Write-Verbose "Could not retrieve process information: $_"
            }
        }
        
        return $status
    } catch {
        Write-Error "Failed to get CloudWorkstation service status: $_"
    }
}

<#
.SYNOPSIS
Starts the CloudWorkstation daemon service.

.DESCRIPTION
Starts the CloudWorkstation Windows service and waits for it to reach running state.

.PARAMETER Timeout
Timeout in seconds to wait for service to start (default: 30).

.EXAMPLE
Start-CloudWorkstationService
Starts the CloudWorkstation service.

.EXAMPLE  
Start-CloudWorkstationService -Timeout 60
Starts the CloudWorkstation service with 60 second timeout.
#>
function Start-CloudWorkstationService {
    [CmdletBinding(SupportsShouldProcess)]
    param(
        [int]$Timeout = 30
    )
    
    try {
        $service = Get-Service -Name "CloudWorkstationDaemon" -ErrorAction Stop
        
        if ($service.Status -eq "Running") {
            Write-Output "CloudWorkstation service is already running"
            return
        }
        
        if ($PSCmdlet.ShouldProcess("CloudWorkstationDaemon", "Start Service")) {
            Write-Output "Starting CloudWorkstation service..."
            Start-Service -Name "CloudWorkstationDaemon"
            
            # Wait for service to start
            $service.WaitForStatus("Running", [TimeSpan]::FromSeconds($Timeout))
            
            Write-Output "CloudWorkstation service started successfully"
        }
    } catch {
        Write-Error "Failed to start CloudWorkstation service: $_"
    }
}

<#
.SYNOPSIS
Stops the CloudWorkstation daemon service.

.DESCRIPTION
Stops the CloudWorkstation Windows service and waits for it to reach stopped state.

.PARAMETER Timeout
Timeout in seconds to wait for service to stop (default: 30).

.PARAMETER Force
Force stop the service even if it has dependent services.

.EXAMPLE
Stop-CloudWorkstationService
Stops the CloudWorkstation service.

.EXAMPLE
Stop-CloudWorkstationService -Force -Timeout 60
Force stops the CloudWorkstation service with 60 second timeout.
#>
function Stop-CloudWorkstationService {
    [CmdletBinding(SupportsShouldProcess)]
    param(
        [int]$Timeout = 30,
        [switch]$Force
    )
    
    try {
        $service = Get-Service -Name "CloudWorkstationDaemon" -ErrorAction Stop
        
        if ($service.Status -eq "Stopped") {
            Write-Output "CloudWorkstation service is already stopped"
            return
        }
        
        if ($PSCmdlet.ShouldProcess("CloudWorkstationDaemon", "Stop Service")) {
            Write-Output "Stopping CloudWorkstation service..."
            
            if ($Force) {
                Stop-Service -Name "CloudWorkstationDaemon" -Force
            } else {
                Stop-Service -Name "CloudWorkstationDaemon"
            }
            
            # Wait for service to stop
            $service.WaitForStatus("Stopped", [TimeSpan]::FromSeconds($Timeout))
            
            Write-Output "CloudWorkstation service stopped successfully"
        }
    } catch {
        Write-Error "Failed to stop CloudWorkstation service: $_"
    }
}

<#
.SYNOPSIS
Restarts the CloudWorkstation daemon service.

.DESCRIPTION
Stops and starts the CloudWorkstation Windows service.

.PARAMETER Timeout
Timeout in seconds to wait for service operations (default: 30).

.EXAMPLE
Restart-CloudWorkstationService
Restarts the CloudWorkstation service.
#>
function Restart-CloudWorkstationService {
    [CmdletBinding(SupportsShouldProcess)]
    param(
        [int]$Timeout = 30
    )
    
    if ($PSCmdlet.ShouldProcess("CloudWorkstationDaemon", "Restart Service")) {
        Stop-CloudWorkstationService -Timeout $Timeout
        Start-Sleep -Seconds 2
        Start-CloudWorkstationService -Timeout $Timeout
    }
}

#endregion

#region Instance Management

<#
.SYNOPSIS
Gets CloudWorkstation instances.

.DESCRIPTION
Retrieves list of CloudWorkstation instances using the CLI.

.EXAMPLE
Get-CloudWorkstationInstances
Returns all CloudWorkstation instances.
#>
function Get-CloudWorkstationInstances {
    [CmdletBinding()]
    param()
    
    try {
        if (Test-Path $CloudWorkstationPaths.CWS) {
            $output = & $CloudWorkstationPaths.CWS list --format json 2>&1
            if ($LASTEXITCODE -eq 0) {
                return $output | ConvertFrom-Json
            } else {
                throw "CloudWorkstation CLI returned error code $LASTEXITCODE"
            }
        } else {
            throw "CloudWorkstation CLI not found"
        }
    } catch {
        Write-Error "Failed to get CloudWorkstation instances: $_"
    }
}

<#
.SYNOPSIS
Gets CloudWorkstation templates.

.DESCRIPTION
Retrieves list of available CloudWorkstation templates.

.EXAMPLE
Get-CloudWorkstationTemplates
Returns all available templates.
#>
function Get-CloudWorkstationTemplates {
    [CmdletBinding()]
    param()
    
    try {
        if (Test-Path $CloudWorkstationPaths.CWS) {
            $output = & $CloudWorkstationPaths.CWS templates --format json 2>&1
            if ($LASTEXITCODE -eq 0) {
                return $output | ConvertFrom-Json
            } else {
                throw "CloudWorkstation CLI returned error code $LASTEXITCODE"
            }
        } else {
            throw "CloudWorkstation CLI not found"
        }
    } catch {
        Write-Error "Failed to get CloudWorkstation templates: $_"
    }
}

<#
.SYNOPSIS
Launches a new CloudWorkstation instance.

.DESCRIPTION
Launches a new CloudWorkstation instance using the specified template.

.PARAMETER TemplateName
Name of the template to use for the instance.

.PARAMETER InstanceName
Name for the new instance.

.PARAMETER Size
Size of the instance (S, M, L, XL).

.PARAMETER Spot
Use spot instances for cost savings.

.EXAMPLE
New-CloudWorkstationInstance -TemplateName "python-ml" -InstanceName "my-ml-project"
Launches a Python ML instance named "my-ml-project".

.EXAMPLE
New-CloudWorkstationInstance -TemplateName "r-research" -InstanceName "analysis" -Size L -Spot
Launches a large R research instance with spot pricing.
#>
function New-CloudWorkstationInstance {
    [CmdletBinding(SupportsShouldProcess)]
    param(
        [Parameter(Mandatory)]
        [string]$TemplateName,
        
        [Parameter(Mandatory)]
        [string]$InstanceName,
        
        [ValidateSet("S", "M", "L", "XL")]
        [string]$Size = "M",
        
        [switch]$Spot
    )
    
    try {
        if (-not (Test-Path $CloudWorkstationPaths.CWS)) {
            throw "CloudWorkstation CLI not found"
        }
        
        $launchArgs = @("launch", $TemplateName, $InstanceName, "--size", $Size)
        
        if ($Spot) {
            $launchArgs += "--spot"
        }
        
        if ($PSCmdlet.ShouldProcess("$TemplateName -> $InstanceName", "Launch Instance")) {
            Write-Output "Launching CloudWorkstation instance..."
            Write-Output "Template: $TemplateName"
            Write-Output "Instance: $InstanceName"
            Write-Output "Size: $Size"
            if ($Spot) { Write-Output "Spot: Enabled" }
            
            $output = & $CloudWorkstationPaths.CWS @launchArgs 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Output "Instance launched successfully"
                return $output
            } else {
                throw "CloudWorkstation CLI returned error code $LASTEXITCODE`n$output"
            }
        }
    } catch {
        Write-Error "Failed to launch CloudWorkstation instance: $_"
    }
}

<#
.SYNOPSIS
Removes a CloudWorkstation instance.

.DESCRIPTION
Terminates and removes a CloudWorkstation instance.

.PARAMETER InstanceName
Name of the instance to remove.

.PARAMETER Force
Skip confirmation prompt.

.EXAMPLE
Remove-CloudWorkstationInstance -InstanceName "my-instance"
Removes the specified instance with confirmation.

.EXAMPLE
Remove-CloudWorkstationInstance -InstanceName "my-instance" -Force
Removes the specified instance without confirmation.
#>
function Remove-CloudWorkstationInstance {
    [CmdletBinding(SupportsShouldProcess, ConfirmImpact = "High")]
    param(
        [Parameter(Mandatory)]
        [string]$InstanceName,
        
        [switch]$Force
    )
    
    try {
        if (-not (Test-Path $CloudWorkstationPaths.CWS)) {
            throw "CloudWorkstation CLI not found"
        }
        
        if ($Force -or $PSCmdlet.ShouldProcess($InstanceName, "Terminate Instance")) {
            Write-Output "Terminating CloudWorkstation instance: $InstanceName"
            
            $output = & $CloudWorkstationPaths.CWS terminate $InstanceName 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Output "Instance terminated successfully"
                return $output
            } else {
                throw "CloudWorkstation CLI returned error code $LASTEXITCODE`n$output"
            }
        }
    } catch {
        Write-Error "Failed to terminate CloudWorkstation instance: $_"
    }
}

#endregion

#region GUI Integration

<#
.SYNOPSIS
Launches the CloudWorkstation GUI.

.DESCRIPTION
Starts the CloudWorkstation graphical user interface.

.EXAMPLE
Start-CloudWorkstationGUI
Launches the CloudWorkstation GUI application.
#>
function Start-CloudWorkstationGUI {
    [CmdletBinding()]
    param()
    
    try {
        if (Test-Path $CloudWorkstationPaths.CWSGUI) {
            Write-Output "Launching CloudWorkstation GUI..."
            Start-Process $CloudWorkstationPaths.CWSGUI
        } else {
            Write-Warning "CloudWorkstation GUI not found at $($CloudWorkstationPaths.CWSGUI)"
            Write-Output "GUI may not be available in this installation"
            Write-Output "Use 'cws tui' for terminal interface or 'cws --help' for CLI"
        }
    } catch {
        Write-Error "Failed to launch CloudWorkstation GUI: $_"
    }
}

<#
.SYNOPSIS
Launches the CloudWorkstation TUI.

.DESCRIPTION
Starts the CloudWorkstation terminal user interface.

.EXAMPLE
Start-CloudWorkstationTUI
Launches the CloudWorkstation TUI application.
#>
function Start-CloudWorkstationTUI {
    [CmdletBinding()]
    param()
    
    try {
        if (Test-Path $CloudWorkstationPaths.CWS) {
            Write-Output "Launching CloudWorkstation TUI..."
            & $CloudWorkstationPaths.CWS tui
        } else {
            throw "CloudWorkstation CLI not found"
        }
    } catch {
        Write-Error "Failed to launch CloudWorkstation TUI: $_"
    }
}

#endregion

#region Utility Functions

<#
.SYNOPSIS
Tests CloudWorkstation installation.

.DESCRIPTION
Performs comprehensive tests of CloudWorkstation installation and configuration.

.EXAMPLE
Test-CloudWorkstationInstallation
Tests the CloudWorkstation installation.
#>
function Test-CloudWorkstationInstallation {
    [CmdletBinding()]
    param()
    
    $results = @{
        InstallationPath = $false
        CLIExecutable = $false
        DaemonExecutable = $false
        ServiceInstalled = $false
        ServiceRunning = $false
        DaemonConnectivity = $false
        Overall = $false
    }
    
    try {
        Write-Output "Testing CloudWorkstation installation..."
        Write-Output ""
        
        # Test installation path
        if (Test-Path $CloudWorkstationPaths.InstallPath) {
            Write-Output "✓ Installation path exists: $($CloudWorkstationPaths.InstallPath)"
            $results.InstallationPath = $true
        } else {
            Write-Output "✗ Installation path not found: $($CloudWorkstationPaths.InstallPath)"
        }
        
        # Test CLI executable
        if (Test-Path $CloudWorkstationPaths.CWS) {
            Write-Output "✓ CLI executable exists: $($CloudWorkstationPaths.CWS)"
            $results.CLIExecutable = $true
            
            try {
                $version = & $CloudWorkstationPaths.CWS --version 2>&1
                if ($LASTEXITCODE -eq 0) {
                    Write-Output "  Version: $version"
                } else {
                    Write-Output "  Warning: Version check failed"
                }
            } catch {
                Write-Output "  Warning: Could not get version"
            }
        } else {
            Write-Output "✗ CLI executable not found: $($CloudWorkstationPaths.CWS)"
        }
        
        # Test daemon executable
        if (Test-Path $CloudWorkstationPaths.CWSD) {
            Write-Output "✓ Daemon executable exists: $($CloudWorkstationPaths.CWSD)"
            $results.DaemonExecutable = $true
        } else {
            Write-Output "✗ Daemon executable not found: $($CloudWorkstationPaths.CWSD)"
        }
        
        # Test Windows service
        try {
            $service = Get-Service -Name "CloudWorkstationDaemon" -ErrorAction Stop
            Write-Output "✓ Windows service installed: $($service.DisplayName)"
            $results.ServiceInstalled = $true
            
            if ($service.Status -eq "Running") {
                Write-Output "✓ Service is running"
                $results.ServiceRunning = $true
            } else {
                Write-Output "✗ Service is not running (Status: $($service.Status))"
            }
        } catch {
            Write-Output "✗ Windows service not installed"
        }
        
        # Test daemon connectivity
        try {
            $client = New-Object System.Net.WebClient
            $client.Timeout = 5000
            $response = $client.DownloadString("http://localhost:8947/api/v1/health")
            Write-Output "✓ Daemon connectivity successful"
            $results.DaemonConnectivity = $true
        } catch {
            Write-Output "✗ Daemon connectivity failed"
            Write-Output "  Daemon may not be running or accessible"
        }
        
        Write-Output ""
        
        # Overall result
        $results.Overall = $results.InstallationPath -and $results.CLIExecutable -and 
                          $results.ServiceInstalled -and $results.ServiceRunning -and
                          $results.DaemonConnectivity
        
        if ($results.Overall) {
            Write-Output "✓ CloudWorkstation installation test PASSED"
        } else {
            Write-Output "✗ CloudWorkstation installation test FAILED"
            Write-Output ""
            Write-Output "Troubleshooting:"
            if (-not $results.ServiceRunning) {
                Write-Output "  Try: Restart-CloudWorkstationService"
            }
            if (-not $results.DaemonConnectivity) {
                Write-Output "  Check Windows Firewall settings for port 8947"
                Write-Output "  Check service logs in Event Viewer"
            }
        }
        
        return $results
    } catch {
        Write-Error "Failed to test CloudWorkstation installation: $_"
    }
}

<#
.SYNOPSIS
Opens CloudWorkstation documentation.

.DESCRIPTION
Opens CloudWorkstation documentation in the default browser or file viewer.

.PARAMETER Document
Specific document to open (GettingStarted, UserGuide, TUIGuide, Troubleshooting).

.EXAMPLE
Open-CloudWorkstationDocumentation
Opens the getting started documentation.

.EXAMPLE
Open-CloudWorkstationDocumentation -Document UserGuide
Opens the user guide documentation.
#>
function Open-CloudWorkstationDocumentation {
    [CmdletBinding()]
    param(
        [ValidateSet("GettingStarted", "UserGuide", "TUIGuide", "Troubleshooting")]
        [string]$Document = "GettingStarted"
    )
    
    try {
        $docsPath = Join-Path $CloudWorkstationPaths.InstallPath "docs"
        
        $docFiles = @{
            "GettingStarted" = "GETTING_STARTED.md"
            "UserGuide" = "GUI_USER_GUIDE.md"
            "TUIGuide" = "TUI_USER_GUIDE.md"
            "Troubleshooting" = "TROUBLESHOOTING.md"
        }
        
        $docFile = Join-Path $docsPath $docFiles[$Document]
        
        if (Test-Path $docFile) {
            Write-Output "Opening $Document documentation..."
            Start-Process $docFile
        } else {
            Write-Warning "Documentation file not found: $docFile"
            Write-Output "Online documentation: https://github.com/scttfrdmn/cloudworkstation"
        }
    } catch {
        Write-Error "Failed to open CloudWorkstation documentation: $_"
    }
}

#endregion

#region Module Initialization

# Initialize module
Write-Verbose "CloudWorkstation PowerShell Module v$ModuleVersion loaded"
Write-Verbose "Installation path: $($CloudWorkstationPaths.InstallPath)"

# Check if CloudWorkstation is installed
if (-not (Test-Path $CloudWorkstationPaths.CWS)) {
    Write-Warning "CloudWorkstation CLI not found. Please ensure CloudWorkstation is properly installed."
}

#endregion

#region Aliases

# Create convenient aliases
New-Alias -Name "cws" -Value $CloudWorkstationPaths.CWS -ErrorAction SilentlyContinue
New-Alias -Name "cwsd" -Value $CloudWorkstationPaths.CWSD -ErrorAction SilentlyContinue
New-Alias -Name "cws-gui" -Value $CloudWorkstationPaths.CWSGUI -ErrorAction SilentlyContinue

#endregion

# Export module members
Export-ModuleMember -Function @(
    'Get-CloudWorkstationVersion',
    'Get-CloudWorkstationServiceStatus',
    'Start-CloudWorkstationService',
    'Stop-CloudWorkstationService', 
    'Restart-CloudWorkstationService',
    'Get-CloudWorkstationInstances',
    'Get-CloudWorkstationTemplates',
    'New-CloudWorkstationInstance',
    'Remove-CloudWorkstationInstance',
    'Start-CloudWorkstationGUI',
    'Start-CloudWorkstationTUI',
    'Test-CloudWorkstationInstallation',
    'Open-CloudWorkstationDocumentation'
) -Alias @('cws', 'cwsd', 'cws-gui') -Variable @('CloudWorkstationPaths')