package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"nodepass-pro/backend/internal/domain/tunneltemplate"
	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// TunnelTemplateRepository 隧道模板仓储实现
type TunnelTemplateRepository struct {
	db *gorm.DB
}

// NewTunnelTemplateRepository 创建隧道模板仓储
func NewTunnelTemplateRepository(db *gorm.DB) *TunnelTemplateRepository {
	return &TunnelTemplateRepository{db: db}
}

// Create 创建模板
func (r *TunnelTemplateRepository) Create(ctx context.Context, template *tunneltemplate.TunnelTemplate) error {
	model, err := r.toModel(template)
	if err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("创建模板失败: %w", err)
	}

	template.ID = model.ID
	return nil
}

// FindByID 根据 ID 查找模板
func (r *TunnelTemplateRepository) FindByID(ctx context.Context, id uint) (*tunneltemplate.TunnelTemplate, error) {
	var model models.TunnelTemplate
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tunneltemplate.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("查找模板失败: %w", err)
	}

	return r.toDomain(&model)
}

// Update 更新模板
func (r *TunnelTemplateRepository) Update(ctx context.Context, template *tunneltemplate.TunnelTemplate) error {
	model, err := r.toModel(template)
	if err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return fmt.Errorf("更新模板失败: %w", err)
	}

	return nil
}

// Delete 删除模板
func (r *TunnelTemplateRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.TunnelTemplate{}, id).Error; err != nil {
		return fmt.Errorf("删除模板失败: %w", err)
	}
	return nil
}

// List 列表查询
func (r *TunnelTemplateRepository) List(ctx context.Context, filter tunneltemplate.ListFilter) ([]*tunneltemplate.TunnelTemplate, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.TunnelTemplate{})

	// 用户只能看到自己的模板和公开模板
	if filter.UserID > 0 {
		query = query.Where("user_id = ? OR is_public = ?", filter.UserID, true)
	}

	// 协议过滤
	if filter.Protocol != nil && *filter.Protocol != "" {
		query = query.Where("protocol = ?", *filter.Protocol)
	}

	// 公开状态过滤
	if filter.IsPublic != nil {
		query = query.Where("is_public = ?", *filter.IsPublic)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计模板总数失败: %w", err)
	}

	// 分页查询
	var models []*models.TunnelTemplate
	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Order("id DESC").Offset(offset).Limit(filter.PageSize).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("查询模板列表失败: %w", err)
	}

	// 转换为领域对象
	templates := make([]*tunneltemplate.TunnelTemplate, 0, len(models))
	for _, model := range models {
		template, err := r.toDomain(model)
		if err != nil {
			// 跳过无法解析的模板
			continue
		}
		templates = append(templates, template)
	}

	return templates, total, nil
}

// FindByUserAndName 根据用户和名称查找模板
func (r *TunnelTemplateRepository) FindByUserAndName(ctx context.Context, userID uint, name string) (*tunneltemplate.TunnelTemplate, error) {
	var model models.TunnelTemplate
	if err := r.db.WithContext(ctx).Where("user_id = ? AND name = ?", userID, name).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tunneltemplate.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("查找模板失败: %w", err)
	}

	return r.toDomain(&model)
}

// IncrementUsageCount 增加使用次数
func (r *TunnelTemplateRepository) IncrementUsageCount(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Model(&models.TunnelTemplate{}).
		Where("id = ?", id).
		Update("usage_count", gorm.Expr("usage_count + 1")).Error; err != nil {
		return fmt.Errorf("增加使用次数失败: %w", err)
	}
	return nil
}

// toModel 转换为数据库模型
func (r *TunnelTemplateRepository) toModel(template *tunneltemplate.TunnelTemplate) (*models.TunnelTemplate, error) {
	model := &models.TunnelTemplate{
		ID:          template.ID,
		UserID:      template.UserID,
		Name:        template.Name,
		Description: template.Description,
		Protocol:    template.Protocol,
		IsPublic:    template.IsPublic,
		UsageCount:  template.UsageCount,
		CreatedAt:   template.CreatedAt,
		UpdatedAt:   template.UpdatedAt,
	}

	// 序列化配置
	if template.Config != nil {
		configData, err := json.Marshal(r.configToJSON(template.Config))
		if err != nil {
			return nil, fmt.Errorf("序列化配置失败: %w", err)
		}
		model.ConfigJSON = string(configData)
	}

	return model, nil
}

