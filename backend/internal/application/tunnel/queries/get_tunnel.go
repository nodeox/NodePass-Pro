package queries

import (
	"context"
	
	"nodepass-pro/backend/internal/domain/tunnel"
	"nodepass-pro/backend/internal/infrastructure/cache"
)

// GetTunnelQuery 获取隧道查询
type GetTunnelQuery struct {
	TunnelID uint
	UserID   uint
}

// GetTunnelResult 获取隧道结果
type GetTunnelResult struct {
	Tunnel    *tunnel.Tunnel
	IsRunning bool
}

// GetTunnelHandler 获取隧道处理器
type GetTunnelHandler struct {
	tunnelRepo  tunnel.Repository
	tunnelCache *cache.TunnelCache
}

// NewGetTunnelHandler 创建处理器
func NewGetTunnelHandler(repo tunnel.Repository, cache *cache.TunnelCache) *GetTunnelHandler {
	return &GetTunnelHandler{
		tunnelRepo:  repo,
		tunnelCache: cache,
	}
}

// Handle 处理查询（Cache-Aside 模式）
func (h *GetTunnelHandler) Handle(ctx context.Context, query GetTunnelQuery) (*GetTunnelResult, error) {
	// 1. 先查缓存
	if cached, err := h.tunnelCache.Get(ctx, query.TunnelID); err == nil && cached != nil {
		// 缓存命中，构建实体
		t := &tunnel.Tunnel{
			ID:         uint(cached["id"].(float64)),
			UserID:     uint(cached["user_id"].(float64)),
			Name:       cached["name"].(string),
			Protocol:   cached["protocol"].(string),
			ListenPort: int(cached["listen_port"].(float64)),
			Status:     cached["status"].(string),
		}
		
		// 验证权限
		if t.UserID != query.UserID {
			return nil, tunnel.ErrTunnelNotFound
		}
		
		isRunning, _ := h.tunnelCache.IsRunning(ctx, query.TunnelID)
		return &GetTunnelResult{Tunnel: t, IsRunning: isRunning}, nil
	}
	
	// 2. 缓存未命中，查数据库
	t, err := h.tunnelRepo.FindByID(ctx, query.TunnelID)
	if err != nil {
		return nil, err
	}
	
	// 3. 验证权限
	if t.UserID != query.UserID {
		return nil, tunnel.ErrTunnelNotFound
	}
	
	// 4. 写入缓存
	tunnelData := map[string]interface{}{
		"id":          t.ID,
		"user_id":     t.UserID,
		"name":        t.Name,
		"protocol":    t.Protocol,
		"listen_port": t.ListenPort,
		"status":      t.Status,
	}
	h.tunnelCache.Set(ctx, t.ID, tunnelData)
	
	isRunning := t.IsRunning()
	return &GetTunnelResult{Tunnel: t, IsRunning: isRunning}, nil
}

// ListTunnelsQuery 列表查询
type ListTunnelsQuery struct {
	UserID      uint
	Status      string
	Protocol    string
	EnabledOnly bool
	Page        int
	PageSize    int
}

// ListTunnelsResult 列表结果
type ListTunnelsResult struct {
	Tunnels []*tunnel.Tunnel
	Total   int64
}

// ListTunnelsHandler 列表查询处理器
type ListTunnelsHandler struct {
	tunnelRepo  tunnel.Repository
	tunnelCache *cache.TunnelCache
}

// NewListTunnelsHandler 创建处理器
func NewListTunnelsHandler(repo tunnel.Repository, cache *cache.TunnelCache) *ListTunnelsHandler {
	return &ListTunnelsHandler{
		tunnelRepo:  repo,
		tunnelCache: cache,
	}
}

// Handle 处理查询
func (h *ListTunnelsHandler) Handle(ctx context.Context, query ListTunnelsQuery) (*ListTunnelsResult, error) {
	filter := tunnel.ListFilter{
		Page:        query.Page,
		PageSize:    query.PageSize,
		UserID:      query.UserID,
		Status:      query.Status,
		Protocol:    query.Protocol,
		EnabledOnly: query.EnabledOnly,
	}
	
	tunnels, total, err := h.tunnelRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	
	// 批量更新运行状态（从 Redis 获取）
	for _, t := range tunnels {
		if isRunning, err := h.tunnelCache.IsRunning(ctx, t.ID); err == nil {
			if isRunning {
				t.Status = "running"
			}
		}
	}
	
	return &ListTunnelsResult{
		Tunnels: tunnels,
		Total:   total,
	}, nil
}

// GetTunnelTrafficQuery 获取隧道流量查询
type GetTunnelTrafficQuery struct {
	TunnelID uint
	UserID   uint
}

// GetTunnelTrafficResult 获取隧道流量结果
type GetTunnelTrafficResult struct {
	TunnelID   uint
	TrafficIn  int64
	TrafficOut int64
	Total      int64
}

// GetTunnelTrafficHandler 获取隧道流量处理器
type GetTunnelTrafficHandler struct {
	tunnelRepo  tunnel.Repository
	tunnelCache *cache.TunnelCache
}

// NewGetTunnelTrafficHandler 创建处理器
func NewGetTunnelTrafficHandler(repo tunnel.Repository, cache *cache.TunnelCache) *GetTunnelTrafficHandler {
	return &GetTunnelTrafficHandler{
		tunnelRepo:  repo,
		tunnelCache: cache,
	}
}

// Handle 处理查询（优先从 Redis 获取实时数据）
func (h *GetTunnelTrafficHandler) Handle(ctx context.Context, query GetTunnelTrafficQuery) (*GetTunnelTrafficResult, error) {
	// 1. 验证权限
	t, err := h.tunnelRepo.FindByID(ctx, query.TunnelID)
	if err != nil {
		return nil, err
	}
	
	if t.UserID != query.UserID {
		return nil, tunnel.ErrTunnelNotFound
	}
	
	// 2. 从 Redis 获取实时流量
	inBytes, outBytes, err := h.tunnelCache.GetTraffic(ctx, query.TunnelID)
	if err != nil {
		// Redis 失败，降级到数据库
		return &GetTunnelTrafficResult{
			TunnelID:   t.ID,
			TrafficIn:  t.TrafficIn,
			TrafficOut: t.TrafficOut,
			Total:      t.TrafficIn + t.TrafficOut,
		}, nil
	}
	
	// 3. 合并数据库和 Redis 的数据
	totalIn := t.TrafficIn + inBytes
	totalOut := t.TrafficOut + outBytes
	
	return &GetTunnelTrafficResult{
		TunnelID:   t.ID,
		TrafficIn:  totalIn,
		TrafficOut: totalOut,
		Total:      totalIn + totalOut,
	}, nil
}
