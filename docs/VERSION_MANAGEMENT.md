# 版本管理系统

## 概述

NodePass-Pro 采用统一的版本管理系统，确保所有组件（后端、前端、节点客户端、授权中心）版本同步。

## 版本号规范

采用语义化版本号（Semantic Versioning）：`主版本号.次版本号.修订号`

- **主版本号**：不兼容的 API 修改
- **次版本号**：向下兼容的功能性新增
- **修订号**：向下兼容的问题修正

### 当前版本

**v1.0.0** - 2026-03-08

## 组件版本

所有组件保持统一版本号：

| 组件 | 版本 | 文件位置 |
|------|------|---------|
| 根目录 | 1.0.0 | `VERSION` |
| 后端服务 | 1.0.0 | `backend/internal/version/version.go` |
| 前端应用 | 1.0.0 | `frontend/package.json` |
| 节点客户端 | 1.0.0 | `nodeclient/internal/agent/agent.go` |
| 授权中心 | 1.0.0 | `license-center/web-ui/package.json` |

## 版本配置文件

### version.yaml

统一的版本配置文件，包含：
- 主版本号
- 各组件版本
- 版本兼容性矩阵
- 更新日志

```yaml
version: "1.0.0"

components:
  backend:
    version: "1.0.0"
    min_compatible_version: "0.9.0"

  frontend:
    version: "1.0.0"
    min_compatible_version: "0.9.0"

  node_client:
    version: "1.0.0"
    min_compatible_version: "0.9.0"

  license_center:
    version: "1.0.0"
    min_compatible_version: "0.9.0"

compatibility:
  - backend_version: "1.0.0"
    min_frontend_version: "1.0.0"
    min_node_client_version: "0.9.0"
    min_license_center_version: "1.0.0"
```

## 版本管理工具

### 1. 版本检查工具

检查所有组件版本是否一致：

```bash
./check-version.sh
```

**输出示例**：
```
==========================================
NodePass-Pro 版本检查工具
==========================================

📋 读取各组件版本...

根目录版本：1.0.0
version.yaml：1.0.0
后端版本：1.0.0
前端版本：1.0.0
节点客户端版本：1.0.0
授权中心版本：1.0.0

==========================================
版本一致性检查
==========================================

✅ 所有组件版本一致：1.0.0
```

### 2. 版本同步工具

统一更新所有组件的版本号：

```bash
./sync-version.sh
```

**功能**：
- 交互式输入新版本号
- 自动更新所有组件版本
- 显示更新后的版本信息
- 可选创建 Git 标签

**使用示例**：
```bash
$ ./sync-version.sh

==========================================
NodePass-Pro 版本同步工具
==========================================

当前版本：1.0.0

请输入新版本号（留空保持当前版本）: 1.1.0

新版本号：1.1.0

==========================================
开始同步版本...
==========================================

📝 更新 VERSION 文件...
✓ VERSION 文件已更新

📝 更新 version.yaml...
✓ version.yaml 已更新

📝 更新后端版本...
✓ 后端版本已更新

📝 更新前端版本...
✓ 前端版本已更新

📝 更新节点客户端版本...
✓ 节点客户端版本已更新

📝 更新授权中心版本...
✓ 授权中心版本已更新

==========================================
✅ 所有组件版本已同步为：1.1.0
==========================================
```

## 版本兼容性

### 兼容性矩阵

| 后端版本 | 最低前端版本 | 最低节点客户端版本 | 最低授权中心版本 |
|---------|------------|------------------|----------------|
| 1.0.0 | 1.0.0 | 0.9.0 | 1.0.0 |
| 0.1.0 | 0.1.0 | 0.1.0 | 0.2.0 |

### 兼容性检查

系统启动时会自动检查各组件版本兼容性：

```go
// 后端版本信息
version.Version = "1.0.0"
version.BuildTime = &buildTime
version.GitCommit = &gitCommit
version.GitBranch = &gitBranch
version.GoVersion = &goVersion
```

## 版本发布流程

### 1. 准备发布

```bash
# 1. 检查当前版本
./check-version.sh

# 2. 确保所有更改已提交
git status

# 3. 运行测试
make test
```

### 2. 更新版本

```bash
# 使用版本同步工具
./sync-version.sh

# 输入新版本号，例如：1.1.0
```

### 3. 更新 version.yaml

编辑 `version.yaml`，添加更新日志：

