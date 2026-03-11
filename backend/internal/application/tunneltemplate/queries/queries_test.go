package queries

import (
	"context"
	"testing"

	"nodepass-pro/backend/internal/domain/tunneltemplate"

	"github.com/stretchr/testify/assert"
)

func TestGetTemplateHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewGetTemplateHandler(repo)

	template := tunneltemplate.NewTunnelTemplate(1, "test-template", "tcp", &tunneltemplate.TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8080,
	})
	template.ID = 1

	query := GetTemplateQuery{
		TemplateID: 1,
		UserID:     1,
	}

	repo.On("FindByID", ctx, uint(1)).Return(template, nil)

	result, err := handler.Handle(ctx, query)
	assert.NoError(t, err)
	assert.Equal(t, template, result)

	repo.AssertExpectations(t)
}

func TestGetTemplateHandler_Handle_Unauthorized(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewGetTemplateHandler(repo)

	template := tunneltemplate.NewTunnelTemplate(1, "test-template", "tcp", &tunneltemplate.TemplateConfig{})
	template.ID = 1

	query := GetTemplateQuery{
		TemplateID: 1,
		UserID:     2, // 不同的用户
	}

	repo.On("FindByID", ctx, uint(1)).Return(template, nil)

	_, err := handler.Handle(ctx, query)
	assert.ErrorIs(t, err, tunneltemplate.ErrUnauthorized)

	repo.AssertExpectations(t)
}

func TestGetTemplateHandler_Handle_PublicTemplate(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewGetTemplateHandler(repo)

	template := tunneltemplate.NewTunnelTemplate(1, "test-template", "tcp", &tunneltemplate.TemplateConfig{})
	template.ID = 1
	template.MakePublic()

	query := GetTemplateQuery{
		TemplateID: 1,
		UserID:     2, // 不同的用户，但模板是公开的
	}

	repo.On("FindByID", ctx, uint(1)).Return(template, nil)

	result, err := handler.Handle(ctx, query)
	assert.NoError(t, err)
	assert.Equal(t, template, result)

	repo.AssertExpectations(t)
}

func TestListTemplatesHandler_Handle(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewListTemplatesHandler(repo)

	templates := []*tunneltemplate.TunnelTemplate{
		tunneltemplate.NewTunnelTemplate(1, "template1", "tcp", &tunneltemplate.TemplateConfig{}),
		tunneltemplate.NewTunnelTemplate(1, "template2", "udp", &tunneltemplate.TemplateConfig{}),
	}

	query := ListTemplatesQuery{
		UserID:   1,
		Page:     1,
		PageSize: 20,
	}

	repo.On("List", ctx, tunneltemplate.ListFilter{
		UserID:   1,
		Protocol: (*string)(nil),
		IsPublic: (*bool)(nil),
		Page:     1,
		PageSize: 20,
	}).Return(templates, int64(2), nil)

	result, total, err := handler.Handle(ctx, query)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, result, 2)

	repo.AssertExpectations(t)
}

func TestListTemplatesHandler_Handle_DefaultValues(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewListTemplatesHandler(repo)

	query := ListTemplatesQuery{
		UserID:   1,
		Page:     0, // 应该使用默认值 1
		PageSize: 0, // 应该使用默认值 20
	}

	repo.On("List", ctx, tunneltemplate.ListFilter{
		UserID:   1,
		Protocol: (*string)(nil),
		IsPublic: (*bool)(nil),
		Page:     1,
		PageSize: 20,
	}).Return([]*tunneltemplate.TunnelTemplate{}, int64(0), nil)

	_, _, err := handler.Handle(ctx, query)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestListTemplatesHandler_Handle_MaxPageSize(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	handler := NewListTemplatesHandler(repo)

	query := ListTemplatesQuery{
		UserID:   1,
		Page:     1,
		PageSize: 300, // 超过最大值，应该使用 200
	}

	repo.On("List", ctx, tunneltemplate.ListFilter{
		UserID:   1,
		Protocol: (*string)(nil),
		IsPublic: (*bool)(nil),
		Page:     1,
		PageSize: 200,
	}).Return([]*tunneltemplate.TunnelTemplate{}, int64(0), nil)

	_, _, err := handler.Handle(ctx, query)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}
