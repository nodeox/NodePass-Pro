package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/nodeautomation"
)

// IsolateNodeCommand 隔离节点命令
type IsolateNodeCommand struct {
	NodeInstanceID uint
	Reason         string
	IsolatedBy     string
}

// IsolateNodeHandler 隔离节点处理器
type IsolateNodeHandler struct {
	repo nodeautomation.Repository
}

// NewIsolateNodeHandler 创建处理器
func NewIsolateNodeHandler(repo nodeautomation.Repository) *IsolateNodeHandler {
	return &IsolateNodeHandler{repo: repo}
}

// Handle 处理命令
func (h *IsolateNodeHandler) Handle(ctx context.Context, cmd IsolateNodeCommand) error {
	// 检查是否已经隔离
	existing, err := h.repo.FindActiveIsolation(ctx, cmd.NodeInstanceID)
	if err == nil && existing != nil {
		return nil // 已经隔离
	}

	// 创建隔离记录
	isolation := nodeautomation.NewNodeIsolation(cmd.NodeInstanceID, cmd.Reason, cmd.IsolatedBy)

	if err := h.repo.CreateIsolation(ctx, isolation); err != nil {
		return fmt.Errorf("创建隔离记录失败: %w", err)
	}

	return nil
}
