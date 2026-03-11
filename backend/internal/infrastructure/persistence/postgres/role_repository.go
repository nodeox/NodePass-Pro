package postgres

import (
	"context"
	"errors"
	"fmt"

	"nodepass-pro/backend/internal/domain/role"
	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RoleRepository 角色仓储实现
type RoleRepository struct {
	db *gorm.DB
}

// NewRoleRepository 创建角色仓储
func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// Create 创建角色
func (r *RoleRepository) Create(ctx context.Context, roleEntity *role.Role) error {
	model := r.toModel(roleEntity)

	// 开启事务
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建角色
		if err := tx.Create(model).Error; err != nil {
			if isDuplicateKeyError(err) {
				return role.ErrRoleAlreadyExists
			}
			return fmt.Errorf("创建角色失败: %w", err)
		}

		roleEntity.ID = model.ID

		// 创建权限
		if len(roleEntity.Permissions) > 0 {
			permissions := make([]*models.RolePermission, len(roleEntity.Permissions))
			for i, p := range roleEntity.Permissions {
				permissions[i] = &models.RolePermission{
					RoleID:     model.ID,
					Permission: p.Code,
				}
			}
			if err := tx.Create(&permissions).Error; err != nil {
				return fmt.Errorf("创建角色权限失败: %w", err)
			}
		}

		return nil
	})
}

// FindByID 根据 ID 查找
func (r *RoleRepository) FindByID(ctx context.Context, id uint) (*role.Role, error) {
	var model models.Role
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, role.ErrRoleNotFound
		}
		return nil, fmt.Errorf("查找角色失败: %w", err)
	}

	// 加载权限
	permissions, err := r.loadPermissions(ctx, id)
	if err != nil {
		return nil, err
	}

	return r.toDomain(&model, permissions), nil
}

// FindByCode 根据 Code 查找
func (r *RoleRepository) FindByCode(ctx context.Context, code string) (*role.Role, error) {
	var model models.Role
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, role.ErrRoleNotFound
		}
		return nil, fmt.Errorf("查找角色失败: %w", err)
	}

	// 加载权限
	permissions, err := r.loadPermissions(ctx, model.ID)
	if err != nil {
		return nil, err
	}

	return r.toDomain(&model, permissions), nil
}

// Update 更新角色
func (r *RoleRepository) Update(ctx context.Context, roleEntity *role.Role) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新角色基本信息
		model := r.toModel(roleEntity)
		if err := tx.Save(model).Error; err != nil {
			return fmt.Errorf("更新角色失败: %w", err)
		}

		// 删除旧权限
		if err := tx.Where("role_id = ?", roleEntity.ID).Delete(&models.RolePermission{}).Error; err != nil {
			return fmt.Errorf("删除旧权限失败: %w", err)
		}

		// 创建新权限
		if len(roleEntity.Permissions) > 0 {
			permissions := make([]*models.RolePermission, len(roleEntity.Permissions))
			for i, p := range roleEntity.Permissions {
				permissions[i] = &models.RolePermission{
					RoleID:     roleEntity.ID,
					Permission: p.Code,
				}
			}
			if err := tx.Create(&permissions).Error; err != nil {
				return fmt.Errorf("创建新权限失败: %w", err)
			}
		}

		return nil
	})
}

// Delete 删除角色
func (r *RoleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除权限
		if err := tx.Where("role_id = ?", id).Delete(&models.RolePermission{}).Error; err != nil {
			return fmt.Errorf("删除角色权限失败: %w", err)
		}

		// 删除角色
		result := tx.Delete(&models.Role{}, id)
		if result.Error != nil {
			return fmt.Errorf("删除角色失败: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return role.ErrRoleNotFound
		}

		return nil
	})
}

// List 列表查询
func (r *RoleRepository) List(ctx context.Context, filter role.ListFilter) ([]*role.Role, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Role{})

	// 应用过滤条件
	if !filter.IncludeDisabled {
		query = query.Where("is_enabled = ?", true)
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		query = query.Where("code LIKE ? OR name LIKE ?", like, like)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计角色总数失败: %w", err)
	}

	// 分页查询
	var models []*models.Role
	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Order("is_system DESC, id ASC").Offset(offset).Limit(filter.PageSize).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("查询角色列表失败: %w", err)
	}

	// 批量加载权限
	roleIDs := make([]uint, len(models))
	for i, m := range models {
		roleIDs[i] = m.ID
	}
	permissionsMap, err := r.loadPermissionsMap(ctx, roleIDs)
	if err != nil {
		return nil, 0, err
	}

	// 转换为领域对象
	roles := make([]*role.Role, len(models))
	for i, m := range models {
		roles[i] = r.toDomain(m, permissionsMap[m.ID])
	}

	return roles, total, nil
}

