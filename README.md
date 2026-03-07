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
./scripts/deploy.sh --with-caddy --domain panel.example.com --email admin@example.com
```

3) 自定义 Caddy 端口（例如 8080/8443）

```bash
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro
./scripts/deploy.sh --with-caddy --domain panel.example.com --caddy-http-port 8080 --caddy-https-port 8443
```

4) 停止服务

```bash
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro
./scripts/deploy.sh --down
```

说明：

- 启用 Caddy 时会自动生成配置文件：`deploy/caddy/Caddyfile`；
- Caddy 会将 `/api/*` 与 `/ws` 反代到后端，其余路径反代到前端；
- 请确保域名 DNS 已解析到部署机器，否则自动证书签发会失败。
