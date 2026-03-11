package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/healthcheck"
)

// PerformHealthCheckCommand 执行健康检查命令
type PerformHealthCheckCommand struct {
	NodeInstanceID uint
}

// PerformHealthCheckHandler 执行健康检查处理器
type PerformHealthCheckHandler struct {
	repo    healthcheck.Repository
	checker healthcheck.Checker
}

// NewPerformHealthCheckHandler 创建处理器
func NewPerformHealthCheckHandler(repo healthcheck.Repository, checker healthcheck.Checker) *PerformHealthCheckHandler {
	return &PerformHealthCheckHandler{
		repo:    repo,
		checker: checker,
	}
}

// Handle 处理命令
func (h *PerformHealthCheckHandler) Handle(ctx context.Context, cmd PerformHealthCheckCommand) (*healthcheck.HealthRecord, error) {
	// 获取健康检查配置
	check, err := h.repo.FindHealthCheckByNodeInstance(ctx, cmd.NodeInstanceID)
	if err != nil {
		// 如果没有配置，使用默认 TCP 检查
		check = healthcheck.NewHealthCheck(cmd.NodeInstanceID, healthcheck.CheckTypeTCP)
	}

	// 执行健康检查
	record, err := h.checker.Check(ctx, cmd.NodeInstanceID, check)
	if err != nil {
		return nil, fmt.Errorf("执行健康检查失败: %w", err)
	}

	// 保存检查记录
	if err := h.repo.CreateHealthRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("保存健康检查记录失败: %w", err)
	}

	// 更新质量评分
	if err := h.updateQualityScore(ctx, cmd.NodeInstanceID); err != nil {
		// 记录错误但不影响主流程
		fmt.Printf("更新质量评分失败: %v\n", err)
	}

	return record, nil
}

// updateQualityScore 更新质量评分
func (h *PerformHealthCheckHandler) updateQualityScore(ctx context.Context, nodeInstanceID uint) error {
	// 获取最近 100 条健康检查记录
	records, err := h.repo.FindHealthRecordsByNodeInstance(ctx, nodeInstanceID, 100)
	if err != nil || len(records) == 0 {
		return err
	}

	// 获取或创建质量评分
	score, err := h.repo.FindQualityScoreByNodeInstance(ctx, nodeInstanceID)
	if err != nil {
		score = healthcheck.NewQualityScore(nodeInstanceID)
	}

	// 根据记录更新评分
	score.UpdateFromRecords(records)

	// 保存评分
	return h.repo.CreateOrUpdateQualityScore(ctx, score)
}
