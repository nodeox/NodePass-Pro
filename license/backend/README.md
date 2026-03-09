# License Unified Backend

统一授权 + 版本系统后端（Go + Gin + Gorm）。

## 运行

```bash
cp .env.example .env
# 必须在 .env 中设置 BOOTSTRAP_ADMIN_PASSWORD
go mod tidy
go run ./cmd/server
```

默认端口：`8091`

默认管理员（首次启动自动创建，账号可改）：

- 用户名：`admin`
- 密码：来自 `BOOTSTRAP_ADMIN_PASSWORD`（必填，无默认值）

## Docker 镜像

后端 Dockerfile 位于：

- `backend/Dockerfile`

与前端组合部署请使用上级目录 `license/docker-compose.yml`。

## 核心接口

- `POST /api/v1/auth/login` 管理员登录
- `POST /api/v1/verify` 统一校验（授权 + 版本）
- `GET /api/v1/dashboard` 控制台统计（需登录）
- `GET/POST /api/v1/plans` 套餐管理（需登录）
- `PUT/DELETE /api/v1/plans/:id` 套餐编辑/删除（需登录）
- `POST /api/v1/plans/:id/clone` 套餐克隆（需登录）
- `POST /api/v1/licenses/generate` 授权生成（需登录）
- `GET /api/v1/licenses` 授权查询（需登录）
- `PUT /api/v1/licenses/:id` 授权编辑（需登录）
- `DELETE /api/v1/licenses/:id` 授权物理删除（需登录）
- `GET /api/v1/licenses/:id/activations` 查看设备绑定（需登录）
- `DELETE /api/v1/licenses/:id/activations/:activation_id` 解绑单设备（需登录）
- `DELETE /api/v1/licenses/:id/activations` 清空设备绑定（需登录）
- `POST /api/v1/licenses/batch/delete` 批量物理删除（需登录）
- `POST /api/v1/licenses/batch/revoke` 批量吊销（需登录）
- `POST /api/v1/licenses/batch/restore` 批量恢复（需登录）
- `POST /api/v1/licenses/batch/update` 批量更新（需登录，支持 `plan_id/status/expires_at/max_machines/metadata_json/note`）
- `GET/POST /api/v1/releases` 产品发布管理（需登录）
- `GET /api/v1/releases/recycle` 发布回收站列表（需登录）
- `GET /api/v1/version-sync/configs` GitHub 镜像同步配置列表（需登录，内置 backend/frontend/nodeclient）
- `GET/PUT /api/v1/version-sync/config` GitHub 镜像同步配置（需登录，默认 nodeclient，可通过 `product` 指定目标）
- `POST /api/v1/version-sync/manual` 手动触发 GitHub 镜像拉取（需登录，支持 `product` 指定目标）
- `PUT /api/v1/releases/:id` 更新发布记录（需登录，可做上线/下线）
- `POST /api/v1/releases/upload` 手动上传版本安装包并创建发布（需登录，`multipart/form-data`）
- `PUT /api/v1/releases/:id/file` 替换发布安装包（需登录，`multipart/form-data`）
- `GET /api/v1/releases/:id/file` 下载版本安装包（需登录）
- `DELETE /api/v1/releases/:id` 发布移入回收站（需登录）
- `POST /api/v1/releases/:id/restore` 从回收站恢复发布（需登录）
- `DELETE /api/v1/releases/:id/purge` 从回收站永久删除（需登录，同时物理删除安装包）
- `GET/POST /api/v1/version-policies` 版本策略管理（需登录）
- `PUT/DELETE /api/v1/version-policies/:id` 版本策略编辑/删除（需登录）
- `GET /api/v1/verify-logs` 校验日志（需登录）
- `POST /api/v1/commercial/trials/issue` 发放试用（需登录）
- `POST /api/v1/commercial/orders/renew|upgrade|transfer` 创建商业化订单（需登录）
- `GET /api/v1/commercial/orders` 订单列表（需登录）
- `POST /api/v1/commercial/orders/:id/mark-paid` 手动确认支付（需登录）
- `POST /api/v1/commercial/payments/callback/:channel` 支付回调（公开）

套餐状态字段：

- `status` 仅支持 `active` / `disabled`
- `DELETE /api/v1/plans/:id` 支持 `?force=true`：强制删除套餐并同时删除其关联授权
- `GET /api/v1/plans` 返回附加统计字段：
  - `license_count`：关联授权总数
  - `active_license_count`：活跃授权数
  - `activation_count`：设备绑定总数
- `POST /api/v1/plans/:id/clone` 支持可选字段：
  - `code`、`name`、`description`、`status`

版本发布安装包字段：

- `file_name`：安装包文件名
- `file_size`：安装包大小（字节）
- `file_sha256`：安装包 SHA256 摘要

上传目录配置：

- 环境变量 `RELEASE_UPLOAD_DIR`（默认 `./uploads/releases`）

GitHub 镜像同步配置关键字段：

- `product`：同步目标产品（仅支持 `backend` / `frontend` / `nodeclient`）
- `enabled`：是否启用镜像同步
- `auto_sync`：是否启用自动拉取
- `interval_minutes`：自动拉取间隔（分钟，最小 5）
- `github_owner` / `github_repo`：同步目标仓库
- `channel`：同步入库时映射到发布记录的渠道

## 商业化回调约束

- 支付回调支持状态：`paid` / `failed` / `canceled`
- 回调按 `order_no` 做幂等处理：重复 `paid` 回调不会重复执行授权变更
- 已 `paid` 的订单不允许被 `failed`/`canceled` 回退
- 当 `paid` 回调携带 `amount_cents` 且与订单金额不一致时，接口返回 `400`，订单保持原状态，并记录 `payment_amount_mismatch` 事件
- 签名头：
  - `X-Callback-Signature`
  - `X-Callback-Timestamp`（Unix 秒）
  - `X-Callback-Nonce`
- 签名原文：
  - `channel={channel}&order_no={order_no}&status={status}&amount_cents={amount_cents}&payment_txn_id={payment_txn_id}&timestamp={timestamp}&nonce={nonce}`
- 签名算法：`HMAC-SHA256(secret, signing_text)`，输出 hex 小写
- 订单状态机（一期收紧）：
  - `pending -> paid|failed|canceled`
- `paid|failed|canceled` 为终态，仅允许相同状态的重复回调（幂等）

## 授权编辑字段

`PUT /api/v1/licenses/:id` 当前支持编辑：

- `key`
- `plan_id`
- `customer`
- `status`（`active` / `revoked` / `expired`）
- `expires_at`
- `clear_expires_at`（`true` 时清空到期时间）
- `max_machines`
- `clear_max_machines`（`true` 时清空覆盖值，回退到套餐限制）
- `metadata_json`
- `note`

## 授权列表查询参数

`GET /api/v1/licenses` 支持参数：

- `page` / `page_size`
- `customer`
- `status`
- `plan_id`
- `expire_from`（RFC3339 或 `YYYY-MM-DD`）
- `expire_to`（RFC3339 或 `YYYY-MM-DD`）
- `sort_by`（`created_at` / `expires_at` / `status`）
- `sort_order`（`asc` / `desc`，未传默认 `desc`）
