$ErrorActionPreference = 'Stop'

$packageName = 'cloudworkstation'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

# Remove shortcut
$startMenuPath = [Environment]::GetFolderPath('CommonStartMenu')
$shortcutPath = Join-Path $startMenuPath 'Programs\CloudWorkstation\CloudWorkstation.lnk'

if (Test-Path $shortcutPath) {
  Remove-Item $shortcutPath -Force
}

# Try to remove the shortcut directory if empty
$shortcutDir = Split-Path $shortcutPath
if (Test-Path $shortcutDir) {
  if ((Get-ChildItem $shortcutDir | Measure-Object).Count -eq 0) {
    Remove-Item $shortcutDir -Force
  }
}

# Remove from PATH
Uninstall-BinFile -Name 'cws'

Write-Host "CloudWorkstation has been uninstalled."