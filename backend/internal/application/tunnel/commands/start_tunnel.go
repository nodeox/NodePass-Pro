package commands

import (
	"context"
	
	"nodepass-pro/backend/internal/domain/tunnel"
	"nodepass-pro/backend/internal/infrastructure/cache"
)

// StartTunnelCommand 启动隧道命令
type StartTunnelCommand struct {
	TunnelID uint
	UserID   uint
}

// StartTunnelResult 启动隧道结果
type StartTunnelResult struct {
	Success bool
}

// StartTunnelHandler 启动隧道处理器
type StartTunnelHandler struct {
	tunnelRepo  tunnel.Repository
	tunnelCache *cache.TunnelCache
}

// NewStartTunnelHandler 创建处理器
func NewStartTunnelHandler(
	tunnelRepo tunnel.Repository,
	tunnelCache *cache.TunnelCache,
) *StartTunnelHandler {
	return &StartTunnelHandler{
		tunnelRepo:  tunnelRepo,
		tunnelCache: tunnelCache,
	}
}

// Handle 处理命令
func (h *StartTunnelHandler) Handle(ctx context.Context, cmd StartTunnelCommand) (*StartTunnelResult, error) {
	// 1. 查询隧道
	t, err := h.tunnelRepo.FindByID(ctx, cmd.TunnelID)
	if err != nil {
		return nil, err
	}
	
	// 2. 验证权限
	if t.UserID != cmd.UserID {
		return nil, tunnel.ErrTunnelNotFound
	}
	
	// 3. 检查是否可以启动
	if !t.CanStart() {
		return nil, tunnel.ErrTunnelDisabled
	}
	
	// 4. 启动隧道
	if err := t.Start(); err != nil {
		return nil, err
	}
	
	// 5. 更新数据库
	if err := h.tunnelRepo.Update(ctx, t); err != nil {
		return nil, err
	}
	
	// 6. 更新缓存
	h.tunnelCache.SetRunning(ctx, t.ID, true)
	h.tunnelCache.Delete(ctx, t.ID) // 清除旧缓存
	
	return &StartTunnelResult{Success: true}, nil
}
