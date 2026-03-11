package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"nodepass-pro/backend/internal/domain/nodegroup"
)

// MockNodeGroupRepository Mock 仓储
type MockNodeGroupRepository struct {
	groups map[uint]*nodegroup.NodeGroup
	nextID uint
}

func NewMockNodeGroupRepository() *MockNodeGroupRepository {
	return &MockNodeGroupRepository{
		groups: make(map[uint]*nodegroup.NodeGroup),
		nextID: 1,
	}
}

func (m *MockNodeGroupRepository) Create(ctx context.Context, group *nodegroup.NodeGroup) error {
	group.ID = m.nextID
	m.nextID++
	m.groups[group.ID] = group
	return nil
}

func (m *MockNodeGroupRepository) FindByID(ctx context.Context, id uint) (*nodegroup.NodeGroup, error) {
	group, ok := m.groups[id]
	if !ok {
		return nil, nodegroup.ErrNodeGroupNotFound
	}
	return group, nil
}

func (m *MockNodeGroupRepository) Update(ctx context.Context, group *nodegroup.NodeGroup) error {
	if _, ok := m.groups[group.ID]; !ok {
		return nodegroup.ErrNodeGroupNotFound
	}
	m.groups[group.ID] = group
	return nil
}

func (m *MockNodeGroupRepository) Delete(ctx context.Context, id uint) error {
	if _, ok := m.groups[id]; !ok {
		return nodegroup.ErrNodeGroupNotFound
	}
	delete(m.groups, id)
	return nil
}

func (m *MockNodeGroupRepository) FindByUserID(ctx context.Context, userID uint) ([]*nodegroup.NodeGroup, error) {
	var result []*nodegroup.NodeGroup
	for _, g := range m.groups {
		if g.UserID == userID {
			result = append(result, g)
		}
	}
	return result, nil
}

func (m *MockNodeGroupRepository) FindByType(ctx context.Context, groupType nodegroup.NodeGroupType) ([]*nodegroup.NodeGroup, error) {
	var result []*nodegroup.NodeGroup
	for _, g := range m.groups {
		if g.Type == groupType {
			result = append(result, g)
		}
	}
	return result, nil
}

func (m *MockNodeGroupRepository) FindByUserIDAndType(ctx context.Context, userID uint, groupType nodegroup.NodeGroupType) ([]*nodegroup.NodeGroup, error) {
	var result []*nodegroup.NodeGroup
	for _, g := range m.groups {
		if g.UserID == userID && g.Type == groupType {
			result = append(result, g)
		}
	}
	return result, nil
}

func (m *MockNodeGroupRepository) List(ctx context.Context, filter nodegroup.ListFilter) ([]*nodegroup.NodeGroup, int64, error) {
	var result []*nodegroup.NodeGroup
	for _, g := range m.groups {
		if filter.UserID > 0 && g.UserID != filter.UserID {
			continue
		}
		if filter.Type != "" && g.Type != filter.Type {
			continue
		}
		if filter.EnabledOnly && !g.IsEnabled {
			continue
		}
		result = append(result, g)
	}
	return result, int64(len(result)), nil
}

