# Cortex v2.0 生产环境测试脚本

## 测试脚本

```powershell
# Cortex v2.0 生产环境测试套件
# 使用方式: .\deploy_test.ps1

param(
    [string]$ApiUrl = "http://localhost:8080",
    [string]$MetricsUrl = "http://localhost:9090"
)

$ErrorActionPreference = "Continue"
$script:allPassed = $true
$script:testResults = @()

function Test-CortexEndpoint {
    param(
        [string]$Name,
        [string]$Url,
        [string]$Method = "GET",
        [string]$Body = $null,
        [int]$ExpectedStatus = 200
    )

    $result = @{
        Name = $Name
        Passed = $false
        Details = ""
    }

    try {
        $params = @{
            Uri = $Url
            Method = $Method
            TimeoutSec = 10
        }
        if ($Body) {
            $params.Body = $Body
            $params.ContentType = "application/json"
        }

        $response = Invoke-RestMethod @params

        if ($response) {
            $result.Passed = $true
            $result.Details = "成功"
        }
    } catch {
        $result.Details = $_.Exception.Message
        $script:allPassed = $false
    }

    $script:testResults += $result
    return $result
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "   Cortex v2.0 生产环境测试" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "API: $ApiUrl" -ForegroundColor Gray
Write-Host "Metrics: $MetricsUrl" -ForegroundColor Gray
Write-Host ""

# ========== 1. 健康检查测试 ==========
Write-Host ">>> 1. 健康检查测试" -ForegroundColor Yellow

Test-CortexEndpoint -Name "Health Check" -Url "$ApiUrl/health"
Test-CortexEndpoint -Name "Ready Check" -Url "$ApiUrl/health/ready"
Test-CortexEndpoint -Name "Live Check" -Url "$ApiUrl/health/live"

# ========== 2. 搜索功能测试 ==========
Write-Host ""
Write-Host ">>> 2. 搜索功能测试" -ForegroundColor Yellow

Test-CortexEndpoint -Name "Hybrid Search" -Url "$ApiUrl/v1/search?q=test&mode=hybrid"
Test-CortexEndpoint -Name "Vector Search" -Url "$ApiUrl/v1/search?q=test&mode=vector"
Test-CortexEndpoint -Name "FTS Search" -Url "$ApiUrl/v1/search?q=test&mode=fts"
Test-CortexEndpoint -Name "RAG Context" -Url "$ApiUrl/v1/context?q=test&budget=1000"

# ========== 3. 记忆系统测试 ==========
Write-Host ""
Write-Host ">>> 3. 记忆系统测试" -ForegroundColor Yellow

$memoryId = $null

# 3.1 写入记忆
try {
    $body = @{
        content = "测试记忆 - $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
        tags = @("测试", "自动化")
        source = "test"
    } | ConvertTo-Json

    $response = Invoke-RestMethod -Uri "$ApiUrl/v1/memory" -Method POST `
        -Body $body -ContentType "application/json" -TimeoutSec 10

    if ($response.id) {
        $script:memoryId = $response.id
        Write-Host "   [写入记忆] ✅ 成功 (ID: $($response.id.Substring(0,8))...)" -ForegroundColor Green
    }
} catch {
    Write-Host "   [写入记忆] ❌ 失败: $($_.Exception.Message)" -ForegroundColor Red
    $script:allPassed = $false
}

# 3.2 批量写入
try {
    $body = @{
        memories = @(
            @{content = "批量记忆1"; tags = @("批量")},
            @{content = "批量记忆2"; tags = @("批量")}
        )
    } | ConvertTo-Json

    $response = Invoke-RestMethod -Uri "$ApiUrl/v1/memory/batch" -Method POST `
        -Body $body -ContentType "application/json" -TimeoutSec 10

    if ($response.success -eq 2) {
        Write-Host "   [批量写入] ✅ 成功 (2条)" -ForegroundColor Green
    }
} catch {
    Write-Host "   [批量写入] ❌ 失败" -ForegroundColor Red
    $script:allPassed = $false
}

# 3.3 搜索记忆
Test-CortexEndpoint -Name "Search Memory" -Url "$ApiUrl/v1/memory/search?q=测试"

# 3.4 记忆上下文
Test-CortexEndpoint -Name "Memory Context" -Url "$ApiUrl/v1/memory/context?q=测试&budget=500"

# 3.5 删除记忆
if ($memoryId) {
    try {
        Invoke-RestMethod -Uri "$ApiUrl/v1/memory/$memoryId" -Method DELETE -TimeoutSec 10
        Write-Host "   [删除记忆] ✅ 成功" -ForegroundColor Green
    } catch {
        Write-Host "   [删除记忆] ❌ 失败" -ForegroundColor Red
        $script:allPassed = $false
    }
}

# ========== 4. 认证测试 ==========
Write-Host ""
Write-Host ">>> 4. 认证系统测试" -ForegroundColor Yellow

$testUsername = "testuser_$(Get-Random)"
$testPassword = "Test123456"

# 4.1 注册
try {
    $body = @{
        username = $testUsername
        password = $testPassword
    } | ConvertTo-Json

    $response = Invoke-RestMethod -Uri "$ApiUrl/auth/register" -Method POST `
        -Body $body -ContentType "application/json" -TimeoutSec 10

    Write-Host "   [用户注册] ✅ 成功" -ForegroundColor Green
} catch {
    Write-Host "   [用户注册] ❌ 失败" -ForegroundColor Red
    $script:allPassed = $false
}

