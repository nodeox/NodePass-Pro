# 节点自动化模块文档

## 概述

节点自动化模块提供自动化运维能力，包括自动扩缩容、自动故障转移、自动恢复和节点隔离等功能。

## 功能特性

- ✅ 自动扩缩容（Auto-Scaling）
- ✅ 自动故障转移（Auto-Failover）
- ✅ 自动恢复（Auto-Recovery）
- ✅ 节点隔离（Node Isolation）
- ✅ 操作记录与审计
- ✅ 策略配置管理

## 架构设计

### 领域层

```go
// AutomationPolicy 自动化策略实体
type AutomationPolicy struct {
    ID                    uint
    NodeGroupID           uint      // 节点组 ID
    Enabled               bool      // 是否启用
    AutoScalingEnabled    bool      // 自动扩缩容开关
    AutoFailoverEnabled   bool      // 自动故障转移开关
    AutoRecoveryEnabled   bool      // 自动恢复开关
    MinNodes              int       // 最小节点数
    MaxNodes              int       // 最大节点数
    ScaleUpThreshold      float64   // 扩容阈值（CPU 使用率）
    ScaleDownThreshold    float64   // 缩容阈值（CPU 使用率）
    ScaleCooldown         int       // 扩缩容冷却时间（秒）
    FailoverTimeout       int       // 故障转移超时（秒）
    RecoveryCheckInterval int       // 恢复检查间隔（秒）
    CreatedAt             time.Time
    UpdatedAt             time.Time
}

// AutomationAction 自动化操作记录
type AutomationAction struct {
    ID             uint
    NodeGroupID    uint
    ActionType     ActionType    // 操作类型
    TargetNodeID   *uint         // 目标节点 ID
    Status         ActionStatus  // 操作状态
    Reason         string        // 操作原因
    Details        *string       // 详细信息
    ExecutedAt     time.Time     // 执行时间
    CompletedAt    *time.Time    // 完成时间
}

// NodeIsolation 节点隔离记录
type NodeIsolation struct {
    ID             uint
    NodeInstanceID uint
    Reason         string
    IsolatedBy     string        // 隔离发起者
    IsolatedAt     time.Time
    RecoveredAt    *time.Time
    IsActive       bool
}
```

### 枚举类型

```go
// ActionType 操作类型
type ActionType string

const (
    ActionTypeScaleUp      ActionType = "scale_up"       // 扩容
    ActionTypeScaleDown    ActionType = "scale_down"     // 缩容
    ActionTypeFailover     ActionType = "failover"       // 故障转移
    ActionTypeRecover      ActionType = "recover"        // 恢复
    ActionTypeIsolate      ActionType = "isolate"        // 隔离
)

// ActionStatus 操作状态
type ActionStatus string

const (
    ActionStatusPending    ActionStatus = "pending"      // 待执行
    ActionStatusExecuting  ActionStatus = "executing"    // 执行中
    ActionStatusCompleted  ActionStatus = "completed"    // 已完成
    ActionStatusFailed     ActionStatus = "failed"       // 失败
)
```

### 业务规则

#### 1. 扩缩容规则

```go
func (p *AutomationPolicy) ShouldScaleUp(avgCPU float64, currentNodes int) bool {
    if !p.Enabled || !p.AutoScalingEnabled {
        return false
    }

    // CPU 使用率超过阈值且未达到最大节点数
    return avgCPU > p.ScaleUpThreshold && currentNodes < p.MaxNodes
}

func (p *AutomationPolicy) ShouldScaleDown(avgCPU float64, currentNodes int) bool {
    if !p.Enabled || !p.AutoScalingEnabled {
        return false
    }

    // CPU 使用率低于阈值且超过最小节点数
    return avgCPU < p.ScaleDownThreshold && currentNodes > p.MinNodes
}
```

#### 2. 故障转移规则

```go
func (p *AutomationPolicy) ShouldFailover(nodeDownTime time.Duration) bool {
    if !p.Enabled || !p.AutoFailoverEnabled {
        return false
    }

    // 节点宕机时间超过故障转移超时
    return nodeDownTime > time.Duration(p.FailoverTimeout)*time.Second
}
```

#### 3. 自动恢复规则

```go
func (p *AutomationPolicy) ShouldRecover(isolation *NodeIsolation) bool {
    if !p.Enabled || !p.AutoRecoveryEnabled {
        return false
    }

    // 隔离时间超过恢复检查间隔
    isolationDuration := time.Since(isolation.IsolatedAt)
    return isolationDuration > time.Duration(p.RecoveryCheckInterval)*time.Second
}
```

### 应用层

#### Commands (命令)

