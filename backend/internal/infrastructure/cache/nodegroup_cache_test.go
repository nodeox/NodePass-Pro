package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"

	"nodepass-pro/backend/internal/domain/nodegroup"
)

// NodeGroupCacheTestSuite 测试套件
type NodeGroupCacheTestSuite struct {
	suite.Suite
	cache  *NodeGroupCache
	client *redis.Client
	ctx    context.Context
}

func (s *NodeGroupCacheTestSuite) SetupTest() {
	// 使用 Redis 测试实例
	s.client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试数据库
	})

	// 测试连接
	err := s.client.Ping(context.Background()).Err()
	if err != nil {
		s.T().Skip("Redis not available, skipping cache tests")
		return
	}

	s.cache = NewNodeGroupCache(s.client)
	s.ctx = context.Background()

	// 清空测试数据
	s.client.FlushDB(s.ctx)
}

func (s *NodeGroupCacheTestSuite) TearDownTest() {
	if s.client != nil {
		s.client.FlushDB(s.ctx)
		s.client.Close()
	}
}

func (s *NodeGroupCacheTestSuite) TestSetAndGetGroup() {
	group := &nodegroup.NodeGroup{
		ID:          1,
		UserID:      100,
		Name:        "Test Group",
		Type:        nodegroup.NodeGroupTypeEntry,
		Description: "Test description",
		IsEnabled:   true,
		Config: nodegroup.NodeGroupConfig{
			AllowedProtocols: []string{"tcp", "udp"},
			PortRange:        nodegroup.PortRange{Start: 10000, End: 20000},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 设置缓存
	err := s.cache.SetGroup(s.ctx, group)
	s.NoError(err)

	// 获取缓存
	cached, err := s.cache.GetGroup(s.ctx, 1)
	s.NoError(err)
	s.NotNil(cached)
	s.Equal(group.ID, cached.ID)
	s.Equal(group.Name, cached.Name)
	s.Equal(group.Type, cached.Type)
}

func (s *NodeGroupCacheTestSuite) TestGetGroupNotFound() {
	cached, err := s.cache.GetGroup(s.ctx, 999)
	s.NoError(err)
	s.Nil(cached) // 缓存未命中
}

func (s *NodeGroupCacheTestSuite) TestDeleteGroup() {
	group := &nodegroup.NodeGroup{
		ID:        1,
		UserID:    100,
		Name:      "Test Group",
		Type:      nodegroup.NodeGroupTypeEntry,
		IsEnabled: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 设置缓存
	s.cache.SetGroup(s.ctx, group)

	// 删除缓存
	err := s.cache.DeleteGroup(s.ctx, 1)
	s.NoError(err)

	// 验证已删除
	cached, _ := s.cache.GetGroup(s.ctx, 1)
	s.Nil(cached)
}

func (s *NodeGroupCacheTestSuite) TestSetAndGetGroupList() {
	groups := []*nodegroup.NodeGroup{
		{
			ID:        1,
			UserID:    100,
			Name:      "Group 1",
			Type:      nodegroup.NodeGroupTypeEntry,
			IsEnabled: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        2,
			UserID:    100,
			Name:      "Group 2",
			Type:      nodegroup.NodeGroupTypeEntry,
			IsEnabled: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// 设置列表缓存
	err := s.cache.SetGroupList(s.ctx, 100, nodegroup.NodeGroupTypeEntry, groups)
	s.NoError(err)

	// 获取列表缓存
	cached, err := s.cache.GetGroupList(s.ctx, 100, nodegroup.NodeGroupTypeEntry)
	s.NoError(err)
	s.NotNil(cached)
	s.Len(cached, 2)
	s.Equal("Group 1", cached[0].Name)
}

func (s *NodeGroupCacheTestSuite) TestGetGroupListNotFound() {
	cached, err := s.cache.GetGroupList(s.ctx, 999, nodegroup.NodeGroupTypeEntry)
	s.NoError(err)
	s.Nil(cached)
}

func (s *NodeGroupCacheTestSuite) TestDeleteGroupList() {
	groups := []*nodegroup.NodeGroup{
		{ID: 1, UserID: 100, Name: "Group 1", Type: nodegroup.NodeGroupTypeEntry,
			IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	// 设置列表缓存
	s.cache.SetGroupList(s.ctx, 100, nodegroup.NodeGroupTypeEntry, groups)

	// 删除列表缓存
	err := s.cache.DeleteGroupList(s.ctx, 100, nodegroup.NodeGroupTypeEntry)
	s.NoError(err)

	// 验证已删除
	cached, _ := s.cache.GetGroupList(s.ctx, 100, nodegroup.NodeGroupTypeEntry)
	s.Nil(cached)
}

func (s *NodeGroupCacheTestSuite) TestDeleteUserGroupLists() {
	// 设置多个列表缓存
	groups := []*nodegroup.NodeGroup{
		{ID: 1, UserID: 100, Name: "Group 1", Type: nodegroup.NodeGroupTypeEntry,
			IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	s.cache.SetGroupList(s.ctx, 100, nodegroup.NodeGroupTypeEntry, groups)
	s.cache.SetGroupList(s.ctx, 100, nodegroup.NodeGroupTypeExit, groups)

	// 删除用户的所有列表缓存
	err := s.cache.DeleteUserGroupLists(s.ctx, 100)
	s.NoError(err)

	// 验证都已删除
	cached1, _ := s.cache.GetGroupList(s.ctx, 100, nodegroup.NodeGroupTypeEntry)
	s.Nil(cached1)
	cached2, _ := s.cache.GetGroupList(s.ctx, 100, nodegroup.NodeGroupTypeExit)
	s.Nil(cached2)
}

func (s *NodeGroupCacheTestSuite) TestSetAndGetStats() {
	stats := &nodegroup.NodeGroupStats{
		NodeGroupID:      1,
		TotalNodes:       10,
		OnlineNodes:      8,
		TotalTrafficIn:   1000000,
		TotalTrafficOut:  2000000,
		TotalConnections: 50,
		UpdatedAt:        time.Now(),
	}

	// 设置统计缓存
	err := s.cache.SetStats(s.ctx, stats)
	s.NoError(err)

	// 获取统计缓存
	cached, err := s.cache.GetStats(s.ctx, 1)
	s.NoError(err)
	s.NotNil(cached)
	s.Equal(10, cached.TotalNodes)
	s.Equal(8, cached.OnlineNodes)
	s.Equal(int64(1000000), cached.TotalTrafficIn)
}

func (s *NodeGroupCacheTestSuite) TestGetStatsNotFound() {
	cached, err := s.cache.GetStats(s.ctx, 999)
	s.NoError(err)
	s.Nil(cached)
}

func (s *NodeGroupCacheTestSuite) TestDeleteStats() {
	stats := &nodegroup.NodeGroupStats{
		NodeGroupID: 1,
		TotalNodes:  10,
		UpdatedAt:   time.Now(),
	}

	// 设置统计缓存
	s.cache.SetStats(s.ctx, stats)

	// 删除统计缓存
	err := s.cache.DeleteStats(s.ctx, 1)
	s.NoError(err)

	// 验证已删除
	cached, _ := s.cache.GetStats(s.ctx, 1)
	s.Nil(cached)
}

func (s *NodeGroupCacheTestSuite) TestNodeCount() {
	// 设置节点数量
	err := s.cache.SetNodeCount(s.ctx, 1, 10)
	s.NoError(err)

	// 获取节点数量
	count, err := s.cache.GetNodeCount(s.ctx, 1)
	s.NoError(err)
	s.Equal(10, count)

	// 增加节点数量
	err = s.cache.IncrementNodeCount(s.ctx, 1, 5)
	s.NoError(err)

	// 验证增加后的数量
	count, err = s.cache.GetNodeCount(s.ctx, 1)
	s.NoError(err)
	s.Equal(15, count)

	// 减少节点数量
	err = s.cache.IncrementNodeCount(s.ctx, 1, -3)
	s.NoError(err)

	// 验证减少后的数量
	count, err = s.cache.GetNodeCount(s.ctx, 1)
	s.NoError(err)
	s.Equal(12, count)
}

func (s *NodeGroupCacheTestSuite) TestGetNodeCountNotFound() {
	count, err := s.cache.GetNodeCount(s.ctx, 999)
	s.NoError(err)
	s.Equal(0, count) // 未找到返回 0
}

func TestNodeGroupCacheTestSuite(t *testing.T) {
	suite.Run(t, new(NodeGroupCacheTestSuite))
}
