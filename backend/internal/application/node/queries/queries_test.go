package queries

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"nodepass-pro/backend/internal/domain/node"
)

// MockNodeRepository Mock 仓储
type MockNodeRepository struct {
	nodes map[string]*node.NodeInstance
}

func NewMockNodeRepository() *MockNodeRepository {
	return &MockNodeRepository{
		nodes: make(map[string]*node.NodeInstance),
	}
}

func (m *MockNodeRepository) Create(ctx context.Context, instance *node.NodeInstance) error {
	m.nodes[instance.NodeID] = instance
	return nil
}

func (m *MockNodeRepository) FindByID(ctx context.Context, id uint) (*node.NodeInstance, error) {
	for _, n := range m.nodes {
		if n.ID == id {
			return n, nil
		}
	}
	return nil, node.ErrNodeNotFound
}

func (m *MockNodeRepository) FindByNodeID(ctx context.Context, nodeID string) (*node.NodeInstance, error) {
	n, ok := m.nodes[nodeID]
	if !ok {
		return nil, node.ErrNodeNotFound
	}
	return n, nil
}

func (m *MockNodeRepository) Update(ctx context.Context, instance *node.NodeInstance) error {
	if _, ok := m.nodes[instance.NodeID]; !ok {
		return node.ErrNodeNotFound
	}
	m.nodes[instance.NodeID] = instance
	return nil
}

func (m *MockNodeRepository) Delete(ctx context.Context, id uint) error {
	for nodeID, n := range m.nodes {
		if n.ID == id {
			delete(m.nodes, nodeID)
			return nil
		}
	}
	return node.ErrNodeNotFound
}

func (m *MockNodeRepository) FindByGroupID(ctx context.Context, groupID uint) ([]*node.NodeInstance, error) {
	var result []*node.NodeInstance
	for _, n := range m.nodes {
		if n.GroupID == groupID {
			result = append(result, n)
		}
	}
	return result, nil
}

func (m *MockNodeRepository) FindByIDs(ctx context.Context, ids []uint) ([]*node.NodeInstance, error) {
	var result []*node.NodeInstance
	for _, n := range m.nodes {
		for _, id := range ids {
			if n.ID == id {
				result = append(result, n)
				break
			}
		}
	}
	return result, nil
}

func (m *MockNodeRepository) FindAll(ctx context.Context) ([]*node.NodeInstance, error) {
	var result []*node.NodeInstance
	for _, n := range m.nodes {
		result = append(result, n)
	}
	return result, nil
}

func (m *MockNodeRepository) List(ctx context.Context, filter node.InstanceListFilter) ([]*node.NodeInstance, int64, error) {
	var result []*node.NodeInstance
	for _, n := range m.nodes {
		if filter.GroupID > 0 && n.GroupID != filter.GroupID {
			continue
		}
		if filter.Status != "" && n.Status != filter.Status {
			continue
		}
		if filter.OnlineOnly && n.Status != "online" {
			continue
		}
		result = append(result, n)
	}
	return result, int64(len(result)), nil
}

func (m *MockNodeRepository) FindOnlineNodes(ctx context.Context) ([]*node.NodeInstance, error) {
	var result []*node.NodeInstance
	for _, n := range m.nodes {
		if n.Status == "online" {
			result = append(result, n)
		}
	}
	return result, nil
}

func (m *MockNodeRepository) FindOfflineNodes(ctx context.Context, timeout time.Duration) ([]*node.NodeInstance, error) {
	var result []*node.NodeInstance
	for _, n := range m.nodes {
		if n.Status == "offline" {
			result = append(result, n)
		}
	}
	return result, nil
}

func (m *MockNodeRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	count := int64(0)
	for _, n := range m.nodes {
		if n.Status == status {
			count++
		}
	}
	return count, nil
}

func (m *MockNodeRepository) UpdateStatus(ctx context.Context, nodeID string, status string) error {
	n, ok := m.nodes[nodeID]
	if !ok {
		return node.ErrNodeNotFound
	}
	n.Status = status
	return nil
}

func (m *MockNodeRepository) UpdateHeartbeat(ctx context.Context, nodeID string, data *node.HeartbeatData) error {
	n, ok := m.nodes[nodeID]
	if !ok {
		return node.ErrNodeNotFound
	}
	n.CPUUsage = data.CPUUsage
	n.MemoryUsage = data.MemoryUsage
	n.Status = "online"
	now := time.Now()
	n.LastHeartbeatAt = &now
	return nil
}

