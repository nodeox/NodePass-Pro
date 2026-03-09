# 推送到 GitHub Container Registry 指南

## 📦 GitHub Container Registry (ghcr.io)

GitHub Container Registry 是 GitHub 提供的容器镜像托管服务，与 GitHub 仓库深度集成。

## 🔑 准备工作

### 1. 创建 GitHub Personal Access Token (PAT)

1. 访问 GitHub Token 设置页面：
   ```
   https://github.com/settings/tokens
   ```

2. 点击 **"Generate new token (classic)"**

3. 设置 Token 信息：
   - **Note**: `Docker Push Token` (或其他描述)
   - **Expiration**: 选择过期时间（建议 90 天或更长）
   - **Select scopes**: 勾选以下权限
     - ✅ `write:packages` - 上传容器镜像
     - ✅ `read:packages` - 读取容器镜像
     - ✅ `delete:packages` - 删除容器镜像（可选）
     - ✅ `repo` - 如果镜像关联到私有仓库

4. 点击 **"Generate token"**

5. **重要**: 立即复制生成的 token（只显示一次）

### 2. 登录 GitHub Container Registry

```bash
# 设置 token 为环境变量
export CR_PAT=YOUR_GITHUB_TOKEN

# 登录 ghcr.io
echo $CR_PAT | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

示例：
```bash
export CR_PAT=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
echo $CR_PAT | docker login ghcr.io -u jianshe --password-stdin
```

成功后会显示：
```
Login Succeeded
```

## 🚀 推送镜像

### 方式 1: 使用推送脚本（推荐）

```bash
cd /Users/jianshe/Projects/NodePass-Pro/license-center

# 推送镜像
./push-to-github.sh YOUR_GITHUB_USERNAME 1.0.0

# 示例
./push-to-github.sh jianshe 1.0.0
```

### 方式 2: 手动推送

```bash
# 1. 标记镜像
docker tag nodepass/license-center:1.0.0 ghcr.io/YOUR_USERNAME/license-center:1.0.0
docker tag nodepass/license-center:1.0.0 ghcr.io/YOUR_USERNAME/license-center:latest

# 2. 推送镜像
docker push ghcr.io/YOUR_USERNAME/license-center:1.0.0
docker push ghcr.io/YOUR_USERNAME/license-center:latest
```

## 📋 推送过程

推送时会看到类似输出：

```
🚀 推送到 GitHub Container Registry

[INFO] 源镜像: nodepass/license-center:1.0.0
[INFO] 目标镜像: ghcr.io/jianshe/license-center:1.0.0
[INFO] 最新标签: ghcr.io/jianshe/license-center:latest

[STEP] 标记镜像...
[INFO] 镜像已标记

[STEP] 推送镜像到 GitHub Container Registry...

[INFO] 推送: ghcr.io/jianshe/license-center:1.0.0
The push refers to repository [ghcr.io/jianshe/license-center]
...
1.0.0: digest: sha256:xxx size: 1234
[INFO] ✅ 推送成功

[INFO] 推送: ghcr.io/jianshe/license-center:latest
...
latest: digest: sha256:xxx size: 1234
[INFO] ✅ 推送成功

==========================================
推送完成!
==========================================
```

## 🔓 设置镜像为公开

默认情况下，推送的镜像是私有的。要设置为公开：

1. 访问你的 GitHub Packages 页面：
   ```
   https://github.com/YOUR_USERNAME?tab=packages
   ```

2. 点击 **"license-center"** 包

3. 点击右上角的 **"Package settings"**

4. 滚动到 **"Danger Zone"** 部分

5. 点击 **"Change visibility"**

6. 选择 **"Public"**

7. 输入包名确认：`license-center`

8. 点击 **"I understand, change package visibility"**

## 📥 拉取和使用镜像

### 拉取公开镜像

```bash
# 拉取镜像（无需登录）
docker pull ghcr.io/YOUR_USERNAME/license-center:1.0.0

# 运行容器
docker run -d \
  --name license-center \
  -p 8090:8090 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/configs:/app/configs \
  -e JWT_SECRET="your-secret-key" \
  -e ADMIN_PASSWORD="your-password" \
  --restart unless-stopped \
  ghcr.io/YOUR_USERNAME/license-center:1.0.0
```

### 拉取私有镜像

```bash
# 先登录
export CR_PAT=YOUR_TOKEN
echo $CR_PAT | docker login ghcr.io -u YOUR_USERNAME --password-stdin

