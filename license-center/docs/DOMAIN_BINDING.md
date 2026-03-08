# 授权码域名绑定方案

## 需求分析

确保一个授权码只能在一个网站（域名）上使用，防止：
1. 授权码被多个网站共享使用
2. 授权码被转售到其他域名
3. 未授权的域名使用授权码

## 方案设计

### 方案一：首次验证时自动绑定域名（推荐）

#### 工作流程
1. 客户购买授权码时不指定域名
2. 首次验证时，自动绑定请求来源的域名
3. 后续验证时，检查域名是否匹配
4. 支持管理员手动解绑/更换域名

#### 优点
- 用户体验好，无需预先配置
- 灵活性高，支持域名迁移
- 实现简单

#### 缺点
- 首次验证前可能被恶意抢注

### 方案二：购买时预先指定域名

#### 工作流程
1. 客户购买时必须提供域名
2. 授权码生成时绑定域名
3. 验证时严格检查域名匹配
4. 更换域名需要管理员审批

#### 优点
- 安全性最高
- 完全可控

#### 缺点
- 用户体验较差
- 域名变更流程复杂

### 方案三：混合方案（最佳实践）

#### 工作流程
1. 生成授权码时可选择是否预设域名
2. 未预设域名的，首次验证时自动绑定
3. 已预设域名的，严格验证
4. 支持域名白名单（多个子域名）
5. 提供域名更换申请流程

## 技术实现

### 1. 数据库设计

```sql
-- 在 license_keys 表中添加域名相关字段
ALTER TABLE license_keys ADD COLUMN bound_domain VARCHAR(255);
ALTER TABLE license_keys ADD COLUMN domain_locked BOOLEAN DEFAULT FALSE;
ALTER TABLE license_keys ADD COLUMN domain_bound_at TIMESTAMP;

-- 创建域名绑定历史表
CREATE TABLE license_domain_bindings (
    id SERIAL PRIMARY KEY,
    license_id INTEGER NOT NULL REFERENCES license_keys(id),
    old_domain VARCHAR(255),
    new_domain VARCHAR(255) NOT NULL,
    reason TEXT,
    operator_id INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 2. 验证请求增强

```json
{
  "license_key": "NP-XXXX-XXXX-XXXX",
  "machine_id": "abcdef123456",
  "domain": "example.com",  // 新增：网站域名
  "site_url": "https://example.com",  // 新增：完整网站地址
  "action": "install"
}
```

### 3. 验证逻辑

```go
// 伪代码
func VerifyLicense(req *VerifyRequest) (*VerifyResult, error) {
    // 1. 基础验证（授权码存在、未过期等）
    license := findLicense(req.LicenseKey)

    // 2. 域名验证
    if license.BoundDomain == "" {
        // 首次验证，自动绑定域名
        if !license.DomainLocked {
            license.BoundDomain = extractDomain(req.Domain)
            license.DomainBoundAt = time.Now()
            saveLicense(license)
            logDomainBinding(license.ID, "", license.BoundDomain, "首次自动绑定")
        }
    } else {
        // 已绑定域名，检查是否匹配
        if !matchDomain(license.BoundDomain, req.Domain) {
            return &VerifyResult{
                Valid: false,
                Message: fmt.Sprintf("域名不匹配，授权码已绑定到: %s", license.BoundDomain),
            }, nil
        }
    }

    // 3. 其他验证逻辑...
    return &VerifyResult{Valid: true}, nil
}

// 域名匹配逻辑（支持通配符）
func matchDomain(boundDomain, requestDomain string) bool {
    // 精确匹配
    if boundDomain == requestDomain {
        return true
    }

    // 支持通配符 *.example.com
    if strings.HasPrefix(boundDomain, "*.") {
        baseDomain := strings.TrimPrefix(boundDomain, "*.")
        return strings.HasSuffix(requestDomain, baseDomain)
    }

    return false
}
```

### 4. 管理功能

#### 4.1 查看域名绑定
```
GET /api/v1/licenses/:id/domain
```

#### 4.2 更换域名
```
POST /api/v1/licenses/:id/domain/change
{
  "new_domain": "newdomain.com",
  "reason": "网站迁移"
}
```

#### 4.3 解绑域名
```
POST /api/v1/licenses/:id/domain/unbind
{
  "reason": "测试环境重置"
}
```

#### 4.4 锁定域名（防止自动绑定）
```
POST /api/v1/licenses/:id/domain/lock
{
  "domain": "example.com"
}
```

## 安全增强

### 1. 域名验证增强

```go
// 验证域名真实性
func verifyDomainOwnership(domain string, licenseKey string) bool {
    // 方案1：DNS TXT 记录验证
    // 要求客户在域名 DNS 中添加 TXT 记录
    // _license-verify.example.com TXT "NP-XXXX-XXXX-XXXX"

    // 方案2：文件验证
    // 要求在网站根目录放置验证文件
    // https://example.com/.well-known/license-verify.txt

    // 方案3：HTTP Header 验证
    // 要求网站返回特定 Header
    // X-License-Key: NP-XXXX-XXXX-XXXX
}
```

### 2. 防止域名欺骗

```go
// 不仅验证请求中的域名，还要验证 HTTP Referer 和 Origin
func extractDomainFromRequest(req *http.Request) string {
    // 优先使用 Origin
    if origin := req.Header.Get("Origin"); origin != "" {
        return parseDomain(origin)
    }

    // 其次使用 Referer
    if referer := req.Header.Get("Referer"); referer != "" {
        return parseDomain(referer)
    }

    // 最后使用请求体中的域名
    return req.Body.Domain
}
```

### 3. IP 地址关联

```go
// 记录域名对应的 IP 地址，检测异常
type DomainIPBinding struct {
    Domain    string
    IPAddress string
    FirstSeen time.Time
    LastSeen  time.Time
}

