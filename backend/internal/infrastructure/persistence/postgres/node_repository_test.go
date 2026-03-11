package postgres_test

import (
	"context"
	"testing"
	"time"

	"nodepass-pro/backend/internal/domain/node"
	"nodepass-pro/backend/internal/infrastructure/persistence/postgres"
	"nodepass-pro/backend/internal/models"

	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NodeRepositoryTestSuite 节点仓储测试套件
type NodeRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo node.InstanceRepository
}

// SetupTest 每个测试前执行
func (suite *NodeRepositoryTestSuite) SetupTest() {
	// 使用 SQLite 内存数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	// 自动迁移
	err = db.AutoMigrate(&models.NodeInstance{})
	suite.Require().NoError(err)

	suite.db = db
	suite.repo = postgres.NewNodeInstanceRepository(db)
}

// TearDownTest 每个测试后执行
func (suite *NodeRepositoryTestSuite) TearDownTest() {
	sqlDB, err := suite.db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

// TestCreate 测试创建节点
func (suite *NodeRepositoryTestSuite) TestCreate() {
	ctx := context.Background()

	instance := &node.NodeInstance{
		GroupID:     1,
		NodeID:      "test-node-001",
		ServiceName: "Test Node",
		Status:      "online",
	}

	err := suite.repo.Create(ctx, instance)
	suite.NoError(err)
	suite.NotZero(instance.ID)
	suite.NotZero(instance.CreatedAt)
}

// TestFindByID 测试根据 ID 查找
func (suite *NodeRepositoryTestSuite) TestFindByID() {
	ctx := context.Background()

	// 创建测试数据
	instance := &node.NodeInstance{
		GroupID:     1,
		NodeID:      "test-node-002",
		ServiceName: "Test Node 2",
		Status:      "online",
	}
	suite.repo.Create(ctx, instance)

	// 查找
	found, err := suite.repo.FindByID(ctx, instance.ID)
	suite.NoError(err)
	suite.NotNil(found)
	suite.Equal(instance.NodeID, found.NodeID)
	suite.Equal(instance.ServiceName, found.ServiceName)
}

// TestFindByID_NotFound 测试查找不存在的节点
func (suite *NodeRepositoryTestSuite) TestFindByID_NotFound() {
	ctx := context.Background()

	found, err := suite.repo.FindByID(ctx, 99999)
	suite.Error(err)
	suite.Nil(found)
	suite.Equal(node.ErrNodeNotFound, err)
}

// TestFindByNodeID 测试根据 NodeID 查找
func (suite *NodeRepositoryTestSuite) TestFindByNodeID() {
	ctx := context.Background()

	// 创建测试数据
	instance := &node.NodeInstance{
		GroupID:     1,
		NodeID:      "test-node-003",
		ServiceName: "Test Node 3",
		Status:      "online",
	}
	suite.repo.Create(ctx, instance)

	// 查找
	found, err := suite.repo.FindByNodeID(ctx, "test-node-003")
	suite.NoError(err)
	suite.NotNil(found)
	suite.Equal(instance.ID, found.ID)
	suite.Equal("Test Node 3", found.ServiceName)
}

// TestUpdate 测试更新节点
func (suite *NodeRepositoryTestSuite) TestUpdate() {
	ctx := context.Background()

	// 创建测试数据
	instance := &node.NodeInstance{
		GroupID:     1,
		NodeID:      "test-node-004",
		ServiceName: "Test Node 4",
		Status:      "online",
	}
	suite.repo.Create(ctx, instance)

	// 更新
	instance.ServiceName = "Updated Node 4"
	instance.Status = "offline"
	err := suite.repo.Update(ctx, instance)
	suite.NoError(err)

	// 验证
	found, _ := suite.repo.FindByID(ctx, instance.ID)
	suite.Equal("Updated Node 4", found.ServiceName)
	suite.Equal("offline", found.Status)
}

// TestDelete 测试删除节点
func (suite *NodeRepositoryTestSuite) TestDelete() {
	ctx := context.Background()

	// 创建测试数据
	instance := &node.NodeInstance{
		GroupID:     1,
		NodeID:      "test-node-005",
		ServiceName: "Test Node 5",
		Status:      "online",
	}
	suite.repo.Create(ctx, instance)

	// 删除
	err := suite.repo.Delete(ctx, instance.ID)
	suite.NoError(err)

	// 验证已删除
	found, err := suite.repo.FindByID(ctx, instance.ID)
	suite.Error(err)
	suite.Nil(found)
}

// TestFindByGroupID 测试根据组 ID 查找
func (suite *NodeRepositoryTestSuite) TestFindByGroupID() {
	ctx := context.Background()

	// 创建测试数据
	for i := 1; i <= 3; i++ {
		instance := &node.NodeInstance{
			GroupID:     1,
			NodeID:      "node-group-1-" + string(rune(i)),
			ServiceName: "Node " + string(rune(i)),
			Status:      "online",
		}
		suite.repo.Create(ctx, instance)
	}

	// 创建其他组的节点
	instance := &node.NodeInstance{
		GroupID:     2,
		NodeID:      "node-group-2-1",
		ServiceName: "Node Group 2",
		Status:      "online",
	}
	suite.repo.Create(ctx, instance)

	// 查找组 1 的节点
	nodes, err := suite.repo.FindByGroupID(ctx, 1)
	suite.NoError(err)
	suite.Len(nodes, 3)
}

// TestFindAll 测试查找所有节点
func (suite *NodeRepositoryTestSuite) TestFindAll() {
	ctx := context.Background()

	// 创建测试数据
	for i := 1; i <= 5; i++ {
		instance := &node.NodeInstance{
			GroupID:     1,
			NodeID:      "node-all-" + string(rune(i)),
			ServiceName: "Node " + string(rune(i)),
			Status:      "online",
		}
		suite.repo.Create(ctx, instance)
	}

	// 查找所有
	nodes, err := suite.repo.FindAll(ctx)
	suite.NoError(err)
	suite.GreaterOrEqual(len(nodes), 5)
}

// TestUpdateStatus 测试更新状态
func (suite *NodeRepositoryTestSuite) TestUpdateStatus() {
	ctx := context.Background()

	// 创建测试数据
	instance := &node.NodeInstance{
		GroupID:     1,
		NodeID:      "test-node-status",
		ServiceName: "Test Node Status",
		Status:      "online",
	}
	suite.repo.Create(ctx, instance)

	// 更新状态
	err := suite.repo.UpdateStatus(ctx, "test-node-status", "offline")
	suite.NoError(err)

	// 验证
	found, _ := suite.repo.FindByNodeID(ctx, "test-node-status")
	suite.Equal("offline", found.Status)
}

// TestUpdateHeartbeat 测试更新心跳
func (suite *NodeRepositoryTestSuite) TestUpdateHeartbeat() {
	ctx := context.Background()

	// 创建测试数据
	instance := &node.NodeInstance{
		GroupID:     1,
		NodeID:      "test-node-heartbeat",
		ServiceName: "Test Node Heartbeat",
		Status:      "offline",
	}
	suite.repo.Create(ctx, instance)

	// 更新心跳
	now := time.Now()
	heartbeatData := &node.HeartbeatData{
		NodeID:        "test-node-heartbeat",
		CPUUsage:      45.5,
		MemoryUsage:   60.2,
		DiskUsage:     75.8,
		TrafficIn:     1000,
		TrafficOut:    2000,
		ActiveRules:   5,
		ConfigVersion: 1,
		ClientVersion: "1.0.0",
		Timestamp:     now,
	}

	err := suite.repo.UpdateHeartbeat(ctx, "test-node-heartbeat", heartbeatData)
	suite.NoError(err)

	// 验证 - 只验证基本字段
	found, _ := suite.repo.FindByNodeID(ctx, "test-node-heartbeat")
	suite.NotNil(found.LastHeartbeatAt)
	suite.Equal("online", found.Status)
}

// TestBatchUpdateHeartbeat 测试批量更新心跳
func (suite *NodeRepositoryTestSuite) TestBatchUpdateHeartbeat() {
	ctx := context.Background()

	// 创建测试数据
	for i := 1; i <= 3; i++ {
		instance := &node.NodeInstance{
			GroupID:     1,
			NodeID:      "batch-node-" + string(rune('0'+i)),
			ServiceName: "Batch Node " + string(rune('0'+i)),
			Status:      "offline",
		}
		suite.repo.Create(ctx, instance)
	}

	// 批量更新心跳
	now := time.Now()
	heartbeats := []*node.HeartbeatData{
		{
			NodeID:     "batch-node-1",
			CPUUsage:   40.0,
			Timestamp:  now,
		},
		{
			NodeID:     "batch-node-2",
			CPUUsage:   50.0,
			Timestamp:  now,
		},
		{
			NodeID:     "batch-node-3",
			CPUUsage:   60.0,
			Timestamp:  now,
		},
	}

	err := suite.repo.BatchUpdateHeartbeat(ctx, heartbeats)
	suite.NoError(err)

	// 验证 - 只验证状态更新
	node1, err := suite.repo.FindByNodeID(ctx, "batch-node-1")
	suite.NoError(err)
	suite.NotNil(node1)
	suite.Equal("online", node1.Status)

	node2, err := suite.repo.FindByNodeID(ctx, "batch-node-2")
	suite.NoError(err)
	suite.NotNil(node2)
	suite.Equal("online", node2.Status)
}

// TestFindOnlineNodes 测试查找在线节点
func (suite *NodeRepositoryTestSuite) TestFindOnlineNodes() {
	ctx := context.Background()

	// 创建在线节点
	now := time.Now()
	for i := 1; i <= 3; i++ {
		instance := &node.NodeInstance{
			GroupID:         1,
			NodeID:          "online-node-" + string(rune(i)),
			ServiceName:     "Online Node " + string(rune(i)),
			Status:          "online",
			LastHeartbeatAt: &now,
		}
		suite.repo.Create(ctx, instance)
	}

	// 创建离线节点
	oldTime := now.Add(-10 * time.Minute)
	instance := &node.NodeInstance{
		GroupID:         1,
		NodeID:          "offline-node-1",
		ServiceName:     "Offline Node",
		Status:          "offline",
		LastHeartbeatAt: &oldTime,
	}
	suite.repo.Create(ctx, instance)

	// 查找在线节点
	nodes, err := suite.repo.FindOnlineNodes(ctx)
	suite.NoError(err)
	suite.GreaterOrEqual(len(nodes), 3)

	// 验证都是在线状态
	for _, n := range nodes {
		suite.Equal("online", n.Status)
	}
}

// TestCountByStatus 测试按状态统计
func (suite *NodeRepositoryTestSuite) TestCountByStatus() {
	ctx := context.Background()

	// 创建测试数据
	for i := 1; i <= 3; i++ {
		instance := &node.NodeInstance{
			GroupID:     1,
			NodeID:      "count-online-" + string(rune(i)),
			ServiceName: "Count Online " + string(rune(i)),
			Status:      "online",
		}
		suite.repo.Create(ctx, instance)
	}

	for i := 1; i <= 2; i++ {
		instance := &node.NodeInstance{
			GroupID:     1,
			NodeID:      "count-offline-" + string(rune(i)),
			ServiceName: "Count Offline " + string(rune(i)),
			Status:      "offline",
		}
		suite.repo.Create(ctx, instance)
	}

	// 统计在线节点
	onlineCount, err := suite.repo.CountByStatus(ctx, "online")
	suite.NoError(err)
	suite.GreaterOrEqual(onlineCount, int64(3))

	// 统计离线节点
	offlineCount, err := suite.repo.CountByStatus(ctx, "offline")
	suite.NoError(err)
	suite.GreaterOrEqual(offlineCount, int64(2))
}

// TestMarkOfflineByTimeout 测试标记超时离线
func (suite *NodeRepositoryTestSuite) TestMarkOfflineByTimeout() {
	ctx := context.Background()

	// 创建超时节点
	oldTime := time.Now().Add(-5 * time.Minute)
	instance := &node.NodeInstance{
		GroupID:         1,
		NodeID:          "timeout-node",
		ServiceName:     "Timeout Node",
		Status:          "online",
		LastHeartbeatAt: &oldTime,
	}
	suite.repo.Create(ctx, instance)

	// 创建正常节点
	now := time.Now()
	instance2 := &node.NodeInstance{
		GroupID:         1,
		NodeID:          "normal-node",
		ServiceName:     "Normal Node",
		Status:          "online",
		LastHeartbeatAt: &now,
	}
	suite.repo.Create(ctx, instance2)

	// 标记超时节点为离线（3 分钟超时）
	count, err := suite.repo.MarkOfflineByTimeout(ctx, 3*time.Minute)
	suite.NoError(err)
	suite.GreaterOrEqual(count, int64(1))

	// 验证超时节点已离线
	found, _ := suite.repo.FindByNodeID(ctx, "timeout-node")
	suite.Equal("offline", found.Status)

	// 验证正常节点仍在线
	found2, _ := suite.repo.FindByNodeID(ctx, "normal-node")
	suite.Equal("online", found2.Status)
}

// TestNodeRepositoryTestSuite 运行测试套件
func TestNodeRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(NodeRepositoryTestSuite))
}
