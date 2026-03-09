# 升级迁移指南

本指南帮助现有 NodePass-Pro 用户升级到包含所有安全修复的新版本。

## ⚠️ 升级前准备

### 1. 备份数据

**备份数据库**：
```bash
# SQLite
cp data/nodepass.db data/nodepass.db.backup

# PostgreSQL
pg_dump -U nodepass nodepass > backup_$(date +%Y%m%d).sql

# MySQL
mysqldump -u nodepass -p nodepass > backup_$(date +%Y%m%d).sql
```

**备份配置**：
```bash
cp configs/config.yaml configs/config.yaml.backup
```

### 2. 记录当前版本

```bash
# 查看当前运行的版本
curl http://localhost:8080/health | jq .data.version
```

## 🔄 升级步骤

### 步骤 1: 停止服务

```bash
# 如果使用 systemd
sudo systemctl stop nodepass-backend

# 如果直接运行
# 按 Ctrl+C 或发送 SIGTERM 信号
kill -TERM $(pgrep nodepass-server)
```

### 步骤 2: 更新代码

```bash
# 拉取最新代码
git pull origin main

# 或者下载最新版本
# wget https://github.com/your-repo/nodepass-pro/archive/refs/tags/vX.X.X.tar.gz
```

### 步骤 3: 更新依赖

```bash
go mod tidy
go mod download
```

### 步骤 4: 更新配置文件

**重要**：不要直接覆盖 `config.yaml`，而是手动合并新配置。

```bash
# 查看新增的配置项
diff configs/config.yaml config.example.yaml
```

**需要添加的新配置项**：

```yaml
server:
  strict_csrf: false  # 生产环境建议设置为 true

database:
  # 新增：连接池配置（可选，有默认值）
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600
  conn_max_idle_time: 600
```

**MySQL 用户需要更新 SSL 配置**：
```yaml
database:
  ssl_mode: "true"  # 新增：启用 TLS
```

### 步骤 5: 验证配置

```bash
# 运行验证脚本
./verify-fixes.sh
```

确保：
- JWT secret 已设置且长度 >= 32 字符
- 数据库配置正确
- Redis 配置正确（建议启用）

### 步骤 6: 编译新版本

```bash
go build -o nodepass-server cmd/server/main.go
```

### 步骤 7: 数据库迁移

**自动迁移**（推荐）：
```bash
# 启动服务时会自动执行 AutoMigrate
# 无需手动操作
```

**手动检查**（可选）：
```bash
# 连接数据库查看表结构
# 确认没有丢失数据
```

### 步骤 8: 启动服务

```bash
# 直接运行
./nodepass-server

# 或使用 systemd
sudo systemctl start nodepass-backend
```

### 步骤 9: 验证升级

```bash
# 1. 检查健康状态
curl http://localhost:8080/health

# 2. 检查新端点
curl http://localhost:8080/readiness
curl http://localhost:8080/liveness

# 3. 测试登录
curl -X POST http://localhost:8080/api/v1/auth/login/v2 \
  -H "Content-Type: application/json" \
  -d '{
    "account": "your-email@example.com",
    "password": "your-password"
  }'

# 4. 检查日志中的 request_id
tail -f logs/app.log | grep request_id
```

## 🔍 验证修复

### 1. 管理员密码修复

**测试**：创建或更新管理员账号
```bash
go run cmd/admin-bootstrap/main.go \
  -username testadmin \
  -email test@example.com \
  -password "Test123!"

# 尝试登录
curl -X POST http://localhost:8080/api/v1/auth/login/v2 \
  -H "Content-Type: application/json" \
  -d '{
    "account": "test@example.com",
    "password": "Test123!"
  }'
```

**预期结果**：登录成功，返回 access_token 和 refresh_token

### 2. 验证码功能

**测试**：发送邮箱修改验证码
```bash
# 先登录获取 token
TOKEN="your-access-token"

# 发送验证码
curl -X POST http://localhost:8080/api/v1/auth/email/code \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "password": "your-password",
    "new_email": "new@example.com"
  }'
```

**预期结果**：
- Redis 启用：验证码存储在 Redis 中
- Redis 禁用：验证码存储在数据库中（降级）
- 返回 `debug_code`（开发环境）或发送邮件（生产环境）

### 3. 请求 ID 追踪

