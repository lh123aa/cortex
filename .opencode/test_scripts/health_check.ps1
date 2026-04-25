# Cortex 健康检查测试

$apiUrl = "http://localhost:8080"
$metricsUrl = "http://localhost:9090"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "   Cortex 健康检查测试" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$allPassed = $true

# Test 1: API Health
Write-Host "[测试 1/4] API 健康检查..." -NoNewline
try {
    $response = Invoke-RestMethod -Uri "$apiUrl/health" -Method GET -TimeoutSec 5
    if ($response.status -eq "ok") {
        Write-Host " ✅ 通过" -ForegroundColor Green
    } else {
        Write-Host " ❌ 失败 (状态异常)" -ForegroundColor Red
        $allPassed = $false
    }
} catch {
    Write-Host " ❌ 失败 (连接错误)" -ForegroundColor Red
    $allPassed = $false
}

# Test 2: Ready Check
Write-Host "[测试 2/4] 就绪检查..." -NoNewline
try {
    $response = Invoke-RestMethod -Uri "$apiUrl/health/ready" -Method GET -TimeoutSec 5
    Write-Host " ✅ 通过" -ForegroundColor Green
} catch {
    Write-Host " ❌ 失败" -ForegroundColor Red
    $allPassed = $false
}

# Test 3: Live Check
Write-Host "[测试 3/4] 存活检查..." -NoNewline
try {
    $response = Invoke-RestMethod -Uri "$apiUrl/health/live" -Method GET -TimeoutSec 5
    Write-Host " ✅ 通过" -ForegroundColor Green
} catch {
    Write-Host " ❌ 失败" -ForegroundColor Red
    $allPassed = $false
}

# Test 4: Metrics
Write-Host "[测试 4/4] Prometheus 指标..." -NoNewline
try {
    $response = Invoke-RestMethod -Uri "$metricsUrl/metrics" -Method GET -TimeoutSec 5
    if ($response -match "cortex_") {
        Write-Host " ✅ 通过" -ForegroundColor Green
    } else {
        Write-Host " ❌ 失败 (无指标)" -ForegroundColor Red
        $allPassed = $false
    }
} catch {
    Write-Host " ❌ 失败" -ForegroundColor Red
    $allPassed = $false
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
if ($allPassed) {
    Write-Host "   所有测试通过 ✅" -ForegroundColor Green
} else {
    Write-Host "   部分测试失败 ❌" -ForegroundColor Red
}
Write-Host "========================================" -ForegroundColor Cyan