func (m *MockNodeGroupRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	count := int64(0)
	for _, g := range m.groups {
		if g.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (m *MockNodeGroupRepository) UpdateStats(ctx context.Context, stats *nodegroup.NodeGroupStats) error {
	return nil
}

func (m *MockNodeGroupRepository) GetStats(ctx context.Context, groupID uint) (*nodegroup.NodeGroupStats, error) {
	return &nodegroup.NodeGroupStats{NodeGroupID: groupID}, nil
}

// CreateGroupHandlerTestSuite 测试套件
type CreateGroupHandlerTestSuite struct {
	suite.Suite
	handler *CreateGroupHandler
	repo    *MockNodeGroupRepository
	ctx     context.Context
}

func (s *CreateGroupHandlerTestSuite) SetupTest() {
	s.repo = NewMockNodeGroupRepository()
	s.handler = NewCreateGroupHandler(s.repo)
	s.ctx = context.Background()
}

func (s *CreateGroupHandlerTestSuite) TestCreateEntryGroup() {
	cmd := CreateGroupCommand{
		UserID:      1,
		Name:        "Entry Group 1",
		Type:        nodegroup.NodeGroupTypeEntry,
		Description: "Test entry group",
		Config: nodegroup.NodeGroupConfig{
			AllowedProtocols: []string{"tcp", "udp"},
			PortRange:        nodegroup.PortRange{Start: 10000, End: 20000},
		},
	}

	group, err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)
	s.NotNil(group)
	s.Equal(uint(1), group.ID)
	s.Equal(cmd.UserID, group.UserID)
	s.Equal(cmd.Name, group.Name)
	s.Equal(cmd.Type, group.Type)
	s.True(group.IsEnabled)
}

func (s *CreateGroupHandlerTestSuite) TestCreateExitGroup() {
	cmd := CreateGroupCommand{
		UserID:      1,
		Name:        "Exit Group 1",
		Type:        nodegroup.NodeGroupTypeExit,
		Description: "Test exit group",
		Config: nodegroup.NodeGroupConfig{
			ExitConfig: &nodegroup.ExitGroupConfig{
				LoadBalanceStrategy: nodegroup.LoadBalanceRoundRobin,
				HealthCheckInterval: 30,
				HealthCheckTimeout:  5,
			},
		},
	}

	group, err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)
	s.NotNil(group)
	s.Equal(nodegroup.NodeGroupTypeExit, group.Type)
}

func (s *CreateGroupHandlerTestSuite) TestCreateGroupInvalidType() {
	cmd := CreateGroupCommand{
		UserID: 1,
		Name:   "Invalid Group",
		Type:   "invalid",
	}

	group, err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Equal(nodegroup.ErrInvalidNodeGroupType, err)
	s.Nil(group)
}

func (s *CreateGroupHandlerTestSuite) TestCreateGroupInvalidPortRange() {
	cmd := CreateGroupCommand{
		UserID: 1,
		Name:   "Invalid Port Range",
		Type:   nodegroup.NodeGroupTypeEntry,
		Config: nodegroup.NodeGroupConfig{
			PortRange: nodegroup.PortRange{Start: 20000, End: 10000}, // Start > End
		},
	}

	group, err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Equal(nodegroup.ErrInvalidPortRange, err)
	s.Nil(group)
}

func TestCreateGroupHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(CreateGroupHandlerTestSuite))
}

// UpdateGroupHandlerTestSuite 测试套件
type UpdateGroupHandlerTestSuite struct {
	suite.Suite
	handler *UpdateGroupHandler
	repo    *MockNodeGroupRepository
	ctx     context.Context
}

func (s *UpdateGroupHandlerTestSuite) SetupTest() {
	s.repo = NewMockNodeGroupRepository()
	s.handler = NewUpdateGroupHandler(s.repo)
	s.ctx = context.Background()

	// 预创建一个节点组
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID:      1,
		Name:        "Original Name",
		Type:        nodegroup.NodeGroupTypeEntry,
		Description: "Original Description",
		IsEnabled:   true,
	})
}

func (s *UpdateGroupHandlerTestSuite) TestUpdateGroup() {
	cmd := UpdateGroupCommand{
		ID:          1,
		Name:        "Updated Name",
		Description: "Updated Description",
		Config: nodegroup.NodeGroupConfig{
			AllowedProtocols: []string{"tcp"},
		},
	}

	group, err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)
	s.NotNil(group)
	s.Equal(cmd.Name, group.Name)
	s.Equal(cmd.Description, group.Description)
}

func (s *UpdateGroupHandlerTestSuite) TestUpdateGroupNotFound() {
	cmd := UpdateGroupCommand{
		ID:   999,
		Name: "Not Found",
	}

	group, err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Equal(nodegroup.ErrNodeGroupNotFound, err)
	s.Nil(group)
}

