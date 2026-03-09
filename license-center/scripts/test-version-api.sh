#!/bin/bash

# 版本管理 API 测试脚本
# 用法: ./scripts/test-version-api.sh

set -e

BASE_URL="${BASE_URL:-http://localhost:8080/api/v1}"
TOKEN="${TOKEN:-}"

if [ -z "$TOKEN" ]; then
    echo "请先设置 TOKEN 环境变量"
    echo "示例: export TOKEN=your_jwt_token"
    exit 1
fi

echo "=== 版本管理 API 测试 ==="
echo "Base URL: $BASE_URL"
echo ""

# 1. 获取系统版本信息
echo "1. 获取系统版本信息"
curl -s -X GET "$BASE_URL/versions/system" \
    -H "Authorization: Bearer $TOKEN" \
    | jq '.'
echo ""

# 2. 更新后端版本
echo "2. 更新后端版本"
curl -s -X POST "$BASE_URL/versions/components" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "component": "backend",
        "version": "1.0.0",
        "git_commit": "abc123",
        "git_branch": "main",
        "description": "Initial release"
    }' \
    | jq '.'
echo ""

# 3. 更新前端版本
echo "3. 更新前端版本"
curl -s -X POST "$BASE_URL/versions/components" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "component": "frontend",
        "version": "1.0.0",
        "git_commit": "def456",
        "git_branch": "main",
        "description": "Initial release"
    }' \
    | jq '.'
echo ""

# 4. 更新节点客户端版本
echo "4. 更新节点客户端版本"
curl -s -X POST "$BASE_URL/versions/components" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "component": "node_client",
        "version": "1.0.0",
        "git_commit": "ghi789",
        "git_branch": "main",
        "description": "Initial release"
    }' \
    | jq '.'
echo ""

# 5. 更新授权中心版本
echo "5. 更新授权中心版本"
curl -s -X POST "$BASE_URL/versions/components" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "component": "license_center",
        "version": "1.0.0",
        "git_commit": "jkl012",
        "git_branch": "main",
        "description": "Initial release"
    }' \
    | jq '.'
echo ""

# 6. 创建兼容性配置
echo "6. 创建兼容性配置"
curl -s -X POST "$BASE_URL/versions/compatibility" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "backend_version": "1.0.0",
        "min_frontend_version": "1.0.0",
        "min_node_client_version": "1.0.0",
        "min_license_center_version": "1.0.0",
        "description": "Version 1.0.0 compatibility matrix"
    }' \
    | jq '.'
echo ""

# 7. 获取兼容性配置列表
echo "7. 获取兼容性配置列表"
curl -s -X GET "$BASE_URL/versions/compatibility" \
    -H "Authorization: Bearer $TOKEN" \
    | jq '.'
echo ""

# 8. 获取后端版本历史
echo "8. 获取后端版本历史"
curl -s -X GET "$BASE_URL/versions/components/backend/history?limit=10" \
    -H "Authorization: Bearer $TOKEN" \
    | jq '.'
echo ""

# 9. 再次获取系统版本信息（验证更新）
echo "9. 再次获取系统版本信息（验证更新）"
curl -s -X GET "$BASE_URL/versions/system" \
    -H "Authorization: Bearer $TOKEN" \
    | jq '.'
echo ""

echo "=== 测试完成 ==="
