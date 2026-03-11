package tunneltemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTunnelTemplate(t *testing.T) {
	config := &TemplateConfig{
		RemoteHost: "example.com",
		RemotePort: 8080,
	}

	template := NewTunnelTemplate(1, "test-template", "tcp", config)

	assert.Equal(t, uint(1), template.UserID)
	assert.Equal(t, "test-template", template.Name)
	assert.Equal(t, "tcp", template.Protocol)
	assert.Equal(t, config, template.Config)
	assert.False(t, template.IsPublic)
	assert.Equal(t, 0, template.UsageCount)
}

func TestTunnelTemplate_IsOwnedBy(t *testing.T) {
	template := NewTunnelTemplate(1, "test", "tcp", &TemplateConfig{})

	assert.True(t, template.IsOwnedBy(1))
	assert.False(t, template.IsOwnedBy(2))
}

func TestTunnelTemplate_CanBeAccessedBy(t *testing.T) {
	template := NewTunnelTemplate(1, "test", "tcp", &TemplateConfig{})

	// 所有者可以访问
	assert.True(t, template.CanBeAccessedBy(1))

	// 非所有者不能访问私有模板
	assert.False(t, template.CanBeAccessedBy(2))

	// 公开后任何人都可以访问
	template.MakePublic()
	assert.True(t, template.CanBeAccessedBy(2))
}

func TestTunnelTemplate_MakePublic(t *testing.T) {
	template := NewTunnelTemplate(1, "test", "tcp", &TemplateConfig{})
	assert.False(t, template.IsPublic)

	template.MakePublic()
	assert.True(t, template.IsPublic)
}

func TestTunnelTemplate_MakePrivate(t *testing.T) {
	template := NewTunnelTemplate(1, "test", "tcp", &TemplateConfig{})
	template.MakePublic()
	assert.True(t, template.IsPublic)

	template.MakePrivate()
	assert.False(t, template.IsPublic)
}

func TestTunnelTemplate_UpdateInfo(t *testing.T) {
	template := NewTunnelTemplate(1, "test", "tcp", &TemplateConfig{})

	desc := "new description"
	template.UpdateInfo("new-name", &desc)

	assert.Equal(t, "new-name", template.Name)
	assert.Equal(t, "new description", *template.Description)
}

func TestTunnelTemplate_UpdateConfig(t *testing.T) {
	template := NewTunnelTemplate(1, "test", "tcp", &TemplateConfig{
		RemoteHost: "old.com",
		RemotePort: 8080,
	})

	newConfig := &TemplateConfig{
		RemoteHost: "new.com",
		RemotePort: 9090,
	}

	template.UpdateConfig(newConfig)

	assert.Equal(t, "new.com", template.Config.RemoteHost)
	assert.Equal(t, 9090, template.Config.RemotePort)
}

func TestTunnelTemplate_IncrementUsage(t *testing.T) {
	template := NewTunnelTemplate(1, "test", "tcp", &TemplateConfig{})
	assert.Equal(t, 0, template.UsageCount)

	template.IncrementUsage()
	assert.Equal(t, 1, template.UsageCount)

	template.IncrementUsage()
	assert.Equal(t, 2, template.UsageCount)
}
