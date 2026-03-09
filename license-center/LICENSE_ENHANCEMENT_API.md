# 授权码管理增强功能 API 文档

## 概述

本文档描述了授权码管理系统的所有增强功能，包括：

1. **批量操作** - 批量更新、转移、吊销、恢复、删除
2. **高级搜索** - 多条件搜索、保存搜索条件
3. **模板系统** - 创建模板、从模板生成授权码
4. **统计报表** - 详细统计、趋势分析、过期预警
5. **分组管理** - 按项目/客户分组、分组统计
6. **自动化功能** - 自动续期、过期通知（后端支持）

## 基础信息

- **Base URL**: `http://localhost:8090/api/v1`
- **认证方式**: Bearer Token (JWT)
- **请求头**: `Authorization: Bearer <token>`

---

## 1. 授权码模板管理

### 1.1 查询模板列表

```http
GET /license-templates
```

**响应示例**:
```json
{
  "code": 0,
  "message": "ok",
  "data": [
    {
      "id": 1,
      "name": "标准版模板",
      "description": "用于生成标准版授权码",
      "plan_id": 1,
      "plan": {
        "id": 1,
        "name": "标准版",
        "code": "standard"
      },
      "duration_days": 365,
      "max_machines": 1,
      "max_domains": 1,
      "prefix": "STD",
      "note": "标准版授权",
      "usage_count": 10,
      "is_enabled": true,
      "created_by": 1,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### 1.2 创建模板

```http
POST /license-templates
```

**请求体**:
```json
{
  "name": "企业版模板",
  "description": "用于生成企业版授权码",
  "plan_id": 2,
  "duration_days": 365,
  "max_machines": 10,
  "max_domains": 5,
  "prefix": "ENT",
  "note": "企业版授权"
}
```

### 1.3 从模板生成授权码

```http
POST /license-templates/generate
```

**请求体**:
```json
{
  "template_id": 1,
  "customer": "测试客户",
  "count": 5,
  "expires_at": "2025-12-31T23:59:59Z"
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "生成成功",
  "data": [
    {
      "id": 101,
      "key": "STD-XXXX-XXXX-XXXX",
      "plan_id": 1,
      "customer": "测试客户",
      "status": "active",
      "expires_at": "2025-12-31T23:59:59Z",
      "max_machines": 1,
      "max_domains": 1,
      "note": "标准版授权"
    }
  ]
}
```

### 1.4 更新模板

```http
PUT /license-templates/:id
```

### 1.5 删除模板

```http
DELETE /license-templates/:id
```

### 1.6 启用/禁用模板

```http
POST /license-templates/:id/toggle
```

**请求体**:
```json
{
  "enabled": false
}
```

---

## 2. 授权码分组管理

### 2.1 查询分组列表

```http
GET /license-groups
```

**响应示例**:
```json
{
  "code": 0,
  "message": "ok",
  "data": [
    {
      "id": 1,
      "name": "项目A",
      "description": "项目A的所有授权码",
      "type": "project",
      "color": "#1890ff",
      "icon": "project",
      "sort_order": 1,
      "license_count": 15,
      "is_enabled": true,
      "active_count": 12,
      "expired_count": 2,
      "revoked_count": 1,
      "created_by": 1,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### 2.2 创建分组

```http
POST /license-groups
```

**请求体**:
```json
{
  "name": "项目B",
  "description": "项目B的所有授权码",
  "type": "project",
  "color": "#52c41a",
  "icon": "project",
  "sort_order": 2
}
```

**分组类型**:
- `project` - 项目分组
- `customer` - 客户分组
- `custom` - 自定义分组

### 2.3 添加授权码到分组

```http
POST /license-groups/:id/licenses
```

**请求体**:
```json
{
  "license_ids": [1, 2, 3, 4, 5]
}
```

### 2.4 从分组移除授权码

```http
DELETE /license-groups/:id/licenses
```

**请求体**:
```json
{
  "license_ids": [1, 2]
}
```

### 2.5 获取分组内的授权码

```http
GET /license-groups/:id/licenses?page=1&page_size=20
```

### 2.6 获取分组统计信息

```http
GET /license-groups/:id/stats
```

**响应示例**:
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "total_count": 15,
    "active_count": 12,
    "expired_count": 2,
    "revoked_count": 1
  }
}
```

### 2.7 获取授权码所属的分组

```http
GET /licenses/:id/groups
```

---

## 3. 批量操作

### 3.1 批量更新

```http
POST /licenses/batch/update-enhanced
```

**请求体**:
```json
{
  "license_ids": [1, 2, 3],
  "updates": {
    "status": "active",
    "max_machines": 5,
    "note": "批量更新备注"
  }
}
```

**允许更新的字段**:
- `status` - 状态
- `expires_at` - 过期时间
- `max_machines` - 最大机器数
- `note` - 备注
- `max_domains` - 最大域名数

### 3.2 批量转移客户

```http
POST /licenses/batch/transfer
```

**请求体**:
```json
{
  "license_ids": [1, 2, 3],
  "new_customer": "新客户名称"
}
```

### 3.3 批量吊销

```http
POST /licenses/batch/revoke-enhanced
```

**请求体**:
```json
{
  "license_ids": [1, 2, 3]
}
```

### 3.4 批量恢复

```http
POST /licenses/batch/restore-enhanced
```

**请求体**:
```json
{
  "license_ids": [1, 2, 3]
}
```

### 3.5 批量删除

```http
POST /licenses/batch/delete-enhanced
```

**请求体**:
```json
{
  "license_ids": [1, 2, 3]
}
```

---

## 4. 高级搜索

### 4.1 高级搜索

```http
POST /licenses/search/advanced
```

**请求体**:
```json
{
  "status": ["active", "expired"],
  "customer": "测试",
  "plan_ids": [1, 2],
  "group_ids": [1],
  "expires_from": "2024-01-01T00:00:00Z",
  "expires_to": "2024-12-31T23:59:59Z",
  "created_from": "2024-01-01T00:00:00Z",
  "created_to": "2024-12-31T23:59:59Z",
  "key_pattern": "STD",
  "note": "测试",
  "has_activations": true,
  "page": 1,
  "page_size": 20
}
```

**搜索条件说明**:
- `status` - 状态列表（可多选）
- `customer` - 客户名称（模糊匹配）
- `plan_ids` - 套餐ID列表
- `group_ids` - 分组ID列表
- `expires_from/expires_to` - 过期时间范围
- `created_from/created_to` - 创建时间范围
- `key_pattern` - 授权码模式（模糊匹配）
- `note` - 备注（模糊匹配）
- `has_activations` - 是否有激活记录

**响应示例**:
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [...],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

### 4.2 保存搜索条件

```http
POST /licenses/search/save
```

**请求体**:
```json
{
  "name": "即将过期的授权码",
  "description": "查询30天内即将过期的授权码",
  "filter": {
    "status": ["active"],
    "expires_from": "2024-01-01T00:00:00Z",
    "expires_to": "2024-01-31T23:59:59Z"
  },
  "is_public": false
}
```

### 4.3 查询保存的搜索

```http
GET /licenses/search/saved
```

**响应示例**:
```json
{
  "code": 0,
  "message": "ok",
  "data": [
    {
      "id": 1,
      "name": "即将过期的授权码",
      "description": "查询30天内即将过期的授权码",
      "filter_json": "{...}",
      "usage_count": 5,
      "is_public": false,
      "created_by": 1,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### 4.4 获取保存的搜索详情

```http
GET /licenses/search/saved/:id
```

### 4.5 删除保存的搜索

```http
DELETE /licenses/search/saved/:id
```

---

## 5. 统计与报表

### 5.1 获取详细统计信息

```http
GET /licenses/statistics
```

**响应示例**:
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "total_count": 1000,
    "active_count": 800,
    "expired_count": 150,
    "revoked_count": 50,
    "expiring_in_7_days": 20,
    "expiring_in_30_days": 80,
    "customer_count": 50,
    "by_plan": {
      "standard": 600,
      "enterprise": 400
    },
    "by_status": {
      "active": 800,
      "expired": 150,
      "revoked": 50
    },
    "trend_data": [
      {
        "date": "2024-01-01",
        "count": 10
      },
      {
        "date": "2024-01-02",
        "count": 15
      }
    ]
  }
}
```

**统计指标说明**:
- `total_count` - 总授权码数
- `active_count` - 活跃授权码数
- `expired_count` - 已过期授权码数
- `revoked_count` - 已吊销授权码数
- `expiring_in_7_days` - 7天内即将过期数
- `expiring_in_30_days` - 30天内即将过期数
- `customer_count` - 客户数量
- `by_plan` - 按套餐统计
- `by_status` - 按状态统计
- `trend_data` - 最近7天趋势数据

### 5.2 获取即将过期的授权码

```http
GET /licenses/expiring?days=7
```

**参数**:
- `days` - 提前多少天（默认7天）

**响应示例**:
```json
{
  "code": 0,
  "message": "ok",
  "data": [
    {
      "id": 1,
      "key": "STD-XXXX-XXXX-XXXX",
      "customer": "测试客户",
      "status": "active",
      "expires_at": "2024-01-07T23:59:59Z",
      "plan": {
        "name": "标准版",
        "code": "standard"
      }
    }
  ]
}
```

---

## 6. 使用场景示例

### 场景1: 为新项目批量生成授权码

```bash
# 1. 创建项目分组
curl -X POST http://localhost:8090/api/v1/license-groups \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "项目X",
    "type": "project",
    "color": "#1890ff"
  }'

# 2. 从模板生成授权码
curl -X POST http://localhost:8090/api/v1/license-templates/generate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "template_id": 1,
    "customer": "客户A",
    "count": 10
  }'

