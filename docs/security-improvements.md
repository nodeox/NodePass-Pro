# 安全改进说明

本文档记录了针对代码审查报告中高优先级安全问题的修复。

## 修复内容

### 1. PostgreSQL SSL 模式配置 ✅

**问题**: 数据库连接默认使用 `sslmode=disable`，导致凭证在网络上以明文传输。

**修复**:
- 在 `DatabaseConfig` 中添加了 `SSLMode` 字段
- 修改了 PostgreSQL 连接字符串生成逻辑，默认使用 `require` 而非 `disable`
- 在配置文件中添加了 `ssl_mode` 选项和详细说明

**配置示例**:
```yaml
database:
  type: "postgres"
  host: "127.0.0.1"
  port: 5432
  user: "postgres"
  password: "postgres"
  db_name: "nodepass_panel"
  ssl_mode: "require"  # 生产环境推荐使用 require 或 verify-full
```

**影响**:
- 开发环境：可以继续使用 `ssl_mode: "disable"`
- 生产环境：建议使用 `ssl_mode: "require"` 或 `ssl_mode: "verify-full"`
- 如果不设置 `ssl_mode`，默认使用 `require`

---

### 2. JWT Secret 长度验证 ✅

**问题**: JWT Secret 只检查是否为空和默认值，没有检查长度和复杂度。

**修复**:
- 在 `validateSecurityConfig()` 函数中添加了长度检查
- 要求 JWT Secret 至少 32 字符
- 提供了清晰的错误提示

**配置示例**:
```yaml
jwt:
  # 使用 openssl 生成强随机密钥
  # openssl rand -base64 48
  secret: "your-very-long-and-random-secret-key-at-least-32-characters"
  expire_time: 168
```

**生成强密钥的方法**:
```bash
# 方法 1: 使用 openssl
openssl rand -base64 48

# 方法 2: 使用 /dev/urandom
head -c 48 /dev/urandom | base64

# 方法 3: 使用 Python
python3 -c "import secrets; print(secrets.token_urlsafe(48))"
```

---

### 3. 安全 HTTP 头中间件 ✅

**问题**: 缺少安全相关的 HTTP 响应头。

**修复**:
- 创建了新的 `SecurityHeaders()` 中间件
- 添加了以下安全头：
  - `X-Content-Type-Options: nosniff` - 防止 MIME 类型嗅探
  - `X-Frame-Options: DENY` - 防止点击劫持
  - `X-XSS-Protection: 1; mode=block` - 启用 XSS 过滤器
  - `Strict-Transport-Security` - 强制 HTTPS（仅生产环境）
  - `Content-Security-Policy` - 内容安全策略
  - `Referrer-Policy` - 控制 Referer 信息
  - `Permissions-Policy` - 控制浏览器功能访问

**文件位置**: `/backend/internal/middleware/security_headers.go`

**自动应用**: 中间件已在 `setupRouter()` 中自动应用到所有路由。

---

### 4. SQLite 目录权限 ✅

**问题**: SQLite 数据目录权限为 0755，允许所有用户读取。

**修复**:
- 将目录权限从 `0755` 改为 `0700`
- 只有所有者可以访问数据目录

**影响**:
- 提高了 SQLite 数据库文件的安全性
- 防止其他用户读取敏感数据

---

### 5. 全局速率限制调整 ✅

**问题**: 全局速率限制为 50 QPS，对单个 IP 过高，容易被滥用。

**修复**:
- 将全局速率限制从 `50 QPS, 100 burst` 降低到 `20 QPS, 50 burst`
- 减少了被滥用的风险

**当前速率限制配置**:
- 全局: 20 QPS, 50 burst
- 登录端点: 0.2 QPS, 5 burst (每 5 秒 1 次)
- Telegram 登录: 0.5 QPS, 5 burst (每 2 秒 1 次)
- 心跳端点: 2 QPS, 20 burst (按 node_id 限流)

---

## 验证修改

### 编译测试
```bash
cd backend
go build -o server ./cmd/server/main.go
```

### 配置检查
启动服务器时，系统会自动验证：
1. JWT Secret 是否配置
2. JWT Secret 是否为默认值
3. JWT Secret 长度是否至少 32 字符

如果验证失败，服务器将拒绝启动并显示错误信息。

### 安全头验证
启动服务器后，可以使用 curl 验证安全头：
```bash
curl -I http://localhost:8080/health
```

应该看到以下响应头：
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'; ...
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: geolocation=(), microphone=(), camera=()
```

---

## 迁移指南

### 对于现有部署

1. **更新配置文件**:
   ```bash
   cd /opt/NodePass-Pro/backend/configs
   # 备份现有配置
   cp config.runtime.yaml config.runtime.yaml.backup

   # 添加新的配置项
   # 1. 如果使用 PostgreSQL，添加 ssl_mode 配置
   # 2. 检查 JWT Secret 长度是否至少 32 字符
   ```

2. **生成新的 JWT Secret**（如果当前长度不足）:
   ```bash
   # 生成新密钥
   NEW_SECRET=$(openssl rand -base64 48)
   echo "新的 JWT Secret: $NEW_SECRET"

   # 更新配置文件中的 jwt.secret
   # 注意：更新后所有现有 token 将失效，用户需要重新登录
   ```

3. **重启服务**:
   ```bash
   docker compose restart backend
   ```

### 对于新部署

新部署会自动使用这些安全改进，只需确保：
1. 在配置文件中设置强 JWT Secret（至少 32 字符）
2. 生产环境中设置 `database.ssl_mode: "require"`
3. 生产环境中设置 `server.mode: "release"`

---

## 注意事项

1. **JWT Secret 更新**: 如果更新了 JWT Secret，所有现有的 token 将失效，用户需要重新登录。

2. **PostgreSQL SSL**: 如果数据库服务器不支持 SSL，需要在配置中显式设置 `ssl_mode: "disable"`。

3. **CSP 策略**: 当前的 CSP 策略允许内联样式（Ant Design 需要）。如果前端不使用内联样式，可以移除 `'unsafe-inline'`。

4. **HSTS 头**: `Strict-Transport-Security` 头只在生产环境（`gin.ReleaseMode`）中启用，避免开发环境的 HTTPS 问题。

---

## 后续建议

虽然已经修复了高优先级问题，但仍建议在后续版本中实现：

1. **Token 刷新机制改进**: 实现分离的 access token 和 refresh token
2. **CSRF 令牌 TTL**: 从 24 小时降低到 1-2 小时
3. **API Key 认证**: 为 CLI 客户端实现专门的认证机制
4. **测试覆盖率**: 添加安全相关的单元测试
5. **API 文档**: 使用 Swagger/OpenAPI 生成文档

---

## 相关文件

- `/backend/internal/config/config.go` - 配置结构定义
- `/backend/internal/database/db.go` - 数据库连接逻辑
- `/backend/internal/middleware/security_headers.go` - 安全头中间件（新增）
- `/backend/cmd/server/main.go` - 服务器启动和配置验证
- `/backend/configs/config.yaml` - 配置文件示例

---

## 版本信息

- 修复日期: 2026-03-07
- 修复版本: 基于当前 main 分支
- 审查报告: 参见项目根目录的代码审查报告
