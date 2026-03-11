package commands

import (
	"context"
	"testing"

	"nodepass-pro/backend/internal/domain/tunneltemplate"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateTemplateHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewCreateTemplateHandler(repo)

	cmd := CreateTemplateCommand{
		UserID:   1,
		Name:     "test-template",
		Protocol: "tcp",
		Config: &tunneltemplate.TemplateConfig{
			RemoteHost: "example.com",
			RemotePort: 8080,
		},
	}

	repo.On("FindByUserAndName", ctx, uint(1), "test-template").Return(nil, tunneltemplate.ErrTemplateNotFound)
	repo.On("Create", ctx, mock.AnythingOfType("*tunneltemplate.TunnelTemplate")).Return(nil)

	template, err := handler.Handle(ctx, cmd)
	assert.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, "test-template", template.Name)
	assert.Equal(t, "tcp", template.Protocol)

	repo.AssertExpectations(t)
}

func TestCreateTemplateHandler_Handle_EmptyName(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewCreateTemplateHandler(repo)

	cmd := CreateTemplateCommand{
		UserID:   1,
		Name:     "",
		Protocol: "tcp",
		Config:   &tunneltemplate.TemplateConfig{},
	}

	_, err := handler.Handle(ctx, cmd)
	assert.ErrorIs(t, err, tunneltemplate.ErrInvalidTemplateName)
}

func TestCreateTemplateHandler_Handle_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewCreateTemplateHandler(repo)

	cmd := CreateTemplateCommand{
		UserID:   1,
		Name:     "test-template",
		Protocol: "tcp",
		Config:   &tunneltemplate.TemplateConfig{},
	}

	existing := tunneltemplate.NewTunnelTemplate(1, "test-template", "tcp", &tunneltemplate.TemplateConfig{})
	repo.On("FindByUserAndName", ctx, uint(1), "test-template").Return(existing, nil)

	_, err := handler.Handle(ctx, cmd)
	assert.ErrorIs(t, err, tunneltemplate.ErrTemplateAlreadyExists)

	repo.AssertExpectations(t)
}

func TestUpdateTemplateHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewUpdateTemplateHandler(repo)

	existing := tunneltemplate.NewTunnelTemplate(1, "test-template", "tcp", &tunneltemplate.TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8080,
	})
	existing.ID = 1

	newName := "updated-template"
	cmd := UpdateTemplateCommand{
		TemplateID: 1,
		UserID:     1,
		Name:       &newName,
	}

	repo.On("FindByID", ctx, uint(1)).Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*tunneltemplate.TunnelTemplate")).Return(nil)

	template, err := handler.Handle(ctx, cmd)
	assert.NoError(t, err)
	assert.Equal(t, "updated-template", template.Name)

	repo.AssertExpectations(t)
}

func TestUpdateTemplateHandler_Handle_Unauthorized(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewUpdateTemplateHandler(repo)

	existing := tunneltemplate.NewTunnelTemplate(1, "test-template", "tcp", &tunneltemplate.TemplateConfig{})
	existing.ID = 1

	newName := "updated-template"
	cmd := UpdateTemplateCommand{
		TemplateID: 1,
		UserID:     2, // 不同的用户
		Name:       &newName,
	}

	repo.On("FindByID", ctx, uint(1)).Return(existing, nil)

	_, err := handler.Handle(ctx, cmd)
	assert.ErrorIs(t, err, tunneltemplate.ErrUnauthorized)

	repo.AssertExpectations(t)
}

func TestDeleteTemplateHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewDeleteTemplateHandler(repo)

	existing := tunneltemplate.NewTunnelTemplate(1, "test-template", "tcp", &tunneltemplate.TemplateConfig{})
	existing.ID = 1

	cmd := DeleteTemplateCommand{
		TemplateID: 1,
		UserID:     1,
	}

	repo.On("FindByID", ctx, uint(1)).Return(existing, nil)
	repo.On("Delete", ctx, uint(1)).Return(nil)

	err := handler.Handle(ctx, cmd)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestDeleteTemplateHandler_Handle_Unauthorized(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewDeleteTemplateHandler(repo)

	existing := tunneltemplate.NewTunnelTemplate(1, "test-template", "tcp", &tunneltemplate.TemplateConfig{})
	existing.ID = 1

	cmd := DeleteTemplateCommand{
		TemplateID: 1,
		UserID:     2, // 不同的用户
	}

	repo.On("FindByID", ctx, uint(1)).Return(existing, nil)

	err := handler.Handle(ctx, cmd)
	assert.ErrorIs(t, err, tunneltemplate.ErrUnauthorized)

	repo.AssertExpectations(t)
}

func TestIncrementUsageHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewIncrementUsageHandler(repo)

	cmd := IncrementUsageCommand{
		TemplateID: 1,
	}

	repo.On("IncrementUsageCount", ctx, uint(1)).Return(nil)

	err := handler.Handle(ctx, cmd)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}