# 3. 将生成的授权码添加到分组
curl -X POST http://localhost:8090/api/v1/license-groups/1/licenses \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "license_ids": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
  }'
```

### 场景2: 查找并处理即将过期的授权码

```bash
# 1. 查询30天内即将过期的授权码
curl -X GET "http://localhost:8090/api/v1/licenses/expiring?days=30" \
  -H "Authorization: Bearer <token>"

# 2. 批量延期
curl -X POST http://localhost:8090/api/v1/licenses/batch/update-enhanced \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "license_ids": [1, 2, 3],
    "updates": {
      "expires_at": "2025-12-31T23:59:59Z"
    }
  }'
```

### 场景3: 客户转移

```bash
# 批量转移客户
curl -X POST http://localhost:8090/api/v1/licenses/batch/transfer \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "license_ids": [1, 2, 3, 4, 5],
    "new_customer": "新客户名称"
  }'
```

### 场景4: 高级搜索并保存

```bash
# 1. 执行高级搜索
curl -X POST http://localhost:8090/api/v1/licenses/search/advanced \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "status": ["active"],
    "customer": "测试",
    "expires_from": "2024-01-01T00:00:00Z",
    "expires_to": "2024-12-31T23:59:59Z",
    "page": 1,
    "page_size": 20
  }'

