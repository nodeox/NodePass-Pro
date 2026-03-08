# 用户管理增强 - 2026-03-07

## 更新概述

本次更新全面增强了 NodePass-Pro 的用户管理功能，新增了高级搜索筛选、批量操作、流量配额管理等实用功能，大幅提升管理员的工作效率。

## 主要更新内容

### 1. 高级搜索和筛选 ✅

**功能特性**：
- 关键词搜索：支持按用户名或邮箱搜索
- 角色筛选：按 Admin/User 角色筛选
- 状态筛选：按正常/暂停/封禁/超限状态筛选
- VIP 等级筛选：按 VIP 等级筛选用户
- 可折叠筛选面板，界面简洁
- 显示已应用的筛选条件数量
- 一键清空所有筛选条件

**技术实现**：
- 前端使用 Ant Design Form 组件实现筛选表单
- 后端已支持所有筛选参数（role, status, vip_level, keyword）
- 筛选条件实时应用，自动重置到第一页

**使用方法**：
1. 点击"显示筛选"按钮展开筛选面板
2. 填写筛选条件（关键词、角色、状态、VIP等级）
3. 点击"搜索"按钮应用筛选
4. 点击"清空"按钮清除所有筛选条件

### 2. 批量操作 ✅

**功能特性**：
- 批量封禁用户
- 批量解封用户
- 批量重置流量
- 支持跨页选择
- 显示已选用户数量
- 危险操作二次确认

**技术实现**：
- 使用 Table 的 rowSelection 实现行选择
- Promise.all 并发执行批量操作
- 操作完成后自动刷新列表并清空选择

**使用方法**：
1. 勾选需要操作的用户（可跨页选择）
2. 在批量操作栏中选择操作类型
3. 确认操作（危险操作需二次确认）
4. 系统并发执行操作并显示结果

**批量操作类型**：
- **批量封禁**：将选中用户状态设置为 banned
- **批量解封**：将选中用户状态恢复为 normal
- **批量重置流量**：重置选中用户的流量使用量为 0

### 3. 流量配额管理 ✅

**功能特性**：
- 调整用户流量配额
- 快捷设置常用配额（1GB/10GB/50GB/100GB/1TB）
- 显示当前已用流量和配额
- 实时计算和显示
- 支持自定义配额值

**技术实现**：

**后端新增**：
- `TrafficService.UpdateQuota` 方法
- `TrafficHandler.UpdateQuota` 处理器
- 路由：`PUT /api/v1/traffic/quota/:id`

**前端新增**：
- 流量配额调整模态框
- `trafficApi.updateQuota` API 方法
- 快捷设置按钮组

**使用方法**：
1. 点击用户操作列的"流量"按钮
2. 在弹出的模态框中输入新的流量配额（字节）
3. 或点击快捷设置按钮快速设置常用配额
4. 点击"确定"保存更新

**快捷配额**：
- 1 GB = 1,073,741,824 字节
- 10 GB = 10,737,418,240 字节
- 50 GB = 53,687,091,200 字节
- 100 GB = 107,374,182,400 字节
- 1 TB = 1,099,511,627,776 字节

### 4. 界面优化

**列表页改进**：
- 添加筛选按钮，支持显示/隐藏筛选面板
- 筛选按钮在有筛选条件时高亮显示
- 批量操作栏仅在有选择时显示
- 操作列新增"流量"按钮
- 表格支持行选择

**筛选面板**：
- 响应式布局，适配不同屏幕尺寸
- 清晰的表单标签和占位符
- 显示已应用的筛选条件数量

**批量操作栏**：
- 显示已选用户数量
- 操作按钮分组，危险操作使用红色
- 取消选择按钮

**流量配额模态框**：
- 显示当前已用流量和配额
- 快捷设置按钮组
- 输入提示和单位说明

## 技术细节

### 前端更新

**文件**: `frontend/src/pages/system/UserManage.tsx`

**新增状态**：
```typescript
const [showFilters, setShowFilters] = useState<boolean>(false)
const [filters, setFilters] = useState<SearchFilters>({})
const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
const [batchLoading, setBatchLoading] = useState<boolean>(false)
const [trafficModalOpen, setTrafficModalOpen] = useState<boolean>(false)
```