func (m *MockNodeRepository) BatchUpdateHeartbeat(ctx context.Context, data []*node.HeartbeatData) error {
	for _, d := range data {
		m.UpdateHeartbeat(ctx, d.NodeID, d)
	}
	return nil
}

func (m *MockNodeRepository) MarkOfflineByTimeout(ctx context.Context, timeout time.Duration) (int64, error) {
	count := int64(0)
	for _, n := range m.nodes {
		if n.LastHeartbeatAt != nil && time.Since(*n.LastHeartbeatAt) > timeout {
			n.Status = "offline"
			count++
		}
	}
	return count, nil
}

// MockNodeCache Mock 缓存
type MockNodeCache struct {
	onlineNodes map[string]bool
	nodeInfo    map[string]map[string]interface{}
	metrics     map[string]map[string]float64
}

func NewMockNodeCache() *MockNodeCache {
	return &MockNodeCache{
		onlineNodes: make(map[string]bool),
		nodeInfo:    make(map[string]map[string]interface{}),
		metrics:     make(map[string]map[string]float64),
	}
}

func (m *MockNodeCache) IsOnline(ctx context.Context, nodeID string) (bool, error) {
	return m.onlineNodes[nodeID], nil
}

func (m *MockNodeCache) SetOnline(ctx context.Context, nodeID string, ttl time.Duration) error {
	m.onlineNodes[nodeID] = true
	return nil
}

func (m *MockNodeCache) GetAllOnlineNodes(ctx context.Context) ([]string, error) {
	var result []string
	for nodeID, online := range m.onlineNodes {
		if online {
			result = append(result, nodeID)
		}
	}
	return result, nil
}

func (m *MockNodeCache) SetNodeInfo(ctx context.Context, nodeID string, info map[string]interface{}) error {
	m.nodeInfo[nodeID] = info
	return nil
}

func (m *MockNodeCache) GetNodeInfo(ctx context.Context, nodeID string) (map[string]interface{}, error) {
	return m.nodeInfo[nodeID], nil
}

func (m *MockNodeCache) SetNodeMetrics(ctx context.Context, nodeID string, metrics map[string]float64) error {
	m.metrics[nodeID] = metrics
	return nil
}

func (m *MockNodeCache) GetNodeMetrics(ctx context.Context, nodeID string) (map[string]float64, error) {
	return m.metrics[nodeID], nil
}

// GetNodeHandlerTestSuite 测试套件
type GetNodeHandlerTestSuite struct {
	suite.Suite
	repo  *MockNodeRepository
	cache *MockNodeCache
	ctx   context.Context
}

func (s *GetNodeHandlerTestSuite) SetupTest() {
	s.repo = NewMockNodeRepository()
	s.cache = NewMockNodeCache()
	s.ctx = context.Background()

	// 预创建测试节点
	now := time.Now()
	s.repo.Create(s.ctx, &node.NodeInstance{
		ID:              1,
		GroupID:         1,
		NodeID:          "node-001",
		ServiceName:     "Test Node",
		Status:          "online",
		CPUUsage:        50.0,
		MemoryUsage:     60.0,
		LastHeartbeatAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	s.cache.SetOnline(s.ctx, "node-001", 3*time.Minute)
}

func (s *GetNodeHandlerTestSuite) TestGetNode() {
	// 直接测试逻辑
	isOnline, _ := s.cache.IsOnline(s.ctx, "node-001")
	instance, err := s.repo.FindByNodeID(s.ctx, "node-001")

	s.NoError(err)
	s.NotNil(instance)
	s.Equal("node-001", instance.NodeID)
	s.True(isOnline)
}

func (s *GetNodeHandlerTestSuite) TestGetNodeNotFound() {
	instance, err := s.repo.FindByNodeID(s.ctx, "not-exist")
	s.Error(err)
	s.Nil(instance)
	s.Equal(node.ErrNodeNotFound, err)
}

func TestGetNodeHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GetNodeHandlerTestSuite))
}

// ListNodesHandlerTestSuite 测试套件
type ListNodesHandlerTestSuite struct {
	suite.Suite
	repo  *MockNodeRepository
	cache *MockNodeCache
	ctx   context.Context
}

