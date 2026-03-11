package commands

import (
	"context"
	"fmt"
	"strings"

	"nodepass-pro/backend/internal/domain/tunneltemplate"
)

// CreateTemplateCommand 创建模板命令
type CreateTemplateCommand struct {
	UserID      uint
	Name        string
	Description *string
	Protocol    string
	Config      *tunneltemplate.TemplateConfig
	IsPublic    bool
}

// CreateTemplateHandler 创建模板处理器
type CreateTemplateHandler struct {
	repo tunneltemplate.Repository
}

// NewCreateTemplateHandler 创建处理器
func NewCreateTemplateHandler(repo tunneltemplate.Repository) *CreateTemplateHandler {
	return &CreateTemplateHandler{repo: repo}
}

// Handle 处理命令
func (h *CreateTemplateHandler) Handle(ctx context.Context, cmd CreateTemplateCommand) (*tunneltemplate.TunnelTemplate, error) {
	// 验证名称
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, tunneltemplate.ErrInvalidTemplateName
	}
	if len(name) > 100 {
		return nil, fmt.Errorf("%w: 名称长度不能超过 100", tunneltemplate.ErrInvalidTemplateName)
	}

	// 验证协议
	protocol := strings.TrimSpace(strings.ToLower(cmd.Protocol))
	if protocol == "" {
		return nil, tunneltemplate.ErrInvalidProtocol
	}

	// 验证配置
	if cmd.Config == nil {
		return nil, tunneltemplate.ErrInvalidConfig
	}

	// 检查是否已存在同名模板
	existing, err := h.repo.FindByUserAndName(ctx, cmd.UserID, name)
	if err == nil && existing != nil {
		return nil, tunneltemplate.ErrTemplateAlreadyExists
	}

	// 创建模板
	template := tunneltemplate.NewTunnelTemplate(cmd.UserID, name, protocol, cmd.Config)
	template.Description = cmd.Description
	if cmd.IsPublic {
		template.MakePublic()
	}

	// 保存
	if err := h.repo.Create(ctx, template); err != nil {
		return nil, fmt.Errorf("创建模板失败: %w", err)
	}

	return template, nil
}
