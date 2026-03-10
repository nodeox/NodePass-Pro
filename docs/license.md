# 授权与版本统一校验接口约定

NodePass 的安装脚本、backend 运行时授权、nodeclient 启动校验均可对接统一接口。

## 接口地址

- 推荐：`POST /api/v1/verify`
- 示例：`http://127.0.0.1:8091/api/v1/verify`
- 兼容：旧 `POST /api/v1/license/verify` 仍可使用（backend 已做兼容解析）

## 请求体（统一接口）

```json
{
  "license_key": "NP-XXXX-XXXX",
  "machine_id": "7f4b8d...",
  "hostname": "nodepass-backend",
  "product": "backend",
  "client_version": "1.0.0",
  "channel": "stable"
}
```

补充：`domain/site_url/action/versions/branch/commit` 在新接口中为可选兼容字段，传递也不会报错。

## 响应体（统一接口）

```json
{
  "success": true,
  "data": {
    "verified": true,
    "status": "ok",
    "license": {
      "valid": true,
      "license_id": 1,
      "plan_code": "NP-STD",
      "customer": "Example Inc",
      "expires_at": "2027-03-01T00:00:00Z"
    },
    "version": {
      "compatible": true,
      "status": "upgrade_available",
      "latest_version": "1.2.0",
      "message": "建议升级到 1.2.0"
    }
  }
}
```

## backend 实际调用配置

对应环境变量（`docker-compose.yml` 已接入）：

- `BACKEND_LICENSE_VERIFY_URL`
- `BACKEND_LICENSE_PRODUCT`（默认 `backend`）
- `BACKEND_LICENSE_CHANNEL`（默认 `stable`）
- `BACKEND_LICENSE_CLIENT_VERSION`（可选）
- `BACKEND_LICENSE_REQUIRE_DOMAIN`（默认 `false`）

## nodeclient 实际调用配置

`nodeclient/configs/config.yaml` 可选项：

- `license_enabled`
- `license_verify_url`
- `license_key`
- `license_machine_id`
- `license_product`（默认 `nodeclient`）
- `license_channel`（默认 `stable`）
- `license_timeout`
- `license_fail_open`

当 `license_enabled=true` 时，nodeclient 启动前会执行一次统一校验；失败时启动终止（`license_fail_open=true` 时仅在接口不可达/解析失败场景放行）。
