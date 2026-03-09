#!/bin/bash

# 快速构建 Docker 镜像脚本
# 用法: ./quick-build.sh [tag]

set -e

TAG=${1:-latest}
IMAGE_NAME="nodepass/license-center"

echo "🚀 快速构建 Docker 镜像..."
echo "镜像: ${IMAGE_NAME}:${TAG}"
echo ""

# 获取版本信息
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

echo "📦 版本信息:"
echo "  Version:    ${VERSION}"
echo "  Git Commit: ${GIT_COMMIT}"
echo "  Git Branch: ${GIT_BRANCH}"
echo "  Build Time: ${BUILD_TIME}"
echo ""

# 构建镜像
echo "🔨 构建镜像..."
docker build \
    --file Dockerfile.version \
    --build-arg VERSION="${VERSION}" \
    --build-arg GIT_COMMIT="${GIT_COMMIT}" \
    --build-arg GIT_BRANCH="${GIT_BRANCH}" \
    --build-arg BUILD_TIME="${BUILD_TIME}" \
    --tag "${IMAGE_NAME}:${TAG}" \
    --tag "${IMAGE_NAME}:${VERSION}" \
    .

echo ""
echo "✅ 构建完成!"
echo ""
echo "📋 镜像标签:"
echo "  - ${IMAGE_NAME}:${TAG}"
echo "  - ${IMAGE_NAME}:${VERSION}"
echo ""
echo "🚀 运行镜像:"
echo "  docker run -d -p 8090:8090 ${IMAGE_NAME}:${TAG}"
echo ""
