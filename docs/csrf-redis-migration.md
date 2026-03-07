# CSRF 令牌存储迁移到 Redis

## 问题描述

原有的 CSRF 令牌存储在内存中（使用全局 map），存在以下问题：

1. **多实例部署不兼容**：不同实例之间无法共享 CSRF 令牌
2. **服务重启令牌丢失**：重启后所有令牌失效，影响用户体验
3. **内存占用**：长期运行可能积累大量过期令牌

## 解决方案

将 CSRF 令牌存储迁移到 Redis，实现：

- ✅ 支持分布式部署（多实例共享令牌）
- ✅ 持久化存储（服务重启不影响）
- ✅ 自动过期（利用 Redis TTL 机制）
- ✅ 向后兼容（Redis 未启用时降级为双重提交模式）

## 实现细节

### 1. 令牌存储

**Redis 启用时**：
- 令牌存储在 Redis 中，键格式：`csrf:token:{token}`
- 使用 Redis TTL 自动过期（24 小时）
- 支持分布式部署

**Redis 未启用时**：
- 降级为双重提交模式（Double Submit Cookie）
- 仅验证 Cookie 和 Header 中的令牌是否匹配
- 不进行服务端存储验证

### 2. 路径匹配优化

修复了原有的路径匹配逻辑，避免误判：

**之前的问题**：
```go
// 错误：/api/v1/nodes/123 会被误判为节点客户端路径
if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
    return true
}
```

**修复后**：
```go
// 精确匹配特定路径
exactPaths := []string{
    "/api/v1/nodes/register",
    "/api/v1/nodes/heartbeat",
    "/api/v1/traffic/report",
}

// 模式匹配：/api/v1/nodes/{id}/config
if path 以 "/api/v1/nodes/" 开头 && path 以 "/config" 结尾 {
    return true
}
```

### 3. 其他修复

修复了 `body_limit.go` 中的类型转换错误：
```go
// 错误：string(bytes) 会将数字转为 Unicode 字符
return string(bytes) + " B"

// 正确：使用 strconv.FormatInt
return strconv.FormatInt(bytes, 10) + " B"
```

## 测试

新增测试文件 `csrf_test.go`，包含：

1. **路径匹配测试**：验证节点客户端路径识别的准确性
2. **令牌生成测试**：验证令牌生成的随机性

运行测试：
```bash
cd backend
go test -v ./internal/middleware -run Test
```

## 配置

无需额外配置，自动使用现有的 Redis 配置：

```yaml
redis:
  enabled: true
  addr: "127.0.0.1:6379"
  password: ""
  db: 0
  key_prefix: "nodepass:panel"
  default_ttl: 300
```

CSRF 令牌会使用独立的键前缀 `csrf:token:`，不会与其他缓存冲突。

## 兼容性

- ✅ 完全向后兼容
- ✅ Redis 启用时自动使用 Redis 存储
- ✅ Redis 未启用时降级为双重提交模式
- ✅ 不影响现有 API 接口

## 性能影响

- **Redis 启用时**：每次 CSRF 验证需要一次 Redis 查询（~1ms）
- **Redis 未启用时**：无额外开销，仅验证令牌匹配

## 安全性提升

1. **分布式一致性**：多实例部署时令牌验证一致
2. **自动过期**：利用 Redis TTL 自动清理过期令牌
3. **精确路径匹配**：避免误判导致的安全漏洞

## 迁移步骤

无需手动迁移，代码部署后自动生效：

1. 部署新代码
2. 重启服务
3. 旧的内存令牌自动失效
4. 新令牌自动存储到 Redis（如果启用）

## 监控建议

建议监控以下指标：

- Redis 连接状态
- CSRF 验证失败率
- CSRF 令牌生成/验证延迟

## 相关文件

- `backend/internal/middleware/csrf.go` - CSRF 中间件实现
- `backend/internal/middleware/csrf_test.go` - 单元测试
- `backend/internal/middleware/body_limit.go` - 修复类型转换错误
- `backend/internal/cache/redis.go` - Redis 缓存模块
