# 版本管理对接方案

## 概述

本文档描述如何将各个组件（后端、前端、节点客户端）与授权中心的版本管理系统对接，以及如何通过 GitHub Actions 自动同步版本信息。

## 方案架构

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Backend   │     │  Frontend   │     │ Node Client │
│             │     │             │     │             │
│  启动时上报  │     │  构建时上报  │     │  启动时上报  │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       │                   │                   │
       └───────────────────┼───────────────────┘
                           │
                           ▼
                  ┌─────────────────┐
                  │ License Center  │
                  │  版本管理 API    │
                  └─────────────────┘
                           ▲
                           │
                  ┌────────┴────────┐
                  │ GitHub Actions  │
                  │   自动同步       │
                  └─────────────────┘
```

## 一、组件启动时自动上报版本

### 1.1 后端服务（Go）

#### 步骤 1: 使用版本上报包

已创建 `pkg/version/reporter.go`，包含：
- 版本信息结构
- 自动上报功能
- 异步上报（不阻塞启动）

#### 步骤 2: 在 main.go 中集成

```go
package main

import (
    "log"
    "nodepass-backend/pkg/version"
)

func main() {
    // 打印版本信息
    version.PrintInfo()

    // 创建版本上报器
    reporter := version.NewReporter(
        "http://license-center:8090",  // 授权中心地址
        version.ComponentBackend,       // 组件类型
        "",                             // Token（可选，如果需要认证）
    )

    // 异步上报版本（不阻塞启动）
    reporter.ReportAsync()

    // 继续启动服务...
    startServer()
}
```

#### 步骤 3: 编译时注入版本信息

使用提供的构建脚本：

```bash
# 使用脚本构建
./scripts/build-with-version.sh -o backend cmd/server/main.go

# 或者直接使用 go build
go build -ldflags "\
  -X 'github.com/yourusername/nodepass/pkg/version.Version=1.0.0' \
  -X 'github.com/yourusername/nodepass/pkg/version.GitCommit=$(git rev-parse HEAD)' \
  -X 'github.com/yourusername/nodepass/pkg/version.GitBranch=$(git branch --show-current)' \
  -X 'github.com/yourusername/nodepass/pkg/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
  -o backend cmd/server/main.go
```

### 1.2 前端应用（React）

#### 步骤 1: 创建版本上报模块

创建 `src/utils/versionReporter.ts`:

```typescript
interface VersionInfo {
  component: string
  version: string
  git_commit?: string
  git_branch?: string
  build_time?: string
  description?: string
}

export async function reportVersion(
  licenseCenterUrl: string,
  token?: string
): Promise<void> {
  const versionInfo: VersionInfo = {
    component: 'frontend',
    version: import.meta.env.VITE_APP_VERSION || 'dev',
    git_commit: import.meta.env.VITE_GIT_COMMIT,
    git_branch: import.meta.env.VITE_GIT_BRANCH,
    build_time: import.meta.env.VITE_BUILD_TIME,
    description: 'Auto-reported from frontend',
  }

  try {
    const response = await fetch(`${licenseCenterUrl}/api/v1/versions/components`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
      },
      body: JSON.stringify(versionInfo),
    })

    if (!response.ok) {
      console.warn('Failed to report version:', response.status)
    } else {
      console.log('Version reported successfully:', versionInfo.version)
    }
  } catch (error) {
    console.warn('Failed to report version:', error)
  }
}
```

#### 步骤 2: 在应用启动时调用

在 `src/main.tsx` 或 `src/App.tsx` 中：

```typescript
import { reportVersion } from './utils/versionReporter'

// 应用启动后上报版本
reportVersion('http://localhost:8090').catch(console.warn)
```

#### 步骤 3: 构建时注入版本信息

修改 `vite.config.ts`:

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { execSync } from 'child_process'

const getGitInfo = () => {
  try {
    return {
      version: execSync('git describe --tags --always').toString().trim(),
      commit: execSync('git rev-parse HEAD').toString().trim(),
      branch: execSync('git rev-parse --abbrev-ref HEAD').toString().trim(),
      buildTime: new Date().toISOString(),
    }
  } catch {
    return {
      version: 'dev',
      commit: 'unknown',
      branch: 'unknown',
      buildTime: new Date().toISOString(),
    }
  }
}

const gitInfo = getGitInfo()

export default defineConfig({
  plugins: [react()],
  define: {
    'import.meta.env.VITE_APP_VERSION': JSON.stringify(gitInfo.version),
    'import.meta.env.VITE_GIT_COMMIT': JSON.stringify(gitInfo.commit),
    'import.meta.env.VITE_GIT_BRANCH': JSON.stringify(gitInfo.branch),
    'import.meta.env.VITE_BUILD_TIME': JSON.stringify(gitInfo.buildTime),
  },
})
```

### 1.3 节点客户端（Go）

与后端服务相同，使用 `pkg/version/reporter.go`：

```go
package main

import (
    "nodepass-client/pkg/version"
)

func main() {
    // 创建版本上报器
    reporter := version.NewReporter(
        "http://license-center:8090",
        version.ComponentNodeClient,
        "",
    )

    // 异步上报
    reporter.ReportAsync()

    // 启动客户端...
    startClient()
}
```

## 二、GitHub Actions 自动同步

### 2.1 配置 GitHub Secrets

在 GitHub 仓库设置中添加以下 Secrets：

1. `LICENSE_CENTER_URL`: 授权中心地址（如 `https://license.example.com`）
2. `LICENSE_CENTER_TOKEN`: 授权中心 API Token（管理员 JWT Token）

### 2.2 GitHub Actions 工作流

