package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"nodepass-panel/backend/internal/models"
	"nodepass-panel/backend/internal/utils"

	"gorm.io/gorm"
)

// NodeService 节点管理服务。
type NodeService struct {
	db *gorm.DB
}

// CreateNodeRequest 创建节点请求。
type CreateNodeRequest struct {
	Name              string   `json:"name" binding:"required"`
	Host              string   `json:"host" binding:"required"`
	Port              int      `json:"port" binding:"required"`
	Region            *string  `json:"region"`
	IsSelfHosted      bool     `json:"is_self_hosted"`
	IsPublic          bool     `json:"is_public"`
	TrafficMultiplier *float64 `json:"traffic_multiplier"`
	Description       *string  `json:"description"`
	PanelURL          string   `json:"panel_url"`
	HubURL            string   `json:"hub_url"`
}

// CreateNodeResult 创建节点返回数据。
type CreateNodeResult struct {
	Node           *models.Node `json:"node"`
	Token          string       `json:"token"`
	InstallCommand string       `json:"install_command"`
}

// ListNodeFilters 节点列表过滤条件。
type ListNodeFilters struct {
	Status       string
	IsSelfHosted *bool
	IsPublic     *bool
	Region       string
	Page         int
	PageSize     int
	UserID       *uint
}

// ListNodeResult 节点分页返回。
type ListNodeResult struct {
	List     []models.Node `json:"list"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// UpdateNodeRequest 更新节点请求。
type UpdateNodeRequest struct {
	Name              *string  `json:"name"`
	Status            *string  `json:"status"`
	Host              *string  `json:"host"`
	Port              *int     `json:"port"`
	Region            *string  `json:"region"`
	IsSelfHosted      *bool    `json:"is_self_hosted"`
	IsPublic          *bool    `json:"is_public"`
	TrafficMultiplier *float64 `json:"traffic_multiplier"`
	Description       *string  `json:"description"`
}

// NodeQuotaInfo 节点配额信息。
type NodeQuotaInfo struct {
	UserID                   uint  `json:"user_id"`
	VipLevel                 int   `json:"vip_level"`
	MaxSelfHostedEntryNodes  int   `json:"max_self_hosted_entry_nodes"`
	MaxSelfHostedExitNodes   int   `json:"max_self_hosted_exit_nodes"`
	UsedSelfHostedNodes      int64 `json:"used_self_hosted_nodes"`
	TotalSelfHostedNodeQuota int   `json:"total_self_hosted_node_quota"`
	RemainingSelfHostedQuota int   `json:"remaining_self_hosted_quota"`
}

// NewNodeService 创建节点服务实例。
func NewNodeService(db *gorm.DB) *NodeService {
	return &NodeService{db: db}
}

// CreateNode 创建节点。
func (s *NodeService) CreateNode(userID uint, req *CreateNodeRequest) (*CreateNodeResult, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}
	if userID == 0 {
		return nil, fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}

	name := strings.TrimSpace(req.Name)
	host := strings.TrimSpace(req.Host)
	if name == "" || host == "" {
		return nil, fmt.Errorf("%w: name/host 参数无效", ErrInvalidParams)
	}

	// 验证节点名称
	if err := utils.ValidateNodeName(name); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	// 验证主机地址
	if err := utils.ValidateHost(host); err != nil {
		return nil, fmt.Errorf("%w: 主机地址无效: %v", ErrInvalidParams, err)
	}

	// 验证端口
	if err := utils.ValidatePort(req.Port); err != nil {
		return nil, fmt.Errorf("%w: 端口无效: %v", ErrInvalidParams, err)
	}

	if req.IsSelfHosted {
		if err := s.checkSelfHostedQuota(userID); err != nil {
			return nil, err
		}
	}

	if _, err := s.getUserByID(userID); err != nil {
		return nil, err
	}

	trafficMultiplier := 1.0
	if req.TrafficMultiplier != nil {
		if *req.TrafficMultiplier <= 0 {
			return nil, fmt.Errorf("%w: traffic_multiplier 必须大于 0", ErrInvalidParams)
		}
		trafficMultiplier = *req.TrafficMultiplier
	}

	var createdNode *models.Node
	var plainToken string
	for attempt := 0; attempt < 5; attempt++ {
		token, err := generateNodeToken()
		if err != nil {
			return nil, fmt.Errorf("生成节点 token 失败: %w", err)
		}

		node := &models.Node{
			UserID:            userID,
			Name:              name,
			Status:            "offline",
			Host:              host,
			Port:              req.Port,
			Region:            normalizeOptionalString(req.Region),
			IsSelfHosted:      req.IsSelfHosted,
			IsPublic:          req.IsPublic,
			TrafficMultiplier: trafficMultiplier,
			TokenHash:         hashTokenSHA256(token),
			ConfigVersion:     0,
			Description:       normalizeOptionalString(req.Description),
		}

		if err := s.db.Create(node).Error; err != nil {
			if isDuplicateKeyError(err) {
				continue
			}
			return nil, fmt.Errorf("创建节点失败: %w", err)
		}

		createdNode = node
		plainToken = token
		break
	}
	if createdNode == nil {
		return nil, fmt.Errorf("创建节点失败: token 冲突过多")
	}

	panelHost := extractPanelHost(req.PanelURL)
	hubURL := normalizeURL(req.HubURL)
	if hubURL == "" {
		hubURL = normalizeURL(req.PanelURL)
	}
	if hubURL == "" && panelHost != "" {
		hubURL = "https://" + panelHost
	}
	if panelHost == "" {
		panelHost = "localhost"
	}
	if hubURL == "" {
		hubURL = "http://localhost:8080"
	}

	installCmd := fmt.Sprintf(
		"curl -fsSL https://%s/install.sh | bash -s -- --hub-url %s --token %s",
		panelHost, hubURL, plainToken,
	)

	return &CreateNodeResult{
		Node:           createdNode,
		Token:          plainToken,
		InstallCommand: installCmd,
	}, nil
}

// ListNodes 查询节点列表。
func (s *NodeService) ListNodes(userID uint, filters ListNodeFilters) (*ListNodeResult, error) {
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

	query := s.db.Model(&models.Node{})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if filters.UserID != nil {
		query = query.Where("user_id = ?", *filters.UserID)
	}
	if status := strings.TrimSpace(filters.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if filters.IsSelfHosted != nil {
		query = query.Where("is_self_hosted = ?", *filters.IsSelfHosted)
	}
	if filters.IsPublic != nil {
		query = query.Where("is_public = ?", *filters.IsPublic)
	}
	if region := strings.TrimSpace(filters.Region); region != "" {
		query = query.Where("region = ?", region)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询节点总数失败: %w", err)
	}

	nodes := make([]models.Node, 0, pageSize)
	if err := query.
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&nodes).Error; err != nil {
		return nil, fmt.Errorf("查询节点列表失败: %w", err)
	}

	return &ListNodeResult{
		List:     nodes,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetNode 查询节点详情。
func (s *NodeService) GetNode(userID uint, nodeID uint) (*models.Node, error) {
	if nodeID == 0 {
		return nil, fmt.Errorf("%w: 节点 ID 无效", ErrInvalidParams)
	}

	var node models.Node
	query := s.db.Model(&models.Node{})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if err := query.First(&node, nodeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 节点不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询节点失败: %w", err)
	}
	return &node, nil
}

// UpdateNode 更新节点。
func (s *NodeService) UpdateNode(userID uint, nodeID uint, req *UpdateNodeRequest) (*models.Node, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	node, err := s.GetNode(userID, nodeID)
	if err != nil {
		return nil, err
	}

	if req.IsSelfHosted != nil && *req.IsSelfHosted && !node.IsSelfHosted {
		if err = s.checkSelfHostedQuota(node.UserID); err != nil {
			return nil, err
		}
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name 不能为空", ErrInvalidParams)
		}
		updates["name"] = name
	}
	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if status == "" {
			return nil, fmt.Errorf("%w: status 不能为空", ErrInvalidParams)
		}
		updates["status"] = status
	}
	if req.Host != nil {
		host := strings.TrimSpace(*req.Host)
		if host == "" {
			return nil, fmt.Errorf("%w: host 不能为空", ErrInvalidParams)
		}
		updates["host"] = host
	}
	if req.Port != nil {
		if *req.Port <= 0 {
			return nil, fmt.Errorf("%w: port 无效", ErrInvalidParams)
		}
		updates["port"] = *req.Port
	}
	if req.Region != nil {
		updates["region"] = normalizeOptionalString(req.Region)
	}
	if req.IsSelfHosted != nil {
		updates["is_self_hosted"] = *req.IsSelfHosted
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}
	if req.TrafficMultiplier != nil {
		if *req.TrafficMultiplier <= 0 {
			return nil, fmt.Errorf("%w: traffic_multiplier 必须大于 0", ErrInvalidParams)
		}
		updates["traffic_multiplier"] = *req.TrafficMultiplier
	}
	if req.Description != nil {
		updates["description"] = normalizeOptionalString(req.Description)
	}

	if len(updates) == 0 {
		return node, nil
	}

	if err = s.db.Model(&models.Node{}).Where("id = ?", node.ID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新节点失败: %w", err)
	}

	return s.GetNode(userID, nodeID)
}

// DeleteNode 删除节点并级联删除关联节点配对与规则。
func (s *NodeService) DeleteNode(userID uint, nodeID uint) error {
	node, err := s.GetNode(userID, nodeID)
	if err != nil {
		return err
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %w", tx.Error)
	}

	if err = tx.Where("entry_node_id = ? OR exit_node_id = ?", node.ID, node.ID).
		Delete(&models.NodePair{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除关联节点配对失败: %w", err)
	}
	if err = tx.Where("entry_node_id = ? OR exit_node_id = ?", node.ID, node.ID).
		Delete(&models.Rule{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除关联规则失败: %w", err)
	}
	if err = tx.Where("node_id = ?", node.ID).Delete(&models.NodeConfig{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除关联节点配置失败: %w", err)
	}
	if err = tx.Delete(&models.Node{}, node.ID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除节点失败: %w", err)
	}

	return tx.Commit().Error
}

// GetQuota 获取节点配额使用情况。
func (s *NodeService) GetQuota(userID uint) (*NodeQuotaInfo, error) {
	if userID == 0 {
		return nil, fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}

	user, err := s.getUserByID(userID)
	if err != nil {
		return nil, err
	}

	var used int64
	if err = s.db.Model(&models.Node{}).
		Where("user_id = ? AND is_self_hosted = ?", userID, true).
		Count(&used).Error; err != nil {
		return nil, fmt.Errorf("查询配额使用量失败: %w", err)
	}

	totalQuota := user.MaxSelfHostedEntryNodes + user.MaxSelfHostedExitNodes
	remaining := totalQuota - int(used)
	if remaining < 0 {
		remaining = 0
	}

	return &NodeQuotaInfo{
		UserID:                   user.ID,
		VipLevel:                 user.VipLevel,
		MaxSelfHostedEntryNodes:  user.MaxSelfHostedEntryNodes,
		MaxSelfHostedExitNodes:   user.MaxSelfHostedExitNodes,
		UsedSelfHostedNodes:      used,
		TotalSelfHostedNodeQuota: totalQuota,
		RemainingSelfHostedQuota: remaining,
	}, nil
}

func (s *NodeService) getUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 用户不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

func (s *NodeService) checkSelfHostedQuota(userID uint) error {
	user, err := s.getUserByID(userID)
	if err != nil {
		return err
	}

	totalQuota := user.MaxSelfHostedEntryNodes + user.MaxSelfHostedExitNodes
	if totalQuota <= 0 {
		return fmt.Errorf("%w: 当前 VIP 配额不允许创建自托管节点", ErrQuotaExceeded)
	}

	var used int64
	if err = s.db.Model(&models.Node{}).
		Where("user_id = ? AND is_self_hosted = ?", userID, true).
		Count(&used).Error; err != nil {
		return fmt.Errorf("查询自托管节点配额失败: %w", err)
	}
	if int(used) >= totalQuota {
		return fmt.Errorf("%w: 自托管节点配额已用尽", ErrQuotaExceeded)
	}

	return nil
}

func generateNodeToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashTokenSHA256(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "duplicate") ||
		strings.Contains(errText, "unique constraint") ||
		strings.Contains(errText, "duplicated key")
}

func extractPanelHost(panelURL string) string {
	panelURL = strings.TrimSpace(panelURL)
	if panelURL == "" {
		return ""
	}
	if !strings.Contains(panelURL, "://") {
		panelURL = "https://" + panelURL
	}
	parsed, err := url.Parse(panelURL)
	if err != nil {
		return strings.Trim(strings.TrimPrefix(strings.TrimPrefix(panelURL, "https://"), "http://"), "/")
	}
	return strings.TrimSpace(parsed.Host)
}

func normalizeURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	if !strings.Contains(rawURL, "://") {
		return "https://" + strings.Trim(rawURL, "/")
	}
	return strings.TrimRight(rawURL, "/")
}
