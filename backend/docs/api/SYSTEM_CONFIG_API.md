# 系统配置 API 文档

## 基础信息

- **Base URL**: `/api/v1/configs`
- **认证方式**: Bearer Token
- **Content-Type**: `application/json`

## API 列表

### 1. 创建或更新配置

创建新配置或更新已存在的配置（Upsert 操作）。

**请求**

```http
POST /api/v1/configs
Authorization: Bearer {token}
Content-Type: application/json
```

**请求体**

```json
{
  "key": "app.name",
  "value": "NodePass Pro",
  "description": "应用名称"
}
```

**字段说明**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| key | string | 是 | 配置键，1-100 字符 |
| value | string | 否 | 配置值，可为空 |
| description | string | 否 | 配置描述 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "key": "app.name",
    "value": "NodePass Pro",
    "description": "应用名称",
    "updated_at": "2026-03-11T10:00:00Z"
  }
}
```

---

### 2. 批量更新配置

批量创建或更新多个配置。

**请求**

```http
POST /api/v1/configs/batch
Authorization: Bearer {token}
Content-Type: application/json
```

**请求体**

```json
{
  "configs": [
    {
      "key": "mail.smtp.host",
      "value": "smtp.example.com",
      "description": "SMTP 服务器地址"
    },
    {
      "key": "mail.smtp.port",
      "value": "587",
      "description": "SMTP 端口"
    },
    {
      "key": "mail.smtp.username",
      "value": "noreply@example.com"
    }
  ]
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success_count": 3,
    "failed_count": 0
  }
}
```

---

### 3. 获取配置

根据键获取单个配置。

**请求**

```http
GET /api/v1/configs/{key}
Authorization: Bearer {token}
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| key | string | 配置键 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "key": "app.name",
    "value": "NodePass Pro",
    "description": "应用名称",
    "updated_at": "2026-03-11T10:00:00Z"
  }
}
```

---

### 4. 列出所有配置

获取所有配置列表。

**请求**

```http
GET /api/v1/configs
Authorization: Bearer {token}
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| prefix | string | 否 | 按前缀筛选，如 "mail." |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "key": "app.name",
      "value": "NodePass Pro",
      "description": "应用名称",
      "updated_at": "2026-03-11T10:00:00Z"
    },
    {
      "id": 2,
      "key": "app.version",
      "value": "1.0.0",
      "description": "应用版本",
      "updated_at": "2026-03-11T10:00:00Z"
    }
  ]
}
```

---

### 5. 获取所有配置（Map 格式）

获取所有配置，以键值对 Map 格式返回。

**请求**

```http
GET /api/v1/configs/map
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "app.name": "NodePass Pro",
    "app.version": "1.0.0",
    "mail.smtp.host": "smtp.example.com",
    "mail.smtp.port": "587"
  }
}
```

---

### 6. 删除配置

删除指定的配置。

**请求**

```http
DELETE /api/v1/configs/{key}
Authorization: Bearer {token}
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| key | string | 配置键 |

**响应**

```json
{
  "code": 0,
  "message": "success"
}
```

---

## 配置命名规范

### 推荐的命名空间

```
app.*                应用相关配置
mail.*               邮件相关配置
feature.*            功能开关
system.*             系统限制
performance.*        性能配置
security.*           安全配置
```

### 命名示例

```
app.name                        应用名称
app.version                     应用版本
app.env                         运行环境

mail.smtp.host                  SMTP 服务器
mail.smtp.port                  SMTP 端口
mail.smtp.username              SMTP 用户名
mail.smtp.password              SMTP 密码

feature.registration_enabled    注册功能开关
feature.invite_only             仅邀请注册
feature.email_verification      邮箱验证开关

system.max_users                最大用户数
system.max_nodes_per_user       每用户最大节点数

performance.cache_ttl           缓存 TTL
performance.heartbeat_interval  心跳间隔
```

## 错误码

| 错误码 | 说明 |
|--------|------|
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 无权限 |
| 404 | 配置不存在 |
| 500 | 服务器内部错误 |

