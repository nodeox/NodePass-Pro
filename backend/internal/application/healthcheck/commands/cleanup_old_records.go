package commands

import (
	"context"
	"time"

	"nodepass-pro/backend/internal/domain/healthcheck"
)

// CleanupOldRecordsCommand 清理旧记录命令
type CleanupOldRecordsCommand struct {
	RetentionDays int
}

// CleanupOldRecordsHandler 清理旧记录处理器
type CleanupOldRecordsHandler struct {
	repo healthcheck.Repository
}

// NewCleanupOldRecordsHandler 创建处理器
func NewCleanupOldRecordsHandler(repo healthcheck.Repository) *CleanupOldRecordsHandler {
	return &CleanupOldRecordsHandler{repo: repo}
}

// Handle 处理命令
func (h *CleanupOldRecordsHandler) Handle(ctx context.Context, cmd CleanupOldRecordsCommand) (int64, error) {
	retentionDays := cmd.RetentionDays
	if retentionDays <= 0 {
		retentionDays = 30
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	return h.repo.DeleteOldHealthRecords(ctx, cutoffTime)
}
