# NodePass-Pro License 服务启动报告

## 📋 启动信息

**启动时间**: 2026-03-09 06:15
**项目路径**: /Users/jianshe/Projects/NodePass-Pro/license
**状态**: ✅ 启动成功

## 🎯 服务状态

### 后端服务

| 项目 | 信息 |
|------|------|
| **状态** | ✅ 运行中 |
| **端口** | 8091 |
| **访问地址** | http://localhost:8091 |
| **健康检查** | http://localhost:8091/health |
| **API 前缀** | http://localhost:8091/api/v1 |
| **进程 PID** | 查看 backend.pid |
| **日志文件** | backend.log |
| **数据库** | SQLite (./data/license-unified.db) |
| **模式** | debug |

### 前端服务

| 项目 | 信息 |
|------|------|
| **状态** | ✅ 运行中 |
| **端口** | 5176 |
| **访问地址** | http://localhost:5176 |
| **进程 PID** | 查看 frontend.pid |
| **日志文件** | frontend.log |
| **构建工具** | Vite 7.3.1 |
| **框架** | React + TypeScript |

## 🔌 API 接口

### 认证接口
- `POST /api/v1/auth/login` - 管理员登录
- `GET /api/v1/auth/me` - 获取当前用户信息

### 统一验证
- `POST /api/v1/verify` - 统一校验（授权 + 版本）

### 控制台
- `GET /api/v1/dashboard` - 控制台统计

### 套餐管理
- `GET /api/v1/plans` - 列表查询
- `POST /api/v1/plans` - 创建套餐
- `PUT /api/v1/plans/:id` - 更新套餐
- `DELETE /api/v1/plans/:id` - 删除套餐

### 授权码管理
- `POST /api/v1/licenses/generate` - 生成授权码
- `GET /api/v1/licenses` - 列表查询
- `GET /api/v1/licenses/:id` - 获取详情
- `PUT /api/v1/licenses/:id` - 更新授权码
- `POST /api/v1/licenses/:id/revoke` - 吊销授权码
- `POST /api/v1/licenses/:id/restore` - 恢复授权码
- `GET /api/v1/licenses/:id/activations` - 查看激活记录

### 版本管理
- `GET /api/v1/releases` - 发布列表
- `POST /api/v1/releases` - 创建发布
- `GET /api/v1/version-policies` - 版本策略列表
- `POST /api/v1/version-policies` - 创建版本策略

### 日志查询
- `GET /api/v1/verify-logs` - 验证日志

## 🔐 默认账号

**管理员账号**:
- 用户名: `admin`
- 密码: `admin123456`
- 邮箱: `admin@example.com`

**首次登录建议**: 立即修改默认密码

## 🚀 快速测试

### 1. 测试后端健康检查
```bash
curl http://localhost:8091/health
```

### 2. 测试管理员登录
```bash
curl -X POST http://localhost:8091/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123456"
  }'
```

### 3. 访问前端
在浏览器中打开: http://localhost:5176

### 4. 测试统一验证接口
```bash
curl -X POST http://localhost:8091/api/v1/verify \
  -H "Content-Type: application/json" \
  -d '{
    "license_key": "YOUR_LICENSE_KEY",
    "machine_id": "test-machine-001",
    "product_name": "NodePass-Pro",
    "product_version": "1.0.0"
  }'
```

## 📝 管理命令

### 后端管理

```bash
# 查看后端日志
tail -f backend.log

# 停止后端
kill $(cat backend.pid)

# 重启后端
kill $(cat backend.pid) && nohup go run ./cmd/server > backend.log 2>&1 &
echo $! > backend.pid
```

### 前端管理

```bash
# 查看前端日志
tail -f frontend.log

# 停止前端
kill $(cat frontend.pid)

# 重启前端
kill $(cat frontend.pid) && nohup npm run dev > frontend.log 2>&1 &
echo $! > frontend.pid
```

### 停止所有服务

```bash
# 停止后端
kill $(cat backend.pid) 2>/dev/null

# 停止前端
kill $(cat frontend.pid) 2>/dev/null

# 或者
pkill -f "go run ./cmd/server"
pkill -f "vite"
```

