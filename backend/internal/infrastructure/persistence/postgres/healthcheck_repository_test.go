package postgres

import (
	"context"
	"testing"
	"time"

	"nodepass-pro/backend/internal/domain/healthcheck"
	"nodepass-pro/backend/internal/models"

	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type HealthCheckRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo *HealthCheckRepository
}

func (s *HealthCheckRepositoryTestSuite) SetupSuite() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	s.Require().NoError(err)

	err = db.AutoMigrate(
		&models.User{},
		&models.NodeGroup{},
		&models.NodeInstance{},
		&models.NodeHealthCheck{},
		&models.NodeHealthRecord{},
		&models.NodeQualityScore{},
	)
	s.Require().NoError(err)

	s.db = db
	s.repo = NewHealthCheckRepository(db)
}

func (s *HealthCheckRepositoryTestSuite) SetupTest() {
	s.db.Exec("DELETE FROM node_quality_scores")
	s.db.Exec("DELETE FROM node_health_records")
	s.db.Exec("DELETE FROM node_health_checks")
	s.db.Exec("DELETE FROM node_instances")
	s.db.Exec("DELETE FROM node_groups")
	s.db.Exec("DELETE FROM users")

	// 创建测试数据
	user := &models.User{Username: "test", Email: "test@example.com"}
	s.Require().NoError(s.db.Create(user).Error)

	group := &models.NodeGroup{UserID: user.ID, Name: "test-group"}
	s.Require().NoError(s.db.Create(group).Error)

	host := "127.0.0.1"
	port := 8080
	instance := &models.NodeInstance{
		NodeGroupID: group.ID,
		Name:        "test-instance",
		Host:        &host,
		Port:        &port,
		Status:      models.NodeInstanceStatusOnline,
	}
	s.Require().NoError(s.db.Create(instance).Error)
}

func (s *HealthCheckRepositoryTestSuite) TestCreateHealthCheck() {
	ctx := context.Background()
	check := healthcheck.NewHealthCheck(1, healthcheck.CheckTypeTCP)

	err := s.repo.CreateHealthCheck(ctx, check)
	s.NoError(err)
	s.NotZero(check.ID)
}

func (s *HealthCheckRepositoryTestSuite) TestFindHealthCheckByID() {
	ctx := context.Background()
	check := healthcheck.NewHealthCheck(1, healthcheck.CheckTypeTCP)
	s.Require().NoError(s.repo.CreateHealthCheck(ctx, check))

	found, err := s.repo.FindHealthCheckByID(ctx, check.ID)
	s.NoError(err)
	s.Equal(check.ID, found.ID)
	s.Equal(check.NodeInstanceID, found.NodeInstanceID)
	s.Equal(check.Type, found.Type)
}

func (s *HealthCheckRepositoryTestSuite) TestFindHealthCheckByID_NotFound() {
	ctx := context.Background()

	_, err := s.repo.FindHealthCheckByID(ctx, 999)
	s.ErrorIs(err, healthcheck.ErrHealthCheckNotFound)
}

func (s *HealthCheckRepositoryTestSuite) TestFindHealthCheckByNodeInstance() {
	ctx := context.Background()
	check := healthcheck.NewHealthCheck(1, healthcheck.CheckTypeTCP)
	s.Require().NoError(s.repo.CreateHealthCheck(ctx, check))

	found, err := s.repo.FindHealthCheckByNodeInstance(ctx, 1)
	s.NoError(err)
	s.Equal(check.ID, found.ID)
	s.Equal(uint(1), found.NodeInstanceID)
}

func (s *HealthCheckRepositoryTestSuite) TestUpdateHealthCheck() {
	ctx := context.Background()
	check := healthcheck.NewHealthCheck(1, healthcheck.CheckTypeTCP)
	s.Require().NoError(s.repo.CreateHealthCheck(ctx, check))

	check.UpdateConfig(60, 10, 5, 3, 5)
	check.Disable()

	err := s.repo.UpdateHealthCheck(ctx, check)
	s.NoError(err)

	found, err := s.repo.FindHealthCheckByID(ctx, check.ID)
	s.NoError(err)
	s.Equal(60, found.Interval)
	s.Equal(10, found.Timeout)
	s.False(found.Enabled)
}

func (s *HealthCheckRepositoryTestSuite) TestDeleteHealthCheck() {
	ctx := context.Background()
	check := healthcheck.NewHealthCheck(1, healthcheck.CheckTypeTCP)
	s.Require().NoError(s.repo.CreateHealthCheck(ctx, check))

	err := s.repo.DeleteHealthCheck(ctx, check.ID)
	s.NoError(err)

	_, err = s.repo.FindHealthCheckByID(ctx, check.ID)
	s.ErrorIs(err, healthcheck.ErrHealthCheckNotFound)
}

