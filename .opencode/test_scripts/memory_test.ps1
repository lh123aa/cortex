# Cortex 记忆系统测试

$apiUrl = "http://localhost:8080"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "   Cortex 记忆系统测试" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$allPassed = $true

# Test 1: Write Memory
Write-Host "[测试 1/5] 写入记忆..." -NoNewline
try {
    $body = @{
        content = "测试记忆 - $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
        tags = @("测试", "自动化")
        source = "test"
    } | ConvertTo-Json

    $response = Invoke-RestMethod -Uri "$apiUrl/v1/memory" -Method POST `
        -Body $body `
        -ContentType "application/json" `
        -TimeoutSec 10

    if ($response.id) {
        $script:memoryId = $response.id
        Write-Host " ✅ 通过 (ID: $($response.id.Substring(0,8))...)" -ForegroundColor Green
    } else {
        Write-Host " ❌ 失败 (无ID)" -ForegroundColor Red
        $allPassed = $false
    }
} catch {
    Write-Host " ❌ 失败 ($($_.Exception.Message))" -ForegroundColor Red
    $allPassed = $false
}

# Test 2: Search Memory
Write-Host "[测试 2/5] 搜索记忆..." -NoNewline
try {
    Start-Sleep -Milliseconds 500
    $response = Invoke-RestMethod -Uri "$apiUrl/v1/memory/search?q=测试记忆" -Method GET -TimeoutSec 10
    if ($response.total -gt 0) {
        Write-Host " ✅ 通过 (找到 $($response.total) 条)" -ForegroundColor Green
    } else {
        Write-Host " ⚠️ 警告 (无结果)" -ForegroundColor Yellow
    }
} catch {
    Write-Host " ❌ 失败" -ForegroundColor Red
    $allPassed = $false
}

# Test 3: Get Memory Context
Write-Host "[测试 3/5] 获取记忆上下文..." -NoNewline
try {
    $response = Invoke-RestMethod -Uri "$apiUrl/v1/memory/context?q=测试" -Method GET -TimeoutSec 10
    if ($response.context) {
        Write-Host " ✅ 通过 (Token: $($response.token_count))" -ForegroundColor Green
    } else {
        Write-Host " ❌ 失败 (无上下文)" -ForegroundColor Red
        $allPassed = $false
    }
} catch {
    Write-Host " ❌ 失败" -ForegroundColor Red
    $allPassed = $false
}

# Test 4: Batch Write
Write-Host "[测试 4/5] 批量写入..." -NoNewline
try {
    $body = @{
        memories = @(
            @{content = "批量记忆1"; tags = @("批量")},
            @{content = "批量记忆2"; tags = @("批量")}
        )
    } | ConvertTo-Json

    $response = Invoke-RestMethod -Uri "$apiUrl/v1/memory/batch" -Method POST `
        -Body $body `
        -ContentType "application/json" `
        -TimeoutSec 10

    if ($response.success -eq 2) {
        Write-Host " ✅ 通过" -ForegroundColor Green
    } else {
        Write-Host " ❌ 失败 (success=$($response.success))" -ForegroundColor Red
        $allPassed = $false
    }
} catch {
    Write-Host " ❌ 失败" -ForegroundColor Red
    $allPassed = $false
}

# Test 5: Delete Memory
Write-Host "[测试 5/5] 删除记忆..." -NoNewline
try {
    if ($memoryId) {
        $response = Invoke-RestMethod -Uri "$apiUrl/v1/memory/$memoryId" -Method DELETE -TimeoutSec 10
        Write-Host " ✅ 通过" -ForegroundColor Green
    } else {
        Write-Host " ⚠️ 跳过 (无记忆ID)" -ForegroundColor Yellow
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
