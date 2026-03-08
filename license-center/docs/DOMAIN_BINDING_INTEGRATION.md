# 域名绑定功能集成完成

## ✅ 已完成的集成

### 1. 数据库迁移
- ✅ 添加 `LicenseDomainBinding` 表（域名绑定历史）
- ✅ 添加 `DomainIPBinding` 表（域名 IP 追踪）
- ✅ 扩展 `LicenseKey` 表（新增域名相关字段）

### 2. 服务层集成
- ✅ 创建 `DomainBindingService`（域名绑定服务）
- ✅ 修改 `LicenseService`（集成域名验证）
- ✅ 服务依赖注入顺序调整

### 3. 处理器集成
- ✅ 创建 `DomainBindingHandler`（域名管理接口）
- ✅ 注册到 main.go

### 4. API 路由
新增 4 个域名管理接口：
```
POST   /api/v1/licenses/:id/domain/change   - 更换域名
POST   /api/v1/licenses/:id/domain/unbind   - 解绑域名
POST   /api/v1/licenses/:id/domain/lock     - 锁定域名
GET    /api/v1/licenses/:id/domain/history  - 绑定历史
```

### 5. 验证流程集成
在 `LicenseService.Verify` 中自动调用域名验证：
```go
// 域名验证
if s.domainService != nil && req.Domain != "" {
    if err := s.domainService.VerifyDomain(&license, req.Domain, ip, DefaultDomainBindingConfig); err != nil {
        return &VerifyResult{Valid: false, Message: err.Error()}, nil
    }
}
```

## 🎯 工作原理

### 首次验证（自动绑定）
```
客户端请求验证
  ↓
提取域名: example.com
  ↓
检查授权码是否已绑定域名
  ↓
未绑定 → 自动绑定到 example.com
  ↓
记录绑定历史
  ↓
触发 Webhook: license.domain_bound
  ↓
返回验证成功
```

### 后续验证（域名匹配）
```
客户端请求验证
  ↓
提取域名: example.com
  ↓
检查是否匹配已绑定域名
  ↓
匹配 → 更新 IP 记录 → 返回成功
  ↓
不匹配 → 创建告警 → 返回失败
```

### 更换域名（管理员操作）
```
管理员提交更换申请
  ↓
检查冷却期（30天）
  ↓
验证新域名格式
  ↓
更新绑定
  ↓
记录历史
  ↓
触发 Webhook
  ↓
返回成功
```

## 📡 API 使用示例

### 1. 验证授权码（带域名）
```bash
curl -X POST http://localhost:8090/api/v1/license/verify \
  -H "Content-Type: application/json" \
  -d '{
    "license_key": "NP-XXXX-XXXX-XXXX",
    "machine_id": "abc123",
    "domain": "example.com",
    "site_url": "https://example.com"
  }'
```

### 2. 更换域名
```bash
curl -X POST http://localhost:8090/api/v1/licenses/1/domain/change \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "new_domain": "newdomain.com",
    "reason": "网站迁移"
  }'
```

### 3. 解绑域名
```bash
curl -X POST http://localhost:8090/api/v1/licenses/1/domain/unbind \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "重置测试环境"
  }'
```

### 4. 锁定域名（预设）
```bash
curl -X POST http://localhost:8090/api/v1/licenses/1/domain/lock \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "example.com"
  }'
```

