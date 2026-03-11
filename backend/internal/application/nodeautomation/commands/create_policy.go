package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/nodeautomation"
)

// CreatePolicyCommand 创建策略命令
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

// CreatePolicyHandler 创建策略处理器
type CreatePolicyHandler struct {
	repo nodeautomation.Repository
}

// NewCreatePolicyHandler 创建处理器
func NewCreatePolicyHandler(repo nodeautomation.Repository) *CreatePolicyHandler {
	return &CreatePolicyHandler{repo: repo}
}

// Handle 处理命令
func (h *CreatePolicyHandler) Handle(ctx context.Context, cmd CreatePolicyCommand) (*nodeautomation.AutomationPolicy, error) {
	policy := nodeautomation.NewAutomationPolicy(cmd.NodeGroupID)

	// 设置配置
	policy.AutoScalingEnabled = cmd.AutoScalingEnabled
	policy.AutoFailoverEnabled = cmd.AutoFailoverEnabled
	policy.AutoRecoveryEnabled = cmd.AutoRecoveryEnabled

	if cmd.MinNodes > 0 {
		policy.MinNodes = cmd.MinNodes
	}
	if cmd.MaxNodes > 0 {
		policy.MaxNodes = cmd.MaxNodes
	}
	if cmd.ScaleUpThreshold > 0 {
		policy.ScaleUpThreshold = cmd.ScaleUpThreshold
	}
	if cmd.ScaleDownThreshold > 0 {
		policy.ScaleDownThreshold = cmd.ScaleDownThreshold
	}
	if cmd.ScaleCooldown > 0 {
		policy.ScaleCooldown = cmd.ScaleCooldown
	}
	if cmd.FailoverTimeout > 0 {
		policy.FailoverTimeout = cmd.FailoverTimeout
	}
	if cmd.RecoveryCheckInterval > 0 {
		policy.RecoveryCheckInterval = cmd.RecoveryCheckInterval
	}

	if err := h.repo.CreatePolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("创建自动化策略失败: %w", err)
	}

	return policy, nil
}
