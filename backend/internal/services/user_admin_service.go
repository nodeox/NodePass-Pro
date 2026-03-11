package services

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// UserAdminService 用户管理服务（管理员）。
//
// Deprecated: 此服务已被重构为 DDD 架构。
// 新代码请使用以下模块：
//   - Commands: internal/application/user/commands (CreateUserHandler)
//   - Queries: internal/application/user/queries (GetUserHandler)
//   - Repository: internal/infrastructure/persistence/postgres/user_repository.go
//   - Cache: internal/infrastructure/cache/user_cache.go
// 通过依赖注入容器获取: container.CreateUserHandler, container.GetUserHandler 等
// 此服务将在所有旧代码迁移完成后删除。
type UserAdminService struct {
	db *gorm.DB
}

// UserListFilters 用户列表过滤条件。
type UserListFilters struct {
	Role     string
	Status   string
	VIPLevel *int
	Keyword  string
	Page     int
	PageSize int
}

// UserListResult 用户分页结果。
type UserListResult struct {
	List     []models.User `json:"list"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// UserRoleInfo 用户角色信息。
type UserRoleInfo struct {
	ID          uint    `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsSystem    bool    `json:"is_system"`
	IsEnabled   bool    `json:"is_enabled"`
}

// UserDetailStats 用户详情统计。
type UserDetailStats struct {
	TunnelCount        int64 `json:"tunnel_count"`
	RunningTunnelCount int64 `json:"running_tunnel_count"`
	NodeGroupCount     int64 `json:"node_group_count"`
	NodeInstanceCount  int64 `json:"node_instance_count"`
	ActiveSessionCount int64 `json:"active_session_count"`
	TotalTrafficIn     int64 `json:"total_traffic_in"`
	TotalTrafficOut    int64 `json:"total_traffic_out"`
}

