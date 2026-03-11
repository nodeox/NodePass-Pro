package commands

import (
	"context"
	"time"
	
	"nodepass-pro/backend/internal/domain/traffic"
	"nodepass-pro/backend/internal/infrastructure/cache"
)

// RecordTrafficCommand 记录流量命令
type RecordTrafficCommand struct {
	UserID     uint
	TunnelID   uint
	TrafficIn  int64
	TrafficOut int64
}

// RecordTrafficResult 记录流量结果
type RecordTrafficResult struct {
	Success bool
}

// RecordTrafficHandler 记录流量处理器
type RecordTrafficHandler struct {
	trafficCounter *cache.TrafficCounter
}

// NewRecordTrafficHandler 创建处理器
func NewRecordTrafficHandler(counter *cache.TrafficCounter) *RecordTrafficHandler {
	return &RecordTrafficHandler{
		trafficCounter: counter,
	}
}

// Handle 处理命令（高性能模式：只写 Redis，定时同步到数据库）
func (h *RecordTrafficHandler) Handle(ctx context.Context, cmd RecordTrafficCommand) (*RecordTrafficResult, error) {
	// 1. 增加用户流量（Redis 原子操作）
	if err := h.trafficCounter.IncrementUserTraffic(ctx, cmd.UserID, cmd.TrafficIn, cmd.TrafficOut); err != nil {
		return nil, err
	}
	
	// 2. 增加隧道流量（Redis 原子操作）
	if cmd.TunnelID > 0 {
		if err := h.trafficCounter.IncrementTunnelTraffic(ctx, cmd.TunnelID, cmd.TrafficIn, cmd.TrafficOut); err != nil {
			// 隧道流量统计失败不影响主流程
		}
	}
	
	return &RecordTrafficResult{Success: true}, nil
}

// FlushTrafficCommand 刷新流量到数据库命令
type FlushTrafficCommand struct{}

// FlushTrafficResult 刷新流量结果
type FlushTrafficResult struct {
	FlushedUsers   int
	FlushedTunnels int
}

// FlushTrafficHandler 刷新流量处理器
type FlushTrafficHandler struct {
	recordRepo     traffic.RecordRepository
	trafficCounter *cache.TrafficCounter
}

// NewFlushTrafficHandler 创建处理器
func NewFlushTrafficHandler(
	recordRepo traffic.RecordRepository,
	counter *cache.TrafficCounter,
) *FlushTrafficHandler {
	return &FlushTrafficHandler{
		recordRepo:     recordRepo,
		trafficCounter: counter,
	}
}

// Handle 处理命令（定时任务调用）
func (h *FlushTrafficHandler) Handle(ctx context.Context, cmd FlushTrafficCommand) (*FlushTrafficResult, error) {
	// TODO: 实现批量刷新逻辑
	// 1. 从 Redis 获取所有用户流量
	// 2. 批量创建流量记录
	// 3. 重置 Redis 计数器
	
	return &FlushTrafficResult{
		FlushedUsers:   0,
		FlushedTunnels: 0,
	}, nil
}
