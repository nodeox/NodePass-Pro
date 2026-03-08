# ============================================================================
# Stage 1: 构建后端
# ============================================================================
FROM golang:1.24-bookworm AS go-builder

# 接收构建参数
ARG BUILD_VERSION=0.3.0

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w -X main.appVersion=${BUILD_VERSION} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -trimpath \
    -o /usr/local/bin/license-center \
    ./cmd/server

# ============================================================================
# Stage 2: 最终镜像
# ============================================================================
FROM debian:bookworm-slim

# 安装运行时依赖
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    curl && \
    rm -rf /var/lib/apt/lists/*

# 设置时区
ENV TZ=Asia/Shanghai

WORKDIR /app

# 复制后端二进制文件
COPY --from=go-builder /usr/local/bin/license-center /usr/local/bin/license-center

# 复制配置文件模板
COPY configs/config.yaml /app/configs/config.yaml

# 创建非 root 用户
RUN useradd -m -u 1000 -s /bin/bash appuser && \
    chown -R appuser:appuser /app

USER appuser

EXPOSE 8090

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8090/health || exit 1

CMD ["/usr/local/bin/license-center", "--config", "/app/configs/config.yaml"]
