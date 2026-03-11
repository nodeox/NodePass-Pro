package commands

import (
	"context"
	
	"nodepass-pro/backend/internal/domain/tunnel"
	"nodepass-pro/backend/internal/infrastructure/cache"
)

// StopTunnelCommand 停止隧道命令
type StopTunnelCommand struct {
	TunnelID uint
	UserID   uint
}

// StopTunnelResult 停止隧道结果
type StopTunnelResult struct {
	Success bool
}

// StopTunnelHandler 停止隧道处理器
type StopTunnelHandler struct {
	tunnelRepo  tunnel.Repository
	tunnelCache *cache.TunnelCache
}

// NewStopTunnelHandler 创建处理器
func NewStopTunnelHandler(
	tunnelRepo tunnel.Repository,
	tunnelCache *cache.TunnelCache,
) *StopTunnelHandler {
	return &StopTunnelHandler{
		tunnelRepo:  tunnelRepo,
		tunnelCache: tunnelCache,
	}
}

// Handle 处理命令
func (h *StopTunnelHandler) Handle(ctx context.Context, cmd StopTunnelCommand) (*StopTunnelResult, error) {
	// 1. 查询隧道
	t, err := h.tunnelRepo.FindByID(ctx, cmd.TunnelID)
	if err != nil {
		return nil, err
	}
	
	// 2. 验证权限
	if t.UserID != cmd.UserID {
		return nil, tunnel.ErrTunnelNotFound
	}
	
	// 3. 停止隧道
	t.Stop()
	
	// 4. 更新数据库
	if err := h.tunnelRepo.Update(ctx, t); err != nil {
		return nil, err
	}
	
	// 5. 更新缓存
	h.tunnelCache.SetRunning(ctx, t.ID, false)
	h.tunnelCache.Delete(ctx, t.ID) // 清除旧缓存
	
	return &StopTunnelResult{Success: true}, nil
}
