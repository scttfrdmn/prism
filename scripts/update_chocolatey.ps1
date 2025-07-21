# Update Chocolatey package for CloudWorkstation
# Usage: .\scripts\update_chocolatey.ps1 -Version "0.4.2" -ReleaseDir ".\dist\v0.4.2"

param(
    [Parameter(Mandatory=$true)]
    [string]$Version,
    
    [Parameter(Mandatory=$true)]
    [string]$ReleaseDir
)

# Ensure version is properly formatted
$VersionNum = $Version.TrimStart('v')
$VersionTag = "v$VersionNum"

# Check if release directory exists
if (-Not (Test-Path $ReleaseDir)) {
    Write-Error "Release directory $ReleaseDir does not exist"
    exit 1
}

# Define the archive file
$WindowsArchive = "cloudworkstation-windows-amd64.zip"
$ArchivePath = Join-Path $ReleaseDir $WindowsArchive

# Check if archive file exists
if (-Not (Test-Path $ArchivePath)) {
    Write-Error "Archive file $WindowsArchive not found in $ReleaseDir"
    exit 1
}

# Calculate SHA256 checksum
$Checksum = Get-FileHash -Path $ArchivePath -Algorithm SHA256 | Select-Object -ExpandProperty Hash

# Update the nuspec file
$NuspecPath = "scripts\chocolatey\cloudworkstation.nuspec"
$NuspecContent = Get-Content $NuspecPath -Raw
$NuspecContent = $NuspecContent -replace '<version>.*?</version>', "<version>$VersionNum</version>"
$NuspecContent = $NuspecContent -replace 'releases/tag/v[0-9.]+</releaseNotes>', "releases/tag/$VersionTag</releaseNotes>"
$NuspecContent | Set-Content $NuspecPath

# Update the install script
$InstallScriptPath = "scripts\chocolatey\tools\chocolateyinstall.ps1"
$InstallScript = Get-Content $InstallScriptPath -Raw
$InstallScript = $InstallScript -replace "v[0-9.]+/cloudworkstation-windows-amd64.zip", "$VersionTag/cloudworkstation-windows-amd64.zip"
$InstallScript = $InstallScript -replace '\$checksum\s*=\s*''[a-fA-F0-9]+''|\$checksum\s*=\s*''PLACEHOLDER_SHA256_CHECKSUM''.*', "`$checksum   = '$Checksum' # Updated for $VersionTag"
$InstallScript | Set-Content $InstallScriptPath

Write-Host "Updated Chocolatey package for $VersionTag with checksum: $Checksum"
Write-Host ""
Write-Host "To test the package locally, run:"
Write-Host "choco pack .\scripts\chocolatey\cloudworkstation.nuspec"
Write-Host "choco install cloudworkstation -s ."