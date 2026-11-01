#requires -Version 5.1
<#
.SYNOPSIS
  Cortex - One-step installer for Windows

.DESCRIPTION
  Installs Cortex and runs 'cortex install' to auto-configure and index your
  documents — in a single command.

  安装在仓库内执行时默认从源码构建（保持开发体验）；远程执行
  （irm ... | iex）或显式 -Download 时从 GitHub Releases 下载官方产物，
  并做 SHA256 校验。

  产物命名与 .github/workflows/build.yml 严格一致：
    cortex-windows-amd64.zip / cortex-windows-arm64.zip / SHA256SUMS

.PARAMETER DocDir
  要索引的文档目录，默认当前目录。

.PARAMETER BuildOnly
  只构建/下载二进制，不执行 cortex install。

.PARAMETER FromSource
  强制从当前仓库源码构建（需要 Go 1.25+）。

.PARAMETER Download
  强制从 GitHub Releases 下载（即使在仓库内执行）。

.PARAMETER Version
  指定版本 tag，如 "v4.0.0"；默认 latest。

.PARAMETER InstallDir
  安装目录，默认 "$env:LOCALAPPDATA\Cortex\bin"。

.EXAMPLE
  irm -useb https://raw.githubusercontent.com/lh123aa/cortex/main/scripts/install.ps1 | iex

.EXAMPLE
  .\scripts\install.ps1 -DocDir "C:\Users\Me\Documents"

.EXAMPLE
  .\scripts\install.ps1 -Download -Version v4.0.0 -DocDir "D:\我的笔记"
#>
param(
  [string]$DocDir = ".",
  [switch]$BuildOnly,
  [switch]$FromSource,
  [switch]$Download,
  [string]$Version = "latest",
  [string]$InstallDir = ""
)

$ErrorActionPreference = "Stop"
$Repo = "lh123aa/cortex"
$BinaryName = "cortex.exe"

if (-not $InstallDir) {
  $InstallDir = Join-Path $env:LOCALAPPDATA "Cortex\bin"
}

# 远程执行（irm | iex）时 $PSScriptRoot 为空，源码仓库检测随之失效。
$ScriptDir = if ($PSScriptRoot) { $PSScriptRoot } else { (Get-Location).Path }
$RepoRoot = Split-Path -Parent $ScriptDir
$IsRepoCheckout = Test-Path (Join-Path $RepoRoot "go.mod")

Write-Host ""
Write-Host "  ⚡ Installing Cortex..." -ForegroundColor Cyan
Write-Host ""

function Get-Arch {
  $arch = $env:PROCESSOR_ARCHITECTURE
  if ($arch -eq "ARM64" -or $env:PROCESSOR_ARCHITEW6432 -eq "ARM64") { return "arm64" }
  if ($arch -eq "AMD64") { return "amd64" }
  throw "Unsupported architecture: $arch (only amd64/arm64 are supported)"
}

function Save-File([string]$Url, [string]$OutFile) {
  Invoke-WebRequest -UseBasicParsing -Uri $Url -OutFile $OutFile -TimeoutSec 120
}

# ── 方式 A：从源码构建 ─────────────────────────────────────────
function Build-FromSource {
  Write-Host "  [1/2] Building cortex.exe from source..." -ForegroundColor Yellow

  if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
    Write-Host "  ⚠️  Go not found. Install Go 1.25+ from: https://go.dev/dl/" -ForegroundColor Red
    return $null
  }
  if (-not (Test-Path (Join-Path $RepoRoot "go.mod"))) {
    Write-Host "  ⚠️  Not a Cortex source checkout: $RepoRoot" -ForegroundColor Red
    return $null
  }

  $out = Join-Path $InstallDir $BinaryName
  New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

  Push-Location $RepoRoot
  try {
    # 注入版本号，否则源码构建装出来是 "cortex version dev"，
    # 用户会以为装坏了（CI 构建一直有注入，这里曾经漏掉）。
    $verPkg = "github.com/lh123aa/cortex/internal/version"
    $ldflags = "-s -w"
    $hasGit = $false
    if (Get-Command git -ErrorAction SilentlyContinue) { $hasGit = $true }
    if ($hasGit -and (Test-Path (Join-Path $RepoRoot ".git"))) {
      # 优先用 git describe 得到真实版本（version.go 里的值是未注入时的 "dev" 默认值）
      $srcVersion = (git -C $RepoRoot describe --tags --abbrev=0 2>$null)
      if (-not $srcVersion) { $srcVersion = "" }
      $commit = (git -C $RepoRoot rev-parse --short HEAD 2>$null)
      $buildDate = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")
      $ldflags = "-s -w"
      if ($srcVersion) { $ldflags += " -X $verPkg.Version=$srcVersion" }
      if ($commit) { $ldflags += " -X $verPkg.Commit=$commit" }
      $ldflags += " -X $verPkg.Date=$buildDate"
    }
    $build = go build -trimpath -ldflags="$ldflags" -o $out .\cmd\cortex 2>&1
    if ($LASTEXITCODE -ne 0) {
      Write-Host "  ❌ Build failed: $build" -ForegroundColor Red
      return $null
    }
  } finally {
    Pop-Location
  }

  Write-Host "  ✅ Built: $out" -ForegroundColor Green
  return $out
}