# 拉取镜像
docker pull ghcr.io/YOUR_USERNAME/license-center:1.0.0
```

## 🔗 关联到 GitHub 仓库

将容器镜像关联到 GitHub 仓库：

1. 在 Package 页面点击 **"Connect repository"**

2. 选择对应的仓库（如 `nodepass-pro`）

3. 点击 **"Connect repository"**

关联后的好处：
- 镜像会显示在仓库的 Packages 标签页
- 可以在 README 中显示镜像徽章
- 更好的可见性和管理

## 📊 查看镜像信息

### 在 GitHub 上查看

访问：
```
https://github.com/YOUR_USERNAME?tab=packages
```

或直接访问包页面：
```
https://github.com/users/YOUR_USERNAME/packages/container/license-center
```

### 使用 Docker 命令查看

```bash
# 查看本地镜像
docker images | grep ghcr.io

# 查看镜像详情
docker inspect ghcr.io/YOUR_USERNAME/license-center:1.0.0
```

## 🏷️ 镜像标签管理

### 推送多个标签

```bash
# 推送版本标签
docker push ghcr.io/YOUR_USERNAME/license-center:1.0.0

# 推送 latest 标签
docker push ghcr.io/YOUR_USERNAME/license-center:latest

# 推送其他标签
docker tag nodepass/license-center:1.0.0 ghcr.io/YOUR_USERNAME/license-center:stable
docker push ghcr.io/YOUR_USERNAME/license-center:stable
```

### 删除标签

在 GitHub Package 页面：
1. 点击包名
2. 点击 **"Package settings"**
3. 在 **"Manage versions"** 中选择要删除的版本
4. 点击 **"Delete"**

## 🔐 在 CI/CD 中使用

### GitHub Actions 示例

创建 `.github/workflows/docker-publish.yml`:

```yaml
name: Publish Docker Image

on:
  push:
    tags:
      - 'v*'

jobs:
  push:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - uses: actions/checkout@v3

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v2
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v4
        with:
          images: ghcr.io/${{ github.repository_owner }}/license-center

      - name: Build and push
        uses: docker/build-push-action@v4
        with:
          context: .
          file: ./Dockerfile.version
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          build-args: |
            VERSION=${{ github.ref_name }}
            GIT_COMMIT=${{ github.sha }}
            GIT_BRANCH=${{ github.ref_name }}
            BUILD_TIME=${{ github.event.head_commit.timestamp }}
```

## 📝 docker-compose 配置

使用 GitHub Container Registry 的镜像：

```yaml
version: '3.8'

services:
  license-center:
    image: ghcr.io/YOUR_USERNAME/license-center:1.0.0
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
```

## ❓ 常见问题

### 1. 推送失败：unauthorized

**原因**: Token 权限不足或已过期

**解决**:
```bash
# 重新登录
docker logout ghcr.io
export CR_PAT=NEW_TOKEN
echo $CR_PAT | docker login ghcr.io -u YOUR_USERNAME --password-stdin
```

### 2. 推送失败：denied

**原因**: 包名已存在且无权限

**解决**:
- 检查包名是否正确
- 确认有推送权限
- 如果是组织仓库，确保有组织的 packages 权限

### 3. 拉取私有镜像失败

**原因**: 未登录或权限不足

**解决**:
```bash
# 登录后再拉取
export CR_PAT=YOUR_TOKEN
echo $CR_PAT | docker login ghcr.io -u YOUR_USERNAME --password-stdin
docker pull ghcr.io/YOUR_USERNAME/license-center:1.0.0
```

### 4. 镜像大小限制

GitHub Container Registry 限制：
- 免费账户：500MB 存储
- Pro 账户：2GB 存储
- 单个镜像层：5GB

当前镜像大小：128MB ✅

## 🎯 最佳实践

1. **使用语义化版本标签**
   ```bash
   ghcr.io/username/license-center:1.0.0
   ghcr.io/username/license-center:1.0
   ghcr.io/username/license-center:1
   ghcr.io/username/license-center:latest
   ```

2. **设置镜像为公开**（如果是开源项目）

3. **关联到 GitHub 仓库**

4. **使用 GitHub Actions 自动构建和推送**

5. **定期清理旧版本**

6. **添加镜像徽章到 README**
   ```markdown
   ![Docker Image](https://ghcr-badge.egpl.dev/YOUR_USERNAME/license-center/latest_tag?trim=major&label=latest)
   ```

## 📚 相关链接

- [GitHub Container Registry 文档](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [GitHub Packages 定价](https://docs.github.com/en/billing/managing-billing-for-github-packages/about-billing-for-github-packages)
- [Docker 官方文档](https://docs.docker.com/)

---

**准备好推送了吗？运行以下命令开始：**

```bash
cd /Users/jianshe/Projects/NodePass-Pro/license-center
./push-to-github.sh YOUR_GITHUB_USERNAME 1.0.0
```
