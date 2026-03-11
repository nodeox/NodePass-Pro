# NodePass-Pro 后端重构路线图 v2.0

## 📊 当前状态

**真实进度**: 25% (6/24 模块)
**已完成**: User, Node, Tunnel, Traffic, Auth(70%), VIP(60%)
**待重构**: 18 个模块

---

## 🎯 重构策略

### 原则
1. **优先核心业务** - 先重构高频使用的核心模块
2. **渐进式迁移** - 新旧代码并存，逐步切换
3. **完整闭环** - 每个模块包含 Domain + Application + Infrastructure + Cache + Tests
4. **清理旧代码** - 重构完成后立即清理旧代码

### 分阶段实施
- **Phase 1**: 完善已重构模块 (1 周)
- **Phase 2**: 核心业务模块 (3 周)
- **Phase 3**: 重要功能模块 (2 周)
- **Phase 4**: 辅助功能模块 (2 周)

---

## Phase 1: 完善已重构模块 (Week 1)

### 目标
完善 Auth 和 VIP 模块，清理旧代码，提升测试覆盖率

### Task 1.1: Auth 模块完善 (2 天)
**优先级**: P0

#### 待完成
- [ ] 创建 `internal/infrastructure/cache/auth_cache.go`
  - RefreshToken 缓存
  - 用户会话缓存
  - 登录失败计数器
- [ ] 更新 Application 层使用缓存
- [ ] 编写单元测试
- [ ] 集成到容器

#### 产出
- `internal/infrastructure/cache/auth_cache.go`
- `internal/infrastructure/cache/auth_cache_test.go`
- 更新 `internal/infrastructure/container/container.go`

---

### Task 1.2: VIP 模块完善 (2 天)
**优先级**: P0

#### 待完成
- [ ] 创建 `internal/infrastructure/cache/vip_cache.go`
  - VIP 等级缓存
  - 用户 VIP 状态缓存
- [ ] 补充 Application 层命令
  - 兑换权益码升级 VIP
  - 续费 VIP
- [ ] 编写单元测试
- [ ] 集成到容器

#### 产出
- `internal/infrastructure/cache/vip_cache.go`
- `internal/application/vip/commands/redeem_benefit_code.go`
- `internal/application/vip/commands/renew_vip.go`
- 测试文件

---

### Task 1.3: 清理旧代码 (1 天)
**优先级**: P1

#### 待清理
- [ ] 标记旧代码为 Deprecated
- [ ] 添加迁移指南注释
- [ ] 确保新旧路由并存

#### 文件
- `internal/models/user.go` - 添加 @Deprecated
- `internal/models/node.go` - 添加 @Deprecated
- `internal/services/*` - 添加迁移注释

---

### Task 1.4: 提升测试覆盖率 (2 天)
**优先级**: P1

#### 目标
从 50% 提升到 70%

#### 待完成
- [ ] Tunnel Repository 测试
- [ ] Traffic Repository 测试
- [ ] Auth Commands 测试
- [ ] VIP Commands 测试

---

## Phase 2: 核心业务模块 (Week 2-4)

### Task 2.1: 节点组模块 (3 天)
**优先级**: P0
**依赖**: Node 模块

#### 领域层
```
internal/domain/nodegroup/
├── entity.go          # NodeGroup, NodeGroupConfig
├── repository.go      # NodeGroupRepository
├── value_objects.go   # NodeGroupType, LoadBalanceStrategy
└── errors.go
```

#### 应用层
```
internal/application/nodegroup/
├── commands/
│   ├── create_group.go
│   ├── update_group.go
│   ├── delete_group.go
│   └── enable_group.go
└── queries/
    ├── get_group.go
    ├── list_groups.go
    └── get_group_stats.go
```

#### 基础设施层
```
internal/infrastructure/
├── persistence/postgres/nodegroup/
│   └── nodegroup_repository.go
└── cache/
    └── nodegroup_cache.go
```

#### 测试
- [ ] Domain 实体测试
- [ ] Repository 测试
- [ ] Commands 测试
- [ ] Cache 测试

---

### Task 2.2: 节点实例模块 (3 天)
**优先级**: P0
**依赖**: NodeGroup 模块

