# Cortex 部署指南

## 系统要求

- **CPU**: 4核+ (推荐8核)
- **内存**: 8GB+ (推荐16GB)
- **磁盘**: 50GB+ SSD
- **OS**: Linux/macOS/Windows

## 依赖服务

### 1. Ollama (Embedding服务)

```bash
# 安装Ollama
curl -fsSL https://ollama.com/install.sh | sh

# 启动服务
ollama serve

# 下载embedding模型
ollama pull nomic-embed-text
```

### 2. Prometheus (监控，可选)

```bash
# 使用Docker运行
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus
```

### 3. Grafana (可视化，可选)

```bash
docker run -d \
  --name grafana \
  -p 3000:3000 \
  grafana/grafana
```

---

## 配置文件

### config.yaml 示例

```yaml
cortex:
  db_path: "./data/cortex.db"
  log_level: "info"
  auth_enabled: false

index:
  workers: 8
  batch_size: 32

embedding:
  provider: "ollama"
  ollama:
    base_url: "http://localhost:11434"
    model: "nomic-embed-text"

search:
  cache_ttl: 300  # 5分钟
  top_k: 10

prometheus:
  enabled: true
  port: 9090
```

---

## 二进制部署

### 构建

```bash
# Linux/macOS
GOOS=linux GOARCH=amd64 go build -o cortex-linux-amd64 ./cmd/cortex

# Windows
go build -o cortex.exe ./cmd/cortex
```

### 运行

```bash
# 创建数据目录
mkdir -p ./data

# 运行服务
./cortex serve --config config.yaml
```

---

## Docker部署

### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o cortex ./cmd/cortex

FROM alpine:latest

RUN apk add --no-cache ca-certificates sqlite
WORKDIR /app

COPY --from=builder /app/cortex .
COPY --from=builder /app/config.yaml .

EXPOSE 8080 9090

CMD ["./cortex", "serve"]
```

### 构建和运行

```bash
# 构建
docker build -t cortex:latest .

# 运行
docker run -d \
  --name cortex \
  -p 8080:8080 \
  -p 9090:9090 \
  -v ./data:/app/data \
  -v ./config.yaml:/app/config.yaml \
  cortex:latest
```

### Docker Compose

```yaml
version: '3.8'

services:
  cortex:
    build: .
    ports:
      - "8080:8080"
      - "9090:9090"
    volumes:
      - ./data:/app/data
      - ./config.yaml:/app/config.yaml
    depends_on:
      - ollama
    restart: unless-stopped

  ollama:
    image: ollama/ollama
    ports:
      - "11434:11434"
    volumes:
      - ollama_data:/root/.ollama
    restart: unless-stopped

  prometheus:
    image: prom/prometheus
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    restart: unless-stopped

volumes:
  ollama_data:
```

---

## 部署检查清单

### 部署前

- [ ] 确认系统要求满足
- [ ] 安装并启动Ollama
- [ ] 下载nomic-embed-text模型
- [ ] 准备配置文件
- [ ] 创建数据目录

### 部署中

- [ ] 编译或构建Docker镜像
- [ ] 启动Cortex服务
- [ ] 验证健康检查: `curl http://localhost:8080/health`
- [ ] 检查Prometheus指标: `curl http://localhost:9090/metrics`

### 部署后

- [ ] 索引测试数据
- [ ] 测试搜索功能
- [ ] 测试记忆API
- [ ] 验证缓存工作
- [ ] 检查日志无错误

---

## 运维命令

### 查看日志

```bash
# Docker
docker logs -f cortex

# Systemd
journalctl -u cortex -f
```

### 重启服务

```bash
# Docker
docker restart cortex

# Systemd
systemctl restart cortex
```

### 备份数据库

```bash
cp ./data/cortex.db ./data/cortex.db.backup.$(date +%Y%m%d)
```

### 查看统计数据

```bash
go run cmd/cortex/main.go status
```

---

## 扩缩容

### 水平扩展

Cortex本身是无状态的，可以运行多个实例:

```bash
# 实例1
./cortex serve --config config1.yaml

# 实例2
./cortex serve --config config2.yaml
```

使用负载均衡器(如nginx)分发请求:

```nginx
upstream cortex_backend {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
}

server {
    listen 80;
    location / {
        proxy_pass http://cortex_backend;
    }
}
```

### 性能调优

| 参数 | 默认值 | 调优建议 |
|------|--------|----------|
| workers | 8 | CPU核心数 |
| batch_size | 32 | 增加减少内存使用 |
| cache_ttl | 300 | 根据查询重复率调整 |
| top_k | 10 | 根据结果质量调整 |

---

## 监控告警

### Prometheus告警规则

```yaml
groups:
  - name: cortex
    rules:
      - alert: CortexDown
        expr: up{job="cortex"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Cortex服务不可用"

      - alert: HighSearchLatency
        expr: histogram_quantile(0.95, rate(cortex_search_duration_seconds_bucket[5m])) > 0.2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "搜索延迟过高"
```

---

## 故障排查

### 服务启动失败

1. 检查端口占用: `lsof -i :8080`
2. 检查配置文件格式
3. 检查Ollama服务: `curl http://localhost:11434/api/version`

### 搜索无结果

1. 检查是否已索引文档: `curl http://localhost:8080/v1/docs`
2. 检查向量是否生成: `curl http://localhost:9090/metrics | grep vectors_total`
3. 检查embedding日志

### 内存占用过高

1. 检查HNSW索引大小: `curl http://localhost:9090/metrics | grep hnsw_index_size_bytes`
2. 考虑启用PQ压缩
3. 清理过期缓存: 调用 `/admin/cache/cleanup`
