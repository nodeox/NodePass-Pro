# 节点性能监控 API 文档

## 基础信息

- **Base URL**: `/api/v1/performance`
- **认证方式**: Bearer Token
- **Content-Type**: `application/json`

## API 列表

### 1. 记录性能指标

节点上报性能指标数据。

**请求**

```http
POST /api/v1/performance/metrics
Authorization: Bearer {token}
Content-Type: application/json
```

**请求体**

```json
{
  "node_instance_id": 1,
  "cpu_usage": 45.5,
  "memory_usage": 68.2,
  "disk_usage": 55.0,
  "network_in": 1024000,
  "network_out": 2048000
}
```

**字段说明**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_instance_id | integer | 是 | 节点实例 ID |
| cpu_usage | float | 是 | CPU 使用率 (0-100) |
| memory_usage | float | 是 | 内存使用率 (0-100) |
| disk_usage | float | 是 | 磁盘使用率 (0-100) |
| network_in | integer | 是 | 入站流量 (bytes) |
| network_out | integer | 是 | 出站流量 (bytes) |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 12345,
    "node_instance_id": 1,
    "cpu_usage": 45.5,
    "memory_usage": 68.2,
    "disk_usage": 55.0,
    "network_in": 1024000,
    "network_out": 2048000,
    "recorded_at": "2026-03-11T10:00:00Z"
  }
}
```

---

### 2. 获取性能指标

查询节点的性能指标历史数据。

**请求**

```http
GET /api/v1/performance/metrics?node_instance_id=1&start_time=2026-03-11T00:00:00Z&end_time=2026-03-11T23:59:59Z&limit=100
Authorization: Bearer {token}
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_instance_id | integer | 是 | 节点实例 ID |
| start_time | string | 是 | 开始时间，ISO 8601 格式 |
| end_time | string | 是 | 结束时间，ISO 8601 格式 |
| limit | integer | 否 | 返回数量限制，默认 100 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 12345,
      "node_instance_id": 1,
      "cpu_usage": 45.5,
      "memory_usage": 68.2,
      "disk_usage": 55.0,
      "network_in": 1024000,
      "network_out": 2048000,
      "recorded_at": "2026-03-11T10:00:00Z"
    },
    {
      "id": 12346,
      "node_instance_id": 1,
      "cpu_usage": 48.3,
      "memory_usage": 70.1,
      "disk_usage": 55.2,
      "network_in": 1124000,
      "network_out": 2148000,
      "recorded_at": "2026-03-11T10:05:00Z"
    }
  ]
}
```

---

### 3. 获取性能统计

获取节点在指定时间范围内的性能统计数据。

**请求**

```http
GET /api/v1/performance/stats?node_instance_id=1&start_time=2026-03-11T00:00:00Z&end_time=2026-03-11T23:59:59Z
Authorization: Bearer {token}
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_instance_id | integer | 是 | 节点实例 ID |
| start_time | string | 是 | 开始时间，ISO 8601 格式 |
| end_time | string | 是 | 结束时间，ISO 8601 格式 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "node_instance_id": 1,
    "start_time": "2026-03-11T00:00:00Z",
    "end_time": "2026-03-11T23:59:59Z",
    "avg_cpu": 52.3,
    "max_cpu": 85.0,
    "min_cpu": 25.5,
    "avg_memory": 65.8,
    "max_memory": 90.0,
    "min_memory": 45.2,
    "avg_disk": 55.0,
    "max_disk": 58.5,
    "min_disk": 52.0,
    "total_network_in": 102400000,
    "total_network_out": 204800000
  }
}
```

---

### 4. 创建告警配置

为节点创建性能告警配置。

**请求**

```http
POST /api/v1/performance/alerts
Authorization: Bearer {token}
Content-Type: application/json
```

**请求体**

```json
{
  "node_instance_id": 1,
  "enabled": true,
  "cpu_threshold": 80.0,
  "memory_threshold": 85.0,
  "disk_threshold": 90.0,
  "network_in_limit": 10000000,
  "network_out_limit": 20000000
}
```

**字段说明**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_instance_id | integer | 是 | 节点实例 ID |
| enabled | boolean | 否 | 是否启用，默认 true |
| cpu_threshold | float | 否 | CPU 告警阈值，默认 80.0 |
| memory_threshold | float | 否 | 内存告警阈值，默认 85.0 |
| disk_threshold | float | 否 | 磁盘告警阈值，默认 90.0 |
| network_in_limit | integer | 否 | 入站流量限制，默认 0（不限制） |
| network_out_limit | integer | 否 | 出站流量限制，默认 0（不限制） |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "node_instance_id": 1,
    "enabled": true,
    "cpu_threshold": 80.0,
    "memory_threshold": 85.0,
    "disk_threshold": 90.0,
    "network_in_limit": 10000000,
    "network_out_limit": 20000000,
    "created_at": "2026-03-11T10:00:00Z",
    "updated_at": "2026-03-11T10:00:00Z"
  }
}
```

---

### 5. 获取告警配置

获取节点的告警配置。

**请求**

```http
GET /api/v1/performance/alerts/{node_instance_id}
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
    "node_instance_id": 1,
    "enabled": true,
    "cpu_threshold": 80.0,
    "memory_threshold": 85.0,
    "disk_threshold": 90.0,
    "network_in_limit": 10000000,
    "network_out_limit": 20000000,
    "created_at": "2026-03-11T10:00:00Z",
    "updated_at": "2026-03-11T10:00:00Z"
  }
}
```

---

### 6. 更新告警配置

更新节点的告警配置。

**请求**

