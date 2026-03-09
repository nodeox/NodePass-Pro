# 版本管理系统 - 快速启动指南

## 当前服务状态

✅ **后端服务** (PID: 48290)
- 地址: http://localhost:8090
- 版本: 1.0.0
- 状态: 运行中

✅ **前端服务** (PID: 49032)
- 地址: http://localhost:3000
- 版本: 1.0.0
- 状态: 运行中

## 访问方式

### Web 界面
1. 打开浏览器访问: http://localhost:3000
2. 使用以下凭据登录:
   - 用户名: `admin`
   - 密码: `Y/dbZI+QuaRhw858R8oxmw==`
3. 点击左侧菜单的"版本管理"

### API 接口
- 基础地址: http://localhost:8090/api/v1
- 健康检查: http://localhost:8090/health
- 需要 JWT Token 认证

## 停止服务

```bash
# 停止后端服务
kill 48290

# 停止前端服务
kill 49032
```

## 重新启动

### 后端
```bash
cd /Users/jianshe/Projects/NodePass-Pro/license-center
nohup go run cmd/server/main.go --config configs/config.yaml > /tmp/license-center.log 2>&1 &
```

### 前端
```bash
cd /Users/jianshe/Projects/NodePass-Pro/license-center/web-ui
npm run dev > /tmp/web-ui.log 2>&1 &
```

## 查看日志

```bash
# 后端日志
tail -f /tmp/license-center.log

# 前端日志
tail -f /tmp/web-ui.log
```

## 功能说明

### 版本管理页面包含三个标签页:

1. **组件版本**
   - 查看所有组件的当前版本
   - 更新组件版本
   - 查看版本详情

2. **版本历史**
   - 查看每个组件的历史版本
   - 按时间倒序显示

3. **兼容性配置**
   - 管理版本兼容性矩阵
   - 创建新的兼容性配置

### 当前数据

已初始化的版本数据:
- Backend: 1.0.0
- Frontend: 1.0.0
- Node Client: 1.0.0
- License Center: 1.0.0

兼容性配置:
- Backend 1.0.0 要求所有组件最低版本为 1.0.0

## 配置文件

- 后端配置: `configs/config.yaml`
- 数据库: `data/license-center.db` (SQLite)

## 文档

- 功能文档: `docs/VERSION_MANAGEMENT.md`
- 集成报告: `VERSION_MANAGEMENT_INTEGRATION.md`
- API 测试脚本: `scripts/test-version-api.sh`

## 注意事项

1. 首次启动需要创建 `data` 目录
2. 配置文件中的密码仅用于开发测试
3. 生产环境请修改 JWT Secret 和管理员密码
4. 数据存储在 SQLite 数据库中，重启后数据保留

---

**版本管理系统已就绪，可以开始使用！** 🎉
