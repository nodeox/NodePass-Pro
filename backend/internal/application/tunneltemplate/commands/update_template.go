package commands

import (
	"context"
	"fmt"
	"strings"

	"nodepass-pro/backend/internal/domain/tunneltemplate"
)

// UpdateTemplateCommand 更新模板命令
type UpdateTemplateCommand struct {
	TemplateID  uint
	UserID      uint
	Name        *string
	Description *string
	Config      *tunneltemplate.TemplateConfig
	IsPublic    *bool
}

// UpdateTemplateHandler 更新模板处理器
type UpdateTemplateHandler struct {
	repo tunneltemplate.Repository
}

// NewUpdateTemplateHandler 创建处理器
func NewUpdateTemplateHandler(repo tunneltemplate.Repository) *UpdateTemplateHandler {
	return &UpdateTemplateHandler{repo: repo}
}

// Handle 处理命令
func (h *UpdateTemplateHandler) Handle(ctx context.Context, cmd UpdateTemplateCommand) (*tunneltemplate.TunnelTemplate, error) {
	// 查找模板
	template, err := h.repo.FindByID(ctx, cmd.TemplateID)
	if err != nil {
		return nil, err
	}

	// 检查权限
	if !template.IsOwnedBy(cmd.UserID) {
		return nil, tunneltemplate.ErrUnauthorized
	}

	// 更新基本信息
	if cmd.Name != nil || cmd.Description != nil {
		name := template.Name
		if cmd.Name != nil {
			name = strings.TrimSpace(*cmd.Name)
			if name == "" {
				return nil, tunneltemplate.ErrInvalidTemplateName
			}
			if len(name) > 100 {
				return nil, fmt.Errorf("%w: 名称长度不能超过 100", tunneltemplate.ErrInvalidTemplateName)
			}
		}

		description := template.Description
		if cmd.Description != nil {
			description = cmd.Description
		}

		template.UpdateInfo(name, description)
	}

	// 更新配置
	if cmd.Config != nil {
		template.UpdateConfig(cmd.Config)
	}

	// 更新公开状态
	if cmd.IsPublic != nil {
		if *cmd.IsPublic {
			template.MakePublic()
		} else {
			template.MakePrivate()
		}
	}

	// 保存
	if err := h.repo.Update(ctx, template); err != nil {
		return nil, fmt.Errorf("更新模板失败: %w", err)
	}

	return template, nil
}
