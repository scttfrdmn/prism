$ErrorActionPreference = 'Stop'

$packageName= 'cloudworkstation'
$toolsDir   = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

# Remove shims
Uninstall-BinFile -Name "cws"
Uninstall-BinFile -Name "cwsd"

# Note: This doesn't remove the config directory (~/.cloudworkstation) to preserve user data
Write-Host "CloudWorkstation has been uninstalled."
Write-Host "Your configuration data at $env:USERPROFILE\.cloudworkstation has been preserved."