package services

import (
	"errors"
	"fmt"
	"strings"

	"nodepass-panel/backend/internal/models"

	"gorm.io/gorm"
)

// NodePairService 节点配对服务。
type NodePairService struct {
	db *gorm.DB
}

// CreateNodePairRequest 创建节点配对请求。
type CreateNodePairRequest struct {
	EntryNodeID uint    `json:"entry_node_id" binding:"required"`
	ExitNodeID  uint    `json:"exit_node_id" binding:"required"`
	Name        *string `json:"name"`
	IsEnabled   *bool   `json:"is_enabled"`
	Description *string `json:"description"`
}

// UpdateNodePairRequest 更新节点配对请求。
type UpdateNodePairRequest struct {
	EntryNodeID *uint   `json:"entry_node_id"`
	ExitNodeID  *uint   `json:"exit_node_id"`
	Name        *string `json:"name"`
	IsEnabled   *bool   `json:"is_enabled"`
	Description *string `json:"description"`
}

// NewNodePairService 创建节点配对服务实例。
func NewNodePairService(db *gorm.DB) *NodePairService {
	return &NodePairService{db: db}
}

// CreatePair 创建节点配对。
func (s *NodePairService) CreatePair(userID uint, req *CreateNodePairRequest) (*models.NodePair, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}
	if userID == 0 {
		return nil, fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}
	if req.EntryNodeID == 0 || req.ExitNodeID == 0 {
		return nil, fmt.Errorf("%w: entry_node_id/exit_node_id 必填", ErrInvalidParams)
	}
	if req.EntryNodeID == req.ExitNodeID {
		return nil, fmt.Errorf("%w: 入口和出口节点不能相同", ErrInvalidParams)
	}

	if _, err := s.getAccessibleNode(userID, req.EntryNodeID); err != nil {
		return nil, err
	}
	if _, err := s.getAccessibleNode(userID, req.ExitNodeID); err != nil {
		return nil, err
	}

	if err := s.ensurePairNotExists(userID, req.EntryNodeID, req.ExitNodeID, 0); err != nil {
		return nil, err
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	pair := &models.NodePair{
		UserID:      userID,
		EntryNodeID: req.EntryNodeID,
		ExitNodeID:  req.ExitNodeID,
		Name:        normalizeOptionalString(req.Name),
		IsEnabled:   isEnabled,
		Description: normalizeOptionalString(req.Description),
	}
	if err := s.db.Create(pair).Error; err != nil {
		if isDuplicateKeyError(err) {
			return nil, fmt.Errorf("%w: 相同节点配对已存在", ErrConflict)
		}
		return nil, fmt.Errorf("创建节点配对失败: %w", err)
	}

	return s.getPairByID(userID, pair.ID)
}

// ListPairs 查询节点配对列表（支持预加载入口/出口节点）。
func (s *NodePairService) ListPairs(userID uint) ([]models.NodePair, error) {
	query := s.db.Model(&models.NodePair{}).
		Preload("EntryNode").
		Preload("ExitNode")

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	pairs := make([]models.NodePair, 0)
	if err := query.Order("id DESC").Find(&pairs).Error; err != nil {
		return nil, fmt.Errorf("查询节点配对失败: %w", err)
	}
	return pairs, nil
}

// UpdatePair 更新节点配对。
func (s *NodePairService) UpdatePair(userID uint, pairID uint, req *UpdateNodePairRequest) (*models.NodePair, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}
	pair, err := s.getPairByID(userID, pairID)
	if err != nil {
		return nil, err
	}

	entryNodeID := pair.EntryNodeID
	exitNodeID := pair.ExitNodeID
	if req.EntryNodeID != nil {
		entryNodeID = *req.EntryNodeID
	}
	if req.ExitNodeID != nil {
		exitNodeID = *req.ExitNodeID
	}

	if entryNodeID == 0 || exitNodeID == 0 {
		return nil, fmt.Errorf("%w: entry_node_id/exit_node_id 不能为空", ErrInvalidParams)
	}
	if entryNodeID == exitNodeID {
		return nil, fmt.Errorf("%w: 入口和出口节点不能相同", ErrInvalidParams)
	}

	accessUserID := pair.UserID
	if userID == 0 {
		accessUserID = 0
	}

	if req.EntryNodeID != nil {
		if _, err = s.getAccessibleNode(accessUserID, entryNodeID); err != nil {
			return nil, err
		}
	}
	if req.ExitNodeID != nil {
		if _, err = s.getAccessibleNode(accessUserID, exitNodeID); err != nil {
			return nil, err
		}
	}

	if err = s.ensurePairNotExists(pair.UserID, entryNodeID, exitNodeID, pair.ID); err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"entry_node_id": entryNodeID,
		"exit_node_id":  exitNodeID,
	}
	if req.Name != nil {
		updates["name"] = normalizeOptionalString(req.Name)
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.Description != nil {
		updates["description"] = normalizeOptionalString(req.Description)
	}

	if err = s.db.Model(&models.NodePair{}).Where("id = ?", pair.ID).Updates(updates).Error; err != nil {
		if isDuplicateKeyError(err) {
			return nil, fmt.Errorf("%w: 相同节点配对已存在", ErrConflict)
		}
		return nil, fmt.Errorf("更新节点配对失败: %w", err)
	}

	return s.getPairByID(userID, pairID)
}

