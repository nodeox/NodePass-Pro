#!/bin/bash

# 隧道管理增强功能安装脚本
# 用途：自动安装批量导入导出和模板系统

set -e

echo "=========================================="
echo "NodePass-Pro 隧道管理增强功能安装"
echo "=========================================="
echo ""

# 检查是否在项目根目录
if [ ! -f "VERSION" ]; then
    echo "❌ 错误：请在项目根目录运行此脚本"
    exit 1
fi

echo "✓ 检测到项目根目录"
echo ""

# 1. 安装后端依赖
echo "📦 步骤 1/4: 安装后端依赖..."
cd backend
if ! go mod tidy; then
    echo "❌ 安装后端依赖失败"
    exit 1
fi
echo "✓ 后端依赖安装完成"
echo ""

# 2. 运行数据库迁移
echo "🗄️  步骤 2/4: 运行数据库迁移..."
if ! go run ./cmd/migrate up; then
    echo "❌ 数据库迁移失败"
    echo "提示：请检查数据库连接配置"
    exit 1
fi
echo "✓ 数据库迁移完成"
echo ""

# 3. 编译后端
echo "🔨 步骤 3/4: 编译后端服务..."
if ! go build -o ../bin/nodepass-server ./cmd/server/main.go; then
    echo "❌ 后端编译失败"
    exit 1
fi
cd ..
echo "✓ 后端编译完成"
echo ""

# 4. 前端构建（可选）
echo "🎨 步骤 4/4: 构建前端（可选）..."
read -p "是否构建前端？(y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    cd frontend
    if [ ! -d "node_modules" ]; then
        echo "安装前端依赖..."
        npm install
    fi
    echo "构建前端..."
    if ! npm run build; then
        echo "❌ 前端构建失败"
        exit 1
    fi
    cd ..
    echo "✓ 前端构建完成"
else
    echo "⊘ 跳过前端构建"
fi
echo ""

# 完成
echo "=========================================="
echo "✅ 安装完成！"
echo "=========================================="
echo ""
echo "新增功能："
echo "  • 批量导出隧道配置（JSON/YAML）"
echo "  • 批量导入隧道配置"
echo "  • 隧道配置模板系统"
echo "  • 保存隧道为模板"
echo ""
echo "启动服务："
echo "  cd backend && go run ./cmd/server/main.go"
echo ""
echo "或使用编译后的二进制："
echo "  ./bin/nodepass-server"
echo ""
echo "查看文档："
echo "  docs/tunnel-import-export-template.md"
echo "  docs/tunnel-enhancement-summary.md"
echo ""
echo "=========================================="
