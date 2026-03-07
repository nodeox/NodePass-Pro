# NodeClient 下载目录说明

该目录用于通过 Caddy 对外提供节点客户端二进制与校验文件。

## 自动构建（推荐）

在仓库根目录执行：

```bash
./scripts/build-nodeclient-downloads.sh
```

脚本会自动生成：

- `nodeclient-linux-amd64`
- `nodeclient-linux-amd64.sha256`
- `nodeclient-linux-arm64`
- `nodeclient-linux-arm64.sha256`

并输出到 `deploy/nodeclient/downloads/`。

## 目录结构

请将以下文件放到 `downloads/` 目录：

- `nodeclient-linux-amd64`
- `nodeclient-linux-amd64.sha256`
- `nodeclient-linux-arm64`
- `nodeclient-linux-arm64.sha256`

## 生成校验文件示例

```bash
cd deploy/nodeclient/downloads
sha256sum nodeclient-linux-amd64 > nodeclient-linux-amd64.sha256
sha256sum nodeclient-linux-arm64 > nodeclient-linux-arm64.sha256
```

## 对外访问路径

启用 Caddy 后可通过以下地址访问：

- `https://<前端域名>/nodeclient-install.sh`
- `https://<前端域名>/downloads/nodeclient-linux-amd64`
- `https://<前端域名>/downloads/nodeclient-linux-arm64`
