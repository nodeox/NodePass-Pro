package commands_test

import (
	"context"
	"testing"
	"time"

	"nodepass-pro/backend/internal/application/node/commands"
	"nodepass-pro/backend/internal/domain/node"
	"nodepass-pro/backend/internal/infrastructure/cache"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockNodeRepository 模拟节点仓储
type MockNodeRepository struct {
	mock.Mock
}

func (m *MockNodeRepository) Create(ctx context.Context, instance *node.NodeInstance) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *MockNodeRepository) FindByID(ctx context.Context, id uint) (*node.NodeInstance, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*node.NodeInstance), args.Error(1)
}

func (m *MockNodeRepository) FindByNodeID(ctx context.Context, nodeID string) (*node.NodeInstance, error) {
	args := m.Called(ctx, nodeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*node.NodeInstance), args.Error(1)
}

func (m *MockNodeRepository) Update(ctx context.Context, instance *node.NodeInstance) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *MockNodeRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockNodeRepository) FindByGroupID(ctx context.Context, groupID uint) ([]*node.NodeInstance, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*node.NodeInstance), args.Error(1)
}

func (m *MockNodeRepository) FindByIDs(ctx context.Context, ids []uint) ([]*node.NodeInstance, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*node.NodeInstance), args.Error(1)
}

func (m *MockNodeRepository) FindAll(ctx context.Context) ([]*node.NodeInstance, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*node.NodeInstance), args.Error(1)
}

func (m *MockNodeRepository) List(ctx context.Context, filter node.InstanceListFilter) ([]*node.NodeInstance, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*node.NodeInstance), args.Get(1).(int64), args.Error(2)
}

func (m *MockNodeRepository) FindOnlineNodes(ctx context.Context) ([]*node.NodeInstance, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*node.NodeInstance), args.Error(1)
}

func (m *MockNodeRepository) FindOfflineNodes(ctx context.Context, timeout time.Duration) ([]*node.NodeInstance, error) {
	args := m.Called(ctx, timeout)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*node.NodeInstance), args.Error(1)
}

func (m *MockNodeRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNodeRepository) UpdateStatus(ctx context.Context, nodeID string, status string) error {
	args := m.Called(ctx, nodeID, status)
	return args.Error(0)
}

func (m *MockNodeRepository) UpdateHeartbeat(ctx context.Context, nodeID string, data *node.HeartbeatData) error {
	args := m.Called(ctx, nodeID, data)
	return args.Error(0)
}

func (m *MockNodeRepository) BatchUpdateHeartbeat(ctx context.Context, data []*node.HeartbeatData) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockNodeRepository) MarkOfflineByTimeout(ctx context.Context, timeout time.Duration) (int64, error) {
	args := m.Called(ctx, timeout)
	return args.Get(0).(int64), args.Error(1)
}

// TestHeartbeatHandler_Handle_Success 测试心跳处理成功
func TestHeartbeatHandler_Handle_Success(t *testing.T) {
	// 创建 mock
	mockRepo := new(MockNodeRepository)

	// 创建 Redis 客户端（使用真实 Redis 或 miniredis）
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试数据库
	})
	defer redisClient.Close()

	// 清理测试数据
	ctx := context.Background()
	redisClient.FlushDB(ctx)

	nodeCache := cache.NewNodeCache(redisClient)
	heartbeatBuffer := cache.NewHeartbeatBuffer(redisClient)

	// 创建处理器
	handler := commands.NewHeartbeatHandler(mockRepo, nodeCache, heartbeatBuffer)

	// 准备测试数据
	cmd := commands.HeartbeatCommand{
		NodeID:            "test-node-001",
		Status:            "online",
		CPUUsage:          45.5,
		MemoryUsage:       60.2,
		DiskUsage:         75.8,
		NetworkInBytes:    1024000,
		NetworkOutBytes:   2048000,
		ActiveConnections: 10,
		ConfigVersion:     1,
	}

	// 设置 mock 期望
	mockRepo.On("FindByNodeID", ctx, "test-node-001").Return(&node.NodeInstance{
		ID:            1,
		NodeID:        "test-node-001",
		ConfigVersion: 2, // 服务器配置版本更高
		Status:        "online",
	}, nil)

	// 执行测试
	result, err := handler.Handle(ctx, cmd)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.ConfigUpdated)
	assert.Equal(t, 2, result.NewConfigVersion)

	// 验证 mock 调用
	mockRepo.AssertExpectations(t)

	// 验证 Redis 数据
	// 1. 检查在线状态
	isOnline, err := nodeCache.IsOnline(ctx, "test-node-001")
	assert.NoError(t, err)
	assert.True(t, isOnline)

	// 2. 检查心跳缓冲区
	heartbeats, err := heartbeatBuffer.PopBatch(ctx, "test-node-001", 10)
	assert.NoError(t, err)
	assert.Len(t, heartbeats, 1)
	assert.Equal(t, "test-node-001", heartbeats[0].NodeID)
	assert.Equal(t, 45.5, heartbeats[0].CPUUsage)
}

