# Cortex 搜索功能测试

$apiUrl = "http://localhost:8080"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "   Cortex 搜索功能测试" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$allPassed = $true

# Test 1: Hybrid Search
Write-Host "[测试 1/4] 混合搜索..." -NoNewline
try {
    $response = Invoke-RestMethod -Uri "$apiUrl/v1/search?q=test&top_k=5&mode=hybrid" -Method GET -TimeoutSec 10
    if ($response.total -ge 0) {
        Write-Host " ✅ 通过 (结果: $($response.total))" -ForegroundColor Green
    } else {
        Write-Host " ❌ 失败" -ForegroundColor Red
        $allPassed = $false
    }
} catch {
    Write-Host " ❌ 失败" -ForegroundColor Red
    $allPassed = $false
}

# Test 2: Vector Search
Write-Host "[测试 2/4] 向量搜索..." -NoNewline
try {
    $response = Invoke-RestMethod -Uri "$apiUrl/v1/search?q=test&top_k=5&mode=vector" -Method GET -TimeoutSec 10
    if ($response.total -ge 0) {
        Write-Host " ✅ 通过" -ForegroundColor Green
    } else {
        Write-Host " ❌ 失败" -ForegroundColor Red
        $allPassed = $false
    }
} catch {
    Write-Host " ❌ 失败" -ForegroundColor Red
    $allPassed = $false
}

# Test 3: FTS Search
Write-Host "[测试 3/4] 全文搜索..." -NoNewline
try {
    $response = Invoke-RestMethod -Uri "$apiUrl/v1/search?q=test&top_k=5&mode=fts" -Method GET -TimeoutSec 10
    if ($response.total -ge 0) {
        Write-Host " ✅ 通过" -ForegroundColor Green
    } else {
        Write-Host " ❌ 失败" -ForegroundColor Red
        $allPassed = $false
    }
} catch {
    Write-Host " ❌ 失败" -ForegroundColor Red
    $allPassed = $false
}

# Test 4: RAG Context
Write-Host "[测试 4/4] RAG 上下文..." -NoNewline
try {
    $response = Invoke-RestMethod -Uri "$apiUrl/v1/context?q=test&budget=1000" -Method GET -TimeoutSec 15
    if ($response.context) {
        Write-Host " ✅ 通过 (Token: $($response.token_count))" -ForegroundColor Green
    } else {
        Write-Host " ❌ 失败" -ForegroundColor Red
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
