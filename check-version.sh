#!/bin/bash

# 版本检查工具
# 用途：检查所有组件的版本是否一致

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "=========================================="
echo "NodePass-Pro 版本检查工具"
echo "=========================================="
echo ""

# 检查是否在项目根目录
if [ ! -f "VERSION" ]; then
    echo -e "${RED}❌ 错误：请在项目根目录运行此脚本${NC}"
    exit 1
fi

# 读取各组件版本
echo "📋 读取各组件版本..."
echo ""

# 1. 根目录版本
ROOT_VERSION=$(cat VERSION 2>/dev/null || echo "未找到")
echo -e "${BLUE}根目录版本：${NC}${ROOT_VERSION}"

# 2. version.yaml 版本
YAML_VERSION=$(grep "^version:" version.yaml 2>/dev/null | sed 's/version: "\(.*\)"/\1/' || echo "未找到")
echo -e "${BLUE}version.yaml：${NC}${YAML_VERSION}"

# 3. 后端版本
BACKEND_VERSION=$(grep "var Version" backend/internal/version/version.go 2>/dev/null | sed 's/.*"\(.*\)".*/\1/' || echo "未找到")
echo -e "${BLUE}后端版本：${NC}${BACKEND_VERSION}"

# 4. 前端版本
FRONTEND_VERSION=$(grep '"version"' frontend/package.json 2>/dev/null | head -1 | sed 's/.*: "\(.*\)".*/\1/' || echo "未找到")
echo -e "${BLUE}前端版本：${NC}${FRONTEND_VERSION}"

# 5. 节点客户端版本
NODECLIENT_VERSION=$(grep "var clientVersion" nodeclient/internal/agent/agent.go 2>/dev/null | sed 's/.*"\(.*\)".*/\1/' || echo "未找到")
echo -e "${BLUE}节点客户端版本：${NC}${NODECLIENT_VERSION}"

# 6. 授权中心版本（可选组件）
LICENSE_CENTER_PRESENT=0
if [ -f "license-center/web-ui/package.json" ]; then
    LICENSE_CENTER_PRESENT=1
    LICENSE_VERSION=$(grep '"version"' license-center/web-ui/package.json 2>/dev/null | head -1 | sed 's/.*: "\(.*\)".*/\1/' || echo "未找到")
    echo -e "${BLUE}授权中心版本：${NC}${LICENSE_VERSION}"
else
    LICENSE_VERSION="已移除/不适用"
    echo -e "${BLUE}授权中心版本：${NC}${LICENSE_VERSION}"
fi

echo ""
echo "=========================================="
echo "版本一致性检查"
echo "=========================================="
echo ""

# 检查版本一致性
INCONSISTENT=0

if [ "$ROOT_VERSION" != "$BACKEND_VERSION" ]; then
    echo -e "${RED}❌ 根目录版本 ($ROOT_VERSION) 与后端版本 ($BACKEND_VERSION) 不一致${NC}"
    INCONSISTENT=1
fi

if [ "$ROOT_VERSION" != "$FRONTEND_VERSION" ]; then
    echo -e "${RED}❌ 根目录版本 ($ROOT_VERSION) 与前端版本 ($FRONTEND_VERSION) 不一致${NC}"
    INCONSISTENT=1
fi

if [ "$ROOT_VERSION" != "$NODECLIENT_VERSION" ]; then
    echo -e "${RED}❌ 根目录版本 ($ROOT_VERSION) 与节点客户端版本 ($NODECLIENT_VERSION) 不一致${NC}"
    INCONSISTENT=1
fi

if [ "$LICENSE_CENTER_PRESENT" -eq 1 ] && [ "$ROOT_VERSION" != "$LICENSE_VERSION" ]; then
    echo -e "${RED}❌ 根目录版本 ($ROOT_VERSION) 与授权中心版本 ($LICENSE_VERSION) 不一致${NC}"
    INCONSISTENT=1
fi

if [ "$ROOT_VERSION" != "$YAML_VERSION" ]; then
    echo -e "${YELLOW}⚠️  根目录版本 ($ROOT_VERSION) 与 version.yaml ($YAML_VERSION) 不一致${NC}"
fi

if [ $INCONSISTENT -eq 0 ]; then
    echo -e "${GREEN}✅ 所有组件版本一致：${ROOT_VERSION}${NC}"
    echo ""
    echo "=========================================="
    echo "版本信息汇总"
    echo "=========================================="
    echo ""
    echo "当前版本：${ROOT_VERSION}"
    echo ""
    echo "组件列表："
    echo "  ✓ 后端服务"
    echo "  ✓ 前端应用"
    echo "  ✓ 节点客户端"
    if [ "$LICENSE_CENTER_PRESENT" -eq 1 ]; then
        echo "  ✓ 授权中心"
    else
        echo "  - 授权中心（已移除/不适用）"
    fi
    echo ""
    exit 0
else
    echo ""
    echo -e "${RED}=========================================="
    echo "发现版本不一致！"
    echo "==========================================${NC}"
    echo ""
    echo "建议运行以下命令同步版本："
    echo "  ./sync-version.sh"
    echo ""
    exit 1
fi
