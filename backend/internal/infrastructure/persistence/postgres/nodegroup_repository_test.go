package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"nodepass-pro/backend/internal/domain/nodegroup"
	"nodepass-pro/backend/internal/models"
)

// NodeGroupRepositoryTestSuite 测试套件
type NodeGroupRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo nodegroup.Repository
	ctx  context.Context
}

func (s *NodeGroupRepositoryTestSuite) SetupTest() {
	// 使用内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	s.Require().NoError(err)

	// 自动迁移
	err = db.AutoMigrate(&models.NodeGroup{}, &models.NodeGroupStats{})
	s.Require().NoError(err)

	s.db = db
	s.repo = NewNodeGroupRepository(db)
	s.ctx = context.Background()
}

func (s *NodeGroupRepositoryTestSuite) TearDownTest() {
	sqlDB, _ := s.db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}
}

func (s *NodeGroupRepositoryTestSuite) TestCreate() {
	group := &nodegroup.NodeGroup{
		UserID:      1,
		Name:        "Test Entry Group",
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

	err := s.repo.Create(s.ctx, group)
	s.NoError(err)
	s.NotZero(group.ID)
}

func (s *NodeGroupRepositoryTestSuite) TestFindByID() {
	// 创建测试数据
	group := &nodegroup.NodeGroup{
		UserID:      1,
		Name:        "Test Group",
		Type:        nodegroup.NodeGroupTypeEntry,
		Description: "Test",
		IsEnabled:   true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	s.repo.Create(s.ctx, group)

	// 查找
	found, err := s.repo.FindByID(s.ctx, group.ID)
	s.NoError(err)
	s.NotNil(found)
	s.Equal(group.ID, found.ID)
	s.Equal(group.Name, found.Name)
	s.Equal(group.Type, found.Type)
}

func (s *NodeGroupRepositoryTestSuite) TestFindByIDNotFound() {
	found, err := s.repo.FindByID(s.ctx, 999)
	s.Error(err)
	s.Equal(nodegroup.ErrNodeGroupNotFound, err)
	s.Nil(found)
}

func (s *NodeGroupRepositoryTestSuite) TestUpdate() {
	// 创建测试数据
	group := &nodegroup.NodeGroup{
		UserID:      1,
		Name:        "Original Name",
		Type:        nodegroup.NodeGroupTypeEntry,
		Description: "Original",
		IsEnabled:   true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	s.repo.Create(s.ctx, group)

	// 更新
	group.Name = "Updated Name"
	group.Description = "Updated"
	err := s.repo.Update(s.ctx, group)
	s.NoError(err)

	// 验证
	found, _ := s.repo.FindByID(s.ctx, group.ID)
	s.Equal("Updated Name", found.Name)
	s.Equal("Updated", found.Description)
}

func (s *NodeGroupRepositoryTestSuite) TestUpdateNotFound() {
	group := &nodegroup.NodeGroup{
		ID:   999,
		Name: "Not Found",
	}
	err := s.repo.Update(s.ctx, group)
	s.Error(err)
	s.Equal(nodegroup.ErrNodeGroupNotFound, err)
}

func (s *NodeGroupRepositoryTestSuite) TestDelete() {
	// 创建测试数据
	group := &nodegroup.NodeGroup{
		UserID:    1,
		Name:      "To Delete",
		Type:      nodegroup.NodeGroupTypeEntry,
		IsEnabled: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.repo.Create(s.ctx, group)

	// 删除
	err := s.repo.Delete(s.ctx, group.ID)
	s.NoError(err)

	// 验证已删除
	_, err = s.repo.FindByID(s.ctx, group.ID)
	s.Error(err)
	s.Equal(nodegroup.ErrNodeGroupNotFound, err)
}

func (s *NodeGroupRepositoryTestSuite) TestDeleteNotFound() {
	err := s.repo.Delete(s.ctx, 999)
	s.Error(err)
	s.Equal(nodegroup.ErrNodeGroupNotFound, err)
}

func (s *NodeGroupRepositoryTestSuite) TestFindByUserID() {
	// 创建测试数据
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 1, Name: "User1 Group1", Type: nodegroup.NodeGroupTypeEntry,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 1, Name: "User1 Group2", Type: nodegroup.NodeGroupTypeExit,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 2, Name: "User2 Group1", Type: nodegroup.NodeGroupTypeEntry,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	// 查找用户1的节点组
	groups, err := s.repo.FindByUserID(s.ctx, 1)
	s.NoError(err)
	s.Len(groups, 2)
}

func (s *NodeGroupRepositoryTestSuite) TestFindByType() {
	// 创建测试数据
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 1, Name: "Entry1", Type: nodegroup.NodeGroupTypeEntry,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 1, Name: "Entry2", Type: nodegroup.NodeGroupTypeEntry,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 1, Name: "Exit1", Type: nodegroup.NodeGroupTypeExit,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	// 查找入口组
	groups, err := s.repo.FindByType(s.ctx, nodegroup.NodeGroupTypeEntry)
	s.NoError(err)
	s.Len(groups, 2)
}

func (s *NodeGroupRepositoryTestSuite) TestFindByUserIDAndType() {
	// 创建测试数据
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 1, Name: "User1 Entry", Type: nodegroup.NodeGroupTypeEntry,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 1, Name: "User1 Exit", Type: nodegroup.NodeGroupTypeExit,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 2, Name: "User2 Entry", Type: nodegroup.NodeGroupTypeEntry,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	// 查找用户1的入口组
	groups, err := s.repo.FindByUserIDAndType(s.ctx, 1, nodegroup.NodeGroupTypeEntry)
	s.NoError(err)
	s.Len(groups, 1)
	s.Equal("User1 Entry", groups[0].Name)
}

func (s *NodeGroupRepositoryTestSuite) TestList() {
	// 创建测试数据
	group1 := &nodegroup.NodeGroup{
		UserID: 1, Name: "Group1", Type: nodegroup.NodeGroupTypeEntry,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	group2 := &nodegroup.NodeGroup{
		UserID: 1, Name: "Group2", Type: nodegroup.NodeGroupTypeExit,
		IsEnabled: false, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	group3 := &nodegroup.NodeGroup{
		UserID: 2, Name: "Group3", Type: nodegroup.NodeGroupTypeEntry,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	s.repo.Create(s.ctx, group1)
	s.repo.Create(s.ctx, group2)
	s.repo.Create(s.ctx, group3)

	// 测试无过滤
	groups, total, err := s.repo.List(s.ctx, nodegroup.ListFilter{
		Page:     1,
		PageSize: 10,
	})
	s.NoError(err)
	s.GreaterOrEqual(int(total), 3) // 至少有3条
	s.GreaterOrEqual(len(groups), 3)

	// 测试用户过滤
	groups, total, err = s.repo.List(s.ctx, nodegroup.ListFilter{
		UserID:   1,
		Page:     1,
		PageSize: 10,
	})
	s.NoError(err)
	s.GreaterOrEqual(int(total), 2) // 至少有2条
	s.GreaterOrEqual(len(groups), 2)

	// 测试类型过滤
	groups, total, err = s.repo.List(s.ctx, nodegroup.ListFilter{
		Type:     nodegroup.NodeGroupTypeEntry,
		Page:     1,
		PageSize: 10,
	})
	s.NoError(err)
	s.GreaterOrEqual(int(total), 2) // 至少有2条
	s.GreaterOrEqual(len(groups), 2)

	// 测试启用状态过滤
	groups, total, err = s.repo.List(s.ctx, nodegroup.ListFilter{
		EnabledOnly: true,
		Page:        1,
		PageSize:    10,
	})
	s.NoError(err)
	s.GreaterOrEqual(int(total), 2) // 至少有2条启用的
	s.GreaterOrEqual(len(groups), 2)
	// 验证所有返回的都是启用的
	for _, g := range groups {
		s.True(g.IsEnabled)
	}
}

func (s *NodeGroupRepositoryTestSuite) TestListWithKeyword() {
	// 创建测试数据
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 1, Name: "Production Entry", Type: nodegroup.NodeGroupTypeEntry,
		Description: "Prod environment", IsEnabled: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 1, Name: "Development Entry", Type: nodegroup.NodeGroupTypeEntry,
		Description: "Dev environment", IsEnabled: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	// 搜索关键词
	groups, total, err := s.repo.List(s.ctx, nodegroup.ListFilter{
		Keyword:  "Prod",
		Page:     1,
		PageSize: 10,
	})
	s.NoError(err)
	s.Equal(int64(1), total)
	s.Len(groups, 1)
	s.Equal("Production Entry", groups[0].Name)
}

func (s *NodeGroupRepositoryTestSuite) TestCountByUserID() {
	// 创建测试数据
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 1, Name: "Group1", Type: nodegroup.NodeGroupTypeEntry,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 1, Name: "Group2", Type: nodegroup.NodeGroupTypeExit,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	s.repo.Create(s.ctx, &nodegroup.NodeGroup{
		UserID: 2, Name: "Group3", Type: nodegroup.NodeGroupTypeEntry,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	count, err := s.repo.CountByUserID(s.ctx, 1)
	s.NoError(err)
	s.Equal(int64(2), count)
}

func (s *NodeGroupRepositoryTestSuite) TestUpdateStats() {
	// 创建节点组
	group := &nodegroup.NodeGroup{
		UserID: 1, Name: "Test Group", Type: nodegroup.NodeGroupTypeEntry,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	s.repo.Create(s.ctx, group)

	// 更新统计
	stats := &nodegroup.NodeGroupStats{
		NodeGroupID:      group.ID,
		TotalNodes:       10,
		OnlineNodes:      8,
		TotalTrafficIn:   1000000,
		TotalTrafficOut:  2000000,
		TotalConnections: 50,
		UpdatedAt:        time.Now(),
	}
	err := s.repo.UpdateStats(s.ctx, stats)
	s.NoError(err)

	// 验证
	found, err := s.repo.GetStats(s.ctx, group.ID)
	s.NoError(err)
	s.Equal(10, found.TotalNodes)
	s.Equal(8, found.OnlineNodes)
	s.Equal(int64(1000000), found.TotalTrafficIn)
}

func (s *NodeGroupRepositoryTestSuite) TestGetStats() {
	// 创建节点组
	group := &nodegroup.NodeGroup{
		UserID: 1, Name: "Test Group", Type: nodegroup.NodeGroupTypeEntry,
		IsEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	s.repo.Create(s.ctx, group)

	// 获取统计（应该返回空统计）
	stats, err := s.repo.GetStats(s.ctx, group.ID)
	s.NoError(err)
	s.NotNil(stats)
	s.Equal(group.ID, stats.NodeGroupID)
	s.Equal(0, stats.TotalNodes)
}

func TestNodeGroupRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(NodeGroupRepositoryTestSuite))
}
