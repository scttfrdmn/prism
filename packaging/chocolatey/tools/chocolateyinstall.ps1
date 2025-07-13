$ErrorActionPreference = 'Stop'

$packageName = 'cloudworkstation'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

# Test channel configuration - uncomment for test releases
$preRelease = $env:CHOCOLATEY_PRERELEASE -eq 'true'
$version = if ($preRelease) { '0.4.1-beta' } else { '0.4.1' }
$repoPath = if ($preRelease) { 'releases-dev' } else { 'releases' }

$url = "https://github.com/scttfrdmn/cloudworkstation/$repoPath/download/v$version/cws-windows-amd64.zip"
$checksum = 'REPLACE_WITH_ACTUAL_CHECKSUM_AFTER_BUILDING'
$checksumType = 'sha256'

$packageArgs = @{
  packageName   = $packageName
  unzipLocation = $toolsDir
  url           = $url
  checksum      = $checksum
  checksumType  = $checksumType
}

Install-ChocolateyZipPackage @packageArgs

# Create shortcut
$startMenuPath = [Environment]::GetFolderPath('CommonStartMenu')
$shortcutPath = Join-Path $startMenuPath 'Programs\CloudWorkstation\CloudWorkstation.lnk'
$targetPath = Join-Path $toolsDir 'cws-gui.exe'

# Create directory if it doesn't exist
if (!(Test-Path (Split-Path $shortcutPath))) {
  New-Item -ItemType Directory -Path (Split-Path $shortcutPath) | Out-Null
}

# Create shortcut if GUI executable exists
if (Test-Path $targetPath) {
  Install-ChocolateyShortcut -ShortcutFilePath $shortcutPath -TargetPath $targetPath -Description 'CloudWorkstation - Research environments in the cloud'
}

# Add to PATH
$binPath = Join-Path $toolsDir 'cws.exe'
Install-BinFile -Name 'cws' -Path $binPath

Write-Host "CloudWorkstation v$version has been installed."
Write-Host "To get started, open Command Prompt or PowerShell and run:"
Write-Host "cws test"