// TestHeartbeatHandler_Handle_NodeNotFound 测试节点不存在
func TestHeartbeatHandler_Handle_NodeNotFound(t *testing.T) {
	mockRepo := new(MockNodeRepository)

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})
	defer redisClient.Close()

	ctx := context.Background()
	redisClient.FlushDB(ctx)

	nodeCache := cache.NewNodeCache(redisClient)
	heartbeatBuffer := cache.NewHeartbeatBuffer(redisClient)

	handler := commands.NewHeartbeatHandler(mockRepo, nodeCache, heartbeatBuffer)

	cmd := commands.HeartbeatCommand{
		NodeID:         "non-existent-node",
		ConfigVersion:  1,
	}

	// 节点不存在
	mockRepo.On("FindByNodeID", ctx, "non-existent-node").Return(nil, node.ErrNodeNotFound)

	result, err := handler.Handle(ctx, cmd)

	// 应该返回默认结果，不报错
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.ConfigUpdated)
	assert.Equal(t, 0, result.NewConfigVersion)

	mockRepo.AssertExpectations(t)
}

// TestHeartbeatHandler_FlushHeartbeats 测试批量刷新
func TestHeartbeatHandler_FlushHeartbeats(t *testing.T) {
	mockRepo := new(MockNodeRepository)

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})
	defer redisClient.Close()

	ctx := context.Background()
	redisClient.FlushDB(ctx)

	nodeCache := cache.NewNodeCache(redisClient)
	heartbeatBuffer := cache.NewHeartbeatBuffer(redisClient)

	handler := commands.NewHeartbeatHandler(mockRepo, nodeCache, heartbeatBuffer)

	// 准备测试数据：添加 2 个节点的心跳
	nodeCache.SetOnline(ctx, "node-001", 3*time.Minute)
	nodeCache.SetOnline(ctx, "node-002", 3*time.Minute)

	heartbeatBuffer.Push(ctx, &cache.HeartbeatData{
		NodeID:      "node-001",
		CPUUsage:    50.0,
		MemoryUsage: 60.0,
		Timestamp:   time.Now(),
	})

	heartbeatBuffer.Push(ctx, &cache.HeartbeatData{
		NodeID:      "node-002",
		CPUUsage:    40.0,
		MemoryUsage: 55.0,
		Timestamp:   time.Now(),
	})

	// 设置 mock 期望
	mockRepo.On("BatchUpdateHeartbeat", ctx, mock.AnythingOfType("[]*node.HeartbeatData")).Return(nil)

	// 执行刷新
	err := handler.FlushHeartbeats(ctx)

	// 验证
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestHeartbeatHandler_DetectOfflineNodes 测试离线检测
func TestHeartbeatHandler_DetectOfflineNodes(t *testing.T) {
	mockRepo := new(MockNodeRepository)

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})
	defer redisClient.Close()

	ctx := context.Background()
	redisClient.FlushDB(ctx)

	nodeCache := cache.NewNodeCache(redisClient)
	heartbeatBuffer := cache.NewHeartbeatBuffer(redisClient)

	handler := commands.NewHeartbeatHandler(mockRepo, nodeCache, heartbeatBuffer)

	// 准备测试数据
	// node-001: 在线（Redis 中有记录）
	// node-002: 离线（Redis 中无记录，但数据库中状态为 online）
	nodeCache.SetOnline(ctx, "node-001", 3*time.Minute)

	mockRepo.On("FindAll", ctx).Return([]*node.NodeInstance{
		{ID: 1, NodeID: "node-001", Status: "online"},
		{ID: 2, NodeID: "node-002", Status: "online"}, // 应该被标记为离线
	}, nil)

	mockRepo.On("UpdateStatus", ctx, "node-002", "offline").Return(nil)

	// 执行检测
	count, err := handler.DetectOfflineNodes(ctx)

	// 验证
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
	mockRepo.AssertExpectations(t)
}
