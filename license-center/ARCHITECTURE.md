# NodePass License Center - 完整架构

## 🎉 项目概述

这是一个完整的现代化授权管理系统，包含：
- **后端服务**：Go + Gin + PostgreSQL
- **前端界面**：React + TypeScript + Ant Design + Vite

## 📦 项目结构

```
license-center/
├── cmd/
│   └── server/
│       └── main.go                 # 主程序入口
├── internal/
│   ├── api/                        # API 层
│   ├── cache/                      # Redis 缓存
│   │   └── redis.go
│   ├── config/                     # 配置管理
│   │   └── config.go
│   ├── database/                   # 数据库
│   │   └── db.go
│   ├── handlers/                   # HTTP 处理器
│   │   ├── auth_handler.go
│   │   ├── license_handler.go
│   │   ├── monitoring_handler.go
│   │   └── extension_handler.go
│   ├── middleware/                 # 中间件
│   │   ├── auth.go
│   │   ├── ratelimit.go
│   │   ├── signature.go
│   │   └── ipwhitelist.go
│   ├── models/                     # 数据模型
│   │   ├── admin_user.go
│   │   ├── license_key.go
│   │   ├── license_plan.go
│   │   ├── license_activation.go
│   │   ├── verify_log.go
│   │   └── extensions.go
│   ├── services/                   # 业务逻辑
│   │   ├── auth_service.go
│   │   ├── license_service.go
│   │   ├── webhook_service.go
│   │   ├── monitoring_service.go
│   │   └── extension_service.go
│   └── utils/                      # 工具函数
│       ├── response.go
│       ├── version.go
│       └── license_key.go
├── web-ui/                         # 前端项目
│   ├── src/
│   │   ├── api/                    # API 接口
│   │   ├── components/             # 组件
│   │   ├── layouts/                # 布局
│   │   ├── pages/                  # 页面
│   │   │   ├── Login.tsx
│   │   │   ├── Dashboard.tsx
│   │   │   ├── Licenses.tsx
│   │   │   ├── Plans.tsx
│   │   │   ├── Alerts.tsx
│   │   │   ├── Webhooks.tsx
│   │   │   ├── Tags.tsx
│   │   │   └── Logs.tsx
│   │   ├── store/                  # 状态管理
│   │   ├── types/                  # 类型定义
│   │   ├── utils/                  # 工具函数
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── index.html
│   ├── vite.config.ts
│   ├── tsconfig.json
│   └── package.json
├── configs/
│   └── config.yaml                 # 配置文件
├── go.mod
├── go.sum
├── README.md
└── ENHANCEMENTS.md

```

## 🚀 快速开始

### 后端服务

```bash
# 1. 安装依赖
go mod download

# 2. 配置数据库
# 编辑 configs/config.yaml

# 3. 编译
go build -o license-center ./cmd/server

# 4. 运行
./license-center -config configs/config.yaml
```

后端服务默认运行在 `http://localhost:8090`

### 前端界面

```bash
# 1. 进入前端目录
cd web-ui

# 2. 安装依赖
npm install

# 3. 启动开发服务器
npm run dev

# 4. 构建生产版本
npm run build
```

前端开发服务器默认运行在 `http://localhost:3000`

## 🎯 功能特性

### 后端功能

#### 🔒 安全增强
- API 限流（令牌桶算法）
- 请求签名验证（HMAC-SHA256）
- IP 白名单（支持 CIDR）
- 防重放攻击（Nonce）
- JWT 认证

#### 📊 监控告警
- 实时仪表盘统计
- 验证趋势分析
- 自动告警系统
- Top 客户统计

#### 🚀 功能扩展
- 授权码转移
- 批量操作
- 标签管理
- 自定义字段
- Webhook 通知

#### ⚡ 性能优化
- Redis 缓存
- 数据库索引优化
- 并发控制

### 前端功能

#### 🎨 页面
- 登录页面
- 仪表盘（实时数据 + 图表）
- 授权码管理（CRUD + 批量操作）
- 套餐管理
- 告警管理
- Webhook 管理
- 标签管理
- 验证日志

#### ✨ 特性
- 响应式设计
- 暗色侧边栏
- 实时数据刷新
- 分页和筛选
- 表单验证
- 错误处理
- Token 持久化

## 📡 API 接口

### 认证
- `POST /api/v1/auth/login` - 登录
- `GET /api/v1/auth/me` - 获取当前用户

### 授权验证
- `POST /api/v1/license/verify` - 验证授权码

### 仪表盘
- `GET /api/v1/dashboard` - 仪表盘数据
- `GET /api/v1/verify-trend` - 验证趋势
- `GET /api/v1/top-customers` - Top 客户