// UserSessionSummary 用户会话摘要。
type UserSessionSummary struct {
	ID         uint       `json:"id"`
	IPAddress  string     `json:"ip_address"`
	UserAgent  string     `json:"user_agent"`
	IsRevoked  bool       `json:"is_revoked"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// UserActivitySummary 用户活动摘要。
type UserActivitySummary struct {
	ID           uint      `json:"id"`
	Action       string    `json:"action"`
	ResourceType *string   `json:"resource_type"`
	ResourceID   *uint     `json:"resource_id"`
	Details      *string   `json:"details"`
	IPAddress    *string   `json:"ip_address"`
	CreatedAt    time.Time `json:"created_at"`
	UserAgent    *string   `json:"user_agent"`
}

// UserTunnelSummary 用户隧道摘要。
type UserTunnelSummary struct {
	ID           uint                `json:"id"`
	Name         string              `json:"name"`
	Protocol     string              `json:"protocol"`
	Status       models.TunnelStatus `json:"status"`
	EntryGroupID uint                `json:"entry_group_id"`
	ExitGroupID  *uint               `json:"exit_group_id"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

// UserNodeGroupSummary 用户节点组摘要。
type UserNodeGroupSummary struct {
	ID        uint                 `json:"id"`
	Name      string               `json:"name"`
	Type      models.NodeGroupType `json:"type"`
	IsEnabled bool                 `json:"is_enabled"`
	UpdatedAt time.Time            `json:"updated_at"`
}

// UserDetailResult 用户详情结果。
type UserDetailResult struct {
	User             *models.User           `json:"user"`
	Role             *UserRoleInfo          `json:"role"`
	Permissions      []string               `json:"permissions"`
	Stats            UserDetailStats        `json:"stats"`
	Sessions         []UserSessionSummary   `json:"sessions"`
	RecentActivities []UserActivitySummary  `json:"recent_activities"`
	RecentTunnels    []UserTunnelSummary    `json:"recent_tunnels"`
	RecentNodeGroups []UserNodeGroupSummary `json:"recent_node_groups"`
}

// NewUserAdminService 创建用户管理服务。
func NewUserAdminService(db *gorm.DB) *UserAdminService {
	return &UserAdminService{db: db}
}

// ListUsers 查询用户列表。
func (s *UserAdminService) ListUsers(adminUserID uint, filters UserListFilters) (*UserListResult, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}

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

	query := s.db.Model(&models.User{})

	if role := normalizeRoleCode(filters.Role); role != "" {
		query = query.Where("role = ?", role)
	}
	if status := normalizeUserStatus(filters.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if filters.VIPLevel != nil {
		query = query.Where("vip_level = ?", *filters.VIPLevel)
	}
	if keyword := strings.TrimSpace(filters.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("username LIKE ? OR email LIKE ?", like, like)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询用户总数失败: %w", err)
	}

	items := make([]models.User, 0, pageSize)
	if err := query.
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}

	return &UserListResult{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetUser 获取单个用户详情（基础信息）。
func (s *UserAdminService) GetUser(adminUserID uint, targetUserID uint) (*models.User, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}
	if targetUserID == 0 {
		return nil, fmt.Errorf("%w: 目标用户 ID 无效", ErrInvalidParams)
	}

	return s.getUserByID(targetUserID)
}

// GetUserDetail 获取用户增强详情。
func (s *UserAdminService) GetUserDetail(adminUserID uint, targetUserID uint) (*UserDetailResult, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}
	if targetUserID == 0 {
		return nil, fmt.Errorf("%w: 目标用户 ID 无效", ErrInvalidParams)
	}

	user, err := s.getUserByID(targetUserID)
	if err != nil {
		return nil, err
	}

	roleInfo, err := s.getRoleInfoByCode(user.Role)
	if err != nil {
		return nil, err
	}

	permissions, err := s.getMergedPermissions(targetUserID, user.Role)
	if err != nil {
		return nil, err
	}

	stats, err := s.getUserDetailStats(targetUserID)
	if err != nil {
		return nil, err
	}

	sessions, err := s.getRecentSessions(targetUserID, 10)
	if err != nil {
		return nil, err
	}

	activities, err := s.getRecentActivities(targetUserID, 20)
	if err != nil {
		return nil, err
	}

	tunnels, err := s.getRecentTunnels(targetUserID, 10)
	if err != nil {
		return nil, err
	}

	nodeGroups, err := s.getRecentNodeGroups(targetUserID, 10)
	if err != nil {
		return nil, err
	}

	return &UserDetailResult{
		User:             user,
		Role:             roleInfo,
		Permissions:      permissions,
		Stats:            stats,
		Sessions:         sessions,
		RecentActivities: activities,
		RecentTunnels:    tunnels,
		RecentNodeGroups: nodeGroups,
	}, nil
}

// UpdateUserRole 更新用户角色。
func (s *UserAdminService) UpdateUserRole(adminUserID uint, targetUserID uint, role string) (*models.User, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}
	if targetUserID == 0 {
		return nil, fmt.Errorf("%w: 目标用户 ID 无效", ErrInvalidParams)
	}

	normalizedRole := normalizeRoleCode(role)
	if normalizedRole == "" {
		return nil, fmt.Errorf("%w: role 不能为空", ErrInvalidParams)
	}

	var roleRecord models.Role
	if err := s.db.Where("code = ? AND is_enabled = ?", normalizedRole, true).First(&roleRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 目标角色不存在或已禁用", ErrInvalidParams)
		}
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}

	user, err := s.getUserByID(targetUserID)
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(strings.TrimSpace(user.Role), "admin") && normalizedRole != "admin" {
		var adminCount int64
		if err := s.db.Model(&models.User{}).Where("role = ?", "admin").Count(&adminCount).Error; err != nil {
			return nil, fmt.Errorf("查询管理员数量失败: %w", err)
		}
		if adminCount <= 1 {
			return nil, fmt.Errorf("%w: 至少保留一个管理员账户", ErrConflict)
		}
	}

	if err := s.db.Model(&models.User{}).
		Where("id = ?", targetUserID).
		Update("role", normalizedRole).Error; err != nil {
		return nil, fmt.Errorf("更新用户角色失败: %w", err)
	}

	return s.getUserByID(targetUserID)
}