**CreatePolicyCommand** - 创建自动化策略
```go
type CreatePolicyCommand struct {
    NodeGroupID           uint
    AutoScalingEnabled    bool
    AutoFailoverEnabled   bool
    AutoRecoveryEnabled   bool
    MinNodes              int
    MaxNodes              int
    ScaleUpThreshold      float64
    ScaleDownThreshold    float64
    ScaleCooldown         int
    FailoverTimeout       int
    RecoveryCheckInterval int
}
```

**UpdatePolicyCommand** - 更新自动化策略
```go
type UpdatePolicyCommand struct {
    NodeGroupID           uint
    Enabled               *bool
    AutoScalingEnabled    *bool
    AutoFailoverEnabled   *bool
    AutoRecoveryEnabled   *bool
    MinNodes              *int
    MaxNodes              *int
    ScaleUpThreshold      *float64
    ScaleDownThreshold    *float64
    ScaleCooldown         *int
    FailoverTimeout       *int
    RecoveryCheckInterval *int
}
```

**IsolateNodeCommand** - 隔离节点
```go
type IsolateNodeCommand struct {
    NodeInstanceID uint
    Reason         string
    IsolatedBy     string
}
```

**RecoverNodeCommand** - 恢复节点
```go
type RecoverNodeCommand struct {
    NodeInstanceID uint
}
```

#### Queries (查询)

**GetPolicyQuery** - 获取策略
```go
type GetPolicyQuery struct {
    NodeGroupID uint
}
```

**ListActionsQuery** - 列出操作记录
```go
type ListActionsQuery struct {
    NodeGroupID uint
    ActionType  *ActionType
    Status      *ActionStatus
    Limit       int
}
```

**GetIsolationQuery** - 获取隔离记录
```go
type GetIsolationQuery struct {
    NodeInstanceID uint
}
```

### 基础设施层

#### PostgreSQL 仓储

```go
type AutomationRepository struct {
    db *gorm.DB
}

// CreatePolicy 创建策略
func (r *AutomationRepository) CreatePolicy(ctx context.Context, policy *AutomationPolicy) error {
    return r.db.WithContext(ctx).Create(policy).Error
}

// FindPolicyByNodeGroup 根据节点组查找策略
func (r *AutomationRepository) FindPolicyByNodeGroup(ctx context.Context, nodeGroupID uint) (*AutomationPolicy, error) {
    var policy AutomationPolicy
    err := r.db.WithContext(ctx).
        Where("node_group_id = ?", nodeGroupID).
        First(&policy).Error

    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrPolicyNotFound
        }
        return nil, err
    }

    return &policy, nil
}

// RecordAction 记录操作
func (r *AutomationRepository) RecordAction(ctx context.Context, action *AutomationAction) error {
    return r.db.WithContext(ctx).Create(action).Error
}

// CreateIsolation 创建隔离记录
func (r *AutomationRepository) CreateIsolation(ctx context.Context, isolation *NodeIsolation) error {
    return r.db.WithContext(ctx).Create(isolation).Error
}

// FindActiveIsolation 查找活跃的隔离记录
func (r *AutomationRepository) FindActiveIsolation(ctx context.Context, nodeInstanceID uint) (*NodeIsolation, error) {
    var isolation NodeIsolation
    err := r.db.WithContext(ctx).
        Where("node_instance_id = ?", nodeInstanceID).
        Where("is_active = ?", true).
        First(&isolation).Error

    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, err
    }

    return &isolation, nil
}
```

## 使用示例

### 1. 创建自动化策略

```go
// 为节点组创建自动化策略
cmd := commands.CreatePolicyCommand{
    NodeGroupID:           1,
    AutoScalingEnabled:    true,
    AutoFailoverEnabled:   true,
    AutoRecoveryEnabled:   true,
    MinNodes:              2,      // 最少 2 个节点
    MaxNodes:              10,     // 最多 10 个节点
    ScaleUpThreshold:      80.0,   // CPU > 80% 扩容
    ScaleDownThreshold:    30.0,   // CPU < 30% 缩容
    ScaleCooldown:         300,    // 5 分钟冷却
    FailoverTimeout:       60,     // 60 秒故障转移
    RecoveryCheckInterval: 300,    // 5 分钟检查恢复
}

handler := commands.NewCreatePolicyHandler(repo)
policy, err := handler.Handle(ctx, cmd)
```

### 2. 更新策略

```go
// 禁用自动扩缩容
enabled := false
cmd := commands.UpdatePolicyCommand{
    NodeGroupID:        1,
    AutoScalingEnabled: &enabled,
}

handler := commands.NewUpdatePolicyHandler(repo)
err := handler.Handle(ctx, cmd)
```

