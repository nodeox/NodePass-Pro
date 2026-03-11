package postgres_test

import (
	"context"
	"testing"

	"nodepass-pro/backend/internal/domain/user"
	"nodepass-pro/backend/internal/infrastructure/persistence/postgres"
	"nodepass-pro/backend/internal/models"

	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// UserRepositoryTestSuite 用户仓储测试套件
type UserRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo user.Repository
}

// SetupTest 每个测试前执行
func (suite *UserRepositoryTestSuite) SetupTest() {
	// 使用 SQLite 内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	// 自动迁移
	err = db.AutoMigrate(&models.User{})
	suite.Require().NoError(err)

	suite.db = db
	suite.repo = postgres.NewUserRepository(db)
}

// TearDownTest 每个测试后执行
func (suite *UserRepositoryTestSuite) TearDownTest() {
	sqlDB, err := suite.db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

// TestCreate 测试创建用户
func (suite *UserRepositoryTestSuite) TestCreate() {
	ctx := context.Background()

	u := &user.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		Role:         "user",
		Status:       "active",
		VipLevel:     0,
		TrafficQuota: 10737418240, // 10GB
		TrafficUsed:  0,
	}

	err := suite.repo.Create(ctx, u)
	suite.NoError(err)
	suite.NotZero(u.ID)
	suite.NotZero(u.CreatedAt)
}

// TestFindByID 测试根据 ID 查找
func (suite *UserRepositoryTestSuite) TestFindByID() {
	ctx := context.Background()

	// 创建测试数据
	u := &user.User{
		Username:     "findbyid",
		Email:        "findbyid@example.com",
		PasswordHash: "hashed",
		Role:         "user",
		Status:       "active",
	}
	suite.repo.Create(ctx, u)

	// 查找
	found, err := suite.repo.FindByID(ctx, u.ID)
	suite.NoError(err)
	suite.NotNil(found)
	suite.Equal(u.Username, found.Username)
	suite.Equal(u.Email, found.Email)
}

// TestFindByID_NotFound 测试查找不存在的用户
func (suite *UserRepositoryTestSuite) TestFindByID_NotFound() {
	ctx := context.Background()

	found, err := suite.repo.FindByID(ctx, 99999)
	suite.Error(err)
	suite.Nil(found)
	suite.Equal(user.ErrUserNotFound, err)
}

// TestFindByEmail 测试根据邮箱查找
func (suite *UserRepositoryTestSuite) TestFindByEmail() {
	ctx := context.Background()

	// 创建测试数据
	u := &user.User{
		Username:     "emailuser",
		Email:        "email@example.com",
		PasswordHash: "hashed",
		Role:         "user",
		Status:       "active",
	}
	suite.repo.Create(ctx, u)

	// 查找
	found, err := suite.repo.FindByEmail(ctx, "email@example.com")
	suite.NoError(err)
	suite.NotNil(found)
	suite.Equal(u.ID, found.ID)
	suite.Equal("emailuser", found.Username)
}

// TestFindByEmail_NotFound 测试邮箱不存在
func (suite *UserRepositoryTestSuite) TestFindByEmail_NotFound() {
	ctx := context.Background()

	found, err := suite.repo.FindByEmail(ctx, "nonexistent@example.com")
	suite.Error(err)
	suite.Nil(found)
	suite.Equal(user.ErrUserNotFound, err)
}

// TestFindByUsername 测试根据用户名查找
func (suite *UserRepositoryTestSuite) TestFindByUsername() {
	ctx := context.Background()

	// 创建测试数据
	u := &user.User{
		Username:     "uniqueuser",
		Email:        "unique@example.com",
		PasswordHash: "hashed",
		Role:         "user",
		Status:       "active",
	}
	suite.repo.Create(ctx, u)

	// 查找
	found, err := suite.repo.FindByUsername(ctx, "uniqueuser")
	suite.NoError(err)
	suite.NotNil(found)
	suite.Equal(u.ID, found.ID)
	suite.Equal("unique@example.com", found.Email)
}

// TestUpdate 测试更新用户
func (suite *UserRepositoryTestSuite) TestUpdate() {
	ctx := context.Background()

	// 创建测试数据
	u := &user.User{
		Username:     "updateuser",
		Email:        "update@example.com",
		PasswordHash: "hashed",
		Role:         "user",
		Status:       "active",
		VipLevel:     0,
	}
	suite.repo.Create(ctx, u)

	// 更新
	u.VipLevel = 1
	u.Status = "suspended"
	err := suite.repo.Update(ctx, u)
	suite.NoError(err)

	// 验证
	found, _ := suite.repo.FindByID(ctx, u.ID)
	suite.Equal(1, found.VipLevel)
	suite.Equal("suspended", found.Status)
}

// TestDelete 测试删除用户
func (suite *UserRepositoryTestSuite) TestDelete() {
	ctx := context.Background()

	// 创建测试数据
	u := &user.User{
		Username:     "deleteuser",
		Email:        "delete@example.com",
		PasswordHash: "hashed",
		Role:         "user",
		Status:       "active",
	}
	suite.repo.Create(ctx, u)

	// 删除
	err := suite.repo.Delete(ctx, u.ID)
	suite.NoError(err)

	// 验证已删除
	found, err := suite.repo.FindByID(ctx, u.ID)
	suite.Error(err)
	suite.Nil(found)
}

// TestList 测试列表查询
func (suite *UserRepositoryTestSuite) TestList() {
	ctx := context.Background()

	// 创建测试数据
	for i := 1; i <= 5; i++ {
		u := &user.User{
			Username:     "listuser" + string(rune(i)),
			Email:        "list" + string(rune(i)) + "@example.com",
			PasswordHash: "hashed",
			Role:         "user",
			Status:       "active",
		}
		suite.repo.Create(ctx, u)
	}

	// 查询第一页
	filter := user.ListFilter{
		Page:     1,
		PageSize: 3,
	}
	users, total, err := suite.repo.List(ctx, filter)
	suite.NoError(err)
	suite.GreaterOrEqual(len(users), 3)
	suite.GreaterOrEqual(total, int64(5))
}

// TestList_WithFilters 测试带过滤条件的列表查询
func (suite *UserRepositoryTestSuite) TestList_WithFilters() {
	ctx := context.Background()

	// 创建测试数据
	// VIP 用户
	for i := 1; i <= 2; i++ {
		u := &user.User{
			Username:     "vipuser" + string(rune(i)),
			Email:        "vip" + string(rune(i)) + "@example.com",
			PasswordHash: "hashed",
			Role:         "vip",
			Status:       "active",
			VipLevel:     1,
		}
		suite.repo.Create(ctx, u)
	}

	// 普通用户
	for i := 1; i <= 3; i++ {
		u := &user.User{
			Username:     "normaluser" + string(rune(i)),
			Email:        "normal" + string(rune(i)) + "@example.com",
			PasswordHash: "hashed",
			Role:         "user",
			Status:       "active",
			VipLevel:     0,
		}
		suite.repo.Create(ctx, u)
	}

	// 查询 VIP 角色用户
	filter := user.ListFilter{
		Page:     1,
		PageSize: 10,
		Role:     "vip",
	}
	users, total, err := suite.repo.List(ctx, filter)
	suite.NoError(err)
	suite.GreaterOrEqual(len(users), 2)
	suite.GreaterOrEqual(total, int64(2))

	// 验证都是 VIP 角色
	for _, u := range users {
		suite.Equal("vip", u.Role)
	}
}

// TestUserRepositoryTestSuite 运行测试套件
func TestUserRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(UserRepositoryTestSuite))
}
