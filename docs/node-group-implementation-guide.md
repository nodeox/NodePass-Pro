# 节点分组功能实施指南

## 📋 实施概览

本文档提供节点分组功能的完整实施指南，包括数据库迁移、后端实现、前端开发和测试验证。

---

## 🎯 实施步骤

### 阶段 1：数据库迁移（1天）

#### 1.1 创建迁移文件

```bash
cd backend
# 创建迁移文件
touch migrations/000X_create_node_groups.up.sql
touch migrations/000X_create_node_groups.down.sql
```

#### 1.2 编写迁移 SQL

**Up Migration** (`000X_create_node_groups.up.sql`):
```sql
-- 见前面的数据库设计部分
-- 包含所有表创建和触发器
```

**Down Migration** (`000X_create_node_groups.down.sql`):
```sql
DROP TRIGGER IF EXISTS trigger_update_node_group_stats ON node_instances;
DROP FUNCTION IF EXISTS update_node_group_stats();
DROP TABLE IF EXISTS tunnels;
DROP TABLE IF EXISTS node_group_stats;
DROP TABLE IF EXISTS node_group_relations;
DROP TABLE IF EXISTS node_instances;
DROP TABLE IF EXISTS node_groups;
```

#### 1.3 执行迁移

```bash
# 运行迁移
go run ./cmd/migrate up

# 验证表结构
psql -U postgres -d nodepass_panel -c "\dt"
```

---

### 阶段 2：后端实现（3-4天）

#### 2.1 模型层 ✅

已创建：
- `backend/internal/models/node_group.go`

#### 2.2 服务层

需要创建以下服务：

1. **NodeGroupService** ✅
   - `backend/internal/services/node_group_service.go`

2. **NodeInstanceService**
   - `backend/internal/services/node_instance_service.go`
   - 节点实例管理
   - 心跳处理
   - 健康检查

3. **TunnelService**（重构）
   - `backend/internal/services/tunnel_service.go`
   - 基于节点组的隧道管理

#### 2.3 Handler 层

需要创建以下 Handler：

1. **NodeGroupHandler**
   - `backend/internal/handlers/node_group_handler.go`
   - CRUD 操作
   - 统计信息

2. **NodeInstanceHandler**
   - `backend/internal/handlers/node_instance_handler.go`
   - 节点管理
   - 部署命令生成
   - 心跳上报

3. **TunnelHandler**（重构）
   - `backend/internal/handlers/tunnel_handler.go`

#### 2.4 路由注册

在 `backend/cmd/server/main.go` 中注册路由：

```go
// 节点组管理
nodeGroupHandler := handlers.NewNodeGroupHandler(database.DB)
nodeGroups := authGroup.Group("/node-groups")
{
    nodeGroups.POST("", nodeGroupHandler.Create)
    nodeGroups.GET("", nodeGroupHandler.List)
    nodeGroups.GET("/:id", nodeGroupHandler.Get)
    nodeGroups.PUT("/:id", nodeGroupHandler.Update)
    nodeGroups.DELETE("/:id", nodeGroupHandler.Delete)
    nodeGroups.POST("/:id/toggle", nodeGroupHandler.Toggle)
    nodeGroups.GET("/:id/stats", nodeGroupHandler.GetStats)

    // 部署相关
    nodeGroups.POST("/:id/generate-deploy-command", nodeGroupHandler.GenerateDeployCommand)
    nodeGroups.GET("/:id/nodes", nodeGroupHandler.ListNodes)
    nodeGroups.POST("/:id/nodes", nodeGroupHandler.AddNode)
}

// 节点实例管理
nodeInstanceHandler := handlers.NewNodeInstanceHandler(database.DB)
nodeInstances := authGroup.Group("/node-instances")
{
    nodeInstances.GET("/:id", nodeInstanceHandler.Get)
    nodeInstances.PUT("/:id", nodeInstanceHandler.Update)
    nodeInstances.DELETE("/:id", nodeInstanceHandler.Delete)
    nodeInstances.POST("/:id/restart", nodeInstanceHandler.Restart)
}

// 节点心跳（无需认证，使用 node_id 认证）
api.POST("/node-instances/heartbeat", nodeInstanceHandler.Heartbeat)
```

---

### 阶段 3：前端实现（3-4天）

#### 3.1 类型定义 ✅

已创建：
- `frontend/src/types/nodeGroup.ts`

#### 3.2 API 服务

创建 `frontend/src/services/nodeGroupApi.ts`:

