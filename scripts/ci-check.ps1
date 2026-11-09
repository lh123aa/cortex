# Cortex CI Check Script
# 模拟 GitHub CI quality job 环境；pre-push 钩子调用它
#
# 用法:
#   .\scripts\ci-check.ps1          # 默认：go vet（快，秒级~1 分钟）
#   .\scripts\ci-check.ps1 -Full    # 完整：vet + CGO=1/0 两轮全量测试（慢，10 分钟+）
#
# 设计取舍（2026-11-09 修）：
#   1. 本机 go 不在 PATH 时不再硬失败 —— 只警告并跳过本地检查，
#      因为远端 GitHub Actions（.github/workflows/build.yml 的 quality job）
#      会跑同样的 vet + 全量测试，本地缺 Go 不应让 owner 无法推送。
#   2. 自动探测 Go（PATH / C:\Go\go\bin / GOROOT）并为仓库设置私有
#      GOPATH/GOCACHE，避免 "module cache not found" 这类环境性失败。
#   3. 默认只跑 vet：全量测试在远端已强制，本地重复跑只会让推送等十几分钟。

param(
    [switch]$Full
)

$ErrorActionPreference = "Continue"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot
$script:passed = $true

function Write-Step { param([string]$Title) Write-Host ""; Write-Host "=== $Title ===" -ForegroundColor Cyan }
function Write-Pass { Write-Host "PASS" -ForegroundColor Green }
function Write-Fail { Write-Host "FAIL" -ForegroundColor Red; $script:passed = $false }
function Write-Warn { param([string]$Msg) Write-Host "WARN  $Msg" -ForegroundColor Yellow }

Write-Host "Cortex Pre-Push CI Check" -ForegroundColor Cyan
Write-Host ""

# ── 定位 go ──────────────────────────────────────────────
function Find-Go {
    $cands = @()
    $c = Get-Command go -ErrorAction SilentlyContinue
    if ($c) { $cands += $c.Source }
    if ($env:GOROOT) { $cands += (Join-Path $env:GOROOT "bin\go.exe") }
    $cands += "C:\Go\go\bin\go.exe"
    $cands += "C:\Go\bin\go.exe"
    if ($env:ProgramFiles) { $cands += (Join-Path $env:ProgramFiles "Go\bin\go.exe") }
    if ($env:LOCALAPPDATA) { $cands += (Join-Path $env:LOCALAPPDATA "Programs\Go\bin\go.exe") }
    foreach ($p in $cands) {
        if ($p -and (Test-Path $p)) { return $p }
    }
    return $null
}

$GoExe = Find-Go
if (-not $GoExe) {
    Write-Warn "本机未找到 Go 可执行文件，跳过本地检查。"
    Write-Warn "远端 GitHub Actions 的 quality job 仍会跑 vet + 全量测试（见 .github/workflows/build.yml）。"
    Write-Host ""
    Write-Host "CI checks SKIPPED (no local Go)" -ForegroundColor Yellow
    exit 0
}

Write-Host "go: $GoExe" -ForegroundColor DarkGray
$env:PATH = (Split-Path -Parent $GoExe) + ";" + $env:PATH

# ── 仓库私有 Go 缓存（避免全局环境污染与 module cache not found） ──
if (-not $env:GOPATH) { $env:GOPATH = Join-Path $ProjectRoot ".gopath" }
if (-not $env:GOMODCACHE) { $env:GOMODCACHE = Join-Path $env:GOPATH "pkg\mod" }
if (-not $env:GOCACHE) { $env:GOCACHE = Join-Path $ProjectRoot ".gobuild\cache" }
if (-not $env:GOFLAGS) { $env:GOFLAGS = "-mod=mod" }
New-Item -ItemType Directory -Force -Path $env:GOCACHE | Out-Null

Write-Step "go version"
& $GoExe version
Write-Host ""

Write-Step "go vet"
& $GoExe vet ./...
if ($LASTEXITCODE -eq 0) { Write-Pass } else { Write-Fail }

if ($Full) {
    Write-Step "go test (CGO_ENABLED=1)"
    $env:CGO_ENABLED = "1"
    & $GoExe test -count=1 -timeout=300s ./...
    if ($LASTEXITCODE -eq 0) { Write-Pass } else { Write-Fail }

    Write-Step "go test (CGO_ENABLED=0)"
    $env:CGO_ENABLED = "0"
    & $GoExe test -count=1 -timeout=300s ./...
    if ($LASTEXITCODE -eq 0) { Write-Pass } else { Write-Fail }
} else {
    Write-Host ""
    Write-Host "（未跑全量测试：远端 quality job 会跑。需要本地跑用 -Full）" -ForegroundColor DarkGray
}

Write-Host ""
if ($script:passed) {
    Write-Host "All local CI checks passed!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "CI checks FAILED! Fix the issues above before pushing." -ForegroundColor Red
    exit 1
}
