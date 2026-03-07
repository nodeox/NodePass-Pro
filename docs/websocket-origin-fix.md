# WebSocket Origin 验证修复

## 问题描述

原有的 WebSocket Origin 验证存在安全漏洞：

### 1. 使用 `strings.Contains` 导致绕过风险

**原代码**：
```go
if strings.Contains(origin, "localhost") ||
    strings.Contains(origin, "127.0.0.1") {
    return true
}
```

**问题**：
- `evil.com/localhost` 会被错误地允许
- `127.0.0.1.evil.com` 会被错误地允许
- 攻击者可以通过构造特殊域名绕过验证

### 2. 空 Origin 直接允许

**原代码**：
```go
if origin == "" {
    return true
}
```

**问题**：
- 生产环境下应该拒绝空 Origin
- 可能被用于 CSRF 攻击

### 3. 配置验证不精确

**原代码**：
```go
if strings.Contains(origin, allowed) {
    return true
}
```

**问题**：
- `evil-example.com` 会匹配 `example.com`
- 缺少 scheme 和端口验证

## 解决方案

### 1. 精确的 localhost 检测

```go
func isLocalhost(hostname string) bool {
    hostname = strings.ToLower(strings.TrimSpace(hostname))
    return hostname == "localhost" ||
        hostname == "127.0.0.1" ||
        hostname == "::1" ||
        hostname == "[::1]"
}
```

**改进**：
- 精确匹配，不使用 `Contains`
- 支持 IPv6 本地地址
- 大小写不敏感

### 2. 完整的 Origin 匹配逻辑

```go
func matchOrigin(originURL *url.URL, allowed string) bool {
    // 1. 完整 URL 匹配（包含 scheme 和 host）
    if strings.HasPrefix(allowed, "http://") || strings.HasPrefix(allowed, "https://") {
        allowedURL, _ := url.Parse(allowed)
        return originURL.Scheme == allowedURL.Scheme &&
            strings.EqualFold(originURL.Host, allowedURL.Host)
    }

    // 2. 通配符匹配（*.example.com）
    if strings.HasPrefix(allowed, "*.") {
        domain := strings.TrimPrefix(allowed, "*.")
        hostname := strings.ToLower(originURL.Hostname())
        return strings.HasSuffix(hostname, "."+domain) || hostname == domain
    }

    // 3. 精确主机名匹配
    return strings.EqualFold(originURL.Hostname(), allowed)
}
```

**改进**：
- 支持完整 URL 匹配（验证 scheme 和端口）
- 支持通配符子域名匹配
- 支持精确主机名匹配
- 大小写不敏感

### 3. 环境感知的验证策略

```go
func checkWebSocketOrigin(r *http.Request) bool {
    origin := r.Header.Get("Origin")

    // 从 Referer 提取 origin（如果 Origin 头缺失）
    if origin == "" {
        referer := r.Header.Get("Referer")
        if referer != "" {
            if parsedURL, err := url.Parse(referer); err == nil {
                origin = fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
            }
        }
    }

    isDevelopment := cfg.Server.Mode != "release"

    // 开发模式：允许空 Origin 和 localhost
    if origin == "" {
        if isDevelopment {
            return true
        }
        return false // 生产模式拒绝空 Origin
    }

    // 解析并验证 Origin
    originURL, err := url.Parse(origin)
    if err != nil {
        return false
    }

    // 开发模式：允许 localhost
    if isDevelopment && isLocalhost(originURL.Hostname()) {
        return true
    }

    // 检查配置的允许来源
    for _, allowed := range cfg.Server.AllowedOrigins {
        if matchOrigin(originURL, allowed) {
            return true
        }
    }

    return false
}
```

**改进**：
- 区分开发和生产环境
- 生产环境强制验证 Origin
- 支持从 Referer 提取 Origin
- 完整的错误处理

## 配置示例

### 开发环境

```yaml
server:
  mode: "debug"
  allowed_origins:
    - "localhost"
    - "127.0.0.1"
```

**行为**：
- 自动允许 localhost 和 127.0.0.1
- 允许空 Origin（方便调试）

### 生产环境

```yaml
server:
  mode: "release"
  allowed_origins:
    - "https://panel.example.com"      # 精确匹配
    - "panel.example.com"               # 主机名匹配（任意 scheme）
    - "*.example.com"                   # 通配符匹配所有子域名
```

**行为**：
- 拒绝空 Origin
- 拒绝 localhost（除非明确配置）
- 仅允许配置的来源

## 支持的配置格式

### 1. 完整 URL（推荐用于生产环境）

```yaml
allowed_origins:
  - "https://panel.example.com"
  - "http://localhost:5173"
```