// UpdateUserStatus 更新用户状态。
func (s *UserAdminService) UpdateUserStatus(adminUserID uint, targetUserID uint, status string) (*models.User, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}
	if targetUserID == 0 {
		return nil, fmt.Errorf("%w: 目标用户 ID 无效", ErrInvalidParams)
	}

	normalizedStatus := normalizeUserStatus(status)
	if normalizedStatus == "" {
		return nil, fmt.Errorf("%w: status 仅支持 normal/paused/banned/overlimit", ErrInvalidParams)
	}

	if _, err := s.getUserByID(targetUserID); err != nil {
		return nil, err
	}

	if err := s.db.Model(&models.User{}).
		Where("id = ?", targetUserID).
		Update("status", normalizedStatus).Error; err != nil {
		return nil, fmt.Errorf("更新用户状态失败: %w", err)
	}

	return s.getUserByID(targetUserID)
}

func (s *UserAdminService) getUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 用户不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

func (s *UserAdminService) getRoleInfoByCode(roleCode string) (*UserRoleInfo, error) {
	roleCode = normalizeRoleCode(roleCode)
	if roleCode == "" {
		return nil, nil
	}

	var role models.Role
	if err := s.db.Where("code = ?", roleCode).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询角色信息失败: %w", err)
	}

	return &UserRoleInfo{
		ID:          role.ID,
		Code:        role.Code,
		Name:        role.Name,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		IsEnabled:   role.IsEnabled,
	}, nil
}

func (s *UserAdminService) getMergedPermissions(userID uint, roleCode string) ([]string, error) {
	set := make(map[string]struct{}, 16)

	var userPermissions []string
	if err := s.db.Model(&models.UserPermission{}).
		Where("user_id = ?", userID).
		Pluck("permission", &userPermissions).Error; err != nil {
		return nil, fmt.Errorf("查询用户权限失败: %w", err)
	}
	for _, permission := range userPermissions {
		permission = strings.TrimSpace(permission)
		if permission != "" {
			set[permission] = struct{}{}
		}
	}

	roleCode = normalizeRoleCode(roleCode)
	if roleCode != "" {
		var rolePermissions []string
		if err := s.db.Table("role_permissions AS rp").
			Select("rp.permission").
			Joins("JOIN roles r ON r.id = rp.role_id").
			Where("r.code = ? AND r.is_enabled = ?", roleCode, true).
			Pluck("rp.permission", &rolePermissions).Error; err != nil {
			return nil, fmt.Errorf("查询角色权限失败: %w", err)
		}
		for _, permission := range rolePermissions {
			permission = strings.TrimSpace(permission)
			if permission != "" {
				set[permission] = struct{}{}
			}
		}
	}

	permissions := make([]string, 0, len(set))
	for permission := range set {
		permissions = append(permissions, permission)
	}
	sort.Strings(permissions)
	return permissions, nil
}

