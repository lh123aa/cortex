# ============================================================
# Cortex — 多阶段 Docker 构建
# 目标：~30MB 最终镜像，不含 Go 工具链
# ============================================================

# ---- Stage 1: Build ----
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X github.com/lh123aa/cortex/internal/api.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o /cortex ./cmd/cortex

# ---- Stage 2: Runtime ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -h /home/cortex cortex
USER cortex

COPY --from=builder /cortex /usr/local/bin/cortex

EXPOSE 8080 9090

ENTRYPOINT ["cortex"]
CMD ["serve"]
