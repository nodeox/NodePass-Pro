# NodePass License Center 部署指南

## 目录

- [快速开始](#快速开始)
- [部署方式](#部署方式)
- [配置说明](#配置说明)
- [常用命令](#常用命令)
- [升级指南](#升级指南)
- [故障排查](#故障排查)

## 快速开始

### 一键安装（推荐）

```bash
# 远程一键安装
bash <(curl -fsSL "https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license-center/install.sh") --install

# 或下载后安装
wget https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license-center/install.sh
chmod +x install.sh
./install.sh --install
```

### 手动部署

```bash
# 1. 克隆仓库
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro/license-center

# 2. 配置环境变量
cp .env.example .env
vim .env

# 3. 启动服务
make up
# 或
./scripts/deploy.sh --up
```

## 部署方式

### 方式一：Docker Compose（推荐）

适用于单机部署，包含完整的数据库和应用服务。

```bash
# 构建并启动
docker compose up -d --build

# 查看日志
docker compose logs -f

# 停止服务
docker compose down
```

### 方式二：使用 Makefile

提供了便捷的命令行工具。

```bash
# 查看所有可用命令
make help

# 启动服务
make up

# 查看状态
make status

# 查看日志
make logs

# 重启服务
make restart

# 停止服务
make down
```

### 方式三：使用部署脚本

功能最完整的部署方式。

```bash
# 启动服务
./scripts/deploy.sh --up

# 停止服务
./scripts/deploy.sh --down

# 重启服务
./scripts/deploy.sh --restart

# 查看日志
./scripts/deploy.sh --logs

# 查看状态
./scripts/deploy.sh --status
```

## 配置说明

### 环境变量配置

创建 `.env` 文件：

```bash
# PostgreSQL 数据库配置
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_secure_password
POSTGRES_DB=nodepass_license
POSTGRES_PORT=5432

# 应用配置
APP_PORT=8090
BUILD_VERSION=0.3.0
GIN_MODE=release
```

### 应用配置文件

编辑 `configs/config.yaml`：

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

## 常用命令

### 使用 Makefile

```bash
# 开发相关
make dev              # 启动后端开发模式
make dev-web          # 启动前端开发模式
make build-local      # 本地构建二进制
make build-web        # 构建前端

# 部署相关
make up               # 启动服务（构建）
make start            # 启动服务（不构建）
make down             # 停止服务
make restart          # 重启服务
make logs             # 查看日志
make status           # 查看状态

# 测试相关
make test             # 运行测试
make test-coverage    # 查看覆盖率
make lint             # 代码检查
make fmt              # 格式化代码

# 数据库相关
make db-shell         # 进入数据库 shell
make backup-db        # 备份数据库
make restore-db FILE=backup.sql  # 恢复数据库

# 其他
make health           # 健康检查
make ps               # 查看容器
make exec             # 进入容器 shell
```

### 使用部署脚本

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

### 使用 Docker Compose

```bash
# 启动服务
docker compose up -d

# 查看日志
docker compose logs -f license-center
docker compose logs -f postgres

# 重启服务
docker compose restart

# 停止服务
docker compose down

# 清理所有数据
docker compose down -v
```

## 升级指南

### 自动升级

```bash
# 使用安装脚本升级
./install.sh --upgrade

# 或使用 Makefile
make upgrade
```

### 手动升级

```bash
# 1. 备份数据库
make backup-db

# 2. 备份配置文件
cp configs/config.yaml configs/config.yaml.backup

# 3. 拉取最新代码
git pull origin main

# 4. 重新构建并启动
make down
make up

# 5. 验证服务
make health
```

## 故障排查

### 服务无法启动

```bash
# 查看容器状态
docker compose ps

# 查看日志
docker compose logs license-center
docker compose logs postgres

# 检查端口占用
lsof -i :8090
lsof -i :5432

# 重新构建
docker compose down
docker compose up -d --build --force-recreate
```

### 数据库连接失败

```bash
# 检查数据库容器
docker compose ps postgres

# 进入数据库容器
docker compose exec postgres psql -U postgres -d nodepass_license

# 检查数据库日志
docker compose logs postgres

# 重启数据库
docker compose restart postgres
```

### 健康检查失败

```bash
# 手动检查健康端点
curl http://127.0.0.1:8090/health

# 查看应用日志
docker compose logs -f license-center

# 检查配置文件
cat configs/config.yaml

# 进入容器检查
docker compose exec license-center /bin/bash
```

### 前端无法访问

```bash
# 检查前端是否构建
ls -la web-ui/dist

# 重新构建前端
cd web-ui && npm run build

# 重新构建镜像
docker compose build --no-cache license-center
docker compose up -d
```

### 性能问题

```bash
# 查看容器资源使用
docker stats

# 查看数据库连接数
docker compose exec postgres psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# 优化数据库
docker compose exec postgres psql -U postgres -d nodepass_license -c "VACUUM ANALYZE;"
```

## 生产环境建议

### 安全配置

1. 修改默认密码
2. 启用 HTTPS
3. 配置防火墙
4. 限制数据库访问
5. 定期更新系统

### 性能优化

1. 调整数据库连接池
2. 启用 Redis 缓存
3. 配置 CDN
4. 使用反向代理（Nginx）
5. 监控资源使用

### 备份策略

```bash
# 定期备份数据库（添加到 crontab）
0 2 * * * cd /opt/nodepass-license-center/license-center && make backup-db

# 备份配置文件
cp configs/config.yaml /backup/config.yaml.$(date +%Y%m%d)
```

### 监控告警

1. 配置健康检查
2. 设置日志告警
3. 监控磁盘空间
4. 监控数据库性能
5. 配置 Webhook 通知

## 常见问题

### Q: 如何修改端口？

A: 编辑 `.env` 文件中的 `APP_PORT`，然后重启服务。

### Q: 如何使用外部数据库？

A: 编辑 `configs/config.yaml`，修改数据库连接信息，然后注释掉 `docker-compose.yml` 中的 postgres 服务。

### Q: 如何启用 HTTPS？

A: 建议使用 Nginx 作为反向代理，配置 SSL 证书。

### Q: 如何扩展到多节点？

A: 使用外部数据库和 Redis，然后在多台服务器上部署应用服务。

## 技术支持

- 文档：[README.md](./README.md)
- 架构：[ARCHITECTURE.md](./ARCHITECTURE.md)
- 问题反馈：[GitHub Issues](https://github.com/nodeox/NodePass-Pro/issues)