**匹配规则**：
- 精确匹配 scheme（http/https）
- 精确匹配 host（包含端口）
- ✅ `https://panel.example.com` → 允许
- ❌ `http://panel.example.com` → 拒绝（scheme 不匹配）
- ❌ `https://panel.example.com:8080` → 拒绝（端口不匹配）

### 2. 主机名

```yaml
allowed_origins:
  - "panel.example.com"
```

**匹配规则**：
- 匹配任意 scheme
- 匹配任意端口
- ✅ `https://panel.example.com` → 允许
- ✅ `http://panel.example.com` → 允许
- ✅ `https://panel.example.com:8080` → 允许
- ❌ `https://sub.panel.example.com` → 拒绝

### 3. 通配符

```yaml
allowed_origins:
  - "*.example.com"
```

**匹配规则**：
- 匹配所有子域名
- 也匹配根域名
- ✅ `https://sub.example.com` → 允许
- ✅ `https://api.example.com` → 允许
- ✅ `https://example.com` → 允许
- ❌ `https://notexample.com` → 拒绝
- ❌ `https://example.com.evil.com` → 拒绝

## 测试覆盖

新增测试文件 `origin_test.go`，包含：

### 1. localhost 检测测试
- ✅ 精确匹配 localhost、127.0.0.1、::1
- ✅ 大小写不敏感
- ✅ 拒绝 `localhost.evil.com`
- ✅ 拒绝 `127.0.0.1.evil.com`

### 2. Origin 匹配测试
- ✅ 完整 URL 精确匹配
- ✅ scheme 验证
- ✅ 端口验证
- ✅ 主机名匹配
- ✅ 通配符匹配
- ✅ 大小写不敏感

### 3. 完整验证流程测试
- ✅ 开发模式行为
- ✅ 生产模式行为
- ✅ 从 Referer 提取 Origin
- ✅ 无效 Origin 处理

运行测试：
```bash
cd backend
go test -v ./internal/websocket
```

## 安全性提升

### 修复前的风险

| 攻击场景 | 原实现 | 风险等级 |
|---------|--------|---------|
| `evil.com/localhost` | ✅ 允许 | 🔴 高 |
| `127.0.0.1.evil.com` | ✅ 允许 | 🔴 高 |
| 空 Origin（生产环境） | ✅ 允许 | 🟡 中 |
| `evil-example.com` 匹配 `example.com` | ✅ 允许 | 🔴 高 |

### 修复后的防护

| 攻击场景 | 新实现 | 安全性 |
|---------|--------|--------|
| `evil.com/localhost` | ❌ 拒绝 | ✅ 安全 |
| `127.0.0.1.evil.com` | ❌ 拒绝 | ✅ 安全 |
| 空 Origin（生产环境） | ❌ 拒绝 | ✅ 安全 |
| `evil-example.com` 匹配 `example.com` | ❌ 拒绝 | ✅ 安全 |
| scheme 不匹配 | ❌ 拒绝 | ✅ 安全 |
| 端口不匹配 | ❌ 拒绝 | ✅ 安全 |

## 迁移指南

### 对现有部署的影响

**开发环境**：
- ✅ 无影响，行为保持一致
- localhost 和 127.0.0.1 仍然允许

**生产环境**：
- ⚠️ 需要明确配置 `allowed_origins`
- 如果之前依赖宽松的验证，需要更新配置

### 升级步骤

1. **更新代码**
   ```bash
   git pull
   ```

2. **检查配置**

   编辑 `backend/configs/config.yaml`：
   ```yaml
   server:
     mode: "release"
     allowed_origins:
       - "https://your-panel-domain.com"
       - "*.your-domain.com"  # 如果需要支持子域名
   ```

3. **测试验证**

   启动服务后，检查 WebSocket 连接：
   ```bash
   # 查看日志，确认没有 "WebSocket 连接被拒绝" 警告
   docker compose logs -f backend
   ```

4. **监控告警**

   如果看到拒绝日志，检查：
   - Origin 是否在允许列表中
   - 配置格式是否正确
   - 是否需要添加通配符支持

## 相关文件

- `backend/internal/websocket/hub.go` - WebSocket Hub 实现
- `backend/internal/websocket/origin_test.go` - Origin 验证测试
- `backend/configs/config.yaml` - 配置示例
- `docs/websocket-origin-fix.md` - 本文档

## 参考资料

- [WebSocket Security](https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API/Writing_WebSocket_servers#security)
- [OWASP WebSocket Security](https://owasp.org/www-community/vulnerabilities/WebSocket_Security)
- [RFC 6455 - The WebSocket Protocol](https://tools.ietf.org/html/rfc6455)
