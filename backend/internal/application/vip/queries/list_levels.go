package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/vip"
)

// ListLevelsQuery 列出所有 VIP 等级查询
type ListLevelsQuery struct{}

// ListLevelsResult 列出所有 VIP 等级结果
type ListLevelsResult struct {
	Levels []*VIPLevelInfo `json:"levels"`
	Total  int             `json:"total"`
}

// VIPLevelInfo VIP 等级信息
type VIPLevelInfo struct {
	ID                      uint     `json:"id"`
	Level                   int      `json:"level"`
	Name                    string   `json:"name"`
	Description             *string  `json:"description"`
	TrafficQuota            int64    `json:"traffic_quota"`
	MaxRules                int      `json:"max_rules"`
	MaxBandwidth            int      `json:"max_bandwidth"`
	MaxSelfHostedEntryNodes int      `json:"max_self_hosted_entry_nodes"`
	MaxSelfHostedExitNodes  int      `json:"max_self_hosted_exit_nodes"`
	AccessibleNodeLevel     int      `json:"accessible_node_level"`
	TrafficMultiplier       float64  `json:"traffic_multiplier"`
	CustomFeatures          *string  `json:"custom_features"`
	Price                   *float64 `json:"price"`
	DurationDays            *int     `json:"duration_days"`
}

// ListLevelsHandler 列出所有 VIP 等级查询处理器
type ListLevelsHandler struct {
	vipRepo vip.Repository
}

// NewListLevelsHandler 创建列出所有 VIP 等级查询处理器
func NewListLevelsHandler(vipRepo vip.Repository) *ListLevelsHandler {
	return &ListLevelsHandler{
		vipRepo: vipRepo,
	}
}

// Handle 处理列出所有 VIP 等级查询
func (h *ListLevelsHandler) Handle(ctx context.Context, query ListLevelsQuery) (*ListLevelsResult, error) {
	levels, err := h.vipRepo.ListLevels(ctx)
	if err != nil {
		return nil, err
	}

	levelInfos := make([]*VIPLevelInfo, len(levels))
	for i, level := range levels {
		levelInfos[i] = &VIPLevelInfo{
			ID:                      level.ID,
			Level:                   level.Level,
			Name:                    level.Name,
			Description:             level.Description,
			TrafficQuota:            level.TrafficQuota,
			MaxRules:                level.MaxRules,
			MaxBandwidth:            level.MaxBandwidth,
			MaxSelfHostedEntryNodes: level.MaxSelfHostedEntryNodes,
			MaxSelfHostedExitNodes:  level.MaxSelfHostedExitNodes,
			AccessibleNodeLevel:     level.AccessibleNodeLevel,
			TrafficMultiplier:       level.TrafficMultiplier,
			CustomFeatures:          level.CustomFeatures,
			Price:                   level.Price,
			DurationDays:            level.DurationDays,
		}
	}

	return &ListLevelsResult{
		Levels: levelInfos,
		Total:  len(levelInfos),
	}, nil
}
