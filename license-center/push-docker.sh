#!/bin/bash

# Docker 镜像推送脚本
# 用法: ./push-docker.sh [registry/image-name] [tag]

set -e

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# 默认配置
SOURCE_IMAGE="nodepass/license-center"
TARGET_IMAGE=${1:-$SOURCE_IMAGE}
TAG=${2:-"1.0.0"}

SOURCE_FULL="${SOURCE_IMAGE}:${TAG}"
TARGET_FULL="${TARGET_IMAGE}:${TAG}"
TARGET_LATEST="${TARGET_IMAGE}:latest"

echo "🚀 Docker 镜像推送"
echo ""
log_info "源镜像: ${SOURCE_FULL}"
log_info "目标镜像: ${TARGET_FULL}"
log_info "最新标签: ${TARGET_LATEST}"
echo ""

# 检查源镜像是否存在
if ! docker images | grep -q "${SOURCE_IMAGE}.*${TAG}"; then
    echo "❌ 源镜像不存在: ${SOURCE_FULL}"
    echo ""
    echo "请先构建镜像:"
    echo "  ./quick-build.sh ${TAG}"
    exit 1
fi

# 如果目标镜像名称不同，需要重新标记
if [ "${SOURCE_IMAGE}" != "${TARGET_IMAGE}" ]; then
    log_step "标记镜像..."
    docker tag "${SOURCE_FULL}" "${TARGET_FULL}"
    docker tag "${SOURCE_FULL}" "${TARGET_LATEST}"
    log_info "镜像已标记"
fi

# 推送镜像
log_step "推送镜像到仓库..."

echo ""
echo "推送: ${TARGET_FULL}"
docker push "${TARGET_FULL}"

echo ""
echo "推送: ${TARGET_LATEST}"
docker push "${TARGET_LATEST}"

echo ""
log_info "✅ 镜像推送成功!"
echo ""
log_info "拉取命令:"
echo "  docker pull ${TARGET_FULL}"
echo ""
log_info "运行命令:"
echo "  docker run -d -p 8090:8090 ${TARGET_FULL}"
echo ""
