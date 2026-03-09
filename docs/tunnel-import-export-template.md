# 隧道批量导入导出和模板功能

## 概述

新增了隧道配置的批量导入导出功能和模板管理系统，大幅提升隧道配置的复用性和管理效率。

## 功能特性

### 1. 批量导出

**功能**：
- 导出选中的隧道配置
- 导出所有隧道配置
- 支持 JSON 和 YAML 两种格式
- 导出内容包含完整的隧道配置信息

**使用方法**：
1. 在隧道列表页面，点击"更多操作"按钮
2. 选择"导出选中"或"导出全部"
3. 选择导出格式（JSON 或 YAML）
4. 配置文件将自动下载到本地

**导出格式示例**：

```json
{
  "version": "1.0",
  "export_at": "2026-03-08T10:30:00Z",
  "tunnels": [
    {
      "name": "美国入口到日本出口",
      "description": "用于游戏加速",
      "protocol": "tcp",
      "listen_host": "0.0.0.0",
      "listen_port": 8080,
      "remote_host": "game.example.com",
      "remote_port": 443,
      "config": {
        "load_balance_strategy": "round_robin",
        "ip_type": "auto",
        "enable_proxy_protocol": false,
        "forward_targets": [],
        "health_check_interval": 30,
        "health_check_timeout": 5
      }
    }
  ]
}
```

### 2. 批量导入

**功能**：
- 从 JSON 或 YAML 文件导入隧道配置
- 支持批量创建多个隧道
- 可选择跳过错误继续导入
- 显示详细的导入结果和错误信息

**使用方法**：
1. 点击"更多操作" → "导入隧道"
2. 选择导入格式（JSON 或 YAML）
3. 粘贴配置数据
4. 选择入口节点组和出口节点组（可选）
5. 勾选"跳过错误"选项（推荐）
6. 点击"开始导入"

**注意事项**：
- 导入时必须指定入口节点组
- 出口节点组可选，不指定则为直连模式
- 监听端口会自动分配，避免冲突
- 跳过错误模式下，部分失败不影响其他隧道导入

### 3. 隧道模板

**功能**：
- 保存常用隧道配置为模板
- 快速应用模板创建新隧道
- 支持公开模板供其他用户使用
- 统计模板使用次数

**创建模板**：
1. 在隧道列表中找到要保存的隧道
2. 点击"更多" → "保存为模板"
3. 输入模板名称和描述
4. 选择是否公开模板
5. 点击"保存"

**应用模板**：
1. 点击"更多操作" → "模板管理"
2. 在模板列表中找到需要的模板
3. 点击"应用"按钮
4. 输入新隧道的名称
5. 选择入口和出口节点组
6. 点击"应用"创建隧道

**模板特性**：
- 模板包含完整的协议配置
- 支持私有模板和公开模板
- 公开模板可被所有用户使用
- 自动记录模板使用次数

## API 接口

### 导出接口

**导出选中隧道**：
```
POST /api/v1/tunnels/export
Content-Type: application/json

{
  "tunnel_ids": [1, 2, 3],
  "format": "json"
}
```

**导出所有隧道**：
```
POST /api/v1/tunnels/export-all
Content-Type: application/json

{
  "format": "yaml"
}
```

### 导入接口

```
POST /api/v1/tunnels/import
Content-Type: application/json

{
  "format": "json",
  "data": "...",
  "entry_group_id": 1,
  "exit_group_id": 2,
  "skip_errors": true
}
```

**响应示例**：
```json
{
  "code": 0,
  "message": "导入成功",
  "data": {
    "total": 5,
    "success": 4,
    "failed": 1,
    "errors": [
      {
        "index": 2,
        "name": "测试隧道",
        "message": "监听端口已被占用"
      }
    ],
    "tunnels": [...]
  }
}
```

### 模板接口

**创建模板**：
```
POST /api/v1/tunnel-templates
Content-Type: application/json

{
  "name": "TCP 游戏加速模板",
  "description": "适用于游戏加速场景",
  "protocol": "tcp",
  "config": {...},
  "is_public": false
}
```

**列出模板**：
```
GET /api/v1/tunnel-templates?protocol=tcp&is_public=true&page=1&page_size=20
```

**应用模板**：
```
POST /api/v1/tunnels/apply-template
Content-Type: application/json

{
  "template_id": 1,
  "name": "新隧道名称",
  "description": "描述",
  "entry_group_id": 1,
  "exit_group_id": 2
}
```

