package commands

import (
	"context"
	"time"
	
	"nodepass-pro/backend/internal/domain/node"
	"nodepass-pro/backend/internal/infrastructure/cache"
)

// HeartbeatCommand 心跳命令
type HeartbeatCommand struct {
	NodeID        string
	CPUUsage      float64
	MemoryUsage   float64
	DiskUsage     float64
	TrafficIn     int64
	TrafficOut    int64
	ActiveRules   int
	ConfigVersion int
	ClientVersion string
}

// HeartbeatResult 心跳结果
type HeartbeatResult struct {
	ConfigUpdated     bool
	NewConfigVersion  int
	ShouldRestart     bool
}

// HeartbeatHandler 心跳处理器
type HeartbeatHandler struct {
	nodeRepo        node.InstanceRepository
	nodeCache       *cache.NodeCache
	heartbeatBuffer *cache.HeartbeatBuffer
}

// NewHeartbeatHandler 创建处理器
func NewHeartbeatHandler(
	repo node.InstanceRepository,
	cache *cache.NodeCache,
	buffer *cache.HeartbeatBuffer,
) *HeartbeatHandler {
	return &HeartbeatHandler{
		nodeRepo:        repo,
		nodeCache:       cache,
		heartbeatBuffer: buffer,
	}
}

// Handle 处理心跳（高性能模式：先写 Redis，异步写数据库）
func (h *HeartbeatHandler) Handle(ctx context.Context, cmd HeartbeatCommand) (*HeartbeatResult, error) {
	// 1. 推送到 Redis 缓冲区（异步批量写入数据库）
	heartbeatData := &cache.HeartbeatData{
		NodeID:        cmd.NodeID,
		Status:        "online",
		CPUUsage:      cmd.CPUUsage,
		MemoryUsage:   cmd.MemoryUsage,
		TrafficIn:     cmd.TrafficIn,
		TrafficOut:    cmd.TrafficOut,
		Timestamp:     time.Now(),
	}
	
	if err := h.heartbeatBuffer.Push(ctx, heartbeatData); err != nil {
		// 缓冲区失败，降级到直接写数据库
		return h.handleFallback(ctx, cmd)
	}
	
	// 2. 更新在线状态（3 分钟 TTL）
	if err := h.nodeCache.SetOnline(ctx, cmd.NodeID, 3*time.Minute); err != nil {
		// 忽略缓存错误，不影响主流程
	}
	
	// 3. 更新节点指标（用于实时监控）
	metrics := map[string]float64{
		"cpu":    cmd.CPUUsage,
		"memory": cmd.MemoryUsage,
		"disk":   cmd.DiskUsage,
	}
	h.nodeCache.SetNodeMetrics(ctx, cmd.NodeID, metrics)
	
	// 4. 检查配置版本（从数据库查询）
	instance, err := h.nodeRepo.FindByNodeID(ctx, cmd.NodeID)
	if err != nil {
		// 节点不存在，返回默认结果
		return &HeartbeatResult{
			ConfigUpdated:    false,
			NewConfigVersion: 0,
		}, nil
	}
	
	// 5. 判断是否需要更新配置
	configUpdated := instance.ConfigVersion > cmd.ConfigVersion
	
	return &HeartbeatResult{
		ConfigUpdated:    configUpdated,
		NewConfigVersion: instance.ConfigVersion,
		ShouldRestart:    false,
	}, nil
}

// handleFallback 降级处理（直接写数据库）
func (h *HeartbeatHandler) handleFallback(ctx context.Context, cmd HeartbeatCommand) (*HeartbeatResult, error) {
	data := &node.HeartbeatData{
		NodeID:        cmd.NodeID,
		CPUUsage:      cmd.CPUUsage,
		MemoryUsage:   cmd.MemoryUsage,
		DiskUsage:     cmd.DiskUsage,
		TrafficIn:     cmd.TrafficIn,
		TrafficOut:    cmd.TrafficOut,
		ActiveRules:   cmd.ActiveRules,
		ConfigVersion: cmd.ConfigVersion,
		ClientVersion: cmd.ClientVersion,
		Timestamp:     time.Now(),
	}
	
	if err := h.nodeRepo.UpdateHeartbeat(ctx, cmd.NodeID, data); err != nil {
		return nil, err
	}
	
	instance, err := h.nodeRepo.FindByNodeID(ctx, cmd.NodeID)
	if err != nil {
		return &HeartbeatResult{ConfigUpdated: false}, nil
	}
	
	return &HeartbeatResult{
		ConfigUpdated:    instance.ConfigVersion > cmd.ConfigVersion,
		NewConfigVersion: instance.ConfigVersion,
	}, nil
}

// FlushHeartbeats 批量刷新心跳到数据库（定时任务调用）
func (h *HeartbeatHandler) FlushHeartbeats(ctx context.Context) error {
	// 获取所有在线节点
	nodes, err := h.nodeCache.GetAllOnlineNodes(ctx)
	if err != nil {
		return err
	}
	
	// 批量弹出并写入数据库
	for _, nodeID := range nodes {
		// 每次弹出 100 条
		heartbeats, err := h.heartbeatBuffer.PopBatch(ctx, nodeID, 100)
		if err != nil {
			continue
		}
		
		if len(heartbeats) == 0 {
			continue
		}
		
		// 转换为领域对象
		data := make([]*node.HeartbeatData, len(heartbeats))
		for i, hb := range heartbeats {
			data[i] = &node.HeartbeatData{
				NodeID:      hb.NodeID,
				CPUUsage:    hb.CPUUsage,
				MemoryUsage: hb.MemoryUsage,
				TrafficIn:   hb.TrafficIn,
				TrafficOut:  hb.TrafficOut,
				Timestamp:   hb.Timestamp,
			}
		}
		
		// 批量更新数据库
		if err := h.nodeRepo.BatchUpdateHeartbeat(ctx, data); err != nil {
			// 记录错误但继续处理其他节点
			continue
		}
	}
	
	return nil
}
