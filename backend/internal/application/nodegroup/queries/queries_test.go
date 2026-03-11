package queries

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"nodepass-pro/backend/internal/domain/nodegroup"
)

// MockNodeGroupRepository Mock 仓储（复用 commands 包的实现）
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
	if _, ok := m.groups[groupID]; !ok {
		return nil, nodegroup.ErrNodeGroupNotFound
	}
	return &nodegroup.NodeGroupStats{
		NodeGroupID: groupID,
		TotalNodes:  10,
		OnlineNodes: 8,
	}, nil
}

// GetGroupHandlerTestSuite 测试套件
type GetGroupHandlerTestSuite struct {
	suite.Suite
	handler *GetGroupHandler
	repo    *MockNodeGroupRepository
	ctx     context.Context
}

func (s *GetGroupHandlerTestSuite) SetupTest() {
	s.repo = NewMockNodeGroupRepository()
	s.handler = NewGetGroupHandler(s.repo)
	s.ctx = context.Background()

	// 预创建节点组
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 1,
		Name:   "Test Group",
		Type:   nodegroup.NodeGroupTypeEntry,
	})
}

func (s *GetGroupHandlerTestSuite) TestGetGroup() {
	group, err := s.handler.Handle(s.ctx, GetGroupQuery{ID: 1})
	s.NoError(err)
	s.NotNil(group)
	s.Equal(uint(1), group.ID)
	s.Equal("Test Group", group.Name)
}

func (s *GetGroupHandlerTestSuite) TestGetGroupNotFound() {
	group, err := s.handler.Handle(s.ctx, GetGroupQuery{ID: 999})
	s.Error(err)
	s.Equal(nodegroup.ErrNodeGroupNotFound, err)
	s.Nil(group)
}

func TestGetGroupHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GetGroupHandlerTestSuite))
}

// ListGroupsHandlerTestSuite 测试套件
type ListGroupsHandlerTestSuite struct {
	suite.Suite
	handler *ListGroupsHandler
	repo    *MockNodeGroupRepository
	ctx     context.Context
}

func (s *ListGroupsHandlerTestSuite) SetupTest() {
	s.repo = NewMockNodeGroupRepository()
	s.handler = NewListGroupsHandler(s.repo)
	s.ctx = context.Background()

	// 预创建多个节点组
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID:    1,
		Name:      "Entry Group 1",
		Type:      nodegroup.NodeGroupTypeEntry,
		IsEnabled: true,
	})
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID:    1,
		Name:      "Exit Group 1",
		Type:      nodegroup.NodeGroupTypeExit,
		IsEnabled: true,
	})
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID:    2,
		Name:      "Entry Group 2",
		Type:      nodegroup.NodeGroupTypeEntry,
		IsEnabled: false,
	})
}

func (s *ListGroupsHandlerTestSuite) TestListAllGroups() {
	result, err := s.handler.Handle(s.ctx, ListGroupsQuery{
		Page:     1,
		PageSize: 10,
	})
	s.NoError(err)
	s.NotNil(result)
	s.Equal(int64(3), result.Total)
	s.Len(result.Groups, 3)
}

func (s *ListGroupsHandlerTestSuite) TestListGroupsByUserID() {
	result, err := s.handler.Handle(s.ctx, ListGroupsQuery{
		UserID:   1,
		Page:     1,
		PageSize: 10,
	})
	s.NoError(err)
	s.Equal(int64(2), result.Total)
	s.Len(result.Groups, 2)
}

func (s *ListGroupsHandlerTestSuite) TestListGroupsByType() {
	result, err := s.handler.Handle(s.ctx, ListGroupsQuery{
		Type:     nodegroup.NodeGroupTypeEntry,
		Page:     1,
		PageSize: 10,
	})
	s.NoError(err)
	s.Equal(int64(2), result.Total)
	s.Len(result.Groups, 2)
}

func (s *ListGroupsHandlerTestSuite) TestListEnabledGroupsOnly() {
	result, err := s.handler.Handle(s.ctx, ListGroupsQuery{
		EnabledOnly: true,
		Page:        1,
		PageSize:    10,
	})
	s.NoError(err)
	s.Equal(int64(2), result.Total)
	s.Len(result.Groups, 2)
}

func (s *ListGroupsHandlerTestSuite) TestListGroupsWithMultipleFilters() {
	result, err := s.handler.Handle(s.ctx, ListGroupsQuery{
		UserID:      1,
		Type:        nodegroup.NodeGroupTypeEntry,
		EnabledOnly: true,
		Page:        1,
		PageSize:    10,
	})
	s.NoError(err)
	s.Equal(int64(1), result.Total)
	s.Len(result.Groups, 1)
	s.Equal("Entry Group 1", result.Groups[0].Name)
}

func TestListGroupsHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ListGroupsHandlerTestSuite))
}

// GetGroupStatsHandlerTestSuite 测试套件
type GetGroupStatsHandlerTestSuite struct {
	suite.Suite
	handler *GetGroupStatsHandler
	repo    *MockNodeGroupRepository
	ctx     context.Context
}

func (s *GetGroupStatsHandlerTestSuite) SetupTest() {
	s.repo = NewMockNodeGroupRepository()
	s.handler = NewGetGroupStatsHandler(s.repo)
	s.ctx = context.Background()

	// 预创建节点组
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 1,
		Name:   "Test Group",
		Type:   nodegroup.NodeGroupTypeEntry,
	})
}

func (s *GetGroupStatsHandlerTestSuite) TestGetGroupStats() {
	stats, err := s.handler.Handle(s.ctx, GetGroupStatsQuery{GroupID: 1})
	s.NoError(err)
	s.NotNil(stats)
	s.Equal(uint(1), stats.NodeGroupID)
	s.Equal(10, stats.TotalNodes)
	s.Equal(8, stats.OnlineNodes)
}

func (s *GetGroupStatsHandlerTestSuite) TestGetGroupStatsNotFound() {
	stats, err := s.handler.Handle(s.ctx, GetGroupStatsQuery{GroupID: 999})
	s.Error(err)
	s.Equal(nodegroup.ErrNodeGroupNotFound, err)
	s.Nil(stats)
}

func TestGetGroupStatsHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GetGroupStatsHandlerTestSuite))
}