## 使用场景

### 场景 1：环境迁移

当需要将隧道配置从测试环境迁移到生产环境时：

1. 在测试环境导出所有隧道配置（JSON 格式）
2. 在生产环境导入配置
3. 选择生产环境的节点组
4. 批量创建隧道

### 场景 2：配置备份

定期备份隧道配置：

1. 使用"导出全部"功能
2. 选择 YAML 格式（更易读）
3. 保存到版本控制系统
4. 需要时可快速恢复

### 场景 3：批量部署

为多个客户部署相同配置的隧道：

1. 创建标准配置模板
2. 设置为公开模板
3. 团队成员应用模板
4. 只需修改名称和节点组

### 场景 4：配置复用

常用配置保存为模板：

1. 将 TCP、UDP、WebSocket 等常用配置保存为模板
2. 新建隧道时直接应用模板
3. 减少重复配置工作
4. 确保配置一致性

## 数据库变更

### 新增表

```sql
CREATE TABLE tunnel_templates (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    protocol VARCHAR(20) NOT NULL,
    config_json TEXT NOT NULL,
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    usage_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_tunnel_templates_user_id (user_id),
    INDEX idx_tunnel_templates_is_public (is_public),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

## 文件清单

### 后端文件

1. **models/tunnel_template.go** - 模板数据模型
2. **services/tunnel_template_service.go** - 模板服务层
3. **services/tunnel_import_export.go** - 导入导出服务
4. **handlers/tunnel_template_handler.go** - 模板和导入导出处理器
5. **migrations/0003_create_tunnel_templates.up.sql** - 数据库迁移（创建）
6. **migrations/0003_create_tunnel_templates.down.sql** - 数据库迁移（回滚）

### 前端文件

1. **services/nodeGroupApi.ts** - API 客户端（已更新）
2. **pages/tunnels/TunnelList.tsx** - 隧道列表页面（已更新）

## 安装步骤

### 1. 运行数据库迁移

```bash
cd backend
go run ./cmd/migrate up
```

### 2. 重启后端服务

```bash
cd backend
go run ./cmd/server/main.go
```

### 3. 重新构建前端

```bash
cd frontend
npm run build
```

## 测试建议

### 功能测试

- [ ] 导出单个隧道（JSON 格式）
- [ ] 导出多个隧道（YAML 格式）
- [ ] 导出所有隧道
- [ ] 导入 JSON 格式配置
- [ ] 导入 YAML 格式配置
- [ ] 导入时跳过错误
- [ ] 创建私有模板
- [ ] 创建公开模板
- [ ] 应用模板创建隧道
- [ ] 删除模板

### 边界测试

- [ ] 导入空配置
- [ ] 导入格式错误的数据
- [ ] 导入时节点组不存在
- [ ] 导入时端口冲突
- [ ] 模板名称重复
- [ ] 应用不存在的模板
- [ ] 非所有者删除模板

### 性能测试

- [ ] 导出 100+ 隧道
- [ ] 导入 100+ 隧道配置
- [ ] 查询大量模板列表

## 注意事项

1. **权限控制**：
   - 用户只能导出自己的隧道
   - 管理员可以导出所有隧道
   - 用户只能删除自己创建的模板
   - 公开模板所有人可见和使用

2. **数据安全**：
   - 导出的配置不包含敏感信息（如认证令牌）
   - 导入时会验证所有配置参数
   - 建议定期备份导出的配置文件

3. **兼容性**：
   - 导出格式版本号为 1.0
   - 未来版本会保持向后兼容
   - 导入时会检查版本兼容性

4. **性能优化**：
   - 大批量导入建议分批进行
   - 导出大量隧道可能需要较长时间
   - 建议在低峰期进行批量操作

## 后续优化建议

1. **导入导出增强**：
   - 支持 CSV 格式导入导出
   - 支持从 URL 导入配置
   - 添加配置验证和预览功能
   - 支持增量导入（更新已存在的隧道）

2. **模板增强**：
   - 模板分类和标签
   - 模板评分和评论
   - 模板市场（共享优质模板）
   - 模板版本管理

3. **自动化**：
   - 定时自动导出备份
   - 配置变更通知
   - 批量操作日志记录

4. **集成**：
   - 与 Git 集成（配置版本控制）
   - 与 CI/CD 集成（自动化部署）
   - API 接口文档生成

---

**实现完成时间**：2026-03-08
**实现状态**：✅ 已完成