#### 领域层
```
internal/domain/nodeinstance/
├── entity.go          # NodeInstance, SystemInfo
├── repository.go      # NodeInstanceRepository
├── value_objects.go   # NodeInstanceStatus
└── errors.go
```

#### 应用层
```
internal/application/nodeinstance/
├── commands/
│   ├── register_instance.go
│   ├── update_instance.go
│   ├── update_status.go
│   └── update_config.go
└── queries/
    ├── get_instance.go
    ├── list_instances.go
    └── get_online_instances.go
```

#### 基础设施层
```
internal/infrastructure/
├── persistence/postgres/nodeinstance/
│   └── nodeinstance_repository.go
└── cache/
    └── nodeinstance_cache.go
```

---

### Task 2.3: 权益码模块 (2 天)
**优先级**: P0
**依赖**: VIP 模块

#### 领域层
```
internal/domain/benefitcode/
├── entity.go          # BenefitCode
├── repository.go      # BenefitCodeRepository
├── value_objects.go   # CodeType, CodeStatus
└── errors.go
```

#### 应用层
```
internal/application/benefitcode/
├── commands/
│   ├── generate_code.go
│   ├── redeem_code.go
│   └── revoke_code.go
└── queries/
    ├── get_code.go
    ├── list_codes.go
    └── validate_code.go
```

#### 基础设施层
```
internal/infrastructure/
├── persistence/postgres/benefitcode/
│   └── benefitcode_repository.go
└── cache/
    └── benefitcode_cache.go  # 防重放攻击
```

---

### Task 2.4: 角色权限模块 (3 天)
**优先级**: P0

#### 领域层
```
internal/domain/rbac/
├── entity.go          # Role, Permission
├── repository.go      # RoleRepository, PermissionRepository
├── value_objects.go   # PermissionAction, Resource
└── errors.go
```

#### 应用层
```
internal/application/rbac/
├── commands/
│   ├── create_role.go
│   ├── assign_permission.go
│   ├── assign_role_to_user.go
│   └── revoke_permission.go
└── queries/
    ├── get_role.go
    ├── list_roles.go
    ├── get_user_permissions.go
    └── check_permission.go
```

#### 基础设施层
```
internal/infrastructure/
├── persistence/postgres/rbac/
│   ├── role_repository.go
│   └── permission_repository.go
└── cache/
    └── rbac_cache.go  # 权限缓存
```

---

## Phase 3: 重要功能模块 (Week 5-6)

### Task 3.1: 审计日志模块 (2 天)
**优先级**: P1

#### 领域层
```
internal/domain/audit/
├── entity.go          # AuditLog
├── repository.go      # AuditLogRepository
├── value_objects.go   # Action, Resource, Result
└── errors.go
```

#### 应用层
```
internal/application/audit/
├── commands/
│   └── record_audit.go
└── queries/
    ├── get_audit_log.go
    ├── list_audit_logs.go
    └── search_audit_logs.go
```

#### 基础设施层
```
internal/infrastructure/
├── persistence/postgres/audit/
│   └── audit_repository.go
└── cache/
    └── audit_buffer.go  # 批量写入缓冲
```

---

### Task 3.2: 告警模块 (3 天)
**优先级**: P1

#### 拆分为 3 个子模块

##### 3.2.1 Alert (告警记录)
```
internal/domain/alert/
├── entity.go          # Alert
├── repository.go
└── value_objects.go   # AlertLevel, AlertStatus, AlertType
```

##### 3.2.2 AlertRule (告警规则)
```
internal/domain/alertrule/
├── entity.go          # AlertRule
├── repository.go
└── value_objects.go
```

##### 3.2.3 NotificationChannel (通知渠道)
```
internal/domain/notification/
├── entity.go          # NotificationChannel
├── repository.go
└── value_objects.go   # ChannelType
```

---

### Task 3.3: 节点健康检查模块 (2 天)
**优先级**: P1

#### 领域层
```
internal/domain/nodehealth/
├── entity.go          # NodeHealthCheck, NodeHealthRecord, NodeQualityScore
├── repository.go
└── value_objects.go   # HealthCheckType, HealthCheckStatus
```

