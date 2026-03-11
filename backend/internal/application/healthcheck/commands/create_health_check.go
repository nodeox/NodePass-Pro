package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/healthcheck"
)

// CreateHealthCheckCommand 创建健康检查配置命令
type CreateHealthCheckCommand struct {
	NodeInstanceID   uint
	Type             healthcheck.CheckType
	Interval         int
	Timeout          int
	Retries          int
	SuccessThreshold int
	FailureThreshold int
	HTTPPath         *string
	HTTPMethod       *string
	ExpectedStatus   *int
}

// CreateHealthCheckHandler 创建健康检查配置处理器
type CreateHealthCheckHandler struct {
	repo healthcheck.Repository
}

// NewCreateHealthCheckHandler 创建处理器
func NewCreateHealthCheckHandler(repo healthcheck.Repository) *CreateHealthCheckHandler {
	return &CreateHealthCheckHandler{repo: repo}
}

// Handle 处理命令
func (h *CreateHealthCheckHandler) Handle(ctx context.Context, cmd CreateHealthCheckCommand) (*healthcheck.HealthCheck, error) {
	// 检查是否已存在
	existing, err := h.repo.FindHealthCheckByNodeInstance(ctx, cmd.NodeInstanceID)
	if err == nil && existing != nil {
		return nil, healthcheck.ErrHealthCheckAlreadyExists
	}

	// 创建健康检查配置
	check := healthcheck.NewHealthCheck(cmd.NodeInstanceID, cmd.Type)

	// 更新配置
	if cmd.Interval > 0 || cmd.Timeout > 0 || cmd.Retries > 0 || cmd.SuccessThreshold > 0 || cmd.FailureThreshold > 0 {
		check.UpdateConfig(cmd.Interval, cmd.Timeout, cmd.Retries, cmd.SuccessThreshold, cmd.FailureThreshold)
	}

	// 设置 HTTP 配置
	check.HTTPPath = cmd.HTTPPath
	check.HTTPMethod = cmd.HTTPMethod
	check.ExpectedStatus = cmd.ExpectedStatus

	// 保存
	if err := h.repo.CreateHealthCheck(ctx, check); err != nil {
		return nil, fmt.Errorf("创建健康检查配置失败: %w", err)
	}

	return check, nil
}