func (s *UserAdminService) getUserDetailStats(userID uint) (UserDetailStats, error) {
	stats := UserDetailStats{}

	if err := s.db.Model(&models.Tunnel{}).Where("user_id = ?", userID).Count(&stats.TunnelCount).Error; err != nil {
		return stats, fmt.Errorf("统计用户隧道数量失败: %w", err)
	}
	if err := s.db.Model(&models.Tunnel{}).Where("user_id = ? AND status = ?", userID, models.TunnelStatusRunning).Count(&stats.RunningTunnelCount).Error; err != nil {
		return stats, fmt.Errorf("统计用户运行中隧道数量失败: %w", err)
	}
	if err := s.db.Model(&models.NodeGroup{}).Where("user_id = ?", userID).Count(&stats.NodeGroupCount).Error; err != nil {
		return stats, fmt.Errorf("统计用户节点组数量失败: %w", err)
	}
	if err := s.db.Table("node_instances AS ni").
		Joins("JOIN node_groups ng ON ng.id = ni.node_group_id").
		Where("ng.user_id = ?", userID).
		Count(&stats.NodeInstanceCount).Error; err != nil {
		return stats, fmt.Errorf("统计用户节点实例数量失败: %w", err)
	}
	if err := s.db.Model(&models.RefreshToken{}).
		Where("user_id = ? AND is_revoked = ? AND expires_at > ?", userID, false, time.Now()).
		Count(&stats.ActiveSessionCount).Error; err != nil {
		return stats, fmt.Errorf("统计用户活跃会话数量失败: %w", err)
	}

	type trafficAgg struct {
		TotalTrafficIn  int64
		TotalTrafficOut int64
	}
	traffic := trafficAgg{}
	if err := s.db.Model(&models.Tunnel{}).
		Select("COALESCE(SUM(traffic_in), 0) AS total_traffic_in, COALESCE(SUM(traffic_out), 0) AS total_traffic_out").
		Where("user_id = ?", userID).
		Scan(&traffic).Error; err != nil {
		return stats, fmt.Errorf("统计用户流量失败: %w", err)
	}

	stats.TotalTrafficIn = traffic.TotalTrafficIn
	stats.TotalTrafficOut = traffic.TotalTrafficOut
	return stats, nil
}

func (s *UserAdminService) getRecentSessions(userID uint, limit int) ([]UserSessionSummary, error) {
	items := make([]models.RefreshToken, 0)
	if err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询用户会话失败: %w", err)
	}

	result := make([]UserSessionSummary, 0, len(items))
	for _, item := range items {
		result = append(result, UserSessionSummary{
			ID:         item.ID,
			IPAddress:  item.IPAddress,
			UserAgent:  item.UserAgent,
			IsRevoked:  item.IsRevoked,
			LastUsedAt: item.LastUsedAt,
			ExpiresAt:  item.ExpiresAt,
			CreatedAt:  item.CreatedAt,
		})
	}
	return result, nil
}

func (s *UserAdminService) getRecentActivities(userID uint, limit int) ([]UserActivitySummary, error) {
	items := make([]models.AuditLog, 0)
	if err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询用户活动记录失败: %w", err)
	}

	result := make([]UserActivitySummary, 0, len(items))
	for _, item := range items {
		result = append(result, UserActivitySummary{
			ID:           item.ID,
			Action:       item.Action,
			ResourceType: item.ResourceType,
			ResourceID:   item.ResourceID,
			Details:      item.Details,
			IPAddress:    item.IPAddress,
			CreatedAt:    item.CreatedAt,
			UserAgent:    item.UserAgent,
		})
	}
	return result, nil
}

func (s *UserAdminService) getRecentTunnels(userID uint, limit int) ([]UserTunnelSummary, error) {
	items := make([]UserTunnelSummary, 0)
	if err := s.db.Model(&models.Tunnel{}).
		Select("id, name, protocol, status, entry_group_id, exit_group_id, updated_at").
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(limit).
		Scan(&items).Error; err != nil {
		return nil, fmt.Errorf("查询用户隧道失败: %w", err)
	}
	return items, nil
}

func (s *UserAdminService) getRecentNodeGroups(userID uint, limit int) ([]UserNodeGroupSummary, error) {
	items := make([]UserNodeGroupSummary, 0)
	if err := s.db.Model(&models.NodeGroup{}).
		Select("id, name, type, is_enabled, updated_at").
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(limit).
		Scan(&items).Error; err != nil {
		return nil, fmt.Errorf("查询用户节点组失败: %w", err)
	}
	return items, nil
}

func normalizeUserStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "normal", "paused", "banned", "overlimit":
		return status
	default:
		return ""
	}
}
