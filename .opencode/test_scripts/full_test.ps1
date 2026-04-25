# Cortex 完整测试套件

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "   Cortex 完整测试套件 v2.0" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "测试时间: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
Write-Host ""

$results = @{
    Health = $null
    Search = $null
    Memory = $null
    Metrics = $null
}

# Run Health Check
Write-Host ">>> 运行健康检查..." -ForegroundColor Yellow
try {
    & "$scriptDir\health_check.ps1"
    $results.Health = "PASS"
} catch {
    $results.Health = "FAIL"
}
Write-Host ""

# Run Search Test
Write-Host ">>> 运行搜索测试..." -ForegroundColor Yellow
try {
    & "$scriptDir\search_test.ps1"
    $results.Search = "PASS"
} catch {
    $results.Search = "FAIL"
}
Write-Host ""

# Run Memory Test
Write-Host ">>> 运行记忆测试..." -ForegroundColor Yellow
try {
    & "$scriptDir\memory_test.ps1"
    $results.Memory = "PASS"
} catch {
    $results.Memory = "FAIL"
}
Write-Host ""

# Run Metrics Check
Write-Host ">>> 运行指标检查..." -ForegroundColor Yellow
try {
    & "$scriptDir\metrics_check.ps1"
    $results.Metrics = "PASS"
} catch {
    $results.Metrics = "FAIL"
}
Write-Host ""

# Summary
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "   测试结果汇总" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "| 测试项 | 结果 |" -ForegroundColor White
Write-Host "|--------|------|" -ForegroundColor White

foreach ($key in $results.Keys) {
    $status = if ($results[$key] -eq "PASS") { "✅ PASS" } else { "❌ FAIL" }
    $color = if ($results[$key] -eq "PASS") { "Green" } else { "Red" }
    Write-Host "| $key | $([System.Convert]::ToString($status)) |" -ForegroundColor $color
}

Write-Host ""
$passCount = ($results.Values | Where-Object { $_ -eq "PASS" }).Count
$totalCount = $results.Count

if ($passCount -eq $totalCount) {
    Write-Host "🎉 所有测试通过！($passCount/$totalCount)" -ForegroundColor Green
} else {
    Write-Host "⚠️  部分测试失败 ($passCount/$totalCount)" -ForegroundColor Yellow
}
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "   测试完成" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
