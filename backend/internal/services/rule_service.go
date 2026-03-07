package services

import (
	"errors"
	"fmt"
	"strings"

	"nodepass-panel/backend/internal/models"
	"nodepass-panel/backend/internal/utils"

	"gorm.io/gorm"
)

var (
	allowedRuleModes = map[string]struct{}{
		"single": {},
		"tunnel": {},
	}

	allowedRuleProtocols = map[string]struct{}{
		"tcp":  {},
		"udp":  {},
		"ws":   {},
		"tls":  {},
		"quic": {},
	}
)

// RuleService 转发规则服务。
type RuleService struct {
	db *gorm.DB
}

// CreateRuleRequest 创建规则请求。
type CreateRuleRequest struct {
	Name        string `json:"name" binding:"required"`
	Mode        string `json:"mode" binding:"required"`
	Protocol    string `json:"protocol" binding:"required"`
	EntryNodeID uint   `json:"entry_node_id" binding:"required"`
	ExitNodeID  *uint  `json:"exit_node_id"`

	TargetHost string `json:"target_host" binding:"required"`
	TargetPort int    `json:"target_port" binding:"required"`
	ListenHost string `json:"listen_host"`
	ListenPort int    `json:"listen_port" binding:"required"`

	InstanceID     *string `json:"instance_id"`
	InstanceStatus *string `json:"instance_status"`
	ConfigJSON     *string `json:"config_json"`
}

// UpdateRuleRequest 更新规则请求。
type UpdateRuleRequest struct {
	Name        *string `json:"name"`
	Mode        *string `json:"mode"`
	Protocol    *string `json:"protocol"`
	EntryNodeID *uint   `json:"entry_node_id"`
	ExitNodeID  *uint   `json:"exit_node_id"`

	TargetHost *string `json:"target_host"`
	TargetPort *int    `json:"target_port"`
	ListenHost *string `json:"listen_host"`
	ListenPort *int    `json:"listen_port"`

	InstanceID     *string `json:"instance_id"`
	InstanceStatus *string `json:"instance_status"`
	ConfigJSON     *string `json:"config_json"`
}

// ListRuleFilters 规则查询过滤条件。
type ListRuleFilters struct {
	Status   string
	Mode     string
	Page     int
	PageSize int
	UserID   *uint
}

