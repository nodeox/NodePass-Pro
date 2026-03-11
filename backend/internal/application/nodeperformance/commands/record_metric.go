package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/nodeperformance"
)

// RecordMetricCommand 记录性能指标命令
type RecordMetricCommand struct {
	NodeInstanceID uint
	CPUUsage       float64
	MemoryUsage    float64
	DiskUsage      float64
	NetworkIn      int64
	NetworkOut     int64
	Connections    int
	Latency        *int
	PacketLoss     *float64
}

// RecordMetricHandler 记录性能指标处理器
type RecordMetricHandler struct {
	repo nodeperformance.Repository
}

// NewRecordMetricHandler 创建处理器
func NewRecordMetricHandler(repo nodeperformance.Repository) *RecordMetricHandler {
	return &RecordMetricHandler{repo: repo}
}

// Handle 处理命令
func (h *RecordMetricHandler) Handle(ctx context.Context, cmd RecordMetricCommand) error {
	metric := nodeperformance.NewPerformanceMetric(cmd.NodeInstanceID)
	metric.CPUUsage = cmd.CPUUsage
	metric.MemoryUsage = cmd.MemoryUsage
	metric.DiskUsage = cmd.DiskUsage
	metric.NetworkIn = cmd.NetworkIn
	metric.NetworkOut = cmd.NetworkOut
	metric.Connections = cmd.Connections
	metric.Latency = cmd.Latency
	metric.PacketLoss = cmd.PacketLoss

	if err := h.repo.RecordMetric(ctx, metric); err != nil {
		return fmt.Errorf("记录性能指标失败: %w", err)
	}

	return nil
}
