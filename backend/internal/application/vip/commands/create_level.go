package commands

import (
	"context"
	"fmt"
	"strings"

	"nodepass-pro/backend/internal/domain/vip"
)

// CreateLevelCommand 创建 VIP 等级命令
type CreateLevelCommand struct {
	Level                   int
	Name                    string
	Description             *string
	TrafficQuota            int64
	MaxRules                int
	MaxBandwidth            int
	MaxSelfHostedEntryNodes int
	MaxSelfHostedExitNodes  int
	AccessibleNodeLevel     int
	TrafficMultiplier       float64
	CustomFeatures          *string
	Price                   *float64
	DurationDays            *int
}

// CreateLevelResult 创建 VIP 等级结果
type CreateLevelResult struct {
	ID    uint   `json:"id"`
	Level int    `json:"level"`
	Name  string `json:"name"`
}

// CreateLevelHandler 创建 VIP 等级命令处理器
type CreateLevelHandler struct {
	vipRepo vip.Repository
}

// NewCreateLevelHandler 创建 VIP 等级命令处理器
func NewCreateLevelHandler(vipRepo vip.Repository) *CreateLevelHandler {
	return &CreateLevelHandler{
		vipRepo: vipRepo,
	}
}

// Handle 处理创建 VIP 等级命令
func (h *CreateLevelHandler) Handle(ctx context.Context, cmd CreateLevelCommand) (*CreateLevelResult, error) {
	// 1. 验证输入
	if err := h.validateCommand(cmd); err != nil {
		return nil, err
	}

	// 2. 检查等级是否已存在
	exists, err := h.vipRepo.CheckLevelExists(ctx, cmd.Level)
	if err != nil {
		return nil, fmt.Errorf("检查 VIP 等级失败: %w", err)
	}
	if exists {
		return nil, vip.ErrLevelExists
	}

	// 3. 创建 VIP 等级实体
	level := &vip.VIPLevel{
		Level:                   cmd.Level,
		Name:                    strings.TrimSpace(cmd.Name),
		Description:             cmd.Description,
		TrafficQuota:            cmd.TrafficQuota,
		MaxRules:                cmd.MaxRules,
		MaxBandwidth:            cmd.MaxBandwidth,
		MaxSelfHostedEntryNodes: cmd.MaxSelfHostedEntryNodes,
		MaxSelfHostedExitNodes:  cmd.MaxSelfHostedExitNodes,
		AccessibleNodeLevel:     cmd.AccessibleNodeLevel,
		TrafficMultiplier:       cmd.TrafficMultiplier,
		CustomFeatures:          cmd.CustomFeatures,
		Price:                   cmd.Price,
		DurationDays:            cmd.DurationDays,
	}

	// 4. 持久化
	if err := h.vipRepo.CreateLevel(ctx, level); err != nil {
		return nil, fmt.Errorf("创建 VIP 等级失败: %w", err)
	}

	return &CreateLevelResult{
		ID:    level.ID,
		Level: level.Level,
		Name:  level.Name,
	}, nil
}

func (h *CreateLevelHandler) validateCommand(cmd CreateLevelCommand) error {
	if cmd.Level < 0 {
		return vip.ErrInvalidLevel
	}

	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return fmt.Errorf("VIP 等级名称不能为空")
	}
	if len(name) > 50 {
		return fmt.Errorf("VIP 等级名称长度不能超过 50 个字符")
	}

	if cmd.TrafficQuota < 0 {
		return vip.ErrInvalidTrafficQuota
	}

	if cmd.MaxRules < -1 {
		return fmt.Errorf("最大规则数不能小于 -1")
	}

	if cmd.MaxBandwidth < -1 {
		return fmt.Errorf("最大带宽不能小于 -1")
	}

	if cmd.TrafficMultiplier < 0 {
		return fmt.Errorf("流量倍率不能为负数")
	}

	return nil
}