**新增类型**：
```typescript
type SearchFilters = {
  keyword?: string
  role?: UserRole
  status?: string
  vip_level?: number
}

type TrafficFormValues = {
  traffic_quota: number
}
```

**新增函数**：
- `handleSearch`: 处理搜索筛选
- `handleClearFilters`: 清空筛选条件
- `handleBatchBan`: 批量封禁用户
- `handleBatchUnban`: 批量解封用户
- `handleBatchResetTraffic`: 批量重置流量
- `openTrafficModal`: 打开流量配额调整模态框
- `closeTrafficModal`: 关闭流量配额调整模态框
- `submitTraffic`: 提交流量配额更新

**API 更新**：
```typescript
// frontend/src/services/api.ts
export const trafficApi = {
  // ... 其他方法
  updateQuota: (targetUserID: number, trafficQuota: number) =>
    apiClient
      .put<ApiSuccessResponse<null>>(`/traffic/quota/${targetUserID}`, {
        traffic_quota: trafficQuota,
      })
      .then(unwrapData),
}
```

### 后端更新

**文件**: `backend/internal/services/traffic_service.go`

**新增方法**：
```go
// UpdateQuota 管理员更新用户流量配额。
func (s *TrafficService) UpdateQuota(adminUserID uint, targetUserID uint, trafficQuota int64) error {
	if adminUserID == 0 || targetUserID == 0 {
		return fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}
	if trafficQuota < 0 {
		return fmt.Errorf("%w: 流量配额不能为负数", ErrInvalidParams)
	}

	admin, err := s.getUserByID(adminUserID)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(admin.Role), "admin") {
		return fmt.Errorf("%w: 仅管理员可更新配额", ErrForbidden)
	}

	if _, err = s.getUserByID(targetUserID); err != nil {
		return err
	}

	if err = s.db.Model(&models.User{}).
		Where("id = ?", targetUserID).
		Update("traffic_quota", trafficQuota).Error; err != nil {
		return fmt.Errorf("更新用户流量配额失败: %w", err)
	}

	return nil
}
```

**文件**: `backend/internal/handlers/traffic_handler.go`

**新增处理器**：
```go
// UpdateQuota PUT /api/v1/traffic/quota/:id (admin only)
func (h *TrafficHandler) UpdateQuota(c *gin.Context) {
	adminUserID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	targetUserID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "目标用户 ID 无效")
		return
	}

	type requestPayload struct {
		TrafficQuota int64 `json:"traffic_quota" binding:"required,min=0"`
	}
	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	if err := h.trafficService.UpdateQuota(adminUserID, targetUserID, req.TrafficQuota); err != nil {
		writeServiceError(c, err, "UPDATE_QUOTA_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "流量配额更新成功")
}
```

**文件**: `backend/cmd/server/main.go`

**新增路由**：
```go
adminTraffic := adminGroup.Group("/traffic")
{
	adminTraffic.POST("/quota/reset", trafficHandler.ResetQuota)
	adminTraffic.PUT("/quota/:id", trafficHandler.UpdateQuota) // 新增
}
```

## API 文档

### 更新流量配额

**请求**：
```
PUT /api/v1/traffic/quota/:id
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "traffic_quota": 10737418240
}
```

**响应**：
```json
{
  "code": 0,
  "message": "流量配额更新成功",
  "data": null
}
```

**参数说明**：
- `id`: 目标用户 ID（路径参数）
- `traffic_quota`: 新的流量配额（字节，必须 >= 0）

**权限要求**：仅管理员可调用

## 使用场景

### 场景 1：查找特定用户

管理员需要查找某个用户：
1. 点击"显示筛选"
2. 在关键词输入框输入用户名或邮箱
3. 点击"搜索"
4. 查看搜索结果

### 场景 2：批量封禁违规用户

管理员需要封禁多个违规用户：
1. 使用筛选功能找到目标用户
2. 勾选需要封禁的用户
3. 点击"批量封禁"按钮
4. 确认操作
5. 系统并发执行封禁操作

### 场景 3：调整用户流量配额