func (s *HealthCheckRepositoryTestSuite) TestListEnabledHealthChecks() {
	ctx := context.Background()

	check1 := healthcheck.NewHealthCheck(1, healthcheck.CheckTypeTCP)
	s.Require().NoError(s.repo.CreateHealthCheck(ctx, check1))

	check2 := healthcheck.NewHealthCheck(2, healthcheck.CheckTypeHTTP)
	check2.Disable()
	s.Require().NoError(s.repo.CreateHealthCheck(ctx, check2))

	checks, err := s.repo.ListEnabledHealthChecks(ctx)
	s.NoError(err)
	s.GreaterOrEqual(len(checks), 1)

	// 验证至少有一个是启用的
	hasEnabled := false
	for _, c := range checks {
		if c.ID == check1.ID && c.Enabled {
			hasEnabled = true
			break
		}
	}
	s.True(hasEnabled)
}

func (s *HealthCheckRepositoryTestSuite) TestCreateHealthRecord() {
	ctx := context.Background()
	record := healthcheck.NewHealthRecord(1, healthcheck.CheckTypeTCP, healthcheck.CheckStatusHealthy)
	record.SetLatency(50)

	err := s.repo.CreateHealthRecord(ctx, record)
	s.NoError(err)
	s.NotZero(record.ID)
}

func (s *HealthCheckRepositoryTestSuite) TestFindHealthRecordsByNodeInstance() {
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		record := healthcheck.NewHealthRecord(1, healthcheck.CheckTypeTCP, healthcheck.CheckStatusHealthy)
		s.Require().NoError(s.repo.CreateHealthRecord(ctx, record))
		time.Sleep(time.Millisecond)
	}

	records, err := s.repo.FindHealthRecordsByNodeInstance(ctx, 1, 3)
	s.NoError(err)
	s.Len(records, 3)
}

func (s *HealthCheckRepositoryTestSuite) TestFindHealthRecordsByTimeRange() {
	ctx := context.Background()

	// 创建旧记录
	oldRecord := healthcheck.NewHealthRecord(1, healthcheck.CheckTypeTCP, healthcheck.CheckStatusHealthy)
	s.Require().NoError(s.repo.CreateHealthRecord(ctx, oldRecord))

	// 等待一段时间
	time.Sleep(10 * time.Millisecond)
	startTime := time.Now()

	// 创建新记录
	newRecord := healthcheck.NewHealthRecord(1, healthcheck.CheckTypeTCP, healthcheck.CheckStatusHealthy)
	s.Require().NoError(s.repo.CreateHealthRecord(ctx, newRecord))

	records, err := s.repo.FindHealthRecordsByTimeRange(ctx, 1, startTime)
	s.NoError(err)
	s.Len(records, 1)
	s.Equal(newRecord.ID, records[0].ID)
}

func (s *HealthCheckRepositoryTestSuite) TestDeleteOldHealthRecords() {
	ctx := context.Background()

	// 创建记录
	record := healthcheck.NewHealthRecord(1, healthcheck.CheckTypeTCP, healthcheck.CheckStatusHealthy)
	s.Require().NoError(s.repo.CreateHealthRecord(ctx, record))

	// 删除未来时间之前的记录（应该删除所有）
	cutoffTime := time.Now().Add(1 * time.Hour)
	count, err := s.repo.DeleteOldHealthRecords(ctx, cutoffTime)
	s.NoError(err)
	s.Equal(int64(1), count)

	records, err := s.repo.FindHealthRecordsByNodeInstance(ctx, 1, 10)
	s.NoError(err)
	s.Len(records, 0)
}

func (s *HealthCheckRepositoryTestSuite) TestCreateOrUpdateQualityScore() {
	ctx := context.Background()
	score := healthcheck.NewQualityScore(1)
	score.LatencyScore = 90
	score.StabilityScore = 85
	score.CalculateOverallScore()

	// 创建
	err := s.repo.CreateOrUpdateQualityScore(ctx, score)
	s.NoError(err)
	s.NotZero(score.ID)

	// 更新
	score.LatencyScore = 95
	score.CalculateOverallScore()
	err = s.repo.CreateOrUpdateQualityScore(ctx, score)
	s.NoError(err)

	found, err := s.repo.FindQualityScoreByNodeInstance(ctx, 1)
	s.NoError(err)
	s.Equal(95.0, found.LatencyScore)
}

func (s *HealthCheckRepositoryTestSuite) TestFindQualityScoreByNodeInstance() {
	ctx := context.Background()
	score := healthcheck.NewQualityScore(1)
	s.Require().NoError(s.repo.CreateOrUpdateQualityScore(ctx, score))

	found, err := s.repo.FindQualityScoreByNodeInstance(ctx, 1)
	s.NoError(err)
	s.Equal(score.NodeInstanceID, found.NodeInstanceID)
}

func (s *HealthCheckRepositoryTestSuite) TestListQualityScoresByUser() {
	ctx := context.Background()

	// 获取测试用户和节点实例
	var user models.User
	s.Require().NoError(s.db.First(&user).Error)

	var instance models.NodeInstance
	s.Require().NoError(s.db.First(&instance).Error)

	// 创建质量评分
	score := healthcheck.NewQualityScore(instance.ID)
	score.OverallScore = 90
	s.Require().NoError(s.repo.CreateOrUpdateQualityScore(ctx, score))

	scores, err := s.repo.ListQualityScoresByUser(ctx, user.ID)
	s.NoError(err)
	s.GreaterOrEqual(len(scores), 1)
}

func TestHealthCheckRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(HealthCheckRepositoryTestSuite))
}
