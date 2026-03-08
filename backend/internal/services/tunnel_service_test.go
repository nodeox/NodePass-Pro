package services

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"nodepass-pro/backend/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTunnelStartFailsWhenExitEndpointMissing(t *testing.T) {
	dsn := fmt.Sprintf("file:%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("初始化测试数据库失败: %v", err)
	}

	if err = db.AutoMigrate(
		&models.User{},
		&models.NodeGroup{},
		&models.NodeInstance{},
		&models.NodeGroupRelation{},
		&models.NodeGroupStats{},
		&models.Tunnel{},
	); err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}

	user := &models.User{
		Username:     "test-user",
		Email:        "test@example.com",
		PasswordHash: "hashed-password",
		Role:         "user",
		Status:       "normal",
	}
	if err = db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	entryGroup := &models.NodeGroup{
		UserID:    user.ID,
		Name:      "entry-group",
		Type:      models.NodeGroupTypeEntry,
		IsEnabled: true,
	}
	if err = entryGroup.SetConfig(&models.NodeGroupConfig{
		AllowedProtocols: []string{"tcp"},
		PortRange:        models.PortRange{Start: 10000, End: 20000},
		EntryConfig: &models.EntryGroupConfig{
			RequireExitGroup: true,
		},
	}); err != nil {
		t.Fatalf("设置入口组配置失败: %v", err)
	}
	if err = db.Create(entryGroup).Error; err != nil {
		t.Fatalf("创建入口组失败: %v", err)
	}

	exitGroup := &models.NodeGroup{
		UserID:    user.ID,
		Name:      "exit-group",
		Type:      models.NodeGroupTypeExit,
		IsEnabled: true,
	}
	if err = exitGroup.SetConfig(&models.NodeGroupConfig{
		AllowedProtocols: []string{"tcp"},
		PortRange:        models.PortRange{Start: 10000, End: 20000},
		ExitConfig: &models.ExitGroupConfig{
			LoadBalanceStrategy: string(models.LoadBalanceRoundRobin),
		},
	}); err != nil {
		t.Fatalf("设置出口组配置失败: %v", err)
	}
	if err = db.Create(exitGroup).Error; err != nil {
		t.Fatalf("创建出口组失败: %v", err)
	}

	if err = db.Create(&models.NodeGroupRelation{
		EntryGroupID: entryGroup.ID,
		ExitGroupID:  exitGroup.ID,
		IsEnabled:    true,
	}).Error; err != nil {
		t.Fatalf("创建组关联失败: %v", err)
	}

	// 入口实例在线且地址完整。
	entryHost := "127.0.0.1"
	entryPort := 18080
	entryInstance := &models.NodeInstance{
		NodeGroupID:   entryGroup.ID,
		NodeID:        "entry-instance",
		AuthTokenHash: "hash-entry",
		Name:          "entry-instance",
		Host:          &entryHost,
		Port:          &entryPort,
		Status:        models.NodeInstanceStatusOnline,
		IsEnabled:     true,
		SystemInfo:    "{}",
		TrafficStats:  "{}",
	}
	if err = db.Create(entryInstance).Error; err != nil {
		t.Fatalf("创建入口实例失败: %v", err)
	}

	// 出口实例在线但地址缺失（host/port 都为空），应阻止隧道启动。
	exitInstance := &models.NodeInstance{
		NodeGroupID:   exitGroup.ID,
		NodeID:        "exit-instance",
		AuthTokenHash: "hash-exit",
		Name:          "exit-instance",
		Status:        models.NodeInstanceStatusOnline,
		IsEnabled:     true,
		SystemInfo:    "{}",
		TrafficStats:  "{}",
	}
	if err = db.Create(exitInstance).Error; err != nil {
		t.Fatalf("创建出口实例失败: %v", err)
	}

	tunnel := &models.Tunnel{
		UserID:       user.ID,
		Name:         "test-tunnel",
		EntryGroupID: entryGroup.ID,
		ExitGroupID:  &exitGroup.ID,
		Protocol:     "tcp",
		ListenHost:   "0.0.0.0",
		ListenPort:   19090,
		RemoteHost:   "192.168.1.10",
		RemotePort:   22,
		Status:       models.TunnelStatusStopped,
	}
	if err = tunnel.SetConfig(&models.TunnelConfig{
		LoadBalanceStrategy: models.LoadBalanceRoundRobin,
		IPType:              "auto",
	}); err != nil {
		t.Fatalf("设置隧道配置失败: %v", err)
	}
	if err = db.Create(tunnel).Error; err != nil {
		t.Fatalf("创建隧道失败: %v", err)
	}

	service := NewTunnelService(db)
	startErr := service.Start(user.ID, tunnel.ID)
	if startErr == nil {
		t.Fatalf("预期启动失败，但返回成功")
	}
	if !errors.Is(startErr, ErrInvalidParams) {
		t.Fatalf("预期错误包含 ErrInvalidParams，实际: %v", startErr)
	}

	var tunnelAfter models.Tunnel
	if err = db.First(&tunnelAfter, tunnel.ID).Error; err != nil {
		t.Fatalf("查询隧道状态失败: %v", err)
	}
	if tunnelAfter.Status != models.TunnelStatusStopped {
		t.Fatalf("隧道状态应保持 stopped，实际: %s", tunnelAfter.Status)
	}

	var entryAfter models.NodeInstance
	if err = db.First(&entryAfter, entryInstance.ID).Error; err != nil {
		t.Fatalf("查询入口实例失败: %v", err)
	}
	if entryAfter.ConfigVersion != 0 {
		t.Fatalf("启动失败时入口实例 config_version 不应变化，实际: %d", entryAfter.ConfigVersion)
	}
}
