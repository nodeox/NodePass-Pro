# NodePass NodeClient

NodeClient 是 NodePass Panel 的节点侧代理程序，负责配置拉取、心跳上报与流量上报。

## 一键安装（推荐）

> 需在目标节点以 `root` 用户执行，且系统为 Linux `amd64/arm64`。

```bash
curl -fsSL https://your-panel.com/nodeclient-install.sh | bash -s -- \
  --hub-url https://your-panel.com \
  --token node_xxx
```

可选安装目录：

```bash
curl -fsSL https://your-panel.com/nodeclient-install.sh | bash -s -- \
  --hub-url https://your-panel.com \
  --token node_xxx \
  --install-dir /opt/nodeclient
```

## 本地脚本安装

```bash
sudo bash ./scripts/install.sh \
  --hub-url https://your-panel.com \
  --token node_xxx
```

## 卸载

```bash
sudo bash ./scripts/install.sh --uninstall
```

若使用自定义安装目录：

```bash
sudo bash ./scripts/install.sh --uninstall --install-dir /opt/nodeclient
```

## 升级

```bash
sudo bash ./scripts/install.sh --upgrade
```

若配置文件不存在或需指定面板地址：

```bash
sudo bash ./scripts/install.sh --upgrade --hub-url https://your-panel.com
```

## 查看客户端版本

```bash
sudo bash ./scripts/install.sh --version
```

## 对接统一授权接口（可选）

在 `config.yaml` 中启用以下配置，可让 nodeclient 启动前执行一次授权+版本统一校验：

```yaml
license_enabled: true
license_verify_url: "http://127.0.0.1:8091/api/v1/verify"
license_key: "NP-XXXX-XXXX"
license_product: "nodeclient"
license_channel: "stable"
license_timeout: 10
license_fail_open: false
```

说明：

- `license_enabled=false`（默认）时不执行授权校验；
- 校验失败会阻止 nodeclient 启动；
- 当授权接口不可达且 `license_fail_open=true` 时，允许继续启动。
