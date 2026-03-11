package commands

import (
	"context"
	"crypto/sha256"
	"fmt"

	"nodepass-pro/backend/internal/domain/alert"
)

// CreateAlertCommand 创建告警命令
type CreateAlertCommand struct {
	Type         string
	Level        alert.AlertLevel
	Title        string
	Message      string
	ResourceType string
	ResourceID   uint
	ResourceName string
	Value        string
	Threshold    string
}

// CreateAlertHandler 创建告警处理器
type CreateAlertHandler struct {
	repo alert.Repository
}

// NewCreateAlertHandler 创建处理器
func NewCreateAlertHandler(repo alert.Repository) *CreateAlertHandler {
	return &CreateAlertHandler{repo: repo}
}

// Handle 处理命令
func (h *CreateAlertHandler) Handle(ctx context.Context, cmd CreateAlertCommand) (*alert.Alert, error) {
	// 创建告警实体
	a := alert.NewAlert(cmd.Type, cmd.Title, cmd.Message, cmd.Level, cmd.ResourceType, cmd.ResourceID)
	a.ResourceName = cmd.ResourceName
	a.Value = cmd.Value
	a.Threshold = cmd.Threshold

	// 生成指纹用于去重
	a.Fingerprint = generateFingerprint(cmd.Type, cmd.ResourceType, cmd.ResourceID)

	// 检查是否已存在相同指纹的告警
	existing, err := h.repo.FindByFingerprint(ctx, a.Fingerprint)
	if err == nil && existing != nil && existing.IsFiring() {
		// 已存在，更新现有告警
		existing.Fire(cmd.Value)
		existing.Message = cmd.Message
		if err := h.repo.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("更新告警失败: %w", err)
		}
		return existing, nil
	}

	// 不存在，创建新告警
	if err := h.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("创建告警失败: %w", err)
	}

	return a, nil
}

// generateFingerprint 生成告警指纹
func generateFingerprint(alertType, resourceType string, resourceID uint) string {
	data := fmt.Sprintf("%s:%s:%d", alertType, resourceType, resourceID)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:16])
}