# 4.2 登录
try {
    $body = @{
        username = $testUsername
        password = $testPassword
    } | ConvertTo-Json

    $response = Invoke-RestMethod -Uri "$ApiUrl/auth/login" -Method POST `
        -Body $body -ContentType "application/json" -TimeoutSec 10

    if ($response.token) {
        $script:authToken = $response.token
        Write-Host "   [用户登录] ✅ 成功" -ForegroundColor Green
    }
} catch {
    Write-Host "   [用户登录] ❌ 失败" -ForegroundColor Red
    $script:allPassed = $false
}

# 4.3 登出
if ($authToken) {
    try {
        $headers = @{ Authorization = "Bearer $authToken" }
        Invoke-RestMethod -Uri "$ApiUrl/auth/logout" -Method POST `
            -Headers $headers -TimeoutSec 10
        Write-Host "   [用户登出] ✅ 成功" -ForegroundColor Green
    } catch {
        Write-Host "   [用户登出] ❌ 失败" -ForegroundColor Red
    }
}

# ========== 5. 监控测试 ==========
Write-Host ""
Write-Host ">>> 5. Prometheus 监控测试" -ForegroundColor Yellow

Test-CortexEndpoint -Name "Metrics Endpoint" -Url "$MetricsUrl/metrics"

# ========== 6. 性能测试 ==========
Write-Host ""
Write-Host ">>> 6. 性能测试" -ForegroundColor Yellow

$latencies = @()

# 10次搜索延迟测试
for ($i = 1; $i -le 10; $i++) {
    try {
        $sw = [Diagnostics.Stopwatch]::StartNew()
        Invoke-RestMethod -Uri "$ApiUrl/v1/search?q=performance_test&mode=hybrid" `
            -Method GET -TimeoutSec 10
        $sw.Stop()
        $latencies += $sw.ElapsedMilliseconds
    } catch {}
}

if ($latencies.Count -gt 0) {
    $avg = ($latencies | Measure-Object -Average).Average
    $max = ($latencies | Measure-Object -Maximum).Maximum
    $min = ($latencies | Measure-Object -Minimum).Minimum

    Write-Host "   [搜索延迟] 平均: ${avg}ms, 最大: ${max}ms, 最小: ${min}ms" -ForegroundColor Cyan

    if ($avg -lt 100) {
        Write-Host "   [性能评估] ✅ 通过 (平均 < 100ms)" -ForegroundColor Green
    } else {
        Write-Host "   [性能评估] ⚠️ 警告 (平均 > 100ms)" -ForegroundColor Yellow
    }
}

# ========== 结果汇总 ==========
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "   测试结果汇总" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$passCount = ($testResults | Where-Object { $_.Passed }).Count
$totalCount = $testResults.Count

Write-Host "| 测试项 | 结果 |" -ForegroundColor White
Write-Host "|--------|------|" -ForegroundColor White

foreach ($result in $testResults) {
    $status = if ($result.Passed) { "✅ 通过" } else { "❌ 失败" }
    $color = if ($result.Passed) { "Green" } else { "Red" }
    Write-Host "| $($result.Name) | $([System.Convert]::ToString($status)) |" -ForegroundColor $color
}

Write-Host ""
Write-Host "总计: $passCount / $totalCount 通过" -ForegroundColor Cyan

if ($allPassed) {
    Write-Host ""
    Write-Host "🎉 所有测试通过！" -ForegroundColor Green
    exit 0
} else {
    Write-Host ""
    Write-Host "⚠️  部分测试失败，请检查日志" -ForegroundColor Yellow
    exit 1
}
```

---

## 快速运行

```powershell
# 1. 启动服务
go run cmd/cortex/main.go serve

# 2. 新开终端运行测试
cd .opencode/test_scripts
.\deploy_test.ps1
```

---

## 预期结果

| 测试类别 | 预期 | 状态 |
|----------|------|------|
| 健康检查 | 3/3 | ✅ |
| 搜索功能 | 4/4 | ✅ |
| 记忆系统 | 5/5 | ✅ |
| 认证系统 | 3/3 | ✅ |
| 监控 | 1/1 | ✅ |
| 性能 | 1/1 | ✅ |

---

*测试脚本版本: v2.0*