### 套餐管理
- `GET /api/v1/plans` - 查询套餐
- `POST /api/v1/plans` - 创建套餐
- `PUT /api/v1/plans/:id` - 更新套餐
- `DELETE /api/v1/plans/:id` - 删除套餐

### 授权码管理
- `GET /api/v1/licenses` - 查询授权码
- `POST /api/v1/licenses/generate` - 生成授权码
- `GET /api/v1/licenses/:id` - 获取详情
- `PUT /api/v1/licenses/:id` - 更新授权码
- `DELETE /api/v1/licenses/:id` - 删除授权码
- `POST /api/v1/licenses/:id/revoke` - 吊销
- `POST /api/v1/licenses/:id/restore` - 恢复
- `POST /api/v1/licenses/:id/transfer` - 转移
- `POST /api/v1/licenses/batch/*` - 批量操作

### 告警管理
- `GET /api/v1/alerts` - 查询告警
- `POST /api/v1/alerts/:id/read` - 标记已读
- `POST /api/v1/alerts/read-all` - 全部标记已读
- `DELETE /api/v1/alerts/:id` - 删除告警

### Webhook 管理
- `GET /api/v1/webhooks` - 查询配置
- `POST /api/v1/webhooks` - 创建配置
- `PUT /api/v1/webhooks/:id` - 更新配置
- `DELETE /api/v1/webhooks/:id` - 删除配置

### 标签管理
- `GET /api/v1/tags` - 查询标签
- `POST /api/v1/tags` - 创建标签
- `PUT /api/v1/tags/:id` - 更新标签
- `DELETE /api/v1/tags/:id` - 删除标签

### 日志查询
- `GET /api/v1/verify-logs` - 验证日志

## 🔧 配置说明

### 后端配置 (configs/config.yaml)

```yaml
server:
  port: "8090"
  mode: "release"

database:
  type: "postgres"
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  db_name: "nodepass_license"

redis:
  enabled: true
  host: "localhost"
  port: 6379

security:
  rate_limit:
    enabled: true
    requests_per_second: 10
    burst: 20

monitoring:
  alert:
    enabled: true
    check_interval: 3600
    expiring_days: 30
```

### 前端配置 (web-ui/.env.local)

```env
VITE_API_BASE_URL=http://localhost:8090
```

## 🐳 Docker 部署

```bash
# 构建镜像
docker build -t license-center .

# 运行容器
docker-compose up -d
```

## 📝 默认账号

- 用户名：`admin`
- 密码：`ChangeMe123!`

**⚠️ 首次登录后请立即修改密码！**

## 🛠️ 技术栈

### 后端
- Go 1.24
- Gin (Web 框架)
- GORM (ORM)
- PostgreSQL / MySQL / SQLite
- Redis (缓存)
- JWT (认证)

### 前端
- React 18
- TypeScript
- Vite (构建工具)
- Ant Design 5 (UI 组件)
- TanStack Query (数据请求)
- Zustand (状态管理)
- Recharts (图表)
- Axios (HTTP 客户端)

## 📊 数据库表

- `admin_users` - 管理员
- `license_plans` - 套餐
- `license_keys` - 授权码
- `license_activations` - 机器绑定
- `verify_logs` - 验证日志
- `license_tags` - 标签
- `license_key_tags` - 授权码标签关联
- `webhook_configs` - Webhook 配置
- `webhook_logs` - Webhook 日志
- `alerts` - 告警记录
- `license_transfer_logs` - 转移日志

## 🔐 安全建议

1. 修改默认管理员密码
2. 修改 JWT Secret
3. 启用 HTTPS
4. 配置防火墙
5. 定期备份数据库
6. 启用 IP 白名单（生产环境）
7. 配置 Redis 密码
8. 定期更新依赖

## 📈 性能优化

1. 启用 Redis 缓存
2. 配置数据库连接池
3. 使用 CDN 加速静态资源
4. 启用 Gzip 压缩
5. 配置反向代理（Nginx）
6. 定期清理日志

## 🐛 故障排查

### 后端无法启动
- 检查数据库连接
- 检查端口占用
- 查看日志文件

### 前端无法访问
- 检查后端服务状态
- 检查 API 代理配置
- 清除浏览器缓存

### 授权验证失败
- 检查授权码状态
- 检查过期时间
- 查看验证日志

## 📚 文档

- [增强功能清单](./ENHANCEMENTS.md)
- [前端文档](./web-ui/README.md)
- [API 文档](./docs/api.md)

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 License

MIT License

---

**版本**: v0.2.0
**更新时间**: 2026-03-07