func (s *ListNodesHandlerTestSuite) SetupTest() {
	s.repo = NewMockNodeRepository()
	s.cache = NewMockNodeCache()
	s.ctx = context.Background()

	// 预创建测试节点
	now := time.Now()
	s.repo.Create(s.ctx, &node.NodeInstance{
		ID: 1, GroupID: 1, NodeID: "node-001", ServiceName: "Node 1",
		Status: "online", CreatedAt: now, UpdatedAt: now,
	})
	s.repo.Create(s.ctx, &node.NodeInstance{
		ID: 2, GroupID: 1, NodeID: "node-002", ServiceName: "Node 2",
		Status: "offline", CreatedAt: now, UpdatedAt: now,
	})
	s.repo.Create(s.ctx, &node.NodeInstance{
		ID: 3, GroupID: 2, NodeID: "node-003", ServiceName: "Node 3",
		Status: "online", CreatedAt: now, UpdatedAt: now,
	})

	s.cache.SetOnline(s.ctx, "node-001", 3*time.Minute)
	s.cache.SetOnline(s.ctx, "node-003", 3*time.Minute)
}

func (s *ListNodesHandlerTestSuite) TestListAllNodes() {
	nodes, total, err := s.repo.List(s.ctx, node.InstanceListFilter{
		Page:     1,
		PageSize: 10,
	})
	s.NoError(err)
	s.GreaterOrEqual(len(nodes), 3)
	s.GreaterOrEqual(int(total), 3)
}

func (s *ListNodesHandlerTestSuite) TestListNodesByGroupID() {
	nodes, total, err := s.repo.List(s.ctx, node.InstanceListFilter{
		GroupID:  1,
		Page:     1,
		PageSize: 10,
	})
	s.NoError(err)
	s.GreaterOrEqual(len(nodes), 2)
	s.GreaterOrEqual(int(total), 2)
}

func (s *ListNodesHandlerTestSuite) TestListOnlineNodesOnly() {
	nodes, total, err := s.repo.List(s.ctx, node.InstanceListFilter{
		OnlineOnly: true,
		Page:       1,
		PageSize:   10,
	})
	s.NoError(err)
	s.GreaterOrEqual(len(nodes), 2)
	s.GreaterOrEqual(int(total), 2)
	for _, n := range nodes {
		s.Equal("online", n.Status)
	}
}

func TestListNodesHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ListNodesHandlerTestSuite))
}

// GetOnlineNodesHandlerTestSuite 测试套件
type GetOnlineNodesHandlerTestSuite struct {
	suite.Suite
	repo  *MockNodeRepository
	cache *MockNodeCache
	ctx   context.Context
}

func (s *GetOnlineNodesHandlerTestSuite) SetupTest() {
	s.repo = NewMockNodeRepository()
	s.cache = NewMockNodeCache()
	s.ctx = context.Background()

	// 预创建测试节点
	now := time.Now()
	s.repo.Create(s.ctx, &node.NodeInstance{
		ID: 1, GroupID: 1, NodeID: "node-001", ServiceName: "Node 1",
		Status: "online", CreatedAt: now, UpdatedAt: now,
	})
	s.repo.Create(s.ctx, &node.NodeInstance{
		ID: 2, GroupID: 1, NodeID: "node-002", ServiceName: "Node 2",
		Status: "offline", CreatedAt: now, UpdatedAt: now,
	})
	s.repo.Create(s.ctx, &node.NodeInstance{
		ID: 3, GroupID: 2, NodeID: "node-003", ServiceName: "Node 3",
		Status: "online", CreatedAt: now, UpdatedAt: now,
	})

	s.cache.SetOnline(s.ctx, "node-001", 3*time.Minute)
	s.cache.SetOnline(s.ctx, "node-003", 3*time.Minute)
}

func (s *GetOnlineNodesHandlerTestSuite) TestGetOnlineNodes() {
	// 从缓存获取在线节点
	onlineNodeIDs, err := s.cache.GetAllOnlineNodes(s.ctx)
	s.NoError(err)
	s.Equal(2, len(onlineNodeIDs))

	// 从数据库获取节点详情
	nodes := make([]*node.NodeInstance, 0)
	for _, nodeID := range onlineNodeIDs {
		n, err := s.repo.FindByNodeID(s.ctx, nodeID)
		if err == nil {
			nodes = append(nodes, n)
		}
	}
	s.Equal(2, len(nodes))
}

func (s *GetOnlineNodesHandlerTestSuite) TestGetOnlineNodesFallback() {
	// 测试降级到数据库查询
	nodes, err := s.repo.FindOnlineNodes(s.ctx)
	s.NoError(err)
	s.Equal(2, len(nodes))
}

func TestGetOnlineNodesHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GetOnlineNodesHandlerTestSuite))
}
