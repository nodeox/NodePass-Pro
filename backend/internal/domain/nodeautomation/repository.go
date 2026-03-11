package nodeautomation

import "context"

// Repository 节点自动化仓储接口
type Repository interface {
	// CreatePolicy 创建策略
	CreatePolicy(ctx context.Context, policy *AutomationPolicy) error

	// FindPolicyByNodeGroup 根据节点组查找策略
	FindPolicyByNodeGroup(ctx context.Context, nodeGroupID uint) (*AutomationPolicy, error)

	// UpdatePolicy 更新策略
	UpdatePolicy(ctx context.Context, policy *AutomationPolicy) error

	// DeletePolicy 删除策略
	DeletePolicy(ctx context.Context, nodeGroupID uint) error

	// RecordAction 记录操作
	RecordAction(ctx context.Context, action *AutomationAction) error

	// FindActionsByNodeGroup 根据节点组查找操作记录
	FindActionsByNodeGroup(ctx context.Context, nodeGroupID uint, limit int) ([]*AutomationAction, error)

	// CreateIsolation 创建隔离记录
	CreateIsolation(ctx context.Context, isolation *NodeIsolation) error

	// FindActiveIsolation 查找活跃的隔离记录
	FindActiveIsolation(ctx context.Context, nodeInstanceID uint) (*NodeIsolation, error)

	// UpdateIsolation 更新隔离记录
	UpdateIsolation(ctx context.Context, isolation *NodeIsolation) error
}
