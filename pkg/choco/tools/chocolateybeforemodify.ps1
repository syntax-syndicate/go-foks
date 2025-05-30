$ErrorActionPreference = 'Stop'
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
Write-Host "Stopping foks..."
& "$toolsDir\foks.exe" 'ctl' 'stop'
Write-Host "Waiting 5 seconds for shutdown..."
Start-Sleep -seconds 5