## 使用示例

### cURL

```bash
# 创建或更新配置
curl -X POST http://localhost:8080/api/v1/configs \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "app.name",
    "value": "NodePass Pro",
    "description": "应用名称"
  }'

# 批量更新配置
curl -X POST http://localhost:8080/api/v1/configs/batch \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "configs": [
      {"key": "mail.smtp.host", "value": "smtp.example.com"},
      {"key": "mail.smtp.port", "value": "587"}
    ]
  }'

# 获取配置
curl -X GET http://localhost:8080/api/v1/configs/app.name \
  -H "Authorization: Bearer YOUR_TOKEN"

# 获取所有配置（Map 格式）
curl -X GET http://localhost:8080/api/v1/configs/map \
  -H "Authorization: Bearer YOUR_TOKEN"

# 删除配置
curl -X DELETE http://localhost:8080/api/v1/configs/app.name \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### JavaScript (Fetch)

```javascript
// 创建或更新配置
async function upsertConfig() {
  const response = await fetch('http://localhost:8080/api/v1/configs', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer YOUR_TOKEN',
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      key: 'app.name',
      value: 'NodePass Pro',
      description: '应用名称'
    })
  });

  const data = await response.json();
  console.log(data);
}

// 获取所有配置（Map 格式）
async function getAllConfigsAsMap() {
  const response = await fetch('http://localhost:8080/api/v1/configs/map', {
    headers: {
      'Authorization': 'Bearer YOUR_TOKEN'
    }
  });

  const data = await response.json();
  console.log(data.data); // { "app.name": "NodePass Pro", ... }
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

type UpsertConfigRequest struct {
    Key         string  `json:"key"`
    Value       *string `json:"value,omitempty"`
    Description *string `json:"description,omitempty"`
}

func upsertConfig() error {
    value := "NodePass Pro"
    desc := "应用名称"

    req := UpsertConfigRequest{
        Key:         "app.name",
        Value:       &value,
        Description: &desc,
    }

    body, _ := json.Marshal(req)

    httpReq, _ := http.NewRequest(
        "POST",
        "http://localhost:8080/api/v1/configs",
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

// 获取所有配置为 Map
func getAllConfigsAsMap() (map[string]string, error) {
    httpReq, _ := http.NewRequest(
        "GET",
        "http://localhost:8080/api/v1/configs/map",
        nil,
    )

    httpReq.Header.Set("Authorization", "Bearer YOUR_TOKEN")

    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Code    int               `json:"code"`
        Message string            `json:"message"`
        Data    map[string]string `json:"data"`
    }

    json.NewDecoder(resp.Body).Decode(&result)
    return result.Data, nil
}
```

## 配置类型转换

虽然配置值存储为字符串，但可以在应用层进行类型转换：

```go
// 获取布尔值配置
func GetBoolConfig(key string, defaultValue bool) bool {
    config, err := configService.GetConfig(ctx, key)
    if err != nil {
        return defaultValue
    }

    value := config.GetValueOrDefault(strconv.FormatBool(defaultValue))
    result, _ := strconv.ParseBool(value)
    return result
}

// 获取整数配置
func GetIntConfig(key string, defaultValue int) int {
    config, err := configService.GetConfig(ctx, key)
    if err != nil {
        return defaultValue
    }

    value := config.GetValueOrDefault(strconv.Itoa(defaultValue))
    result, _ := strconv.Atoi(value)
    return result
}

// 获取 JSON 配置
func GetJSONConfig(key string, target interface{}) error {
    config, err := configService.GetConfig(ctx, key)
    if err != nil {
        return err
    }

    if config.Value == nil {
        return errors.New("config value is nil")
    }

    return json.Unmarshal([]byte(*config.Value), target)
}
```

## 权限要求

| 操作 | 所需权限 |
|------|----------|
| 创建/更新配置 | admin |
| 查看配置 | admin |
| 删除配置 | admin |

## 相关文档

- [系统配置模块文档](../modules/SYSTEM_CONFIG.md)
- [认证文档](../guides/AUTHENTICATION.md)
