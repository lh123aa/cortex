# Cortex Prometheus 指标检查

$metricsUrl = "http://localhost:9090"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "   Cortex 指标检查" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Get metrics
try {
    $metrics = Invoke-RestMethod -Uri "$metricsUrl/metrics" -Method GET -TimeoutSec 10
    $metricsText = $metrics -join "`n"

    $checks = @(
        @{ name = "搜索总数"; pattern = "cortex_search_total"; desc = "cortex_search_total" },
        @{ name = "搜索延迟"; pattern = "cortex_search_duration_seconds"; desc = "Histogram" },
        @{ name = "缓存命中"; pattern = "cortex_search_cache_hits_total"; desc = "cortex_search_cache_hits_total" },
        @{ name = "缓存未命中"; pattern = "cortex_search_cache_misses_total"; desc = "cortex_search_cache_misses_total" },
        @{ name = "向量总数"; pattern = "cortex_vectors_total"; desc = "cortex_vectors_total" },
        @{ name = "文档总数"; pattern = "cortex_documents_total"; desc = "cortex_documents_total" },
        @{ name = "HNSW索引大小"; pattern = "cortex_hnsw_index_size_bytes"; desc = "Gauge" },
        @{ name = "Embedding延迟"; pattern = "cortex_embedding_duration_seconds"; desc = "Histogram" }
    )

    foreach ($check in $checks) {
        Write-Host "[$($check.name)] " -NoNewline
        if ($metricsText -match $check.pattern) {
            Write-Host " ✅ 存在" -ForegroundColor Green
        } else {
            Write-Host " ⚠️ 未找到" -ForegroundColor Yellow
        }
    }

    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "   指标检查完成" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan

} catch {
    Write-Host "❌ 无法连接指标服务器: $($_.Exception.Message)" -ForegroundColor Red
}
