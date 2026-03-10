package services

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var roleCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_\-]{1,49}$`)

var defaultRolePermissions = []string{
	"users.read",
	"users.write",
	"roles.read",
	"roles.write",
	"node_groups.read",
	"node_groups.write",
	"tunnels.read",
	"tunnels.write",
	"traffic.read",
	"vip.read",
	"vip.write",
	"benefit_codes.read",
	"benefit_codes.write",
	"announcements.read",
	"announcements.write",
	"system.config.read",
	"system.config.write",
	"audit_logs.read",
}

// RoleAdminService 角色管理服务（管理员）。
type RoleAdminService struct {
	db *gorm.DB
}

// RoleAdminRecord 角色记录。
type RoleAdminRecord struct {
	ID          uint     `json:"id"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	IsSystem    bool     `json:"is_system"`
	IsEnabled   bool     `json:"is_enabled"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Permissions []string `json:"permissions"`
}

// CreateRolePayload 创建角色请求。
type CreateRolePayload struct {
	Code        string
	Name        string
	Description *string
	IsEnabled   *bool
	Permissions []string
}

// UpdateRolePayload 更新角色请求。
type UpdateRolePayload struct {
	Name        *string
	Description *string
	IsEnabled   *bool
}

// NewRoleAdminService 创建角色管理服务。
func NewRoleAdminService(db *gorm.DB) *RoleAdminService {
	return &RoleAdminService{db: db}
}

// ListRoles 获取角色列表。
func (s *RoleAdminService) ListRoles(adminUserID uint, includeDisabled bool, keyword string) ([]RoleAdminRecord, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}
	if err := s.ensureSystemRoles(); err != nil {
		return nil, err
	}

	query := s.db.Model(&models.Role{})
	if !includeDisabled {
		query = query.Where("is_enabled = ?", true)
	}
	if trimmed := strings.TrimSpace(keyword); trimmed != "" {
		like := "%" + trimmed + "%"
		query = query.Where("code LIKE ? OR name LIKE ?", like, like)
	}

	roles := make([]models.Role, 0)
	if err := query.Order("is_system DESC, id ASC").Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("查询角色列表失败: %w", err)
	}

	permissionMap, err := s.getRolePermissionsMap(roles)
	if err != nil {
		return nil, err
	}

	items := make([]RoleAdminRecord, 0, len(roles))
	for _, role := range roles {
		items = append(items, RoleAdminRecord{
			ID:          role.ID,
			Code:        role.Code,
			Name:        role.Name,
			Description: role.Description,
			IsSystem:    role.IsSystem,
			IsEnabled:   role.IsEnabled,
			CreatedAt:   role.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   role.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Permissions: permissionMap[role.ID],
		})
	}

	return items, nil
}

// GetRole 获取角色详情。
func (s *RoleAdminService) GetRole(adminUserID uint, roleID uint) (*RoleAdminRecord, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}
	if roleID == 0 {
		return nil, fmt.Errorf("%w: 角色 ID 无效", ErrInvalidParams)
	}
	if err := s.ensureSystemRoles(); err != nil {
		return nil, err
	}

	var role models.Role
	if err := s.db.First(&role, roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 角色不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}

	permissions, err := s.getRolePermissions(role.ID)
	if err != nil {
		return nil, err
	}

	return &RoleAdminRecord{
		ID:          role.ID,
		Code:        role.Code,
		Name:        role.Name,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		IsEnabled:   role.IsEnabled,
		CreatedAt:   role.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   role.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Permissions: permissions,
	}, nil
}

// CreateRole 创建角色。
func (s *RoleAdminService) CreateRole(adminUserID uint, payload CreateRolePayload) (*RoleAdminRecord, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}
	if err := s.ensureSystemRoles(); err != nil {
		return nil, err
	}

	code := normalizeRoleCode(payload.Code)
	if code == "" || !roleCodePattern.MatchString(code) {
		return nil, fmt.Errorf("%w: 角色编码仅支持小写字母、数字、下划线和短横线，长度 2-50", ErrInvalidParams)
	}
	if code == "admin" || code == "user" {
		return nil, fmt.Errorf("%w: 系统角色编码不可占用", ErrConflict)
	}

	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: 角色名称不能为空", ErrInvalidParams)
	}
	if len([]rune(name)) > 100 {
		return nil, fmt.Errorf("%w: 角色名称长度不能超过 100", ErrInvalidParams)
	}

	normalizedPermissions := normalizePermissionList(payload.Permissions)

	role := models.Role{
		Code:        code,
		Name:        name,
		Description: normalizeOptionalString(payload.Description),
		IsSystem:    false,
		IsEnabled:   true,
	}
	if payload.IsEnabled != nil {
		role.IsEnabled = *payload.IsEnabled
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&role).Error; err != nil {
			if isDuplicateKeyError(err) {
				return fmt.Errorf("%w: 角色编码已存在", ErrConflict)
			}
			return fmt.Errorf("创建角色失败: %w", err)
		}

		if err := replaceRolePermissions(tx, role.ID, normalizedPermissions); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return s.GetRole(adminUserID, role.ID)
}

// UpdateRole 更新角色基本信息。
func (s *RoleAdminService) UpdateRole(adminUserID uint, roleID uint, payload UpdateRolePayload) (*RoleAdminRecord, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}
	if roleID == 0 {
		return nil, fmt.Errorf("%w: 角色 ID 无效", ErrInvalidParams)
	}

	var role models.Role
	if err := s.db.First(&role, roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 角色不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}

	updates := make(map[string]interface{})
	if payload.Name != nil {
		name := strings.TrimSpace(*payload.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: 角色名称不能为空", ErrInvalidParams)
		}
		if len([]rune(name)) > 100 {
			return nil, fmt.Errorf("%w: 角色名称长度不能超过 100", ErrInvalidParams)
		}
		updates["name"] = name
	}
	if payload.Description != nil {
		updates["description"] = normalizeOptionalString(payload.Description)
	}
	if payload.IsEnabled != nil {
		if role.IsSystem && !*payload.IsEnabled {
			return nil, fmt.Errorf("%w: 系统角色不可禁用", ErrConflict)
		}
		updates["is_enabled"] = *payload.IsEnabled
	}
	if len(updates) == 0 {
		return s.GetRole(adminUserID, roleID)
	}

	if err := s.db.Model(&models.Role{}).Where("id = ?", roleID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新角色失败: %w", err)
	}

	return s.GetRole(adminUserID, roleID)
}

// UpdateRolePermissions 更新角色权限。
func (s *RoleAdminService) UpdateRolePermissions(adminUserID uint, roleID uint, permissions []string) (*RoleAdminRecord, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}
	if roleID == 0 {
		return nil, fmt.Errorf("%w: 角色 ID 无效", ErrInvalidParams)
	}

	var role models.Role
	if err := s.db.First(&role, roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 角色不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}

	normalizedPermissions := normalizePermissionList(permissions)
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return replaceRolePermissions(tx, roleID, normalizedPermissions)
	}); err != nil {
		return nil, err
	}

	return s.GetRole(adminUserID, roleID)
}

// DeleteRole 删除角色。
func (s *RoleAdminService) DeleteRole(adminUserID uint, roleID uint) error {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return err
	}
	if roleID == 0 {
		return fmt.Errorf("%w: 角色 ID 无效", ErrInvalidParams)
	}

	var role models.Role
	if err := s.db.First(&role, roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: 角色不存在", ErrNotFound)
		}
		return fmt.Errorf("查询角色失败: %w", err)
	}
	if role.IsSystem {
		return fmt.Errorf("%w: 系统角色不可删除", ErrConflict)
	}

	var usedCount int64
	if err := s.db.Model(&models.User{}).Where("role = ?", role.Code).Count(&usedCount).Error; err != nil {
		return fmt.Errorf("查询角色关联用户失败: %w", err)
	}
	if usedCount > 0 {
		return fmt.Errorf("%w: 角色仍被 %d 个用户使用", ErrConflict, usedCount)
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
			return fmt.Errorf("删除角色权限失败: %w", err)
		}
		if err := tx.Delete(&models.Role{}, roleID).Error; err != nil {
			return fmt.Errorf("删除角色失败: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// ListAvailablePermissions 返回权限候选列表。
func (s *RoleAdminService) ListAvailablePermissions(adminUserID uint) ([]string, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}

	set := make(map[string]struct{}, len(defaultRolePermissions)+16)
	for _, permission := range defaultRolePermissions {
		set[permission] = struct{}{}
	}

	var rolePermissions []string
	if err := s.db.Model(&models.RolePermission{}).
		Distinct("permission").
		Pluck("permission", &rolePermissions).Error; err != nil {
		return nil, fmt.Errorf("查询角色权限失败: %w", err)
	}
	for _, permission := range rolePermissions {
		permission = strings.TrimSpace(permission)
		if permission != "" {
			set[permission] = struct{}{}
		}
	}

	var userPermissions []string
	if err := s.db.Model(&models.UserPermission{}).
		Distinct("permission").
		Pluck("permission", &userPermissions).Error; err != nil {
		return nil, fmt.Errorf("查询用户权限失败: %w", err)
	}
	for _, permission := range userPermissions {
		permission = strings.TrimSpace(permission)
		if permission != "" {
			set[permission] = struct{}{}
		}
	}

	items := make([]string, 0, len(set))
	for permission := range set {
		items = append(items, permission)
	}
	sort.Strings(items)
	return items, nil
}

func (s *RoleAdminService) ensureSystemRoles() error {
	systemRoles := []models.Role{
		{Code: "admin", Name: "管理员", IsSystem: true, IsEnabled: true},
		{Code: "user", Name: "普通用户", IsSystem: true, IsEnabled: true},
	}

	for _, role := range systemRoles {
		if err := s.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "code"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"name":       role.Name,
				"is_system":  role.IsSystem,
				"is_enabled": role.IsEnabled,
			}),
		}).Create(&role).Error; err != nil {
			return fmt.Errorf("初始化系统角色失败: %w", err)
		}
	}

	return nil
}

func (s *RoleAdminService) getRolePermissionsMap(roles []models.Role) (map[uint][]string, error) {
	result := make(map[uint][]string, len(roles))
	if len(roles) == 0 {
		return result, nil
	}

	ids := make([]uint, 0, len(roles))
	for _, role := range roles {
		ids = append(ids, role.ID)
	}

	permissions := make([]models.RolePermission, 0)
	if err := s.db.Where("role_id IN ?", ids).Order("permission ASC").Find(&permissions).Error; err != nil {
		return nil, fmt.Errorf("查询角色权限失败: %w", err)
	}

	for _, permission := range permissions {
		result[permission.RoleID] = append(result[permission.RoleID], permission.Permission)
	}
	for id := range result {
		sort.Strings(result[id])
	}

	return result, nil
}

func (s *RoleAdminService) getRolePermissions(roleID uint) ([]string, error) {
	permissions := make([]string, 0)
	if err := s.db.Model(&models.RolePermission{}).
		Where("role_id = ?", roleID).
		Order("permission ASC").
		Pluck("permission", &permissions).Error; err != nil {
		return nil, fmt.Errorf("查询角色权限失败: %w", err)
	}
	return permissions, nil
}

func replaceRolePermissions(tx *gorm.DB, roleID uint, permissions []string) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
		return fmt.Errorf("清理角色权限失败: %w", err)
	}

	if len(permissions) == 0 {
		return nil
	}

	items := make([]models.RolePermission, 0, len(permissions))
	for _, permission := range permissions {
		items = append(items, models.RolePermission{
			RoleID:     roleID,
			Permission: permission,
		})
	}

	if err := tx.Create(&items).Error; err != nil {
		return fmt.Errorf("写入角色权限失败: %w", err)
	}
	return nil
}

func normalizeRoleCode(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

func normalizePermissionList(permissions []string) []string {
	set := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		normalized := strings.TrimSpace(permission)
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}

	items := make([]string, 0, len(set))
	for permission := range set {
		items = append(items, permission)
	}
	sort.Strings(items)
	return items
}