// ListRuleResult 规则分页数据。
type ListRuleResult struct {
	List     []models.Rule `json:"list"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// NewRuleService 创建规则服务实例。
func NewRuleService(db *gorm.DB) *RuleService {
	return &RuleService{db: db}
}

// CreateRule 创建转发规则。
func (s *RuleService) CreateRule(userID uint, req *CreateRuleRequest) (*models.Rule, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}
	if userID == 0 {
		return nil, fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}

	input, err := s.normalizeCreateRequest(req)
	if err != nil {
		return nil, err
	}

	if _, err = s.ensureUserExists(userID); err != nil {
		return nil, err
	}

	if err = s.checkRuleQuota(userID); err != nil {
		return nil, err
	}

	if _, err = s.getRuleNodeForOwner(userID, input.EntryNodeID); err != nil {
		return nil, err
	}
	entryNode, err := s.getNodeByID(input.EntryNodeID)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(strings.TrimSpace(entryNode.Status), "offline") {
		return nil, fmt.Errorf("%w: 入口节点离线，无法创建规则", ErrInvalidParams)
	}

	if input.Mode == "tunnel" {
		if input.ExitNodeID == nil {
			return nil, fmt.Errorf("%w: tunnel 模式必须指定 exit_node_id", ErrInvalidParams)
		}
		if _, err = s.getRuleNodeForOwner(userID, *input.ExitNodeID); err != nil {
			return nil, err
		}
		exitNode, err := s.getNodeByID(*input.ExitNodeID)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(strings.TrimSpace(exitNode.Status), "online") {
			return nil, fmt.Errorf("%w: tunnel 模式要求出口节点在线", ErrInvalidParams)
		}
		if err = s.ensureEnabledNodePair(userID, input.EntryNodeID, *input.ExitNodeID); err != nil {
			return nil, err
		}
	}

	if err = s.checkRunningPortConflict(input.EntryNodeID, input.ListenPort, 0); err != nil {
		return nil, err
	}

	rule := &models.Rule{
		UserID:         userID,
		Name:           input.Name,
		Mode:           input.Mode,
		Protocol:       input.Protocol,
		EntryNodeID:    input.EntryNodeID,
		ExitNodeID:     input.ExitNodeID,
		TargetHost:     input.TargetHost,
		TargetPort:     input.TargetPort,
		ListenHost:     input.ListenHost,
		ListenPort:     input.ListenPort,
		Status:         "stopped",
		SyncStatus:     "pending",
		InstanceID:     input.InstanceID,
		InstanceStatus: normalizeOptionalString(input.InstanceStatus),
		ConfigJSON:     normalizeOptionalString(input.ConfigJSON),
		ConfigVersion:  0,
	}

	if err = s.db.Create(rule).Error; err != nil {
		return nil, fmt.Errorf("创建规则失败: %w", err)
	}

	created, err := s.GetRule(userID, rule.ID)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// ListRules 查询规则列表（支持状态与模式过滤，并预加载入口/出口节点）。
func (s *RuleService) ListRules(userID uint, filters ListRuleFilters) (*ListRuleResult, error) {
	page := filters.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filters.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	query := s.db.Model(&models.Rule{}).
		Preload("EntryNode").
		Preload("ExitNode")

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if filters.UserID != nil {
		query = query.Where("user_id = ?", *filters.UserID)
	}
	if status := strings.TrimSpace(filters.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if mode := strings.TrimSpace(filters.Mode); mode != "" {
		query = query.Where("mode = ?", mode)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询规则总数失败: %w", err)
	}

	list := make([]models.Rule, 0, pageSize)
	if err := query.Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询规则列表失败: %w", err)
	}

	return &ListRuleResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetRule 查询规则详情。
func (s *RuleService) GetRule(userID uint, ruleID uint) (*models.Rule, error) {
	if ruleID == 0 {
		return nil, fmt.Errorf("%w: 规则 ID 无效", ErrInvalidParams)
	}

	query := s.db.Model(&models.Rule{}).
		Preload("EntryNode").
		Preload("ExitNode")
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	var rule models.Rule
	if err := query.First(&rule, ruleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 规则不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询规则失败: %w", err)
	}
	return &rule, nil
}

// UpdateRule 更新规则，仅允许 stopped 状态修改。
func (s *RuleService) UpdateRule(userID uint, ruleID uint, req *UpdateRuleRequest) (*models.Rule, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	rule, err := s.GetRule(userID, ruleID)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(rule.Status), "stopped") {
		return nil, fmt.Errorf("%w: 仅 stopped 状态规则允许修改", ErrConflict)
	}

	merged, err := s.mergeUpdateRequest(rule, req)
	if err != nil {
		return nil, err
	}

	if _, err = s.getRuleNodeForOwner(rule.UserID, merged.EntryNodeID); err != nil {
		return nil, err
	}
	entryNode, err := s.getNodeByID(merged.EntryNodeID)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(strings.TrimSpace(entryNode.Status), "offline") {
		return nil, fmt.Errorf("%w: 入口节点离线，无法修改规则", ErrInvalidParams)
	}

	if merged.Mode == "tunnel" {
		if merged.ExitNodeID == nil {
			return nil, fmt.Errorf("%w: tunnel 模式必须指定 exit_node_id", ErrInvalidParams)
		}
		if _, err = s.getRuleNodeForOwner(rule.UserID, *merged.ExitNodeID); err != nil {
			return nil, err
		}
		exitNode, exitErr := s.getNodeByID(*merged.ExitNodeID)
		if exitErr != nil {
			return nil, exitErr
		}
		if !strings.EqualFold(strings.TrimSpace(exitNode.Status), "online") {
			return nil, fmt.Errorf("%w: tunnel 模式要求出口节点在线", ErrInvalidParams)
		}
		if err = s.ensureEnabledNodePair(rule.UserID, merged.EntryNodeID, *merged.ExitNodeID); err != nil {
			return nil, err
		}
	}

	if err = s.checkRunningPortConflict(merged.EntryNodeID, merged.ListenPort, rule.ID); err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"name":            merged.Name,
		"mode":            merged.Mode,
		"protocol":        merged.Protocol,
		"entry_node_id":   merged.EntryNodeID,
		"exit_node_id":    merged.ExitNodeID,
		"target_host":     merged.TargetHost,
		"target_port":     merged.TargetPort,
		"listen_host":     merged.ListenHost,
		"listen_port":     merged.ListenPort,
		"instance_id":     normalizeOptionalString(merged.InstanceID),
		"instance_status": normalizeOptionalString(merged.InstanceStatus),
		"config_json":     normalizeOptionalString(merged.ConfigJSON),
		"sync_status":     "pending",
	}
	if err = s.db.Model(&models.Rule{}).Where("id = ?", rule.ID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新规则失败: %w", err)
	}

	if userID > 0 {
		return s.GetRule(userID, rule.ID)
	}
	return s.GetRule(0, rule.ID)
}

// DeleteRule 删除规则，如果规则运行中则先停止。
func (s *RuleService) DeleteRule(userID uint, ruleID uint) error {
	rule, err := s.GetRule(userID, ruleID)
	if err != nil {
		return err
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %w", tx.Error)
	}

	if strings.EqualFold(strings.TrimSpace(rule.Status), "running") {
		if err = tx.Model(&models.Rule{}).
			Where("id = ?", rule.ID).
			Updates(map[string]interface{}{
				"status":      "stopped",
				"sync_status": "pending",
			}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("停止运行中的规则失败: %w", err)
		}
	}

	if err = tx.Delete(&models.Rule{}, rule.ID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除规则失败: %w", err)
	}

	if err = tx.Model(&models.Node{}).
		Where("id = ?", rule.EntryNodeID).
		UpdateColumn("config_version", gorm.Expr("config_version + ?", 1)).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新入口节点配置版本失败: %w", err)
	}

	return tx.Commit().Error
}

// StartRule 启动规则。
func (s *RuleService) StartRule(userID uint, ruleID uint) (*models.Rule, error) {
	rule, err := s.GetRule(userID, ruleID)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(strings.TrimSpace(rule.Status), "running") {
		return nil, fmt.Errorf("%w: 规则已处于运行状态", ErrConflict)
	}

	if _, err = s.getRuleNodeForOwner(rule.UserID, rule.EntryNodeID); err != nil {
		return nil, err
	}
	entryNode, err := s.getNodeByID(rule.EntryNodeID)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(strings.TrimSpace(entryNode.Status), "offline") {
		return nil, fmt.Errorf("%w: 入口节点离线，无法启动规则", ErrInvalidParams)
	}

	if strings.EqualFold(strings.TrimSpace(rule.Mode), "tunnel") {
		if rule.ExitNodeID == nil {
			return nil, fmt.Errorf("%w: tunnel 模式缺少出口节点", ErrInvalidParams)
		}
		if _, err = s.getRuleNodeForOwner(rule.UserID, *rule.ExitNodeID); err != nil {
			return nil, err
		}
		exitNode, exitErr := s.getNodeByID(*rule.ExitNodeID)
		if exitErr != nil {
			return nil, exitErr
		}
		if !strings.EqualFold(strings.TrimSpace(exitNode.Status), "online") {
			return nil, fmt.Errorf("%w: 出口节点不在线，无法启动规则", ErrInvalidParams)
		}
		if err = s.ensureEnabledNodePair(rule.UserID, rule.EntryNodeID, *rule.ExitNodeID); err != nil {
			return nil, err
		}
	}

	if err = s.checkRunningPortConflict(rule.EntryNodeID, rule.ListenPort, rule.ID); err != nil {
		return nil, err
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("开启事务失败: %w", tx.Error)
	}

	if err = tx.Model(&models.Rule{}).Where("id = ?", rule.ID).Updates(map[string]interface{}{
		"status":      "running",
		"sync_status": "pending",
	}).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新规则状态失败: %w", err)
	}

	if err = tx.Model(&models.Node{}).
		Where("id = ?", rule.EntryNodeID).
		UpdateColumn("config_version", gorm.Expr("config_version + ?", 1)).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新入口节点配置版本失败: %w", err)
	}

	if err = tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	if userID > 0 {
		return s.GetRule(userID, rule.ID)
	}
	return s.GetRule(0, rule.ID)
}

// StopRule 停止规则。
func (s *RuleService) StopRule(userID uint, ruleID uint) (*models.Rule, error) {
	rule, err := s.GetRule(userID, ruleID)
	if err != nil {
		return nil, err
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("开启事务失败: %w", tx.Error)
	}

	if err = tx.Model(&models.Rule{}).Where("id = ?", rule.ID).Updates(map[string]interface{}{
		"status":      "stopped",
		"sync_status": "pending",
	}).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新规则状态失败: %w", err)
	}

	if err = tx.Model(&models.Node{}).
		Where("id = ?", rule.EntryNodeID).
		UpdateColumn("config_version", gorm.Expr("config_version + ?", 1)).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新入口节点配置版本失败: %w", err)
	}

	if err = tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	if userID > 0 {
		return s.GetRule(userID, rule.ID)
	}
	return s.GetRule(0, rule.ID)
}

// RestartRule 重启规则（先停止再启动）。
func (s *RuleService) RestartRule(userID uint, ruleID uint) (*models.Rule, error) {
	if _, err := s.StopRule(userID, ruleID); err != nil {
		return nil, err
	}
	return s.StartRule(userID, ruleID)
}

func (s *RuleService) normalizeCreateRequest(req *CreateRuleRequest) (*CreateRuleRequest, error) {
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if _, ok := allowedRuleModes[mode]; !ok {
		return nil, fmt.Errorf("%w: 不支持的 mode", ErrInvalidParams)
	}

	protocol := strings.ToLower(strings.TrimSpace(req.Protocol))
	if _, ok := allowedRuleProtocols[protocol]; !ok {
		return nil, fmt.Errorf("%w: 不支持的 protocol", ErrInvalidParams)
	}

	name := strings.TrimSpace(req.Name)
	targetHost := strings.TrimSpace(req.TargetHost)
	if name == "" || targetHost == "" {
		return nil, fmt.Errorf("%w: name/target_host 不能为空", ErrInvalidParams)
	}

	// 验证规则名称
	if err := utils.ValidateRuleName(name); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	// 验证目标主机
	if err := utils.ValidateHost(targetHost); err != nil {
		return nil, fmt.Errorf("%w: 目标主机无效: %v", ErrInvalidParams, err)
	}

	// 验证端口
	if err := utils.ValidatePort(req.TargetPort); err != nil {
		return nil, fmt.Errorf("%w: 目标端口无效: %v", ErrInvalidParams, err)
	}
	if err := utils.ValidatePort(req.ListenPort); err != nil {
		return nil, fmt.Errorf("%w: 监听端口无效: %v", ErrInvalidParams, err)
	}

	if req.EntryNodeID == 0 {
		return nil, fmt.Errorf("%w: entry_node_id 无效", ErrInvalidParams)
	}

	listenHost := strings.TrimSpace(req.ListenHost)
	if listenHost == "" {
		listenHost = "0.0.0.0"
	} else {
		// 验证监听主机
		if err := utils.ValidateHost(listenHost); err != nil {
			return nil, fmt.Errorf("%w: 监听主机无效: %v", ErrInvalidParams, err)
		}
	}

	normalized := &CreateRuleRequest{
		Name:           name,
		Mode:           mode,
		Protocol:       protocol,
		EntryNodeID:    req.EntryNodeID,
		ExitNodeID:     req.ExitNodeID,
		TargetHost:     targetHost,
		TargetPort:     req.TargetPort,
		ListenHost:     listenHost,
		ListenPort:     req.ListenPort,
		InstanceID:     normalizeOptionalString(req.InstanceID),
		InstanceStatus: normalizeOptionalString(req.InstanceStatus),
		ConfigJSON:     normalizeOptionalString(req.ConfigJSON),
	}

	if mode == "single" {
		if normalized.ExitNodeID != nil {
			return nil, fmt.Errorf("%w: single 模式下 exit_node_id 必须为空", ErrInvalidParams)
		}
	} else {
		if normalized.ExitNodeID == nil {
			return nil, fmt.Errorf("%w: tunnel 模式下 exit_node_id 必填", ErrInvalidParams)
		}
	}

	return normalized, nil
}

func (s *RuleService) mergeUpdateRequest(rule *models.Rule, req *UpdateRuleRequest) (*CreateRuleRequest, error) {
	mode := rule.Mode
	if req.Mode != nil {
		mode = strings.ToLower(strings.TrimSpace(*req.Mode))
	}

	protocol := rule.Protocol
	if req.Protocol != nil {
		protocol = strings.ToLower(strings.TrimSpace(*req.Protocol))
	}

	entryNodeID := rule.EntryNodeID
	if req.EntryNodeID != nil {
		entryNodeID = *req.EntryNodeID
	}

	exitNodeID := rule.ExitNodeID
	if req.ExitNodeID != nil {
		exitNodeID = req.ExitNodeID
	}
	if mode == "single" {
		exitNodeID = nil
	}

	name := rule.Name
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}

	targetHost := rule.TargetHost
	if req.TargetHost != nil {
		targetHost = strings.TrimSpace(*req.TargetHost)
	}

	targetPort := rule.TargetPort
	if req.TargetPort != nil {
		targetPort = *req.TargetPort
	}

	listenHost := rule.ListenHost
	if req.ListenHost != nil {
		listenHost = strings.TrimSpace(*req.ListenHost)
	}
	if listenHost == "" {
		listenHost = "0.0.0.0"
	}

	listenPort := rule.ListenPort
	if req.ListenPort != nil {
		listenPort = *req.ListenPort
	}

	merged := &CreateRuleRequest{
		Name:           name,
		Mode:           mode,
		Protocol:       protocol,
		EntryNodeID:    entryNodeID,
		ExitNodeID:     exitNodeID,
		TargetHost:     targetHost,
		TargetPort:     targetPort,
		ListenHost:     listenHost,
		ListenPort:     listenPort,
		InstanceID:     rule.InstanceID,
		InstanceStatus: rule.InstanceStatus,
		ConfigJSON:     rule.ConfigJSON,
	}
	if req.InstanceID != nil {
		merged.InstanceID = normalizeOptionalString(req.InstanceID)
	}
	if req.InstanceStatus != nil {
		merged.InstanceStatus = normalizeOptionalString(req.InstanceStatus)
	}
	if req.ConfigJSON != nil {
		merged.ConfigJSON = normalizeOptionalString(req.ConfigJSON)
	}

	return s.normalizeCreateRequest(merged)
}

func (s *RuleService) ensureUserExists(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 用户不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

func (s *RuleService) getNodeByID(nodeID uint) (*models.Node, error) {
	var node models.Node
	if err := s.db.First(&node, nodeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 节点不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询节点失败: %w", err)
	}
	return &node, nil
}

func (s *RuleService) getRuleNodeForOwner(userID uint, nodeID uint) (*models.Node, error) {
	var node models.Node
	if err := s.db.Where("id = ? AND (user_id = ? OR is_public = ?)", nodeID, userID, true).
		First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 节点不存在或不可访问", ErrForbidden)
		}
		return nil, fmt.Errorf("查询节点失败: %w", err)
	}
	return &node, nil
}

func (s *RuleService) ensureEnabledNodePair(userID uint, entryNodeID uint, exitNodeID uint) error {
	var count int64
	if err := s.db.Model(&models.NodePair{}).
		Where("user_id = ? AND entry_node_id = ? AND exit_node_id = ? AND is_enabled = ?",
			userID, entryNodeID, exitNodeID, true).
		Count(&count).Error; err != nil {
		return fmt.Errorf("校验节点配对失败: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("%w: 入口与出口节点不存在可用配对", ErrInvalidParams)
	}
	return nil
}

func (s *RuleService) checkRunningPortConflict(entryNodeID uint, listenPort int, excludeRuleID uint) error {
	query := s.db.Model(&models.Rule{}).
		Where("entry_node_id = ? AND listen_port = ? AND status = ?", entryNodeID, listenPort, "running")
	if excludeRuleID > 0 {
		query = query.Where("id <> ?", excludeRuleID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return fmt.Errorf("校验端口冲突失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("%w: 入口节点监听端口已被运行中规则占用", ErrConflict)
	}
	return nil
}

func (s *RuleService) checkRuleQuota(userID uint) error {
	user, err := s.ensureUserExists(userID)
	if err != nil {
		return err
	}

	var count int64
	if err = s.db.Model(&models.Rule{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return fmt.Errorf("查询规则配额失败: %w", err)
	}
	if user.MaxRules < 0 {
		return nil
	}
	if int(count) >= user.MaxRules {
		return fmt.Errorf("%w: 规则数量已达到上限", ErrQuotaExceeded)
	}
	return nil
}
