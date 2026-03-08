# NodePass License Center - 增强版

独立授权管理系统（独立仓可部署），用于 NodePass Pro 的授权校验与授权码管理。

## 新增功能

### 🔒 安全增强
- **API 限流**：基于 IP 的请求频率限制，防止暴力攻击
- **请求签名验证**：HMAC-SHA256 签名机制，防止请求篡改
- **IP 白名单**：支持精确 IP 和 CIDR 网段配置
- **防重放攻击**：基于 Nonce 的请求去重机制

### 📊 监控告警
- **实时监控面板**：授权码、机器绑定、验证请求等核心指标
- **验证趋势分析**：可视化展示验证成功率和趋势
- **自动告警**：授权码即将过期、配额超限等自动告警
- **Top 客户统计**：快速了解主要客户使用情况

### 🚀 功能扩展
- **授权码转移**：支持将授权码转移给其他客户，记录完整转移日志
- **批量操作**：批量吊销、恢复、删除授权码
- **标签管理**：为授权码添加标签，便于分类管理
- **自定义字段**：支持为授权码添加自定义元数据
- **Webhook 通知**：支持配置 Webhook，实时推送事件通知

### ⚡ 性能优化
- **Redis 缓存**：可选的 Redis 缓存支持，提升查询性能
- **数据库索引优化**：优化查询性能
- **并发控制**：合理的并发处理机制

## 快速部署

```bash
git clone <your-license-center-repo>
cd license-center
./scripts/deploy.sh
```

默认地址：`http://127.0.0.1:8090`

- 健康检查：`http://127.0.0.1:8090/health`
- Web 管理面板：`http://127.0.0.1:8090/console`

## 远程一键安装/升级/卸载

```bash
# 安装
bash <(curl -fsSL "https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license-center/install.sh?t=$(date +%s)") --install

# 升级
bash <(curl -fsSL "https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license-center/install.sh?t=$(date +%s)") --upgrade

# 卸载
bash <(curl -fsSL "https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license-center/install.sh?t=$(date +%s)") --uninstall
```

## 配置说明

### 基础配置

```yaml
server:
  port: "8090"
  mode: "release"

database:
  type: "postgres"
  host: "postgres"
  port: 5432
  user: "postgres"
  password: "postgres"
  db_name: "nodepass_license"

jwt:
  secret: "change-this-license-secret"
  expire_hours: 24

admin:
  username: "admin"
  email: "admin@license.local"
  password: "ChangeMe123!"
```

### Redis 缓存（可选）

```yaml
redis:
  enabled: true
  host: "localhost"
  port: 6379
  password: ""
  db: 0
  prefix: "license:"
```

### 安全配置

```yaml
security:
  # API 限流
  rate_limit:
    enabled: true
    requests_per_second: 10
    burst: 20

  # 请求签名验证
  signature:
    enabled: false
    secret: "change-this-signature-secret"
    time_window: 300

  # IP 白名单
  ip_whitelist:
    enabled: false
    allowed_ips:
      - "192.168.1.100"
    allowed_cidrs:
      - "10.0.0.0/8"
```

### 监控配置

```yaml
monitoring:
  # 告警配置
  alert:
    enabled: true
    check_interval: 3600  # 检查间隔（秒）
    expiring_days: 30     # 提前多少天告警

  # 日志清理
  cleanup:
    enabled: true
    verify_log_days: 90   # 验证日志保留天数
    webhook_log_days: 30  # Webhook 日志保留天数
    alert_days: 90        # 告警记录保留天数
```

## API 接口

### 新增接口

#### 监控与统计

```bash
# 获取仪表盘数据
GET /api/v1/dashboard

# 获取验证趋势（最近 N 天）
GET /api/v1/verify-trend?days=7

# 获取 Top 客户
GET /api/v1/top-customers?limit=10
```

#### 告警管理

```bash
# 查询告警
GET /api/v1/alerts?is_read=false&level=warning&page=1&page_size=20

# 标记告警已读
POST /api/v1/alerts/:id/read

# 标记所有告警已读
POST /api/v1/alerts/read-all

# 删除告警
DELETE /api/v1/alerts/:id

# 获取告警统计
GET /api/v1/alert-stats
```

#### 授权码扩展

```bash
# 转移授权码
POST /api/v1/licenses/:id/transfer
{
  "to_customer": "新客户名称",
  "reason": "转移原因"
}

# 批量吊销
POST /api/v1/licenses/batch/revoke
{
  "license_ids": [1, 2, 3]
}

# 批量恢复
POST /api/v1/licenses/batch/restore
{
  "license_ids": [1, 2, 3]
}

# 批量删除
POST /api/v1/licenses/batch/delete
{
  "license_ids": [1, 2, 3]
}
```

#### 标签管理

```bash
# 查询标签
GET /api/v1/tags

# 创建标签
POST /api/v1/tags
{
  "name": "VIP客户",
  "color": "#ff0000"
}

# 为授权码添加标签
POST /api/v1/licenses/:id/tags
{
  "tag_ids": [1, 2]
}

# 获取授权码的标签
GET /api/v1/licenses/:id/tags
```

#### Webhook 管理

```bash
# 查询 Webhook 配置
GET /api/v1/webhooks

# 创建 Webhook
POST /api/v1/webhooks
{
  "name": "通知服务",
  "url": "https://your-webhook-url.com/notify",
  "secret": "webhook-secret",
  "events": ["license.created", "license.expired", "alert.created"],
  "is_enabled": true
}

# 查询 Webhook 日志
GET /api/v1/webhook-logs?webhook_id=1&page=1&page_size=20
```

### Webhook 事件类型

- `license.created` - 授权码创建
- `license.expired` - 授权码过期
- `license.revoked` - 授权码吊销
- `license.transferred` - 授权码转移
- `alert.created` - 告警创建
- `*` - 所有事件

### Webhook 请求格式

```json
{
  "event": "license.expired",
  "timestamp": "2026-03-07T12:00:00Z",
  "data": {
    "license_id": 123,
    "license_key": "NP-XXXX-XXXX-XXXX",
    "customer": "客户名称"
  }
}
```

请求头包含签名：
```
X-Webhook-Signature: <HMAC-SHA256 签名>
```

## 默认管理员

- 用户名：`admin`
- 密码：`ChangeMe123!`

请首次登录后立即修改 `configs/config.yaml` 中的管理员密码与 JWT Secret。

## 技术栈

- **语言**：Go 1.24
- **框架**：Gin
- **数据库**：PostgreSQL / MySQL / SQLite
- **缓存**：Redis（可选）
- **认证**：JWT
- **限流**：golang.org/x/time/rate

## 与 NodePass Pro 对接

在 NodePass Pro 安装脚本中指定授权服务地址：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh) \
  --license-server https://license.yourdomain.com/api/v1/license/verify \
  --license-key NP-XXXX-XXXX-XXXX
```

## 版本历史

### v0.2.0 (2026-03-07)
- ✨ 新增 API 限流、请求签名验证、IP 白名单等安全功能
- ✨ 新增实时监控面板、告警系统
- ✨ 新增授权码转移、批量操作、标签管理
- ✨ 新增 Webhook 通知功能
- ✨ 新增 Redis 缓存支持
- 🔧 优化数据库查询性能
- 📝 完善 API 文档

### v0.1.0
- 🎉 初始版本
- ✅ 授权验证接口
- ✅ 管理员登录与 JWT 认证
- ✅ 套餐管理
- ✅ 授权码管理
- ✅ 机器绑定管理
- ✅ 验证日志与统计
