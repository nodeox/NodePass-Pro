package queries

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

// ListAuditLogsHandlerTestSuite 测试套件
type ListAuditLogsHandlerTestSuite struct {
	suite.Suite
	repo    *MockAuditLogRepository
	handler *ListAuditLogsHandler
	ctx     context.Context
}

func (s *ListAuditLogsHandlerTestSuite) SetupTest() {
	s.repo = NewMockAuditLogRepository()
	s.handler = NewListAuditLogsHandler(s.repo)
	s.ctx = context.Background()

	// 预创建日志
	userID := uint(100)
	for i := 1; i <= 5; i++ {
		log := auditlog.NewAuditLog(&userID, "user.login", "user", nil, "测试", "192.168.1.1", "Mozilla")
		s.repo.Create(s.ctx, log)
	}
}

func (s *ListAuditLogsHandlerTestSuite) TestListAuditLogsSuccess() {
	query := ListAuditLogsQuery{
		Page:     1,
		PageSize: 20,
	}

	result, total, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.GreaterOrEqual(len(result), 5)
	s.GreaterOrEqual(int(total), 5)
}

func (s *ListAuditLogsHandlerTestSuite) TestListAuditLogsByUser() {
	userID := uint(100)
	query := ListAuditLogsQuery{
		UserID:   &userID,
		Page:     1,
		PageSize: 20,
	}

	result, total, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.GreaterOrEqual(len(result), 5)
	s.GreaterOrEqual(int(total), 5)
}

func TestListAuditLogsHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ListAuditLogsHandlerTestSuite))
}

// GetAuditLogHandlerTestSuite 测试套件
type GetAuditLogHandlerTestSuite struct {
	suite.Suite
	repo    *MockAuditLogRepository
	handler *GetAuditLogHandler
	ctx     context.Context
}

func (s *GetAuditLogHandlerTestSuite) SetupTest() {
	s.repo = NewMockAuditLogRepository()
	s.handler = NewGetAuditLogHandler(s.repo)
	s.ctx = context.Background()

	// 预创建日志
	userID := uint(100)
	log := auditlog.NewAuditLog(&userID, "user.login", "user", nil, "测试", "192.168.1.1", "Mozilla")
	s.repo.Create(s.ctx, log)
}

func (s *GetAuditLogHandlerTestSuite) TestGetAuditLogSuccess() {
	query := GetAuditLogQuery{ID: 1}
	result, err := s.handler.Handle(s.ctx, query)
	s.NoError(err)
	s.NotNil(result)
	s.Equal("user.login", result.Action)
}

func (s *GetAuditLogHandlerTestSuite) TestGetAuditLogNotFound() {
	query := GetAuditLogQuery{ID: 999}
	result, err := s.handler.Handle(s.ctx, query)
	s.Error(err)
	s.Nil(result)
	s.Equal(auditlog.ErrAuditLogNotFound, err)
}

func TestGetAuditLogHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GetAuditLogHandlerTestSuite))
}