### 3. 隔离节点

```go
// 手动隔离故障节点
cmd := commands.IsolateNodeCommand{
    NodeInstanceID: 5,
    Reason:         "节点响应超时",
    IsolatedBy:     "admin",
}

handler := commands.NewIsolateNodeHandler(repo)
err := handler.Handle(ctx, cmd)
```

### 4. 恢复节点

```go
// 恢复隔离的节点
cmd := commands.RecoverNodeCommand{
    NodeInstanceID: 5,
}

handler := commands.NewRecoverNodeHandler(repo)
err := handler.Handle(ctx, cmd)
```

### 5. 查询操作记录

```go
// 查询最近的扩缩容操作
actionType := ActionTypeScaleUp
query := queries.ListActionsQuery{
    NodeGroupID: 1,
    ActionType:  &actionType,
    Limit:       10,
}

handler := queries.NewListActionsHandler(repo)
actions, err := handler.Handle(ctx, query)
```

## API 接口

详见 [节点自动化 API 文档](../api/NODE_AUTOMATION_API.md)

## 数据库表结构

```sql
-- 自动化策略表
CREATE TABLE automation_policies (
    id SERIAL PRIMARY KEY,
    node_group_id INT NOT NULL UNIQUE,
    enabled BOOLEAN DEFAULT true,
    auto_scaling_enabled BOOLEAN DEFAULT false,
    auto_failover_enabled BOOLEAN DEFAULT false,
    auto_recovery_enabled BOOLEAN DEFAULT false,
    min_nodes INT DEFAULT 1,
    max_nodes INT DEFAULT 10,
    scale_up_threshold DECIMAL(5,2) DEFAULT 80.0,
    scale_down_threshold DECIMAL(5,2) DEFAULT 30.0,
    scale_cooldown INT DEFAULT 300,
    failover_timeout INT DEFAULT 60,
    recovery_check_interval INT DEFAULT 300,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 自动化操作记录表
CREATE TABLE automation_actions (
    id BIGSERIAL PRIMARY KEY,
    node_group_id INT NOT NULL,
    action_type VARCHAR(20) NOT NULL,
    target_node_id INT,
    status VARCHAR(20) NOT NULL,
    reason TEXT NOT NULL,
    details TEXT,
    executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- 节点隔离记录表
CREATE TABLE node_isolations (
    id SERIAL PRIMARY KEY,
    node_instance_id INT NOT NULL,
    reason TEXT NOT NULL,
    isolated_by VARCHAR(100) NOT NULL,
    isolated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    recovered_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

-- 索引
CREATE UNIQUE INDEX idx_automation_policies_group ON automation_policies(node_group_id);
CREATE INDEX idx_automation_actions_group ON automation_actions(node_group_id);
CREATE INDEX idx_automation_actions_type ON automation_actions(action_type);
CREATE INDEX idx_automation_actions_time ON automation_actions(executed_at DESC);
CREATE INDEX idx_node_isolations_node ON node_isolations(node_instance_id);
CREATE INDEX idx_node_isolations_active ON node_isolations(is_active);
```

## 自动化流程

### 1. 自动扩容流程

```
1. 监控节点组平均 CPU 使用率
2. 如果 CPU > 扩容阈值 且 当前节点数 < 最大节点数
3. 检查冷却时间（距离上次扩缩容是否超过冷却时间）
4. 创建新节点
5. 记录扩容操作
6. 更新最后操作时间
```

### 2. 自动缩容流程

```
1. 监控节点组平均 CPU 使用率
2. 如果 CPU < 缩容阈值 且 当前节点数 > 最小节点数
3. 检查冷却时间
4. 选择负载最低的节点
5. 迁移流量到其他节点
6. 删除节点
7. 记录缩容操作
```

### 3. 自动故障转移流程

```
1. 检测节点心跳超时
2. 如果超时时间 > 故障转移超时
3. 标记节点为故障状态
4. 将流量切换到健康节点
5. 隔离故障节点
6. 记录故障转移操作
7. 发送告警通知
```

### 4. 自动恢复流程

```
1. 定期检查隔离节点
2. 如果隔离时间 > 恢复检查间隔
3. 执行健康检查
4. 如果健康检查通过
5. 恢复节点到正常状态
6. 记录恢复操作
```

## 定时任务

### 1. 扩缩容检查任务

