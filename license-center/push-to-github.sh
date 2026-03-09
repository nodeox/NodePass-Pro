#!/bin/bash

# 推送到 GitHub Container Registry (ghcr.io)
# 用法: ./push-to-github.sh [github-username] [tag]

set -e

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

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

# 配置
GITHUB_USERNAME=${1:-""}
TAG=${2:-"1.0.0"}
SOURCE_IMAGE="nodepass/license-center:${TAG}"
GHCR_IMAGE="ghcr.io/${GITHUB_USERNAME}/license-center"

echo "🚀 推送到 GitHub Container Registry"
echo ""

# 检查 GitHub 用户名
if [ -z "$GITHUB_USERNAME" ]; then
    log_error "请提供 GitHub 用户名"
    echo ""
    echo "用法: $0 <github-username> [tag]"
    echo "示例: $0 yourusername 1.0.0"
    exit 1
fi

log_info "源镜像: ${SOURCE_IMAGE}"
log_info "目标镜像: ${GHCR_IMAGE}:${TAG}"
log_info "最新标签: ${GHCR_IMAGE}:latest"
echo ""

# 检查源镜像是否存在
if ! docker images | grep -q "nodepass/license-center.*${TAG}"; then
    log_error "源镜像不存在: ${SOURCE_IMAGE}"
    echo ""
    echo "请先构建镜像:"
    echo "  ./quick-build.sh ${TAG}"
    exit 1
fi

# 检查是否已登录 GitHub Container Registry
log_step "检查 GitHub Container Registry 登录状态..."
if ! docker info 2>/dev/null | grep -q "ghcr.io"; then
    log_warn "未登录到 GitHub Container Registry"
    echo ""
    echo "请先登录 GitHub Container Registry:"
    echo ""
    echo "1. 创建 GitHub Personal Access Token (PAT):"
    echo "   - 访问: https://github.com/settings/tokens"
    echo "   - 点击 'Generate new token (classic)'"
    echo "   - 选择权限: write:packages, read:packages, delete:packages"
    echo "   - 生成并复制 token"
    echo ""
    echo "2. 登录 ghcr.io:"
    echo "   export CR_PAT=YOUR_TOKEN"
    echo "   echo \$CR_PAT | docker login ghcr.io -u ${GITHUB_USERNAME} --password-stdin"
    echo ""
    read -p "是否已经登录? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_warn "请先登录后再运行此脚本"
        exit 0
    fi
fi

# 标记镜像
log_step "标记镜像..."
docker tag "${SOURCE_IMAGE}" "${GHCR_IMAGE}:${TAG}"
docker tag "${SOURCE_IMAGE}" "${GHCR_IMAGE}:latest"
log_info "镜像已标记"

# 推送镜像
log_step "推送镜像到 GitHub Container Registry..."

echo ""
log_info "推送: ${GHCR_IMAGE}:${TAG}"
if docker push "${GHCR_IMAGE}:${TAG}"; then
    log_info "✅ 推送成功: ${GHCR_IMAGE}:${TAG}"
else
    log_error "❌ 推送失败: ${GHCR_IMAGE}:${TAG}"
    exit 1
fi

echo ""
log_info "推送: ${GHCR_IMAGE}:latest"
if docker push "${GHCR_IMAGE}:latest"; then
    log_info "✅ 推送成功: ${GHCR_IMAGE}:latest"
else
    log_error "❌ 推送失败: ${GHCR_IMAGE}:latest"
    exit 1
fi

# 完成
echo ""
log_info "=========================================="
log_info "推送完成!"
log_info "=========================================="
echo ""
log_info "镜像地址:"
echo "  - ${GHCR_IMAGE}:${TAG}"
echo "  - ${GHCR_IMAGE}:latest"
echo ""
log_info "查看镜像:"
echo "  https://github.com/${GITHUB_USERNAME}?tab=packages"
echo ""
log_info "拉取镜像:"
echo "  docker pull ${GHCR_IMAGE}:${TAG}"
echo ""
log_info "运行容器:"
echo "  docker run -d -p 8090:8090 ${GHCR_IMAGE}:${TAG}"
echo ""
log_info "设置镜像为公开 (可选):"
echo "  1. 访问: https://github.com/${GITHUB_USERNAME}?tab=packages"
echo "  2. 点击 'license-center' 包"
echo "  3. 点击 'Package settings'"
echo "  4. 在 'Danger Zone' 中点击 'Change visibility'"
echo "  5. 选择 'Public'"
echo ""
