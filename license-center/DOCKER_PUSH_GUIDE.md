# Docker 镜像构建和推送指南

## 镜像信息

✅ **镜像已构建成功**

- 镜像名称: `nodepass/license-center`
- 标签: `1.0.0`, `15cb538-dirty`
- 大小: 128MB
- 构建时间: 2026-03-09

## 镜像标签说明

- `nodepass/license-center:1.0.0` - 指定版本标签
- `nodepass/license-center:15cb538-dirty` - Git 提交哈希标签
- `nodepass/license-center:latest` - 最新版本标签

## 推送到 Docker Hub

### 1. 登录 Docker Hub

```bash
docker login
# 输入用户名和密码
```

### 2. 标记镜像（如果需要更改仓库名）

```bash
# 如果要推送到自己的 Docker Hub 账号
docker tag nodepass/license-center:1.0.0 YOUR_USERNAME/license-center:1.0.0
docker tag nodepass/license-center:1.0.0 YOUR_USERNAME/license-center:latest
```

### 3. 推送镜像

```bash
# 推送到 Docker Hub
docker push nodepass/license-center:1.0.0
docker push nodepass/license-center:latest

# 或推送到自己的账号
docker push YOUR_USERNAME/license-center:1.0.0
docker push YOUR_USERNAME/license-center:latest
```

## 推送到私有镜像仓库

### 阿里云容器镜像服务

```bash
# 1. 登录阿里云镜像仓库
docker login --username=YOUR_USERNAME registry.cn-hangzhou.aliyuncs.com

# 2. 标记镜像
docker tag nodepass/license-center:1.0.0 registry.cn-hangzhou.aliyuncs.com/YOUR_NAMESPACE/license-center:1.0.0

# 3. 推送镜像
docker push registry.cn-hangzhou.aliyuncs.com/YOUR_NAMESPACE/license-center:1.0.0
```

### 腾讯云容器镜像服务

```bash
# 1. 登录腾讯云镜像仓库
docker login --username=YOUR_USERNAME ccr.ccs.tencentyun.com

# 2. 标记镜像
docker tag nodepass/license-center:1.0.0 ccr.ccs.tencentyun.com/YOUR_NAMESPACE/license-center:1.0.0

# 3. 推送镜像
docker push ccr.ccs.tencentyun.com/YOUR_NAMESPACE/license-center:1.0.0
```

### Harbor 私有仓库

```bash
# 1. 登录 Harbor
docker login harbor.example.com

# 2. 标记镜像
docker tag nodepass/license-center:1.0.0 harbor.example.com/nodepass/license-center:1.0.0

# 3. 推送镜像
docker push harbor.example.com/nodepass/license-center:1.0.0
```

## 使用推送脚本

### 快速推送到 Docker Hub

```bash
# 使用完整的构建和推送脚本
./build-and-push-docker.sh nodepass/license-center 1.0.0

# 或使用自己的账号
./build-and-push-docker.sh YOUR_USERNAME/license-center 1.0.0
```

### 推送到私有仓库

```bash
# 修改脚本中的镜像名称
./build-and-push-docker.sh registry.cn-hangzhou.aliyuncs.com/YOUR_NAMESPACE/license-center 1.0.0
```

## 拉取和使用镜像

### 从 Docker Hub 拉取

```bash
# 拉取镜像
docker pull nodepass/license-center:1.0.0

# 运行容器
docker run -d \
  --name license-center \
  -p 8090:8090 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/configs:/app/configs \
  -e JWT_SECRET="your-secret-key" \
  -e ADMIN_PASSWORD="your-password" \
  --restart unless-stopped \
  nodepass/license-center:1.0.0
```

### 使用 docker-compose

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  license-center:
    image: nodepass/license-center:1.0.0
    container_name: license-center
    ports:
      - "8090:8090"
    volumes:
      - ./data:/app/data
      - ./configs:/app/configs
    environment:
      - JWT_SECRET=${JWT_SECRET}
      - ADMIN_PASSWORD=${ADMIN_PASSWORD}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8090/health"]
      interval: 30s
      timeout: 5s
      retries: 3
```

启动：

```bash
# 设置环境变量
export JWT_SECRET="your-secret-key"
export ADMIN_PASSWORD="your-password"

# 启动服务
docker-compose up -d
```

## 验证镜像

### 查看镜像信息

```bash
# 查看镜像
docker images nodepass/license-center

# 查看镜像详细信息
docker inspect nodepass/license-center:1.0.0

# 查看镜像标签
docker inspect nodepass/license-center:1.0.0 | grep -A 10 Labels
```

### 测试镜像

```bash
# 启动测试容器
docker run -d \
  --name license-center-test \
  -p 18090:8090 \
  -e JWT_SECRET="test-secret" \
  -e ADMIN_PASSWORD="TestPassword123!" \
  nodepass/license-center:1.0.0

# 等待启动
sleep 10

# 健康检查
curl http://localhost:18090/health

# 查看版本信息
curl http://localhost:18090/

# 停止测试容器
docker rm -f license-center-test
```

## 镜像版本信息

构建的镜像包含以下版本信息：

- **Version**: 15cb538-dirty (Git 描述)
- **Git Commit**: 15cb538bb72d100d2fda303acf733f19acc84658
- **Git Branch**: main
- **Build Time**: 2026-03-09T02:37:51Z

可以通过以下方式查看：

```bash
# 运行容器并查看版本
docker run --rm nodepass/license-center:1.0.0 /usr/local/bin/license-center --version

# 或通过 API 查看
curl http://localhost:8090/
```

## 多架构支持

当前镜像支持：
- ✅ linux/arm64 (Apple Silicon)
- ⚠️ linux/amd64 (需要重新构建)

### 构建多架构镜像

```bash
# 创建并使用 buildx builder
docker buildx create --name multiarch --use

# 构建并推送多架构镜像
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --file Dockerfile.version \
  --build-arg VERSION="1.0.0" \
  --build-arg GIT_COMMIT="$(git rev-parse HEAD)" \
  --build-arg GIT_BRANCH="$(git branch --show-current)" \
  --build-arg BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --tag nodepass/license-center:1.0.0 \
  --tag nodepass/license-center:latest \
  --push \
  .
```

## 常见问题

### 1. 推送失败：unauthorized

```bash
# 重新登录
docker logout
docker login
```

### 2. 推送失败：denied

检查仓库权限，确保有推送权限。

### 3. 镜像太大

当前镜像已优化到 128MB，使用了：
- 多阶段构建
- 精简基础镜像 (debian:bookworm-slim)
- 静态编译 (CGO_ENABLED=0)

### 4. 版本信息不正确

确保在干净的 Git 仓库中构建：

```bash
# 提交所有更改
git add .
git commit -m "Update"

# 创建标签
git tag v1.0.0

# 重新构建
./quick-build.sh 1.0.0
```

## 下一步

1. ✅ 镜像已构建成功
2. ⏭️ 推送到镜像仓库
3. ⏭️ 在生产环境部署
4. ⏭️ 配置 CI/CD 自动构建

## 相关文档

- `build-and-push-docker.sh` - 完整的构建和推送脚本
- `quick-build.sh` - 快速构建脚本
- `Dockerfile.version` - 支持版本注入的 Dockerfile
- `docker-compose.deploy.yml` - 部署配置
