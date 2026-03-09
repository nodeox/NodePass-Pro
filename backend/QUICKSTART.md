# 快速启动指南

## 🚀 立即开始

### 1. 更新配置文件

如果你还没有 `configs/config.yaml`，从示例复制：
```bash
cp config.example.yaml configs/config.yaml
```

**必须修改的配置**：
```yaml
jwt:
  secret: "your-strong-random-secret-at-least-32-characters-long"
```

生成强随机密钥：
```bash
# 方法 1: 使用 openssl
openssl rand -base64 48

# 方法 2: 使用 Python
python3 -c "import secrets; print(secrets.token_urlsafe(48))"
```

### 2. 配置数据库

**SQLite（开发环境）**：
```yaml
database:
  type: "sqlite"
  dsn: "./data/nodepass.db"
```

**PostgreSQL（推荐生产环境）**：
```yaml
database:
  type: "postgres"
  host: "localhost"
  port: 5432
  user: "nodepass"
  password: "your_password"
  db_name: "nodepass"
  ssl_mode: "require"
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600
  conn_max_idle_time: 600
```

**MySQL**：
```yaml
database:
  type: "mysql"
  host: "localhost"
  port: 3306
  user: "nodepass"
  password: "your_password"
  db_name: "nodepass"
  ssl_mode: "true"  # 启用 TLS
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600
  conn_max_idle_time: 600
```

### 3. 配置 Redis（强烈推荐）

```yaml
redis:
  enabled: true
  addr: "127.0.0.1:6379"
  password: ""
  db: 0
```

### 4. 创建管理员账号

```bash
go run cmd/admin-bootstrap/main.go \
  -username admin \
  -email admin@example.com \
  -password "Admin123!"
```

### 5. 启动服务

```bash
go run cmd/server/main.go
```

或编译后运行：
```bash
go build -o nodepass-server cmd/server/main.go
./nodepass-server
```

### 6. 验证服务

```bash
# 健康检查
curl http://localhost:8080/health

# 就绪检查
curl http://localhost:8080/readiness

# 存活检查
curl http://localhost:8080/liveness

# 登录测试
curl -X POST http://localhost:8080/api/v1/auth/login/v2 \
  -H "Content-Type: application/json" \
  -d '{
    "account": "admin@example.com",
    "password": "Admin123!"
  }'
```

## 🔧 生产环境配置

### 安全配置

```yaml
server:
  mode: "release"
  strict_csrf: true
  trust_forwarded_headers: true  # 如果使用反向代理

jwt:
  secret: "使用至少 48 字符的强随机字符串"
  expire_time: 24

database:
  ssl_mode: "require"  # PostgreSQL
  # ssl_mode: "true"   # MySQL
```

### 性能优化

```yaml
database:
  max_idle_conns: 20      # 根据负载调整
  max_open_conns: 200     # 根据负载调整
  conn_max_lifetime: 3600
  conn_max_idle_time: 600

redis:
  enabled: true  # 生产环境必须启用
```

## 📊 监控和日志

### 查看日志

服务使用 zap 日志库，所有日志都包含 `request_id` 用于追踪：

```json
{
  "level": "info",
  "ts": 1234567890.123,
  "msg": "HTTP 请求",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/api/v1/auth/me",
  "status": 200,
  "latency": "5ms",
  "ip": "127.0.0.1"
}
```

### 健康检查端点

- `/health` - 综合健康检查（数据库 + Redis）
- `/readiness` - Kubernetes readiness probe
- `/liveness` - Kubernetes liveness probe

### Kubernetes 配置示例

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nodepass-backend
spec:
  containers:
  - name: backend
    image: nodepass-backend:latest
    ports:
    - containerPort: 8080
    livenessProbe:
      httpGet:
        path: /liveness
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /readiness
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
```

## 🐛 故障排查

### 问题：服务无法启动

**检查 JWT Secret**：
```bash
grep "jwt:" configs/config.yaml -A 2
```

确保 `secret` 不是默认值且长度 >= 32 字符。

### 问题：数据库连接失败

**PostgreSQL**：
```bash
# 测试连接
psql -h localhost -U nodepass -d nodepass

# 检查 SSL 模式
grep "ssl_mode" configs/config.yaml
```

**MySQL**：
```bash
# 测试连接
mysql -h localhost -u nodepass -p nodepass

# 检查 TLS 配置
grep "ssl_mode" configs/config.yaml
```

### 问题：Redis 连接失败

```bash
# 测试 Redis 连接
redis-cli -h 127.0.0.1 -p 6379 ping

# 如果 Redis 不可用，可以临时禁用
# 在 config.yaml 中设置：
redis:
  enabled: false
```

### 问题：验证码无法发送

1. 检查 Redis 是否启用（推荐）
2. 如果 Redis 未启用，验证码会存储在数据库中（降级方案）
3. 检查 SMTP 配置（如果需要邮件发送）

### 问题：CSRF 验证失败

1. 确保前端在请求头中包含 `X-CSRF-Token`
2. 确保前端从 Cookie 中读取 CSRF 令牌
3. 检查 `strict_csrf` 配置

## 📝 常见任务

### 重置管理员密码

```bash
go run cmd/admin-bootstrap/main.go \
  -username admin \
  -email admin@example.com \
  -password "NewPassword123!"
```

### 查看数据库连接池状态

在代码中添加监控端点或查看日志：
```
数据库连接池配置完成 max_idle_conns=10 max_open_conns=100
```

### 清理过期数据

服务会自动运行定时任务：
- 每月重置流量配额
- 每小时检查 VIP 过期
- 每 30 秒检查节点心跳超时
- 每天清理 90 天前的审计日志
- 每小时清理过期的 CSRF 令牌

## 🔐 安全最佳实践

1. **生产环境必须修改 JWT Secret**
2. **启用数据库 SSL/TLS 连接**
3. **启用 Redis 用于验证码和 CSRF 令牌**
4. **设置 `strict_csrf: true`**
5. **使用反向代理（Nginx/Caddy）并启用 HTTPS**
6. **定期备份数据库**
7. **监控日志中的异常请求**
8. **限制管理员账号数量**

## 📚 更多信息

- 详细修复说明：`CODE_REVIEW_FIXES.md`
- 配置示例：`config.example.yaml`
- 验证脚本：`./verify-fixes.sh`

## 🆘 获取帮助

如果遇到问题：
1. 查看日志输出
2. 运行 `./verify-fixes.sh` 检查配置
3. 查看 `CODE_REVIEW_FIXES.md` 了解所有修复
4. 检查 GitHub Issues
