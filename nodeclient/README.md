# NodePass NodeClient

NodeClient 是 NodePass Panel 的节点侧代理程序，负责配置拉取、心跳上报与流量上报。

## 一键安装（推荐）

> 需在目标节点以 `root` 用户执行，且系统为 Linux `amd64/arm64`。

```bash
curl -fsSL https://your-panel.com/install.sh | bash -s -- \
  --hub-url https://your-panel.com \
  --token node_xxx
```

可选安装目录：

```bash
curl -fsSL https://your-panel.com/install.sh | bash -s -- \
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
