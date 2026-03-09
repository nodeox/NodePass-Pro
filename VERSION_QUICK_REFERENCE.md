# 版本管理快速参考

## 一、组件自动上报版本

### 后端服务（Go）

```go
import "nodepass-backend/pkg/version"

func main() {
    // 创建上报器
    reporter := version.NewReporter(
        "http://license-center:8090",
        version.ComponentBackend,
        "",
    )

    // 异步上报
    reporter.ReportAsync()

    // 启动服务...
}
```

### 前端应用（React）

```typescript
import { reportVersion } from './utils/versionReporter'

// 应用启动后上报
reportVersion('http://localhost:8090').catch(console.warn)
```

### 节点客户端（Go）

```go
import "nodepass-client/pkg/version"

func main() {
    reporter := version.NewReporter(
        "http://license-center:8090",
        version.ComponentNodeClient,
        "",
    )
    reporter.ReportAsync()
}
```

## 二、构建命令

### 本地构建

```bash
# 后端
cd license-center
make -f Makefile.version build

# 查看版本
make -f Makefile.version version

# 运行
make -f Makefile.version run
```

### Docker 构建

```bash
# 单个服务
docker build \
  --build-arg VERSION=$(git describe --tags --always) \
  --build-arg GIT_COMMIT=$(git rev-parse HEAD) \
  --build-arg GIT_BRANCH=$(git branch --show-current) \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -t nodepass-backend:latest .

# 所有服务
./deploy-with-version.sh build
```

### 跨平台构建

```bash
# Linux
make -f Makefile.version build-linux

# macOS
make -f Makefile.version build-darwin

# Windows
make -f Makefile.version build-windows

# 所有平台
make -f Makefile.version build-all
```

## 三、部署命令

### 使用部署脚本

```bash
# 完整部署
./deploy-with-version.sh deploy

# 仅构建
./deploy-with-version.sh build

# 启动服务
./deploy-with-version.sh start

# 停止服务
./deploy-with-version.sh stop

# 重启服务
./deploy-with-version.sh restart

# 查看状态
./deploy-with-version.sh status

# 验证版本
./deploy-with-version.sh verify

# 查看日志
./deploy-with-version.sh logs
./deploy-with-version.sh logs license-center

# 清理
./deploy-with-version.sh clean
```

### 使用 Docker Compose

```bash
# 设置版本信息
export VERSION=$(git describe --tags --always)
export GIT_COMMIT=$(git rev-parse HEAD)
export GIT_BRANCH=$(git branch --show-current)
export BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# 构建并启动
docker-compose -f docker-compose.version.yml up -d --build

# 停止
docker-compose -f docker-compose.version.yml down
```

## 四、GitHub Actions

### 配置 Secrets

在 GitHub 仓库设置中添加：
- `LICENSE_CENTER_URL`: 授权中心地址
- `LICENSE_CENTER_TOKEN`: API Token

### 触发同步

```bash
# 推送代码（自动触发）
git push origin main

# 创建标签（自动触发）
git tag v1.0.1
git push origin v1.0.1

# 手动触发
# 在 GitHub Actions 页面点击 "Run workflow"
```

## 五、API 测试

### 登录获取 Token

```bash
TOKEN=$(curl -s -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your_password"}' \
  | jq -r '.data.token')
```

### 上报版本

```bash
curl -X POST http://localhost:8090/api/v1/versions/components \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "component": "backend",
    "version": "1.0.0",
    "git_commit": "abc123",
    "git_branch": "main",
    "description": "Manual report"
  }'
```

### 查询版本

```bash
# 系统版本信息
curl -s http://localhost:8090/api/v1/versions/system \
  -H "Authorization: Bearer $TOKEN" | jq '.'

# 组件版本
curl -s http://localhost:8090/api/v1/versions/components/backend \
  -H "Authorization: Bearer $TOKEN" | jq '.'

# 版本历史
curl -s http://localhost:8090/api/v1/versions/components/backend/history?limit=10 \
  -H "Authorization: Bearer $TOKEN" | jq '.'

# 兼容性配置
curl -s http://localhost:8090/api/v1/versions/compatibility \
  -H "Authorization: Bearer $TOKEN" | jq '.'
```

## 六、环境变量

### 后端/节点客户端

```bash
# 授权中心地址
LICENSE_CENTER_URL=http://license-center:8090

# API Token（可选）
LICENSE_CENTER_TOKEN=your_jwt_token

# 是否启用版本上报
VERSION_REPORT_ENABLED=true
```

### 前端

```bash
# 构建时注入
VITE_APP_VERSION=1.0.0
VITE_GIT_COMMIT=abc123
VITE_GIT_BRANCH=main
VITE_BUILD_TIME=2024-03-08T12:00:00Z
VITE_LICENSE_CENTER_URL=http://localhost:8090
```

## 七、故障排查

### 检查服务状态

```bash
# Docker 容器
docker ps

# 服务健康检查
curl http://localhost:8090/health

# 查看日志
docker logs nodepass-license-center
docker logs nodepass-backend
```

### 测试版本上报

```bash
# 手动上报
make -f Makefile.version report-version

# 查看上报结果
curl -s http://localhost:8090/api/v1/versions/system | jq '.'
```

### 常见问题

**问题**: 版本上报失败
```bash
# 检查授权中心是否可访问
curl http://license-center:8090/health

# 检查 Token 是否有效
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8090/api/v1/auth/me
```

**问题**: GitHub Actions 失败
```bash
# 检查 Secrets 配置
# 检查授权中心是否可从外网访问
# 检查 Token 是否过期
```

## 八、文件位置

```
NodePass-Pro/
├── VERSION_INTEGRATION_GUIDE.md          # 完整集成指南
├── VERSION_QUICK_REFERENCE.md            # 本文档
├── deploy-with-version.sh                # 部署脚本
├── docker-compose.version.yml            # Docker Compose 配置
├── .github/workflows/version-sync.yml    # GitHub Actions
├── license-center/
│   ├── pkg/version/reporter.go           # 版本上报包
│   ├── scripts/build-with-version.sh     # 构建脚本
│   ├── Makefile.version                  # Makefile
│   └── web-ui/src/utils/versionReporter.ts  # 前端上报
└── ...
```

## 九、最佳实践

1. **使用语义化版本**: v1.0.0, v1.0.1, v1.1.0
2. **自动化构建**: 使用 GitHub Actions 或 CI/CD
3. **容器化部署**: 使用 Docker 确保版本一致性
4. **定期检查**: 在版本管理页面查看各组件版本
5. **版本兼容性**: 及时更新兼容性配置

## 十、快速开始

```bash
# 1. 克隆仓库
git clone https://github.com/yourusername/nodepass.git
cd nodepass

# 2. 部署所有服务
./deploy-with-version.sh deploy

# 3. 访问版本管理页面
open http://localhost:3000/versions

# 4. 查看版本信息
./deploy-with-version.sh verify
```

---

**更多详细信息请参考**: `VERSION_INTEGRATION_GUIDE.md`
