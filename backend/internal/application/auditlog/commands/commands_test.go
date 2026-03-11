package commands

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"nodepass-pro/backend/internal/domain/auditlog"
)

// MockAuditLogRepository Mock 仓储
type MockAuditLogRepository struct {
	logs   map[uint]*auditlog.AuditLog
	nextID uint
}

func NewMockAuditLogRepository() *MockAuditLogRepository {
	return &MockAuditLogRepository{
		logs:   make(map[uint]*auditlog.AuditLog),
		nextID: 1,
	}
}

func (m *MockAuditLogRepository) Create(ctx context.Context, log *auditlog.AuditLog) error {
	log.ID = m.nextID
	m.nextID++
	m.logs[log.ID] = log
	return nil
}

func (m *MockAuditLogRepository) BatchCreate(ctx context.Context, logs []*auditlog.AuditLog) error {
	for _, log := range logs {
		m.Create(ctx, log)
	}
	return nil
}

func (m *MockAuditLogRepository) FindByID(ctx context.Context, id uint) (*auditlog.AuditLog, error) {
	log, ok := m.logs[id]
	if !ok {
		return nil, auditlog.ErrAuditLogNotFound
	}
	return log, nil
}

func (m *MockAuditLogRepository) List(ctx context.Context, filter auditlog.ListFilter) ([]*auditlog.AuditLog, int64, error) {
	var result []*auditlog.AuditLog
	for _, log := range m.logs {
		if filter.UserID != nil && (log.UserID == nil || *log.UserID != *filter.UserID) {
			continue
		}
		if filter.Action != "" && log.Action != filter.Action {
			continue
		}
		if filter.ResourceType != "" && log.ResourceType != filter.ResourceType {
			continue
		}
		result = append(result, log)
	}
	return result, int64(len(result)), nil
}

func (m *MockAuditLogRepository) CountByAction(ctx context.Context, action string, startTime, endTime time.Time) (int64, error) {
	return 0, nil
}

func (m *MockAuditLogRepository) CountByUser(ctx context.Context, userID uint, startTime, endTime time.Time) (int64, error) {
	return 0, nil
}

func (m *MockAuditLogRepository) DeleteOldLogs(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

// CreateAuditLogHandlerTestSuite 测试套件
type CreateAuditLogHandlerTestSuite struct {
	suite.Suite
	repo    *MockAuditLogRepository
	handler *CreateAuditLogHandler
	ctx     context.Context
}

func (s *CreateAuditLogHandlerTestSuite) SetupTest() {
	s.repo = NewMockAuditLogRepository()
	s.handler = NewCreateAuditLogHandler(s.repo)
	s.ctx = context.Background()
}

func (s *CreateAuditLogHandlerTestSuite) TestCreateAuditLogSuccess() {
	userID := uint(100)
	cmd := CreateAuditLogCommand{
		UserID:       &userID,
		Action:       "user.login",
		ResourceType: "user",
		Details:      "用户登录",
		IPAddress:    "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
	}

	err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)

	// 验证日志已创建
	s.Equal(1, len(s.repo.logs))
}

func (s *CreateAuditLogHandlerTestSuite) TestCreateAuditLogSystemAction() {
	cmd := CreateAuditLogCommand{
		UserID:       nil, // 系统操作
		Action:       "system.cleanup",
		ResourceType: "system",
		Details:      "系统清理",
		IPAddress:    "127.0.0.1",
		UserAgent:    "System",
	}

	err := s.handler.Handle(s.ctx, cmd)
	s.NoError(err)

	// 验证日志已创建
	s.Equal(1, len(s.repo.logs))
	log := s.repo.logs[1]
	s.Nil(log.UserID)
	s.True(log.IsSystemAction())
}

func TestCreateAuditLogHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(CreateAuditLogHandlerTestSuite))
}
