package queries

import (
	"context"
	
	"nodepass-pro/backend/internal/domain/node"
	"nodepass-pro/backend/internal/infrastructure/cache"
)

// GetNodeQuery 获取节点查询
type GetNodeQuery struct {
	NodeID string
}

// GetNodeResult 获取节点结果
type GetNodeResult struct {
	Node     *node.NodeInstance
	IsOnline bool
}

// GetNodeHandler 获取节点处理器
type GetNodeHandler struct {
	nodeRepo  node.InstanceRepository
	nodeCache *cache.NodeCache
}

// NewGetNodeHandler 创建处理器
func NewGetNodeHandler(repo node.InstanceRepository, cache *cache.NodeCache) *GetNodeHandler {
	return &GetNodeHandler{
		nodeRepo:  repo,
		nodeCache: cache,
	}
}

// Handle 处理查询（Cache-Aside 模式）
func (h *GetNodeHandler) Handle(ctx context.Context, query GetNodeQuery) (*GetNodeResult, error) {
	// 1. 先查缓存的在线状态
	isOnline, _ := h.nodeCache.IsOnline(ctx, query.NodeID)
	
	// 2. 查数据库获取完整信息
	instance, err := h.nodeRepo.FindByNodeID(ctx, query.NodeID)
	if err != nil {
		return nil, err
	}
	
	// 3. 更新缓存
	nodeInfo := map[string]interface{}{
		"id":          instance.ID,
		"node_id":     instance.NodeID,
		"group_id":    instance.GroupID,
		"status":      instance.Status,
		"cpu_usage":   instance.CPUUsage,
		"memory_usage": instance.MemoryUsage,
	}
	h.nodeCache.SetNodeInfo(ctx, query.NodeID, nodeInfo)
	
	return &GetNodeResult{
		Node:     instance,
		IsOnline: isOnline,
	}, nil
}

// ListNodesQuery 列表查询
type ListNodesQuery struct {
	GroupID    uint
	Status     string
	OnlineOnly bool
	Page       int
	PageSize   int
}

// ListNodesResult 列表结果
type ListNodesResult struct {
	Nodes []*node.NodeInstance
	Total int64
}

// ListNodesHandler 列表查询处理器
type ListNodesHandler struct {
	nodeRepo  node.InstanceRepository
	nodeCache *cache.NodeCache
}

// NewListNodesHandler 创建处理器
func NewListNodesHandler(repo node.InstanceRepository, cache *cache.NodeCache) *ListNodesHandler {
	return &ListNodesHandler{
		nodeRepo:  repo,
		nodeCache: cache,
	}
}

// Handle 处理查询
func (h *ListNodesHandler) Handle(ctx context.Context, query ListNodesQuery) (*ListNodesResult, error) {
	filter := node.InstanceListFilter{
		Page:       query.Page,
		PageSize:   query.PageSize,
		GroupID:    query.GroupID,
		Status:     query.Status,
		OnlineOnly: query.OnlineOnly,
	}
	
	nodes, total, err := h.nodeRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	
	// 批量更新在线状态（从 Redis 获取）
	for _, n := range nodes {
		if isOnline, err := h.nodeCache.IsOnline(ctx, n.NodeID); err == nil {
			if isOnline {
				n.Status = "online"
			}
		}
	}
	
	return &ListNodesResult{
		Nodes: nodes,
		Total: total,
	}, nil
}

// GetOnlineNodesQuery 获取在线节点查询
type GetOnlineNodesQuery struct{}

// GetOnlineNodesResult 获取在线节点结果
type GetOnlineNodesResult struct {
	Nodes []*node.NodeInstance
	Count int
}

// GetOnlineNodesHandler 获取在线节点处理器
type GetOnlineNodesHandler struct {
	nodeRepo  node.InstanceRepository
	nodeCache *cache.NodeCache
}

// NewGetOnlineNodesHandler 创建处理器
func NewGetOnlineNodesHandler(repo node.InstanceRepository, cache *cache.NodeCache) *GetOnlineNodesHandler {
	return &GetOnlineNodesHandler{
		nodeRepo:  repo,
		nodeCache: cache,
	}
}

// Handle 处理查询（优先从 Redis 获取）
func (h *GetOnlineNodesHandler) Handle(ctx context.Context, query GetOnlineNodesQuery) (*GetOnlineNodesResult, error) {
	// 1. 从 Redis 获取在线节点列表
	onlineNodeIDs, err := h.nodeCache.GetAllOnlineNodes(ctx)
	if err != nil {
		// Redis 失败，降级到数据库查询
		return h.handleFallback(ctx)
	}
	
	if len(onlineNodeIDs) == 0 {
		return &GetOnlineNodesResult{
			Nodes: []*node.NodeInstance{},
			Count: 0,
		}, nil
	}
	
	// 2. 从数据库批量查询节点详情（可以考虑加缓存）
	nodes := make([]*node.NodeInstance, 0, len(onlineNodeIDs))
	for _, nodeID := range onlineNodeIDs {
		n, err := h.nodeRepo.FindByNodeID(ctx, nodeID)
		if err != nil {
			continue
		}
		nodes = append(nodes, n)
	}
	
	return &GetOnlineNodesResult{
		Nodes: nodes,
		Count: len(nodes),
	}, nil
}

// handleFallback 降级处理
func (h *GetOnlineNodesHandler) handleFallback(ctx context.Context) (*GetOnlineNodesResult, error) {
	nodes, err := h.nodeRepo.FindOnlineNodes(ctx)
	if err != nil {
		return nil, err
	}
	
	return &GetOnlineNodesResult{
		Nodes: nodes,
		Count: len(nodes),
	}, nil
}
