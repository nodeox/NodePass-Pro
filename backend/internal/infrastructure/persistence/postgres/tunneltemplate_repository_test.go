package postgres

import (
	"context"
	"testing"

	"nodepass-pro/backend/internal/domain/tunneltemplate"
	"nodepass-pro/backend/internal/models"

	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TunnelTemplateRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo *TunnelTemplateRepository
}

func (s *TunnelTemplateRepositoryTestSuite) SetupSuite() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	s.Require().NoError(err)

	err = db.AutoMigrate(
		&models.User{},
		&models.TunnelTemplate{},
	)
	s.Require().NoError(err)

	s.db = db
	s.repo = NewTunnelTemplateRepository(db)
}

func (s *TunnelTemplateRepositoryTestSuite) SetupTest() {
	s.db.Exec("DELETE FROM tunnel_templates")
	s.db.Exec("DELETE FROM users")

	// 创建测试用户
	user := &models.User{Username: "test", Email: "test@example.com"}
	s.Require().NoError(s.db.Create(user).Error)
}

func (s *TunnelTemplateRepositoryTestSuite) TestCreate() {
	ctx := context.Background()
	template := tunneltemplate.NewTunnelTemplate(1, "test-template", "tcp", &tunneltemplate.TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8080,
	})

	err := s.repo.Create(ctx, template)
	s.NoError(err)
	s.NotZero(template.ID)
}

func (s *TunnelTemplateRepositoryTestSuite) TestFindByID() {
	ctx := context.Background()
	template := tunneltemplate.NewTunnelTemplate(1, "test-template", "tcp", &tunneltemplate.TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8080,
	})
	s.Require().NoError(s.repo.Create(ctx, template))

	found, err := s.repo.FindByID(ctx, template.ID)
	s.NoError(err)
	s.Equal(template.ID, found.ID)
	s.Equal(template.Name, found.Name)
	s.Equal(template.Protocol, found.Protocol)
}

func (s *TunnelTemplateRepositoryTestSuite) TestFindByID_NotFound() {
	ctx := context.Background()

	_, err := s.repo.FindByID(ctx, 999)
	s.ErrorIs(err, tunneltemplate.ErrTemplateNotFound)
}

func (s *TunnelTemplateRepositoryTestSuite) TestUpdate() {
	ctx := context.Background()
	template := tunneltemplate.NewTunnelTemplate(1, "test-template", "tcp", &tunneltemplate.TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8080,
	})
	s.Require().NoError(s.repo.Create(ctx, template))

	desc := "updated description"
	template.UpdateInfo("updated-name", &desc)
	template.MakePublic()

	err := s.repo.Update(ctx, template)
	s.NoError(err)

	found, err := s.repo.FindByID(ctx, template.ID)
	s.NoError(err)
	s.Equal("updated-name", found.Name)
	s.Equal("updated description", *found.Description)
	s.True(found.IsPublic)
}

func (s *TunnelTemplateRepositoryTestSuite) TestDelete() {
	ctx := context.Background()
	template := tunneltemplate.NewTunnelTemplate(1, "test-template", "tcp", &tunneltemplate.TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8080,
	})
	s.Require().NoError(s.repo.Create(ctx, template))

	err := s.repo.Delete(ctx, template.ID)
	s.NoError(err)

	_, err = s.repo.FindByID(ctx, template.ID)
	s.ErrorIs(err, tunneltemplate.ErrTemplateNotFound)
}

func (s *TunnelTemplateRepositoryTestSuite) TestList() {
	ctx := context.Background()

	// 创建多个模板
	template1 := tunneltemplate.NewTunnelTemplate(1, "template1", "tcp", &tunneltemplate.TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8080,
	})
	s.Require().NoError(s.repo.Create(ctx, template1))

	template2 := tunneltemplate.NewTunnelTemplate(1, "template2", "udp", &tunneltemplate.TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8081,
	})
	template2.MakePublic()
	s.Require().NoError(s.repo.Create(ctx, template2))

	// 查询所有
	filter := tunneltemplate.ListFilter{
		UserID:   1,
		Page:     1,
		PageSize: 10,
	}
	templates, total, err := s.repo.List(ctx, filter)
	s.NoError(err)
	s.Equal(int64(2), total)
	s.Len(templates, 2)
}

