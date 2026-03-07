#!/bin/bash
# NodePass-Pro 快速启动脚本

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}NodePass-Pro 节点分组架构 - 快速启动${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 检查当前目录
if [ ! -f "backend/cmd/server/main.go" ]; then
    echo -e "${YELLOW}错误：请在项目根目录运行此脚本${NC}"
    exit 1
fi

echo -e "${GREEN}[1/4] 检查编译状态...${NC}"
if [ ! -f "/tmp/nodepass-server" ]; then
    echo "后端未编译，正在编译..."
    cd backend
    go build -o /tmp/nodepass-server ./cmd/server/main.go
    cd ..
fi
echo "✓ 后端编译完成"

if [ ! -d "frontend/dist" ]; then
    echo "前端未编译，正在编译..."
    cd frontend
    npm run build
    cd ..
fi
echo "✓ 前端编译完成"
echo ""

echo -e "${GREEN}[2/4] 数据库迁移${NC}"
echo "请确保数据库已启动，然后运行："
echo "  cd backend && go run ./cmd/migrate up"
echo ""
read -p "是否已完成数据库迁移？(y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "请先完成数据库迁移"
    exit 1
fi
echo ""

echo -e "${GREEN}[3/4] 启动服务${NC}"
echo "后端服务将在 http://localhost:8080 启动"
echo "前端文件位于 frontend/dist/ 目录"
echo ""

echo -e "${GREEN}[4/4] 测试功能${NC}"
echo "1. 访问 http://localhost:8080"
echo "2. 使用管理员账号登录"
echo "3. 创建节点组"
echo "4. 生成部署命令"
echo "5. 创建隧道"
echo ""

echo -e "${GREEN}运行集成测试：${NC}"
echo "  ./tests/integration_test.sh"
echo ""

echo -e "${GREEN}启动后端服务：${NC}"
cd backend
go run ./cmd/server/main.go
