# Build script for Hrunner release artifacts and NSIS installer
$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot
Set-Location $RootDir

Write-Host "==> Compiling Hrunner binaries (pure Go, -ldflags='-s -w')..." -ForegroundColor Cyan

go build -ldflags="-s -w" -o "$RootDir\hrunner.exe" .\cmd\hrunner
Write-Host "  ✓ Built hrunner.exe ($( (Get-Item "$RootDir\hrunner.exe").Length ) bytes)" -ForegroundColor Green

go build -ldflags="-s -w" -o "$RootDir\hlauncher.exe" .\cmd\hlauncher
Write-Host "  ✓ Built hlauncher.exe ($( (Get-Item "$RootDir\hlauncher.exe").Length ) bytes)" -ForegroundColor Green

go build -ldflags="-s -w" -o "$RootDir\hbuild.exe" .\cmd\hbuild
Write-Host "  ✓ Built hbuild.exe ($( (Get-Item "$RootDir\hbuild.exe").Length ) bytes)" -ForegroundColor Green

# Ensure dist directory
$DistDir = Join-Path $RootDir "dist"
if (!(Test-Path $DistDir)) {
    New-Item -ItemType Directory -Path $DistDir | Out-Null
}

# Check for NSIS compiler
$makensis = Get-Command makensis.exe -ErrorAction SilentlyContinue
if ($null -eq $makensis) {
    # Check standard NSIS installation path
    $nsisDefault = "${env:ProgramFiles(x86)}\NSIS\makensis.exe"
    if (Test-Path $nsisDefault) {
        $makensis = $nsisDefault
    }
}

if ($makensis) {
    Write-Host "==> Compiling NSIS Installer: dist\HrunnerSetup.exe..." -ForegroundColor Cyan
    & $makensis "$RootDir\packaging\nsis\installer.nsi"
    if (Test-Path "$DistDir\HrunnerSetup.exe") {
        Write-Host "  ✓ Generated HrunnerSetup.exe ($( (Get-Item "$DistDir\HrunnerSetup.exe").Length ) bytes)" -ForegroundColor Green
    }
} else {
    Write-Host "NOTE: NSIS (makensis.exe) not found on PATH. To build HrunnerSetup.exe, install NSIS from https://nsis.sourceforge.io" -ForegroundColor Yellow
}
