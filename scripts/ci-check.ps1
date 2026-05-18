# Cortex CI Check Script
# 模拟 GitHub CI quality job 环境
# 推送前必跑，确保代码质量
#
# 用法: .\scripts\ci-check.ps1

$ErrorActionPreference = "Stop"
$passed = $true

function Write-Step {
    param([string]$Title)
    Write-Host ""
    Write-Host "=== $Title ===" -ForegroundColor Cyan
}

function Write-Pass {
    Write-Host "✅ PASS" -ForegroundColor Green
}

function Write-Fail {
    Write-Host "❌ FAIL" -ForegroundColor Red
    $global:passed = $false
}

Write-Host "╔════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║     Cortex Pre-Push CI Check           ║" -ForegroundColor Cyan
Write-Host "╚════════════════════════════════════════╝" -ForegroundColor Cyan

# Step 1: go vet
Write-Step "go vet"
go vet ./...
if ($LASTEXITCODE -eq 0) {
    Write-Pass
} else {
    Write-Fail
}

# Step 2: CGO_ENABLED=1 tests
Write-Step "go test (CGO_ENABLED=1)"
$env:CGO_ENABLED = "1"
go test -count=1 -timeout=300s ./...
if ($LASTEXITCODE -eq 0) {
    Write-Pass
} else {
    Write-Fail
}

# Step 3: CGO_ENABLED=0 tests
Write-Step "go test (CGO_ENABLED=0)"
$env:CGO_ENABLED = "0"
go test -count=1 -timeout=300s ./...
if ($LASTEXITCODE -eq 0) {
    Write-Pass
} else {
    Write-Fail
}

# Summary
Write-Host ""
if ($passed) {
    Write-Host "✅ All CI checks passed!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "❌ CI checks FAILED! Fix the issues above before pushing." -ForegroundColor Red
    exit 1
}
