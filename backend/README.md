# NodePass Pro Backend

NodePass Pro 后端基于 Go + Gin + Gorm，提供节点、规则、流量、VIP、公告、审计等 API。

## 环境要求

- Go `1.21+`
- Linux/macOS（开发环境）

## 配置文件

- 默认配置：[`backend/configs/config.yaml`](https://github.com/nodeox/NodePass-Pro/blob/main/backend/configs/config.yaml)
- 默认数据库：PostgreSQL
- 可切换：SQLite / MySQL / PostgreSQL（通过 `database.type` 与 `database.dsn`）

## 运行时授权校验

可在配置文件中启用运行时授权校验：

```yaml
license:
  enabled: true
  license_key: "NP-XXXX-XXXX-XXXX"
  machine_id: ""
  verify_interval: 300
  fail_open: false
  offline_grace_seconds: 600
```

说明：

- 校验失败或授权过期后，业务 API 会被拒绝；
- 仍保留健康检查接口：`/health` 与 `/api/v1/license/status`；
- 使用部署脚本时会自动注入授权相关环境变量。

## 数据库切换示例

- PostgreSQL（默认）

```yaml
database:
  type: "postgres"
  dsn: "host=127.0.0.1 user=postgres password=postgres dbname=nodepass_panel port=5432 sslmode=disable TimeZone=Asia/Shanghai"
```

- SQLite

```yaml
database:
  type: "sqlite"
  dsn: "./data/nodepass.db"
```

- MySQL

```yaml
database:
  type: "mysql"
  dsn: "root:password@tcp(127.0.0.1:3306)/nodepass_panel?charset=utf8mb4&parseTime=True&loc=Local"
```

## Redis 缓存示例

```yaml
redis:
  enabled: true
  addr: "127.0.0.1:6379"
  password: ""
  db: 0
  key_prefix: "nodepass:panel"
  default_ttl: 300
```

## 本地启动

```bash
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro/backend
go run ./cmd/server
```

服务默认监听 `:8080`，可通过配置修改端口。

## Docker Compose 一键启动（PostgreSQL + Redis + Backend）

在仓库根目录执行：

```bash
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro
docker compose up -d --build
```

使用容器专用配置文件：

- `backend/configs/config.docker.yaml`

查看服务状态：

```bash
docker compose ps
docker compose logs -f backend
```

停止并清理：

```bash
docker compose down
```

## 构建

```bash
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro/backend
go build -o ./bin/server ./cmd/server
```

## 版本

- 后端模块版本文件：`backend/VERSION`
- 运行中的后端版本可通过 `GET /health` 或 `GET /api/v1/ping` 返回值查看

## 快速验证

```bash
curl -s http://127.0.0.1:8080/health
curl -s http://127.0.0.1:8080/api/v1/ping
```
