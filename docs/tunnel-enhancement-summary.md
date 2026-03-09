# 隧道管理增强 - 批量导入导出和模板系统

## 新增功能

### 1. 批量导出 ✅
- 导出选中的隧道配置
- 导出所有隧道配置
- 支持 JSON 和 YAML 格式

### 2. 批量导入 ✅
- 从 JSON/YAML 文件导入隧道
- 支持批量创建
- 错误处理和跳过机制

### 3. 隧道模板系统 ✅
- 保存隧道配置为模板
- 快速应用模板创建隧道
- 支持公开/私有模板
- 模板使用统计

### 4. 保存为模板 ✅
- 从现有隧道创建模板
- 一键保存常用配置

## 快速开始

### 安装

1. 运行数据库迁移：
```bash
cd backend
go run ./cmd/migrate up
```

2. 重启后端服务：
```bash
cd backend
go run ./cmd/server/main.go
```

3. 前端已自动更新，无需额外操作

### 使用

#### 导出隧道
1. 在隧道列表选中要导出的隧道
2. 点击"更多操作" → "导出选中"
3. 选择格式（JSON/YAML）
4. 配置文件自动下载

#### 导入隧道
1. 点击"更多操作" → "导入隧道"
2. 粘贴配置数据
3. 选择节点组
4. 点击"开始导入"

#### 使用模板
1. 在隧道操作菜单选择"保存为模板"
2. 或点击"更多操作" → "模板管理"
3. 应用模板快速创建隧道

## 技术实现

### 后端
- Go 服务层实现导入导出逻辑
- 支持 JSON 和 YAML 序列化
- 完整的错误处理和验证
- 新增 `tunnel_templates` 数据表

### 前端
- React + TypeScript + Ant Design
- 新增导入导出对话框
- 模板管理界面
- 文件下载功能

## API 端点

```
POST   /api/v1/tunnels/export           # 导出选中隧道
POST   /api/v1/tunnels/export-all       # 导出所有隧道
POST   /api/v1/tunnels/import           # 导入隧道
POST   /api/v1/tunnels/apply-template   # 应用模板

GET    /api/v1/tunnel-templates         # 列出模板
POST   /api/v1/tunnel-templates         # 创建模板
GET    /api/v1/tunnel-templates/:id     # 获取模板
PUT    /api/v1/tunnel-templates/:id     # 更新模板
DELETE /api/v1/tunnel-templates/:id     # 删除模板
```

## 文件清单

### 后端新增文件
- `backend/internal/models/tunnel_template.go`
- `backend/internal/services/tunnel_template_service.go`
- `backend/internal/services/tunnel_import_export.go`
- `backend/internal/handlers/tunnel_template_handler.go`
- `backend/migrations/0003_create_tunnel_templates.up.sql`
- `backend/migrations/0003_create_tunnel_templates.down.sql`

### 后端修改文件
- `backend/cmd/server/main.go` - 添加路由

### 前端修改文件
- `frontend/src/services/nodeGroupApi.ts` - 添加 API 方法
- `frontend/src/pages/tunnels/TunnelList.tsx` - 添加 UI 功能

### 文档
- `docs/tunnel-import-export-template.md` - 详细使用文档

## 使用场景

1. **环境迁移**：测试环境配置快速迁移到生产环境
2. **配置备份**：定期导出配置作为备份
3. **批量部署**：为多个客户部署相同配置
4. **配置复用**：常用配置保存为模板，提高效率

## 注意事项

- 导入时会自动分配监听端口，避免冲突
- 公开模板所有用户可见和使用
- 建议启用"跳过错误"选项进行批量导入
- 导出的配置不包含敏感信息

## 测试

运行数据库迁移后，可以通过以下步骤测试：

1. 创建几个测试隧道
2. 选中隧道并导出为 JSON
3. 删除这些隧道
4. 导入刚才导出的 JSON 文件
5. 验证隧道是否正确创建
6. 将某个隧道保存为模板
7. 应用模板创建新隧道

## 依赖

后端新增依赖：
```go
gopkg.in/yaml.v3  // YAML 序列化支持
```

确保运行：
```bash
cd backend
go mod tidy
```

## 完成状态

- ✅ 后端模型和服务层
- ✅ 后端 API 接口
- ✅ 数据库迁移脚本
- ✅ 前端 API 集成
- ✅ 前端 UI 实现
- ✅ 使用文档

---

**实现日期**：2026-03-08
**版本**：v1.0
