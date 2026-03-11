# 节点自动化 API 文档

## 基础信息

- **Base URL**: `/api/v1/automation`
- **认证方式**: Bearer Token
- **Content-Type**: `application/json`

## API 列表

### 1. 创建自动化策略

为节点组创建自动化策略。

**请求**

```http
POST /api/v1/automation/policies
Authorization: Bearer {token}
Content-Type: application/json
```

**请求体**

```json
{
  "node_group_id": 1,
  "auto_scaling_enabled": true,
  "auto_failover_enabled": true,
  "auto_recovery_enabled": true,
  "min_nodes": 2,
  "max_nodes": 10,
  "scale_up_threshold": 80.0,
  "scale_down_threshold": 30.0,
  "scale_cooldown": 300,
  "failover_timeout": 60,
  "recovery_check_interval": 300
}
```

**字段说明**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_group_id | integer | 是 | 节点组 ID |
| auto_scaling_enabled | boolean | 否 | 自动扩缩容开关，默认 false |
| auto_failover_enabled | boolean | 否 | 自动故障转移开关，默认 false |
| auto_recovery_enabled | boolean | 否 | 自动恢复开关，默认 false |
| min_nodes | integer | 否 | 最小节点数，默认 1 |
| max_nodes | integer | 否 | 最大节点数，默认 10 |
| scale_up_threshold | float | 否 | 扩容阈值（CPU %），默认 80.0 |
| scale_down_threshold | float | 否 | 缩容阈值（CPU %），默认 30.0 |
| scale_cooldown | integer | 否 | 扩缩容冷却时间（秒），默认 300 |
| failover_timeout | integer | 否 | 故障转移超时（秒），默认 60 |
| recovery_check_interval | integer | 否 | 恢复检查间隔（秒），默认 300 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "node_group_id": 1,
    "enabled": true,
    "auto_scaling_enabled": true,
    "auto_failover_enabled": true,
    "auto_recovery_enabled": true,
    "min_nodes": 2,
    "max_nodes": 10,
    "scale_up_threshold": 80.0,
    "scale_down_threshold": 30.0,
    "scale_cooldown": 300,
    "failover_timeout": 60,
    "recovery_check_interval": 300,
    "created_at": "2026-03-11T10:00:00Z",
    "updated_at": "2026-03-11T10:00:00Z"
  }
}
```

---

### 2. 获取自动化策略

获取节点组的自动化策略。

**请求**

```http
GET /api/v1/automation/policies/{node_group_id}
Authorization: Bearer {token}
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| node_group_id | integer | 节点组 ID |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "node_group_id": 1,
    "enabled": true,
    "auto_scaling_enabled": true,
    "auto_failover_enabled": true,
    "auto_recovery_enabled": true,
    "min_nodes": 2,
    "max_nodes": 10,
    "scale_up_threshold": 80.0,
    "scale_down_threshold": 30.0,
    "scale_cooldown": 300,
    "failover_timeout": 60,
    "recovery_check_interval": 300,
    "created_at": "2026-03-11T10:00:00Z",
    "updated_at": "2026-03-11T10:00:00Z"
  }
}
```

---

### 3. 更新自动化策略

更新节点组的自动化策略。

**请求**

```http
PUT /api/v1/automation/policies/{node_group_id}
Authorization: Bearer {token}
Content-Type: application/json
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| node_group_id | integer | 节点组 ID |

**请求体**

```json
{
  "enabled": false,
  "auto_scaling_enabled": false
}
```

**字段说明**

所有字段均为可选，只更新提供的字段。

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "node_group_id": 1,
    "enabled": false,
    "auto_scaling_enabled": false,
    "auto_failover_enabled": true,
    "auto_recovery_enabled": true,
    "min_nodes": 2,
    "max_nodes": 10,
    "scale_up_threshold": 80.0,
    "scale_down_threshold": 30.0,
    "scale_cooldown": 300,
    "failover_timeout": 60,
    "recovery_check_interval": 300,
    "created_at": "2026-03-11T10:00:00Z",
    "updated_at": "2026-03-11T11:00:00Z"
  }
}
```

---

### 4. 删除自动化策略

删除节点组的自动化策略。

**请求**

```http
DELETE /api/v1/automation/policies/{node_group_id}
Authorization: Bearer {token}
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| node_group_id | integer | 节点组 ID |

**响应**

```json
{
  "code": 0,
  "message": "success"
}
```

---

### 5. 隔离节点

手动隔离故障节点。

**请求**