// CountUsersByRole 统计使用该角色的用户数量
func (r *RoleRepository) CountUsersByRole(ctx context.Context, roleCode string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Where("role = ?", roleCode).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计用户数量失败: %w", err)
	}
	return count, nil
}

// EnsureSystemRoles 确保系统角色存在
func (r *RoleRepository) EnsureSystemRoles(ctx context.Context) error {
	systemRoles := []models.Role{
		{Code: "admin", Name: "管理员", IsSystem: true, IsEnabled: true},
		{Code: "user", Name: "普通用户", IsSystem: true, IsEnabled: true},
	}

	for _, roleModel := range systemRoles {
		if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "code"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"name":       roleModel.Name,
				"is_system":  roleModel.IsSystem,
				"is_enabled": roleModel.IsEnabled,
			}),
		}).Create(&roleModel).Error; err != nil {
			return fmt.Errorf("初始化系统角色失败: %w", err)
		}
	}

	return nil
}

// GetAvailablePermissions 获取所有可用权限
func (r *RoleRepository) GetAvailablePermissions(ctx context.Context) ([]role.Permission, error) {
	// 默认权限列表
	defaultPermissions := []string{
		"users.read", "users.write",
		"roles.read", "roles.write",
		"node_groups.read", "node_groups.write",
		"tunnels.read", "tunnels.write",
		"traffic.read",
		"vip.read", "vip.write",
		"benefit_codes.read", "benefit_codes.write",
		"announcements.read", "announcements.write",
		"system.config.read", "system.config.write",
		"audit_logs.read",
	}

	set := make(map[string]struct{})
	for _, p := range defaultPermissions {
		set[p] = struct{}{}
	}

	// 从数据库加载已使用的权限
	var rolePermissions []string
	if err := r.db.WithContext(ctx).Model(&models.RolePermission{}).
		Distinct("permission").
		Pluck("permission", &rolePermissions).Error; err != nil {
		return nil, fmt.Errorf("查询角色权限失败: %w", err)
	}
	for _, p := range rolePermissions {
		if p != "" {
			set[p] = struct{}{}
		}
	}

	// 转换为权限对象
	permissions := make([]role.Permission, 0, len(set))
	for p := range set {
		permissions = append(permissions, role.NewPermission(p, ""))
	}

	return permissions, nil
}

// loadPermissions 加载角色权限
func (r *RoleRepository) loadPermissions(ctx context.Context, roleID uint) ([]role.Permission, error) {
	var models []*models.RolePermission
	if err := r.db.WithContext(ctx).Where("role_id = ?", roleID).Order("permission ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("加载角色权限失败: %w", err)
	}

	permissions := make([]role.Permission, len(models))
	for i, m := range models {
		permissions[i] = role.NewPermission(m.Permission, "")
	}

	return permissions, nil
}

// loadPermissionsMap 批量加载权限
func (r *RoleRepository) loadPermissionsMap(ctx context.Context, roleIDs []uint) (map[uint][]role.Permission, error) {
	if len(roleIDs) == 0 {
		return make(map[uint][]role.Permission), nil
	}

	var models []*models.RolePermission
	if err := r.db.WithContext(ctx).Where("role_id IN ?", roleIDs).Order("permission ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("批量加载权限失败: %w", err)
	}

	result := make(map[uint][]role.Permission)
	for _, m := range models {
		result[m.RoleID] = append(result[m.RoleID], role.NewPermission(m.Permission, ""))
	}

	return result, nil
}

// toModel 转换为数据库模型
func (r *RoleRepository) toModel(roleEntity *role.Role) *models.Role {
	var desc *string
	if roleEntity.Description != "" {
		desc = &roleEntity.Description
	}

	return &models.Role{
		ID:          roleEntity.ID,
		Code:        roleEntity.Code,
		Name:        roleEntity.Name,
		Description: desc,
		IsSystem:    roleEntity.IsSystem,
		IsEnabled:   roleEntity.IsEnabled,
		CreatedAt:   roleEntity.CreatedAt,
		UpdatedAt:   roleEntity.UpdatedAt,
	}
}

// toDomain 转换为领域对象
func (r *RoleRepository) toDomain(model *models.Role, permissions []role.Permission) *role.Role {
	desc := ""
	if model.Description != nil {
		desc = *model.Description
	}

	return &role.Role{
		ID:          model.ID,
		Code:        model.Code,
		Name:        model.Name,
		Description: desc,
		IsSystem:    model.IsSystem,
		IsEnabled:   model.IsEnabled,
		Permissions: permissions,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}
