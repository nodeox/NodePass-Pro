# 安全功能增强 - Refresh Token 机制

## 功能概述

实现了完整的 Refresh Token 机制，提供更安全的认证体系。

---

## 主要特性

### 1. Token 分离

- **Access Token**: 短期（30分钟），用于 API 访问
- **Refresh Token**: 长期（7天），用于刷新 Access Token

### 2. Token 轮换（Rotation）

- 每次刷新时生成新的 Refresh Token
- 自动撤销旧的 Refresh Token
- 防止 Token 被盗用后长期滥用

### 3. Token 撤销

- 支持单个 Token 撤销（登出）
- 支持撤销用户所有 Token（强制登出所有设备）
- Token 存储在数据库中，可随时撤销

### 4. 安全特性

- Refresh Token 使用 SHA256 哈希存储
- 记录创建时的 IP 和 User-Agent
- 记录最后使用时间
- 自动清理过期 Token

---

## API 接口

### 1. 登录（V2）

**POST** `/api/v1/auth/login/v2`

**请求**:
```json
{
  "account": "user@example.com",
  "password": "password123"
}
```

**响应**:
```json
{
  "code": 0,
  "message": "登录成功",
  "data": {
    "access_token": "eyJhbGc...",
    "refresh_token": "random-base64-string",
    "expires_in": 1800,
    "token_type": "Bearer",
    "user": {
      "id": 1,
      "username": "user",
      "email": "user@example.com",
      ...
    }
  }
}
```

### 2. 刷新 Token

**POST** `/api/v1/auth/refresh/v2`

**请求**:
```json
{
  "refresh_token": "your-refresh-token"
}
```

**响应**:
```json
{
  "code": 0,
  "message": "刷新成功",
  "data": {
    "access_token": "new-access-token",
    "refresh_token": "new-refresh-token",
    "expires_in": 1800,
    "token_type": "Bearer",
    "user": {...}
  }
}
```

### 3. 登出

**POST** `/api/v1/auth/logout`

**请求**:
```json
{
  "refresh_token": "your-refresh-token"
}
```

**响应**:
```json
{
  "code": 0,
  "message": "登出成功",
  "data": null
}
```

### 4. 撤销所有 Token

**POST** `/api/v1/auth/revoke-all`

需要认证（Bearer Token）

**响应**:
```json
{
  "code": 0,
  "message": "已撤销所有登录会话",
  "data": null
}
```

---

## 数据库模型

### RefreshToken 表

```sql
CREATE TABLE refresh_tokens (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP,

    user_id BIGINT NOT NULL,
    token_hash VARCHAR(64) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    is_revoked BOOLEAN DEFAULT FALSE,

    ip_address VARCHAR(45),
    user_agent VARCHAR(512),
    last_used_at TIMESTAMP,

    INDEX idx_user_id (user_id),
    INDEX idx_token_hash (token_hash),
    INDEX idx_expires_at (expires_at),
    INDEX idx_is_revoked (is_revoked)
);
```

---

## 使用示例

### 前端集成

```typescript
// 登录
const login = async (account: string, password: string) => {
  const response = await fetch('/api/v1/auth/login/v2', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ account, password })
  })

  const data = await response.json()

  // 存储 tokens
  localStorage.setItem('access_token', data.data.access_token)
  localStorage.setItem('refresh_token', data.data.refresh_token)

  return data.data
}

// 自动刷新
const refreshToken = async () => {
  const refreshToken = localStorage.getItem('refresh_token')

  const response = await fetch('/api/v1/auth/refresh/v2', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken })
  })

  const data = await response.json()

  // 更新 tokens
  localStorage.setItem('access_token', data.data.access_token)
  localStorage.setItem('refresh_token', data.data.refresh_token)

  return data.data.access_token
}

// API 请求拦截器
axios.interceptors.response.use(
  response => response,
  async error => {
    if (error.response?.status === 401) {
      try {
        const newToken = await refreshToken()
        error.config.headers.Authorization = `Bearer ${newToken}`
        return axios(error.config)
      } catch (refreshError) {
        // 刷新失败，跳转登录页
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

// 登出
const logout = async () => {
  const refreshToken = localStorage.getItem('refresh_token')

  await fetch('/api/v1/auth/logout', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken })
  })

  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
}
```

---

## 安全建议

1. **Access Token 存储**:
   - 推荐存储在内存中（最安全）
   - 或使用 sessionStorage（关闭浏览器后清除）
   - 避免使用 localStorage（XSS 风险）

2. **Refresh Token 存储**:
   - 可以使用 localStorage（需要 XSS 防护）
   - 或使用 HttpOnly Cookie（最安全，但需要后端支持）

3. **Token 刷新策略**:
   - 在 Access Token 过期前 5 分钟自动刷新
   - 或在收到 401 错误时刷新

4. **安全措施**:
   - 启用 HTTPS
   - 实施 CSP 策略防止 XSS
   - 定期清理过期 Token
   - 监控异常的 Token 使用模式

---

## 迁移指南

### 从旧版 API 迁移

旧版 API (`/api/v1/auth/login`) 仍然可用，但建议迁移到新版：

1. 更新登录接口：`/auth/login` → `/auth/login/v2`
2. 存储 `refresh_token`
3. 实现自动刷新逻辑
4. 更新登出逻辑

### 兼容性

- 旧版 API 返回的 `token` 字段改为 `access_token`
- 旧版 API 不返回 `refresh_token`
- 新旧 API 可以共存

---

## 后续计划

- [ ] 实现 HttpOnly Cookie 存储 Refresh Token
- [ ] 添加 Token 使用统计和异常检测
- [ ] 实现设备管理（查看和撤销特定设备的 Token）
- [ ] 添加 Token 刷新通知（邮件/Telegram）

---

## 相关文件

- `backend/internal/models/refresh_token.go` - 数据模型
- `backend/internal/services/refresh_token_service.go` - Token 服务
- `backend/internal/services/auth_refresh_token.go` - 认证增强
- `backend/internal/handlers/auth_handler.go` - API 处理器
- `backend/cmd/server/main.go` - 路由配置