```http
POST /api/v1/automation/isolations
Authorization: Bearer {token}
Content-Type: application/json
```

**请求体**

```json
{
  "node_instance_id": 5,
  "reason": "节点响应超时",
  "isolated_by": "admin"
}
```

**字段说明**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_instance_id | integer | 是 | 节点实例 ID |
| reason | string | 是 | 隔离原因 |
| isolated_by | string | 是 | 隔离发起者 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "node_instance_id": 5,
    "reason": "节点响应超时",
    "isolated_by": "admin",
    "isolated_at": "2026-03-11T10:00:00Z",
    "recovered_at": null,
    "is_active": true
  }
}
```

---

### 6. 恢复节点

恢复已隔离的节点。

**请求**

```http
POST /api/v1/automation/isolations/{node_instance_id}/recover
Authorization: Bearer {token}
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| node_instance_id | integer | 节点实例 ID |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "node_instance_id": 5,
    "reason": "节点响应超时",
    "isolated_by": "admin",
    "isolated_at": "2026-03-11T10:00:00Z",
    "recovered_at": "2026-03-11T10:30:00Z",
    "is_active": false
  }
}
```

---

### 7. 获取隔离记录

获取节点的隔离记录。

**请求**

```http
GET /api/v1/automation/isolations/{node_instance_id}
Authorization: Bearer {token}
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| node_instance_id | integer | 节点实例 ID |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "node_instance_id": 5,
    "reason": "节点响应超时",
    "isolated_by": "admin",
    "isolated_at": "2026-03-11T10:00:00Z",
    "recovered_at": null,
    "is_active": true
  }
}
```

---

### 8. 列出操作记录

获取自动化操作记录列表。

**请求**

```http
GET /api/v1/automation/actions?node_group_id=1&action_type=scale_up&status=completed&limit=10
Authorization: Bearer {token}
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_group_id | integer | 是 | 节点组 ID |
| action_type | string | 否 | 操作类型：scale_up/scale_down/failover/recover/isolate |
| status | string | 否 | 操作状态：pending/executing/completed/failed |
| limit | integer | 否 | 返回数量限制，默认 10 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "node_group_id": 1,
      "action_type": "scale_up",
      "target_node_id": 10,
      "status": "completed",
      "reason": "CPU 使用率超过 80%",
      "details": "新增节点 ID: 10",
      "executed_at": "2026-03-11T10:00:00Z",
      "completed_at": "2026-03-11T10:05:00Z"
    },
    {
      "id": 2,
      "node_group_id": 1,
      "action_type": "failover",
      "target_node_id": 5,
      "status": "completed",
      "reason": "节点心跳超时",
      "details": "流量已切换到其他节点",
      "executed_at": "2026-03-11T09:00:00Z",
      "completed_at": "2026-03-11T09:02:00Z"
    }
  ]
}
```

---

## 操作类型

| 类型 | 说明 |
|------|------|
| scale_up | 扩容 |
| scale_down | 缩容 |
| failover | 故障转移 |
| recover | 恢复 |
| isolate | 隔离 |

## 操作状态

| 状态 | 说明 |
|------|------|
| pending | 待执行 |
| executing | 执行中 |
| completed | 已完成 |
| failed | 失败 |

## 错误码

| 错误码 | 说明 |
|--------|------|
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 409 | 冲突（如节点已隔离） |
| 500 | 服务器内部错误 |

## 使用示例

### cURL

```bash
# 创建自动化策略
curl -X POST http://localhost:8080/api/v1/automation/policies \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "node_group_id": 1,
    "auto_scaling_enabled": true,
    "auto_failover_enabled": true,
    "auto_recovery_enabled": true,
    "min_nodes": 2,
    "max_nodes": 10,
    "scale_up_threshold": 80.0,
    "scale_down_threshold": 30.0
  }'

# 获取自动化策略
curl -X GET http://localhost:8080/api/v1/automation/policies/1 \
  -H "Authorization: Bearer YOUR_TOKEN"

# 隔离节点
curl -X POST http://localhost:8080/api/v1/automation/isolations \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "node_instance_id": 5,
    "reason": "节点响应超时",
    "isolated_by": "admin"
  }'

# 恢复节点
curl -X POST http://localhost:8080/api/v1/automation/isolations/5/recover \
  -H "Authorization: Bearer YOUR_TOKEN"

# 列出操作记录
curl -X GET "http://localhost:8080/api/v1/automation/actions?node_group_id=1&limit=10" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### JavaScript (Fetch)

