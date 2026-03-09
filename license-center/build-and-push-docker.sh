#!/bin/bash

# Docker 镜像构建和推送脚本
# 用法: ./build-and-push-docker.sh [registry/image-name] [tag]

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# 默认配置
DEFAULT_REGISTRY="docker.io"
DEFAULT_IMAGE_NAME="nodepass/license-center"
DEFAULT_TAG="latest"

# 从参数获取配置
IMAGE_NAME=${1:-$DEFAULT_IMAGE_NAME}
TAG=${2:-$DEFAULT_TAG}

# 完整镜像名称
FULL_IMAGE_NAME="${IMAGE_NAME}:${TAG}"

# 获取版本信息
log_step "获取版本信息..."
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

log_info "版本信息:"
echo "  镜像名称: ${FULL_IMAGE_NAME}"
echo "  版本:     ${VERSION}"
echo "  提交:     ${GIT_COMMIT}"
echo "  分支:     ${GIT_BRANCH}"
echo "  构建时间: ${BUILD_TIME}"
echo ""

# 确认构建
read -p "是否继续构建并推送镜像? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_warn "构建已取消"
    exit 0
fi

# 检查 Docker
log_step "检查 Docker..."
if ! command -v docker &> /dev/null; then
    log_error "Docker 未安装"
    exit 1
fi

if ! docker info &> /dev/null; then
    log_error "Docker 守护进程未运行"
    exit 1
fi

log_info "Docker 检查通过"

# 构建镜像
log_step "构建 Docker 镜像..."
docker build \
    --file Dockerfile.version \
    --build-arg VERSION="${VERSION}" \
    --build-arg GIT_COMMIT="${GIT_COMMIT}" \
    --build-arg GIT_BRANCH="${GIT_BRANCH}" \
    --build-arg BUILD_TIME="${BUILD_TIME}" \
    --tag "${FULL_IMAGE_NAME}" \
    --tag "${IMAGE_NAME}:${VERSION}" \
    .

if [ $? -eq 0 ]; then
    log_info "镜像构建成功"
else
    log_error "镜像构建失败"
    exit 1
fi

# 显示镜像信息
log_step "镜像信息:"
docker images | grep "${IMAGE_NAME}" | head -5

# 测试镜像
log_step "测试镜像..."
log_info "启动测试容器..."

# 停止并删除旧的测试容器
docker rm -f license-center-test 2>/dev/null || true

# 启动测试容器
docker run -d \
    --name license-center-test \
    -p 18090:8090 \
    -e JWT_SECRET="test-secret-key-for-docker-build" \
    -e ADMIN_PASSWORD="TestPassword123!" \
    "${FULL_IMAGE_NAME}"

# 等待容器启动
log_info "等待容器启动..."
sleep 10

# 健康检查
log_info "执行健康检查..."
if curl -f http://localhost:18090/health > /dev/null 2>&1; then
    log_info "健康检查通过 ✓"

    # 显示版本信息
    log_info "版本信息:"
    curl -s http://localhost:18090/ | python3 -m json.tool 2>/dev/null || curl -s http://localhost:18090/
else
    log_error "健康检查失败 ✗"
    log_info "查看容器日志:"
    docker logs license-center-test
    docker rm -f license-center-test
    exit 1
fi

# 停止测试容器
log_info "停止测试容器..."
docker rm -f license-center-test

# 推送镜像
log_step "推送镜像到仓库..."
read -p "是否推送镜像到仓库? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    log_info "推送镜像: ${FULL_IMAGE_NAME}"
    docker push "${FULL_IMAGE_NAME}"

    log_info "推送镜像: ${IMAGE_NAME}:${VERSION}"
    docker push "${IMAGE_NAME}:${VERSION}"

    log_info "镜像推送成功 ✓"
else
    log_warn "跳过推送镜像"
fi

# 生成部署命令
log_step "部署命令:"
echo ""
echo "# 拉取镜像"
echo "docker pull ${FULL_IMAGE_NAME}"
echo ""
echo "# 运行容器"
echo "docker run -d \\"
echo "  --name license-center \\"
echo "  -p 8090:8090 \\"
echo "  -v \$(pwd)/data:/app/data \\"
echo "  -v \$(pwd)/configs:/app/configs \\"
echo "  -e JWT_SECRET=\"your-secret-key\" \\"
echo "  -e ADMIN_PASSWORD=\"your-password\" \\"
echo "  --restart unless-stopped \\"
echo "  ${FULL_IMAGE_NAME}"
echo ""

# 生成 docker-compose.yml
log_step "生成 docker-compose.yml..."
cat > docker-compose.deploy.yml <<EOF
version: '3.8'

services:
  license-center:
    image: ${FULL_IMAGE_NAME}
    container_name: license-center
    ports:
      - "8090:8090"
    volumes:
      - ./data:/app/data
      - ./configs:/app/configs
    environment:
      - JWT_SECRET=\${JWT_SECRET}
      - ADMIN_PASSWORD=\${ADMIN_PASSWORD}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8090/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s

networks:
  default:
    name: nodepass-network
EOF

log_info "docker-compose.deploy.yml 已生成"

# 完成
echo ""
log_info "=========================================="
log_info "构建完成!"
log_info "=========================================="
echo ""
log_info "镜像标签:"
echo "  - ${FULL_IMAGE_NAME}"
echo "  - ${IMAGE_NAME}:${VERSION}"
echo ""
log_info "使用 docker-compose 部署:"
echo "  docker-compose -f docker-compose.deploy.yml up -d"
echo ""
log_info "查看镜像:"
echo "  docker images | grep ${IMAGE_NAME}"
echo ""