# 2. 保存搜索条件
curl -X POST http://localhost:8090/api/v1/licenses/search/save \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "2024年测试客户",
    "description": "2024年所有测试客户的活跃授权码",
    "filter": {
      "status": ["active"],
      "customer": "测试",
      "expires_from": "2024-01-01T00:00:00Z",
      "expires_to": "2024-12-31T23:59:59Z"
    }
  }'
```

---

## 7. 错误码说明

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 8. 数据模型

### LicenseTemplate (授权码模板)

```typescript
interface LicenseTemplate {
  id: number
  name: string
  description: string
  plan_id: number
  plan: LicensePlan
  duration_days?: number
  max_machines?: number
  max_domains?: number
  prefix: string
  note: string
  usage_count: number
  is_enabled: boolean
  created_by: number
  created_at: string
  updated_at: string
}
```

### LicenseGroup (授权码分组)

```typescript
interface LicenseGroup {
  id: number
  name: string
  description: string
  type: 'project' | 'customer' | 'custom'
  color: string
  icon: string
  sort_order: number
  license_count: number
  is_enabled: boolean
  created_by: number
  created_at: string
  updated_at: string
}
```

### SavedSearch (保存的搜索)

```typescript
interface SavedSearch {
  id: number
  name: string
  description: string
  filter_json: string
  usage_count: number
  is_public: boolean
  created_by: number
  created_at: string
  updated_at: string
}
```

---

## 9. 最佳实践

### 9.1 模板使用

- 为常用的授权码配置创建模板
- 使用有意义的前缀区分不同类型的授权码
- 定期检查模板使用情况，优化模板配置

### 9.2 分组管理

- 按项目或客户进行分组，便于管理
- 使用不同颜色和图标区分分组
- 定期查看分组统计，了解授权码使用情况

### 9.3 批量操作

- 批量操作前先使用高级搜索确认目标授权码
- 重要操作前做好数据备份
- 批量删除操作需谨慎，建议先吊销再删除

### 9.4 搜索优化

- 保存常用的搜索条件，提高效率
- 使用高级搜索的多个条件组合，精确定位目标
- 定期清理不再使用的保存搜索

---

## 10. 注意事项

1. **权限控制**: 所有 API 都需要管理员权限
2. **批量操作限制**: 单次批量操作建议不超过 200 条记录
3. **搜索性能**: 复杂搜索条件可能影响性能，建议合理使用分页
4. **数据一致性**: 批量操作会在事务中执行，确保数据一致性
5. **模板管理**: 删除模板不会影响已生成的授权码
6. **分组管理**: 删除分组不会删除授权码，只删除关联关系

---

## 11. 更新日志

### v1.0.0 (2024-01-01)

- ✅ 实现授权码模板系统
- ✅ 实现授权码分组管理
- ✅ 实现批量操作功能
- ✅ 实现高级搜索功能
- ✅ 实现保存搜索功能
- ✅ 实现详细统计报表
- ✅ 实现过期预警功能
- ✅ 添加数据库模型和迁移
- ✅ 添加完整的 API 接口