func (s *TunnelTemplateRepositoryTestSuite) TestList_FilterByProtocol() {
	ctx := context.Background()

	template1 := tunneltemplate.NewTunnelTemplate(1, "template1", "tcp", &tunneltemplate.TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8080,
	})
	s.Require().NoError(s.repo.Create(ctx, template1))

	template2 := tunneltemplate.NewTunnelTemplate(1, "template2", "udp", &tunneltemplate.TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8081,
	})
	s.Require().NoError(s.repo.Create(ctx, template2))

	protocol := "tcp"
	filter := tunneltemplate.ListFilter{
		UserID:   1,
		Protocol: &protocol,
		Page:     1,
		PageSize: 10,
	}
	templates, total, err := s.repo.List(ctx, filter)
	s.NoError(err)
	s.Equal(int64(1), total)
	s.Len(templates, 1)
	s.Equal("tcp", templates[0].Protocol)
}

func (s *TunnelTemplateRepositoryTestSuite) TestList_FilterByIsPublic() {
	ctx := context.Background()

	template1 := tunneltemplate.NewTunnelTemplate(1, "template1", "tcp", &tunneltemplate.TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8080,
	})
	s.Require().NoError(s.repo.Create(ctx, template1))

	template2 := tunneltemplate.NewTunnelTemplate(1, "template2", "tcp", &tunneltemplate.TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8081,
	})
	template2.MakePublic()
	s.Require().NoError(s.repo.Create(ctx, template2))

	isPublic := true
	filter := tunneltemplate.ListFilter{
		UserID:   1,
		IsPublic: &isPublic,
		Page:     1,
		PageSize: 10,
	}
	templates, total, err := s.repo.List(ctx, filter)
	s.NoError(err)
	s.Equal(int64(1), total)
	s.Len(templates, 1)
	s.True(templates[0].IsPublic)
}

func (s *TunnelTemplateRepositoryTestSuite) TestFindByUserAndName() {
	ctx := context.Background()
	template := tunneltemplate.NewTunnelTemplate(1, "test-template", "tcp", &tunneltemplate.TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8080,
	})
	s.Require().NoError(s.repo.Create(ctx, template))

	found, err := s.repo.FindByUserAndName(ctx, 1, "test-template")
	s.NoError(err)
	s.Equal(template.ID, found.ID)
	s.Equal(template.Name, found.Name)
}

func (s *TunnelTemplateRepositoryTestSuite) TestFindByUserAndName_NotFound() {
	ctx := context.Background()

	_, err := s.repo.FindByUserAndName(ctx, 1, "non-existent")
	s.ErrorIs(err, tunneltemplate.ErrTemplateNotFound)
}

func (s *TunnelTemplateRepositoryTestSuite) TestIncrementUsageCount() {
	ctx := context.Background()
	template := tunneltemplate.NewTunnelTemplate(1, "test-template", "tcp", &tunneltemplate.TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8080,
	})
	s.Require().NoError(s.repo.Create(ctx, template))

	err := s.repo.IncrementUsageCount(ctx, template.ID)
	s.NoError(err)

	found, err := s.repo.FindByID(ctx, template.ID)
	s.NoError(err)
	s.Equal(1, found.UsageCount)

	err = s.repo.IncrementUsageCount(ctx, template.ID)
	s.NoError(err)

	found, err = s.repo.FindByID(ctx, template.ID)
	s.NoError(err)
	s.Equal(2, found.UsageCount)
}

func TestTunnelTemplateRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(TunnelTemplateRepositoryTestSuite))
}
