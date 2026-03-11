package commands

import (
	"context"
	
	"nodepass-pro/backend/internal/domain/tunnel"
	"nodepass-pro/backend/internal/domain/user"
	"nodepass-pro/backend/internal/infrastructure/cache"
)

// CreateTunnelCommand 创建隧道命令
type CreateTunnelCommand struct {
	UserID      uint
	Name        string
	Description string
	Protocol    string
	Mode        string
	ListenHost  string
	ListenPort  int
	TargetHost  string
	TargetPort  int
	EntryNodeID uint
	ExitNodeID  uint
}

// CreateTunnelResult 创建隧道结果
type CreateTunnelResult struct {
	Tunnel *tunnel.Tunnel
}

// CreateTunnelHandler 创建隧道处理器
type CreateTunnelHandler struct {
	tunnelRepo  tunnel.Repository
	userRepo    user.Repository
	tunnelCache *cache.TunnelCache
}

// NewCreateTunnelHandler 创建处理器
func NewCreateTunnelHandler(
	tunnelRepo tunnel.Repository,
	userRepo user.Repository,
	tunnelCache *cache.TunnelCache,
) *CreateTunnelHandler {
	return &CreateTunnelHandler{
		tunnelRepo:  tunnelRepo,
		userRepo:    userRepo,
		tunnelCache: tunnelCache,
	}
}

// Handle 处理命令
func (h *CreateTunnelHandler) Handle(ctx context.Context, cmd CreateTunnelCommand) (*CreateTunnelResult, error) {
	// 1. 验证用户是否存在
	u, err := h.userRepo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	
	// 2. 检查用户配额
	currentCount, err := h.tunnelRepo.CountByUserID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	
	if !u.CanCreateTunnel(int(currentCount)) {
		return nil, tunnel.ErrQuotaExceeded
	}
	
	// 3. 检查端口冲突（先查 Redis 缓存）
	conflict, err := h.tunnelCache.CheckPortConflict(ctx, cmd.ListenPort)
	if err == nil && conflict {
		return nil, tunnel.ErrPortConflict
	}
	
	// 4. 双重检查（查数据库）
	if existing, err := h.tunnelRepo.FindByPort(ctx, cmd.ListenPort); err == nil && existing != nil {
		return nil, tunnel.ErrPortConflict
	}
	
	// 5. 创建隧道实体
	newTunnel := &tunnel.Tunnel{
		UserID:      cmd.UserID,
		Name:        cmd.Name,
		Description: cmd.Description,
		Protocol:    cmd.Protocol,
		Mode:        cmd.Mode,
		ListenHost:  cmd.ListenHost,
		ListenPort:  cmd.ListenPort,
		TargetHost:  cmd.TargetHost,
		TargetPort:  cmd.TargetPort,
		EntryNodeID: cmd.EntryNodeID,
		ExitNodeID:  cmd.ExitNodeID,
		Status:      "stopped",
		IsEnabled:   true,
		TrafficIn:   0,
		TrafficOut:  0,
	}
	
	// 6. 验证配置
	if err := newTunnel.Validate(); err != nil {
		return nil, err
	}
	
	// 7. 保存到数据库
	if err := h.tunnelRepo.Create(ctx, newTunnel); err != nil {
		return nil, err
	}
	
	// 8. 预留端口（写入 Redis）
	h.tunnelCache.ReservePort(ctx, cmd.ListenPort, newTunnel.ID)
	
	// 9. 写入缓存
	tunnelData := map[string]interface{}{
		"id":          newTunnel.ID,
		"user_id":     newTunnel.UserID,
		"name":        newTunnel.Name,
		"protocol":    newTunnel.Protocol,
		"listen_port": newTunnel.ListenPort,
		"status":      newTunnel.Status,
	}
	h.tunnelCache.Set(ctx, newTunnel.ID, tunnelData)
	
	return &CreateTunnelResult{Tunnel: newTunnel}, nil
}