## 🔧 配置文件

### 后端配置
- **位置**: `backend/.env`
- **模板**: `backend/.env.example`

**主要配置项**:
```bash
SERVER_PORT=8091
GIN_MODE=debug
DB_DRIVER=sqlite
DB_DSN=./data/license-unified.db
JWT_SECRET=please-change-this-secret
JWT_EXPIRE_HOURS=24
```

### 前端配置
- **位置**: `frontend/.env`
- **模板**: `frontend/.env.example`

**主要配置项**:
```bash
VITE_API_BASE=/api/v1
```

## 📊 数据存储

### 数据库
- **类型**: SQLite
- **位置**: `backend/data/license-unified.db`
- **备份**: 定期复制数据库文件

### 日志文件
- **后端日志**: `backend/backend.log`
- **前端日志**: `frontend/frontend.log`

## 🎯 功能特性

### 统一授权系统
- ✅ 授权码生成和管理
- ✅ 机器绑定和激活
- ✅ 授权码吊销和恢复
- ✅ 套餐管理
- ✅ 验证日志

### 版本管理系统
- ✅ 产品发布管理
- ✅ 版本策略配置
- ✅ 版本兼容性检查
- ✅ 统一验证接口

### 管理后台
- ✅ 控制台统计
- ✅ 授权码管理
- ✅ 套餐管理
- ✅ 版本管理
- ✅ 日志查询

## 🔄 开发工作流

### 后端开发
```bash
cd backend

# 安装依赖
go mod tidy

# 运行开发服务器
go run ./cmd/server

# 构建
go build -o license-server ./cmd/server
```

### 前端开发
```bash
cd frontend

# 安装依赖
npm install

# 运行开发服务器
npm run dev

# 构建生产版本
npm run build

# 预览生产构建
npm run preview
```

## ⚠️ 注意事项

### 1. 端口占用
确保以下端口未被占用：
- 8091 - 后端服务
- 5176 - 前端服务

### 2. 数据备份
定期备份数据库文件：
```bash
cp backend/data/license-unified.db backend/data/license-unified.db.backup
```

### 3. 日志管理
日志文件会持续增长，建议定期清理：
```bash
# 清空日志
> backend/backend.log
> frontend/frontend.log
```

### 4. 生产部署
生产环境建议：
- 修改默认密码
- 更改 JWT_SECRET
- 设置 GIN_MODE=release
- 使用 PostgreSQL/MySQL 替代 SQLite
- 配置反向代理（Nginx）
- 启用 HTTPS

## 🐛 故障排查

### 后端无法启动
1. 检查端口占用: `lsof -i :8091`
2. 查看日志: `tail -f backend/backend.log`
3. 检查数据库文件权限
4. 验证 Go 环境: `go version`

### 前端无法启动
1. 检查端口占用: `lsof -i :5176`
2. 查看日志: `tail -f frontend/frontend.log`
3. 重新安装依赖: `npm install`
4. 清除缓存: `rm -rf node_modules && npm install`

### API 请求失败
1. 检查后端是否运行: `curl http://localhost:8091/health`
2. 检查 CORS 配置
3. 验证 API 路径
4. 查看后端日志

## 📖 相关文档

### 项目文档
- `README.md` - 项目说明
- `backend/README.md` - 后端文档
- `frontend/README.md` - 前端文档（如果有）

### API 文档
- 查看后端启动日志中的路由列表
- 使用 Postman 或类似工具测试 API

## ✅ 启动检查清单

- [x] 后端服务启动成功
- [x] 前端服务启动成功
- [x] 后端健康检查通过
- [x] 前端页面可访问
- [x] 数据库文件创建
- [x] 日志文件生成
- [x] 进程 PID 记录

## 🎉 总结

**服务状态**: 所有服务运行正常

**访问地址**:
- 前端: http://localhost:5176
- 后端: http://localhost:8091
- 健康检查: http://localhost:8091/health

**默认账号**:
- 用户名: admin
- 密码: admin123456

可以开始使用了！🚀

---

**启动完成时间**: 2026-03-09 06:15
**状态**: ✅ 成功