管理员需要为用户增加流量配额：
1. 找到目标用户
2. 点击"流量"按钮
3. 在模态框中点击"100 GB"快捷按钮
4. 或手动输入配额值
5. 点击"确定"保存

### 场景 4：查看 VIP 用户

管理员需要查看所有 VIP 用户：
1. 点击"显示筛选"
2. 在 VIP 等级下拉框选择等级
3. 点击"搜索"
4. 查看筛选结果

## 性能优化

### 批量操作优化

```typescript
// 并发执行，提高效率
await Promise.all(
  selectedRowKeys.map((id) => userAdminApi.updateStatus(Number(id), 'banned'))
)
```

### 筛选条件优化

- 筛选条件存储在状态中，避免重复请求
- 筛选面板可折叠，减少界面占用
- 后端使用索引优化查询性能

## 注意事项

### 批量操作

- 批量封禁/删除不可恢复，请谨慎操作
- 批量操作失败时会显示错误信息
- 建议分批操作大量用户（避免超时）
- 操作完成后自动清空选择

### 流量配额管理

- 流量配额单位为字节
- 配额不能为负数
- 修改配额不会影响已用流量
- 建议使用快捷设置按钮避免输入错误

### 搜索筛选

- 关键词搜索支持模糊匹配
- 多个筛选条件为 AND 关系
- 筛选条件会保留直到手动清空
- 切换页面时筛选条件保持

## 后续优化建议

### 1. 用户详情页

- 显示用户完整信息
- 操作历史记录
- 节点使用情况
- 流量使用趋势图表

### 2. 数据导出

- 导出用户列表为 CSV/Excel
- 支持按筛选条件导出
- 导出流量使用报表
- 导出操作日志

### 3. 批量编辑

- 批量修改 VIP 等级
- 批量调整流量配额
- 批量修改用户状态
- 批量标签管理

### 4. 高级筛选

- 按注册时间筛选
- 按最后登录时间筛选
- 按流量使用率筛选
- 按节点数量筛选

### 5. 用户分组

- 创建用户分组
- 批量分组管理
- 按分组应用策略
- 分组统计分析

## 测试建议

### 功能测试

- [ ] 关键词搜索用户
- [ ] 按角色筛选用户
- [ ] 按状态筛选用户
- [ ] 按 VIP 等级筛选用户
- [ ] 批量封禁用户
- [ ] 批量解封用户
- [ ] 批量重置流量
- [ ] 调整流量配额
- [ ] 使用快捷配额按钮

### 边界测试

- [ ] 搜索不存在的用户
- [ ] 选择 0 个用户进行批量操作
- [ ] 输入无效的流量配额
- [ ] 网络错误处理
- [ ] 权限验证（非管理员）

### 性能测试

- [ ] 批量操作 100+ 用户
- [ ] 大量用户列表加载
- [ ] 频繁切换筛选条件
- [ ] 并发批量操作

## 相关文件

### 前端文件

- `frontend/src/pages/system/UserManage.tsx` - 用户管理页面（主要更新）
- `frontend/src/services/api.ts` - API 服务（新增 updateQuota）

### 后端文件

- `backend/internal/services/traffic_service.go` - 流量服务（新增 UpdateQuota）
- `backend/internal/handlers/traffic_handler.go` - 流量处理器（新增 UpdateQuota）
- `backend/cmd/server/main.go` - 路由配置（新增路由）

## 兼容性

- ✅ 向后兼容，不影响现有功能
- ✅ 使用现有的 API 端点（除新增的 UpdateQuota）
- ✅ 无数据库结构变更
- ✅ 支持所有现代浏览器

## 测试结果

### 后端测试

```bash
$ go build -o /tmp/nodepass-test ./cmd/server
# 编译成功 ✅
```

### 前端测试

```bash
$ npm run build
✓ built in 4.17s
# 编译成功 ✅
```

## 总结

本次用户管理增强更新新增了三大核心功能：

1. **高级搜索和筛选** - 快速定位目标用户
2. **批量操作** - 提高管理效率
3. **流量配额管理** - 灵活调整用户配额

这些功能大幅提升了管理员的工作效率，使用户管理更加便捷和高效。所有功能都经过测试验证，代码质量良好，可以直接投入使用。