```javascript
// 创建自动化策略
async function createPolicy() {
  const response = await fetch('http://localhost:8080/api/v1/automation/policies', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer YOUR_TOKEN',
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      node_group_id: 1,
      auto_scaling_enabled: true,
      auto_failover_enabled: true,
      auto_recovery_enabled: true,
      min_nodes: 2,
      max_nodes: 10,
      scale_up_threshold: 80.0,
      scale_down_threshold: 30.0
    })
  });

  const data = await response.json();
  console.log(data);
}

// 隔离节点
async function isolateNode() {
  const response = await fetch('http://localhost:8080/api/v1/automation/isolations', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer YOUR_TOKEN',
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      node_instance_id: 5,
      reason: '节点响应超时',
      isolated_by: 'admin'
    })
  });

  const data = await response.json();
  console.log(data);
}

// 列出操作记录
async function listActions() {
  const params = new URLSearchParams({
    node_group_id: 1,
    limit: 10
  });

  const response = await fetch(
    `http://localhost:8080/api/v1/automation/actions?${params}`,
    {
      headers: {
        'Authorization': 'Bearer YOUR_TOKEN'
      }
    }
  );

  const data = await response.json();
  console.log(data);
}
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

type CreatePolicyRequest struct {
    NodeGroupID           uint    `json:"node_group_id"`
    AutoScalingEnabled    bool    `json:"auto_scaling_enabled"`
    AutoFailoverEnabled   bool    `json:"auto_failover_enabled"`
    AutoRecoveryEnabled   bool    `json:"auto_recovery_enabled"`
    MinNodes              int     `json:"min_nodes"`
    MaxNodes              int     `json:"max_nodes"`
    ScaleUpThreshold      float64 `json:"scale_up_threshold"`
    ScaleDownThreshold    float64 `json:"scale_down_threshold"`
}

func createPolicy() error {
    req := CreatePolicyRequest{
        NodeGroupID:         1,
        AutoScalingEnabled:  true,
        AutoFailoverEnabled: true,
        AutoRecoveryEnabled: true,
        MinNodes:            2,
        MaxNodes:            10,
        ScaleUpThreshold:    80.0,
        ScaleDownThreshold:  30.0,
    }

    body, _ := json.Marshal(req)

    httpReq, _ := http.NewRequest(
        "POST",
        "http://localhost:8080/api/v1/automation/policies",
        bytes.NewBuffer(body),
    )

    httpReq.Header.Set("Authorization", "Bearer YOUR_TOKEN")
    httpReq.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return nil
}

type IsolateNodeRequest struct {
    NodeInstanceID uint   `json:"node_instance_id"`
    Reason         string `json:"reason"`
    IsolatedBy     string `json:"isolated_by"`
}

func isolateNode() error {
    req := IsolateNodeRequest{
        NodeInstanceID: 5,
        Reason:         "节点响应超时",
        IsolatedBy:     "admin",
    }

    body, _ := json.Marshal(req)

    httpReq, _ := http.NewRequest(
        "POST",
        "http://localhost:8080/api/v1/automation/isolations",
        bytes.NewBuffer(body),
    )

    httpReq.Header.Set("Authorization", "Bearer YOUR_TOKEN")
    httpReq.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return nil
}
```

## 配置建议

### 扩缩容配置

| 参数 | 建议值 | 说明 |
|------|--------|------|
| min_nodes | 2 | 保证高可用 |
| max_nodes | 10-20 | 根据业务需求 |
| scale_up_threshold | 70-80% | CPU 使用率 |
| scale_down_threshold | 20-30% | CPU 使用率 |
| scale_cooldown | 300-600s | 5-10 分钟 |

### 故障转移配置

| 参数 | 建议值 | 说明 |
|------|--------|------|
| failover_timeout | 60-120s | 1-2 分钟 |

### 自动恢复配置

| 参数 | 建议值 | 说明 |
|------|--------|------|
| recovery_check_interval | 300-600s | 5-10 分钟 |

## 权限要求

| 操作 | 所需权限 |
|------|----------|
| 创建/更新策略 | admin |
| 查看策略 | user（节点组所有者）、admin |
| 删除策略 | admin |
| 隔离/恢复节点 | admin |
| 查看操作记录 | user（节点组所有者）、admin |

## 相关文档

- [节点自动化模块文档](../modules/NODE_AUTOMATION.md)
- [节点性能监控 API](./NODE_PERFORMANCE_API.md)
- [认证文档](../guides/AUTHENTICATION.md)