**测试**：查看响应头
```bash
curl -v http://localhost:8080/api/v1/ping
```

**预期结果**：响应头包含 `X-Request-ID`
```
< X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
```

### 4. 健康检查

**测试**：访问健康检查端点
```bash
curl http://localhost:8080/health | jq
```

**预期结果**：
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z",
  "checks": {
    "database": {
      "status": "healthy"
    },
    "redis": {
      "status": "healthy"
    }
  }
}
```

## 🐛 常见升级问题

### 问题 1: 编译错误 - uuid 包未找到

**解决**：
```bash
go get github.com/google/uuid
go mod tidy
```

### 问题 2: 管理员无法登录

**原因**：可能是旧版本创建的管理员账号密码被双重哈希

**解决**：重新设置管理员密码
```bash
go run cmd/admin-bootstrap/main.go \
  -username admin \
  -email admin@example.com \
  -password "NewPassword123!"
```

### 问题 3: CSRF 验证失败

**原因**：新版本修复了 CSRF Cookie 的 HttpOnly 设置

**解决**：
1. 清除浏览器 Cookie
2. 重新登录
3. 确保前端从 Cookie 读取 CSRF 令牌并放入请求头

### 问题 4: 验证码无法使用

**原因**：验证码服务改为优先使用 Redis

**解决**：
1. 启用 Redis（推荐）
2. 或者依赖数据库降级方案（自动）

### 问题 5: 数据库连接池配置不生效

**原因**：配置文件未更新

**解决**：在 `config.yaml` 中添加：
```yaml
database:
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600
  conn_max_idle_time: 600
```

### 问题 6: MySQL TLS 连接失败

**原因**：新版本默认启用 TLS

**解决**：
```yaml
database:
  ssl_mode: "true"  # 或 "false" 禁用（不推荐）
```

如果 MySQL 服务器不支持 TLS：
```yaml
database:
  ssl_mode: "false"
```

## 📊 性能影响

### 预期改进

1. **验证码性能**：使用 Redis 后，验证码操作速度提升 10-100 倍
2. **请求追踪**：Request ID 开销可忽略（< 1ms）
3. **连接池优化**：可根据负载调整，提升并发能力

### 监控指标

升级后监控以下指标：
- 数据库连接数（应该更稳定）
- Redis 命中率（验证码、CSRF 令牌）
- 请求响应时间（应该略有改善）
- 错误率（应该降低）

## 🔄 回滚计划

如果升级后出现问题，可以回滚：

### 1. 停止新版本服务

```bash
sudo systemctl stop nodepass-backend
# 或 kill 进程
```

### 2. 恢复旧版本

```bash
# 恢复代码
git checkout <previous-version-tag>

# 恢复配置
cp configs/config.yaml.backup configs/config.yaml

# 重新编译
go build -o nodepass-server cmd/server/main.go
```

### 3. 恢复数据库（如果需要）

```bash
# PostgreSQL
psql -U nodepass nodepass < backup_YYYYMMDD.sql

# MySQL
mysql -u nodepass -p nodepass < backup_YYYYMMDD.sql

# SQLite
cp data/nodepass.db.backup data/nodepass.db
```

### 4. 启动旧版本

```bash
./nodepass-server
```

## ✅ 升级检查清单

升级完成后，确认以下项目：

- [ ] 服务正常启动
- [ ] 健康检查端点返回正常
- [ ] 用户可以正常登录
- [ ] 管理员可以正常登录
- [ ] 验证码功能正常
- [ ] CSRF 保护正常工作
- [ ] 日志包含 request_id
- [ ] 数据库连接稳定
- [ ] Redis 连接正常（如果启用）
- [ ] 所有 API 端点正常响应
- [ ] WebSocket 连接正常
- [ ] 定时任务正常运行

## 📞 获取支持

如果升级过程中遇到问题：

1. 查看 `CODE_REVIEW_FIXES.md` 了解所有修复
2. 运行 `./verify-fixes.sh` 检查配置
3. 查看服务日志
4. 在 GitHub 提交 Issue
5. 联系技术支持

## 🎉 升级完成

恭喜！你已经成功升级到包含所有安全修复的新版本。

新版本包含：
- ✅ 3 个严重问题修复
- ✅ 3 个安全问题修复
- ✅ 3 个代码质量改进
- ✅ 3 个新功能

享受更安全、更可靠的 NodePass-Pro！
