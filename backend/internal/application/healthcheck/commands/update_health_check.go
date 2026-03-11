package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/healthcheck"
)

// UpdateHealthCheckCommand 更新健康检查配置命令
type UpdateHealthCheckCommand struct {
	NodeInstanceID   uint
	Interval         *int
	Timeout          *int
	Retries          *int
	SuccessThreshold *int
	FailureThreshold *int
	Enabled          *bool
	HTTPPath         *string
	HTTPMethod       *string
	ExpectedStatus   *int
}

// UpdateHealthCheckHandler 更新健康检查配置处理器
type UpdateHealthCheckHandler struct {
	repo healthcheck.Repository
}

// NewUpdateHealthCheckHandler 创建处理器
func NewUpdateHealthCheckHandler(repo healthcheck.Repository) *UpdateHealthCheckHandler {
	return &UpdateHealthCheckHandler{repo: repo}
}

// Handle 处理命令
func (h *UpdateHealthCheckHandler) Handle(ctx context.Context, cmd UpdateHealthCheckCommand) error {
	// 查找健康检查配置
	check, err := h.repo.FindHealthCheckByNodeInstance(ctx, cmd.NodeInstanceID)
	if err != nil {
		return err
	}

	// 更新配置
	if cmd.Interval != nil || cmd.Timeout != nil || cmd.Retries != nil || cmd.SuccessThreshold != nil || cmd.FailureThreshold != nil {
		interval := check.Interval
		timeout := check.Timeout
		retries := check.Retries
		successThreshold := check.SuccessThreshold
		failureThreshold := check.FailureThreshold

		if cmd.Interval != nil {
			interval = *cmd.Interval
		}
		if cmd.Timeout != nil {
			timeout = *cmd.Timeout
		}
		if cmd.Retries != nil {
			retries = *cmd.Retries
		}
		if cmd.SuccessThreshold != nil {
			successThreshold = *cmd.SuccessThreshold
		}
		if cmd.FailureThreshold != nil {
			failureThreshold = *cmd.FailureThreshold
		}

		check.UpdateConfig(interval, timeout, retries, successThreshold, failureThreshold)
	}

	// 更新启用状态
	if cmd.Enabled != nil {
		if *cmd.Enabled {
			check.Enable()
		} else {
			check.Disable()
		}
	}

	// 更新 HTTP 配置
	if cmd.HTTPPath != nil {
		check.HTTPPath = cmd.HTTPPath
	}
	if cmd.HTTPMethod != nil {
		check.HTTPMethod = cmd.HTTPMethod
	}
	if cmd.ExpectedStatus != nil {
		check.ExpectedStatus = cmd.ExpectedStatus
	}

	// 保存
	if err := h.repo.UpdateHealthCheck(ctx, check); err != nil {
		return fmt.Errorf("更新健康检查配置失败: %w", err)
	}

	return nil
}
