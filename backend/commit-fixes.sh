#!/bin/bash

# Git 提交脚本 - NodePass-Pro 后端安全修复

echo "准备提交代码审查修复..."
echo ""

# 检查 Git 状态
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "错误: 不在 Git 仓库中"
    exit 1
fi

# 显示修改的文件
echo "修改的文件:"
git status --short

echo ""
echo "是否要创建提交? (y/n)"
read -r response

if [[ "$response" != "y" ]]; then
    echo "已取消"
    exit 0
fi

# 添加所有修改的文件
git add \
    cmd/admin-bootstrap/main.go \
    cmd/server/main.go \
    internal/database/db.go \
    internal/config/config.go \
    internal/services/auth_service.go \
    internal/middleware/csrf.go \
    internal/middleware/rate_limit.go \
    internal/middleware/logger.go \
    internal/handlers/auth_handler.go \
    internal/services/verification_code_service.go \
    internal/handlers/helpers.go \
    internal/handlers/health_handler.go \
    internal/middleware/request_id.go \
    config.example.yaml \
    CODE_REVIEW_FIXES.md \
    QUICKSTART.md \
    UPGRADE.md \
    SUMMARY.md \
    verify-fixes.sh \
    go.mod \
    go.sum

# 创建提交
git commit -m "fix: 代码审查安全修复和功能增强

## 🔴 严重问题修复

- 修复管理员密码重复哈希漏洞 (admin-bootstrap)
- 添加 MySQL SSL/TLS 支持
- 修复登录时间更新的数据竞态条件

## 🔒 安全问题修复

- 修复 CSRF Cookie HttpOnly 配置错误
- 优化验证码存��机制 (优先 Redis，降级数据库)
- 改进随机数生成实现

## 💎 代码质量改进

- 数据库连接池参数可配置化
- 提取重复的字段验证逻辑
- 修复限流器 goroutine 泄漏

## 🚀 新增功能

- 请求 ID 追踪 (UUID)
- 日志增强 (包含 request_id)
- 健康检查端点 (/health, /readiness, /liveness)

## 📝 文档

- 添加详细修复说明 (CODE_REVIEW_FIXES.md)
- 添加快速启动指南 (QUICKSTART.md)
- 添加升级迁移指南 (UPGRADE.md)
- 添加配置示例 (config.example.yaml)
- 添加验证脚本 (verify-fixes.sh)

## 🔧 依赖

- 添加 github.com/google/uuid

## ✅ 测试

- 编译检查通过
- 单元测试通过
- go vet 通过
- 代码格式化完成

详细信息请查看 CODE_REVIEW_FIXES.md 和 SUMMARY.md
"

echo ""
echo "✅ 提交完成！"
echo ""
echo "下一步:"
echo "1. 查看提交: git show"
echo "2. 推送到远程: git push origin main"
echo "3. 创建 Pull Request (如果需要)"
echo ""
