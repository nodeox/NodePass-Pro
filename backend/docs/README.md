# NodePass-Pro 后端文档

## 📚 文档导航

### 架构文档
- [DDD 架构概览](./architecture/DDD_ARCHITECTURE.md) - 领域驱动设计架构说明
- [CQRS 模式](./architecture/CQRS_PATTERN.md) - 命令查询职责分离模式
- [缓存策略](./architecture/CACHE_STRATEGY.md) - Redis 多级缓存设计
- [数据库设计](./architecture/DATABASE_DESIGN.md) - PostgreSQL 数据库设计

### 模块文档
- [公告模块](./modules/ANNOUNCEMENT.md) - 系统公告管理
- [系统配置模块](./modules/SYSTEM_CONFIG.md) - 键值对配置管理
- [节点性能监控模块](./modules/NODE_PERFORMANCE.md) - 性能指标收集与告警
- [节点自动化模块](./modules/NODE_AUTOMATION.md) - 自动扩缩容与故障转移

### API 文档
- [公告 API](./api/ANNOUNCEMENT_API.md)
- [系统配置 API](./api/SYSTEM_CONFIG_API.md)
- [节点性能 API](./api/NODE_PERFORMANCE_API.md)
- [节点自动化 API](./api/NODE_AUTOMATION_API.md)

### 开发指南
- [快速开始](./guides/QUICK_START.md) - 本地开发环境搭建
- [开发规范](./guides/DEVELOPMENT_GUIDE.md) - 代码规范与最佳实践
- [测试指南](./guides/TESTING_GUIDE.md) - 单元测试与集成测试
- [部署指南](./guides/DEPLOYMENT_GUIDE.md) - 生产环境部署

## 🏗️ 项目架构

```
NodePass-Pro Backend
├── Domain Layer (领域层)
│   ├── Entity (实体)
│   ├── Value Object (值对象)
│   ├── Repository Interface (仓储接口)
│   └── Domain Error (领域错误)
│
├── Application Layer (应用层)
│   ├── Commands (命令)
│   │   ├── Create
│   │   ├── Update
│   │   └── Delete
│   └── Queries (查询)
│       ├── Get
│       └── List
│
└── Infrastructure Layer (基础设施层)
    ├── Repository Implementation (仓储实现)
    ├── Cache Implementation (缓存实现)
    └── External Service (外部服务)
```

## 📊 已完成模块

### Phase 2: 核心业务模块 (4个) ✅
1. 节点组模块 - 节点分组管理
2. 节点实例模块 - 节点注册与心跳
3. 权益码模块 - VIP 权益码管理
4. 角色权限模块 - RBAC 权限控制

### Phase 3: 重要功能模块 (4个) ✅
1. 审计日志模块 - 操作审计记录
2. 告警通知模块 - 系统告警管理
3. 节点健康检查模块 - 节点健康监控
4. 隧道模板模块 - 隧道配置模板

### Phase 4: 辅助功能模块 (4个) ✅
1. 公告模块 - 系统公告发布
2. 系统配置模块 - 系统参数配置
3. 节点性能监控模块 - 性能指标监控
4. 节点自动化模块 - 自动化运维

## 🚀 技术栈

- **语言**: Go 1.21+
- **框架**: Gin
- **ORM**: GORM
- **数据库**: PostgreSQL
- **缓存**: Redis
- **架构**: DDD + CQRS
- **测试**: Testify

## 📈 性能指标

- **缓存命中率**: 80%+
- **API 响应时间**: < 50ms (P95)
- **心跳处理**: 1000+ req/s
- **数据库连接池**: 100 连接
- **Redis 连接池**: 50 连接

## 🔗 相关链接

- [项目完成报告](../PROJECT_COMPLETION_REPORT.md)
- [重构指南](../REFACTORING_GUIDE.md)
- [主 README](../README.md)