func (s *UpdateGroupHandlerTestSuite) TestUpdateGroupInvalidPortRange() {
	cmd := UpdateGroupCommand{
		ID:   1,
		Name: "Invalid Port Range",
		Config: nodegroup.NodeGroupConfig{
			PortRange: nodegroup.PortRange{Start: 20000, End: 10000},
		},
	}

	group, err := s.handler.Handle(s.ctx, cmd)
	s.Error(err)
	s.Equal(nodegroup.ErrInvalidPortRange, err)
	s.Nil(group)
}

func TestUpdateGroupHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(UpdateGroupHandlerTestSuite))
}

// DeleteGroupHandlerTestSuite 测试套件
type DeleteGroupHandlerTestSuite struct {
	suite.Suite
	handler *DeleteGroupHandler
	repo    *MockNodeGroupRepository
	ctx     context.Context
}

func (s *DeleteGroupHandlerTestSuite) SetupTest() {
	s.repo = NewMockNodeGroupRepository()
	s.handler = NewDeleteGroupHandler(s.repo)
	s.ctx = context.Background()

	// 预创建一个节点组
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 1,
		Name:   "Test Group",
		Type:   nodegroup.NodeGroupTypeEntry,
	})
}

func (s *DeleteGroupHandlerTestSuite) TestDeleteGroup() {
	err := s.handler.Handle(s.ctx, DeleteGroupCommand{ID: 1})
	s.NoError(err)

	// 验证已删除
	_, err = s.repo.FindByID(s.ctx, 1)
	s.Error(err)
	s.Equal(nodegroup.ErrNodeGroupNotFound, err)
}

func (s *DeleteGroupHandlerTestSuite) TestDeleteGroupNotFound() {
	err := s.handler.Handle(s.ctx, DeleteGroupCommand{ID: 999})
	s.Error(err)
	s.Equal(nodegroup.ErrNodeGroupNotFound, err)
}

func TestDeleteGroupHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteGroupHandlerTestSuite))
}

// EnableGroupHandlerTestSuite 测试套件
type EnableGroupHandlerTestSuite struct {
	suite.Suite
	handler *EnableGroupHandler
	repo    *MockNodeGroupRepository
	ctx     context.Context
}

func (s *EnableGroupHandlerTestSuite) SetupTest() {
	s.repo = NewMockNodeGroupRepository()
	s.handler = NewEnableGroupHandler(s.repo)
	s.ctx = context.Background()

	// 预创建一个节点组
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID:    1,
		Name:      "Test Group",
		Type:      nodegroup.NodeGroupTypeEntry,
		IsEnabled: true,
	})
}

func (s *EnableGroupHandlerTestSuite) TestDisableGroup() {
	err := s.handler.Handle(s.ctx, EnableGroupCommand{ID: 1, Enabled: false})
	s.NoError(err)

	// 验证已禁用
	group, _ := s.repo.FindByID(s.ctx, 1)
	s.False(group.IsEnabled)
}

func (s *EnableGroupHandlerTestSuite) TestEnableGroup() {
	// 先禁用
	s.handler.Handle(s.ctx, EnableGroupCommand{ID: 1, Enabled: false})

	// 再启用
	err := s.handler.Handle(s.ctx, EnableGroupCommand{ID: 1, Enabled: true})
	s.NoError(err)

	// 验证已启用
	group, _ := s.repo.FindByID(s.ctx, 1)
	s.True(group.IsEnabled)
}

func (s *EnableGroupHandlerTestSuite) TestEnableGroupNotFound() {
	err := s.handler.Handle(s.ctx, EnableGroupCommand{ID: 999, Enabled: true})
	s.Error(err)
	s.Equal(nodegroup.ErrNodeGroupNotFound, err)
}

func TestEnableGroupHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(EnableGroupHandlerTestSuite))
}