```yaml
changelog:
  - version: "1.1.0"
    date: "2026-03-15"
    changes:
      - "新增功能 A"
      - "修复 Bug B"
      - "性能优化 C"
```

### 4. 创建 Git 标签

```bash
# 提交版本更改
git add .
git commit -m "chore: bump version to 1.1.0"

# 创建标签
git tag -a v1.1.0 -m "Release version 1.1.0"

# 推送到远程
git push origin main
git push origin v1.1.0
```

### 5. 构建发布

```bash
# 构建后端
cd backend
go build -ldflags "-X nodepass-pro/backend/internal/version.Version=1.1.0" -o ../bin/nodepass-server ./cmd/server/main.go

# 构建前端
cd frontend
npm run build

# 构建节点客户端
cd nodeclient
go build -o ../bin/nodeclient ./cmd/client/main.go

# 构建授权中心
cd license-center/web-ui
npm run build
```

## 版本查询

### 后端版本查询

```bash
# 通过命令行
./bin/nodepass-server --version

# 通过 API
curl http://localhost:8080/api/v1/ping
```

**响应示例**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "pong",
    "version": "1.0.0"
  }
}
```

### 前端版本查询

前端版本显示在页面底部或关于页面。

### 节点客户端版本查询

```bash
./bin/nodeclient --version
```

**输出**：
```
1.0.0
```

## 构建信息

### 编译时注入版本信息

```bash
# 后端构建
go build -ldflags "\
  -X nodepass-pro/backend/internal/version.Version=1.0.0 \
  -X nodepass-pro/backend/internal/version.GitCommit=$(git rev-parse HEAD) \
  -X nodepass-pro/backend/internal/version.GitBranch=$(git rev-parse --abbrev-ref HEAD) \
  -X nodepass-pro/backend/internal/version.GoVersion=$(go version | awk '{print $3}') \
  -X nodepass-pro/backend/internal/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o bin/nodepass-server ./cmd/server/main.go
```

### 查看构建信息

```bash
# 通过 API 查看完整版本信息
curl http://localhost:8080/api/v1/version
```

**响应示例**：
```json
{
  "version": "1.0.0",
  "build_time": "2026-03-08T10:30:00Z",
  "git_commit": "abc123def456",
  "git_branch": "main",
  "go_version": "go1.21.5"
}
```

## 版本历史

### v1.0.0 (2026-03-08)

**新增功能**：
- ✅ 隧道批量导入导出功能
- ✅ 隧道模板系统
- ✅ 统一版本管理系统
- ✅ 版本同步和检查工具

**优化改进**：
- 性能优化
- 代码重构
- 文档完善

### v0.1.0 (2026-01-01)

**初始版本**：
- 基础隧道管理功能
- 节点组管理
- 用户认证和授权
- 流量统计

## 最佳实践

### 1. 版本发布前

- ✅ 运行 `./check-version.sh` 检查版本一致性
- ✅ 运行完整测试套件
- ✅ 更新 CHANGELOG
- ✅ 更新文档

### 2. 版本发布

- ✅ 使用 `./sync-version.sh` 统一更新版本
- ✅ 创建 Git 标签
- ✅ 构建发布包
- ✅ 发布到生产环境

### 3. 版本发布后

- ✅ 验证各组件版本
- ✅ 监控系统运行状态
- ✅ 收集用户反馈

## 常见问题

### Q: 如何查看当前版本？

```bash
./check-version.sh
```

### Q: 如何更新版本？

```bash
./sync-version.sh
```

### Q: 版本不一致怎么办？

运行版本同步工具：
```bash
./sync-version.sh
```

### Q: 如何回滚版本？

```bash
# 1. 检出旧版本标签
git checkout v0.1.0

# 2. 或者手动修改版本号
./sync-version.sh
# 输入旧版本号
```

### Q: 如何添加构建信息？

编辑 `backend/internal/version/version.go`，添加构建时变量。

## 相关文件

- `VERSION` - 根目录版本文件
- `version.yaml` - 版本配置文件
- `sync-version.sh` - 版本同步工具
- `check-version.sh` - 版本检查工具
- `backend/internal/version/version.go` - 后端版本定义
- `frontend/package.json` - 前端版本定义
- `nodeclient/internal/agent/agent.go` - 节点客户端版本定义
- `license-center/web-ui/package.json` - 授权中心版本定义

---

**维护者**：NodePass-Pro 团队
**更新日期**：2026-03-08
**文档版本**：1.0.0
