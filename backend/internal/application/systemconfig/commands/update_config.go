package commands

import (
	"context"
	"fmt"
	"strings"

	"nodepass-pro/backend/internal/domain/systemconfig"
)

// UpdateConfigCommand 更新配置命令
type UpdateConfigCommand struct {
	Key   string
	Value *string
}

// UpdateConfigHandler 更新配置处理器
type UpdateConfigHandler struct {
	repo systemconfig.Repository
}

// NewUpdateConfigHandler 创建处理器
func NewUpdateConfigHandler(repo systemconfig.Repository) *UpdateConfigHandler {
	return &UpdateConfigHandler{repo: repo}
}

// Handle 处理命令
func (h *UpdateConfigHandler) Handle(ctx context.Context, cmd UpdateConfigCommand) error {
	// 验证键
	key := strings.TrimSpace(cmd.Key)
	if key == "" {
		return systemconfig.ErrInvalidKey
	}

	// 查找或创建配置
	config, err := h.repo.FindByKey(ctx, key)
	if err != nil && err != systemconfig.ErrConfigNotFound {
		return err
	}

	if config == nil {
		// 创建新配置
		config = systemconfig.NewSystemConfig(key, cmd.Value)
	} else {
		// 更新现有配置
		config.UpdateValue(cmd.Value)
	}

	// 保存
	if err := h.repo.Upsert(ctx, config); err != nil {
		return fmt.Errorf("更新配置失败: %w", err)
	}

	return nil
}
