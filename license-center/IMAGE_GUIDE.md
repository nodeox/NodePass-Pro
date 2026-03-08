# Docker 镜像使用指南

## 概述

NodePass License Center 支持两种部署方式：
1. **源码构建** - 从源码构建 Docker 镜像（默认）
2. **预构建镜像** - 使用预构建的 Docker 镜像（推荐生产环境）

## 一、构建镜像

### 方式一：使用 Makefile

```bash
# 构建镜像（用于本地测试）
make build

# 构建并保存为文件
make build-image

# 构建多架构镜像（需要推送到仓库）
make multi-arch
```

### 方式二：使用构建脚本

```bash
# 构建镜像
./scripts/build-image.sh --build

# 构建并保存为 tar.gz 文件
./scripts/build-image.sh --save

# 自定义版本和名称
./scripts/build-image.sh --save --version 0.3.0 --image-name mycompany/license-center

# 构建多架构镜像
./scripts/build-image.sh --multi-arch \
  --platform linux/amd64,linux/arm64 \
  --registry registry.example.com \
  --push
```

### 构建输出

构建完成后，镜像文件保存在 `dist/` 目录：

```
dist/
├── license-center-0.3.0.tar.gz         # 镜像文件
└── license-center-0.3.0.tar.gz.sha256  # 校验和文件
```

## 二、加载镜像

### 从本地文件加载

```bash
# 使用构建脚本
./scripts/build-image.sh --load

# 或使用 Makefile
make load-image

# 或使用 Docker 命令
gunzip -c dist/license-center-0.3.0.tar.gz | docker load
```

### 从 Docker Hub 拉取

```bash
# 拉取指定版本
docker pull nodepass/license-center:0.3.0

# 拉取最新版本
docker pull nodepass/license-center:latest
```

### 从私有仓库拉取

```bash
# 登录私有仓库
docker login registry.example.com

# 拉取镜像
docker pull registry.example.com/nodepass/license-center:0.3.0
```

## 三、使用预构建镜像部署

### 方式一：使用 install.sh（推荐）

#### 从 Docker Hub 拉取

```bash
# 使用预构建镜像安装
bash install.sh --install --use-image

# 指定版本
bash install.sh --install --use-image --image-version 0.3.0
```

#### 从 URL 下载

```bash
# 从 URL 下载镜像文件
bash install.sh --install --use-image \
  --image-url https://example.com/license-center-0.3.0.tar.gz
```

#### 从本地文件

```bash
# 使用本地镜像文件
bash install.sh --install --use-image \
  --image-file /path/to/license-center-0.3.0.tar.gz
```

#### 从私有仓库

```bash
# 使用私有仓库镜像
bash install.sh --install --use-image \
  --image-name registry.example.com/nodepass/license-center \
  --image-version 0.3.0
```

### 方式二：使用 docker-compose.prod.yml

#### 1. 准备配置文件

```bash
# 复制环境变量模板
cp .env.prod.example .env

# 编辑配置
vim .env
```

配置示例：

```bash
# 数据库配置
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_secure_password
POSTGRES_DB=nodepass_license

# 应用配置
APP_PORT=8090
BUILD_VERSION=0.3.0

# 镜像配置
IMAGE_NAME=nodepass/license-center
# REGISTRY=registry.example.com  # 如果使用私有仓库
```

#### 2. 启动服务

```bash
# 使用生产配置启动
docker compose -f docker-compose.prod.yml up -d

# 查看日志
docker compose -f docker-compose.prod.yml logs -f

# 查看状态
docker compose -f docker-compose.prod.yml ps
```

### 方式三：直接运行容器

```bash
# 运行容器
docker run -d \
  --name license-center \
  -p 8090:8090 \
  -v $(pwd)/configs/config.yaml:/app/configs/config.yaml:ro \
  -e TZ=Asia/Shanghai \
  -e GIN_MODE=release \
  nodepass/license-center:0.3.0

# 查看日志
docker logs -f license-center

# 停止容器
docker stop license-center
```

## 四、推送镜像到仓库

### 推送到 Docker Hub

```bash
# 登录 Docker Hub
docker login

# 构建并推送
./scripts/build-image.sh --push --registry docker.io

# 或使用 Makefile
make push-image
```

### 推送到私有仓库

```bash
# 登录私有仓库
docker login registry.example.com

# 构建并推送
./scripts/build-image.sh --push \
  --registry registry.example.com \
  --image-name mycompany/license-center \
  --version 0.3.0
```

### 推送多架构镜像

```bash
# 构建并推送多架构镜像
./scripts/build-image.sh --multi-arch \
  --platform linux/amd64,linux/arm64 \
  --registry registry.example.com \
  --image-name mycompany/license-center \
  --version 0.3.0
```

## 五、镜像管理

### 查看镜像

