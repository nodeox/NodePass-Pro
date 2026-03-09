# 版本管理功能

## 概述

版本管理功能提供了一个集中式的 Web 界面，用于管理和监控 NodePass 系统中所有组件的版本信息，包括：

- 后端服务 (Backend)
- 前端应用 (Frontend)
- 节点客户端 (Node Client)
- 授权中心 (License Center)

## 功能特性

### 1. 系统版本概览

- 实时显示所有组件的当前版本
- 版本兼容性状态检查
- 兼容性警告和错误提示

### 2. 组件版本管理

每个组件都有独立的版本卡片，显示：

- 版本号
- Git Commit Hash
- Git Branch
- 构建时间
- 版本描述
- 最后更新时间

### 3. 版本历史

- 查看每个组件的版本历史记录
- 支持查看最近 20 条版本记录
- 显示版本状态（当前/历史）

### 4. 版本更新

通过 Web 界面更新组件版本，支持填写：

- 版本号（必填）
- Git Commit Hash
- Git Branch
- 构建时间
- 版本描述

### 5. 兼容性配置

- 定义不同后端版本对其他组件的最低版本要求
- 自动检查当前系统的版本兼容性
- 支持创建和管理多个兼容性配置

## 数据库表结构

### component_versions 表

存储组件版本信息：

```sql
CREATE TABLE component_versions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    component VARCHAR(50) NOT NULL,
    version VARCHAR(50) NOT NULL,
    build_time DATETIME,
    git_commit VARCHAR(100),
    git_branch VARCHAR(100),
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    UNIQUE KEY idx_component (component)
);
```

### version_compatibility 表

存储版本兼容性配置：

```sql
CREATE TABLE version_compatibility (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    backend_version VARCHAR(50) NOT NULL,
    min_frontend_version VARCHAR(50) NOT NULL,
    min_node_client_version VARCHAR(50) NOT NULL,
    min_license_center_version VARCHAR(50) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    UNIQUE KEY idx_backend_version (backend_version)
);
```

## API 接口

### 获取系统版本信息

```bash
GET /api/v1/versions/system
```

返回所有组件的当前版本和兼容性状态。

### 获取组件版本

```bash
GET /api/v1/versions/components/:component
```

参数：
- `component`: backend, frontend, node_client, license_center

### 更新组件版本

```bash
POST /api/v1/versions/components
```

请求体：
```json
{
  "component": "backend",
  "version": "1.0.0",
  "git_commit": "abc123",
  "git_branch": "main",
  "build_time": "2024-03-08T12:00:00Z",
  "description": "Initial release"
}
```

### 获取版本历史

```bash
GET /api/v1/versions/components/:component/history?limit=20
```

### 获取兼容性配置列表

```bash
GET /api/v1/versions/compatibility
```

### 创建兼容性配置

```bash
POST /api/v1/versions/compatibility
```

请求体：
```json
{
  "backend_version": "1.0.0",
  "min_frontend_version": "1.0.0",
  "min_node_client_version": "1.0.0",
  "min_license_center_version": "1.0.0",
  "description": "Version 1.0.0 compatibility matrix"
}
```

### 检查版本兼容性

```bash
GET /api/v1/versions/compatibility/:version
```

## Web 界面使用

### 访问版本管理页面

1. 登录授权中心管理后台
2. 点击左侧菜单的"版本管理"
3. 进入版本管理页面

### 查看系统版本

在页面顶部可以看到：
- 兼容性状态（兼容/不兼容）
- 兼容性错误和警告信息

### 更新组件版本

1. 在对应组件卡片上点击"更新"按钮
2. 填写版本信息
3. 点击确定提交

### 查看版本历史

1. 在对应组件卡片上点击"历史"按钮
2. 或切换到"版本历史"标签页
3. 选择要查看的组件

### 管理兼容性配置

1. 切换到"兼容性配置"标签页
2. 点击"新增配置"按钮
3. 填写后端版本和各组件的最低版本要求
4. 点击确定提交

## 测试

使用提供的测试脚本测试 API：

```bash
# 设置 JWT Token
export TOKEN="your_jwt_token_here"

# 运行测试
./scripts/test-version-api.sh
```

测试脚本会：
1. 获取系统版本信息
2. 更新所有组件版本
3. 创建兼容性配置
4. 验证版本历史
5. 验证兼容性检查

## 版本兼容性检查逻辑

系统会自动检查版本兼容性：

1. 查找当前后端版本对应的兼容性配置
2. 比较前端版本是否满足最低要求（不满足则报错）
3. 比较节点客户端版本是否满足最低要求（不满足则警告）
4. 比较授权中心版本是否满足最低要求（不满足则警告）

版本比较采用语义化版本规则（Semantic Versioning）。

## 注意事项

1. **版本号格式**：建议使用语义化版本号格式（如 1.0.0）
2. **兼容性配置**：每个后端版本只能有一个活跃的兼容性配置
3. **版本历史**：更新版本时会自动将旧版本标记为非活跃状态
4. **权限要求**：所有版本管理 API 都需要管理员权限

## 集成说明

版本管理功能已集成到授权中心（License Center）中：

- **前端路由**：`/versions`
- **菜单位置**：左侧导航栏最后一项
- **图标**：CodeOutlined
- **数据库**：使用 GORM AutoMigrate 自动创建表

## 未来扩展

可能的功能扩展：

1. 版本自动上报（组件启动时自动上报版本）
2. 版本变更通知（Webhook 集成）
3. 版本回滚功能
4. 版本发布计划管理
5. 版本依赖关系图
6. 版本变更日志自动生成
