# 授权验证接口约定

安装脚本在执行正式部署前会调用授权接口：

- 默认地址：`https://license.nodepass.pro/api/v1/license/verify`
- 可通过参数覆盖：`--license-server <URL>`

## 请求体

```json
{
  "license_key": "NP-XXXX-XXXX",
  "machine_id": "7f4b8d...",
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

## 响应体（推荐）

```json
{
  "success": true,
  "data": {
    "valid": true,
    "license_id": "LIC-2026-0001",
    "customer": "Example Inc",
    "plan": "enterprise",
    "expires_at": "2027-03-01T00:00:00Z",
    "version_policy": {
      "min_panel_version": "0.1.0",
      "max_panel_version": "1.9.9",
      "min_backend_version": "0.1.0",
      "max_backend_version": "1.9.9",
      "min_frontend_version": "0.1.0",
      "max_frontend_version": "1.9.9",
      "min_nodeclient_version": "0.1.0",
      "max_nodeclient_version": "1.9.9"
    },
    "message": "ok"
  }
}
```

说明：

- `data.valid=false` 时，安装脚本会立即终止；
- 即使接口返回通过，脚本仍会基于 `version_policy` 进行本地二次版本校验；
- 通过校验后，安装目录会写入 `.nodepass-license` 快照文件。
