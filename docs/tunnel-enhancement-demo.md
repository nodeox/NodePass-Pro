# 隧道管理增强功能演示

## 功能概览

本次更新为 NodePass-Pro 添加了强大的隧道配置管理功能，包括批量导入导出和模板系统。

## 🎯 核心功能

### 1. 批量导出

**场景**：需要备份或迁移隧道配置

```bash
# 在前端界面操作
1. 选中需要导出的隧道（可多选）
2. 点击"更多操作" → "导出选中"
3. 选择格式：JSON 或 YAML
4. 文件自动下载：tunnels-export-{timestamp}.json
```

**导出示例**：
```json
{
  "version": "1.0",
  "export_at": "2026-03-08T10:30:00Z",
  "tunnels": [
    {
      "name": "游戏加速隧道",
      "protocol": "tcp",
      "listen_host": "0.0.0.0",
      "listen_port": 8080,
      "remote_host": "game.example.com",
      "remote_port": 443,
      "config": {
        "load_balance_strategy": "round_robin",
        "ip_type": "auto"
      }
    }
  ]
}
```

### 2. 批量导入

**场景**：从备份恢复或批量创建隧道

```bash
# 在前端界面操作
1. 点击"更多操作" → "导入隧道"
2. 选择格式（JSON/YAML）
3. 粘贴配置数据
4. 选择入口节点组（必选）
5. 选择出口节点组（可选）
6. 勾选"跳过错误"（推荐）
7. 点击"开始导入"
```

**导入结果**：
```
✅ 导入完成：成功 4 个，失败 1 个

错误详情：
  3. 测试隧道: 监听端口已被占用
```

### 3. 隧道模板

**场景**：保存常用配置，快速创建相似隧道

#### 创建模板
```bash
1. 找到配置良好的隧道
2. 点击"更多" → "保存为模板"
3. 输入模板名称："TCP 游戏加速模板"
4. 添加描述："适用于游戏加速场景"
5. 选择是否公开（公开后其他用户可用）
6. 保存
```

#### 应用模板
```bash
1. 点击"更多操作" → "模板管理"
2. 浏览可用模板列表
3. 找到需要的模板，点击"应用"
4. 输入新隧道名称
5. 选择节点组
6. 创建完成
```

## 📊 使用场景

### 场景 1：环境迁移

**需求**：将测试环境的 50 个隧道配置迁移到生产环境

**步骤**：
1. 测试环境：导出全部隧道（JSON 格式）
2. 生产环境：导入配置
3. 选择生产环境的节点组
4. 一键创建 50 个隧道

**时间节省**：从 2 小时手动配置 → 5 分钟批量导入

### 场景 2：配置备份

**需求**：每周备份隧道配置

**步骤**：
1. 导出全部隧道（YAML 格式，更易读）
2. 保存到 Git 仓库
3. 配置变更可追踪
4. 需要时快速恢复

### 场景 3：标准化部署

**需求**：为 100 个客户部署标准配置

**步骤**：
1. 创建标准配置模板（TCP、UDP、WebSocket）
2. 设置为公开模板
3. 团队成员应用模板
4. 只需修改客户名称和节点组

**效率提升**：配置时间减少 80%

### 场景 4：快速测试

**需求**：测试不同协议和负载均衡策略

**步骤**：
1. 保存各种配置为模板
2. 快速应用模板创建测试隧道
3. 测试完成后批量删除
4. 下次测试再次应用模板

## 🔧 API 使用示例

### 导出隧道（cURL）

```bash
# 导出选中的隧道
curl -X POST http://localhost:8080/api/v1/tunnels/export \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tunnel_ids": [1, 2, 3],
    "format": "json"
  }'

# 导出所有隧道
curl -X POST http://localhost:8080/api/v1/tunnels/export-all \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "yaml"
  }'
```

### 导入隧道（cURL）

```bash
curl -X POST http://localhost:8080/api/v1/tunnels/import \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "json",
    "data": "{\"version\":\"1.0\",\"tunnels\":[...]}",
    "entry_group_id": 1,
    "exit_group_id": 2,
    "skip_errors": true
  }'
```

### 创建模板（cURL）

```bash
curl -X POST http://localhost:8080/api/v1/tunnel-templates \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "TCP 游戏加速模板",
    "description": "适用于游戏加速场景",
    "protocol": "tcp",
    "config": {
      "remote_host": "game.example.com",
      "remote_port": 443,
      "load_balance_strategy": "round_robin",
      "ip_type": "auto",
      "enable_proxy_protocol": false,
      "forward_targets": []
    },
    "is_public": false
  }'
```

### 应用模板（cURL）

```bash
curl -X POST http://localhost:8080/api/v1/tunnels/apply-template \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "template_id": 1,
    "name": "客户A游戏加速",
    "description": "为客户A部署",
    "entry_group_id": 1,
    "exit_group_id": 2
  }'
```

## 📸 界面截图说明

### 导出功能
- 隧道列表页面新增"更多操作"按钮
- 下拉菜单包含：导出选中、导出全部、导入隧道、模板管理
- 导出对话框支持选择 JSON 或 YAML 格式

### 导入功能
- 导入对话框包含格式选择、数据输入、节点组选择
- 支持"跳过错误"选项
- 显示详细的导入结果和错误信息

### 模板功能
- 隧道操作菜单新增"保存为模板"选项
- 模板管理对话框显示所有可用模板
- 支持应用模板和删除模板操作

## 🚀 性能优化

- 批量导出使用流式处理，支持大量隧道
- 导入采用并发创建，提高效率
- 模板查询使用索引优化
- 前端使用虚拟滚动处理大列表

## 🔒 安全特性

- 导出的配置不包含敏感信息
- 导入时验证所有配置参数
- 模板权限控制（私有/公开）
- API 接口需要认证

## 📝 注意事项

1. **导入时端口处理**：监听端口会自动分配，避免冲突
2. **节点组要求**：导入时必须指定入口节点组
3. **错误处理**：建议启用"跳过错误"选项进行批量导入
4. **模板权限**：公开模板所有用户可见和使用
5. **数据备份**：建议定期导出配置作为备份

## 🎓 最佳实践

1. **命名规范**：使用清晰的命名，如"TCP-游戏-美国-日本"
2. **模板分类**：按协议和用途创建模板
3. **定期备份**：每周导出一次完整配置
4. **版本控制**：将导出的配置提交到 Git
5. **测试先行**：在测试环境验证导入配置

## 📚 相关文档

- [详细使用文档](./tunnel-import-export-template.md)
- [功能总结](./tunnel-enhancement-summary.md)
- [隧道管理增强](./tunnel-enhancement.md)

## 🆘 常见问题

**Q: 导入失败怎么办？**
A: 检查配置格式是否正确，节点组是否存在，启用"跳过错误"选项。

**Q: 模板可以修改吗？**
A: 可以，在模板管理中选择模板进行编辑。

**Q: 导出的配置可以在其他系统使用吗？**
A: 可以，但需要确保目标系统有相应的节点组。

**Q: 公开模板会被其他用户修改吗？**
A: 不会，其他用户只能查看和应用，不能修改原模板。

---

**更新日期**：2026-03-08
**版本**：v1.0
