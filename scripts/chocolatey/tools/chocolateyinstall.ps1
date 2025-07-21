$ErrorActionPreference = 'Stop'

$packageName= 'cloudworkstation'
$toolsDir   = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$url        = 'https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.2/cloudworkstation-windows-amd64.zip'
$checksum   = 'PLACEHOLDER_SHA256_CHECKSUM' # Will be updated during release process
$checksumType = 'sha256'

$packageArgs = @{
  packageName   = $packageName
  unzipLocation = $toolsDir
  url           = $url
  checksum      = $checksum
  checksumType  = $checksumType
}

Install-ChocolateyZipPackage @packageArgs

# Create shims for the executables
$cws_path = Join-Path $toolsDir "cws.exe"
$cwsd_path = Join-Path $toolsDir "cwsd.exe"

Install-BinFile -Name "cws" -Path $cws_path
Install-BinFile -Name "cwsd" -Path $cwsd_path

# Create config directory if it doesn't exist
$configDir = Join-Path $env:USERPROFILE ".cloudworkstation"
if (!(Test-Path $configDir)) {
    New-Item -ItemType Directory -Path $configDir | Out-Null
    Write-Host "Created CloudWorkstation configuration directory at $configDir"
}

Write-Host "CloudWorkstation has been installed!"
Write-Host "Start the daemon with: cwsd start"
Write-Host "Start the GUI with: cws gui"
Write-Host "Launch your first workstation with: cws launch python-research my-project"