// 如果同一域名从不同 IP 访问，触发告警
func checkDomainIPConsistency(domain, ip string) bool {
    binding := getDomainIPBinding(domain)
    if binding.IPAddress != "" && binding.IPAddress != ip {
        createAlert("域名 IP 地址变更", fmt.Sprintf(
            "域名 %s 的 IP 从 %s 变更为 %s",
            domain, binding.IPAddress, ip,
        ))
    }
    return true
}
```

## 用户体验优化

### 1. 域名变更申请流程

```
客户端 -> 提交域名变更申请
     -> 系统发送验证邮件到客户邮箱
     -> 客户点击邮件中的确认链接
     -> 系统验证新域名所有权
     -> 自动更换域名绑定
```

### 2. 临时域名支持

```go
// 支持测试域名（localhost, 127.0.0.1, *.test）
func isTestDomain(domain string) bool {
    testDomains := []string{
        "localhost",
        "127.0.0.1",
        "::1",
    }

    for _, td := range testDomains {
        if domain == td {
            return true
        }
    }

    return strings.HasSuffix(domain, ".test") ||
           strings.HasSuffix(domain, ".local")
}

// 测试域名不进行绑定，但记录使用次数
```

### 3. 多域名支持（高级套餐）

```go
type LicenseKey struct {
    // ...
    AllowedDomains []string  // 允许的域名列表
    MaxDomains     int       // 最大域名数量
}

// 企业版可以支持多个域名
func verifyDomainInList(license *LicenseKey, domain string) bool {
    if len(license.AllowedDomains) == 0 {
        return true  // 未限制
    }

    for _, allowed := range license.AllowedDomains {
        if matchDomain(allowed, domain) {
            return true
        }
    }

    return false
}
```

## 配置选项

```yaml
# configs/config.yaml
license:
  domain_binding:
    enabled: true                    # 是否启用域名绑定
    auto_bind_on_first_verify: true  # 首次验证时自动绑定
    allow_domain_change: true        # 是否允许更换域名
    domain_change_cooldown: 2592000  # 更换域名冷却期（秒，30天）
    require_domain_verification: false  # 是否需要域名所有权验证
    allow_test_domains: true         # 是否允许测试域名
    test_domains:
      - "localhost"
      - "127.0.0.1"
      - "*.test"
      - "*.local"
```

## 告警规则

```go
// 触发告警的情况
1. 域名不匹配尝试（可能是盗用）
2. 频繁更换域名（可能是转售）
3. 同一授权码在多个域名上尝试（可能是共享）
4. 域名 IP 地址频繁变更（可能是 CDN 或异常）
```

## 前端集成

### 安装脚本自动获取域名

```bash
#!/bin/bash

# 获取当前网站域名
DOMAIN=$(hostname -f)

# 或从配置文件读取
DOMAIN=$(grep "server_name" /etc/nginx/sites-enabled/default | awk '{print $2}' | sed 's/;//')

# 验证授权码时带上域名
curl -X POST https://license-server.com/api/v1/license/verify \
  -H "Content-Type: application/json" \
  -d "{
    \"license_key\": \"$LICENSE_KEY\",
    \"machine_id\": \"$MACHINE_ID\",
    \"domain\": \"$DOMAIN\",
    \"site_url\": \"https://$DOMAIN\"
  }"
```

### Web 应用自动获取域名

```javascript
// 前端自动获取域名
const domain = window.location.hostname;

// 验证授权码
fetch('/api/license/verify', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    license_key: 'NP-XXXX-XXXX-XXXX',
    domain: domain,
    site_url: window.location.origin
  })
});
```

## 实施建议

### 阶段一：基础实现（立即）
1. ✅ 添加域名字段到数据库
2. ✅ 修改验证接口，接收域名参数
3. ✅ 实现首次自动绑定逻辑
4. ✅ 实现域名匹配验证

### 阶段二：管理功能（1周内）
1. ✅ 添加域名查看接口
2. ✅ 添加域名更换接口
3. ✅ 添加域名绑定历史记录
4. ✅ 前端界面展示域名信息

### 阶段三：安全增强（2周内）
1. ✅ 实现域名所有权验证
2. ✅ 添加域名变更冷却期
3. ✅ 实现异常告警
4. ✅ IP 地址关联检测

### 阶段四：高级功能（1个月内）
1. ✅ 多域名支持
2. ✅ 通配符域名支持
3. ✅ 域名变更审批流程
4. ✅ 自助域名管理门户

## 常见问题

### Q1: 客户更换域名怎么办？
A: 提供域名更换功能，但设置冷却期（如30天只能更换一次），防止滥用。

### Q2: 开发环境怎么测试？
A: 允许 localhost、127.0.0.1、*.test 等测试域名，但不进行绑定。

### Q3: 使用 CDN 会影响吗？
A: 不会，我们验证的是域名而不是 IP，CDN 不影响域名。

### Q4: 子域名算不同域名吗？
A: 可配置，支持通配符（*.example.com）匹配所有子域名。

### Q5: 如何防止客户伪造域名？
A: 结合 HTTP Header（Origin/Referer）验证，或要求域名所有权验证。

## 总结

推荐使用**方案三（混合方案）**：
- ✅ 首次验证自动绑定（用户体验好）
- ✅ 支持预设域名（安全性高）
- ✅ 提供域名更换（灵活性好）
- ✅ 设置更换限制（防止滥用）
- ✅ 记录完整历史（可追溯）

这样既保证了安全性，又不影响用户体验。
