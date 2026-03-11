package queries

import (
	"context"
	"time"
	
	"nodepass-pro/backend/internal/domain/traffic"
	"nodepass-pro/backend/internal/infrastructure/cache"
)

// GetUserTrafficQuery 获取用户流量查询
type GetUserTrafficQuery struct {
	UserID    uint
	StartTime time.Time
	EndTime   time.Time
}

// GetUserTrafficResult 获取用户流量结果
type GetUserTrafficResult struct {
	UserID     uint
	TrafficIn  int64
	TrafficOut int64
	Total      int64
	Period     string
}

// GetUserTrafficHandler 获取用户流量处理器
type GetUserTrafficHandler struct {
	recordRepo     traffic.RecordRepository
	trafficCounter *cache.TrafficCounter
}

// NewGetUserTrafficHandler 创建处理器
func NewGetUserTrafficHandler(
	recordRepo traffic.RecordRepository,
	counter *cache.TrafficCounter,
) *GetUserTrafficHandler {
	return &GetUserTrafficHandler{
		recordRepo:     recordRepo,
		trafficCounter: counter,
	}
}

// Handle 处理查询（合并数据库历史数据 + Redis 实时数据）
func (h *GetUserTrafficHandler) Handle(ctx context.Context, query GetUserTrafficQuery) (*GetUserTrafficResult, error) {
	// 1. 从数据库获取历史数据
	dbIn, dbOut, err := h.recordRepo.SumByUserID(ctx, query.UserID, query.StartTime, query.EndTime)
	if err != nil {
		return nil, err
	}
	
	// 2. 从 Redis 获取实时数据（未同步到数据库的部分）
	redisIn, redisOut, err := h.trafficCounter.GetUserTraffic(ctx, query.UserID)
	if err != nil {
		// Redis 失败，只返回数据库数据
		return &GetUserTrafficResult{
			UserID:     query.UserID,
			TrafficIn:  dbIn,
			TrafficOut: dbOut,
			Total:      dbIn + dbOut,
			Period:     formatPeriod(query.StartTime, query.EndTime),
		}, nil
	}
	
	// 3. 合并数据
	totalIn := dbIn + redisIn
	totalOut := dbOut + redisOut
	
	return &GetUserTrafficResult{
		UserID:     query.UserID,
		TrafficIn:  totalIn,
		TrafficOut: totalOut,
		Total:      totalIn + totalOut,
		Period:     formatPeriod(query.StartTime, query.EndTime),
	}, nil
}

// GetTunnelTrafficQuery 获取隧道流量查询
type GetTunnelTrafficQuery struct {
	TunnelID  uint
	StartTime time.Time
	EndTime   time.Time
}

// GetTunnelTrafficResult 获取隧道流量结果
type GetTunnelTrafficResult struct {
	TunnelID   uint
	TrafficIn  int64
	TrafficOut int64
	Total      int64
	Period     string
}

// GetTunnelTrafficHandler 获取隧道流量处理器
type GetTunnelTrafficHandler struct {
	recordRepo     traffic.RecordRepository
	trafficCounter *cache.TrafficCounter
}

// NewGetTunnelTrafficHandler 创建处理器
func NewGetTunnelTrafficHandler(
	recordRepo traffic.RecordRepository,
	counter *cache.TrafficCounter,
) *GetTunnelTrafficHandler {
	return &GetTunnelTrafficHandler{
		recordRepo:     recordRepo,
		trafficCounter: counter,
	}
}

// Handle 处理查询
func (h *GetTunnelTrafficHandler) Handle(ctx context.Context, query GetTunnelTrafficQuery) (*GetTunnelTrafficResult, error) {
	// 1. 从数据库获取历史数据
	dbIn, dbOut, err := h.recordRepo.SumByTunnelID(ctx, query.TunnelID, query.StartTime, query.EndTime)
	if err != nil {
		return nil, err
	}
	
	// 2. 从 Redis 获取实时数据
	redisIn, redisOut, err := h.trafficCounter.GetTunnelTraffic(ctx, query.TunnelID)
	if err != nil {
		// Redis 失败，只返回数据库数据
		return &GetTunnelTrafficResult{
			TunnelID:   query.TunnelID,
			TrafficIn:  dbIn,
			TrafficOut: dbOut,
			Total:      dbIn + dbOut,
			Period:     formatPeriod(query.StartTime, query.EndTime),
		}, nil
	}
	
	// 3. 合并数据
	totalIn := dbIn + redisIn
	totalOut := dbOut + redisOut
	
	return &GetTunnelTrafficResult{
		TunnelID:   query.TunnelID,
		TrafficIn:  totalIn,
		TrafficOut: totalOut,
		Total:      totalIn + totalOut,
		Period:     formatPeriod(query.StartTime, query.EndTime),
	}, nil
}

// formatPeriod 格式化时间段
func formatPeriod(start, end time.Time) string {
	if start.IsZero() && end.IsZero() {
		return "all_time"
	}
	if start.IsZero() {
		return "until_" + end.Format("2006-01-02")
	}
	if end.IsZero() {
		return "from_" + start.Format("2006-01-02")
	}
	return start.Format("2006-01-02") + "_to_" + end.Format("2006-01-02")
}
