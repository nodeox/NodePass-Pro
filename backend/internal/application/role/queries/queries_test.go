package queries

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"nodepass-pro/backend/internal/domain/role"
)

// MockRoleRepository Mock 仓储
type MockRoleRepository struct {
	roles      map[uint]*role.Role
	byCode     map[string]*role.Role
	nextID     uint
	userCounts map[string]int64
}

func NewMockRoleRepository() *MockRoleRepository {
	return &MockRoleRepository{
		roles:      make(map[uint]*role.Role),
		byCode:     make(map[string]*role.Role),
		nextID:     1,
		userCounts: make(map[string]int64),
	}
}

func (m *MockRoleRepository) Create(ctx context.Context, r *role.Role) error {
	if _, exists := m.byCode[r.Code]; exists {
		return role.ErrRoleAlreadyExists
	}
	r.ID = m.nextID
	m.nextID++
	m.roles[r.ID] = r
	m.byCode[r.Code] = r
	return nil
}

func (m *MockRoleRepository) FindByID(ctx context.Context, id uint) (*role.Role, error) {
	r, ok := m.roles[id]
	if !ok {
		return nil, role.ErrRoleNotFound
	}
	return r, nil
}

func (m *MockRoleRepository) FindByCode(ctx context.Context, code string) (*role.Role, error) {
	r, ok := m.byCode[code]
	if !ok {
		return nil, role.ErrRoleNotFound
	}
	return r, nil
}

func (m *MockRoleRepository) Update(ctx context.Context, r *role.Role) error {
	if _, ok := m.roles[r.ID]; !ok {
		return role.ErrRoleNotFound
	}
	m.roles[r.ID] = r
	m.byCode[r.Code] = r
	return nil
}

func (m *MockRoleRepository) Delete(ctx context.Context, id uint) error {
	r, ok := m.roles[id]
	if !ok {
		return role.ErrRoleNotFound
	}
	delete(m.roles, id)
	delete(m.byCode, r.Code)
	return nil
}

func (m *MockRoleRepository) List(ctx context.Context, filter role.ListFilter) ([]*role.Role, int64, error) {
	var result []*role.Role
	for _, r := range m.roles {
		if !filter.IncludeDisabled && !r.IsEnabled {
			continue
		}
		result = append(result, r)
	}
	return result, int64(len(result)), nil
}

func (m *MockRoleRepository) CountUsersByRole(ctx context.Context, roleCode string) (int64, error) {
	return m.userCounts[roleCode], nil
}

func (m *MockRoleRepository) EnsureSystemRoles(ctx context.Context) error {
	return nil
}

func (m *MockRoleRepository) GetAvailablePermissions(ctx context.Context) ([]role.Permission, error) {
	return []role.Permission{
		role.NewPermission("users.read", ""),
		role.NewPermission("users.write", ""),
	}, nil
}

// GetRoleHandlerTestSuite 测试套件
type GetRoleHandlerTestSuite struct {
	suite.Suite
	repo    *MockRoleRepository
	handler *GetRoleHandler
	ctx     context.Context
}

func (s *GetRoleHandlerTestSuite) SetupTest() {
	s.repo = NewMockRoleRepository()
	s.handler = NewGetRoleHandler(s.repo)
	s.ctx = context.Background()

	// 预创建角色
	r, _ := role.NewRole("developer", "开发者", "开发者角色", false)
	s.repo.Create(s.ctx, r)
}

func (s *GetRoleHandlerTestSuite) TestGetRoleSuccess() {
	query := GetRoleQuery{RoleID: 1}
	result, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.NotNil(result)
	s.Equal("developer", result.Code)
}

func (s *GetRoleHandlerTestSuite) TestGetRoleNotFound() {
	query := GetRoleQuery{RoleID: 999}
	result, err := s.handler.Handle(s.ctx, query)
	s.Error(err)
	s.Nil(result)
	s.Equal(role.ErrRoleNotFound, err)
}

func TestGetRoleHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GetRoleHandlerTestSuite))
}

// ListRolesHandlerTestSuite 测试套件
type ListRolesHandlerTestSuite struct {
	suite.Suite
	repo    *MockRoleRepository
	handler *ListRolesHandler
	ctx     context.Context
}

func (s *ListRolesHandlerTestSuite) SetupTest() {
	s.repo = NewMockRoleRepository()
	s.handler = NewListRolesHandler(s.repo)
	s.ctx = context.Background()

	// 预创建角色
	r1, _ := role.NewRole("developer", "开发者", "", false)
	s.repo.Create(s.ctx, r1)

	r2, _ := role.NewRole("tester", "测试员", "", false)
	s.repo.Create(s.ctx, r2)
}

func (s *ListRolesHandlerTestSuite) TestListRolesSuccess() {
	query := ListRolesQuery{
		Page:     1,
		PageSize: 20,
	}

	result, total, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.GreaterOrEqual(len(result), 2)
	s.GreaterOrEqual(int(total), 2)
}

func TestListRolesHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ListRolesHandlerTestSuite))
}

// CheckPermissionHandlerTestSuite 测试套件
type CheckPermissionHandlerTestSuite struct {
	suite.Suite
	repo    *MockRoleRepository
	handler *CheckPermissionHandler
	ctx     context.Context
}

func (s *CheckPermissionHandlerTestSuite) SetupTest() {
	s.repo = NewMockRoleRepository()
	s.handler = NewCheckPermissionHandler(s.repo)
	s.ctx = context.Background()

	// 预创建角色
	r, _ := role.NewRole("developer", "开发者", "", false)
	r.SetPermissions([]role.Permission{
		role.NewPermission("users.read", ""),
		role.NewPermission("tunnels.read", ""),
	})
	s.repo.Create(s.ctx, r)
}

func (s *CheckPermissionHandlerTestSuite) TestCheckPermissionHasPermission() {
	query := CheckPermissionQuery{
		RoleCode:       "developer",
		PermissionCode: "users.read",
	}

	result, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.NotNil(result)
	s.True(result.HasPermission)
}

func (s *CheckPermissionHandlerTestSuite) TestCheckPermissionNoPermission() {
	query := CheckPermissionQuery{
		RoleCode:       "developer",
		PermissionCode: "users.write",
	}

	result, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.NotNil(result)
	s.False(result.HasPermission)
}

func TestCheckPermissionHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(CheckPermissionHandlerTestSuite))
}

// GetAvailablePermissionsHandlerTestSuite 测试套件
type GetAvailablePermissionsHandlerTestSuite struct {
	suite.Suite
	repo    *MockRoleRepository
	handler *GetAvailablePermissionsHandler
	ctx     context.Context
}

func (s *GetAvailablePermissionsHandlerTestSuite) SetupTest() {
	s.repo = NewMockRoleRepository()
	s.handler = NewGetAvailablePermissionsHandler(s.repo)
	s.ctx = context.Background()
}

func (s *GetAvailablePermissionsHandlerTestSuite) TestGetAvailablePermissions() {
	query := GetAvailablePermissionsQuery{AdminID: 1}
	result, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.NotNil(result)
	s.GreaterOrEqual(len(result), 2)
}

func TestGetAvailablePermissionsHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GetAvailablePermissionsHandlerTestSuite))
}
