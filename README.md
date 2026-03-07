# NodePass Pro

NodePass Pro 是一个前后端分离的 TCP/UDP 流量转发管理系统，包含面板后端、前端管理端与节点客户端。

项目仓库：[https://github.com/nodeox/NodePass-Pro](https://github.com/nodeox/NodePass-Pro)

## 模块导航

- `backend/`：面板后端服务（Go）
- `frontend/`：面板前端管理端（React + TypeScript）
- `nodeclient/`：节点客户端（Go）

## 文档入口

- 后端说明：[backend/README.md](https://github.com/nodeox/NodePass-Pro/blob/main/backend/README.md)
- 前端说明：[frontend/README.md](https://github.com/nodeox/NodePass-Pro/blob/main/frontend/README.md)
- 节点客户端说明：[nodeclient/README.md](https://github.com/nodeox/NodePass-Pro/blob/main/nodeclient/README.md)

## 本地一键启动（PostgreSQL + Redis + Backend + Frontend）

```bash
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro
docker compose up -d --build
```

启动后默认访问地址：

- 前端管理端（仅本机监听）：[http://127.0.0.1:5173](http://127.0.0.1:5173)
- 后端 API：[http://localhost:8080/api/v1](http://localhost:8080/api/v1)

## 一键部署脚本（可选 Caddy 反代）

脚本路径：`scripts/deploy.sh`

### 远程交互式一键部署（推荐）

可直接使用远程引导脚本（风格与 `bash <(curl -sL xxx/install.sh)` 一致）：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh)
```

脚本能力：

- 自动检测运行环境并安装缺失依赖（`git`/`curl`/`docker`/`docker compose`）；
- 交互式问答部署（前端域名、后端域名、数据库类型与连接、Redis、JWT 等）；
- 交互式创建/更新管理员账号（用户名、邮箱、密码）；
- 自动生成运行配置：`backend/configs/config.runtime.yaml`；
- 自动调用 `scripts/deploy.sh` 完成部署；
- 部署成功后输出完整访问信息与管理员登录信息；
- 默认安装目录：`/opt/NodePass-Pro`（可用 `--install-dir` 覆盖）。

非交互模式示例：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh) \
  --non-interactive --with-caddy --frontend-domain panel.example.com --backend-domain api.example.com \
  --email admin@example.com --admin-username admin --admin-email admin@example.com --admin-password 'YourStrongPassword'
```

1) 仅部署核心服务（PostgreSQL + Redis + Backend + Frontend）

```bash
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro
./scripts/deploy.sh
```

2) 启用 Caddy 反向代理（自动 HTTPS）

```bash
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro
./scripts/deploy.sh --with-caddy --frontend-domain panel.example.com --email admin@example.com
```

3) 自定义 Caddy 端口（例如 8080/8443）

```bash
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro
./scripts/deploy.sh --with-caddy --frontend-domain panel.example.com --backend-domain api.example.com --caddy-http-port 8080 --caddy-https-port 8443
```

4) 停止服务

```bash
./scripts/deploy.sh --down
```

说明：

- 启用 Caddy 时会自动生成配置文件：`deploy/caddy/Caddyfile`；
- Caddy 默认将前端域名的 `/api/*`、`/ws` 反代到后端，其余路径反代到前端；
- Caddy 同时暴露节点安装入口：`/nodeclient-install.sh` 与 `/downloads/*`；
- 节点二进制文件放置说明见：`deploy/nodeclient/README.md`；
- 可通过 `--backend-domain` 暴露独立后端域名；
- 可通过环境变量 `BACKEND_CONFIG_FILE` 与 `FRONTEND_BIND` 覆盖后端配置文件和前端监听地址；
- 请确保域名 DNS 已解析到部署机器，否则自动证书签发会失败。
