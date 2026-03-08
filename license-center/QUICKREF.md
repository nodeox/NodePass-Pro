# 快速参考

## 常用命令速查

### Makefile 命令

```bash
# 查看帮助
make help

# 部署管理
make up              # 启动服务（构建镜像）
make start           # 启动服务（不重新构建）
make down            # 停止服务
make restart         # 重启服务
make logs            # 查看日志
make status          # 查看服务状态

# 开发模式
make dev             # 启动后端开发模式
make dev-web         # 启动前端开发模式
make build-local     # 本地构建二进制
make build-web       # 构建前端

# 测试
make test            # 运行测试
make test-coverage   # 查看测试覆盖率
make lint            # 代码检查
make fmt             # 格式化代码

# 数据库
make db-shell        # 进入数据库 shell
make backup-db       # 备份数据库
make restore-db FILE=backup.sql  # 恢复数据库

# 其他
make health          # 健康检查
make ps              # 查看容器
make exec            # 进入容器 shell
make clean           # 清理所有数据
```

### 部署脚本命令

```bash
# 基本操作
./scripts/deploy.sh --up          # 启动服务
./scripts/deploy.sh --down        # 停止服务
./scripts/deploy.sh --restart     # 重启服务
./scripts/deploy.sh --logs        # 查看日志
./scripts/deploy.sh --status      # 查看状态

# 高级操作
./scripts/deploy.sh --build-only  # 仅构建镜像
./scripts/deploy.sh --pull        # 拉取基础镜像
./scripts/deploy.sh --clean       # 清理所有数据
./scripts/deploy.sh --no-build    # 启动但不重新构建
```

### Docker Compose 命令

```bash
# 服务管理
docker compose up -d              # 启动服务
docker compose down               # 停止服务
docker compose restart            # 重启服务
docker compose ps                 # 查看状态

# 日志查看
docker compose logs -f            # 查看所有日志
docker compose logs -f license-center  # 查看应用日志
docker compose logs -f postgres   # 查看数据库日志

# 容器操作
docker compose exec license-center /bin/bash  # 进入应用容器
docker compose exec postgres psql -U postgres -d nodepass_license  # 进入数据库

# 清理
docker compose down -v            # 停止并删除卷
docker compose down --remove-orphans  # 停止并删除孤立容器
```

### 安装脚本命令

```bash
# 安装
./install.sh --install

# 升级
./install.sh --upgrade

# 卸载
./install.sh --uninstall

# 自定义安装目录
./install.sh --install --install-dir /data/license-center

# 跳过健康检查
./install.sh --install --skip-health-check
```

## 配置文件

### 环境变量 (.env)

```bash
# PostgreSQL 配置
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_secure_password
POSTGRES_DB=nodepass_license
POSTGRES_PORT=5432

# 应用配置
APP_PORT=8090
BUILD_VERSION=0.3.0
GIN_MODE=release
```

### 应用配置 (configs/config.yaml)

```yaml
server:
  port: 8090
  mode: release

database:
  type: postgres
  host: postgres
  port: 5432
  user: postgres
  password: postgres
  dbname: nodepass_license

admin:
  username: admin
  password: your_admin_password

security:
  jwt_secret: your_jwt_secret
  rate_limit: 100
```

## 端口说明

| 端口 | 服务 | 说明 |
|------|------|------|
| 8090 | License Center | 主应用服务 |
| 5432 | PostgreSQL | 数据库服务 |

## 目录结构

```
license-center/
├── cmd/                    # 应用入口
├── internal/               # 内部代码
├── configs/                # 配置文件
├── web-ui/                 # 前端代码
├── scripts/                # 部署脚本
├── docs/                   # 文档
├── Dockerfile              # Docker 构建文件
├── docker-compose.yml      # Docker Compose 配置
├── Makefile                # Make 命令
├── install.sh              # 安装脚本
├── .env.example            # 环境变量模板
└── README.md               # 项目说明
```

## API 端点

### 健康检查
```bash
GET /health
```

### 管理面板
```bash
GET /console
```

### API 接口
```bash
POST /api/v1/admin/login          # 管理员登录
GET  /api/v1/licenses              # 获取授权码列表
POST /api/v1/licenses              # 创建授权码
GET  /api/v1/licenses/:id          # 获取授权码详情
PUT  /api/v1/licenses/:id          # 更新授权码
DELETE /api/v1/licenses/:id        # 删除授权码
POST /api/v1/verify                # 验证授权码
```

## 故障排查

### 服务无法启动

```bash
# 查看日志
make logs
# 或
docker compose logs -f

# 检查端口占用
lsof -i :8090
lsof -i :5432

# 重新构建
make down
make up
```

### 数据库连接失败

```bash
# 检查数据库状态
docker compose ps postgres

# 查看数据库日志
docker compose logs postgres

# 进入数据库
make db-shell
```

### 健康检查失败

```bash
# 手动检查
curl http://127.0.0.1:8090/health

# 查看应用日志
docker compose logs -f license-center

# 检查配置
cat configs/config.yaml
```

## 备份与恢复

### 备份数据库

```bash
# 使用 Makefile
make backup-db

# 手动备份
docker compose exec -T postgres pg_dump -U postgres nodepass_license > backup.sql
```

### 恢复数据库

```bash
# 使用 Makefile
make restore-db FILE=backup.sql

# 手动恢复
docker compose exec -T postgres psql -U postgres nodepass_license < backup.sql
```

### 备份配置

```bash
# 备份配置文件
cp configs/config.yaml configs/config.yaml.backup

# 备份环境变量
cp .env .env.backup
```

## 性能优化

### 数据库优化

```bash
# 进入数据库
make db-shell

# 执行优化
VACUUM ANALYZE;

# 查看连接数
SELECT count(*) FROM pg_stat_activity;
```

### 查看资源使用

```bash
# 查看容器资源
docker stats

# 查看磁盘使用
df -h

# 查看日志大小
du -sh /var/lib/docker/containers/*
```

## 安全建议

1. 修改默认密码
2. 启用 HTTPS
3. 配置防火墙
4. 限制数据库访问
5. 定期备份数据
6. 监控日志
7. 及时更新版本

## 监控指标

### 关键指标

- 授权码总数
- 活跃授权码数
- 验证请求数
- 验证成功率
- 响应时间
- 错误率

### 查看指标

```bash
# 访问仪表盘
http://127.0.0.1:8090/console

# 查看日志
make logs

# 查看数据库统计
make db-shell
SELECT * FROM license_keys;
SELECT * FROM verification_logs;
```

## 升级流程

```bash
# 1. 备份数据
make backup-db

# 2. 停止服务
make down

# 3. 拉取最新代码
git pull

# 4. 更新配置
cp .env.example .env
vim .env

# 5. 重新构建
make up

# 6. 验证服务
make health
```

## 技术支持

- 文档：[README.md](./README.md)
- 部署：[DEPLOYMENT.md](./DEPLOYMENT.md)
- 架构：[ARCHITECTURE.md](./ARCHITECTURE.md)
- 更新日志：[CHANGELOG.md](./CHANGELOG.md)
- 问题反馈：[GitHub Issues](https://github.com/nodeox/NodePass-Pro/issues)