// DeletePair 删除节点配对。
func (s *NodePairService) DeletePair(userID uint, pairID uint) error {
	pair, err := s.getPairByID(userID, pairID)
	if err != nil {
		return err
	}
	if err = s.db.Delete(&models.NodePair{}, pair.ID).Error; err != nil {
		return fmt.Errorf("删除节点配对失败: %w", err)
	}
	return nil
}

// TogglePair 切换节点配对启用状态。
func (s *NodePairService) TogglePair(userID uint, pairID uint) (*models.NodePair, error) {
	pair, err := s.getPairByID(userID, pairID)
	if err != nil {
		return nil, err
	}

	newStatus := !pair.IsEnabled
	if err = s.db.Model(&models.NodePair{}).Where("id = ?", pair.ID).Update("is_enabled", newStatus).Error; err != nil {
		return nil, fmt.Errorf("切换节点配对状态失败: %w", err)
	}

	return s.getPairByID(userID, pairID)
}

// FindByNodes 根据入口/出口节点查询配对，用于规则创建时校验关系。
func (s *NodePairService) FindByNodes(entryNodeID uint, exitNodeID uint) (*models.NodePair, error) {
	if entryNodeID == 0 || exitNodeID == 0 {
		return nil, fmt.Errorf("%w: 节点 ID 无效", ErrInvalidParams)
	}

	var pair models.NodePair
	if err := s.db.Preload("EntryNode").Preload("ExitNode").
		Where("entry_node_id = ? AND exit_node_id = ?", entryNodeID, exitNodeID).
		First(&pair).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 节点配对不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询节点配对失败: %w", err)
	}
	return &pair, nil
}

func (s *NodePairService) getPairByID(userID uint, pairID uint) (*models.NodePair, error) {
	if pairID == 0 {
		return nil, fmt.Errorf("%w: 节点配对 ID 无效", ErrInvalidParams)
	}

	query := s.db.Model(&models.NodePair{}).
		Preload("EntryNode").
		Preload("ExitNode")

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	var pair models.NodePair
	if err := query.First(&pair, pairID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 节点配对不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询节点配对失败: %w", err)
	}
	return &pair, nil
}

func (s *NodePairService) getAccessibleNode(userID uint, nodeID uint) (*models.Node, error) {
	if nodeID == 0 {
		return nil, fmt.Errorf("%w: 节点 ID 无效", ErrInvalidParams)
	}

	query := s.db.Model(&models.Node{}).Where("id = ?", nodeID)
	if userID > 0 {
		isAdmin, err := s.isAdminUser(userID)
		if err != nil {
			return nil, err
		}
		if !isAdmin {
			query = query.Where("(user_id = ? OR is_public = ?)", userID, true)
		}
	}

	var node models.Node
	if err := query.First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 节点不存在或不可访问", ErrForbidden)
		}
		return nil, fmt.Errorf("查询节点失败: %w", err)
	}
	return &node, nil
}

func (s *NodePairService) isAdminUser(userID uint) (bool, error) {
	var user models.User
	if err := s.db.Select("id", "role").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, fmt.Errorf("%w: 用户不存在", ErrNotFound)
		}
		return false, fmt.Errorf("查询用户失败: %w", err)
	}
	return strings.EqualFold(strings.TrimSpace(user.Role), "admin"), nil
}

func (s *NodePairService) ensurePairNotExists(userID uint, entryNodeID uint, exitNodeID uint, excludeID uint) error {
	query := s.db.Model(&models.NodePair{}).
		Where("user_id = ? AND entry_node_id = ? AND exit_node_id = ?", userID, entryNodeID, exitNodeID)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return fmt.Errorf("校验节点配对唯一性失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("%w: 相同节点配对已存在", ErrConflict)
	}
	return nil
}

func sanitizeOptionalName(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