```typescript
import apiClient from './api'
import type {
  NodeGroup,
  NodeInstance,
  CreateNodeGroupPayload,
  DeployNodePayload,
  DeployCommandResponse,
} from '../types/nodeGroup'

export const nodeGroupApi = {
  // 节点组管理
  list: (params?: { type?: string; enabled?: boolean }) =>
    apiClient
      .get<ApiSuccessResponse<PaginationResult<NodeGroup>>>('/node-groups', { params })
      .then(unwrapData),

  get: (id: number) =>
    apiClient
      .get<ApiSuccessResponse<NodeGroup>>(`/node-groups/${id}`)
      .then(unwrapData),

  create: (payload: CreateNodeGroupPayload) =>
    apiClient
      .post<ApiSuccessResponse<NodeGroup>>('/node-groups', payload)
      .then(unwrapData),

  update: (id: number, payload: Partial<NodeGroup>) =>
    apiClient
      .put<ApiSuccessResponse<NodeGroup>>(`/node-groups/${id}`, payload)
      .then(unwrapData),

  delete: (id: number) =>
    apiClient
      .delete<ApiSuccessResponse<null>>(`/node-groups/${id}`)
      .then(unwrapData),

  toggle: (id: number) =>
    apiClient
      .post<ApiSuccessResponse<NodeGroup>>(`/node-groups/${id}/toggle`)
      .then(unwrapData),

  // 部署管理
  generateDeployCommand: (id: number, payload: DeployNodePayload) =>
    apiClient
      .post<ApiSuccessResponse<DeployCommandResponse>>(
        `/node-groups/${id}/generate-deploy-command`,
        payload
      )
      .then(unwrapData),

  listNodes: (id: number) =>
    apiClient
      .get<ApiSuccessResponse<PaginationResult<NodeInstance>>>(
        `/node-groups/${id}/nodes`
      )
      .then(unwrapData),
}
```

#### 3.3 页面组件

创建以下页面：

1. **节点组列表页**
   - `frontend/src/pages/NodeGroups/index.tsx`
   - 显示所有节点组
   - 分类显示（入口/出口）
   - 快速操作按钮

2. **创建节点组页**
   - `frontend/src/pages/NodeGroups/CreateNodeGroup.tsx`
   - 表单向导
   - 配置验证

3. **节点组详情页**
   - `frontend/src/pages/NodeGroups/NodeGroupDetail.tsx`
   - 基本信息
   - 节点列表
   - 统计图表

4. **部署节点页**
   - `frontend/src/pages/NodeGroups/DeployNode.tsx`
   - 配置表单
   - 命令生成
   - 一键复制

5. **节点管理页**
   - `frontend/src/pages/NodeGroups/NodeManagement.tsx`
   - 节点列表
   - 状态监控
   - 批量操作

#### 3.4 路由配置

在 `frontend/src/router.tsx` 中添加路由：

```typescript
{
  path: '/node-groups',
  element: <NodeGroupsPage />,
},
{
  path: '/node-groups/create',
  element: <CreateNodeGroupPage />,
},
{
  path: '/node-groups/:id',
  element: <NodeGroupDetailPage />,
},
{
  path: '/node-groups/:id/deploy',
  element: <DeployNodePage />,
},
{
  path: '/node-groups/:id/nodes',
  element: <NodeManagementPage />,
},
```

---

### 阶段 4：节点客户端适配（2天）

#### 4.1 客户端配置

修改 `nodeclient` 以支持节点组：

```go
// nodeclient/internal/config/config.go
type Config struct {
    // 节点组配置
    GroupID           uint   `json:"group_id"`
    NodeID            string `json:"node_id"`
    ServiceName       string `json:"service_name"`
    ConnectionAddress string `json:"connection_address"`

    // 网络配置
    ExitNetwork string `json:"exit_network"`

    // 运行配置
    DebugMode bool `json:"debug_mode"`
    AutoStart bool `json:"auto_start"`

    // 面板连接
    HubURL string `json:"hub_url"`
    Token  string `json:"token"`
}
```

#### 4.2 心跳上报

```go
// nodeclient/internal/heartbeat/heartbeat.go
func (h *HeartbeatService) Report() error {
    payload := HeartbeatPayload{
        NodeID: h.config.NodeID,
        SystemInfo: h.collectSystemInfo(),
        TrafficStats: h.collectTrafficStats(),
    }

    resp, err := h.client.Post(
        h.config.HubURL + "/api/v1/node-instances/heartbeat",
        "application/json",
        bytes.NewBuffer(payload),
    )
    // ...
}
```

---

### 阶段 5：测试验证（1-2天）

#### 5.1 单元测试

```bash
# 后端测试
cd backend
go test ./internal/services/... -v
go test ./internal/handlers/... -v

# 前端测试
cd frontend
npm run test
```

#### 5.2 集成测试

创建测试脚本 `tests/node_group_integration_test.sh`:

```bash
#!/bin/bash

# 1. 创建入口节点组
echo "创建入口节点组..."
ENTRY_GROUP=$(curl -X POST http://localhost:8080/api/v1/node-groups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试入口组",
    "type": "entry",
    "config": {
      "allowed_protocols": ["tcp", "udp"],
      "port_range": {"start": 10000, "end": 20000},
      "entry_config": {
        "require_exit_group": true,
        "traffic_multiplier": 1.0,
        "dns_load_balance": true
      }
    }
  }')

# 2. 创建出口节点组
echo "创建出口节点组..."
EXIT_GROUP=$(curl -X POST http://localhost:8080/api/v1/node-groups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试出口组",
    "type": "exit",
    "config": {
      "allowed_protocols": ["tcp", "udp"],
      "exit_config": {
        "load_balance_strategy": "round_robin",
        "health_check_interval": 30,
        "health_check_timeout": 5
      }
    }
  }')

# 3. 生成部署命令
echo "生成部署命令..."
DEPLOY_CMD=$(curl -X POST http://localhost:8080/api/v1/node-groups/1/generate-deploy-command \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "nodepass-node-1",
    "debug_mode": true
  }')

echo "部署命令: $DEPLOY_CMD"

# 4. 创建隧道
echo "创建隧道..."
TUNNEL=$(curl -X POST http://localhost:8080/api/v1/tunnels \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试隧道",
    "entry_group_id": 1,
    "exit_group_id": 2,
    "protocol": "tcp",
    "remote_port": 22,
    "remote_host": "192.168.1.100"
  }')

echo "测试完成！"
```

#### 5.3 性能测试

```bash
# 使用 Apache Bench 测试 API 性能
ab -n 1000 -c 10 -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/node-groups
```

---

## 📊 实施时间表

| 阶段 | 任务 | 预计时间 | 负责人 |
|------|------|---------|--------|
| 1 | 数据库迁移 | 1天 | 后端 |
| 2 | 后端实现 | 3-4天 | 后端 |
| 3 | 前端实现 | 3-4天 | 前端 |
| 4 | 客户端适配 | 2天 | 后端 |
| 5 | 测试验证 | 1-2天 | 全员 |
| **总计** | | **10-13天** | |

---

## ✅ 验收标准

### 功能验收

- [ ] 可以创建入口节点组和出口节点组
- [ ] 可以配置节点组的端口范围、协议、流量倍率等
- [ ] 可以生成节点部署命令
- [ ] 节点可以成功注册并上报心跳
- [ ] 可以基于节点组创建隧道
- [ ] 节点组统计信息正确显示
- [ ] 负载均衡策略正常工作

### 性能验收

- [ ] API 响应时间 < 200ms（P95）
- [ ] 支持至少 100 个节点组
- [ ] 支持至少 1000 个节点实例
- [ ] 心跳上报延迟 < 1s

### 安全验收

- [ ] 节点组隔离（用户只能访问自己的节点组）
- [ ] 节点认证（使用 node_id 认证）
- [ ] 配置验证（防止无效配置）
- [ ] 权限控制（管理员/普通用户）

---

## 🚀 部署清单

### 数据库

- [ ] 执行数据库迁移
- [ ] 验证表结构
- [ ] 创建索引
- [ ] 配置触发器

### 后端

- [ ] 更新代码
- [ ] 运行测试
- [ ] 构建二进制
- [ ] 重启服务

### 前端

- [ ] 更新代码
- [ ] 运行测试
- [ ] 构建生产版本
- [ ] 部署静态文件

### 客户端

- [ ] 更新客户端代码
- [ ] 构建多平台二进制
- [ ] 更新安装脚本
- [ ] 发布新版本

---

## 📝 后续优化

### 短期（1-2周）

1. **DNS 负载均衡**
   - 实现智能 DNS 解析
   - 支持地理位置路由

2. **健康检查增强**
   - 主动健康检查
   - 自动故障转移

3. **监控告警**
   - 节点离线告警
   - 流量异常告警

### 中期（1-2月）

1. **自动扩缩容**
   - 根据负载自动添加/移除节点
   - 成本优化

2. **高级负载均衡**
   - 基于延迟的路由
   - 基于带宽的路由

3. **多地域部署**
   - 跨地域节点组
   - 全球负载均衡

### 长期（3-6月）

1. **智能调度**
   - AI 驱动的流量调度
   - 预测性扩容

2. **边缘计算**
   - 边缘节点支持
   - CDN 集成

---

## 📚 相关文档

- [数据库设计文档](./database-design.md)
- [API 接口文档](./api-documentation.md)
- [前端组件文档](./frontend-components.md)
- [部署指南](./deployment-guide.md)

---

## 🆘 常见问题

### Q1: 如何迁移现有的节点到节点组？

A: 创建迁移脚本，将现有节点转换为节点实例并关联到默认节点组。

### Q2: 节点组和原有的节点配对有什么区别？

A: 节点组是更高层次的抽象，支持多对多关系和负载均衡，而节点配对是一对一关系。

### Q3: 如何处理节点组的版本升级？

A: 使用滚动更新策略，逐个更新节点实例，确保服务不中断。

---

**准备好开始实施了吗？** 🚀

如果需要我帮你实现某个具体部分，请告诉我！
