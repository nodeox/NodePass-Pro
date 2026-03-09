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
- 授权接口约定：[docs/license.md](https://github.com/nodeox/NodePass-Pro/blob/main/docs/license.md)

## 本地一键启动（PostgreSQL + Redis + Backend + Frontend）

**重要：首次启动前必须配置 JWT 密钥**

```bash
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro

# 1. 复制环境变量配置文件
cp .env.example .env

# 2. 生成 JWT 密钥并写入 .env 文件
echo "JWT_SECRET=$(openssl rand -base64 48)" >> .env

# 3. 启动服务
docker compose up -d --build
```

启动后默认访问地址：

- 前端管理端（仅本机监听）：[http://127.0.0.1:5173](http://127.0.0.1:5173)
- 后端 API：[http://localhost:8080/api/v1](http://localhost:8080/api/v1)

**安全提示**：
- JWT 密钥至少需要 32 字符，推荐 64 字符以上
- 不要在配置文件中硬编码密钥，必须使用环境变量
- 生产环境必须使用强随机密钥

## 一键部署脚本（可选 Caddy 反代）

脚本路径：`scripts/deploy.sh`

### 远程交互式一键部署（推荐）

可直接使用远程引导脚本（风格与 `bash <(curl -sL xxx/install.sh)` 一致）：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh)
```

版本查看：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh) --version
```

升级：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh) --upgrade --license-key NP-XXXX-XXXX
```

> 发布说明（2026-03-08）  
> 自此版本起，当 `BACKEND_LICENSE_ENABLED=true` 时，必须同时配置 `BACKEND_LICENSE_DOMAIN`（建议同时配置 `BACKEND_LICENSE_SITE_URL`）。若仅开启授权开关而未配置域名，后端会在运行时授权校验中拒绝通过，业务 API 将被拦截。

> 发布说明（2026-03-09）  
> 自此版本起，`install.sh` 默认采用“最小部署清单”模式（不保留源码）；`scripts/deploy.sh` 默认使用预构建镜像，且默认不构建 `nodeclient` 二进制。如需保留源码请传 `--with-source`，如需本地构建镜像请传 `--build-image`，如需构建 `nodeclient` 请传 `--build-nodeclient`（建议与 `--with-source` 配合）。

卸载：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh) --uninstall
```

脚本能力：

- 自动检测运行环境并安装缺失依赖（`git`/`curl`/`docker`/`docker compose`）；
- 默认拉取最小部署清单（不保留完整源码）；
- 交互式问答部署（前端域名、后端域名、数据库类型与连接、Redis、JWT 等）；
- 交互式创建/更新管理员账号（用户名、邮箱、密码）；
- 自动生成运行配置：`backend/configs/config.runtime.yaml`；
- 自动调用 `scripts/deploy.sh` 完成部署；
- 支持 `install / upgrade / uninstall / version` 四种一键动作；
- 部署前强制调用授权接口校验（通过后才会正式安装）；
- 部署成功后输出完整访问信息与管理员登录信息；
- 默认安装目录：`/opt/NodePass-Pro`（可用 `--install-dir` 覆盖）。

非交互模式示例：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh) \
  --license-key NP-XXXX-XXXX \
  --non-interactive --with-caddy --frontend-domain panel.example.com --backend-domain api.example.com \
  --email admin@example.com --admin-username admin --admin-email admin@example.com --admin-password 'YourStrongPassword'
```

保留源码部署（可选）：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh) \
  --license-key NP-XXXX-XXXX --license-domain panel.example.com \
  --non-interactive --with-source
```

保留源码并构建 nodeclient 下载包（可选）：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh) \
  --license-key NP-XXXX-XXXX --license-domain panel.example.com \
  --non-interactive --with-source --build-nodeclient
```

授权接口地址由系统内置，不对外暴露也不支持命令行覆盖。

1) 仅部署核心服务（PostgreSQL + Redis + Backend + Frontend）

```bash
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro
./scripts/deploy.sh --license-key NP-XXXX-XXXX
```

说明：
- 未提供可用生产域名时，`scripts/deploy.sh` 会默认关闭后端运行时授权（避免误拦截业务 API）。
- 如需启用运行时授权，请显式提供域名：

```bash
./scripts/deploy.sh --license-key NP-XXXX-XXXX --license-domain panel.example.com --license-site-url https://panel.example.com
```

2) 启用 Caddy 反向代理（自动 HTTPS）

```bash
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro
./scripts/deploy.sh --license-key NP-XXXX-XXXX --with-caddy --frontend-domain panel.example.com --email admin@example.com
```

3) 自定义 Caddy 端口（例如 8080/8443）

```bash
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro
./scripts/deploy.sh --license-key NP-XXXX-XXXX --with-caddy --frontend-domain panel.example.com --backend-domain api.example.com --caddy-http-port 8080 --caddy-https-port 8443
```

4) 停止服务

```bash
./scripts/deploy.sh --down
```

说明：

- 版本文件：
  - `VERSION`（面板版本）
  - `backend/VERSION`
  - `frontend/VERSION`
  - `nodeclient/VERSION`
- 可使用 `./scripts/version.sh show` 查看版本，`./scripts/version.sh set ...` 更新版本；
- 授权验证脚本：`scripts/license-verify.py`（会按版本策略校验 panel/backend/frontend/nodeclient）；
- 运行时授权 E2E 联调脚本：`tests/license_runtime_e2e.sh`（真实 `license-center` + Docker backend 启动回归）；
- `install.sh` 与 `scripts/deploy.sh` 在非 `--down` 模式下都会强制授权校验；
- 后端运行时会持续校验授权，授权过期/失效后业务 API 会被拒绝（保留 `/health` 与 `/api/v1/license/status`）；
- 启用 Caddy 时会自动生成配置文件：`deploy/caddy/Caddyfile`；
- Caddy 默认将前端域名的 `/api/*`、`/ws` 反代到后端，其余路径反代到前端；
- Caddy 同时暴露节点安装入口：`/nodeclient-install.sh` 与 `/downloads/*`；
- 节点二进制文件放置说明见：`deploy/nodeclient/README.md`；
- `scripts/deploy.sh` 默认不构建 nodeclient 二进制；
- 如需构建并生成 `.sha256` 到 `deploy/nodeclient/downloads/`，使用：`./scripts/deploy.sh --build-nodeclient`；
- 可用 `./scripts/deploy.sh --skip-nodeclient-build` 作为兼容参数显式关闭构建；
- 可通过 `--backend-domain` 暴露独立后端域名；
- 可通过环境变量 `BACKEND_CONFIG_FILE` 与 `FRONTEND_BIND` 覆盖后端配置文件和前端监听地址；
- 请确保域名 DNS 已解析到部署机器，否则自动证书签发会失败。