```go
// 每分钟执行一次
func (s *AutomationService) CheckAutoScaling(ctx context.Context) error {
    // 获取所有启用自动扩缩容的节点组
    policies, err := s.repo.FindEnabledPolicies(ctx)
    if err != nil {
        return err
    }

    for _, policy := range policies {
        if !policy.AutoScalingEnabled {
            continue
        }

        // 获取节点组的平均 CPU 使用率
        avgCPU, err := s.getNodeGroupAvgCPU(ctx, policy.NodeGroupID)
        if err != nil {
            continue
        }

        // 获取当前节点数
        currentNodes, err := s.getNodeGroupNodeCount(ctx, policy.NodeGroupID)
        if err != nil {
            continue
        }

        // 检查是否需要扩容
        if policy.ShouldScaleUp(avgCPU, currentNodes) {
            s.scaleUp(ctx, policy)
        }

        // 检查是否需要缩容
        if policy.ShouldScaleDown(avgCPU, currentNodes) {
            s.scaleDown(ctx, policy)
        }
    }

    return nil
}
```

### 2. 故障转移检查任务

```go
// 每 30 秒执行一次
func (s *AutomationService) CheckAutoFailover(ctx context.Context) error {
    // 获取所有启用自动故障转移的节点组
    policies, err := s.repo.FindEnabledPolicies(ctx)
    if err != nil {
        return err
    }

    for _, policy := range policies {
        if !policy.AutoFailoverEnabled {
            continue
        }

        // 获取节点组中的故障节点
        failedNodes, err := s.getFailedNodes(ctx, policy.NodeGroupID)
        if err != nil {
            continue
        }

        for _, node := range failedNodes {
            downTime := time.Since(node.LastHeartbeatAt)

            // 检查是否需要故障转移
            if policy.ShouldFailover(downTime) {
                s.failover(ctx, policy, node)
            }
        }
    }

    return nil
}
```

### 3. 自动恢复检查任务

```go
// 每 5 分钟执行一次
func (s *AutomationService) CheckAutoRecovery(ctx context.Context) error {
    // 获取所有启用自动恢复的节点组
    policies, err := s.repo.FindEnabledPolicies(ctx)
    if err != nil {
        return err
    }

    for _, policy := range policies {
        if !policy.AutoRecoveryEnabled {
            continue
        }

        // 获取隔离的节点
        isolations, err := s.repo.FindActiveIsolations(ctx, policy.NodeGroupID)
        if err != nil {
            continue
        }

        for _, isolation := range isolations {
            // 检查是否需要恢复
            if policy.ShouldRecover(isolation) {
                s.recover(ctx, policy, isolation)
            }
        }
    }

    return nil
}
```

## 最佳实践

### 1. 扩缩容策略

- **扩容阈值**: 建议设置为 70-80%
- **缩容阈值**: 建议设置为 20-30%
- **冷却时间**: 建议设置为 5-10 分钟
- **最小节点数**: 至少保留 2 个节点保证高可用
- **最大节点数**: 根据业务需求和成本设置

### 2. 故障转移策略

- **超时时间**: 建议设置为 60-120 秒
- **健康检查**: 结合心跳和主动探测
- **流量切换**: 使用渐进式切换，避免流量突增

### 3. 自动恢复策略

- **检查间隔**: 建议设置为 5-10 分钟
- **健康检查**: 多次检查确认节点恢复
- **恢复策略**: 先恢复到隔离状态，观察一段时间后完全恢复

## 监控与告警

### 1. 关键指标

- 扩缩容操作次数
- 故障转移次数
- 节点隔离数量
- 自动恢复成功率
- 操作执行时间

### 2. 告警规则

- 频繁扩缩容（1 小时内超过 5 次）
- 故障转移失败
- 节点长时间隔离（超过 1 小时）
- 达到最大/最小节点数限制

## 测试

```go
func TestAutomationPolicy_ShouldScaleUp(t *testing.T) {
    policy := &AutomationPolicy{
        Enabled:            true,
        AutoScalingEnabled: true,
        MaxNodes:           10,
        ScaleUpThreshold:   80.0,
    }

    tests := []struct {
        name         string
        avgCPU       float64
        currentNodes int
        want         bool
    }{
        {
            name:         "CPU 超过阈值且未达到最大节点数",
            avgCPU:       85.0,
            currentNodes: 5,
            want:         true,
        },
        {
            name:         "CPU 未超过阈值",
            avgCPU:       75.0,
            currentNodes: 5,
            want:         false,
        },
        {
            name:         "已达到最大节点数",
            avgCPU:       85.0,
            currentNodes: 10,
            want:         false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := policy.ShouldScaleUp(tt.avgCPU, tt.currentNodes)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

## 相关文档

- [节点自动化 API 文档](../api/NODE_AUTOMATION_API.md)
- [节点性能监控模块](./NODE_PERFORMANCE.md)
- [DDD 架构文档](../architecture/DDD_ARCHITECTURE.md)