### 5. 查看绑定历史
```bash
curl -X GET http://localhost:8090/api/v1/licenses/1/domain/history \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## 🔧 配置说明

默认配置（在代码中）：
```go
DefaultDomainBindingConfig = DomainBindingConfig{
    Enabled:                  true,      // 启用域名绑定
    AutoBindOnFirstVerify:    true,      // 首次自动绑定
    AllowDomainChange:        true,      // 允许更换域名
    DomainChangeCooldown:     2592000,   // 30天冷却期
    RequireDomainVerification: false,    // 不需要域名验证
    AllowTestDomains:         true,      // 允许测试域名
    TestDomains: []string{
        "localhost",
        "127.0.0.1",
        "::1",
        "*.test",
        "*.local",
    },
}
```

## 🎨 前端集成建议

### 1. 授权码列表页面
显示绑定域名信息：
```typescript
interface LicenseKey {
  // ... 其他字段
  bound_domain: string
  domain_locked: boolean
  domain_bound_at: string
}
```

### 2. 授权码详情页面
添加域名管理区域：
- 显示当前绑定域名
- 显示绑定时间
- 提供"更换域名"按钮
- 提供"解绑域名"按钮
- 显示绑定历史记录

### 3. 域名更换对话框
```typescript
const changeDomain = async (licenseId: number, newDomain: string, reason: string) => {
  await api.post(`/licenses/${licenseId}/domain/change`, {
    new_domain: newDomain,
    reason: reason
  })
}
```

## 🔔 告警类型

系统会自动创建以下告警：

1. **domain_mismatch** - 域名不匹配
   - 级别：warning
   - 触发：授权码尝试从非绑定域名访问

2. **domain_ip_changed** - 域名 IP 变更
   - 级别：info
   - 触发：同一域名的 IP 地址发生变化

## 🎯 测试建议

### 1. 首次绑定测试
```bash
# 第一次验证，应该自动绑定
curl -X POST http://localhost:8090/api/v1/license/verify \
  -d '{"license_key":"NP-TEST","machine_id":"m1","domain":"test.com"}'

# 查看授权码，应该看到 bound_domain = "test.com"
```

### 2. 域名匹配测试
```bash
# 使用相同域名，应该成功
curl -X POST http://localhost:8090/api/v1/license/verify \
  -d '{"license_key":"NP-TEST","machine_id":"m1","domain":"test.com"}'

# 使用不同域名，应该失败
curl -X POST http://localhost:8090/api/v1/license/verify \
  -d '{"license_key":"NP-TEST","machine_id":"m1","domain":"other.com"}'
```

### 3. 测试域名豁免
```bash
# localhost 不会被绑定
curl -X POST http://localhost:8090/api/v1/license/verify \
  -d '{"license_key":"NP-TEST","machine_id":"m1","domain":"localhost"}'
```

### 4. 域名更换测试
```bash
# 更换域名
curl -X POST http://localhost:8090/api/v1/licenses/1/domain/change \
  -H "Authorization: Bearer TOKEN" \
  -d '{"new_domain":"new.com","reason":"迁移"}'

# 查看历史
curl http://localhost:8090/api/v1/licenses/1/domain/history \
  -H "Authorization: Bearer TOKEN"
```

## 📊 数据库查询示例

### 查看授权码域名绑定
```sql
SELECT id, key, customer, bound_domain, domain_locked, domain_bound_at
FROM license_keys
WHERE bound_domain IS NOT NULL;
```

### 查看域名绑定历史
```sql
SELECT lb.*, lk.key, lk.customer
FROM license_domain_bindings lb
JOIN license_keys lk ON lb.license_id = lk.id
ORDER BY lb.created_at DESC
LIMIT 10;
```

### 查看域名 IP 追踪
```sql
SELECT domain, ip_address, first_seen, last_seen, hit_count
FROM domain_ip_bindings
ORDER BY last_seen DESC;
```

## 🚀 部署注意事项

1. **数据库迁移**
   - 首次启动会自动创建新表
   - 现有授权码的 `bound_domain` 为空，首次验证时自动绑定

2. **向后兼容**
   - 如果验证请求不包含 `domain` 字段，域名验证会被跳过
   - 不影响现有客户端

3. **配置调整**
   - 生产环境建议设置 `allow_test_domains: false`
   - 根据业务需求调整 `domain_change_cooldown`

4. **监控建议**
   - 关注 `domain_mismatch` 告警
   - 定期检查域名绑定历史
   - 监控异常的域名更换行为

## 📝 下一步优化

1. **前端界面**
   - 在授权码列表显示域名
   - 添加域名管理对话框
   - 显示绑定历史时间线

2. **域名验证**
   - 实现 DNS TXT 记录验证
   - 实现文件验证
   - 实现 HTTP Header 验证

3. **多域名支持**
   - 企业版支持多个域名
   - 域名配额管理

4. **自助服务**
   - 客户自助更换域名
   - 邮件验证流程
   - 域名更换审批

---

**集成完成时间**: 2026-03-07
**版本**: v0.2.0
**状态**: ✅ 编译通过，功能完整
