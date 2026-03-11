# 公告 API 文档

## 基础信息

- **Base URL**: `/api/v1/announcements`
- **认证方式**: Bearer Token
- **Content-Type**: `application/json`

## API 列表

### 1. 创建公告

创建一个新的系统公告。

**请求**

```http
POST /api/v1/announcements
Authorization: Bearer {token}
Content-Type: application/json
```

**请求体**

```json
{
  "title": "系统维护通知",
  "content": "系统将于今晚 22:00-24:00 进行维护，期间服务可能中断。",
  "type": "warning",
  "is_enabled": true,
  "start_time": "2026-03-11T22:00:00Z",
  "end_time": "2026-03-12T00:00:00Z"
}
```

**字段说明**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 是 | 公告标题，1-200 字符 |
| content | string | 是 | 公告内容，1-5000 字符 |
| type | string | 是 | 公告类型：info/warning/error/success |
| is_enabled | boolean | 否 | 是否启用，默认 true |
| start_time | string | 否 | 开始时间，ISO 8601 格式 |
| end_time | string | 否 | 结束时间，ISO 8601 格式 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "title": "系统维护通知",
    "content": "系统将于今晚 22:00-24:00 进行维护，期间服务可能中断。",
    "type": "warning",
    "is_enabled": true,
    "start_time": "2026-03-11T22:00:00Z",
    "end_time": "2026-03-12T00:00:00Z",
    "created_at": "2026-03-11T10:00:00Z",
    "updated_at": "2026-03-11T10:00:00Z"
  }
}
```

---

### 2. 获取公告详情

根据 ID 获取公告详情。

**请求**

```http
GET /api/v1/announcements/{id}
Authorization: Bearer {token}
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | integer | 公告 ID |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "title": "系统维护通知",
    "content": "系统将于今晚 22:00-24:00 进行维护，期间服务可能中断。",
    "type": "warning",
    "is_enabled": true,
    "start_time": "2026-03-11T22:00:00Z",
    "end_time": "2026-03-12T00:00:00Z",
    "created_at": "2026-03-11T10:00:00Z",
    "updated_at": "2026-03-11T10:00:00Z"
  }
}
```

---

### 3. 列出公告

获取公告列表，支持筛选。

**请求**

```http
GET /api/v1/announcements?only_enabled=true&only_active=true
Authorization: Bearer {token}
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| only_enabled | boolean | 否 | 只返回启用的公告 |
| only_active | boolean | 否 | 只返回有效期内的公告 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "title": "系统维护通知",
      "content": "系统将于今晚 22:00-24:00 进行维护。",
      "type": "warning",
      "is_enabled": true,
      "start_time": "2026-03-11T22:00:00Z",
      "end_time": "2026-03-12T00:00:00Z",
      "created_at": "2026-03-11T10:00:00Z",
      "updated_at": "2026-03-11T10:00:00Z"
    },
    {
      "id": 2,
      "title": "新功能上线",
      "content": "新版本已上线，欢迎体验。",
      "type": "info",
      "is_enabled": true,
      "start_time": null,
      "end_time": null,
      "created_at": "2026-03-10T10:00:00Z",
      "updated_at": "2026-03-10T10:00:00Z"
    }
  ]
}
```

---

### 4. 更新公告

更新公告信息。

**请求**

```http
PUT /api/v1/announcements/{id}
Authorization: Bearer {token}
Content-Type: application/json
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | integer | 公告 ID |

**请求体**

```json
{
  "title": "系统维护通知（已延期）",
  "is_enabled": false
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
    "title": "系统维护通知（已延期）",
    "content": "系统将于今晚 22:00-24:00 进行维护。",
    "type": "warning",
    "is_enabled": false,
    "start_time": "2026-03-11T22:00:00Z",
    "end_time": "2026-03-12T00:00:00Z",
    "created_at": "2026-03-11T10:00:00Z",
    "updated_at": "2026-03-11T11:00:00Z"
  }
}
```

---

### 5. 删除公告

删除指定的公告。

**请求**

```http
DELETE /api/v1/announcements/{id}
Authorization: Bearer {token}
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | integer | 公告 ID |

**响应**

```json
{
  "code": 0,
  "message": "success"
}
```

---

## 公告类型

| 类型 | 说明 | 使用场景 |
|------|------|----------|
| info | 信息 | 一般通知、新功能介绍 |
| warning | 警告 | 维护通知、功能变更 |
| error | 错误 | 紧急通知、服务中断 |
| success | 成功 | 好消息、活动通知 |

## 错误码

| 错误码 | 说明 |
|--------|------|
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 无权限 |
| 404 | 公告不存在 |
| 500 | 服务器内部错误 |

## 使用示例

### cURL

```bash
# 创建公告
curl -X POST http://localhost:8080/api/v1/announcements \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "系统维护通知",
    "content": "系统将于今晚进行维护",
    "type": "warning",
    "is_enabled": true
  }'

# 获取有效公告列表
curl -X GET "http://localhost:8080/api/v1/announcements?only_enabled=true&only_active=true" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 更新公告
curl -X PUT http://localhost:8080/api/v1/announcements/1 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "is_enabled": false
  }'

# 删除公告
curl -X DELETE http://localhost:8080/api/v1/announcements/1 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### JavaScript (Fetch)

```javascript
// 创建公告
async function createAnnouncement() {
  const response = await fetch('http://localhost:8080/api/v1/announcements', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer YOUR_TOKEN',
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      title: '系统维护通知',
      content: '系统将于今晚进行维护',
      type: 'warning',
      is_enabled: true
    })
  });

  const data = await response.json();
  console.log(data);
}

// 获取有效公告列表
async function getActiveAnnouncements() {
  const response = await fetch(
    'http://localhost:8080/api/v1/announcements?only_enabled=true&only_active=true',
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

type CreateAnnouncementRequest struct {
    Title     string `json:"title"`
    Content   string `json:"content"`
    Type      string `json:"type"`
    IsEnabled bool   `json:"is_enabled"`
}

func createAnnouncement() error {
    req := CreateAnnouncementRequest{
        Title:     "系统维护通知",
        Content:   "系统将于今晚进行维护",
        Type:      "warning",
        IsEnabled: true,
    }

    body, _ := json.Marshal(req)

    httpReq, _ := http.NewRequest(
        "POST",
        "http://localhost:8080/api/v1/announcements",
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

## 权限要求

| 操作 | 所需权限 |
|------|----------|
| 创建公告 | admin |
| 查看公告 | 所有用户 |
| 更新公告 | admin |
| 删除公告 | admin |

## 相关文档

- [公告模块文档](../modules/ANNOUNCEMENT.md)
- [认证文档](../guides/AUTHENTICATION.md)
