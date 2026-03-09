#!/bin/bash

# 快速升级脚本（无交互）
# 用法: ./quick-upgrade.sh [version]

set -e

VERSION=${1:-"latest"}
CONTAINER_NAME="license-center"
IMAGE_NAME="ghcr.io/nodeox/license-center"

echo "🚀 快速升级到版本: ${VERSION}"

# 备份
echo "📦 备份数据..."
BACKUP_DIR="./backups/backup_$(date +%Y%m%d_%H%M%S)"
mkdir -p ${BACKUP_DIR}
[ -d "./data" ] && cp -r ./data ${BACKUP_DIR}/
[ -d "./configs" ] && cp -r ./configs ${BACKUP_DIR}/
echo "✅ 备份完成: ${BACKUP_DIR}"

# 停止并删除旧容器
echo "🛑 停止旧容器..."
docker stop ${CONTAINER_NAME} 2>/dev/null || true
docker rm ${CONTAINER_NAME} 2>/dev/null || true

# 拉取新镜像
echo "📥 拉取新镜像..."
docker pull ${IMAGE_NAME}:${VERSION}

# 启动新容器
echo "🚀 启动新容器..."
docker run -d \
  --name ${CONTAINER_NAME} \
  -p 8090:8090 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/configs:/app/configs \
  -e JWT_SECRET="${JWT_SECRET:-default-secret}" \
  -e ADMIN_PASSWORD="${ADMIN_PASSWORD:-ChangeMe123!}" \
  --restart unless-stopped \
  ${IMAGE_NAME}:${VERSION}

# 等待启动
echo "⏳ 等待服务启动..."
sleep 10

# 健康检查
if curl -f http://localhost:8090/health > /dev/null 2>&1; then
    echo "✅ 升级成功！"
    echo ""
    echo "访问地址: http://localhost:8090"
    echo "备份位置: ${BACKUP_DIR}"
else
    echo "❌ 健康检查失败"
    echo "查看日志: docker logs ${CONTAINER_NAME}"
    exit 1
fi
