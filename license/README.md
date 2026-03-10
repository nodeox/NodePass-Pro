# NodePass Unified License + Version System

基于 `license-center` 能力从零重写的“授权系统 + 版本系统”一体化实现。

## 目标

- 授权校验与版本校验合并为一个核心接口
- 前后端分离，数据库持久化
- 支持管理端可视化操作（套餐、授权、版本、策略、日志）
- 支持商业化闭环基础能力（试用、订单、支付回调）

## 技术栈

- 后端：Go 1.22 / Gin / Gorm / JWT / SQLite(MySQL/PostgreSQL 可切换)
- 前端：Vite / React 18 / TypeScript / Ant Design 5 / Zustand / Axios
- 数据库：默认 SQLite (`backend/data/license-unified.db`)

## 目录结构

```text
license/
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── config/           # 环境配置
│   │   ├── database/         # DB 初始化 + 迁移 + 种子数据
│   │   ├── models/           # 数据模型
│   │   ├── services/         # 核心业务（授权+版本合一）
│   │   ├── handlers/         # API 处理器
│   │   ├── middleware/       # JWT 中间件
│   │   └── router/           # 路由注册
│   └── .env.example
└── frontend/
    ├── src/pages/            # 登录/仪表盘/套餐/授权/版本/日志
    ├── src/store/            # Zustand 状态
    └── src/utils/            # Axios 请求封装
```

## 核心合一接口

`POST /api/v1/verify`

单次请求同时完成：

- 授权有效性校验（状态、到期、设备数）
- 设备绑定/续活
- 版本兼容性校验（最低支持版本、最新版本、是否强制升级）

返回统一结果：

- `verified`: 是否最终通过
- `license`: 授权维度明细
- `version`: 版本维度明细
- `status`: 最终状态（如 `ok` / `upgrade_required` / `invalid_license`）

## 主要管理接口

- `POST /api/v1/auth/login`
- `GET /api/v1/dashboard`
- `GET/POST/PUT/DELETE /api/v1/plans`
- `POST /api/v1/plans/:id/clone`
- `POST /api/v1/licenses/generate`
- `GET /api/v1/licenses`
- `PUT /api/v1/licenses/:id`
- `DELETE /api/v1/licenses/:id`
- `GET /api/v1/licenses/:id/activations`
- `DELETE /api/v1/licenses/:id/activations/:activation_id`
- `DELETE /api/v1/licenses/:id/activations`
- `POST /api/v1/licenses/batch/delete`
- `POST /api/v1/licenses/batch/revoke`
- `POST /api/v1/licenses/batch/restore`
- `POST /api/v1/licenses/batch/update`
- `POST /api/v1/licenses/:id/revoke`
- `POST /api/v1/licenses/:id/restore`
- `GET/POST /api/v1/releases`
- `GET /api/v1/releases/recycle`
- `GET /api/v1/version-sync/configs`
- `GET/PUT /api/v1/version-sync/config`
- `POST /api/v1/version-sync/manual`
- `PUT /api/v1/releases/:id`
- `POST /api/v1/releases/upload`
- `PUT /api/v1/releases/:id/file`
- `GET /api/v1/releases/:id/file`
- `DELETE /api/v1/releases/:id`
- `POST /api/v1/releases/:id/restore`
- `DELETE /api/v1/releases/:id/purge`
- `GET/POST /api/v1/version-policies`
- `PUT/DELETE /api/v1/version-policies/:id`
- `GET /api/v1/verify-logs`
- `POST /api/v1/commercial/trials/issue`
- `POST /api/v1/commercial/orders/renew`
- `POST /api/v1/commercial/orders/upgrade`
- `POST /api/v1/commercial/orders/transfer`
- `GET /api/v1/commercial/orders`
- `GET /api/v1/commercial/orders/:id`
- `POST /api/v1/commercial/orders/:id/mark-paid`
- `POST /api/v1/commercial/payments/callback/:channel`

商业化回调约束（一期）：

