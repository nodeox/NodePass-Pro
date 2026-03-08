# Docker 镜像使用指南（v0.4.0）

## 概述

NodePass License Center 推荐优先使用 GitHub Container Registry（GHCR）预构建多架构镜像：

- 镜像仓库：`ghcr.io/nodeox/license-center`
- 支持架构：`linux/amd64`、`linux/arm64`
- 推荐标签：`main`（主分支构建）、`latest`（默认分支最新稳定构建）

## 一、快速部署（推荐）

### 1. 一键安装（预构建镜像）

```bash
bash <(curl -fsSL "https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license-center/install.sh?t=$(date +%s)") --install
```

### 2. 非交互自动化安装

```bash
bash <(curl -fsSL "https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license-center/install.sh?t=$(date +%s)") \
  --install --non-interactive \
  --image-name ghcr.io/nodeox/license-center \
  --image-version main \
  --admin-username admin --admin-email admin@example.com
```

### 3. 非交互 + 域名 HTTPS

```bash
bash <(curl -fsSL "https://raw.githubusercontent.com/nodeox/NodePass-Pro/main/license-center/install.sh?t=$(date +%s)") \
  --install --non-interactive --enable-caddy \
  --domain license.example.com --cert-email ops@example.com \
  --admin-username admin --admin-email admin@example.com
```

## 二、直接使用镜像

### 1. 拉取 GHCR 镜像

```bash
# 主分支构建（推荐）
docker pull ghcr.io/nodeox/license-center:main

# 默认分支最新镜像
docker pull ghcr.io/nodeox/license-center:latest
```

### 2. 查看多架构清单

```bash
docker manifest inspect ghcr.io/nodeox/license-center:main
```

### 3. 使用 docker-compose.prod.yml 启动

```bash
cp .env.prod.example .env
```

`.env` 示例：

```bash
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_secure_password
POSTGRES_DB=nodepass_license
POSTGRES_PORT=5432

APP_BIND=0.0.0.0
APP_PORT=8090
BUILD_VERSION=main
GIN_MODE=release
IMAGE_NAME=ghcr.io/nodeox/license-center

# 可选 HTTPS 代理
ENABLE_HTTPS_PROXY=false
CADDY_DOMAIN=
CADDY_EMAIL=
CADDY_HTTP_PORT=80
CADDY_HTTPS_PORT=443
```

启动：

```bash
docker compose -f docker-compose.prod.yml up -d
docker compose -f docker-compose.prod.yml logs -f
```

## 三、离线部署（本地镜像文件）

### 1. 构建并保存镜像

```bash
./scripts/build-image.sh --save --version main
```

默认输出：

```text
dist/
├── license-center-main.tar.gz
└── license-center-main.tar.gz.sha256
```

### 2. 在目标机器加载

```bash
./scripts/build-image.sh --load --version main
# 或
gunzip -c dist/license-center-main.tar.gz | docker load
```

### 3. 一键安装时指定本地文件

```bash
bash install.sh --install --use-image --image-file /path/to/license-center-main.tar.gz
```

## 四、自定义镜像构建

### 1. 本地构建

```bash
./scripts/build-image.sh --build --version main
```

### 2. 推送到私有仓库

```bash
./scripts/build-image.sh --push \
  --registry registry.example.com \
  --image license-center \
  --version main
```

### 3. 构建并推送多架构

```bash
./scripts/build-image.sh --multi-arch \
  --platform linux/amd64,linux/arm64 \
  --registry registry.example.com \
  --image license-center \
  --version main
```

## 五、回滚与版本固定

### 回滚到指定标签

```bash
# 例：如果你发布了自定义稳定标签
BUILD_VERSION=v0.4.0
```

然后重启：

```bash
docker compose -f docker-compose.prod.yml up -d
```

### 生产建议

- 使用固定标签（例如 `v0.4.0` 或内部发布号），避免长期使用 `latest`
- 首次部署可用 `main` 验证，稳定后切换到固定版本标签
- 保留最近 1-2 个可回滚镜像包与校验文件

## 六、常见问题

### Q: 为什么推荐 GHCR？

A: 与仓库 Actions 发布流程直接打通，主分支变更可自动生成最新多架构镜像。

### Q: 如何确认拉到的是本机架构镜像？

A: 使用 `docker manifest inspect` 查看清单，Docker 会自动选择匹配架构。

### Q: 开启 HTTPS 一定要自己写 Nginx 吗？

A: 不需要。`install.sh` 已支持 `--enable-caddy` 自动生成反代并申请证书。

## 七、相关文档

- [README.md](./README.md)
- [DEPLOYMENT.md](./DEPLOYMENT.md)
- [CHANGELOG.md](./CHANGELOG.md)
