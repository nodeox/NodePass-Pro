package queries

import (
	"context"
	
	"nodepass-pro/backend/internal/domain/user"
	"nodepass-pro/backend/internal/infrastructure/cache"
)

// GetUserQuery 获取用户查询
type GetUserQuery struct {
	UserID uint
}

// GetUserResult 获取用户结果
type GetUserResult struct {
	User *user.User
}

// GetUserHandler 获取用户处理器
type GetUserHandler struct {
	userRepo  user.Repository
	userCache *cache.UserCache
}

// NewGetUserHandler 创建处理器
func NewGetUserHandler(repo user.Repository, cache *cache.UserCache) *GetUserHandler {
	return &GetUserHandler{
		userRepo:  repo,
		userCache: cache,
	}
}

// Handle 处理查询（Cache-Aside 模式）
func (h *GetUserHandler) Handle(ctx context.Context, query GetUserQuery) (*GetUserResult, error) {
	// 1. 先查缓存
	if cached, err := h.userCache.Get(ctx, query.UserID); err == nil && cached != nil {
		// 缓存命中，转换为实体
		u := &user.User{
			ID:       uint(cached["id"].(float64)),
			Username: cached["username"].(string),
			Email:    cached["email"].(string),
			Role:     cached["role"].(string),
			Status:   cached["status"].(string),
		}
		return &GetUserResult{User: u}, nil
	}
	
	// 2. 缓存未命中，查数据库
	u, err := h.userRepo.FindByID(ctx, query.UserID)
	if err != nil {
		return nil, err
	}
	
	// 3. 写入缓存
	userData := map[string]interface{}{
		"id":       u.ID,
		"username": u.Username,
		"email":    u.Email,
		"role":     u.Role,
		"status":   u.Status,
	}
	h.userCache.Set(ctx, u.ID, userData)
	
	return &GetUserResult{User: u}, nil
}
