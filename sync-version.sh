#!/bin/bash

# 版本同步脚本
# 用途：统一更新所有组件的版本号

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "NodePass-Pro 版本同步工具"
echo "=========================================="
echo ""

# 检查是否在项目根目录
if [ ! -f "VERSION" ]; then
    echo -e "${RED}❌ 错误：请在项目根目录运行此脚本${NC}"
    exit 1
fi

# 读取当前版本
CURRENT_VERSION=$(cat VERSION)
echo -e "${GREEN}当前版本：${CURRENT_VERSION}${NC}"
echo ""

# 询问新版本号
read -p "请输入新版本号（留空保持当前版本）: " NEW_VERSION

if [ -z "$NEW_VERSION" ]; then
    NEW_VERSION=$CURRENT_VERSION
    echo -e "${YELLOW}保持当前版本：${NEW_VERSION}${NC}"
else
    echo -e "${GREEN}新版本号：${NEW_VERSION}${NC}"
fi

echo ""
echo "=========================================="
echo "开始同步版本..."
echo "=========================================="
echo ""

# 1. 更新根目录 VERSION 文件
echo "📝 更新 VERSION 文件..."
echo "$NEW_VERSION" > VERSION
echo -e "${GREEN}✓ VERSION 文件已更新${NC}"

# 2. 更新 version.yaml
echo "📝 更新 version.yaml..."
if [ -f "version.yaml" ]; then
    sed -i '' "s/^version: .*/version: \"$NEW_VERSION\"/" version.yaml
    sed -i '' "s/version: \"[0-9.]*\"/version: \"$NEW_VERSION\"/g" version.yaml
    echo -e "${GREEN}✓ version.yaml 已更新${NC}"
else
    echo -e "${YELLOW}⚠ version.yaml 不存在，跳过${NC}"
fi

# 3. 更新后端版本
echo "📝 更新后端版本..."
if [ -f "backend/internal/version/version.go" ]; then
    sed -i '' "s/var Version = \"[0-9.]*\"/var Version = \"$NEW_VERSION\"/" backend/internal/version/version.go
    echo -e "${GREEN}✓ 后端版本已更新${NC}"
else
    echo -e "${YELLOW}⚠ 后端版本文件不存在${NC}"
fi

# 4. 更新前端版本
echo "📝 更新前端版本..."
if [ -f "frontend/package.json" ]; then
    sed -i '' "s/\"version\": \"[0-9.]*\"/\"version\": \"$NEW_VERSION\"/" frontend/package.json
    echo -e "${GREEN}✓ 前端版本已更新${NC}"
else
    echo -e "${YELLOW}⚠ 前端 package.json 不存在${NC}"
fi

# 5. 更新节点客户端版本
echo "📝 更新节点客户端版本..."
if [ -f "nodeclient/internal/agent/agent.go" ]; then
    sed -i '' "s/var clientVersion = \"[0-9.]*\"/var clientVersion = \"$NEW_VERSION\"/" nodeclient/internal/agent/agent.go
    echo -e "${GREEN}✓ 节点客户端版本已更新${NC}"
else
    echo -e "${YELLOW}⚠ 节点客户端版本文件不存在${NC}"
fi

# 6. 更新授权中心版本
echo "📝 更新授权中心版本..."
if [ -f "license-center/web-ui/package.json" ]; then
    sed -i '' "s/\"version\": \"[0-9.]*\"/\"version\": \"$NEW_VERSION\"/" license-center/web-ui/package.json
    echo -e "${GREEN}✓ 授权中心版本已更新${NC}"
else
    echo -e "${YELLOW}⚠ 授权中心 package.json 不存在${NC}"
fi

echo ""
echo "=========================================="
echo "版本同步完成！"
echo "=========================================="
echo ""

# 显示更新后的版本信息
echo "各组件版本信息："
echo ""

echo "📦 根目录版本："
cat VERSION
echo ""

echo "🔧 后端版本："
grep "var Version" backend/internal/version/version.go | sed 's/.*"\(.*\)".*/\1/' || echo "未找到"
echo ""

echo "🎨 前端版本："
grep '"version"' frontend/package.json | head -1 | sed 's/.*"\(.*\)".*/\1/' || echo "未找到"
echo ""

echo "🖥️  节点客户端版本："
grep "var clientVersion" nodeclient/internal/agent/agent.go | sed 's/.*"\(.*\)".*/\1/' || echo "未找到"
echo ""

echo "🔐 授权中心版本："
grep '"version"' license-center/web-ui/package.json | head -1 | sed 's/.*"\(.*\)".*/\1/' || echo "未找到"
echo ""

echo "=========================================="
echo -e "${GREEN}✅ 所有组件版本已同步为：${NEW_VERSION}${NC}"
echo "=========================================="
echo ""

# 询问是否创建 Git 标签
read -p "是否创建 Git 标签？(y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    TAG_NAME="v${NEW_VERSION}"
    echo "创建 Git 标签：${TAG_NAME}"

    git add VERSION version.yaml backend/internal/version/version.go frontend/package.json nodeclient/internal/agent/agent.go license-center/web-ui/package.json 2>/dev/null || true

    read -p "请输入标签描述: " TAG_MESSAGE
    if [ -z "$TAG_MESSAGE" ]; then
        TAG_MESSAGE="Release version ${NEW_VERSION}"
    fi

    git commit -m "chore: bump version to ${NEW_VERSION}" 2>/dev/null || echo "没有需要提交的更改"
    git tag -a "${TAG_NAME}" -m "${TAG_MESSAGE}"

    echo -e "${GREEN}✓ Git 标签已创建：${TAG_NAME}${NC}"
    echo ""
    echo "推送到远程仓库："
    echo "  git push origin main"
    echo "  git push origin ${TAG_NAME}"
fi

echo ""
echo "完成！"