已创建 `.github/workflows/version-sync.yml`，功能：

- 监听 push 到 main/develop 分支
- 监听 tag 推送（v*）
- 支持手动触发
- 自动检测修改的组件
- 上报版本信息到授权中心

### 2.3 触发方式

#### 自动触发（推荐）

```bash
# 提交代码
git add .
git commit -m "feat: add new feature"
git push origin main

# 或创建 tag
git tag v1.0.1
git push origin v1.0.1
```

#### 手动触发

在 GitHub 仓库页面：
1. 进入 Actions 标签
2. 选择 "Version Sync" 工作流
3. 点击 "Run workflow"

## 三、Docker 镜像版本同步

### 3.1 Dockerfile 中注入版本

```dockerfile
FROM golang:1.21-alpine AS builder

ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown
ARG BUILD_TIME=unknown

WORKDIR /app
COPY . .

RUN go build -ldflags "\
    -X 'github.com/yourusername/nodepass/pkg/version.Version=${VERSION}' \
    -X 'github.com/yourusername/nodepass/pkg/version.GitCommit=${GIT_COMMIT}' \
    -X 'github.com/yourusername/nodepass/pkg/version.GitBranch=${GIT_BRANCH}' \
    -X 'github.com/yourusername/nodepass/pkg/version.BuildTime=${BUILD_TIME}'" \
    -o app cmd/server/main.go

FROM alpine:latest
COPY --from=builder /app/app /app
CMD ["/app"]
```

### 3.2 构建 Docker 镜像

```bash
docker build \
  --build-arg VERSION=$(git describe --tags --always) \
  --build-arg GIT_COMMIT=$(git rev-parse HEAD) \
  --build-arg GIT_BRANCH=$(git branch --show-current) \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -t nodepass-backend:latest .
```

### 3.3 GitHub Actions 构建镜像

创建 `.github/workflows/docker-build.yml`:

```yaml
name: Build Docker Image

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Get version info
        id: version
        run: |
          echo "version=${GITHUB_REF#refs/tags/}" >> $GITHUB_OUTPUT
          echo "git_commit=$(git rev-parse HEAD)" >> $GITHUB_OUTPUT
          echo "git_branch=$(git rev-parse --abbrev-ref HEAD)" >> $GITHUB_OUTPUT
          echo "build_time=$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> $GITHUB_OUTPUT

      - name: Build and push
        uses: docker/build-push-action@v4
        with:
          context: .
          push: true
          tags: |
            yourusername/nodepass-backend:${{ steps.version.outputs.version }}
            yourusername/nodepass-backend:latest
          build-args: |
            VERSION=${{ steps.version.outputs.version }}
            GIT_COMMIT=${{ steps.version.outputs.git_commit }}
            GIT_BRANCH=${{ steps.version.outputs.git_branch }}
            BUILD_TIME=${{ steps.version.outputs.build_time }}
```

## 四、配置说明

### 4.1 环境变量配置

各组件需要配置以下环境变量：

```bash
# 授权中心地址
LICENSE_CENTER_URL=http://license-center:8090

# API Token（可选，如果需要认证）
LICENSE_CENTER_TOKEN=your_jwt_token_here

# 是否启用版本上报
VERSION_REPORT_ENABLED=true
```

### 4.2 授权中心配置

如果需要认证，可以创建一个专门的 API Token：

```bash
# 登录获取 Token
curl -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "your_password"}'

# 使用返回的 token 配置到各组件
```

### 4.3 网络配置

确保各组件可以访问授权中心：

- 开发环境：使用 localhost 或 127.0.0.1
- Docker 环境：使用服务名（如 license-center）
- 生产环境：使用域名或 IP 地址

## 五、测试验证

### 5.1 手动测试上报

```bash
# 测试后端上报
curl -X POST http://localhost:8090/api/v1/versions/components \
  -H "Content-Type: application/json" \
  -d '{
    "component": "backend",
    "version": "1.0.0",
    "git_commit": "abc123",
    "git_branch": "main",
    "description": "Manual test"
  }'

# 查看系统版本
curl http://localhost:8090/api/v1/versions/system
```

### 5.2 验证 GitHub Actions

1. 推送代码到 GitHub
2. 查看 Actions 标签页
3. 确认工作流执行成功
4. 在授权中心查看版本是否更新

## 六、故障排查

### 6.1 上报失败

检查：
- 授权中心是否可访问
- Token 是否有效
- 网络连接是否正常

查看日志：
```bash
# 后端日志
tail -f /var/log/backend.log

# 授权中心日志
tail -f /tmp/license-center.log
```

### 6.2 GitHub Actions 失败

检查：
- Secrets 是否正确配置
- 授权中心 URL 是否可从 GitHub 访问
- Token 是否过期

## 七、最佳实践

1. **版本号规范**: 使用语义化版本（Semantic Versioning）
2. **自动化**: 优先使用 GitHub Actions 自动同步
3. **容错处理**: 版本上报失败不应影响服务启动
4. **日志记录**: 记录版本上报的成功和失败
5. **定期检查**: 定期检查版本信息是否同步

## 八、文件清单

- `pkg/version/reporter.go` - 版本上报包
- `scripts/build-with-version.sh` - 构建脚本
- `.github/workflows/version-sync.yml` - GitHub Actions 工作流
- `VERSION_INTEGRATION_GUIDE.md` - 本文档

## 总结

通过以上方案，实现了：
- ✅ 组件启动时自动上报版本
- ✅ GitHub Actions 自动同步版本
- ✅ Docker 镜像版本管理
- ✅ 统一的版本管理平台
- ✅ 完整的版本追踪和历史记录
