#requires -Version 5.1
<#
.SYNOPSIS
  Cortex - One-step installer for Windows
.DESCRIPTION
  Downloads/builds Cortex and runs 'cortex install' to auto-configure
  and index your documents — in a single command.

  Usage:
    .\install.ps1                     # build + install (uses current dir)
    .\install.ps1 -DocDir "D:\我的笔记"  # build + install with custom path
    .\install.ps1 -BuildOnly           # only build the binary

.EXAMPLE
  .\install.ps1 -DocDir "C:\Users\Me\Documents"
#>

param(
  [string]$DocDir = ".",
  [switch]$BuildOnly
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$BinaryPath = Join-Path $RepoRoot "bin\cortex.exe"

Write-Host ""
Write-Host "  ⚡ Installing Cortex..." -ForegroundColor Cyan
Write-Host ""

# ── Step 1: Build ──────────────────────────────────────────────
Write-Host "  [1/2] Building cortex.exe..." -ForegroundColor Yellow

if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
  Write-Host "  ⚠️  Go not found. Install Go 1.21+ from: https://go.dev/dl/" -ForegroundColor Red
  $choice = Read-Host "  Install Go via winget? (y/n)"
  if ($choice -eq "y") {
    winget install GoLang.Go
    refreshenv
  } else {
    exit 1
  }
}

Push-Location $RepoRoot
try {
  $build = go build -ldflags="-s -w" -o $BinaryPath .\cmd\cortex 2>&1
  if ($LASTEXITCODE -ne 0) {
    Write-Host "  ❌ Build failed: $build" -ForegroundColor Red
    exit 1
  }
} finally {
  Pop-Location
}

Write-Host "  ✅ Built: $BinaryPath" -ForegroundColor Green

if ($BuildOnly) {
  Write-Host ""
  Write-Host "  ✅ Build complete! Run manually:" -ForegroundColor Cyan
  Write-Host "      $BinaryPath install <your-docs-dir>"
  Write-Host ""
  return
}

# ── Step 2: Install (auto-config + index) ──────────────────────
Write-Host ""
Write-Host "  [2/2] Running cortex install..." -ForegroundColor Yellow
Write-Host ""

$resolvedDir = if ($DocDir -eq ".") { (Get-Location).Path } else { $DocDir }
& $BinaryPath install $resolvedDir

Write-Host ""
Write-Host "  💡  Add to PATH for convenience:" -ForegroundColor Cyan
Write-Host "       [Environment]::SetEnvironmentVariable('PATH',"
Write-Host "         [Environment]::GetEnvironmentVariable('PATH','User') + ';$BinaryPath.Replace('cortex.exe','').TrimEnd('\')',"
Write-Host "         'User')"
Write-Host ""