// toDomain 转换为领域对象
func (r *TunnelTemplateRepository) toDomain(model *models.TunnelTemplate) (*tunneltemplate.TunnelTemplate, error) {
	template := &tunneltemplate.TunnelTemplate{
		ID:          model.ID,
		UserID:      model.UserID,
		Name:        model.Name,
		Description: model.Description,
		Protocol:    model.Protocol,
		IsPublic:    model.IsPublic,
		UsageCount:  model.UsageCount,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}

	// 反序列化配置
	if model.ConfigJSON != "" {
		var configData map[string]interface{}
		if err := json.Unmarshal([]byte(model.ConfigJSON), &configData); err != nil {
			return nil, fmt.Errorf("反序列化配置失败: %w", err)
		}
		template.Config = r.jsonToConfig(configData)
	}

	return template, nil
}

// configToJSON 转换配置为 JSON 对象
func (r *TunnelTemplateRepository) configToJSON(config *tunneltemplate.TemplateConfig) map[string]interface{} {
	result := map[string]interface{}{
		"remote_host":            config.RemoteHost,
		"remote_port":            config.RemotePort,
		"load_balance_strategy":  config.LoadBalanceStrategy,
		"ip_type":                config.IPType,
		"enable_proxy_protocol":  config.EnableProxyProtocol,
		"health_check_interval":  config.HealthCheckInterval,
		"health_check_timeout":   config.HealthCheckTimeout,
	}

	if config.ListenHost != nil {
		result["listen_host"] = *config.ListenHost
	}
	if config.ListenPort != nil {
		result["listen_port"] = *config.ListenPort
	}

	// 转换 ForwardTargets
	if len(config.ForwardTargets) > 0 {
		targets := make([]map[string]interface{}, len(config.ForwardTargets))
		for i, target := range config.ForwardTargets {
			targets[i] = map[string]interface{}{
				"host":   target.Host,
				"port":   target.Port,
				"weight": target.Weight,
			}
		}
		result["forward_targets"] = targets
	}

	if config.ProtocolConfig != nil {
		result["protocol_config"] = config.ProtocolConfig
	}

	return result
}

// jsonToConfig 转换 JSON 对象为配置
func (r *TunnelTemplateRepository) jsonToConfig(data map[string]interface{}) *tunneltemplate.TemplateConfig {
	config := &tunneltemplate.TemplateConfig{}

	if v, ok := data["listen_host"].(string); ok {
		config.ListenHost = &v
	}
	if v, ok := data["listen_port"].(float64); ok {
		port := int(v)
		config.ListenPort = &port
	}
	if v, ok := data["remote_host"].(string); ok {
		config.RemoteHost = v
	}
	if v, ok := data["remote_port"].(float64); ok {
		config.RemotePort = int(v)
	}
	if v, ok := data["load_balance_strategy"].(string); ok {
		config.LoadBalanceStrategy = v
	}
	if v, ok := data["ip_type"].(string); ok {
		config.IPType = v
	}
	if v, ok := data["enable_proxy_protocol"].(bool); ok {
		config.EnableProxyProtocol = v
	}
	if v, ok := data["health_check_interval"].(float64); ok {
		config.HealthCheckInterval = int(v)
	}
	if v, ok := data["health_check_timeout"].(float64); ok {
		config.HealthCheckTimeout = int(v)
	}

	// 转换 ForwardTargets
	if targets, ok := data["forward_targets"].([]interface{}); ok {
		config.ForwardTargets = make([]tunneltemplate.ForwardTarget, 0, len(targets))
		for _, t := range targets {
			if targetMap, ok := t.(map[string]interface{}); ok {
				target := tunneltemplate.ForwardTarget{}
				if host, ok := targetMap["host"].(string); ok {
					target.Host = host
				}
				if port, ok := targetMap["port"].(float64); ok {
					target.Port = int(port)
				}
				if weight, ok := targetMap["weight"].(float64); ok {
					target.Weight = int(weight)
				}
				config.ForwardTargets = append(config.ForwardTargets, target)
			}
		}
	}

	if v, ok := data["protocol_config"].(map[string]interface{}); ok {
		config.ProtocolConfig = v
	}

	return config
}
