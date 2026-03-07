# NodePass License Center

独立授权管理系统（独立仓可部署），用于 NodePass Pro 的授权校验与授权码管理。

## 功能

- 授权校验接口：`POST /api/v1/license/verify`
- 管理员登录与 JWT 认证
- 套餐管理（版本策略、机器上限、有效期）
- 授权码批量生成、吊销/恢复、删除
- 机器绑定管理（解绑）
- 验证日志与统计
- 时间限制生效：授权码到期后自动拒绝验证，并定时标记为 `expired`

## 快速部署

```bash
git clone <your-license-center-repo>
cd license-center
./scripts/deploy.sh
```

默认地址：`http://127.0.0.1:8090`

## 远程一键安装/升级/卸载

```bash
# 安装
bash <(curl -fsSL "https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license-center/install.sh?t=$(date +%s)") --install

# 升级
bash <(curl -fsSL "https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license-center/install.sh?t=$(date +%s)") --upgrade

# 卸载
bash <(curl -fsSL "https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license-center/install.sh?t=$(date +%s)") --uninstall
```

## 默认管理员

- 用户名：`admin`
- 密码：`ChangeMe123!`

请首次登录后立即修改 `configs/config.yaml` 中的管理员密码与 JWT Secret。

## 关键接口

### 1) 管理员登录

`POST /api/v1/auth/login`

```json
{
  "username": "admin",
  "password": "ChangeMe123!"
}
```

### 2) 授权验证（给安装脚本调用）

`POST /api/v1/license/verify`

```json
{
  "license_key": "NP-XXXX-XXXX-XXXX",
  "machine_id": "abcdef123456",
  "machine_name": "prod-node-01",
  "action": "install",
  "versions": {
    "panel": "0.1.0",
    "backend": "0.1.0",
    "frontend": "0.1.0",
    "nodeclient": "0.1.0"
  },
  "branch": "main",
  "commit": "abc1234"
}
```

响应示例：

```json
{
  "success": true,
  "data": {
    "valid": true,
    "message": "ok",
    "license_id": 1,
    "plan": "enterprise",
    "customer": "ACME",
    "expires_at": "2027-01-01T00:00:00Z",
    "version_policy": {
      "min_panel_version": "0.1.0",
      "max_panel_version": "9.9.9"
    }
  },
  "message": "ok",
  "timestamp": "2026-03-07T00:00:00Z"
}
```

## 与 NodePass Pro 对接

在 NodePass Pro 安装脚本中指定授权服务地址：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/install.sh) \
  --license-server https://license.yourdomain.com/api/v1/license/verify \
  --license-key NP-XXXX-XXXX-XXXX
```
