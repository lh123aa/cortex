# ============================================================
# Cortex — 多阶段 Docker 构建
# 目标：~30MB 最终镜像，不含 Go 工具链
# ============================================================

# ---- Stage 1: Build ----
FROM golang:1.25-alpine AS builder

# git 用于取版本号；gcc/musl-dev 仅为兼容可能的 CGO 依赖预装，
# 实际构建使用 CGO_ENABLED=0（保证产出静态二进制）。
RUN apk add --no-cache gcc musl-dev git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# 版本注入必须指向 internal/version（唯一版本源）。
# 旧路径 internal/api.Version 已改为别名，-X 注入会被 init 覆盖回 "dev"。
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w \
      -X github.com/lh123aa/cortex/internal/version.Version=${VERSION} \
      -X github.com/lh123aa/cortex/internal/version.Commit=${COMMIT} \
      -X github.com/lh123aa/cortex/internal/version.Date=${BUILD_DATE}" \
    -o /cortex ./cmd/cortex

# ---- Stage 2: Runtime ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -h /home/cortex cortex
USER cortex

COPY --from=builder /cortex /usr/local/bin/cortex

EXPOSE 8080 9090

ENTRYPOINT ["cortex"]
CMD ["serve"]
