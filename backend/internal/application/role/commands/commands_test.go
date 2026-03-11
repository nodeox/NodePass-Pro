package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"nodepass-pro/backend/internal/domain/role"
)

// MockRoleRepository Mock 仓储
type MockRoleRepository struct {
	roles   map[uint]*role.Role
	byCode  map[string]*role.Role
	nextID  uint
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

// CreateRoleHandlerTestSuite 测试套件
type CreateRoleHandlerTestSuite struct {
	suite.Suite
	repo    *MockRoleRepository
	handler *CreateRoleHandler
	ctx     context.Context
}

func (s *CreateRoleHandlerTestSuite) SetupTest() {
	s.repo = NewMockRoleRepository()
	s.handler = NewCreateRoleHandler(s.repo)
	s.ctx = context.Background()
}

func (s *CreateRoleHandlerTestSuite) TestCreateRoleSuccess() {
	cmd := CreateRoleCommand{
		AdminID:     1,
		Code:        "developer",
		Name:        "开发者",
		Description: "开发者角色",
		Permissions: []string{"users.read", "tunnels.read"},
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)
	s.NotNil(result)
	s.Equal("developer", result.Code)
	s.Equal("开发者", result.Name)
	s.Len(result.Permissions, 2)
}

func (s *CreateRoleHandlerTestSuite) TestCreateRoleInvalidCode() {
	cmd := CreateRoleCommand{
		AdminID:     1,
		Code:        "INVALID CODE",
		Name:        "测试",
		Permissions: []string{},
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Nil(result)
	s.Equal(role.ErrRoleCodeInvalid, err)
}

func (s *CreateRoleHandlerTestSuite) TestCreateRoleAlreadyExists() {
	// 先创建一个角色
	cmd1 := CreateRoleCommand{
		AdminID:     1,
		Code:        "developer",
		Name:        "开发者",
		Permissions: []string{},
	}
	s.handler.Handle(s.ctx, cmd1)

	// 尝试创建重复的角色
	cmd2 := CreateRoleCommand{
		AdminID:     1,
		Code:        "developer",
		Name:        "开发者2",
		Permissions: []string{},
	}

	result, err := s.handler.Handle(s.ctx, cmd2)
	s.Error(err)
	s.Nil(result)
	s.Equal(role.ErrRoleAlreadyExists, err)
}

func TestCreateRoleHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(CreateRoleHandlerTestSuite))
}

// UpdateRoleHandlerTestSuite 测试套件
type UpdateRoleHandlerTestSuite struct {
	suite.Suite
	repo    *MockRoleRepository
	handler *UpdateRoleHandler
	ctx     context.Context
}

func (s *UpdateRoleHandlerTestSuite) SetupTest() {
	s.repo = NewMockRoleRepository()
	s.handler = NewUpdateRoleHandler(s.repo)
	s.ctx = context.Background()

	// 预创建角色
	r, _ := role.NewRole("developer", "开发者", "开发者角色", false)
	s.repo.Create(s.ctx, r)
}

func (s *UpdateRoleHandlerTestSuite) TestUpdateRoleSuccess() {
	newName := "高级开发者"
	cmd := UpdateRoleCommand{
		AdminID: 1,
		RoleID:  1,
		Name:    &newName,
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)
	s.NotNil(result)
	s.Equal("高级开发者", result.Name)
}

func (s *UpdateRoleHandlerTestSuite) TestUpdateRoleNotFound() {
	newName := "测试"
	cmd := UpdateRoleCommand{
		AdminID: 1,
		RoleID:  999,
		Name:    &newName,
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Nil(result)
	s.Equal(role.ErrRoleNotFound, err)
}

func TestUpdateRoleHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(UpdateRoleHandlerTestSuite))
}

// DeleteRoleHandlerTestSuite 测试套件
type DeleteRoleHandlerTestSuite struct {
	suite.Suite
	repo    *MockRoleRepository
	handler *DeleteRoleHandler
	ctx     context.Context
}

func (s *DeleteRoleHandlerTestSuite) SetupTest() {
	s.repo = NewMockRoleRepository()
	s.handler = NewDeleteRoleHandler(s.repo)
	s.ctx = context.Background()

	// 预创建角色
	r, _ := role.NewRole("developer", "开发者", "开发者角色", false)
	s.repo.Create(s.ctx, r)
}

func (s *DeleteRoleHandlerTestSuite) TestDeleteRoleSuccess() {
	cmd := DeleteRoleCommand{
		AdminID: 1,
		RoleID:  1,
	}

	err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)

	// 验证已删除
	_, err = s.repo.FindByID(s.ctx, 1)
	s.Error(err)
	s.Equal(role.ErrRoleNotFound, err)
}

func (s *DeleteRoleHandlerTestSuite) TestDeleteRoleInUse() {
	// 设置角色被使用
	s.repo.userCounts["developer"] = 5

	cmd := DeleteRoleCommand{
		AdminID: 1,
		RoleID:  1,
	}

	err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Equal(role.ErrRoleInUse, err)
}

func (s *DeleteRoleHandlerTestSuite) TestDeleteSystemRole() {
	// 创建系统角色
	r, _ := role.NewRole("admin", "管理员", "", true)
	s.repo.Create(s.ctx, r)

	cmd := DeleteRoleCommand{
		AdminID: 1,
		RoleID:  r.ID,
	}

	err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Equal(role.ErrSystemRoleCannotDelete, err)
}

func TestDeleteRoleHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteRoleHandlerTestSuite))
}

// AssignPermissionsHandlerTestSuite 测试套件
type AssignPermissionsHandlerTestSuite struct {
	suite.Suite
	repo    *MockRoleRepository
	handler *AssignPermissionsHandler
	ctx     context.Context
}

func (s *AssignPermissionsHandlerTestSuite) SetupTest() {
	s.repo = NewMockRoleRepository()
	s.handler = NewAssignPermissionsHandler(s.repo)
	s.ctx = context.Background()

	// 预创建角色
	r, _ := role.NewRole("developer", "开发者", "开发者角色", false)
	s.repo.Create(s.ctx, r)
}

func (s *AssignPermissionsHandlerTestSuite) TestAssignPermissionsSuccess() {
	cmd := AssignPermissionsCommand{
		AdminID:     1,
		RoleID:      1,
		Permissions: []string{"users.read", "users.write", "tunnels.read"},
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)
	s.NotNil(result)
	s.Len(result.Permissions, 3)
}

func (s *AssignPermissionsHandlerTestSuite) TestAssignPermissionsNotFound() {
	cmd := AssignPermissionsCommand{
		AdminID:     1,
		RoleID:      999,
		Permissions: []string{"users.read"},
	}

	result, err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Nil(result)
	s.Equal(role.ErrRoleNotFound, err)
}

func TestAssignPermissionsHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(AssignPermissionsHandlerTestSuite))
}
