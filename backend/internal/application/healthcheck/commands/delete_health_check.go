package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/healthcheck"
)

// DeleteHealthCheckCommand 删除健康检查配置命令
type DeleteHealthCheckCommand struct {
	NodeInstanceID uint
}

// DeleteHealthCheckHandler 删除健康检查配置处理器
type DeleteHealthCheckHandler struct {
	repo healthcheck.Repository
}

// NewDeleteHealthCheckHandler 创建处理器
func NewDeleteHealthCheckHandler(repo healthcheck.Repository) *DeleteHealthCheckHandler {
	return &DeleteHealthCheckHandler{repo: repo}
}

// Handle 处理命令
func (h *DeleteHealthCheckHandler) Handle(ctx context.Context, cmd DeleteHealthCheckCommand) error {
	// 查找健康检查配置
	check, err := h.repo.FindHealthCheckByNodeInstance(ctx, cmd.NodeInstanceID)
	if err != nil {
		return err
	}

	// 删除
	if err := h.repo.DeleteHealthCheck(ctx, check.ID); err != nil {
		return fmt.Errorf("删除健康检查配置失败: %w", err)
	}

	return nil
}
