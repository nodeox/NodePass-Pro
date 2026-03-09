# 快速开始 - 隧道批量导入导出和模板功能

## 🚀 5 分钟快速上手

### 步骤 1：安装（2 分钟）

```bash
cd /Users/jianshe/Projects/NodePass-Pro
./install-tunnel-enhancement.sh
```

安装脚本会自动：
- ✅ 安装后端依赖
- ✅ 运行数据库迁移
- ✅ 编译后端服务
- ✅ 可选：构建前端

### 步骤 2：启动服务（1 分钟）

```bash
# 启动后端
cd backend
go run ./cmd/server/main.go

# 或使用编译后的二进制
./bin/nodepass-server
```

### 步骤 3：体验功能（2 分钟）

#### 🎯 导出隧道

1. 登录系统，进入"我的隧道"页面
2. 选中几个隧道（勾选复选框）
3. 点击"更多操作" → "导出选中"
4. 选择 JSON 格式
5. 文件自动下载 ✅

#### 📥 导入隧道

1. 点击"更多操作" → "导入隧道"
2. 选择格式：JSON
3. 粘贴刚才导出的内容
4. 选择入口节点组
5. 勾选"跳过错误"
6. 点击"开始导入" ✅

#### 📋 使用模板

1. 找到一个配置好的隧道
2. 点击"更多" → "保存为模板"
3. 输入模板名称："我的标准配置"
4. 保存 ✅
5. 点击"更多操作" → "模板管理"
6. 找到刚创建的模板，点击"应用"
7. 输入新隧道名称，选择节点组
8. 创建完成 ✅

## 📖 常用操作

### 导出所有隧道

```bash
# 方式 1：通过界面
更多操作 → 导出全部 → 选择格式

# 方式 2：通过 API
curl -X POST http://localhost:8080/api/v1/tunnels/export-all \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"format": "json"}'
```

### 批量导入

```bash
# 准备配置文件 tunnels.json
{
  "version": "1.0",
  "tunnels": [
    {
      "name": "隧道1",
      "protocol": "tcp",
      "remote_host": "example.com",
      "remote_port": 443,
      ...
    }
  ]
}

# 通过界面导入
更多操作 → 导入隧道 → 粘贴内容 → 开始导入
```

### 创建和应用模板

```bash
# 1. 保存为模板
隧道操作 → 更多 → 保存为模板

# 2. 应用模板
更多操作 → 模板管理 → 选择模板 → 应用
```

## 🎯 实用技巧

### 技巧 1：快速备份

每周自动备份配置：

```bash
#!/bin/bash
# backup-tunnels.sh

DATE=$(date +%Y%m%d)
curl -X POST http://localhost:8080/api/v1/tunnels/export-all \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"format": "yaml"}' \
  | jq -r '.data.data' > "backup-$DATE.yaml"

# 提交到 Git
git add "backup-$DATE.yaml"
git commit -m "Backup tunnels $DATE"
git push
```

### 技巧 2：环境迁移

从测试环境迁移到生产环境：

```bash
# 1. 测试环境导出
curl ... > test-tunnels.json

# 2. 修改配置（可选）
# 编辑 test-tunnels.json，修改节点组 ID

# 3. 生产环境导入
# 通过界面导入，选择生产环境的节点组
```

### 技巧 3：标准化部署

为多个客户部署标准配置：

```bash
# 1. 创建标准模板
- TCP 游戏加速模板
- UDP 语音通话模板
- WebSocket 实时通信模板

# 2. 设置为公开模板

# 3. 团队成员应用模板
- 只需修改客户名称
- 选择对应的节点组
- 一键创建
```

## 📚 更多资源

- [详细使用文档](./tunnel-import-export-template.md)
- [功能演示](./tunnel-enhancement-demo.md)
- [实现总结](./IMPLEMENTATION_SUMMARY.md)
- [API 文档](./api-documentation.md)

## ❓ 常见问题

**Q: 导入失败怎么办？**
```
A: 检查以下几点：
1. 配置格式是否正确（JSON/YAML）
2. 节点组是否存在
3. 启用"跳过错误"选项
4. 查看错误详情
```

**Q: 如何批量修改配置？**
```
A:
1. 导出配置
2. 使用文本编辑器批量修改
3. 重新导入
```

**Q: 模板可以共享吗？**
```
A: 可以，创建模板时选择"公开模板"
```

**Q: 导出的配置包含密码吗？**
```
A: 不包含，导出的配置不含敏感信息
```

## 🆘 获取帮助

遇到问题？

1. 查看[详细文档](./tunnel-import-export-template.md)
2. 查看[功能演示](./tunnel-enhancement-demo.md)
3. 提交 [GitHub Issue](https://github.com/your-repo/issues)

## 🎉 开始使用

现在你已经掌握了基本用法，开始体验强大的隧道管理功能吧！

```bash
# 启动服务
cd backend && go run ./cmd/server/main.go

# 访问前端
http://localhost:3000
```

祝使用愉快！🚀

---

**更新日期**：2026-03-08
**版本**：v1.0