- 回调状态支持 `paid` / `failed` / `canceled`
- 同一订单重复 `paid` 回调为幂等，不会重复改授权
- 已支付订单禁止回退状态
- `paid` 回调若传入 `amount_cents` 且与订单金额不一致，会返回 `400` 并记录事件日志
- 支持按渠道回调验签（`X-Callback-Signature` + `X-Callback-Timestamp` + `X-Callback-Nonce`）
- 状态机收紧为 `pending -> paid|failed|canceled`，终态只允许同态幂等回调

授权编辑（`PUT /api/v1/licenses/:id`）支持字段：

- `key`、`plan_id`、`customer`、`status`
- `expires_at` / `clear_expires_at`
- `max_machines` / `clear_max_machines`
- `metadata_json`、`note`

授权列表 `GET /api/v1/licenses` 支持筛选与分页参数：

- `page`、`page_size`
- `customer`、`status`、`plan_id`
- `expire_from`、`expire_to`
- `sort_by`（`created_at` / `expires_at` / `status`）
- `sort_order`（`asc` / `desc`，未传默认 `desc`）

## 快速启动

### 1) 启动后端

```bash
cd /Users/jianshe/Projects/NodePass-Pro/license/backend
cp .env.example .env
# 必须设置 BOOTSTRAP_ADMIN_PASSWORD
go mod tidy
go run ./cmd/server
```

默认端口：`8091`

默认管理员（首次启动自动创建）：

- 用户名：`admin`
- 密码：来自 `BOOTSTRAP_ADMIN_PASSWORD`（必填，无默认值）

### 2) 启动前端

```bash
cd /Users/jianshe/Projects/NodePass-Pro/license/frontend
cp .env.example .env
npm install
npm run dev
```

默认地址：`http://127.0.0.1:5176`

## Docker 部署（推荐）

本目录已提供：

- `backend/Dockerfile`
- `frontend/Dockerfile`
- `frontend/nginx.conf`
- `docker-compose.yml`
- `scripts/install-remote.sh`

本地打包镜像：

```bash
cd /Users/jianshe/Projects/NodePass-Pro/license
cp .env.docker.example .env.docker
# 修改 JWT_SECRET / BOOTSTRAP_ADMIN_PASSWORD
docker compose build
```

远程服务器一键安装（执行一条命令）：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license/scripts/install-remote.sh)
```

说明：

- 脚本已改为交互式，会提示你输入：
  - 管理员密码
  - 绑定域名（可选）
  - 证书邮箱（可选）
  - 部署端口、分支、安装目录
- 脚本内置环境自检：
  - 自动安装 Docker / Docker Compose（缺失时）
  - 自动处理 docker 权限（必要时使用 `sudo docker`）
  - 自动尝试放行防火墙端口（`ufw` / `firewalld`）
- 部署默认使用“镜像拉取模式”（`pull + --no-build`），不会在服务器本地编译。
- 如需改为源码编译模式，可设置：`DEPLOY_WITH_BUILD=true`
- 当你填写域名时，会自动启用 Caddy（80/443）并自动申请/续期 HTTPS 证书。

常用可选变量（放在命令前）：

- `PANEL_PORT`（默认 `8088`）
- `BRANCH`（默认 `main`）
- `INSTALL_DIR`（默认 `/opt/NodePass-Pro`）
- `JWT_SECRET`（不传则脚本自动生成）
- `PANEL_DOMAIN`（可直接预设域名，跳过输入）
- `ACME_EMAIL`（可直接预设证书邮箱）
- `DEPLOY_WITH_BUILD`（默认 `false`；为 `true` 时启用本地编译部署）

## 已完成验证

- 后端 `go build ./...` 通过
- 前端 `npm run build` 通过
- 端到端烟雾测试通过：
  - 登录 -> 新建发布/策略 -> 生成授权 -> 调用统一校验接口
  - 登录 -> 试用发放 -> 续费/升级/转移订单 -> 支付回调 -> 订单事件验证
