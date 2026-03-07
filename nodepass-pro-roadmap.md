# NodePass 流量转发管理系统 - 开发路线图与开发计划

## 📋 项目概览

**项目名称**: NodePass Panel
**项目仓库**: [https://github.com/nodeox/NodePass-Pro](https://github.com/nodeox/NodePass-Pro)
**技术栈**: Go + Gin + Gorm + React + TypeScript + Ant Design
**开发模式**: 前后端分离

---

## 🎯 开发目标

构建一个完整的流量转发管理系统，包括：
- 后端 API 服务 (Go)
- 前端管理界面 (React)
- 节点客户端 (Go)

---

## 📅 开发路线图

### 阶段一：基础架构搭建（第 1-2 周）

**目标**: 搭建项目基础框架，实现核心功能的骨架

#### 1.1 后端基础架构（3-4 天）

**任务清单**:
- [x] 创建项目目录结构
- [ ] 初始化 Go 模块
- [ ] 配置管理（Viper）
- [ ] 数据库连接（Gorm + SQLite/MySQL/PostgreSQL）
- [ ] 数据库迁移脚本
- [ ] JWT 认证中间件
- [ ] 统一响应格式
- [ ] 错误处理中间件
- [ ] 日志系统（Zap）
- [ ] 基础路由框架

**交付物**:
```
backend/
├── cmd/server/main.go          ✓ 服务启动
├── internal/config/config.go   ✓ 配置加载
├── internal/database/db.go     ✓ 数据库连接
├── internal/middleware/        ✓ 中间件
├── internal/utils/             ✓ 工具函数
└── configs/config.yaml         ✓ 配置文件
```

**验证标准**:
- 服务能够启动并监听端口
- 数据库连接成功
- 配置文件正确加载
- 日志正常输出

#### 1.2 数据库设计与迁移（2-3 天）

**任务清单**:
- [ ] 编写数据库迁移脚本（15+ 张表）
- [ ] 定义 Gorm 模型
- [ ] 创建数据库索引
- [ ] 初始化系统配置数据
- [ ] 创建默认 VIP 等级
- [ ] 数据库迁移测试

**核心表**:
- users（用户表）
- nodes（节点表）
- node_pairs（节点配对表）
- rules（规则表）
- traffic_records（流量记录表）
- vip_levels（VIP 等级表）
- benefit_codes（权益码表）
- system_config（系统配置表）
- audit_logs（审计日志表）

**交付物**:
```
backend/
├── migrations/
│   └── 001_initial_schema.sql  ✓ 数据库迁移脚本
└── internal/models/
    ├── user.go                 ✓ 用户模型
    ├── node.go                 ✓ 节点模型
    ├── rule.go                 ✓ 规则模型
    └── ...                     ✓ 其他模型
```

**验证标准**:
- 所有表创建成功
- 索引创建成功
- 外键约束正确
- 默认数据插入成功

#### 1.3 前端基础架构（2-3 天）

**任务清单**:
- [ ] 初始化 Vite + React + TypeScript 项目
- [ ] 配置 Ant Design
- [ ] 配置 TailwindCSS
- [ ] 配置路由（React Router）
- [ ] 配置状态管理（Zustand）
- [ ] 配置 Axios 拦截器
- [ ] 创建基础布局组件
- [ ] 创建登录/注册页面

**交付物**:
```
frontend/
├── src/
│   ├── layouts/
│   │   └── MainLayout.tsx      ✓ 主布局
│   ├── pages/
│   │   └── auth/
│   │       ├── Login.tsx       ✓ 登录页
│   │       └── Register.tsx    ✓ 注册页
│   ├── services/
│   │   └── api.ts              ✓ API 客户端
│   ├── store/
│   │   └── auth.ts             ✓ 认证状态
│   └── router.tsx              ✓ 路由配置
└── package.json
```

**验证标准**:
- 前端项目能够启动
- 路由跳转正常
- 布局显示正确
- API 请求能够发送

---

### 阶段二：用户认证与权限系统（第 3 周）

**目标**: 实现完整的用户认证和权限管理系统

#### 2.1 用户认证 API（2-3 天）

**任务清单**:
- [ ] 用户注册 API
- [ ] 用户登录 API
- [ ] JWT Token 生成与验证
- [ ] 密码加密（bcrypt）
- [ ] 刷新 Token API
- [ ] 获取当前用户信息 API
- [ ] 修改密码 API
- [ ] 登出 API

**API 端点**:
```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
POST   /api/v1/auth/refresh
GET    /api/v1/auth/me
PUT    /api/v1/auth/password
```

**验证标准**:
- 用户能够注册
- 用户能够登录并获取 Token
- Token 验证正确
- 密码加密存储

#### 2.2 权限系统（2 天）

**任务清单**:
- [ ] 角色权限中间件
- [ ] 用户权限检查
- [ ] 管理员权限验证
- [ ] 权限字符串管理
- [ ] 审计日志记录

**验证标准**:
- 管理员能够访问管理接口
- 普通用户无法访问管理接口
- 操作记录到审计日志

#### 2.3 前端认证集成（2 天）

**任务清单**:
- [ ] 登录表单实现
- [ ] 注册表单实现
- [ ] Token 存储（localStorage）
- [ ] 路由守卫
- [ ] 自动刷新 Token
- [ ] 登出功能

**验证标准**:
- 用户能够登录
- 未登录自动跳转登录页
- Token 过期自动刷新
- 登出清除状态

---

### 阶段三：节点管理系统（第 4 周）

**目标**: 实现节点的创建、管理、配对功能

#### 3.1 节点管理 API（3 天）

**任务清单**:
- [ ] 创建节点 API
- [ ] 获取节点列表 API
- [ ] 获取节点详情 API
- [ ] 更新节点信息 API
- [ ] 删除节点 API
- [ ] 节点 Token 生成
- [ ] 自托管配额检查

**API 端点**:
```
POST   /api/v1/nodes
GET    /api/v1/nodes
GET    /api/v1/nodes/:id
PUT    /api/v1/nodes/:id
DELETE /api/v1/nodes/:id
GET    /api/v1/nodes/quota
```

**验证标准**:
- 能够创建节点并获取 Token
- 自托管配额限制生效
- 节点列表正确显示

#### 3.2 节点配对管理（2 天）

**任务清单**:
- [ ] 创建节点配对 API
- [ ] 获取配对列表 API
- [ ] 更新配对 API
- [ ] 删除配对 API
- [ ] 启用/禁用配对 API
- [ ] 配对验证逻辑

**API 端点**:
```
POST   /api/v1/node-pairs
GET    /api/v1/node-pairs
PUT    /api/v1/node-pairs/:id
DELETE /api/v1/node-pairs/:id
PUT    /api/v1/node-pairs/:id/toggle
```

**验证标准**:
- 能够创建节点配对
- 配对关系正确存储
- 启用/禁用功能正常

#### 3.3 前端节点管理（2 天）

**任务清单**:
- [ ] 节点列表页面
- [ ] 创建节点表单
- [ ] 编辑节点表单
- [ ] 删除节点确认
- [ ] 节点配对管理界面
- [ ] 配额显示

**验证标准**:
- 界面美观易用
- 表单验证正确
- 配额限制提示

---

### 阶段四：规则管理系统（第 5 周）

**目标**: 实现转发规则的创建和管理

#### 4.1 规则管理 API（3 天）

**任务清单**:
- [ ] 创建规则 API
- [ ] 获取规则列表 API
- [ ] 获取规则详情 API
- [ ] 更新规则 API
- [ ] 删除规则 API
- [ ] 启动规则 API
- [ ] 停止规则 API
- [ ] 规则验证逻辑（节点配对检查）

**API 端点**:
```
POST   /api/v1/rules
GET    /api/v1/rules
GET    /api/v1/rules/:id
PUT    /api/v1/rules/:id
DELETE /api/v1/rules/:id
POST   /api/v1/rules/:id/start
POST   /api/v1/rules/:id/stop
POST   /api/v1/rules/:id/restart
```

**验证标准**:
- 单节点转发规则创建成功
- 隧道转发规则创建成功
- 节点配对验证正确
- 端口冲突检测正常

#### 4.2 配置下发服务（2 天）

**任务清单**:
- [ ] 生成节点配置
- [ ] 配置版本管理
- [ ] 配置缓存
- [ ] 配置更新通知

**验证标准**:
- 配置生成正确
- 版本号递增
- 配置包含出口节点信息

#### 4.3 前端规则管理（2 天）

**任务清单**:
- [ ] 规则列表页面
- [ ] 创建规则表单（单节点/隧道）
- [ ] 编辑规则表单
- [ ] 删除规则确认
- [ ] 启动/停止按钮
- [ ] 规则状态显示

**验证标准**:
- 转发模式切换正常
- 节点选择器正确
- 配对状态提示

---

### 阶段五：节点客户端开发（第 6 周）

**目标**: 开发节点客户端，实现配置下发和离线容错

#### 5.1 节点客户端核心（3 天）

**任务清单**:
- [ ] Agent 核心框架
- [ ] 配置缓存实现
- [ ] 节点注册
- [ ] 心跳服务
- [ ] 配置拉取
- [ ] 离线容错逻辑

**交付物**:
```
nodeclient/
├── cmd/client/main.go
├── internal/
│   ├── agent/agent.go          ✓ Agent 核心
│   ├── config/cache.go         ✓ 配置缓存
│   ├── heartbeat/heartbeat.go  ✓ 心跳服务
│   └── api/client.go           ✓ API 客户端
└── configs/config.yaml
```

**验证标准**:
- 节点能够注册
- 心跳正常上报
- 配置正确缓存
- 离线时使用缓存配置

#### 5.2 NodePass 集成（2 天）

**任务清单**:
- [ ] NodePass 命令构建
- [ ] 规则启动/停止
- [ ] 实例管理
- [ ] 状态监控
- [ ] 流量统计

**验证标准**:
- 能够启动 NodePass 实例
- 单节点转发正常
- 隧道转发正常
- 流量统计准确

#### 5.3 一键安装脚本（1 天）

**任务清单**:
- [ ] 安装脚本编写
- [ ] 系统检测
- [ ] 二进制下载
- [ ] 配置文件生成
- [ ] systemd 服务创建
- [ ] 自动启动

**验证标准**:
- 脚本能够一键安装
- 服务自动启动
- 开机自启动

---

### 阶段六：流量管理系统（第 7 周）

**目标**: 实现流量统计、配额管理和超限处理

#### 6.1 流量统计 API（2 天）

**任务清单**:
- [ ] 流量上报 API
- [ ] 流量记录存储
- [ ] 流量倍率计算
- [ ] 流量聚合统计
- [ ] 获取流量使用情况 API
- [ ] 获取流量记录 API

**API 端点**:
```
POST   /api/v1/nodes/traffic/report
GET    /api/v1/traffic/quota
GET    /api/v1/traffic/usage
GET    /api/v1/traffic/records
```

**验证标准**:
- 流量上报成功
- 倍率计算正确
- 统计数据准确

#### 6.2 配额管理（2 天）

**任务清单**:
- [ ] 配额检查
- [ ] 超限处理
- [ ] 配额重置 API
- [ ] 批量同步配额
- [ ] 每月自动重置（定时任务）

**验证标准**:
- 超限自动暂停规则
- 配额重置正常
- 定时任务执行

#### 6.3 前端流量统计（2 天）

**任务清单**:
- [ ] 流量统计页面
- [ ] 配额显示
- [ ] 流量趋势图（ECharts）
- [ ] 按规则统计
- [ ] 时间筛选

**验证标准**:
- 图表显示正确
- 数据实时更新
- 筛选功能正常

---

### 阶段七：VIP 体系与权益码（第 8 周）

**目标**: 实现 VIP 等级系统和权益码功能

#### 7.1 VIP 体系 API（2 天）

**任务清单**:
- [ ] VIP 等级管理 API
- [ ] 用户升级 API
- [ ] VIP 权益应用
- [ ] VIP 过期检查（定时任务）
- [ ] VIP 历史记录

**API 端点**:
```
GET    /api/v1/vip/levels
POST   /api/v1/vip/levels
PUT    /api/v1/vip/levels/:id
GET    /api/v1/vip/my-level
POST   /api/v1/users/:id/vip/upgrade
```

**验证标准**:
- VIP 等级创建成功
- 用户升级正常
- 权益自动应用
- 过期自动降级

#### 7.2 权益码系统（2 天）

**任务清单**:
- [ ] 生成权益码 API
- [ ] 权益码列表 API
- [ ] 兑换权益码 API
- [ ] 批量操作 API
- [ ] 权益码验证

**API 端点**:
```
POST   /api/v1/benefit-codes/generate
GET    /api/v1/benefit-codes
POST   /api/v1/benefit-codes/redeem
POST   /api/v1/benefit-codes/batch-delete
```

**验证标准**:
- 权益码生成成功
- 兑换功能正常
- 状态更新正确

#### 7.3 前端 VIP 与权益码（2 天）

**任务清单**:
- [ ] VIP 等级管理页面
- [ ] 我的 VIP 页面
- [ ] 权益码管理页面
- [ ] 权益码兑换界面
- [ ] 升级选项展示

**验证标准**:
- 界面美观
- 功能完整
- 交互流畅

---

### 阶段八：Telegram 集成（第 9 周）

**目标**: 实现 Telegram Bot 和 Widget 登录

#### 8.1 Telegram Bot（2 天）

**任务清单**:
- [ ] Bot 初始化
- [ ] 命令处理（/start, /bind, /status）
- [ ] 账户绑定
- [ ] 通知发送
- [ ] Webhook 处理

**验证标准**:
- Bot 能够响应命令
- 账户绑定成功
- 通知发送正常

#### 8.2 Telegram Widget 登录（2 天）

**任务清单**:
- [ ] Widget 验证
- [ ] SSO 登录
- [ ] 登录回调处理
- [ ] 会话管理

**验证标准**:
- Widget 登录成功
- 验证签名正确
- 会话创建正常

#### 8.3 前端 Telegram 集成（1 天）

**任务清单**:
- [ ] Telegram Widget 组件
- [ ] 绑定/解绑界面
- [ ] 登录按钮

**验证标准**:
- Widget 显示正常
- 登录流程顺畅

---

### 阶段九：系统管理与监控（第 10 周）

**目标**: 实现系统配置、公告、审计日志和监控

#### 9.1 系统管理 API（2 天）

**任务清单**:
- [ ] 系统配置 API
- [ ] 公告管理 API
- [ ] 审计日志 API
- [ ] 系统统计 API

**API 端点**:
```
GET    /api/v1/system/config
PUT    /api/v1/system/config
GET    /api/v1/announcements
POST   /api/v1/announcements
GET    /api/v1/audit-logs
GET    /api/v1/system/stats
```

**验证标准**:
- 配置读写正常
- 公告显示正确
- 日志记录完整

#### 9.2 WebSocket 实时推送（2 天）

**任务清单**:
- [ ] WebSocket 服务
- [ ] 连接管理
- [ ] 消息广播
- [ ] 心跳保活
- [ ] 事件推送

**验证标准**:
- 连接稳定
- 消息实时推送
- 断线重连正常

#### 9.3 前端系统管理（2 天）

**任务清单**:
- [ ] 系统配置页面
- [ ] 公告管理页面
- [ ] 审计日志页面
- [ ] 仪表盘页面
- [ ] WebSocket 集成

**验证标准**:
- 管理界面完整
- 实时更新正常
- 数据展示清晰

---

### 阶段十：测试与优化（第 11-12 周）

**目标**: 全面测试、性能优化、文档完善

#### 10.1 功能测试（3 天）

**任务清单**:
- [ ] 用户认证测试
- [ ] 节点管理测试
- [ ] 规则管理测试
- [ ] 流量统计测试
- [ ] VIP 体系测试
- [ ] Telegram 集成测试
- [ ] 边界条件测试

#### 10.2 性能优化（3 天）

**任务清单**:
- [ ] 数据库查询优化
- [ ] API 响应时间优化
- [ ] 前端加载优化
- [ ] WebSocket 性能优化
- [ ] 缓存策略优化

#### 10.3 安全加固（2 天）

**任务清单**:
- [ ] SQL 注入防护
- [ ] XSS 防护
- [ ] CSRF 防护
- [ ] 速率限制
- [ ] 敏感数据加密

#### 10.4 文档完善（2 天）

**任务清单**:
- [ ] API 文档（Swagger）
- [ ] 部署文档
- [ ] 用户手册
- [ ] 开发文档
- [ ] README

---

## 📊 开发进度追踪

### 里程碑

| 里程碑 | 目标 | 预计完成时间 | 状态 |
|--------|------|-------------|------|
| M1 | 基础架构搭建完成 | 第 2 周末 | 🔄 进行中 |
| M2 | 用户认证系统完成 | 第 3 周末 | ⏳ 待开始 |
| M3 | 节点管理系统完成 | 第 4 周末 | ⏳ 待开始 |
| M4 | 规则管理系统完成 | 第 5 周末 | ⏳ 待开始 |
| M5 | 节点客户端完成 | 第 6 周末 | ⏳ 待开始 |
| M6 | 流量管理系统完成 | 第 7 周末 | ⏳ 待开始 |
| M7 | VIP 与权益码完成 | 第 8 周末 | ⏳ 待开始 |
| M8 | Telegram 集成完成 | 第 9 周末 | ⏳ 待开始 |
| M9 | 系统管理完成 | 第 10 周末 | ⏳ 待开始 |
| M10 | 测试与优化完成 | 第 12 周末 | ⏳ 待开始 |

### 每周目标

**第 1 周**: 后端基础架构 + 数据库设计
**第 2 周**: 前端基础架构 + 数据库迁移
**第 3 周**: 用户认证与权限系统
**第 4 周**: 节点管理系统
**第 5 周**: 规则管理系统
**第 6 周**: 节点客户端开发
**第 7 周**: 流量管理系统
**第 8 周**: VIP 体系与权益码
**第 9 周**: Telegram 集成
**第 10 周**: 系统管理与监控
**第 11-12 周**: 测试与优化

---

## 🛠️ 开发工具与环境

### 必需工具

**后端开发**:
- Go 1.21+
- PostgreSQL 14+ / MySQL 8+ / SQLite 3
- Postman / Insomnia (API 测试)
- Air (热重载)

**前端开发**:
- Node.js 18+
- npm / yarn / pnpm
- VS Code + 插件

**通用工具**:
- Git
- Docker (可选)
- Make (可选)

### 开发环境配置

```bash
# 后端
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro/backend
go mod init nodepass-panel/backend
go mod tidy
air  # 热重载开发

# 前端
cd ../frontend
npm install
npm run dev

# 节点客户端
cd ../nodeclient
go mod init nodepass-panel/nodeclient
go mod tidy
go run cmd/client/main.go
```

---

## 📝 开发规范

### 代码规范

**Go 代码**:
- 遵循 Go 官方代码规范
- 使用 gofmt 格式化
- 使用 golint 检查
- 注释清晰完整

**TypeScript 代码**:
- 遵循 ESLint 规则
- 使用 Prettier 格式化
- 类型定义完整
- 组件注释清晰

### Git 提交规范

```
feat: 新功能
fix: 修复 bug
docs: 文档更新
style: 代码格式调整
refactor: 重构
test: 测试相关
chore: 构建/工具相关
```

### 分支管理

```
main        - 主分支（生产环境）
develop     - 开发分支
feature/*   - 功能分支
bugfix/*    - 修复分支
release/*   - 发布分支
```

---

## 🎯 关键成功因素

1. **按阶段交付**: 每个阶段完成后进行验收
2. **持续测试**: 开发过程中持续测试
3. **文档同步**: 代码和文档同步更新
4. **代码审查**: 重要功能进行代码审查
5. **性能监控**: 关注性能指标
6. **安全优先**: 安全问题优先处理

---

## 📞 支持与反馈

**架构文档**: [nodepass-pro-architecture.md](https://github.com/nodeox/NodePass-Pro/blob/main/nodepass-pro-architecture.md)
**开发路线图**: [nodepass-pro-roadmap.md](https://github.com/nodeox/NodePass-Pro/blob/main/nodepass-pro-roadmap.md)
**项目仓库**: [https://github.com/nodeox/NodePass-Pro](https://github.com/nodeox/NodePass-Pro)

---

**最后更新**: 2026-03-06
**版本**: v1.0