#### 应用层
```
internal/application/nodehealth/
├── commands/
│   ├── create_health_check.go
│   ├── record_health_result.go
│   └── update_quality_score.go
└── queries/
    ├── get_health_status.go
    └── get_quality_score.go
```

---

### Task 3.4: 隧道模板模块 (2 天)
**优先级**: P1

#### 领域层
```
internal/domain/tunneltemplate/
├── entity.go          # TunnelTemplate, TunnelTemplateConfig
├── repository.go
└── errors.go
```

#### 应用层
```
internal/application/tunneltemplate/
├── commands/
│   ├── create_template.go
│   ├── update_template.go
│   ├── delete_template.go
│   └── apply_template.go
└── queries/
    ├── get_template.go
    ├── list_templates.go
    └── list_public_templates.go
```

---

## Phase 4: 辅助功能模块 (Week 7-8)

### Task 4.1: 公告模块 (1 天)
**优先级**: P2

```
internal/domain/announcement/
internal/application/announcement/
internal/infrastructure/persistence/postgres/announcement/
internal/infrastructure/cache/announcement_cache.go
```

---

### Task 4.2: 系统设置模块 (1 天)
**优先级**: P2

```
internal/domain/systemconfig/
internal/application/systemconfig/
internal/infrastructure/persistence/postgres/systemconfig/
internal/infrastructure/cache/systemconfig_cache.go
```

---

### Task 4.3: 节点性能监控模块 (2 天)
**优先级**: P2

```
internal/domain/nodeperformance/
├── entity.go  # NodePerformanceMetric, NodePerformanceAlert, NodePerformanceSummary
├── repository.go
└── value_objects.go
```

---

### Task 4.4: 节点自动化模块 (2 天)
**优先级**: P2

```
internal/domain/nodeautomation/
├── entity.go  # NodeAutomationPolicy, NodeAutomationAction, NodeIsolation
├── repository.go
└── value_objects.go
```

---

## 📋 每个模块的标准交付物

### 必须完成
1. ✅ Domain 层（实体、仓储接口、值对象、错误）
2. ✅ Application 层（Commands、Queries）
3. ✅ Infrastructure 层（Repository 实现、Cache 实现）
4. ✅ 单元测试（覆盖率 > 70%）
5. ✅ 集成到 Container
6. ✅ 更新文档

### 可选
- 集成测试
- 性能测试
- API 文档

---

## 🎯 里程碑

### Milestone 1: 完善已重构模块 (Week 1 结束)
- Auth 模块 100%
- VIP 模块 100%
- 测试覆盖率 70%

### Milestone 2: 核心业务完成 (Week 4 结束)
- NodeGroup 模块 100%
- NodeInstance 模块 100%
- BenefitCode 模块 100%
- RBAC 模块 100%
- 核心业务重构进度 80%

### Milestone 3: 重要功能完成 (Week 6 结束)
- Audit 模块 100%
- Alert 模块 100%
- NodeHealth 模块 100%
- TunnelTemplate 模块 100%
- 整体重构进度 70%

### Milestone 4: 全部完成 (Week 8 结束)
- 所有模块 100%
- 测试覆盖率 80%
- 旧代码清理完成
- 整体重构进度 100%

---

## 📊 进度跟踪

### Week 1 (当前周)
- [ ] Task 1.1: Auth 模块完善
- [ ] Task 1.2: VIP 模块完善
- [ ] Task 1.3: 清理旧代码
- [ ] Task 1.4: 提升测试覆盖率

### Week 2
- [ ] Task 2.1: 节点组模块

### Week 3
- [ ] Task 2.2: 节点实例模块
- [ ] Task 2.3: 权益码模块

### Week 4
- [ ] Task 2.4: 角色权限模块

### Week 5
- [ ] Task 3.1: 审计日志模块
- [ ] Task 3.2: 告警模块

### Week 6
- [ ] Task 3.3: 节点健康检查模块
- [ ] Task 3.4: 隧道模板模块

### Week 7-8
- [ ] Task 4.1-4.4: 辅助功能模块

---

## 🚀 开始执行

**当前任务**: Task 1.1 - Auth 模块完善
**预计完成**: 2 天
**负责人**: AI Assistant

---

**创建时间**: 2026-03-11
**版本**: v2.0
**状态**: 执行中