```http
PUT /api/v1/performance/alerts/{node_instance_id}
Authorization: Bearer {token}
Content-Type: application/json
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| node_instance_id | integer | 节点实例 ID |

**请求体**

```json
{
  "enabled": false,
  "cpu_threshold": 85.0
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
    "node_instance_id": 1,
    "enabled": false,
    "cpu_threshold": 85.0,
    "memory_threshold": 85.0,
    "disk_threshold": 90.0,
    "network_in_limit": 10000000,
    "network_out_limit": 20000000,
    "created_at": "2026-03-11T10:00:00Z",
    "updated_at": "2026-03-11T11:00:00Z"
  }
}
```

---

### 7. 删除告警配置

删除节点的告警配置。

**请求**

```http
DELETE /api/v1/performance/alerts/{node_instance_id}
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
  "message": "success"
}
```

---

## 性能指标说明

### CPU 使用率
- **单位**: 百分比 (%)
- **范围**: 0-100
- **建议阈值**: 80%

### 内存使用率
- **单位**: 百分比 (%)
- **范围**: 0-100
- **建议阈值**: 85%

### 磁盘使用率
- **单位**: 百分比 (%)
- **范围**: 0-100
- **建议阈值**: 90%

### 网络流量
- **单位**: bytes
- **说明**:
  - network_in: 入站流量（下载）
  - network_out: 出站流量（上传）

## 错误码

| 错误码 | 说明 |
|--------|------|
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

## 使用示例

### cURL

```bash
# 记录性能指标
curl -X POST http://localhost:8080/api/v1/performance/metrics \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "node_instance_id": 1,
    "cpu_usage": 45.5,
    "memory_usage": 68.2,
    "disk_usage": 55.0,
    "network_in": 1024000,
    "network_out": 2048000
  }'

# 获取性能指标
curl -X GET "http://localhost:8080/api/v1/performance/metrics?node_instance_id=1&start_time=2026-03-11T00:00:00Z&end_time=2026-03-11T23:59:59Z&limit=100" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 获取性能统计
curl -X GET "http://localhost:8080/api/v1/performance/stats?node_instance_id=1&start_time=2026-03-11T00:00:00Z&end_time=2026-03-11T23:59:59Z" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 创建告警配置
curl -X POST http://localhost:8080/api/v1/performance/alerts \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "node_instance_id": 1,
    "enabled": true,
    "cpu_threshold": 80.0,
    "memory_threshold": 85.0,
    "disk_threshold": 90.0
  }'
```

### JavaScript (Fetch)

```javascript
// 记录性能指标
async function recordMetric() {
  const response = await fetch('http://localhost:8080/api/v1/performance/metrics', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer YOUR_TOKEN',
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      node_instance_id: 1,
      cpu_usage: 45.5,
      memory_usage: 68.2,
      disk_usage: 55.0,
      network_in: 1024000,
      network_out: 2048000
    })
  });

  const data = await response.json();
  console.log(data);
}

// 获取性能统计
async function getStats() {
  const params = new URLSearchParams({
    node_instance_id: 1,
    start_time: '2026-03-11T00:00:00Z',
    end_time: '2026-03-11T23:59:59Z'
  });

  const response = await fetch(
    `http://localhost:8080/api/v1/performance/stats?${params}`,
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
    "time"
)

type RecordMetricRequest struct {
    NodeInstanceID uint    `json:"node_instance_id"`
    CPUUsage       float64 `json:"cpu_usage"`
    MemoryUsage    float64 `json:"memory_usage"`
    DiskUsage      float64 `json:"disk_usage"`
    NetworkIn      uint64  `json:"network_in"`
    NetworkOut     uint64  `json:"network_out"`
}

func recordMetric() error {
    req := RecordMetricRequest{
        NodeInstanceID: 1,
        CPUUsage:       45.5,
        MemoryUsage:    68.2,
        DiskUsage:      55.0,
        NetworkIn:      1024000,
        NetworkOut:     2048000,
    }

    body, _ := json.Marshal(req)

    httpReq, _ := http.NewRequest(
        "POST",
        "http://localhost:8080/api/v1/performance/metrics",
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

// 获取性能统计
func getStats(nodeInstanceID uint, startTime, endTime time.Time) error {
    url := fmt.Sprintf(
        "http://localhost:8080/api/v1/performance/stats?node_instance_id=%d&start_time=%s&end_time=%s",
        nodeInstanceID,
        startTime.Format(time.RFC3339),
        endTime.Format(time.RFC3339),
    )

    httpReq, _ := http.NewRequest("GET", url, nil)
    httpReq.Header.Set("Authorization", "Bearer YOUR_TOKEN")

    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var result struct {
        Code    int    `json:"code"`
        Message string `json:"message"`
        Data    struct {
            AvgCPU    float64 `json:"avg_cpu"`
            MaxCPU    float64 `json:"max_cpu"`
            AvgMemory float64 `json:"avg_memory"`
            MaxMemory float64 `json:"max_memory"`
        } `json:"data"`
    }

    json.NewDecoder(resp.Body).Decode(&result)
    return nil
}
```

## 采集频率建议

| 环境 | 采集频率 | 数据保留 |
|------|----------|----------|
| 生产环境 | 1-5 分钟 | 30 天 |
| 开发环境 | 10-30 分钟 | 7 天 |
| 高负载节点 | 30 秒 - 1 分钟 | 7 天 |

## 权限要求

| 操作 | 所需权限 |
|------|----------|
| 记录性能指标 | node（节点自身） |
| 查看性能指标 | user（节点所有者）、admin |
| 创建/更新告警配置 | user（节点所有者）、admin |
| 删除告警配置 | user（节点所有者）、admin |

## 相关文档

- [节点性能监控模块文档](../modules/NODE_PERFORMANCE.md)
- [节点自动化 API](./NODE_AUTOMATION_API.md)
- [认证文档](../guides/AUTHENTICATION.md)