# ── 方式 B：从 GitHub Releases 下载 ────────────────────────────
function Install-FromRelease {
  $arch = Get-Arch
  $archiveName = "cortex-windows-$arch.zip"

  if ($Version -eq "latest") {
    $baseUrl = "https://github.com/$Repo/releases/latest/download"
  } else {
    $tag = if ($Version.StartsWith("v")) { $Version } else { "v$Version" }
    $baseUrl = "https://github.com/$Repo/releases/download/$tag"
  }

  $work = Join-Path ([System.IO.Path]::GetTempPath()) ("cortex-install-" + [Guid]::NewGuid().ToString("N"))
  New-Item -ItemType Directory -Force -Path $work | Out-Null

  try {
    $archivePath = Join-Path $work $archiveName
    Write-Host "  [1/2] Downloading $baseUrl/$archiveName" -ForegroundColor Yellow
    Save-File -Url "$baseUrl/$archiveName" -OutFile $archivePath

    # SHA256 校验（拿不到 SHA256SUMS 时仅告警）
    try {
      $sumsPath = Join-Path $work "SHA256SUMS"
      Save-File -Url "$baseUrl/SHA256SUMS" -OutFile $sumsPath
      $line = Get-Content $sumsPath | Where-Object { $_ -match [regex]::Escape($archiveName) } | Select-Object -First 1
      if ($line) {
        $expected = ($line -split '\s+')[0].ToLower()
        $actual = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLower()
        if ($expected -ne $actual) {
          throw "SHA256 mismatch for ${archiveName}: expected $expected, got $actual"
        }
        Write-Host "  ✅ SHA256 verified" -ForegroundColor Green
      }
    } catch {
      if ($_.Exception.Message -like "SHA256 mismatch*") { throw }
      Write-Host "  ⚠️  SHA256SUMS unavailable, skipping checksum verification" -ForegroundColor Yellow
    }

    Expand-Archive -Path $archivePath -DestinationPath $work -Force
    $extracted = Join-Path $work $BinaryName
    if (-not (Test-Path $extracted)) { throw "Archive does not contain $BinaryName" }

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    $out = Join-Path $InstallDir $BinaryName
    Copy-Item -Force $extracted $out

    Write-Host "  ✅ Installed: $out" -ForegroundColor Green
    return $out
  } finally {
    Remove-Item -Recurse -Force $work -ErrorAction SilentlyContinue
  }
}

# ── 安装主流程 ─────────────────────────────────────────────────
$useSource = $FromSource -or (-not $Download -and $IsRepoCheckout)

if ($useSource) {
  $exe = Build-FromSource
  if (-not $exe) {
    Write-Host "  ⚠️  Source build unavailable, falling back to GitHub Releases..." -ForegroundColor Yellow
    $exe = Install-FromRelease
  }
} else {
  try {
    $exe = Install-FromRelease
  } catch {
    Write-Host "  ⚠️  Download failed: $($_.Exception.Message)" -ForegroundColor Yellow
    Write-Host "  ⚠️  Falling back to source build..." -ForegroundColor Yellow
    $exe = Build-FromSource
    if (-not $exe) { throw }
  }
}

if (-not $exe) {
  Write-Host "  ❌ Installation failed" -ForegroundColor Red
  exit 1
}

# 加入用户 PATH（幂等）
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if (-not $userPath) { $userPath = "" }
if (-not ($userPath.Split(";") | Where-Object { $_ -and ($_.TrimEnd("\") -ieq $InstallDir.TrimEnd("\")) })) {
  [Environment]::SetEnvironmentVariable("PATH", ($userPath.TrimEnd(";") + ";" + $InstallDir), "User")
  Write-Host "  ✅ Added $InstallDir to user PATH (restart terminal to take effect)" -ForegroundColor Green
}

# 当前会话也立即可用
if (-not ($env:PATH.Split(";") | Where-Object { $_ -and ($_.TrimEnd("\") -ieq $InstallDir.TrimEnd("\")) })) {
  $env:PATH = "$env:PATH;$InstallDir"
}

& $exe --version

if ($BuildOnly) {
  Write-Host ""
  Write-Host "  ✅ Build complete! Run manually:" -ForegroundColor Cyan
  Write-Host "      $exe install <your-docs-dir>"
  Write-Host ""
  return
}

# ── 运行 cortex install（自动配置 + 索引） ─────────────────────
Write-Host ""
Write-Host "  [2/2] Running cortex install..." -ForegroundColor Yellow
Write-Host ""

$resolvedDir = if ($DocDir -eq ".") { (Get-Location).Path } else { $DocDir }
# cortex install 会把 zap 日志写到 stderr。PowerShell 把原生命令的 stderr
# 当成错误流，即使命令成功也会让脚本以 exit 1 结束（用户看到「安装失败」）。
# 用 Start-Process 显式重定向两个流，才能真正拿到退出码。
$logOut = Join-Path ([System.IO.Path]::GetTempPath()) ("cortex-install-" + [Guid]::NewGuid().ToString("N") + ".log")
$proc = Start-Process -FilePath $exe -ArgumentList @("install", "`"$resolvedDir`"") `
  -NoNewWindow -Wait -PassThru -RedirectStandardOutput $logOut `
  -RedirectStandardError "$logOut.err"
if (Test-Path $logOut) {
  Get-Content $logOut | Where-Object { $_ -notmatch '"level":"(debug|info)"' } | ForEach-Object { Write-Host "  $_" }
}
if ($proc.ExitCode -ne 0) {
  Write-Host "  ⚠️  cortex install exited with code $($proc.ExitCode)" -ForegroundColor Yellow
  if (Test-Path "$logOut.err") {
    Get-Content "$logOut.err" | Where-Object { $_ -notmatch '"level":"(debug|info)"' } |
      Select-Object -First 5 | ForEach-Object { Write-Host "     $_" -ForegroundColor DarkYellow }
  }
  Write-Host "     (binary is installed; run 'cortex install' manually to inspect)" -ForegroundColor DarkYellow
}
Remove-Item $logOut, "$logOut.err" -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "  🎉  Cortex is ready!" -ForegroundColor Cyan
Write-Host ""
