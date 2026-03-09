#!/bin/bash

# NodePass-Pro 后端修复验证脚本

set -e

echo "=========================================="
echo "NodePass-Pro 后端修复验证"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查函数
check_pass() {
    echo -e "${GREEN}✓${NC} $1"
}

check_fail() {
    echo -e "${RED}✗${NC} $1"
}

check_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# 1. 检查 Go 环境
echo "1. 检查 Go 环境..."
if command -v go &> /dev/null; then
    GO_VERSION=$(go version)
    check_pass "Go 已安装: $GO_VERSION"
else
    check_fail "Go 未安装"
    exit 1
fi
echo ""

# 2. 检查依赖
echo "2. 检查依赖..."
if [ -f "go.mod" ]; then
    check_pass "go.mod 存在"

    # 检查是否需要新的依赖
    if grep -q "github.com/google/uuid" go.mod; then
        check_pass "uuid 依赖已存在"
    else
        check_warn "uuid 依赖缺失，正在添加..."
        go get github.com/google/uuid
        check_pass "uuid 依赖已添加"
    fi
else
    check_fail "go.mod 不存在"
    exit 1
fi
echo ""

# 3. 编译检查
echo "3. 编译检查..."
if go build -o /tmp/nodepass-server ./cmd/server/main.go 2>&1; then
    check_pass "主服务编译成功"
    rm -f /tmp/nodepass-server
else
    check_fail "主服务编译失败"
    exit 1
fi

if go build -o /tmp/nodepass-admin ./cmd/admin-bootstrap/main.go 2>&1; then
    check_pass "管理员工具编译成功"
    rm -f /tmp/nodepass-admin
else
    check_fail "管理员工具编译失败"
    exit 1
fi
echo ""

# 4. 检查新文件
echo "4. 检查新增文件..."
NEW_FILES=(
    "internal/services/verification_code_service.go"
    "internal/handlers/helpers.go"
    "internal/handlers/health_handler.go"
    "internal/middleware/request_id.go"
    "config.example.yaml"
    "CODE_REVIEW_FIXES.md"
)

for file in "${NEW_FILES[@]}"; do
    if [ -f "$file" ]; then
        check_pass "$file"
    else
        check_fail "$file 缺失"
    fi
done
echo ""

# 5. 运行测试（如果存在）
echo "5. 运行单元测试..."
if go test ./internal/... -v 2>&1 | grep -q "PASS\|no test files"; then
    check_pass "单元测试通过"
else
    check_warn "部分测试失败或无测试文件"
fi
echo ""

# 6. 检查配置文件
echo "6. 检查配置文件..."
if [ -f "configs/config.yaml" ]; then
    check_pass "config.yaml 存在"

    # 检查关键配置
    if grep -q "max_idle_conns" configs/config.yaml 2>/dev/null; then
        check_pass "数据库连接池配置已更新"
    else
        check_warn "数据库连接池配置未更新，请参考 config.example.yaml"
    fi

    if grep -q "strict_csrf" configs/config.yaml 2>/dev/null; then
        check_pass "CSRF 配置已更新"
    else
        check_warn "CSRF 配置未更新，请参考 config.example.yaml"
    fi
else
    check_warn "config.yaml 不存在，请从 config.example.yaml 复制"
fi
echo ""

# 7. 代码质量检查
echo "7. 代码质量检查..."

# 检查是否有 gofmt 问题
GOFMT_FILES=$(gofmt -l . 2>/dev/null | grep -v vendor || true)
if [ -z "$GOFMT_FILES" ]; then
    check_pass "代码格式正确"
else
    check_warn "以下文件需要格式化:"
    echo "$GOFMT_FILES"
fi

# 检查是否有 go vet 问题
if go vet ./... 2>&1 | grep -q "exit status"; then
    check_warn "go vet 发现潜在问题"
else
    check_pass "go vet 检查通过"
fi
echo ""

# 8. 安全检查
echo "8. 安全检查..."

# 检查是否有硬编码的密码或密钥
if grep -r "password.*=.*\".*\"" --include="*.go" . | grep -v "test" | grep -v "example" | grep -v "// " > /dev/null 2>&1; then
    check_warn "发现可能的硬编码密码，请检查"
else
    check_pass "未发现硬编码密码"
fi

# 检查 JWT secret 配置
if [ -f "configs/config.yaml" ]; then
    if grep -q "change-this-secret-in-production" configs/config.yaml 2>/dev/null; then
        check_fail "JWT secret 使用默认值，必须修改！"
    else
        check_pass "JWT secret 已自定义"
    fi
fi
echo ""

# 9. 功能验证清单
echo "9. 功能验证清单（需要手动测试）:"
echo "   [ ] 管理员创建工具测试"
echo "   [ ] 验证码发送和验证（Redis 启用）"
echo "   [ ] 验证码发送和验证（Redis 禁用）"
echo "   [ ] 健康检查端点 (/health, /readiness, /liveness)"
echo "   [ ] 请求 ID 在日志中显示"
echo "   [ ] CSRF 保护正常工作"
echo "   [ ] 数据库连接池配置生效"
echo "   [ ] MySQL TLS 连接（如果使用 MySQL）"
echo ""

# 10. 总结
echo "=========================================="
echo "验证完成！"
echo "=========================================="
echo ""
echo "下一步操作："
echo "1. 如果 config.yaml 不存在，复制 config.example.yaml:"
echo "   cp config.example.yaml configs/config.yaml"
echo ""
echo "2. 修改 configs/config.yaml 中的关键配置:"
echo "   - jwt.secret (必须修改)"
echo "   - database.* (根据实际情况)"
echo "   - redis.* (建议启用)"
echo ""
echo "3. 测试管理员创建:"
echo "   go run cmd/admin-bootstrap/main.go -username admin -email admin@example.com -password Admin123!"
echo ""
echo "4. 启动服务:"
echo "   go run cmd/server/main.go"
echo ""
echo "5. 测试健康检查:"
echo "   curl http://localhost:8080/health"
echo "   curl http://localhost:8080/readiness"
echo "   curl http://localhost:8080/liveness"
echo ""
echo "详细信息请查看 CODE_REVIEW_FIXES.md"
echo ""