```bash
# 查看本地镜像
docker images nodepass/license-center

# 查看镜像详情
docker inspect nodepass/license-center:0.3.0

# 查看镜像历史
docker history nodepass/license-center:0.3.0
```

### 删除镜像

```bash
# 删除指定版本
docker rmi nodepass/license-center:0.3.0

# 删除所有版本
docker rmi $(docker images nodepass/license-center -q)

# 清理未使用的镜像
docker image prune -a
```

### 导出/导入镜像

```bash
# 导出镜像
docker save nodepass/license-center:0.3.0 | gzip > license-center-0.3.0.tar.gz

# 导入镜像
gunzip -c license-center-0.3.0.tar.gz | docker load

# 传输到其他服务器
scp license-center-0.3.0.tar.gz user@server:/tmp/
ssh user@server "gunzip -c /tmp/license-center-0.3.0.tar.gz | docker load"
```

## 六、镜像验证

### 验证镜像完整性

```bash
# 验证 SHA256 校验和
sha256sum -c dist/license-center-0.3.0.tar.gz.sha256

# 或使用 shasum（macOS）
shasum -a 256 -c dist/license-center-0.3.0.tar.gz.sha256
```

### 测试镜像

```bash
# 运行健康检查
docker run --rm nodepass/license-center:0.3.0 curl -f http://localhost:8090/health

# 查看版本信息
docker run --rm nodepass/license-center:0.3.0 /usr/local/bin/license-center --version

# 进入容器检查
docker run --rm -it nodepass/license-center:0.3.0 /bin/bash
```

## 七、常见问题

### Q: 如何选择部署方式？

**源码构建：**
- 适合开发环境
- 需要修改代码
- 构建时间较长

**预构建镜像：**
- 适合生产环境
- 快速部署
- 镜像体积小
- 部署一致性好

### Q: 镜像文件太大怎么办？

```bash
# 使用压缩
gzip -9 license-center-0.3.0.tar

# 使用 Docker 导出（自动压缩）
docker save nodepass/license-center:0.3.0 | gzip -9 > license-center-0.3.0.tar.gz

# 清理构建缓存
docker builder prune -a
```

### Q: 如何更新镜像？

```bash
# 拉取最新镜像
docker pull nodepass/license-center:latest

# 重启服务
docker compose -f docker-compose.prod.yml up -d

# 或使用 install.sh 升级
bash install.sh --upgrade --use-image
```

### Q: 如何回滚到旧版本？

```bash
# 修改 .env 文件中的版本号
BUILD_VERSION=0.2.0

# 重启服务
docker compose -f docker-compose.prod.yml up -d
```

### Q: 多架构镜像如何使用？

```bash
# Docker 会自动选择匹配的架构
docker pull nodepass/license-center:0.3.0

# 查看镜像支持的架构
docker manifest inspect nodepass/license-center:0.3.0
```

## 八、最佳实践

### 生产环境建议

1. **使用预构建镜像**
   ```bash
   bash install.sh --install --use-image
   ```

2. **使用固定版本号**
   ```bash
   BUILD_VERSION=0.3.0  # 不要使用 latest
   ```

3. **使用私有仓库**
   ```bash
   REGISTRY=registry.example.com
   IMAGE_NAME=mycompany/license-center
   ```

4. **定期备份镜像**
   ```bash
   # 定期导出镜像
   docker save nodepass/license-center:0.3.0 | gzip > backup/license-center-0.3.0.tar.gz
   ```

5. **验证镜像完整性**
   ```bash
   # 使用 SHA256 校验
   sha256sum -c license-center-0.3.0.tar.gz.sha256
   ```

### 开发环境建议

1. **使用源码构建**
   ```bash
   make up
   ```

2. **使用 latest 标签**
   ```bash
   docker pull nodepass/license-center:latest
   ```

3. **频繁更新镜像**
   ```bash
   docker compose pull
   docker compose up -d
   ```

## 九、镜像信息

### 镜像规格

- **基础镜像：** debian:bookworm-slim
- **构建方式：** 多阶段构建
- **镜像大小：** ~200MB（压缩后 ~80MB）
- **支持架构：** linux/amd64, linux/arm64
- **运行用户：** appuser (uid=1000)
- **暴露端口：** 8090

### 镜像标签

- `0.3.0` - 稳定版本
- `0.3` - 次版本
- `0` - 主版本
- `latest` - 最新版本

### 镜像仓库

- **Docker Hub：** `docker.io/nodepass/license-center`
- **GitHub：** `ghcr.io/nodeox/license-center`
- **私有仓库：** 根据配置

## 十、技术支持

- **文档：** [README.md](./README.md)
- **部署指南：** [DEPLOYMENT.md](./DEPLOYMENT.md)
- **问题反馈：** [GitHub Issues](https://github.com/nodeox/NodePass-Pro/issues)
